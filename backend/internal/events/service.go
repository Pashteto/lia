package events

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gofrs/uuid"

	"github.com/Pashteto/lia/internal/categories"
	"github.com/Pashteto/lia/internal/models"
	"github.com/Pashteto/lia/internal/venues"
	"github.com/Pashteto/lia/pkg/logger"
)

// Domain errors. The HTTP layer maps these to status codes.
var (
	// ErrInvalidInput indicates the event failed validation.
	ErrInvalidInput = errors.New("invalid input")
	// ErrNotFound indicates no event matched the query.
	ErrNotFound = errors.New("not found")
	// ErrQuotaExceeded indicates the organizer has reached their monthly event limit.
	ErrQuotaExceeded = errors.New("monthly event limit reached")
	// ErrNotEditable indicates the event is in a status that cannot be edited
	// (only draft events are editable).
	ErrNotEditable = errors.New("not editable")
	// ErrCapacityBelowOccupied indicates a capacity change below the number of
	// occupied seats (going+accepted). The message carries the occupied count.
	ErrCapacityBelowOccupied = errors.New("capacity below occupied seats")
)

// startOfMonthMoscow returns midnight on the first day of t's calendar month
// in the Europe/Moscow timezone.
func startOfMonthMoscow(t time.Time) time.Time {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		// Fall back to UTC if timezone data is unavailable.
		loc = time.UTC
	}
	moscow := t.In(loc)
	return time.Date(moscow.Year(), moscow.Month(), 1, 0, 0, 0, 0, loc)
}

// Service is the events business-logic interface.
type Service interface {
	// Create validates and persists a new event.
	Create(ctx context.Context, event *models.Event) error
	// Update applies a partial update to an event owned by ownerID. Only draft
	// events are editable; non-owners get ErrNotFound (existence is not leaked).
	Update(ctx context.Context, id, ownerID uuid.UUID, p UpdateParams) (*models.Event, error)
	// GetByID returns a single event by its string UUID.
	GetByID(ctx context.Context, id string) (*models.Event, error)
	// List returns events filtered by status, optionally restricted to a
	// [from, to) start-time window (nil bounds mean "unbounded" on that side).
	List(ctx context.Context, status string, from, to *time.Time, organizerOwnerID *uuid.UUID) ([]*models.Event, error)
	// ListByOrganizer returns all events (any status) created by the given user.
	ListByOrganizer(ctx context.Context, organizerID uuid.UUID) ([]*models.Event, error)
	// Nearby returns published events nearest to (lat, lon), within 50 km,
	// up to limit results. Both lat and lon are required.
	Nearby(ctx context.Context, lat, lon *float64, limit int) ([]*NearbyResult, error)
	// ListForCalendar returns published events in [from, to) created by any of
	// the given organizer (owner) ids, fully enriched. Empty organizerIDs yields
	// no rows. Used by the personal calendar's followed-organizers stream.
	ListForCalendar(ctx context.Context, organizerIDs []uuid.UUID, from, to time.Time) ([]*models.Event, error)
	// GetEnriched returns the events with the given ids, fully enriched
	// (categories, venue, cover, organizer, seats). Used to re-enrich a merged
	// set of calendar rows uniformly.
	GetEnriched(ctx context.Context, ids []uuid.UUID) ([]*models.Event, error)
}

// calendarListLimit caps a single calendar range query. A personal calendar
// window (≤ ~3 months across followed organizers) stays well under this.
const calendarListLimit = 500

// UpdateParams is a partial event update. A nil pointer field means "preserve
// the current value"; a non-nil field overwrites it. CategoryIDs is nil to
// preserve, non-nil to replace the category set.
type UpdateParams struct {
	Title                   *string
	Description             *string
	Format                  *string
	PriceType               *string
	PriceMin                *int64
	PriceMax                *int64
	ExternalURL             *string
	VenueID                 *uuid.UUID
	CoverFileID             *uuid.UUID
	CategoryIDs             []uuid.UUID
	StartsAt                *time.Time
	EndsAt                  *time.Time
	Status                  *string
	SignupMode              *string
	CuratorQuestion         *string
	ExternalRegistrationURL *string
	Capacity                *int
}

