package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-openapi/runtime"

	"github.com/Pashteto/lia/internal/http/handlers"
	apimodels "github.com/Pashteto/lia/internal/http/models"
)

func TestIsVerified(t *testing.T) {
	if handlers.IsVerified(nil) {
		t.Fatal("nil principal must be unverified")
	}
	if handlers.IsVerified(&apimodels.User{}) {
		t.Fatal("zero-value principal must be unverified")
	}
	v := true
	if !handlers.IsVerified(&apimodels.User{EmailVerified: v}) {
		t.Fatal("verified principal must pass")
	}
}

func TestUnverifiedResponder(t *testing.T) {
	rr := httptest.NewRecorder()
	handlers.UnverifiedResponder().WriteResponse(rr, runtime.JSONProducer())
	if rr.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "email_not_verified") {
		t.Fatalf("body missing code: %s", rr.Body.String())
	}
}
