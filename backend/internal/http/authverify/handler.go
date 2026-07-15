// Package authverify implements the public email-verification proxy endpoints
// (/auth/request-verification, /auth/verify-email) that forward to GateGuard
// via the shared auth.Signer.
package authverify

import (
	"encoding/json"
	"net/http"

	authpkg "github.com/Pashteto/lia/internal/http/auth"
)

// Deps carries the signer used to reach GateGuard.
type Deps struct {
	Signer authpkg.Signer
}

type handler struct {
	deps Deps
	mux  *http.ServeMux
}

// NewHandler builds the verify endpoints (mounted ahead of the swagger mux).
func NewHandler(deps Deps) http.Handler {
	h := &handler{deps: deps, mux: http.NewServeMux()}
	h.mux.HandleFunc("POST /auth/request-verification", h.request)
	h.mux.HandleFunc("POST /auth/verify-email", h.verify)
	return h
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) { h.mux.ServeHTTP(w, r) }

func writeErr(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": msg})
}

func (h *handler) request(w http.ResponseWriter, r *http.Request) {
	if h.deps.Signer == nil {
		writeErr(w, http.StatusServiceUnavailable, "auth backend not configured")
		return
	}
	var body struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Email == "" {
		writeErr(w, http.StatusBadRequest, "email is required")
		return
	}
	if err := h.deps.Signer.RequestEmailVerification(r.Context(), body.Email); err != nil {
		writeErr(w, http.StatusServiceUnavailable, "could not send verification")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) verify(w http.ResponseWriter, r *http.Request) {
	if h.deps.Signer == nil {
		writeErr(w, http.StatusServiceUnavailable, "auth backend not configured")
		return
	}
	var body struct {
		Email string `json:"email"`
		Code  string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Email == "" || body.Code == "" {
		writeErr(w, http.StatusBadRequest, "email and code are required")
		return
	}
	if err := h.deps.Signer.VerifyEmail(r.Context(), body.Email, body.Code); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid or expired code")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
