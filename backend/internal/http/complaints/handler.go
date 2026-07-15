// Package complaints provides the public (authed) submit handler for filing a
// report against an event: POST /api/v1/events/{id}/complaints. Mounted ahead
// of the go-swagger mux in internal/http/module.go (mirrors internal/http/organizers).
package complaints

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/gofrs/uuid"

	complaintsdomain "github.com/Pashteto/lia/internal/complaints"
	domain "github.com/Pashteto/lia/internal/models"
)

// Deps are the collaborators the handler needs.
type Deps struct {
	Authenticate func(token string) (*domain.User, error)
	Complaints   complaintsdomain.Service
}

type handler struct {
	deps Deps
	mux  *http.ServeMux
}

// NewHandler returns the mounted public complaints handler.
func NewHandler(deps Deps) http.Handler {
	h := &handler{deps: deps, mux: http.NewServeMux()}
	h.mux.HandleFunc("POST /api/v1/events/{id}/complaints", h.submit)
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

func (h *handler) submit(w http.ResponseWriter, r *http.Request) {
	u := h.principal(r)
	if u == nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !u.EmailVerified {
		writeJSON(w, http.StatusForbidden, map[string]string{
			"code":    "email_not_verified",
			"message": "Подтвердите электронную почту, чтобы продолжить",
		})
		return
	}
	eventID, err := uuid.FromString(r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	var body struct {
		Category string `json:"category"`
		Note     string `json:"note"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	created, err := h.deps.Complaints.Submit(r.Context(), u.UUID, "event", eventID, body.Category, body.Note)
	switch {
	case err == nil && created:
		writeJSON(w, http.StatusCreated, map[string]string{"status": "received"})
	case err == nil && !created:
		// Idempotent repeat of an already-open complaint.
		writeJSON(w, http.StatusOK, map[string]string{"status": "received"})
	case errors.Is(err, complaintsdomain.ErrInvalidCategory):
		writeErr(w, http.StatusBadRequest, "Некорректная категория жалобы")
	case errors.Is(err, complaintsdomain.ErrTargetNotFound):
		writeErr(w, http.StatusNotFound, "Событие не найдено")
	default:
		writeErr(w, http.StatusInternalServerError, "submit failed")
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
