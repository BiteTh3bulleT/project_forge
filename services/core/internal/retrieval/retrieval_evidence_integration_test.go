package retrieval

import (
	"context"
	"database/sql"
	"testing"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/embeddings"
	"forge/projectforge/services/core/internal/memory"
	"forge/projectforge/services/core/internal/search"
	"forge/projectforge/services/core/internal/store"
)

func TestInternalJobRetryReturnsImmutableEvidenceBeforeSearch(t *testing.T) {
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	svc := New(st.DB, search.New(st.DB), embeddings.New(st.DB), memory.New(st.DB))
	installTestForgeKAuthority(t, svc, st.DB)
	sourceID := seedSource(t, st.DB, "/repo")
	chunkID, _, _ := seedChunk(t, st.DB, sourceID, "retry.go", "/repo/retry.go", "stable job retry evidence")
	seedRetrievalJob(t, st.DB, "job-retry")
	jobID := "job-retry"
	req := RunRequest{
		Query: "stable job retry", Mode: ModeKeyword, Limit: 4, SelectForPacket: 1, JobID: &jobID,
		Actor: domain.ActorIdentity{ID: "forge.jobs", Kind: "service"}, Source: domain.SourceInternal,
		Scope:         domain.ForgeScope{WorkspaceID: "/repo", LaneID: "control.semantic"},
		Provenance:    domain.Provenance{Actor: "forge.jobs", ActorType: "system", Source: "job.packet.prep"},
		CorrelationID: "job-job-retry-retrieval", TraceID: "job-job-retry-retrieval",
		RequestID: "retrieval-evidence-job-job-retry", IdempotencyKey: "retrieval-evidence-job-job-retry",
		RequestedAt: 1760000000000,
	}
	first, err := svc.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("first run: %v", err)
	}
	if len(first.Results) == 0 {
		t.Fatal("first run did not commit results")
	}
	originalFileID := *first.Results[0].FileID
	if _, err := st.DB.Exec(`DELETE FROM chunks WHERE id=?`, chunkID); err != nil {
		t.Fatalf("delete source chunk with immutable evidence: %v", err)
	}
	if _, err := st.DB.Exec(`DELETE FROM files WHERE id=?`, originalFileID); err != nil {
		t.Fatalf("delete source file with immutable evidence: %v", err)
	}
	if _, err := st.DB.Exec(`DELETE FROM jobs WHERE id=?`, jobID); err != nil {
		t.Fatalf("delete source job with immutable evidence: %v", err)
	}
	second, err := svc.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("immutable retry: %v", err)
	}
	if second.ID != first.ID || len(second.Results) != len(first.Results) {
		t.Fatalf("retry returned a different evidence bundle: first=%#v second=%#v", first, second)
	}
	if second.Results[0].ChunkID == nil || *second.Results[0].ChunkID != chunkID ||
		second.Results[0].FileID == nil || *second.Results[0].FileID != originalFileID {
		t.Fatalf("detached evidence lost original source ids: %#v", second.Results[0])
	}
	var liveChunkID, liveFileID sql.NullInt64
	var liveJobID sql.NullString
	var originalChunkID, originalStoredFileID int64
	var originalJobID string
	if err := st.DB.QueryRow(`
SELECT rr.chunk_id,rr.file_id,r.job_id,rr.original_chunk_id,rr.original_file_id,r.original_job_id
FROM retrieval_results rr JOIN retrieval_runs r ON r.id=rr.retrieval_run_id
WHERE r.id=? AND rr.rank_index=0`, first.ID).Scan(
		&liveChunkID, &liveFileID, &liveJobID, &originalChunkID, &originalStoredFileID, &originalJobID,
	); err != nil {
		t.Fatal(err)
	}
	if liveChunkID.Valid || liveFileID.Valid || liveJobID.Valid || originalChunkID != chunkID ||
		originalStoredFileID != originalFileID || originalJobID != jobID {
		t.Fatalf("unexpected live/original identity split: live=(%v,%v,%v) original=(%d,%d,%q)", liveChunkID, liveFileID, liveJobID, originalChunkID, originalStoredFileID, originalJobID)
	}
	if _, err := st.DB.Exec(`DELETE FROM sources WHERE id=?`, sourceID); err != nil {
		t.Fatalf("delete source root with immutable evidence: %v", err)
	}
	detached, err := svc.GetRun(context.Background(), first.ID)
	if err != nil || detached.Results[0].AbsPath != "/repo/retry.go" || detached.Results[0].Snippet == "" {
		t.Fatalf("detached evidence content was not preserved: run=%#v err=%v", detached, err)
	}
	for table, want := range map[string]int{"retrieval_runs": 1, "semantic_idempotency_keys": 1, "forge_k_audit_outbox": 1} {
		var got int
		if err := st.DB.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&got); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if got != want {
			t.Fatalf("%s count=%d want=%d", table, got, want)
		}
	}
}

