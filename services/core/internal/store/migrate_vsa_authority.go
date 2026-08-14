package store

import (
	"database/sql"
	"fmt"
)

// ensureMemoryVSAProjectionAuthority upgrades legacy databases without
// assigning invented workspace authority. Existing rows retain empty scope
// and manifest identity and are therefore ignored by governed readers.
func ensureMemoryVSAProjectionAuthority(db *sql.DB) error {
	upgrades := map[string][]struct{ name, ddl string }{
		"memory_observations": {
			{name: "workspace_id", ddl: "TEXT NOT NULL DEFAULT ''"},
			{name: "lane_id", ddl: "TEXT NOT NULL DEFAULT ''"},
		},
		"memory_observation_links": {
			{name: "workspace_id", ddl: "TEXT NOT NULL DEFAULT ''"},
			{name: "lane_id", ddl: "TEXT NOT NULL DEFAULT ''"},
		},
		"memory_vsa_pointers": {
			{name: "workspace_id", ddl: "TEXT NOT NULL DEFAULT ''"},
			{name: "lane_id", ddl: "TEXT NOT NULL DEFAULT ''"},
			{name: "manifest_hash", ddl: "TEXT NOT NULL DEFAULT ''"},
			{name: "support_count", ddl: "INTEGER NOT NULL DEFAULT 0"},
			{name: "noise_count", ddl: "INTEGER NOT NULL DEFAULT 0"},
		},
		"memory_vsa_role_bindings": {
			{name: "workspace_id", ddl: "TEXT NOT NULL DEFAULT ''"},
			{name: "lane_id", ddl: "TEXT NOT NULL DEFAULT ''"},
			{name: "manifest_hash", ddl: "TEXT NOT NULL DEFAULT ''"},
		},
		"memory_vsa_associations": {
			{name: "workspace_id", ddl: "TEXT NOT NULL DEFAULT ''"},
			{name: "lane_id", ddl: "TEXT NOT NULL DEFAULT ''"},
			{name: "manifest_hash", ddl: "TEXT NOT NULL DEFAULT ''"},
		},
	}
	for table, additions := range upgrades {
		if err := ensureVSAProjectionColumns(db, table, additions); err != nil {
			return err
		}
	}
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_memory_vsa_pointer_manifest ON memory_vsa_pointers(workspace_id, lane_id, manifest_hash, observation_id)`,
		`CREATE INDEX IF NOT EXISTS idx_memory_vsa_binding_manifest ON memory_vsa_role_bindings(workspace_id, lane_id, manifest_hash, observation_id)`,
		`CREATE INDEX IF NOT EXISTS idx_memory_vsa_assoc_manifest ON memory_vsa_associations(workspace_id, lane_id, manifest_hash, from_observation_id, to_observation_id)`,
		`CREATE INDEX IF NOT EXISTS idx_memory_vsa_manifest_scope ON memory_vsa_projection_manifests(workspace_id, lane_id, created_at DESC)`,
	}
	for _, statement := range indexes {
		if _, err := db.Exec(statement); err != nil {
			return fmt.Errorf("create memory VSA projection index: %w", err)
		}
	}
	return nil
}

func ensureVSAProjectionColumns(db *sql.DB, table string, additions []struct{ name, ddl string }) error {
	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return err
	}
	existing := map[string]struct{}{}
	for rows.Next() {
		var cid, notNull, pk int
		var name, columnType string
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &pk); err != nil {
			_ = rows.Close()
			return err
		}
		existing[name] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return err
	}
	_ = rows.Close()
	for _, addition := range additions {
		if _, ok := existing[addition.name]; ok {
			continue
		}
		if _, err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, addition.name, addition.ddl)); err != nil {
			return err
		}
	}
	return nil
}
