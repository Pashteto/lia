CREATE TABLE app_settings (
    key        TEXT PRIMARY KEY,
    value      jsonb NOT NULL DEFAULT '{}',
    updated_at timestamptz NOT NULL DEFAULT now(),
    updated_by uuid
);

INSERT INTO app_settings (key, value) VALUES
    ('organizers.auto_verify_all', '{"enabled": false}')
    ON CONFLICT (key) DO NOTHING;
