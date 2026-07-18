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
}

func (s *stubSvc) Invite(_ context.Context, _, _ uuid.UUID, _ bool, emails []string, _ string) (int, error) {
	return len(emails), nil
}
func (s *stubSvc) Preview(_ context.Context, _ string) (*inv.Preview, error) {
	return s.previewResp, s.previewErr
}
func (s *stubSvc) AcceptByToken(_ context.Context, _, _ string, _ uuid.UUID, _ bool) error {
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
