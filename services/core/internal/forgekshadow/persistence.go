package forgekshadow

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

const diagnosticPersistenceSchemaVersion = 1

type DiagnosticPersistenceOptions struct {
	Enabled         bool
	RetentionDays   int
	MaxPayloadBytes int
	Now             func() time.Time
}

type PersistedDiagnosticReport struct {
	ReportID         string         `json:"report_id"`
	ReportKind       string         `json:"report_kind"`
	WorkspaceID      string         `json:"workspace_id"`
	RequestID        string         `json:"request_id"`
	CorrelationID    string         `json:"correlation_id"`
	ObservedAt       time.Time      `json:"observed_at"`
	StoredAt         time.Time      `json:"stored_at"`
	ExpiresAt        time.Time      `json:"expires_at"`
	SummaryJSON      map[string]any `json:"summary_json"`
	WarningsJSON     []string       `json:"warnings_json"`
	MetadataJSON     map[string]any `json:"metadata_json"`
	NoEffectVerified bool           `json:"no_effect_verified"`
	SchemaVersion    int            `json:"schema_version"`
}

type DiagnosticPersistenceRepository interface {
	StoreDiagnosticReport(context.Context, PersistedDiagnosticReport) error
}

type diagnosticPersistenceSink struct {
	primary Sink
	repo    DiagnosticPersistenceRepository
	opts    DiagnosticPersistenceOptions
}

func NewDiagnosticPersistenceSink(primary Sink, repo DiagnosticPersistenceRepository, opts DiagnosticPersistenceOptions) Sink {
	if primary == nil {
		primary = NewMemorySink(DefaultMaxReports)
	}
	return &diagnosticPersistenceSink{primary: primary, repo: repo, opts: opts}
}

func (s *diagnosticPersistenceSink) Store(ctx context.Context, report DiagnosticReport) error {
	if s == nil {
		return nil
	}
	if err := s.primary.Store(ctx, report); err != nil {
		return err
	}
	if !s.opts.Enabled || s.repo == nil {
		return nil
	}
	record, err := BuildPersistedDiagnosticReport(report, s.opts)
	if err != nil {
		return nil
	}
	if err := s.repo.StoreDiagnosticReport(ctx, record); err != nil {
		return nil
	}
	return nil
}

func (s *diagnosticPersistenceSink) List() []DiagnosticReport {
	if s == nil || s.primary == nil {
		return nil
	}
	return s.primary.List()
}

func BuildPersistedDiagnosticReport(report DiagnosticReport, opts DiagnosticPersistenceOptions) (PersistedDiagnosticReport, error) {
	if opts.RetentionDays <= 0 || opts.MaxPayloadBytes <= 0 {
		return PersistedDiagnosticReport{}, ErrDiagnosticPersistenceInput
	}
	now := time.Now().UTC
	if opts.Now != nil {
		now = opts.Now
	}
	storedAt := now().UTC()
	reportID := strings.TrimSpace(report.Comparison.ReportID)
	if reportID == "" {
		reportID = firstNonEmpty(
			reportIDFromRoute(report.RouteEnvelope),
			reportIDFromChat(report.ChatMetadata),
			reportIDFromRetrieval(report.RetrievalMetadata),
			reportIDFromAdvisory(report.Advisory),
			report.Observation.ObservationID,
		)
	}
	if reportID == "" {
		return PersistedDiagnosticReport{}, fmt.Errorf("%w: report id required", ErrDiagnosticPersistenceInput)
	}

	kind := diagnosticReportKind(report)
	workspaceID := firstNonEmpty(report.Comparison.WorkspaceID, report.Observation.WorkspaceID, workspaceFromTypedReport(report))
	requestID := firstNonEmpty(report.Comparison.RequestID, report.Observation.RequestID, requestFromTypedReport(report))
	correlationID := correlationFromTypedReport(report)
	observedAt := report.Observation.ObservedAt.UTC()
	if observedAt.IsZero() {
		observedAt = observedAtFromTypedReport(report)
	}
	if observedAt.IsZero() {
		observedAt = storedAt
	}
	metadata, err := persistedDiagnosticMetadata(report)
	if err != nil {
		return PersistedDiagnosticReport{}, err
	}
	summary := persistedDiagnosticSummary(report, kind)
	warnings := persistedDiagnosticWarnings(report)
	record := PersistedDiagnosticReport{
		ReportID:         reportID,
		ReportKind:       kind,
		WorkspaceID:      workspaceID,
		RequestID:        requestID,
		CorrelationID:    correlationID,
		ObservedAt:       observedAt,
		StoredAt:         storedAt,
		ExpiresAt:        storedAt.Add(time.Duration(opts.RetentionDays) * 24 * time.Hour),
		SummaryJSON:      summary,
		WarningsJSON:     warnings,
		MetadataJSON:     metadata,
		NoEffectVerified: report.Comparison.NoEffectVerified,
		SchemaVersion:    diagnosticPersistenceSchemaVersion,
	}
	if len(record.SummaryJSON) == 0 {
		record.SummaryJSON = map[string]any{}
	}
	if len(record.MetadataJSON) == 0 {
		record.MetadataJSON = map[string]any{}
	}
	if len(record.WarningsJSON) == 0 {
		record.WarningsJSON = nil
	}
	raw, err := json.Marshal(record)
	if err != nil {
		return PersistedDiagnosticReport{}, err
	}
	if len(raw) > opts.MaxPayloadBytes {
		return PersistedDiagnosticReport{}, ErrDiagnosticPayloadTooLarge
	}
	return record, nil
}

