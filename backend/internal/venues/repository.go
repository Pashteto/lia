// Package venues is the venue domain module of the Lia monolith. It owns the
// venue entity, search, and find-or-create. Identity only — geo arrives later.
package venues

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-pg/pg/v10"
	"github.com/gofrs/uuid"

	"github.com/Pashteto/lia/internal/models"
)

// DefaultSearchLimit caps Search results when no limit is given.
const DefaultSearchLimit = 20

// Repository defines venue persistence operations.
type Repository interface {
	// Search returns venues whose name matches q (case-insensitive substring),
	// ordered by name. Empty q returns the first `limit` venues by name.
	Search(q string, limit int) ([]*models.Venue, error)
	// GetByID returns a single venue by primary key.
	GetByID(id uuid.UUID) (*models.Venue, error)
	// GetByIDs returns the venues matching the given ids.
	GetByIDs(ids []uuid.UUID) ([]*models.Venue, error)
	// FindOrCreateByName returns an existing venue whose lower(name) matches
	// v.Name, else inserts v and returns it.
	FindOrCreateByName(v *models.Venue) (*models.Venue, error)
}

type pgRepository struct {
	db *pg.DB
}

// NewRepository creates a PostgreSQL-backed venue repository.
func NewRepository(db *pg.DB) Repository {
	return &pgRepository{db: db}
}

func (r *pgRepository) Search(q string, limit int) ([]*models.Venue, error) {
	if limit <= 0 {
		limit = DefaultSearchLimit
	}
	var list []*models.Venue
	query := r.db.Model(&list)
	if strings.TrimSpace(q) != "" {
		query = query.Where("name ILIKE ?", "%"+strings.TrimSpace(q)+"%")
	}
	if err := query.Order("name ASC").Limit(limit).Select(); err != nil {
		return nil, fmt.Errorf("search venues from db: %w", err)
	}
	return list, nil
}

func (r *pgRepository) GetByID(id uuid.UUID) (*models.Venue, error) {
	venue := &models.Venue{ID: id}
	if err := r.db.Model(venue).WherePK().Select(); err != nil {
		return nil, fmt.Errorf("get venue %s from db: %w", id, err)
	}
	return venue, nil
}

func (r *pgRepository) GetByIDs(ids []uuid.UUID) ([]*models.Venue, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var list []*models.Venue
	if err := r.db.Model(&list).Where("id IN (?)", pg.In(ids)).Select(); err != nil {
		return nil, fmt.Errorf("get venues by ids from db: %w", err)
	}
	return list, nil
}

// FindOrCreateByName is non-atomic (SELECT then INSERT): two concurrent calls
// with the same normalized name can create two rows. Acceptable per spec (no
// unique constraint on name); a future migration may add a partial unique index
// if dedup needs hardening.
func (r *pgRepository) FindOrCreateByName(v *models.Venue) (*models.Venue, error) {
	existing := new(models.Venue)
	err := r.db.Model(existing).
		Where("lower(name) = lower(?)", v.Name).
		Limit(1).
		Select()
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, pg.ErrNoRows) {
		return nil, fmt.Errorf("find venue by name: %w", err)
	}
	if _, err := r.db.Model(v).Insert(); err != nil {
		return nil, fmt.Errorf("insert venue %q: %w", v.Name, err)
	}
	return v, nil
}
