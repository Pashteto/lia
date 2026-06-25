CREATE TYPE user_role AS ENUM ('common', 'viewer', 'billing', 'admin');

ALTER TABLE users ADD COLUMN IF NOT EXISTS role user_role DEFAULT 'common' NOT NULL;