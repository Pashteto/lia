// Package organizers is a planned domain module of the Lia monolith.
//
// Responsibility: organizer accounts and membership (museums, galleries,
// independent curators), organizer profiles, and verification status.
//
// Status: skeleton. Implement following the events module pattern
// (internal/events): a Repository over the shared *pg.DB, a Service with
// business rules, an HTTP handler wired via http.Module, and migrations under
// db/migrations. See docs/event_discovery_mvp_technical_stack.md §2.2, §13.
package organizers
