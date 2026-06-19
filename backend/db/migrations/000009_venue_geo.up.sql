ALTER TABLE venues
    ADD COLUMN lat double precision,
    ADD COLUMN lon double precision,
    ADD COLUMN geog geography(Point, 4326)
        GENERATED ALWAYS AS (ST_SetSRID(ST_MakePoint(lon, lat), 4326)::geography) STORED;

CREATE INDEX IF NOT EXISTS venues_geog_gist
    ON venues USING gist (geog);
