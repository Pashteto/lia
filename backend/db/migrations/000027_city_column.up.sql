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

-- Public listing filters by city and orders by start time.
CREATE INDEX IF NOT EXISTS events_city_starts_at_idx
    ON events (city, starts_at);
