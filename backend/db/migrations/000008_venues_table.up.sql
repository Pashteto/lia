-- Dedicated venue entity. Replaces the denormalized events.venue_name /
-- events.venue_metro columns (migration 000005). Identity only — coordinates
-- and PostGIS "events nearby" are a separate later spec.
CREATE TABLE IF NOT EXISTS venues
(
    id          uuid NOT NULL
        CONSTRAINT venue_id_pkey PRIMARY KEY,
    name        text NOT NULL,
    address     text NOT NULL DEFAULT '',
    metro       text NOT NULL DEFAULT '',
    district    text NOT NULL DEFAULT '',
    created_at  timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- case-insensitive search + find-or-create lookup
CREATE INDEX IF NOT EXISTS venue_name_lower_idx
    ON venues (lower(name));

-- reuse update_updated_at_column() from migration 000001
CREATE TRIGGER update_venue_updated_at
    BEFORE UPDATE
    ON venues
    FOR EACH ROW
EXECUTE PROCEDURE update_updated_at_column();

-- Backfill: one venue per distinct normalized venue_name, carrying its metro.
-- gen_random_uuid() is built into PostgreSQL 13+.
INSERT INTO venues (id, name, metro)
SELECT gen_random_uuid(), v.name, v.metro
FROM (
    SELECT DISTINCT ON (lower(btrim(venue_name)))
           btrim(venue_name)  AS name,
           btrim(venue_metro) AS metro
    FROM events
    WHERE btrim(coalesce(venue_name, '')) <> ''
    ORDER BY lower(btrim(venue_name)), venue_name
) v;

-- Link events to the matching venue.
UPDATE events e
SET venue_id = ven.id
FROM venues ven
WHERE lower(btrim(e.venue_name)) = lower(ven.name)
  AND btrim(coalesce(e.venue_name, '')) <> '';

-- Drop the now-normalized denormalized columns.
ALTER TABLE events
    DROP COLUMN IF EXISTS venue_name,
    DROP COLUMN IF EXISTS venue_metro;
