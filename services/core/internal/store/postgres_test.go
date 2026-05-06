package store

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestNewPostgresConnectorValidatesDSN(t *testing.T) {
	if _, err := NewPostgresConnector(""); !errors.Is(err, ErrPostgresDSNRequired) {
		t.Fatalf("expected ErrPostgresDSNRequired, got %v", err)
	}
	if _, err := NewPostgresConnector("sqlite://not-postgres"); !errors.Is(err, ErrInvalidPostgresDSN) {
		t.Fatalf("expected ErrInvalidPostgresDSN, got %v", err)
	}

	conn, err := NewPostgresConnector("postgres://forge:forge@localhost:5432/forge?sslmode=disable")
	if err != nil {
		t.Fatalf("NewPostgresConnector failed: %v", err)
	}
	if conn.DSN() != "postgres://forge:forge@localhost:5432/forge?sslmode=disable" {
		t.Fatalf("unexpected DSN: %q", conn.DSN())
	}
}

func TestPostgresMigrationRegistryHasFoundationSchemaOnly(t *testing.T) {
	migrations := PostgresMigrations()
	if got, want := len(migrations), 4; got != want {
		t.Fatalf("expected %d foundation migrations, got %d", want, got)
	}
	names := []string{migrations[0].Name, migrations[1].Name, migrations[2].Name, migrations[3].Name}
	wantNames := []string{"create_schema_migration_table", "storage_metadata_foundation", "shadow_diagnostics_schema", "shadow_diagnostic_persistence_metadata"}
	if !reflect.DeepEqual(names, wantNames) {
		t.Fatalf("unexpected migration names: got %v want %v", names, wantNames)
	}

	combined := strings.ToLower(migrations[0].SQL + migrations[1].SQL + migrations[2].SQL + migrations[3].SQL)
	required := []string{
		"forge_schema_migrations",
		"storage_backend_metadata",
		"storage_migration_audit",
		"shadow_diagnostic_reports",
		"shadow_diagnostic_report_events",
		"shadow_diagnostic_redactions",
		"expires_at",
		"no_effect_verified",
		"schema_version",
		"jsonb",
		"timestamptz",
		"create index if not exists",
	}
	for _, needle := range required {
		if !strings.Contains(combined, needle) {
			t.Fatalf("expected migration SQL to include %q", needle)
		}
	}
	forbidden := []string{"memory_notes", "retrieval_runs", "retrieval_results", "embedding_records", "chunks_fts"}
	for _, needle := range forbidden {
		if strings.Contains(combined, needle) {
			t.Fatalf("postgres foundation must not migrate live memory/retrieval table %q", needle)
		}
	}
}

func TestPostgresMigrationRegistryMatchesSQLFiles(t *testing.T) {
	files := map[int]string{
		1: "0001_create_schema_migration_table.sql",
		2: "0002_storage_metadata_foundation.sql",
		3: "0003_shadow_diagnostics_schema.sql",
		4: "0004_shadow_diagnostic_persistence_metadata.sql",
	}
	for _, migration := range PostgresMigrations() {
		name, ok := files[migration.Version]
		if !ok {
			t.Fatalf("missing migration file expectation for version %d", migration.Version)
		}
		raw, err := os.ReadFile(filepath.Join("..", "..", "migrations", "postgres", name))
		if err != nil {
			t.Fatalf("read postgres migration file %s: %v", name, err)
		}
		if normalizeSQLForTest(string(raw)) != normalizeSQLForTest(migration.SQL) {
			t.Fatalf("registry SQL for migration %d does not match %s", migration.Version, name)
		}
	}
}

