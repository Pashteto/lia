-- Add «Прогулки и экскурсии» to the curated category taxonomy (see 000006, 000021).
INSERT INTO categories (id, slug, label, sort_order) VALUES
    (gen_random_uuid(), 'walk', 'Прогулки и экскурсии', 90)
ON CONFLICT (slug) DO NOTHING;
