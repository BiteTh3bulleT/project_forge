package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"forge/projectforge/services/core/internal/modelruntime"
)

const modelRuntimeRequestBodyLimit = 1 << 20

// modelRuntimeService is the API-facing abstraction for model runtime operations.
// It is expected to be implemented by services/core/internal/modelruntime.
type modelRuntimeService interface {
	ListModels(ctx context.Context, req ModelRuntimeListRequest) ([]ModelRuntimeModel, error)
	GetModel(ctx context.Context, modelID string, req ModelRuntimeRequestMeta) (ModelRuntimeModel, error)
	ImportModel(ctx context.Context, req ModelRuntimeImportRequest) (ModelRuntimeImportResult, error)
	ScanModels(ctx context.Context, req ModelRuntimeControlRequest) ([]ModelRuntimeModel, error)
	VerifyModel(ctx context.Context, modelID string, req ModelRuntimeControlRequest) (ModelRuntimeModel, error)
	EnableModel(ctx context.Context, modelID string, req ModelRuntimeControlRequest) (ModelRuntimeModel, error)
	DisableModel(ctx context.Context, modelID string, req ModelRuntimeControlRequest) (ModelRuntimeModel, error)
	ArchiveModel(ctx context.Context, modelID string, req ModelRuntimeControlRequest) (ModelRuntimeModel, error)
	RemoveModel(ctx context.Context, modelID string, req ModelRuntimeControlRequest) (ModelRuntimeRemoveResult, error)
	DeleteModelFiles(ctx context.Context, modelID string, req ModelRuntimeControlRequest) (ModelRuntimeDeleteFilesResult, error)
	LoadModel(ctx context.Context, modelID string, req ModelRuntimeControlRequest) (ModelRuntimeLoadResult, error)
	UnloadModel(ctx context.Context, modelID string, req ModelRuntimeControlRequest) (ModelRuntimeLoadResult, error)
	Chat(ctx context.Context, req ModelRuntimeChatRequest) (ModelRuntimeChatResult, error)
	Compatibility(ctx context.Context, modelID string, req ModelRuntimeRequestMeta) (ModelRuntimeCompatibility, error)
	Health(ctx context.Context, req ModelRuntimeRequestMeta) (ModelRuntimeHealth, error)
	QueueStatus(ctx context.Context, req ModelRuntimeRequestMeta) (ModelRuntimeQueueStatus, error)
	LoadedStatus(ctx context.Context, req ModelRuntimeRequestMeta) (ModelRuntimeLoadedStatus, error)
	Usage(ctx context.Context, req ModelRuntimeRequestMeta) (ModelRuntimeUsageSummary, error)
	Backends(ctx context.Context, req ModelRuntimeRequestMeta) ([]ModelRuntimeBackendStatus, error)
}

type modelRuntimeStreamingService interface {
	StreamChat(ctx context.Context, req ModelRuntimeChatRequest, onToken func(ModelRuntimeChatStreamToken) error) (ModelRuntimeChatResult, error)
}