type PostgresDiagnosticRepository struct {
	db *sql.DB
}

func NewPostgresDiagnosticRepository(db *sql.DB) *PostgresDiagnosticRepository {
	return &PostgresDiagnosticRepository{db: db}
}

func (r *PostgresDiagnosticRepository) StoreDiagnosticReport(ctx context.Context, record PersistedDiagnosticReport) error {
	if r == nil || r.db == nil {
		return ErrDiagnosticPersistenceInput
	}
	summary, err := json.Marshal(record.SummaryJSON)
	if err != nil {
		return err
	}
	warnings, err := json.Marshal(record.WarningsJSON)
	if err != nil {
		return err
	}
	metadata, err := json.Marshal(record.MetadataJSON)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `
INSERT INTO shadow_diagnostic_reports (
  report_id, report_kind, workspace_id, request_id, correlation_id,
  observed_at, stored_at, expires_at, summary_json, warnings_json,
  metadata_json, no_effect_verified, schema_version
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb, $10::jsonb, $11::jsonb, $12, $13)
ON CONFLICT (report_id) DO UPDATE SET
  report_kind = EXCLUDED.report_kind,
  workspace_id = EXCLUDED.workspace_id,
  request_id = EXCLUDED.request_id,
  correlation_id = EXCLUDED.correlation_id,
  observed_at = EXCLUDED.observed_at,
  stored_at = EXCLUDED.stored_at,
  expires_at = EXCLUDED.expires_at,
  summary_json = EXCLUDED.summary_json,
  warnings_json = EXCLUDED.warnings_json,
  metadata_json = EXCLUDED.metadata_json,
  no_effect_verified = EXCLUDED.no_effect_verified,
  schema_version = EXCLUDED.schema_version
`, record.ReportID, record.ReportKind, record.WorkspaceID, record.RequestID, record.CorrelationID,
		record.ObservedAt, record.StoredAt, record.ExpiresAt, string(summary), string(warnings),
		string(metadata), record.NoEffectVerified, record.SchemaVersion)
	return err
}

