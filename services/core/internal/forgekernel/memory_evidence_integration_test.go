package forgekernel_test

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"

	"forge/projectforge/services/core/internal/aios/controllane"
	"forge/projectforge/services/core/internal/aios/domain"
	. "forge/projectforge/services/core/internal/forgekernel"
	"forge/projectforge/services/core/internal/memory"
	"forge/projectforge/services/core/internal/memory/vsaprojection"
)

func TestForgeKMaterializesAdmittedEvidenceForAuthenticatedUserAndRevisesAppendOnly(t *testing.T) {
	ctx := context.Background()
	selection, st, _ := newLiveSQLiteAuthority(t)
	scope := domain.ForgeScope{WorkspaceID: "ws-memory-evidence", LaneID: "memory.semantic", SelectedPaths: []string{"artifact:case-1"}}
	admitMemoryEvidenceSource(t, ctx, selection, "admit-memory-v1", "case-memory", "exhibit-memory-v1", "ruling-memory-v1", "a", scope)

	materialize := memoryEvidenceRequest(domain.ActionMaterializeAdmittedEvidence, "materialize-memory-v1", scope, map[string]any{
		"exhibitId": "exhibit-memory-v1", "rulingId": "ruling-memory-v1",
	})
	result, err := selection.Processor.Process(ctx, materialize)
	if err != nil || !result.Success {
		t.Fatalf("authenticated user materialization: err=%v result=%#v", err, result)
	}
	read := controllane.NewSQLiteSemanticStore(st.DB)
	first, ok := read.FindMemoryEvidence(materialize.ID+":memory_evidence", scope)
	if !ok || first.RowID <= 0 || first.Revision != 1 || first.RootEvidenceID != first.EvidenceID ||
		first.CourtExhibitID != "exhibit-memory-v1" || first.ContentSummary != "Court summary exhibit-memory-v1" ||
		first.ContentHash != "sha256:"+strings.Repeat("a", 64) || first.CommittedBy != AuthorityOwnerForgeK {
		t.Fatalf("unexpected immutable evidence: ok=%v evidence=%#v", ok, first)
	}
	if _, replayErr := selection.Processor.Process(ctx, materialize); replayErr != nil {
		t.Fatalf("idempotent replay failed: %v", replayErr)
	}
	assertTableCountWhere(t, st.DB, "forge_k_memory_evidence", "syscall_id", materialize.ID, 1)
	firstProjection, err := read.PlanMemoryAcceleration(ctx, vsaprojection.Scope{WorkspaceID: scope.WorkspaceID, LaneID: scope.LaneID}, vsaprojection.DefaultAlgorithm())
	if err != nil || firstProjection.Manifest.SourceCount != 1 || len(firstProjection.Pointers) != 1 || firstProjection.Pointers[0].EvidenceID != first.EvidenceID {
		t.Fatalf("initial governed VSA source set: projection=%#v err=%v", firstProjection, err)
	}
	firstRebuild := memoryAccelerationRequest("rebuild-memory-v1", scope, firstProjection.Manifest.ManifestHash, "")
	if rebuildResult, rebuildErr := selection.Processor.Process(ctx, firstRebuild); rebuildErr != nil || !rebuildResult.Success {
		t.Fatalf("initial governed VSA rebuild: err=%v result=%#v", rebuildErr, rebuildResult)
	}

	admitMemoryEvidenceSource(t, ctx, selection, "admit-memory-v2", "case-memory", "exhibit-memory-v2", "ruling-memory-v2", "b", scope)
	revise := memoryEvidenceRequest(domain.ActionReviseMemoryEvidence, "revise-memory-v2", scope, map[string]any{
		"exhibitId": "exhibit-memory-v2", "rulingId": "ruling-memory-v2", "priorEvidenceId": first.EvidenceID,
	})
	revisedResult, err := selection.Processor.Process(ctx, revise)
	if err != nil || !revisedResult.Success {
		t.Fatalf("revise memory evidence: err=%v result=%#v", err, revisedResult)
	}
	second, ok := read.FindMemoryEvidence(revise.ID+":memory_evidence", scope)
	if !ok || second.Revision != 2 || second.RootEvidenceID != first.EvidenceID || second.ContentHash != "sha256:"+strings.Repeat("b", 64) {
		t.Fatalf("unexpected replacement evidence: ok=%v evidence=%#v", ok, second)
	}
	firstAgain, ok := read.FindMemoryEvidence(first.EvidenceID, scope)
	if !ok || !reflect.DeepEqual(firstAgain, first) || !read.HasMemoryEvidenceSupersession(first.EvidenceID) {
		t.Fatalf("original evidence was rewritten or lineage missing: first=%#v again=%#v", first, firstAgain)
	}
	revisedProjection, err := read.PlanMemoryAcceleration(ctx, vsaprojection.Scope{WorkspaceID: scope.WorkspaceID, LaneID: scope.LaneID}, vsaprojection.DefaultAlgorithm())
	if err != nil || revisedProjection.Manifest.SourceCount != 1 || len(revisedProjection.Pointers) != 1 || revisedProjection.Pointers[0].EvidenceID != second.EvidenceID {
		t.Fatalf("revised governed VSA leaf set: projection=%#v err=%v", revisedProjection, err)
	}
	if _, err := st.DB.Exec(`CREATE TRIGGER fail_memory_vsa_swap_e2e BEFORE INSERT ON forge_k_memory_vsa_pointers BEGIN SELECT RAISE(ABORT,'injected governed swap failure'); END`); err != nil {
		t.Fatal(err)
	}
	failedRebuild := memoryAccelerationRequest("rebuild-memory-v2-failure", scope, revisedProjection.Manifest.ManifestHash, firstProjection.Manifest.ManifestHash)
	if failedResult, _ := selection.Processor.Process(ctx, failedRebuild); failedResult.Success {
		t.Fatalf("injected governed swap failure succeeded: %#v", failedResult)
	}
	var activeHead, activeEvidenceID string
	if err := st.DB.QueryRow(`SELECT manifest_hash FROM memory_vsa_projection_heads WHERE workspace_id=? AND lane_id=?`, scope.WorkspaceID, scope.LaneID).Scan(&activeHead); err != nil {
		t.Fatal(err)
	}
	if err := st.DB.QueryRow(`SELECT memory_evidence_id FROM forge_k_memory_vsa_pointers WHERE workspace_id=? AND lane_id=?`, scope.WorkspaceID, scope.LaneID).Scan(&activeEvidenceID); err != nil {
		t.Fatal(err)
	}
	if activeHead != firstProjection.Manifest.ManifestHash || activeEvidenceID != first.EvidenceID {
		t.Fatalf("failed swap changed active projection: head=%s evidence=%s", activeHead, activeEvidenceID)
	}
	assertTableCountWhere(t, st.DB, "forge_k_memory_evidence", "root_evidence_id", first.EvidenceID, 2)
	assertTableCountWhere(t, st.DB, "forge_k_memory_evidence_supersessions", "superseded_evidence_id", first.EvidenceID, 1)
	if _, err := st.DB.Exec(`DROP TRIGGER fail_memory_vsa_swap_e2e`); err != nil {
		t.Fatal(err)
	}
	if replacementResult, replacementErr := selection.Processor.Process(ctx, failedRebuild); replacementErr != nil || !replacementResult.Success {
		t.Fatalf("replacement governed VSA rebuild: err=%v result=%#v", replacementErr, replacementResult)
	}
	if err := st.DB.QueryRow(`SELECT manifest_hash FROM memory_vsa_projection_heads WHERE workspace_id=? AND lane_id=?`, scope.WorkspaceID, scope.LaneID).Scan(&activeHead); err != nil {
		t.Fatal(err)
	}
	if err := st.DB.QueryRow(`SELECT memory_evidence_id FROM forge_k_memory_vsa_pointers WHERE workspace_id=? AND lane_id=?`, scope.WorkspaceID, scope.LaneID).Scan(&activeEvidenceID); err != nil {
		t.Fatal(err)
	}
	if activeHead != revisedProjection.Manifest.ManifestHash || activeEvidenceID != second.EvidenceID {
		t.Fatalf("replacement projection not active: head=%s evidence=%s", activeHead, activeEvidenceID)
	}
	if _, err := st.DB.Exec(`INSERT INTO settings(key,value) VALUES('retrieval_vsa_mode','active') ON CONFLICT(key) DO UPDATE SET value=excluded.value`); err != nil {
		t.Fatal(err)
	}
	signals, err := memory.New(st.DB).ComputeVSAQuerySignals(ctx, memory.VSAQuerySignalsRequest{
		WorkspaceID: scope.WorkspaceID, LaneID: scope.LaneID, Query: second.ContentSummary,
		Candidates: []memory.VSAQueryCandidate{{ChunkID: 1, AbsPath: second.RawRef}},
	})
	if err != nil {
		t.Fatal(err)
	}
	signal, ok := signals[1]
	if !ok || signal.MemoryEvidenceID != second.EvidenceID || signal.MemoryEvidenceRowID == nil || signal.ObservationID != nil || signal.AppliedScore == 0 {
		t.Fatalf("replacement evidence did not exclusively drive governed scoring: %+v", signals)
	}
	assertTableCountWhere(t, st.DB, "forge_k_memory_evidence_supersessions", "syscall_id", revise.ID, 1)
	if _, err := st.DB.Exec(`UPDATE forge_k_memory_evidence SET content_summary='tampered' WHERE evidence_id=?`, first.EvidenceID); err == nil {
		t.Fatal("immutable evidence accepted UPDATE")
	}
	if _, err := st.DB.Exec(`DELETE FROM forge_k_memory_evidence_supersessions WHERE superseded_evidence_id=?`, first.EvidenceID); err == nil {
		t.Fatal("append-only supersession accepted DELETE")
	}
}

