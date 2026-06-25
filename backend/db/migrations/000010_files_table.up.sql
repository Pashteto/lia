CREATE TABLE IF NOT EXISTS files (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    storage_key   text NOT NULL UNIQUE,
    content_type  text NOT NULL,
    size          bigint NOT NULL DEFAULT 0,
    owner_user_id uuid NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    created_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_files_owner ON files (owner_user_id);
CREATE INDEX IF NOT EXISTS idx_files_created_at ON files (created_at);
