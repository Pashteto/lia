package feedback

import (
	"context"
	"fmt"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/gofrs/uuid"
)

type pgRepository struct{ db *pg.DB }

var _ Repository = (*pgRepository)(nil)

func NewRepository(db *pg.DB) Repository { return &pgRepository{db: db} }

func (r *pgRepository) EventGate(ctx context.Context, eventID uuid.UUID) (uuid.UUID, time.Time, bool, error) {
	var row struct {
		OrganizerID uuid.UUID `pg:"organizer_id"`
		EndsAt      time.Time `pg:"ends_at"`
	}
	_, err := r.db.QueryOneContext(ctx, &row,
		`SELECT organizer_id, COALESCE(ends_at, starts_at) AS ends_at
		   FROM events WHERE id = ?`, eventID)
	if err == pg.ErrNoRows {
		return uuid.Nil, time.Time{}, false, nil
	}
	if err != nil {
		return uuid.Nil, time.Time{}, false, fmt.Errorf("event gate %s: %w", eventID, err)
	}
	return row.OrganizerID, row.EndsAt, true, nil
}

func (r *pgRepository) HasActiveRsvp(ctx context.Context, eventID, userID uuid.UUID) (bool, error) {
	n, err := r.db.ModelContext(ctx, (*struct {
		tableName struct{} `pg:"event_rsvps"`
	})(nil)).
		Where("event_id = ? AND user_id = ? AND status IN ('going','accepted')", eventID, userID).
		Count()
	if err != nil {
		return false, fmt.Errorf("active rsvp: %w", err)
	}
	return n > 0, nil
}

func (r *pgRepository) ExistsForUser(ctx context.Context, eventID, userID uuid.UUID) (bool, error) {
	var exists bool
	_, err := r.db.QueryOneContext(ctx, pg.Scan(&exists),
		`SELECT EXISTS(SELECT 1 FROM event_feedback WHERE event_id = ? AND user_id = ?)`, eventID, userID)
	if err != nil {
		return false, fmt.Errorf("exists for user: %w", err)
	}
	return exists, nil
}

func (r *pgRepository) Insert(ctx context.Context, f Feedback) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO event_feedback (event_id, user_id, rating, comment)
		 VALUES (?, ?, ?, NULLIF(?, ''))`,
		f.EventID, f.UserID, f.Rating, f.Comment)
	if err != nil {
		return fmt.Errorf("insert feedback: %w", err)
	}
	return nil
}

func (r *pgRepository) ListForEvent(ctx context.Context, eventID uuid.UUID) ([]Item, error) {
	var rows []Item
	// Name only — email is never selected (private author identity, public-safe).
	_, err := r.db.QueryContext(ctx, &rows,
		`SELECT f.rating AS rating, COALESCE(f.comment,'') AS comment,
		        COALESCE(u.name,'') AS author_name, f.created_at AS created_at
		   FROM event_feedback f
		   LEFT JOIN users u ON u.uuid = f.user_id
		  WHERE f.event_id = ?
		  ORDER BY f.created_at DESC`, eventID)
	if err != nil {
		return nil, fmt.Errorf("list feedback: %w", err)
	}
	return rows, nil
}
