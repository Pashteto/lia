DROP INDEX IF EXISTS events_city_starts_at_idx;
DROP INDEX IF EXISTS venue_city_name_lower_idx;
CREATE INDEX IF NOT EXISTS venue_name_lower_idx ON venues (lower(name));
ALTER TABLE events DROP COLUMN IF EXISTS city;
ALTER TABLE venues DROP COLUMN IF EXISTS city;
