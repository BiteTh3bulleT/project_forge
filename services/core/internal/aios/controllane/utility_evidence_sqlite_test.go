package controllane

import (
	"context"
	"database/sql"
	"testing"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/forgekernel"
	"forge/projectforge/services/core/internal/store"
)

func TestSQLiteForgeKRetrievalUsefulnessIsAtomicImmutableAndReplaySafe(t *testing.T) {
	ctx := context.Background()
	authority, st := newSQLiteUtilityAuthority(t)
	resultID := seedSQLiteRetrievalUsefulnessTarget(t, st.DB)

	req := utilityUsefulnessRequest(resultID, "utility-usefulness-1")
	first, err := authority.Process(ctx, req)
	if err != nil || !first.Success {
		t.Fatalf("record usefulness: err=%v result=%+v", err, first)
	}
	second, err := authority.Process(ctx, req)
	if err != nil || !second.Success {
		t.Fatalf("replay usefulness: err=%v result=%+v", err, second)
	}
	if len(second.CommittedObjectIDs) != len(first.CommittedObjectIDs) {
		t.Fatalf("replay changed committed proof shape: first=%v second=%v", first.CommittedObjectIDs, second.CommittedObjectIDs)
	}
	conflict := req
	conflict.Payload = cloneIntegrityValue(req.Payload)
	conflict.Payload["label"] = "noisy"
	conflicted, err := authority.Process(ctx, conflict)
	if err != nil || conflicted.Success || conflicted.DeterministicErrCode != domain.ErrDuplicate {
		t.Fatalf("conflicting idempotency fingerprint did not fail closed: err=%v result=%+v", err, conflicted)
	}
	mismatched := utilityUsefulnessRequest(resultID, "utility-usefulness-scope-mismatch")
	mismatched.Scope.SelectedPaths = []string{"/other"}
	mismatchResult, err := authority.Process(ctx, mismatched)
	if err != nil || mismatchResult.Success {
		t.Fatalf("mismatched selected paths did not fail closed: err=%v result=%+v", err, mismatchResult)
	}

	assertSQLiteCount(t, st.DB, `SELECT COUNT(*) FROM forge_k_retrieval_usefulness_events WHERE syscall_id=?`, 1, req.ID)
	assertSQLiteCount(t, st.DB, `SELECT COUNT(*) FROM retrieval_usefulness_projection WHERE retrieval_result_id=? AND noncanonical=1`, 1, resultID)
	assertSQLiteCount(t, st.DB, `SELECT COUNT(*) FROM journal_events WHERE id=?`, 1, req.ID+":journal_event")
	assertSQLiteCount(t, st.DB, `SELECT COUNT(*) FROM forge_k_audit_outbox WHERE syscall_id=?`, 1, req.ID)
	assertSQLiteCount(t, st.DB, `SELECT COUNT(*) FROM semantic_idempotency_keys WHERE idempotency_key=?`, 1, req.IdempotencyKey)
	assertSQLiteCount(t, st.DB, `SELECT COUNT(*) FROM forge_k_retrieval_usefulness_events WHERE syscall_id=?`, 0, mismatched.ID)
	assertSQLiteCount(t, st.DB, `SELECT COUNT(*) FROM memory_usefulness_events`, 0)

	if _, err := st.DB.Exec(`UPDATE forge_k_retrieval_usefulness_events SET label='noisy' WHERE syscall_id=?`, req.ID); err == nil {
		t.Fatal("immutable usefulness evidence accepted UPDATE")
	}
	var baseLabel, projectionLabel string
	if err := st.DB.QueryRow(`SELECT usefulness_label FROM retrieval_results WHERE id=?`, resultID).Scan(&baseLabel); err != nil {
		t.Fatal(err)
	}
	if err := st.DB.QueryRow(`SELECT label FROM retrieval_usefulness_projection WHERE retrieval_result_id=?`, resultID).Scan(&projectionLabel); err != nil {
		t.Fatal(err)
	}
	if baseLabel != "unknown" || projectionLabel != "useful" {
		t.Fatalf("source/projection separation lost: base=%q projection=%q", baseLabel, projectionLabel)
	}
}

