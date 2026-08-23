// Package organizers implements the 1:1 organizer profile per user and the
// admin verification workflow (draft → pending → verified/rejected, resubmit +
// revoke). Each transition writes organizer_verification_history + audit_log in
// one tx (mirrors internal/moderation). Submit short-circuits to verified when
// the global app setting organizers.auto_verify_all is on OR the org's
// auto_verify flag is set. See spec 2026-06-26-organizer-entity-verification-design.md.
package organizers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"

	"github.com/Pashteto/lia/internal/settings"
)

var (
	// ErrInvalidTransition: the organizer is not in the status a transition requires. Maps to 409.
	ErrInvalidTransition = errors.New("organizers: invalid status transition")
	// ErrReasonRequired: reject/revoke called without a reason. Maps to 400.
	ErrReasonRequired = errors.New("organizers: reason required")
	// ErrNameRequired: upsert called without a name. Maps to 400.
	ErrNameRequired = errors.New("organizers: name required")
	// ErrNotFound: no organizer profile for the owner/id. Maps to 404.
	ErrNotFound = errors.New("organizers: not found")
)

// Organizer is the domain entity for an organizer profile.
type Organizer struct {
	tableName struct{} `pg:"organizers,discard_unknown_columns"` //nolint:unused // go-pg table marker

	ID                 uuid.UUID  `pg:"id,pk,type:uuid"`
	OwnerUserID        uuid.UUID  `pg:"owner_user_id,type:uuid"`
	Name               string     `pg:"name,use_zero"`
	Description        string     `pg:"description,use_zero"`
	WebsiteURL         string     `pg:"website_url,use_zero"`
	LogoFileID         uuid.UUID  `pg:"logo_file_id,type:uuid,use_zero"`
	VerificationStatus string     `pg:"verification_status,use_zero"`
	AutoVerify         bool       `pg:"auto_verify,use_zero"`
	VerifiedAt         *time.Time `pg:"verified_at"` // nullable; ORM scans NULL → nil
	// DailyEventLimit overrides the global daily event-creation cap for this
	// organizer. nil means "use the default"; 0 means uncapped.
	DailyEventLimit *int   `pg:"daily_event_limit"`
	LatestReason    string `pg:"-"` // transient, not a column
	// Registry columns «Событий» / «Жалоб». Transient, batch-loaded by List.
	EventsCount     int `pg:"-"`
	ComplaintsCount int `pg:"-"`
}

// Input is the editable subset of an organizer profile.
type Input struct {
	Name        string
	Description string
	WebsiteURL  string
	LogoFileID  uuid.UUID
}

// HistoryEntry is one verification transition.
type HistoryEntry struct {
	FromStatus  string    `pg:"from_status"`
	ToStatus    string    `pg:"to_status"`
	Reason      string    `pg:"reason,use_zero"`
	ActorUserID uuid.UUID `pg:"actor_user_id"`
	CreatedAt   time.Time `pg:"created_at"`
}

// ListFilter selects organizers for the admin queue/search.
type ListFilter struct {
	Status string // "", "pending", "verified", "rejected", "draft"
	Query  string // case-insensitive name/owner-email search
}

// Counts is the admin overview summary contribution.
type Counts struct {
	OrganizersPending int `json:"organizers_pending"`
	// OrganizersTotal is every organizer profile regardless of verification
	// status — the «Организаторов» tile, which had nothing to render.
	OrganizersTotal int `json:"organizers_total"`
}

// Repository persists organizers + verification transitions.
type Repository interface {
	GetByOwner(ctx context.Context, ownerID uuid.UUID) (*Organizer, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Organizer, error)
	Upsert(ctx context.Context, ownerID uuid.UUID, in Input) (*Organizer, error)
	Submit(ctx context.Context, id, actorID uuid.UUID, autoVerify bool) (newStatus string, err error)
	Verify(ctx context.Context, id, actorID uuid.UUID) error
	Reject(ctx context.Context, id, actorID uuid.UUID, reason string) error
	Revoke(ctx context.Context, id, actorID uuid.UUID, reason string) error
	SetAutoVerify(ctx context.Context, id, actorID uuid.UUID, enabled bool) error
	SetDailyEventLimit(ctx context.Context, id, actorID uuid.UUID, limit *int) error
	List(ctx context.Context, f ListFilter) ([]Organizer, error)
	History(ctx context.Context, id uuid.UUID) ([]HistoryEntry, error)
	Counts(ctx context.Context) (Counts, error)
}

// Service is the organizers use-case layer.
type Service interface {
	GetByOwner(ctx context.Context, ownerID uuid.UUID) (*Organizer, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Organizer, error)
	Upsert(ctx context.Context, ownerID uuid.UUID, in Input) (*Organizer, error)
	Submit(ctx context.Context, ownerID uuid.UUID) (newStatus string, err error)
	// EnsureForOwner gives an event-creating user the organizer profile their
	// events imply. No-op when they already have one.
	EnsureForOwner(ctx context.Context, ownerID uuid.UUID, name string) (*Organizer, error)
	// DailyEventLimit reports an owner's per-day event cap override. ok is
	// false when they have none and the global default applies.
	DailyEventLimit(ctx context.Context, ownerID uuid.UUID) (limit int, ok bool, err error)
	// IsVerifiedOwner reports whether the owner's organizer profile passed
	// admin verification. Missing profile → false (not an error).
	IsVerifiedOwner(ctx context.Context, ownerID uuid.UUID) (bool, error)
	Verify(ctx context.Context, id, actorID uuid.UUID) error
	Reject(ctx context.Context, id, actorID uuid.UUID, reason string) error
	Revoke(ctx context.Context, id, actorID uuid.UUID, reason string) error
	SetAutoVerify(ctx context.Context, id, actorID uuid.UUID, enabled bool) error
	// SetDailyEventLimit overrides (nil clears) this organizer's daily cap.
	SetDailyEventLimit(ctx context.Context, id, actorID uuid.UUID, limit *int) error
	List(ctx context.Context, f ListFilter) ([]Organizer, error)
	GetWithHistory(ctx context.Context, id uuid.UUID) (*Organizer, []HistoryEntry, error)
	Overview(ctx context.Context) (Counts, error)
}