func TestForgeKMemoryEvidenceFailsClosedOnCallerContentScopeAndHashTamper(t *testing.T) {
	ctx := context.Background()
	selection, st, _ := newLiveSQLiteAuthority(t)
	scope := domain.ForgeScope{WorkspaceID: "ws-memory-evidence", LaneID: "memory.semantic", SelectedPaths: []string{"artifact:case-2"}}
	admitMemoryEvidenceSource(t, ctx, selection, "admit-memory-tamper", "case-tamper", "exhibit-tamper", "ruling-tamper", "c", scope)

	cases := map[string]func(domain.SyscallRequest) domain.SyscallRequest{
		"caller content": func(req domain.SyscallRequest) domain.SyscallRequest {
			req.Payload["contentSummary"] = "caller invented"
			return req
		},
		"workspace": func(req domain.SyscallRequest) domain.SyscallRequest { req.Scope.WorkspaceID = "other"; return req },
		"lane":      func(req domain.SyscallRequest) domain.SyscallRequest { req.Scope.LaneID = "other"; return req },
		"selected paths": func(req domain.SyscallRequest) domain.SyscallRequest {
			req.Scope.SelectedPaths = []string{"artifact:other"}
			return req
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			req := memoryEvidenceRequest(domain.ActionMaterializeAdmittedEvidence, "materialize-tamper-"+strings.ReplaceAll(name, " ", "-"), scope, map[string]any{
				"exhibitId": "exhibit-tamper", "rulingId": "ruling-tamper",
			})
			req = mutate(req)
			result, _ := selection.Processor.Process(ctx, req)
			if result.Success {
				t.Fatalf("tampered request succeeded: %#v", result)
			}
			assertTableCountWhere(t, st.DB, "forge_k_memory_evidence", "syscall_id", req.ID, 0)
		})
	}
	if _, err := st.DB.Exec(`UPDATE court_rulings SET content_hash=? WHERE id=?`, "sha256:"+strings.Repeat("d", 64), "ruling-tamper"); err == nil {
		t.Fatal("test precondition: Court immutability unexpectedly allowed hash tamper")
	}
}

