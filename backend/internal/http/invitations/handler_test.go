package invitations_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofrs/uuid"

	httpinv "github.com/Pashteto/lia/internal/http/invitations"
	inv "github.com/Pashteto/lia/internal/invitations"
	domain "github.com/Pashteto/lia/internal/models"
)

// stubSvc implements the full inv.Service interface as no-ops, with a couple
// of fields to steer behavior for specific tests.
type stubSvc struct {
	invited int

	previewResp *inv.Preview
	previewErr  error

	listMine []inv.MineItem

	// acceptTokenCalled records whether AcceptByToken reached the service —
	// used to prove the HTTP handler no longer blocks unverified acceptors
	// before the service's own verify-on-accept logic runs.
	acceptTokenCalled bool
	acceptVerifiedArg bool
}

func (s *stubSvc) Invite(_ context.Context, _, _ uuid.UUID, _ bool, emails []string, _ string) (int, error) {
	return len(emails), nil
}
func (s *stubSvc) Preview(_ context.Context, _ string) (*inv.Preview, error) {
	return s.previewResp, s.previewErr
}
func (s *stubSvc) AcceptByToken(_ context.Context, _, _ string, _ uuid.UUID, verified bool) error {
	s.acceptTokenCalled = true
	s.acceptVerifiedArg = verified
	return nil
}
func (s *stubSvc) DeclineByToken(_ context.Context, _, _ string) error { return nil }
func (s *stubSvc) ListMine(_ context.Context, _ string) ([]inv.MineItem, error) {
	return s.listMine, nil
}
func (s *stubSvc) AcceptByID(_ context.Context, _ uuid.UUID, _ string, _ uuid.UUID, _ bool) error {
	return nil
}
func (s *stubSvc) DeclineByID(_ context.Context, _ uuid.UUID, _ string) error { return nil }

func TestInvite_RequiresVerified(t *testing.T) {
	deps := httpinv.Deps{
		Authenticate: func(string) (*domain.User, error) {
			return &domain.User{UUID: uuid.Must(uuid.NewV4()), Email: "o@x.com", EmailVerified: false}, nil
		},
		Service: &stubSvc{},
		BaseURL: "https://presence.tarski.ru",
	}
	h := httpinv.NewHandler(deps)

	req := httptest.NewRequest("POST", "/api/v1/events/"+uuid.Must(uuid.NewV4()).String()+"/invitations",
		strings.NewReader(`{"emails":["a@x.com"]}`))
	req.Header.Set("Authorization", "Bearer x")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("unverified organizer must get 403, got %d", rr.Code)
	}
}

func TestInvite_Unauthorized(t *testing.T) {
	deps := httpinv.Deps{
		Authenticate: func(string) (*domain.User, error) {
			return nil, nil
		},
		Service: &stubSvc{},
		BaseURL: "https://presence.tarski.ru",
	}
	h := httpinv.NewHandler(deps)

	req := httptest.NewRequest("POST", "/api/v1/events/"+uuid.Must(uuid.NewV4()).String()+"/invitations",
		strings.NewReader(`{"emails":["a@x.com"]}`))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("no Authorization header must get 401, got %d", rr.Code)
	}
}

// TestAcceptToken_UnverifiedUserReachesService is the regression guard for
// the Task 10 Critical finding: the handler used to reject every accept from
// an unverified user with a 403 before the service's verify-on-accept logic
// (service.accept()) ever ran. The handler must now let unverified acceptors
// through to the service — verification is the service's job, not the
// handler's — and respond 204 on success.
func TestAcceptToken_UnverifiedUserReachesService(t *testing.T) {
	svc := &stubSvc{}
	deps := httpinv.Deps{
		Authenticate: func(string) (*domain.User, error) {
			return &domain.User{UUID: uuid.Must(uuid.NewV4()), Email: "invitee@x.com", EmailVerified: false}, nil
		},
		Service: svc,
		BaseURL: "https://presence.tarski.ru",
	}
	h := httpinv.NewHandler(deps)

	req := httptest.NewRequest("POST", "/api/v1/invitations/some-token/accept", nil)
	req.Header.Set("Authorization", "Bearer x")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("unverified invitee accept must reach the service and return 204, got %d body=%s", rr.Code, rr.Body.String())
	}
	if !svc.acceptTokenCalled {
		t.Fatal("AcceptByToken was never called — handler still blocking unverified acceptors")
	}
	if svc.acceptVerifiedArg {
		t.Fatal("expected verified=false to be passed through to the service")
	}
}

func TestPreview_PublicHappyPath(t *testing.T) {
	eventID := uuid.Must(uuid.NewV4())
	deps := httpinv.Deps{
		Authenticate: func(string) (*domain.User, error) {
			t.Fatal("preview must not authenticate")
			return nil, nil
		},
		Service: &stubSvc{previewResp: &inv.Preview{EventID: eventID, EventTitle: "Party", Status: "pending"}},
		BaseURL: "https://presence.tarski.ru",
	}
	h := httpinv.NewHandler(deps)

	req := httptest.NewRequest("GET", "/api/v1/invitations/some-token", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("public preview must be 200 without auth, got %d body=%s", rr.Code, rr.Body.String())
	}
}
