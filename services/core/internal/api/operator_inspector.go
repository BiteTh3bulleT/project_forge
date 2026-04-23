package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"forge/projectforge/services/core/internal/audit"
)

type contextSnapshotInspectorCounts struct {
	State     int `json:"state"`
	OpenLoops int `json:"openLoops"`
	Notes     int `json:"notes"`
	Links     int `json:"links"`
	Models    int `json:"models"`
	Artifacts int `json:"artifacts"`
	Events    int `json:"events"`
}

type contextSnapshotInspectorSummary struct {
	ID                  string                         `json:"id"`
	Query               string                         `json:"query"`
	WorkspaceID         string                         `json:"workspaceId"`
	LaneID              string                         `json:"laneId"`
	SelectedPaths       []string                       `json:"selectedPaths"`
	SnapshotKind        string                         `json:"snapshotKind"`
	SnapshotFingerprint string                         `json:"snapshotFingerprint"`
	ParentSnapshotID    string                         `json:"parentSnapshotId"`
	RenderArtifactRefID string                         `json:"renderArtifactRefId"`
	CreatedAtMs         int64                          `json:"createdAtMs"`
	CorrelationID       string                         `json:"correlationId"`
	TraceID             string                         `json:"traceId"`
	SyscallID           string                         `json:"syscallId"`
	AuditID             string                         `json:"auditId"`
	ProposedBy          string                         `json:"proposedBy"`
	CommittedBy         string                         `json:"committedBy"`
	Counts              contextSnapshotInspectorCounts `json:"counts"`
	HasHeader           bool                           `json:"hasHeader"`
	HasGraph            bool                           `json:"hasGraph"`
	HasDelta            bool                           `json:"hasDelta"`
	HasRestoreScores    bool                           `json:"hasRestoreScores"`
	HasResumeHints      bool                           `json:"hasResumeHints"`
}

type contextSnapshotInspectorDetail struct {
	Summary             contextSnapshotInspectorSummary `json:"summary"`
	Budget              json.RawMessage                 `json:"budget"`
	InclusionReasons    json.RawMessage                 `json:"inclusionReasons"`
	Header              json.RawMessage                 `json:"header"`
	Graph               json.RawMessage                 `json:"graph"`
	Delta               json.RawMessage                 `json:"delta"`
	RestoreScores       json.RawMessage                 `json:"restoreScores"`
	ResumeHints         json.RawMessage                 `json:"resumeHints"`
	Metadata            json.RawMessage                 `json:"metadata"`
	IncludedStateIDs    []string                        `json:"includedStateIds"`
	IncludedOpenLoops   []string                        `json:"includedOpenLoops"`
	IncludedNoteIDs     []string                        `json:"includedNoteIds"`
	IncludedLinkIDs     []string                        `json:"includedLinkIds"`
	IncludedModelIDs    []string                        `json:"includedModelIds"`
	IncludedArtifactIDs []string                        `json:"includedArtifactIds"`
	IncludedEventIDs    []string                        `json:"includedEventIds"`
}

type contextSnapshotInspectorRow struct {
	ID                    string
	Query                 string
	WorkspaceID           string
	LaneID                string
	SnapshotKind          string
	SnapshotFingerprint   string
	ParentSnapshotID      string
	SelectedPathsJSON     string
	IncludedStateJSON     string
	IncludedOpenLoopsJSON string
	IncludedNotesJSON     string
	IncludedLinksJSON     string
	IncludedModelsJSON    string
	IncludedArtifactsJSON string
	IncludedEventsJSON    string
	HeaderJSON            string
	GraphJSON             string
	DeltaJSON             string
	RestoreScoresJSON     string
	RenderArtifactRefID   string
	ResumeHintsJSON       string
	BudgetJSON            string
	InclusionReasonsJSON  string
	CreatedAtMs           int64
	CorrelationID         string
	TraceID               string
	SyscallID             string
	MetadataJSON          string
	ProposedBy            string
	CommittedBy           string
	AuditID               string
}

type auditTraceLookupResult struct {
	CorrelationID string                 `json:"correlationId"`
	Records       []audit.Record         `json:"records"`
	Report        correlationTraceReport `json:"report"`
}