type ModelRuntimeRequestMeta struct {
	CorrelationID string            `json:"correlationId,omitempty"`
	TraceID       string            `json:"traceId,omitempty"`
	WorkspaceID   string            `json:"workspaceId,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

type ModelRuntimeListRequest struct {
	Meta ModelRuntimeRequestMeta `json:"meta"`
}

func modelRuntimeRouteModelID(r *http.Request) string {
	raw := strings.TrimSpace(chi.URLParam(r, "id"))
	if raw == "" {
		return ""
	}
	decoded, err := url.PathUnescape(raw)
	if err != nil {
		return raw
	}
	return strings.TrimSpace(decoded)
}

type ModelRuntimeControlRequest struct {
	Meta     ModelRuntimeRequestMeta `json:"meta"`
	Actor    string                  `json:"actor,omitempty"`
	Source   string                  `json:"source,omitempty"`
	Metadata map[string]any          `json:"metadata,omitempty"`
}

type modelRuntimeManagementBody struct {
	CorrelationID string         `json:"correlationId"`
	TraceID       string         `json:"traceId"`
	WorkspaceID   string         `json:"workspaceId"`
	LaneID        string         `json:"laneId"`
	Actor         string         `json:"actor"`
	Source        string         `json:"source"`
	CapabilityID  string         `json:"capabilityId"`
	ApprovalID    string         `json:"approvalId"`
	DryRun        bool           `json:"dryRun"`
	Preferred     bool           `json:"preferred"`
	Metadata      map[string]any `json:"metadata"`
}

type ModelRuntimeChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ModelRuntimeChatRequest struct {
	ModelID       string                    `json:"modelId"`
	Backend       string                    `json:"backend,omitempty"`
	Role          string                    `json:"role,omitempty"`
	WorkloadClass string                    `json:"workloadClass,omitempty"`
	Messages      []ModelRuntimeChatMessage `json:"messages,omitempty"`
	Prompt        string                    `json:"prompt,omitempty"`
	Parameters    map[string]any            `json:"parameters,omitempty"`
	MaxTokens     int                       `json:"maxTokens,omitempty"`
	TimeoutMs     int                       `json:"timeoutMs,omitempty"`
	MaxAttempts   int                       `json:"maxAttempts,omitempty"`
	Stream        bool                      `json:"stream,omitempty"`
	Actor         string                    `json:"actor,omitempty"`
	Source        string                    `json:"source,omitempty"`
	Meta          ModelRuntimeRequestMeta   `json:"meta"`
	Provenance    map[string]any            `json:"provenance,omitempty"`
	Metadata      map[string]any            `json:"metadata,omitempty"`
}

type ModelRuntimeUsage struct {
	PromptTokens     int `json:"promptTokens,omitempty"`
	CompletionTokens int `json:"completionTokens,omitempty"`
	TotalTokens      int `json:"totalTokens,omitempty"`
}

type ModelRuntimeChatResult struct {
	Content      string                                `json:"content"`
	FinishReason string                                `json:"finishReason,omitempty"`
	Usage        ModelRuntimeUsage                     `json:"usage,omitempty"`
	DurationMs   int64                                 `json:"durationMs,omitempty"`
	ExecutionID  string                                `json:"executionId,omitempty"`
	AttemptCount int                                   `json:"attemptCount,omitempty"`
	Role         string                                `json:"role,omitempty"`
	Checkpoint   *modelruntime.ChatExecutionCheckpoint `json:"checkpoint,omitempty"`
	Backend      string                                `json:"backend,omitempty"`
	ModelID      string                                `json:"modelId,omitempty"`
	AuditID      string                                `json:"auditId,omitempty"`
	Artifacts    []string                              `json:"artifacts,omitempty"`
	Warnings     []string                              `json:"warnings,omitempty"`
}

type ModelRuntimeChatStreamToken struct {
	Text    string `json:"text,omitempty"`
	Index   int    `json:"index,omitempty"`
	Done    bool   `json:"done,omitempty"`
	Backend string `json:"backend,omitempty"`
	ModelID string `json:"modelId,omitempty"`
}

type ModelRuntimeHealth struct {
	OK              bool           `json:"ok"`
	Status          string         `json:"status,omitempty"`
	Backend         string         `json:"backend,omitempty"`
	RuntimeEnabled  bool           `json:"runtimeEnabled,omitempty"`
	GPUAware        bool           `json:"gpuAware,omitempty"`
	DegradedReasons []string       `json:"degradedReasons,omitempty"`
	PolicyWarnings  []string       `json:"policyWarnings,omitempty"`
	Details         map[string]any `json:"details,omitempty"`
}

type ModelRuntimeQueueStatus struct {
	Depth       int               `json:"depth"`
	Active      map[string]string `json:"active,omitempty"`
	Pending     []string          `json:"pending,omitempty"`
	Scheduler   string            `json:"scheduler,omitempty"`
	PolicyState string            `json:"policyState,omitempty"`
}

type ModelRuntimeLoadedModel struct {
	ModelID    string         `json:"modelId"`
	Backend    string         `json:"backend,omitempty"`
	Status     string         `json:"status,omitempty"`
	LoadedAtMs int64          `json:"loadedAtMs,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

type ModelRuntimeLoadedStatus struct {
	Count  int                       `json:"count"`
	Models []ModelRuntimeLoadedModel `json:"models"`
}

type ModelRuntimeModel struct {
	ID           string         `json:"id"`
	DisplayName  string         `json:"displayName,omitempty"`
	Family       string         `json:"family,omitempty"`
	Backend      string         `json:"backend,omitempty"`
	Format       string         `json:"format,omitempty"`
	Status       string         `json:"status,omitempty"`
	Capabilities []string       `json:"capabilities,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

type ModelRuntimeLoadResult struct {
	ModelID    string            `json:"modelId"`
	Backend    string            `json:"backend,omitempty"`
	Status     string            `json:"status,omitempty"`
	Loaded     bool              `json:"loaded"`
	Metadata   map[string]any    `json:"metadata,omitempty"`
	Warnings   []string          `json:"warnings,omitempty"`
	LoadedAtMs int64             `json:"loadedAtMs,omitempty"`
	Details    map[string]string `json:"details,omitempty"`
}

type ModelRuntimeImportRequest struct {
	Path          string                  `json:"path"`
	ID            string                  `json:"id,omitempty"`
	DisplayName   string                  `json:"displayName,omitempty"`
	Family        string                  `json:"family,omitempty"`
	Backend       string                  `json:"backend,omitempty"`
	Capabilities  []string                `json:"capabilities,omitempty"`
	License       string                  `json:"license,omitempty"`
	Quantization  string                  `json:"quantization,omitempty"`
	ContextLength int                     `json:"contextLength,omitempty"`
	Preferred     bool                    `json:"preferred,omitempty"`
	Actor         string                  `json:"actor,omitempty"`
	Source        string                  `json:"source,omitempty"`
	Metadata      map[string]any          `json:"metadata,omitempty"`
	Meta          ModelRuntimeRequestMeta `json:"meta"`
}

type ModelRuntimeImportResult struct {
	Model       ModelRuntimeModel `json:"model"`
	Duplicate   bool              `json:"duplicate"`
	ManagedPath string            `json:"managedPath,omitempty"`
	SourcePath  string            `json:"sourcePath,omitempty"`
	Warnings    []string          `json:"warnings,omitempty"`
}

type ModelRuntimeRemoveResult struct {
	ModelID     string `json:"modelId"`
	RemovedPath string `json:"removedPath,omitempty"`
}

type ModelRuntimeDeleteFilesResult struct {
	ModelID     string `json:"modelId"`
	DeletedPath string `json:"deletedPath,omitempty"`
	Deleted     bool   `json:"deleted"`
}

type ModelRuntimeCompatibility struct {
	ModelID            string         `json:"modelId"`
	Backend            string         `json:"backend,omitempty"`
	Status             string         `json:"status,omitempty"`
	Loaded             bool           `json:"loaded"`
	BackendConfigured  bool           `json:"backendConfigured"`
	BackendHealthy     bool           `json:"backendHealthy"`
	SupportedByBackend bool           `json:"supportedByBackend"`
	CanGenerate        bool           `json:"canGenerate"`
	Preferred          bool           `json:"preferred,omitempty"`
	Warnings           []string       `json:"warnings,omitempty"`
	Details            map[string]any `json:"details,omitempty"`
}

type ModelRuntimeBackendStatus struct {
	Kind        string         `json:"kind"`
	Name        string         `json:"name"`
	Healthy     bool           `json:"healthy"`
	Detail      string         `json:"detail,omitempty"`
	LoadedModel string         `json:"loadedModel,omitempty"`
	Meta        map[string]any `json:"meta,omitempty"`
	Supervision any            `json:"supervision,omitempty"`
}

type ModelRuntimeUsageSummary struct {
	Registered int                       `json:"registered"`
	Imported   int                       `json:"imported"`
	Verified   int                       `json:"verified"`
	Available  int                       `json:"available"`
	Disabled   int                       `json:"disabled"`
	Archived   int                       `json:"archived"`
	Loaded     int                       `json:"loaded"`
	QueueDepth int                       `json:"queueDepth"`
	Running    int                       `json:"running"`
	Completed  int                       `json:"completed"`
	Backends   map[string]map[string]any `json:"backends,omitempty"`
}

type modelRuntimeError struct {
	status  int
	code    string
	message string
}

func (e *modelRuntimeError) Error() string {
	if e == nil {
		return ""
	}
	if strings.TrimSpace(e.message) != "" {
		return e.message
	}
	return e.code
}

func (e *modelRuntimeError) StatusCode() int {
	if e == nil || e.status == 0 {
		return http.StatusInternalServerError
	}
	return e.status
}

func (e *modelRuntimeError) ErrorCode() string {
	if e == nil || strings.TrimSpace(e.code) == "" {
		return "MODEL_RUNTIME_ERROR"
	}
	return e.code
}

type modelRuntimeStatusCoder interface {
	error
	StatusCode() int
}

type modelRuntimeCodeCarrier interface {
	ErrorCode() string
}

func (s *Server) handleForgeModelsList(w http.ResponseWriter, r *http.Request) {
	runtimeSvc, meta, ok := s.requireModelRuntime(w, r, "model.runtime.list")
	if !ok {
		return
	}
	models, err := runtimeSvc.ListModels(r.Context(), ModelRuntimeListRequest{
		Meta: meta,
	})
	if err != nil {
		s.writeModelRuntimeError(w, err, meta)
		return
	}
	sort.Slice(models, func(i, j int) bool {
		return strings.ToLower(models[i].ID) < strings.ToLower(models[j].ID)
	})
	writeJSON(w, http.StatusOK, map[string]any{
		"models":        models,
		"correlationId": meta.CorrelationID,
		"traceId":       meta.TraceID,
		"workspaceId":   meta.WorkspaceID,
	})
}

func (s *Server) handleForgeModelImport(w http.ResponseWriter, r *http.Request) {
	runtimeSvc, initialMeta, ok := s.requireModelRuntime(w, r, "model.runtime.import")
	if !ok {
		return
	}
	var body struct {
		Path          string         `json:"path"`
		ID            string         `json:"id"`
		DisplayName   string         `json:"displayName"`
		Family        string         `json:"family"`
		Backend       string         `json:"backend"`
		Capabilities  []string       `json:"capabilities"`
		License       string         `json:"license"`
		Quantization  string         `json:"quantization"`
		ContextLength int            `json:"contextLength"`
		Preferred     bool           `json:"preferred"`
		Actor         string         `json:"actor"`
		Source        string         `json:"source"`
		CapabilityID  string         `json:"capabilityId"`
		ApprovalID    string         `json:"approvalId"`
		DryRun        bool           `json:"dryRun"`
		Metadata      map[string]any `json:"metadata"`
		CorrelationID string         `json:"correlationId"`
		TraceID       string         `json:"traceId"`
		WorkspaceID   string         `json:"workspaceId"`
		LaneID        string         `json:"laneId"`
	}
	if err := decodeOptionalJSONBody(r, &body); err != nil {
		s.writeModelRuntimeDecodeError(w, err, initialMeta)
		return
	}
	if strings.TrimSpace(body.Path) == "" {
		s.writeModelRuntimeError(w, &modelRuntimeError{status: http.StatusBadRequest, code: "MODEL_PATH_REQUIRED", message: "path is required"}, initialMeta)
		return
	}
	meta := requestAuditMetaForBackup(r, body.CorrelationID, body.TraceID, body.WorkspaceID, "model.runtime.import")
	metaReq := modelRuntimeMetaFromRequestAudit(meta)
	govReq := modelManagementGovernanceRequest{
		Operation:     "import",
		ModelID:       strings.TrimSpace(body.ID),
		Path:          strings.TrimSpace(body.Path),
		Backend:       strings.TrimSpace(body.Backend),
		Actor:         strings.TrimSpace(body.Actor),
		Source:        strings.TrimSpace(body.Source),
		WorkspaceID:   metaReq.WorkspaceID,
		LaneID:        strings.TrimSpace(body.LaneID),
		CorrelationID: metaReq.CorrelationID,
		TraceID:       metaReq.TraceID,
		CapabilityID:  strings.TrimSpace(body.CapabilityID),
		ApprovalID:    strings.TrimSpace(body.ApprovalID),
		Preferred:     body.Preferred,
		DryRun:        body.DryRun,
		Metadata:      body.Metadata,
	}
	decision, err := s.enforceModelManagementGovernance(r.Context(), runtimeSvc, govReq)
	if err != nil {
		s.writeModelRuntimeError(w, err, metaReq)
		return
	}
	if s.writeModelManagementGovernanceResult(w, r, metaReq, govReq, decision) {
		return
	}
	result, err := runtimeSvc.ImportModel(r.Context(), ModelRuntimeImportRequest{
		Path:          strings.TrimSpace(body.Path),
		ID:            strings.TrimSpace(body.ID),
		DisplayName:   strings.TrimSpace(body.DisplayName),
		Family:        strings.TrimSpace(body.Family),
		Backend:       strings.TrimSpace(body.Backend),
		Capabilities:  append([]string(nil), body.Capabilities...),
		License:       strings.TrimSpace(body.License),
		Quantization:  strings.TrimSpace(body.Quantization),
		ContextLength: body.ContextLength,
		Preferred:     body.Preferred,
		Actor:         strings.TrimSpace(body.Actor),
		Source:        strings.TrimSpace(body.Source),
		Metadata:      modelManagementMetadata(body.Metadata, decision),
		Meta:          metaReq,
	})
	if err != nil {
		s.writeModelRuntimeError(w, err, metaReq)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": result, "correlationId": metaReq.CorrelationID, "traceId": metaReq.TraceID, "workspaceId": metaReq.WorkspaceID})
}

func (s *Server) handleForgeModelsScan(w http.ResponseWriter, r *http.Request) {
	runtimeSvc, initialMeta, ok := s.requireModelRuntime(w, r, "model.runtime.scan")
	if !ok {
		return
	}
	var body modelRuntimeManagementBody
	if err := decodeOptionalJSONBody(r, &body); err != nil {
		s.writeModelRuntimeDecodeError(w, err, initialMeta)
		return
	}
	meta := requestAuditMetaForBackup(r, body.CorrelationID, body.TraceID, body.WorkspaceID, "model.runtime.scan")
	metaReq := modelRuntimeMetaFromRequestAudit(meta)
	govReq := modelManagementGovernanceRequest{
		Operation:     "scan",
		Actor:         strings.TrimSpace(body.Actor),
		Source:        strings.TrimSpace(body.Source),
		WorkspaceID:   metaReq.WorkspaceID,
		LaneID:        strings.TrimSpace(body.LaneID),
		CorrelationID: metaReq.CorrelationID,
		TraceID:       metaReq.TraceID,
		CapabilityID:  strings.TrimSpace(body.CapabilityID),
		ApprovalID:    strings.TrimSpace(body.ApprovalID),
		DryRun:        body.DryRun,
		Metadata:      body.Metadata,
	}
	decision, err := s.enforceModelManagementGovernance(r.Context(), runtimeSvc, govReq)
	if err != nil {
		s.writeModelRuntimeError(w, err, metaReq)
		return
	}
	if s.writeModelManagementGovernanceResult(w, r, metaReq, govReq, decision) {
		return
	}
	models, err := runtimeSvc.ScanModels(r.Context(), ModelRuntimeControlRequest{Meta: metaReq, Actor: strings.TrimSpace(body.Actor), Source: strings.TrimSpace(body.Source), Metadata: modelManagementMetadata(body.Metadata, decision)})
	if err != nil {
		s.writeModelRuntimeError(w, err, metaReq)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"models": models, "count": len(models), "correlationId": metaReq.CorrelationID, "traceId": metaReq.TraceID, "workspaceId": metaReq.WorkspaceID})
}

func (s *Server) handleForgeModelGet(w http.ResponseWriter, r *http.Request) {
	runtimeSvc, meta, ok := s.requireModelRuntime(w, r, "model.runtime.get")
	if !ok {
		return
	}
	id := modelRuntimeRouteModelID(r)
	if id == "" {
		s.writeModelRuntimeError(w, &modelRuntimeError{status: http.StatusBadRequest, code: "MODEL_ID_REQUIRED", message: "model id is required"}, meta)
		return
	}
	model, err := runtimeSvc.GetModel(r.Context(), id, meta)
	if err != nil {
		s.writeModelRuntimeError(w, err, meta)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"model":         model,
		"correlationId": meta.CorrelationID,
		"traceId":       meta.TraceID,
		"workspaceId":   meta.WorkspaceID,
	})
}

func (s *Server) handleForgeModelCompatibility(w http.ResponseWriter, r *http.Request) {
	runtimeSvc, meta, ok := s.requireModelRuntime(w, r, "model.runtime.compatibility")
	if !ok {
		return
	}
	id := modelRuntimeRouteModelID(r)
	if id == "" {
		s.writeModelRuntimeError(w, &modelRuntimeError{status: http.StatusBadRequest, code: "MODEL_ID_REQUIRED", message: "model id is required"}, meta)
		return
	}
	compat, err := runtimeSvc.Compatibility(r.Context(), id, meta)
	if err != nil {
		s.writeModelRuntimeError(w, err, meta)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"compatibility": compat, "correlationId": meta.CorrelationID, "traceId": meta.TraceID, "workspaceId": meta.WorkspaceID})
}

func (s *Server) handleForgeModelLoad(w http.ResponseWriter, r *http.Request) {
	s.handleForgeModelControl(w, r, true)
}

func (s *Server) handleForgeModelUnload(w http.ResponseWriter, r *http.Request) {
	s.handleForgeModelControl(w, r, false)
}

func (s *Server) handleForgeModelVerify(w http.ResponseWriter, r *http.Request) {
	s.handleForgeModelManagement(w, r, "verify")
}

func (s *Server) handleForgeModelEnable(w http.ResponseWriter, r *http.Request) {
	s.handleForgeModelManagement(w, r, "enable")
}

func (s *Server) handleForgeModelDisable(w http.ResponseWriter, r *http.Request) {
	s.handleForgeModelManagement(w, r, "disable")
}

func (s *Server) handleForgeModelArchive(w http.ResponseWriter, r *http.Request) {
	s.handleForgeModelManagement(w, r, "archive")
}

func (s *Server) handleForgeModelRemove(w http.ResponseWriter, r *http.Request) {
	s.handleForgeModelManagement(w, r, "remove")
}

func (s *Server) handleForgeModelDeleteFile(w http.ResponseWriter, r *http.Request) {
	s.handleForgeModelManagement(w, r, "delete_file")
}

func (s *Server) handleForgeModelControl(w http.ResponseWriter, r *http.Request, load bool) {
	runtimeSvc, initialMeta, ok := s.requireModelRuntime(w, r, "model.runtime.control")
	if !ok {
		return
	}
	id := modelRuntimeRouteModelID(r)
	if id == "" {
		s.writeModelRuntimeError(w, &modelRuntimeError{status: http.StatusBadRequest, code: "MODEL_ID_REQUIRED", message: "model id is required"}, initialMeta)
		return
	}

	var body modelRuntimeManagementBody
	if err := decodeOptionalJSONBody(r, &body); err != nil {
		s.writeModelRuntimeDecodeError(w, err, initialMeta)
		return
	}

	meta := requestAuditMetaForBackup(r, body.CorrelationID, body.TraceID, body.WorkspaceID, "model.runtime.control")
	metaReq := modelRuntimeMetaFromRequestAudit(meta)
	action := "unload"
	if load {
		action = "load"
	}
	govReq := modelManagementGovernanceRequest{
		Operation:     action,
		ModelID:       id,
		Actor:         strings.TrimSpace(body.Actor),
		Source:        strings.TrimSpace(body.Source),
		WorkspaceID:   metaReq.WorkspaceID,
		LaneID:        strings.TrimSpace(body.LaneID),
		CorrelationID: metaReq.CorrelationID,
		TraceID:       metaReq.TraceID,
		CapabilityID:  strings.TrimSpace(body.CapabilityID),
		ApprovalID:    strings.TrimSpace(body.ApprovalID),
		DryRun:        body.DryRun,
		Metadata:      body.Metadata,
	}
	decision, err := s.enforceModelManagementGovernance(r.Context(), runtimeSvc, govReq)
	if err != nil {
		s.writeModelRuntimeError(w, err, metaReq)
		return
	}
	if s.writeModelManagementGovernanceResult(w, r, metaReq, govReq, decision) {
		return
	}
	controlReq := ModelRuntimeControlRequest{
		Meta:     metaReq,
		Actor:    strings.TrimSpace(body.Actor),
		Source:   strings.TrimSpace(body.Source),
		Metadata: modelManagementMetadata(body.Metadata, decision),
	}

	var result ModelRuntimeLoadResult
	if load {
		result, err = runtimeSvc.LoadModel(r.Context(), id, controlReq)
	} else {
		result, err = runtimeSvc.UnloadModel(r.Context(), id, controlReq)
	}
	if err != nil {
		s.writeModelRuntimeError(w, err, metaReq)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"result":        result,
		"correlationId": metaReq.CorrelationID,
		"traceId":       metaReq.TraceID,
		"workspaceId":   metaReq.WorkspaceID,
	})
}

func (s *Server) handleForgeModelManagement(w http.ResponseWriter, r *http.Request, action string) {
	runtimeSvc, initialMeta, ok := s.requireModelRuntime(w, r, "model.runtime."+action)
	if !ok {
		return
	}
	id := modelRuntimeRouteModelID(r)
	if id == "" {
		s.writeModelRuntimeError(w, &modelRuntimeError{status: http.StatusBadRequest, code: "MODEL_ID_REQUIRED", message: "model id is required"}, initialMeta)
		return
	}
	var body modelRuntimeManagementBody
	if err := decodeOptionalJSONBody(r, &body); err != nil {
		s.writeModelRuntimeDecodeError(w, err, initialMeta)
		return
	}
	meta := requestAuditMetaForBackup(r, body.CorrelationID, body.TraceID, body.WorkspaceID, "model.runtime."+action)
	metaReq := modelRuntimeMetaFromRequestAudit(meta)
	govReq := modelManagementGovernanceRequest{
		Operation:     action,
		ModelID:       id,
		Actor:         strings.TrimSpace(body.Actor),
		Source:        strings.TrimSpace(body.Source),
		WorkspaceID:   metaReq.WorkspaceID,
		LaneID:        strings.TrimSpace(body.LaneID),
		CorrelationID: metaReq.CorrelationID,
		TraceID:       metaReq.TraceID,
		CapabilityID:  strings.TrimSpace(body.CapabilityID),
		ApprovalID:    strings.TrimSpace(body.ApprovalID),
		Preferred:     body.Preferred,
		DryRun:        body.DryRun,
		Metadata:      body.Metadata,
	}
	decision, err := s.enforceModelManagementGovernance(r.Context(), runtimeSvc, govReq)
	if err != nil {
		s.writeModelRuntimeError(w, err, metaReq)
		return
	}
	if s.writeModelManagementGovernanceResult(w, r, metaReq, govReq, decision) {
		return
	}
	controlReq := ModelRuntimeControlRequest{Meta: metaReq, Actor: strings.TrimSpace(body.Actor), Source: strings.TrimSpace(body.Source), Metadata: modelManagementMetadata(body.Metadata, decision)}
	switch action {
	case "verify":
		model, err := runtimeSvc.VerifyModel(r.Context(), id, controlReq)
		if err != nil {
			s.writeModelRuntimeError(w, err, metaReq)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"model": model, "correlationId": metaReq.CorrelationID, "traceId": metaReq.TraceID, "workspaceId": metaReq.WorkspaceID})
	case "enable":
		model, err := runtimeSvc.EnableModel(r.Context(), id, controlReq)
		if err != nil {
			s.writeModelRuntimeError(w, err, metaReq)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"model": model, "correlationId": metaReq.CorrelationID, "traceId": metaReq.TraceID, "workspaceId": metaReq.WorkspaceID})
	case "disable":
		model, err := runtimeSvc.DisableModel(r.Context(), id, controlReq)
		if err != nil {
			s.writeModelRuntimeError(w, err, metaReq)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"model": model, "correlationId": metaReq.CorrelationID, "traceId": metaReq.TraceID, "workspaceId": metaReq.WorkspaceID})
	case "archive":
		model, err := runtimeSvc.ArchiveModel(r.Context(), id, controlReq)
		if err != nil {
			s.writeModelRuntimeError(w, err, metaReq)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"model": model, "correlationId": metaReq.CorrelationID, "traceId": metaReq.TraceID, "workspaceId": metaReq.WorkspaceID})
	case "remove":
		result, err := runtimeSvc.RemoveModel(r.Context(), id, controlReq)
		if err != nil {
			s.writeModelRuntimeError(w, err, metaReq)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"result": result, "correlationId": metaReq.CorrelationID, "traceId": metaReq.TraceID, "workspaceId": metaReq.WorkspaceID})
	case "delete_file":
		result, err := runtimeSvc.DeleteModelFiles(r.Context(), id, controlReq)
		if err != nil {
			s.writeModelRuntimeError(w, err, metaReq)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"result": result, "correlationId": metaReq.CorrelationID, "traceId": metaReq.TraceID, "workspaceId": metaReq.WorkspaceID})
	default:
		s.writeModelRuntimeError(w, &modelRuntimeError{status: http.StatusNotImplemented, code: "MODEL_RUNTIME_ACTION_UNSUPPORTED", message: "unsupported model runtime action"}, metaReq)
	}
}

func (s *Server) handleForgeModelChat(w http.ResponseWriter, r *http.Request) {
	runtimeSvc, initialMeta, ok := s.requireModelRuntime(w, r, "model.runtime.chat")
	if !ok {
		return
	}
	pathModelID := modelRuntimeRouteModelID(r)
	if pathModelID == "" {
		s.writeModelRuntimeError(w, &modelRuntimeError{status: http.StatusBadRequest, code: "MODEL_ID_REQUIRED", message: "model id is required"}, initialMeta)
		return
	}

	var body struct {
		ModelID       string                    `json:"modelId"`
		Backend       string                    `json:"backend"`
		Role          string                    `json:"role"`
		WorkloadClass string                    `json:"workloadClass"`
		Messages      []ModelRuntimeChatMessage `json:"messages"`
		Prompt        string                    `json:"prompt"`
		Parameters    map[string]any            `json:"parameters"`
		MaxTokens     int                       `json:"maxTokens"`
		TimeoutMs     int                       `json:"timeoutMs"`
		Stream        bool                      `json:"stream"`
		Actor         string                    `json:"actor"`
		Source        string                    `json:"source"`
		CorrelationID string                    `json:"correlationId"`
		TraceID       string                    `json:"traceId"`
		WorkspaceID   string                    `json:"workspaceId"`
		Provenance    map[string]any            `json:"provenance"`
		Metadata      map[string]any            `json:"metadata"`
	}
	if err := decodeModelRuntimeJSONBody(r, &body); err != nil {
		s.writeModelRuntimeDecodeError(w, err, initialMeta)
		return
	}
	if bodyModelID := strings.TrimSpace(body.ModelID); bodyModelID != "" && bodyModelID != pathModelID {
		s.writeModelRuntimeError(w, &modelRuntimeError{status: http.StatusBadRequest, code: "MODEL_ID_MISMATCH", message: "model id in body must match route model id"}, initialMeta)
		return
	}
	if body.MaxTokens < 0 {
		s.writeModelRuntimeError(w, &modelRuntimeError{status: http.StatusBadRequest, code: "MAX_TOKENS_INVALID", message: "maxTokens must be >= 0"}, initialMeta)
		return
	}
	if body.TimeoutMs < 0 {
		s.writeModelRuntimeError(w, &modelRuntimeError{status: http.StatusBadRequest, code: "TIMEOUT_INVALID", message: "timeoutMs must be >= 0"}, initialMeta)
		return
	}
	if err := validateChatMessages(body.Messages); err != nil {
		s.writeModelRuntimeError(w, err, initialMeta)
		return
	}
	if len(body.Messages) == 0 && strings.TrimSpace(body.Prompt) == "" {
		s.writeModelRuntimeError(w, &modelRuntimeError{status: http.StatusBadRequest, code: "PROMPT_REQUIRED", message: "messages or prompt is required"}, initialMeta)
		return
	}

	meta := requestAuditMetaForBackup(r, body.CorrelationID, body.TraceID, body.WorkspaceID, "model.runtime.chat")
	metaReq := modelRuntimeMetaFromRequestAudit(meta)
	if body.Stream {
		streamRuntime, ok := runtimeSvc.(modelRuntimeStreamingService)
		if !ok {
			s.writeModelRuntimeError(w, &modelRuntimeError{status: http.StatusNotImplemented, code: "STREAM_UNSUPPORTED", message: "streaming is unavailable for the current model runtime service"}, metaReq)
			return
		}
		s.streamForgeModelChat(w, r, streamRuntime, ModelRuntimeChatRequest{
			ModelID:       pathModelID,
			Backend:       strings.TrimSpace(body.Backend),
			Role:          strings.TrimSpace(body.Role),
			WorkloadClass: strings.TrimSpace(body.WorkloadClass),
			Messages:      body.Messages,
			Prompt:        strings.TrimSpace(body.Prompt),
			Parameters:    body.Parameters,
			MaxTokens:     body.MaxTokens,
			TimeoutMs:     body.TimeoutMs,
			Stream:        true,
			Actor:         strings.TrimSpace(body.Actor),
			Source:        strings.TrimSpace(body.Source),
			Meta:          metaReq,
			Provenance:    body.Provenance,
			Metadata:      body.Metadata,
		})
		return
	}
	result, err := runtimeSvc.Chat(r.Context(), ModelRuntimeChatRequest{
		ModelID:       pathModelID,
		Backend:       strings.TrimSpace(body.Backend),
		Role:          strings.TrimSpace(body.Role),
		WorkloadClass: strings.TrimSpace(body.WorkloadClass),
		Messages:      body.Messages,
		Prompt:        strings.TrimSpace(body.Prompt),
		Parameters:    body.Parameters,
		MaxTokens:     body.MaxTokens,
		TimeoutMs:     body.TimeoutMs,
		Stream:        body.Stream,
		Actor:         strings.TrimSpace(body.Actor),
		Source:        strings.TrimSpace(body.Source),
		Meta:          metaReq,
		Provenance:    body.Provenance,
		Metadata:      body.Metadata,
	})
	if err != nil {
		s.writeModelRuntimeError(w, err, metaReq)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"result":        result,
		"correlationId": metaReq.CorrelationID,
		"traceId":       metaReq.TraceID,
		"workspaceId":   metaReq.WorkspaceID,
	})
}

func (s *Server) handleForgeModelRuntimeHealth(w http.ResponseWriter, r *http.Request) {
	runtimeSvc, meta, ok := s.requireModelRuntime(w, r, "model.runtime.health")
	if !ok {
		return
	}
	health, err := runtimeSvc.Health(r.Context(), meta)
	if err != nil {
		s.writeModelRuntimeError(w, err, meta)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"health":        health,
		"correlationId": meta.CorrelationID,
		"traceId":       meta.TraceID,
		"workspaceId":   meta.WorkspaceID,
	})
}

func (s *Server) handleForgeModelRuntimeBackends(w http.ResponseWriter, r *http.Request) {
	runtimeSvc, meta, ok := s.requireModelRuntime(w, r, "model.runtime.backends")
	if !ok {
		return
	}
	backends, err := runtimeSvc.Backends(r.Context(), meta)
	if err != nil {
		s.writeModelRuntimeError(w, err, meta)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"backends": backends, "correlationId": meta.CorrelationID, "traceId": meta.TraceID, "workspaceId": meta.WorkspaceID})
}

func (s *Server) handleForgeModelRuntimeUsage(w http.ResponseWriter, r *http.Request) {
	runtimeSvc, meta, ok := s.requireModelRuntime(w, r, "model.runtime.usage")
	if !ok {
		return
	}
	usage, err := runtimeSvc.Usage(r.Context(), meta)
	if err != nil {
		s.writeModelRuntimeError(w, err, meta)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"usage": usage, "correlationId": meta.CorrelationID, "traceId": meta.TraceID, "workspaceId": meta.WorkspaceID})
}

func (s *Server) handleForgeModelRuntimeQueue(w http.ResponseWriter, r *http.Request) {
	runtimeSvc, meta, ok := s.requireModelRuntime(w, r, "model.runtime.queue")
	if !ok {
		return
	}
	status, err := runtimeSvc.QueueStatus(r.Context(), meta)
	if err != nil {
		s.writeModelRuntimeError(w, err, meta)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"queue":         status,
		"correlationId": meta.CorrelationID,
		"traceId":       meta.TraceID,
		"workspaceId":   meta.WorkspaceID,
	})
}

func (s *Server) handleForgeModelRuntimeLoaded(w http.ResponseWriter, r *http.Request) {
	runtimeSvc, meta, ok := s.requireModelRuntime(w, r, "model.runtime.loaded")
	if !ok {
		return
	}
	status, err := runtimeSvc.LoadedStatus(r.Context(), meta)
	if err != nil {
		s.writeModelRuntimeError(w, err, meta)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"loaded":        status,
		"correlationId": meta.CorrelationID,
		"traceId":       meta.TraceID,
		"workspaceId":   meta.WorkspaceID,
	})
}

func (s *Server) handleV1Models(w http.ResponseWriter, r *http.Request) {
	runtimeSvc, meta, ok := s.requireModelRuntime(w, r, "model.runtime.openai.list")
	if !ok {
		return
	}
	models, err := runtimeSvc.ListModels(r.Context(), ModelRuntimeListRequest{Meta: meta})
	if err != nil {
		s.writeModelRuntimeError(w, err, meta)
		return
	}
	sort.Slice(models, func(i, j int) bool {
		return strings.ToLower(models[i].ID) < strings.ToLower(models[j].ID)
	})
	items := make([]map[string]any, 0, len(models))
	for _, model := range models {
		items = append(items, map[string]any{
			"id":           model.ID,
			"object":       "model",
			"owned_by":     "forge",
			"status":       model.Status,
			"backend":      model.Backend,
			"capabilities": model.Capabilities,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"object":        "list",
		"data":          items,
		"correlationId": meta.CorrelationID,
		"traceId":       meta.TraceID,
		"workspaceId":   meta.WorkspaceID,
	})
}

func (s *Server) handleV1ChatCompletions(w http.ResponseWriter, r *http.Request) {
	runtimeSvc, initialMeta, ok := s.requireModelRuntime(w, r, "model.runtime.openai.chat")
	if !ok {
		return
	}
	var body struct {
		Model         string                    `json:"model"`
		Role          string                    `json:"role"`
		WorkloadClass string                    `json:"workloadClass"`
		Messages      []ModelRuntimeChatMessage `json:"messages"`
		MaxTokens     int                       `json:"max_tokens"`
		TimeoutMs     int                       `json:"timeout_ms"`
		Stream        bool                      `json:"stream"`
		User          string                    `json:"user"`
		Metadata      map[string]any            `json:"metadata"`
		CorrelationID string                    `json:"correlationId"`
		TraceID       string                    `json:"traceId"`
		WorkspaceID   string                    `json:"workspaceId"`
	}
	if err := decodeModelRuntimeJSONBody(r, &body); err != nil {
		s.writeModelRuntimeDecodeError(w, err, initialMeta)
		return
	}
	modelID := strings.TrimSpace(body.Model)
	if modelID == "" {
		s.writeModelRuntimeError(w, &modelRuntimeError{status: http.StatusBadRequest, code: "MODEL_REQUIRED", message: "model is required"}, initialMeta)
		return
	}
	if len(body.Messages) == 0 {
		s.writeModelRuntimeError(w, &modelRuntimeError{status: http.StatusBadRequest, code: "MESSAGES_REQUIRED", message: "messages are required"}, initialMeta)
		return
	}
	if body.MaxTokens < 0 {
		s.writeModelRuntimeError(w, &modelRuntimeError{status: http.StatusBadRequest, code: "MAX_TOKENS_INVALID", message: "max_tokens must be >= 0"}, initialMeta)
		return
	}
	if body.TimeoutMs < 0 {
		s.writeModelRuntimeError(w, &modelRuntimeError{status: http.StatusBadRequest, code: "TIMEOUT_INVALID", message: "timeout_ms must be >= 0"}, initialMeta)
		return
	}
	if err := validateChatMessages(body.Messages); err != nil {
		s.writeModelRuntimeError(w, err, initialMeta)
		return
	}

	meta := requestAuditMetaForBackup(r, body.CorrelationID, body.TraceID, body.WorkspaceID, "model.runtime.openai.chat")
	metaReq := modelRuntimeMetaFromRequestAudit(meta)
	if body.Stream {
		streamRuntime, ok := runtimeSvc.(modelRuntimeStreamingService)
		if !ok {
			s.writeModelRuntimeError(w, &modelRuntimeError{status: http.StatusNotImplemented, code: "STREAM_UNSUPPORTED", message: "streaming is unavailable for the current model runtime service"}, metaReq)
			return
		}
		s.streamOpenAICompatChatCompletions(w, r, streamRuntime, ModelRuntimeChatRequest{
			ModelID:       modelID,
			Role:          strings.TrimSpace(body.Role),
			WorkloadClass: strings.TrimSpace(body.WorkloadClass),
			Messages:      body.Messages,
			MaxTokens:     body.MaxTokens,
			TimeoutMs:     body.TimeoutMs,
			Stream:        true,
			Actor:         strings.TrimSpace(body.User),
			Source:        "openai_compat",
			Meta:          metaReq,
			Metadata:      body.Metadata,
		})
		return
	}
	result, err := runtimeSvc.Chat(r.Context(), ModelRuntimeChatRequest{
		ModelID:       modelID,
		Role:          strings.TrimSpace(body.Role),
		WorkloadClass: strings.TrimSpace(body.WorkloadClass),
		Messages:      body.Messages,
		MaxTokens:     body.MaxTokens,
		TimeoutMs:     body.TimeoutMs,
		Stream:        body.Stream,
		Actor:         strings.TrimSpace(body.User),
		Source:        "openai_compat",
		Meta:          metaReq,
		Metadata:      body.Metadata,
	})
	if err != nil {
		s.writeModelRuntimeError(w, err, metaReq)
		return
	}

	totalTokens := result.Usage.TotalTokens
	if totalTokens == 0 {
		totalTokens = result.Usage.PromptTokens + result.Usage.CompletionTokens
	}
	finishReason := strings.TrimSpace(result.FinishReason)
	if finishReason == "" {
		finishReason = "stop"
	}
	responseModelID := modelID
	if strings.TrimSpace(result.ModelID) != "" {
		responseModelID = strings.TrimSpace(result.ModelID)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":      fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano()),
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   responseModelID,
		"choices": []map[string]any{
			{
				"index": 0,
				"message": map[string]any{
					"role":    "assistant",
					"content": result.Content,
				},
				"finish_reason": finishReason,
			},
		},
		"usage": map[string]any{
			"prompt_tokens":     result.Usage.PromptTokens,
			"completion_tokens": result.Usage.CompletionTokens,
			"total_tokens":      totalTokens,
		},
		"correlationId": metaReq.CorrelationID,
		"traceId":       metaReq.TraceID,
		"workspaceId":   metaReq.WorkspaceID,
	})
}

func (s *Server) streamForgeModelChat(w http.ResponseWriter, r *http.Request, runtimeSvc modelRuntimeStreamingService, req ModelRuntimeChatRequest) {
	prepareModelRuntimeSSE(w)
	result, err := runtimeSvc.StreamChat(r.Context(), req, func(token ModelRuntimeChatStreamToken) error {
		if token.Done || strings.TrimSpace(token.Text) == "" {
			return nil
		}
		return writeModelRuntimeSSE(w, "token", token)
	})
	if err != nil {
		_ = writeModelRuntimeSSE(w, "error", map[string]any{
			"error": map[string]any{
				"code":    modelRuntimeErrorCode(err),
				"message": modelRuntimeErrorMessage(err),
			},
			"correlationId": req.Meta.CorrelationID,
			"traceId":       req.Meta.TraceID,
			"workspaceId":   req.Meta.WorkspaceID,
		})
		flushModelRuntimeSSE(w)
		return
	}
	_ = writeModelRuntimeSSE(w, "result", map[string]any{
		"result":        result,
		"correlationId": req.Meta.CorrelationID,
		"traceId":       req.Meta.TraceID,
		"workspaceId":   req.Meta.WorkspaceID,
	})
	_ = writeModelRuntimeSSE(w, "done", map[string]any{"done": true})
	flushModelRuntimeSSE(w)
}

func (s *Server) streamOpenAICompatChatCompletions(w http.ResponseWriter, r *http.Request, runtimeSvc modelRuntimeStreamingService, req ModelRuntimeChatRequest) {
	prepareModelRuntimeSSE(w)
	id := fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano())
	created := time.Now().Unix()
	modelID := strings.TrimSpace(req.ModelID)
	result, err := runtimeSvc.StreamChat(r.Context(), req, func(token ModelRuntimeChatStreamToken) error {
		if token.Done || strings.TrimSpace(token.Text) == "" {
			return nil
		}
		if strings.TrimSpace(token.ModelID) != "" {
			modelID = strings.TrimSpace(token.ModelID)
		}
		return writeModelRuntimeSSEData(w, map[string]any{
			"id":      id,
			"object":  "chat.completion.chunk",
			"created": created,
			"model":   modelID,
			"choices": []map[string]any{
				{
					"index": token.Index,
					"delta": map[string]any{
						"content": token.Text,
					},
					"finish_reason": nil,
				},
			},
			"correlationId": req.Meta.CorrelationID,
			"traceId":       req.Meta.TraceID,
			"workspaceId":   req.Meta.WorkspaceID,
		})
	})
	if err != nil {
		_ = writeModelRuntimeSSEData(w, map[string]any{
			"error": map[string]any{
				"code":    modelRuntimeErrorCode(err),
				"message": modelRuntimeErrorMessage(err),
			},
			"correlationId": req.Meta.CorrelationID,
			"traceId":       req.Meta.TraceID,
			"workspaceId":   req.Meta.WorkspaceID,
		})
		flushModelRuntimeSSE(w)
		return
	}
	if strings.TrimSpace(result.ModelID) != "" {
		modelID = strings.TrimSpace(result.ModelID)
	}
	finishReason := strings.TrimSpace(result.FinishReason)
	if finishReason == "" {
		finishReason = "stop"
	}
	totalTokens := result.Usage.TotalTokens
	if totalTokens == 0 {
		totalTokens = result.Usage.PromptTokens + result.Usage.CompletionTokens
	}
	_ = writeModelRuntimeSSEData(w, map[string]any{
		"id":      id,
		"object":  "chat.completion.chunk",
		"created": created,
		"model":   modelID,
		"choices": []map[string]any{
			{
				"index":         0,
				"delta":         map[string]any{},
				"finish_reason": finishReason,
			},
		},
		"usage": map[string]any{
			"prompt_tokens":     result.Usage.PromptTokens,
			"completion_tokens": result.Usage.CompletionTokens,
			"total_tokens":      totalTokens,
		},
		"correlationId": req.Meta.CorrelationID,
		"traceId":       req.Meta.TraceID,
		"workspaceId":   req.Meta.WorkspaceID,
	})
	_, _ = io.WriteString(w, "data: [DONE]\n\n")
	flushModelRuntimeSSE(w)
}

func prepareModelRuntimeSSE(w http.ResponseWriter) {
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
}

func writeModelRuntimeSSE(w http.ResponseWriter, event string, payload any) error {
	if strings.TrimSpace(event) != "" {
		if _, err := fmt.Fprintf(w, "event: %s\n", strings.TrimSpace(event)); err != nil {
			return err
		}
	}
	return writeModelRuntimeSSEData(w, payload)
}

func writeModelRuntimeSSEData(w http.ResponseWriter, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "data: %s\n\n", raw); err != nil {
		return err
	}
	flushModelRuntimeSSE(w)
	return nil
}

func flushModelRuntimeSSE(w http.ResponseWriter) {
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}

func modelRuntimeErrorCode(err error) string {
	_, code, _ := mapModelRuntimeError(err)
	return code
}

func modelRuntimeErrorMessage(err error) string {
	_, _, message := mapModelRuntimeError(err)
	return message
}

func (s *Server) requireModelRuntime(w http.ResponseWriter, r *http.Request, fallbackPrefix string) (modelRuntimeService, ModelRuntimeRequestMeta, bool) {
	meta := requestAuditMetaForBackup(r, "", "", "", fallbackPrefix)
	metaReq := modelRuntimeMetaFromRequestAudit(meta)
	if s.modelRuntime == nil {
		s.writeModelRuntimeError(w, &modelRuntimeError{
			status:  http.StatusServiceUnavailable,
			code:    "MODEL_RUNTIME_UNAVAILABLE",
			message: "model runtime is unavailable",
		}, metaReq)
		return nil, metaReq, false
	}
	return s.modelRuntime, metaReq, true
}

func (s *Server) writeModelRuntimeError(w http.ResponseWriter, err error, meta ModelRuntimeRequestMeta) {
	status, code, message := mapModelRuntimeError(err)
	writeJSON(w, status, map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
		"correlationId": meta.CorrelationID,
		"traceId":       meta.TraceID,
		"workspaceId":   meta.WorkspaceID,
	})
}

func mapModelRuntimeError(err error) (int, string, string) {
	if err == nil {
		return http.StatusInternalServerError, "MODEL_RUNTIME_ERROR", "model runtime error"
	}

	var runtimeErr *modelRuntimeError
	if errors.As(err, &runtimeErr) {
		return runtimeErr.StatusCode(), runtimeErr.ErrorCode(), runtimeErr.Error()
	}

	var statusCarrier modelRuntimeStatusCoder
	status := http.StatusInternalServerError
	if errors.As(err, &statusCarrier) {
		if code := statusCarrier.StatusCode(); code >= 400 {
			status = code
		}
	}

	code := "MODEL_RUNTIME_ERROR"
	var codeCarrier modelRuntimeCodeCarrier
	if errors.As(err, &codeCarrier) {
		if c := strings.TrimSpace(codeCarrier.ErrorCode()); c != "" {
			code = c
		}
	}

	if errors.Is(err, sql.ErrNoRows) {
		return http.StatusNotFound, "MODEL_NOT_FOUND", "model not found"
	}
	if errors.Is(err, context.Canceled) {
		return http.StatusRequestTimeout, "MODEL_REQUEST_CANCELED", "model request canceled"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return http.StatusGatewayTimeout, "MODEL_REQUEST_TIMEOUT", "model request timed out"
	}

	message := strings.TrimSpace(err.Error())
	if message == "" {
		message = "model runtime error"
	}
	lower := strings.ToLower(message)
	if code == "MODEL_RUNTIME_ERROR" {
		if strings.Contains(lower, "scheduler") || strings.Contains(lower, "busy") || strings.Contains(lower, "queue") {
			return http.StatusTooManyRequests, "MODEL_SCHEDULER_BUSY", message
		}
		if strings.Contains(lower, "policy") || strings.Contains(lower, "permission") || strings.Contains(lower, "denied") {
			return http.StatusForbidden, "MODEL_POLICY_DENIED", message
		}
	}
	return status, code, message
}

func modelRuntimeMetaFromRequestAudit(meta requestAuditMeta) ModelRuntimeRequestMeta {
	return ModelRuntimeRequestMeta{
		CorrelationID: meta.CorrelationID,
		TraceID:       meta.TraceID,
		WorkspaceID:   meta.WorkspaceID,
	}
}

func decodeOptionalJSONBody(r *http.Request, target any) error {
	if r.Body == nil {
		return nil
	}
	raw, err := readModelRuntimeRequestBody(r)
	if err != nil {
		return err
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return err
	}
	return nil
}

func decodeModelRuntimeJSONBody(r *http.Request, target any) error {
	raw, err := readModelRuntimeRequestBody(r)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return err
	}
	return nil
}

func readModelRuntimeRequestBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, io.EOF
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, modelRuntimeRequestBodyLimit+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > modelRuntimeRequestBodyLimit {
		return nil, &modelRuntimeError{
			status:  http.StatusRequestEntityTooLarge,
			code:    "REQUEST_BODY_TOO_LARGE",
			message: fmt.Sprintf("model runtime request body too large: limit %d bytes", modelRuntimeRequestBodyLimit),
		}
	}
	return raw, nil
}

func (s *Server) writeModelRuntimeDecodeError(w http.ResponseWriter, err error, meta ModelRuntimeRequestMeta) {
	var runtimeErr *modelRuntimeError
	if errors.As(err, &runtimeErr) {
		s.writeModelRuntimeError(w, runtimeErr, meta)
		return
	}
	s.writeModelRuntimeError(w, &modelRuntimeError{status: http.StatusBadRequest, code: "INVALID_JSON", message: "invalid json body"}, meta)
}

func validateChatMessages(messages []ModelRuntimeChatMessage) error {
	for i, msg := range messages {
		role := strings.TrimSpace(msg.Role)
		content := strings.TrimSpace(msg.Content)
		if role == "" {
			return &modelRuntimeError{
				status:  http.StatusBadRequest,
				code:    "MESSAGE_ROLE_REQUIRED",
				message: fmt.Sprintf("messages[%d].role is required", i),
			}
		}
		switch role {
		case "system", "user", "assistant", "tool":
		default:
			return &modelRuntimeError{
				status:  http.StatusBadRequest,
				code:    "MESSAGE_ROLE_INVALID",
				message: fmt.Sprintf("messages[%d].role %q is invalid", i, role),
			}
		}
		if content == "" {
			return &modelRuntimeError{
				status:  http.StatusBadRequest,
				code:    "MESSAGE_CONTENT_REQUIRED",
				message: fmt.Sprintf("messages[%d].content is required", i),
			}
		}
	}
	return nil
}
