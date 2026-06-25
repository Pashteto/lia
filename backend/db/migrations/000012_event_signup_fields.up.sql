ALTER TABLE events ADD COLUMN IF NOT EXISTS signup_mode text NOT NULL DEFAULT 'open';
ALTER TABLE events ADD COLUMN IF NOT EXISTS capacity integer;
ALTER TABLE events ADD COLUMN IF NOT EXISTS curator_question text NOT NULL DEFAULT '';
ALTER TABLE events ADD COLUMN IF NOT EXISTS external_registration_url text NOT NULL DEFAULT '';

ALTER TABLE events DROP CONSTRAINT IF EXISTS events_signup_mode_check;
ALTER TABLE events ADD CONSTRAINT events_signup_mode_check
  CHECK (signup_mode IN ('open', 'application', 'external'));
ALTER TABLE events DROP CONSTRAINT IF EXISTS events_capacity_check;
ALTER TABLE events ADD CONSTRAINT events_capacity_check
  CHECK (capacity IS NULL OR capacity > 0);
