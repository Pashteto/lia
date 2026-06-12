// Package users is a planned domain module of the Lia monolith.
//
// Responsibility: end-user accounts and profiles, interests, saved events, and
// RSVP-owner identity. Auth (email magic link / OTP) lives in a sibling concern.
//
// Note: the template ships a users example wired through the central
// repository/service layers (internal/repository, internal/service). This
// package is where the richer user domain logic will consolidate.
//
// Status: skeleton. See docs/event_discovery_mvp_technical_stack.md §6, §13.
package users
