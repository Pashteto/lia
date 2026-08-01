DROP INDEX IF EXISTS venues_name_trgm_idx;

-- The extensions are left installed on purpose: they are database-wide and
-- another module may have started using them. Dropping the index is enough to
-- undo what this migration added for venues.
