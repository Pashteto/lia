// Package invitations exposes the event-invitations Service over HTTP: an
// organizer inviting emails to an event, a public token preview, and
// accept/decline both by token (from the invite email) and by id (from the
// authed "my invitations" list). Mounted ahead of the go-swagger mux in
// internal/http/module.go (mirrors internal/http/complaints).
package invitations

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gofrs/uuid"

	invdomain "github.com/Pashteto/lia/internal/invitations"
	domain "github.com/Pashteto/lia/internal/models"
)

// Deps wires the handler.
type Deps struct {
	Authenticate func(token string) (*domain.User, error)
	Service      invdomain.Service
	BaseURL      string
}

type handler struct {
	deps Deps
	mux  *http.ServeMux
}

// NewHandler builds the invitations HTTP surface (mounted ahead of the swagger mux).
func NewHandler(deps Deps) http.Handler {
	h := &handler{deps: deps, mux: http.NewServeMux()}
	h.mux.HandleFunc("POST /api/v1/events/{id}/invitations", h.create)
	h.mux.HandleFunc("GET /api/v1/invitations/{token}", h.preview)
	h.mux.HandleFunc("POST /api/v1/invitations/{token}/accept", h.acceptToken)
	h.mux.HandleFunc("POST /api/v1/invitations/{token}/decline", h.declineToken)
	h.mux.HandleFunc("GET /api/v1/me/invitations", h.listMine)
	h.mux.HandleFunc("POST /api/v1/me/invitations/{id}/accept", h.acceptID)
	h.mux.HandleFunc("POST /api/v1/me/invitations/{id}/decline", h.declineID)
	return h
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) { h.mux.ServeHTTP(w, r) }

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

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"message": msg})
}
func writeUnverified(w http.ResponseWriter) {
	writeJSON(w, http.StatusForbidden, map[string]string{"code": "email_not_verified", "message": "Подтвердите электронную почту, чтобы продолжить"})
}

func (h *handler) create(w http.ResponseWriter, r *http.Request) {
	u := h.principal(r)
	if u == nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !u.EmailVerified {
		writeUnverified(w)
		return
	}
	eventID, err := uuid.FromString(r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "bad event id")
		return
	}
	var body struct {
		Emails []string `json:"emails"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Emails) == 0 {
		writeErr(w, http.StatusBadRequest, "emails are required")
		return
	}
	n, err := h.deps.Service.Invite(r.Context(), eventID, u.UUID, u.EmailVerified, body.Emails, h.deps.BaseURL)
	if err != nil {
		h.mapErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]int{"invited": n})
}

func (h *handler) preview(w http.ResponseWriter, r *http.Request) {
	p, err := h.deps.Service.Preview(r.Context(), r.PathValue("token"))
	if err != nil {
		h.mapErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"event_id": p.EventID.String(), "event_title": p.EventTitle, "status": p.Status,
	})
}

func (h *handler) acceptToken(w http.ResponseWriter, r *http.Request)  { h.respond(w, r, true, true) }
func (h *handler) declineToken(w http.ResponseWriter, r *http.Request) { h.respond(w, r, true, false) }
func (h *handler) acceptID(w http.ResponseWriter, r *http.Request)     { h.respond(w, r, false, true) }
func (h *handler) declineID(w http.ResponseWriter, r *http.Request)    { h.respond(w, r, false, false) }

func (h *handler) respond(w http.ResponseWriter, r *http.Request, byToken, accept bool) {
	u := h.principal(r)
	if u == nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if accept && !u.EmailVerified {
		writeUnverified(w)
		return
	}
	var err error
	switch {
	case byToken && accept:
		err = h.deps.Service.AcceptByToken(r.Context(), r.PathValue("token"), u.Email, u.UUID, u.EmailVerified)
	case byToken && !accept:
		err = h.deps.Service.DeclineByToken(r.Context(), r.PathValue("token"), u.Email)
	case !byToken && accept:
		id, e := uuid.FromString(r.PathValue("id"))
		if e != nil {
			writeErr(w, http.StatusBadRequest, "bad id")
			return
		}
		err = h.deps.Service.AcceptByID(r.Context(), id, u.Email, u.UUID, u.EmailVerified)
	default:
		id, e := uuid.FromString(r.PathValue("id"))
		if e != nil {
			writeErr(w, http.StatusBadRequest, "bad id")
			return
		}
		err = h.deps.Service.DeclineByID(r.Context(), id, u.Email)
	}
	if err != nil {
		h.mapErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) listMine(w http.ResponseWriter, r *http.Request) {
	u := h.principal(r)
	if u == nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	list, err := h.deps.Service.ListMine(r.Context(), u.Email)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "could not list invitations")
		return
	}
	out := make([]map[string]any, 0, len(list))
	for _, i := range list {
		row := map[string]any{
			"id": i.ID.String(), "event_id": i.EventID.String(), "token": i.Token, "status": i.Status,
			"event_title": i.EventTitle, "inviter_name": i.InviterName,
		}
		// Omit a zero start time rather than emitting a bogus "0001-01-01".
		if !i.EventStartsAt.IsZero() {
			row["event_starts_at"] = i.EventStartsAt.Format(time.RFC3339)
		}
		out = append(out, row)
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *handler) mapErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, invdomain.ErrNotVerified):
		writeUnverified(w)
	case errors.Is(err, invdomain.ErrNotOwner):
		writeErr(w, http.StatusForbidden, "not event owner")
	case errors.Is(err, invdomain.ErrEmailMismatch):
		writeErr(w, http.StatusForbidden, "invitation addressed to another email")
	case errors.Is(err, invdomain.ErrNotPending):
		writeErr(w, http.StatusConflict, "invitation already handled")
	case errors.Is(err, invdomain.ErrNotFound):
		writeErr(w, http.StatusNotFound, "invitation not found")
	default:
		writeErr(w, http.StatusInternalServerError, "internal error")
	}
}
