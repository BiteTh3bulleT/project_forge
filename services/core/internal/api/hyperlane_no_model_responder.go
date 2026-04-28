package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"forge/projectforge/services/core/internal/aios/hyperlane"
	"forge/projectforge/services/core/internal/chat"
	"forge/projectforge/services/core/internal/gateway"
)

type hyperlaneResponderResult struct {
	Content                 string
	Route                   string
	IntentType              string
	Confidence              float64
	MatchedRule             string
	ContextCompileAvoided   bool
	ContextMetadataRead     bool
	StructuredStateReadOnly bool
	Details                 map[string]any
}

func (s *Server) maybeRespondHyperlaneNoModel(
	ctx context.Context,
	threadID, userMessageID int64,
	lastUserContent string,
) (*chat.Message, bool) {
	start := time.Now()
	intent := gateway.ParseHyperlaneIntent(lastUserContent)
	if intent.RequiresModel || !gateway.SupportsNoModelHyperlaneRoute(intent) {
		return nil, false
	}

	result := s.respondHyperlaneNoModel(ctx, intent)
	latency := time.Since(start).Milliseconds()
	latencyTrace := map[string]any{
		"total_request_ms":        latency,
		"hyperlane_ms":            latency,
		"restore_ms":              int64(0),
		"context_compile_ms":      int64(0),
		"modelruntime_ms":         int64(0),
		"gateway_preflight_ms":    int64(0),
		"gateway_execution_ms":    int64(0),
		"model_calls_avoided":     1,
		"fresh_compile_avoided":   0,
		"tokens_estimated":        0,
		"context_budget_class":    "tiny",
		"output_mode":             "brief",
		"hyperlane_intent_type":   result.IntentType,
		"hyperlane_route":         result.Route,
		"hyperlane_confidence":    result.Confidence,
		"hyperlane_matched_rule":  result.MatchedRule,
		"modelruntime_avoided":    true,
		"context_compile_avoided": result.ContextCompileAvoided,
		"gateway_avoided":         true,
	}
	metadata := map[string]any{
		"replyToUserMessageId":       userMessageID,
		"hyperlane_intent_type":      result.IntentType,
		"hyperlane_route":            result.Route,
		"hyperlane_confidence":       result.Confidence,
		"hyperlane_matched_rule":     result.MatchedRule,
		"modelruntime_avoided":       true,
		"context_compile_avoided":    result.ContextCompileAvoided,
		"gateway_avoided":            true,
		"latency_ms":                 latency,
		"hyperlaneNoModel":           true,
		"contextMetadataRead":        result.ContextMetadataRead,
		"structuredStateReadOnly":    result.StructuredStateReadOnly,
		"canonicalWriteCommitted":    false,
		"nonCanonicalEvidenceOnly":   true,
		"toolGatewayActivity":        map[string]any{"executionState": "skipped", "toolCallEmitted": false, "reason": "hyperlane_no_model_structured_response"},
		"modelRuntimeActivity":       map[string]any{"executionState": "skipped", "reason": "hyperlane_no_model_structured_response"},
		"hyperlane_structured_route": result.Details,
		"chatLatencyTrace":           latencyTrace,
	}

	am, err := s.chat.AppendMessage(ctx, threadID, "assistant", result.Content, metadata)
	if err != nil {
		return nil, true
	}
	_ = s.log.Emit(ctx, "chat.message.assistant", map[string]any{
		"threadId":            threadID,
		"messageId":           am.ID,
		"ok":                  true,
		"hyperlaneRoute":      result.Route,
		"modelruntimeAvoided": true,
		"gatewayAvoided":      true,
	})
	return am, true
}

