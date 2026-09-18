-- Post-moderation needs a "an admin has looked at this" marker.
--
-- The queue is post-hoc: events are already published when they reach it, so
-- «одобрить» has no status to move to and, without this column, did nothing at
-- all — every published event stayed in the queue forever (prod, 2026-09-18).
-- reviewed_at is that marker; the waiting queue is `status = 'published' AND
-- reviewed_at IS NULL`.
--
-- Deliberately NOT a foreign key on reviewed_by: users live in GateGuard, not
-- in this database (same reason audit_log.actor_user_id has no FK).
ALTER TABLE events
    ADD COLUMN reviewed_at timestamptz,
    ADD COLUMN reviewed_by uuid;

-- The queue reads exactly this predicate, ordered by starts_at.
CREATE INDEX events_unreviewed_idx ON events (starts_at)
    WHERE status = 'published' AND reviewed_at IS NULL;
