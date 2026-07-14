//go:build integration

// Insert a feedback row, then ListForEvent returns it with the author name and
// no email; ExistsForUser is true afterwards; EventGate returns the seeded owner
// and end instant. (Seed events/users/event_rsvps directly with INSERTs.)
package feedback_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Pashteto/lia/internal/feedback"
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

// seedEndedEvent inserts an event owned by ownerID that already ended.
func seedEndedEvent(t *testing.T, db *pg.DB, ownerID uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.Must(uuid.NewV4())
	endsAt := time.Now().Add(-24 * time.Hour)
	if _, err := db.Exec(
		`INSERT INTO events (id, title, status, organizer_id, starts_at, ends_at, created_at, updated_at)
		 VALUES (?, 'feedback test event', 'published', ?, ?, ?, now(), now())`,
		id, ownerID, endsAt.Add(-time.Hour), endsAt); err != nil {
		t.Fatalf("insert test event: %v", err)
	}
	t.Cleanup(func() {
		db.Exec(`DELETE FROM event_feedback WHERE event_id = ?`, id) //nolint:errcheck
		db.Exec(`DELETE FROM event_rsvps WHERE event_id = ?`, id)    //nolint:errcheck
		db.Exec(`DELETE FROM events WHERE id = ?`, id)               //nolint:errcheck
	})
	return id
}

func seedUser(t *testing.T, db *pg.DB, name string) uuid.UUID {
	t.Helper()
	id := uuid.Must(uuid.NewV4())
	email := "feedback-test-" + id.String() + "@example.test"
	if _, err := db.Exec(
		`INSERT INTO users (uuid, email, name) VALUES (?, ?, ?)`,
		id, email, name); err != nil {
		t.Fatalf("insert test user: %v", err)
	}
	t.Cleanup(func() {
		db.Exec(`DELETE FROM users WHERE uuid = ?`, id) //nolint:errcheck
	})
	return id
}

func seedGoingRsvp(t *testing.T, db *pg.DB, eventID, userID uuid.UUID) {
	t.Helper()
	if _, err := db.Exec(
		`INSERT INTO event_rsvps (event_id, user_id, status) VALUES (?, ?, 'going')`,
		eventID, userID); err != nil {
		t.Fatalf("insert test rsvp: %v", err)
	}
}

func TestRepository_EndToEnd(t *testing.T) {
	db := openTestDB(t)
	repo := feedback.NewRepository(db)
	ctx := context.Background()

	owner := seedUser(t, db, "Owner Person")
	author := seedUser(t, db, "Author Person")
	eventID := seedEndedEvent(t, db, owner)
	seedGoingRsvp(t, db, eventID, author)

	gotOwner, endsAt, exists, err := repo.EventGate(ctx, eventID)
	if err != nil {
		t.Fatalf("EventGate: %v", err)
	}
	if !exists {
		t.Fatalf("EventGate: exists = false, want true")
	}
	if gotOwner != owner {
		t.Fatalf("EventGate owner = %s, want %s", gotOwner, owner)
	}
	if !endsAt.Before(time.Now()) {
		t.Fatalf("EventGate endsAt = %v, want in the past", endsAt)
	}

	active, err := repo.HasActiveRsvp(ctx, eventID, author)
	if err != nil {
		t.Fatalf("HasActiveRsvp: %v", err)
	}
	if !active {
		t.Fatalf("HasActiveRsvp = false, want true")
	}

	existsBefore, err := repo.ExistsForUser(ctx, eventID, author)
	if err != nil {
		t.Fatalf("ExistsForUser (before): %v", err)
	}
	if existsBefore {
		t.Fatalf("ExistsForUser (before) = true, want false")
	}

	if err := repo.Insert(ctx, feedback.Feedback{
		EventID: eventID,
		UserID:  author,
		Rating:  5,
		Comment: "great event",
	}); err != nil {
		t.Fatalf("Insert: %v", err)
	}

	existsAfter, err := repo.ExistsForUser(ctx, eventID, author)
	if err != nil {
		t.Fatalf("ExistsForUser (after): %v", err)
	}
	if !existsAfter {
		t.Fatalf("ExistsForUser (after) = false, want true")
	}

	items, err := repo.ListForEvent(ctx, eventID)
	if err != nil {
		t.Fatalf("ListForEvent: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("ListForEvent len = %d, want 1", len(items))
	}
	item := items[0]
	if item.Rating != 5 {
		t.Fatalf("item.Rating = %d, want 5", item.Rating)
	}
	if item.AuthorName != "Author Person" {
		t.Fatalf("item.AuthorName = %q, want %q", item.AuthorName, "Author Person")
	}
	if item.Comment != "great event" {
		t.Fatalf("item.Comment = %q, want %q", item.Comment, "great event")
	}
	// Item has no email field at all — the join only ever selects name.
}
