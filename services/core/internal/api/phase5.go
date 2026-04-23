package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/audit"
	"forge/projectforge/services/core/internal/backup"
	"forge/projectforge/services/core/internal/gateway"
	"forge/projectforge/services/core/internal/lanes"
	"forge/projectforge/services/core/internal/permissions"
	"forge/projectforge/services/core/internal/release"
)

// Phase 5 handlers: tool execution gateway, action lanes, permission
// profiles, audit traces, backup / export / import, and release readiness.

type gatewayInvokeBody struct {
	ToolID              string         `json:"toolId"`
	LaneID              string         `json:"laneId"`
	Domain              string         `json:"domain"`
	Action              string         `json:"action"`
	RiskClass           string         `json:"riskClass"`
	ExecutionLevel      string         `json:"executionLevel"`
	CorrelationID       string         `json:"correlationId"`
	TraceID             string         `json:"traceId,omitempty"`
	Source              string         `json:"source,omitempty"`
	WorkspaceID         string         `json:"workspaceId,omitempty"`
	IntentID            string         `json:"intentId,omitempty"`
	CharterID           string         `json:"charterId,omitempty"`
	BudgetID            string         `json:"budgetId,omitempty"`
	ApprovalID          string         `json:"approvalId,omitempty"`
	ProvenanceActor     string         `json:"provenanceActor,omitempty"`
	ProvenanceActorType string         `json:"provenanceActorType,omitempty"`
	Paths               []string       `json:"paths"`
	Input               map[string]any `json:"input"`
	JobID               *string        `json:"jobId,omitempty"`
	PacketID            *int64         `json:"packetId,omitempty"`
	DryRun              bool           `json:"dryRun"`
	Initiator           string         `json:"initiator"`
	Metadata            map[string]any `json:"metadata,omitempty"`
}

type gatewayCapabilityStatusUpdateBody struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
}

func requiresCapabilityStatusReason(status domain.ToolCapabilityStatus) bool {
	switch status {
	case domain.ToolCapabilityDisabled, domain.ToolCapabilityStubbed, domain.ToolCapabilityDeferred, domain.ToolCapabilityDeprecated:
		return true
	default:
		return false
	}
}

func (s *Server) handleGatewayTools(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"tools": s.gateway.Tools()})
}

func (s *Server) handleGatewayCapabilities(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"capabilities": s.gateway.Capabilities()})
}

func (s *Server) handleGatewayInvoke(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var body gatewayInvokeBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	result, err := s.gateway.Execute(ctx, gateway.Request{
		ToolID:              body.ToolID,
		LaneID:              body.LaneID,
		Domain:              body.Domain,
		Action:              body.Action,
		RiskClass:           body.RiskClass,
		ExecutionLevel:      body.ExecutionLevel,
		CorrelationID:       body.CorrelationID,
		TraceID:             body.TraceID,
		Source:              body.Source,
		WorkspaceID:         body.WorkspaceID,
		IntentID:            body.IntentID,
		CharterID:           body.CharterID,
		BudgetID:            body.BudgetID,
		ApprovalID:          body.ApprovalID,
		ProvenanceActor:     body.ProvenanceActor,
		ProvenanceActorType: body.ProvenanceActorType,
		Paths:               body.Paths,
		Input:               body.Input,
		JobID:               body.JobID,
		PacketID:            body.PacketID,
		Initiator:           body.Initiator,
		DryRun:              body.DryRun,
		Metadata:            body.Metadata,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = s.log.Emit(ctx, "gateway.tool.invoked", map[string]any{
		"toolId":        body.ToolID,
		"laneId":        body.LaneID,
		"status":        result.Status,
		"correlationId": result.CorrelationID,
	})
	writeJSON(w, http.StatusOK, map[string]any{"result": result})
}

func (s *Server) handleGatewayInvocations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	status := r.URL.Query().Get("status")
	invs, err := s.gateway.ListInvocations(ctx, limit, status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"invocations": invs})
}