func TestSQLiteForgeKRestoreFeedbackPreservesOriginalAndRollsBackWithJournal(t *testing.T) {
	ctx := context.Background()
	authority, st := newSQLiteUtilityAuthority(t)
	seedSQLiteRestoreOutcome(t, st.DB)

	req := utilityRestoreFeedbackRequest("utility-restore-feedback-1")
	result, err := authority.Process(ctx, req)
	if err != nil || !result.Success {
		t.Fatalf("record restore feedback: err=%v result=%+v", err, result)
	}
	assertSQLiteCount(t, st.DB, `SELECT COUNT(*) FROM forge_k_restore_outcome_feedback_events WHERE syscall_id=?`, 1, req.ID)
	assertSQLiteCount(t, st.DB, `SELECT COUNT(*) FROM restore_outcome_feedback_projection WHERE restore_outcome_id=? AND noncanonical=1`, 1, "restore-source-1")
	var original, projection string
	if err := st.DB.QueryRow(`SELECT outcome FROM restore_outcome_events WHERE id='restore-source-1'`).Scan(&original); err != nil {
		t.Fatal(err)
	}
	if err := st.DB.QueryRow(`SELECT outcome FROM restore_outcome_feedback_projection WHERE restore_outcome_id='restore-source-1'`).Scan(&projection); err != nil {
		t.Fatal(err)
	}
	if original != "unknown" || projection != "operator_corrected" {
		t.Fatalf("original/projection separation lost: original=%q projection=%q", original, projection)
	}
	signals, warnings := listRestoreOutcomeSignals(ctx, NewSQLiteSemanticStore(st.DB), domain.ForgeScope{
		WorkspaceID: "ws-restore", LaneID: "control.semantic", SelectedPaths: []string{"/repo"},
	}, "restore query", 10)
	if len(warnings) != 0 || len(signals) != 1 || signals[0].Outcome != RestoreOutcomeOperatorCorrected ||
		signals[0].Metadata["utilityEvidenceSource"] != "governed_feedback_projection" {
		t.Fatalf("SQLite effective utility view ignored governed projection: signals=%+v warnings=%v", signals, warnings)
	}
	adjustment, trace := restoreOutcomeUtilityAdjustment(compileContextRestoreCandidateScore{
		SnapshotID: "snapshot-restore-source-1", ContextPacketID: "ctx-restore-source-1",
	}, "restore query", signals)
	if adjustment >= 0 || trace["outcome_sources"] == nil {
		t.Fatalf("governed SQLite projection did not change restore score deterministically: adjustment=%v trace=%+v", adjustment, trace)
	}
	if _, err := st.DB.Exec(`DELETE FROM forge_k_restore_outcome_feedback_events WHERE syscall_id=?`, req.ID); err == nil {
		t.Fatal("immutable restore feedback evidence accepted DELETE")
	}

	rollbackReq := utilityRestoreFeedbackRequest("utility-restore-feedback-rollback")
	rollbackReq.IdempotencyKey = "utility-restore-feedback-rollback-idem"
	if _, err := st.DB.Exec(`CREATE TRIGGER fail_utility_journal BEFORE INSERT ON journal_events WHEN NEW.id='utility-restore-feedback-rollback:journal_event' BEGIN SELECT RAISE(FAIL, 'forced utility journal failure'); END`); err != nil {
		t.Fatalf("create journal failure trigger: %v", err)
	}
	rolledBack, err := authority.Process(ctx, rollbackReq)
	if rolledBack.Success {
		t.Fatalf("forced journal failure did not fail closed: err=%v result=%+v", err, rolledBack)
	}
	assertSQLiteCount(t, st.DB, `SELECT COUNT(*) FROM forge_k_restore_outcome_feedback_events WHERE syscall_id=?`, 0, rollbackReq.ID)
	assertSQLiteCount(t, st.DB, `SELECT COUNT(*) FROM restore_outcome_feedback_projection WHERE latest_event_id=?`, 0, "restore-outcome-feedback:"+rollbackReq.ID)
	assertSQLiteCount(t, st.DB, `SELECT COUNT(*) FROM forge_k_audit_outbox WHERE syscall_id=?`, 0, rollbackReq.ID)
	assertSQLiteCount(t, st.DB, `SELECT COUNT(*) FROM semantic_idempotency_keys WHERE idempotency_key=?`, 0, rollbackReq.IdempotencyKey)
}

func newSQLiteUtilityAuthority(t *testing.T) (forgekernel.Processor, *store.Store) {
	t.Helper()
	processor, _, st := newSQLiteKernel(t, nil)
	authorization, err := NewProductionAuthorizationService(ProductionAuthorizationOptions{
		Registry: NewStaticActionRegistry(), DB: st.DB, ServicePrincipal: NewForgeCoreServicePrincipal(),
	})
	if err != nil {
		t.Fatal(err)
	}
	selection, err := forgekernel.SelectAuthority(processor, authorization)
	if err != nil {
		t.Fatal(err)
	}
	return selection.Processor, st
}

