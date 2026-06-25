//go:build integration

package moderation

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/gofrs/uuid"
)

// openTestDB opens a *pg.DB from the TEST_DATABASE_URL env var and skips the
// test if it is unset. It mirrors the s3smoke gating pattern used for the
// storage smoke test (build tag + env var).
//
// Run with a migrated Postgres:
//
//	TEST_DATABASE_URL=postgres://lia:lia@localhost:5432/lia_test?sslmode=disable \
//	  go test -tags=integration ./internal/moderation/ -v
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
	// quick connectivity check
	if _, err := db.Exec("SELECT 1"); err != nil {
		db.Close()
		t.Fatalf("connect to test DB: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// insertPublishedEvent inserts a minimal event row with status='published' and
// returns its ID. The event is cleaned up in a t.Cleanup defer.
func insertPublishedEvent(t *testing.T, db *pg.DB) uuid.UUID {
	t.Helper()
	id := uuid.Must(uuid.NewV4())
	_, err := db.Exec(
		`INSERT INTO events (id, title, status, starts_at, created_at, updated_at)
		 VALUES (?, 'integration test event', 'published', ?, now(), now())`,
		id, time.Now().Add(24*time.Hour),
	)
	if err != nil {
		t.Fatalf("insert test event: %v", err)
	}
	t.Cleanup(func() {
		// cascades also delete event_status_history rows (ON DELETE CASCADE)
		db.Exec(`DELETE FROM events WHERE id = ?`, id)        //nolint:errcheck
		db.Exec(`DELETE FROM audit_log WHERE target_id = ?`, id) //nolint:errcheck
	})
	return id
}

// queryEventStatus returns the current status of an event row.
func queryEventStatus(t *testing.T, db *pg.DB, eventID uuid.UUID) string {
	t.Helper()
	var status string
	if _, err := db.QueryOne(pg.Scan(&status), `SELECT status FROM events WHERE id = ?`, eventID); err != nil {
		t.Fatalf("query event status: %v", err)
	}
	return status
}

// TestTransition_TakedownThenReinstate exercises the full repository transition
// cycle:
//  1. Insert a published event.
//  2. Takedown → assert status=rejected, history row, audit row, reason stored.
//  3. Reinstate → assert status=published, second history+audit row written.
//  4. Second Takedown on a published event succeeds again (sanity).
//  5. Double-takedown on an already-rejected event returns ErrInvalidTransition.
func TestTransition_TakedownThenReinstate(t *testing.T) {
	db := openTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	actor := uuid.Must(uuid.NewV4())
	eventID := insertPublishedEvent(t, db)

	// --- 1. Takedown published → rejected ---
	if err := repo.Takedown(ctx, eventID, actor, "spam"); err != nil {
		t.Fatalf("Takedown: %v", err)
	}
	if got := queryEventStatus(t, db, eventID); got != "rejected" {
		t.Errorf("after Takedown: status = %q, want rejected", got)
	}

	// Assert event_status_history row
	var histCount int
	if _, err := db.QueryOne(pg.Scan(&histCount),
		`SELECT count(*) FROM event_status_history
		 WHERE event_id = ? AND from_status = 'published' AND to_status = 'rejected'`,
		eventID); err != nil {
		t.Fatalf("query history: %v", err)
	}
	if histCount != 1 {
		t.Errorf("expected 1 history row after Takedown, got %d", histCount)
	}

	// Assert audit_log row
	var auditCount int
	if _, err := db.QueryOne(pg.Scan(&auditCount),
		`SELECT count(*) FROM audit_log
		 WHERE target_id = ? AND action = 'event.takedown'`,
		eventID); err != nil {
		t.Fatalf("query audit: %v", err)
	}
	if auditCount != 1 {
		t.Errorf("expected 1 audit row after Takedown, got %d", auditCount)
	}

	// Assert reason stored via LatestReason
	reason, err := repo.LatestReason(ctx, eventID)
	if err != nil {
		t.Fatalf("LatestReason: %v", err)
	}
	if reason != "spam" {
		t.Errorf("LatestReason = %q, want spam", reason)
	}

	// --- 2. Reinstate rejected → published ---
	if err := repo.Reinstate(ctx, eventID, actor); err != nil {
		t.Fatalf("Reinstate: %v", err)
	}
	if got := queryEventStatus(t, db, eventID); got != "published" {
		t.Errorf("after Reinstate: status = %q, want published", got)
	}

	// Assert second history row
	if _, err := db.QueryOne(pg.Scan(&histCount),
		`SELECT count(*) FROM event_status_history WHERE event_id = ?`,
		eventID); err != nil {
		t.Fatalf("query history count: %v", err)
	}
	if histCount != 2 {
		t.Errorf("expected 2 history rows after Reinstate, got %d", histCount)
	}

	// Assert second audit row
	if _, err := db.QueryOne(pg.Scan(&auditCount),
		`SELECT count(*) FROM audit_log WHERE target_id = ?`,
		eventID); err != nil {
		t.Fatalf("query audit count: %v", err)
	}
	if auditCount != 2 {
		t.Errorf("expected 2 audit rows after Reinstate, got %d", auditCount)
	}

	// --- 3. Second Takedown (now published again) ---
	if err := repo.Takedown(ctx, eventID, actor, "second takedown"); err != nil {
		t.Fatalf("second Takedown: %v", err)
	}
	if got := queryEventStatus(t, db, eventID); got != "rejected" {
		t.Errorf("after second Takedown: status = %q, want rejected", got)
	}

	// --- 4. Double-takedown on already-rejected → ErrInvalidTransition ---
	err = repo.Takedown(ctx, eventID, actor, "duplicate")
	if !errors.Is(err, ErrInvalidTransition) {
		t.Errorf("double-takedown: err = %v, want ErrInvalidTransition", err)
	}
}

// TestCounts verifies the overview query returns expected buckets.
func TestCounts_Integration(t *testing.T) {
	db := openTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	// Capture baseline before inserting test rows.
	before, err := repo.Counts(ctx)
	if err != nil {
		t.Fatalf("Counts baseline: %v", err)
	}

	// Insert one published event.
	eventID := insertPublishedEvent(t, db)
	actor := uuid.Must(uuid.NewV4())

	after, err := repo.Counts(ctx)
	if err != nil {
		t.Fatalf("Counts after insert: %v", err)
	}
	if after.EventsTotal != before.EventsTotal+1 {
		t.Errorf("total: got %d, want %d", after.EventsTotal, before.EventsTotal+1)
	}
	if after.EventsPublished != before.EventsPublished+1 {
		t.Errorf("published: got %d, want %d", after.EventsPublished, before.EventsPublished+1)
	}

	// Take it down — published count drops, removed count rises.
	if err := repo.Takedown(ctx, eventID, actor, "counts test"); err != nil {
		t.Fatalf("Takedown for counts: %v", err)
	}
	final, err := repo.Counts(ctx)
	if err != nil {
		t.Fatalf("Counts after takedown: %v", err)
	}
	if final.EventsPublished != before.EventsPublished {
		t.Errorf("published after takedown: got %d, want %d", final.EventsPublished, before.EventsPublished)
	}
	if final.EventsRemoved != before.EventsRemoved+1 {
		t.Errorf("removed after takedown: got %d, want %d", final.EventsRemoved, before.EventsRemoved+1)
	}
}
