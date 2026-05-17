package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/approvals"
	"forge/projectforge/services/core/internal/audit"
	"forge/projectforge/services/core/internal/backup"
	"forge/projectforge/services/core/internal/gateway"
	"forge/projectforge/services/core/internal/lanes"
	"forge/projectforge/services/core/internal/permissions"
	"forge/projectforge/services/core/internal/release"
)

// Phase 5 handlers: tool execution gateway, action lanes, permission
// profiles, audit traces, backup / export / import, and release readiness.

const phase5JSONRequestBodyLimit int64 = 1 << 20

var errPhase5RequestBodyTooLarge = errors.New("phase5 json request body too large")

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
	Status        string `json:"status"`
	Reason        string `json:"reason"`
	Actor         string `json:"actor"`
	ActorKind     string `json:"actorKind"`
	Source        string `json:"source"`
	ApprovalID    string `json:"approvalId"`
	CorrelationID string `json:"correlationId"`
	TraceID       string `json:"traceId"`
	DryRun        bool   `json:"dryRun"`
}

func decodePhase5JSONBody(w http.ResponseWriter, r *http.Request, dst any) error {
	body, err := readPhase5RequestBody(w, r)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, dst); err != nil {
		return err
	}
	return nil
}

func readPhase5RequestBody(w http.ResponseWriter, r *http.Request) ([]byte, error) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, phase5JSONRequestBodyLimit))
	if err != nil {
		if strings.Contains(err.Error(), "request body too large") {
			return nil, errPhase5RequestBodyTooLarge
		}
		return nil, err
	}
	return body, nil
}

func writePhase5DecodeError(w http.ResponseWriter, err error) {
	if errors.Is(err, errPhase5RequestBodyTooLarge) {
		http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
		return
	}
	http.Error(w, "invalid json", http.StatusBadRequest)
}

