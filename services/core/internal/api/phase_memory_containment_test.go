package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"forge/projectforge/services/core/internal/memory"
	"forge/projectforge/services/core/internal/store"
)

func TestMemoryRepairEndpointIsExplicitProposalOnly(t *testing.T) {
	t.Parallel()

	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	obsID := seedObservation(t, st, "/repo/repair-preview.go", time.Now().UnixMilli())
	if _, err := st.DB.Exec(`UPDATE memory_observations SET stale = 1 WHERE id = ?`, obsID); err != nil {
		t.Fatalf("mark observation stale: %v", err)
	}
	s := &Server{st: st, memory: memory.New(st.DB)}

	for _, body := range []string{"", `{"dryRun":false}`} {
		req := httptest.NewRequest(http.MethodPost, "/api/memory/repair/run", bytes.NewBufferString(body))
		rr := httptest.NewRecorder()
		s.handleRunMemoryRepair(rr, req)
		if rr.Code != http.StatusConflict {
			t.Fatalf("non-dry repair status=%d want=%d body=%s", rr.Code, http.StatusConflict, rr.Body.String())
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/api/memory/repair/run", bytes.NewBufferString(`{"dryRun":true,"limit":10}`))
	rr := httptest.NewRecorder()
	s.handleRunMemoryRepair(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("repair preview status=%d want=%d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	var payload struct {
		Report memory.MaintenancePreview `json:"report"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode repair preview: %v", err)
	}
	if !payload.Report.DryRun || !payload.Report.ProposalOnly || payload.Report.Candidates != 1 || payload.Report.CandidateIDs[0] != obsID {
		t.Fatalf("unexpected repair preview: %+v", payload.Report)
	}

	var repairRuns int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM memory_repair_runs`).Scan(&repairRuns); err != nil {
		t.Fatalf("count repair runs: %v", err)
	}
	if repairRuns != 0 {
		t.Fatalf("proposal-only repair created %d run rows", repairRuns)
	}
	var stale int
	if err := st.DB.QueryRow(`SELECT stale FROM memory_observations WHERE id = ?`, obsID).Scan(&stale); err != nil {
		t.Fatalf("read observation: %v", err)
	}
	if stale != 1 {
		t.Fatalf("proposal-only repair mutated historical observation stale=%d", stale)
	}
}
