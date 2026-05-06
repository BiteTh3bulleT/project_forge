CREATE TABLE IF NOT EXISTS shadow_diagnostic_reports (
  report_id TEXT PRIMARY KEY,
  report_kind TEXT NOT NULL,
  workspace_id TEXT NOT NULL DEFAULT '',
  request_id TEXT NOT NULL DEFAULT '',
  correlation_id TEXT NOT NULL DEFAULT '',
  observed_at TIMESTAMPTZ,
  stored_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  summary_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  warnings_json JSONB NOT NULL DEFAULT '[]'::jsonb,
  metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS idx_shadow_diagnostic_reports_workspace
  ON shadow_diagnostic_reports(workspace_id, stored_at DESC);
CREATE INDEX IF NOT EXISTS idx_shadow_diagnostic_reports_request
  ON shadow_diagnostic_reports(request_id);
CREATE INDEX IF NOT EXISTS idx_shadow_diagnostic_reports_kind
  ON shadow_diagnostic_reports(report_kind);

CREATE TABLE IF NOT EXISTS shadow_diagnostic_report_events (
  event_id BIGSERIAL PRIMARY KEY,
  report_id TEXT NOT NULL REFERENCES shadow_diagnostic_reports(report_id) ON DELETE CASCADE,
  event_type TEXT NOT NULL,
  event_ref TEXT NOT NULL DEFAULT '',
  payload_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_shadow_diagnostic_report_events_report
  ON shadow_diagnostic_report_events(report_id, created_at);

CREATE TABLE IF NOT EXISTS shadow_diagnostic_redactions (
  redaction_id BIGSERIAL PRIMARY KEY,
  report_id TEXT NOT NULL REFERENCES shadow_diagnostic_reports(report_id) ON DELETE CASCADE,
  redaction_kind TEXT NOT NULL,
  field_path TEXT NOT NULL,
  reason TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_shadow_diagnostic_redactions_report
  ON shadow_diagnostic_redactions(report_id, created_at);
