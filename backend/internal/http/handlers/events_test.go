package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/gofrs/uuid"

	eventsdomain "github.com/Pashteto/lia/internal/events"
	"github.com/Pashteto/lia/internal/http/models"
	eventsops "github.com/Pashteto/lia/internal/http/server/operations/events"
	domainmodels "github.com/Pashteto/lia/internal/models"
	organizersdomain "github.com/Pashteto/lia/internal/organizers"
	rsvpdomain "github.com/Pashteto/lia/internal/rsvp"
)

// mockEventsService captures the event passed to Create.
type mockEventsService struct {
	created          *domainmodels.Event
	createErr        error
	updated          *domainmodels.Event
	updateErr        error
	updateOwner      uuid.UUID
	getByID          *domainmodels.Event
	listStatusArg    string
	listFromArg      *time.Time
	listToArg        *time.Time
	listOrganizerArg *uuid.UUID
}

func (m *mockEventsService) Create(_ context.Context, e *domainmodels.Event) error {
	m.created = e
	return m.createErr
}
func (m *mockEventsService) Update(_ context.Context, id, ownerID uuid.UUID, _ eventsdomain.UpdateParams) (*domainmodels.Event, error) {
	m.updateOwner = ownerID
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	ev := &domainmodels.Event{ID: id, OrganizerID: ownerID, Title: "T", Status: domainmodels.EventDraft, StartsAt: time.Now()}
	m.updated = ev
	return ev, nil
}
func (m *mockEventsService) GetByID(context.Context, string) (*domainmodels.Event, error) {
	return m.getByID, nil
}
func (m *mockEventsService) List(_ context.Context, status string, from, to *time.Time, organizerOwnerID *uuid.UUID) ([]*domainmodels.Event, error) {
	m.listStatusArg = status
	m.listFromArg = from
	m.listToArg = to
	m.listOrganizerArg = organizerOwnerID
	return nil, nil
}
func (m *mockEventsService) ListByOrganizer(context.Context, uuid.UUID) ([]*domainmodels.Event, error) {
	return nil, nil
}
func (m *mockEventsService) Nearby(context.Context, *float64, *float64, int) ([]*eventsdomain.NearbyResult, error) {
	return nil, nil
}
func (m *mockEventsService) ListForCalendar(context.Context, []uuid.UUID, time.Time, time.Time) ([]*domainmodels.Event, error) {
	return nil, nil
}
func (m *mockEventsService) GetEnriched(context.Context, []uuid.UUID) ([]*domainmodels.Event, error) {
	return nil, nil
}

func TestCreateEvent_QuotaExceeded_Returns429(t *testing.T) {
	svc := &mockEventsService{createErr: fmt.Errorf("%w: 10/10 this month", eventsdomain.ErrQuotaExceeded)}
	h := NewCreateEvent(svc, nil)

	title := "Quota Test"
	starts := strfmt.DateTime(time.Now())
	params := eventsops.CreateEventParams{
		Body: &models.EventInput{
			Title:    &title,
			StartsAt: &starts,
		},
	}
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/events", nil)
	params.HTTPRequest = req

	pu := strfmt.UUID(uuid.Must(uuid.NewV4()).String())
	email := strfmt.Email("u@example.com")
	name := "U"
	status := "active"
	principal := &models.User{UUID: pu, Email: &email, Name: &name, Status: &status, EmailVerified: true}

	resp := h.Handle(params, principal)
	if resp == nil {
		t.Fatal("nil responder")
	}
	tooMany, ok := resp.(*eventsops.CreateEventTooManyRequests)
	if !ok {
		t.Fatalf("expected *CreateEventTooManyRequests, got %T", resp)
	}
	if tooMany.Payload == nil {
		t.Fatal("expected non-nil payload")
	}
	if tooMany.Payload.Code == nil || *tooMany.Payload.Code != 429 {
		t.Errorf("expected payload code 429, got %v", tooMany.Payload.Code)
	}
	const wantMsg = "Достигнут лимит: 10 событий в месяц. Лимит обновится 1-го числа."
	if tooMany.Payload.Message == nil || *tooMany.Payload.Message != wantMsg {
		t.Errorf("expected payload message %q, got %v", wantMsg, tooMany.Payload.Message)
	}
}

