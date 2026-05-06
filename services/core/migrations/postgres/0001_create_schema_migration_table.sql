CREATE TABLE IF NOT EXISTS forge_schema_migrations (
  version INTEGER PRIMARY KEY,
  name TEXT NOT NULL,
  applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE forge_schema_migrations
  ADD COLUMN IF NOT EXISTS checksum TEXT NOT NULL DEFAULT '';
