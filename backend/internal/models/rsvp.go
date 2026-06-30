package models

import (
	"context"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
)

// RsvpStatus is the lifecycle state of a user's registration on an event.
// Stored verbatim as text (CHECK-constrained in db/migrations/000013).
type RsvpStatus string

const (
	// RsvpGoing — confirmed seat (open mode, or accepted application within capacity).
	RsvpGoing RsvpStatus = "going"
	// RsvpWaitlist — registered but no seat free; promoted FIFO when one frees up.
	RsvpWaitlist RsvpStatus = "waitlist"
	// RsvpApplied — application submitted, awaiting the organizer's decision.
	RsvpApplied RsvpStatus = "applied"
	// RsvpAccepted — application accepted by the organizer.
	RsvpAccepted RsvpStatus = "accepted"
	// RsvpDeclined — application declined by the organizer.
	RsvpDeclined RsvpStatus = "declined"
	// RsvpWithdrawn — applicant withdrew before a decision.
	RsvpWithdrawn RsvpStatus = "withdrawn"
	// RsvpCancelled — user cancelled a going/waitlist registration.
	RsvpCancelled RsvpStatus = "cancelled"
)

// validRsvpStatuses is the set accepted by Validate (mirrors the DB CHECK).
var validRsvpStatuses = map[RsvpStatus]struct{}{
	RsvpGoing: {}, RsvpWaitlist: {}, RsvpApplied: {}, RsvpAccepted: {},
	RsvpDeclined: {}, RsvpWithdrawn: {}, RsvpCancelled: {},
}

// IsActive reports whether the status represents a live registration (counts
// toward attendance lists). Terminal statuses (declined/withdrawn/cancelled) are not active.
func (s RsvpStatus) IsActive() bool {
	switch s {
	case RsvpGoing, RsvpWaitlist, RsvpApplied, RsvpAccepted:
		return true
	default:
		return false
	}
}

// Rsvp is a user's registration on an event.
//
//nolint:govet // field alignment kept for readability
type Rsvp struct {
	tableName struct{} `pg:"event_rsvps,discard_unknown_columns"` //nolint:unused

	ID                uuid.UUID  `pg:"id,pk,type:uuid"`
	EventID           uuid.UUID  `pg:"event_id,type:uuid,use_zero"`
	UserID            uuid.UUID  `pg:"user_id,type:uuid,use_zero"`
	Status            RsvpStatus `pg:"status,use_zero"`
	ApplicationAnswer string     `pg:"application_answer,use_zero"`
	CreatedAt         time.Time  `pg:"created_at,notnull,default:now()"`
	UpdatedAt         time.Time  `pg:"updated_at,notnull,default:now()"`

	// Event is a transient read-model populated by joins (e.g. MyPractices).
	Event *Event `pg:"-"`

	// ApplicantName is a transient display-name read-model populated only for
	// the organizer-facing applications list. Name only — never email.
	ApplicantName string `pg:"-"`
}

// Validate checks required fields and a known status.
func (r *Rsvp) Validate() error {
	if r.EventID == uuid.Nil {
		return newValidationError("event_id", "is required")
	}
	if r.UserID == uuid.Nil {
		return newValidationError("user_id", "is required")
	}
	if _, ok := validRsvpStatuses[r.Status]; !ok {
		return newValidationError("status", "invalid value")
	}
	return nil
}

// BeforeInsert generates an ID when missing and stamps timestamps.
func (r *Rsvp) BeforeInsert(ctx context.Context) (context.Context, error) {
	if r.ID == uuid.Nil {
		id, err := uuid.NewV4()
		if err != nil {
			return ctx, fmt.Errorf("generate UUID: %w", err)
		}
		r.ID = id
	}
	now := time.Now()
	if r.CreatedAt.IsZero() {
		r.CreatedAt = now
	}
	r.UpdatedAt = now
	return ctx, nil
}

// BeforeUpdate refreshes the updated_at timestamp.
func (r *Rsvp) BeforeUpdate(ctx context.Context) (context.Context, error) {
	r.UpdatedAt = time.Now()
	return ctx, nil
}
