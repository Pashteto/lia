ALTER TABLE IF EXISTS users
    DROP COLUMN IF EXISTS organizations;

DROP INDEX IF EXISTS users_organizations_idx;
