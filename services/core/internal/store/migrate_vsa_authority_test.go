package store

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestMemoryVSAProjectionAuthorityMigrationLeavesLegacyRowsUntrusted(t *testing.T) {
	db, err := sql.Open("sqlite", t.TempDir()+"/legacy.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	legacy := []string{
		`CREATE TABLE memory_observations(id INTEGER PRIMARY KEY, type TEXT NOT NULL)`,
		`CREATE TABLE memory_observation_links(id INTEGER PRIMARY KEY)`,
		`CREATE TABLE memory_vsa_pointers(id INTEGER PRIMARY KEY, observation_id INTEGER NOT NULL)`,
		`CREATE TABLE memory_vsa_role_bindings(id INTEGER PRIMARY KEY, observation_id INTEGER NOT NULL)`,
		`CREATE TABLE memory_vsa_associations(id INTEGER PRIMARY KEY, from_observation_id INTEGER NOT NULL, to_observation_id INTEGER NOT NULL)`,
		`CREATE TABLE memory_vsa_projection_manifests(manifest_hash TEXT PRIMARY KEY, workspace_id TEXT NOT NULL, lane_id TEXT NOT NULL, created_at INTEGER NOT NULL)`,
		`INSERT INTO memory_vsa_projection_manifests(manifest_hash,workspace_id,lane_id,created_at) VALUES('legacy-manifest','ws','lane',1)`,
		`INSERT INTO memory_observations(id,type) VALUES(1,'legacy')`,
		`INSERT INTO memory_vsa_pointers(id,observation_id) VALUES(1,1)`,
	}
	for _, statement := range legacy {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("legacy schema: %v", err)
		}
	}
	if err := ensureMemoryVSAProjectionAuthority(db); err != nil {
		t.Fatalf("upgrade: %v", err)
	}
	if err := ensureMemoryVSAProjectionAuthority(db); err != nil {
		t.Fatalf("idempotent upgrade: %v", err)
	}
	var workspaceID, laneID, manifestHash string
	if err := db.QueryRow(`SELECT workspace_id,lane_id,manifest_hash FROM memory_vsa_pointers WHERE id=1`).Scan(&workspaceID, &laneID, &manifestHash); err != nil {
		t.Fatal(err)
	}
	if workspaceID != "" || laneID != "" || manifestHash != "" {
		t.Fatalf("legacy projection gained invented authority: %q/%q %q", workspaceID, laneID, manifestHash)
	}
	var sourceKind string
	if err := db.QueryRow(`SELECT source_kind FROM memory_vsa_projection_manifests WHERE manifest_hash='legacy-manifest'`).Scan(&sourceKind); err != nil {
		t.Fatal(err)
	}
	if sourceKind != "" {
		t.Fatalf("legacy manifest gained governed source kind %q", sourceKind)
	}
}

func TestMemoryVSAManifestSchemaIsImmutable(t *testing.T) {
	st, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	_, err = st.DB.Exec(`INSERT INTO memory_vsa_projection_manifests(
manifest_hash,workspace_id,lane_id,source_set_hash,link_set_hash,algorithm_name,algorithm_version,
dimensions,seed,source_count,link_count,manifest_json,syscall_id,correlation_id,trace_id,created_at
) VALUES('sha256:m','ws','lane','sha256:s','sha256:l','forge.vsa.observation_projection','1',128,17,0,0,'{}','sys','corr','trace',1)`)
	if err != nil {
		t.Fatalf("insert manifest: %v", err)
	}
	if _, err := st.DB.Exec(`UPDATE memory_vsa_projection_manifests SET source_count=1 WHERE manifest_hash='sha256:m'`); err == nil {
		t.Fatal("manifest update unexpectedly succeeded")
	}
	if _, err := st.DB.Exec(`DELETE FROM memory_vsa_projection_manifests WHERE manifest_hash='sha256:m'`); err == nil {
		t.Fatal("manifest delete unexpectedly succeeded")
	}
}
