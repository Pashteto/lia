package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofrs/uuid"

	complaintsdomain "github.com/Pashteto/lia/internal/complaints"
	domain "github.com/Pashteto/lia/internal/models"
	"github.com/Pashteto/lia/internal/moderation"
)

func authFn(role string) func(string) (*domain.User, error) {
	return func(tok string) (*domain.User, error) {
		if tok == "" {
			return nil, http.ErrNoCookie // any non-nil error → 401
		}
		return &domain.User{UUID: uuid.Must(uuid.NewV4()), Email: "u@x", Name: "U", Role: role}, nil
	}
}

type stubMod struct {
	moderation.Service
	overviewErr  error
	takedownErr  error
	reinstateErr error
}

func (s stubMod) Overview(context.Context) (moderation.Counts, error) {
	if s.overviewErr != nil {
		return moderation.Counts{}, s.overviewErr
	}
	return moderation.Counts{EventsTotal: 3, EventsPublished: 2, EventsRemoved: 1}, nil
}

func (s stubMod) Takedown(ctx context.Context, id uuid.UUID, by uuid.UUID, reason string) error {
	if s.takedownErr != nil {
		return s.takedownErr
	}
	return nil
}

func (s stubMod) Reinstate(ctx context.Context, id uuid.UUID, by uuid.UUID) error {
	if s.reinstateErr != nil {
		return s.reinstateErr
	}
	return nil
}

func newTestHandler(role string) http.Handler {
	return NewHandler(Deps{
		Authenticate: authFn(role),
		Moderation:   stubMod{},
	})
}

func newTestHandlerWithMod(role string, mod stubMod) http.Handler {
	return NewHandler(Deps{
		Authenticate: authFn(role),
		Moderation:   mod,
	})
}

func TestOverview_403ForCommon(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/overview", nil)
	r.Header.Set("Authorization", "Bearer x")
	w := httptest.NewRecorder()
	newTestHandler("common").ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

func TestOverview_200ForAdmin(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/overview", nil)
	r.Header.Set("Authorization", "Bearer x")
	w := httptest.NewRecorder()
	newTestHandler("admin").ServeHTTP(w, r)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"events_total":3`) {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestOverview_401Anon(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/overview", nil) // no header
	w := httptest.NewRecorder()
	newTestHandler("admin").ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestTakedown_400OnEmptyReason(t *testing.T) {
	id := uuid.Must(uuid.NewV4()).String()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/admin/moderation/events/"+id+"/takedown",
		strings.NewReader(`{"reason":""}`))
	r.Header.Set("Authorization", "Bearer x")
	w := httptest.NewRecorder()
	mod := stubMod{takedownErr: moderation.ErrReasonRequired}
	newTestHandlerWithMod("admin", mod).ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestTakedown_409OnInvalidTransition(t *testing.T) {
	id := uuid.Must(uuid.NewV4()).String()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/admin/moderation/events/"+id+"/takedown",
		strings.NewReader(`{"reason":"spam"}`))
	r.Header.Set("Authorization", "Bearer x")
	w := httptest.NewRecorder()
	mod := stubMod{takedownErr: moderation.ErrInvalidTransition}
	newTestHandlerWithMod("admin", mod).ServeHTTP(w, r)
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", w.Code)
	}
}

func TestReinstate_409OnInvalidTransition(t *testing.T) {
	id := uuid.Must(uuid.NewV4()).String()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/admin/moderation/events/"+id+"/reinstate", nil)
	r.Header.Set("Authorization", "Bearer x")
	w := httptest.NewRecorder()
	mod := stubMod{reinstateErr: moderation.ErrInvalidTransition}
	newTestHandlerWithMod("admin", mod).ServeHTTP(w, r)
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", w.Code)
	}
}

func TestAuthMe_401Anon(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/auth/me", nil) // no Authorization header
	w := httptest.NewRecorder()
	newTestHandler("admin").ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestAuthMe_200IncludesEmailVerified(t *testing.T) {
	h := NewHandler(Deps{
		Authenticate: func(tok string) (*domain.User, error) {
			return &domain.User{UUID: uuid.Must(uuid.NewV4()), Email: "u@x", Name: "U", Role: "common", EmailVerified: true}, nil
		},
		Moderation: stubMod{},
	})
	r := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	r.Header.Set("Authorization", "Bearer x")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	v, ok := body["email_verified"]
	if !ok {
		t.Fatalf("response missing email_verified key: %v", body)
	}
	if verified, ok := v.(bool); !ok || !verified {
		t.Fatalf("email_verified = %v, want true", v)
	}
}

type stubComplaints struct {
	complaintsdomain.Service
	resolveErr error
	openCount  int
}

func (s stubComplaints) ListInbox(context.Context) ([]complaintsdomain.EventReportGroup, error) {
	return []complaintsdomain.EventReportGroup{}, nil
}
func (s stubComplaints) TargetDetail(context.Context, string, uuid.UUID) ([]complaintsdomain.Complaint, error) {
	return nil, nil
}
func (s stubComplaints) Resolve(context.Context, string, uuid.UUID, uuid.UUID, string, string) error {
	return s.resolveErr
}
func (s stubComplaints) OpenEventCount(context.Context) (int, error) { return s.openCount, nil }

func newHandlerWithComplaints(role string, c complaintsdomain.Service) http.Handler {
	return NewHandler(Deps{Authenticate: authFn(role), Moderation: stubMod{}, Complaints: c})
}

func TestComplaints_403ForCommon(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/complaints", nil)
	r.Header.Set("Authorization", "Bearer x")
	w := httptest.NewRecorder()
	newHandlerWithComplaints("common", stubComplaints{}).ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

func TestResolve_400OnResolutionRequired(t *testing.T) {
	id := uuid.Must(uuid.NewV4()).String()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/admin/complaints/events/"+id+"/resolve",
		strings.NewReader(`{"action":"takedown","resolution":""}`))
	r.Header.Set("Authorization", "Bearer x")
	w := httptest.NewRecorder()
	c := stubComplaints{resolveErr: complaintsdomain.ErrResolutionRequired}
	newHandlerWithComplaints("admin", c).ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestResolve_409OnInvalidTransition(t *testing.T) {
	id := uuid.Must(uuid.NewV4()).String()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/admin/complaints/events/"+id+"/resolve",
		strings.NewReader(`{"action":"takedown","resolution":"scam"}`))
	r.Header.Set("Authorization", "Bearer x")
	w := httptest.NewRecorder()
	c := stubComplaints{resolveErr: moderation.ErrInvalidTransition}
	newHandlerWithComplaints("admin", c).ServeHTTP(w, r)
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", w.Code)
	}
}

func TestResolve_200OK(t *testing.T) {
	id := uuid.Must(uuid.NewV4()).String()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/admin/complaints/events/"+id+"/resolve",
		strings.NewReader(`{"action":"dismiss"}`))
	r.Header.Set("Authorization", "Bearer x")
	w := httptest.NewRecorder()
	newHandlerWithComplaints("admin", stubComplaints{}).ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestOverview_IncludesComplaintsOpen(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/overview", nil)
	r.Header.Set("Authorization", "Bearer x")
	w := httptest.NewRecorder()
	newHandlerWithComplaints("admin", stubComplaints{openCount: 4}).ServeHTTP(w, r)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"complaints_open":4`) {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}