// ownerSettableStatus reports whether an owner may set the given status via the
// edit endpoint. Moderation statuses (pending_review, rejected) are excluded.
func ownerSettableStatus(s models.EventStatus) bool {
	switch s {
	case models.EventDraft, models.EventPublished, models.EventCancelled:
		return true
	default:
		return false
	}
}

// isNoRows reports whether err is (or wraps) a go-pg "no rows" error. Mirrors
// the detection used in GetByID.
func isNoRows(err error) bool {
	if wrapped := errors.Unwrap(err); wrapped != nil &&
		wrapped.Error() == "pg: no rows in result set" {
		return true
	}
	return false
}

// CategoryValidator resolves and validates category ids. Satisfied by
// categories.Service. Kept as a local interface so the events service stays
// testable with a fake.
type CategoryValidator interface {
	Validate(ctx context.Context, ids []uuid.UUID) ([]*models.Category, error)
}

// VenueValidator resolves and validates a venue id. Satisfied by venues.Service.
type VenueValidator interface {
	Validate(ctx context.Context, id uuid.UUID) (*models.Venue, error)
}

type service struct {
	repo         Repository
	categories   CategoryValidator
	venues       VenueValidator
	monthlyLimit int
}

// NewService creates an events service backed by the given repository, a
// category validator, a venue validator, and a monthly creation limit per
// organizer. monthlyLimit <= 0 means unlimited.
func NewService(repo Repository, categories CategoryValidator, venues VenueValidator, monthlyLimit int) Service {
	return &service{repo: repo, categories: categories, venues: venues, monthlyLimit: monthlyLimit}
}

func (s *service) Create(ctx context.Context, event *models.Event) error {
	if event == nil {
		return fmt.Errorf("%w: event is required", ErrInvalidInput)
	}

	if err := event.Validate(); err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidInput, err.Error())
	}

	resolved, err := s.categories.Validate(ctx, event.CategoryIDs)
	if err != nil {
		if errors.Is(err, categories.ErrInvalidInput) {
			return fmt.Errorf("%w: %s", ErrInvalidInput, err.Error())
		}
		return fmt.Errorf("validate categories: %w", err)
	}
	event.Categories = resolved

	venue, err := s.venues.Validate(ctx, event.VenueID)
	if err != nil {
		if errors.Is(err, venues.ErrInvalidInput) {
			return fmt.Errorf("%w: %s", ErrInvalidInput, err.Error())
		}
		return fmt.Errorf("validate venue: %w", err)
	}
	event.Venue = venue

	// Quota check: if a monthly limit is configured, reject once the organizer
	// has reached it for the current calendar month (Europe/Moscow).
	if s.monthlyLimit > 0 && event.OrganizerID != uuid.Nil {
		since := startOfMonthMoscow(time.Now())
		n, err := s.repo.CountByOrganizerSince(event.OrganizerID, since)
		if err != nil {
			return fmt.Errorf("quota check: %w", err)
		}
		if n >= s.monthlyLimit {
			return fmt.Errorf("%w: %d/%d this month", ErrQuotaExceeded, n, s.monthlyLimit)
		}
	}

	if err := s.repo.Create(event); err != nil {
		return fmt.Errorf("create event: %w", err)
	}

	logger.Log().Infof("event created via service: %s", event.ID)
	return nil
}

func (s *service) GetByID(_ context.Context, id string) (*models.Event, error) {
	parsed, err := uuid.FromString(id)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid event id %q", ErrInvalidInput, id)
	}

	event, err := s.repo.GetByID(parsed)
	if err != nil {
		if wrapped := errors.Unwrap(err); wrapped != nil &&
			wrapped.Error() == "pg: no rows in result set" {
			return nil, fmt.Errorf("%w: event %s", ErrNotFound, id)
		}
		return nil, fmt.Errorf("get event by id: %w", err)
	}

	return event, nil
}

