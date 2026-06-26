//go:build integration

package complaints

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/gofrs/uuid"
)

func openTestDB(t *testing.T) *pg.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set — skipping integration test")
	}
	opts, err := pg.ParseURL(dsn)
	if err != nil {
		t.Fatalf("parse TEST_DATABASE_URL: %v", err)
	}
	db := pg.Connect(opts)
	if _, err := db.Exec("SELECT 1"); err != nil {
		db.Close()
		t.Fatalf("connect to test DB: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func insertPublishedEvent(t *testing.T, db *pg.DB) uuid.UUID {
	t.Helper()
	id := uuid.Must(uuid.NewV4())
	if _, err := db.Exec(
		`INSERT INTO events (id, title, status, starts_at, created_at, updated_at)
		 VALUES (?, 'complaints test event', 'published', ?, now(), now())`,
		id, time.Now().Add(24*time.Hour)); err != nil {
		t.Fatalf("insert test event: %v", err)
	}
	t.Cleanup(func() {
		db.Exec(`DELETE FROM complaints WHERE target_id = ?`, id) //nolint:errcheck
		db.Exec(`DELETE FROM events WHERE id = ?`, id)            //nolint:errcheck
		db.Exec(`DELETE FROM audit_log WHERE target_id = ?`, id)  //nolint:errcheck
	})
	return id
}

// TestInsert_DedupAndResolve exercises the open-dup index, grouping, and the
// cascading resolve + audit.
func TestInsert_DedupAndResolve(t *testing.T) {
	db := openTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	eventID := insertPublishedEvent(t, db)
	reporter := uuid.Must(uuid.NewV4())

	// First open complaint inserts.
	created, err := repo.Insert(ctx, Complaint{TargetType: "event", TargetID: eventID, ReporterUserID: reporter, Category: "spam", Note: "bad", Status: "open"})
	if err != nil || !created {
		t.Fatalf("first insert: created=%v err=%v", created, err)
	}
	// Repeat open complaint from same reporter is a no-op.
	created, err = repo.Insert(ctx, Complaint{TargetType: "event", TargetID: eventID, ReporterUserID: reporter, Category: "fraud", Note: "again", Status: "open"})
	if err != nil || created {
		t.Fatalf("dup insert: created=%v err=%v, want created=false", created, err)
	}
	// A different reporter inserts.
	if _, err := repo.Insert(ctx, Complaint{TargetType: "event", TargetID: eventID, ReporterUserID: uuid.Must(uuid.NewV4()), Category: "other", Note: "", Status: "open"}); err != nil {
		t.Fatalf("second reporter insert: %v", err)
	}

	groups, err := repo.InboxGroups(ctx)
	if err != nil {
		t.Fatalf("inbox: %v", err)
	}
	var g *EventReportGroup
	for i := range groups {
		if groups[i].TargetID == eventID {
			g = &groups[i]
		}
	}
	if g == nil || g.ReportCount != 2 {
		t.Fatalf("group = %+v, want ReportCount 2", g)
	}

	// Dismiss cascades to all open complaints + writes one audit row.
	actor := uuid.Must(uuid.NewV4())
	n, err := repo.ResolveOpenForTarget(ctx, "event", eventID, actor, "dismissed", "not a violation")
	if err != nil || n != 2 {
		t.Fatalf("resolve: n=%d err=%v, want 2", n, err)
	}
	open, err := repo.OpenEventCount(ctx)
	if err != nil {
		t.Fatalf("open count: %v", err)
	}
	_ = open // baseline-relative; just assert the event is gone from groups
	groups, _ = repo.InboxGroups(ctx)
	for _, gr := range groups {
		if gr.TargetID == eventID {
			t.Fatalf("event still in inbox after dismiss")
		}
	}
	var auditCount int
	if _, err := db.QueryOne(pg.Scan(&auditCount),
		`SELECT count(*) FROM audit_log WHERE target_id = ? AND action = 'complaint.resolve'`, eventID); err != nil {
		t.Fatalf("audit count: %v", err)
	}
	if auditCount != 1 {
		t.Fatalf("audit rows = %d, want 1", auditCount)
	}
}
