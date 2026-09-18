DROP INDEX IF EXISTS events_unreviewed_idx;
ALTER TABLE events
    DROP COLUMN IF EXISTS reviewed_by,
    DROP COLUMN IF EXISTS reviewed_at;
