CREATE TABLE organizers (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_user_id       uuid NOT NULL UNIQUE,
    name                TEXT NOT NULL,
    description         TEXT NOT NULL DEFAULT '',
    website_url         TEXT NOT NULL DEFAULT '',
    logo_file_id        uuid NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    verification_status TEXT NOT NULL DEFAULT 'draft'
                        CHECK (verification_status IN ('draft','pending','verified','rejected')),
    auto_verify         boolean NOT NULL DEFAULT false,
    verified_at         timestamptz,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX organizers_status_idx ON organizers (verification_status, created_at DESC);

CREATE TABLE organizer_verification_history (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organizer_id  uuid NOT NULL REFERENCES organizers(id) ON DELETE CASCADE,
    from_status   TEXT NOT NULL,
    to_status     TEXT NOT NULL,
    actor_user_id uuid NOT NULL,
    reason        TEXT,
    created_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX organizer_verification_history_org_idx
    ON organizer_verification_history (organizer_id, created_at DESC);
