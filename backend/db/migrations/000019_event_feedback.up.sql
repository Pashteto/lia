CREATE TABLE event_feedback (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id   uuid NOT NULL,
    user_id    uuid NOT NULL,
    rating     smallint NOT NULL CHECK (rating BETWEEN 1 AND 5),
    comment    text,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (event_id, user_id)
);

-- Owner's list/aggregate query is by event.
CREATE INDEX event_feedback_event_idx ON event_feedback (event_id);
