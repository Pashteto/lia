package admin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	adminusers "github.com/Pashteto/lia/internal/adminusers"
	complaintsdomain "github.com/Pashteto/lia/internal/complaints"
	eventsdomain "github.com/Pashteto/lia/internal/events"
	hygiene "github.com/Pashteto/lia/internal/hygiene"
	domain "github.com/Pashteto/lia/internal/models"
	"github.com/Pashteto/lia/internal/moderation"
	"github.com/Pashteto/lia/internal/organizers"
	platformsdomain "github.com/Pashteto/lia/internal/platforms"
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
	approveErr   error
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

func (s stubMod) Approve(ctx context.Context, id uuid.UUID, by uuid.UUID) error {
	if s.approveErr != nil {
		return s.approveErr
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

func TestApprove_204OnSuccess(t *testing.T) {
	id := uuid.Must(uuid.NewV4()).String()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/admin/moderation/events/"+id+"/approve", nil)
	r.Header.Set("Authorization", "Bearer x")
	w := httptest.NewRecorder()
	newTestHandlerWithMod("admin", stubMod{}).ServeHTTP(w, r)
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", w.Code)
	}
}

func TestApprove_409OnInvalidTransition(t *testing.T) {
	id := uuid.Must(uuid.NewV4()).String()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/admin/moderation/events/"+id+"/approve", nil)
	r.Header.Set("Authorization", "Bearer x")
	w := httptest.NewRecorder()
	mod := stubMod{approveErr: moderation.ErrInvalidTransition}
	newTestHandlerWithMod("admin", mod).ServeHTTP(w, r)
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", w.Code)
	}
}

