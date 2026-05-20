package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"forge/projectforge/services/core/internal/audit"
	"forge/projectforge/services/core/internal/gateway"
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
	ID                   string                         `json:"id"`
	Query                string                         `json:"query"`
	WorkspaceID          string                         `json:"workspaceId"`
	LaneID               string                         `json:"laneId"`
	SelectedPaths        []string                       `json:"selectedPaths"`
	SnapshotKind         string                         `json:"snapshotKind"`
	SnapshotFingerprint  string                         `json:"snapshotFingerprint"`
	ParentSnapshotID     string                         `json:"parentSnapshotId"`
	RenderArtifactRefID  string                         `json:"renderArtifactRefId"`
	CreatedAtMs          int64                          `json:"createdAtMs"`
	CorrelationID        string                         `json:"correlationId"`
	TraceID              string                         `json:"traceId"`
	SyscallID            string                         `json:"syscallId"`
	AuditID              string                         `json:"auditId"`
	ProposedBy           string                         `json:"proposedBy"`
	CommittedBy          string                         `json:"committedBy"`
	Counts               contextSnapshotInspectorCounts `json:"counts"`
	HasHeader            bool                           `json:"hasHeader"`
	HasGraph             bool                           `json:"hasGraph"`
	HasDelta             bool                           `json:"hasDelta"`
	HasRestoreScores     bool                           `json:"hasRestoreScores"`
	HasResumeHints       bool                           `json:"hasResumeHints"`
	HasRestoreTrace      bool                           `json:"hasRestoreTrace"`
	RestoreTrace         json.RawMessage                `json:"restoreTrace,omitempty"`
	EvidenceClass        string                         `json:"evidenceClass"`
	NonCanonicalEvidence bool                           `json:"nonCanonicalEvidence"`
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
	RestoreTrace        json.RawMessage                 `json:"restoreTrace"`
	RestorePackage      json.RawMessage                 `json:"restorePackage"`
	Metadata            json.RawMessage                 `json:"metadata"`
	IncludedStateIDs    []string                        `json:"includedStateIds"`
	IncludedOpenLoops   []string                        `json:"includedOpenLoops"`
	IncludedNoteIDs     []string                        `json:"includedNoteIds"`
	IncludedLinkIDs     []string                        `json:"includedLinkIds"`
	IncludedModelIDs    []string                        `json:"includedModelIds"`
	IncludedArtifactIDs []string                        `json:"includedArtifactIds"`
	IncludedEventIDs    []string                        `json:"includedEventIds"`
}

type processHealthRuntime struct {
	Available       bool            `json:"available"`
	State           string          `json:"state,omitempty"`
	SafeMode        bool            `json:"safeMode"`
	SafeModeReasons []string        `json:"safeModeReasons,omitempty"`
	RuntimeEnabled  bool            `json:"runtimeEnabled"`
	GPUAware        bool            `json:"gpuAware"`
	Health          json.RawMessage `json:"health,omitempty"`
	Queue           json.RawMessage `json:"queue,omitempty"`
	Loaded          json.RawMessage `json:"loaded,omitempty"`
	Usage           json.RawMessage `json:"usage,omitempty"`
	Error           string          `json:"error,omitempty"`
	Warnings        []string        `json:"warnings,omitempty"`
}

type processHealthInvocation struct {
	CorrelationID string `json:"correlationId"`
	InvocationID  int64  `json:"invocationId"`
	ToolID        string `json:"toolId"`
	Action        string `json:"action"`
	Domain        string `json:"domain"`
	LaneID        string `json:"laneId,omitempty"`
	Initiator     string `json:"initiator"`
	Status        string `json:"status"`
	PolicyOutcome string `json:"policyOutcome"`
	RiskClass     string `json:"riskClass"`
	WriteIntent   bool   `json:"writeIntent"`
	DeniedReason  string `json:"deniedReason,omitempty"`
	StartedAtMs   int64  `json:"startedAtMs"`
	CompletedAtMs int64  `json:"completedAtMs,omitempty"`
	DurationMs    int64  `json:"durationMs,omitempty"`
	TraceID       string `json:"traceId,omitempty"`
}