func (s *Server) handleGatewayCapabilityStatusUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		http.Error(w, "capability id is required", http.StatusBadRequest)
		return
	}
	var body gatewayCapabilityStatusUpdateBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	status := strings.TrimSpace(strings.ToLower(body.Status))
	if status == "" {
		http.Error(w, "status is required", http.StatusBadRequest)
		return
	}
	parsedStatus := domain.ToolCapabilityStatus(status)
	if !domain.IsKnownToolCapabilityStatus(parsedStatus) {
		http.Error(w, "unknown capability status", http.StatusBadRequest)
		return
	}
	reason := strings.TrimSpace(body.Reason)
	if requiresCapabilityStatusReason(parsedStatus) && reason == "" {
		http.Error(w, "reason is required for deferred/disabled/stubbed/deprecated capability status transitions", http.StatusBadRequest)
		return
	}
	if s.gateway == nil {
		http.Error(w, "gateway unavailable", http.StatusServiceUnavailable)
		return
	}

	previous, updated, ok, err := s.gateway.UpdateCapabilityStatus(id, parsedStatus)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !ok {
		http.Error(w, "capability not found", http.StatusNotFound)
		return
	}

	if s.auditSvc != nil {
		_, _ = s.auditSvc.Record(ctx, audit.CreateRequest{
			Category:    "gateway",
			Action:      "tool.capability.status.updated",
			Actor:       "api",
			SubjectType: "tool_capability",
			SubjectID:   updated.ID,
			Outcome:     "ok",
			Summary:     "tool capability status updated",
			Payload: map[string]any{
				"capabilityId":     updated.ID,
				"previousStatus":   string(previous.Status),
				"newStatus":        string(updated.Status),
				"requestedStatus":  status,
				"transitionReason": reason,
				"requestPath":      r.URL.Path,
			},
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"capability":     updated,
		"previousStatus": string(previous.Status),
		"auditCategory":  "tool.capability.status.updated",
	})
}

func (s *Server) handleListLanes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	list, err := s.lanes.List(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"lanes": list})
}

