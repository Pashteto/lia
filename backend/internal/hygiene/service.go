package hygiene

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gofrs/uuid"

	"github.com/Pashteto/lia/internal/models"
	"github.com/Pashteto/lia/internal/moderation"
)

// Events is the subset of the events domain service this package needs.
type Events interface {
	List(ctx context.Context, status string, from, to *time.Time, organizerOwnerID *uuid.UUID, city string) ([]*models.Event, error)
}

// Moderator is the subset of the moderation service used to hide events.
// Reusing it keeps every hide transactional and audit-logged.
type Moderator interface {
	Takedown(ctx context.Context, eventID, actorID uuid.UUID, reason string) error
}

// Result reports a bulk hide. Skipped counts events whose status had already
// moved (someone else got there first) — not a failure.
type Result struct {
	Hidden  int `json:"hidden"`
	Skipped int `json:"skipped"`
}

// Service is the hygiene use-case layer.
type Service interface {
	List(ctx context.Context) ([]Issue, error)
	HideAll(ctx context.Context, actorID uuid.UUID) (Result, error)
}

type service struct {
	events Events
	mod    Moderator
}

// NewService returns a hygiene Service over the published-events feed.
func NewService(events Events, mod Moderator) Service {
	return &service{events: events, mod: mod}
}

func (s *service) List(ctx context.Context) ([]Issue, error) {
	events, err := s.events.List(ctx, "published", nil, nil, nil, "")
	if err != nil {
		return nil, fmt.Errorf("list published events: %w", err)
	}
	return Detect(candidates(events)), nil
}

func (s *service) HideAll(ctx context.Context, actorID uuid.UUID) (Result, error) {
	issues, err := s.List(ctx)
	if err != nil {
		return Result{}, err
	}
	var res Result
	for _, issue := range issues {
		switch err := s.mod.Takedown(ctx, issue.EventID, actorID, ReasonFor(issue.Kind)); {
		case err == nil:
			res.Hidden++
		case errors.Is(err, moderation.ErrInvalidTransition):
			res.Skipped++
		default:
			return res, fmt.Errorf("hide %s: %w", issue.EventID, err)
		}
	}
	return res, nil
}

// candidates maps loaded events onto the detector's input. Organizer is a
// transient read-model and may be nil when the event has no organizer row.
func candidates(events []*models.Event) []Candidate {
	out := make([]Candidate, 0, len(events))
	for _, e := range events {
		if e == nil {
			continue
		}
		c := Candidate{
			EventID: e.ID, Title: e.Title,
			PriceType: e.PriceType, PriceMin: e.PriceMin, PriceMax: e.PriceMax,
		}
		if e.Organizer != nil {
			c.OrganizerName = e.Organizer.Name
		}
		out = append(out, c)
	}
	return out
}
