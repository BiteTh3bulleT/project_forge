package forgekshadow

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"forge/projectforge/services/core/internal/store"
)

func TestDiagnosticPersistenceDisabledMeansNoWrite(t *testing.T) {
	sink := &recordingDiagnosticPersistenceSink{}
	wrapped := NewDiagnosticPersistenceSink(NewMemorySink(10), sink, DiagnosticPersistenceOptions{
		Enabled:         false,
		RetentionDays:   30,
		MaxPayloadBytes: 1024,
		Now:             fixedNow,
	})
	if err := wrapped.Store(context.Background(), sampleDiagnosticReport(t)); err != nil {
		t.Fatalf("store disabled persistence: %v", err)
	}
	if len(sink.records) != 0 {
		t.Fatalf("disabled persistence wrote %d records", len(sink.records))
	}
	if got := wrapped.List(); len(got) != 1 {
		t.Fatalf("in-memory sink should still store report, got %d", len(got))
	}
}

func TestDiagnosticPersistenceEnabledWritesSafeRow(t *testing.T) {
	sink := &recordingDiagnosticPersistenceSink{}
	wrapped := NewDiagnosticPersistenceSink(NewMemorySink(10), sink, DiagnosticPersistenceOptions{
		Enabled:         true,
		RetentionDays:   7,
		MaxPayloadBytes: 4096,
		Now:             fixedNow,
	})
	if err := wrapped.Store(context.Background(), sampleDiagnosticReport(t)); err != nil {
		t.Fatalf("store enabled persistence: %v", err)
	}
	if len(sink.records) != 1 {
		t.Fatalf("expected one persisted record, got %d", len(sink.records))
	}
	record := sink.records[0]
	if record.ReportID == "" || record.ReportKind == "" || record.SchemaVersion == 0 {
		t.Fatalf("missing persisted diagnostic identity: %#v", record)
	}
	if !record.NoEffectVerified {
		t.Fatalf("persisted diagnostics must preserve no-effect verification")
	}
	if !record.ExpiresAt.Equal(fixedNow().Add(7 * 24 * time.Hour)) {
		t.Fatalf("unexpected expiry: %s", record.ExpiresAt)
	}
	if record.SummaryJSON["canonical_truth"] == true {
		t.Fatalf("diagnostic persistence must not mark canonical truth")
	}
}

func TestDiagnosticPersistenceStoresControlLaneValidationSummary(t *testing.T) {
	observer := NewObserverWithSink(Config{Enabled: true, ControlLaneValidationEnabled: true}, NewMemorySink(10), fixedNow)
	input := sampleControlLaneValidationInput()
	input.Action = "COMPARE_REF_SHAPE"
	input.ValidationKind = "ref_shape_comparison"
	input.Decision = "drift"
	input.Passed = true
	input.Match = false
	input.AddedRefCount = 1
	input.RemovedRefCount = 2
	input.UnchangedRefCount = 3
	if err := observer.ObserveControlLaneValidation(context.Background(), input); err != nil {
		t.Fatalf("observe control lane validation: %v", err)
	}
	reports := observer.Reports()
	if len(reports) != 1 {
		t.Fatalf("expected one diagnostic report, got %d", len(reports))
	}
	record, err := BuildPersistedDiagnosticReport(reports[0], DiagnosticPersistenceOptions{
		Enabled:         true,
		RetentionDays:   30,
		MaxPayloadBytes: 4096,
		Now:             fixedNow,
	})
	if err != nil {
		t.Fatalf("build persisted control lane validation report: %v", err)
	}
	if record.ReportKind != "control_lane_validation" {
		t.Fatalf("report kind=%q, want control_lane_validation", record.ReportKind)
	}
	if record.SummaryJSON["action"] != "COMPARE_REF_SHAPE" || record.SummaryJSON["validation_kind"] != "ref_shape_comparison" {
		t.Fatalf("unexpected summary identity: %#v", record.SummaryJSON)
	}
	if record.SummaryJSON["added_ref_count"] != 1 || record.SummaryJSON["removed_ref_count"] != 2 || record.SummaryJSON["unchanged_ref_count"] != 3 {
		t.Fatalf("unexpected persisted drift counts: %#v", record.SummaryJSON)
	}
}

func TestDiagnosticPersistenceRejectsUnsafeMetadata(t *testing.T) {
	report := sampleDiagnosticReport(t)
	report.Observation.Metadata["prompt"] = "raw prompt"
	_, err := BuildPersistedDiagnosticReport(report, DiagnosticPersistenceOptions{
		Enabled:         true,
		RetentionDays:   30,
		MaxPayloadBytes: 4096,
		Now:             fixedNow,
	})
	if !errors.Is(err, ErrUnsafeMetadata) {
		t.Fatalf("expected unsafe metadata rejection, got %v", err)
	}
}

func TestDiagnosticPersistenceRejectsOversizedPayload(t *testing.T) {
	report := sampleDiagnosticReport(t)
	report.Observation.Metadata["safe_summary"] = strings.Repeat("x", 400)
	_, err := BuildPersistedDiagnosticReport(report, DiagnosticPersistenceOptions{
		Enabled:         true,
		RetentionDays:   30,
		MaxPayloadBytes: 128,
		Now:             fixedNow,
	})
	if !errors.Is(err, ErrDiagnosticPayloadTooLarge) {
		t.Fatalf("expected payload-too-large error, got %v", err)
	}
}

