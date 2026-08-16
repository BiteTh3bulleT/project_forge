package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/forgekernel"
	"forge/projectforge/services/core/internal/forgekernel/semanticdiff"
	"forge/projectforge/services/core/internal/store"
)

func TestCourtAdmissionRouteDerivesAuthorityFieldsFromGovernedRetrievalEvidence(t *testing.T) {
	srv, st := newBackupAuditHarness(t)
	processor := &capturingUtilityProcessor{}
	srv.kernelAuthority = forgekernel.Selection{Processor: processor, SingleAuthority: true}
	srv.kernelAuthorizationReady = true
	resultID := seedGovernedRetrievalEvidenceForAdmission(t, st)
	body := []byte(`{"retrievalResultId":` + jsonNumber(resultID) + `,"workspaceId":"workspace-a","laneId":"lane-a","selectedPaths":["/repo/a.go"]}`)
	req := withRouteParam(httptest.NewRequest(http.MethodPost, "/api/court/cases/case-a/exhibits", bytes.NewReader(body)), "caseId", "case-a")
	req.Header.Set("Idempotency-Key", "admit-retrieval-a")
	rr := httptest.NewRecorder()

	srv.handleAdmitRetrievalEvidence(rr, req)

	if rr.Code != http.StatusCreated || len(processor.requests) != 1 {
		t.Fatalf("status=%d requests=%d body=%s", rr.Code, len(processor.requests), rr.Body.String())
	}
	got := processor.requests[0]
	if got.Action != domain.ActionAdmitEvidence || got.IdempotencyKey != "admit-retrieval-a" || got.Source != domain.SourceUser {
		t.Fatalf("request=%+v", got)
	}
	if got.Payload["caseId"] != "case-a" || got.Payload["sourceType"] != "governed_retrieval_result" || got.Payload["contentHash"] == "" {
		t.Fatalf("derived payload=%#v", got.Payload)
	}
	if _, present := got.Payload["decision"]; present {
		t.Fatalf("route injected decision claim: %#v", got.Payload)
	}
	policy, ok := got.Payload["policyRefs"].([]string)
	if !ok || len(policy) != 1 || policy[0] != retrievalAdmissionPolicy {
		t.Fatalf("policy refs=%#v", got.Payload["policyRefs"])
	}
}

func TestGovernedMemoryAndSemanticRoutesBuildNarrowKernelRequests(t *testing.T) {
	srv, st := newBackupAuditHarness(t)
	processor := &capturingUtilityProcessor{}
	srv.kernelAuthority = forgekernel.Selection{Processor: processor, SingleAuthority: true}
	srv.kernelAuthorizationReady = true
	seedGovernedVSAEvidenceAPI(t, st, "/repo/a.go")

	materialize := withRouteParam(httptest.NewRequest(http.MethodPost, "/api/memory/evidence/exhibit-1/materializations", bytes.NewBufferString(`{"workspaceId":"workspace-a","laneId":"lane-a","selectedPaths":["/repo/a.go"]}`)), "exhibitId", "exhibit-1")
	materialize.Header.Set("Idempotency-Key", "materialize-a")
	materializeRR := httptest.NewRecorder()
	srv.handleMaterializeAdmittedEvidence(materializeRR, materialize)
	if materializeRR.Code != http.StatusCreated || len(processor.requests) != 1 {
		t.Fatalf("materialize status=%d requests=%d body=%s", materializeRR.Code, len(processor.requests), materializeRR.Body.String())
	}
	if got := processor.requests[0]; got.Action != domain.ActionMaterializeAdmittedEvidence || got.Payload["exhibitId"] != "exhibit-1" || got.Payload["rulingId"] != "ruling-1" {
		t.Fatalf("materialize request=%+v", got)
	}

	diff := httptest.NewRequest(http.MethodPost, "/api/memory/evidence/diffs", bytes.NewBufferString(`{"workspaceId":"workspace-a","laneId":"lane-a","selectedPaths":["/repo/a.go"],"leftEvidenceId":"evidence-1","rightEvidenceId":"evidence-2"}`))
	diff.Header.Set("Idempotency-Key", "diff-a")
	diffRR := httptest.NewRecorder()
	srv.handleComputeSemanticDiff(diffRR, diff)
	if diffRR.Code != http.StatusCreated || len(processor.requests) != 2 {
		t.Fatalf("diff status=%d requests=%d body=%s", diffRR.Code, len(processor.requests), diffRR.Body.String())
	}
	if got := processor.requests[1]; got.Action != domain.ActionComputeSemanticDiff || got.Payload["operatorVersion"] != semanticdiff.OperatorVersion {
		t.Fatalf("diff request=%+v", got)
	}
}

