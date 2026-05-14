package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"forge/projectforge/services/core/internal/approvals"
	"forge/projectforge/services/core/internal/audit"
)

const modelManagementCapabilityID = "model.management"

type modelManagementRisk string

const (
	modelManagementRiskLow    modelManagementRisk = "low"
	modelManagementRiskMedium modelManagementRisk = "medium"
	modelManagementRiskHigh   modelManagementRisk = "high"
)

type modelManagementGovernanceRequest struct {
	Operation     string
	ModelID       string
	Path          string
	Backend       string
	Provider      string
	RiskClass     string
	Actor         string
	Source        string
	WorkspaceID   string
	LaneID        string
	CorrelationID string
	TraceID       string
	CapabilityID  string
	ApprovalID    string
	Preferred     bool
	DryRun        bool
	Metadata      map[string]any
}

type modelManagementGovernanceDecision struct {
	Operation         string              `json:"operation"`
	ModelID           string              `json:"modelId,omitempty"`
	Backend           string              `json:"backend,omitempty"`
	Provider          string              `json:"provider,omitempty"`
	RiskClass         string              `json:"riskClass"`
	CapabilityID      string              `json:"capabilityId"`
	ApprovalID        string              `json:"approvalId,omitempty"`
	ApprovalRequestID *int64              `json:"approvalRequestId,omitempty"`
	RequiresApproval  bool                `json:"requiresApproval"`
	Approved          bool                `json:"approved"`
	DryRun            bool                `json:"dryRun"`
	Reason            string              `json:"reason,omitempty"`
	Metadata          map[string]any      `json:"metadata,omitempty"`
	Fields            map[string]any      `json:"-"`
	Scope             map[string]any      `json:"-"`
	HTTPStatus        int                 `json:"-"`
	ErrorCode         string              `json:"-"`
	ErrorMessage      string              `json:"-"`
	Risk              modelManagementRisk `json:"-"`
}

