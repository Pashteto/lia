package invitations

import (
	"context"
	"errors"
	"time"

	"github.com/gofrs/uuid"
)

// Invitation is an organizer-issued invite for a person (by email) to attend
// a specific event. Rows live in the event_invitations table.
type Invitation struct {
	tableName struct{} `pg:"event_invitations"`

	ID            uuid.UUID `pg:"id"`
	EventID       uuid.UUID `pg:"event_id"`
	InviterUserID uuid.UUID `pg:"inviter_user_id"`
	InviteeEmail  string    `pg:"invitee_email"`
	Token         string    `pg:"token"`
	Status        string    `pg:"status"`
	CreatedAt     time.Time `pg:"created_at"`
	RespondedAt   time.Time `pg:"responded_at"`
	ExpiresAt     time.Time `pg:"expires_at"`
}

// Repository is the data-access layer over event_invitations.
type Repository interface {
	Insert(ctx context.Context, inv Invitation) error
	GetByToken(ctx context.Context, token string) (*Invitation, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Invitation, error)
	ListPendingByEmail(ctx context.Context, email string) ([]Invitation, error)
	SetStatus(ctx context.Context, id uuid.UUID, status string) error
	ExpireOverdue(ctx context.Context) error
}

// ErrNotFound is returned when an invitation lookup finds no matching row.
var ErrNotFound = errors.New("invitation not found")
