DROP INDEX IF EXISTS event_category_idx;
ALTER TABLE events
    DROP COLUMN IF EXISTS category,
    DROP COLUMN IF EXISTS venue_name,
    DROP COLUMN IF EXISTS venue_metro;
