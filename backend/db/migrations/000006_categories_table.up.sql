-- Curated event category taxonomy. Replaces the denormalized events.category
-- text column (migration 000005) — see 000007 for the join + backfill.
CREATE TABLE IF NOT EXISTS categories
(
    id          uuid NOT NULL
        CONSTRAINT category_id_pkey PRIMARY KEY,
    slug        text NOT NULL
        CONSTRAINT category_slug_unique UNIQUE,
    label       text NOT NULL,
    sort_order  integer NOT NULL DEFAULT 0,
    created_at  timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- reuse update_updated_at_column() from migration 000001
CREATE TRIGGER update_category_updated_at
    BEFORE UPDATE
    ON categories
    FOR EACH ROW
EXECUTE PROCEDURE update_updated_at_column();

-- Seed the curated set. gen_random_uuid() is built into PostgreSQL 13+
-- (the project's postgis image ships PG 14+).
INSERT INTO categories (id, slug, label, sort_order) VALUES
    (gen_random_uuid(), 'lecture',     'Лекции',        10),
    (gen_random_uuid(), 'workshop',    'Мастер-классы', 20),
    (gen_random_uuid(), 'mediation',   'Медиации',      30),
    (gen_random_uuid(), 'concert',     'Концерты',      40),
    (gen_random_uuid(), 'exhibition',  'Выставки',      50),
    (gen_random_uuid(), 'performance', 'Спектакли',     60),
    (gen_random_uuid(), 'film',        'Кино',          70),
    (gen_random_uuid(), 'festival',    'Фестивали',     80)
ON CONFLICT (slug) DO NOTHING;
