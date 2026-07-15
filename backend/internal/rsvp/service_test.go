package rsvp

import (
	"context"
	"errors"
	"testing"

	"github.com/gofrs/uuid"

	"github.com/Pashteto/lia/internal/models"
	"github.com/go-pg/pg/v10"
)

// fakeRepo is an in-memory Repository for service tests.
type fakeRepo struct {
	event *models.Event
	rsvps map[uuid.UUID]*models.Rsvp // by rsvp id
	seats int                        // current going count
}

func newFake(e *models.Event) *fakeRepo {
	return &fakeRepo{event: e, rsvps: map[uuid.UUID]*models.Rsvp{}}
}

func (f *fakeRepo) GetEvent(id uuid.UUID) (*models.Event, error) {
	if f.event == nil || f.event.ID != id {
		return nil, pg.ErrNoRows
	}
	return f.event, nil
}
func (f *fakeRepo) GetUserRsvp(eventID, userID uuid.UUID) (*models.Rsvp, error) {
	for _, r := range f.rsvps {
		if r.EventID == eventID && r.UserID == userID {
			return r, nil
		}
	}
	return nil, pg.ErrNoRows
}
func (f *fakeRepo) GetRsvpByID(id uuid.UUID) (*models.Rsvp, error) {
	if r, ok := f.rsvps[id]; ok {
		return r, nil
	}
	return nil, pg.ErrNoRows
}
func (f *fakeRepo) CountActiveSeats(uuid.UUID) (int, error) { return f.seats, nil }
func (f *fakeRepo) SignUpTx(eventID, userID uuid.UUID, decide SeatDecider, answer string) (*models.Rsvp, error) {
	if r, _ := f.GetUserRsvp(eventID, userID); r != nil && r.Status.IsActive() {
		return nil, ErrConflict
	}
	status := decide(f.seats, f.event.Capacity)
	if status == models.RsvpGoing {
		f.seats++
	}
	row := &models.Rsvp{ID: uuid.Must(uuid.NewV4()), EventID: eventID, UserID: userID, Status: status, ApplicationAnswer: answer}
	f.rsvps[row.ID] = row
	return row, nil
}
func (f *fakeRepo) CancelTx(eventID, userID uuid.UUID) error {
	r, err := f.GetUserRsvp(eventID, userID)
	if err != nil || !r.Status.IsActive() {
		return pg.ErrNoRows
	}
	// mirrors real repo: free a seat when going OR accepted
	freedSeat := r.Status == models.RsvpGoing || r.Status == models.RsvpAccepted
	newStatus := models.RsvpCancelled
	if r.Status == models.RsvpApplied || r.Status == models.RsvpAccepted {
		newStatus = models.RsvpWithdrawn
	}
	r.Status = newStatus
	if freedSeat {
		f.seats--
		// promote oldest waitlist
		for _, w := range f.rsvps {
			if w.EventID == eventID && w.Status == models.RsvpWaitlist {
				w.Status = models.RsvpGoing
				f.seats++
				break
			}
		}
	}
	return nil
}
func (f *fakeRepo) DecideTx(eventID, rsvpID uuid.UUID, accept bool) (*models.Rsvp, error) {
	r, ok := f.rsvps[rsvpID]
	if !ok {
		return nil, pg.ErrNoRows
	}
	// mirrors real repo: reject rsvps belonging to a different event
	if r.EventID != eventID {
		return nil, pg.ErrNoRows
	}
	if r.Status != models.RsvpApplied {
		return nil, ErrConflict
	}
	if !accept {
		r.Status = models.RsvpDeclined
	} else if seatAvailable(f.seats, f.event.Capacity) {
		r.Status = models.RsvpAccepted
		f.seats++ // accepted row occupies a seat (mirrors real capacity accounting)
	} else {
		r.Status = models.RsvpWaitlist
	}
	return r, nil
}
func (f *fakeRepo) ListByUser(userID uuid.UUID, st []models.RsvpStatus) ([]*models.Rsvp, error) {
	var out []*models.Rsvp
	for _, r := range f.rsvps {
		if r.UserID == userID {
			out = append(out, r)
		}
	}
	return out, nil
}
func (f *fakeRepo) ListByEvent(eventID uuid.UUID, st []models.RsvpStatus) ([]*models.Rsvp, error) {
	var out []*models.Rsvp
	for _, r := range f.rsvps {
		if r.EventID == eventID {
			out = append(out, r)
		}
	}
	return out, nil
}
func (f *fakeRepo) LoadApplicantNames(_ []*models.Rsvp) error { return nil }

