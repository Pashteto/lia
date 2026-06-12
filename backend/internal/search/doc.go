// Package search is a planned domain module of the Lia monolith.
//
// Responsibility: event discovery search and filtering — PostgreSQL full-text +
// trigram search, weighted ranking, and PostGIS distance filters. A dedicated
// search engine (Meilisearch/OpenSearch) is deferred per the tech stack.
//
// Status: skeleton. See docs/event_discovery_mvp_technical_stack.md §3.7.
package search