func (s *service) Update(ctx context.Context, id, ownerID uuid.UUID, p UpdateParams) (*models.Event, error) {
	if id == uuid.Nil {
		return nil, fmt.Errorf("%w: id is required", ErrInvalidInput)
	}

	event, err := s.repo.GetByID(id)
	if err != nil {
		if isNoRows(err) {
			return nil, fmt.Errorf("%w: event %s", ErrNotFound, id)
		}
		return nil, fmt.Errorf("get event by id: %w", err)
	}

	// Non-owner access is indistinguishable from not-found (no existence leak).
	if event.OrganizerID != ownerID {
		return nil, fmt.Errorf("%w: event %s", ErrNotFound, id)
	}

	// Draft and published events are editable. Moderation/terminal statuses are not.
	if event.Status != models.EventDraft && event.Status != models.EventPublished {
		return nil, fmt.Errorf("%w: event %s is %s", ErrNotEditable, id, event.Status)
	}
	wasPublished := event.Status == models.EventPublished

	// Signup mode is locked once published (would strip meaning from existing RSVPs).
	if wasPublished && p.SignupMode != nil && *p.SignupMode != event.SignupMode {
		return nil, fmt.Errorf("%w: режим записи нельзя изменить после публикации", ErrInvalidInput)
	}

	if p.Title != nil {
		event.Title = *p.Title
	}
	if p.Description != nil {
		event.Description = *p.Description
	}
	if p.Format != nil {
		event.Format = *p.Format
	}
	if p.PriceType != nil {
		event.PriceType = *p.PriceType
	}
	if p.PriceMin != nil {
		event.PriceMin = p.PriceMin
	}
	if p.PriceMax != nil {
		event.PriceMax = p.PriceMax
	}
	if p.ExternalURL != nil {
		event.ExternalURL = *p.ExternalURL
	}
	if p.SignupMode != nil {
		event.SignupMode = *p.SignupMode
	}
	if p.CuratorQuestion != nil {
		event.CuratorQuestion = *p.CuratorQuestion
	}
	if p.ExternalRegistrationURL != nil {
		event.ExternalRegistrationURL = *p.ExternalRegistrationURL
	}
	if p.VenueID != nil {
		event.VenueID = *p.VenueID
	}
	if p.CoverFileID != nil {
		event.CoverFileID = *p.CoverFileID
	}
	if p.StartsAt != nil {
		event.StartsAt = *p.StartsAt
	}
	if p.EndsAt != nil {
		event.EndsAt = p.EndsAt
	}

	if p.Status != nil {
		target, err := models.EventStatusFromString(*p.Status)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrInvalidInput, err.Error())
		}
		if !ownerSettableStatus(target) {
			return nil, fmt.Errorf("%w: status %q is not settable", ErrInvalidInput, *p.Status)
		}
		event.Status = target
		if target == models.EventPublished && event.PublishedAt == nil {
			now := time.Now()
			event.PublishedAt = &now
		}
	}

	if p.CategoryIDs != nil {
		resolved, err := s.categories.Validate(ctx, p.CategoryIDs)
		if err != nil {
			if errors.Is(err, categories.ErrInvalidInput) {
				return nil, fmt.Errorf("%w: %s", ErrInvalidInput, err.Error())
			}
			return nil, fmt.Errorf("validate categories: %w", err)
		}
		event.CategoryIDs = p.CategoryIDs
		event.Categories = resolved
	}

	if p.VenueID != nil {
		venue, err := s.venues.Validate(ctx, event.VenueID)
		if err != nil {
			if errors.Is(err, venues.ErrInvalidInput) {
				return nil, fmt.Errorf("%w: %s", ErrInvalidInput, err.Error())
			}
			return nil, fmt.Errorf("validate venue: %w", err)
		}
		event.Venue = venue
	}

	if err := event.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidInput, err.Error())
	}

	// Pre-validate a capacity change BEFORE any field write, so a rejection
	// (invalid value or below current occupancy) never leaves a partial,
	// unaudited edit (e.g. a title change) committed. SetCapacityTx below
	// remains the authoritative, race-safe apply (re-checked under FOR UPDATE).
	if p.Capacity != nil {
		if *p.Capacity <= 0 {
			return nil, fmt.Errorf("%w: лимит мест должен быть больше нуля", ErrInvalidInput)
		}
		occupied, err := s.repo.CountOccupiedSeats(id)
		if err != nil {
			return nil, fmt.Errorf("count occupied seats: %w", err)
		}
		if *p.Capacity < occupied {
			return nil, fmt.Errorf("%w: %d occupied", ErrCapacityBelowOccupied, occupied)
		}
	}

	if err := s.repo.Update(event); err != nil {
		return nil, fmt.Errorf("update event: %w", err)
	}

	// Capacity change: guarded, atomic, promotes the waitlist. The narrow race
	// (occupancy changing between the pre-check above and this locked apply)
	// is still safe: SetCapacityTx re-checks under FOR UPDATE and can 409 here.
	if p.Capacity != nil {
		if _, err := s.repo.SetCapacityTx(id, p.Capacity); err != nil {
			if errors.Is(err, ErrCapacityBelowOccupied) {
				return nil, err // handler maps to 409
			}
			return nil, fmt.Errorf("reconcile capacity: %w", err)
		}
	}

	// Audit published edits (draft edits are not audited, matching create/publish).
	if wasPublished {
		if err := s.repo.WriteEditAudit(ctx, id, ownerID); err != nil {
			return nil, fmt.Errorf("write edit audit: %w", err)
		}
	}

	reloaded, err := s.repo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("reload event: %w", err)
	}
	return reloaded, nil
}

