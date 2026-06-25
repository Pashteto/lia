ALTER TABLE users
    ADD COLUMN organizations UUID[] DEFAULT '{}';

CREATE INDEX users_organizations_idx ON users USING GIN (organizations);
