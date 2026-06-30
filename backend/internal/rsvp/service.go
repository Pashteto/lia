package rsvp

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/gofrs/uuid"

	"github.com/Pashteto/lia/internal/models"
	"github.com/Pashteto/lia/pkg/logger"
)

// Domain errors. The HTTP layer maps these to status codes.
var (
	ErrInvalidInput = errors.New("invalid input")
	ErrNotFound     = errors.New("not found")
	ErrConflict     = errors.New("conflict")              // duplicate registration / wrong state
	ErrForbidden    = errors.New("forbidden")             // caller is not the event organizer
	ErrExternal     = errors.New("external registration") // signup happens on organizer's site
)

// PracticeRow is one attendance row for /me/practices: the rsvp plus its event.
type PracticeRow struct {
	Rsvp  *models.Rsvp
	Event *models.Event
}

// Service is the RSVP business-logic interface.
type Service interface {
	SignUp(ctx context.Context, eventID, userID uuid.UUID, answer string) (*models.Rsvp, error)
	Cancel(ctx context.Context, eventID, userID uuid.UUID) error
	MyPractices(ctx context.Context, userID uuid.UUID, tab string) ([]*PracticeRow, error)
	MyApplications(ctx context.Context, userID uuid.UUID, status string) ([]*models.Rsvp, error)
	// ListActiveEventsInRange returns the events the user has an active RSVP to
	// (going/waitlist/applied/accepted) whose start falls in [from, to). Used by
	// the personal calendar's "agreed to participate" stream.
	ListActiveEventsInRange(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]*models.Event, error)
	ListApplications(ctx context.Context, eventID, organizerID uuid.UUID) ([]*models.Rsvp, error)
	Decide(ctx context.Context, eventID, organizerID, rsvpID uuid.UUID, accept bool) (*models.Rsvp, error)
	CalendarICS(ctx context.Context, eventID uuid.UUID) ([]byte, error)
}

type service struct{ repo Repository }

// NewService creates an RSVP service backed by the given repository.
func NewService(repo Repository) Service { return &service{repo: repo} }

func isNoRows(err error) bool { return errors.Is(err, pg.ErrNoRows) }

func (s *service) SignUp(_ context.Context, eventID, userID uuid.UUID, answer string) (*models.Rsvp, error) {
	if eventID == uuid.Nil || userID == uuid.Nil {
		return nil, fmt.Errorf("%w: event and user are required", ErrInvalidInput)
	}
	event, err := s.repo.GetEvent(eventID)
	if err != nil {
		if isNoRows(err) {
			return nil, fmt.Errorf("%w: event %s", ErrNotFound, eventID)
		}
		return nil, fmt.Errorf("load event: %w", err)
	}

	switch event.SignupMode {
	case "external":
		// Caller registers on the organizer's site; surface the URL via the error.
		return nil, fmt.Errorf("%w: %s", ErrExternal, event.ExternalRegistrationURL)
	case "application":
		row, err := s.repo.SignUpTx(eventID, userID,
			func(int, *int) models.RsvpStatus { return models.RsvpApplied }, answer)
		return wrapSignupErr(row, err)
	default: // "" or "open"
		row, err := s.repo.SignUpTx(eventID, userID, openSeatDecider, "")
		return wrapSignupErr(row, err)
	}
}

// openSeatDecider gives a going seat when capacity allows, else waitlist.
func openSeatDecider(seatsTaken int, capacity *int) models.RsvpStatus {
	if seatAvailable(seatsTaken, capacity) {
		return models.RsvpGoing
	}
	return models.RsvpWaitlist
}

func wrapSignupErr(row *models.Rsvp, err error) (*models.Rsvp, error) {
	if err != nil {
		if errors.Is(err, ErrConflict) {
			return nil, err
		}
		return nil, fmt.Errorf("sign up: %w", err)
	}
	return row, nil
}

func (s *service) Cancel(_ context.Context, eventID, userID uuid.UUID) error {
	if err := s.repo.CancelTx(eventID, userID); err != nil {
		if isNoRows(err) {
			return fmt.Errorf("%w: no active registration", ErrNotFound)
		}
		return fmt.Errorf("cancel: %w", err)
	}
	return nil
}

