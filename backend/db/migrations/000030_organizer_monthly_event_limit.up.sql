-- Per-organizer override of the monthly event-creation cap.
--
-- Mirrors daily_event_limit from migration 24. The global default lives in
-- config (EVENTS_MONTHLY_LIMIT, 10); NULL here means "use the default", a
-- number overrides it for this organizer alone, and 0 means "no monthly cap".
--
-- The reason it exists: the editorial account that republishes announcements
-- creates them in batches and hit the global cap on every import, which was
-- worked around by raising EVENTS_MONTHLY_LIMIT on the box and putting it back
-- afterwards — three times in September 2026 alone, each one a recreate of the
-- production container.
ALTER TABLE organizers ADD COLUMN monthly_event_limit int;

COMMENT ON COLUMN organizers.monthly_event_limit IS
    'Per-organizer monthly event-creation cap; NULL = use the global default, 0 = uncapped';
