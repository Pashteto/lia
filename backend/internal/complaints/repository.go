package complaints

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/gofrs/uuid"
)

type pgRepository struct{ db *pg.DB }

// NewRepository returns a pg-backed complaints Repository.
func NewRepository(db *pg.DB) Repository { return &pgRepository{db: db} }

// Insert adds an open complaint. The partial unique index makes a repeat open
// complaint from the same reporter a no-op; returns false in that case.
func (r *pgRepository) Insert(ctx context.Context, c Complaint) (bool, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO complaints (target_type, target_id, reporter_user_id, category, note, status)
		 VALUES (?, ?, ?, ?, NULLIF(?, ''), 'open')
		 ON CONFLICT (target_type, target_id, reporter_user_id) WHERE status = 'open' DO NOTHING`,
		c.TargetType, c.TargetID, c.ReporterUserID, c.Category, c.Note)
	if err != nil {
		return false, fmt.Errorf("insert complaint: %w", err)
	}
	return res.RowsAffected() > 0, nil
}

func (r *pgRepository) EventExists(ctx context.Context, id uuid.UUID) (bool, error) {
	var exists bool
	if _, err := r.db.QueryOneContext(ctx, pg.Scan(&exists),
		`SELECT EXISTS(SELECT 1 FROM events WHERE id = ?)`, id); err != nil {
		return false, fmt.Errorf("event exists: %w", err)
	}
	return exists, nil
}

// InboxGroups returns open complaints grouped by event. Aggregation is done in
// Go (clear + testable; demo-scale row counts). Rows are ordered newest-first
// per target so the first non-empty note per group is the latest.
func (r *pgRepository) InboxGroups(ctx context.Context) ([]EventReportGroup, error) {
	var rows []struct {
		TargetID  uuid.UUID `pg:"target_id"`
		Title     string    `pg:"title"`
		Status    string    `pg:"status"`
		Category  string    `pg:"category"`
		Note      string    `pg:"note"`
		CreatedAt time.Time `pg:"created_at"`
	}
	if _, err := r.db.QueryContext(ctx, &rows,
		`SELECT c.target_id, e.title, e.status, c.category,
		        coalesce(c.note, '') AS note, c.created_at
		 FROM complaints c
		 JOIN events e ON e.id = c.target_id
		 WHERE c.status = 'open' AND c.target_type = 'event'
		 ORDER BY c.target_id, c.created_at DESC`); err != nil {
		return nil, fmt.Errorf("inbox groups: %w", err)
	}

	byTarget := map[uuid.UUID]*EventReportGroup{}
	order := []uuid.UUID{}
	for _, row := range rows {
		g := byTarget[row.TargetID]
		if g == nil {
			g = &EventReportGroup{
				TargetID: row.TargetID, EventTitle: row.Title, EventStatus: row.Status,
				Categories: map[string]int{}, LatestAt: row.CreatedAt,
			}
			byTarget[row.TargetID] = g
			order = append(order, row.TargetID)
		}
		g.ReportCount++
		g.Categories[row.Category]++
		if g.LatestNote == "" && row.Note != "" {
			g.LatestNote = row.Note // rows are created_at DESC → newest non-empty wins
		}
	}

	out := make([]EventReportGroup, 0, len(order))
	for _, id := range order {
		out = append(out, *byTarget[id])
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].ReportCount != out[j].ReportCount {
			return out[i].ReportCount > out[j].ReportCount
		}
		return out[i].LatestAt.After(out[j].LatestAt)
	})
	return out, nil
}

func (r *pgRepository) TargetComplaints(ctx context.Context, targetType string, targetID uuid.UUID) ([]Complaint, error) {
	var rows []Complaint
	if _, err := r.db.QueryContext(ctx, &rows,
		`SELECT id, target_type, target_id, reporter_user_id, category,
		        coalesce(note, '') AS note, status, coalesce(resolution, '') AS resolution,
		        resolved_by, resolved_at, created_at
		 FROM complaints
		 WHERE target_type = ? AND target_id = ?
		 ORDER BY created_at DESC`,
		targetType, targetID); err != nil {
		return nil, fmt.Errorf("target complaints: %w", err)
	}
	return rows, nil
}

// ResolveOpenForTarget flips every open complaint for the target to `status`
// and writes one audit_log row, in one tx. Returns the affected count.
func (r *pgRepository) ResolveOpenForTarget(ctx context.Context, targetType string, targetID, actorID uuid.UUID, status, resolution string) (int, error) {
	var affected int
	err := r.db.RunInTransaction(ctx, func(tx *pg.Tx) error {
		res, err := tx.ExecContext(ctx,
			`UPDATE complaints
			 SET status = ?, resolution = NULLIF(?, ''), resolved_by = ?, resolved_at = now()
			 WHERE target_type = ? AND target_id = ? AND status = 'open'`,
			status, resolution, actorID, targetType, targetID)
		if err != nil {
			return fmt.Errorf("resolve complaints: %w", err)
		}
		affected = res.RowsAffected()
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO audit_log (actor_user_id, action, target_type, target_id, metadata)
			 VALUES (?, 'complaint.resolve', ?, ?,
			         jsonb_build_object('status', ?::text, 'resolution', NULLIF(?, ''), 'resolved_count', ?::int))`,
			actorID, targetType, targetID, status, resolution, affected); err != nil {
			return fmt.Errorf("insert audit log: %w", err)
		}
		return nil
	})
	return affected, err
}

func (r *pgRepository) OpenEventCount(ctx context.Context) (int, error) {
	var n int
	if _, err := r.db.QueryOneContext(ctx, pg.Scan(&n),
		`SELECT count(DISTINCT target_id) FROM complaints
		 WHERE status = 'open' AND target_type = 'event'`); err != nil {
		return 0, fmt.Errorf("open event count: %w", err)
	}
	return n, nil
}
