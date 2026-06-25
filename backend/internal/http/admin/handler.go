// Package admin provides plain net/http handlers for the staff-only /admin
// surface and /auth/me. Mounted ahead of the go-swagger mux in
// internal/http/module.go (mirrors internal/http/uploads).
package admin

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gofrs/uuid"

	eventsdomain "github.com/Pashteto/lia/internal/events"
	domain "github.com/Pashteto/lia/internal/models"
	"github.com/Pashteto/lia/internal/moderation"
)

// Deps are the collaborators the admin handler needs.
type Deps struct {
	Authenticate func(token string) (*domain.User, error)
	Moderation   moderation.Service
	Events       eventsdomain.Service        // List(ctx, status) for the queue
	LatestReason func(eventID uuid.UUID) (string, error) // moderation.Repository.LatestReason bound
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
	writeJSON(w, http.StatusOK, c)
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
