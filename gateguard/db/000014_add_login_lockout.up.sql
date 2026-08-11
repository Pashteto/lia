-- Temporary account lockout after N consecutive failed password logins.
-- failed_login_attempts counts consecutive failures (reset on success or once a
-- lock expires); login_locked_until, when in the future, blocks sign-in. Paired
-- with the per-IP rate limiter on the Lia edge — this bounds distributed
-- (per-account) brute force that a single-IP limiter cannot see.
ALTER TABLE users ADD COLUMN IF NOT EXISTS failed_login_attempts int NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS login_locked_until timestamptz;
