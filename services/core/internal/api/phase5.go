package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

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
	ToolID         string         `json:"toolId"`
	LaneID         string         `json:"laneId"`
	Domain         string         `json:"domain"`
	Action         string         `json:"action"`
	RiskClass      string         `json:"riskClass"`
	ExecutionLevel string         `json:"executionLevel"`
	CorrelationID  string         `json:"correlationId"`
	Paths          []string       `json:"paths"`
	Input          map[string]any `json:"input"`
	JobID          *string        `json:"jobId,omitempty"`
	PacketID       *int64         `json:"packetId,omitempty"`
	DryRun         bool           `json:"dryRun"`
	Initiator      string         `json:"initiator"`
}

func (s *Server) handleGatewayTools(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"tools": s.gateway.Tools()})
}

func (s *Server) handleGatewayInvoke(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var body gatewayInvokeBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	result, err := s.gateway.Execute(ctx, gateway.Request{
		ToolID:         body.ToolID,
		LaneID:         body.LaneID,
		Domain:         body.Domain,
		Action:         body.Action,
		RiskClass:      body.RiskClass,
		ExecutionLevel: body.ExecutionLevel,
		CorrelationID:  body.CorrelationID,
		Paths:          body.Paths,
		Input:          body.Input,
		JobID:          body.JobID,
		PacketID:       body.PacketID,
		Initiator:      body.Initiator,
		DryRun:         body.DryRun,
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
	correlation := chi.URLParam(r, "correlationId")
	records, err := s.auditSvc.Trace(ctx, correlation)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"correlationId": correlation, "records": records})
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
	_, _ = s.auditSvc.Record(ctx, audit.CreateRequest{
		Category:    "backup",
		Action:      "bundle.created",
		SubjectType: "bundle",
		SubjectID:   strconv.FormatInt(b.ID, 10),
		Outcome:     "ok",
		Summary:     "bundle " + b.Kind + " created",
		Payload:     map[string]any{"kind": b.Kind, "label": b.Label, "file": b.FilePath},
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
	_, _ = s.auditSvc.Record(ctx, audit.CreateRequest{
		Category:    "backup",
		Action:      "bundle.restored",
		SubjectType: "bundle",
		SubjectID:   body.FilePath,
		Outcome:     "ok",
		Summary:     "bundle restored",
		Payload:     map[string]any{"dryRun": body.DryRun, "file": body.FilePath},
	})
	_ = s.log.Emit(ctx, "backup.bundle.restored", map[string]any{"file": body.FilePath, "dryRun": body.DryRun})
	writeJSON(w, http.StatusOK, map[string]any{"result": result})
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
