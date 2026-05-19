package evaluations

import (
	"context"
	"database/sql"
	"math"
	"reflect"
	"testing"
	"time"

	"forge/projectforge/services/core/internal/store"
)

func TestSaveValidatesRatingsAndPersistsDefaults(t *testing.T) {
	ctx := context.Background()
	svc, cleanup := newTestService(t)
	defer cleanup()
	insertJob(t, svc.db, "job-1", "codex", "analysis.safe_local")

	if _, err := svc.Save(ctx, SaveRequest{JobID: "", QualityRating: 5, UsefulnessRating: 5, CorrectnessConfidence: 5, PacketQualityRating: 5, AdapterSuitability: 5}); err == nil {
		t.Fatalf("Save accepted blank job id")
	}
	if _, err := svc.Save(ctx, SaveRequest{JobID: "job-1", QualityRating: 0, UsefulnessRating: 5, CorrectnessConfidence: 5, PacketQualityRating: 5, AdapterSuitability: 5}); err == nil {
		t.Fatalf("Save accepted out-of-range rating")
	}

	record, err := svc.Save(ctx, SaveRequest{
		JobID:                 "job-1",
		Success:               true,
		QualityRating:         5,
		UsefulnessRating:      4,
		CorrectnessConfidence: 5,
		PacketQualityRating:   4,
		AdapterSuitability:    5,
		RetryRecommended:      false,
		InfluenceRouting:      true,
		Notes:                 "  useful  ",
		Scorer:                "  ",
	})
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	if !record.Success || record.Scorer != "operator" {
		t.Fatalf("record success/scorer = %+v, want success with default operator scorer", record)
	}
	if record.Notes != "  useful  " {
		t.Fatalf("notes = %q, want preserved operator notes", record.Notes)
	}
}

func TestLatestListAndDossierFilter(t *testing.T) {
	ctx := context.Background()
	svc, cleanup := newTestService(t)
	defer cleanup()
	dossierID := insertDossier(t, svc.db, "Primary dossier")
	otherDossierID := insertDossier(t, svc.db, "Other dossier")
	insertJob(t, svc.db, "job-1", "codex", "analysis.safe_local")
	insertJob(t, svc.db, "job-2", "ollama", "ollama.summary")
	insertJob(t, svc.db, "job-3", "codex", "analysis.safe_local")

	mustSave(t, svc, SaveRequest{JobID: "job-1", DossierID: &dossierID, Success: true, QualityRating: 4, UsefulnessRating: 4, CorrectnessConfidence: 4, PacketQualityRating: 4, AdapterSuitability: 4, InfluenceRouting: true})
	firstJob2 := mustSave(t, svc, SaveRequest{JobID: "job-2", DossierID: &dossierID, Success: false, QualityRating: 2, UsefulnessRating: 2, CorrectnessConfidence: 3, PacketQualityRating: 3, AdapterSuitability: 2, RetryRecommended: true, InfluenceRouting: false, Scorer: "critic"})
	latestJob2 := mustSave(t, svc, SaveRequest{JobID: "job-2", DossierID: &dossierID, Success: true, QualityRating: 5, UsefulnessRating: 5, CorrectnessConfidence: 5, PacketQualityRating: 5, AdapterSuitability: 5, InfluenceRouting: true, Scorer: "operator"})
	mustSave(t, svc, SaveRequest{JobID: "job-3", DossierID: &otherDossierID, Success: true, QualityRating: 3, UsefulnessRating: 3, CorrectnessConfidence: 3, PacketQualityRating: 3, AdapterSuitability: 3, InfluenceRouting: true})

	latest, err := svc.LatestByJob(ctx, "job-2")
	if err != nil {
		t.Fatalf("LatestByJob failed: %v", err)
	}
	if latest == nil || latest.ID != latestJob2.ID || latest.ID == firstJob2.ID {
		t.Fatalf("latest job-2 = %+v, want newest record %d", latest, latestJob2.ID)
	}
	missing, err := svc.LatestByJob(ctx, "missing")
	if err != nil {
		t.Fatalf("LatestByJob missing failed: %v", err)
	}
	if missing != nil {
		t.Fatalf("LatestByJob missing = %+v, want nil", missing)
	}

	records, err := svc.List(ctx, 10, &dossierID)
	if err != nil {
		t.Fatalf("List filtered failed: %v", err)
	}
	if got := recordIDs(records); !reflect.DeepEqual(got, []int64{latestJob2.ID, firstJob2.ID, 1}) {
		t.Fatalf("filtered record ids = %v, want newest primary dossier records", got)
	}
}