func TestExistingEvidenceDoesNotBypassUserAuthorizationOrScope(t *testing.T) {
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	svc := New(st.DB, search.New(st.DB), embeddings.New(st.DB), memory.New(st.DB))
	installTestForgeKAuthority(t, svc, st.DB)
	sourceID := seedSource(t, st.DB, "/repo")
	seedChunk(t, st.DB, sourceID, "auth.go", "/repo/auth.go", "authorization proof")
	seedRetrievalJob(t, st.DB, "job-auth")
	jobID := "job-auth"
	internal := RunRequest{
		Query: "authorization", Mode: ModeKeyword, Limit: 2, SelectForPacket: 1, JobID: &jobID,
		Actor: domain.ActorIdentity{ID: "forge.jobs", Kind: "service"}, Source: domain.SourceInternal,
		Scope:         domain.ForgeScope{WorkspaceID: "/repo", LaneID: "control.semantic"},
		Provenance:    domain.Provenance{Actor: "forge.jobs", ActorType: "system", Source: "job.packet.prep"},
		CorrelationID: "corr-auth", TraceID: "trace-auth", RequestID: "retrieval-auth-existing",
		IdempotencyKey: "retrieval-auth-existing", RequestedAt: 1760000000000,
	}
	if _, err := svc.Run(context.Background(), internal); err != nil {
		t.Fatalf("seed evidence: %v", err)
	}
	wrongScope := internal
	wrongScope.Scope.SelectedPaths = []string{"/other"}
	if _, err := svc.Run(context.Background(), wrongScope); err == nil {
		t.Fatal("mismatched selected-path scope reused evidence")
	}
	user := internal
	user.Actor = domain.ActorIdentity{ID: "operator", Kind: "user"}
	user.Source = domain.SourceUser
	user.Provenance = domain.Provenance{Actor: "operator", ActorType: "user", Source: "local_loopback"}
	if _, err := svc.Run(context.Background(), user); err == nil {
		t.Fatal("user request bypassed Kernel origin authorization through existing evidence")
	}
}

func TestRetrievalEvidenceCommitRollsBackAsOneTransaction(t *testing.T) {
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	svc := New(st.DB, search.New(st.DB), embeddings.New(st.DB), memory.New(st.DB))
	installTestForgeKAuthority(t, svc, st.DB)
	sourceID := seedSource(t, st.DB, "/repo")
	seedChunk(t, st.DB, sourceID, "rollback.go", "/repo/rollback.go", "atomic rollback evidence")
	if _, err := st.DB.Exec(`
CREATE TRIGGER reject_retrieval_selection
BEFORE INSERT ON retrieval_result_selection BEGIN
  SELECT RAISE(ABORT, 'injected retrieval selection failure');
END`); err != nil {
		t.Fatal(err)
	}
	req := RunRequest{
		Query: "atomic rollback", Mode: ModeKeyword, Limit: 2, SelectForPacket: 1,
		Actor: domain.ActorIdentity{ID: "forge.core", Kind: "service"}, Source: domain.SourceInternal,
		Scope:         domain.ForgeScope{WorkspaceID: "/repo", LaneID: "control.semantic"},
		Provenance:    domain.Provenance{Actor: "forge.core", ActorType: "system", Source: "test.rollback"},
		CorrelationID: "corr-rollback", TraceID: "trace-rollback", RequestID: "retrieval-rollback",
		IdempotencyKey: "retrieval-rollback", RequestedAt: 1760000000000,
	}
	if _, err := svc.Run(context.Background(), req); err == nil {
		t.Fatal("injected selection failure did not reject retrieval commit")
	}
	for _, table := range []string{"retrieval_runs", "retrieval_results", "retrieval_result_selection", "semantic_idempotency_keys", "forge_k_audit_outbox", "journal_events"} {
		var count int
		if err := st.DB.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if count != 0 {
			t.Fatalf("atomic rollback left %d rows in %s", count, table)
		}
	}
}

func seedRetrievalJob(t *testing.T, db *sql.DB, id string) {
	t.Helper()
	if _, err := db.Exec(`
INSERT INTO jobs(
  id,created_at,updated_at,title,requested_action,target_adapter,initiating_source,
  execution_boundary,risk_class,status,approval_status,write_intent,metadata_json
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		id, int64(1), int64(1), "retrieval", "retrieve", "forge.jobs", "internal_cell",
		"read", "low", "running", "not_required", 0, `{}`,
	); err != nil {
		t.Fatalf("seed retrieval job: %v", err)
	}
}