func TestApprove_403ForCommon(t *testing.T) {
	id := uuid.Must(uuid.NewV4()).String()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/admin/moderation/events/"+id+"/approve", nil)
	r.Header.Set("Authorization", "Bearer x")
	w := httptest.NewRecorder()
	newTestHandlerWithMod("common", stubMod{}).ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
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

type stubUsers struct {
	got  adminusers.Filter
	rows []adminusers.Row
}

func (s *stubUsers) List(_ context.Context, f adminusers.Filter) ([]adminusers.Row, error) {
	s.got = f
	return s.rows, nil
}

func TestAdminUsers_403ForCommon(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	r.Header.Set("Authorization", "Bearer x")
	w := httptest.NewRecorder()
	NewHandler(Deps{Authenticate: authFn("common"), Users: &stubUsers{}}).ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

func TestAdminUsers_200Shape(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	users := &stubUsers{rows: []adminusers.Row{{
		ID: id, Name: "Анна", Email: "anna@example.com", Role: "common",
		CreatedAt: time.Date(2026, 3, 4, 10, 0, 0, 0, time.UTC), Bookings: 14, IsOrganizer: false,
	}}}
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users?q=%20анна%20&limit=7&offset=3", nil)
	r.Header.Set("Authorization", "Bearer x")
	w := httptest.NewRecorder()
	NewHandler(Deps{Authenticate: authFn("admin"), Users: users}).ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var got []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 1 || got[0]["id"] != id.String() || got[0]["bookings"].(float64) != 14 {
		t.Fatalf("body = %v", got)
	}
	if got[0]["created_at"] != "2026-03-04T10:00:00Z" {
		t.Fatalf("created_at = %v, want RFC3339 UTC", got[0]["created_at"])
	}
	if users.got.Limit != 7 || users.got.Offset != 3 || users.got.Query != " анна " {
		t.Fatalf("filter = %+v (query is trimmed by the service, not the handler)", users.got)
	}
}

func TestAdminUsers_503WhenUnwired(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	r.Header.Set("Authorization", "Bearer x")
	w := httptest.NewRecorder()
	NewHandler(Deps{Authenticate: authFn("admin")}).ServeHTTP(w, r)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
}

type stubHygiene struct {
	issues  []hygiene.Issue
	result  hygiene.Result
	hideErr error
	actor   uuid.UUID
}

func (s *stubHygiene) List(context.Context) ([]hygiene.Issue, error) { return s.issues, nil }

func (s *stubHygiene) HideAll(_ context.Context, actorID uuid.UUID) (hygiene.Result, error) {
	s.actor = actorID
	return s.result, s.hideErr
}

func TestHygiene_403ForCommon(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/hygiene", nil)
	r.Header.Set("Authorization", "Bearer x")
	w := httptest.NewRecorder()
	NewHandler(Deps{Authenticate: authFn("common"), Hygiene: &stubHygiene{}}).ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

func TestHygiene_200ShapeIncludesCount(t *testing.T) {
	eid := uuid.Must(uuid.NewV4())
	stub := &stubHygiene{issues: []hygiene.Issue{{
		Kind: hygiene.KindTestData, EventID: eid, Title: "QA Тур", OrganizerName: "QA Block8",
	}}}
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/hygiene", nil)
	r.Header.Set("Authorization", "Bearer x")
	w := httptest.NewRecorder()
	NewHandler(Deps{Authenticate: authFn("admin"), Hygiene: stub}).ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var body struct {
		Issues []map[string]any `json:"issues"`
		Count  int              `json:"count"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Count != 1 || len(body.Issues) != 1 || body.Issues[0]["kind"] != "test_data" {
		t.Fatalf("body = %+v", body)
	}
	if body.Issues[0]["event_id"] != eid.String() {
		t.Fatalf("event_id = %v", body.Issues[0]["event_id"])
	}
}

func TestHygieneHideAll_200ReturnsCounts(t *testing.T) {
	stub := &stubHygiene{result: hygiene.Result{Hidden: 3, Skipped: 1}}
	r := httptest.NewRequest(http.MethodPost, "/api/v1/admin/hygiene/hide-all", nil)
	r.Header.Set("Authorization", "Bearer x")
	w := httptest.NewRecorder()
	NewHandler(Deps{Authenticate: authFn("admin"), Hygiene: stub}).ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var body map[string]int
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["hidden"] != 3 || body["skipped"] != 1 {
		t.Fatalf("body = %v", body)
	}
	if stub.actor == uuid.Nil {
		t.Fatalf("actor id not passed through")
	}
}

func TestHygieneHideAll_403ForCommon(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/api/v1/admin/hygiene/hide-all", nil)
	r.Header.Set("Authorization", "Bearer x")
	w := httptest.NewRecorder()
	NewHandler(Deps{Authenticate: authFn("common"), Hygiene: &stubHygiene{}}).ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

func TestHygiene_503WhenUnwired(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/hygiene", nil)
	r.Header.Set("Authorization", "Bearer x")
	w := httptest.NewRecorder()
	NewHandler(Deps{Authenticate: authFn("admin")}).ServeHTTP(w, r)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
}

// stubEvents serves a fixed list from the moderation queue endpoint.
type stubEvents struct {
	eventsdomain.Service
	list []*domain.Event
}

func (s stubEvents) List(context.Context, string, *time.Time, *time.Time, *uuid.UUID) ([]*domain.Event, error) {
	return s.list, nil
}

// The moderation detail labels the timestamp «подано», but the payload carried
// only starts_at — when the event happens, not when it was submitted. Publishing
// is the submission moment for a post-hoc queue, so published_at has to travel.
func TestListEvents_CarriesPublishedAt(t *testing.T) {
	published := time.Date(2026, 7, 5, 9, 0, 0, 0, time.UTC)
	h := NewHandler(Deps{
		Authenticate: authFn("admin"),
		Moderation:   stubMod{},
		Events: stubEvents{list: []*domain.Event{
			{ID: uuid.Must(uuid.NewV4()), Title: "С датой", StatusSQL: "published",
				StartsAt: time.Date(2026, 8, 15, 9, 0, 0, 0, time.UTC), PublishedAt: &published},
			{ID: uuid.Must(uuid.NewV4()), Title: "Без даты", StatusSQL: "published",
				StartsAt: time.Date(2026, 8, 15, 9, 0, 0, 0, time.UTC)},
		}},
	})
	req := httptest.NewRequest("GET", "/api/v1/admin/moderation/events", nil)
	req.Header.Set("Authorization", "Bearer t")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var got []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0]["published_at"] != "2026-07-05T09:00:00Z" {
		t.Fatalf("published_at = %v, want RFC3339 UTC", got[0]["published_at"])
	}
	if _, ok := got[1]["published_at"]; ok {
		t.Fatalf("unpublished event must omit published_at, got %v", got[1]["published_at"])
	}
}

// stubOrgs serves organizer counts for the overview tiles.
type stubOrgs struct {
	organizers.Service
	counts organizers.Counts
	orgs   []organizers.Organizer
}

func (s stubOrgs) Overview(context.Context) (organizers.Counts, error) { return s.counts, nil }

func (s *stubUsers) Count(context.Context) (int, error) { return len(s.rows), nil }

// «ОРГАНИЗАТОРОВ —» and «ПОЛЬЗОВАТЕЛЕЙ —» were permanently blank on the admin
// overview: the payload carried neither total, despite 28 users in the DB.
func TestOverview_IncludesOrganizerAndUserTotals(t *testing.T) {
	users := &stubUsers{rows: []adminusers.Row{
		{ID: uuid.Must(uuid.NewV4())}, {ID: uuid.Must(uuid.NewV4())}, {ID: uuid.Must(uuid.NewV4())},
	}}
	h := NewHandler(Deps{
		Authenticate: authFn("admin"),
		Moderation:   stubMod{},
		Organizers:   stubOrgs{counts: organizers.Counts{OrganizersPending: 2, OrganizersTotal: 9}},
		Users:        users,
	})
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/overview", nil)
	r.Header.Set("Authorization", "Bearer x")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var got map[string]int
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got["organizers_total"] != 9 {
		t.Fatalf("organizers_total = %d, want 9", got["organizers_total"])
	}
	if got["users_total"] != 3 {
		t.Fatalf("users_total = %d, want 3", got["users_total"])
	}
	if got["organizers_pending"] != 2 {
		t.Fatalf("organizers_pending = %d, want 2 (existing field must survive)", got["organizers_pending"])
	}
}

// The overview must still answer when the optional collaborators are unwired.
func TestOverview_OmitsTotalsWhenUnwired(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/overview", nil)
	r.Header.Set("Authorization", "Bearer x")
	w := httptest.NewRecorder()
	newTestHandler("admin").ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var got map[string]int
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := got["users_total"]; ok {
		t.Fatalf("users_total must be absent when Users is unwired, got %v", got["users_total"])
	}
}

type stubPlatforms struct {
	platformsdomain.Service
	rows     []*platformsdomain.TrustedPlatform
	addErr   error
	deactErr error
	added    *platformsdomain.TrustedPlatform
	deactID  uuid.UUID
}

func (s *stubPlatforms) List(context.Context) ([]*platformsdomain.TrustedPlatform, error) {
	return s.rows, nil
}

func (s *stubPlatforms) Add(_ context.Context, domainSuffix, displayName, category string) (*platformsdomain.TrustedPlatform, error) {
	if s.addErr != nil {
		return nil, s.addErr
	}
	p := &platformsdomain.TrustedPlatform{
		ID: uuid.Must(uuid.NewV4()), DomainSuffix: domainSuffix, DisplayName: displayName,
		Category: category, IsActive: true,
	}
	s.added = p
	return p, nil
}

func (s *stubPlatforms) Deactivate(_ context.Context, id uuid.UUID) error {
	s.deactID = id
	return s.deactErr
}

func newHandlerWithPlatforms(role string, p platformsdomain.Service) http.Handler {
	return NewHandler(Deps{Authenticate: authFn(role), Moderation: stubMod{}, Platforms: p})
}

func TestTrustedPlatforms_403ForCommon(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/trusted-platforms", nil)
	r.Header.Set("Authorization", "Bearer x")
	w := httptest.NewRecorder()
	newHandlerWithPlatforms("common", &stubPlatforms{}).ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

func TestTrustedPlatforms_200ListsRows(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	stub := &stubPlatforms{rows: []*platformsdomain.TrustedPlatform{
		{ID: id, DomainSuffix: "timepad.ru", DisplayName: "Timepad", Category: "ticketing", IsActive: true},
	}}
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/trusted-platforms", nil)
	r.Header.Set("Authorization", "Bearer x")
	w := httptest.NewRecorder()
	newHandlerWithPlatforms("admin", stub).ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var got []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 1 || got[0]["domain_suffix"] != "timepad.ru" || got[0]["is_active"] != true {
		t.Fatalf("body = %v", got)
	}
}

func TestTrustedPlatforms_503WhenUnwired(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/trusted-platforms", nil)
	r.Header.Set("Authorization", "Bearer x")
	w := httptest.NewRecorder()
	newTestHandler("admin").ServeHTTP(w, r)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", w.Code)
	}
}

func TestTrustedPlatforms_POST_422OnInvalidCategory(t *testing.T) {
	stub := &stubPlatforms{addErr: errors.New("invalid category")}
	r := httptest.NewRequest(http.MethodPost, "/api/v1/admin/trusted-platforms",
		strings.NewReader(`{"domain_suffix":"foo.ru","display_name":"Foo","category":"bogus"}`))
	r.Header.Set("Authorization", "Bearer x")
	w := httptest.NewRecorder()
	newHandlerWithPlatforms("admin", stub).ServeHTTP(w, r)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", w.Code)
	}
}

func TestTrustedPlatforms_POST_201OnSuccess(t *testing.T) {
	stub := &stubPlatforms{}
	r := httptest.NewRequest(http.MethodPost, "/api/v1/admin/trusted-platforms",
		strings.NewReader(`{"domain_suffix":"timepad.ru","display_name":"Timepad","category":"ticketing"}`))
	r.Header.Set("Authorization", "Bearer x")
	w := httptest.NewRecorder()
	newHandlerWithPlatforms("admin", stub).ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", w.Code)
	}
	var got map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got["domain_suffix"] != "timepad.ru" || got["display_name"] != "Timepad" {
		t.Fatalf("body = %v", got)
	}
	if stub.added == nil {
		t.Fatalf("Add was not called")
	}
}

func TestTrustedPlatforms_DELETE_404OnUnknownID(t *testing.T) {
	stub := &stubPlatforms{deactErr: errors.New("not found")}
	id := uuid.Must(uuid.NewV4()).String()
	r := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/trusted-platforms/"+id, nil)
	r.Header.Set("Authorization", "Bearer x")
	w := httptest.NewRecorder()
	newHandlerWithPlatforms("admin", stub).ServeHTTP(w, r)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestTrustedPlatforms_DELETE_204OnSuccess(t *testing.T) {
	stub := &stubPlatforms{}
	id := uuid.Must(uuid.NewV4())
	r := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/trusted-platforms/"+id.String(), nil)
	r.Header.Set("Authorization", "Bearer x")
	w := httptest.NewRecorder()
	newHandlerWithPlatforms("admin", stub).ServeHTTP(w, r)
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", w.Code)
	}
	if stub.deactID != id {
		t.Fatalf("deactID = %v, want %v", stub.deactID, id)
	}
}

func (s stubOrgs) List(context.Context, organizers.ListFilter) ([]organizers.Organizer, error) {
	return s.orgs, nil
}

// The organizer registry's «СОБЫТИЙ» and «ЖАЛОБ» columns rendered a hard-coded
// dash — the payload carried neither count.
func TestSearchOrganizers_CarriesCounts(t *testing.T) {
	h := NewHandler(Deps{
		Authenticate: authFn("admin"),
		Moderation:   stubMod{},
		Organizers: stubOrgs{orgs: []organizers.Organizer{{
			ID: uuid.Must(uuid.NewV4()), Name: "Мастерская", VerificationStatus: "verified",
			EventsCount: 12, ComplaintsCount: 3,
		}}},
	})
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/organizers", nil)
	r.Header.Set("Authorization", "Bearer x")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var got []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0]["events_count"] != float64(12) {
		t.Fatalf("events_count = %v, want 12", got[0]["events_count"])
	}
	if got[0]["complaints_count"] != float64(3) {
		t.Fatalf("complaints_count = %v, want 3", got[0]["complaints_count"])
	}
}