func TestPostgresMigrationRunnerDoesNotRunForSQLite(t *testing.T) {
	st, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open sqlite store failed: %v", err)
	}
	defer st.Close()

	runner := NewPostgresMigrationRunner(PostgresMigrations())
	if err := runner.RunForBackend(st.DB, "sqlite"); err != nil {
		t.Fatalf("sqlite backend should skip postgres migrations without error: %v", err)
	}

	var name string
	err = st.DB.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'forge_schema_migrations'`).Scan(&name)
	if err == nil {
		t.Fatalf("postgres migration table should not be created in sqlite default store")
	}
}

func TestPostgresMigrationRunnerEmptyMigrationSetWorks(t *testing.T) {
	fake := newRecordingPostgresExecutor(nil)
	if err := NewPostgresMigrationRunner(nil).run(context.Background(), fake); err != nil {
		t.Fatalf("empty migration set failed: %v", err)
	}
	if len(fake.calls) != 0 {
		t.Fatalf("empty migration set should not call executor, got %v", fake.calls)
	}
}

func TestPostgresMigrationRunnerRunsInDeterministicOrder(t *testing.T) {
	migrations := []PostgresMigration{
		{Version: 2, Name: "second", SQL: "SELECT 2;"},
		{Version: 1, Name: "first", SQL: "SELECT 1;"},
	}
	fake := newRecordingPostgresExecutor(nil)
	if err := NewPostgresMigrationRunner(migrations).run(context.Background(), fake); err != nil {
		t.Fatalf("migration run failed: %v", err)
	}
	want := []string{"ensure", "applied", "execute:1", "record:1", "execute:2", "record:2"}
	if !reflect.DeepEqual(fake.calls, want) {
		t.Fatalf("unexpected call order: got %v want %v", fake.calls, want)
	}
}

func TestPostgresMigrationRunnerSkipsAlreadyAppliedMigration(t *testing.T) {
	migrations := []PostgresMigration{
		{Version: 1, Name: "first", SQL: "SELECT 1;"},
		{Version: 2, Name: "second", SQL: "SELECT 2;"},
	}
	fake := newRecordingPostgresExecutor(map[int]bool{1: true})
	if err := NewPostgresMigrationRunner(migrations).run(context.Background(), fake); err != nil {
		t.Fatalf("migration run failed: %v", err)
	}
	want := []string{"ensure", "applied", "execute:2", "record:2"}
	if !reflect.DeepEqual(fake.calls, want) {
		t.Fatalf("unexpected call order: got %v want %v", fake.calls, want)
	}
}

func TestPostgresMigrationRunnerReportsFailedMigration(t *testing.T) {
	fake := newRecordingPostgresExecutor(nil)
	fake.failExecuteVersion = 2
	err := NewPostgresMigrationRunner([]PostgresMigration{
		{Version: 1, Name: "first", SQL: "SELECT 1;"},
		{Version: 2, Name: "second", SQL: "SELECT 2;"},
	}).run(context.Background(), fake)
	if err == nil || !strings.Contains(err.Error(), "0002_second") {
		t.Fatalf("expected failed migration error with version/name, got %v", err)
	}
	if got, want := fake.calls, []string{"ensure", "applied", "execute:1", "record:1", "execute:2"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected calls after failure: got %v want %v", got, want)
	}
}

func TestPostgresMigrationRunnerRejectsDuplicateVersions(t *testing.T) {
	err := NewPostgresMigrationRunner([]PostgresMigration{
		{Version: 1, Name: "first", SQL: "SELECT 1;"},
		{Version: 1, Name: "duplicate", SQL: "SELECT 2;"},
	}).run(context.Background(), newRecordingPostgresExecutor(nil))
	if err == nil || !strings.Contains(err.Error(), "duplicate postgres migration version") {
		t.Fatalf("expected duplicate version error, got %v", err)
	}
}

func TestPostgresMigrationRunnerRecordsMigrationVersions(t *testing.T) {
	fake := newRecordingPostgresExecutor(nil)
	migrations := []PostgresMigration{
		{Version: 1, Name: "first", SQL: "SELECT 1;"},
		{Version: 2, Name: "second", SQL: "SELECT 2;"},
	}
	if err := NewPostgresMigrationRunner(migrations).run(context.Background(), fake); err != nil {
		t.Fatalf("migration run failed: %v", err)
	}
	if got, want := fake.recorded, []int{1, 2}; !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected recorded migrations: got %v want %v", got, want)
	}
}

func TestPostgresMigrationRunnerIntegrationOptional(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("FORGE_POSTGRES_TEST_DSN"))
	if dsn == "" {
		t.Skip("FORGE_POSTGRES_TEST_DSN not set")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open postgres test database: %v", err)
	}
	defer db.Close()
	ctx := context.Background()
	runner := NewPostgresMigrationRunner(PostgresMigrations())
	if err := runner.Run(ctx, db); err != nil {
		t.Fatalf("first migration run failed: %v", err)
	}
	if err := runner.Run(ctx, db); err != nil {
		t.Fatalf("idempotent migration rerun failed: %v", err)
	}
	requiredTables := []string{
		"forge_schema_migrations",
		"storage_backend_metadata",
		"storage_migration_audit",
		"shadow_diagnostic_reports",
		"shadow_diagnostic_report_events",
		"shadow_diagnostic_redactions",
	}
	for _, table := range requiredTables {
		var exists bool
		err := db.QueryRowContext(ctx, `SELECT EXISTS (
SELECT 1 FROM information_schema.tables
WHERE table_schema = 'public' AND table_name = $1
)`, table).Scan(&exists)
		if err != nil {
			t.Fatalf("check table %s: %v", table, err)
		}
		if !exists {
			t.Fatalf("expected postgres table %s to exist", table)
		}
	}
	var migrationCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM forge_schema_migrations`).Scan(&migrationCount); err != nil {
		t.Fatalf("count schema migrations: %v", err)
	}
	if migrationCount < len(PostgresMigrations()) {
		t.Fatalf("expected at least %d recorded migrations, got %d", len(PostgresMigrations()), migrationCount)
	}
}

type recordingPostgresExecutor struct {
	applied            map[int]bool
	failExecuteVersion int
	calls              []string
	recorded           []int
}

func newRecordingPostgresExecutor(applied map[int]bool) *recordingPostgresExecutor {
	copied := map[int]bool{}
	for version, ok := range applied {
		copied[version] = ok
	}
	return &recordingPostgresExecutor{applied: copied}
}

func (e *recordingPostgresExecutor) EnsureVersionTable(context.Context) error {
	e.calls = append(e.calls, "ensure")
	return nil
}

func (e *recordingPostgresExecutor) AppliedVersions(context.Context) (map[int]bool, error) {
	e.calls = append(e.calls, "applied")
	copied := map[int]bool{}
	for version, ok := range e.applied {
		copied[version] = ok
	}
	return copied, nil
}

func (e *recordingPostgresExecutor) ExecuteMigration(_ context.Context, migration PostgresMigration) error {
	e.calls = append(e.calls, "execute:"+itoaForTest(migration.Version))
	if migration.Version == e.failExecuteVersion {
		return errors.New("forced migration failure")
	}
	return nil
}

func (e *recordingPostgresExecutor) RecordMigration(_ context.Context, migration PostgresMigration) error {
	e.calls = append(e.calls, "record:"+itoaForTest(migration.Version))
	e.recorded = append(e.recorded, migration.Version)
	return nil
}

func normalizeSQLForTest(raw string) string {
	return strings.Join(strings.Fields(raw), " ")
}

func itoaForTest(version int) string {
	return strconv.Itoa(version)
}
