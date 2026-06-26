// Package admin provides plain net/http handlers for the staff-only /admin
// surface and /auth/me. Mounted ahead of the go-swagger mux in
// internal/http/module.go (mirrors internal/http/uploads).
package admin

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/gofrs/uuid"

	complaints "github.com/Pashteto/lia/internal/complaints"
	eventsdomain "github.com/Pashteto/lia/internal/events"
	domain "github.com/Pashteto/lia/internal/models"
	"github.com/Pashteto/lia/internal/moderation"
	"github.com/Pashteto/lia/internal/organizers"
	"github.com/Pashteto/lia/internal/settings"
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
	h.mux.HandleFunc("GET /api/v1/admin/moderation/organizers", h.staff(h.listOrganizers))
	h.mux.HandleFunc("GET /api/v1/admin/organizers", h.staff(h.searchOrganizers))
	h.mux.HandleFunc("GET /api/v1/admin/organizers/{id}", h.staff(h.organizerDetail))
	h.mux.HandleFunc("POST /api/v1/admin/moderation/organizers/{id}/verify", h.staff(h.verifyOrganizer))
	h.mux.HandleFunc("POST /api/v1/admin/moderation/organizers/{id}/reject", h.staff(h.rejectOrganizer))
	h.mux.HandleFunc("POST /api/v1/admin/moderation/organizers/{id}/revoke", h.staff(h.revokeOrganizer))
	h.mux.HandleFunc("POST /api/v1/admin/organizers/{id}/auto-verify", h.staff(h.setAutoVerify))
	h.mux.HandleFunc("GET /api/v1/admin/settings", h.staff(h.getSettings))
	h.mux.HandleFunc("PUT /api/v1/admin/settings", h.staff(h.putSettings))
	h.mux.HandleFunc("GET /api/v1/admin/complaints", h.staff(h.listComplaints))
	h.mux.HandleFunc("GET /api/v1/admin/complaints/events/{id}", h.staff(h.complaintDetail))
	h.mux.HandleFunc("POST /api/v1/admin/complaints/events/{id}/resolve", h.staff(h.resolveComplaints))
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
	writeJSON(w, http.StatusOK, map[string]string{
		"id": u.UUID.String(), "email": u.Email, "name": u.Name, "role": u.Role,
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
	ID            string `json:"id"`
	Title         string `json:"title"`
	Status        string `json:"status"`
	StartsAt      string `json:"starts_at"`
	CoverURL      string `json:"cover_url,omitempty"`
	OrganizerName string `json:"organizer_name,omitempty"`
	Reason        string `json:"reason,omitempty"`
}

func (h *handler) listEvents(w http.ResponseWriter, r *http.Request, _ *domain.User) {
	if h.deps.Events == nil {
		writeErr(w, http.StatusServiceUnavailable, "events service not available")
		return
	}
	status := r.URL.Query().Get("status")
	if status != "published" && status != "rejected" {
		status = "published"
	}
	events, err := h.deps.Events.List(r.Context(), status)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "list failed")
		return
	}
	out := make([]adminEventJSON, 0, len(events))
	for _, e := range events {
		j := adminEventJSON{
			ID:       e.ID.String(),
			Title:    e.Title,
			Status:   e.StatusSQL,
			StartsAt: e.StartsAt.Format("2006-01-02T15:04:05Z07:00"),
			CoverURL: e.CoverURL,
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
}

func toAdminOrganizerJSON(o organizers.Organizer) adminOrganizerJSON {
	return adminOrganizerJSON{
		ID: o.ID.String(), Name: o.Name, Description: o.Description, WebsiteURL: o.WebsiteURL,
		VerificationStatus: o.VerificationStatus, AutoVerify: o.AutoVerify, LatestReason: o.LatestReason,
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
	History []historyJSON `json:"history"`
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
	writeJSON(w, http.StatusOK, out)
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