func TestCreateEvent_UnverifiedForbidden(t *testing.T) {
	svc := &mockEventsService{}
	h := NewCreateEvent(svc, nil)

	title := "Test Event"
	starts := strfmt.DateTime(time.Now())
	params := eventsops.CreateEventParams{
		Body: &models.EventInput{
			Title:    &title,
			StartsAt: &starts,
		},
	}
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/events", nil)
	params.HTTPRequest = req

	pu := strfmt.UUID(uuid.Must(uuid.NewV4()).String())
	email := strfmt.Email("u@example.com")
	name := "U"
	status := "active"
	// EmailVerified deliberately left false (zero value).
	principal := &models.User{UUID: pu, Email: &email, Name: &name, Status: &status}

	resp := h.Handle(params, principal)
	if resp == nil {
		t.Fatal("nil responder")
	}

	rr := httptest.NewRecorder()
	resp.WriteResponse(rr, runtime.JSONProducer())
	if rr.Code != http.StatusForbidden {
		t.Fatalf("want 403 for unverified, got %d", rr.Code)
	}
}

func TestCreateEvent_SetsOrganizerFromPrincipal(t *testing.T) {
	svc := &mockEventsService{}
	h := NewCreateEvent(svc, nil)

	title := "Test Event"
	starts := strfmt.DateTime(time.Now())
	// A client-supplied organizer_id must be IGNORED in favor of the principal.
	bodyOrganizer := strfmt.UUID(uuid.Must(uuid.NewV4()).String())
	params := eventsops.CreateEventParams{
		Body: &models.EventInput{
			Title:       &title,
			StartsAt:    &starts,
			OrganizerID: bodyOrganizer,
		},
	}
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/events", nil)
	params.HTTPRequest = req

	principalUUID := uuid.Must(uuid.NewV4())
	pu := strfmt.UUID(principalUUID.String())
	email := strfmt.Email("u@example.com")
	name := "U"
	status := "active"
	principal := &models.User{UUID: pu, Email: &email, Name: &name, Status: &status, EmailVerified: true}

	resp := h.Handle(params, principal)
	if resp == nil {
		t.Fatal("nil responder")
	}
	if svc.created == nil {
		t.Fatal("Create was not called")
	}
	if svc.created.OrganizerID != principalUUID {
		t.Errorf("expected organizer %s (from principal), got %s", principalUUID, svc.created.OrganizerID)
	}
}

func updateParams(t *testing.T) eventsops.UpdateEventParams {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPatch, "/api/v1/events/x", nil)
	return eventsops.UpdateEventParams{
		HTTPRequest: req,
		ID:          strfmt.UUID(uuid.Must(uuid.NewV4()).String()),
		Body:        &models.EventPatch{},
	}
}

func testPrincipal() *models.User {
	pu := strfmt.UUID(uuid.Must(uuid.NewV4()).String())
	email := strfmt.Email("u@example.com")
	name := "U"
	status := "active"
	return &models.User{UUID: pu, Email: &email, Name: &name, Status: &status, EmailVerified: true}
}

func TestUpdateEvent_Unauthenticated_Returns401(t *testing.T) {
	h := NewUpdateEvent(&mockEventsService{})
	resp := h.Handle(updateParams(t), nil)
	if _, ok := resp.(*eventsops.UpdateEventUnauthorized); !ok {
		t.Fatalf("expected *UpdateEventUnauthorized, got %T", resp)
	}
}

