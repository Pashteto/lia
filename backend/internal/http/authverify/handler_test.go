package authverify

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeSigner struct {
	err error
}

func (f *fakeSigner) SignIn(_ context.Context, _, _ string) (string, error) { return "", nil }
func (f *fakeSigner) SignUpPassword(_ context.Context, _, _, _ string) (string, error) {
	return "", nil
}
func (f *fakeSigner) SignInPassword(_ context.Context, _, _ string) (string, error) { return "", nil }
func (f *fakeSigner) RequestEmailVerification(_ context.Context, _ string) error    { return f.err }
func (f *fakeSigner) VerifyEmail(_ context.Context, _, _ string) error              { return f.err }

func TestRequestVerification_MissingEmail_Returns400(t *testing.T) {
	h := NewHandler(Deps{Signer: &fakeSigner{}})
	req := httptest.NewRequest(http.MethodPost, "/auth/request-verification", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestRequestVerification_NilSigner_Returns503(t *testing.T) {
	h := NewHandler(Deps{Signer: nil})
	req := httptest.NewRequest(http.MethodPost, "/auth/request-verification", strings.NewReader(`{"email":"a@b.com"}`))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}
}

func TestRequestVerification_Success_Returns204(t *testing.T) {
	h := NewHandler(Deps{Signer: &fakeSigner{}})
	req := httptest.NewRequest(http.MethodPost, "/auth/request-verification", strings.NewReader(`{"email":"a@b.com"}`))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
}

func TestVerifyEmail_MissingCode_Returns400(t *testing.T) {
	h := NewHandler(Deps{Signer: &fakeSigner{}})
	req := httptest.NewRequest(http.MethodPost, "/auth/verify-email", strings.NewReader(`{"email":"a@b.com"}`))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestVerifyEmail_SignerError_Returns400(t *testing.T) {
	h := NewHandler(Deps{Signer: &fakeSigner{err: context.DeadlineExceeded}})
	req := httptest.NewRequest(http.MethodPost, "/auth/verify-email", strings.NewReader(`{"email":"a@b.com","code":"123456"}`))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestVerifyEmail_Success_Returns204(t *testing.T) {
	h := NewHandler(Deps{Signer: &fakeSigner{}})
	req := httptest.NewRequest(http.MethodPost, "/auth/verify-email", strings.NewReader(`{"email":"a@b.com","code":"123456"}`))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
}
