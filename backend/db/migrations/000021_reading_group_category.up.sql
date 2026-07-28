-- Add «Читательские группы» to the curated category taxonomy (see 000006).
INSERT INTO categories (id, slug, label, sort_order) VALUES
    (gen_random_uuid(), 'reading-group', 'Читательские группы', 15)
ON CONFLICT (slug) DO NOTHING;