func TestForgeKMemoryEvidenceRevisionRollbackAndConcurrentLeafCAS(t *testing.T) {
	ctx := context.Background()
	selection, st, _ := newLiveSQLiteAuthority(t)
	scope := domain.ForgeScope{WorkspaceID: "ws-memory-evidence", LaneID: "memory.semantic", SelectedPaths: []string{"artifact:case-3"}}
	admitMemoryEvidenceSource(t, ctx, selection, "admit-race-v1", "case-race", "exhibit-race-v1", "ruling-race-v1", "1", scope)
	initial := memoryEvidenceRequest(domain.ActionMaterializeAdmittedEvidence, "materialize-race-v1", scope, map[string]any{"exhibitId": "exhibit-race-v1", "rulingId": "ruling-race-v1"})
	if result, err := selection.Processor.Process(ctx, initial); err != nil || !result.Success {
		t.Fatalf("initial materialization: err=%v result=%#v", err, result)
	}
	priorID := initial.ID + ":memory_evidence"
	admitMemoryEvidenceSource(t, ctx, selection, "admit-race-v2", "case-race", "exhibit-race-v2", "ruling-race-v2", "2", scope)

	if _, err := st.DB.Exec(`CREATE TRIGGER fail_memory_supersession BEFORE INSERT ON forge_k_memory_evidence_supersessions BEGIN SELECT RAISE(ABORT,'injected supersession failure'); END`); err != nil {
		t.Fatal(err)
	}
	failed := memoryEvidenceRequest(domain.ActionReviseMemoryEvidence, "revise-rollback", scope, map[string]any{"exhibitId": "exhibit-race-v2", "rulingId": "ruling-race-v2", "priorEvidenceId": priorID})
	failedResult, _ := selection.Processor.Process(ctx, failed)
	if failedResult.Success {
		t.Fatalf("injected failure succeeded: %#v", failedResult)
	}
	for _, table := range []string{"forge_k_memory_evidence", "forge_k_memory_evidence_supersessions", "journal_events", "forge_k_audit_outbox"} {
		assertTableCountWhere(t, st.DB, table, "syscall_id", failed.ID, 0)
	}
	assertTableCountWhere(t, st.DB, "semantic_idempotency_keys", "idempotency_key", failed.IdempotencyKey, 0)
	if _, err := st.DB.Exec(`DROP TRIGGER fail_memory_supersession`); err != nil {
		t.Fatal(err)
	}

	admitMemoryEvidenceSource(t, ctx, selection, "admit-race-v3", "case-race", "exhibit-race-v3", "ruling-race-v3", "3", scope)
	requests := []domain.SyscallRequest{
		memoryEvidenceRequest(domain.ActionReviseMemoryEvidence, "revise-race-v2", scope, map[string]any{"exhibitId": "exhibit-race-v2", "rulingId": "ruling-race-v2", "priorEvidenceId": priorID}),
		memoryEvidenceRequest(domain.ActionReviseMemoryEvidence, "revise-race-v3", scope, map[string]any{"exhibitId": "exhibit-race-v3", "rulingId": "ruling-race-v3", "priorEvidenceId": priorID}),
	}
	start := make(chan struct{})
	var wg sync.WaitGroup
	successes := 0
	var mu sync.Mutex
	for _, req := range requests {
		req := req
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			result, _ := selection.Processor.Process(ctx, req)
			if result.Success {
				mu.Lock()
				successes++
				mu.Unlock()
			}
		}()
	}
	close(start)
	wg.Wait()
	if successes != 1 {
		t.Fatalf("concurrent prior-leaf revisions successes=%d, want exactly one", successes)
	}
	assertTableCountWhere(t, st.DB, "forge_k_memory_evidence_supersessions", "superseded_evidence_id", priorID, 1)
}

