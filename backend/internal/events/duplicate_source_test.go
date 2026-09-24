package events

import (
	"errors"
	"fmt"
	"testing"
)

// --- Duplicate source translation ---
//
// Migration 31 makes (source_url, starts_at) unique for imported events, so a
// second import of the same announcement now fails in Postgres. The editor must
// see why, not a 503, so the constraint violation is translated on the way out.

// fakePGError implements pg.Error so the translation can be tested without a
// database. Field 'C' carries the SQLSTATE, 'n' the constraint name.
type fakePGError struct {
	code       string
	constraint string
}

func (e fakePGError) Error() string { return "fake pg error " + e.code }
func (e fakePGError) Field(f byte) string {
	switch f {
	case 'C':
		return e.code
	case 'n':
		return e.constraint
	}
	return ""
}
func (e fakePGError) IntegrityViolation() bool { return e.code == "23505" }

func TestIsDuplicateSource_RecognisesTheSourceIndex(t *testing.T) {
	err := fmt.Errorf("insert event: %w",
		fakePGError{code: "23505", constraint: "events_source_url_starts_at_idx"})
	if !isDuplicateSource(err) {
		t.Error("a unique violation on the source index was not recognised")
	}
}

func TestIsDuplicateSource_IgnoresOtherConstraints(t *testing.T) {
	err := fmt.Errorf("insert event: %w",
		fakePGError{code: "23505", constraint: "event_rsvps_event_user_unique"})
	if isDuplicateSource(err) {
		t.Error("an unrelated unique violation was reported as a duplicate source")
	}
}

func TestIsDuplicateSource_IgnoresOrdinaryErrors(t *testing.T) {
	if isDuplicateSource(errors.New("connection refused")) {
		t.Error("a plain error was reported as a duplicate source")
	}
	if isDuplicateSource(nil) {
		t.Error("nil was reported as a duplicate source")
	}
}