func (s *Server) handleContextSnapshotList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	db, err := s.traceDB()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	limit := parsePositiveLimit(r.URL.Query().Get("limit"), 50, 200)
	workspaceID := strings.TrimSpace(r.URL.Query().Get("workspaceId"))
	laneID := strings.TrimSpace(r.URL.Query().Get("laneId"))
	correlationID := strings.TrimSpace(r.URL.Query().Get("correlationId"))
	snapshotKind := strings.TrimSpace(r.URL.Query().Get("snapshotKind"))
	queryText := strings.TrimSpace(r.URL.Query().Get("query"))

	rows, err := db.QueryContext(ctx, `
SELECT id, query, workspace_id, lane_id, snapshot_kind, snapshot_fingerprint, parent_snapshot_id, selected_paths_json,
       included_state_json, included_open_loops_json, included_notes_json, included_links_json, included_models_json,
       included_artifacts_json, included_events_json, header_json, graph_json, delta_json, restore_scores_json,
       render_artifact_ref_id, resume_hints_json, budget_json, inclusion_reasons_json, created_at, correlation_id,
       trace_id, syscall_id, metadata_json, proposed_by, committed_by, audit_id
FROM context_packet_snapshots
WHERE (? = '' OR workspace_id = ?)
  AND (? = '' OR lane_id = ?)
  AND (? = '' OR correlation_id = ?)
  AND (? = '' OR snapshot_kind = ?)
  AND (? = '' OR query LIKE ?)
ORDER BY created_at DESC, id DESC
LIMIT ?`,
		workspaceID, workspaceID,
		laneID, laneID,
		correlationID, correlationID,
		snapshotKind, snapshotKind,
		queryText, likeQueryValue(queryText),
		limit,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	records, err := scanContextSnapshotInspectorRows(rows)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	out := make([]contextSnapshotInspectorSummary, 0, len(records))
	for _, record := range records {
		out = append(out, summarizeContextSnapshotRow(record))
	}
	writeJSON(w, http.StatusOK, map[string]any{"snapshots": out})
}

func (s *Server) handleContextSnapshotGet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	db, err := s.traceDB()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		http.Error(w, "snapshot id is required", http.StatusBadRequest)
		return
	}
	rows, err := db.QueryContext(ctx, `
SELECT id, query, workspace_id, lane_id, snapshot_kind, snapshot_fingerprint, parent_snapshot_id, selected_paths_json,
       included_state_json, included_open_loops_json, included_notes_json, included_links_json, included_models_json,
       included_artifacts_json, included_events_json, header_json, graph_json, delta_json, restore_scores_json,
       render_artifact_ref_id, resume_hints_json, budget_json, inclusion_reasons_json, created_at, correlation_id,
       trace_id, syscall_id, metadata_json, proposed_by, committed_by, audit_id
FROM context_packet_snapshots
WHERE id = ?`, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	records, err := scanContextSnapshotInspectorRows(rows)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(records) == 0 {
		http.Error(w, "snapshot not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"snapshot": detailContextSnapshotRow(records[0])})
}

func (s *Server) handleAuditTraceLookup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	correlationID := strings.TrimSpace(r.URL.Query().Get("correlationId"))
	traceID := strings.TrimSpace(r.URL.Query().Get("traceId"))
	if correlationID == "" && traceID == "" {
		http.Error(w, "correlationId or traceId is required", http.StatusBadRequest)
		return
	}
	if correlationID != "" {
		result, err := s.buildAuditTraceLookupResult(ctx, correlationID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"mode":          "correlation",
			"correlationId": correlationID,
			"records":       result.Records,
			"report":        result.Report,
			"reports":       []auditTraceLookupResult{result},
		})
		return
	}

	correlationIDs, err := s.listCorrelationIDsByTraceID(ctx, traceID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	reports := make([]auditTraceLookupResult, 0, len(correlationIDs))
	for _, candidate := range correlationIDs {
		result, buildErr := s.buildAuditTraceLookupResult(ctx, candidate)
		if buildErr != nil {
			http.Error(w, buildErr.Error(), http.StatusInternalServerError)
			return
		}
		reports = append(reports, result)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"mode":           "trace",
		"traceId":        traceID,
		"correlationIds": correlationIDs,
		"reports":        reports,
	})
}

