package rsvp

import (
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"github.com/Pashteto/lia/internal/models"
)

func TestBuildICS(t *testing.T) {
	start := time.Date(2026, 7, 1, 18, 0, 0, 0, time.UTC)
	e := &models.Event{
		ID:          uuid.Must(uuid.NewV4()),
		Title:       "Чтение вслух",
		Description: "встреча",
		StartsAt:    start,
	}
	out := string(buildICS(e))
	for _, want := range []string{
		"BEGIN:VCALENDAR", "BEGIN:VEVENT", "END:VEVENT", "END:VCALENDAR",
		"SUMMARY:Чтение вслух", "UID:" + e.ID.String(), "DTSTART:20260701T180000Z",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("ics missing %q in:\n%s", want, out)
		}
	}
}