func TestUpdateEvent_NotFound_Returns404(t *testing.T) {
	svc := &mockEventsService{updateErr: fmt.Errorf("%w: event x", eventsdomain.ErrNotFound)}
	h := NewUpdateEvent(svc)
	resp := h.Handle(updateParams(t), testPrincipal())
	if _, ok := resp.(*eventsops.UpdateEventNotFound); !ok {
		t.Fatalf("expected *UpdateEventNotFound, got %T", resp)
	}
}

func TestUpdateEvent_Locked_Returns409(t *testing.T) {
	svc := &mockEventsService{updateErr: fmt.Errorf("%w: event x is published", eventsdomain.ErrNotEditable)}
	h := NewUpdateEvent(svc)
	resp := h.Handle(updateParams(t), testPrincipal())
	if _, ok := resp.(*eventsops.UpdateEventConflict); !ok {
		t.Fatalf("expected *UpdateEventConflict, got %T", resp)
	}
}

func TestUpdateEvent_CapacityBelowOccupied_Returns409(t *testing.T) {
	svc := &mockEventsService{updateErr: fmt.Errorf("%w: 3 occupied", eventsdomain.ErrCapacityBelowOccupied)}
	h := NewUpdateEvent(svc)
	resp := h.Handle(updateParams(t), testPrincipal())
	if _, ok := resp.(*eventsops.UpdateEventConflict); !ok {
		t.Fatalf("expected *UpdateEventConflict, got %T", resp)
	}
}

func TestUpdateEvent_Invalid_Returns400(t *testing.T) {
	svc := &mockEventsService{updateErr: fmt.Errorf("%w: bad", eventsdomain.ErrInvalidInput)}
	h := NewUpdateEvent(svc)
	resp := h.Handle(updateParams(t), testPrincipal())
	if _, ok := resp.(*eventsops.UpdateEventBadRequest); !ok {
		t.Fatalf("expected *UpdateEventBadRequest, got %T", resp)
	}
}

func TestUpdateEvent_Success_Returns200(t *testing.T) {
	h := NewUpdateEvent(&mockEventsService{})
	resp := h.Handle(updateParams(t), testPrincipal())
	if _, ok := resp.(*eventsops.UpdateEventOK); !ok {
		t.Fatalf("expected *UpdateEventOK, got %T", resp)
	}
}

func getByIDParams(t *testing.T, auth string) eventsops.GetEventByIDParams {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/events/x", nil)
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	return eventsops.GetEventByIDParams{
		HTTPRequest: req,
		ID:          strfmt.UUID(uuid.Must(uuid.NewV4()).String()),
	}
}

func TestGetEventByID_AnonymousDraft_Returns404(t *testing.T) {
	owner := uuid.Must(uuid.NewV4())
	svc := &mockEventsService{getByID: &domainmodels.Event{
		ID: uuid.Must(uuid.NewV4()), OrganizerID: owner,
		Status: domainmodels.EventDraft, Title: "D", StartsAt: time.Now(),
	}}
	// checkAuth returns unauthorized for any token (anonymous caller).
	h := NewGetEventByID(svc, nil, func(string) (*models.User, error) { return nil, errors.New("no auth") })

	resp := h.Handle(getByIDParams(t, ""))
	if _, ok := resp.(*eventsops.GetEventByIDNotFound); !ok {
		t.Fatalf("expected *GetEventByIDNotFound, got %T", resp)
	}
}

func TestGetEventByID_OwnerDraft_Returns200(t *testing.T) {
	owner := uuid.Must(uuid.NewV4())
	svc := &mockEventsService{getByID: &domainmodels.Event{
		ID: uuid.Must(uuid.NewV4()), OrganizerID: owner,
		Status: domainmodels.EventDraft, Title: "D", StartsAt: time.Now(),
	}}
	ownerPrincipal := &models.User{}
	pu := strfmt.UUID(owner.String())
	ownerPrincipal.UUID = pu
	h := NewGetEventByID(svc, nil, func(string) (*models.User, error) { return ownerPrincipal, nil })

	resp := h.Handle(getByIDParams(t, "Bearer tok"))
	if _, ok := resp.(*eventsops.GetEventByIDOK); !ok {
		t.Fatalf("expected *GetEventByIDOK, got %T", resp)
	}
}

