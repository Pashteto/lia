package models

import (
	"context"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
)

// Event represents a discoverable cultural event.
// Mirrors the domain model in docs/event_discovery_mvp_technical_stack.md §13.
//
//nolint:govet // field alignment kept for readability and conventional ordering
type Event struct {
	tableName struct{} `pg:"events,discard_unknown_columns"` //nolint:unused // go-pg table marker

	ID          uuid.UUID   `pg:"id,pk,type:uuid"`
	// OrganizerID / VenueID use the zero UUID to mean "unset" rather than SQL
	// NULL: go-pg + gofrs cannot scan NULL into a uuid field. Real FK references
	// arrive with the organizers / venues modules.
	OrganizerID uuid.UUID   `pg:"organizer_id,type:uuid,use_zero"`
	VenueID     uuid.UUID   `pg:"venue_id,type:uuid,use_zero"`
	CoverFileID uuid.UUID   `pg:"cover_file_id,type:uuid,use_zero"`
	// CoverURL is a transient field (not a DB column) populated by the repository
	// when cover_file_id is non-zero. It is the publicly fetchable URL of the cover image.
	CoverURL    string      `pg:"-"`
	Title       string      `pg:"title,notnull"`
	Description string      `pg:"description,use_zero"`
	// Venue is normalized into the venues entity (migration 000008). It is the
	// loaded read model (not a column); VenueID is the loose reference (zero
	// UUID = "no venue", no DB FK — see migration comment).
	Venue *Venue `pg:"-"`
	// Category is normalized into the categories taxonomy (migration 000006/7).
	// CategoryIDs is write-only input (set from the API), Categories is the
	// loaded read model; neither is a column on the events table.
	CategoryIDs []uuid.UUID `pg:"-"`
	Categories  []*Category `pg:"-"`
	Status      EventStatus `pg:"-"`
	StatusSQL   string      `pg:"status,use_zero"`
	Format      string      `pg:"format,use_zero"`
	PriceType   string      `pg:"price_type,use_zero"`
	PriceMin    *int64      `pg:"price_min"`
	PriceMax    *int64      `pg:"price_max"`
	ExternalURL string      `pg:"external_ticket_url,use_zero"`
	StartsAt    time.Time   `pg:"starts_at,notnull"`
	EndsAt      *time.Time  `pg:"ends_at"`
	PublishedAt *time.Time  `pg:"published_at"`
	CreatedAt   time.Time   `pg:"created_at,notnull,default:now()"`
	UpdatedAt   time.Time   `pg:"updated_at,notnull,default:now()"`
}

// Validate checks that the event has the required fields and valid data.
// ID and timestamps are managed by the repository/database and are not checked.
func (e *Event) Validate() error {
	if e.Title == "" {
		return newValidationError("title", "is required")
	}

	if e.StartsAt.IsZero() {
		return newValidationError("starts_at", "is required")
	}

	if e.EndsAt != nil && e.EndsAt.Before(e.StartsAt) {
		return newValidationError("ends_at", "must be after starts_at")
	}

	if e.Status < EventDraft || e.Status >= eventStatusUnsupported {
		return newValidationError("status", "invalid value")
	}

	return nil
}

// BeforeInsert generates a UUID if missing and serializes the Status enum.
func (e *Event) BeforeInsert(ctx context.Context) (context.Context, error) {
	if e.ID == uuid.Nil {
		newUUID, err := uuid.NewV4()
		if err != nil {
			return ctx, fmt.Errorf("generate UUID: %w", err)
		}
		e.ID = newUUID
	}

	status := e.Status.String()
	if status == "" {
		return ctx, fmt.Errorf("invalid status value: %d", e.Status)
	}
	e.StatusSQL = status

	// Set timestamps Go-side so the response carries them without a RETURNING
	// round-trip (the DB also defaults these via CURRENT_TIMESTAMP).
	now := time.Now()
	if e.CreatedAt.IsZero() {
		e.CreatedAt = now
	}
	if e.UpdatedAt.IsZero() {
		e.UpdatedAt = now
	}

	return ctx, nil
}

// BeforeUpdate serializes the Status enum and refreshes the timestamp.
func (e *Event) BeforeUpdate(ctx context.Context) (context.Context, error) {
	status := e.Status.String()
	if status == "" {
		return ctx, fmt.Errorf("invalid status value: %d", e.Status)
	}
	e.StatusSQL = status
	e.UpdatedAt = time.Now()

	return ctx, nil
}

// AfterSelect converts the StatusSQL string back to the Status enum.
func (e *Event) AfterSelect(_ context.Context) error {
	status, err := EventStatusFromString(e.StatusSQL)
	if err != nil {
		return fmt.Errorf("parse event status: %w", err)
	}
	e.Status = status

	return nil
}
