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
		"forge_k_memory_vsa_pointers",
		"forge_k_memory_vsa_role_bindings",
		"forge_k_memory_vsa_associations",
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
		"idx_forge_k_memory_vsa_pointer_head",
		"idx_forge_k_memory_vsa_binding_head",
		"idx_forge_k_memory_vsa_assoc_head",
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
	for table, columns := range map[string][]string{
		"memory_vsa_projection_manifests": {"source_kind"},
		"retrieval_result_vsa_signals":    {"memory_evidence_row_id", "memory_evidence_id"},
	} {
		for _, column := range columns {
			requireTableColumn(t, st, table, column)
		}
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

func requireTableColumn(t *testing.T, st *Store, table, column string) {
	t.Helper()
	rows, err := st.DB.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var cid, notNull, pk int
		var name, columnType string
		var defaultValue any
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &pk); err != nil {
			t.Fatal(err)
		}
		if name == column {
			return
		}
	}
	t.Fatalf("expected column %s.%s", table, column)
}

func requireSchemaObject(t *testing.T, st *Store, name, kind string) {
	t.Helper()
	var found string
	err := st.DB.QueryRow(`SELECT name FROM sqlite_master WHERE type = ? AND name = ?`, kind, name).Scan(&found)
	if err != nil {
		t.Fatalf("expected %s %q to exist: %v", kind, name, err)
	}
}
