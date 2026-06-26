package organizers

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-pg/pg/v10"
	"github.com/gofrs/uuid"
)

const zeroUUID = "00000000-0000-0000-0000-000000000000"

type pgRepository struct{ db *pg.DB }

// NewRepository returns a pg-backed organizers Repository.
func NewRepository(db *pg.DB) Repository { return &pgRepository{db: db} }

func scanOrganizer(dst *Organizer) []interface{} {
	return []interface{}{
		&dst.ID, &dst.OwnerUserID, &dst.Name, &dst.Description, &dst.WebsiteURL,
		&dst.LogoFileID, &dst.VerificationStatus, &dst.AutoVerify, &dst.VerifiedAt,
	}
}

const orgCols = `id, owner_user_id, name, description, website_url, logo_file_id,
                 verification_status, auto_verify, verified_at`

func (r *pgRepository) GetByOwner(ctx context.Context, ownerID uuid.UUID) (*Organizer, error) {
	var o Organizer
	_, err := r.db.QueryOneContext(ctx, pg.Scan(scanOrganizer(&o)...),
		`SELECT `+orgCols+` FROM organizers WHERE owner_user_id = ?`, ownerID)
	if err != nil {
		if err == pg.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get organizer by owner: %w", err)
	}
	r.fillLatestReason(ctx, &o)
	return &o, nil
}

func (r *pgRepository) GetByID(ctx context.Context, id uuid.UUID) (*Organizer, error) {
	var o Organizer
	_, err := r.db.QueryOneContext(ctx, pg.Scan(scanOrganizer(&o)...),
		`SELECT `+orgCols+` FROM organizers WHERE id = ?`, id)
	if err != nil {
		if err == pg.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get organizer by id: %w", err)
	}
	r.fillLatestReason(ctx, &o)
	return &o, nil
}

func (r *pgRepository) fillLatestReason(ctx context.Context, o *Organizer) {
	if o.VerificationStatus != "rejected" {
		return
	}
	var reason string
	if _, err := r.db.QueryOneContext(ctx, pg.Scan(&reason),
		`SELECT coalesce(reason, '') FROM organizer_verification_history
		  WHERE organizer_id = ? AND to_status = 'rejected'
		  ORDER BY created_at DESC LIMIT 1`, o.ID); err == nil {
		o.LatestReason = reason
	}
}

func (r *pgRepository) Upsert(ctx context.Context, ownerID uuid.UUID, in Input) (*Organizer, error) {
	logo := in.LogoFileID
	var o Organizer
	_, err := r.db.QueryOneContext(ctx, pg.Scan(scanOrganizer(&o)...),
		`INSERT INTO organizers (owner_user_id, name, description, website_url, logo_file_id)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT (owner_user_id) DO UPDATE
		   SET name = EXCLUDED.name, description = EXCLUDED.description,
		       website_url = EXCLUDED.website_url, logo_file_id = EXCLUDED.logo_file_id,
		       updated_at = now()
		 RETURNING `+orgCols,
		ownerID, in.Name, in.Description, in.WebsiteURL, logo)
	if err != nil {
		return nil, fmt.Errorf("upsert organizer: %w", err)
	}
	return &o, nil
}

// transition flips verification_status from→to inside one tx, writing a history
// row and an audit_log row. autoActor is the zero-uuid for system/auto actions.
func (r *pgRepository) transition(ctx context.Context, id, actorID uuid.UUID, from, to, action, reason string, meta string) error {
	return r.db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		res, err := tx.ExecContext(ctx,
			`UPDATE organizers
			    SET verification_status = ?,
			        verified_at = CASE WHEN ? = 'verified' THEN now() ELSE verified_at END,
			        updated_at = now()
			  WHERE id = ? AND verification_status = ?`,
			to, to, id, from)
		if err != nil {
			return fmt.Errorf("update organizer status: %w", err)
		}
		if res.RowsAffected() == 0 {
			return ErrInvalidTransition
		}
		return r.writeHistoryAudit(ctx, tx, id, actorID, from, to, action, reason, meta)
	})
}

func (r *pgRepository) writeHistoryAudit(ctx context.Context, tx *pg.Tx, id, actorID uuid.UUID, from, to, action, reason, meta string) error {
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO organizer_verification_history (organizer_id, from_status, to_status, actor_user_id, reason)
		 VALUES (?, ?, ?, ?, NULLIF(?, ''))`,
		id, from, to, actorID, reason); err != nil {
		return fmt.Errorf("insert verification history: %w", err)
	}
	if meta == "" {
		meta = "{}"
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO audit_log (actor_user_id, action, target_type, target_id, metadata)
		 VALUES (?, ?, 'organizer', ?, ?::jsonb)`,
		actorID, action, id, meta); err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}
	return nil
}

