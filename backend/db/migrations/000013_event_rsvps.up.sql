CREATE TABLE IF NOT EXISTS event_rsvps (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id           uuid NOT NULL,
    user_id            uuid NOT NULL,
    status             text NOT NULL,
    application_answer text NOT NULL DEFAULT '',
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT event_rsvps_status_check CHECK (
        status IN ('going','waitlist','applied','accepted','declined','withdrawn','cancelled')
    ),
    CONSTRAINT event_rsvps_event_user_unique UNIQUE (event_id, user_id)
);

CREATE INDEX IF NOT EXISTS event_rsvps_event_status_idx ON event_rsvps (event_id, status);
CREATE INDEX IF NOT EXISTS event_rsvps_user_idx ON event_rsvps (user_id);
