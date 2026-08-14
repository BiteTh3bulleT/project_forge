package controllane

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"forge/projectforge/services/core/internal/memory/vsaprojection"
	forgestore "forge/projectforge/services/core/internal/store"
)

func TestMemoryAccelerationRebuildRequiresCallerTransaction(t *testing.T) {
	db := openAccelerationTestStore(t)
	_, err := NewSQLiteSemanticStore(db).RebuildMemoryAcceleration(context.Background(), MemoryAccelerationRebuildRequest{})
	if !errors.Is(err, ErrMemoryAccelerationTransactionRequired) {
		t.Fatalf("error = %v, want transaction required", err)
	}
}

func TestMemoryAccelerationRebuildScopesManifestAndFiltersLegacyRows(t *testing.T) {
	db := openAccelerationTestStore(t)
	seedAccelerationSources(t, db)

	commit := rebuildAcceleration(t, db, "", "syscall-1")
	if commit.Manifest.SourceCount != 2 || commit.Manifest.LinkCount != 1 {
		t.Fatalf("manifest counts = sources %d links %d", commit.Manifest.SourceCount, commit.Manifest.LinkCount)
	}
	if commit.PointerCount != 2 || commit.AssociationCount != 1 {
		t.Fatalf("projection counts = pointers %d associations %d", commit.PointerCount, commit.AssociationCount)
	}
	var head string
	if err := db.QueryRow(`SELECT manifest_hash FROM memory_vsa_projection_heads WHERE workspace_id='workspace-a' AND lane_id='lane-a'`).Scan(&head); err != nil {
		t.Fatal(err)
	}
	if head != commit.Manifest.ManifestHash {
		t.Fatalf("head = %q, want %q", head, commit.Manifest.ManifestHash)
	}
	var createdAt int64
	if err := db.QueryRow(`SELECT created_at FROM memory_vsa_projection_manifests WHERE manifest_hash=?`, head).Scan(&createdAt); err != nil {
		t.Fatal(err)
	}
	if createdAt != 100 {
		t.Fatalf("manifest created_at = %d, want sealed timestamp 100", createdAt)
	}
	var scoped, legacy int
	if err := db.QueryRow(`SELECT COUNT(*) FROM memory_vsa_pointers WHERE workspace_id='workspace-a' AND lane_id='lane-a' AND manifest_hash=?`, head).Scan(&scoped); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM memory_vsa_pointers WHERE workspace_id='' OR lane_id='' OR manifest_hash=''`).Scan(&legacy); err != nil {
		t.Fatal(err)
	}
	if scoped != 2 || legacy != 0 {
		t.Fatalf("pointer rows = scoped %d legacy %d", scoped, legacy)
	}
}

func TestMemoryAccelerationRebuildRejectsScopeWithoutGovernedSources(t *testing.T) {
	db := openAccelerationTestStore(t)
	if _, err := db.Exec(`INSERT INTO memory_observations(created_at,updated_at,observed_at,type) VALUES(1,1,1,'legacy')`); err != nil {
		t.Fatal(err)
	}
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	store := newSQLiteSemanticStore(tx)
	store.SetCommitMetadata(accelerationCommitMetadata("syscall-empty"))
	_, err = store.RebuildMemoryAcceleration(context.Background(), MemoryAccelerationRebuildRequest{
		Scope:     vsaprojection.Scope{WorkspaceID: "workspace-empty", LaneID: "lane-empty"},
		Algorithm: vsaprojection.DefaultAlgorithm(), ExpectedManifestHash: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", RequestedAtMs: 100,
	})
	_ = tx.Rollback()
	if !errors.Is(err, ErrMemoryAccelerationNoGovernedSources) {
		t.Fatalf("empty governed source error = %v", err)
	}
	var heads int
	if err := db.QueryRow(`SELECT COUNT(*) FROM memory_vsa_projection_heads`).Scan(&heads); err != nil || heads != 0 {
		t.Fatalf("heads after empty scope = %d err=%v", heads, err)
	}
}

func TestMemoryAccelerationRebuildRejectsManifestAndHeadDivergence(t *testing.T) {
	db := openAccelerationTestStore(t)
	seedAccelerationSources(t, db)

	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	store := newSQLiteSemanticStore(tx)
	store.SetCommitMetadata(accelerationCommitMetadata("syscall-bad-manifest"))
	_, err = store.RebuildMemoryAcceleration(context.Background(), MemoryAccelerationRebuildRequest{
		Scope: vsaprojection.Scope{WorkspaceID: "workspace-a", LaneID: "lane-a"}, Algorithm: vsaprojection.DefaultAlgorithm(),
		ExpectedManifestHash: "sha256:wrong", RequestedAtMs: 100,
	})
	_ = tx.Rollback()
	if err == nil || errors.Is(err, ErrMemoryAccelerationHeadConflict) {
		t.Fatalf("manifest divergence error = %v", err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM memory_vsa_projection_heads`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("head count after rejected identity = %d err=%v", count, err)
	}

	first := rebuildAcceleration(t, db, "", "syscall-1")
	if _, err := db.Exec(`UPDATE memory_observations SET summary='changed source' WHERE workspace_id='workspace-a' AND id=(SELECT MIN(id) FROM memory_observations WHERE workspace_id='workspace-a')`); err != nil {
		t.Fatal(err)
	}
	expected := expectedAccelerationManifest(t, db)
	tx, err = db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	store = newSQLiteSemanticStore(tx)
	store.SetCommitMetadata(accelerationCommitMetadata("syscall-head-conflict"))
	_, err = store.RebuildMemoryAcceleration(context.Background(), MemoryAccelerationRebuildRequest{
		Scope: vsaprojection.Scope{WorkspaceID: "workspace-a", LaneID: "lane-a"}, Algorithm: vsaprojection.DefaultAlgorithm(),
		ExpectedManifestHash: expected, ExpectedPriorManifestHash: "sha256:not-the-head", RequestedAtMs: 200,
	})
	_ = tx.Rollback()
	if !errors.Is(err, ErrMemoryAccelerationHeadConflict) {
		t.Fatalf("head divergence error = %v", err)
	}
	var head string
	if err := db.QueryRow(`SELECT manifest_hash FROM memory_vsa_projection_heads WHERE workspace_id='workspace-a' AND lane_id='lane-a'`).Scan(&head); err != nil {
		t.Fatal(err)
	}
	if head != first.Manifest.ManifestHash {
		t.Fatalf("head changed after conflict: %q", head)
	}
}

