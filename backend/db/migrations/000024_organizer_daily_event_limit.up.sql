-- Per-organizer override of the daily event-creation cap.
--
-- The global default lives in config (EVENTS_DAILY_LIMIT, 3). NULL here means
-- "use the default"; a number overrides it for this organizer alone, which is
-- how an admin lifts the cap for a venue that genuinely runs many events a day.
-- 0 is a valid value and means "no daily cap for this organizer".
ALTER TABLE organizers ADD COLUMN daily_event_limit int;

COMMENT ON COLUMN organizers.daily_event_limit IS
    'Per-organizer daily event-creation cap; NULL = use the global default, 0 = uncapped';
