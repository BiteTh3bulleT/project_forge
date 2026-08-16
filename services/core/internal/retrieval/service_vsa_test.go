package retrieval

import (
	"context"
	"database/sql"
	"encoding/json"
	"math"
	"strconv"
	"testing"
	"time"

	"forge/projectforge/services/core/internal/aios/controllane"
	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/embeddings"
	"forge/projectforge/services/core/internal/forgekernel"
	"forge/projectforge/services/core/internal/memory"
	"forge/projectforge/services/core/internal/search"
	"forge/projectforge/services/core/internal/store"
)

func TestRunCommitsRetrievalEvidenceWithoutDirectVSAWrites(t *testing.T) {
	t.Parallel()

	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	searchSvc := search.New(st.DB)
	embedSvc := embeddings.New(st.DB)
	memorySvc := memory.New(st.DB)
	svc := New(st.DB, searchSvc, embedSvc, memorySvc)
	installTestForgeKAuthority(t, svc, st.DB)

	sourceID := seedSource(t, st.DB, "/repo")
	chunkA, absA, relA := seedChunk(t, st.DB, sourceID, "a.go", "/repo/a.go", "vsa query alpha")
	_, absB, _ := seedChunk(t, st.DB, sourceID, "b.go", "/repo/b.go", "vsa query beta")

	setSetting(t, st.DB, "retrieval_vsa_dims", "16")
	setSetting(t, st.DB, "retrieval_vsa_seed", "17")
	setSetting(t, st.DB, "retrieval_vsa_weight_associative", "1")
	setSetting(t, st.DB, "retrieval_vsa_weight_role_match", "0")
	setSetting(t, st.DB, "retrieval_vsa_weight_relational", "0")
	setSetting(t, st.DB, "retrieval_vsa_weight_feedback", "0")
	setSetting(t, st.DB, "retrieval_vsa_max_additive", "0.05")

	engine := memory.NewVSAEngine(16, 17)
	seedObservationWithPointer(t, st.DB, absA, engine.EncodeText("vsa query"), 16)
	seedObservationWithPointer(t, st.DB, absB, engine.EncodeText("irrelevant context"), 16)

	ctx := context.Background()
	baseReq := RunRequest{
		Query:           "vsa query",
		Mode:            ModeKeyword,
		Limit:           2,
		SelectForPacket: 2,
		Actor:           domain.ActorIdentity{ID: "forge.core", Kind: "service"},
		Source:          domain.SourceInternal,
		Scope:           domain.ForgeScope{WorkspaceID: "/repo", LaneID: "control.semantic"},
		Provenance:      domain.Provenance{Actor: "forge.core", ActorType: "system", Source: "test.retrieval"},
		CorrelationID:   "corr-retrieval-vsa",
		TraceID:         "trace-retrieval-vsa",
		RequestedAt:     1760000000000,
	}

	runs := make([]*Run, 0, 3)
	for index, configuredMode := range []string{"off", "shadow", "active"} {
		setSetting(t, st.DB, "retrieval_vsa_mode", configuredMode)
		req := baseReq
		req.RequestID = "retrieval-k20g-" + configuredMode
		req.IdempotencyKey = req.RequestID
		req.RequestedAt += int64(index)
		run, err := svc.Run(ctx, req)
		if err != nil {
			t.Fatalf("run with legacy VSA mode %s: %v", configuredMode, err)
		}
		if len(run.Results) != 2 {
			t.Fatalf("mode %s results=%d, want 2", configuredMode, len(run.Results))
		}
		for _, result := range run.Results {
			assertVSAInfluenceDisabled(t, result.SelectionReason)
		}
		runs = append(runs, run)
	}

	var observationCount int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM memory_observations`).Scan(&observationCount); err != nil {
		t.Fatalf("observation count: %v", err)
	}
	if observationCount != 2 {
		t.Fatalf("retrieval must not duplicate result evidence into memory_observations: got %d want seeded 2", observationCount)
	}
	var resultObservationLinks int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM retrieval_result_observations`).Scan(&resultObservationLinks); err != nil {
		t.Fatalf("result observation link count: %v", err)
	}
	if resultObservationLinks != 0 {
		t.Fatalf("retrieval must not create legacy result-observation links: got %d", resultObservationLinks)
	}
	var selectionCount int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM retrieval_result_selection`).Scan(&selectionCount); err != nil {
		t.Fatalf("selection evidence count: %v", err)
	}
	wantSelectionCount := 0
	for _, run := range runs {
		wantSelectionCount += len(run.Results)
	}
	if selectionCount != wantSelectionCount {
		t.Fatalf("selection evidence count=%d want=%d", selectionCount, wantSelectionCount)
	}
	var signalCount int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM retrieval_result_vsa_signals`).Scan(&signalCount); err != nil {
		t.Fatalf("signal count: %v", err)
	}
	if signalCount != 0 {
		t.Fatalf("retrieval must not persist legacy VSA signals: got %d", signalCount)
	}

	if chunkA <= 0 || relA == "" {
		t.Fatalf("seed invariant failed")
	}
}

