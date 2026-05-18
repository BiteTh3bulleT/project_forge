package store

import "testing"

func TestMigrateCreatesSQLiteSchemaVersionTable(t *testing.T) {
	dataDir := t.TempDir()
	st, err := Open(dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	var (
		rowCount  int
		version   int
		updatedAt int64
	)
	if err := st.DB.QueryRow(`SELECT COUNT(*), COALESCE(MAX(version), 0), COALESCE(MAX(updated_at), 0) FROM schema_version`).Scan(&rowCount, &version, &updatedAt); err != nil {
		t.Fatalf("query schema_version: %v", err)
	}
	if rowCount != 1 || version != 1 || updatedAt <= 0 {
		t.Fatalf("unexpected schema_version row count=%d version=%d updatedAt=%d", rowCount, version, updatedAt)
	}

	if _, err := st.DB.Exec(`UPDATE schema_version SET version = 2 WHERE id = 1`); err != nil {
		t.Fatalf("bump schema_version: %v", err)
	}
	if err := st.Close(); err != nil {
		t.Fatalf("close store: %v", err)
	}

	reopened, err := Open(dataDir)
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	defer reopened.Close()

	if err := reopened.DB.QueryRow(`SELECT COUNT(*), COALESCE(MAX(version), 0) FROM schema_version`).Scan(&rowCount, &version); err != nil {
		t.Fatalf("query reopened schema_version: %v", err)
	}
	if rowCount != 1 || version != 2 {
		t.Fatalf("schema_version migration should be idempotent and non-downgrading, count=%d version=%d", rowCount, version)
	}
}
