package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofrs/uuid"

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
