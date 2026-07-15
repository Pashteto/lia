package invitations

import (
	"context"
	"database/sql"
	"errors"

	"github.com/go-pg/pg/v10"
	"github.com/gofrs/uuid"
)

type pgRepository struct{ db *pg.DB }

// NewRepository builds the go-pg backed invitation repository.
func NewRepository(db *pg.DB) Repository { return &pgRepository{db: db} }

func (r *pgRepository) Insert(ctx context.Context, inv Invitation) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO event_invitations
		   (id, event_id, inviter_user_id, invitee_email, token, status, created_at, expires_at)
		 VALUES (?, ?, ?, lower(?), ?, 'pending', now(), ?)
		 ON CONFLICT (event_id, lower(invitee_email)) WHERE status='pending' DO NOTHING`,
		inv.ID, inv.EventID, inv.InviterUserID, inv.InviteeEmail, inv.Token, inv.ExpiresAt)
	return err
}

func (r *pgRepository) GetByToken(ctx context.Context, token string) (*Invitation, error) {
	return r.getBy(ctx, `token = ?`, token)
}

func (r *pgRepository) GetByID(ctx context.Context, id uuid.UUID) (*Invitation, error) {
	return r.getBy(ctx, `id = ?`, id)
}

func (r *pgRepository) getBy(ctx context.Context, where string, arg any) (*Invitation, error) {
	var inv Invitation
	_, err := r.db.QueryOneContext(ctx, pg.Scan(
		&inv.ID, &inv.EventID, &inv.InviterUserID, &inv.InviteeEmail,
		&inv.Token, &inv.Status, &inv.CreatedAt, &inv.RespondedAt, &inv.ExpiresAt),
		`SELECT id, event_id, inviter_user_id, invitee_email, token, status,
		        created_at, COALESCE(responded_at, 'epoch'), expires_at
		   FROM event_invitations WHERE `+where, arg)
	if errors.Is(err, pg.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

func (r *pgRepository) ListPendingByEmail(ctx context.Context, email string) ([]Invitation, error) {
	var out []Invitation
	_, err := r.db.QueryContext(ctx, &out,
		`SELECT id, event_id, inviter_user_id, invitee_email, token, status,
		        created_at, COALESCE(responded_at,'epoch') AS responded_at, expires_at
		   FROM event_invitations
		  WHERE lower(invitee_email) = lower(?) AND status = 'pending' AND expires_at > now()
		  ORDER BY created_at DESC`, email)
	return out, err
}

func (r *pgRepository) SetStatus(ctx context.Context, id uuid.UUID, status string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE event_invitations SET status = ?, responded_at = now() WHERE id = ?`, status, id)
	return err
}

func (r *pgRepository) ExpireOverdue(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE event_invitations SET status='expired'
		  WHERE status='pending' AND expires_at <= now()`)
	return err
}