func (s *Server) enforceModelManagementGovernance(ctx context.Context, runtimeSvc modelRuntimeService, req modelManagementGovernanceRequest) (modelManagementGovernanceDecision, error) {
	req.Operation = strings.TrimSpace(req.Operation)
	req.ModelID = strings.TrimSpace(req.ModelID)
	req.Actor = strings.TrimSpace(req.Actor)
	req.Source = strings.TrimSpace(req.Source)
	req.WorkspaceID = strings.TrimSpace(req.WorkspaceID)
	req.LaneID = firstNonEmptyTrimmed(req.LaneID, metadataStringAny(req.Metadata, "laneId"))
	req.CorrelationID = strings.TrimSpace(req.CorrelationID)
	req.TraceID = strings.TrimSpace(req.TraceID)
	req.CapabilityID = firstNonEmptyTrimmed(req.CapabilityID, metadataStringAny(req.Metadata, "capabilityId"))
	req.ApprovalID = firstNonEmptyTrimmed(req.ApprovalID, metadataStringAny(req.Metadata, "approvalId"))
	req.Provider = strings.TrimSpace(req.Provider)
	req.Backend = strings.TrimSpace(req.Backend)
	if req.WorkspaceID == "" {
		req.WorkspaceID = "workspace:default"
	}

	risk, backend, provider := s.modelManagementRisk(ctx, runtimeSvc, req)
	req.Backend = firstNonEmptyTrimmed(req.Backend, backend)
	req.Provider = firstNonEmptyTrimmed(req.Provider, provider)
	req.RiskClass = string(risk)

	decision := modelManagementGovernanceDecision{
		Operation:        req.Operation,
		ModelID:          req.ModelID,
		Backend:          req.Backend,
		Provider:         req.Provider,
		RiskClass:        string(risk),
		CapabilityID:     modelManagementCapabilityID,
		RequiresApproval: risk == modelManagementRiskHigh,
		DryRun:           req.DryRun,
		Risk:             risk,
		Metadata: map[string]any{
			"authority": "modelruntime_management",
		},
	}

	if req.Actor == "" {
		decision.HTTPStatus = http.StatusForbidden
		decision.ErrorCode = "MODEL_GOVERNANCE_ACTOR_REQUIRED"
		decision.ErrorMessage = "model management requires explicit actor"
		decision.Reason = decision.ErrorMessage
		return decision, nil
	}
	if req.Source == "" {
		decision.HTTPStatus = http.StatusForbidden
		decision.ErrorCode = "MODEL_GOVERNANCE_SOURCE_REQUIRED"
		decision.ErrorMessage = "model management requires explicit source"
		decision.Reason = decision.ErrorMessage
		return decision, nil
	}
	if req.CapabilityID != modelManagementCapabilityID {
		decision.HTTPStatus = http.StatusForbidden
		decision.ErrorCode = "MODEL_GOVERNANCE_CAPABILITY_REQUIRED"
		decision.ErrorMessage = "model management requires capability model.management"
		decision.Reason = decision.ErrorMessage
		return decision, nil
	}
	if risk == modelManagementRiskHigh && (isExternalModelBackend(req.Backend) || isExternalProvider(req.Provider)) && !s.modelManagementProviderConfigured(req.Backend, req.Provider) {
		decision.HTTPStatus = http.StatusForbidden
		decision.ErrorCode = "MODEL_GOVERNANCE_PROVIDER_CONFIG_REQUIRED"
		decision.ErrorMessage = "provider-backed model management requires explicit provider configuration"
		decision.Reason = decision.ErrorMessage
		return decision, nil
	}

	shapeHash, fields := s.modelManagementApprovalFingerprint(req, 0)
	scope := map[string]any{
		"approvalFingerprintVersion": "model.management.v1",
		"approvalShapeHash":          shapeHash,
		"approvalFingerprintHash":    shapeHash,
		"approvalFingerprintFields":  fields,
		"operation":                  req.Operation,
		"modelId":                    req.ModelID,
		"backend":                    req.Backend,
		"provider":                   req.Provider,
		"riskClass":                  string(risk),
		"capabilityId":               modelManagementCapabilityID,
		"laneId":                     req.LaneID,
		"workspaceId":                req.WorkspaceID,
		"writeIntent":                true,
	}
	decision.Fields = fields
	decision.Scope = scope

	if risk != modelManagementRiskHigh {
		decision.Approved = true
		decision.Reason = "model management policy allowed"
		return decision, nil
	}

	if req.DryRun {
		decision.Reason = "approval required for high-risk model management operation"
		return decision, nil
	}

	if req.ApprovalID == "" {
		if s.approvals == nil {
			decision.HTTPStatus = http.StatusForbidden
			decision.ErrorCode = "MODEL_GOVERNANCE_APPROVAL_UNAVAILABLE"
			decision.ErrorMessage = "approval service unavailable for high-risk model management operation"
			decision.Reason = decision.ErrorMessage
			return decision, nil
		}
		jobID := modelManagementApprovalJobID(shapeHash)
		if err := s.ensureModelManagementApprovalJob(ctx, jobID, req, string(risk)); err != nil {
			return decision, err
		}
		ar, err := s.approvals.OpenRequestForJob(ctx, jobID, approvals.CreateRequestInput{
			JobID:            jobID,
			RequestedAction:  "model.runtime." + req.Operation,
			RiskClass:        string(modelManagementRiskHigh),
			RequestedAdapter: "modelruntime",
			WriteIntent:      true,
			ScopeSnapshot:    scope,
			RequestSummary:   fmt.Sprintf("Model management %s for %s", req.Operation, firstNonEmptyTrimmed(req.ModelID, req.Path, "model")),
		})
		if err != nil {
			return decision, err
		}
		if ar != nil {
			v := ar.ID
			decision.ApprovalRequestID = &v
			grantHash, grantFields := s.modelManagementApprovalFingerprint(req, ar.ID)
			scope["approvalRequestId"] = ar.ID
			scope["approvalFingerprintHash"] = grantHash
			scope["approvalFingerprintFields"] = grantFields
			if err := s.updateModelManagementApprovalScope(ctx, ar.ID, scope); err != nil {
				return decision, err
			}
		}
		decision.Reason = "approval required for high-risk model management operation"
		return decision, nil
	}

	requestID, err := strconv.ParseInt(req.ApprovalID, 10, 64)
	if err != nil || requestID <= 0 {
		decision.HTTPStatus = http.StatusForbidden
		decision.ErrorCode = "MODEL_GOVERNANCE_APPROVAL_INVALID"
		decision.ErrorMessage = "approvalId must be a positive integer"
		decision.Reason = decision.ErrorMessage
		return decision, nil
	}
	if s.approvals == nil {
		decision.HTTPStatus = http.StatusForbidden
		decision.ErrorCode = "MODEL_GOVERNANCE_APPROVAL_UNAVAILABLE"
		decision.ErrorMessage = "approval service unavailable for high-risk model management operation"
		decision.Reason = decision.ErrorMessage
		return decision, nil
	}
	ar, err := s.approvals.GetRequest(ctx, requestID)
	if err != nil {
		decision.HTTPStatus = http.StatusForbidden
		decision.ErrorCode = "MODEL_GOVERNANCE_APPROVAL_NOT_FOUND"
		decision.ErrorMessage = fmt.Sprintf("approval request %d not found", requestID)
		decision.Reason = decision.ErrorMessage
		return decision, nil
	}
	expected := modelManagementApprovalHashFromScope(ar.ScopeSnapshot)
	actual, _ := s.modelManagementApprovalFingerprint(req, requestID)
	if expected == "" {
		decision.HTTPStatus = http.StatusForbidden
		decision.ErrorCode = "MODEL_GOVERNANCE_APPROVAL_FINGERPRINT_MISSING"
		decision.ErrorMessage = fmt.Sprintf("approval request %d is missing model management fingerprint", requestID)
		decision.Reason = decision.ErrorMessage
		return decision, nil
	}
	if actual != expected {
		decision.HTTPStatus = http.StatusForbidden
		decision.ErrorCode = "MODEL_GOVERNANCE_APPROVAL_FINGERPRINT_MISMATCH"
		decision.ErrorMessage = fmt.Sprintf("approval request %d fingerprint mismatch", requestID)
		decision.Reason = decision.ErrorMessage
		return decision, nil
	}
	if ar.Decision == nil || !strings.EqualFold(strings.TrimSpace(ar.Decision.Decision), "approved") {
		decision.HTTPStatus = http.StatusForbidden
		decision.ErrorCode = "MODEL_GOVERNANCE_APPROVAL_REQUIRED"
		decision.ErrorMessage = fmt.Sprintf("approval request %d is not approved", requestID)
		decision.Reason = decision.ErrorMessage
		return decision, nil
	}
	decision.ApprovalID = req.ApprovalID
	decision.Approved = true
	decision.Reason = "approved model management operation"
	return decision, nil
}

