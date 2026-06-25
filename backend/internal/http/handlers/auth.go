package handlers

import (
	"fmt"
	"net/http"

	"github.com/go-openapi/runtime/middleware"

	authpkg "github.com/Pashteto/lia/internal/http/auth"
	apimodels "github.com/Pashteto/lia/internal/http/models"
	authops "github.com/Pashteto/lia/internal/http/server/operations/auth"
	"github.com/Pashteto/lia/pkg/logger"
)

// DemoLogin handles POST /auth/demo-login — DEMO-ONLY token minting via GateGuard
// SignInOAuth (no Google). Never enable in real production.
type DemoLogin struct {
	signer authpkg.Signer
}

// NewDemoLogin creates the handler. A nil signer (gatekeeper not configured)
// makes every request return 503.
func NewDemoLogin(signer authpkg.Signer) *DemoLogin {
	return &DemoLogin{signer: signer}
}

// Handle POST /auth/demo-login.
func (h *DemoLogin) Handle(params authops.DemoLoginParams) middleware.Responder {
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
