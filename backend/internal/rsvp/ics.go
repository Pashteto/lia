package rsvp

import (
	"fmt"
	"strings"

	"github.com/Pashteto/lia/internal/models"
)

// buildICS renders a minimal RFC-5545 VEVENT for the event. Times are emitted in
// UTC (Z) — calendar apps localize for display; no OAuth, per spec (.ics only).
func buildICS(e *models.Event) []byte {
	const layout = "20060102T150405Z"
	var b strings.Builder
	b.WriteString("BEGIN:VCALENDAR\r\n")
	b.WriteString("VERSION:2.0\r\n")
	b.WriteString("PRODID:-//Lia//RSVP//RU\r\n")
	b.WriteString("CALSCALE:GREGORIAN\r\n")
	b.WriteString("BEGIN:VEVENT\r\n")
	fmt.Fprintf(&b, "UID:%s\r\n", e.ID.String())
	fmt.Fprintf(&b, "DTSTAMP:%s\r\n", nowFn().UTC().Format(layout))
	fmt.Fprintf(&b, "DTSTART:%s\r\n", e.StartsAt.UTC().Format(layout))
	if e.EndsAt != nil && !e.EndsAt.IsZero() {
		fmt.Fprintf(&b, "DTEND:%s\r\n", e.EndsAt.UTC().Format(layout))
	}
	fmt.Fprintf(&b, "SUMMARY:%s\r\n", icsEscape(e.Title))
	if e.Description != "" {
		fmt.Fprintf(&b, "DESCRIPTION:%s\r\n", icsEscape(e.Description))
	}
	if e.Venue != nil && e.Venue.Address != "" {
		fmt.Fprintf(&b, "LOCATION:%s\r\n", icsEscape(e.Venue.Address))
	}
	b.WriteString("END:VEVENT\r\n")
	b.WriteString("END:VCALENDAR\r\n")
	return []byte(b.String())
}

// icsEscape escapes the RFC-5545 special characters in a text value.
func icsEscape(s string) string {
	r := strings.NewReplacer("\\", "\\\\", ";", "\\;", ",", "\\,", "\n", "\\n")
	return r.Replace(s)
}