func (s *Server) writeModelManagementGovernanceResult(w http.ResponseWriter, r *http.Request, meta ModelRuntimeRequestMeta, req modelManagementGovernanceRequest, decision modelManagementGovernanceDecision) bool {
	if decision.HTTPStatus > 0 {
		s.auditModelManagementGovernance(r.Context(), req, decision, "denied")
		s.writeModelRuntimeError(w, &modelRuntimeError{status: decision.HTTPStatus, code: decision.ErrorCode, message: decision.ErrorMessage}, meta)
		return true
	}
	if decision.DryRun {
		s.auditModelManagementGovernance(r.Context(), req, decision, "dry_run")
		writeJSON(w, http.StatusOK, map[string]any{
			"governance":    decision,
			"correlationId": meta.CorrelationID,
			"traceId":       meta.TraceID,
			"workspaceId":   meta.WorkspaceID,
		})
		return true
	}
	if decision.RequiresApproval && !decision.Approved {
		s.auditModelManagementGovernance(r.Context(), req, decision, "needs_approval")
		writeJSON(w, http.StatusAccepted, map[string]any{
			"governance":    decision,
			"correlationId": meta.CorrelationID,
			"traceId":       meta.TraceID,
			"workspaceId":   meta.WorkspaceID,
		})
		return true
	}
	s.auditModelManagementGovernance(r.Context(), req, decision, "authorized")
	return false
}