func openEvent(cap *int) *models.Event {
	return &models.Event{ID: uuid.Must(uuid.NewV4()), OrganizerID: uuid.Must(uuid.NewV4()), SignupMode: "open", Capacity: cap}
}

func TestSignUpOpenFillsThenWaitlists(t *testing.T) {
	cap1 := 1
	e := openEvent(&cap1)
	svc := NewService(newFake(e))
	r1, err := svc.SignUp(context.Background(), e.ID, uuid.Must(uuid.NewV4()), "")
	if err != nil || r1.Status != models.RsvpGoing {
		t.Fatalf("first signup want going, got %v err %v", r1, err)
	}
	r2, err := svc.SignUp(context.Background(), e.ID, uuid.Must(uuid.NewV4()), "")
	if err != nil || r2.Status != models.RsvpWaitlist {
		t.Fatalf("second signup want waitlist, got %v err %v", r2, err)
	}
}

func TestSignUpDuplicateConflicts(t *testing.T) {
	e := openEvent(nil)
	svc := NewService(newFake(e))
	u := uuid.Must(uuid.NewV4())
	if _, err := svc.SignUp(context.Background(), e.ID, u, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SignUp(context.Background(), e.ID, u, ""); !errors.Is(err, ErrConflict) {
		t.Fatalf("want ErrConflict, got %v", err)
	}
}

func TestCancelPromotesWaitlist(t *testing.T) {
	cap1 := 1
	e := openEvent(&cap1)
	f := newFake(e)
	svc := NewService(f)
	uGoing := uuid.Must(uuid.NewV4())
	if _, err := svc.SignUp(context.Background(), e.ID, uGoing, ""); err != nil {
		t.Fatal(err)
	}
	uWait := uuid.Must(uuid.NewV4())
	if _, err := svc.SignUp(context.Background(), e.ID, uWait, ""); err != nil {
		t.Fatal(err)
	}
	if err := svc.Cancel(context.Background(), e.ID, uGoing); err != nil {
		t.Fatal(err)
	}
	promoted, _ := f.GetUserRsvp(e.ID, uWait)
	if promoted.Status != models.RsvpGoing {
		t.Fatalf("waitlisted user should be promoted, got %s", promoted.Status)
	}
}

func TestSignUpExternalReturnsErrExternal(t *testing.T) {
	e := &models.Event{ID: uuid.Must(uuid.NewV4()), SignupMode: "external", ExternalRegistrationURL: "https://x"}
	svc := NewService(newFake(e))
	_, err := svc.SignUp(context.Background(), e.ID, uuid.Must(uuid.NewV4()), "")
	if !errors.Is(err, ErrExternal) {
		t.Fatalf("want ErrExternal, got %v", err)
	}
}

func TestApplicationDecideAcceptDecline(t *testing.T) {
	e := &models.Event{ID: uuid.Must(uuid.NewV4()), OrganizerID: uuid.Must(uuid.NewV4()), SignupMode: "application", CuratorQuestion: "?"}
	f := newFake(e)
	svc := NewService(f)
	app, err := svc.SignUp(context.Background(), e.ID, uuid.Must(uuid.NewV4()), "хочу прийти")
	if err != nil || app.Status != models.RsvpApplied {
		t.Fatalf("want applied, got %v err %v", app, err)
	}
	got, err := svc.Decide(context.Background(), e.ID, e.OrganizerID, app.ID, true)
	if err != nil || got.Status != models.RsvpAccepted {
		t.Fatalf("accept want accepted, got %v err %v", got, err)
	}
}

func TestDecideByNonOrganizerForbidden(t *testing.T) {
	e := &models.Event{ID: uuid.Must(uuid.NewV4()), OrganizerID: uuid.Must(uuid.NewV4()), SignupMode: "application", CuratorQuestion: "?"}
	f := newFake(e)
	svc := NewService(f)
	app, _ := svc.SignUp(context.Background(), e.ID, uuid.Must(uuid.NewV4()), "x")
	_, err := svc.Decide(context.Background(), e.ID, uuid.Must(uuid.NewV4()), app.ID, true)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("want ErrForbidden, got %v", err)
	}
}