type service struct {
	repo Repository
	set  settings.Service
}

// NewService returns an organizers Service. set provides the global auto-verify flag.
func NewService(repo Repository, set settings.Service) Service {
	return &service{repo: repo, set: set}
}

func (s *service) GetByOwner(ctx context.Context, ownerID uuid.UUID) (*Organizer, error) {
	return s.repo.GetByOwner(ctx, ownerID)
}

func (s *service) IsVerifiedOwner(ctx context.Context, ownerID uuid.UUID) (bool, error) {
	org, err := s.repo.GetByOwner(ctx, ownerID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return org.VerificationStatus == "verified", nil
}
func (s *service) GetByID(ctx context.Context, id uuid.UUID) (*Organizer, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *service) Upsert(ctx context.Context, ownerID uuid.UUID, in Input) (*Organizer, error) {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return nil, ErrNameRequired
	}
	in.Description = strings.TrimSpace(in.Description)
	in.WebsiteURL = strings.TrimSpace(in.WebsiteURL)
	return s.repo.Upsert(ctx, ownerID, in)
}

// Submit moves the owner's profile draft|rejected → pending, or → verified when
// the global flag or the org's auto_verify is set.
func (s *service) Submit(ctx context.Context, ownerID uuid.UUID) (string, error) {
	org, err := s.repo.GetByOwner(ctx, ownerID)
	if err != nil {
		return "", err
	}
	global, err := s.set.Bool(ctx, settings.KeyAutoVerifyAll)
	if err != nil {
		return "", err
	}
	return s.repo.Submit(ctx, org.ID, ownerID, global || org.AutoVerify)
}

// EnsureForOwner backs the "publishing an event makes you an organizer" rule.
//
// Before this, `organizers` rows were created only by the profile form, so
// somebody who just created events was an organizer in every functional sense
// yet invisible to the admin registry, unfollowable, and could never carry a
// verification badge. The profile is created from the user's own display name
// and auto-verified — repo.Submit with autoVerify writes the history and audit
// rows, so the promotion is as traceable as a moderator's click.
//
// Idempotent: an existing profile is returned untouched, whatever its
// verification status — this must never resurrect a profile a moderator revoked.
func (s *service) EnsureForOwner(ctx context.Context, ownerID uuid.UUID, name string) (*Organizer, error) {
	existing, err := s.repo.GetByOwner(ctx, ownerID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	name = strings.TrimSpace(name)
	if name == "" {
		name = "Организатор"
	}
	org, err := s.repo.Upsert(ctx, ownerID, Input{Name: name})
	if err != nil {
		return nil, err
	}
	if _, err := s.repo.Submit(ctx, org.ID, ownerID, true); err != nil {
		return nil, fmt.Errorf("auto-verify organizer %s: %w", org.ID, err)
	}
	return s.repo.GetByID(ctx, org.ID)
}

// DailyEventLimit reads the owner's override. A missing profile is not an
// error here — the caller falls back to the global default — because the
// profile is created alongside the first event, and the create path asks about
// the limit before that has happened.
func (s *service) DailyEventLimit(ctx context.Context, ownerID uuid.UUID) (int, bool, error) {
	org, err := s.repo.GetByOwner(ctx, ownerID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return 0, false, nil
		}
		return 0, false, err
	}
	if org == nil || org.DailyEventLimit == nil {
		return 0, false, nil
	}
	return *org.DailyEventLimit, true, nil
}

func (s *service) SetDailyEventLimit(ctx context.Context, id, actorID uuid.UUID, limit *int) error {
	return s.repo.SetDailyEventLimit(ctx, id, actorID, limit)
}

func (s *service) Verify(ctx context.Context, id, actorID uuid.UUID) error {
	return s.repo.Verify(ctx, id, actorID)
}

func (s *service) Reject(ctx context.Context, id, actorID uuid.UUID, reason string) error {
	if strings.TrimSpace(reason) == "" {
		return ErrReasonRequired
	}
	return s.repo.Reject(ctx, id, actorID, strings.TrimSpace(reason))
}

func (s *service) Revoke(ctx context.Context, id, actorID uuid.UUID, reason string) error {
	if strings.TrimSpace(reason) == "" {
		return ErrReasonRequired
	}
	return s.repo.Revoke(ctx, id, actorID, strings.TrimSpace(reason))
}

func (s *service) SetAutoVerify(ctx context.Context, id, actorID uuid.UUID, enabled bool) error {
	return s.repo.SetAutoVerify(ctx, id, actorID, enabled)
}

func (s *service) List(ctx context.Context, f ListFilter) ([]Organizer, error) {
	return s.repo.List(ctx, f)
}

func (s *service) GetWithHistory(ctx context.Context, id uuid.UUID) (*Organizer, []HistoryEntry, error) {
	org, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	hist, err := s.repo.History(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	return org, hist, nil
}

func (s *service) Overview(ctx context.Context) (Counts, error) { return s.repo.Counts(ctx) }
