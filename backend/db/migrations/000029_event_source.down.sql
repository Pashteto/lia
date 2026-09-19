ALTER TABLE events
    DROP COLUMN IF EXISTS source_url,
    DROP COLUMN IF EXISTS source_label;
