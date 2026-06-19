package models

import (
	"context"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
)

// Venue is a physical place an event happens at (migration 000008). Identity
// only; coordinates/geo arrive in a later spec.
//
//nolint:govet // field alignment kept for readability and conventional ordering
type Venue struct {
	tableName struct{} `pg:"venues,discard_unknown_columns"` //nolint:unused // go-pg table marker

	ID        uuid.UUID `pg:"id,pk,type:uuid"`
	Name      string    `pg:"name,notnull"`
	Address   string    `pg:"address,use_zero"`
	Metro     string    `pg:"metro,use_zero"`
	District  string    `pg:"district,use_zero"`
	Lat       *float64  `pg:"lat"`
	Lon       *float64  `pg:"lon"`
	CreatedAt time.Time `pg:"created_at,notnull,default:now()"`
	UpdatedAt time.Time `pg:"updated_at,notnull,default:now()"`
}

// BeforeInsert generates a UUID if missing and stamps timestamps.
func (v *Venue) BeforeInsert(ctx context.Context) (context.Context, error) {
	if v.ID == uuid.Nil {
		newUUID, err := uuid.NewV4()
		if err != nil {
			return ctx, fmt.Errorf("generate UUID: %w", err)
		}
		v.ID = newUUID
	}
	now := time.Now()
	if v.CreatedAt.IsZero() {
		v.CreatedAt = now
	}
	if v.UpdatedAt.IsZero() {
		v.UpdatedAt = now
	}
	return ctx, nil
}