// Submit moves draft|rejected → pending, or → verified when autoVerify. It reads
// the current status inside the tx so history records the true from_status.
func (r *pgRepository) Submit(ctx context.Context, id, actorID uuid.UUID, autoVerify bool) (string, error) {
	to := "pending"
	action := "organizer.submit"
	meta := ""
	actor := actorID
	if autoVerify {
		to = "verified"
		action = "organizer.verify"
		meta = `{"auto":true,"source":"submit"}`
		actor = uuid.FromStringOrNil(zeroUUID)
	}
	err := r.db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		var from string
		if _, err := tx.QueryOneContext(ctx, pg.Scan(&from),
			`SELECT verification_status FROM organizers WHERE id = ? FOR UPDATE`, id); err != nil {
			if err == pg.ErrNoRows {
				return ErrNotFound
			}
			return fmt.Errorf("lock organizer: %w", err)
		}
		if from != "draft" && from != "rejected" {
			return ErrInvalidTransition
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE organizers
			    SET verification_status = ?,
			        verified_at = CASE WHEN ? = 'verified' THEN now() ELSE verified_at END,
			        updated_at = now()
			  WHERE id = ?`, to, to, id); err != nil {
			return fmt.Errorf("update organizer status: %w", err)
		}
		return r.writeHistoryAudit(ctx, tx, id, actor, from, to, action, "", meta)
	})
	if err != nil {
		return "", err
	}
	return to, nil
}

func (r *pgRepository) Verify(ctx context.Context, id, actorID uuid.UUID) error {
	return r.transition(ctx, id, actorID, "pending", "verified", "organizer.verify", "", "")
}

func (r *pgRepository) Reject(ctx context.Context, id, actorID uuid.UUID, reason string) error {
	return r.transition(ctx, id, actorID, "pending", "rejected", "organizer.reject", reason,
		`{}`)
}

func (r *pgRepository) Revoke(ctx context.Context, id, actorID uuid.UUID, reason string) error {
	return r.transition(ctx, id, actorID, "verified", "rejected", "organizer.revoke", reason,
		`{}`)
}

func (r *pgRepository) SetAutoVerify(ctx context.Context, id, actorID uuid.UUID, enabled bool) error {
	return r.db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		res, err := tx.ExecContext(ctx,
			`UPDATE organizers SET auto_verify = ?, updated_at = now() WHERE id = ?`, enabled, id)
		if err != nil {
			return fmt.Errorf("update auto_verify: %w", err)
		}
		if res.RowsAffected() == 0 {
			return ErrNotFound
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO audit_log (actor_user_id, action, target_type, target_id, metadata)
			 VALUES (?, 'organizer.set_auto_verify', 'organizer', ?, jsonb_build_object('enabled', ?::boolean))`,
			actorID, id, enabled); err != nil {
			return fmt.Errorf("insert audit log: %w", err)
		}
		return nil
	})
}

func (r *pgRepository) List(ctx context.Context, f ListFilter) ([]Organizer, error) {
	var orgs []Organizer
	where := []string{"1=1"}
	args := []interface{}{}
	if f.Status != "" {
		where = append(where, "o.verification_status = ?")
		args = append(args, f.Status)
	}
	if q := strings.TrimSpace(f.Query); q != "" {
		where = append(where, "(o.name ILIKE ? OR u.email ILIKE ?)")
		args = append(args, "%"+q+"%", "%"+q+"%")
	}
	query := `SELECT ` + prefixCols("o") + `
	            FROM organizers o
	            LEFT JOIN users u ON u.uuid = o.owner_user_id
	           WHERE ` + strings.Join(where, " AND ") + `
	           ORDER BY o.created_at DESC`
	if _, err := r.db.QueryContext(ctx, &orgs, query, args...); err != nil {
		return nil, fmt.Errorf("list organizers: %w", err)
	}
	return orgs, nil
}

// prefixCols renders orgCols with a table alias for join queries.
func prefixCols(alias string) string {
	cols := strings.Split(orgCols, ",")
	for i, c := range cols {
		cols[i] = alias + "." + strings.TrimSpace(c)
	}
	return strings.Join(cols, ", ")
}

func (r *pgRepository) History(ctx context.Context, id uuid.UUID) ([]HistoryEntry, error) {
	var hist []HistoryEntry
	if _, err := r.db.QueryContext(ctx, &hist,
		`SELECT from_status AS from_status, to_status AS to_status,
		        coalesce(reason, '') AS reason, actor_user_id AS actor_user_id, created_at AS created_at
		   FROM organizer_verification_history
		  WHERE organizer_id = ?
		  ORDER BY created_at DESC`, id); err != nil {
		return nil, fmt.Errorf("organizer history: %w", err)
	}
	return hist, nil
}

func (r *pgRepository) Counts(ctx context.Context) (Counts, error) {
	var c Counts
	_, err := r.db.QueryOneContext(ctx, pg.Scan(&c.OrganizersPending),
		`SELECT count(*) FROM organizers WHERE verification_status = 'pending'`)
	if err != nil {
		return Counts{}, fmt.Errorf("count pending organizers: %w", err)
	}
	return c, nil
}

func (r *pgRepository) VerifiedByOwners(ctx context.Context, ownerIDs []uuid.UUID) (map[uuid.UUID]VerifiedOrg, error) {
	out := make(map[uuid.UUID]VerifiedOrg)
	if len(ownerIDs) == 0 {
		return out, nil
	}
	var rows []struct {
		OwnerUserID uuid.UUID `pg:"owner_user_id"`
		ID          uuid.UUID `pg:"id"`
		Name        string    `pg:"name,use_zero"`
		LogoKey     string    `pg:"logo_key,use_zero"`
	}
	if _, err := r.db.QueryContext(ctx, &rows,
		`SELECT o.owner_user_id, o.id, o.name, COALESCE(f.storage_key, '') AS logo_key
		   FROM organizers o
		   LEFT JOIN files f ON f.id = o.logo_file_id
		  WHERE o.verification_status = 'verified' AND o.owner_user_id IN (?)`,
		pg.In(ownerIDs)); err != nil {
		return nil, fmt.Errorf("verified by owners: %w", err)
	}
	for _, row := range rows {
		out[row.OwnerUserID] = VerifiedOrg{ID: row.ID, Name: row.Name, LogoKey: row.LogoKey}
	}
	return out, nil
}
