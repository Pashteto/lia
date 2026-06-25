-- Password credentials + email-verification (stub) support.
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash text;
ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verified boolean NOT NULL DEFAULT false;
ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verification_token text;
ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verification_sent_at timestamp without time zone;

CREATE INDEX IF NOT EXISTS user_email_verification_token_idx
    ON users USING btree(email_verification_token);
