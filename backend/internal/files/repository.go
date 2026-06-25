// Package files is the file-metadata domain module of the Lia monolith.
// Blob bytes live in storage.Storage; this package owns the files DB table.
package files

import (
	"errors"
	"fmt"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/gofrs/uuid"

	"github.com/Pashteto/lia/internal/models"
)

// Repository defines file metadata persistence operations.
type Repository interface {
	// Create inserts a file row. Sets f.ID to a new UUID if zero.
	Create(f *models.File) error
	// GetByID returns a single file by primary key.
	GetByID(id uuid.UUID) (*models.File, error)
	// ListOrphansOlderThan returns files that are referenced by neither
	// events.cover_file_id nor users.avatar_file_id and are older than d.
	// NOTE: the referenced columns (cover_file_id, avatar_file_id) are added
	// in migration 000011 (Phase 4). This method is only called at cleanup
	// time (Phase 6), by which point those columns exist.
	ListOrphansOlderThan(d time.Duration) ([]*models.File, error)
	// Delete removes a file row by id (idempotent — no error if not found).
	Delete(id uuid.UUID) error
}

// orphanQuery finds files that are not referenced by any event cover or user
// avatar and were created more than d ago.
const orphanQuery = `
SELECT f.* FROM files f
WHERE f.created_at < (now() - ?0::interval)
  AND NOT EXISTS (SELECT 1 FROM events e WHERE e.cover_file_id = f.id)
  AND NOT EXISTS (SELECT 1 FROM users u WHERE u.avatar_file_id = f.id)
`

type pgRepository struct {
	db *pg.DB
}

// NewRepository creates a PostgreSQL-backed file repository.
func NewRepository(db *pg.DB) Repository {
	return &pgRepository{db: db}
}

func (r *pgRepository) Create(f *models.File) error {
	if f.ID == uuid.Nil {
		id, err := uuid.NewV4()
		if err != nil {
			return fmt.Errorf("generate file id: %w", err)
		}
		f.ID = id
	}
	if _, err := r.db.Model(f).Insert(); err != nil {
		return fmt.Errorf("insert file: %w", err)
	}
	return nil
}

func (r *pgRepository) GetByID(id uuid.UUID) (*models.File, error) {
	f := &models.File{ID: id}
	if err := r.db.Model(f).WherePK().Select(); err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get file %s from db: %w", id, err)
	}
	return f, nil
}

func (r *pgRepository) ListOrphansOlderThan(d time.Duration) ([]*models.File, error) {
	interval := fmt.Sprintf("%d seconds", int(d.Seconds()))
	var list []*models.File
	if _, err := r.db.Query(&list, orphanQuery, interval); err != nil {
		return nil, fmt.Errorf("list orphan files: %w", err)
	}
	return list, nil
}

func (r *pgRepository) Delete(id uuid.UUID) error {
	f := &models.File{ID: id}
	if _, err := r.db.Model(f).WherePK().Delete(); err != nil && !errors.Is(err, pg.ErrNoRows) {
		return fmt.Errorf("delete file %s: %w", id, err)
	}
	return nil
}
