package platforms

import (
	"time"

	"github.com/gofrs/uuid"
)

// TrustedPlatform is one whitelist row. DomainSuffix is punycode, no scheme.
type TrustedPlatform struct {
	tableName struct{} `pg:"trusted_platforms,discard_unknown_columns"` //nolint:unused

	ID           uuid.UUID `pg:"id,pk,type:uuid" json:"id"`
	DomainSuffix string    `pg:"domain_suffix,notnull" json:"domain_suffix"`
	DisplayName  string    `pg:"display_name,notnull" json:"display_name"`
	Category     string    `pg:"category,notnull" json:"category"`
	IsActive     bool      `pg:"is_active,use_zero" json:"is_active"`
	CreatedAt    time.Time `pg:"created_at,notnull,default:now()" json:"created_at"`
}