func (s *Server) respondHyperlaneNoModel(ctx context.Context, intent hyperlane.Intent) hyperlaneResponderResult {
	base := hyperlaneResponderResult{
		Route:                   intent.Route,
		IntentType:              string(intent.Type),
		Confidence:              intent.Confidence,
		MatchedRule:             intent.MatchedRule,
		ContextCompileAvoided:   true,
		StructuredStateReadOnly: true,
		Details:                 map[string]any{},
	}
	switch intent.Route {
	case hyperlane.RouteStatusQuery:
		return s.hyperlaneStatusResponse(ctx, base)
	case hyperlane.RouteDiagnosticsQuery:
		return s.hyperlaneDiagnosticsResponse(ctx, base)
	case hyperlane.RouteModelRuntimeStatus:
		return s.hyperlaneModelRuntimeStatusResponse(ctx, base)
	case hyperlane.RouteRestoreInspection:
		base.ContextMetadataRead = true
		return s.hyperlaneRestoreInspectionResponse(ctx, base)
	case hyperlane.RouteDreamReportInspection:
		base.ContextMetadataRead = true
		return s.hyperlaneDreamReportInspectionResponse(ctx, base)
	default:
		base.Content = "Hyperlane could not resolve a supported no-model route. Falling back to normal assistant handling."
		return base
	}
}

func (s *Server) hyperlaneStatusResponse(ctx context.Context, result hyperlaneResponderResult) hyperlaneResponderResult {
	safeModeReasons := []string{}
	if s.cfg.SafeModeForceCPUOnly {
		safeModeReasons = append(safeModeReasons, "safe_mode.force_cpu_only is enabled")
	}
	runtimeSummary := s.readModelRuntimeStructuredSummary(ctx)
	result.Details = map[string]any{
		"core":                "available",
		"cpuAuthoritative":    true,
		"safeMode":            s.cfg.SafeModeForceCPUOnly,
		"safeModeReasons":     safeModeReasons,
		"modelRuntimeSummary": runtimeSummary,
		"workspaceScoped":     strings.TrimSpace(s.cfg.WorkspaceDir) != "",
	}

	var b strings.Builder
	b.WriteString("FORGE status: core available; CPU/RAM authority is active.")
	if s.cfg.SafeModeForceCPUOnly {
		b.WriteString(" Safe mode is active")
		if len(safeModeReasons) > 0 {
			b.WriteString(" (" + strings.Join(safeModeReasons, "; ") + ")")
		}
		b.WriteString(".")
	} else {
		b.WriteString(" Safe mode is not forced.")
	}
	b.WriteString(fmt.Sprintf(" Model runtime service configured: %t; registry status rows: %d; loaded rows: %d. Fast path: no model call.", s.modelRuntime != nil, runtimeSummary.RegistryStatusRows, runtimeSummary.ActiveLoadedRows))
	result.Content = b.String()
	return result
}

func (s *Server) hyperlaneDiagnosticsResponse(ctx context.Context, result hyperlaneResponderResult) hyperlaneResponderResult {
	events := s.countRecentEvents(ctx)
	gatewayCounts := s.countRecentGatewayStates(ctx)
	result.Details = map[string]any{
		"recentEvents":       events,
		"recentGatewayState": gatewayCounts,
		"diagnosticsRoute":   "/api/process/health?correlationId=<id>",
	}
	result.Content = fmt.Sprintf(
		"Diagnostics fast path. Diagnostics summary: recent event rows=%d, recent error events=%d, gateway invocations=%d, denied=%d, needs_approval=%d. For a process trace, use `/api/process/health?correlationId=<id>` or `traceId=<id>`.",
		events.Total,
		events.Errors,
		gatewayCounts.Total,
		gatewayCounts.Denied,
		gatewayCounts.NeedsApproval,
	)
	return result
}

func (s *Server) hyperlaneModelRuntimeStatusResponse(ctx context.Context, result hyperlaneResponderResult) hyperlaneResponderResult {
	summary := s.readModelRuntimeStructuredSummary(ctx)
	result.Details = map[string]any{
		"serviceConfigured":  s.modelRuntime != nil,
		"runtimeEnabled":     s.cfg.EnableModelRuntime,
		"defaultModelId":     strings.TrimSpace(s.cfg.ModelDefaultID),
		"safeMode":           s.cfg.SafeModeForceCPUOnly,
		"registryStatusRows": summary.RegistryStatusRows,
		"activeLoadedRows":   summary.ActiveLoadedRows,
		"latestLoadedModel":  summary.LatestLoadedModel,
		"latestLoadedStatus": summary.LatestLoadedStatus,
		"queue":              "not queried on no-model path",
		"health":             "not queried on no-model path",
		"warnings":           summary.Warnings,
	}
	result.Content = fmt.Sprintf(
		"Modelruntime fast path. Model runtime status: service configured=%t, runtime enabled=%t, safe mode=%t. Registry status rows=%d; active loaded rows=%d; latest loaded model=%s (%s). Queue/health were not queried on the no-model path.",
		s.modelRuntime != nil,
		s.cfg.EnableModelRuntime,
		s.cfg.SafeModeForceCPUOnly,
		summary.RegistryStatusRows,
		summary.ActiveLoadedRows,
		emptyDash(summary.LatestLoadedModel),
		emptyDash(summary.LatestLoadedStatus),
	)
	return result
}

