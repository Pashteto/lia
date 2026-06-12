-- Denormalized category + venue fields on events so the discovery UI can show
-- the category kicker and venue line. These will be normalized into dedicated
-- categories / venues modules later (see internal/{venues} and the tech-stack doc).
ALTER TABLE events
    ADD COLUMN category    text NOT NULL DEFAULT '',
    ADD COLUMN venue_name  text NOT NULL DEFAULT '',
    ADD COLUMN venue_metro text NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS event_category_idx
    ON events USING btree(category);
