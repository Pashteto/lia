-- Backfill organizer profiles for people who already created events.
--
-- Until now an `organizers` row was written only by the profile form, so a user
-- who went straight to "создать событие" was an organizer in every functional
-- sense yet absent from the admin registry and the verification board, shown as
-- «Зритель» in the user list, and impossible to follow. New events now mint the
-- profile (organizers.EnsureForOwner); this catches everyone who came before.
--
-- Auto-verified, matching the new create-time behaviour. The zero uuid is the
-- same "system actor" the auto-verify path already writes to history.

WITH created AS (
    INSERT INTO organizers (owner_user_id, name, verification_status, verified_at)
    SELECT DISTINCT e.organizer_id,
           COALESCE(NULLIF(TRIM(u.name), ''), 'Организатор'),
           'verified',
           now()
      FROM events e
      JOIN users u ON u.uuid = e.organizer_id
     WHERE NOT EXISTS (
               SELECT 1 FROM organizers o WHERE o.owner_user_id = e.organizer_id
           )
    ON CONFLICT (owner_user_id) DO NOTHING
    RETURNING id
)
INSERT INTO organizer_verification_history (organizer_id, from_status, to_status, actor_user_id, reason)
SELECT id, 'draft', 'verified',
       '00000000-0000-0000-0000-000000000000'::uuid,
       'auto: профиль заведён по существующим событиям'
  FROM created;