func (s *service) List(_ context.Context, status string, from, to *time.Time, organizerOwnerID *uuid.UUID) ([]*models.Event, error) {
	if status != "" {
		if _, err := models.EventStatusFromString(status); err != nil {
			return nil, fmt.Errorf("%w: %s", ErrInvalidInput, err.Error())
		}
	}

	filter := ListFilter{Status: status, From: from, To: to}
	if organizerOwnerID != nil {
		filter.OrganizerIDs = []uuid.UUID{*organizerOwnerID}
	}

	list, err := s.repo.List(filter)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}

	return list, nil
}

func (s *service) ListByOrganizer(_ context.Context, organizerID uuid.UUID) ([]*models.Event, error) {
	list, err := s.repo.List(ListFilter{OrganizerID: organizerID})
	if err != nil {
		return nil, fmt.Errorf("list events by organizer: %w", err)
	}

	return list, nil
}

func (s *service) Nearby(_ context.Context, lat, lon *float64, limit int) ([]*NearbyResult, error) {
	if lat == nil || lon == nil {
		return nil, fmt.Errorf("%w: lat and lon are required", ErrInvalidInput)
	}
	if *lat < -90 || *lat > 90 || *lon < -180 || *lon > 180 {
		return nil, fmt.Errorf("%w: coordinates out of range", ErrInvalidInput)
	}
	res, err := s.repo.Nearby(*lat, *lon, limit)
	if err != nil {
		return nil, fmt.Errorf("nearby events: %w", err)
	}
	return res, nil
}

func (s *service) ListForCalendar(_ context.Context, organizerIDs []uuid.UUID, from, to time.Time) ([]*models.Event, error) {
	if len(organizerIDs) == 0 {
		return nil, nil
	}
	list, err := s.repo.List(ListFilter{
		Status:       models.EventPublished.String(),
		OrganizerIDs: organizerIDs,
		From:         &from,
		To:           &to,
		Limit:        calendarListLimit,
	})
	if err != nil {
		return nil, fmt.Errorf("list events for calendar: %w", err)
	}
	return list, nil
}

func (s *service) GetEnriched(_ context.Context, ids []uuid.UUID) ([]*models.Event, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	// Limit must cover the exact id set so a merged calendar never silently
	// truncates (the default list limit is far smaller than a busy window).
	list, err := s.repo.List(ListFilter{IDs: ids, Limit: len(ids)})
	if err != nil {
		return nil, fmt.Errorf("get enriched events: %w", err)
	}
	return list, nil
}
