// Package categories is the category-taxonomy domain module of the Lia monolith.
// It owns the curated categories list and validation of category references.
package categories

import (
	"fmt"

	"github.com/go-pg/pg/v10"
	"github.com/gofrs/uuid"

	"github.com/Pashteto/lia/internal/models"
)

// Repository defines category persistence operations.
type Repository interface {
	// List returns all categories ordered by sort_order.
	List() ([]*models.Category, error)
	// GetByIDs returns the categories matching the given ids (order by sort_order).
	// Results contain at most one entry per id (id is the primary key); callers
	// rely on this to detect missing ids by comparing lengths.
	GetByIDs(ids []uuid.UUID) ([]*models.Category, error)
}

type pgRepository struct {
	db *pg.DB
}

// NewRepository creates a PostgreSQL-backed category repository.
func NewRepository(db *pg.DB) Repository {
	return &pgRepository{db: db}
}

func (r *pgRepository) List() ([]*models.Category, error) {
	var list []*models.Category
	if err := r.db.Model(&list).Order("sort_order ASC").Select(); err != nil {
		return nil, fmt.Errorf("list categories from db: %w", err)
	}
	return list, nil
}

func (r *pgRepository) GetByIDs(ids []uuid.UUID) ([]*models.Category, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var list []*models.Category
	if err := r.db.Model(&list).
		Where("id IN (?)", pg.In(ids)).
		Order("sort_order ASC").
		Select(); err != nil {
		return nil, fmt.Errorf("get categories by ids from db: %w", err)
	}
	return list, nil
}
