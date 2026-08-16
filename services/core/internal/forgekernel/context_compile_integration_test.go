package forgekernel_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"forge/projectforge/services/core/internal/aios/controllane"
	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/forgekernel/contextcompile"
)

func TestForgeKContextCompilerUsesOnlyCurrentAdmittedExactScopeEvidence(t *testing.T) {
	ctx := context.Background()
	selection, st, _ := newLiveSQLiteAuthority(t)
	scope := domain.ForgeScope{WorkspaceID: "ws-context-k", LaneID: "control.semantic", SelectedPaths: []string{"/workspace/project"}}

	admitMemoryEvidenceSource(t, ctx, selection, "context-admit-main", "context-case-main", "context-exhibit-main", "context-ruling-main", "a", scope)
	materialize := memoryEvidenceRequest(domain.ActionMaterializeAdmittedEvidence, "context-materialize-main", scope, map[string]any{
		"exhibitId": "context-exhibit-main", "rulingId": "context-ruling-main",
	})
	if result, err := selection.Processor.Process(ctx, materialize); err != nil || !result.Success {
		t.Fatalf("materialize admitted source: err=%v result=%#v", err, result)
	}

	otherScope := domain.ForgeScope{WorkspaceID: scope.WorkspaceID, LaneID: scope.LaneID, SelectedPaths: []string{"/workspace/other"}}
	admitMemoryEvidenceSource(t, ctx, selection, "context-admit-other", "context-case-other", "context-exhibit-other", "context-ruling-other", "b", otherScope)
	other := memoryEvidenceRequest(domain.ActionMaterializeAdmittedEvidence, "context-materialize-other", otherScope, map[string]any{
		"exhibitId": "context-exhibit-other", "rulingId": "context-ruling-other",
	})
	if result, err := selection.Processor.Process(ctx, other); err != nil || !result.Success {
		t.Fatalf("materialize other-scope source: err=%v result=%#v", err, result)
	}

	// Historical legacy rows remain inspectable, but are never context authority.
	if _, err := st.DB.ExecContext(ctx, `INSERT INTO memory_notes(id,type,title,content,status,confidence,workspace_id,lane_id,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?)`,
		"legacy-context-note", "fact", "legacy", "must not enter prompt context", "active", 1.0, scope.WorkspaceID, scope.LaneID, 1, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := st.DB.ExecContext(ctx, `INSERT INTO context_packet_snapshots(id,query,workspace_id,lane_id,selected_paths_json,created_at) VALUES(?,?,?,?,?,?)`,
		"legacy-context-snapshot", "compile current evidence", scope.WorkspaceID, scope.LaneID, `["/workspace/project"]`, 1); err != nil {
		t.Fatal(err)
	}

	req := liveContextCompileRequest("context-compile-main", scope, "compile current evidence")
	result, err := selection.Processor.Process(ctx, req)
	if err != nil || !result.Success {
		t.Fatalf("compile governed context: err=%v result=%#v", err, result)
	}
	if result.StateSummary["kernelContextCompiler"] != true || result.StateSummary["legacyContextInputs"] != false {
		t.Fatalf("context authority flags: %#v", result.StateSummary)
	}

	var inputRaw, decisionRaw, sourcesRaw string
	if err := st.DB.QueryRowContext(ctx, `SELECT input_json,decision_json,sources_json FROM forge_k_context_bundles WHERE syscall_id=?`, req.ID).Scan(&inputRaw, &decisionRaw, &sourcesRaw); err != nil {
		t.Fatal(err)
	}
	var input contextcompile.Input
	var decision contextcompile.Decision
	var sources []controllane.GovernedContextSource
	if err := json.Unmarshal([]byte(inputRaw), &input); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(decisionRaw), &decision); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(sourcesRaw), &sources); err != nil {
		t.Fatal(err)
	}
	if err := contextcompile.VerifyDecision(input, decision); err != nil {
		t.Fatalf("stored pure decision verification: %v", err)
	}
	wantEvidence := materialize.ID + ":memory_evidence"
	if len(input.SourceManifest.Sources) != 1 || input.SourceManifest.Sources[0].EvidenceID != wantEvidence ||
		len(sources) != 1 || sources[0].EvidenceID != wantEvidence {
		t.Fatalf("unexpected governed source set: manifest=%#v sources=%#v", input.SourceManifest.Sources, sources)
	}
	if strings.Contains(sourcesRaw, "legacy-context-note") || strings.Contains(inputRaw, other.ID) {
		t.Fatalf("legacy or cross-scope evidence entered governed bundle: input=%s sources=%s", inputRaw, sourcesRaw)
	}
	assertTableCountWhere(t, st.DB, "forge_k_context_bundles", "syscall_id", req.ID, 1)
	assertTableCountWhere(t, st.DB, "journal_events", "syscall_id", req.ID, 1)
	assertTableCountWhere(t, st.DB, "forge_k_audit_outbox", "syscall_id", req.ID, 1)
	assertTableCountWhere(t, st.DB, "semantic_idempotency_keys", "idempotency_key", req.IdempotencyKey, 1)

	if replay, replayErr := selection.Processor.Process(ctx, req); replayErr != nil || !replay.Success {
		t.Fatalf("verified context replay: err=%v result=%#v", replayErr, replay)
	}
	assertTableCountWhere(t, st.DB, "forge_k_context_bundles", "syscall_id", req.ID, 1)
	var revision int64
	if err := st.DB.QueryRowContext(ctx, `SELECT revision FROM forge_k_context_snapshot_heads WHERE workspace_id=? AND lane_id=?`, scope.WorkspaceID, scope.LaneID).Scan(&revision); err != nil || revision != 1 {
		t.Fatalf("replay changed context head: revision=%d err=%v", revision, err)
	}
	if _, err := st.DB.ExecContext(ctx, `UPDATE forge_k_context_bundles SET query='tampered' WHERE syscall_id=?`, req.ID); err == nil {
		t.Fatal("immutable governed context bundle accepted UPDATE")
	}
}

