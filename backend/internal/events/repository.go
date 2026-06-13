// Package events is the events domain module of the Lia monolith.
// It owns event persistence and business logic. The HTTP transport wires its
// service in via http.Module.SetEventsService; see internal/application.go.
package events

import (
	"context"
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

	// Insert the event and its category links atomically. No Returning("*"):
	// a nullable venue_id read back as NULL cannot be scanned into uuid.UUID.
	// ID and timestamps are set Go-side in BeforeInsert.
	err := r.db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
		if _, err := tx.Model(event).Insert(); err != nil {
			return fmt.Errorf("insert event %q: %w", event.Title, err)
		}
		for _, c := range event.Categories {
			if _, err := tx.Exec(
				`INSERT INTO event_categories (event_id, category_id) VALUES (?, ?)
				 ON CONFLICT DO NOTHING`,
				event.ID, c.ID,
			); err != nil {
				return fmt.Errorf("link event %s to category %s: %w", event.ID, c.ID, err)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("create event %q: %w", event.Title, err)
	}

	logger.Log().Infof("event created: %s (ID: %s)", event.Title, event.ID)
	return nil
}

func (r *pgRepository) GetByID(id uuid.UUID) (*models.Event, error) {
	event := &models.Event{ID: id}

	if err := r.db.Model(event).WherePK().Select(); err != nil {
		return nil, fmt.Errorf("get event %s from db: %w", id, err)
	}

	if err := r.loadCategories([]*models.Event{event}); err != nil {
		return nil, err
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

	if err := r.loadCategories(list); err != nil {
		return nil, err
	}

	return list, nil
}

// loadCategories populates Categories on each event via the event_categories
// join, in a single query (no N+1).
func (r *pgRepository) loadCategories(events []*models.Event) error {
	if len(events) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, 0, len(events))
	byID := make(map[uuid.UUID]*models.Event, len(events))
	for _, e := range events {
		ids = append(ids, e.ID)
		byID[e.ID] = e
		e.Categories = nil
	}

	var rows []struct {
		Slug    string    `pg:"slug"`
		Label   string    `pg:"label"`
		EventID uuid.UUID `pg:"event_id"`
		ID      uuid.UUID `pg:"id"`
	}
	if _, err := r.db.Query(&rows,
		`SELECT ec.event_id, c.id, c.slug, c.label
		 FROM event_categories ec
		 JOIN categories c ON c.id = ec.category_id
		 WHERE ec.event_id IN (?)
		 ORDER BY c.sort_order ASC`,
		pg.In(ids),
	); err != nil {
		return fmt.Errorf("load event categories: %w", err)
	}

	for _, row := range rows {
		if e, ok := byID[row.EventID]; ok {
			e.Categories = append(e.Categories, &models.Category{
				ID: row.ID, Slug: row.Slug, Label: row.Label,
			})
		}
	}
	return nil
}