func TestGetEventByID_AnonymousPublished_Returns200(t *testing.T) {
	svc := &mockEventsService{getByID: &domainmodels.Event{
		ID: uuid.Must(uuid.NewV4()), OrganizerID: uuid.Must(uuid.NewV4()),
		Status: domainmodels.EventPublished, Title: "P", StartsAt: time.Now(),
	}}
	h := NewGetEventByID(svc, nil, func(string) (*models.User, error) { return nil, errors.New("no auth") })

	resp := h.Handle(getByIDParams(t, ""))
	if _, ok := resp.(*eventsops.GetEventByIDOK); !ok {
		t.Fatalf("expected *GetEventByIDOK, got %T", resp)
	}
}

// stubRsvpStatus answers StatusForUser with a fixed status; the rest of the
// RSVP surface is unreachable from GetEventByID.
type stubRsvpStatus struct{ status domainmodels.RsvpStatus }

func (s stubRsvpStatus) StatusForUser(context.Context, uuid.UUID, uuid.UUID) (domainmodels.RsvpStatus, error) {
	return s.status, nil
}
func (stubRsvpStatus) CountPendingApplications(context.Context, []uuid.UUID) (map[uuid.UUID]int, error) {
	return map[uuid.UUID]int{}, nil
}
func (stubRsvpStatus) SignUp(context.Context, uuid.UUID, uuid.UUID, string) (*domainmodels.Rsvp, error) {
	return nil, nil
}
func (stubRsvpStatus) Cancel(context.Context, uuid.UUID, uuid.UUID) error { return nil }
func (stubRsvpStatus) MyPractices(context.Context, uuid.UUID, string) ([]*rsvpdomain.PracticeRow, error) {
	return nil, nil
}
func (stubRsvpStatus) MyApplications(context.Context, uuid.UUID, string) ([]*domainmodels.Rsvp, error) {
	return nil, nil
}
func (stubRsvpStatus) ListActiveEventsInRange(context.Context, uuid.UUID, time.Time, time.Time) ([]*domainmodels.Event, error) {
	return nil, nil
}
func (stubRsvpStatus) ListApplications(context.Context, uuid.UUID, uuid.UUID) ([]*domainmodels.Rsvp, error) {
	return nil, nil
}
func (stubRsvpStatus) Decide(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, bool) (*domainmodels.Rsvp, error) {
	return nil, nil
}
func (stubRsvpStatus) CalendarICS(context.Context, uuid.UUID) ([]byte, error) { return nil, nil }
func (stubRsvpStatus) SetEventEnricher(rsvpdomain.EventEnricher)              {}

func TestCanSeeEvent(t *testing.T) {
	cases := []struct {
		name    string
		status  string
		owner   bool
		hasRsvp bool
		want    bool
	}{
		{"published is public", "published", false, false, true},
		{"owner sees their draft", "draft", true, false, true},
		{"stranger does not see a draft", "draft", false, false, false},
		{"registered attendee still sees a cancelled event", "cancelled", false, true, true},
		{"stranger does not see a cancelled event", "cancelled", false, false, false},
		{"an rsvp does not unlock a draft", "draft", false, true, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := canSeeEvent(c.status, c.owner, c.hasRsvp); got != c.want {
				t.Fatalf("canSeeEvent(%q, owner=%v, rsvp=%v) = %v, want %v",
					c.status, c.owner, c.hasRsvp, got, c.want)
			}
		})
	}
}