func (s *Server) hyperlaneRestoreInspectionResponse(ctx context.Context, result hyperlaneResponderResult) hyperlaneResponderResult {
	row, found, err := s.latestRestoreSnapshot(ctx)
	if err != nil {
		result.Details = map[string]any{"error": err.Error(), "workspaceId": strings.TrimSpace(s.cfg.WorkspaceDir)}
		result.Content = "Restore inspection: structured restore state is unavailable: " + err.Error()
		return result
	}
	if !found {
		result.Details = map[string]any{"count": 0, "workspaceId": strings.TrimSpace(s.cfg.WorkspaceDir)}
		result.Content = "No restore package is attached to this chat context. Restore inspection: no restore snapshots were found for the current workspace."
		return result
	}
	summary := summarizeContextSnapshotRow(row)
	result.Details = map[string]any{
		"snapshotId":           row.ID,
		"workspaceId":          row.WorkspaceID,
		"laneId":               row.LaneID,
		"createdAtMs":          row.CreatedAtMs,
		"hasRestoreScores":     summary.HasRestoreScores,
		"hasResumeHints":       summary.HasResumeHints,
		"requiresFreshCompile": restoreRequiresFreshCompile(row.RestoreScoresJSON, row.ResumeHintsJSON),
		"freshCompileReason":   restoreFreshCompileReason(row.RestoreScoresJSON, row.ResumeHintsJSON),
		"evidenceClass":        "non_canonical_evidence",
	}
	result.Content = fmt.Sprintf(
		"Restore inspection: latest restore snapshot `%s` for current workspace, created %d. Restore scores=%t, resume hints=%t, requires fresh compile=%t%s.",
		row.ID,
		row.CreatedAtMs,
		summary.HasRestoreScores,
		summary.HasResumeHints,
		restoreRequiresFreshCompile(row.RestoreScoresJSON, row.ResumeHintsJSON),
		formatReasonSuffix(restoreFreshCompileReason(row.RestoreScoresJSON, row.ResumeHintsJSON)),
	)
	return result
}

func (s *Server) hyperlaneDreamReportInspectionResponse(ctx context.Context, result hyperlaneResponderResult) hyperlaneResponderResult {
	rec, count, found, err := s.latestDreamReport(ctx)
	if err != nil {
		result.Details = map[string]any{"error": err.Error(), "workspaceId": strings.TrimSpace(s.cfg.WorkspaceDir)}
		result.Content = "Dream report inspection: structured Dream report state is unavailable: " + err.Error()
		return result
	}
	if !found {
		result.Details = map[string]any{"count": count, "workspaceId": strings.TrimSpace(s.cfg.WorkspaceDir)}
		result.Content = "Dream Mode report inspection: no Dream reports were found for the current workspace."
		return result
	}
	result.Details = map[string]any{
		"count":                   count,
		"reportId":                rec.ID,
		"workspaceId":             rec.WorkspaceID,
		"laneId":                  rec.LaneID,
		"mode":                    rec.Mode,
		"dryRun":                  rec.DryRun,
		"status":                  rec.Status,
		"candidatesConsidered":    rec.CandidatesConsidered,
		"proposalsGenerated":      rec.ProposalsGenerated,
		"nonCanonicalEvidence":    rec.NonCanonicalEvidence,
		"canonicalWriteCommitted": rec.CanonicalWriteCommitted,
	}
	result.Content = fmt.Sprintf(
		"Dream Mode report inspection: %d report(s) for current workspace. Latest `%s` is %s/%s, dry-run=%t, candidates=%d, proposals=%d.",
		count,
		rec.ID,
		rec.Mode,
		rec.Status,
		rec.DryRun,
		rec.CandidatesConsidered,
		rec.ProposalsGenerated,
	)
	return result
}

