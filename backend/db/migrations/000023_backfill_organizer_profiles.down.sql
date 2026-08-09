-- Remove only the profiles this migration minted: auto-verified, never edited
-- (empty description and website), and carrying the backfill history row.
-- Anything a human touched afterwards is left alone.
DELETE FROM organizers o
 WHERE o.description = ''
   AND o.website_url = ''
   AND o.verification_status = 'verified'
   AND EXISTS (
           SELECT 1 FROM organizer_verification_history h
            WHERE h.organizer_id = o.id
              AND h.reason = 'auto: профиль заведён по существующим событиям'
       );
