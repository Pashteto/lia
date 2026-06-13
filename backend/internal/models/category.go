package models

import (
	"context"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
)

// Category is a curated event category in the taxonomy (migration 000006).
//
//nolint:govet // field alignment kept for readability and conventional ordering
type Category struct {
	tableName struct{} `pg:"categories,discard_unknown_columns"` //nolint:unused // go-pg table marker

	ID        uuid.UUID `pg:"id,pk,type:uuid"`
	Slug      string    `pg:"slug,notnull"`
	Label     string    `pg:"label,notnull"`
	SortOrder int       `pg:"sort_order,use_zero"`
	CreatedAt time.Time `pg:"created_at,notnull,default:now()"`
	UpdatedAt time.Time `pg:"updated_at,notnull,default:now()"`
}

// BeforeInsert generates a UUID if missing and stamps timestamps. Categories are
// seeded via migration today; this keeps the model usable if inserts are added.
func (c *Category) BeforeInsert(ctx context.Context) (context.Context, error) {
	if c.ID == uuid.Nil {
		newUUID, err := uuid.NewV4()
		if err != nil {
			return ctx, fmt.Errorf("generate UUID: %w", err)
		}
		c.ID = newUUID
	}
	now := time.Now()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = now
	}
	return ctx, nil
}