// The gate consults the RSVP lookup, so a cancelled event must survive the
// handler end-to-end for someone who is going — not just in canSeeEvent.
func TestGetEventByID_CancelledWithRsvp_Returns200(t *testing.T) {
	attendee := uuid.Must(uuid.NewV4())
	svc := &mockEventsService{getByID: &domainmodels.Event{
		ID: uuid.Must(uuid.NewV4()), OrganizerID: uuid.Must(uuid.NewV4()),
		Status: domainmodels.EventCancelled, Title: "C", StartsAt: time.Now(),
	}}
	principal := &models.User{}
	principal.UUID = strfmt.UUID(attendee.String())
	h := NewGetEventByID(svc, stubRsvpStatus{status: domainmodels.RsvpGoing},
		func(string) (*models.User, error) { return principal, nil })

	resp := h.Handle(getByIDParams(t, "Bearer tok"))
	if _, ok := resp.(*eventsops.GetEventByIDOK); !ok {
		t.Fatalf("expected *GetEventByIDOK, got %T", resp)
	}
}

func TestGetEventByID_CancelledWithoutRsvp_Returns404(t *testing.T) {
	svc := &mockEventsService{getByID: &domainmodels.Event{
		ID: uuid.Must(uuid.NewV4()), OrganizerID: uuid.Must(uuid.NewV4()),
		Status: domainmodels.EventCancelled, Title: "C", StartsAt: time.Now(),
	}}
	principal := &models.User{}
	principal.UUID = strfmt.UUID(uuid.Must(uuid.NewV4()).String())
	h := NewGetEventByID(svc, stubRsvpStatus{status: ""},
		func(string) (*models.User, error) { return principal, nil })

	resp := h.Handle(getByIDParams(t, "Bearer tok"))
	if _, ok := resp.(*eventsops.GetEventByIDNotFound); !ok {
		t.Fatalf("expected *GetEventByIDNotFound, got %T", resp)
	}
}

func TestListEvents_ForcesPublished(t *testing.T) {
	svc := &mockEventsService{}
	h := NewListEvents(svc, nil)
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/events", nil)
	h.Handle(eventsops.ListEventsParams{HTTPRequest: req})
	if svc.listStatusArg != "published" {
		t.Fatalf("expected list to force published, got %q", svc.listStatusArg)
	}
	if svc.listFromArg != nil || svc.listToArg != nil {
		t.Fatalf("expected nil from/to without query params, got %v / %v", svc.listFromArg, svc.listToArg)
	}
}

func TestListEvents_PassesDateWindow(t *testing.T) {
	svc := &mockEventsService{}
	h := NewListEvents(svc, nil)
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/events", nil)
	from := strfmt.DateTime(time.Date(2026, 6, 27, 0, 0, 0, 0, time.UTC))
	to := strfmt.DateTime(time.Date(2026, 6, 28, 0, 0, 0, 0, time.UTC))
	h.Handle(eventsops.ListEventsParams{HTTPRequest: req, From: &from, To: &to})
	if svc.listFromArg == nil || !svc.listFromArg.Equal(time.Time(from)) {
		t.Fatalf("expected from=%v threaded through, got %v", time.Time(from), svc.listFromArg)
	}
	if svc.listToArg == nil || !svc.listToArg.Equal(time.Time(to)) {
		t.Fatalf("expected to=%v threaded through, got %v", time.Time(to), svc.listToArg)
	}
}

type fakeOrganizers struct {
	org *organizersdomain.Organizer
	err error
}

