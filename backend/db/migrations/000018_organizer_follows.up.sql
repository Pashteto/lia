-- organizer_follows: a user subscribes to an organizer's events.
-- organizer_user_id is the organizer's OWNER user id (= events.organizer_id), so
-- joining follows -> events is a direct organizer_id IN (...) with no indirection.
-- The public API addresses organizers by organizers.id; handlers resolve that to
-- the owner user id before persisting. No FKs — matches the loose-reference
-- convention used by events.organizer_id and event_rsvps.user_id.
CREATE TABLE organizer_follows (
    user_id           uuid NOT NULL,
    organizer_user_id uuid NOT NULL,
    created_at        timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, organizer_user_id)
);
CREATE INDEX organizer_follows_user_idx      ON organizer_follows (user_id);
CREATE INDEX organizer_follows_organizer_idx ON organizer_follows (organizer_user_id);
