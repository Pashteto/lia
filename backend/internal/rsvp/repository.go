package rsvp

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-pg/pg/v10"
	"github.com/gofrs/uuid"

	"github.com/Pashteto/lia/internal/models"
	"github.com/Pashteto/lia/pkg/logger"
)

// SeatDecider returns the status to assign given seats taken and capacity.
type SeatDecider func(seatsTaken int, capacity *int) models.RsvpStatus

// seatAvailable reports whether one more going seat fits under capacity.
// A nil capacity means unlimited.
func seatAvailable(seatsTaken int, capacity *int) bool {
	if capacity == nil {
		return true
	}
	return seatsTaken < *capacity
}

// Repository defines RSVP persistence.
type Repository interface {
	GetEvent(id uuid.UUID) (*models.Event, error)
	GetUserRsvp(eventID, userID uuid.UUID) (*models.Rsvp, error)
	GetRsvpByID(id uuid.UUID) (*models.Rsvp, error)
	CountActiveSeats(eventID uuid.UUID) (int, error)
	SignUpTx(eventID, userID uuid.UUID, decide SeatDecider, answer string) (*models.Rsvp, error)
	CancelTx(eventID, userID uuid.UUID) error
	DecideTx(eventID, rsvpID uuid.UUID, accept bool) (*models.Rsvp, error)
	ListByUser(userID uuid.UUID, statuses []models.RsvpStatus) ([]*models.Rsvp, error)
	ListByEvent(eventID uuid.UUID, statuses []models.RsvpStatus) ([]*models.Rsvp, error)
}

type pgRepository struct{ db *pg.DB }

// NewRepository creates a PostgreSQL-backed RSVP repository.
func NewRepository(db *pg.DB) Repository { return &pgRepository{db: db} }

func (r *pgRepository) GetEvent(id uuid.UUID) (*models.Event, error) {
	e := &models.Event{ID: id}
	if err := r.db.Model(e).WherePK().Select(); err != nil {
		return nil, fmt.Errorf("get event %s: %w", id, err)
	}
	return e, nil
}

func (r *pgRepository) GetUserRsvp(eventID, userID uuid.UUID) (*models.Rsvp, error) {
	out := new(models.Rsvp)
	if err := r.db.Model(out).
		Where("event_id = ? AND user_id = ?", eventID, userID).Select(); err != nil {
		return nil, fmt.Errorf("get rsvp for event %s user %s: %w", eventID, userID, err)
	}
	return out, nil
}

func (r *pgRepository) GetRsvpByID(id uuid.UUID) (*models.Rsvp, error) {
	out := &models.Rsvp{ID: id}
	if err := r.db.Model(out).WherePK().Select(); err != nil {
		return nil, fmt.Errorf("get rsvp %s: %w", id, err)
	}
	return out, nil
}

func (r *pgRepository) CountActiveSeats(eventID uuid.UUID) (int, error) {
	n, err := r.db.Model((*models.Rsvp)(nil)).
		Where("event_id = ? AND status = ?", eventID, models.RsvpGoing).Count()
	if err != nil {
		return 0, fmt.Errorf("count seats for event %s: %w", eventID, err)
	}
	return n, nil
}

