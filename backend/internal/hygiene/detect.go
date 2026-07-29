// Package hygiene detects content that should not be in the public feed —
// test/QA leftovers and nonsense prices — and hides it in bulk through the
// moderation domain. It backs the A4 «Гигиена контента» rail.
package hygiene

import (
	"regexp"

	"github.com/gofrs/uuid"
)

// Kind enumerates the detected issue types rendered by the A4 rail.
type Kind string

const (
	KindTestData        Kind = "test_data"
	KindSuspiciousPrice Kind = "suspicious_price"
)

// SuspiciousPriceRUB is the whole-ruble threshold above which a ticket price
// reads as data-entry nonsense rather than a real price (the handoff's
// «от 100 500 ₽» case). Deliberate constant, not configuration.
const SuspiciousPriceRUB int64 = 100000

// testPattern deliberately mirrors frontend/lib/admin-test-heuristic.ts
// (`/QA|тест|test|\.test\b|bla\s*bla/i`). Keep the two in sync: TypeScript
// cannot be imported here, and the frontend colours rows with its own copy.
var testPattern = regexp.MustCompile(`(?i)QA|тест|test|\.test\b|bla\s*bla`)

// Candidate is one published event considered for hygiene detection.
type Candidate struct {
	EventID       uuid.UUID
	Title         string
	OrganizerName string
	PriceType     string // "free" | anything else
	PriceMin      *int64
	PriceMax      *int64
}

// Issue is one detected problem — one rendered block in the A4 rail.
type Issue struct {
	Kind          Kind      `json:"kind"`
	EventID       uuid.UUID `json:"event_id"`
	Title         string    `json:"title"`
	OrganizerName string    `json:"organizer_name,omitempty"`
	PriceRUB      *int64    `json:"price_rub,omitempty"`
}

// Detect is pure: it maps candidates to issues, preserving input order. An
// event yields at most one issue and test data outranks a suspicious price
// (hiding it is the same action either way; the caption should name the
// stronger signal).
func Detect(candidates []Candidate) []Issue {
	issues := make([]Issue, 0, len(candidates))
	for _, c := range candidates {
		switch {
		case testPattern.MatchString(c.Title) || testPattern.MatchString(c.OrganizerName):
			issues = append(issues, Issue{
				Kind: KindTestData, EventID: c.EventID,
				Title: c.Title, OrganizerName: c.OrganizerName,
			})
		case suspiciousPrice(c) != nil:
			issues = append(issues, Issue{
				Kind: KindSuspiciousPrice, EventID: c.EventID,
				Title: c.Title, OrganizerName: c.OrganizerName, PriceRUB: suspiciousPrice(c),
			})
		}
	}
	return issues
}

// suspiciousPrice returns the offending price, or nil when the candidate's
// price is free, unset, or plausible.
func suspiciousPrice(c Candidate) *int64 {
	if c.PriceType == "free" {
		return nil
	}
	for _, p := range []*int64{c.PriceMin, c.PriceMax} {
		if p != nil && *p >= SuspiciousPriceRUB {
			return p
		}
	}
	return nil
}

// ReasonFor is the moderation reason written to event_status_history and
// audit_log when an issue is hidden from the feed.
func ReasonFor(k Kind) string {
	if k == KindSuspiciousPrice {
		return "Гигиена контента: подозрительная цена"
	}
	return "Гигиена контента: тестовые данные"
}
