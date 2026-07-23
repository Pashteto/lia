// Package geocode is the auth-gated HTTP proxy in front of the Yandex Geocoder.
// It mirrors the hand-mounted handler shape of internal/http/follows.
package geocode

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	geo "github.com/Pashteto/lia/internal/geocode"
	domain "github.com/Pashteto/lia/internal/models"
)

// Geocoder is the subset of *geo.Client the handler needs (for test injection).
type Geocoder interface {
	Geocode(ctx context.Context, q string) ([]geo.Result, error)
	SearchPlaces(ctx context.Context, q string) ([]geo.Result, error)
}

// Deps are the handler's injected dependencies.
type Deps struct {
	Authenticate func(token string) (*domain.User, error)
	Client       Geocoder
}

type handler struct {
	deps Deps
	mux  *http.ServeMux
}

// NewHandler builds the /api/v1/geocode handler.
func NewHandler(deps Deps) http.Handler {
	h := &handler{deps: deps}
	h.mux = http.NewServeMux()
	h.mux.HandleFunc("GET /api/v1/geocode", h.geocode)
	h.mux.HandleFunc("GET /api/v1/places", h.places)
	return h
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

// principal reads the Bearer token and resolves the current user, or nil.
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

func (h *handler) geocode(w http.ResponseWriter, r *http.Request) {
	if h.principal(r) == nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	results, err := h.deps.Client.Geocode(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		writeErr(w, http.StatusServiceUnavailable, "geocode_failed")
		return
	}
	writeJSON(w, http.StatusOK, results)
}

func (h *handler) places(w http.ResponseWriter, r *http.Request) {
	if h.principal(r) == nil {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	results, err := h.deps.Client.SearchPlaces(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		writeErr(w, http.StatusServiceUnavailable, "places_failed")
		return
	}
	writeJSON(w, http.StatusOK, results)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