type processHealthCorrelationReport struct {
	CorrelationID          string                    `json:"correlationId"`
	ProcessInvocations     []processHealthInvocation `json:"processInvocations"`
	TotalInvocations       int                       `json:"totalInvocations"`
	ProcessInvocationCount int                       `json:"processInvocationCount"`
}

type processHealthTraceResponse struct {
	CorrelationIDs []string                         `json:"correlationIds"`
	CorrelationID  string                           `json:"correlationId,omitempty"`
	TraceID        string                           `json:"traceId,omitempty"`
	Reports        []processHealthCorrelationReport `json:"reports"`
	Runtime        processHealthRuntime             `json:"runtime"`
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
	RestoreTraceJSON      string
	RestorePackageJSON    string
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

func isProcessGatewayInvocation(rec gateway.InvocationRecord) bool {
	tool := strings.ToLower(strings.TrimSpace(rec.ToolID))
	domain := strings.ToLower(strings.TrimSpace(rec.Domain))
	if domain == "proc" || domain == "process" {
		return true
	}
	if strings.HasPrefix(tool, "proc.") || strings.HasPrefix(tool, "process.") {
		return true
	}
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(rec.Action)), "process") {
		return true
	}
	return false
}

func normalizeProcessInvocation(rec gateway.InvocationRecord) processHealthInvocation {
	item := processHealthInvocation{
		CorrelationID: strings.TrimSpace(rec.CorrelationID),
		InvocationID:  rec.ID,
		ToolID:        strings.TrimSpace(rec.ToolID),
		Action:        strings.TrimSpace(rec.Action),
		Domain:        strings.TrimSpace(rec.Domain),
		Initiator:     strings.TrimSpace(rec.Initiator),
		Status:        strings.TrimSpace(rec.Status),
		PolicyOutcome: strings.TrimSpace(rec.PolicyOutcome),
		RiskClass:     strings.TrimSpace(rec.RiskClass),
		WriteIntent:   rec.WriteIntent,
		DeniedReason:  strings.TrimSpace(rec.DeniedReason),
		StartedAtMs:   rec.CreatedAtMs,
		TraceID:       strings.TrimSpace(rec.TraceID),
	}
	if rec.LaneID != nil && strings.TrimSpace(*rec.LaneID) != "" {
		item.LaneID = strings.TrimSpace(*rec.LaneID)
	}
	if rec.CompletedAtMs != nil && *rec.CompletedAtMs > 0 {
		item.CompletedAtMs = *rec.CompletedAtMs
		if item.CompletedAtMs >= item.StartedAtMs {
			item.DurationMs = item.CompletedAtMs - item.StartedAtMs
		}
	}
	if item.DurationMs < 0 {
		item.DurationMs = 0
	}
	return item
}

func (s *Server) handleContextSnapshotList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	db, err := s.traceDB()
	if err != nil {
		writeAPIRequestError(w, http.StatusServiceUnavailable, err)
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
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()

	records, err := scanContextSnapshotInspectorRows(rows)
	if err != nil {
		writeAPIRequestError(w, http.StatusInternalServerError, err)
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
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		writeAPIError(w, http.StatusBadRequest, "request_failed", "snapshot id is required", nil)
		return
	}
	record, ok, err := s.getContextSnapshotInspectorRow(ctx, id, strings.TrimSpace(r.URL.Query().Get("workspaceId")), strings.TrimSpace(r.URL.Query().Get("laneId")), false)
	if err != nil {
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		writeAPIError(w, http.StatusNotFound, "request_failed", "snapshot not found", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"snapshot": detailContextSnapshotRow(record)})
}

