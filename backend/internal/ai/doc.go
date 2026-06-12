// Package ai is a planned domain module of the Lia monolith.
//
// Responsibility: the curatorial AI search assistant — parse a natural-language
// query into structured filters, run them against the real events DB, and
// summarize. The AI never invents events; it answers only over backend results.
// Provider via an adapter (GigaChat / YandexGPT; others only if permitted).
//
// Status: skeleton. See docs/event_discovery_mvp_technical_stack.md §5.7.
package ai
