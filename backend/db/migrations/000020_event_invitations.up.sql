-- event_invitations: organizer invites a person (by email) to a specific event.
-- Keyed by invitee_email so an invite can exist before the person registers.
-- No FKs (loose uuid refs), matching the repo convention.
CREATE TABLE IF NOT EXISTS event_invitations (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id         uuid NOT NULL,
    inviter_user_id  uuid NOT NULL,
    invitee_email    text NOT NULL,
    token            text NOT NULL,
    status           text NOT NULL DEFAULT 'pending'
                       CHECK (status IN ('pending','accepted','declined','revoked','expired')),
    created_at       timestamptz NOT NULL DEFAULT now(),
    responded_at     timestamptz,
    expires_at       timestamptz NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS event_invitations_token_idx ON event_invitations (token);
CREATE INDEX IF NOT EXISTS event_invitations_email_status_idx ON event_invitations (lower(invitee_email), status);
-- At most one live (pending) invite per (event, email).
CREATE UNIQUE INDEX IF NOT EXISTS event_invitations_event_email_pending_idx
    ON event_invitations (event_id, lower(invitee_email)) WHERE status = 'pending';
