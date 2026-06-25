-- migration schema
CREATE TABLE IF NOT EXISTS schema_migrations
(
  version BIGINT NOT NULL
    CONSTRAINT schema_migrations_pkey
      PRIMARY KEY,
  dirty BOOLEAN NOT NULL
);

-- update trigger
CREATE OR REPLACE function update_update_at_column()
  RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE 'plpgsql';