package forgekernel_test

import (
	"context"
	"strings"
	"testing"

	"forge/projectforge/services/core/internal/aios/controllane"
	"forge/projectforge/services/core/internal/aios/domain"
	. "forge/projectforge/services/core/internal/forgekernel"
	"forge/projectforge/services/core/internal/forgekernel/semanticdiff"
)

func TestForgeKComputesSemanticDiffFromCurrentAdmittedEvidenceAndReplays(t *testing.T) {
	ctx := context.Background()
	selection, st, _ := newLiveSQLiteAuthority(t)
	scope := domain.ForgeScope{WorkspaceID: "ws-semantic-diff", LaneID: "memory.semantic", SelectedPaths: []string{"artifact:semantic-diff"}}
	leftID := admitAndMaterializeDiffSource(t, ctx, selection, "left", "a", scope)
	rightID := admitAndMaterializeDiffSource(t, ctx, selection, "right", "b", scope)

	req := memoryEvidenceRequest(domain.ActionComputeSemanticDiff, "semantic-diff-live", scope, map[string]any{
		"leftEvidenceId": leftID, "rightEvidenceId": rightID,
		"operatorVersion": semanticdiff.OperatorVersion,
	})
	result, err := selection.Processor.Process(ctx, req)
	if err != nil || !result.Success {
		t.Fatalf("semantic diff commit: err=%v result=%#v", err, result)
	}
	wantIDs := []string{req.ID + ":semantic_operation", req.ID + ":semantic_result", req.ID + ":semantic_object", req.ID + ":journal_event"}
	for _, id := range wantIDs {
		if !contains(result.CommittedObjectIDs, id) {
			t.Fatalf("committed IDs missing %s: %+v", id, result.CommittedObjectIDs)
		}
	}
	read := controllane.NewSQLiteSemanticStore(st.DB)
	operation, ok := read.FindSemanticDiffOperation(req.ID+":semantic_operation", scope)
	if !ok || operation.Left.EvidenceID != leftID || operation.Right.EvidenceID != rightID ||
		operation.OperatorVersion != semanticdiff.OperatorVersion || operation.SourceManifestHash == "" ||
		operation.CommittedBy != AuthorityOwnerForgeK {
		t.Fatalf("unexpected operation: ok=%t operation=%#v", ok, operation)
	}
	storedResult, ok := read.FindSemanticDiffResult(req.ID + ":semantic_result")
	if !ok || storedResult.ContentHash == "" || storedResult.SourceManifestHash != operation.SourceManifestHash {
		t.Fatalf("unexpected result: ok=%t result=%#v", ok, storedResult)
	}
	object, ok := read.FindSemanticDerivedObject(req.ID+":semantic_object", scope)
	if !ok || object.CanonicalTruth || object.ObjectClass != semanticdiff.DerivedObjectClass ||
		object.ContentHash != storedResult.ContentHash || object.SourceManifestHash != operation.SourceManifestHash {
		t.Fatalf("unexpected derived object: ok=%t object=%#v", ok, object)
	}
	proof, ok := read.GetIdempotency(req.IdempotencyKey)
	if !ok || proof.Plan.Details["semanticSourceManifestHash"] != operation.SourceManifestHash ||
		proof.Plan.Details["semanticResultContentHash"] != storedResult.ContentHash ||
		proof.Plan.Details["semanticObjectClass"] != semanticdiff.DerivedObjectClass {
		t.Fatalf("sealed semantic commitments missing: ok=%t plan=%#v", ok, proof.Plan)
	}

	replay, replayErr := selection.Processor.Process(ctx, req)
	if replayErr != nil || !replay.Success || !contains(replay.Warnings, "idempotent replay") {
		t.Fatalf("semantic diff replay: err=%v result=%#v", replayErr, replay)
	}
	for _, table := range []string{"forge_k_semantic_diff_operations", "forge_k_semantic_diff_results", "forge_k_semantic_derived_objects"} {
		assertTableCountWhere(t, st.DB, table, "syscall_id", req.ID, 1)
	}

	conflict := req
	conflict.ID = "semantic-diff-conflicting-retry"
	conflict.CorrelationID = "corr-" + conflict.ID
	conflict.TraceID = "trace-" + conflict.ID
	conflict.Payload = map[string]any{
		"leftEvidenceId": rightID, "rightEvidenceId": leftID,
		"operatorVersion": semanticdiff.OperatorVersion,
	}
	if conflictResult, _ := selection.Processor.Process(ctx, conflict); conflictResult.Success {
		t.Fatalf("conflicting idempotency payload succeeded: %#v", conflictResult)
	}

	if _, err := st.DB.Exec(`UPDATE forge_k_semantic_diff_results SET content='tampered' WHERE id=?`, storedResult.ID); err == nil {
		t.Fatal("immutable semantic result accepted UPDATE")
	}
	if _, err := st.DB.Exec(`DELETE FROM forge_k_semantic_derived_objects WHERE id=?`, object.ID); err == nil {
		t.Fatal("immutable semantic object accepted DELETE")
	}
}