func (f *fakeOrganizers) GetByID(_ context.Context, _ uuid.UUID) (*organizersdomain.Organizer, error) {
	return f.org, f.err
}
func (f *fakeOrganizers) GetByOwner(_ context.Context, _ uuid.UUID) (*organizersdomain.Organizer, error) {
	return nil, nil
}
func (f *fakeOrganizers) Upsert(_ context.Context, _ uuid.UUID, _ organizersdomain.Input) (*organizersdomain.Organizer, error) {
	return nil, nil
}
func (f *fakeOrganizers) Submit(_ context.Context, _ uuid.UUID) (string, error) { return "", nil }
func (f *fakeOrganizers) IsVerifiedOwner(_ context.Context, _ uuid.UUID) (bool, error) {
	return f.org != nil && f.org.VerificationStatus == "verified", f.err
}
func (f *fakeOrganizers) EnsureForOwner(_ context.Context, _ uuid.UUID, _ string) (*organizersdomain.Organizer, error) {
	return f.org, f.err
}
func (f *fakeOrganizers) DailyEventLimit(_ context.Context, _ uuid.UUID) (int, bool, error) {
	return 0, false, nil
}
func (f *fakeOrganizers) SetDailyEventLimit(_ context.Context, _, _ uuid.UUID, _ *int) error {
	return nil
}
func (f *fakeOrganizers) Verify(_ context.Context, _, _ uuid.UUID) error { return nil }
func (f *fakeOrganizers) Reject(_ context.Context, _, _ uuid.UUID, _ string) error {
	return nil
}
func (f *fakeOrganizers) Revoke(_ context.Context, _, _ uuid.UUID, _ string) error {
	return nil
}
func (f *fakeOrganizers) SetAutoVerify(_ context.Context, _, _ uuid.UUID, _ bool) error {
	return nil
}
func (f *fakeOrganizers) List(_ context.Context, _ organizersdomain.ListFilter) ([]organizersdomain.Organizer, error) {
	return nil, nil
}
func (f *fakeOrganizers) GetWithHistory(_ context.Context, _ uuid.UUID) (*organizersdomain.Organizer, []organizersdomain.HistoryEntry, error) {
	return nil, nil, nil
}
func (f *fakeOrganizers) Overview(_ context.Context) (organizersdomain.Counts, error) {
	return organizersdomain.Counts{}, nil
}

func TestListEvents_OrganizerResolvesToVerifiedOwner(t *testing.T) {
	owner := uuid.Must(uuid.NewV4())
	profile := uuid.Must(uuid.NewV4())
	evSvc := &mockEventsService{}
	orgSvc := &fakeOrganizers{org: &organizersdomain.Organizer{OwnerUserID: owner, VerificationStatus: "verified"}}
	h := NewListEvents(evSvc, orgSvc)
	pid := strfmt.UUID(profile.String())
	resp := h.Handle(eventsops.ListEventsParams{HTTPRequest: httptest.NewRequest("GET", "/events", nil), OrganizerID: &pid})
	if _, ok := resp.(*eventsops.ListEventsOK); !ok {
		t.Fatalf("expected *ListEventsOK, got %T", resp)
	}
	if evSvc.listOrganizerArg == nil || *evSvc.listOrganizerArg != owner {
		t.Fatalf("expected owner %s passed to List, got %v", owner, evSvc.listOrganizerArg)
	}
}

func TestListEvents_UnverifiedOrganizerReturnsEmpty(t *testing.T) {
	evSvc := &mockEventsService{}
	orgSvc := &fakeOrganizers{org: &organizersdomain.Organizer{VerificationStatus: "pending"}}
	h := NewListEvents(evSvc, orgSvc)
	pid := strfmt.UUID(uuid.Must(uuid.NewV4()).String())
	resp := h.Handle(eventsops.ListEventsParams{HTTPRequest: httptest.NewRequest("GET", "/events", nil), OrganizerID: &pid})
	ok200, ok := resp.(*eventsops.ListEventsOK)
	if !ok {
		t.Fatalf("expected *ListEventsOK, got %T", resp)
	}
	if len(ok200.Payload) != 0 {
		t.Fatalf("expected empty payload, got %d items", len(ok200.Payload))
	}
	if evSvc.listOrganizerArg != nil {
		t.Fatalf("expected List not to receive an organizer filter for unverified org")
	}
}

