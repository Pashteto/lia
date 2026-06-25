ALTER TABLE users
    ADD COLUMN preferred_stacks int[] DEFAULT '{}'::int[];