func (s *service) MyPractices(_ context.Context, userID uuid.UUID, tab string) ([]*PracticeRow, error) {
	rows, err := s.repo.ListByUser(userID,
		[]models.RsvpStatus{models.RsvpGoing, models.RsvpWaitlist, models.RsvpAccepted})
	if err != nil {
		return nil, fmt.Errorf("my practices: %w", err)
	}
	out := make([]*PracticeRow, 0, len(rows))
	for _, r := range rows {
		if r.Event == nil {
			continue
		}
		isPast := r.Event.StartsAt.Before(nowFn())
		if tab == "past" && !isPast {
			continue
		}
		if tab != "past" && isPast {
			continue
		}
		out = append(out, &PracticeRow{Rsvp: r, Event: r.Event})
	}
	return out, nil
}

func (s *service) MyApplications(_ context.Context, userID uuid.UUID, status string) ([]*models.Rsvp, error) {
	want := []models.RsvpStatus{models.RsvpApplied, models.RsvpAccepted, models.RsvpDeclined, models.RsvpWithdrawn}
	if status != "" {
		want = []models.RsvpStatus{models.RsvpStatus(status)}
	}
	rows, err := s.repo.ListByUser(userID, want)
	if err != nil {
		return nil, fmt.Errorf("my applications: %w", err)
	}
	return rows, nil
}

func (s *service) ListActiveEventsInRange(_ context.Context, userID uuid.UUID, from, to time.Time) ([]*models.Event, error) {
	rows, err := s.repo.ListByUser(userID,
		[]models.RsvpStatus{models.RsvpGoing, models.RsvpWaitlist, models.RsvpApplied, models.RsvpAccepted})
	if err != nil {
		return nil, fmt.Errorf("active events in range: %w", err)
	}
	seen := make(map[uuid.UUID]struct{}, len(rows))
	out := make([]*models.Event, 0, len(rows))
	for _, r := range rows {
		if r.Event == nil {
			continue
		}
		if r.Event.StartsAt.Before(from) || !r.Event.StartsAt.Before(to) {
			continue
		}
		if _, dup := seen[r.Event.ID]; dup {
			continue
		}
		seen[r.Event.ID] = struct{}{}
		out = append(out, r.Event)
	}
	return out, nil
}

func (s *service) ListApplications(_ context.Context, eventID, organizerID uuid.UUID) ([]*models.Rsvp, error) {
	event, err := s.repo.GetEvent(eventID)
	if err != nil {
		if isNoRows(err) {
			return nil, fmt.Errorf("%w: event %s", ErrNotFound, eventID)
		}
		return nil, fmt.Errorf("load event: %w", err)
	}
	if event.OrganizerID != organizerID {
		return nil, fmt.Errorf("%w: not the organizer", ErrForbidden)
	}
	rows, err := s.repo.ListByEvent(eventID,
		[]models.RsvpStatus{models.RsvpApplied, models.RsvpAccepted, models.RsvpDeclined})
	if err != nil {
		return nil, fmt.Errorf("list applications: %w", err)
	}
	if err := s.repo.LoadApplicantNames(rows); err != nil {
		return nil, fmt.Errorf("load applicant names: %w", err)
	}
	return rows, nil
}

func (s *service) Decide(_ context.Context, eventID, organizerID, rsvpID uuid.UUID, accept bool) (*models.Rsvp, error) {
	event, err := s.repo.GetEvent(eventID)
	if err != nil {
		if isNoRows(err) {
			return nil, fmt.Errorf("%w: event %s", ErrNotFound, eventID)
		}
		return nil, fmt.Errorf("load event: %w", err)
	}
	if event.OrganizerID != organizerID {
		return nil, fmt.Errorf("%w: not the organizer", ErrForbidden)
	}
	row, err := s.repo.DecideTx(eventID, rsvpID, accept)
	if err != nil {
		switch {
		case isNoRows(err):
			return nil, fmt.Errorf("%w: rsvp %s", ErrNotFound, rsvpID)
		case errors.Is(err, ErrConflict):
			return nil, err
		default:
			return nil, fmt.Errorf("decide: %w", err)
		}
	}
	logger.Log().Infof("rsvp %s decided accept=%v -> %s", rsvpID, accept, row.Status)
	return row, nil
}

func (s *service) CalendarICS(_ context.Context, eventID uuid.UUID) ([]byte, error) {
	event, err := s.repo.GetEvent(eventID)
	if err != nil {
		if isNoRows(err) {
			return nil, fmt.Errorf("%w: event %s", ErrNotFound, eventID)
		}
		return nil, fmt.Errorf("load event: %w", err)
	}
	if event.StartsAt.IsZero() {
		return nil, fmt.Errorf("%w: event has no start time", ErrInvalidInput)
	}
	return buildICS(event), nil
}

// nowFn returns the current time; overridable in tests.
var nowFn = func() time.Time { return time.Now() }