// FIX A: organizer of event A must not be able to mutate an rsvp on event B.
func TestDecideForeignRsvpNotFound(t *testing.T) {
	// Event A: the organizer controls this event.
	eA := &models.Event{ID: uuid.Must(uuid.NewV4()), OrganizerID: uuid.Must(uuid.NewV4()), SignupMode: "application"}
	fA := newFake(eA)
	svcA := NewService(fA)

	// Event B: different event, same fake store trick — we inject rsvpB directly.
	eB := &models.Event{ID: uuid.Must(uuid.NewV4()), OrganizerID: uuid.Must(uuid.NewV4()), SignupMode: "application"}
	rsvpB := &models.Rsvp{
		ID:      uuid.Must(uuid.NewV4()),
		EventID: eB.ID,
		UserID:  uuid.Must(uuid.NewV4()),
		Status:  models.RsvpApplied,
	}
	// Plant rsvpB into fA's map so fA.DecideTx sees it and can check event ownership.
	fA.rsvps[rsvpB.ID] = rsvpB

	// Decide on event A using rsvpB's ID — must return ErrNotFound, not mutate rsvpB.
	_, err := svcA.Decide(context.Background(), eA.ID, eA.OrganizerID, rsvpB.ID, true)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound for foreign rsvp, got %v", err)
	}
	if rsvpB.Status != models.RsvpApplied {
		t.Fatalf("foreign rsvp must not be mutated, got status %s", rsvpB.Status)
	}
}

// FIX B: accept when capacity full must put the applicant on the waitlist.
func TestDecideAcceptWhenFullWaitlists(t *testing.T) {
	cap1 := 1
	e := &models.Event{
		ID:          uuid.Must(uuid.NewV4()),
		OrganizerID: uuid.Must(uuid.NewV4()),
		SignupMode:  "application",
		Capacity:    &cap1,
	}
	f := newFake(e)
	svc := NewService(f)

	// Fill the one seat directly.
	goingRsvp := &models.Rsvp{
		ID:      uuid.Must(uuid.NewV4()),
		EventID: e.ID,
		UserID:  uuid.Must(uuid.NewV4()),
		Status:  models.RsvpGoing,
	}
	f.rsvps[goingRsvp.ID] = goingRsvp
	f.seats = 1

	// A second user applies.
	app, err := svc.SignUp(context.Background(), e.ID, uuid.Must(uuid.NewV4()), "me please")
	if err != nil || app.Status != models.RsvpApplied {
		t.Fatalf("want applied, got %v err %v", app, err)
	}

	// Organizer accepts — capacity is full, so the result must be waitlist.
	got, err := svc.Decide(context.Background(), e.ID, e.OrganizerID, app.ID, true)
	if err != nil {
		t.Fatalf("Decide returned error: %v", err)
	}
	if got.Status != models.RsvpWaitlist {
		t.Fatalf("want RsvpWaitlist when full, got %s", got.Status)
	}
}

