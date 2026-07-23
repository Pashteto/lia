package authverify

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
func (f *fakeSigner) MarkEmailVerified(_ context.Context, _ string) error           { return f.err }

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

func TestVerify_MapsGRPCCodesToJSONCodes(t *testing.T) {
	cases := []struct {
		name     string
		grpcErr  error
		wantHTTP int
		wantCode string
	}{
		{"expired", status.Error(codes.DeadlineExceeded, "x"), http.StatusBadRequest, "verification_expired"},
		{"invalid", status.Error(codes.InvalidArgument, "x"), http.StatusBadRequest, "verification_invalid"},
		{"locked", status.Error(codes.ResourceExhausted, "x"), http.StatusTooManyRequests, "verification_attempts_exceeded"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wrapped := fmt.Errorf("gateguard verify email: %w", tc.grpcErr)
			h := NewHandler(Deps{Signer: &fakeSigner{err: wrapped}})
			req := httptest.NewRequest("POST", "/auth/verify-email",
				strings.NewReader(`{"email":"a@b.c","code":"123456"}`))
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != tc.wantHTTP {
				t.Fatalf("status: want %d, got %d", tc.wantHTTP, rec.Code)
			}
			var body struct {
				Code string `json:"code"`
			}
			if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if body.Code != tc.wantCode {
				t.Fatalf("code: want %q, got %q", tc.wantCode, body.Code)
			}
		})
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
