-- event publication lifecycle (see docs/event_discovery_mvp_technical_stack.md §2.2)
CREATE TYPE event_status AS ENUM (
    'draft',
    'pending_review',
    'published',
    'rejected',
    'cancelled'
);

-- events table
-- organizer_id / venue_id are loose references for the scaffold; FKs will be
-- added when the organizers and venues modules land.
CREATE TABLE IF NOT EXISTS events
(
    id                  uuid NOT NULL
        CONSTRAINT event_id_pkey PRIMARY KEY,
    -- zero UUID means "unset" (see internal/models/event.go); never SQL NULL
    organizer_id        uuid NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    venue_id            uuid NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    title               text NOT NULL,
    description         text NOT NULL DEFAULT '',
    status              event_status NOT NULL DEFAULT 'draft',
    format              text NOT NULL DEFAULT 'offline',
    price_type          text NOT NULL DEFAULT 'free',
    price_min           integer,
    price_max           integer,
    external_ticket_url text NOT NULL DEFAULT '',
    starts_at           timestamptz NOT NULL,
    ends_at             timestamptz,
    published_at        timestamptz,
    created_at          timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- primary discovery query: published events ordered by start time
CREATE INDEX IF NOT EXISTS event_status_starts_at_idx
    ON events USING btree(status, starts_at);

CREATE INDEX IF NOT EXISTS event_organizer_idx
    ON events USING btree(organizer_id);

-- update trigger (reuses update_updated_at_column() from 000001)
CREATE TRIGGER update_event_updated_at
    BEFORE UPDATE
    ON events
    FOR EACH ROW
EXECUTE PROCEDURE update_updated_at_column();