func TestForgeKSemanticDiffFailsClosedForLegacySourceAndRollsBackJournalCollision(t *testing.T) {
	ctx := context.Background()
	selection, st, _ := newLiveSQLiteAuthority(t)
	scope := domain.ForgeScope{WorkspaceID: "ws-semantic-diff-fail", LaneID: "memory.semantic", SelectedPaths: []string{"artifact:semantic-diff-fail"}}
	leftID := admitAndMaterializeDiffSource(t, ctx, selection, "fail-left", "c", scope)
	rightID := admitAndMaterializeDiffSource(t, ctx, selection, "fail-right", "d", scope)

	base := memoryEvidenceRequest(domain.ActionComputeSemanticDiff, "semantic-diff-fail", scope, map[string]any{
		"leftEvidenceId": leftID, "rightEvidenceId": rightID,
		"operatorVersion": semanticdiff.OperatorVersion,
	})
	adapter := base
	adapter.ID = "semantic-diff-adapter"
	adapter.IdempotencyKey = "idem-semantic-diff-adapter"
	adapter.Source = domain.SourceAdapter
	adapter.Actor = domain.ActorIdentity{ID: "adapter", Kind: string(domain.SourceAdapter)}
	adapter.Provenance = domain.Provenance{Actor: "adapter", ActorType: "adapter", Source: "test", TraceID: adapter.TraceID}
	if result, _ := selection.Processor.Process(ctx, adapter); result.Success {
		t.Fatalf("adapter executed semantic algebra: %#v", result)
	}

	if _, err := st.DB.ExecContext(ctx, `INSERT INTO journal_events(id,type,source,workspace_id,created_at) VALUES(?,?,?,?,?)`,
		base.ID+":journal_event", "preexisting", "test", scope.WorkspaceID, base.RequestedAt); err != nil {
		t.Fatal(err)
	}
	result, _ := selection.Processor.Process(ctx, base)
	if result.Success {
		t.Fatalf("journal collision succeeded: %#v", result)
	}
	for _, table := range []string{"forge_k_semantic_diff_operations", "forge_k_semantic_diff_results", "forge_k_semantic_derived_objects", "forge_k_audit_outbox"} {
		assertTableCountWhere(t, st.DB, table, "syscall_id", base.ID, 0)
	}
	assertTableCountWhere(t, st.DB, "semantic_idempotency_keys", "idempotency_key", base.IdempotencyKey, 0)
}

func admitAndMaterializeDiffSource(t *testing.T, ctx context.Context, selection Selection, suffix, hashChar string, scope domain.ForgeScope) string {
	t.Helper()
	admitID := "admit-semantic-" + suffix
	exhibitID := "exhibit-semantic-" + suffix
	rulingID := "ruling-semantic-" + suffix
	admitMemoryEvidenceSource(t, ctx, selection, admitID, "case-semantic", exhibitID, rulingID, hashChar, scope)
	materializeID := "materialize-semantic-" + suffix
	request := memoryEvidenceRequest(domain.ActionMaterializeAdmittedEvidence, materializeID, scope, map[string]any{
		"exhibitId": exhibitID, "rulingId": rulingID,
	})
	result, err := selection.Processor.Process(ctx, request)
	if err != nil || !result.Success {
		t.Fatalf("materialize %s: err=%v result=%#v", suffix, err, result)
	}
	return materializeID + ":memory_evidence"
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == target {
			return true
		}
	}
	return false
}
