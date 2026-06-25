// Package events is the events domain module of the Lia monolith.
// It owns event persistence and business logic. The HTTP transport wires its
// service in via http.Module.SetEventsService; see internal/application.go.
package events

import (
	"context"
	"fmt"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/gofrs/uuid"

	"github.com/Pashteto/lia/internal/models"
	"github.com/Pashteto/lia/internal/storage"
	"github.com/Pashteto/lia/pkg/logger"
)

// ListFilter narrows an event listing. Zero values mean "no constraint".
type ListFilter struct {
	// Status, when non-empty, restricts to events in that publication state.
	Status string
	// OrganizerID, when non-zero, restricts to events created by that user
	// (used by the "my events" listing). Any status is returned.
	OrganizerID uuid.UUID
	// Limit caps the number of rows returned (defaults to DefaultListLimit).
	Limit int
}

// DefaultListLimit is applied when ListFilter.Limit is unset.
const DefaultListLimit = 50

// NearbyResult wraps an event with its distance from the query point.
type NearbyResult struct {
	Event     *models.Event
	DistanceM float64
}

// nearbyRow is an internal scan target that embeds Event and adds DistanceM.
type nearbyRow struct {
	models.Event
	DistanceM float64 `pg:"distance_m"`
}

// Repository defines event persistence operations.
type Repository interface {
	// Create inserts a new event (ID auto-generated via BeforeInsert).
	Create(event *models.Event) error
	// GetByID returns a single event by primary key.
	GetByID(id uuid.UUID) (*models.Event, error)
	// List returns events matching the filter, newest start first.
	List(filter ListFilter) ([]*models.Event, error)
	// Nearby returns published events whose venue has coordinates, ordered
	// nearest-first, capped at 50 km from (lat, lon).
	Nearby(lat, lon float64, limit int) ([]*NearbyResult, error)
	// Update persists changes to an existing event. When event.CategoryIDs is
	// non-nil, the event_categories links are replaced to match. Returns
	// pg.ErrNoRows when no row matches.
	Update(event *models.Event) error
	// CountByOrganizerSince returns the number of events created by the given
	// organizer at or after since (all statuses, draft + published).
	CountByOrganizerSince(organizer uuid.UUID, since time.Time) (int, error)
}

// pgRepository is a go-pg backed Repository.
type pgRepository struct {
	db    *pg.DB
	store storage.Storage // nil when storage is disabled
}

