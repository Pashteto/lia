package handlers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-openapi/strfmt"

	authpkg "github.com/Pashteto/lia/internal/http/auth"
	"github.com/Pashteto/lia/internal/http/models"
	authops "github.com/Pashteto/lia/internal/http/server/operations/auth"
)

type fakeSigner struct {
	token string
	err   error
}

func (f *fakeSigner) SignIn(_ context.Context, _, _ string) (string, error) {
	return f.token, f.err
}
func (f *fakeSigner) SignUpPassword(_ context.Context, _, _, _ string) (string, error) {
	return f.token, f.err
}
func (f *fakeSigner) SignInPassword(_ context.Context, _, _ string) (string, error) {
	return f.token, f.err
}
func (f *fakeSigner) RequestEmailVerification(_ context.Context, _ string) error {
	return f.err
}
func (f *fakeSigner) VerifyEmail(_ context.Context, _, _ string) error {
	return f.err
}
func (f *fakeSigner) MarkEmailVerified(_ context.Context, _ string) error {
	return f.err
}

var _ authpkg.Signer = (*fakeSigner)(nil)

func TestDemoLogin_ReturnsToken(t *testing.T) {
	h := NewDemoLogin(&fakeSigner{token: "jwt-xyz"}, true)
	email := strfmt.Email("a@b.com")
	params := authops.DemoLoginParams{Body: &models.DemoLoginInput{Email: &email, Name: "A"}}
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/demo-login", nil)
	params.HTTPRequest = req

	resp := h.Handle(params)
	ok, isOK := resp.(*authops.DemoLoginOK)
	if !isOK {
		t.Fatalf("expected DemoLoginOK, got %T", resp)
	}
	if ok.Payload == nil || ok.Payload.Token == nil || *ok.Payload.Token != "jwt-xyz" {
		t.Errorf("token mismatch: %+v", ok.Payload)
	}
}

func TestDemoLogin_SignerError(t *testing.T) {
	h := NewDemoLogin(&fakeSigner{err: fmt.Errorf("gateguard down")}, true)
	email := strfmt.Email("a@b.com")
	params := authops.DemoLoginParams{Body: &models.DemoLoginInput{Email: &email}}
	req, _ := http.NewRequest(http.MethodPost, "/", nil)
	params.HTTPRequest = req

	if _, isUnavail := h.Handle(params).(*authops.DemoLoginServiceUnavailable); !isUnavail {
		t.Error("expected DemoLoginServiceUnavailable on signer error")
	}
}

func TestDemoLogin_NoSigner(t *testing.T) {
	h := NewDemoLogin(nil, true) // gatekeeper not configured
	email := strfmt.Email("a@b.com")
	params := authops.DemoLoginParams{Body: &models.DemoLoginInput{Email: &email}}
	req, _ := http.NewRequest(http.MethodPost, "/", nil)
	params.HTTPRequest = req

	if _, isUnavail := h.Handle(params).(*authops.DemoLoginServiceUnavailable); !isUnavail {
		t.Error("expected DemoLoginServiceUnavailable when no signer is configured")
	}
}

// In production (mock auth off) the demo-login route must not mint tokens; it
// answers 404 as if unregistered — even with a working signer.
func TestDemoLogin_DisabledReturns404(t *testing.T) {
	h := NewDemoLogin(&fakeSigner{token: "jwt-xyz"}, false)
	email := strfmt.Email("a@b.com")
	params := authops.DemoLoginParams{Body: &models.DemoLoginInput{Email: &email, Name: "A"}}
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/demo-login", nil)
	params.HTTPRequest = req

	resp := h.Handle(params)
	if _, isOK := resp.(*authops.DemoLoginOK); isOK {
		t.Fatal("disabled demo-login must NOT return a token")
	}
	rec := httptest.NewRecorder()
	resp.WriteResponse(rec, nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 when demo-login disabled, got %d", rec.Code)
	}
}
