package platformspublic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofrs/uuid"

	"github.com/Pashteto/lia/internal/platforms"
)

// fakeService implements platforms.Service against a fixed set of rows, for
// handler tests that don't need real persistence.
type fakeService struct {
	active []*platforms.TrustedPlatform
}

func (f *fakeService) Check(context.Context, string) (string, bool, error) { return "", false, nil }
func (f *fakeService) ListActive(context.Context) ([]*platforms.TrustedPlatform, error) {
	return f.active, nil
}
func (f *fakeService) List(context.Context) ([]*platforms.TrustedPlatform, error) {
	return f.active, nil
}
func (f *fakeService) Add(context.Context, string, string, string) (*platforms.TrustedPlatform, error) {
	return nil, nil
}
func (f *fakeService) Deactivate(context.Context, uuid.UUID) error { return nil }

func TestHandlerListsActivePlatforms(t *testing.T) {
	svc := &fakeService{active: []*platforms.TrustedPlatform{
		{DomainSuffix: "timepad.ru", DisplayName: "TimePad", Category: "ticketing"},
		{DomainSuffix: "vk.com", DisplayName: "VK", Category: "social"},
	}}
	h := NewHandler(Deps{Platforms: svc})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/trusted-platforms", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200", rr.Code)
	}
	var got []row
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d rows, want 2", len(got))
	}
	if got[0].DomainSuffix != "timepad.ru" || got[0].DisplayName != "TimePad" || got[0].Category != "ticketing" {
		t.Fatalf("got[0] = %+v", got[0])
	}
}

func TestHandlerRejectsNonGet(t *testing.T) {
	h := NewHandler(Deps{Platforms: &fakeService{}})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trusted-platforms", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("code = %d, want 405", rr.Code)
	}
}