func (s *Server) handleSaveLane(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var body lanes.Lane
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	saved, err := s.lanes.Save(ctx, body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_ = s.log.Emit(ctx, "gateway.lane.saved", map[string]any{"lane": saved.ID})
	writeJSON(w, http.StatusOK, map[string]any{"lane": saved})
}

func (s *Server) handleDeleteLane(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")
	if err := s.lanes.Delete(ctx, id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListPermissionProfiles(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	list, err := s.permissions.List(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	active, _ := s.permissions.Active(ctx)
	summary, _ := s.permissions.Summary(ctx)
	writeJSON(w, http.StatusOK, map[string]any{
		"profiles": list,
		"active":   active,
		"summary":  summary,
	})
}

func (s *Server) handleSavePermissionProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var body permissions.Profile
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	saved, err := s.permissions.Save(ctx, body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_ = s.log.Emit(ctx, "permissions.profile.saved", map[string]any{"profile": saved.ID, "active": saved.Active})
	writeJSON(w, http.StatusOK, map[string]any{"profile": saved})
}

func (s *Server) handleActivatePermissionProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")
	active, err := s.permissions.Activate(ctx, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_, _ = s.auditSvc.Record(ctx, audit.CreateRequest{
		Category:    "permissions",
		Action:      "profile.activated",
		SubjectType: "profile",
		SubjectID:   active.ID,
		Outcome:     "ok",
		Summary:     "permission profile activated: " + active.Name,
	})
	_ = s.log.Emit(ctx, "permissions.profile.activated", map[string]any{"profile": active.ID})
	writeJSON(w, http.StatusOK, map[string]any{"profile": active})
}

func (s *Server) handleDeletePermissionProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")
	if err := s.permissions.Delete(ctx, id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAuditList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	category := r.URL.Query().Get("category")
	correlation := r.URL.Query().Get("correlationId")
	jobID := r.URL.Query().Get("jobId")
	outcome := r.URL.Query().Get("outcome")
	records, err := s.auditSvc.List(ctx, audit.ListFilter{
		Limit:         limit,
		Category:      category,
		CorrelationID: correlation,
		JobID:         jobID,
		Outcome:       outcome,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"records": records})
}

func (s *Server) handleAuditTrace(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	correlation := strings.TrimSpace(chi.URLParam(r, "correlationId"))
	if correlation == "" {
		http.Error(w, "correlation id is required", http.StatusBadRequest)
		return
	}
	report, err := s.buildCorrelationTraceReport(ctx, correlation)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"correlationId": correlation,
		"records":       report.AuditRecords,
		"report":        report,
	})
}

func (s *Server) handleListBundles(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	list, err := s.backup.List(ctx, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	backupDir, exportDir := s.backup.Dirs()
	writeJSON(w, http.StatusOK, map[string]any{
		"bundles":    list,
		"backupDir":  backupDir,
		"exportDir":  exportDir,
		"knownKinds": backup.KnownKinds,
	})
}

func (s *Server) handleCreateBundle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var body backup.CreateBundleRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	body.Kind = strings.TrimSpace(body.Kind)
	b, err := s.backup.CreateBundle(ctx, body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	meta := requestAuditMetaForBackup(r, "", "", "", "backup.bundle.create")
	_, _ = s.auditSvc.Record(ctx, audit.CreateRequest{
		CorrelationID: meta.CorrelationID,
		Category:      "backup",
		Action:        "bundle.created",
		SubjectType:   "bundle",
		SubjectID:     strconv.FormatInt(b.ID, 10),
		Outcome:       "ok",
		Summary:       "bundle " + b.Kind + " created",
		Payload: requestAuditPayload(map[string]any{
			"kind":        b.Kind,
			"label":       b.Label,
			"file":        b.FilePath,
			"requestPath": r.URL.Path,
		}, meta),
	})
	_ = s.log.Emit(ctx, "backup.bundle.created", map[string]any{"id": b.ID, "kind": b.Kind})
	writeJSON(w, http.StatusOK, map[string]any{"bundle": b})
}

func (s *Server) handleDeleteBundle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	if err := s.backup.Delete(ctx, id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleRestoreBundle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var body backup.RestoreBundleRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	result, err := s.backup.RestoreBundle(ctx, body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	outcome := "ok"
	if len(result.Errors) > 0 || len(result.Unsupported) > 0 {
		outcome = "partial"
	}
	meta := requestAuditMetaForBackup(r, "", "", "", "backup.bundle.restore")
	_, _ = s.auditSvc.Record(ctx, audit.CreateRequest{
		CorrelationID: meta.CorrelationID,
		Category:      "backup",
		Action:        "bundle.restored",
		SubjectType:   "bundle",
		SubjectID:     body.FilePath,
		Outcome:       outcome,
		Summary:       "bundle restored",
		Payload: requestAuditPayload(map[string]any{
			"dryRun":      body.DryRun,
			"file":        body.FilePath,
			"bundleKind":  result.BundleKind,
			"imported":    result.Imported,
			"skipped":     result.Skipped,
			"unsupported": result.Unsupported,
			"errors":      result.Errors,
			"requestPath": r.URL.Path,
		}, meta),
	})
	_ = s.log.Emit(ctx, "backup.bundle.restored", map[string]any{
		"file":        body.FilePath,
		"dryRun":      body.DryRun,
		"outcome":     outcome,
		"unsupported": len(result.Unsupported),
		"errors":      len(result.Errors),
	})
	writeJSON(w, http.StatusOK, map[string]any{"result": result})
}

func requestAuditMetaForBackup(r *http.Request, bodyCorrelation, bodyTrace, bodyWorkspace, fallbackPrefix string) requestAuditMeta {
	meta := requestAuditMeta{
		CorrelationID: firstNonEmptyTrimmed(
			strings.TrimSpace(bodyCorrelation),
			strings.TrimSpace(r.URL.Query().Get("correlationId")),
			strings.TrimSpace(r.Header.Get("X-Correlation-ID")),
			strings.TrimSpace(r.Header.Get("X-Request-ID")),
			strings.TrimSpace(middleware.GetReqID(r.Context())),
		),
		TraceID: firstNonEmptyTrimmed(
			strings.TrimSpace(bodyTrace),
			strings.TrimSpace(r.URL.Query().Get("traceId")),
			strings.TrimSpace(r.Header.Get("X-Trace-ID")),
		),
		WorkspaceID: firstNonEmptyTrimmed(
			strings.TrimSpace(bodyWorkspace),
			strings.TrimSpace(r.URL.Query().Get("workspaceId")),
			strings.TrimSpace(r.Header.Get("X-Workspace-ID")),
		),
	}
	if meta.CorrelationID == "" {
		meta.CorrelationID = fmt.Sprintf("%s:%d", fallbackPrefix, time.Now().UnixNano())
	}
	return meta
}

func requestAuditPayload(base map[string]any, meta requestAuditMeta) map[string]any {
	out := make(map[string]any, len(base)+3)
	for k, v := range base {
		out[k] = v
	}
	out["correlationId"] = meta.CorrelationID
	if meta.TraceID != "" {
		out["traceId"] = meta.TraceID
	}
	if meta.WorkspaceID != "" {
		out["workspaceId"] = meta.WorkspaceID
	}
	return out
}

func (s *Server) handleReleaseReadiness(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	cl, err := s.release.CheckReadiness(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"checklist": cl})
}

func (s *Server) handleReleaseArtifacts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	list, err := s.release.List(ctx, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"artifacts": list})
}

func (s *Server) handleReleaseRecord(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var body release.ArtifactRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	artifact, err := s.release.RecordArtifact(ctx, body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_ = s.log.Emit(ctx, "release.artifact.recorded", map[string]any{"id": artifact.ID, "kind": artifact.Kind})
	writeJSON(w, http.StatusOK, map[string]any{"artifact": artifact})
}

func (s *Server) handleFirstRun(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sum, err := s.release.FirstRun(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"firstRun": sum})
}