// accepted application seats count toward capacity: a second applicant must be
// waitlisted when the only seat is already consumed by an accepted row.
func TestAcceptSecondApplicantWaitlistsWhenFirstAccepted(t *testing.T) {
	cap1 := 1
	e := &models.Event{
		ID:          uuid.Must(uuid.NewV4()),
		OrganizerID: uuid.Must(uuid.NewV4()),
		SignupMode:  "application",
		Capacity:    &cap1,
	}
	f := newFake(e)
	svc := NewService(f)

	// Applicant A applies and is accepted — consumes the only seat.
	userA := uuid.Must(uuid.NewV4())
	appA, err := svc.SignUp(context.Background(), e.ID, userA, "first")
	if err != nil || appA.Status != models.RsvpApplied {
		t.Fatalf("A: want applied, got %v err %v", appA, err)
	}
	acceptedA, err := svc.Decide(context.Background(), e.ID, e.OrganizerID, appA.ID, true)
	if err != nil || acceptedA.Status != models.RsvpAccepted {
		t.Fatalf("A: want accepted, got %v err %v", acceptedA, err)
	}

	// Applicant B applies.
	userB := uuid.Must(uuid.NewV4())
	appB, err := svc.SignUp(context.Background(), e.ID, userB, "second")
	if err != nil || appB.Status != models.RsvpApplied {
		t.Fatalf("B: want applied, got %v err %v", appB, err)
	}

	// Organizer accepts B — capacity is full (accepted A occupies it), so B must waitlist.
	gotB, err := svc.Decide(context.Background(), e.ID, e.OrganizerID, appB.ID, true)
	if err != nil {
		t.Fatalf("B Decide returned error: %v", err)
	}
	if gotB.Status != models.RsvpWaitlist {
		t.Fatalf("want RsvpWaitlist for B (accepted A fills capacity), got %s", gotB.Status)
	}
}

func TestStatusForUser(t *testing.T) {
	cap1 := 5
	e := openEvent(&cap1)
	userID := uuid.Must(uuid.NewV4())

	t.Run("returns the user's status", func(t *testing.T) {
		f := newFake(e)
		row := &models.Rsvp{ID: uuid.Must(uuid.NewV4()), EventID: e.ID, UserID: userID, Status: models.RsvpGoing}
		f.rsvps[row.ID] = row
		svc := NewService(f)

		got, err := svc.StatusForUser(context.Background(), e.ID, userID)
		if err != nil {
			t.Fatalf("StatusForUser returned error: %v", err)
		}
		if got != models.RsvpGoing {
			t.Fatalf("want RsvpGoing, got %s", got)
		}
	})

	t.Run("no rsvp -> empty status, no error", func(t *testing.T) {
		f := newFake(e) // no rsvps planted -> GetUserRsvp returns pg.ErrNoRows
		svc := NewService(f)

		got, err := svc.StatusForUser(context.Background(), e.ID, userID)
		if err != nil {
			t.Fatalf("StatusForUser returned error: %v", err)
		}
		if got != models.RsvpStatus("") {
			t.Fatalf("want empty status, got %q", got)
		}
	})
}

// FIX B: cancelling an accepted seat must promote a waitlisted row to going.
func TestCancelAcceptedPromotesWaitlist(t *testing.T) {
	cap1 := 1
	e := &models.Event{
		ID:          uuid.Must(uuid.NewV4()),
		OrganizerID: uuid.Must(uuid.NewV4()),
		SignupMode:  "application",
		Capacity:    &cap1,
	}
	f := newFake(e)

	uAccepted := uuid.Must(uuid.NewV4())
	acceptedRsvp := &models.Rsvp{
		ID:      uuid.Must(uuid.NewV4()),
		EventID: e.ID,
		UserID:  uAccepted,
		Status:  models.RsvpAccepted,
	}
	f.rsvps[acceptedRsvp.ID] = acceptedRsvp
	f.seats = 1 // accepted occupies a seat

	uWaitlist := uuid.Must(uuid.NewV4())
	waitRsvp := &models.Rsvp{
		ID:      uuid.Must(uuid.NewV4()),
		EventID: e.ID,
		UserID:  uWaitlist,
		Status:  models.RsvpWaitlist,
	}
	f.rsvps[waitRsvp.ID] = waitRsvp

	svc := NewService(f)
	if err := svc.Cancel(context.Background(), e.ID, uAccepted); err != nil {
		t.Fatalf("Cancel returned error: %v", err)
	}

	// The previously waitlisted user must now be going.
	promoted, _ := f.GetUserRsvp(e.ID, uWaitlist)
	if promoted.Status != models.RsvpGoing {
		t.Fatalf("want waitlisted user promoted to going, got %s", promoted.Status)
	}
	// The cancelled user's rsvp should be withdrawn (was accepted).
	cancelled, _ := f.GetUserRsvp(e.ID, uAccepted)
	if cancelled.Status != models.RsvpWithdrawn {
		t.Fatalf("want accepted->withdrawn on cancel, got %s", cancelled.Status)
	}
}
