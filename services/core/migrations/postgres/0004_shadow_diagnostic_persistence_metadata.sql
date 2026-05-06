ALTER TABLE shadow_diagnostic_reports
  ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ;
ALTER TABLE shadow_diagnostic_reports
  ADD COLUMN IF NOT EXISTS no_effect_verified BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE shadow_diagnostic_reports
  ADD COLUMN IF NOT EXISTS schema_version INTEGER NOT NULL DEFAULT 1;

CREATE INDEX IF NOT EXISTS idx_shadow_diagnostic_reports_expires
  ON shadow_diagnostic_reports(expires_at);