func TestListEvents_UnknownOrganizerReturnsEmpty(t *testing.T) {
	evSvc := &mockEventsService{}
	orgSvc := &fakeOrganizers{err: organizersdomain.ErrNotFound}
	h := NewListEvents(evSvc, orgSvc)
	pid := strfmt.UUID(uuid.Must(uuid.NewV4()).String())
	resp := h.Handle(eventsops.ListEventsParams{HTTPRequest: httptest.NewRequest("GET", "/events", nil), OrganizerID: &pid})
	ok200, ok := resp.(*eventsops.ListEventsOK)
	if !ok {
		t.Fatalf("expected *ListEventsOK, got %T", resp)
	}
	if len(ok200.Payload) != 0 {
		t.Fatalf("expected empty payload, got %d items", len(ok200.Payload))
	}
	if evSvc.listOrganizerArg != nil {
		t.Fatalf("expected List not to be called for unknown organizer")
	}
}

func TestListEvents_OrganizerLookupErrorReturns503(t *testing.T) {
	evSvc := &mockEventsService{}
	orgSvc := &fakeOrganizers{err: errors.New("db down")}
	h := NewListEvents(evSvc, orgSvc)
	pid := strfmt.UUID(uuid.Must(uuid.NewV4()).String())
	resp := h.Handle(eventsops.ListEventsParams{HTTPRequest: httptest.NewRequest("GET", "/events", nil), OrganizerID: &pid})
	if _, ok := resp.(*eventsops.ListEventsServiceUnavailable); !ok {
		t.Fatalf("expected *ListEventsServiceUnavailable, got %T", resp)
	}
}

func TestQuotaMessage_DailyUsesTheActualLimit(t *testing.T) {
	cases := []struct {
		limit int
		want  string
	}{
		{1, "Достигнут дневной лимит: 1 событие. Лимит обновится завтра."},
		{3, "Достигнут дневной лимит: 3 события. Лимит обновится завтра."},
		{5, "Достигнут дневной лимит: 5 событий. Лимит обновится завтра."},
		{11, "Достигнут дневной лимит: 11 событий. Лимит обновится завтра."},
		{21, "Достигнут дневной лимит: 21 событие. Лимит обновится завтра."},
	}
	for _, c := range cases {
		err := &eventsdomain.QuotaError{Limit: c.limit, Used: c.limit, Period: "day"}
		if got := quotaMessage(err); got != c.want {
			t.Errorf("limit %d: got %q, want %q", c.limit, got, c.want)
		}
	}
}

func TestQuotaMessage_MonthlyKeepsItsWording(t *testing.T) {
	err := fmt.Errorf("%w: 10/10 this month", eventsdomain.ErrQuotaExceeded)
	const want = "Достигнут лимит: 10 событий в месяц. Лимит обновится 1-го числа."
	if got := quotaMessage(err); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// A daily QuotaError must still be recognised as a quota rejection so the
// handler answers 429 rather than 503.
func TestCreateEvent_DailyQuota_Returns429(t *testing.T) {
	svc := &mockEventsService{createErr: &eventsdomain.QuotaError{Limit: 3, Used: 3, Period: "day"}}
	h := NewCreateEvent(svc, nil)

	title := "Fourth today"
	starts := strfmt.DateTime(time.Now())
	params := eventsops.CreateEventParams{
		Body: &models.EventInput{Title: &title, StartsAt: &starts},
	}
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/events", nil)
	params.HTTPRequest = req

	resp := h.Handle(params, testPrincipal())
	tooMany, ok := resp.(*eventsops.CreateEventTooManyRequests)
	if !ok {
		t.Fatalf("expected *CreateEventTooManyRequests, got %T", resp)
	}
	if tooMany.Payload.Message == nil || *tooMany.Payload.Message != "Достигнут дневной лимит: 3 события. Лимит обновится завтра." {
		t.Errorf("unexpected message: %v", tooMany.Payload.Message)
	}
}
