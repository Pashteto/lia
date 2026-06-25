ALTER TABLE events DROP CONSTRAINT IF EXISTS events_signup_mode_check;
ALTER TABLE events DROP CONSTRAINT IF EXISTS events_capacity_check;
ALTER TABLE events DROP COLUMN IF EXISTS external_registration_url;
ALTER TABLE events DROP COLUMN IF EXISTS curator_question;
ALTER TABLE events DROP COLUMN IF EXISTS capacity;
ALTER TABLE events DROP COLUMN IF EXISTS signup_mode;
