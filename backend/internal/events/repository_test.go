//go:build integration

package events

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/gofrs/uuid"
)

// openTestDB opens a *pg.DB from the TEST_DATABASE_URL env var and skips the
// test if it is unset. Mirrors the pattern used in internal/moderation.
//
// Run with a migrated Postgres:
//
//	TEST_DATABASE_URL=postgres://lia:lia@localhost:5432/lia_test?sslmode=disable \
//	  go test -tags=integration ./internal/events/ -run TestSetCapacityTx -v
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

// newTestRepo returns a pgRepository (as Repository) and the underlying *pg.DB
// for direct seed/assert queries.
func newTestRepo(t *testing.T) (Repository, *pg.DB) {
	t.Helper()
	db := openTestDB(t)
	return NewRepository(db, nil), db
}

// seedPublishedEvent inserts a minimal published event with the given
// capacity (nil = unlimited) and returns its ID. Cleaned up via t.Cleanup.
func seedPublishedEvent(t *testing.T, db *pg.DB, capacity int) uuid.UUID {
	t.Helper()
	id := uuid.Must(uuid.NewV4())
	_, err := db.Exec(
		`INSERT INTO events (id, title, status, capacity, starts_at, created_at, updated_at)
		 VALUES (?, 'integration test event', 'published', ?, ?, now(), now())`,
		id, capacity, time.Now().Add(24*time.Hour),
	)
	if err != nil {
		t.Fatalf("seed published event: %v", err)
	}
	t.Cleanup(func() {
		db.Exec(`DELETE FROM event_rsvps WHERE event_id = ?`, id) //nolint:errcheck
		db.Exec(`DELETE FROM events WHERE id = ?`, id)             //nolint:errcheck
		db.Exec(`DELETE FROM audit_log WHERE target_id = ?`, id)   //nolint:errcheck
	})
	return id
}

// seedRsvp inserts a minimal rsvp row for a fresh random user on eventID with
// the given status and returns the rsvp's ID.
func seedRsvp(t *testing.T, db *pg.DB, eventID uuid.UUID, status string) uuid.UUID {
	t.Helper()
	id := uuid.Must(uuid.NewV4())
	userID := uuid.Must(uuid.NewV4())
	_, err := db.Exec(
		`INSERT INTO event_rsvps (id, event_id, user_id, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, now(), now())`,
		id, eventID, userID, status,
	)
	if err != nil {
		t.Fatalf("seed rsvp: %v", err)
	}
	return id
}

// assertRsvpStatus fails the test if the rsvp's status does not match want.
func assertRsvpStatus(t *testing.T, db *pg.DB, rsvpID uuid.UUID, want string) {
	t.Helper()
	var got string
	if _, err := db.QueryOne(pg.Scan(&got), `SELECT status FROM event_rsvps WHERE id = ?`, rsvpID); err != nil {
		t.Fatalf("query rsvp status: %v", err)
	}
	if got != want {
		t.Errorf("rsvp %s status = %q, want %q", rsvpID, got, want)
	}
}

func intp(n int) *int { return &n }

func TestSetCapacityTx_PromotesWaitlist(t *testing.T) {
	r, db := newTestRepo(t)
	ev := seedPublishedEvent(t, db, 1)
	seedRsvp(t, db, ev, "going") // occupies the 1 seat
	wl := seedRsvp(t, db, ev, "waitlist")

	promoted, err := r.SetCapacityTx(ev, intp(2))
	if err != nil {
		t.Fatalf("SetCapacityTx: %v", err)
	}
	if promoted != 1 {
		t.Fatalf("want 1 promoted, got %d", promoted)
	}
	assertRsvpStatus(t, db, wl, "going")
}

func TestSetCapacityTx_BelowOccupied(t *testing.T) {
	r, db := newTestRepo(t)
	ev := seedPublishedEvent(t, db, 2)
	seedRsvp(t, db, ev, "going")
	seedRsvp(t, db, ev, "going")
	_, err := r.SetCapacityTx(ev, intp(1))
	if !errors.Is(err, ErrCapacityBelowOccupied) {
		t.Fatalf("want ErrCapacityBelowOccupied, got %v", err)
	}
}
