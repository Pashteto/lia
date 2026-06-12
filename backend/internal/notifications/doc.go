// Package notifications is a planned domain module of the Lia monolith.
//
// Responsibility: transactional email (confirmations, magic links, RSVP
// reminders, moderation notices) and later APNs push. Backed by a job queue
// (PostgreSQL outbox + Redis/Asynq) per the tech stack.
//
// Status: skeleton. See docs/event_discovery_mvp_technical_stack.md §5.1, §5.2.
package notifications
