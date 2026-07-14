// Package feedback implements private post-event ratings: a participant who had
// an active RSVP on an ended event leaves a 1-5 star rating + optional comment
// (one per person); the event owner (and admin) reads them privately. See spec
// docs/superpowers/specs/2026-07-14-post-event-feedback-design.md.
package feedback

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/gofrs/uuid"
)

var (
	ErrNotEnded         = errors.New("feedback: event has not ended") // 422
	ErrNotParticipant   = errors.New("feedback: not a participant")   // 403
	ErrAlreadySubmitted = errors.New("feedback: already submitted")   // 409
	ErrInvalidRating    = errors.New("feedback: rating must be 1..5") // 422
	ErrForbidden        = errors.New("feedback: not the event owner") // 403
	ErrNotFound         = errors.New("feedback: event not found")     // 404
)

type Feedback struct {
	ID        uuid.UUID
	EventID   uuid.UUID
	UserID    uuid.UUID
	Rating    int
	Comment   string
	CreatedAt time.Time
}

type Item struct {
	Rating     int       `json:"rating"`
	Comment    string    `json:"comment,omitempty"`
	AuthorName string    `json:"author_name"`
	CreatedAt  time.Time `json:"created_at"`
}

type Summary struct {
	Average float64 `json:"average"`
	Count   int     `json:"count"`
	Items   []Item  `json:"items"`
}

type Repository interface {
	EventGate(ctx context.Context, eventID uuid.UUID) (ownerID uuid.UUID, endsAt time.Time, exists bool, err error)
	HasActiveRsvp(ctx context.Context, eventID, userID uuid.UUID) (bool, error)
	ExistsForUser(ctx context.Context, eventID, userID uuid.UUID) (bool, error)
	Insert(ctx context.Context, f Feedback) error
	ListForEvent(ctx context.Context, eventID uuid.UUID) ([]Item, error)
}

type Service interface {
	Submit(ctx context.Context, userID, eventID uuid.UUID, rating int, comment string) error
	ForOwner(ctx context.Context, eventID, requesterID uuid.UUID, isAdmin bool) (Summary, error)
	MyFeedback(ctx context.Context, userID, eventID uuid.UUID) (bool, error)
}

type service struct{ repo Repository }

func NewService(repo Repository) Service { return &service{repo: repo} }

func (s *service) Submit(ctx context.Context, userID, eventID uuid.UUID, rating int, comment string) error {
	if rating < 1 || rating > 5 {
		return ErrInvalidRating
	}
	owner, endsAt, exists, err := s.repo.EventGate(ctx, eventID)
	if err != nil {
		return err
	}
	_ = owner
	if !exists {
		return ErrNotFound
	}
	if !endsAt.Before(time.Now()) {
		return ErrNotEnded
	}
	active, err := s.repo.HasActiveRsvp(ctx, eventID, userID)
	if err != nil {
		return err
	}
	if !active {
		return ErrNotParticipant
	}
	already, err := s.repo.ExistsForUser(ctx, eventID, userID)
	if err != nil {
		return err
	}
	if already {
		return ErrAlreadySubmitted
	}
	return s.repo.Insert(ctx, Feedback{
		EventID: eventID, UserID: userID, Rating: rating, Comment: strings.TrimSpace(comment),
	})
}

func (s *service) ForOwner(ctx context.Context, eventID, requesterID uuid.UUID, isAdmin bool) (Summary, error) {
	owner, _, exists, err := s.repo.EventGate(ctx, eventID)
	if err != nil {
		return Summary{}, err
	}
	if !exists {
		return Summary{}, ErrNotFound
	}
	if !isAdmin && owner != requesterID {
		return Summary{}, ErrForbidden
	}
	items, err := s.repo.ListForEvent(ctx, eventID)
	if err != nil {
		return Summary{}, err
	}
	sum := Summary{Count: len(items), Items: items}
	if len(items) > 0 {
		total := 0
		for _, it := range items {
			total += it.Rating
		}
		sum.Average = float64(total) / float64(len(items))
	}
	return sum, nil
}

func (s *service) MyFeedback(ctx context.Context, userID, eventID uuid.UUID) (bool, error) {
	return s.repo.ExistsForUser(ctx, eventID, userID)
}
