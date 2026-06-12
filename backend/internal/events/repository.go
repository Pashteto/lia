// Package events is the events domain module of the Lia monolith.
// It owns event persistence and business logic. The HTTP transport wires its
// service in via http.Module.SetEventsService; see internal/application.go.
package events

import (
	"fmt"

	"github.com/go-pg/pg/v10"
	"github.com/gofrs/uuid"

	"github.com/Pashteto/lia/internal/models"
	"github.com/Pashteto/lia/pkg/logger"
)

// ListFilter narrows an event listing. Zero values mean "no constraint".
type ListFilter struct {
	// Status, when non-empty, restricts to events in that publication state.
	Status string
	// Limit caps the number of rows returned (defaults to DefaultListLimit).
	Limit int
}

// DefaultListLimit is applied when ListFilter.Limit is unset.
const DefaultListLimit = 50

// Repository defines event persistence operations.
type Repository interface {
	// Create inserts a new event (ID auto-generated via BeforeInsert).
	Create(event *models.Event) error
	// GetByID returns a single event by primary key.
	GetByID(id uuid.UUID) (*models.Event, error)
	// List returns events matching the filter, newest start first.
	List(filter ListFilter) ([]*models.Event, error)
}

// pgRepository is a go-pg backed Repository.
type pgRepository struct {
	db *pg.DB
}

// NewRepository creates a PostgreSQL-backed event repository.
func NewRepository(db *pg.DB) Repository {
	return &pgRepository{db: db}
}

func (r *pgRepository) Create(event *models.Event) error {
	logger.Log().Infof("creating event: %s", event.Title)

	// No Returning("*"): a nullable venue_id read back as NULL cannot be scanned
	// into uuid.UUID. The ID and timestamps are set Go-side in BeforeInsert.
	if _, err := r.db.Model(event).Insert(); err != nil {
		return fmt.Errorf("insert event %q into db: %w", event.Title, err)
	}

	logger.Log().Infof("event created: %s (ID: %s)", event.Title, event.ID)
	return nil
}

func (r *pgRepository) GetByID(id uuid.UUID) (*models.Event, error) {
	event := &models.Event{ID: id}

	if err := r.db.Model(event).WherePK().Select(); err != nil {
		return nil, fmt.Errorf("get event %s from db: %w", id, err)
	}

	return event, nil
}

func (r *pgRepository) List(filter ListFilter) ([]*models.Event, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = DefaultListLimit
	}

	var list []*models.Event
	query := r.db.Model(&list)

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	if err := query.Order("starts_at ASC").Limit(limit).Select(); err != nil {
		return nil, fmt.Errorf("list events from db: %w", err)
	}

	return list, nil
}
