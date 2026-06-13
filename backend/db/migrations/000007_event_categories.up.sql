-- Many-to-many between events and the curated categories taxonomy.
CREATE TABLE IF NOT EXISTS event_categories
(
    event_id    uuid NOT NULL
        CONSTRAINT event_categories_event_id_fkey
            REFERENCES events(id) ON DELETE CASCADE,
    category_id uuid NOT NULL
        CONSTRAINT event_categories_category_id_fkey
            REFERENCES categories(id) ON DELETE RESTRICT,
    CONSTRAINT event_categories_pkey PRIMARY KEY (event_id, category_id)
);

-- reverse lookup: "events in category X"
CREATE INDEX IF NOT EXISTS event_categories_category_idx
    ON event_categories USING btree(category_id);

-- Backfill from the denormalized events.category text. The create form stored
-- free-text Russian LABELS (not slugs), so match on categories.label,
-- case-insensitive and trimmed. Non-empty values that match no seeded label are
-- left uncategorized (acceptable at current demo-scale data volume).
INSERT INTO event_categories (event_id, category_id)
SELECT e.id, c.id
FROM events e
JOIN categories c ON lower(btrim(e.category)) = lower(c.label)
WHERE btrim(coalesce(e.category, '')) <> ''
ON CONFLICT DO NOTHING;

-- Drop the now-normalized denormalized column + its index.
DROP INDEX IF EXISTS event_category_idx;
ALTER TABLE events DROP COLUMN IF EXISTS category;
