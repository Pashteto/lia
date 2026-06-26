package follows

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gofrs/uuid"

	"github.com/Pashteto/lia/internal/models"
	organizers "github.com/Pashteto/lia/internal/organizers"
)

// ErrNotFound: the organizer profile does not exist or is not verified. Maps to
// 404 — mirrors the public organizer page, which never leaks non-verified
// profiles. The API addresses organizers by organizers.id (profile id).
var ErrNotFound = errors.New("follows: organizer not found")

// FollowedOrganizer is one row of the "organizers I follow" list.
type FollowedOrganizer struct {
	ProfileID  uuid.UUID
	OwnerID    uuid.UUID
	Name       string
	LogoFileID uuid.UUID
}

// EventLister resolves calendar events for a set of organizer owner ids. Satisfied
// by events.Service; kept local so follows stays testable with a fake.
type EventLister interface {
	ListForCalendar(ctx context.Context, organizerIDs []uuid.UUID, from, to time.Time) ([]*models.Event, error)
}

// Service is the follows use-case layer.
type Service interface {
	// Follow subscribes the user to the organizer profile. profileID is
	// organizers.id; 404 unless the profile exists and is verified.
	Follow(ctx context.Context, userID, profileID uuid.UUID) error
	// Unfollow removes the subscription. Idempotent; same verified-only resolution.
	Unfollow(ctx context.Context, userID, profileID uuid.UUID) error
	// IsFollowing reports whether the user follows the organizer identified by its
	// OWNER user id (the caller already holds it, e.g. the public org page).
	IsFollowing(ctx context.Context, userID, ownerUserID uuid.UUID) (bool, error)
	// ListFollowed returns the verified organizer profiles the user follows.
	ListFollowed(ctx context.Context, userID uuid.UUID) ([]FollowedOrganizer, error)
	// ListEventsFromFollowed returns published events in [from, to) from every
	// organizer the user follows (fully enriched, calendar-ready).
	ListEventsFromFollowed(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]*models.Event, error)
}

type service struct {
	repo   Repository
	orgs   organizers.Service
	events EventLister
}

// NewService returns a follows Service. orgs resolves profile id -> owner id and
// gates on verified status; events resolves followed-organizer calendar events.
func NewService(repo Repository, orgs organizers.Service, events EventLister) Service {
	return &service{repo: repo, orgs: orgs, events: events}
}

// resolveOwner maps a public profile id (organizers.id) to the owner user id,
// returning ErrNotFound for unknown or non-verified profiles (no leak).
func (s *service) resolveOwner(ctx context.Context, profileID uuid.UUID) (uuid.UUID, error) {
	org, err := s.orgs.GetByID(ctx, profileID)
	if err != nil {
		if errors.Is(err, organizers.ErrNotFound) {
			return uuid.Nil, ErrNotFound
		}
		return uuid.Nil, fmt.Errorf("resolve organizer %s: %w", profileID, err)
	}
	if org.VerificationStatus != "verified" {
		return uuid.Nil, ErrNotFound
	}
	return org.OwnerUserID, nil
}

func (s *service) Follow(ctx context.Context, userID, profileID uuid.UUID) error {
	owner, err := s.resolveOwner(ctx, profileID)
	if err != nil {
		return err
	}
	return s.repo.Add(ctx, userID, owner)
}

func (s *service) Unfollow(ctx context.Context, userID, profileID uuid.UUID) error {
	owner, err := s.resolveOwner(ctx, profileID)
	if err != nil {
		return err
	}
	return s.repo.Remove(ctx, userID, owner)
}

func (s *service) IsFollowing(ctx context.Context, userID, ownerUserID uuid.UUID) (bool, error) {
	if userID == uuid.Nil || ownerUserID == uuid.Nil {
		return false, nil
	}
	return s.repo.IsFollowing(ctx, userID, ownerUserID)
}

func (s *service) ListFollowed(ctx context.Context, userID uuid.UUID) ([]FollowedOrganizer, error) {
	rows, err := s.repo.ListFollowedOrganizers(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]FollowedOrganizer, 0, len(rows))
	for _, row := range rows {
		out = append(out, FollowedOrganizer{
			ProfileID: row.ProfileID, OwnerID: row.OwnerID,
			Name: row.Name, LogoFileID: row.LogoFileID,
		})
	}
	return out, nil
}

func (s *service) ListEventsFromFollowed(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]*models.Event, error) {
	ownerIDs, err := s.repo.ListFollowedOwnerIDs(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(ownerIDs) == 0 {
		return nil, nil
	}
	return s.events.ListForCalendar(ctx, ownerIDs, from, to)
}