type recentEventCounts struct {
	Total  int `json:"total"`
	Errors int `json:"errors"`
}

func (s *Server) countRecentEvents(ctx context.Context) recentEventCounts {
	out := recentEventCounts{}
	if s == nil || s.st == nil || s.st.DB == nil {
		return out
	}
	_ = s.st.DB.QueryRowContext(ctx, `SELECT COUNT(1) FROM events WHERE created_at >= ?`, time.Now().Add(-24*time.Hour).UnixMilli()).Scan(&out.Total)
	_ = s.st.DB.QueryRowContext(ctx, `SELECT COUNT(1) FROM events WHERE created_at >= ? AND lower(type) LIKE '%error%'`, time.Now().Add(-24*time.Hour).UnixMilli()).Scan(&out.Errors)
	return out
}

type recentGatewayCounts struct {
	Total         int `json:"total"`
	Denied        int `json:"denied"`
	NeedsApproval int `json:"needsApproval"`
}

func (s *Server) countRecentGatewayStates(ctx context.Context) recentGatewayCounts {
	out := recentGatewayCounts{}
	if s == nil || s.st == nil || s.st.DB == nil {
		return out
	}
	since := time.Now().Add(-24 * time.Hour).UnixMilli()
	_ = s.st.DB.QueryRowContext(ctx, `SELECT COUNT(1) FROM gateway_invocations WHERE created_at >= ?`, since).Scan(&out.Total)
	_ = s.st.DB.QueryRowContext(ctx, `SELECT COUNT(1) FROM gateway_invocations WHERE created_at >= ? AND status = 'denied'`, since).Scan(&out.Denied)
	_ = s.st.DB.QueryRowContext(ctx, `SELECT COUNT(1) FROM gateway_invocations WHERE created_at >= ? AND status = 'needs_approval'`, since).Scan(&out.NeedsApproval)
	return out
}

type modelRuntimeStructuredSummary struct {
	RegistryStatusRows int      `json:"registryStatusRows"`
	ActiveLoadedRows   int      `json:"activeLoadedRows"`
	LatestLoadedModel  string   `json:"latestLoadedModel,omitempty"`
	LatestLoadedStatus string   `json:"latestLoadedStatus,omitempty"`
	Warnings           []string `json:"warnings,omitempty"`
}

func (s *Server) readModelRuntimeStructuredSummary(ctx context.Context) modelRuntimeStructuredSummary {
	out := modelRuntimeStructuredSummary{}
	if s == nil || s.st == nil || s.st.DB == nil {
		out.Warnings = append(out.Warnings, "store unavailable")
		return out
	}
	if err := s.st.DB.QueryRowContext(ctx, `SELECT COUNT(1) FROM model_registry_status`).Scan(&out.RegistryStatusRows); err != nil {
		out.Warnings = append(out.Warnings, "model_registry_status: "+err.Error())
	}
	if err := s.st.DB.QueryRowContext(ctx, `SELECT COUNT(1) FROM model_runtime_loads WHERE unloaded_at IS NULL`).Scan(&out.ActiveLoadedRows); err != nil {
		out.Warnings = append(out.Warnings, "model_runtime_loads: "+err.Error())
	}
	err := s.st.DB.QueryRowContext(ctx, `SELECT model_id, status FROM model_runtime_loads ORDER BY loaded_at DESC, id DESC LIMIT 1`).Scan(&out.LatestLoadedModel, &out.LatestLoadedStatus)
	if err != nil && err != sql.ErrNoRows {
		out.Warnings = append(out.Warnings, "latest model_runtime_loads: "+err.Error())
	}
	return out
}