func (s *Server) getContextSnapshotInspectorRow(ctx context.Context, id, workspaceID, laneID string, requireWorkspace bool) (contextSnapshotInspectorRow, bool, error) {
	db, err := s.traceDB()
	if err != nil {
		return contextSnapshotInspectorRow{}, false, err
	}
	if requireWorkspace && strings.TrimSpace(workspaceID) == "" {
		return contextSnapshotInspectorRow{}, false, fmt.Errorf("workspaceId required")
	}
	rows, err := db.QueryContext(ctx, `
SELECT id, query, workspace_id, lane_id, snapshot_kind, snapshot_fingerprint, parent_snapshot_id, selected_paths_json,
       included_state_json, included_open_loops_json, included_notes_json, included_links_json, included_models_json,
       included_artifacts_json, included_events_json, header_json, graph_json, delta_json, restore_scores_json,
       render_artifact_ref_id, resume_hints_json, budget_json, inclusion_reasons_json, created_at, correlation_id,
       trace_id, syscall_id, metadata_json, proposed_by, committed_by, audit_id
FROM context_packet_snapshots
WHERE id = ?
  AND (? = '' OR workspace_id = ?)
  AND (? = '' OR lane_id = ?)`, id, strings.TrimSpace(workspaceID), strings.TrimSpace(workspaceID), strings.TrimSpace(laneID), strings.TrimSpace(laneID))
	if err != nil {
		return contextSnapshotInspectorRow{}, false, err
	}
	defer rows.Close()

	records, err := scanContextSnapshotInspectorRows(rows)
	if err != nil {
		return contextSnapshotInspectorRow{}, false, err
	}
	if len(records) == 0 {
		return contextSnapshotInspectorRow{}, false, nil
	}
	return records[0], true, nil
}

func (s *Server) handleContextRestoreRecent(w http.ResponseWriter, r *http.Request) {
	if strings.TrimSpace(r.URL.Query().Get("workspaceId")) == "" {
		writeAPIError(w, http.StatusBadRequest, "request_failed", "workspaceId required", nil)
		return
	}
	if strings.TrimSpace(r.URL.Query().Get("snapshotKind")) == "" {
		next := new(http.Request)
		*next = *r
		nextURL := *r.URL
		query := nextURL.Query()
		query.Set("snapshotKind", "restore")
		nextURL.RawQuery = query.Encode()
		next.URL = &nextURL
		r = next
	}
	s.handleContextSnapshotList(w, r)
}

func (s *Server) handleContextRestoreGet(w http.ResponseWriter, r *http.Request) {
	record, ok := s.loadScopedRestoreSnapshot(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"snapshot":                detailContextSnapshotRow(record),
		"evidenceClass":           "non_canonical_evidence",
		"nonCanonicalEvidence":    true,
		"canonicalWriteCommitted": false,
	})
}

func (s *Server) handleContextRestoreCandidates(w http.ResponseWriter, r *http.Request) {
	record, ok := s.loadScopedRestoreSnapshot(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"snapshotId":              record.ID,
		"candidates":              restoreScoreCandidatesJSON(record.RestoreScoresJSON),
		"score":                   rawJSONOrDefault(record.RestoreScoresJSON, "{}"),
		"evidenceClass":           "non_canonical_evidence",
		"nonCanonicalEvidence":    true,
		"canonicalWriteCommitted": false,
	})
}

func (s *Server) handleContextRestoreScore(w http.ResponseWriter, r *http.Request) {
	record, ok := s.loadScopedRestoreSnapshot(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"snapshotId":                 record.ID,
		"score":                      rawJSONOrDefault(record.RestoreScoresJSON, "{}"),
		"scoreBreakdown":             restoreScoreCandidatesJSON(record.RestoreScoresJSON),
		"restorePackage":             rawJSONOrDefault(record.RestorePackageJSON, "{}"),
		"resumeHints":                rawJSONOrDefault(record.ResumeHintsJSON, "{}"),
		"requiresFreshCompile":       restoreRequiresFreshCompile(record.RestoreScoresJSON, record.ResumeHintsJSON),
		"requiresFreshCompileReason": restoreFreshCompileReason(record.RestoreScoresJSON, record.ResumeHintsJSON),
		"renderArtifactRefId":        record.RenderArtifactRefID,
		"evidenceClass":              "non_canonical_evidence",
		"nonCanonicalEvidence":       true,
		"canonicalWriteCommitted":    false,
	})
}

