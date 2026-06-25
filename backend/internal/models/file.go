package models

import (
	"time"

	"github.com/gofrs/uuid"
)

// File is an uploaded blob's metadata row (the bytes live in storage.Storage).
type File struct {
	tableName   struct{}  `pg:"files,discard_unknown_columns"` //nolint:unused
	ID          uuid.UUID `pg:"id,pk,type:uuid"`
	StorageKey  string    `pg:"storage_key,notnull"`
	ContentType string    `pg:"content_type,notnull"`
	Size        int64     `pg:"size,use_zero"`
	OwnerUserID uuid.UUID `pg:"owner_user_id,type:uuid,use_zero"`
	CreatedAt   time.Time `pg:"created_at,notnull,default:now()"`
}
