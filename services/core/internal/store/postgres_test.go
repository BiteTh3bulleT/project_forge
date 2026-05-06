package store

import (
	"errors"
	"strings"
	"testing"
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

func TestPostgresMigrationRegistryHasVersionTableOnly(t *testing.T) {
	migrations := PostgresMigrations()
	if len(migrations) != 1 {
		t.Fatalf("expected one scaffold migration, got %d", len(migrations))
	}
	if migrations[0].Version != 1 {
		t.Fatalf("expected migration version 1, got %d", migrations[0].Version)
	}
	if !strings.Contains(migrations[0].SQL, "forge_schema_migrations") {
		t.Fatalf("expected schema migration table SQL, got %q", migrations[0].SQL)
	}
	if strings.Contains(migrations[0].SQL, "memory_notes") || strings.Contains(migrations[0].SQL, "retrieval_runs") {
		t.Fatalf("postgres scaffold must not migrate live memory/retrieval tables yet")
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