func utilityUsefulnessRequest(resultID int64, id string) domain.SyscallRequest {
	return domain.SyscallRequest{
		ID: id, Action: domain.ActionRecordRetrievalUsefulness,
		Actor: domain.ActorIdentity{ID: "forge.core", Kind: "service"}, Source: domain.SourceInternal,
		Scope:         domain.ForgeScope{WorkspaceID: "ws-retrieval", LaneID: "control.semantic", SelectedPaths: []string{"/repo"}},
		Payload:       map[string]any{"resultId": resultID, "label": "useful", "note": "bounded utility evidence", "metadata": map[string]any{"reason": "selected result helped"}},
		Provenance:    domain.Provenance{Actor: "forge.core", ActorType: "system", Source: "utility-test", TraceID: "trace-" + id},
		CorrelationID: "corr-" + id, TraceID: "trace-" + id, RequestedAt: 200,
		IdempotencyKey: id + "-idem", RequiredCapability: CapRetrievalUsefulnessRecord,
	}
}

func utilityRestoreFeedbackRequest(id string) domain.SyscallRequest {
	return domain.SyscallRequest{
		ID: id, Action: domain.ActionRecordRestoreOutcomeFeedback,
		Actor: domain.ActorIdentity{ID: "forge.core", Kind: "service"}, Source: domain.SourceInternal,
		Scope: domain.ForgeScope{WorkspaceID: "ws-restore", LaneID: "control.semantic", SelectedPaths: []string{"/repo"}},
		Payload: map[string]any{
			"restoreOutcomeId": "restore-source-1", "outcome": "operator_corrected", "outcomeConfidence": 0.9,
			"operatorFeedback": "use newer evidence", "correctionSummary": "selection was stale", "metadata": map[string]any{"reason": "operator review"},
		},
		Provenance:    domain.Provenance{Actor: "forge.core", ActorType: "system", Source: "utility-test", TraceID: "trace-" + id},
		CorrelationID: "corr-" + id, TraceID: "trace-" + id, RequestedAt: 300,
		IdempotencyKey: id + "-idem", RequiredCapability: CapRestoreOutcomeFeedback,
	}
}

func seedSQLiteRetrievalUsefulnessTarget(t *testing.T, db *sql.DB) int64 {
	t.Helper()
	result, err := db.Exec(`INSERT INTO retrieval_runs(evidence_id,created_at,query,mode,workspace_id,lane_id,selected_paths_json,syscall_id,provenance_id,provenance_json,proposed_by,committed_by) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`,
		"source-retrieval-run", 100, "utility target", "keyword", "ws-retrieval", "control.semantic", `["/repo"]`, "source-retrieval-syscall", "source-retrieval-provenance", `{"actor":"forge.core","actorType":"system"}`, "internal", "forge_k.kernel")
	if err != nil {
		t.Fatalf("seed retrieval run: %v", err)
	}
	runID, _ := result.LastInsertId()
	result, err = db.Exec(`INSERT INTO retrieval_results(evidence_id,retrieval_run_id,abs_path,rel_path,rank_index,usefulness_label) VALUES(?,?,?,?,?,?)`,
		"source-retrieval-result", runID, "/repo/source.go", "source.go", 0, "unknown")
	if err != nil {
		t.Fatalf("seed retrieval result: %v", err)
	}
	resultID, _ := result.LastInsertId()
	return resultID
}

func seedSQLiteRestoreOutcome(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO context_packet_snapshots(id,query,workspace_id,lane_id,selected_paths_json,created_at,syscall_id,committed_by) VALUES(?,?,?,?,?,?,?,?)`,
		"ctx-restore-source-1", "restore query", "ws-restore", "control.semantic", `["/repo"]`, 100, "compile-source-1", "forge_kernel"); err != nil {
		t.Fatalf("seed context snapshot: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO restore_outcome_events(id,created_at,updated_at,workspace_id,lane_id,query,context_packet_id,snapshot_id,outcome,syscall_id,committed_by) VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
		"restore-source-1", 100, 100, "ws-restore", "control.semantic", "restore query", "ctx-restore-source-1", "snapshot-restore-source-1", "unknown", "compile-source-1", "forge_kernel"); err != nil {
		t.Fatalf("seed restore outcome: %v", err)
	}
}

func assertSQLiteCount(t *testing.T, db *sql.DB, query string, want int, args ...any) {
	t.Helper()
	var got int
	if err := db.QueryRow(query, args...).Scan(&got); err != nil {
		t.Fatalf("count query failed: %v", err)
	}
	if got != want {
		t.Fatalf("count=%d want=%d for %s", got, want, query)
	}
}
