package store

import "testing"

func TestMigrateCreatesModelRuntimeTablesAndIndexes(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	st, err := Open(dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}

	tables := []string{
		"model_manifests",
		"model_registry_status",
		"model_runtime_loads",
	}
	for _, table := range tables {
		requireSchemaObject(t, st, table, "table")
	}

	indexes := []string{
		"idx_model_manifests_backend",
		"idx_model_manifests_format",
		"idx_model_registry_status_status",
		"idx_model_runtime_loads_model",
		"idx_model_runtime_loads_status",
	}
	for _, index := range indexes {
		requireSchemaObject(t, st, index, "index")
	}
	_ = st.Close()

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