func TestMemoryAccelerationSwapFailureRollsBackOldProjection(t *testing.T) {
	db := openAccelerationTestStore(t)
	seedAccelerationSources(t, db)
	first := rebuildAcceleration(t, db, "", "syscall-1")
	if _, err := db.Exec(`UPDATE memory_observations SET summary='new source identity' WHERE workspace_id='workspace-a' AND id=(SELECT MIN(id) FROM memory_observations WHERE workspace_id='workspace-a')`); err != nil {
		t.Fatal(err)
	}
	expected := expectedAccelerationManifest(t, db)
	if _, err := db.Exec(`CREATE TRIGGER fail_vsa_pointer_swap BEFORE INSERT ON memory_vsa_pointers BEGIN SELECT RAISE(ABORT,'injected swap failure'); END`); err != nil {
		t.Fatal(err)
	}

	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	store := newSQLiteSemanticStore(tx)
	store.SetCommitMetadata(accelerationCommitMetadata("syscall-failed-swap"))
	_, err = store.RebuildMemoryAcceleration(context.Background(), MemoryAccelerationRebuildRequest{
		Scope: vsaprojection.Scope{WorkspaceID: "workspace-a", LaneID: "lane-a"}, Algorithm: vsaprojection.DefaultAlgorithm(),
		ExpectedManifestHash: expected, ExpectedPriorManifestHash: first.Manifest.ManifestHash, RequestedAtMs: 200,
	})
	_ = tx.Rollback()
	if err == nil {
		t.Fatal("swap unexpectedly succeeded")
	}
	var head string
	if err := db.QueryRow(`SELECT manifest_hash FROM memory_vsa_projection_heads WHERE workspace_id='workspace-a' AND lane_id='lane-a'`).Scan(&head); err != nil {
		t.Fatal(err)
	}
	if head != first.Manifest.ManifestHash {
		t.Fatalf("head after failed swap = %q", head)
	}
	var active, failedManifest int
	if err := db.QueryRow(`SELECT COUNT(*) FROM memory_vsa_pointers WHERE manifest_hash=?`, first.Manifest.ManifestHash).Scan(&active); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM memory_vsa_projection_manifests WHERE manifest_hash=?`, expected).Scan(&failedManifest); err != nil {
		t.Fatal(err)
	}
	if active != 2 || failedManifest != 0 {
		t.Fatalf("rollback state = active pointers %d failed manifests %d", active, failedManifest)
	}
}

func openAccelerationTestStore(t *testing.T) *sql.DB {
	t.Helper()
	st, err := forgestore.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st.DB
}

func seedAccelerationSources(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, statement := range []string{
		`INSERT INTO memory_observations(workspace_id,lane_id,created_at,updated_at,observed_at,type,raw_content,summary,source_path,entities_json,tags_json,related_files_json,lineage_json) VALUES('workspace-a','lane-a',1,1,1,'file','alpha body','alpha summary','/a.txt','["alpha"]','["tag"]','[]','[]')`,
		`INSERT INTO memory_observations(workspace_id,lane_id,created_at,updated_at,observed_at,type,raw_content,summary,source_path,entities_json,tags_json,related_files_json,lineage_json) VALUES('workspace-a','lane-a',2,2,2,'file','beta body','beta summary','/b.txt','["beta"]','[]','[]','[]')`,
		`INSERT INTO memory_observations(created_at,updated_at,observed_at,type,raw_content,summary,source_path) VALUES(3,3,3,'legacy','legacy body','legacy untrusted','/legacy.txt')`,
		`INSERT INTO memory_observation_links(workspace_id,lane_id,created_at,from_observation_id,to_observation_id,relation_type) SELECT 'workspace-a','lane-a',4,MIN(id),MAX(id),'supports' FROM memory_observations WHERE workspace_id='workspace-a'`,
		`INSERT INTO memory_usefulness_events(created_at,observation_id,signal,weight) SELECT 5,MIN(id),'useful',1 FROM memory_observations WHERE workspace_id='workspace-a'`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("seed: %v\n%s", err, statement)
		}
	}
}

func expectedAccelerationManifest(t *testing.T, db *sql.DB) string {
	t.Helper()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	store := newSQLiteSemanticStore(tx)
	scope := vsaprojection.Scope{WorkspaceID: "workspace-a", LaneID: "lane-a"}
	sources, err := store.loadMemoryAccelerationSources(context.Background(), scope)
	if err != nil {
		t.Fatal(err)
	}
	links, err := store.loadMemoryAccelerationLinks(context.Background(), scope)
	if err != nil {
		t.Fatal(err)
	}
	projection, err := vsaprojection.Build(scope, vsaprojection.DefaultAlgorithm(), sources, links)
	if err != nil {
		t.Fatal(err)
	}
	return projection.Manifest.ManifestHash
}

func rebuildAcceleration(t *testing.T, db *sql.DB, prior, syscallID string) MemoryAccelerationCommit {
	t.Helper()
	expected := expectedAccelerationManifest(t, db)
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	store := newSQLiteSemanticStore(tx)
	store.SetCommitMetadata(accelerationCommitMetadata(syscallID))
	commit, err := store.RebuildMemoryAcceleration(context.Background(), MemoryAccelerationRebuildRequest{
		Scope: vsaprojection.Scope{WorkspaceID: "workspace-a", LaneID: "lane-a"}, Algorithm: vsaprojection.DefaultAlgorithm(),
		ExpectedManifestHash: expected, ExpectedPriorManifestHash: prior, RequestedAtMs: 100,
	})
	if err != nil {
		_ = tx.Rollback()
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	return commit
}

func accelerationCommitMetadata(syscallID string) CommitMetadata {
	return CommitMetadata{SyscallID: syscallID, CorrelationID: "corr-" + syscallID, TraceID: "trace-" + syscallID, CommittedBy: "forge_k.kernel"}
}