func (s *Server) buildAuditTraceLookupResult(ctx context.Context, correlationID string) (auditTraceLookupResult, error) {
	report, err := s.buildCorrelationTraceReport(ctx, correlationID)
	if err != nil {
		return auditTraceLookupResult{}, err
	}
	return auditTraceLookupResult{
		CorrelationID: correlationID,
		Records:       report.AuditRecords,
		Report:        report,
	}, nil
}

func (s *Server) listCorrelationIDsByTraceID(ctx context.Context, traceID string) ([]string, error) {
	db, err := s.traceDB()
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, `
SELECT DISTINCT correlation_id
FROM (
  SELECT correlation_id FROM provenance_records WHERE trace_id = ?
  UNION
  SELECT correlation_id FROM journal_events WHERE trace_id = ?
  UNION
  SELECT correlation_id FROM artifact_refs WHERE trace_id = ?
  UNION
  SELECT correlation_id FROM context_packet_snapshots WHERE trace_id = ?
  UNION
  SELECT correlation_id FROM memory_notes WHERE trace_id = ?
  UNION
  SELECT correlation_id FROM state_items WHERE trace_id = ?
  UNION
  SELECT correlation_id FROM open_loops WHERE trace_id = ?
  UNION
  SELECT correlation_id FROM derived_models WHERE trace_id = ?
  UNION
  SELECT correlation_id FROM contradiction_records WHERE trace_id = ?
  UNION
  SELECT correlation_id FROM supersession_records WHERE trace_id = ?
  UNION
  SELECT correlation_id FROM gateway_invocations WHERE json_extract(scope_json, '$.traceId') = ?
) AS trace_matches
WHERE correlation_id <> ''
ORDER BY correlation_id ASC`,
		traceID, traceID, traceID, traceID, traceID, traceID, traceID, traceID, traceID, traceID, traceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]string, 0, 8)
	for rows.Next() {
		var correlationID string
		if err := rows.Scan(&correlationID); err != nil {
			return nil, err
		}
		correlationID = strings.TrimSpace(correlationID)
		if correlationID != "" {
			out = append(out, correlationID)
		}
	}
	return out, rows.Err()
}

