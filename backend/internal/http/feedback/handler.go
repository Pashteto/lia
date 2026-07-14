// Package feedback provides the post-event feedback HTTP surface, mounted ahead
// of the swagger mux in internal/http/module.go (mirrors internal/http/complaints).
package feedback

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/gofrs/uuid"

	fbdomain "github.com/Pashteto/lia/internal/feedback"
	domain "github.com/Pashteto/lia/internal/models"
)

// Deps are the collaborators the handler needs.
type Deps struct {
	Authenticate func(token string) (*domain.User, error)
	Feedback     fbdomain.Service
}

type handler struct {
	deps Deps
	mux  *http.ServeMux
}

// NewHandler returns the mounted feedback handler.
func NewHandler(deps Deps) http.Handler {
	h := &handler{deps: deps, mux: http.NewServeMux()}
	h.mux.HandleFunc("POST /api/v1/events/{id}/feedback", h.submit)
	h.mux.HandleFunc("GET /api/v1/events/{id}/feedback", h.list)
	h.mux.HandleFunc("GET /api/v1/me/feedback", h.mine)
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
	eventID, err := uuid.FromString(r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	var body struct {
		Rating  int    `json:"rating"`
		Comment string `json:"comment"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	err = h.deps.Feedback.Submit(r.Context(), u.UUID, eventID, body.Rating, body.Comment)
	switch {
	case err == nil:
		writeJSON(w, http.StatusCreated, map[string]string{"status": "received"})
	case errors.Is(err, fbdomain.ErrInvalidRating):
		writeErr(w, http.StatusUnprocessableEntity, "Оценка должна быть от 1 до 5")
	case errors.Is(err, fbdomain.ErrNotEnded):
		writeErr(w, http.StatusUnprocessableEntity, "Отзыв можно оставить после завершения события")
	case errors.Is(err, fbdomain.ErrNotParticipant):
		writeErr(w, http.StatusForbidden, "Отзыв могут оставить только участники")
	case errors.Is(err, fbdomain.ErrAlreadySubmitted):
		writeErr(w, http.StatusConflict, "Вы уже оставили отзыв")
	case errors.Is(err, fbdomain.ErrNotFound):
		writeErr(w, http.StatusNotFound, "Событие не найдено")
	default:
		writeErr(w, http.StatusInternalServerError, "submit failed")
	}
}

func (h *handler) list(w http.ResponseWriter, r *http.Request) {
	u := h.principal(r)
	if u == nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	eventID, err := uuid.FromString(r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return
	}
	isAdmin := u.Role == "admin"
	sum, err := h.deps.Feedback.ForOwner(r.Context(), eventID, u.UUID, isAdmin)
	switch {
	case err == nil:
		writeJSON(w, http.StatusOK, sum)
	case errors.Is(err, fbdomain.ErrForbidden):
		writeErr(w, http.StatusForbidden, "Недостаточно прав")
	case errors.Is(err, fbdomain.ErrNotFound):
		writeErr(w, http.StatusNotFound, "Событие не найдено")
	default:
		writeErr(w, http.StatusInternalServerError, "list failed")
	}
}

func (h *handler) mine(w http.ResponseWriter, r *http.Request) {
	u := h.principal(r)
	if u == nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	eventID, err := uuid.FromString(r.URL.Query().Get("event_id"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid event_id")
		return
	}
	submitted, err := h.deps.Feedback.MyFeedback(r.Context(), u.UUID, eventID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "lookup failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"submitted": submitted})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
