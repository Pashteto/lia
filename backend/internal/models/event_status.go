package models

import (
	"fmt"
	"strings"
)

// EventStatus represents the publication lifecycle state of an event.
type EventStatus int

const (
	// EventDraft is a private draft, not yet submitted.
	EventDraft EventStatus = iota

	// EventPendingReview is awaiting moderation.
	EventPendingReview

	// EventPublished is live and publicly visible.
	EventPublished

	// EventRejected was declined by a moderator.
	EventRejected

	// EventCancelled was cancelled by the organizer.
	EventCancelled

	// eventStatusUnsupported is an internal sentinel for invalid statuses.
	eventStatusUnsupported
)

// eventStatuses maps EventStatus enum values to their string representations.
// Order must match the event_status enum in db/migrations/000004_events_table.
var eventStatuses = [...]string{
	EventDraft:         "draft",
	EventPendingReview: "pending_review",
	EventPublished:     "published",
	EventRejected:      "rejected",
	EventCancelled:     "cancelled",
}

// String returns the string representation of the EventStatus.
func (s EventStatus) String() string {
	if s < 0 || int(s) >= len(eventStatuses) {
		return ""
	}
	return eventStatuses[s]
}

// EventStatusFromString parses a string into an EventStatus enum.
// The comparison is case-insensitive.
func EventStatusFromString(s string) (EventStatus, error) {
	for i, r := range eventStatuses {
		if strings.EqualFold(s, r) {
			return EventStatus(i), nil
		}
	}
	return eventStatusUnsupported, fmt.Errorf("invalid event status value %q", s)
}
