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
	ID                 uuid.UUID
	OwnerUserID        uuid.UUID
	Name               string
	Description        string
	WebsiteURL         string
	LogoFileID         uuid.UUID
	VerificationStatus string
	AutoVerify         bool
	VerifiedAt         *time.Time
	LatestReason       string // populated on reads when status == rejected
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
}

// VerifiedOrg is the minimal verified-org read-model for the event badge.
type VerifiedOrg struct {
	ID      uuid.UUID
	Name    string
	LogoKey string // files.storage_key; resolved to a URL by the caller
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
	List(ctx context.Context, f ListFilter) ([]Organizer, error)
	History(ctx context.Context, id uuid.UUID) ([]HistoryEntry, error)
	Counts(ctx context.Context) (Counts, error)
	VerifiedByOwners(ctx context.Context, ownerIDs []uuid.UUID) (map[uuid.UUID]VerifiedOrg, error)
}

// Service is the organizers use-case layer.
type Service interface {
	GetByOwner(ctx context.Context, ownerID uuid.UUID) (*Organizer, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Organizer, error)
	Upsert(ctx context.Context, ownerID uuid.UUID, in Input) (*Organizer, error)
	Submit(ctx context.Context, ownerID uuid.UUID) (newStatus string, err error)
	Verify(ctx context.Context, id, actorID uuid.UUID) error
	Reject(ctx context.Context, id, actorID uuid.UUID, reason string) error
	Revoke(ctx context.Context, id, actorID uuid.UUID, reason string) error
	SetAutoVerify(ctx context.Context, id, actorID uuid.UUID, enabled bool) error
	List(ctx context.Context, f ListFilter) ([]Organizer, error)
	GetWithHistory(ctx context.Context, id uuid.UUID) (*Organizer, []HistoryEntry, error)
	Overview(ctx context.Context) (Counts, error)
	VerifiedByOwners(ctx context.Context, ownerIDs []uuid.UUID) (map[uuid.UUID]VerifiedOrg, error)
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

func (s *service) VerifiedByOwners(ctx context.Context, ownerIDs []uuid.UUID) (map[uuid.UUID]VerifiedOrg, error) {
	return s.repo.VerifiedByOwners(ctx, ownerIDs)
}
