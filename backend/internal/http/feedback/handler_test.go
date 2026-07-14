package feedback

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofrs/uuid"

	fbdomain "github.com/Pashteto/lia/internal/feedback"
	domain "github.com/Pashteto/lia/internal/models"
)

func authFn(ok bool) func(string) (*domain.User, error) {
	return func(tok string) (*domain.User, error) {
		if !ok || tok == "" {
			return nil, http.ErrNoCookie
		}
		return &domain.User{UUID: uuid.Must(uuid.NewV4()), Email: "u@x", Role: "common"}, nil
	}
}

// stubService implements fbdomain.Service.
type stubService struct {
	submitErr    error
	forOwnerSum  fbdomain.Summary
	forOwnerErr  error
	myFeedback   bool
	myFeedbackEr error
}

func (s stubService) Submit(_ context.Context, _, _ uuid.UUID, _ int, _ string) error {
	return s.submitErr
}

func (s stubService) ForOwner(_ context.Context, _, _ uuid.UUID, _ bool) (fbdomain.Summary, error) {
	return s.forOwnerSum, s.forOwnerErr
}

func (s stubService) MyFeedback(_ context.Context, _, _ uuid.UUID) (bool, error) {
	return s.myFeedback, s.myFeedbackEr
}

func newH(authOK bool, svc stubService) http.Handler {
	return NewHandler(Deps{Authenticate: authFn(authOK), Feedback: svc})
}

func submitReq(t *testing.T, authOK bool, svc stubService, body string) *httptest.ResponseRecorder {
	t.Helper()
	id := uuid.Must(uuid.NewV4()).String()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/events/"+id+"/feedback", strings.NewReader(body))
	if authOK {
		r.Header.Set("Authorization", "Bearer x")
	}
	w := httptest.NewRecorder()
	newH(authOK, svc).ServeHTTP(w, r)
	return w
}

func listReq(t *testing.T, authOK bool, svc stubService) *httptest.ResponseRecorder {
	t.Helper()
	id := uuid.Must(uuid.NewV4()).String()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/events/"+id+"/feedback", nil)
	if authOK {
		r.Header.Set("Authorization", "Bearer x")
	}
	w := httptest.NewRecorder()
	newH(authOK, svc).ServeHTTP(w, r)
	return w
}

func TestSubmit_401Anon(t *testing.T) {
	if w := submitReq(t, false, stubService{}, `{"rating":5}`); w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestSubmit_422NotEnded(t *testing.T) {
	if w := submitReq(t, true, stubService{submitErr: fbdomain.ErrNotEnded}, `{"rating":5}`); w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", w.Code)
	}
}

func TestSubmit_403NotParticipant(t *testing.T) {
	if w := submitReq(t, true, stubService{submitErr: fbdomain.ErrNotParticipant}, `{"rating":5}`); w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

func TestSubmit_409AlreadySubmitted(t *testing.T) {
	if w := submitReq(t, true, stubService{submitErr: fbdomain.ErrAlreadySubmitted}, `{"rating":5}`); w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", w.Code)
	}
}

func TestSubmit_201Created(t *testing.T) {
	if w := submitReq(t, true, stubService{}, `{"rating":5,"comment":"great"}`); w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", w.Code)
	}
}

func TestList_200Owner(t *testing.T) {
	sum := fbdomain.Summary{Average: 4.5, Count: 2}
	w := listReq(t, true, stubService{forOwnerSum: sum})
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestList_403Forbidden(t *testing.T) {
	if w := listReq(t, true, stubService{forOwnerErr: fbdomain.ErrForbidden}); w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}
