// Package admin provides plain net/http handlers for the staff-only /admin
// surface and /auth/me. Mounted ahead of the go-swagger mux in
// internal/http/module.go (mirrors internal/http/uploads).
package admin

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gofrs/uuid"

	adminusers "github.com/Pashteto/lia/internal/adminusers"
	complaints "github.com/Pashteto/lia/internal/complaints"
	eventsdomain "github.com/Pashteto/lia/internal/events"
	hygiene "github.com/Pashteto/lia/internal/hygiene"
	domain "github.com/Pashteto/lia/internal/models"
	"github.com/Pashteto/lia/internal/moderation"
	"github.com/Pashteto/lia/internal/organizers"
	platforms "github.com/Pashteto/lia/internal/platforms"
	"github.com/Pashteto/lia/internal/settings"
	"github.com/Pashteto/lia/pkg/logger"
)

// Deps are the collaborators the admin handler needs.
type Deps struct {
	Authenticate func(token string) (*domain.User, error)
	Moderation   moderation.Service
	Events       eventsdomain.Service                    // List(ctx, status) for the queue
	LatestReason func(eventID uuid.UUID) (string, error) // moderation.Repository.LatestReason bound
	Organizers   organizers.Service
	Settings     settings.Service
	Complaints   complaints.Service
	Users        adminusers.Service
	Hygiene      hygiene.Service
	Platforms    platforms.Service
}

type handler struct {
	deps Deps
	mux  *http.ServeMux
}

// NewHandler returns the mounted admin handler.
func NewHandler(deps Deps) http.Handler {
	h := &handler{deps: deps, mux: http.NewServeMux()}
	h.mux.HandleFunc("GET /auth/me", h.me)
	h.mux.HandleFunc("GET /api/v1/admin/overview", h.staff(h.overview))
	h.mux.HandleFunc("GET /api/v1/admin/moderation/events", h.staff(h.listEvents))
	h.mux.HandleFunc("POST /api/v1/admin/moderation/events/{id}/takedown", h.staff(h.takedown))
	h.mux.HandleFunc("POST /api/v1/admin/moderation/events/{id}/reinstate", h.staff(h.reinstate))
	h.mux.HandleFunc("POST /api/v1/admin/moderation/events/{id}/approve", h.staff(h.approve))
	h.mux.HandleFunc("GET /api/v1/admin/moderation/organizers", h.staff(h.listOrganizers))
	h.mux.HandleFunc("GET /api/v1/admin/organizers", h.staff(h.searchOrganizers))
	h.mux.HandleFunc("GET /api/v1/admin/organizers/{id}", h.staff(h.organizerDetail))
	h.mux.HandleFunc("GET /api/v1/admin/users", h.staff(h.listUsers))
	h.mux.HandleFunc("GET /api/v1/admin/hygiene", h.staff(h.listHygiene))
	h.mux.HandleFunc("POST /api/v1/admin/hygiene/hide-all", h.staff(h.hideAllHygiene))
	h.mux.HandleFunc("POST /api/v1/admin/moderation/organizers/{id}/verify", h.staff(h.verifyOrganizer))
	h.mux.HandleFunc("POST /api/v1/admin/moderation/organizers/{id}/reject", h.staff(h.rejectOrganizer))
	h.mux.HandleFunc("POST /api/v1/admin/moderation/organizers/{id}/revoke", h.staff(h.revokeOrganizer))
	h.mux.HandleFunc("POST /api/v1/admin/organizers/{id}/auto-verify", h.staff(h.setAutoVerify))
	h.mux.HandleFunc("POST /api/v1/admin/organizers/{id}/daily-limit", h.staff(h.setDailyLimit))
	h.mux.HandleFunc("GET /api/v1/admin/settings", h.staff(h.getSettings))
	h.mux.HandleFunc("PUT /api/v1/admin/settings", h.staff(h.putSettings))
	h.mux.HandleFunc("GET /api/v1/admin/complaints", h.staff(h.listComplaints))
	h.mux.HandleFunc("GET /api/v1/admin/complaints/events/{id}", h.staff(h.complaintDetail))
	h.mux.HandleFunc("POST /api/v1/admin/complaints/events/{id}/resolve", h.staff(h.resolveComplaints))
	h.mux.HandleFunc("GET /api/v1/admin/trusted-platforms", h.staff(h.listPlatforms))
	h.mux.HandleFunc("POST /api/v1/admin/trusted-platforms", h.staff(h.addPlatform))
	h.mux.HandleFunc("DELETE /api/v1/admin/trusted-platforms/{id}", h.staff(h.removePlatform))
	return h
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) { h.mux.ServeHTTP(w, r) }

