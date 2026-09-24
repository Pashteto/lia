-- One imported announcement, one event.
--
-- Nothing stopped a re-import until now: the collectors read state.json to skip
-- posts they already brought in, but nothing ever writes that file, and the
-- backend had no uniqueness on events at all. The only accidental brake was the
-- monthly quota returning 429 — and migration 30 removes exactly that brake for
-- the import account, so the guard has to become a real one.
--
-- The key is (source_url, starts_at), not source_url alone: one post routinely
-- announces a series ("ближайшие корайтинги: 3 октября, 17 октября, 31 октября"),
-- and those are separate events that must stay creatable from one source. What
-- the index forbids is the same source at the same start time — a true repeat.
--
-- Partial, so the events organizers post about their own venues (and the 87
-- imported before migration 29) keep their empty source_url without colliding.
CREATE UNIQUE INDEX IF NOT EXISTS events_source_url_starts_at_idx
    ON events (source_url, starts_at)
    WHERE source_url <> '';