func requiresCapabilityStatusReason(status domain.ToolCapabilityStatus) bool {
	return domain.IsKnownToolCapabilityStatus(status)
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
	if err := decodePhase5JSONBody(w, r, &body); err != nil {
		writePhase5DecodeError(w, err)
		return
	}
	actor := authenticatedActorName(r)
	actorSource := authenticatedActorSource(r)
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
		ProvenanceActor:     actor,
		ProvenanceActorType: actorSource,
		Paths:               body.Paths,
		Input:               body.Input,
		JobID:               body.JobID,
		PacketID:            body.PacketID,
		Initiator:           actor,
		DryRun:              body.DryRun,
		Metadata:            body.Metadata,
	})
	if err != nil {
		writeAPIRequestError(w, http.StatusInternalServerError, err)
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
		writeAPIRequestError(w, http.StatusInternalServerError, err)
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
	if err := decodePhase5JSONBody(w, r, &body); err != nil {
		writePhase5DecodeError(w, err)
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
	if s.gateway == nil {
		http.Error(w, "gateway unavailable", http.StatusServiceUnavailable)
		return
	}

	previous, ok := s.gateway.Capability(id)
	if !ok {
		http.Error(w, "capability not found", http.StatusNotFound)
		return
	}
	if previous.Status != parsedStatus && requiresCapabilityStatusReason(parsedStatus) && reason == "" {
		http.Error(w, "reason is required for capability status changes", http.StatusBadRequest)
		return
	}
	transition := gateway.ClassifyCapabilityStatusTransition(previous, parsedStatus)
	actor := strings.TrimSpace(body.Actor)
	if actor == "" {
		actor = strings.TrimSpace(r.Header.Get("X-Forge-Actor"))
	}
	if actor == "" {
		actor = strings.TrimSpace(r.Header.Get("X-Actor"))
	}
	if actor == "" && previous.Status != parsedStatus {
		http.Error(w, "actor is required for capability status changes", http.StatusBadRequest)
		return
	}
	actorKind := strings.TrimSpace(body.ActorKind)
	if actorKind == "" {
		actorKind = strings.TrimSpace(body.Source)
	}
	if actorKind == "" {
		actorKind = strings.TrimSpace(r.Header.Get("X-Forge-Actor-Kind"))
	}
	if actorKind == "" && previous.Status != parsedStatus {
		http.Error(w, "actorKind or source is required for capability status changes", http.StatusBadRequest)
		return
	}
	correlationID := strings.TrimSpace(body.CorrelationID)
	if correlationID == "" {
		correlationID = middleware.GetReqID(ctx)
	}
	traceID := strings.TrimSpace(body.TraceID)
	gov := s.evaluateCapabilityStatusGovernance(ctx, capabilityStatusGovernanceRequest{
		Capability:    previous,
		Requested:     parsedStatus,
		Reason:        reason,
		Actor:         actor,
		ActorKind:     actorKind,
		Source:        body.Source,
		ApprovalID:    body.ApprovalID,
		CorrelationID: correlationID,
		TraceID:       traceID,
		DryRun:        body.DryRun,
		Transition:    transition,
	})
	if gov.HTTPStatus > 0 {
		s.auditCapabilityStatusGovernance(ctx, previous, parsedStatus, gov, "denied")
		writeJSON(w, gov.HTTPStatus, map[string]any{
			"success":          false,
			"capabilityId":     previous.ID,
			"previousStatus":   string(previous.Status),
			"newStatus":        status,
			"riskClass":        transition.RiskClass,
			"approvalRequired": transition.RequiresApproval,
			"rejectionReason":  gov.ErrorMessage,
			"errorCode":        gov.ErrorCode,
			"correlationId":    correlationID,
			"traceId":          traceID,
		})
		return
	}
	if gov.RequiresApproval && !gov.Approved {
		s.auditCapabilityStatusGovernance(ctx, previous, parsedStatus, gov, "needs_approval")
		writeJSON(w, http.StatusAccepted, map[string]any{
			"success":           false,
			"capabilityId":      previous.ID,
			"previousStatus":    string(previous.Status),
			"newStatus":         status,
			"riskClass":         transition.RiskClass,
			"approvalRequired":  true,
			"approvalRequestId": gov.ApprovalRequestID,
			"rejectionReason":   gov.Reason,
			"correlationId":     correlationID,
			"traceId":           traceID,
		})
		return
	}
	if body.DryRun {
		s.auditCapabilityStatusGovernance(ctx, previous, parsedStatus, gov, "dry_run")
		writeJSON(w, http.StatusOK, map[string]any{
			"success":          true,
			"dryRun":           true,
			"capabilityId":     previous.ID,
			"previousStatus":   string(previous.Status),
			"newStatus":        status,
			"riskClass":        transition.RiskClass,
			"approvalRequired": transition.RequiresApproval,
			"correlationId":    correlationID,
			"traceId":          traceID,
		})
		return
	}

	approvalRequestID := gov.ApprovalRequestID
	previous, updated, ok, err := s.gateway.UpdateCapabilityStatusWithMetadata(ctx, id, parsedStatus, gateway.CapabilityStatusUpdateMetadata{
		Actor:             actor,
		ActorKind:         actorKind,
		Reason:            reason,
		RiskClass:         string(previous.Risk),
		TransitionRisk:    transition.RiskClass,
		ApprovalRequestID: approvalRequestID,
		CorrelationID:     correlationID,
		TraceID:           traceID,
	})
	if err != nil {
		writeAPIRequestError(w, http.StatusBadRequest, err)
		return
	}
	if !ok {
		http.Error(w, "capability not found", http.StatusNotFound)
		return
	}

	s.auditCapabilityStatusGovernance(ctx, previous, parsedStatus, gov, "ok")

	writeJSON(w, http.StatusOK, map[string]any{
		"success":           true,
		"capability":        updated,
		"previousStatus":    string(previous.Status),
		"newStatus":         string(updated.Status),
		"riskClass":         transition.RiskClass,
		"approvalRequired":  transition.RequiresApproval,
		"approvalRequestId": approvalRequestID,
		"correlationId":     correlationID,
		"traceId":           traceID,
		"auditCategory":     "tool.capability.status.updated",
	})
}

type capabilityStatusGovernanceRequest struct {
	Capability    domain.ToolCapability
	Requested     domain.ToolCapabilityStatus
	Reason        string
	Actor         string
	ActorKind     string
	Source        string
	ApprovalID    string
	CorrelationID string
	TraceID       string
	DryRun        bool
	Transition    gateway.CapabilityStatusTransition
}

type capabilityStatusGovernanceDecision struct {
	RequiresApproval  bool
	Approved          bool
	DryRun            bool
	ApprovalID        string
	ApprovalRequestID *int64
	Reason            string
	CorrelationID     string
	TraceID           string
	Actor             string
	ActorKind         string
	ErrorCode         string
	ErrorMessage      string
	HTTPStatus        int
	ShapeHash         string
	Fields            map[string]any
}

func (s *Server) evaluateCapabilityStatusGovernance(ctx context.Context, req capabilityStatusGovernanceRequest) capabilityStatusGovernanceDecision {
	shapeHash, fields := capabilityStatusApprovalFingerprint(req, 0)
	decision := capabilityStatusGovernanceDecision{
		RequiresApproval: req.Transition.RequiresApproval,
		DryRun:           req.DryRun,
		Reason:           "capability status governance allowed",
		ShapeHash:        shapeHash,
		Fields:           fields,
		CorrelationID:    req.CorrelationID,
		TraceID:          req.TraceID,
		Actor:            req.Actor,
		ActorKind:        req.ActorKind,
	}
	if !req.Transition.RequiresApproval {
		decision.Approved = true
		return decision
	}
	decision.Reason = "approval required for high-risk capability status transition"
	if req.DryRun {
		return decision
	}
	if strings.TrimSpace(req.ApprovalID) == "" {
		if s.approvals == nil {
			decision.HTTPStatus = http.StatusForbidden
			decision.ErrorCode = "CAPABILITY_STATUS_APPROVAL_UNAVAILABLE"
			decision.ErrorMessage = "approval service unavailable for high-risk capability status transition"
			decision.Reason = decision.ErrorMessage
			return decision
		}
		jobID := capabilityStatusApprovalJobID(shapeHash)
		if err := s.ensureCapabilityStatusApprovalJob(ctx, jobID, req); err != nil {
			decision.HTTPStatus = http.StatusInternalServerError
			decision.ErrorCode = "CAPABILITY_STATUS_APPROVAL_JOB_FAILED"
			decision.ErrorMessage = err.Error()
			decision.Reason = decision.ErrorMessage
			return decision
		}
		scope := capabilityStatusApprovalScope(req, shapeHash, shapeHash, fields, 0)
		ar, err := s.approvals.OpenRequestForJob(ctx, jobID, approvals.CreateRequestInput{
			JobID:            jobID,
			RequestedAction:  "gateway.capability.status.update",
			RiskClass:        req.Transition.RiskClass,
			RequestedAdapter: "gateway",
			WriteIntent:      true,
			ScopeSnapshot:    scope,
			RequestSummary:   fmt.Sprintf("Capability status change %s: %s -> %s", req.Capability.ID, req.Capability.Status, req.Requested),
		})
		if err != nil {
			decision.HTTPStatus = http.StatusInternalServerError
			decision.ErrorCode = "CAPABILITY_STATUS_APPROVAL_REQUEST_FAILED"
			decision.ErrorMessage = err.Error()
			decision.Reason = decision.ErrorMessage
			return decision
		}
		if ar != nil {
			v := ar.ID
			decision.ApprovalRequestID = &v
			grantHash, grantFields := capabilityStatusApprovalFingerprint(req, ar.ID)
			scope = capabilityStatusApprovalScope(req, shapeHash, grantHash, grantFields, ar.ID)
			if err := s.updateCapabilityStatusApprovalScope(ctx, ar.ID, scope); err != nil {
				decision.HTTPStatus = http.StatusInternalServerError
				decision.ErrorCode = "CAPABILITY_STATUS_APPROVAL_SCOPE_FAILED"
				decision.ErrorMessage = err.Error()
				decision.Reason = decision.ErrorMessage
				return decision
			}
		}
		return decision
	}
	requestID, err := strconv.ParseInt(strings.TrimSpace(req.ApprovalID), 10, 64)
	if err != nil || requestID <= 0 {
		decision.HTTPStatus = http.StatusForbidden
		decision.ErrorCode = "CAPABILITY_STATUS_APPROVAL_INVALID"
		decision.ErrorMessage = "approvalId must be a positive integer"
		decision.Reason = decision.ErrorMessage
		return decision
	}
	if s.approvals == nil {
		decision.HTTPStatus = http.StatusForbidden
		decision.ErrorCode = "CAPABILITY_STATUS_APPROVAL_UNAVAILABLE"
		decision.ErrorMessage = "approval service unavailable for high-risk capability status transition"
		decision.Reason = decision.ErrorMessage
		return decision
	}
	ar, err := s.approvals.GetRequest(ctx, requestID)
	if err != nil {
		decision.HTTPStatus = http.StatusForbidden
		decision.ErrorCode = "CAPABILITY_STATUS_APPROVAL_NOT_FOUND"
		decision.ErrorMessage = fmt.Sprintf("approval request %d not found", requestID)
		decision.Reason = decision.ErrorMessage
		return decision
	}
	expected := capabilityStatusApprovalHashFromScope(ar.ScopeSnapshot)
	actual, _ := capabilityStatusApprovalFingerprint(req, requestID)
	if expected == "" {
		decision.HTTPStatus = http.StatusForbidden
		decision.ErrorCode = "CAPABILITY_STATUS_APPROVAL_FINGERPRINT_MISSING"
		decision.ErrorMessage = fmt.Sprintf("approval request %d is missing capability status fingerprint", requestID)
		decision.Reason = decision.ErrorMessage
		return decision
	}
	if actual != expected {
		decision.HTTPStatus = http.StatusForbidden
		decision.ErrorCode = "CAPABILITY_STATUS_APPROVAL_FINGERPRINT_MISMATCH"
		decision.ErrorMessage = fmt.Sprintf("approval request %d fingerprint mismatch", requestID)
		decision.Reason = decision.ErrorMessage
		return decision
	}
	if ar.Decision == nil || !strings.EqualFold(strings.TrimSpace(ar.Decision.Decision), "approved") {
		decision.HTTPStatus = http.StatusForbidden
		decision.ErrorCode = "CAPABILITY_STATUS_APPROVAL_REQUIRED"
		decision.ErrorMessage = fmt.Sprintf("approval request %d is not approved", requestID)
		decision.Reason = decision.ErrorMessage
		return decision
	}
	decision.ApprovalID = req.ApprovalID
	decision.ApprovalRequestID = &requestID
	decision.Approved = true
	decision.Reason = "approved capability status transition"
	return decision
}

func capabilityStatusApprovalFingerprint(req capabilityStatusGovernanceRequest, approvalRequestID int64) (string, map[string]any) {
	effects := make([]string, 0, len(req.Capability.Effect))
	for _, effect := range req.Capability.Effect {
		effects = append(effects, string(effect))
	}
	fields := map[string]any{
		"version":        "gateway.capability.status.v1",
		"capabilityId":   strings.TrimSpace(req.Capability.ID),
		"previousStatus": string(req.Capability.Status),
		"newStatus":      string(req.Requested),
		"capabilityRisk": string(req.Capability.Risk),
		"transitionRisk": req.Transition.RiskClass,
		"effects":        effects,
		"actor":          strings.TrimSpace(req.Actor),
		"actorKind":      strings.TrimSpace(req.ActorKind),
		"source":         strings.TrimSpace(req.Source),
		"reason":         strings.TrimSpace(req.Reason),
		"writeIntent":    true,
	}
	if approvalRequestID > 0 {
		fields["approvalRequestId"] = approvalRequestID
	}
	body, _ := json.Marshal(fields)
	sum := sha256.Sum256(body)
	return "sha256:" + hex.EncodeToString(sum[:]), fields
}

func capabilityStatusApprovalScope(req capabilityStatusGovernanceRequest, shapeHash, fingerprintHash string, fields map[string]any, requestID int64) map[string]any {
	scope := map[string]any{
		"approvalFingerprintVersion": "gateway.capability.status.v1",
		"approvalShapeHash":          shapeHash,
		"approvalFingerprintHash":    fingerprintHash,
		"approvalFingerprintFields":  fields,
		"capabilityId":               req.Capability.ID,
		"previousStatus":             string(req.Capability.Status),
		"newStatus":                  string(req.Requested),
		"riskClass":                  req.Transition.RiskClass,
		"capabilityRisk":             string(req.Capability.Risk),
		"actor":                      req.Actor,
		"actorKind":                  req.ActorKind,
		"reason":                     req.Reason,
		"correlationId":              req.CorrelationID,
		"traceId":                    req.TraceID,
		"publicDecisionAllowed":      false,
		"decisionAuthority":          "non_public_approval_authority_required",
	}
	if requestID > 0 {
		scope["approvalRequestId"] = requestID
	}
	return scope
}

func capabilityStatusApprovalHashFromScope(raw json.RawMessage) string {
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

func capabilityStatusApprovalJobID(shapeHash string) string {
	clean := strings.TrimPrefix(strings.TrimSpace(shapeHash), "sha256:")
	if len(clean) > 24 {
		clean = clean[:24]
	}
	if clean == "" {
		clean = "unknown"
	}
	return "cap-status-" + clean
}

func (s *Server) ensureCapabilityStatusApprovalJob(ctx context.Context, jobID string, req capabilityStatusGovernanceRequest) error {
	now := time.Now().UnixMilli()
	meta := map[string]any{
		"templateId":     "capability_status_approval",
		"capabilityId":   req.Capability.ID,
		"previousStatus": string(req.Capability.Status),
		"newStatus":      string(req.Requested),
		"transitionRisk": req.Transition.RiskClass,
		"actor":          req.Actor,
		"actorKind":      req.ActorKind,
		"reason":         req.Reason,
		"correlationId":  req.CorrelationID,
		"traceId":        req.TraceID,
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
		"Capability status approval",
		"gateway.capability.status.update",
		"gateway",
		firstNonEmptyTrimmed(req.Source, req.ActorKind, "forge_api"),
		"runtime_authority",
		req.Transition.RiskClass,
		"awaiting_approval",
		"pending",
		1,
		string(raw),
	)
	if err != nil {
		return fmt.Errorf("insert capability status approval job: %w", err)
	}
	return nil
}

func (s *Server) updateCapabilityStatusApprovalScope(ctx context.Context, requestID int64, scope map[string]any) error {
	raw, err := json.Marshal(scope)
	if err != nil {
		return err
	}
	_, err = s.st.DB.ExecContext(ctx, `UPDATE approval_requests SET scope_snapshot_json = ? WHERE id = ?`, string(raw), requestID)
	return err
}

func (s *Server) auditCapabilityStatusGovernance(ctx context.Context, capability domain.ToolCapability, requested domain.ToolCapabilityStatus, decision capabilityStatusGovernanceDecision, outcome string) {
	if s.auditSvc == nil {
		return
	}
	var approvalRequestID *int64
	if decision.ApprovalRequestID != nil {
		v := *decision.ApprovalRequestID
		approvalRequestID = &v
	}
	_, _ = s.auditSvc.Record(ctx, audit.CreateRequest{
		CorrelationID:     decision.CorrelationID,
		Category:          "gateway",
		Action:            "tool.capability.status.updated",
		Actor:             firstNonEmptyTrimmed(decision.Actor, "api"),
		SubjectType:       "tool_capability",
		SubjectID:         capability.ID,
		ApprovalRequestID: approvalRequestID,
		RiskClass:         stringFromMap(decision.Fields, "transitionRisk"),
		Outcome:           outcome,
		Summary:           "tool capability status governance " + outcome,
		Payload: map[string]any{
			"capabilityId":      capability.ID,
			"previousStatus":    string(capability.Status),
			"newStatus":         string(requested),
			"requestedStatus":   string(requested),
			"transitionReason":  stringFromMap(decision.Fields, "reason"),
			"actor":             decision.Actor,
			"actorKind":         decision.ActorKind,
			"correlationId":     decision.CorrelationID,
			"traceId":           decision.TraceID,
			"riskClass":         stringFromMap(decision.Fields, "transitionRisk"),
			"capabilityRisk":    string(capability.Risk),
			"approvalId":        decision.ApprovalID,
			"approvalRequestId": decision.ApprovalRequestID,
			"approvalRequired":  decision.RequiresApproval,
			"approved":          decision.Approved,
			"reason":            decision.Reason,
			"errorCode":         decision.ErrorCode,
			"errorMessage":      decision.ErrorMessage,
		},
	})
}

func stringFromMap(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

func (s *Server) handleListLanes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	list, err := s.lanes.List(ctx)
	if err != nil {
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"lanes": list})
}

func (s *Server) handleSaveLane(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var body lanes.Lane
	if err := decodePhase5JSONBody(w, r, &body); err != nil {
		writePhase5DecodeError(w, err)
		return
	}
	saved, err := s.lanes.Save(ctx, body)
	if err != nil {
		writeAPIRequestError(w, http.StatusBadRequest, err)
		return
	}
	_ = s.log.Emit(ctx, "gateway.lane.saved", map[string]any{"lane": saved.ID})
	writeJSON(w, http.StatusOK, map[string]any{"lane": saved})
}

func (s *Server) handleDeleteLane(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")
	if err := s.lanes.Delete(ctx, id); err != nil {
		writeAPIRequestError(w, http.StatusBadRequest, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListPermissionProfiles(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	list, err := s.permissions.List(ctx)
	if err != nil {
		writeAPIRequestError(w, http.StatusInternalServerError, err)
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
	if err := decodePhase5JSONBody(w, r, &body); err != nil {
		writePhase5DecodeError(w, err)
		return
	}
	saved, err := s.permissions.Save(ctx, body)
	if err != nil {
		writeAPIRequestError(w, http.StatusBadRequest, err)
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
		writeAPIRequestError(w, http.StatusBadRequest, err)
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
		writeAPIRequestError(w, http.StatusBadRequest, err)
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
		writeAPIRequestError(w, http.StatusInternalServerError, err)
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
		writeAPIRequestError(w, http.StatusInternalServerError, err)
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
		writeAPIRequestError(w, http.StatusInternalServerError, err)
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
	if err := decodePhase5JSONBody(w, r, &body); err != nil {
		writePhase5DecodeError(w, err)
		return
	}
	body.Kind = strings.TrimSpace(body.Kind)
	b, err := s.backup.CreateBundle(ctx, body)
	if err != nil {
		writeAPIRequestError(w, http.StatusBadRequest, err)
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
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleRestoreBundle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var body backup.RestoreBundleRequest
	if err := decodePhase5JSONBody(w, r, &body); err != nil {
		writePhase5DecodeError(w, err)
		return
	}
	resolvedPath, err := s.backup.ResolveRestorePath(body.FilePath)
	if err != nil {
		writeAPIRequestError(w, http.StatusBadRequest, err)
		return
	}
	body.FilePath = resolvedPath
	meta := requestAuditMetaForBackup(r, "", "", "", "backup.bundle.restore")
	gov, err := s.evaluateBackupRestoreGovernance(ctx, body, meta)
	if err != nil {
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	if gov.HTTPStatus > 0 {
		s.auditBackupRestoreGovernance(ctx, body, gov, "denied")
		http.Error(w, gov.ErrorMessage, gov.HTTPStatus)
		return
	}
	if gov.RequiresApproval && !gov.Approved {
		s.auditBackupRestoreGovernance(ctx, body, gov, "needs_approval")
		writeJSON(w, http.StatusAccepted, map[string]any{
			"governance": gov,
		})
		return
	}
	result, err := s.backup.RestoreBundle(ctx, body)
	if err != nil {
		writeAPIRequestError(w, http.StatusBadRequest, err)
		return
	}
	outcome := "ok"
	if len(result.Errors) > 0 || len(result.Unsupported) > 0 {
		outcome = "partial"
	}
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

type backupRestoreGovernanceDecision struct {
	RequiresApproval  bool           `json:"requiresApproval"`
	Approved          bool           `json:"approved"`
	DryRun            bool           `json:"dryRun"`
	ApprovalID        string         `json:"approvalId,omitempty"`
	ApprovalRequestID *int64         `json:"approvalRequestId,omitempty"`
	Reason            string         `json:"reason"`
	Scope             map[string]any `json:"-"`
	Fields            map[string]any `json:"-"`
	HTTPStatus        int            `json:"-"`
	ErrorCode         string         `json:"errorCode,omitempty"`
	ErrorMessage      string         `json:"errorMessage,omitempty"`
}

func (s *Server) evaluateBackupRestoreGovernance(ctx context.Context, req backup.RestoreBundleRequest, meta requestAuditMeta) (backupRestoreGovernanceDecision, error) {
	shapeHash, fields := backupRestoreApprovalFingerprint(req, 0)
	scope := backupRestoreApprovalScope(req, meta, shapeHash, shapeHash, fields, 0)
	decision := backupRestoreGovernanceDecision{
		RequiresApproval: !req.DryRun,
		Approved:         req.DryRun,
		DryRun:           req.DryRun,
		Reason:           "dry-run restore does not require approval",
		Scope:            scope,
		Fields:           fields,
	}
	if req.DryRun {
		return decision, nil
	}
	decision.Reason = "approval required for non-dry-run backup restore"
	if strings.TrimSpace(req.ApprovalID) == "" {
		if s.approvals == nil {
			decision.HTTPStatus = http.StatusForbidden
			decision.ErrorCode = "BACKUP_RESTORE_APPROVAL_UNAVAILABLE"
			decision.ErrorMessage = "approval service unavailable for backup restore"
			decision.Reason = decision.ErrorMessage
			return decision, nil
		}
		jobID := backupRestoreApprovalJobID(shapeHash)
		if err := s.ensureBackupRestoreApprovalJob(ctx, jobID, req, meta); err != nil {
			return decision, err
		}
		ar, err := s.approvals.OpenRequestForJob(ctx, jobID, approvals.CreateRequestInput{
			JobID:            jobID,
			RequestedAction:  "backup.restore",
			RiskClass:        "critical",
			RequestedAdapter: "backup",
			WriteIntent:      true,
			ScopeSnapshot:    scope,
			RequestSummary:   "Restore FORGE backup bundle into live store",
		})
		if err != nil {
			return decision, err
		}
		if ar != nil {
			v := ar.ID
			decision.ApprovalRequestID = &v
			grantHash, grantFields := backupRestoreApprovalFingerprint(req, ar.ID)
			scope = backupRestoreApprovalScope(req, meta, shapeHash, grantHash, grantFields, ar.ID)
			decision.Scope = scope
			decision.Fields = grantFields
			if err := s.updateBackupRestoreApprovalScope(ctx, ar.ID, scope); err != nil {
				return decision, err
			}
		}
		return decision, nil
	}
	requestID, err := strconv.ParseInt(strings.TrimSpace(req.ApprovalID), 10, 64)
	if err != nil || requestID <= 0 {
		decision.HTTPStatus = http.StatusForbidden
		decision.ErrorCode = "BACKUP_RESTORE_APPROVAL_INVALID"
		decision.ErrorMessage = "approvalId must be a positive integer"
		decision.Reason = decision.ErrorMessage
		return decision, nil
	}
	if s.approvals == nil {
		decision.HTTPStatus = http.StatusForbidden
		decision.ErrorCode = "BACKUP_RESTORE_APPROVAL_UNAVAILABLE"
		decision.ErrorMessage = "approval service unavailable for backup restore"
		decision.Reason = decision.ErrorMessage
		return decision, nil
	}
	ar, err := s.approvals.GetRequest(ctx, requestID)
	if err != nil {
		decision.HTTPStatus = http.StatusForbidden
		decision.ErrorCode = "BACKUP_RESTORE_APPROVAL_NOT_FOUND"
		decision.ErrorMessage = fmt.Sprintf("approval request %d not found", requestID)
		decision.Reason = decision.ErrorMessage
		return decision, nil
	}
	expected := backupRestoreApprovalHashFromScope(ar.ScopeSnapshot)
	actual, _ := backupRestoreApprovalFingerprint(req, requestID)
	if expected == "" {
		decision.HTTPStatus = http.StatusForbidden
		decision.ErrorCode = "BACKUP_RESTORE_APPROVAL_FINGERPRINT_MISSING"
		decision.ErrorMessage = fmt.Sprintf("approval request %d is missing backup restore fingerprint", requestID)
		decision.Reason = decision.ErrorMessage
		return decision, nil
	}
	if actual != expected {
		decision.HTTPStatus = http.StatusForbidden
		decision.ErrorCode = "BACKUP_RESTORE_APPROVAL_FINGERPRINT_MISMATCH"
		decision.ErrorMessage = fmt.Sprintf("approval request %d fingerprint mismatch", requestID)
		decision.Reason = decision.ErrorMessage
		return decision, nil
	}
	if ar.Decision == nil || !strings.EqualFold(strings.TrimSpace(ar.Decision.Decision), "approved") {
		decision.HTTPStatus = http.StatusForbidden
		decision.ErrorCode = "BACKUP_RESTORE_APPROVAL_REQUIRED"
		decision.ErrorMessage = fmt.Sprintf("approval request %d is not approved", requestID)
		decision.Reason = decision.ErrorMessage
		return decision, nil
	}
	decision.ApprovalID = req.ApprovalID
	decision.ApprovalRequestID = &requestID
	decision.Approved = true
	decision.Reason = "approved backup restore"
	return decision, nil
}

func backupRestoreApprovalFingerprint(req backup.RestoreBundleRequest, approvalRequestID int64) (string, map[string]any) {
	fields := map[string]any{
		"version":     "backup.restore.v1",
		"operation":   "restore",
		"filePath":    strings.TrimSpace(req.FilePath),
		"sections":    normalizedBackupRestoreSections(req.Sections),
		"capability":  "backup.restore",
		"riskClass":   "critical",
		"writeIntent": true,
	}
	if approvalRequestID > 0 {
		fields["approvalRequestId"] = approvalRequestID
	}
	body, _ := json.Marshal(fields)
	sum := sha256.Sum256(body)
	return "sha256:" + hex.EncodeToString(sum[:]), fields
}

func normalizedBackupRestoreSections(sections []string) []string {
	out := make([]string, 0, len(sections))
	seen := map[string]bool{}
	for _, section := range sections {
		section = strings.ToLower(strings.TrimSpace(section))
		if section == "" || seen[section] {
			continue
		}
		seen[section] = true
		out = append(out, section)
	}
	sort.Strings(out)
	return out
}

func backupRestoreApprovalScope(req backup.RestoreBundleRequest, meta requestAuditMeta, shapeHash, fingerprintHash string, fields map[string]any, requestID int64) map[string]any {
	scope := map[string]any{
		"approvalFingerprintVersion": "backup.restore.v1",
		"approvalShapeHash":          shapeHash,
		"approvalFingerprintHash":    fingerprintHash,
		"approvalFingerprintFields":  fields,
		"filePath":                   strings.TrimSpace(req.FilePath),
		"sections":                   normalizedBackupRestoreSections(req.Sections),
		"riskClass":                  "critical",
		"writeIntent":                true,
		"correlationId":              meta.CorrelationID,
		"traceId":                    meta.TraceID,
		"workspaceId":                meta.WorkspaceID,
	}
	if requestID > 0 {
		scope["approvalRequestId"] = requestID
	}
	return scope
}

func backupRestoreApprovalHashFromScope(raw json.RawMessage) string {
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

func backupRestoreApprovalJobID(shapeHash string) string {
	clean := strings.TrimPrefix(strings.TrimSpace(shapeHash), "sha256:")
	if len(clean) > 24 {
		clean = clean[:24]
	}
	if clean == "" {
		clean = "unknown"
	}
	return "backup-restore-" + clean
}

func (s *Server) ensureBackupRestoreApprovalJob(ctx context.Context, jobID string, req backup.RestoreBundleRequest, meta requestAuditMeta) error {
	now := time.Now().UnixMilli()
	jobMeta := map[string]any{
		"templateId":    "backup_restore_approval",
		"filePath":      req.FilePath,
		"sections":      normalizedBackupRestoreSections(req.Sections),
		"correlationId": meta.CorrelationID,
		"traceId":       meta.TraceID,
		"workspaceId":   meta.WorkspaceID,
	}
	raw, _ := json.Marshal(jobMeta)
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
		"Backup restore approval",
		"backup.restore",
		"backup",
		"forge_api",
		"runtime_authority",
		"critical",
		"awaiting_approval",
		"pending",
		1,
		string(raw),
	)
	if err != nil {
		return fmt.Errorf("insert backup restore approval job: %w", err)
	}
	return nil
}

func (s *Server) updateBackupRestoreApprovalScope(ctx context.Context, requestID int64, scope map[string]any) error {
	raw, err := json.Marshal(scope)
	if err != nil {
		return err
	}
	_, err = s.st.DB.ExecContext(ctx, `UPDATE approval_requests SET scope_snapshot_json = ? WHERE id = ?`, string(raw), requestID)
	return err
}

func (s *Server) auditBackupRestoreGovernance(ctx context.Context, req backup.RestoreBundleRequest, gov backupRestoreGovernanceDecision, outcome string) {
	_, _ = s.auditSvc.Record(ctx, audit.CreateRequest{
		Category:    "backup",
		Action:      "bundle.restore.governance",
		SubjectType: "bundle",
		SubjectID:   req.FilePath,
		RiskClass:   "critical",
		Outcome:     outcome,
		Summary:     "backup restore governance " + outcome,
		Payload: map[string]any{
			"file":              req.FilePath,
			"sections":          normalizedBackupRestoreSections(req.Sections),
			"dryRun":            req.DryRun,
			"approvalRequired":  gov.RequiresApproval,
			"approvalRequestId": gov.ApprovalRequestID,
			"approved":          gov.Approved,
			"reason":            gov.Reason,
			"errorCode":         gov.ErrorCode,
			"errorMessage":      gov.ErrorMessage,
		},
	})
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
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"checklist": cl})
}

func (s *Server) handleReleaseArtifacts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	list, err := s.release.List(ctx, limit)
	if err != nil {
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"artifacts": list})
}

func (s *Server) handleReleaseRecord(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var body release.ArtifactRequest
	if err := decodePhase5JSONBody(w, r, &body); err != nil {
		writePhase5DecodeError(w, err)
		return
	}
	artifact, err := s.release.RecordArtifact(ctx, body)
	if err != nil {
		writeAPIRequestError(w, http.StatusBadRequest, err)
		return
	}
	_ = s.log.Emit(ctx, "release.artifact.recorded", map[string]any{"id": artifact.ID, "kind": artifact.Kind})
	writeJSON(w, http.StatusOK, map[string]any{"artifact": artifact})
}

func (s *Server) handleFirstRun(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sum, err := s.release.FirstRun(ctx)
	if err != nil {
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"firstRun": sum})
}