// principal extracts + authenticates the bearer token. Returns nil on failure.
func (h *handler) principal(r *http.Request) *domain.User {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return nil
	}
	u, err := h.deps.Authenticate(strings.TrimPrefix(authHeader, "Bearer "))
	if err != nil || u == nil {
		return nil
	}
	return u
}

// staff wraps a handler with auth + role gate.
func (h *handler) staff(next func(http.ResponseWriter, *http.Request, *domain.User)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := h.principal(r)
		if u == nil {
			writeErr(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		if u.Role != "admin" {
			writeErr(w, http.StatusForbidden, "Недостаточно прав")
			return
		}
		next(w, r, u)
	}
}

func (h *handler) me(w http.ResponseWriter, r *http.Request) {
	u := h.principal(r)
	if u == nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id": u.UUID.String(), "email": u.Email, "name": u.Name, "role": u.Role,
		"email_verified": u.EmailVerified,
		// Registration month for the profile identity strip (U6).
		"created_at": u.CreatedAt,
	})
}

func (h *handler) overview(w http.ResponseWriter, r *http.Request, _ *domain.User) {
	if h.deps.Moderation == nil {
		writeErr(w, http.StatusServiceUnavailable, "moderation service not available")
		return
	}
	c, err := h.deps.Moderation.Overview(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "overview failed")
		return
	}
	resp := map[string]int{
		"events_total":     c.EventsTotal,
		"events_published": c.EventsPublished,
		"events_removed":   c.EventsRemoved,
	}
	if h.deps.Organizers != nil {
		if oc, oerr := h.deps.Organizers.Overview(r.Context()); oerr == nil {
			resp["organizers_pending"] = oc.OrganizersPending
			resp["organizers_total"] = oc.OrganizersTotal
		}
	}
	if h.deps.Users != nil {
		if n, uerr := h.deps.Users.Count(r.Context()); uerr == nil {
			resp["users_total"] = n
		}
	}
	if h.deps.Complaints != nil {
		if n, cerr := h.deps.Complaints.OpenEventCount(r.Context()); cerr == nil {
			resp["complaints_open"] = n
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

type adminEventJSON struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Status   string `json:"status"`
	StartsAt string `json:"starts_at"`
	// When the event went live. The moderation queue is post-hoc, so this is
	// the «подано» moment the detail pane labels; starts_at is when the event
	// happens, which is a different thing entirely.
	PublishedAt   string `json:"published_at,omitempty"`
	CoverURL      string `json:"cover_url,omitempty"`
	OrganizerName string `json:"organizer_name,omitempty"`
	Reason        string `json:"reason,omitempty"`
	// ExternalRegistrationURL/ExternalPlatformName let the «Ссылки» queue show
	// the URL under judgment without a detail fetch — pending_review events are
	// not publicly visible, so the admin detail pane can't always load them.
	ExternalRegistrationURL string `json:"external_registration_url,omitempty"`
	ExternalPlatformName    string `json:"external_platform_name,omitempty"`
}

func (h *handler) listEvents(w http.ResponseWriter, r *http.Request, _ *domain.User) {
	if h.deps.Events == nil {
		writeErr(w, http.StatusServiceUnavailable, "events service not available")
		return
	}
	status := r.URL.Query().Get("status")
	if status != "published" && status != "rejected" && status != "pending_review" {
		status = "published"
	}
	events, err := h.deps.Events.List(r.Context(), status, nil, nil, nil)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "list failed")
		return
	}
	out := make([]adminEventJSON, 0, len(events))
	for _, e := range events {
		j := adminEventJSON{
			ID:                      e.ID.String(),
			Title:                   e.Title,
			Status:                  e.StatusSQL,
			StartsAt:                e.StartsAt.Format("2006-01-02T15:04:05Z07:00"),
			CoverURL:                e.CoverURL,
			ExternalRegistrationURL: e.ExternalRegistrationURL,
			ExternalPlatformName:    e.ExternalPlatformName,
		}
		if e.PublishedAt != nil {
			j.PublishedAt = e.PublishedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
		}
		if e.Organizer != nil {
			j.OrganizerName = e.Organizer.Name
		}
		if status == "rejected" && h.deps.LatestReason != nil {
			if reason, rerr := h.deps.LatestReason(e.ID); rerr == nil {
				j.Reason = reason
			}
		}
		out = append(out, j)
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *handler) takedown(w http.ResponseWriter, r *http.Request, u *domain.User) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	switch err := h.deps.Moderation.Takedown(r.Context(), id, u.UUID, body.Reason); err {
	case nil:
		writeJSON(w, http.StatusOK, map[string]string{"status": "rejected"})
	case moderation.ErrReasonRequired:
		writeErr(w, http.StatusBadRequest, "Укажите причину снятия")
	case moderation.ErrInvalidTransition:
		writeErr(w, http.StatusConflict, "Событие нельзя снять из текущего статуса")
	default:
		writeErr(w, http.StatusInternalServerError, "takedown failed")
	}
}

