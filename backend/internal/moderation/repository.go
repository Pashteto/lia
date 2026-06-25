package moderation

import (
	"context"
	"fmt"

	"github.com/go-pg/pg/v10"
	"github.com/gofrs/uuid"
)

type pgRepository struct{ db *pg.DB }

// NewRepository returns a pg-backed moderation Repository.
func NewRepository(db *pg.DB) Repository { return &pgRepository{db: db} }

// transition flips an event's status from->to inside one transaction, writing
// an event_status_history row and an audit_log row. It is a no-op error
// (ErrInvalidTransition) when the event is not currently in `from`.
func (r *pgRepository) transition(ctx context.Context, eventID, actorID uuid.UUID, from, to, action, reason string) error {
	return r.db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		res, err := tx.ExecContext(ctx,
			`UPDATE events SET status = ?, updated_at = now() WHERE id = ? AND status = ?`,
			to, eventID, from)
		if err != nil {
			return fmt.Errorf("update event status: %w", err)
		}
		if res.RowsAffected() == 0 {
			return ErrInvalidTransition
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO event_status_history (event_id, from_status, to_status, actor_user_id, reason)
			 VALUES (?, ?, ?, ?, NULLIF(?, ''))`,
			eventID, from, to, actorID, reason); err != nil {
			return fmt.Errorf("insert status history: %w", err)
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO audit_log (actor_user_id, action, target_type, target_id, metadata)
			 VALUES (?, ?, 'event', ?, jsonb_build_object('reason', NULLIF(?, '')))`,
			actorID, action, eventID, reason); err != nil {
			return fmt.Errorf("insert audit log: %w", err)
		}
		return nil
	})
}

func (r *pgRepository) Takedown(ctx context.Context, eventID, actorID uuid.UUID, reason string) error {
	return r.transition(ctx, eventID, actorID, "published", "rejected", "event.takedown", reason)
}

func (r *pgRepository) Reinstate(ctx context.Context, eventID, actorID uuid.UUID) error {
	return r.transition(ctx, eventID, actorID, "rejected", "published", "event.reinstate", "")
}

func (r *pgRepository) Counts(ctx context.Context) (Counts, error) {
	var c Counts
	_, err := r.db.QueryOneContext(ctx, pg.Scan(&c.EventsTotal, &c.EventsPublished, &c.EventsRemoved),
		`SELECT count(*),
		        count(*) FILTER (WHERE status = 'published'),
		        count(*) FILTER (WHERE status = 'rejected')
		 FROM events`)
	if err != nil {
		return Counts{}, fmt.Errorf("count events: %w", err)
	}
	return c, nil
}

func (r *pgRepository) LatestReason(ctx context.Context, eventID uuid.UUID) (string, error) {
	var reason string
	_, err := r.db.QueryOneContext(ctx, pg.Scan(&reason),
		`SELECT coalesce(reason, '') FROM event_status_history
		 WHERE event_id = ? AND to_status = 'rejected'
		 ORDER BY created_at DESC LIMIT 1`, eventID)
	if err != nil {
		if err == pg.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("latest reason: %w", err)
	}
	return reason, nil
}
