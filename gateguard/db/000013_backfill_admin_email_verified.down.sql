-- Irreversible in spirit: we don't record which admin accounts were
-- email_verified=false before this migration ran, and any admin who has since
-- completed real verification must not be falsely un-verified. No-op keeps
-- `migrate down` consistent without destroying legitimate verification state.
SELECT 1;
