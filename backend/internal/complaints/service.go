// Package complaints implements user-filed reports against events and the
// staff resolution workflow (grouped per event; takedown reuses the moderation
// domain). See spec docs/superpowers/specs/2026-06-26-complaints-reports-design.md.
package complaints

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/gofrs/uuid"

	"github.com/Pashteto/lia/internal/moderation"
)

var validCategories = map[string]bool{
	"spam": true, "fraud": true, "inappropriate": true, "duplicate": true, "other": true,
}

// Domain errors (mapped to HTTP status by the handlers).
var (
	ErrInvalidCategory    = errors.New("complaints: invalid category")       // 400
	ErrTargetNotFound     = errors.New("complaints: target not found")       // 404
	ErrResolutionRequired = errors.New("complaints: resolution required")    // 400
	ErrInvalidAction      = errors.New("complaints: invalid resolve action") // 400
)

// Complaint is one filed report.
type Complaint struct {
	ID             uuid.UUID
	TargetType     string
	TargetID       uuid.UUID
	ReporterUserID uuid.UUID
	Category       string
	Note           string
	Status         string
	Resolution     string
	ResolvedBy     *uuid.UUID
	ResolvedAt     *time.Time
	CreatedAt      time.Time
}

// EventReportGroup is one row of the grouped admin inbox.
type EventReportGroup struct {
	TargetID    uuid.UUID      `json:"event_id"`
	EventTitle  string         `json:"event_title"`
	EventStatus string         `json:"event_status"`
	ReportCount int            `json:"report_count"`
	Categories  map[string]int `json:"categories"`
	LatestNote  string         `json:"latest_note"`
	LatestAt    time.Time      `json:"latest_at"`
}

// Repository persists complaints and resolves them atomically (with audit).
type Repository interface {
	Insert(ctx context.Context, c Complaint) (bool, error) // false = idempotent skip (open dup)
	InboxGroups(ctx context.Context) ([]EventReportGroup, error)
	TargetComplaints(ctx context.Context, targetType string, targetID uuid.UUID) ([]Complaint, error)
	ResolveOpenForTarget(ctx context.Context, targetType string, targetID, actorID uuid.UUID, status, resolution string) (int, error)
	OpenEventCount(ctx context.Context) (int, error)
	EventExists(ctx context.Context, id uuid.UUID) (bool, error)
}

// Service is the complaints use-case layer.
type Service interface {
	Submit(ctx context.Context, reporterID uuid.UUID, targetType string, targetID uuid.UUID, category, note string) (bool, error)
	ListInbox(ctx context.Context) ([]EventReportGroup, error)
	TargetDetail(ctx context.Context, targetType string, targetID uuid.UUID) ([]Complaint, error)
	Resolve(ctx context.Context, targetType string, targetID, actorID uuid.UUID, action, resolution string) error
	OpenEventCount(ctx context.Context) (int, error)
}

type service struct {
	repo Repository
	mod  moderation.Service
}

// NewService returns a complaints Service. mod is used by the takedown branch
// of Resolve to reuse the moderation status transition.
func NewService(repo Repository, mod moderation.Service) Service {
	return &service{repo: repo, mod: mod}
}

func (s *service) Submit(ctx context.Context, reporterID uuid.UUID, targetType string, targetID uuid.UUID, category, note string) (bool, error) {
	if targetType == "" {
		targetType = "event"
	}
	if !validCategories[category] {
		return false, ErrInvalidCategory
	}
	exists, err := s.repo.EventExists(ctx, targetID)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, ErrTargetNotFound
	}
	return s.repo.Insert(ctx, Complaint{
		TargetType:     targetType,
		TargetID:       targetID,
		ReporterUserID: reporterID,
		Category:       category,
		Note:           strings.TrimSpace(note),
		Status:         "open",
	})
}

func (s *service) ListInbox(ctx context.Context) ([]EventReportGroup, error) {
	return s.repo.InboxGroups(ctx)
}

func (s *service) TargetDetail(ctx context.Context, targetType string, targetID uuid.UUID) ([]Complaint, error) {
	if targetType == "" {
		targetType = "event"
	}
	return s.repo.TargetComplaints(ctx, targetType, targetID)
}

func (s *service) Resolve(ctx context.Context, targetType string, targetID, actorID uuid.UUID, action, resolution string) error {
	if targetType == "" {
		targetType = "event"
	}
	resolution = strings.TrimSpace(resolution)
	switch action {
	case "takedown":
		if resolution == "" {
			return ErrResolutionRequired
		}
		// Reuse the moderation transition (its own tx: status + history + audit).
		// On ErrInvalidTransition (event not published) we surface it and leave
		// the complaints open.
		if err := s.mod.Takedown(ctx, targetID, actorID, resolution); err != nil {
			return err
		}
		_, err := s.repo.ResolveOpenForTarget(ctx, targetType, targetID, actorID, "resolved", resolution)
		return err
	case "dismiss":
		_, err := s.repo.ResolveOpenForTarget(ctx, targetType, targetID, actorID, "dismissed", resolution)
		return err
	default:
		return ErrInvalidAction
	}
}

func (s *service) OpenEventCount(ctx context.Context) (int, error) {
	return s.repo.OpenEventCount(ctx)
}
