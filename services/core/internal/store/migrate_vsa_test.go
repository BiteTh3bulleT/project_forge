package store

import "testing"

func TestMigrateCreatesVSATablesAndIndexes(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	st, err := Open(dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}

	tables := []string{
		"memory_vsa_pointers",
		"memory_vsa_role_bindings",
		"memory_vsa_associations",
		"retrieval_result_vsa_signals",
		"memory_vsa_reindex_runs",
		"memory_vsa_reindex_items",
	}
	for _, table := range tables {
		requireSchemaObject(t, st, table, "table")
	}

	indexes := []string{
		"idx_memory_vsa_pointers_obs",
		"idx_memory_vsa_pointers_fp",
		"idx_memory_vsa_bindings_obs",
		"idx_memory_vsa_assoc_from",
		"idx_memory_vsa_assoc_to",
		"idx_result_vsa_signals_run",
		"idx_result_vsa_signals_obs",
		"idx_result_vsa_signals_mode",
		"idx_memory_vsa_reindex_runs_created",
		"idx_memory_vsa_reindex_runs_dossier",
		"idx_memory_vsa_reindex_items_run",
		"idx_memory_vsa_reindex_items_obs",
	}
	for _, index := range indexes {
		requireSchemaObject(t, st, index, "index")
	}
	_ = st.Close()

	// Re-open to assert migration idempotency over an already-migrated DB.
	st2, err := Open(dataDir)
	if err != nil {
		t.Fatalf("re-open store: %v", err)
	}
	defer st2.Close()
	for _, table := range tables {
		requireSchemaObject(t, st2, table, "table")
	}
	for _, index := range indexes {
		requireSchemaObject(t, st2, index, "index")
	}
}

func requireSchemaObject(t *testing.T, st *Store, name, kind string) {
	t.Helper()
	var found string
	err := st.DB.QueryRow(`SELECT name FROM sqlite_master WHERE type = ? AND name = ?`, kind, name).Scan(&found)
	if err != nil {
		t.Fatalf("expected %s %q to exist: %v", kind, name, err)
	}
}
