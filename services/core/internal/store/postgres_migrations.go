package store

import (
	"context"
	"database/sql"
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
			SQL: `
CREATE TABLE IF NOT EXISTS forge_schema_migrations (
  version INTEGER PRIMARY KEY,
  name TEXT NOT NULL,
  applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
`,
		},
	}
}

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
	return r.Run(context.Background(), db)
}

func (r PostgresMigrationRunner) Run(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, migration := range r.migrations {
		if _, err := tx.ExecContext(ctx, migration.SQL); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO forge_schema_migrations(version, name)
VALUES ($1, $2)
ON CONFLICT (version) DO NOTHING
`, migration.Version, migration.Name); err != nil {
			return err
		}
	}
	return tx.Commit()
}