func TestForgeKMemoryEvidenceJournalFailureRollsBackEvidenceAndProof(t *testing.T) {
	ctx := context.Background()
	selection, st, _ := newLiveSQLiteAuthority(t)
	scope := domain.ForgeScope{WorkspaceID: "ws-memory-evidence", LaneID: "memory.semantic", SelectedPaths: []string{"artifact:journal"}}
	admitMemoryEvidenceSource(t, ctx, selection, "admit-memory-journal", "case-journal", "exhibit-journal", "ruling-journal", "e", scope)
	req := memoryEvidenceRequest(domain.ActionMaterializeAdmittedEvidence, "materialize-journal-failure", scope, map[string]any{"exhibitId": "exhibit-journal", "rulingId": "ruling-journal"})
	if _, err := st.DB.ExecContext(ctx, `INSERT INTO journal_events(id,type,source,workspace_id,created_at) VALUES(?,?,?,?,?)`, req.ID+":journal_event", "preexisting", "test", scope.WorkspaceID, req.RequestedAt); err != nil {
		t.Fatal(err)
	}
	result, _ := selection.Processor.Process(ctx, req)
	if result.Success {
		t.Fatalf("journal collision succeeded: %#v", result)
	}
	assertTableCountWhere(t, st.DB, "forge_k_memory_evidence", "syscall_id", req.ID, 0)
	assertTableCountWhere(t, st.DB, "semantic_idempotency_keys", "idempotency_key", req.IdempotencyKey, 0)
	assertTableCountWhere(t, st.DB, "forge_k_audit_outbox", "syscall_id", req.ID, 0)
}

