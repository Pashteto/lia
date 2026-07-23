-- Accounts created before email-verification shipped carry email_verified=false.
-- Grandfather pre-existing admin accounts so they aren't shown the "unverified"
-- banner in the UI. Targets by role (not by email list) because the real admin
-- addresses live only in prod environment config, not in this repo.
-- `role` is the `user_role` enum ('common', 'viewer', 'billing', 'admin')
-- added in 000005_add_user_roles.up.sql; 'admin' is the exact stored value.
UPDATE users
SET email_verified = true
WHERE role = 'admin'
  AND email_verified = false;
