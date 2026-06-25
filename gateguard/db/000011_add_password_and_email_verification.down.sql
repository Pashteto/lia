DROP INDEX IF EXISTS user_email_verification_token_idx;
ALTER TABLE users DROP COLUMN IF EXISTS email_verification_sent_at;
ALTER TABLE users DROP COLUMN IF EXISTS email_verification_token;
ALTER TABLE users DROP COLUMN IF EXISTS email_verified;
ALTER TABLE users DROP COLUMN IF EXISTS password_hash;
