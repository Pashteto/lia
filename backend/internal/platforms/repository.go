package platforms

import (
	"context"
	"fmt"

	"github.com/go-pg/pg/v10"
	"github.com/gofrs/uuid"
)

type Repository interface {
	ListActive(ctx context.Context) ([]*TrustedPlatform, error)
	List(ctx context.Context) ([]*TrustedPlatform, error)
	Create(ctx context.Context, p *TrustedPlatform) error
	SetActive(ctx context.Context, id uuid.UUID, active bool) error
}

type pgRepository struct{ db *pg.DB }

func NewRepository(db *pg.DB) Repository { return &pgRepository{db: db} }

func (r *pgRepository) ListActive(ctx context.Context) ([]*TrustedPlatform, error) {
	var out []*TrustedPlatform
	err := r.db.ModelContext(ctx, &out).Where("is_active").Order("domain_suffix ASC").Select()
	if err != nil {
		return nil, fmt.Errorf("list active platforms: %w", err)
	}
	return out, nil
}

func (r *pgRepository) List(ctx context.Context) ([]*TrustedPlatform, error) {
	var out []*TrustedPlatform
	if err := r.db.ModelContext(ctx, &out).Order("domain_suffix ASC").Select(); err != nil {
		return nil, fmt.Errorf("list platforms: %w", err)
	}
	return out, nil
}

func (r *pgRepository) Create(ctx context.Context, p *TrustedPlatform) error {
	if _, err := r.db.ModelContext(ctx, p).Insert(); err != nil {
		return fmt.Errorf("insert platform: %w", err)
	}
	return nil
}

func (r *pgRepository) SetActive(ctx context.Context, id uuid.UUID, active bool) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE trusted_platforms SET is_active = ? WHERE id = ?`, active, id)
	if err != nil {
		return fmt.Errorf("set platform active: %w", err)
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("%w: %s", ErrNotFound, id)
	}
	return nil
}
