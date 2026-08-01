-- Venue search was `name ILIKE '%q%'` and nothing else, so "Фонтанки" found
-- none of the venues on the Fontanka and "Noodome" missed «Noôdome». Widening
-- the search to name/address/metro needs two stock contrib extensions:
--
--   unaccent — folds diacritics, so unaccent('Noôdome') = 'Noodome'
--   pg_trgm  — trigram similarity, so a near-miss like "NoDom" still ranks
--              «Noôdome» first instead of returning nothing
--
-- Both ship with the postgres image; CREATE EXTENSION needs superuser, which
-- the migration role already is. Deploy order matters: run this BEFORE the
-- backend image that calls unaccent()/similarity(), or venue search 500s.
CREATE EXTENSION IF NOT EXISTS unaccent;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Trigram index for the fuzzy branch. venues is small (~175 rows) so this is
-- about not regressing later, not about today's plan.
CREATE INDEX IF NOT EXISTS venues_name_trgm_idx
    ON venues USING gin (name gin_trgm_ops);
