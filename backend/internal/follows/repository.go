// Package follows implements the user→organizer subscription ("follow") used by
// the personal calendar. A follow row stores the organizer's OWNER user id
// (= events.organizer_id) so listing a follower's events is a direct
// organizer_id IN (...) query. The public API addresses organizers by
// organizers.id; the service resolves that to the owner id before persisting.
package follows

import (
	"context"
	"fmt"

	"github.com/go-pg/pg/v10"
	"github.com/gofrs/uuid"
)

// Follow is the organizer_follows row.
type Follow struct {
	tableName struct{} `pg:"organizer_follows,discard_unknown_columns"` //nolint:unused // go-pg table marker

	UserID          uuid.UUID `pg:"user_id,type:uuid,use_zero"`
	OrganizerUserID uuid.UUID `pg:"organizer_user_id,type:uuid,use_zero"`
}

// FollowedOrg is one row of the "organizers I follow" list, joined to the
// organizers profile for display.
type FollowedOrg struct {
	ProfileID  uuid.UUID `pg:"profile_id,type:uuid"`
	OwnerID    uuid.UUID `pg:"owner_id,type:uuid"`
	Name       string    `pg:"name,use_zero"`
	LogoFileID uuid.UUID `pg:"logo_file_id,type:uuid,use_zero"`
}

// Repository persists organizer follows.
type Repository interface {
	Add(ctx context.Context, userID, organizerUserID uuid.UUID) error
	Remove(ctx context.Context, userID, organizerUserID uuid.UUID) error
	IsFollowing(ctx context.Context, userID, organizerUserID uuid.UUID) (bool, error)
	ListFollowedOwnerIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	// ListFollowedOrganizers returns the verified organizer profiles the user
	// follows, in a single JOIN (no N+1).
	ListFollowedOrganizers(ctx context.Context, userID uuid.UUID) ([]FollowedOrg, error)
}

type pgRepository struct{ db *pg.DB }

// NewRepository creates a PostgreSQL-backed follows repository.
func NewRepository(db *pg.DB) Repository { return &pgRepository{db: db} }

func (r *pgRepository) Add(ctx context.Context, userID, organizerUserID uuid.UUID) error {
	if _, err := r.db.ExecContext(ctx,
		`INSERT INTO organizer_follows (user_id, organizer_user_id)
		 VALUES (?, ?) ON CONFLICT (user_id, organizer_user_id) DO NOTHING`,
		userID, organizerUserID,
	); err != nil {
		return fmt.Errorf("add follow %s->%s: %w", userID, organizerUserID, err)
	}
	return nil
}

func (r *pgRepository) Remove(ctx context.Context, userID, organizerUserID uuid.UUID) error {
	if _, err := r.db.ExecContext(ctx,
		`DELETE FROM organizer_follows WHERE user_id = ? AND organizer_user_id = ?`,
		userID, organizerUserID,
	); err != nil {
		return fmt.Errorf("remove follow %s->%s: %w", userID, organizerUserID, err)
	}
	return nil
}

func (r *pgRepository) IsFollowing(ctx context.Context, userID, organizerUserID uuid.UUID) (bool, error) {
	count, err := r.db.ModelContext(ctx, (*Follow)(nil)).
		Where("user_id = ?", userID).
		Where("organizer_user_id = ?", organizerUserID).
		Count()
	if err != nil {
		return false, fmt.Errorf("is following %s->%s: %w", userID, organizerUserID, err)
	}
	return count > 0, nil
}

func (r *pgRepository) ListFollowedOwnerIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	if _, err := r.db.QueryContext(ctx, &ids,
		`SELECT organizer_user_id FROM organizer_follows WHERE user_id = ?`,
		userID,
	); err != nil {
		return nil, fmt.Errorf("list followed owner ids for %s: %w", userID, err)
	}
	return ids, nil
}

func (r *pgRepository) ListFollowedOrganizers(ctx context.Context, userID uuid.UUID) ([]FollowedOrg, error) {
	var rows []FollowedOrg
	if _, err := r.db.QueryContext(ctx, &rows,
		`SELECT o.id AS profile_id, o.owner_user_id AS owner_id, o.name AS name,
		        o.logo_file_id AS logo_file_id
		   FROM organizer_follows f
		   JOIN organizers o ON o.owner_user_id = f.organizer_user_id
		  WHERE f.user_id = ?
		  ORDER BY o.name ASC`,
		userID,
	); err != nil {
		return nil, fmt.Errorf("list followed organizers for %s: %w", userID, err)
	}
	return rows, nil
}