func TestPathUsefulnessScoresIgnoreLegacyLabelsAndUseGovernedProjection(t *testing.T) {
	t.Parallel()
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()
	svc := New(st.DB, search.New(st.DB), embeddings.New(st.DB), memory.New(st.DB))

	result, err := st.DB.Exec(`INSERT INTO retrieval_runs(evidence_id,created_at,query,mode,workspace_id,lane_id,selected_paths_json,syscall_id,provenance_id,provenance_json,proposed_by,committed_by) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`,
		"run-evidence-usefulness", 100, "utility", "keyword", "ws-utility", "control.semantic", `["/repo"]`, "source-syscall", "source-prov", `{"actor":"forge.core"}`, "internal", "forge_k.kernel")
	if err != nil {
		t.Fatalf("insert retrieval run: %v", err)
	}
	runID, _ := result.LastInsertId()
	legacyResult, err := st.DB.Exec(`INSERT INTO retrieval_results(evidence_id,retrieval_run_id,abs_path,rel_path,rank_index,usefulness_label) VALUES(?,?,?,?,?,?)`,
		"result-evidence-legacy-label", runID, "/repo/legacy.go", "legacy.go", 0, "useful")
	if err != nil {
		t.Fatalf("insert legacy-labeled result: %v", err)
	}
	if _, err := legacyResult.LastInsertId(); err != nil {
		t.Fatal(err)
	}
	governedResult, err := st.DB.Exec(`INSERT INTO retrieval_results(evidence_id,retrieval_run_id,abs_path,rel_path,rank_index,usefulness_label) VALUES(?,?,?,?,?,?)`,
		"result-evidence-governed", runID, "/repo/governed.go", "governed.go", 1, "unknown")
	if err != nil {
		t.Fatalf("insert governed result: %v", err)
	}
	governedID, _ := governedResult.LastInsertId()
	otherRun, err := st.DB.Exec(`INSERT INTO retrieval_runs(evidence_id,created_at,query,mode,workspace_id,lane_id,selected_paths_json,syscall_id,provenance_id,provenance_json,proposed_by,committed_by) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`,
		"run-evidence-other-workspace", 100, "utility", "keyword", "ws-other", "control.semantic", `["/repo"]`, "source-syscall-other", "source-prov-other", `{"actor":"forge.core"}`, "internal", "forge_k.kernel")
	if err != nil {
		t.Fatalf("insert other-workspace run: %v", err)
	}
	otherRunID, _ := otherRun.LastInsertId()
	if _, err := st.DB.Exec(`INSERT INTO retrieval_results(evidence_id,retrieval_run_id,abs_path,rel_path,rank_index,usefulness_label) VALUES(?,?,?,?,?,?)`,
		"result-evidence-other-workspace", otherRunID, "/repo/governed.go", "governed.go", 0, "useful"); err != nil {
		t.Fatalf("insert other-workspace result: %v", err)
	}
	if _, err := st.DB.Exec(`INSERT INTO provenance_records(id,actor,actor_type,source,trace_id,workspace_id,lane_id,selected_paths_json,metadata_json,created_at,proposed_by,committed_by,syscall_id,correlation_id) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		"utility-provenance", "forge.core", "system", "test", "trace-utility", "ws-utility", "control.semantic", `["/repo"]`, `{}`, 101, "internal", "forge_k.kernel", "utility-syscall", "corr-utility"); err != nil {
		t.Fatalf("insert utility provenance: %v", err)
	}
	if _, err := st.DB.Exec(`INSERT INTO forge_k_retrieval_usefulness_events(id,created_at,retrieval_result_id,retrieval_run_id,target_evidence_id,workspace_id,lane_id,selected_paths_json,label,note,prior_projection_json,source_provenance_json,metadata_json,correlation_id,trace_id,syscall_id,provenance_id,provenance_json,proposed_by,committed_by) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		"utility-event", 101, governedID, runID, "result-evidence-governed", "ws-utility", "control.semantic", `["/repo"]`, "useful", "governed", `{}`, `{"actor":"forge.core"}`, `{}`, "corr-utility", "trace-utility", "utility-syscall", "utility-provenance", `{}`, "internal", "forge_k.kernel"); err != nil {
		t.Fatalf("insert governed event: %v", err)
	}
	if _, err := st.DB.Exec(`INSERT INTO retrieval_usefulness_projection(retrieval_result_id,latest_event_id,label,note,updated_at,noncanonical) VALUES(?,?,?,?,?,1)`,
		governedID, "utility-event", "useful", "governed", 101); err != nil {
		t.Fatalf("insert usefulness projection: %v", err)
	}

	scores, err := svc.pathUsefulnessScores(context.Background(), domain.ForgeScope{WorkspaceID: "ws-utility", LaneID: "control.semantic"})
	if err != nil {
		t.Fatal(err)
	}
	if scores["/repo/legacy.go"] != 0 {
		t.Fatalf("legacy mutable label influenced score: %v", scores)
	}
	if scores["/repo/governed.go"] != 0.12 {
		t.Fatalf("governed projection did not influence score: %v", scores)
	}
	otherScores, err := svc.pathUsefulnessScores(context.Background(), domain.ForgeScope{WorkspaceID: "ws-other", LaneID: "control.semantic"})
	if err != nil {
		t.Fatal(err)
	}
	if otherScores["/repo/governed.go"] != 0 {
		t.Fatalf("workspace A utility projection crossed into workspace B: %v", otherScores)
	}
}

