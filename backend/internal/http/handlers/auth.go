package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"

	authpkg "github.com/Pashteto/lia/internal/http/auth"
	apimodels "github.com/Pashteto/lia/internal/http/models"
	authops "github.com/Pashteto/lia/internal/http/server/operations/auth"
	"github.com/Pashteto/lia/pkg/logger"
)

// DemoLogin handles POST /auth/demo-login — DEMO-ONLY token minting via GateGuard
// SignInOAuth (no Google). It mints a valid session for any email with no
// password, so it MUST stay off in real production: it is gated by the same
// mock-auth switch as the JWT bypass. When disabled the route answers 404, so
// it is indistinguishable from an unregistered path.
type DemoLogin struct {
	signer  authpkg.Signer
	enabled bool
}

// NewDemoLogin creates the handler. `enabled` should track HTTP_MOCK_AUTH: the
// endpoint is a non-production control and is served only in mock-auth (dev/demo)
// mode. A nil signer (gatekeeper not configured) makes every request return 503.
func NewDemoLogin(signer authpkg.Signer, enabled bool) *DemoLogin {
	return &DemoLogin{signer: signer, enabled: enabled}
}

// Handle POST /auth/demo-login.
func (h *DemoLogin) Handle(params authops.DemoLoginParams) middleware.Responder {
	if !h.enabled {
		// In production (mock auth off) the demo-login route does not exist.
		return notFound(fmt.Errorf("path %s was not found", params.HTTPRequest.URL.Path))
	}
	if h.signer == nil {
		return authops.NewDemoLoginServiceUnavailable().
			WithPayload(DefaultError(http.StatusServiceUnavailable, fmt.Errorf("auth backend not configured"), nil))
	}
	if params.Body == nil || params.Body.Email == nil || *params.Body.Email == "" {
		return authops.NewDemoLoginBadRequest().
			WithPayload(DefaultError(http.StatusBadRequest, fmt.Errorf("email is required"), nil))
	}

	email := params.Body.Email.String()
	token, err := h.signer.SignIn(params.HTTPRequest.Context(), email, params.Body.Name)
	if err != nil {
		logger.Log().Errorf("demo-login: %s", err.Error())
		return authops.NewDemoLoginServiceUnavailable().
			WithPayload(DefaultError(http.StatusServiceUnavailable, fmt.Errorf("auth backend error"), nil))
	}

	return authops.NewDemoLoginOK().WithPayload(&apimodels.DemoLoginResponse{Token: &token})
}

// notFound writes a plain 404 JSON response matching the swagger mux's own
// not-found shape (no generated 404 responder exists for this operation).
func notFound(err error) middleware.Responder {
	payload := DefaultError(http.StatusNotFound, err, nil)
	return middleware.ResponderFunc(func(w http.ResponseWriter, _ runtime.Producer) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(payload)
	})
}
