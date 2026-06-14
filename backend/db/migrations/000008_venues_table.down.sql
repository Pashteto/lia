-- Re-add the denormalized columns and repopulate from the linked venue.
-- NOTE: lossy round-trip — only name + metro are restored; address/district
-- are dropped with the venues table, and events whose venue_id is the zero
-- UUID (no venue) keep the blank column default.
ALTER TABLE events
    ADD COLUMN IF NOT EXISTS venue_name  text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS venue_metro text NOT NULL DEFAULT '';

UPDATE events e
SET venue_name  = ven.name,
    venue_metro = ven.metro
FROM venues ven
WHERE ven.id = e.venue_id;

DROP TABLE IF EXISTS venues;