func TestForgeKContextCompilerJournalFailureRollsBackBundleHeadAndProof(t *testing.T) {
	ctx := context.Background()
	selection, st, _ := newLiveSQLiteAuthority(t)
	scope := domain.ForgeScope{WorkspaceID: "ws-context-rollback", LaneID: "control.semantic", SelectedPaths: []string{"/workspace/rollback"}}
	req := liveContextCompileRequest("context-compile-rollback", scope, "empty admitted context is valid")
	if _, err := st.DB.ExecContext(ctx, `INSERT INTO journal_events(id,type,source,workspace_id,created_at) VALUES(?,?,?,?,?)`, req.ID+":journal_event", "preexisting", "test", scope.WorkspaceID, req.RequestedAt); err != nil {
		t.Fatal(err)
	}
	result, _ := selection.Processor.Process(ctx, req)
	if result.Success {
		t.Fatalf("journal collision succeeded: %#v", result)
	}
	for _, table := range []string{"forge_k_context_bundles", "forge_k_context_snapshot_heads", "forge_k_audit_outbox"} {
		var count int
		if err := st.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&count); err != nil || count != 0 {
			t.Fatalf("%s rollback count=%d err=%v", table, count, err)
		}
	}
	assertTableCountWhere(t, st.DB, "semantic_idempotency_keys", "idempotency_key", req.IdempotencyKey, 0)
}

func liveContextCompileRequest(id string, scope domain.ForgeScope, query string) domain.SyscallRequest {
	req := memoryEvidenceRequest(domain.ActionCompileContext, id, scope, map[string]any{
		"query": query, "budget": map[string]any{"maxTokens": 2048, "maxEvents": 16, "maxNotes": 16},
		"persistSnapshot": true, "snapshotKind": "chat",
	})
	req.RequiredCapability = controllane.CapContextCompile
	return req
}
