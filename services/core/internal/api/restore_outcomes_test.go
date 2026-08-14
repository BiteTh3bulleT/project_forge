package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"forge/projectforge/services/core/internal/forgekernel"
)

func TestRestoreOutcomeFeedbackAPIRoutesThroughForgeKWithoutMutatingOriginal(t *testing.T) {
	srv, st := newBackupAuditHarness(t)
	processor := &capturingUtilityProcessor{}
	srv.kernelAuthority = forgekernel.Selection{Processor: processor}
	srv.kernelAuthorizationReady = true
	_, err := st.DB.Exec(`INSERT INTO restore_outcome_events(id,created_at,updated_at,workspace_id,lane_id,query,context_packet_id,snapshot_id,snapshot_kind,restore_score,requires_fresh_compile,selected_evidence_json,selected_state_keys_json,selected_loop_ids_json,selected_artifact_ids_json,outcome,outcome_confidence,downstream_action_type,downstream_object_id,metadata_json) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		"restore-outcome-api", 1000, 1000, "ws-api", "control.semantic", "restore blockers", "ctx-api", "snap-api", "restore", 0.55, 0, `["note-a"]`, `[]`, `[]`, `[]`, "unknown", 0, "compile_context", "ctx-api", `{}`)
	if err != nil {
		t.Fatalf("insert outcome: %v", err)
	}
	_, err = st.DB.Exec(`INSERT INTO provenance_records(id,actor,actor_type,source,trace_id,workspace_id,lane_id,selected_paths_json,metadata_json,created_at,proposed_by,committed_by,syscall_id,correlation_id) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		"prov-restore-feedback-api", "operator", "user", "api", "trace-feedback", "ws-api", "control.semantic", `["/workspace/project"]`, `{}`, 1001, "user", "forge_k.kernel", "sys-feedback", "corr-feedback")
	if err != nil {
		t.Fatalf("insert feedback provenance: %v", err)
	}
	_, err = st.DB.Exec(`INSERT INTO forge_k_restore_outcome_feedback_events(id,created_at,restore_outcome_id,original_outcome,workspace_id,lane_id,selected_paths_json,outcome,outcome_confidence,operator_feedback,correction_summary,prior_projection_json,projection_snapshot_json,metadata_json,correlation_id,trace_id,syscall_id,provenance_id,provenance_json,proposed_by,committed_by) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		"restore-feedback-event-api", 1001, "restore-outcome-api", "unknown", "ws-api", "control.semantic", `["/workspace/project"]`, "operator_corrected", 0.9, "wrong evidence", "use newer notes", `{}`, `{}`, `{}`, "corr-feedback", "trace-feedback", "sys-feedback", "prov-restore-feedback-api", `{}`, "user", "forge_k.kernel")
	if err != nil {
		t.Fatalf("insert immutable feedback event: %v", err)
	}
	_, err = st.DB.Exec(`INSERT INTO restore_outcome_feedback_projection(restore_outcome_id,latest_event_id,workspace_id,lane_id,outcome,outcome_confidence,operator_feedback,correction_summary,updated_by,updated_at,metadata_json,noncanonical) VALUES(?,?,?,?,?,?,?,?,?,?,?,1)`,
		"restore-outcome-api", "restore-feedback-event-api", "ws-api", "control.semantic", "operator_corrected", 0.9, "wrong evidence", "use newer notes", "operator", 1001, `{}`)
	if err != nil {
		t.Fatalf("insert feedback projection: %v", err)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/context/restore/outcomes?workspaceId=ws-api", nil)
	listRR := httptest.NewRecorder()
	srv.handleRestoreOutcomeList(listRR, listReq)
	if listRR.Code != http.StatusOK {
		t.Fatalf("list outcomes status=%d body=%s", listRR.Code, listRR.Body.String())
	}
	var listResp struct {
		Outcomes []struct {
			Outcome struct {
				Outcome string `json:"outcome"`
			} `json:"outcome"`
			FeedbackProjection *struct {
				Outcome      string `json:"outcome"`
				NonCanonical bool   `json:"nonCanonical"`
			} `json:"feedbackProjection"`
		} `json:"outcomes"`
	}
	if err := json.Unmarshal(listRR.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(listResp.Outcomes) != 1 {
		t.Fatalf("expected one outcome, got %+v", listResp.Outcomes)
	}
	if listResp.Outcomes[0].Outcome.Outcome != "unknown" || listResp.Outcomes[0].FeedbackProjection == nil ||
		listResp.Outcomes[0].FeedbackProjection.Outcome != "operator_corrected" || !listResp.Outcomes[0].FeedbackProjection.NonCanonical {
		t.Fatalf("read must preserve original and label separate projection: %+v", listResp.Outcomes[0])
	}

	wrongReq := withRouteParam(httptest.NewRequest(http.MethodGet, "/api/context/restore/outcomes/restore-outcome-api?workspaceId=ws-other", nil), "id", "restore-outcome-api")
	wrongRR := httptest.NewRecorder()
	srv.handleRestoreOutcomeGet(wrongRR, wrongReq)
	if wrongRR.Code != http.StatusNotFound {
		t.Fatalf("wrong workspace should not read outcome, got status=%d body=%s", wrongRR.Code, wrongRR.Body.String())
	}
	rightReq := withRouteParam(httptest.NewRequest(http.MethodGet, "/api/context/restore/outcomes/restore-outcome-api?workspaceId=ws-api&laneId=control.semantic", nil), "id", "restore-outcome-api")
	rightRR := httptest.NewRecorder()
	srv.handleRestoreOutcomeGet(rightRR, rightReq)
	if rightRR.Code != http.StatusOK || !strings.Contains(rightRR.Body.String(), `"feedbackProjection"`) ||
		!strings.Contains(rightRR.Body.String(), `"outcome":"unknown"`) || !strings.Contains(rightRR.Body.String(), `"outcome":"operator_corrected"`) {
		t.Fatalf("scoped read did not separate original evidence and feedback projection: status=%d body=%s", rightRR.Code, rightRR.Body.String())
	}

	body := []byte(`{"workspaceId":"ws-api","laneId":"control.semantic","selectedPaths":["/workspace/project"],"idempotencyKey":"restore-feedback-api-1","outcome":"operator_corrected","outcomeConfidence":0.9,"operatorFeedback":"wrong evidence","correctionSummary":"use newer notes","updatedBy":"spoofed-actor"}`)
	feedbackReq := withRouteParam(httptest.NewRequest(http.MethodPost, "/api/context/restore/outcomes/restore-outcome-api/feedback", bytes.NewReader(body)), "id", "restore-outcome-api")
	feedbackRR := httptest.NewRecorder()
	srv.handleRestoreOutcomeFeedback(feedbackRR, feedbackReq)
	if feedbackRR.Code != http.StatusOK {
		t.Fatalf("feedback status=%d body=%s", feedbackRR.Code, feedbackRR.Body.String())
	}
	if len(processor.requests) != 1 {
		t.Fatalf("utility syscall count=%d", len(processor.requests))
	}
	req := processor.requests[0]
	if req.Action != "RECORD_RESTORE_OUTCOME_FEEDBACK" || req.IdempotencyKey != "restore-feedback-api-1" {
		t.Fatalf("unexpected utility request: %+v", req)
	}
	if req.Scope.WorkspaceID != "ws-api" || req.Scope.LaneID != "control.semantic" || len(req.Scope.SelectedPaths) != 1 {
		t.Fatalf("utility request scope mismatch: %+v", req.Scope)
	}
	if req.Provenance.Actor == "spoofed-actor" || req.Provenance.Actor != "operator" {
		t.Fatalf("API trusted caller-filled provenance: %+v", req.Provenance)
	}
	var noteCount int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM memory_notes`).Scan(&noteCount); err != nil {
		t.Fatalf("count memory_notes: %v", err)
	}
	if noteCount != 0 {
		t.Fatalf("feedback must not mutate canonical memory, note count=%d", noteCount)
	}
	var outcome, correction string
	if err := st.DB.QueryRow(`SELECT outcome, correction_summary FROM restore_outcome_events WHERE id=?`, "restore-outcome-api").Scan(&outcome, &correction); err != nil {
		t.Fatalf("read outcome: %v", err)
	}
	if outcome != "unknown" || correction != "" {
		t.Fatalf("K-routed feedback mutated original evidence: outcome=%q correction=%q", outcome, correction)
	}
}

func TestRestoreOutcomeFeedbackStillRejectsOversizeRequestBody(t *testing.T) {
	t.Parallel()

	s := &Server{}
	oversizeBody := `{"workspaceId":"ws-api","outcome":"operator_corrected","operatorFeedback":"` + strings.Repeat("a", int(restoreOutcomeFeedbackRequestBodyLimit)+1) + `"}`
	req := withRouteParam(
		httptest.NewRequest(http.MethodPost, "/api/context/restore/outcomes/restore-outcome-api/feedback", bytes.NewReader([]byte(oversizeBody))),
		"id",
		"restore-outcome-api",
	)
	rr := httptest.NewRecorder()

	s.handleRestoreOutcomeFeedback(rr, req)

	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status=%d want=%d body=%s", rr.Code, http.StatusRequestEntityTooLarge, rr.Body.String())
	}
	if !strings.Contains(strings.ToLower(rr.Body.String()), "too large") {
		t.Fatalf("expected too-large response, got %q", rr.Body.String())
	}
}