func (s *Server) modelManagementRisk(ctx context.Context, runtimeSvc modelRuntimeService, req modelManagementGovernanceRequest) (modelManagementRisk, string, string) {
	op := strings.ToLower(strings.TrimSpace(req.Operation))
	backend := strings.TrimSpace(req.Backend)
	provider := strings.TrimSpace(req.Provider)
	if req.ModelID != "" && runtimeSvc != nil {
		if model, err := runtimeSvc.GetModel(ctx, req.ModelID, ModelRuntimeRequestMeta{
			CorrelationID: req.CorrelationID,
			TraceID:       req.TraceID,
			WorkspaceID:   req.WorkspaceID,
		}); err == nil {
			backend = firstNonEmptyTrimmed(backend, model.Backend)
			provider = firstNonEmptyTrimmed(provider, metadataStringAny(model.Metadata, "provider"), metadataStringAny(model.Metadata, "source"))
		}
	}
	if provider == "" && backend != "" {
		provider = backend
	}

	switch op {
	case "import", "archive", "remove", "delete_file", "load", "unload":
		return modelManagementRiskHigh, backend, provider
	case "enable":
		if req.Preferred || isExternalModelBackend(backend) || isExternalProvider(provider) {
			return modelManagementRiskHigh, backend, provider
		}
		return modelManagementRiskMedium, backend, provider
	case "scan", "verify", "disable":
		return modelManagementRiskMedium, backend, provider
	default:
		return modelManagementRiskMedium, backend, provider
	}
}

func isExternalModelBackend(backend string) bool {
	switch strings.ToLower(strings.TrimSpace(backend)) {
	case "openai_compat", "vllm":
		return true
	default:
		return false
	}
}

func isExternalProvider(provider string) bool {
	provider = strings.ToLower(strings.TrimSpace(provider))
	return strings.HasPrefix(provider, "http://") || strings.HasPrefix(provider, "https://") || strings.Contains(provider, "openai") || strings.Contains(provider, "vllm")
}

func (s *Server) modelManagementProviderConfigured(backend, provider string) bool {
	backend = strings.ToLower(strings.TrimSpace(backend))
	provider = strings.ToLower(strings.TrimSpace(provider))
	switch {
	case backend == "openai_compat" || strings.Contains(provider, "openai"):
		return strings.TrimSpace(s.cfg.ModelOpenAICompatEndpoint) != ""
	case backend == "vllm" || strings.Contains(provider, "vllm"):
		return strings.TrimSpace(s.cfg.ModelVLLMEndpoint) != ""
	case strings.HasPrefix(provider, "http://") || strings.HasPrefix(provider, "https://"):
		return strings.TrimSpace(s.cfg.ModelOpenAICompatEndpoint) == provider || strings.TrimSpace(s.cfg.ModelVLLMEndpoint) == provider
	default:
		return true
	}
}

func (s *Server) modelManagementApprovalFingerprint(req modelManagementGovernanceRequest, approvalRequestID int64) (string, map[string]any) {
	cleanPath := ""
	if strings.TrimSpace(req.Path) != "" {
		cleanPath = filepath.ToSlash(filepath.Clean(strings.TrimSpace(req.Path)))
	}
	fields := map[string]any{
		"version":     "model.management.v1",
		"operation":   strings.ToLower(strings.TrimSpace(req.Operation)),
		"modelId":     strings.TrimSpace(req.ModelID),
		"path":        cleanPath,
		"backend":     strings.TrimSpace(req.Backend),
		"provider":    strings.TrimSpace(req.Provider),
		"actor":       strings.TrimSpace(req.Actor),
		"source":      strings.TrimSpace(req.Source),
		"workspaceId": strings.TrimSpace(req.WorkspaceID),
		"laneId":      strings.TrimSpace(req.LaneID),
		"capability":  modelManagementCapabilityID,
		"riskClass":   strings.TrimSpace(req.RiskClass),
		"writeIntent": true,
		"preferred":   req.Preferred,
	}
	if approvalRequestID > 0 {
		fields["approvalRequestId"] = approvalRequestID
	}
	body, _ := json.Marshal(fields)
	sum := sha256.Sum256(body)
	return "sha256:" + hex.EncodeToString(sum[:]), fields
}