// SignUpTx locks the event row, counts going seats, and inserts/re-activates the
// caller's row with the status chosen by decide(). UNIQUE(event,user) guarantees
// one row; a prior terminal row is transitioned in place.
func (r *pgRepository) SignUpTx(eventID, userID uuid.UUID, decide SeatDecider, answer string) (*models.Rsvp, error) {
	var result *models.Rsvp
	err := r.db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
		// Lock the event row to serialize concurrent sign-ups (last-seat race).
		var capacity *int
		if _, err := tx.QueryOne(pg.Scan(&capacity),
			`SELECT capacity FROM events WHERE id = ? FOR UPDATE`, eventID); err != nil {
			return fmt.Errorf("lock event %s: %w", eventID, err)
		}

		seats, err := tx.Model((*models.Rsvp)(nil)).
			Where("event_id = ? AND status = ?", eventID, models.RsvpGoing).Count()
		if err != nil {
			return fmt.Errorf("count seats: %w", err)
		}
		status := decide(seats, capacity)

		existing := new(models.Rsvp)
		err = tx.Model(existing).Where("event_id = ? AND user_id = ?", eventID, userID).Select()
		switch {
		case err == nil:
			if existing.Status.IsActive() {
				return ErrConflict // already registered/applied
			}
			existing.Status = status
			existing.ApplicationAnswer = answer
			if _, uerr := tx.Model(existing).
				Column("status", "application_answer", "updated_at").WherePK().Update(); uerr != nil {
				return fmt.Errorf("reactivate rsvp: %w", uerr)
			}
			result = existing
		case errors.Is(err, pg.ErrNoRows):
			row := &models.Rsvp{EventID: eventID, UserID: userID, Status: status, ApplicationAnswer: answer}
			if _, ierr := tx.Model(row).Insert(); ierr != nil {
				return fmt.Errorf("insert rsvp: %w", ierr)
			}
			result = row
		default:
			return fmt.Errorf("select existing rsvp: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// CancelTx terminates the caller's active row. If a going seat was freed, the
// oldest waitlist row (by created_at) is promoted to going — same transaction.
func (r *pgRepository) CancelTx(eventID, userID uuid.UUID) error {
	return r.db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
		if _, err := tx.Exec(`SELECT id FROM events WHERE id = ? FOR UPDATE`, eventID); err != nil {
			return fmt.Errorf("lock event %s: %w", eventID, err)
		}
		cur := new(models.Rsvp)
		if err := tx.Model(cur).Where("event_id = ? AND user_id = ?", eventID, userID).Select(); err != nil {
			if errors.Is(err, pg.ErrNoRows) {
				return pg.ErrNoRows
			}
			return fmt.Errorf("select rsvp: %w", err)
		}
		if !cur.Status.IsActive() {
			return pg.ErrNoRows // nothing active to cancel
		}
		freedSeat := cur.Status == models.RsvpGoing || cur.Status == models.RsvpAccepted

		newStatus := models.RsvpCancelled
		if cur.Status == models.RsvpApplied || cur.Status == models.RsvpAccepted {
			newStatus = models.RsvpWithdrawn
		}
		cur.Status = newStatus
		if _, err := tx.Model(cur).Column("status", "updated_at").WherePK().Update(); err != nil {
			return fmt.Errorf("cancel rsvp: %w", err)
		}

		if freedSeat {
			next := new(models.Rsvp)
			err := tx.Model(next).
				Where("event_id = ? AND status = ?", eventID, models.RsvpWaitlist).
				Order("created_at ASC").Limit(1).Select()
			if err == nil {
				next.Status = models.RsvpGoing
				if _, uerr := tx.Model(next).Column("status", "updated_at").WherePK().Update(); uerr != nil {
					return fmt.Errorf("promote waitlist: %w", uerr)
				}
				logger.Log().Infof("rsvp: promoted %s from waitlist on event %s", next.ID, eventID)
			} else if !errors.Is(err, pg.ErrNoRows) {
				return fmt.Errorf("find waitlist head: %w", err)
			}
		}
		return nil
	})
}

// DecideTx accepts (or waitlists if full) / declines an applied row.
// The rsvp must belong to eventID; if not, pg.ErrNoRows is returned so the
// service maps it to ErrNotFound (prevents cross-event mutation).
func (r *pgRepository) DecideTx(eventID, rsvpID uuid.UUID, accept bool) (*models.Rsvp, error) {
	var result *models.Rsvp
	err := r.db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
		row := &models.Rsvp{ID: rsvpID}
		if err := tx.Model(row).WherePK().Select(); err != nil {
			if errors.Is(err, pg.ErrNoRows) {
				return pg.ErrNoRows
			}
			return fmt.Errorf("select rsvp %s: %w", rsvpID, err)
		}
		if row.EventID != eventID {
			return pg.ErrNoRows // rsvp belongs to a different event
		}
		if row.Status != models.RsvpApplied {
			return ErrConflict // already decided / not an application
		}
		if !accept {
			row.Status = models.RsvpDeclined
		} else {
			var capacity *int
			if _, err := tx.QueryOne(pg.Scan(&capacity),
				`SELECT capacity FROM events WHERE id = ? FOR UPDATE`, row.EventID); err != nil {
				return fmt.Errorf("lock event: %w", err)
			}
			seats, err := tx.Model((*models.Rsvp)(nil)).
				Where("event_id = ? AND status = ?", row.EventID, models.RsvpGoing).Count()
			if err != nil {
				return fmt.Errorf("count seats: %w", err)
			}
			if seatAvailable(seats, capacity) {
				row.Status = models.RsvpAccepted
			} else {
				row.Status = models.RsvpWaitlist
			}
		}
		if _, err := tx.Model(row).Column("status", "updated_at").WherePK().Update(); err != nil {
			return fmt.Errorf("update decision: %w", err)
		}
		result = row
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *pgRepository) ListByUser(userID uuid.UUID, statuses []models.RsvpStatus) ([]*models.Rsvp, error) {
	var rows []*models.Rsvp
	q := r.db.Model(&rows).Where("user_id = ?", userID)
	if len(statuses) > 0 {
		q = q.Where("status IN (?)", pg.In(statuses))
	}
	if err := q.Order("created_at DESC").Select(); err != nil {
		return nil, fmt.Errorf("list rsvps for user %s: %w", userID, err)
	}
	return r.attachEvents(rows)
}

func (r *pgRepository) ListByEvent(eventID uuid.UUID, statuses []models.RsvpStatus) ([]*models.Rsvp, error) {
	var rows []*models.Rsvp
	q := r.db.Model(&rows).Where("event_id = ?", eventID)
	if len(statuses) > 0 {
		q = q.Where("status IN (?)", pg.In(statuses))
	}
	if err := q.Order("created_at ASC").Select(); err != nil {
		return nil, fmt.Errorf("list rsvps for event %s: %w", eventID, err)
	}
	return rows, nil
}

// attachEvents batch-loads the Event for each rsvp (no N+1).
func (r *pgRepository) attachEvents(rows []*models.Rsvp) ([]*models.Rsvp, error) {
	if len(rows) == 0 {
		return rows, nil
	}
	ids := make([]uuid.UUID, 0, len(rows))
	seen := map[uuid.UUID]struct{}{}
	for _, row := range rows {
		if _, ok := seen[row.EventID]; !ok {
			seen[row.EventID] = struct{}{}
			ids = append(ids, row.EventID)
		}
	}
	var events []*models.Event
	if err := r.db.Model(&events).Where("id IN (?)", pg.In(ids)).Select(); err != nil {
		return nil, fmt.Errorf("attach events: %w", err)
	}
	byID := make(map[uuid.UUID]*models.Event, len(events))
	for _, e := range events {
		byID[e.ID] = e
	}
	for _, row := range rows {
		row.Event = byID[row.EventID]
	}
	return rows, nil
}