func installTestForgeKAuthority(t *testing.T, svc *Service, db *sql.DB) {
	t.Helper()
	registry := controllane.NewStaticActionRegistry()
	adapter := controllane.NewProcessor(controllane.ProcessorOptions{
		Registry: registry, TxRunner: controllane.NewSQLiteTransactionRunner(db),
	})
	authorization, err := controllane.NewProductionAuthorizationService(controllane.ProductionAuthorizationOptions{
		Registry: registry, DB: db, ServicePrincipal: controllane.NewForgeCoreServicePrincipal(),
	})
	if err != nil {
		t.Fatalf("production authorization: %v", err)
	}
	selection, err := forgekernel.SelectAuthority(adapter, authorization)
	if err != nil {
		t.Fatalf("select FORGE-K authority: %v", err)
	}
	svc.SetSyscallProcessor(selection.Processor)
}

func assertVSAInfluenceDisabled(t *testing.T, raw json.RawMessage) {
	t.Helper()
	var reason map[string]any
	if err := json.Unmarshal(raw, &reason); err != nil {
		t.Fatalf("decode selection reason: %v", err)
	}
	if got, _ := reason["vsaInfluence"].(string); got != "disabled_unscoped" {
		t.Fatalf("selection reason vsaInfluence=%q, want disabled_unscoped", got)
	}
	if _, present := reason["vsa"]; present {
		t.Fatalf("selection reason contains legacy unscoped VSA evidence: %s", string(raw))
	}
}

func setSetting(t *testing.T, db *sql.DB, key, value string) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO settings(key, value) VALUES(?, ?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, key, value); err != nil {
		t.Fatalf("set setting %s: %v", key, err)
	}
}

func seedSource(t *testing.T, db *sql.DB, path string) int64 {
	t.Helper()
	now := time.Now().UnixMilli()
	res, err := db.Exec(`INSERT INTO sources(path, created_at) VALUES(?, ?)`, path, now)
	if err != nil {
		t.Fatalf("insert source: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
}

func seedChunk(t *testing.T, db *sql.DB, sourceID int64, relPath, absPath, content string) (int64, string, string) {
	t.Helper()
	now := time.Now().UnixMilli()
	res, err := db.Exec(`
INSERT INTO files(source_id, rel_path, abs_path, size_bytes, mtime_ns, content_sha256, indexed_at)
VALUES(?,?,?,?,?,?,?)`,
		sourceID,
		relPath,
		absPath,
		len(content),
		now*1_000_000,
		"sha-"+strconv.FormatInt(now, 10)+"-"+relPath,
		now,
	)
	if err != nil {
		t.Fatalf("insert file: %v", err)
	}
	fileID, _ := res.LastInsertId()
	chunkRes, err := db.Exec(`INSERT INTO chunks(file_id, chunk_index, content) VALUES(?,?,?)`, fileID, 0, content)
	if err != nil {
		t.Fatalf("insert chunk: %v", err)
	}
	chunkID, _ := chunkRes.LastInsertId()
	return chunkID, absPath, relPath
}

func seedObservationWithPointer(t *testing.T, db *sql.DB, sourcePath string, pointer []float64, dims int) int64 {
	t.Helper()
	now := time.Now().UnixMilli()
	obsRes, err := db.Exec(`
INSERT INTO memory_observations(
  created_at, updated_at, observed_at, type, raw_content, summary, source_path, confidence, verification_state
) VALUES(?,?,?,?,?,?,?,?,?)`,
		now,
		now,
		now,
		"retrieval_result",
		sourcePath,
		sourcePath,
		sourcePath,
		0.9,
		"checked",
	)
	if err != nil {
		t.Fatalf("insert observation: %v", err)
	}
	observationID, _ := obsRes.LastInsertId()
	pointerJSON, _ := json.Marshal(pointer)
	if _, err := db.Exec(`
INSERT INTO memory_vsa_pointers(
  observation_id, dims, pointer_json, norm, source_fingerprint, stale, metadata_json, created_at, updated_at
) VALUES(?,?,?,?,?,?,?,?,?)`,
		observationID,
		dims,
		string(pointerJSON),
		vectorNorm(pointer),
		"fp-"+strconv.FormatInt(observationID, 10),
		0,
		`{"seeded":true}`,
		now,
		now,
	); err != nil {
		t.Fatalf("insert vsa pointer: %v", err)
	}
	return observationID
}

func vectorNorm(vec []float64) float64 {
	sum := 0.0
	for _, v := range vec {
		sum += v * v
	}
	return math.Sqrt(sum)
}
