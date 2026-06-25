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

type stubMod struct{ moderation.Service }

func (stubMod) Overview(context.Context) (moderation.Counts, error) {
	return moderation.Counts{EventsTotal: 3, EventsPublished: 2, EventsRemoved: 1}, nil
}

func newTestHandler(role string) http.Handler {
	return NewHandler(Deps{
		Authenticate: authFn(role),
		Moderation:   stubMod{},
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
