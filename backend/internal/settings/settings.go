// Package settings is a minimal key/value store over app_settings for
// runtime-toggleable global flags (e.g. organizers.auto_verify_all). Changes
// are audited (settings.update in audit_log) because some flags gate
// detective controls. See spec 2026-06-26-organizer-entity-verification-design.md.
package settings

import (
	"context"
	"fmt"

	"github.com/go-pg/pg/v10"
	"github.com/gofrs/uuid"
)

// KeyAutoVerifyAll, when true, auto-verifies every submitted organizer draft.
const KeyAutoVerifyAll = "organizers.auto_verify_all"

// Repository persists boolean settings backed by app_settings.value->>'enabled'.
type Repository interface {
	GetBool(ctx context.Context, key string) (bool, error)
	SetBool(ctx context.Context, key string, actorID uuid.UUID, val bool) error
	All(ctx context.Context) (map[string]bool, error)
}

// Service is the settings use-case layer.
type Service interface {
	Bool(ctx context.Context, key string) (bool, error)
	SetBool(ctx context.Context, key string, actorID uuid.UUID, val bool) error
	All(ctx context.Context) (map[string]bool, error)
}

type service struct{ repo Repository }

// NewService returns a settings Service backed by repo.
func NewService(repo Repository) Service { return &service{repo: repo} }

func (s *service) Bool(ctx context.Context, key string) (bool, error) {
	return s.repo.GetBool(ctx, key)
}
func (s *service) SetBool(ctx context.Context, key string, actorID uuid.UUID, val bool) error {
	return s.repo.SetBool(ctx, key, actorID, val)
}
func (s *service) All(ctx context.Context) (map[string]bool, error) { return s.repo.All(ctx) }

type pgRepository struct{ db *pg.DB }

// NewRepository returns a pg-backed settings Repository.
func NewRepository(db *pg.DB) Repository { return &pgRepository{db: db} }

func (r *pgRepository) GetBool(ctx context.Context, key string) (bool, error) {
	var enabled bool
	_, err := r.db.QueryOneContext(ctx, pg.Scan(&enabled),
		`SELECT coalesce((value->>'enabled')::boolean, false) FROM app_settings WHERE key = ?`, key)
	if err != nil {
		if err == pg.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("get setting %q: %w", key, err)
	}
	return enabled, nil
}

func (r *pgRepository) All(ctx context.Context) (map[string]bool, error) {
	var rows []struct {
		Key     string `pg:"key"`
		Enabled bool   `pg:"enabled,use_zero"`
	}
	if _, err := r.db.QueryContext(ctx, &rows,
		`SELECT key, coalesce((value->>'enabled')::boolean, false) AS enabled FROM app_settings`); err != nil {
		return nil, fmt.Errorf("list settings: %w", err)
	}
	out := make(map[string]bool, len(rows))
	for _, row := range rows {
		out[row.Key] = row.Enabled
	}
	return out, nil
}

// SetBool upserts the flag and writes a settings.update audit row in one tx.
func (r *pgRepository) SetBool(ctx context.Context, key string, actorID uuid.UUID, val bool) error {
	return r.db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO app_settings (key, value, updated_at, updated_by)
			 VALUES (?, jsonb_build_object('enabled', ?::boolean), now(), ?)
			 ON CONFLICT (key) DO UPDATE
			   SET value = jsonb_build_object('enabled', ?::boolean), updated_at = now(), updated_by = ?`,
			key, val, actorID, val, actorID); err != nil {
			return fmt.Errorf("upsert setting: %w", err)
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO audit_log (actor_user_id, action, target_type, target_id, metadata)
			 VALUES (?, 'settings.update', 'setting', '00000000-0000-0000-0000-000000000000',
			         jsonb_build_object('key', ?::text, 'enabled', ?::boolean))`,
			actorID, key, val); err != nil {
			return fmt.Errorf("insert audit log: %w", err)
		}
		return nil
	})
}
