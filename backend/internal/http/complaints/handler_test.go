package complaints

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofrs/uuid"

	complaintsdomain "github.com/Pashteto/lia/internal/complaints"
	domain "github.com/Pashteto/lia/internal/models"
)

func authFn(ok bool) func(string) (*domain.User, error) {
	return func(tok string) (*domain.User, error) {
		if !ok || tok == "" {
			return nil, http.ErrNoCookie
		}
		return &domain.User{UUID: uuid.Must(uuid.NewV4()), Email: "u@x", Role: "common", EmailVerified: true}, nil
	}
}

// stubService implements complaintsdomain.Service; only Submit is exercised here.
type stubService struct {
	created bool
	err     error
}

func (s stubService) Submit(_ context.Context, _ uuid.UUID, _ string, _ uuid.UUID, _, _ string) (bool, error) {
	return s.created, s.err
}
func (s stubService) ListInbox(context.Context) ([]complaintsdomain.EventReportGroup, error) {
	return nil, nil
}
func (s stubService) TargetDetail(context.Context, string, uuid.UUID) ([]complaintsdomain.Complaint, error) {
	return nil, nil
}
func (s stubService) Resolve(context.Context, string, uuid.UUID, uuid.UUID, string, string) error {
	return nil
}
func (s stubService) OpenEventCount(context.Context) (int, error) { return 0, nil }

func newH(authOK bool, svc stubService) http.Handler {
	return NewHandler(Deps{Authenticate: authFn(authOK), Complaints: svc})
}

func req(t *testing.T, authOK bool, svc stubService, body string) *httptest.ResponseRecorder {
	t.Helper()
	id := uuid.Must(uuid.NewV4()).String()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/events/"+id+"/complaints", strings.NewReader(body))
	if authOK {
		r.Header.Set("Authorization", "Bearer x")
	}
	w := httptest.NewRecorder()
	newH(authOK, svc).ServeHTTP(w, r)
	return w
}

func TestSubmit_401Anon(t *testing.T) {
	if w := req(t, false, stubService{}, `{"category":"spam"}`); w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestSubmit_201Created(t *testing.T) {
	if w := req(t, true, stubService{created: true}, `{"category":"spam"}`); w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", w.Code)
	}
}

func TestSubmit_200Idempotent(t *testing.T) {
	if w := req(t, true, stubService{created: false}, `{"category":"spam"}`); w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestSubmit_400BadCategory(t *testing.T) {
	if w := req(t, true, stubService{err: complaintsdomain.ErrInvalidCategory}, `{"category":"x"}`); w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestSubmit_404NoEvent(t *testing.T) {
	if w := req(t, true, stubService{err: complaintsdomain.ErrTargetNotFound}, `{"category":"spam"}`); w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestSubmit_403Unverified(t *testing.T) {
	id := uuid.Must(uuid.NewV4()).String()
	deps := Deps{
		Authenticate: func(tok string) (*domain.User, error) {
			return &domain.User{UUID: uuid.Must(uuid.NewV4()), Email: "u@x", Role: "common", EmailVerified: false}, nil
		},
		Complaints: stubService{created: true},
	}
	h := NewHandler(deps)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/events/"+id+"/complaints", strings.NewReader(`{"category":"spam"}`))
	r.Header.Set("Authorization", "Bearer x")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
	if !strings.Contains(w.Body.String(), "email_not_verified") {
		t.Fatalf("body missing code: %s", w.Body.String())
	}
}
