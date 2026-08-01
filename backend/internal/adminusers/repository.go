package adminusers

import (
	"context"
	"fmt"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/gofrs/uuid"
)

type pgRepository struct{ db *pg.DB }

// NewRepository returns a pg-backed registry Repository.
func NewRepository(db *pg.DB) Repository { return &pgRepository{db: db} }

// row mirrors the aliased columns of the listing query. go-pg maps snake_case
// columns onto these fields by name.
// users.created_at is NOT NULL (migration 000002), so a plain time.Time is safe.
type row struct {
	ID          uuid.UUID `pg:"id"`
	Name        string    `pg:"name"`
	Email       string    `pg:"email"`
	Role        string    `pg:"role"`
	CreatedAt   time.Time `pg:"created_at"`
	Bookings    int       `pg:"bookings"`
	IsOrganizer bool      `pg:"is_organizer"`
}

// listSQL joins the three sources A4 needs in a single statement:
//   - event_rsvps aggregated per user (only statuses that mean "записан")
//   - organizers ownership (role derivation on the client)
//
// Deleted users are excluded: the registry is an operations tool.
const listSQL = `
SELECT u.uuid                        AS id,
       coalesce(u.name, '')          AS name,
       coalesce(u.email, '')         AS email,
       coalesce(u.role, 'common')    AS role,
       u.created_at                  AS created_at,
       coalesce(r.n, 0)              AS bookings,
       (o.id IS NOT NULL)            AS is_organizer
  FROM users u
  LEFT JOIN (
        SELECT user_id, count(*) AS n
          FROM event_rsvps
         WHERE status IN ('going', 'applied', 'accepted', 'waitlist')
         GROUP BY user_id
  ) r ON r.user_id = u.uuid
  LEFT JOIN organizers o ON o.owner_user_id = u.uuid
 WHERE u.status = 'active'
   AND (?0 = '' OR u.name ILIKE ?1 OR u.email ILIKE ?1)
 ORDER BY u.created_at DESC, u.uuid DESC
 LIMIT ?2 OFFSET ?3`

// Count returns every active user. Same population as listSQL's WHERE clause —
// deleted users stay out of the registry, so they stay out of the tile too.
func (r *pgRepository) Count(ctx context.Context) (int, error) {
	var n int
	if _, err := r.db.QueryOneContext(ctx, pg.Scan(&n),
		`SELECT count(*) FROM users WHERE status = 'active'`); err != nil {
		return 0, fmt.Errorf("count admin users: %w", err)
	}
	return n, nil
}

func (r *pgRepository) List(ctx context.Context, f Filter) ([]Row, error) {
	var rows []row
	like := "%" + f.Query + "%"
	if _, err := r.db.QueryContext(ctx, &rows, listSQL, f.Query, like, f.Limit, f.Offset); err != nil {
		return nil, fmt.Errorf("list admin users: %w", err)
	}
	out := make([]Row, 0, len(rows))
	for _, x := range rows {
		out = append(out, Row{
			ID: x.ID, Name: x.Name, Email: x.Email, Role: x.Role,
			CreatedAt: x.CreatedAt, Bookings: x.Bookings, IsOrganizer: x.IsOrganizer,
		})
	}
	return out, nil
}
