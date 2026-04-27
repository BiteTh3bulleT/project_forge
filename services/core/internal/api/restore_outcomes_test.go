package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRestoreOutcomeFeedbackAPIIsScopedAndNonCanonical(t *testing.T) {
	srv, st := newBackupAuditHarness(t)
	_, err := st.DB.Exec(`INSERT INTO restore_outcome_events(id,created_at,updated_at,workspace_id,lane_id,query,context_packet_id,snapshot_id,snapshot_kind,restore_score,requires_fresh_compile,selected_evidence_json,selected_state_keys_json,selected_loop_ids_json,selected_artifact_ids_json,outcome,outcome_confidence,downstream_action_type,downstream_object_id,metadata_json) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		"restore-outcome-api", 1000, 1000, "ws-api", "control.semantic", "restore blockers", "ctx-api", "snap-api", "restore", 0.55, 0, `["note-a"]`, `[]`, `[]`, `[]`, "unknown", 0, "compile_context", "ctx-api", `{}`)
	if err != nil {
		t.Fatalf("insert outcome: %v", err)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/context/restore/outcomes?workspaceId=ws-api", nil)
	listRR := httptest.NewRecorder()
	srv.handleRestoreOutcomeList(listRR, listReq)
	if listRR.Code != http.StatusOK {
		t.Fatalf("list outcomes status=%d body=%s", listRR.Code, listRR.Body.String())
	}
	var listResp struct {
		Outcomes []map[string]any `json:"outcomes"`
	}
	if err := json.Unmarshal(listRR.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(listResp.Outcomes) != 1 {
		t.Fatalf("expected one outcome, got %+v", listResp.Outcomes)
	}

	wrongReq := withRouteParam(httptest.NewRequest(http.MethodGet, "/api/context/restore/outcomes/restore-outcome-api?workspaceId=ws-other", nil), "id", "restore-outcome-api")
	wrongRR := httptest.NewRecorder()
	srv.handleRestoreOutcomeGet(wrongRR, wrongReq)
	if wrongRR.Code != http.StatusNotFound {
		t.Fatalf("wrong workspace should not read outcome, got status=%d body=%s", wrongRR.Code, wrongRR.Body.String())
	}

	body := []byte(`{"workspaceId":"ws-api","laneId":"control.semantic","outcome":"operator_corrected","outcomeConfidence":0.9,"operatorFeedback":"wrong evidence","correctionSummary":"use newer notes","updatedBy":"operator-test"}`)
	feedbackReq := withRouteParam(httptest.NewRequest(http.MethodPost, "/api/context/restore/outcomes/restore-outcome-api/feedback", bytes.NewReader(body)), "id", "restore-outcome-api")
	feedbackRR := httptest.NewRecorder()
	srv.handleRestoreOutcomeFeedback(feedbackRR, feedbackReq)
	if feedbackRR.Code != http.StatusOK {
		t.Fatalf("feedback status=%d body=%s", feedbackRR.Code, feedbackRR.Body.String())
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
	if outcome != "operator_corrected" || correction != "use newer notes" {
		t.Fatalf("feedback not persisted as non-canonical evidence: outcome=%q correction=%q", outcome, correction)
	}
}