// NewRepository creates a PostgreSQL-backed event repository.
// store may be nil when the storage backend is disabled — loadCover
// will no-op safely in that case.
func NewRepository(db *pg.DB, store storage.Storage) Repository {
	return &pgRepository{db: db, store: store}
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

func (r *pgRepository) Update(event *models.Event) error {
	logger.Log().Infof("updating event: %s", event.ID)

	err := r.db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
		res, err := tx.Model(event).
			Column(
				"title", "description", "venue_id", "cover_file_id", "status",
				"format", "price_type", "price_min", "price_max",
				"external_ticket_url", "starts_at", "ends_at", "published_at",
				"signup_mode", "capacity", "curator_question", "external_registration_url",
				"updated_at",
			).
			WherePK().
			Update()
		if err != nil {
			return fmt.Errorf("update event %s: %w", event.ID, err)
		}
		if res.RowsAffected() == 0 {
			return pg.ErrNoRows
		}

		// Replace category links only when the caller provided a new set.
		// A nil CategoryIDs means "preserve existing links".
		if event.CategoryIDs != nil {
			if _, err := tx.Exec(`DELETE FROM event_categories WHERE event_id = ?`, event.ID); err != nil {
				return fmt.Errorf("clear event %s categories: %w", event.ID, err)
			}
			for _, cid := range event.CategoryIDs {
				if _, err := tx.Exec(
					`INSERT INTO event_categories (event_id, category_id) VALUES (?, ?)
					 ON CONFLICT DO NOTHING`,
					event.ID, cid,
				); err != nil {
					return fmt.Errorf("link event %s to category %s: %w", event.ID, cid, err)
				}
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("update event %s: %w", event.ID, err)
	}
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

	if err := r.loadVenues([]*models.Event{event}); err != nil {
		return nil, err
	}

	if err := r.loadCover([]*models.Event{event}); err != nil {
		return nil, err
	}

	if err := r.loadOrganizers([]*models.Event{event}); err != nil {
		return nil, err
	}

	if err := r.loadSeats([]*models.Event{event}); err != nil {
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

	if filter.OrganizerID != uuid.Nil {
		query = query.Where("organizer_id = ?", filter.OrganizerID)
	}

	if err := query.Order("starts_at ASC").Limit(limit).Select(); err != nil {
		return nil, fmt.Errorf("list events from db: %w", err)
	}

	if err := r.loadCategories(list); err != nil {
		return nil, err
	}

	if err := r.loadVenues(list); err != nil {
		return nil, err
	}

	if err := r.loadCover(list); err != nil {
		return nil, err
	}

	if err := r.loadOrganizers(list); err != nil {
		return nil, err
	}

	if err := r.loadSeats(list); err != nil {
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

// Nearby returns published events whose venue has coordinates, ordered
// nearest-first, within 50 km of the given point. limit defaults to
// DefaultListLimit when <= 0.
func (r *pgRepository) Nearby(lat, lon float64, limit int) ([]*NearbyResult, error) {
	if limit <= 0 {
		limit = DefaultListLimit
	}
	var rows []nearbyRow
	_, err := r.db.Query(&rows, `
		SELECT e.*, ST_Distance(v.geog, ST_SetSRID(ST_MakePoint(?0, ?1), 4326)::geography) AS distance_m
		FROM events e
		JOIN venues v ON v.id = e.venue_id
		WHERE v.geog IS NOT NULL
		  AND e.status = 'published'
		  AND ST_DWithin(v.geog, ST_SetSRID(ST_MakePoint(?0, ?1), 4326)::geography, 50000)
		ORDER BY v.geog <-> ST_SetSRID(ST_MakePoint(?0, ?1), 4326)::geography
		LIMIT ?2`,
		lon, lat, limit)
	if err != nil {
		return nil, fmt.Errorf("nearby events from db: %w", err)
	}
	events := make([]*models.Event, len(rows))
	results := make([]*NearbyResult, len(rows))
	for i := range rows {
		e := rows[i].Event
		// go-pg does not call AfterSelect for raw-SQL scans; invoke it manually
		// so Event.Status (the Go enum) is populated from Event.StatusSQL.
		if err := e.AfterSelect(context.Background()); err != nil {
			return nil, fmt.Errorf("nearby events: scan row %d: %w", i, err)
		}
		events[i] = &e
		results[i] = &NearbyResult{Event: &e, DistanceM: rows[i].DistanceM}
	}
	if err := r.loadCategories(events); err != nil {
		return nil, err
	}
	if err := r.loadVenues(events); err != nil {
		return nil, err
	}
	if err := r.loadCover(events); err != nil {
		return nil, err
	}
	return results, nil
}

// loadVenues populates Venue on each event whose venue_id is set (non-zero),
// in a single query (no N+1).
func (r *pgRepository) loadVenues(events []*models.Event) error {
	if len(events) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, 0, len(events))
	seen := make(map[uuid.UUID]struct{})
	for _, e := range events {
		if e.VenueID != uuid.Nil {
			if _, ok := seen[e.VenueID]; !ok {
				seen[e.VenueID] = struct{}{}
				ids = append(ids, e.VenueID)
			}
		}
	}
	if len(ids) == 0 {
		return nil
	}

	var venues []*models.Venue
	if err := r.db.Model(&venues).Where("id IN (?)", pg.In(ids)).Select(); err != nil {
		return fmt.Errorf("load event venues: %w", err)
	}

	byID := make(map[uuid.UUID]*models.Venue, len(venues))
	for _, v := range venues {
		byID[v.ID] = v
	}
	// A venue_id with no matching row (e.g. a stale/dangling reference) is left
	// as e.Venue == nil — intentional, since venue_id is a loose reference (no FK).
	for _, e := range events {
		if v, ok := byID[e.VenueID]; ok {
			e.Venue = v
		}
	}
	return nil
}

// loadCover populates CoverURL on each event whose cover_file_id is set
// (non-zero), in a single query (no N+1). No-ops when store is nil (storage
// disabled) or when no event has a cover set.
func (r *pgRepository) loadCover(events []*models.Event) error {
	if r.store == nil || len(events) == 0 {
		return nil
	}
	// Collect unique non-zero cover_file_ids.
	ids := make([]uuid.UUID, 0, len(events))
	seen := make(map[uuid.UUID]struct{})
	for _, e := range events {
		if e.CoverFileID != uuid.Nil {
			if _, ok := seen[e.CoverFileID]; !ok {
				seen[e.CoverFileID] = struct{}{}
				ids = append(ids, e.CoverFileID)
			}
		}
	}
	if len(ids) == 0 {
		return nil
	}

	var rows []struct {
		ID         uuid.UUID `pg:"id"`
		StorageKey string    `pg:"storage_key"`
	}
	if _, err := r.db.Query(&rows,
		`SELECT id, storage_key FROM files WHERE id IN (?)`,
		pg.In(ids),
	); err != nil {
		return fmt.Errorf("load cover files: %w", err)
	}

	byID := make(map[uuid.UUID]string, len(rows))
	for _, row := range rows {
		byID[row.ID] = row.StorageKey
	}
	for _, e := range events {
		if key, ok := byID[e.CoverFileID]; ok {
			e.CoverURL = r.store.URL(key)
		}
	}
	return nil
}

// loadOrganizers populates the public Organizer read-model on each event from
// the users table in a single query (no N+1). It exposes display name + avatar
// only — email and other private user fields are deliberately excluded, since
// event responses are public. Avatar URL is resolved via storage when set.
func (r *pgRepository) loadOrganizers(events []*models.Event) error {
	if len(events) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, 0, len(events))
	seen := make(map[uuid.UUID]struct{})
	for _, e := range events {
		if e.OrganizerID != uuid.Nil {
			if _, ok := seen[e.OrganizerID]; !ok {
				seen[e.OrganizerID] = struct{}{}
				ids = append(ids, e.OrganizerID)
			}
		}
	}
	if len(ids) == 0 {
		return nil
	}

	var rows []struct {
		UUID       uuid.UUID `pg:"uuid"`
		Name       string    `pg:"name,use_zero"`
		StorageKey string    `pg:"storage_key,use_zero"`
	}
	if _, err := r.db.Query(&rows,
		`SELECT u.uuid, u.name, COALESCE(f.storage_key, '') AS storage_key
		   FROM users u
		   LEFT JOIN files f ON f.id = u.avatar_file_id
		  WHERE u.uuid IN (?)`,
		pg.In(ids),
	); err != nil {
		return fmt.Errorf("load organizers: %w", err)
	}

	type orgInfo struct {
		name       string
		storageKey string
	}
	byID := make(map[uuid.UUID]orgInfo, len(rows))
	for _, row := range rows {
		byID[row.UUID] = orgInfo{name: row.Name, storageKey: row.StorageKey}
	}
	for _, e := range events {
		info, ok := byID[e.OrganizerID]
		if !ok {
			continue
		}
		org := &models.Organizer{UUID: e.OrganizerID, Name: info.name}
		if info.storageKey != "" && r.store != nil {
			org.AvatarURL = r.store.URL(info.storageKey)
		}
		e.Organizer = org
	}
	return nil
}

// loadSeats populates SeatsRemaining on each event that has a capacity, by
// counting going RSVPs in a single query (no N+1). Events with nil capacity are
// left as unlimited (SeatsRemaining stays nil).
func (r *pgRepository) loadSeats(events []*models.Event) error {
	if len(events) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, 0, len(events))
	for _, e := range events {
		if e.Capacity != nil {
			ids = append(ids, e.ID)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	var rows []struct {
		EventID uuid.UUID `pg:"event_id"`
		Going   int       `pg:"going"`
	}
	if _, err := r.db.Query(&rows,
		`SELECT event_id, COUNT(*) AS going FROM event_rsvps
		 WHERE event_id IN (?) AND status = 'going' GROUP BY event_id`,
		pg.In(ids),
	); err != nil {
		return fmt.Errorf("load seats: %w", err)
	}
	goingByID := make(map[uuid.UUID]int, len(rows))
	for _, row := range rows {
		goingByID[row.EventID] = row.Going
	}
	for _, e := range events {
		if e.Capacity == nil {
			continue
		}
		remaining := *e.Capacity - goingByID[e.ID]
		if remaining < 0 {
			remaining = 0
		}
		e.SeatsRemaining = &remaining
	}
	return nil
}

// CountByOrganizerSince returns the number of events (any status) created by
// the given organizer at or after since.
func (r *pgRepository) CountByOrganizerSince(organizer uuid.UUID, since time.Time) (int, error) {
	count, err := r.db.Model((*models.Event)(nil)).
		Where("organizer_id = ?", organizer).
		Where("created_at >= ?", since).
		Count()
	if err != nil {
		return 0, fmt.Errorf("count events for organizer %s since %s: %w", organizer, since.Format(time.RFC3339), err)
	}
	return count, nil
}