func (s *Server) handleContextRestoreResumeHints(w http.ResponseWriter, r *http.Request) {
	record, ok := s.loadScopedRestoreSnapshot(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"snapshotId":                 record.ID,
		"resumeHints":                rawJSONOrDefault(record.ResumeHintsJSON, "{}"),
		"requiresFreshCompile":       restoreRequiresFreshCompile(record.RestoreScoresJSON, record.ResumeHintsJSON),
		"requiresFreshCompileReason": restoreFreshCompileReason(record.RestoreScoresJSON, record.ResumeHintsJSON),
		"evidenceClass":              "non_canonical_evidence",
		"nonCanonicalEvidence":       true,
		"canonicalWriteCommitted":    false,
	})
}

func (s *Server) loadScopedRestoreSnapshot(w http.ResponseWriter, r *http.Request) (contextSnapshotInspectorRow, bool) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		writeAPIError(w, http.StatusBadRequest, "request_failed", "snapshot id is required", nil)
		return contextSnapshotInspectorRow{}, false
	}
	record, found, err := s.getContextSnapshotInspectorRow(r.Context(), id, strings.TrimSpace(r.URL.Query().Get("workspaceId")), strings.TrimSpace(r.URL.Query().Get("laneId")), true)
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "workspaceId required") {
			status = http.StatusBadRequest
		}
		writeAPIRequestError(w, status, err)
		return contextSnapshotInspectorRow{}, false
	}
	if !found {
		writeAPIError(w, http.StatusNotFound, "request_failed", "restore snapshot not found", nil)
		return contextSnapshotInspectorRow{}, false
	}
	return record, true
}

