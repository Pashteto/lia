-- Re-add the denormalized column and repopulate it from the first linked
-- category's label (by sort_order) per event.
ALTER TABLE events ADD COLUMN IF NOT EXISTS category text NOT NULL DEFAULT '';

-- NOTE: lossy round-trip — only the first category by sort_order is restored.
-- Events linked to multiple categories will lose all but one.
UPDATE events e
SET category = sub.label
FROM (
    SELECT ec.event_id,
           c.label,
           row_number() OVER (PARTITION BY ec.event_id ORDER BY c.sort_order) AS rn
    FROM event_categories ec
    JOIN categories c ON c.id = ec.category_id
) sub
WHERE sub.event_id = e.id AND sub.rn = 1;

CREATE INDEX IF NOT EXISTS event_category_idx
    ON events USING btree(category);

DROP TABLE IF EXISTS event_categories;