func (h *handler) reinstate(w http.ResponseWriter, r *http.Request, u *domain.User) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	switch err := h.deps.Moderation.Reinstate(r.Context(), id, u.UUID); err {
	case nil:
		writeJSON(w, http.StatusOK, map[string]string{"status": "published"})
	case moderation.ErrInvalidTransition:
		writeErr(w, http.StatusConflict, "Событие нельзя вернуть из текущего статуса")
	default:
		writeErr(w, http.StatusInternalServerError, "reinstate failed")
	}
}

// approve promotes a pending_review event (unknown external-registration
// domain awaiting staff review) to published and marks its URL verified.
func (h *handler) approve(w http.ResponseWriter, r *http.Request, u *domain.User) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	switch err := h.deps.Moderation.Approve(r.Context(), id, u.UUID); err {
	case nil:
		w.WriteHeader(http.StatusNoContent)
	case moderation.ErrInvalidTransition:
		writeErr(w, http.StatusConflict, "Событие нельзя одобрить из текущего статуса")
	default:
		writeErr(w, http.StatusInternalServerError, "approve failed")
	}
}

func pathID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.FromString(r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return uuid.Nil, false
	}
	return id, true
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

type adminOrganizerJSON struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Description        string `json:"description"`
	WebsiteURL         string `json:"website_url"`
	VerificationStatus string `json:"verification_status"`
	AutoVerify         bool   `json:"auto_verify"`
	LatestReason       string `json:"latest_reason,omitempty"`
	// A3 registry columns «Событий» / «Жалоб», which had no data to render.
	EventsCount     int `json:"events_count"`
	ComplaintsCount int `json:"complaints_count"`
	// DailyEventLimit is this organizer's override of the global daily
	// creation cap. Absent means "use the default".
	DailyEventLimit *int `json:"daily_event_limit,omitempty"`
}

func toAdminOrganizerJSON(o organizers.Organizer) adminOrganizerJSON {
	return adminOrganizerJSON{
		ID: o.ID.String(), Name: o.Name, Description: o.Description, WebsiteURL: o.WebsiteURL,
		VerificationStatus: o.VerificationStatus, AutoVerify: o.AutoVerify, LatestReason: o.LatestReason,
		EventsCount: o.EventsCount, ComplaintsCount: o.ComplaintsCount,
		DailyEventLimit: o.DailyEventLimit,
	}
}

func (h *handler) listOrganizers(w http.ResponseWriter, r *http.Request, _ *domain.User) {
	if h.deps.Organizers == nil {
		writeErr(w, http.StatusServiceUnavailable, "organizers service not available")
		return
	}
	status := r.URL.Query().Get("status")
	switch status {
	case "pending", "verified", "rejected", "draft":
	default:
		status = "pending"
	}
	orgs, err := h.deps.Organizers.List(r.Context(), organizers.ListFilter{Status: status})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "list failed")
		return
	}
	out := make([]adminOrganizerJSON, 0, len(orgs))
	for _, o := range orgs {
		out = append(out, toAdminOrganizerJSON(o))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *handler) searchOrganizers(w http.ResponseWriter, r *http.Request, _ *domain.User) {
	if h.deps.Organizers == nil {
		writeErr(w, http.StatusServiceUnavailable, "organizers service not available")
		return
	}
	orgs, err := h.deps.Organizers.List(r.Context(), organizers.ListFilter{Query: r.URL.Query().Get("q")})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "search failed")
		return
	}
	out := make([]adminOrganizerJSON, 0, len(orgs))
	for _, o := range orgs {
		out = append(out, toAdminOrganizerJSON(o))
	}
	writeJSON(w, http.StatusOK, out)
}

type organizerDetailJSON struct {
	adminOrganizerJSON
	History []historyJSON    `json:"history"`
	Events  []adminEventJSON `json:"events"`
}