func scanContextSnapshotInspectorRows(rows *sql.Rows) ([]contextSnapshotInspectorRow, error) {
	out := []contextSnapshotInspectorRow{}
	for rows.Next() {
		var row contextSnapshotInspectorRow
		if err := rows.Scan(
			&row.ID,
			&row.Query,
			&row.WorkspaceID,
			&row.LaneID,
			&row.SnapshotKind,
			&row.SnapshotFingerprint,
			&row.ParentSnapshotID,
			&row.SelectedPathsJSON,
			&row.IncludedStateJSON,
			&row.IncludedOpenLoopsJSON,
			&row.IncludedNotesJSON,
			&row.IncludedLinksJSON,
			&row.IncludedModelsJSON,
			&row.IncludedArtifactsJSON,
			&row.IncludedEventsJSON,
			&row.HeaderJSON,
			&row.GraphJSON,
			&row.DeltaJSON,
			&row.RestoreScoresJSON,
			&row.RenderArtifactRefID,
			&row.ResumeHintsJSON,
			&row.BudgetJSON,
			&row.InclusionReasonsJSON,
			&row.CreatedAtMs,
			&row.CorrelationID,
			&row.TraceID,
			&row.SyscallID,
			&row.MetadataJSON,
			&row.ProposedBy,
			&row.CommittedBy,
			&row.AuditID,
		); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func summarizeContextSnapshotRow(row contextSnapshotInspectorRow) contextSnapshotInspectorSummary {
	return contextSnapshotInspectorSummary{
		ID:                  row.ID,
		Query:               row.Query,
		WorkspaceID:         row.WorkspaceID,
		LaneID:              row.LaneID,
		SelectedPaths:       decodeJSONStringSlice(row.SelectedPathsJSON),
		SnapshotKind:        row.SnapshotKind,
		SnapshotFingerprint: row.SnapshotFingerprint,
		ParentSnapshotID:    row.ParentSnapshotID,
		RenderArtifactRefID: row.RenderArtifactRefID,
		CreatedAtMs:         row.CreatedAtMs,
		CorrelationID:       row.CorrelationID,
		TraceID:             row.TraceID,
		SyscallID:           row.SyscallID,
		AuditID:             row.AuditID,
		ProposedBy:          row.ProposedBy,
		CommittedBy:         row.CommittedBy,
		Counts: contextSnapshotInspectorCounts{
			State:     jsonArrayCount(row.IncludedStateJSON),
			OpenLoops: jsonArrayCount(row.IncludedOpenLoopsJSON),
			Notes:     jsonArrayCount(row.IncludedNotesJSON),
			Links:     jsonArrayCount(row.IncludedLinksJSON),
			Models:    jsonArrayCount(row.IncludedModelsJSON),
			Artifacts: jsonArrayCount(row.IncludedArtifactsJSON),
			Events:    jsonArrayCount(row.IncludedEventsJSON),
		},
		HasHeader:        hasStructuredJSON(row.HeaderJSON),
		HasGraph:         hasStructuredJSON(row.GraphJSON),
		HasDelta:         hasStructuredJSON(row.DeltaJSON),
		HasRestoreScores: hasStructuredJSON(row.RestoreScoresJSON),
		HasResumeHints:   hasStructuredJSON(row.ResumeHintsJSON),
	}
}

func detailContextSnapshotRow(row contextSnapshotInspectorRow) contextSnapshotInspectorDetail {
	return contextSnapshotInspectorDetail{
		Summary:             summarizeContextSnapshotRow(row),
		Budget:              rawJSONOrDefault(row.BudgetJSON, "{}"),
		InclusionReasons:    rawJSONOrDefault(row.InclusionReasonsJSON, "{}"),
		Header:              rawJSONOrDefault(row.HeaderJSON, "{}"),
		Graph:               rawJSONOrDefault(row.GraphJSON, "{}"),
		Delta:               rawJSONOrDefault(row.DeltaJSON, "{}"),
		RestoreScores:       rawJSONOrDefault(row.RestoreScoresJSON, "{}"),
		ResumeHints:         rawJSONOrDefault(row.ResumeHintsJSON, "{}"),
		Metadata:            rawJSONOrDefault(row.MetadataJSON, "{}"),
		IncludedStateIDs:    decodeJSONStringSlice(row.IncludedStateJSON),
		IncludedOpenLoops:   decodeJSONStringSlice(row.IncludedOpenLoopsJSON),
		IncludedNoteIDs:     decodeJSONStringSlice(row.IncludedNotesJSON),
		IncludedLinkIDs:     decodeJSONStringSlice(row.IncludedLinksJSON),
		IncludedModelIDs:    decodeJSONStringSlice(row.IncludedModelsJSON),
		IncludedArtifactIDs: decodeJSONStringSlice(row.IncludedArtifactsJSON),
		IncludedEventIDs:    decodeJSONStringSlice(row.IncludedEventsJSON),
	}
}

func parsePositiveLimit(raw string, fallback, max int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return fallback
	}
	if value > max {
		return max
	}
	return value
}

func likeQueryValue(value string) string {
	if value == "" {
		return ""
	}
	return "%" + value + "%"
}

func decodeJSONStringSlice(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || trimmed == "null" {
		return []string{}
	}
	var direct []string
	if err := json.Unmarshal([]byte(trimmed), &direct); err == nil {
		if direct == nil {
			return []string{}
		}
		return direct
	}
	var generic []any
	if err := json.Unmarshal([]byte(trimmed), &generic); err != nil {
		return []string{}
	}
	out := make([]string, 0, len(generic))
	for _, item := range generic {
		if value, ok := item.(string); ok && strings.TrimSpace(value) != "" {
			out = append(out, strings.TrimSpace(value))
		}
	}
	return out
}

func jsonArrayCount(raw string) int {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || trimmed == "null" {
		return 0
	}
	var array []any
	if err := json.Unmarshal([]byte(trimmed), &array); err == nil {
		return len(array)
	}
	var object map[string]any
	if err := json.Unmarshal([]byte(trimmed), &object); err == nil {
		return len(object)
	}
	return 0
}

func hasStructuredJSON(raw string) bool {
	trimmed := strings.TrimSpace(raw)
	switch trimmed {
	case "", "null", "{}", "[]":
		return false
	default:
		return true
	}
}

func rawJSONOrDefault(raw, fallback string) json.RawMessage {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		trimmed = fallback
	}
	return json.RawMessage(trimmed)
}
