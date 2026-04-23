package retrieval

import (
	"context"
	"database/sql"
	"encoding/json"
	"math"
	"strconv"
	"testing"
	"time"

	"forge/projectforge/services/core/internal/embeddings"
	"forge/projectforge/services/core/internal/memory"
	"forge/projectforge/services/core/internal/search"
	"forge/projectforge/services/core/internal/store"
)

func TestRunVSAModesOffShadowActive(t *testing.T) {
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
	req := RunRequest{
		Query:           "vsa query",
		Mode:            ModeKeyword,
		Limit:           2,
		SelectForPacket: 2,
	}

	setSetting(t, st.DB, "retrieval_vsa_mode", "off")
	offRun, err := svc.Run(ctx, req)
	if err != nil {
		t.Fatalf("run off mode: %v", err)
	}
	if len(offRun.Results) != 2 {
		t.Fatalf("off mode results=%d, want 2", len(offRun.Results))
	}
	var offSignalCount int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM retrieval_result_vsa_signals WHERE retrieval_run_id = ?`, offRun.ID).Scan(&offSignalCount); err != nil {
		t.Fatalf("off signal count: %v", err)
	}
	if offSignalCount != 0 {
		t.Fatalf("off mode should not persist vsa signals, got %d", offSignalCount)
	}

	setSetting(t, st.DB, "retrieval_vsa_mode", "shadow")
	shadowRun, err := svc.Run(ctx, req)
	if err != nil {
		t.Fatalf("run shadow mode: %v", err)
	}
	if len(shadowRun.Results) != len(offRun.Results) {
		t.Fatalf("shadow results=%d, want %d", len(shadowRun.Results), len(offRun.Results))
	}
	for i := range offRun.Results {
		if shadowRun.Results[i].RelPath != offRun.Results[i].RelPath {
			t.Fatalf("shadow ranking parity failed at index %d: got %q want %q", i, shadowRun.Results[i].RelPath, offRun.Results[i].RelPath)
		}
		if math.Abs(shadowRun.Results[i].HybridScore-offRun.Results[i].HybridScore) > 0.000001 {
			t.Fatalf("shadow score parity failed at index %d: got %.6f want %.6f", i, shadowRun.Results[i].HybridScore, offRun.Results[i].HybridScore)
		}
		assertSelectionReasonVSAMode(t, shadowRun.Results[i].SelectionReason, "shadow")
	}

	var shadowSignalCount int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM retrieval_result_vsa_signals WHERE retrieval_run_id = ?`, shadowRun.ID).Scan(&shadowSignalCount); err != nil {
		t.Fatalf("shadow signal count: %v", err)
	}
	if shadowSignalCount != len(shadowRun.Results) {
		t.Fatalf("shadow signal count=%d, want %d", shadowSignalCount, len(shadowRun.Results))
	}
	var shadowMaxApplied float64
	if err := st.DB.QueryRow(`SELECT COALESCE(MAX(ABS(applied_score)), 0) FROM retrieval_result_vsa_signals WHERE retrieval_run_id = ?`, shadowRun.ID).Scan(&shadowMaxApplied); err != nil {
		t.Fatalf("shadow max applied: %v", err)
	}
	if shadowMaxApplied > 0.000001 {
		t.Fatalf("shadow applied score should be zero, got %.6f", shadowMaxApplied)
	}

	setSetting(t, st.DB, "retrieval_vsa_mode", "active")
	activeRun, err := svc.Run(ctx, req)
	if err != nil {
		t.Fatalf("run active mode: %v", err)
	}
	if len(activeRun.Results) != len(shadowRun.Results) {
		t.Fatalf("active results=%d, want %d", len(activeRun.Results), len(shadowRun.Results))
	}

	shadowByPath := map[string]Result{}
	for _, row := range shadowRun.Results {
		shadowByPath[row.RelPath] = row
	}
	seenPositiveDelta := false
	for _, row := range activeRun.Results {
		shadowRow, ok := shadowByPath[row.RelPath]
		if !ok {
			t.Fatalf("active row path %q missing from shadow run", row.RelPath)
		}
		delta := row.HybridScore - shadowRow.HybridScore
		if delta > 0.000001 {
			seenPositiveDelta = true
		}
		if math.Abs(delta) > 0.050001 {
			t.Fatalf("active delta %.6f exceeded max additive clamp for %s", delta, row.RelPath)
		}
		assertSelectionReasonVSAMode(t, row.SelectionReason, "active")
	}
	if !seenPositiveDelta {
		t.Fatalf("expected at least one positive active additive delta")
	}

	var activeMaxApplied float64
	if err := st.DB.QueryRow(`SELECT COALESCE(MAX(ABS(applied_score)), 0) FROM retrieval_result_vsa_signals WHERE retrieval_run_id = ?`, activeRun.ID).Scan(&activeMaxApplied); err != nil {
		t.Fatalf("active max applied: %v", err)
	}
	if activeMaxApplied > 0.050001 {
		t.Fatalf("active applied score exceeded clamp: %.6f", activeMaxApplied)
	}
	var activePositive int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM retrieval_result_vsa_signals WHERE retrieval_run_id = ? AND applied_score > 0`, activeRun.ID).Scan(&activePositive); err != nil {
		t.Fatalf("active positive applied count: %v", err)
	}
	if activePositive == 0 {
		t.Fatalf("expected persisted positive applied VSA signal in active mode")
	}

	if chunkA <= 0 || relA == "" {
		t.Fatalf("seed invariant failed")
	}
}

func assertSelectionReasonVSAMode(t *testing.T, raw json.RawMessage, wantMode string) {
	t.Helper()
	var reason map[string]any
	if err := json.Unmarshal(raw, &reason); err != nil {
		t.Fatalf("decode selection reason: %v", err)
	}
	vsaAny, ok := reason["vsa"]
	if !ok {
		t.Fatalf("selection reason missing vsa block: %s", string(raw))
	}
	vsa, ok := vsaAny.(map[string]any)
	if !ok {
		t.Fatalf("selection reason vsa block type=%T", vsaAny)
	}
	mode, _ := vsa["mode"].(string)
	if mode != wantMode {
		t.Fatalf("selection reason vsa mode=%q, want %q", mode, wantMode)
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