type historyJSON struct {
	FromStatus string `json:"from_status"`
	ToStatus   string `json:"to_status"`
	Reason     string `json:"reason,omitempty"`
	Actor      string `json:"actor_user_id"`
	CreatedAt  string `json:"created_at"`
}

func (h *handler) organizerDetail(w http.ResponseWriter, r *http.Request, _ *domain.User) {
	if h.deps.Organizers == nil {
		writeErr(w, http.StatusServiceUnavailable, "organizers service not available")
		return
	}
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	o, hist, err := h.deps.Organizers.GetWithHistory(r.Context(), id)
	if err == organizers.ErrNotFound {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "detail failed")
		return
	}
	out := organizerDetailJSON{adminOrganizerJSON: toAdminOrganizerJSON(*o)}
	for _, e := range hist {
		out.History = append(out.History, historyJSON{
			FromStatus: e.FromStatus, ToStatus: e.ToStatus, Reason: e.Reason,
			Actor: e.ActorUserID.String(), CreatedAt: e.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	// Every event this organizer owns, any status — the registry row is only
	// useful if you can get from it to the work being moderated.
	if h.deps.Events != nil {
		events, eerr := h.deps.Events.ListByOrganizer(r.Context(), o.OwnerUserID)
		if eerr != nil {
			// The profile is still worth showing without its events.
			logger.Log().Errorf("list events for organizer %s: %s", o.ID, eerr.Error())
		}
		for _, e := range events {
			j := adminEventJSON{
				ID:       e.ID.String(),
				Title:    e.Title,
				Status:   e.StatusSQL,
				StartsAt: e.StartsAt.Format("2006-01-02T15:04:05Z07:00"),
				CoverURL: e.CoverURL,
			}
			if e.PublishedAt != nil {
				j.PublishedAt = e.PublishedAt.UTC().Format("2006-01-02T15:04:05Z07:00")
			}
			j.OrganizerName = o.Name
			out.Events = append(out.Events, j)
		}
	}

	writeJSON(w, http.StatusOK, out)
}

// setDailyLimit sets or clears an organizer's daily event-creation override.
// Body: {"limit": 5} to set, {"limit": null} to fall back to the global default.
func (h *handler) setDailyLimit(w http.ResponseWriter, r *http.Request, u *domain.User) {
	if h.deps.Organizers == nil {
		writeErr(w, http.StatusServiceUnavailable, "organizers service not available")
		return
	}
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var body struct {
		Limit *int `json:"limit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	if body.Limit != nil && *body.Limit < 0 {
		writeErr(w, http.StatusBadRequest, "Лимит не может быть отрицательным")
		return
	}
	if err := h.deps.Organizers.SetDailyEventLimit(r.Context(), id, u.UUID, body.Limit); err != nil {
		if err == organizers.ErrNotFound {
			writeErr(w, http.StatusNotFound, "not found")
			return
		}
		writeErr(w, http.StatusInternalServerError, "update failed")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) verifyOrganizer(w http.ResponseWriter, r *http.Request, u *domain.User) {
	if h.deps.Organizers == nil {
		writeErr(w, http.StatusServiceUnavailable, "organizers service not available")
		return
	}
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	switch err := h.deps.Organizers.Verify(r.Context(), id, u.UUID); err {
	case nil:
		writeJSON(w, http.StatusOK, map[string]string{"status": "verified"})
	case organizers.ErrInvalidTransition:
		writeErr(w, http.StatusConflict, "Профиль нельзя подтвердить из текущего статуса")
	default:
		writeErr(w, http.StatusInternalServerError, "verify failed")
	}
}

func (h *handler) rejectOrganizer(w http.ResponseWriter, r *http.Request, u *domain.User) {
	if h.deps.Organizers == nil {
		writeErr(w, http.StatusServiceUnavailable, "organizers service not available")
		return
	}
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	switch err := h.deps.Organizers.Reject(r.Context(), id, u.UUID, body.Reason); err {
	case nil:
		writeJSON(w, http.StatusOK, map[string]string{"status": "rejected"})
	case organizers.ErrReasonRequired:
		writeErr(w, http.StatusBadRequest, "Укажите причину отклонения")
	case organizers.ErrInvalidTransition:
		writeErr(w, http.StatusConflict, "Профиль нельзя отклонить из текущего статуса")
	default:
		writeErr(w, http.StatusInternalServerError, "reject failed")
	}
}

func (h *handler) revokeOrganizer(w http.ResponseWriter, r *http.Request, u *domain.User) {
	if h.deps.Organizers == nil {
		writeErr(w, http.StatusServiceUnavailable, "organizers service not available")
		return
	}
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	switch err := h.deps.Organizers.Revoke(r.Context(), id, u.UUID, body.Reason); err {
	case nil:
		writeJSON(w, http.StatusOK, map[string]string{"status": "rejected"})
	case organizers.ErrReasonRequired:
		writeErr(w, http.StatusBadRequest, "Укажите причину отзыва")
	case organizers.ErrInvalidTransition:
		writeErr(w, http.StatusConflict, "Профиль нельзя отозвать из текущего статуса")
	default:
		writeErr(w, http.StatusInternalServerError, "revoke failed")
	}
}

func (h *handler) setAutoVerify(w http.ResponseWriter, r *http.Request, u *domain.User) {
	if h.deps.Organizers == nil {
		writeErr(w, http.StatusServiceUnavailable, "organizers service not available")
		return
	}
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var body struct {
		Enabled bool `json:"enabled"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	switch err := h.deps.Organizers.SetAutoVerify(r.Context(), id, u.UUID, body.Enabled); err {
	case nil:
		writeJSON(w, http.StatusOK, map[string]bool{"auto_verify": body.Enabled})
	case organizers.ErrNotFound:
		writeErr(w, http.StatusNotFound, "not found")
	default:
		writeErr(w, http.StatusInternalServerError, "set auto-verify failed")
	}
}

func (h *handler) listComplaints(w http.ResponseWriter, r *http.Request, _ *domain.User) {
	if h.deps.Complaints == nil {
		writeErr(w, http.StatusServiceUnavailable, "complaints service not available")
		return
	}
	groups, err := h.deps.Complaints.ListInbox(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "list failed")
		return
	}
	if groups == nil {
		groups = []complaints.EventReportGroup{}
	}
	writeJSON(w, http.StatusOK, groups)
}

type complaintJSON struct {
	ID         string `json:"id"`
	Category   string `json:"category"`
	Note       string `json:"note,omitempty"`
	Status     string `json:"status"`
	Resolution string `json:"resolution,omitempty"`
	Reporter   string `json:"reporter_user_id"`
	CreatedAt  string `json:"created_at"`
}

func (h *handler) complaintDetail(w http.ResponseWriter, r *http.Request, _ *domain.User) {
	if h.deps.Complaints == nil {
		writeErr(w, http.StatusServiceUnavailable, "complaints service not available")
		return
	}
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	items, err := h.deps.Complaints.TargetDetail(r.Context(), "event", id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "detail failed")
		return
	}
	out := make([]complaintJSON, 0, len(items))
	for _, c := range items {
		out = append(out, complaintJSON{
			ID: c.ID.String(), Category: c.Category, Note: c.Note, Status: c.Status,
			Resolution: c.Resolution, Reporter: c.ReporterUserID.String(),
			CreatedAt: c.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *handler) resolveComplaints(w http.ResponseWriter, r *http.Request, u *domain.User) {
	if h.deps.Complaints == nil {
		writeErr(w, http.StatusServiceUnavailable, "complaints service not available")
		return
	}
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var body struct {
		Action     string `json:"action"`
		Resolution string `json:"resolution"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	err := h.deps.Complaints.Resolve(r.Context(), "event", id, u.UUID, body.Action, body.Resolution)
	switch {
	case err == nil:
		writeJSON(w, http.StatusOK, map[string]string{"status": "resolved"})
	case errors.Is(err, complaints.ErrResolutionRequired):
		writeErr(w, http.StatusBadRequest, "Укажите причину")
	case errors.Is(err, complaints.ErrInvalidAction):
		writeErr(w, http.StatusBadRequest, "Некорректное действие")
	case errors.Is(err, moderation.ErrInvalidTransition):
		writeErr(w, http.StatusConflict, "Событие нельзя снять из текущего статуса")
	default:
		writeErr(w, http.StatusInternalServerError, "resolve failed")
	}
}

func (h *handler) getSettings(w http.ResponseWriter, r *http.Request, _ *domain.User) {
	if h.deps.Settings == nil {
		writeErr(w, http.StatusServiceUnavailable, "settings service not available")
		return
	}
	all, err := h.deps.Settings.All(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "settings failed")
		return
	}
	writeJSON(w, http.StatusOK, all)
}

func (h *handler) putSettings(w http.ResponseWriter, r *http.Request, u *domain.User) {
	if h.deps.Settings == nil {
		writeErr(w, http.StatusServiceUnavailable, "settings service not available")
		return
	}
	var body struct {
		Key     string `json:"key"`
		Enabled bool   `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Key == "" {
		writeErr(w, http.StatusBadRequest, "key required")
		return
	}
	if err := h.deps.Settings.SetBool(r.Context(), body.Key, u.UUID, body.Enabled); err != nil {
		writeErr(w, http.StatusInternalServerError, "update failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{body.Key: body.Enabled})
}

type adminUserJSON struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	Role        string `json:"role"`
	CreatedAt   string `json:"created_at"`
	Bookings    int    `json:"bookings"`
	IsOrganizer bool   `json:"is_organizer"`
}

// listUsers serves the A4 registry. Pagination is bounded by the service
// (DefaultLimit / MaxLimit); q matches name or email.
func (h *handler) listUsers(w http.ResponseWriter, r *http.Request, _ *domain.User) {
	if h.deps.Users == nil {
		writeErr(w, http.StatusServiceUnavailable, "users service not available")
		return
	}
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	rows, err := h.deps.Users.List(r.Context(), adminusers.Filter{
		Query: q.Get("q"), Limit: limit, Offset: offset,
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "users list failed")
		return
	}
	out := make([]adminUserJSON, 0, len(rows))
	for _, u := range rows {
		out = append(out, adminUserJSON{
			ID: u.ID.String(), Name: u.Name, Email: u.Email, Role: u.Role,
			CreatedAt: u.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
			Bookings:  u.Bookings, IsOrganizer: u.IsOrganizer,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

// listHygiene serves the A4 «Гигиена контента» rail: detected test data and
// nonsense prices currently visible in the public feed.
func (h *handler) listHygiene(w http.ResponseWriter, r *http.Request, _ *domain.User) {
	if h.deps.Hygiene == nil {
		writeErr(w, http.StatusServiceUnavailable, "hygiene service not available")
		return
	}
	issues, err := h.deps.Hygiene.List(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "hygiene list failed")
		return
	}
	if issues == nil {
		issues = []hygiene.Issue{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"issues": issues, "count": len(issues)})
}

// hideAllHygiene takes down every currently detected event (published →
// rejected, audit-logged per event). Events whose status already moved are
// reported as skipped, not failed.
func (h *handler) hideAllHygiene(w http.ResponseWriter, r *http.Request, u *domain.User) {
	if h.deps.Hygiene == nil {
		writeErr(w, http.StatusServiceUnavailable, "hygiene service not available")
		return
	}
	res, err := h.deps.Hygiene.HideAll(r.Context(), u.UUID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "hygiene hide-all failed")
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// listPlatforms serves every trusted-platform row, including inactive ones,
// for the admin whitelist screen.
func (h *handler) listPlatforms(w http.ResponseWriter, r *http.Request, _ *domain.User) {
	if h.deps.Platforms == nil {
		writeErr(w, http.StatusServiceUnavailable, "platforms service not available")
		return
	}
	rows, err := h.deps.Platforms.List(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "platforms list failed")
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

// addPlatform adds a new trusted-platform whitelist entry.
func (h *handler) addPlatform(w http.ResponseWriter, r *http.Request, _ *domain.User) {
	if h.deps.Platforms == nil {
		writeErr(w, http.StatusServiceUnavailable, "platforms service not available")
		return
	}
	var body struct {
		DomainSuffix string `json:"domain_suffix"`
		DisplayName  string `json:"display_name"`
		Category     string `json:"category"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	p, err := h.deps.Platforms.Add(r.Context(), body.DomainSuffix, body.DisplayName, body.Category)
	switch {
	case err == nil:
		writeJSON(w, http.StatusCreated, p)
	case errors.Is(err, platforms.ErrInvalidInput):
		writeErr(w, http.StatusUnprocessableEntity, err.Error())
	default:
		writeErr(w, http.StatusInternalServerError, "add platform failed")
	}
}

// removePlatform deactivates (does not hard-delete) a trusted-platform row.
func (h *handler) removePlatform(w http.ResponseWriter, r *http.Request, _ *domain.User) {
	if h.deps.Platforms == nil {
		writeErr(w, http.StatusServiceUnavailable, "platforms service not available")
		return
	}
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	switch err := h.deps.Platforms.Deactivate(r.Context(), id); {
	case err == nil:
		w.WriteHeader(http.StatusNoContent)
	case errors.Is(err, platforms.ErrNotFound):
		writeErr(w, http.StatusNotFound, "not found")
	default:
		writeErr(w, http.StatusInternalServerError, "deactivate platform failed")
	}
}