func (r *PostgresDiagnosticRepository) GetDiagnosticReport(ctx context.Context, reportID string) (PersistedDiagnosticReport, error) {
	rows, err := r.queryReports(ctx, `
SELECT report_id, report_kind, workspace_id, request_id, correlation_id,
       observed_at, stored_at, expires_at, summary_json, warnings_json,
       metadata_json, no_effect_verified, schema_version
FROM shadow_diagnostic_reports
WHERE report_id = $1
`, reportID)
	if err != nil {
		return PersistedDiagnosticReport{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		return PersistedDiagnosticReport{}, sql.ErrNoRows
	}
	record, err := scanPersistedDiagnosticReport(rows)
	if err != nil {
		return PersistedDiagnosticReport{}, err
	}
	return record, rows.Err()
}

func (r *PostgresDiagnosticRepository) ListDiagnosticReports(ctx context.Context, workspaceID string, limit int) ([]PersistedDiagnosticReport, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.queryReports(ctx, `
SELECT report_id, report_kind, workspace_id, request_id, correlation_id,
       observed_at, stored_at, expires_at, summary_json, warnings_json,
       metadata_json, no_effect_verified, schema_version
FROM shadow_diagnostic_reports
WHERE workspace_id = $1
ORDER BY stored_at DESC, report_id ASC
LIMIT $2
`, workspaceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectPersistedDiagnosticReports(rows)
}

func (r *PostgresDiagnosticRepository) ListExpiredDiagnosticReports(ctx context.Context, cutoff time.Time, limit int) ([]PersistedDiagnosticReport, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.queryReports(ctx, `
SELECT report_id, report_kind, workspace_id, request_id, correlation_id,
       observed_at, stored_at, expires_at, summary_json, warnings_json,
       metadata_json, no_effect_verified, schema_version
FROM shadow_diagnostic_reports
WHERE expires_at <= $1
ORDER BY expires_at ASC, report_id ASC
LIMIT $2
`, cutoff, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectPersistedDiagnosticReports(rows)
}

func (r *PostgresDiagnosticRepository) queryReports(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	if r == nil || r.db == nil {
		return nil, ErrDiagnosticPersistenceInput
	}
	return r.db.QueryContext(ctx, query, args...)
}

func collectPersistedDiagnosticReports(rows *sql.Rows) ([]PersistedDiagnosticReport, error) {
	var out []PersistedDiagnosticReport
	for rows.Next() {
		record, err := scanPersistedDiagnosticReport(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, record)
	}
	return out, rows.Err()
}

func scanPersistedDiagnosticReport(scanner interface {
	Scan(dest ...any) error
}) (PersistedDiagnosticReport, error) {
	var record PersistedDiagnosticReport
	var summaryRaw, warningsRaw, metadataRaw []byte
	err := scanner.Scan(
		&record.ReportID,
		&record.ReportKind,
		&record.WorkspaceID,
		&record.RequestID,
		&record.CorrelationID,
		&record.ObservedAt,
		&record.StoredAt,
		&record.ExpiresAt,
		&summaryRaw,
		&warningsRaw,
		&metadataRaw,
		&record.NoEffectVerified,
		&record.SchemaVersion,
	)
	if err != nil {
		return PersistedDiagnosticReport{}, err
	}
	if err := json.Unmarshal(summaryRaw, &record.SummaryJSON); err != nil {
		return PersistedDiagnosticReport{}, err
	}
	if err := json.Unmarshal(warningsRaw, &record.WarningsJSON); err != nil {
		return PersistedDiagnosticReport{}, err
	}
	if err := json.Unmarshal(metadataRaw, &record.MetadataJSON); err != nil {
		return PersistedDiagnosticReport{}, err
	}
	return record, nil
}

func persistedDiagnosticMetadata(report DiagnosticReport) (map[string]any, error) {
	merged := map[string]any{
		"diagnostic_only":         true,
		"no_raw_content_verified": true,
		"schema_version":          diagnosticPersistenceSchemaVersion,
	}
	for key, value := range report.Observation.Metadata {
		merged[key] = value
	}
	for key, value := range typedMetadata(report) {
		merged[key] = value
	}
	return safeMetadata(merged)
}

func typedMetadata(report DiagnosticReport) map[string]any {
	switch {
	case report.RouteEnvelope != nil:
		return report.RouteEnvelope.Metadata
	case report.ChatMetadata != nil:
		return report.ChatMetadata.Metadata
	case report.RetrievalMetadata != nil:
		return report.RetrievalMetadata.Metadata
	case report.Advisory != nil:
		return report.Advisory.Metadata
	default:
		return nil
	}
}

func persistedDiagnosticSummary(report DiagnosticReport, kind string) map[string]any {
	summary := map[string]any{
		"diagnostic_only":    true,
		"no_effect_verified": report.Comparison.NoEffectVerified,
		"report_kind":        kind,
	}
	if report.RouteEnvelope != nil {
		summary["route_class"] = report.RouteEnvelope.RouteClass
		summary["route_pattern"] = report.RouteEnvelope.RoutePattern
		summary["status_code"] = report.RouteEnvelope.StatusCode
		summary["duration_ms"] = report.RouteEnvelope.DurationMS
	}
	if report.ChatMetadata != nil {
		summary["operation_kind"] = report.ChatMetadata.OperationKind
		summary["thread_id"] = report.ChatMetadata.ThreadID
		summary["message_id"] = report.ChatMetadata.MessageID
		summary["message_count"] = report.ChatMetadata.MessageCount
		summary["duration_ms"] = report.ChatMetadata.DurationMS
	}
	if report.RetrievalMetadata != nil {
		summary["retrieval_run_id"] = report.RetrievalMetadata.RetrievalRunID
		summary["retrieval_result_id"] = report.RetrievalMetadata.RetrievalResultID
		summary["source_type"] = report.RetrievalMetadata.SourceType
		summary["source_ref_id"] = report.RetrievalMetadata.SourceRefID
		summary["source_hash"] = report.RetrievalMetadata.SourceHash
		summary["result_count"] = report.RetrievalMetadata.ResultCount
		summary["selected_count"] = report.RetrievalMetadata.SelectedCount
		summary["score_summary"] = report.RetrievalMetadata.ScoreSummary
		summary["ranking_position"] = report.RetrievalMetadata.RankingPosition
		summary["retrieval_strategy"] = report.RetrievalMetadata.RetrievalStrategy
		summary["index_type"] = report.RetrievalMetadata.IndexType
		summary["freshness_status"] = report.RetrievalMetadata.FreshnessStatus
		summary["duration_ms"] = report.RetrievalMetadata.DurationMS
	}
	if report.Advisory != nil {
		summary["route_metadata_count"] = report.Advisory.EvidenceSummary.RouteMetadataCount
		summary["chat_metadata_count"] = report.Advisory.EvidenceSummary.ChatMetadataCount
		summary["retrieval_metadata_count"] = report.Advisory.EvidenceSummary.RetrievalMetadataCount
		summary["safe_ref_count"] = report.Advisory.EvidenceSummary.SafeRefCount
		summary["risk_flag_count"] = len(report.Advisory.RiskSummary.RiskFlags)
		summary["warning_count"] = report.Advisory.RiskSummary.WarningCount
	}
	return summary
}

func persistedDiagnosticWarnings(report DiagnosticReport) []string {
	warnings := append([]string(nil), report.Comparison.Warnings...)
	if report.RouteEnvelope != nil {
		warnings = append(warnings, report.RouteEnvelope.Warnings...)
	}
	if report.ChatMetadata != nil {
		warnings = append(warnings, report.ChatMetadata.Warnings...)
	}
	if report.RetrievalMetadata != nil {
		warnings = append(warnings, report.RetrievalMetadata.Warnings...)
	}
	if report.Advisory != nil {
		warnings = append(warnings, report.Advisory.Warnings...)
	}
	for i := range warnings {
		warnings[i] = strings.TrimSpace(warnings[i])
	}
	sort.Strings(warnings)
	out := warnings[:0]
	for _, warning := range warnings {
		if warning == "" {
			continue
		}
		if len(out) == 0 || out[len(out)-1] != warning {
			out = append(out, warning)
		}
	}
	return out
}

func diagnosticReportKind(report DiagnosticReport) string {
	switch {
	case report.Advisory != nil:
		return "shadow_advisory"
	case report.RetrievalMetadata != nil:
		return "retrieval_metadata"
	case report.ChatMetadata != nil:
		return "chat_metadata"
	case report.RouteEnvelope != nil:
		return "route_envelope"
	default:
		return "comparison"
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func reportIDFromRoute(value *RouteEnvelopeObservation) string {
	if value == nil {
		return ""
	}
	return value.ObservationID
}

func reportIDFromChat(value *ChatMetadataObservation) string {
	if value == nil {
		return ""
	}
	return value.ObservationID
}

func reportIDFromRetrieval(value *RetrievalMetadataObservation) string {
	if value == nil {
		return ""
	}
	return value.ObservationID
}

func reportIDFromAdvisory(value *ShadowAdvisoryReport) string {
	if value == nil {
		return ""
	}
	return value.ReportID
}

func workspaceFromTypedReport(report DiagnosticReport) string {
	switch {
	case report.RouteEnvelope != nil:
		return report.RouteEnvelope.WorkspaceID
	case report.ChatMetadata != nil:
		return report.ChatMetadata.WorkspaceID
	case report.RetrievalMetadata != nil:
		return report.RetrievalMetadata.WorkspaceID
	case report.Advisory != nil:
		return report.Advisory.WorkspaceID
	default:
		return ""
	}
}

func requestFromTypedReport(report DiagnosticReport) string {
	switch {
	case report.RouteEnvelope != nil:
		return report.RouteEnvelope.RequestID
	case report.ChatMetadata != nil:
		return report.ChatMetadata.RequestID
	case report.RetrievalMetadata != nil:
		return report.RetrievalMetadata.RequestID
	case report.Advisory != nil:
		return report.Advisory.RequestID
	default:
		return ""
	}
}

func correlationFromTypedReport(report DiagnosticReport) string {
	switch {
	case report.RouteEnvelope != nil:
		return report.RouteEnvelope.CorrelationID
	case report.ChatMetadata != nil:
		return report.ChatMetadata.CorrelationID
	case report.RetrievalMetadata != nil:
		return report.RetrievalMetadata.CorrelationID
	case report.Advisory != nil:
		return report.Advisory.CorrelationID
	default:
		return ""
	}
}

func observedAtFromTypedReport(report DiagnosticReport) time.Time {
	switch {
	case report.RouteEnvelope != nil:
		return report.RouteEnvelope.ObservedAt.UTC()
	case report.ChatMetadata != nil:
		return report.ChatMetadata.ObservedAt.UTC()
	case report.RetrievalMetadata != nil:
		return report.RetrievalMetadata.ObservedAt.UTC()
	case report.Advisory != nil:
		return report.Advisory.GeneratedAt.UTC()
	default:
		return time.Time{}
	}
}
