package models

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
)

func TestRsvpValidate(t *testing.T) {
	good := &Rsvp{EventID: uuid.Must(uuid.NewV4()), UserID: uuid.Must(uuid.NewV4()), Status: RsvpGoing}
	if err := good.Validate(); err != nil {
		t.Fatalf("expected valid rsvp, got %v", err)
	}

	noEvent := &Rsvp{UserID: uuid.Must(uuid.NewV4()), Status: RsvpGoing}
	if err := noEvent.Validate(); err == nil {
		t.Fatal("expected error when event_id is missing")
	}

	badStatus := &Rsvp{EventID: uuid.Must(uuid.NewV4()), UserID: uuid.Must(uuid.NewV4()), Status: "bogus"}
	if err := badStatus.Validate(); err == nil {
		t.Fatal("expected error for invalid status")
	}
}

func TestRsvpBeforeInsertSetsID(t *testing.T) {
	r := &Rsvp{EventID: uuid.Must(uuid.NewV4()), UserID: uuid.Must(uuid.NewV4()), Status: RsvpApplied}
	if _, err := r.BeforeInsert(context.Background()); err != nil {
		t.Fatalf("BeforeInsert: %v", err)
	}
	if r.ID == uuid.Nil {
		t.Fatal("expected ID to be generated")
	}
}
