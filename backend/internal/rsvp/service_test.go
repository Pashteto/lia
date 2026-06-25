package rsvp

import (
	"context"
	"errors"
	"testing"

	"github.com/gofrs/uuid"

	"github.com/go-pg/pg/v10"
	"github.com/Pashteto/lia/internal/models"
)

// fakeRepo is an in-memory Repository for service tests.
type fakeRepo struct {
	event   *models.Event
	rsvps   map[uuid.UUID]*models.Rsvp // by rsvp id
	seats   int                        // current going count
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
	if r.Status == models.RsvpGoing {
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
	r.Status = models.RsvpCancelled
	return nil
}
func (f *fakeRepo) DecideTx(rsvpID uuid.UUID, accept bool) (*models.Rsvp, error) {
	r, ok := f.rsvps[rsvpID]
	if !ok {
		return nil, pg.ErrNoRows
	}
	if r.Status != models.RsvpApplied {
		return nil, ErrConflict
	}
	if !accept {
		r.Status = models.RsvpDeclined
	} else if seatAvailable(f.seats, f.event.Capacity) {
		r.Status = models.RsvpAccepted
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
