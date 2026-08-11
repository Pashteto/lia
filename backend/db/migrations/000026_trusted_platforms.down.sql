-- 000026_trusted_platforms.down.sql
ALTER TABLE events DROP COLUMN IF EXISTS external_url_verified;
ALTER TABLE events DROP COLUMN IF EXISTS capacity_limited;
DROP TABLE IF EXISTS trusted_platforms;