func (s *Server) latestRestoreSnapshot(ctx context.Context) (contextSnapshotInspectorRow, bool, error) {
	db, err := s.traceDB()
	if err != nil {
		return contextSnapshotInspectorRow{}, false, err
	}
	rows, err := db.QueryContext(ctx, `
SELECT id, query, workspace_id, lane_id, snapshot_kind, snapshot_fingerprint, parent_snapshot_id, selected_paths_json,
       included_state_json, included_open_loops_json, included_notes_json, included_links_json, included_models_json,
       included_artifacts_json, included_events_json, header_json, graph_json, delta_json, restore_scores_json,
       render_artifact_ref_id, resume_hints_json, budget_json, inclusion_reasons_json, created_at, correlation_id,
       trace_id, syscall_id, metadata_json, proposed_by, committed_by, audit_id
FROM context_packet_snapshots
WHERE workspace_id = ? AND snapshot_kind = 'restore'
ORDER BY created_at DESC, id DESC
LIMIT 1`, strings.TrimSpace(s.cfg.WorkspaceDir))
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

func (s *Server) latestDreamReport(ctx context.Context) (dreamReportNoModelRecord, int, bool, error) {
	var count int
	workspaceID := strings.TrimSpace(s.cfg.WorkspaceDir)
	if s == nil || s.st == nil || s.st.DB == nil {
		return dreamReportNoModelRecord{}, 0, false, fmt.Errorf("store unavailable")
	}
	if err := s.st.DB.QueryRowContext(ctx, `SELECT COUNT(1) FROM dream_reports WHERE workspace_id = ?`, workspaceID).Scan(&count); err != nil {
		return dreamReportNoModelRecord{}, 0, false, err
	}
	row := s.st.DB.QueryRowContext(ctx, `SELECT id,created_at,completed_at,workspace_id,lane_id,mode,dry_run,status,
time_window_start,time_window_end,candidates_considered,proposals_generated,summary_json,
candidates_json,salience_scores_json,memory_tier_proposals_json,repair_proposals_json,
snapshot_hygiene_proposals_json,warnings_json,trace_json,correlation_id,trace_id,
COALESCE(syscall_id,''),COALESCE(audit_id,''),proposed_by,committed_by,metadata_json
FROM dream_reports WHERE workspace_id = ? ORDER BY created_at DESC, id DESC LIMIT 1`, workspaceID)
	rec, err := scanDreamReportNoModel(row)
	if err == sql.ErrNoRows {
		return dreamReportNoModelRecord{}, count, false, nil
	}
	if err != nil {
		return dreamReportNoModelRecord{}, count, false, err
	}
	return rec, count, true, nil
}

type dreamReportNoModelRecord struct {
	ID                      string
	WorkspaceID             string
	LaneID                  string
	Mode                    string
	DryRun                  bool
	Status                  string
	CandidatesConsidered    int
	ProposalsGenerated      int
	NonCanonicalEvidence    bool
	CanonicalWriteCommitted bool
}

func scanDreamReportNoModel(row *sql.Row) (dreamReportNoModelRecord, error) {
	var rec dreamReportNoModelRecord
	var createdAt, completedAt, windowStart, windowEnd int64
	var dryRun int
	var summaryJSON, candidatesJSON, scoresJSON, tierJSON, repairJSON, snapshotJSON, warningsJSON, traceJSON string
	var correlationID, traceID, syscallID, auditID, proposedBy, committedBy, metadataJSON string
	if err := row.Scan(
		&rec.ID, &createdAt, &completedAt, &rec.WorkspaceID, &rec.LaneID, &rec.Mode, &dryRun, &rec.Status,
		&windowStart, &windowEnd, &rec.CandidatesConsidered, &rec.ProposalsGenerated, &summaryJSON,
		&candidatesJSON, &scoresJSON, &tierJSON, &repairJSON, &snapshotJSON, &warningsJSON, &traceJSON,
		&correlationID, &traceID, &syscallID, &auditID, &proposedBy, &committedBy, &metadataJSON,
	); err != nil {
		return dreamReportNoModelRecord{}, err
	}
	rec.DryRun = dryRun != 0
	rec.NonCanonicalEvidence = true
	rec.CanonicalWriteCommitted = false
	return rec, nil
}

func emptyDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return strings.TrimSpace(value)
}

func formatReasonSuffix(reason string) string {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return ""
	}
	return " (" + reason + ")"
}

func jsonObject(raw string) map[string]any {
	out := map[string]any{}
	_ = json.Unmarshal([]byte(strings.TrimSpace(raw)), &out)
	return out
}