func admitMemoryEvidenceSource(t *testing.T, ctx context.Context, selection Selection, requestID, caseID, exhibitID, rulingID, hashChar string, scope domain.ForgeScope) {
	t.Helper()
	req := liveCourtRequest(domain.ActionAdmitEvidence, requestID)
	req.Scope = scope
	req.Payload = map[string]any{
		"caseId": caseID, "exhibitId": exhibitID, "rulingId": rulingID,
		"sourceType": "artifact", "sourceRefs": []string{"artifact:" + exhibitID},
		"contentSummary": "Court summary " + exhibitID, "rawRef": "artifact:" + exhibitID,
		"contentHash": "sha256:" + strings.Repeat(hashChar, 64), "policyRefs": []string{"policy:court-v1"},
	}
	result, err := selection.Processor.Process(ctx, req)
	if err != nil || !result.Success {
		t.Fatalf("admit source %s: err=%v result=%#v", exhibitID, err, result)
	}
}

func memoryEvidenceRequest(action domain.SemanticActionType, id string, scope domain.ForgeScope, payload map[string]any) domain.SyscallRequest {
	return domain.SyscallRequest{
		ID: id, Action: action, Actor: domain.ActorIdentity{ID: "operator", Kind: string(domain.SourceUser)}, Source: domain.SourceUser,
		Scope: scope, Payload: payload,
		Provenance:    domain.Provenance{Actor: "operator", ActorType: "user", Source: "test", TraceID: "trace-" + id},
		CorrelationID: "corr-" + id, TraceID: "trace-" + id, RequestedAt: 1760000000000, IdempotencyKey: "idem-" + id,
	}
}

func memoryAccelerationRequest(id string, scope domain.ForgeScope, expectedManifestHash, expectedPriorManifestHash string) domain.SyscallRequest {
	req := memoryEvidenceRequest(domain.ActionRebuildMemoryAcceleration, id, scope, map[string]any{
		"algorithmName": vsaprojection.AlgorithmName, "algorithmVersion": vsaprojection.AlgorithmVersion,
		"dimensions": vsaprojection.DefaultDims, "seed": int(vsaprojection.DefaultSeed),
		"expectedManifestHash": expectedManifestHash, "expectedPriorManifestHash": expectedPriorManifestHash,
		"requestedAtMs": int64(1760000000000),
	})
	req.RequiredCapability = controllane.CapMemoryAccelerationRebuild
	return req
}

func assertTableCountWhere(t *testing.T, db interface {
	QueryRow(query string, args ...any) *sql.Row
}, table, column, value string, want int) {
	t.Helper()
	var count int
	if err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s=?", table, column), value).Scan(&count); err != nil || count != want {
		t.Fatalf("%s count for %s=%s: got=%d want=%d err=%v", table, column, value, count, want, err)
	}
}