func TestDiagnosticPersistenceFailureIsIsolated(t *testing.T) {
	wrapped := NewDiagnosticPersistenceSink(NewMemorySink(10), failingDiagnosticPersistenceSink{}, DiagnosticPersistenceOptions{
		Enabled:         true,
		RetentionDays:   30,
		MaxPayloadBytes: 4096,
		Now:             fixedNow,
	})
	if err := wrapped.Store(context.Background(), sampleDiagnosticReport(t)); err != nil {
		t.Fatalf("persistence failure should not fail live diagnostic sink: %v", err)
	}
	if got := wrapped.List(); len(got) != 1 {
		t.Fatalf("in-memory sink should still store report after persistence failure, got %d", len(got))
	}
}

func TestDiagnosticPersistenceDoesNotStoreRawContent(t *testing.T) {
	record, err := BuildPersistedDiagnosticReport(sampleDiagnosticReport(t), DiagnosticPersistenceOptions{
		Enabled:         true,
		RetentionDays:   30,
		MaxPayloadBytes: 4096,
		Now:             fixedNow,
	})
	if err != nil {
		t.Fatalf("build persisted diagnostic report: %v", err)
	}
	raw := strings.ToLower(mustJSONForTest(t, record))
	for _, forbidden := range []string{"prompt", "completion", "message_body", "source_text", "chunk_text", "embedding_vector", "token", "secret"} {
		if strings.Contains(raw, forbidden) {
			t.Fatalf("persisted diagnostic contains forbidden raw marker %q: %s", forbidden, raw)
		}
	}
}

func TestPostgresDiagnosticRepositoryIntegrationOptional(t *testing.T) {
	dsn := requireIntegrationEnvOrSkip(t, "FORGE_POSTGRES_TEST_DSN")
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open postgres test database: %v", err)
	}
	defer db.Close()
	if err := store.NewPostgresMigrationRunner(store.PostgresMigrations()).Run(context.Background(), db); err != nil {
		t.Fatalf("run postgres migrations: %v", err)
	}
	repo := NewPostgresDiagnosticRepository(db)
	record, err := BuildPersistedDiagnosticReport(sampleDiagnosticReport(t), DiagnosticPersistenceOptions{
		Enabled:         true,
		RetentionDays:   30,
		MaxPayloadBytes: 4096,
		Now:             fixedNow,
	})
	if err != nil {
		t.Fatalf("build record: %v", err)
	}
	if err := repo.StoreDiagnosticReport(context.Background(), record); err != nil {
		t.Fatalf("store diagnostic report: %v", err)
	}
	got, err := repo.GetDiagnosticReport(context.Background(), record.ReportID)
	if err != nil {
		t.Fatalf("get diagnostic report: %v", err)
	}
	if got.ReportID != record.ReportID || got.ReportKind != record.ReportKind {
		t.Fatalf("unexpected fetched record: %#v", got)
	}
	list, err := repo.ListDiagnosticReports(context.Background(), record.WorkspaceID, 10)
	if err != nil {
		t.Fatalf("list diagnostic reports: %v", err)
	}
	if len(list) == 0 {
		t.Fatalf("expected listed diagnostic reports")
	}
	expired, err := repo.ListExpiredDiagnosticReports(context.Background(), fixedNow().Add(31*24*time.Hour), 10)
	if err != nil {
		t.Fatalf("list expired diagnostic reports: %v", err)
	}
	if len(expired) == 0 {
		t.Fatalf("expected retention expiry query to find persisted record")
	}
}

func requireIntegrationEnvOrSkip(t *testing.T, name string) string {
	t.Helper()
	value := strings.TrimSpace(os.Getenv(name))
	if value != "" {
		return value
	}
	if os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != "" {
		t.Fatalf("%s must be set in CI for integration coverage", name)
	}
	t.Skipf("%s not set", name)
	return ""
}

type recordingDiagnosticPersistenceSink struct {
	records []PersistedDiagnosticReport
}

func (s *recordingDiagnosticPersistenceSink) StoreDiagnosticReport(_ context.Context, record PersistedDiagnosticReport) error {
	s.records = append(s.records, record)
	return nil
}

type failingDiagnosticPersistenceSink struct{}

func (failingDiagnosticPersistenceSink) StoreDiagnosticReport(context.Context, PersistedDiagnosticReport) error {
	return errors.New("forced persistence failure")
}

func sampleDiagnosticReport(t *testing.T) DiagnosticReport {
	t.Helper()
	observer := NewObserverWithSink(Config{Enabled: true, RetrievalMetadataEnabled: true}, NewMemorySink(10), fixedNow)
	if err := observer.ObserveRetrievalMetadata(context.Background(), RetrievalMetadataInput{
		WorkspaceID:       "workspace-a",
		RequestID:         "request-a",
		CorrelationID:     "correlation-a",
		RetrievalRunID:    "run-1",
		RetrievalResultID: "result-1",
		SourceType:        "file",
		SourceRefID:       "file-ref-1",
		SourceHash:        "sha256:abc",
		ResultCount:       4,
		SelectedCount:     2,
		ScoreSummary:      "high-confidence",
		RankingPosition:   1,
		RetrievalStrategy: "hybrid",
		IndexType:         "fts",
		FreshnessStatus:   "fresh",
		Duration:          25 * time.Millisecond,
		Metadata: map[string]any{
			"safe_summary": "metadata only",
		},
	}); err != nil {
		t.Fatalf("observe retrieval metadata: %v", err)
	}
	reports := observer.Reports()
	if len(reports) != 1 {
		t.Fatalf("expected one diagnostic report, got %d", len(reports))
	}
	return reports[0]
}

func mustJSONForTest(t *testing.T, value any) string {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal JSON: %v", err)
	}
	return string(raw)
}
