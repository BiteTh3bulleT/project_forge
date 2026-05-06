package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"sort"
)

type PostgresMigration struct {
	Version int
	Name    string
	SQL     string
}

func PostgresMigrations() []PostgresMigration {
	return []PostgresMigration{
		{
			Version: 1,
			Name:    "create_schema_migration_table",
			SQL:     postgresMigrationVersionTableSQL,
		},
		{
			Version: 2,
			Name:    "storage_metadata_foundation",
			SQL: `
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
`,
		},
		{
			Version: 3,
			Name:    "shadow_diagnostics_schema",
			SQL: `
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
`,
		},
		{
			Version: 4,
			Name:    "shadow_diagnostic_persistence_metadata",
			SQL: `
ALTER TABLE shadow_diagnostic_reports
  ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ;
ALTER TABLE shadow_diagnostic_reports
  ADD COLUMN IF NOT EXISTS no_effect_verified BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE shadow_diagnostic_reports
  ADD COLUMN IF NOT EXISTS schema_version INTEGER NOT NULL DEFAULT 1;

CREATE INDEX IF NOT EXISTS idx_shadow_diagnostic_reports_expires
  ON shadow_diagnostic_reports(expires_at);
`,
		},
	}
}

const postgresMigrationVersionTableSQL = `
CREATE TABLE IF NOT EXISTS forge_schema_migrations (
  version INTEGER PRIMARY KEY,
  name TEXT NOT NULL,
  applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE forge_schema_migrations
  ADD COLUMN IF NOT EXISTS checksum TEXT NOT NULL DEFAULT '';
`

type PostgresMigrationRunner struct {
	migrations []PostgresMigration
}

func NewPostgresMigrationRunner(migrations []PostgresMigration) PostgresMigrationRunner {
	copied := append([]PostgresMigration(nil), migrations...)
	return PostgresMigrationRunner{migrations: copied}
}

func (r PostgresMigrationRunner) RunForBackend(db *sql.DB, backend string) error {
	if backend != string(BackendPostgres) {
		return nil
	}
	if db == nil {
		return errors.New("postgres migration runner requires database handle")
	}
	return r.Run(context.Background(), db)
}

func (r PostgresMigrationRunner) Run(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return errors.New("postgres migration runner requires database handle")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := r.run(ctx, postgresSQLMigrationExecutor{tx: tx}); err != nil {
		return err
	}
	return tx.Commit()
}

type postgresMigrationExecutor interface {
	EnsureVersionTable(context.Context) error
	AppliedVersions(context.Context) (map[int]bool, error)
	ExecuteMigration(context.Context, PostgresMigration) error
	RecordMigration(context.Context, PostgresMigration) error
}

func (r PostgresMigrationRunner) run(ctx context.Context, exec postgresMigrationExecutor) error {
	migrations, err := normalizedPostgresMigrations(r.migrations)
	if err != nil {
		return err
	}
	if len(migrations) == 0 {
		return nil
	}
	if err := exec.EnsureVersionTable(ctx); err != nil {
		return fmt.Errorf("ensure postgres migration version table: %w", err)
	}
	applied, err := exec.AppliedVersions(ctx)
	if err != nil {
		return fmt.Errorf("load applied postgres migrations: %w", err)
	}
	for _, migration := range migrations {
		if applied[migration.Version] {
			continue
		}
		if err := exec.ExecuteMigration(ctx, migration); err != nil {
			return fmt.Errorf("apply postgres migration %04d_%s: %w", migration.Version, migration.Name, err)
		}
		if err := exec.RecordMigration(ctx, migration); err != nil {
			return fmt.Errorf("record postgres migration %04d_%s: %w", migration.Version, migration.Name, err)
		}
	}
	return nil
}

func normalizedPostgresMigrations(migrations []PostgresMigration) ([]PostgresMigration, error) {
	out := append([]PostgresMigration(nil), migrations...)
	sort.Slice(out, func(i, j int) bool {
		return out[i].Version < out[j].Version
	})
	seen := map[int]string{}
	for _, migration := range out {
		if migration.Version <= 0 {
			return nil, fmt.Errorf("invalid postgres migration version %d", migration.Version)
		}
		if migration.Name == "" {
			return nil, fmt.Errorf("postgres migration %d missing name", migration.Version)
		}
		if migration.SQL == "" {
			return nil, fmt.Errorf("postgres migration %d missing SQL", migration.Version)
		}
		if prev, ok := seen[migration.Version]; ok {
			return nil, fmt.Errorf("duplicate postgres migration version %d (%s, %s)", migration.Version, prev, migration.Name)
		}
		seen[migration.Version] = migration.Name
	}
	return out, nil
}

func (m PostgresMigration) Checksum() string {
	sum := sha256.Sum256([]byte(m.SQL))
	return fmt.Sprintf("%x", sum[:])
}

type postgresSQLMigrationExecutor struct {
	tx *sql.Tx
}

func (e postgresSQLMigrationExecutor) EnsureVersionTable(ctx context.Context) error {
	_, err := e.tx.ExecContext(ctx, postgresMigrationVersionTableSQL)
	return err
}

func (e postgresSQLMigrationExecutor) AppliedVersions(ctx context.Context) (map[int]bool, error) {
	rows, err := e.tx.QueryContext(ctx, `SELECT version FROM forge_schema_migrations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int]bool{}
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		out[version] = true
	}
	return out, rows.Err()
}

func (e postgresSQLMigrationExecutor) ExecuteMigration(ctx context.Context, migration PostgresMigration) error {
	_, err := e.tx.ExecContext(ctx, migration.SQL)
	return err
}

func (e postgresSQLMigrationExecutor) RecordMigration(ctx context.Context, migration PostgresMigration) error {
	_, err := e.tx.ExecContext(ctx, `
INSERT INTO forge_schema_migrations(version, name, checksum)
VALUES ($1, $2, $3)
ON CONFLICT (version) DO NOTHING
`, migration.Version, migration.Name, migration.Checksum())
	if err != nil {
		return err
	}
	if migration.Version == 1 {
		return nil
	}
	_, err = e.tx.ExecContext(ctx, `
INSERT INTO storage_migration_audit(migration_version, migration_name, action, status)
VALUES ($1, $2, 'apply', 'applied')
ON CONFLICT DO NOTHING
`, migration.Version, migration.Name)
	return err
}