func modelManagementApprovalHashFromScope(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var scope map[string]any
	if err := json.Unmarshal(raw, &scope); err != nil {
		return ""
	}
	if v, ok := scope["approvalFingerprintHash"].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

func modelManagementApprovalJobID(shapeHash string) string {
	clean := strings.TrimPrefix(strings.TrimSpace(shapeHash), "sha256:")
	if len(clean) > 24 {
		clean = clean[:24]
	}
	if clean == "" {
		clean = "unknown"
	}
	return "model-mgmt-" + clean
}

func (s *Server) updateModelManagementApprovalScope(ctx context.Context, requestID int64, scope map[string]any) error {
	if s == nil || s.st == nil || s.st.DB == nil {
		return fmt.Errorf("store unavailable")
	}
	raw, err := json.Marshal(scope)
	if err != nil {
		return err
	}
	_, err = s.st.DB.ExecContext(ctx, `UPDATE approval_requests SET scope_snapshot_json = ? WHERE id = ?`, string(raw), requestID)
	return err
}

func (s *Server) ensureModelManagementApprovalJob(ctx context.Context, jobID string, req modelManagementGovernanceRequest, riskClass string) error {
	if s == nil || s.st == nil || s.st.DB == nil {
		return fmt.Errorf("store unavailable")
	}
	now := time.Now().UnixMilli()
	meta := map[string]any{
		"templateId":    "model_management_approval",
		"operation":     req.Operation,
		"modelId":       req.ModelID,
		"path":          req.Path,
		"backend":       req.Backend,
		"provider":      req.Provider,
		"workspaceId":   req.WorkspaceID,
		"laneId":        req.LaneID,
		"correlationId": req.CorrelationID,
		"traceId":       req.TraceID,
	}
	raw, _ := json.Marshal(meta)
	_, err := s.st.DB.ExecContext(ctx, `
INSERT OR IGNORE INTO jobs(
  id, created_at, updated_at, queued_at,
  title, requested_action, target_adapter, initiating_source,
  execution_boundary, risk_class, status, approval_status, write_intent,
  metadata_json
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		jobID,
		now,
		now,
		nil,
		"Model management approval",
		"model.runtime."+strings.TrimSpace(req.Operation),
		"modelruntime",
		firstNonEmptyTrimmed(req.Source, "forge_api"),
		"runtime_authority",
		riskClass,
		"awaiting_approval",
		"pending",
		1,
		string(raw),
	)
	if err != nil {
		return fmt.Errorf("insert model management approval job: %w", err)
	}
	return nil
}

func (s *Server) auditModelManagementGovernance(ctx context.Context, req modelManagementGovernanceRequest, decision modelManagementGovernanceDecision, outcome string) {
	if s == nil || s.auditSvc == nil {
		return
	}
	payload := map[string]any{
		"operation":         req.Operation,
		"modelId":           req.ModelID,
		"path":              req.Path,
		"backend":           decision.Backend,
		"provider":          decision.Provider,
		"riskClass":         decision.RiskClass,
		"capabilityId":      modelManagementCapabilityID,
		"approvalId":        decision.ApprovalID,
		"approvalRequestId": decision.ApprovalRequestID,
		"correlationId":     req.CorrelationID,
		"traceId":           req.TraceID,
		"workspaceId":       req.WorkspaceID,
		"laneId":            req.LaneID,
		"actor":             req.Actor,
		"source":            req.Source,
		"reason":            decision.Reason,
		"dryRun":            req.DryRun,
	}
	_, _ = s.auditSvc.Record(ctx, audit.CreateRequest{
		CorrelationID:     req.CorrelationID,
		Category:          "model_runtime",
		Action:            "model." + strings.TrimSpace(req.Operation),
		Actor:             firstNonEmptyTrimmed(req.Actor, "api"),
		SubjectType:       "model",
		SubjectID:         firstNonEmptyTrimmed(req.ModelID, req.Path, "model"),
		ApprovalRequestID: decision.ApprovalRequestID,
		RiskClass:         decision.RiskClass,
		Outcome:           outcome,
		Summary:           fmt.Sprintf("model management %s %s", req.Operation, outcome),
		Payload:           payload,
	})
}

func modelManagementMetadata(meta map[string]any, decision modelManagementGovernanceDecision) map[string]any {
	out := cloneAnyMap(meta)
	out["modelGovernance"] = map[string]any{
		"operation":         decision.Operation,
		"riskClass":         decision.RiskClass,
		"capabilityId":      modelManagementCapabilityID,
		"approvalId":        decision.ApprovalID,
		"approvalRequestId": decision.ApprovalRequestID,
	}
	out["riskClass"] = decision.RiskClass
	out["capabilityId"] = modelManagementCapabilityID
	if decision.ApprovalID != "" {
		out["approvalId"] = decision.ApprovalID
	}
	if decision.ApprovalRequestID != nil {
		out["approvalRequestId"] = *decision.ApprovalRequestID
	}
	return out
}

func metadataStringAny(meta map[string]any, key string) string {
	if meta == nil {
		return ""
	}
	if v, ok := meta[key].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}
