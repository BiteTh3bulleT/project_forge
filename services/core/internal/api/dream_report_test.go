package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"forge/projectforge/services/core/internal/config"
	"forge/projectforge/services/core/internal/store"
)

func TestDreamRunReportPersistenceAndScopedReadAPI(t *testing.T) {
	dataDir := t.TempDir()
	workspaceDir := t.TempDir()
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	seedDreamAPIFixture(t, st)

	srv := NewServer(st, config.Config{DataDir: dataDir, WorkspaceDir: workspaceDir})
	t.Cleanup(func() { srv.ShutdownWatch() })

	before := countRows(t, st, "memory_notes")
	body := []byte(`{"workspaceId":"ws-dream","laneId":"control.semantic","mode":"nap","persistReport":false,"correlationId":"corr-api","traceId":"trace-api"}`)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/dream/run", bytes.NewReader(body)))
	if rr.Code != http.StatusOK {
		t.Fatalf("dream run status=%d body=%s", rr.Code, rr.Body.String())
	}
	if got := countRows(t, st, "dream_reports"); got != 0 {
		t.Fatalf("persistReport=false wrote dream_reports row count=%d", got)
	}
	if got := countRows(t, st, "memory_notes"); got != before {
		t.Fatalf("dry-run mutated canonical memory_notes: before=%d after=%d", before, got)
	}

	body = []byte(`{"workspaceId":"ws-dream","laneId":"control.semantic","mode":"nap","persistReport":true,"correlationId":"corr-api","traceId":"trace-api","metadata":{"operatorReview":"pending"}}`)
	rr = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/dream/run", bytes.NewReader(body)))
	if rr.Code != http.StatusOK {
		t.Fatalf("dream persisted run status=%d body=%s", rr.Code, rr.Body.String())
	}
	var runResp struct {
		Persisted               bool   `json:"persisted"`
		ReportID                string `json:"reportId"`
		NonCanonicalEvidence    bool   `json:"nonCanonicalEvidence"`
		CanonicalWriteCommitted bool   `json:"canonicalWriteCommitted"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&runResp); err != nil {
		t.Fatalf("decode run response: %v", err)
	}
	if !runResp.Persisted || runResp.ReportID == "" || !runResp.NonCanonicalEvidence || runResp.CanonicalWriteCommitted {
		t.Fatalf("unexpected persisted run response: %+v", runResp)
	}
	if got := countRows(t, st, "dream_reports"); got != 1 {
		t.Fatalf("persistReport=true row count=%d want 1", got)
	}
	if got := countRows(t, st, "memory_notes"); got != before {
		t.Fatalf("persisting report mutated canonical memory_notes: before=%d after=%d", before, got)
	}

	getURL := "/api/dream/reports/" + runResp.ReportID + "?workspaceId=ws-dream&laneId=control.semantic"
	rr = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, getURL, nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("get report status=%d body=%s", rr.Code, rr.Body.String())
	}
	var report struct {
		ID                      string            `json:"id"`
		WorkspaceID             string            `json:"workspaceId"`
		LaneID                  string            `json:"laneId"`
		EvidenceClass           string            `json:"evidenceClass"`
		Candidates              []json.RawMessage `json:"candidates"`
		SalienceScores          []json.RawMessage `json:"salienceScores"`
		MemoryTierProposals     []json.RawMessage `json:"memoryTierProposals"`
		Trace                   json.RawMessage   `json:"trace"`
		NonCanonicalEvidence    bool              `json:"nonCanonicalEvidence"`
		CanonicalWriteCommitted bool              `json:"canonicalWriteCommitted"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&report); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	if report.ID != runResp.ReportID || report.WorkspaceID != "ws-dream" || report.LaneID != "control.semantic" {
		t.Fatalf("unexpected report scope: %+v", report)
	}
	if len(report.Candidates) == 0 || len(report.SalienceScores) == 0 || len(report.MemoryTierProposals) == 0 || len(report.Trace) == 0 {
		t.Fatalf("report missing inspectability fields: %+v", report)
	}
	if report.EvidenceClass != "non_canonical_evidence" || !report.NonCanonicalEvidence || report.CanonicalWriteCommitted {
		t.Fatalf("report authority flags wrong: %+v", report)
	}

	rr = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/dream/reports/"+runResp.ReportID+"/candidates?workspaceId=ws-dream&laneId=control.semantic", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("candidates status=%d body=%s", rr.Code, rr.Body.String())
	}
	var candidatesResp struct {
		Candidates              []json.RawMessage `json:"candidates"`
		SalienceScores          []json.RawMessage `json:"salienceScores"`
		EvidenceClass           string            `json:"evidenceClass"`
		NonCanonicalEvidence    bool              `json:"nonCanonicalEvidence"`
		DryRun                  bool              `json:"dryRun"`
		CanonicalWriteCommitted bool              `json:"canonicalWriteCommitted"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&candidatesResp); err != nil {
		t.Fatalf("decode candidates: %v", err)
	}
	if len(candidatesResp.Candidates) == 0 || len(candidatesResp.SalienceScores) == 0 || candidatesResp.EvidenceClass != "non_canonical_evidence" || !candidatesResp.NonCanonicalEvidence || !candidatesResp.DryRun || candidatesResp.CanonicalWriteCommitted {
		t.Fatalf("unexpected candidates response: %+v", candidatesResp)
	}

	rr = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/dream/reports/"+runResp.ReportID+"/proposals?workspaceId=ws-dream&laneId=control.semantic", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("proposals status=%d body=%s", rr.Code, rr.Body.String())
	}
	var proposalsResp struct {
		MemoryTierProposals      []json.RawMessage `json:"memoryTierProposals"`
		RepairProposals          []json.RawMessage `json:"repairProposals"`
		SnapshotHygieneProposals []json.RawMessage `json:"snapshotHygieneProposals"`
		ReviewItems              []json.RawMessage `json:"reviewItems"`
		EvidenceClass            string            `json:"evidenceClass"`
		NonCanonicalEvidence     bool              `json:"nonCanonicalEvidence"`
		CanonicalWriteCommitted  bool              `json:"canonicalWriteCommitted"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&proposalsResp); err != nil {
		t.Fatalf("decode proposals: %v", err)
	}
	if len(proposalsResp.MemoryTierProposals) == 0 || proposalsResp.EvidenceClass != "non_canonical_evidence" || !proposalsResp.NonCanonicalEvidence || proposalsResp.CanonicalWriteCommitted {
		t.Fatalf("unexpected proposals response: %+v", proposalsResp)
	}

	rr = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/dream/reports/"+runResp.ReportID+"/warnings?workspaceId=ws-dream&laneId=control.semantic", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("warnings status=%d body=%s", rr.Code, rr.Body.String())
	}
	var warningsResp struct {
		Warnings             []string          `json:"warnings"`
		ReviewItems          []json.RawMessage `json:"reviewItems"`
		EvidenceClass        string            `json:"evidenceClass"`
		NonCanonicalEvidence bool              `json:"nonCanonicalEvidence"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&warningsResp); err != nil {
		t.Fatalf("decode warnings: %v", err)
	}
	if warningsResp.EvidenceClass != "non_canonical_evidence" || !warningsResp.NonCanonicalEvidence {
		t.Fatalf("unexpected warnings response: %+v", warningsResp)
	}

	rr = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/dream/reports?workspaceId=ws-dream&laneId=control.semantic&mode=nap", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("list report status=%d body=%s", rr.Code, rr.Body.String())
	}
	var listResp struct {
		Reports []struct {
			ID string `json:"id"`
		} `json:"reports"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&listResp); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(listResp.Reports) != 1 || listResp.Reports[0].ID != runResp.ReportID {
		t.Fatalf("unexpected list response: %+v", listResp)
	}

	rr = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/dream/reports/"+runResp.ReportID+"?workspaceId=ws-other", nil))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("wrong workspace should not return report, status=%d body=%s", rr.Code, rr.Body.String())
	}

	rr = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/dream/reports/missing?workspaceId=ws-dream", nil))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("missing report status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func seedDreamAPIFixture(t *testing.T, st *store.Store) {
	t.Helper()
	now := time.Now().UnixMilli()
	if _, err := st.DB.Exec(`INSERT INTO journal_events(id,type,source,actor,workspace_id,lane_id,payload_json,created_at,metadata_json) VALUES(?,?,?,?,?,?,?,?,?)`,
		"evt-api", "user_correction", "operator", "user", "ws-dream", "control.semantic", `{"text":"corrected dream api preference"}`, now-1000, `{}`); err != nil {
		t.Fatalf("seed journal: %v", err)
	}
	if _, err := st.DB.Exec(`INSERT INTO memory_notes(id,type,title,content,workspace_id,lane_id,confidence,status,created_at,updated_at,metadata_json) VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
		"note-api", "fact", "Dream API", "persist report evidence only", "ws-dream", "control.semantic", 0.9, "active", now-2000, now-2000, `{}`); err != nil {
		t.Fatalf("seed memory note: %v", err)
	}
}

func countRows(t *testing.T, st *store.Store, table string) int {
	t.Helper()
	var count int
	if err := st.DB.QueryRow("SELECT COUNT(1) FROM " + table).Scan(&count); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return count
}
