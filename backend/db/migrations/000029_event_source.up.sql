-- Source attribution for imported events.
--
-- The venues whose channels we republish agreed on one condition: every
-- imported event must credit its source with a link. There was nowhere to put
-- that link — provenance rested on the organizer ("Редакция PRESENCE") and on
-- external_registration_url, which is a different thing entirely: that field is
-- where a visitor signs up, and it is checked against the trusted-platform
-- whitelist. A Telegram channel is not on that whitelist, so crediting a source
-- there would have pushed every imported event into moderation.
--
-- Both columns are nullable on purpose: events an organizer posts about their
-- own venue have no external source, and the 87 events imported before this
-- migration are left untouched.
ALTER TABLE events
    ADD COLUMN IF NOT EXISTS source_url   TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS source_label TEXT NOT NULL DEFAULT '';

COMMENT ON COLUMN events.source_url IS 'Ссылка на первоисточник анонса (пост в канале, страница площадки). Пусто у событий от самих организаторов.';
COMMENT ON COLUMN events.source_label IS 'Как подписать источник в карточке; пусто — подставляется хост из source_url.';