func TestGovernedEvidenceRoutesRejectSpoofedFieldsAndMissingIdempotency(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)
	processor := &capturingUtilityProcessor{}
	srv.kernelAuthority = forgekernel.Selection{Processor: processor, SingleAuthority: true}
	srv.kernelAuthorizationReady = true

	missingKey := httptest.NewRequest(http.MethodPost, "/api/memory/evidence/diffs", bytes.NewBufferString(`{"workspaceId":"ws","laneId":"lane","selectedPaths":["/a"],"leftEvidenceId":"a","rightEvidenceId":"b"}`))
	missingRR := httptest.NewRecorder()
	srv.handleComputeSemanticDiff(missingRR, missingKey)
	if missingRR.Code != http.StatusBadRequest {
		t.Fatalf("missing key status=%d body=%s", missingRR.Code, missingRR.Body.String())
	}

	spoof := httptest.NewRequest(http.MethodPost, "/api/memory/evidence/diffs", bytes.NewBufferString(`{"workspaceId":"ws","laneId":"lane","selectedPaths":["/a"],"leftEvidenceId":"a","rightEvidenceId":"b","actor":"root"}`))
	spoof.Header.Set("Idempotency-Key", "spoof")
	spoofRR := httptest.NewRecorder()
	srv.handleComputeSemanticDiff(spoofRR, spoof)
	if spoofRR.Code != http.StatusBadRequest || len(processor.requests) != 0 {
		t.Fatalf("spoof status=%d requests=%d body=%s", spoofRR.Code, len(processor.requests), spoofRR.Body.String())
	}
}

func TestGovernedEvidenceIngressCommitsAdmissionAndMaterializationEndToEnd(t *testing.T) {
	srv, st := newBackupAuditHarness(t)
	resultID := seedGovernedRetrievalEvidenceForAdmission(t, st)
	body := []byte(`{"retrievalResultId":` + jsonNumber(resultID) + `,"workspaceId":"workspace-a","laneId":"lane-a","selectedPaths":["/repo/a.go"]}`)
	admitReq := httptest.NewRequest(http.MethodPost, "/api/court/cases/case-live/exhibits", bytes.NewReader(body))
	admitReq.RemoteAddr = "127.0.0.1:42000"
	admitReq.Header.Set("Idempotency-Key", "admit-live-a")
	admitRR := httptest.NewRecorder()
	srv.Handler().ServeHTTP(admitRR, admitReq)
	if admitRR.Code != http.StatusCreated {
		t.Fatalf("admit status=%d body=%s", admitRR.Code, admitRR.Body.String())
	}
	var admitted struct {
		Result struct {
			StateSummary map[string]any `json:"stateSummary"`
		} `json:"result"`
	}
	if err := json.Unmarshal(admitRR.Body.Bytes(), &admitted); err != nil {
		t.Fatal(err)
	}
	courthouse, _ := admitted.Result.StateSummary["courthouse"].(map[string]any)
	exhibitID, _ := courthouse["exhibitId"].(string)
	if exhibitID == "" {
		t.Fatalf("missing exhibit identity: %s", admitRR.Body.String())
	}

	materializeReq := httptest.NewRequest(http.MethodPost, "/api/memory/evidence/"+exhibitID+"/materializations", bytes.NewBufferString(`{"workspaceId":"workspace-a","laneId":"lane-a","selectedPaths":["/repo/a.go"]}`))
	materializeReq.RemoteAddr = "127.0.0.1:42001"
	materializeReq.Header.Set("Idempotency-Key", "materialize-live-a")
	materializeRR := httptest.NewRecorder()
	srv.Handler().ServeHTTP(materializeRR, materializeReq)
	if materializeRR.Code != http.StatusCreated {
		t.Fatalf("materialize status=%d body=%s", materializeRR.Code, materializeRR.Body.String())
	}
	for table, want := range map[string]int{
		"court_exhibits": 1, "court_rulings": 1, "forge_k_memory_evidence": 1,
		"forge_k_audit_outbox": 2, "semantic_idempotency_keys": 2,
	} {
		var got int
		if err := st.DB.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&got); err != nil || got != want {
			t.Fatalf("table=%s count=%d want=%d err=%v", table, got, want, err)
		}
	}
}

func seedGovernedRetrievalEvidenceForAdmission(t *testing.T, st *store.Store) int64 {
	t.Helper()
	paths, _ := json.Marshal([]string{"/repo/a.go"})
	result, err := st.DB.Exec(`
INSERT INTO retrieval_runs(
 created_at,query,mode,weighting_json,notes,workspace_id,lane_id,selected_paths_json,
 syscall_id,correlation_id,trace_id,provenance_id,provenance_json,proposed_by,committed_by,
 transaction_id,journal_event_id,audit_outbox_id,idempotency_key,authorization_fingerprint
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		10, "forge", "hybrid", `{}`, "", "workspace-a", "lane-a", string(paths),
		"retrieval-syscall-a", "corr-a", "trace-a", "prov-a", `{"actor":"operator"}`, "operator", forgekernel.AuthorityOwnerForgeK,
		"tx-a", "journal-a", "outbox-a", "retrieval-a", "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	)
	if err != nil {
		t.Fatal(err)
	}
	runID, _ := result.LastInsertId()
	result, err = st.DB.Exec(`
INSERT INTO retrieval_results(evidence_id,retrieval_run_id,abs_path,rel_path,rank_index,snippet)
VALUES(?,?,?,?,?,?)`, "retrieval-result-evidence-a", runID, "/repo/a.go", "a.go", 0, "deterministic retrieval evidence")
	if err != nil {
		t.Fatal(err)
	}
	id, _ := result.LastInsertId()
	return id
}

func jsonNumber(value int64) string {
	raw, _ := json.Marshal(value)
	return string(raw)
}
