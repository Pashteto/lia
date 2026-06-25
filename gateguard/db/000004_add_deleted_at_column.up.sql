ALTER TABLE users
    ADD COLUMN IF NOT EXISTS deleted_at timestamp without time zone DEFAULT NULL;
