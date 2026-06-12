// Package venues is a planned domain module of the Lia monolith.
//
// Responsibility: physical venues with geocoded coordinates (PostGIS), address
// normalization, metro/district, and "events nearby" geo queries.
//
// Status: skeleton. Implement following the events module pattern
// (internal/events). PostGIS is already enabled (db/migrations/000003).
// See docs/event_discovery_mvp_technical_stack.md §3.4, §5.4, §13.
package venues
