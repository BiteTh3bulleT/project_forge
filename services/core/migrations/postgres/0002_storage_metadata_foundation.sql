CREATE TABLE IF NOT EXISTS storage_backend_metadata (
  key TEXT PRIMARY KEY,
  backend TEXT NOT NULL,
  value_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_storage_backend_metadata_backend
  ON storage_backend_metadata(backend);

CREATE TABLE IF NOT EXISTS storage_migration_audit (
  id BIGSERIAL PRIMARY KEY,
  migration_version INTEGER NOT NULL,
  migration_name TEXT NOT NULL,
  action TEXT NOT NULL,
  status TEXT NOT NULL,
  detail_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_storage_migration_audit_version
  ON storage_migration_audit(migration_version);
CREATE INDEX IF NOT EXISTS idx_storage_migration_audit_created_at
  ON storage_migration_audit(created_at);
