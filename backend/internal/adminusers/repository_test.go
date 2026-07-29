//go:build integration

package adminusers

import (
	"context"
	"os"
	"testing"

	"github.com/go-pg/pg/v10"
	"github.com/gofrs/uuid"
)

// Run with a migrated Postgres:
//
//	TEST_DATABASE_URL=postgres://lia:lia@localhost:5432/lia_test?sslmode=disable \
//	  go test -tags=integration ./internal/adminusers/ -v
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

func TestList_CountsOnlyActiveRsvpStatuses(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	repo := NewRepository(db)

	uid := uuid.Must(uuid.NewV4())
	eid1 := uuid.Must(uuid.NewV4())
	eid2 := uuid.Must(uuid.NewV4())
	if _, err := db.Exec(
		`INSERT INTO users (uuid, email, name, status, role) VALUES (?, ?, 'Registry Probe', 'active', 'common')`,
		uid, "registry-probe-"+uid.String()+"@example.test"); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM users WHERE uuid = ?`, uid) })

	if _, err := db.Exec(
		`INSERT INTO event_rsvps (event_id, user_id, status) VALUES (?, ?, 'going'), (?, ?, 'cancelled')`,
		eid1, uid, eid2, uid); err != nil {
		t.Fatalf("insert rsvps: %v", err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM event_rsvps WHERE user_id = ?`, uid) })

	rows, err := repo.List(ctx, Filter{Query: "Registry Probe", Limit: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
	if rows[0].Bookings != 1 {
		t.Fatalf("Bookings = %d, want 1 (cancelled must not count)", rows[0].Bookings)
	}
	if rows[0].IsOrganizer {
		t.Fatalf("IsOrganizer = true, want false")
	}
}
