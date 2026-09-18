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
		return r.logTransition(ctx, tx, eventID, actorID, from, to, action, reason)
	})
}

// logTransition writes the history + audit rows for a status change. Shared by
// transition and by the hand-written transactions that touch extra columns.
func (r *pgRepository) logTransition(ctx context.Context, tx *pg.Tx, eventID, actorID uuid.UUID, from, to, action, reason string) error {
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
}

func (r *pgRepository) Takedown(ctx context.Context, eventID, actorID uuid.UUID, reason string) error {
	return r.transition(ctx, eventID, actorID, "published", "rejected", "event.takedown", reason)
}

// Reinstate returns a rejected event to the feed. It also stamps reviewed_at:
// the admin has just made a judgment on this event, so it must not reappear in
// the post-moderation queue as if nobody had looked at it.
func (r *pgRepository) Reinstate(ctx context.Context, eventID, actorID uuid.UUID) error {
	return r.db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		res, err := tx.ExecContext(ctx,
			`UPDATE events
			    SET status = 'published',
			        reviewed_at = now(),
			        reviewed_by = ?,
			        updated_at = now()
			  WHERE id = ? AND status = 'rejected'`, actorID, eventID)
		if err != nil {
			return fmt.Errorf("reinstate event: %w", err)
		}
		if res.RowsAffected() == 0 {
			return ErrInvalidTransition
		}
		return r.logTransition(ctx, tx, eventID, actorID, "rejected", "published", "event.reinstate", "")
	})
}

// Review marks an already-published event as checked, which is what «одобрить»
// means in post-moderation: there is no status to move to, the event is already
// live. Without it the queue could never drain. Idempotent by the
// `reviewed_at IS NULL` guard — a second click is ErrInvalidTransition, not a
// second audit row.
func (r *pgRepository) Review(ctx context.Context, eventID, actorID uuid.UUID) error {
	return r.db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		res, err := tx.ExecContext(ctx,
			`UPDATE events
			    SET reviewed_at = now(), reviewed_by = ?, updated_at = now()
			  WHERE id = ? AND status = 'published' AND reviewed_at IS NULL`,
			actorID, eventID)
		if err != nil {
			return fmt.Errorf("review event: %w", err)
		}
		if res.RowsAffected() == 0 {
			return ErrInvalidTransition
		}
		// No event_status_history row: the status did not change. The audit log
		// is what records that a human cleared this event.
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO audit_log (actor_user_id, action, target_type, target_id, metadata)
			 VALUES (?, 'event.review', 'event', ?, '{}'::jsonb)`, actorID, eventID); err != nil {
			return fmt.Errorf("insert audit log: %w", err)
		}
		return nil
	})
}

// Approve transitions an event from pending_review to published, marking its
// external-registration URL as verified. Unlike transition, it touches extra
// columns (external_url_verified, published_at), so it gets its own transaction
// rather than reusing the fixed from/to helper.
func (r *pgRepository) Approve(ctx context.Context, eventID, actorID uuid.UUID) error {
	return r.db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		res, err := tx.ExecContext(ctx,
			`UPDATE events
			    SET status = 'published',
			        external_url_verified = true,
			        published_at = COALESCE(published_at, now()),
			        reviewed_at = now(),
			        reviewed_by = ?,
			        updated_at = now()
			  WHERE id = ? AND status = 'pending_review'`, actorID, eventID)
		if err != nil {
			return fmt.Errorf("approve event: %w", err)
		}
		if res.RowsAffected() == 0 {
			return ErrInvalidTransition
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO event_status_history (event_id, from_status, to_status, actor_user_id, reason)
			 VALUES (?, 'pending_review', 'published', ?, NULL)`, eventID, actorID); err != nil {
			return fmt.Errorf("insert status history: %w", err)
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO audit_log (actor_user_id, action, target_type, target_id, metadata)
			 VALUES (?, 'event.approve', 'event', ?, '{}'::jsonb)`, actorID, eventID); err != nil {
			return fmt.Errorf("insert audit log: %w", err)
		}
		return nil
	})
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
