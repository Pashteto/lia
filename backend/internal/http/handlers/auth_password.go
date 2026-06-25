package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/go-openapi/runtime/middleware"

	authpkg "github.com/Pashteto/lia/internal/http/auth"
	apimodels "github.com/Pashteto/lia/internal/http/models"
	authops "github.com/Pashteto/lia/internal/http/server/operations/auth"
	"github.com/Pashteto/lia/pkg/logger"
)

// Register handles POST /auth/register — password sign-up via GateGuard.
type Register struct {
	signer authpkg.Signer
}

// NewRegister creates the handler. A nil signer returns 503.
func NewRegister(signer authpkg.Signer) *Register {
	return &Register{signer: signer}
}

// Handle POST /auth/register.
func (h *Register) Handle(params authops.RegisterParams) middleware.Responder {
	if h.signer == nil {
		return authops.NewRegisterServiceUnavailable().
			WithPayload(DefaultError(http.StatusServiceUnavailable, fmt.Errorf("auth backend not configured"), nil))
	}
	if params.Body == nil || params.Body.Email == nil || *params.Body.Email == "" ||
		params.Body.Password == nil || *params.Body.Password == "" {
		return authops.NewRegisterBadRequest().
			WithPayload(DefaultError(http.StatusBadRequest, fmt.Errorf("email and password are required"), nil))
	}

	email := params.Body.Email.String()
	token, err := h.signer.SignUpPassword(params.HTTPRequest.Context(), email, params.Body.Name, *params.Body.Password)
	if err != nil {
		if errors.Is(err, authpkg.ErrUserExists) {
			return authops.NewRegisterConflict().
				WithPayload(DefaultError(http.StatusConflict, fmt.Errorf("email already registered"), nil))
		}
		logger.Log().Errorf("register: %s", err.Error())
		return authops.NewRegisterServiceUnavailable().
			WithPayload(DefaultError(http.StatusServiceUnavailable, fmt.Errorf("auth backend error"), nil))
	}

	return authops.NewRegisterOK().WithPayload(&apimodels.DemoLoginResponse{Token: &token})
}

// Login handles POST /auth/login — password sign-in via GateGuard.
type Login struct {
	signer authpkg.Signer
}

// NewLogin creates the handler. A nil signer returns 503.
func NewLogin(signer authpkg.Signer) *Login {
	return &Login{signer: signer}
}

// Handle POST /auth/login.
func (h *Login) Handle(params authops.LoginParams) middleware.Responder {
	if h.signer == nil {
		return authops.NewLoginServiceUnavailable().
			WithPayload(DefaultError(http.StatusServiceUnavailable, fmt.Errorf("auth backend not configured"), nil))
	}
	if params.Body == nil || params.Body.Email == nil || *params.Body.Email == "" ||
		params.Body.Password == nil || *params.Body.Password == "" {
		return authops.NewLoginBadRequest().
			WithPayload(DefaultError(http.StatusBadRequest, fmt.Errorf("email and password are required"), nil))
	}

	email := params.Body.Email.String()
	token, err := h.signer.SignInPassword(params.HTTPRequest.Context(), email, *params.Body.Password)
	if err != nil {
		if errors.Is(err, authpkg.ErrInvalidCredentials) {
			return authops.NewLoginUnauthorized().
				WithPayload(DefaultError(http.StatusUnauthorized, fmt.Errorf("invalid email or password"), nil))
		}
		logger.Log().Errorf("login: %s", err.Error())
		return authops.NewLoginServiceUnavailable().
			WithPayload(DefaultError(http.StatusServiceUnavailable, fmt.Errorf("auth backend error"), nil))
	}

	return authops.NewLoginOK().WithPayload(&apimodels.DemoLoginResponse{Token: &token})
}