func (s *Server) handleProcessHealthTrace(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	correlationID := strings.TrimSpace(r.URL.Query().Get("correlationId"))
	traceID := strings.TrimSpace(r.URL.Query().Get("traceId"))
	if correlationID == "" && traceID == "" {
		writeAPIError(w, http.StatusBadRequest, "request_failed", "correlationId or traceId is required", nil)
		return
	}

	correlationIDs := []string{}
	if correlationID != "" {
		correlationIDs = append(correlationIDs, correlationID)
	} else {
		ids, err := s.listCorrelationIDsByTraceID(ctx, traceID)
		if err != nil {
			writeAPIRequestError(w, http.StatusInternalServerError, err)
			return
		}
		correlationIDs = ids
	}
	if len(correlationIDs) == 0 {
		writeAPIError(w, http.StatusNotFound, "request_failed", "no correlated correlation ids", nil)
		return
	}
	if s.gateway == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "request_failed", "gateway unavailable", nil)
		return
	}

	reports := make([]processHealthCorrelationReport, 0, len(correlationIDs))
	for _, cid := range correlationIDs {
		allInvocations, err := s.gateway.ListInvocationsByCorrelation(ctx, cid, 500)
		if err != nil {
			writeAPIRequestError(w, http.StatusInternalServerError, err)
			return
		}
		items := make([]processHealthInvocation, 0, len(allInvocations))
		for _, inv := range allInvocations {
			if isProcessGatewayInvocation(inv) {
				items = append(items, normalizeProcessInvocation(inv))
			}
		}
		reports = append(reports, processHealthCorrelationReport{
			CorrelationID:          cid,
			ProcessInvocations:     items,
			TotalInvocations:       len(allInvocations),
			ProcessInvocationCount: len(items),
		})
	}

	out := processHealthTraceResponse{
		CorrelationIDs: correlationIDs,
		TraceID:        traceID,
		Runtime: processHealthRuntime{
			Available:      false,
			State:          "unavailable",
			SafeMode:       s.cfg.SafeModeForceCPUOnly,
			RuntimeEnabled: s.modelRuntime != nil,
			GPUAware:       s.cfg.GPUEnabled && !s.cfg.SafeModeForceCPUOnly,
		},
		Reports: reports,
	}
	if s.cfg.SafeModeForceCPUOnly {
		out.Runtime.SafeModeReasons = append(out.Runtime.SafeModeReasons, "safe_mode.force_cpu_only is enabled")
	}
	if correlationID != "" {
		out.CorrelationID = correlationID
	}

	if s.modelRuntime != nil {
		out.Runtime.Available = true
		rtMeta := modelRuntimeMetaFromRequestAudit(requestAuditMetaForBackup(r, correlationID, traceID, "", "operator.process.health"))
		health, err := s.modelRuntime.Health(ctx, rtMeta)
		if err != nil {
			out.Runtime.Warnings = append(out.Runtime.Warnings, "health: "+err.Error())
		} else {
			out.Runtime.State = strings.TrimSpace(health.Status)
			out.Runtime.RuntimeEnabled = health.RuntimeEnabled
			out.Runtime.GPUAware = health.GPUAware
			out.Runtime.SafeMode = s.cfg.SafeModeForceCPUOnly || !health.GPUAware
			out.Runtime.SafeModeReasons = append(out.Runtime.SafeModeReasons, health.DegradedReasons...)
			out.Runtime.Warnings = append(out.Runtime.Warnings, health.PolicyWarnings...)
			healthBytes, _ := json.Marshal(health)
			out.Runtime.Health = rawJSONOrDefault(string(healthBytes), "null")
		}

		queue, err := s.modelRuntime.QueueStatus(ctx, rtMeta)
		if err != nil {
			out.Runtime.Warnings = append(out.Runtime.Warnings, "queue: "+err.Error())
		} else {
			queueBytes, _ := json.Marshal(queue)
			out.Runtime.Queue = rawJSONOrDefault(string(queueBytes), "null")
		}

		loaded, err := s.modelRuntime.LoadedStatus(ctx, rtMeta)
		if err != nil {
			out.Runtime.Warnings = append(out.Runtime.Warnings, "loaded: "+err.Error())
		} else {
			loadedBytes, _ := json.Marshal(loaded)
			out.Runtime.Loaded = rawJSONOrDefault(string(loadedBytes), "null")
		}

		usage, err := s.modelRuntime.Usage(ctx, rtMeta)
		if err != nil {
			out.Runtime.Warnings = append(out.Runtime.Warnings, "usage: "+err.Error())
		} else {
			usageBytes, _ := json.Marshal(usage)
			out.Runtime.Usage = rawJSONOrDefault(string(usageBytes), "null")
		}
	}

	if len(out.Runtime.Warnings) > 0 {
		out.Runtime.Error = strings.Join(out.Runtime.Warnings, "; ")
	}

	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleAuditTraceLookup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	correlationID := strings.TrimSpace(r.URL.Query().Get("correlationId"))
	traceID := strings.TrimSpace(r.URL.Query().Get("traceId"))
	if correlationID == "" && traceID == "" {
		writeAPIError(w, http.StatusBadRequest, "request_failed", "correlationId or traceId is required", nil)
		return
	}
	if correlationID != "" {
		result, err := s.buildAuditTraceLookupResult(ctx, correlationID)
		if err != nil {
			writeAPIRequestError(w, http.StatusInternalServerError, err)
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
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	reports := make([]auditTraceLookupResult, 0, len(correlationIDs))
	for _, candidate := range correlationIDs {
		result, buildErr := s.buildAuditTraceLookupResult(ctx, candidate)
		if buildErr != nil {
			writeAPIInternalError(w, buildErr)
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
		row.RestoreTraceJSON = extractMetadataFieldJSON(row.MetadataJSON, "restore_trace_json")
		if !hasStructuredJSON(row.RestoreTraceJSON) {
			row.RestoreTraceJSON = extractMetadataFieldJSON(row.MetadataJSON, "restore_trace")
		}
		row.RestorePackageJSON = extractMetadataFieldJSON(row.MetadataJSON, "restore_package_json")
		if !hasStructuredJSON(row.RestorePackageJSON) {
			row.RestorePackageJSON = extractMetadataFieldJSON(row.MetadataJSON, "restore_package")
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
		HasHeader:            hasStructuredJSON(row.HeaderJSON),
		HasGraph:             hasStructuredJSON(row.GraphJSON),
		HasDelta:             hasStructuredJSON(row.DeltaJSON),
		HasRestoreScores:     hasStructuredJSON(row.RestoreScoresJSON),
		HasResumeHints:       hasStructuredJSON(row.ResumeHintsJSON),
		HasRestoreTrace:      hasStructuredJSON(row.RestoreTraceJSON),
		RestoreTrace:         rawJSONOrDefault(row.RestoreTraceJSON, "{}"),
		EvidenceClass:        "non_canonical_evidence",
		NonCanonicalEvidence: true,
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
		RestoreTrace:        rawJSONOrDefault(row.RestoreTraceJSON, "{}"),
		RestorePackage:      rawJSONOrDefault(row.RestorePackageJSON, "{}"),
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

func extractMetadataFieldJSON(rawJSON, field string) string {
	trimmed := strings.TrimSpace(rawJSON)
	if trimmed == "" || trimmed == "null" {
		return "{}"
	}
	var metadata map[string]any
	if err := json.Unmarshal([]byte(trimmed), &metadata); err != nil || len(metadata) == 0 {
		return "{}"
	}
	v, ok := metadata[field]
	if !ok || v == nil {
		return "{}"
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func rawJSONOrDefault(raw, fallback string) json.RawMessage {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		trimmed = fallback
	}
	return json.RawMessage(trimmed)
}

func restoreScoreCandidatesJSON(raw string) json.RawMessage {
	record := map[string]any{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &record); err != nil {
		return json.RawMessage("[]")
	}
	for _, key := range []string{"scores", "score_breakdown", "scoreBreakdown", "candidates"} {
		if value, ok := record[key]; ok {
			if encoded, err := json.Marshal(value); err == nil && strings.TrimSpace(string(encoded)) != "null" {
				return json.RawMessage(encoded)
			}
		}
	}
	return json.RawMessage("[]")
}

func restoreRequiresFreshCompile(scoresJSON, hintsJSON string) bool {
	for _, raw := range []string{hintsJSON, scoresJSON} {
		record := map[string]any{}
		if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &record); err != nil {
			continue
		}
		for _, key := range []string{"requires_fresh_compile", "requiresFreshCompile"} {
			if value, ok := record[key]; ok {
				switch typed := value.(type) {
				case bool:
					return typed
				case string:
					parsed, _ := strconv.ParseBool(strings.TrimSpace(typed))
					return parsed
				}
			}
		}
	}
	return false
}

func restoreFreshCompileReason(scoresJSON, hintsJSON string) string {
	for _, raw := range []string{hintsJSON, scoresJSON} {
		record := map[string]any{}
		if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &record); err != nil {
			continue
		}
		for _, key := range []string{"requires_fresh_compile_reason", "requiresFreshCompileReason", "fresh_compile_reason", "freshCompileReason", "reason"} {
			if value, ok := record[key]; ok {
				if text := strings.TrimSpace(fmt.Sprint(value)); text != "" && text != "<nil>" {
					return text
				}
			}
		}
		if restoreRequiresFreshCompile(scoresJSON, hintsJSON) {
			if decision := strings.TrimSpace(fmt.Sprint(record["decision"])); decision != "" && decision != "<nil>" {
				return "restore decision: " + decision
			}
		}
	}
	if restoreRequiresFreshCompile(scoresJSON, hintsJSON) {
		return "restore metadata requires fresh compile"
	}
	return ""
}
