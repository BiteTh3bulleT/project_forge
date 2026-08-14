package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"forge/projectforge/services/core/internal/forgekernel"
	"forge/projectforge/services/core/internal/memory"
	"forge/projectforge/services/core/internal/store"
)

func TestHandleGetObservationVSAExcludesLegacyUnscopedProjection(t *testing.T) {
	t.Parallel()

	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	now := time.Now().UnixMilli()
	obsID := seedObservation(t, st, "/repo/obs-vsa.go", now)
	if _, err := st.DB.Exec(`
INSERT INTO memory_vsa_pointers(
  observation_id, dims, pointer_json, norm, source_fingerprint, stale, metadata_json, created_at, updated_at
) VALUES(?,?,?,?,?,?,?,?,?)`,
		obsID, 8, `[0.5,0.5,0,0,0,0,0,0]`, 0.7071, "obs-fp", 0, `{"seeded":true}`, now, now,
	); err != nil {
		t.Fatalf("insert vsa pointer: %v", err)
	}

	s := &Server{st: st, memory: memory.New(st.DB)}
	req := withRouteParam(httptest.NewRequest(http.MethodGet, "/api/memory/observations/"+strconv.FormatInt(obsID, 10)+"/vsa", nil), "id", strconv.FormatInt(obsID, 10))
	rr := httptest.NewRecorder()
	s.handleGetObservationVSA(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status code=%d want=%d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}

	var payload struct {
		Detail memory.ObservationVSADetail `json:"detail"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.Detail.ObservationID != obsID {
		t.Fatalf("observation id=%d want=%d", payload.Detail.ObservationID, obsID)
	}
	if payload.Detail.Pointer != nil {
		t.Fatalf("legacy unscoped pointer leaked into detail: %+v", payload.Detail.Pointer)
	}
}

func TestHandleVSARebuildRoutesScopedManifestThroughForgeK(t *testing.T) {
	srv, st := newBackupAuditHarness(t)
	processor := &capturingUtilityProcessor{}
	srv.kernelAuthority = forgekernel.Selection{Processor: processor}
	srv.kernelAuthorizationReady = true
	if _, err := st.DB.Exec(`
INSERT INTO memory_observations(workspace_id,lane_id,created_at,updated_at,observed_at,type,raw_content,summary,source_path)
VALUES('workspace-a','lane-a',1,1,1,'file','alpha body','alpha summary','/repo/a.go')`); err != nil {
		t.Fatal(err)
	}
	dryBody := []byte(`{"dryRun":true,"workspaceId":"workspace-a","laneId":"lane-a","dimensions":128,"seed":17}`)
	dryRR := httptest.NewRecorder()
	srv.handleRunVSAReindex(dryRR, httptest.NewRequest(http.MethodPost, "/api/memory/vsa/reindex/run", bytes.NewReader(dryBody)))
	if dryRR.Code != http.StatusOK {
		t.Fatalf("dry proposal status=%d body=%s", dryRR.Code, dryRR.Body.String())
	}
	var proposal struct {
		Manifest struct {
			ManifestHash string `json:"manifestHash"`
		} `json:"manifest"`
		ExpectedPrior string `json:"expectedPriorManifestHash"`
	}
	if err := json.Unmarshal(dryRR.Body.Bytes(), &proposal); err != nil || proposal.Manifest.ManifestHash == "" {
		t.Fatalf("proposal = %+v err=%v", proposal, err)
	}
	commitBody, _ := json.Marshal(map[string]any{
		"dryRun": false, "workspaceId": "workspace-a", "laneId": "lane-a",
		"dimensions": 128, "seed": 17, "idempotencyKey": "vsa-rebuild-1",
		"expectedManifestHash":      proposal.Manifest.ManifestHash,
		"expectedPriorManifestHash": proposal.ExpectedPrior,
	})
	commitRR := httptest.NewRecorder()
	srv.handleRunVSAReindex(commitRR, httptest.NewRequest(http.MethodPost, "/api/memory/vsa/reindex/run", bytes.NewReader(commitBody)))
	if commitRR.Code != http.StatusOK {
		t.Fatalf("commit status=%d body=%s", commitRR.Code, commitRR.Body.String())
	}
	if len(processor.requests) != 1 {
		t.Fatalf("syscall count=%d", len(processor.requests))
	}
	req := processor.requests[0]
	if req.Action != "REBUILD_MEMORY_ACCELERATION" || req.Scope.WorkspaceID != "workspace-a" || req.Scope.LaneID != "lane-a" || req.IdempotencyKey != "vsa-rebuild-1" {
		t.Fatalf("governed request = %+v", req)
	}
	if req.Payload["requestedAtMs"] != req.RequestedAt || req.Payload["expectedManifestHash"] != proposal.Manifest.ManifestHash {
		t.Fatalf("unsealed rebuild payload = %+v requestedAt=%d", req.Payload, req.RequestedAt)
	}
}

func TestHandleRetrievalVSASignalEndpoints(t *testing.T) {
	t.Parallel()

	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	now := time.Now().UnixMilli()
	sourceID := seedSource(t, st, "/repo")
	fileID := seedFile(t, st, sourceID, "doc.go", "/repo/doc.go", now)
	chunkID := seedChunk(t, st, fileID, "vsa retrieval body")
	runID := seedRetrievalRun(t, st, now)
	resultID := seedRetrievalResult(t, st, runID, chunkID, fileID, "/repo/doc.go", "doc.go")
	if _, err := st.DB.Exec(`
INSERT INTO retrieval_result_vsa_signals(
  retrieval_result_id, retrieval_run_id, observation_id, mode,
  associative_score, role_match_score, relational_score, feedback_score,
  additive_score, applied_score, explain_json, created_at
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`,
		resultID, runID, nil, "shadow", 0.8, 0.1, 0.0, 0.0, 0.9, 0.0, `{"seeded":true}`, now,
	); err != nil {
		t.Fatalf("insert retrieval vsa signal: %v", err)
	}

	s := &Server{st: st, memory: memory.New(st.DB)}

	{
		req := withRouteParam(httptest.NewRequest(http.MethodGet, "/api/retrieval/runs/"+strconv.FormatInt(runID, 10)+"/vsa-signals", nil), "id", strconv.FormatInt(runID, 10))
		rr := httptest.NewRecorder()
		s.handleGetRetrievalRunVSASignals(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("run signals status code=%d want=%d body=%s", rr.Code, http.StatusOK, rr.Body.String())
		}
		var payload struct {
			Signals []memory.RetrievalResultVSASignal `json:"signals"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode run signals: %v", err)
		}
		if len(payload.Signals) != 1 {
			t.Fatalf("signals=%d want=1", len(payload.Signals))
		}
		if payload.Signals[0].RetrievalResultID != resultID {
			t.Fatalf("result id=%d want=%d", payload.Signals[0].RetrievalResultID, resultID)
		}
	}

	{
		req := withRouteParam(httptest.NewRequest(http.MethodGet, "/api/retrieval/results/"+strconv.FormatInt(resultID, 10)+"/vsa-signal", nil), "id", strconv.FormatInt(resultID, 10))
		rr := httptest.NewRecorder()
		s.handleGetRetrievalResultVSASignal(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("result signal status code=%d want=%d body=%s", rr.Code, http.StatusOK, rr.Body.String())
		}
		var payload struct {
			Signal memory.RetrievalResultVSASignal `json:"signal"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode result signal: %v", err)
		}
		if payload.Signal.RetrievalResultID != resultID {
			t.Fatalf("result id=%d want=%d", payload.Signal.RetrievalResultID, resultID)
		}
		if payload.Signal.Mode != "shadow" {
			t.Fatalf("mode=%q want=shadow", payload.Signal.Mode)
		}
	}
}

func TestHandleVSAReindexEndpoints(t *testing.T) {
	t.Parallel()

	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	now := time.Now().UnixMilli()
	_ = seedObservation(t, st, "/repo/reindex.go", now)
	setAPISetting(t, st, "retrieval_vsa_mode", "active")
	setAPISetting(t, st, "retrieval_vsa_dims", "16")
	setAPISetting(t, st, "retrieval_vsa_seed", "17")

	s := &Server{st: st, memory: memory.New(st.DB)}

	{
		body := map[string]any{
			"mode":        "manual",
			"limit":       10,
			"triggeredBy": "test",
			"reason":      "api-test",
			"note":        "vsa reindex from api test",
			"dryRun":      true,
		}
		raw, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/api/memory/vsa/reindex/run", bytes.NewReader(raw))
		rr := httptest.NewRecorder()
		s.handleRunVSAReindex(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("run reindex status=%d want=%d body=%s", rr.Code, http.StatusOK, rr.Body.String())
		}
		var payload struct {
			Report memory.MaintenancePreview `json:"report"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode run reindex payload: %v", err)
		}
		if !payload.Report.DryRun || !payload.Report.ProposalOnly || payload.Report.Candidates != 1 {
			t.Fatalf("unexpected proposal-only preview: %+v", payload.Report)
		}
	}

	{
		req := httptest.NewRequest(http.MethodGet, "/api/memory/vsa/reindex-runs?limit=5", nil)
		rr := httptest.NewRecorder()
		s.handleListVSAReindexRuns(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("list reindex runs status=%d want=%d body=%s", rr.Code, http.StatusOK, rr.Body.String())
		}
		var payload struct {
			Runs []memory.VSAReindexRun `json:"runs"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode list runs payload: %v", err)
		}
		if len(payload.Runs) != 0 {
			t.Fatalf("dry-run must not create reindex runs: %+v", payload.Runs)
		}
	}

	for _, tc := range []struct {
		raw  string
		want int
	}{
		{raw: `{"limit":10}`, want: http.StatusConflict},
		{raw: `{"limit":10,"dryRun":false}`, want: http.StatusBadRequest},
	} {
		req := httptest.NewRequest(http.MethodPost, "/api/memory/vsa/reindex/run", bytes.NewReader([]byte(tc.raw)))
		rr := httptest.NewRecorder()
		s.handleRunVSAReindex(rr, req)
		if rr.Code != tc.want {
			t.Fatalf("non-dry reindex status=%d want=%d body=%s", rr.Code, tc.want, rr.Body.String())
		}
	}

	for _, table := range []string{"memory_vsa_pointers", "memory_vsa_reindex_runs", "memory_vsa_reindex_items"} {
		var count int
		if err := st.DB.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if count != 0 {
			t.Fatalf("proposal-only reindex mutated %s: count=%d", table, count)
		}
	}
}

func TestHandleGetDossierVSASummary(t *testing.T) {
	t.Parallel()

	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	now := time.Now().UnixMilli()
	dossierID := seedDossier(t, st, "vsa-summary-test", now)
	obsID := seedDossierObservation(t, st, dossierID, "/repo/dossier-vsa.go", now)
	if _, err := st.DB.Exec(`
INSERT INTO memory_vsa_pointers(
  observation_id, dims, pointer_json, norm, source_fingerprint, stale, metadata_json, created_at, updated_at
) VALUES(?,?,?,?,?,?,?,?,?)`,
		obsID, 8, `[0.5,0.5,0,0,0,0,0,0]`, 0.7071, "dossier-fp", 0, `{}`, now, now,
	); err != nil {
		t.Fatalf("insert dossier pointer: %v", err)
	}

	s := &Server{st: st, memory: memory.New(st.DB)}
	req := withRouteParam(
		httptest.NewRequest(http.MethodGet, "/api/memory/dossiers/"+strconv.FormatInt(dossierID, 10)+"/vsa-summary", nil),
		"id",
		strconv.FormatInt(dossierID, 10),
	)
	rr := httptest.NewRecorder()
	s.handleGetDossierVSASummary(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status code=%d want=%d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	var payload struct {
		Summary memory.DossierVSASummary `json:"summary"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.Summary.DossierID != dossierID {
		t.Fatalf("dossier id=%d want=%d", payload.Summary.DossierID, dossierID)
	}
	if payload.Summary.PointerCount != 0 || payload.Summary.Health != "unindexed" {
		t.Fatalf("legacy pointer influenced governed summary: %+v", payload.Summary)
	}
}

func setAPISetting(t *testing.T, st *store.Store, key, value string) {
	t.Helper()
	if _, err := st.DB.Exec(`INSERT INTO settings(key, value) VALUES(?, ?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, key, value); err != nil {
		t.Fatalf("set setting %s: %v", key, err)
	}
}

func seedObservation(t *testing.T, st *store.Store, sourcePath string, now int64) int64 {
	t.Helper()
	res, err := st.DB.Exec(`
INSERT INTO memory_observations(
  created_at, updated_at, observed_at, type, raw_content, summary, source_path, confidence, verification_state
) VALUES(?,?,?,?,?,?,?,?,?)`,
		now, now, now, "retrieval_result", sourcePath, sourcePath, sourcePath, 0.9, "checked",
	)
	if err != nil {
		t.Fatalf("insert observation: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
}

func seedDossier(t *testing.T, st *store.Store, name string, now int64) int64 {
	t.Helper()
	res, err := st.DB.Exec(`
INSERT INTO dossiers(
  created_at, updated_at, name, description, primary_paths_json, related_repos_json,
  constraints_json, preferred_adapters_json, important_files_json, routing_notes
) VALUES(?,?,?,?,?,?,?,?,?,?)`,
		now, now, name, "", "[]", "[]", "[]", "[]", "[]", "",
	)
	if err != nil {
		t.Fatalf("insert dossier: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
}

func seedDossierObservation(t *testing.T, st *store.Store, dossierID int64, sourcePath string, now int64) int64 {
	t.Helper()
	res, err := st.DB.Exec(`
INSERT INTO memory_observations(
  created_at, updated_at, observed_at, type, raw_content, summary, dossier_id, source_path, confidence, verification_state
) VALUES(?,?,?,?,?,?,?,?,?,?)`,
		now, now, now, "retrieval_result", sourcePath, sourcePath, dossierID, sourcePath, 0.9, "checked",
	)
	if err != nil {
		t.Fatalf("insert dossier observation: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
}

func seedSource(t *testing.T, st *store.Store, path string) int64 {
	t.Helper()
	now := time.Now().UnixMilli()
	res, err := st.DB.Exec(`INSERT INTO sources(path, created_at) VALUES(?, ?)`, path, now)
	if err != nil {
		t.Fatalf("insert source: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
}

func seedFile(t *testing.T, st *store.Store, sourceID int64, relPath, absPath string, now int64) int64 {
	t.Helper()
	res, err := st.DB.Exec(`
INSERT INTO files(source_id, rel_path, abs_path, size_bytes, mtime_ns, content_sha256, indexed_at)
VALUES(?,?,?,?,?,?,?)`,
		sourceID, relPath, absPath, 100, now*1_000_000, "sha-"+relPath, now,
	)
	if err != nil {
		t.Fatalf("insert file: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
}

func seedChunk(t *testing.T, st *store.Store, fileID int64, content string) int64 {
	t.Helper()
	res, err := st.DB.Exec(`INSERT INTO chunks(file_id, chunk_index, content) VALUES(?,?,?)`, fileID, 0, content)
	if err != nil {
		t.Fatalf("insert chunk: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
}

func seedRetrievalRun(t *testing.T, st *store.Store, now int64) int64 {
	t.Helper()
	res, err := st.DB.Exec(`
INSERT INTO retrieval_runs(created_at, query, mode, weighting_json, notes)
VALUES(?,?,?,?,?)`,
		now, "vsa query", "keyword", `{"mode":"keyword"}`, "",
	)
	if err != nil {
		t.Fatalf("insert retrieval run: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
}

func seedRetrievalResult(t *testing.T, st *store.Store, runID, chunkID, fileID int64, absPath, relPath string) int64 {
	t.Helper()
	res, err := st.DB.Exec(`
INSERT INTO retrieval_results(
  retrieval_run_id, chunk_id, file_id, abs_path, rel_path, rank_index,
  keyword_score, semantic_score, hybrid_score, snippet, selected_for_packet, usefulness_label, usefulness_note
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		runID, chunkID, fileID, absPath, relPath, 0,
		1.0, 0.0, 1.0, "snippet", 1, "unknown", "",
	)
	if err != nil {
		t.Fatalf("insert retrieval result: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
}
