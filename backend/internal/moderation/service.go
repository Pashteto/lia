// Package moderation implements post-moderation actions over events:
// take-down (published → rejected) and reinstate (rejected → published),
// each logged to event_status_history + audit_log. See spec
// docs/superpowers/specs/2026-06-26-moderation-admin-foundation-design.md.
package moderation

import (
	"context"
	"errors"
	"strings"

	"github.com/gofrs/uuid"
)

// Counts is the admin overview summary.
type Counts struct {
	EventsTotal     int `json:"events_total"`
	EventsPublished int `json:"events_published"`
	EventsRemoved   int `json:"events_removed"`
}

// ErrInvalidTransition is returned when an event is not in the status a
// transition requires (e.g. take-down of a non-published event). Maps to 409.
var ErrInvalidTransition = errors.New("moderation: invalid status transition")

// ErrReasonRequired is returned when a take-down has no reason. Maps to 400.
var ErrReasonRequired = errors.New("moderation: reason required")

// Repository persists moderation transitions atomically (status + history + audit).
type Repository interface {
	Takedown(ctx context.Context, eventID, actorID uuid.UUID, reason string) error
	Reinstate(ctx context.Context, eventID, actorID uuid.UUID) error
	Counts(ctx context.Context) (Counts, error)
	LatestReason(ctx context.Context, eventID uuid.UUID) (string, error)
}

// Service is the moderation use-case layer.
type Service interface {
	Takedown(ctx context.Context, eventID, actorID uuid.UUID, reason string) error
	Reinstate(ctx context.Context, eventID, actorID uuid.UUID) error
	Overview(ctx context.Context) (Counts, error)
}

type service struct{ repo Repository }

// NewService returns a moderation Service backed by repo.
func NewService(repo Repository) Service { return &service{repo: repo} }

func (s *service) Takedown(ctx context.Context, eventID, actorID uuid.UUID, reason string) error {
	if strings.TrimSpace(reason) == "" {
		return ErrReasonRequired
	}
	return s.repo.Takedown(ctx, eventID, actorID, strings.TrimSpace(reason))
}

func (s *service) Reinstate(ctx context.Context, eventID, actorID uuid.UUID) error {
	return s.repo.Reinstate(ctx, eventID, actorID)
}

func (s *service) Overview(ctx context.Context) (Counts, error) { return s.repo.Counts(ctx) }