func TestAdapterMetrics(t *testing.T) {
	ctx := context.Background()
	svc, cleanup := newTestService(t)
	defer cleanup()
	dossierID := insertDossier(t, svc.db, "Metrics dossier")
	insertJob(t, svc.db, "codex-1", "codex", "analysis.safe_local")
	insertJob(t, svc.db, "codex-2", "codex", "analysis.safe_local")
	insertJob(t, svc.db, "ollama-1", "ollama", "ollama.summary")

	mustSave(t, svc, SaveRequest{JobID: "codex-1", DossierID: &dossierID, Success: true, QualityRating: 5, UsefulnessRating: 4, CorrectnessConfidence: 5, PacketQualityRating: 4, AdapterSuitability: 5, InfluenceRouting: true})
	mustSave(t, svc, SaveRequest{JobID: "codex-2", DossierID: &dossierID, Success: false, QualityRating: 3, UsefulnessRating: 2, CorrectnessConfidence: 3, PacketQualityRating: 3, AdapterSuitability: 3, RetryRecommended: true, InfluenceRouting: true})
	mustSave(t, svc, SaveRequest{JobID: "ollama-1", DossierID: &dossierID, Success: true, QualityRating: 4, UsefulnessRating: 5, CorrectnessConfidence: 4, PacketQualityRating: 4, AdapterSuitability: 4, InfluenceRouting: true})

	metrics, err := svc.AdapterMetrics(ctx, &dossierID)
	if err != nil {
		t.Fatalf("AdapterMetrics failed: %v", err)
	}
	if len(metrics) != 2 {
		t.Fatalf("metrics = %+v, want two adapters", metrics)
	}
	codex := metricsByAdapter(metrics)["codex"]
	if codex.Runs != 2 || !near(codex.SuccessRate, 0.5) || !near(codex.AvgQuality, 4.0) || !near(codex.RetryRate, 0.5) {
		t.Fatalf("codex metrics = %+v, want averaged two-run metrics", codex)
	}
	ollama := metricsByAdapter(metrics)["ollama"]
	if ollama.Runs != 1 || !near(ollama.SuccessRate, 1.0) || !near(ollama.AvgAdapterSuitability, 4.0) {
		t.Fatalf("ollama metrics = %+v, want single successful run", ollama)
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

func mustSave(t *testing.T, svc *Service, req SaveRequest) *Record {
	t.Helper()
	record, err := svc.Save(context.Background(), req)
	if err != nil {
		t.Fatalf("Save(%s) failed: %v", req.JobID, err)
	}
	return record
}

func insertDossier(t *testing.T, db *sql.DB, name string) int64 {
	t.Helper()
	now := time.Now().UnixMilli()
	res, err := db.Exec(`
INSERT INTO dossiers(created_at, updated_at, name, description)
VALUES(?,?,?,?)`, now, now, name, "")
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

func recordIDs(records []Record) []int64 {
	out := make([]int64, 0, len(records))
	for _, record := range records {
		out = append(out, record.ID)
	}
	return out
}

func metricsByAdapter(metrics []AdapterMetric) map[string]AdapterMetric {
	out := map[string]AdapterMetric{}
	for _, metric := range metrics {
		out[metric.Adapter] = metric
	}
	return out
}

func near(got, want float64) bool {
	return math.Abs(got-want) < 0.0001
}
