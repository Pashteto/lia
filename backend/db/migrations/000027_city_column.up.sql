-- Second city (СПб). City lives on BOTH venues and events (denormalized):
-- listings filter events without a join, and venue-less events (external,
-- drafts) still carry a city. Slug values ('msk','spb') are validated by the
-- backend against its constant whitelist — no lookup table (YAGNI for 2 rows).
-- See docs/superpowers/specs/2026-08-23-spb-city-launch-design.md.
ALTER TABLE venues ADD COLUMN IF NOT EXISTS city text NOT NULL DEFAULT 'msk';
ALTER TABLE events ADD COLUMN IF NOT EXISTS city text NOT NULL DEFAULT 'msk';

-- Venue search is always city-scoped now; replaces the plain name index.
DROP INDEX IF EXISTS venue_name_lower_idx;
CREATE INDEX IF NOT EXISTS venue_city_name_lower_idx
    ON venues (city, lower(name));

-- The public listing predicate is status AND city AND starts_at. Replace the
-- old (status, starts_at) index (migration 000004) with the full three-column
-- form — the old one is its prefix, so nothing else regresses.
DROP INDEX IF EXISTS event_status_starts_at_idx;
CREATE INDEX IF NOT EXISTS events_status_city_starts_at_idx
    ON events (status, city, starts_at);

-- Backfill correction: the 2026-07/08 venue seed batches already included
-- Petersburg venues; without this they would all read as default 'msk'.
-- Identified by coordinates (the SPb bounding box is nowhere near Moscow's
-- 55.7/37.6). Events held at those venues follow. Venues without coordinates
-- stay 'msk' (none of the seeded SPb rows lack them).
UPDATE venues SET city = 'spb'
WHERE lat BETWEEN 59.5 AND 60.3 AND lon BETWEEN 29.3 AND 31.0;
UPDATE events SET city = 'spb'
WHERE venue_id IN (SELECT id FROM venues WHERE city = 'spb');
