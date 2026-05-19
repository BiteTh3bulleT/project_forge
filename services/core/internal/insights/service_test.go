package insights

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"forge/projectforge/services/core/internal/store"
)

func TestGeneratePersistsAdapterRetrievalAndReviewInsights(t *testing.T) {
	ctx := context.Background()
	svc, cleanup := newTestService(t)
	defer cleanup()
	dossierID := insertDossier(t, svc.db, "Insight dossier")
	for _, id := range []string{"codex-1", "codex-2", "codex-3"} {
		insertJob(t, svc.db, id, "codex", "analysis.safe_local")
		insertEvaluation(t, svc.db, id, dossierID, true, 5, false)
	}
	insertRetrievalRunWithResults(t, svc.db, dossierID, []string{"useful", "useful", "noisy"})
	insertReviewRecords(t, svc.db, dossierID, []string{"pending", "pending", "pending"})

	records, err := svc.Generate(ctx, GenerateRequest{DossierID: &dossierID})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("generated insights = %d, want adapter/retrieval/review", len(records))
	}
	byAdapter := recordsByAdapter(records)
	if got := byAdapter["codex"]; got == nil || !strings.Contains(got.Recommendation, "advisory-preferred") {
		t.Fatalf("codex insight = %+v, want preferred recommendation", got)
	}
	if got := byAdapter["retrieval"]; got == nil || !strings.Contains(got.Recommendation, "strong useful-hit ratio") {
		t.Fatalf("retrieval insight = %+v, want useful-hit recommendation", got)
	}
	if got := byAdapter["review"]; got == nil || !strings.Contains(got.Recommendation, "Review queue is accumulating") {
		t.Fatalf("review insight = %+v, want pending queue recommendation", got)
	}
	for _, record := range records {
		if record.DossierID == nil || *record.DossierID != dossierID {
			t.Fatalf("record dossier id = %v, want %d", record.DossierID, dossierID)
		}
		if !json.Valid(record.Reasons) || !json.Valid(record.Evidence) {
			t.Fatalf("record has invalid json: reasons=%q evidence=%q", string(record.Reasons), string(record.Evidence))
		}
		if record.Confidence < 0.05 || record.Confidence > 0.95 {
			t.Fatalf("confidence = %f, want clamped range", record.Confidence)
		}
	}

	listed, err := svc.List(ctx, 10, &dossierID)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(listed) != 3 {
		t.Fatalf("listed insights = %d, want 3", len(listed))
	}
}

func TestGenerateWithoutSignalsReturnsNoInsights(t *testing.T) {
	ctx := context.Background()
	svc, cleanup := newTestService(t)
	defer cleanup()
	dossierID := insertDossier(t, svc.db, "Empty dossier")

	records, err := svc.Generate(ctx, GenerateRequest{DossierID: &dossierID})
	if err != nil {
		t.Fatalf("Generate empty failed: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("empty generated insights = %+v, want none", records)
	}
}

func newTestService(t *testing.T) (*Service, func()) {
	t.Helper()
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	return New(st.DB), func() {
		if err := st.Close(); err != nil {
			t.Fatalf("close store: %v", err)
		}
	}
}

func insertDossier(t *testing.T, db *sql.DB, name string) int64 {
	t.Helper()
	now := time.Now().UnixMilli()
	res, err := db.Exec(`INSERT INTO dossiers(created_at, updated_at, name, description) VALUES(?,?,?,?)`, now, now, name, "")
	if err != nil {
		t.Fatalf("insert dossier %q: %v", name, err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("dossier last insert id: %v", err)
	}
	return id
}

func insertJob(t *testing.T, db *sql.DB, id, adapter, action string) {
	t.Helper()
	now := time.Now().UnixMilli()
	_, err := db.Exec(`
INSERT INTO jobs(
  id, created_at, updated_at, queued_at, title, requested_action, target_adapter,
  initiating_source, execution_boundary, risk_class, status, approval_status,
  write_intent, metadata_json
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		id,
		now,
		now,
		now,
		id,
		action,
		adapter,
		"test",
		"test",
		"read_only",
		"succeeded",
		"not_required",
		0,
		`{"templateId":"search_packet"}`,
	)
	if err != nil {
		t.Fatalf("insert job %q: %v", id, err)
	}
}

func insertEvaluation(t *testing.T, db *sql.DB, jobID string, dossierID int64, success bool, quality int, retry bool) {
	t.Helper()
	now := time.Now().UnixMilli()
	_, err := db.Exec(`
INSERT INTO evaluation_records(
  created_at, job_id, dossier_id, success, quality_rating, usefulness_rating,
  correctness_confidence, packet_quality_rating, adapter_suitability,
  retry_recommended, influence_routing, notes, scorer
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		now,
		jobID,
		dossierID,
		boolToInt(success),
		quality,
		quality,
		quality,
		quality,
		quality,
		boolToInt(retry),
		1,
		"",
		"operator",
	)
	if err != nil {
		t.Fatalf("insert evaluation for %s: %v", jobID, err)
	}
}

func insertRetrievalRunWithResults(t *testing.T, db *sql.DB, dossierID int64, labels []string) {
	t.Helper()
	now := time.Now().UnixMilli()
	res, err := db.Exec(`INSERT INTO retrieval_runs(created_at, query, mode, dossier_id, weighting_json, notes) VALUES(?,?,?,?,?,?)`, now, "query", "hybrid", dossierID, `{}`, "")
	if err != nil {
		t.Fatalf("insert retrieval run: %v", err)
	}
	runID, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("retrieval run last insert id: %v", err)
	}
	for i, label := range labels {
		_, err := db.Exec(`
INSERT INTO retrieval_results(retrieval_run_id, abs_path, rel_path, rank_index, usefulness_label)
VALUES(?,?,?,?,?)`, runID, "abs", "rel", i, label)
		if err != nil {
			t.Fatalf("insert retrieval result %d: %v", i, err)
		}
	}
}

func insertReviewRecords(t *testing.T, db *sql.DB, dossierID int64, statuses []string) {
	t.Helper()
	now := time.Now().UnixMilli()
	for i, status := range statuses {
		_, err := db.Exec(`
INSERT INTO review_records(created_at, updated_at, target_type, target_id, dossier_id, status, summary, notes, annotations_json, reviewer)
VALUES(?,?,?,?,?,?,?,?,?,?)`, now+int64(i), now+int64(i), "job", "job", dossierID, status, "", "", "[]", "operator")
		if err != nil {
			t.Fatalf("insert review %d: %v", i, err)
		}
	}
}

func recordsByAdapter(records []Record) map[string]*Record {
	out := map[string]*Record{}
	for i := range records {
		out[records[i].AdapterID] = &records[i]
	}
	return out
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
