package gateway

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/approvals"
	"forge/projectforge/services/core/internal/audit"
	"forge/projectforge/services/core/internal/lanes"
	"forge/projectforge/services/core/internal/permissions"
)

// Gateway is the single authorized entry point for tool execution in FORGE.
// Every controlled tool action, bounded file operation, git helper, and
// exporter flows through Gateway.Execute so scope, risk, permission, and
// audit can be enforced uniformly. There are no hidden execution paths.
type Gateway struct {
	db           *sql.DB
	perms        *permissions.Service
	lanes        *lanes.Service
	approvals    *approvals.Service
	audit        *audit.Service
	workspace    string
	dataDir      string
	tools        map[string]Tool
	capabilities *ToolCapabilityRegistry
	policy       ToolPolicyEvaluator
	defaultMaxB  int64
}

type Options struct {
	DB                 *sql.DB
	Permissions        *permissions.Service
	Lanes              *lanes.Service
	Approvals          *approvals.Service
	Audit              *audit.Service
	WorkspaceDir       string
	DataDir            string
	CapabilityRegistry *ToolCapabilityRegistry
	AutonomyPolicy     ToolAutonomyAuthorizer
	RiskClassifier     ToolRiskClassifier
}

// Tool is a registered bounded capability that the gateway may invoke after
// permission and lane checks succeed. Tools are deliberately small,
// composable primitives — not adapters. Adapters talk to external models;
// tools perform local controlled actions (read, write, inspect, export).
type Tool interface {
	ID() string
	Domain() string
	Action() string
	RiskClass() string
	ExecutionLevel() string
	Executes() bool
	UsesNetwork() bool
	WriteIntent() bool
	Description() string
	Execute(ctx context.Context, req Request) (Result, error)
}

// Request is what a caller hands to the gateway. Lane must resolve; the
// action and risk come from the lane unless explicitly refined by the caller
// (which still cannot widen what the lane permits).
type Request struct {
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
	Initiator           string         `json:"initiator"`
	DryRun              bool           `json:"dryRun"`
	Metadata            map[string]any `json:"metadata,omitempty"`
}

// Result is the outcome of a gateway invocation. It always carries the
// invocation id and the effective lane / profile used so callers and auditors
// can reconstruct exactly what happened.
type Result struct {
	InvocationID   int64             `json:"invocationId"`
	CorrelationID  string            `json:"correlationId"`
	TraceID        string            `json:"traceId,omitempty"`
	Status         string            `json:"status"`
	PolicyOutcome  string            `json:"policyOutcome"`
	Allowed        bool              `json:"allowed"`
	DeniedReason   string            `json:"deniedReason,omitempty"`
	RiskClass      string            `json:"riskClass"`
	ExecutionLevel string            `json:"executionLevel"`
	Lane           string            `json:"lane"`
	Tool           string            `json:"tool"`
	Domain         string            `json:"domain"`
	Action         string            `json:"action"`
	CapabilityID   string            `json:"capabilityId,omitempty"`
	ProfileID      string            `json:"profileId"`
	Data           map[string]any    `json:"data"`
	Artifacts      []ResultArtifact  `json:"artifacts"`
	Message        string            `json:"message"`
	Metadata       map[string]string `json:"metadata"`
}

type ResultArtifact struct {
	Type    string `json:"type"`
	Path    string `json:"path"`
	Summary string `json:"summary"`
}

const (
	StatusOK          = "ok"
	StatusDenied      = "denied"
	StatusError       = "error"
	StatusNeedsApprov = "needs_approval"
	StatusDryRun      = "dry_run"
	StatusUnsupported = "unsupported"
	StatusDisabled    = "disabled"

	maxCorrelationIDBytes                 = 256
	maxGatewayRequestMetadataBytes        = 512
	maxGatewayInvocationInputJSONBytes    = 256 << 10
	maxApprovalFingerprintStringBytes     = 4 << 10
	maxApprovalFingerprintFieldNameBytes  = 128
	maxApprovalFingerprintCollectionItems = 64

	OutcomeAllow           = "allow"
	OutcomeRequireApproval = "require_approval"
	OutcomeDeny            = "deny"
)

var errCorrelationIDTooLarge = errors.New("correlation id too large")

var errGatewayRequestMetadataTooLarge = errors.New("gateway request metadata too large")

func normalizeCorrelationID(raw string) (string, error) {
	correlationID := strings.TrimSpace(raw)
	if len(correlationID) > maxCorrelationIDBytes {
		return "", fmt.Errorf("%w: %d > %d bytes", errCorrelationIDTooLarge, len(correlationID), maxCorrelationIDBytes)
	}
	return correlationID, nil
}

func normalizeGatewayRequestMetadata(raw, label string, limit int) (string, error) {
	value := strings.TrimSpace(raw)
	if limit <= 0 {
		return "", fmt.Errorf("%s limit must be positive", label)
	}
	if len(value) > limit {
		return "", fmt.Errorf("%w: %s %d > %d bytes", errGatewayRequestMetadataTooLarge, label, len(value), limit)
	}
	return value, nil
}

func (g *Gateway) Execute(ctx context.Context, req Request) (*Result, error) {
	correlationID, err := normalizeCorrelationID(req.CorrelationID)
	if err != nil {
		return nil, err
	}
	req.CorrelationID = correlationID
	if req.CorrelationID == "" {
		req.CorrelationID = newCorrelationID()
	}
	initiator, err := normalizeGatewayRequestMetadata(req.Initiator, "initiator", maxGatewayRequestMetadataBytes)
	if err != nil {
		return nil, err
	}
	req.Initiator = initiator
	if req.Initiator == "" {
		req.Initiator = "operator"
	}
	source, err := normalizeGatewayRequestMetadata(req.Source, "source", maxGatewayRequestMetadataBytes)
	if err != nil {
		return nil, err
	}
	req.Source = source
	if req.Source == "" {
		req.Source = "user"
	}
	traceID, err := normalizeGatewayRequestMetadata(req.TraceID, "traceId", maxGatewayRequestMetadataBytes)
	if err != nil {
		return nil, err
	}
	req.TraceID = traceID
	workspaceID, err := normalizeGatewayRequestMetadata(req.WorkspaceID, "workspaceId", maxGatewayRequestMetadataBytes)
	if err != nil {
		return nil, err
	}
	req.WorkspaceID = workspaceID
	if req.WorkspaceID == "" {
		req.WorkspaceID = workspaceIDFromPath(g.workspace)
	}
	provenanceActor, err := normalizeGatewayRequestMetadata(req.ProvenanceActor, "provenanceActor", maxGatewayRequestMetadataBytes)
	if err != nil {
		return nil, err
	}
	req.ProvenanceActor = provenanceActor
	if req.ProvenanceActor == "" {
		req.ProvenanceActor = req.Initiator
	}
	provenanceActorType, err := normalizeGatewayRequestMetadata(req.ProvenanceActorType, "provenanceActorType", maxGatewayRequestMetadataBytes)
	if err != nil {
		return nil, err
	}
	req.ProvenanceActorType = provenanceActorType
	intentID, err := normalizeGatewayRequestMetadata(req.IntentID, "intentId", maxGatewayRequestMetadataBytes)
	if err != nil {
		return nil, err
	}
	req.IntentID = intentID
	charterID, err := normalizeGatewayRequestMetadata(req.CharterID, "charterId", maxGatewayRequestMetadataBytes)
	if err != nil {
		return nil, err
	}
	req.CharterID = charterID
	budgetID, err := normalizeGatewayRequestMetadata(req.BudgetID, "budgetId", maxGatewayRequestMetadataBytes)
	if err != nil {
		return nil, err
	}
	req.BudgetID = budgetID
	approvalID, err := normalizeGatewayRequestMetadata(req.ApprovalID, "approvalId", maxGatewayRequestMetadataBytes)
	if err != nil {
		return nil, err
	}
	req.ApprovalID = approvalID
	if req.JobID != nil {
		jobID, err := normalizeGatewayRequestMetadata(*req.JobID, "jobId", maxGatewayRequestMetadataBytes)
		if err != nil {
			return nil, err
		}
		req.JobID = &jobID
	}
	if strings.TrimSpace(req.ProvenanceActorType) == "" {
		req.ProvenanceActorType = "service"
	}
	effectiveToolID := req.ToolID
	capability, hasCapability := domain.ToolCapability{}, false
	if g.capabilities != nil {
		capability, hasCapability = g.capabilities.Resolve(req.ToolID)
		if hasCapability {
			if mapped := metadataString(capability.Metadata, "gatewayToolId"); mapped != "" {
				effectiveToolID = mapped
			}
			if strings.TrimSpace(req.Domain) == "" {
				req.Domain = capability.Domain
			}
		}
	}
	if strings.TrimSpace(req.Domain) == "" {
		req.Domain = toolDomainFromID(effectiveToolID)
	}

	paths := resolvePaths(g.workspace, req.Paths)
	if strings.TrimSpace(req.LaneID) == "" {
		req.LaneID = effectiveToolID
	}

	tool, hasAdapter := g.tools[effectiveToolID]
	if hasCapability {
		policyDecision := g.policy.Evaluate(ctx, ToolPolicyInput{
			Request:       req,
			Capability:    capability,
			ResolvedPaths: paths,
			HasAdapter:    hasAdapter,
			UsesNetwork:   hasAdapter && tool.UsesNetwork(),
		})
		riskFromCapability := gatewayRiskClassFromToolRisk(policyDecision.Risk.Risk)
		if policyDecision.Status == StatusDisabled || policyDecision.Status == StatusUnsupported {
			return g.recordCapabilityTerminal(ctx, req, capability, policyDecision, riskFromCapability)
		}
		if !policyDecision.Allowed && !policyDecision.RequiresApproval {
			return g.denied(ctx, req, req.LaneID, effectiveToolID, riskFromCapability, explainToolPolicyDecision(policyDecision))
		}
		if policyDecision.RequiresApproval && !hasAdapter {
			return g.recordCapabilityTerminal(ctx, req, capability, policyDecision, riskFromCapability)
		}
		req.Metadata = mergeGatewayMetadata(req.Metadata, map[string]any{
			"toolCapabilityId": capability.ID,
			"toolRisk":         policyDecision.Risk.Risk,
		})
		if policyDecision.RequiresApproval {
			req.Metadata = mergeGatewayMetadata(req.Metadata, map[string]any{
				"policyApprovalRequired": true,
				"policyApprovalReason":   nonEmpty(policyDecision.Reason, "tool capability policy requires approval"),
			})
		}
	}

	lane, err := g.lanes.Get(ctx, req.LaneID)
	if err != nil {
		return g.denied(ctx, req, "", "", "", fmt.Sprintf("lane %q not found", req.LaneID))
	}
	if !lane.Enabled {
		return g.denied(ctx, req, lane.ID, "", lane.RiskClass, fmt.Sprintf("lane %q disabled", lane.ID))
	}

	tool, ok := g.tools[effectiveToolID]
	if !ok {
		return g.denied(ctx, req, lane.ID, effectiveToolID, lane.RiskClass, fmt.Sprintf("tool %q not registered in gateway", effectiveToolID))
	}

	if lane.WriteIntent != tool.WriteIntent() && tool.WriteIntent() && !lane.WriteIntent {
		return g.denied(ctx, req, lane.ID, tool.ID(), lane.RiskClass, fmt.Sprintf("tool %q requires write intent but lane %q is read-only", tool.ID(), lane.ID))
	}

	for _, p := range paths {
		if !laneCovers(lane, p) {
			return g.denied(ctx, req, lane.ID, tool.ID(), lane.RiskClass, fmt.Sprintf("path %q outside lane %q allowed scope", p, lane.ID))
		}
	}

	capabilityRiskClass := ""
	if hasCapability {
		capabilityRiskClass = gatewayRiskClassFromToolRisk(capability.Risk)
	}
	risk := effectiveRiskClass(req.RiskClass, lane.RiskClass, tool.RiskClass(), capabilityRiskClass)
	level := normalizeExecutionLevel(req.ExecutionLevel)
	if level == "" {
		level = normalizeExecutionLevel(tool.ExecutionLevel())
	}
	if level == "" {
		level = executionLevelFromRisk(risk)
	}
	if levelRank(level) > levelRank(tool.ExecutionLevel()) {
		return g.denied(ctx, req, lane.ID, tool.ID(), risk, fmt.Sprintf("requested execution level %s exceeds tool max %s", level, tool.ExecutionLevel()))
	}

	check := permissions.CheckRequest{
		ToolID:      tool.ID(),
		LaneID:      lane.ID,
		Action:      req.Action,
		Paths:       paths,
		Reads:       !tool.WriteIntent(),
		Writes:      tool.WriteIntent(),
		Executes:    tool.Executes(),
		UsesNetwork: tool.UsesNetwork(),
		RiskClass:   legacyRiskClass(risk),
		JobID:       req.JobID,
	}
	if tool.WriteIntent() {
		check.WriteBytes = writeBytesFromInput(req.Input)
		if check.WriteBytes == 0 {
			check.WriteBytes = 1
		}
	}

	decision, profile, err := g.perms.Check(ctx, check)
	if err != nil {
		return g.denied(ctx, req, lane.ID, tool.ID(), risk, fmt.Sprintf("permission check failed: %v", err))
	}
	profileID := ""
	if profile != nil {
		profileID = profile.ID
	}
	if !decision.Allowed {
		return g.deniedWith(ctx, req, lane.ID, tool.ID(), risk, profileID, decision.Reason)
	}

	intrinsicApprovalReason := gatewayToolIntrinsicApprovalReason(tool, risk, level)
	requiresApproval := decision.RequiresApproval || lane.RequiresApproval || metadataBool(req.Metadata, "policyApprovalRequired") || intrinsicApprovalReason != ""
	policyApprovalReason := metadataString(req.Metadata, "policyApprovalReason")
	if req.DryRun {
		return g.recordDryRun(ctx, req, lane, tool, risk, level, profileID)
	}
	approvalGranted := false
	if strings.TrimSpace(req.ApprovalID) != "" {
		granted, grantErr := g.approvalGrantPresent(ctx, req, lane, tool, risk, level, paths)
		if grantErr != nil {
			return g.denied(ctx, req, lane.ID, tool.ID(), risk, fmt.Sprintf("approval check failed: %v", grantErr))
		}
		if !granted {
			return g.denied(ctx, req, lane.ID, tool.ID(), risk, fmt.Sprintf("approval request %s is not approved", strings.TrimSpace(req.ApprovalID)))
		}
		approvalGranted = true
	} else if g.jobApprovalStatusGranted(ctx, req.JobID) {
		granted, grantErr := g.jobApprovalFingerprintGranted(ctx, req, lane, tool, risk, level, paths)
		if grantErr != nil {
			return g.denied(ctx, req, lane.ID, tool.ID(), risk, fmt.Sprintf("approval check failed: %v", grantErr))
		}
		if !granted {
			return g.denied(ctx, req, lane.ID, tool.ID(), risk, "approved job is missing matching gateway approval fingerprint")
		}
		approvalGranted = true
	}
	if requiresApproval {
		if !approvalGranted {
			granted, grantErr := g.approvalGrantPresent(ctx, req, lane, tool, risk, level, paths)
			if grantErr != nil {
				return g.denied(ctx, req, lane.ID, tool.ID(), risk, fmt.Sprintf("approval check failed: %v", grantErr))
			}
			approvalGranted = granted
		}
		requiresApproval = !approvalGranted
	}
	if requiresApproval {
		needsApprovalReason := strings.TrimSpace(decision.Reason)
		if strings.TrimSpace(policyApprovalReason) != "" {
			if needsApprovalReason != "" {
				needsApprovalReason = policyApprovalReason + "; " + needsApprovalReason
			} else {
				needsApprovalReason = policyApprovalReason
			}
		}
		if strings.TrimSpace(intrinsicApprovalReason) != "" {
			if needsApprovalReason != "" {
				needsApprovalReason = intrinsicApprovalReason + "; " + needsApprovalReason
			} else {
				needsApprovalReason = intrinsicApprovalReason
			}
		}
		return g.recordNeedsApproval(ctx, req, lane, tool, risk, level, profileID, needsApprovalReason)
	}
	if hasCapability && isSelfInitiated(req) {
		if err := g.policy.ConsumeAuthorizedToolRequest(ctx, ToolAutonomyRequest{
			Request:     req,
			Capability:  capability,
			Risk:        g.policy.riskClassifier.Classify(capability, req),
			UsesNetwork: tool.UsesNetwork(),
		}); err != nil {
			return g.denied(ctx, req, lane.ID, tool.ID(), risk, fmt.Sprintf("autonomy budget consume failed: %v", err))
		}
	}
	invID, err := g.openInvocation(ctx, req, lane, tool, risk, level, profileID, "running", nil)
	if err != nil {
		return nil, err
	}

	execCtx := ctx
	cancelExec := func() {}
	if hasCapability && capability.ResourceLimits.MaxDurationMs > 0 {
		execCtx, cancelExec = context.WithTimeout(ctx, time.Duration(capability.ResourceLimits.MaxDurationMs)*time.Millisecond)
	}
	defer cancelExec()

	execResult, execErr := tool.Execute(execCtx, req)
	now := time.Now().UnixMilli()
	if execErr == nil && hasCapability && capability.ResourceLimits.MaxOutputBytes > 0 {
		bytes, _ := json.Marshal(execResult.Data)
		if len(bytes) > capability.ResourceLimits.MaxOutputBytes {
			truncated := map[string]any{
				"truncated":      true,
				"maxOutputBytes": capability.ResourceLimits.MaxOutputBytes,
				"actualBytes":    len(bytes),
			}
			execResult.Data = mergeGatewayMetadata(execResult.Data, map[string]any{
				"_outputLimit": truncated,
			})
			execResult.Data["warnings"] = appendStringWarning(execResult.Data["warnings"], "output exceeded maxOutputBytes and was truncated")
		}
	}
	if execErr != nil && errors.Is(execCtx.Err(), context.DeadlineExceeded) {
		execErr = fmt.Errorf("tool execution timed out after %dms", capability.ResourceLimits.MaxDurationMs)
	}
	if execErr != nil {
		resultBytes, _ := json.Marshal(map[string]any{"error": execErr.Error()})
		_, _ = g.db.ExecContext(ctx, `
UPDATE gateway_invocations
SET status='error', completed_at=?, result_json=?
WHERE id=?`, now, string(resultBytes), invID)
		_, _ = g.audit.Record(ctx, audit.CreateRequest{
			CorrelationID:       req.CorrelationID,
			Category:            "gateway",
			Action:              "tool.error",
			Actor:               req.Initiator,
			SubjectType:         "tool",
			SubjectID:           tool.ID(),
			JobID:               req.JobID,
			GatewayInvocationID: &invID,
			RiskClass:           risk,
			Outcome:             "error",
			Summary:             fmt.Sprintf("tool %q error: %v", tool.ID(), execErr),
			Payload:             gatewayAuditContextPayload(req, map[string]any{"error": execErr.Error()}),
		})
		return &Result{
			InvocationID:   invID,
			CorrelationID:  req.CorrelationID,
			TraceID:        req.TraceID,
			Status:         StatusError,
			PolicyOutcome:  OutcomeAllow,
			Allowed:        true,
			RiskClass:      risk,
			ExecutionLevel: level,
			Lane:           lane.ID,
			Tool:           tool.ID(),
			Domain:         tool.Domain(),
			Action:         tool.Action(),
			CapabilityID:   capabilityIDFromRequest(req),
			ProfileID:      profileID,
			Message:        execErr.Error(),
			Data:           map[string]any{},
		}, nil
	}

	execResult.InvocationID = invID
	execResult.CorrelationID = req.CorrelationID
	execResult.Lane = lane.ID
	execResult.Tool = tool.ID()
	execResult.ProfileID = profileID
	execResult.RiskClass = risk
	execResult.ExecutionLevel = level
	execResult.Status = StatusOK
	execResult.PolicyOutcome = OutcomeAllow
	execResult.Allowed = true
	execResult.Domain = tool.Domain()
	execResult.Action = tool.Action()
	execResult.TraceID = req.TraceID
	execResult.CapabilityID = capabilityIDFromRequest(req)
	if execResult.Data == nil {
		execResult.Data = map[string]any{}
	}

	resultBytes, _ := json.Marshal(execResult.Data)
	artifactsBytes, _ := json.Marshal(execResult.Artifacts)
	_, _ = g.db.ExecContext(ctx, `
UPDATE gateway_invocations
SET status='ok', completed_at=?, result_json=?, artifacts_json=?
WHERE id=?`, now, string(resultBytes), string(artifactsBytes), invID)

	_, _ = g.audit.Record(ctx, audit.CreateRequest{
		CorrelationID:       req.CorrelationID,
		Category:            "gateway",
		Action:              "tool.executed",
		Actor:               req.Initiator,
		SubjectType:         "tool",
		SubjectID:           tool.ID(),
		JobID:               req.JobID,
		GatewayInvocationID: &invID,
		RiskClass:           risk,
		Outcome:             "ok",
		Summary:             fmt.Sprintf("tool %q via lane %q", tool.ID(), lane.ID),
		Payload: gatewayAuditContextPayload(req, map[string]any{
			"paths":         paths,
			"action":        req.Action,
			"artifactCount": len(execResult.Artifacts),
			"artifacts":     summarizeResultArtifacts(execResult.Artifacts),
		}),
	})

	return &execResult, nil
}

// ExecuteAndWait executes a request and, when the gateway opens an approval
// request, blocks until that approval reaches a terminal state. Approved
// requests are re-run with the approval id attached; denied, cancelled, and
// expired requests return a terminal denied result tied to the original
// invocation.
func (g *Gateway) ExecuteAndWait(ctx context.Context, req Request) (*Result, error) {
	first, err := g.Execute(ctx, req)
	if err != nil || first == nil || first.Status != StatusNeedsApprov {
		return first, err
	}
	if g.approvals == nil {
		first.Status = StatusDenied
		first.PolicyOutcome = OutcomeDeny
		first.Allowed = false
		first.DeniedReason = "approval service unavailable"
		first.Message = first.DeniedReason
		return first, nil
	}
	approvalID := approvalRequestIDFromResult(first)
	if approvalID <= 0 {
		first.Status = StatusDenied
		first.PolicyOutcome = OutcomeDeny
		first.Allowed = false
		first.DeniedReason = "approval request was not created"
		first.Message = first.DeniedReason
		return first, nil
	}

	wait := g.approvals.Wait(ctx, approvalID)
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-wait:
	}
	approvalReq, err := g.approvals.GetRequest(ctx, approvalID)
	if err != nil {
		first.Status = StatusDenied
		first.PolicyOutcome = OutcomeDeny
		first.Allowed = false
		first.DeniedReason = fmt.Sprintf("approval request lookup failed: %v", err)
		first.Message = first.DeniedReason
		return first, nil
	}
	decision := ""
	if approvalReq.Decision != nil {
		decision = strings.TrimSpace(strings.ToLower(approvalReq.Decision.Decision))
	}
	if approvalReq.Status == "expired" {
		first.Status = StatusDenied
		first.PolicyOutcome = OutcomeDeny
		first.Allowed = false
		first.DeniedReason = fmt.Sprintf("approval request #%d expired", approvalID)
		first.Message = first.DeniedReason
		return first, nil
	}
	if decision != "approved" {
		if decision == "" {
			decision = nonEmpty(approvalReq.Status, "unresolved")
		}
		first.Status = StatusDenied
		first.PolicyOutcome = OutcomeDeny
		first.Allowed = false
		first.DeniedReason = fmt.Sprintf("approval request #%d %s", approvalID, decision)
		first.Message = first.DeniedReason
		return first, nil
	}

	rerun := req
	rerun.ApprovalID = strconv.FormatInt(approvalID, 10)
	jobID := strings.TrimSpace(approvalReq.JobID)
	if jobID != "" {
		rerun.JobID = &jobID
	}
	rerun.Metadata = mergeGatewayMetadata(rerun.Metadata, map[string]any{
		"approvedRequestId": approvalID,
		"approvalDecision":  decision,
	})
	return g.Execute(ctx, rerun)
}

func approvalRequestIDFromResult(result *Result) int64 {
	if result == nil || result.Data == nil {
		return 0
	}
	switch v := result.Data["approvalRequestId"].(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	case json.Number:
		id, _ := v.Int64()
		return id
	case string:
		id, _ := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		return id
	default:
		return 0
	}
}

func (g *Gateway) openInvocation(ctx context.Context, req Request, lane *lanes.Lane, tool Tool, risk, level, profileID, status string, approvalRequestID *int64) (int64, error) {
	scope := map[string]any{
		"paths":       req.Paths,
		"lane":        lane.ID,
		"workspaceId": req.WorkspaceID,
		"intentId":    req.IntentID,
		"charterId":   req.CharterID,
		"budgetId":    req.BudgetID,
		"source":      req.Source,
		"traceId":     req.TraceID,
	}
	scopeBytes, _ := json.Marshal(scope)
	inputMetadata := mergeGatewayMetadata(req.Metadata, map[string]any{
		"approvalId":          req.ApprovalID,
		"provenanceActor":     req.ProvenanceActor,
		"provenanceActorType": req.ProvenanceActorType,
		"capabilityId":        capabilityIDFromRequest(req),
	})
	inputBytes, err := marshalGatewayInvocationInput(req.Input, inputMetadata)
	if err != nil {
		return 0, err
	}
	now := time.Now().UnixMilli()
	writeIntent := 0
	if tool.WriteIntent() {
		writeIntent = 1
	}
	res, err := g.db.ExecContext(ctx, `
INSERT INTO gateway_invocations(
  correlation_id, created_at, tool_id, lane_id, job_id, packet_id,
  initiator, action, risk_class, write_intent, scope_json, input_json,
  status, permission_profile_id, approval_request_id
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		req.CorrelationID, now, tool.ID(), lane.ID, req.JobID, req.PacketID,
		req.Initiator, nonEmpty(req.Action, tool.Action()), nonEmpty(risk, tool.RiskClass()), writeIntent,
		string(scopeBytes), string(inputBytes), status, profileID, nullInt64(approvalRequestID),
	)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return id, nil
}

func (g *Gateway) denied(ctx context.Context, req Request, laneID, toolID, risk, reason string) (*Result, error) {
	return g.deniedWith(ctx, req, laneID, toolID, risk, "", reason)
}

func (g *Gateway) deniedWith(ctx context.Context, req Request, laneID, toolID, risk, profileID, reason string) (*Result, error) {
	now := time.Now().UnixMilli()
	scopeBytes, _ := json.Marshal(map[string]any{
		"paths":        req.Paths,
		"lane":         laneID,
		"workspaceId":  req.WorkspaceID,
		"traceId":      req.TraceID,
		"source":       req.Source,
		"intentId":     req.IntentID,
		"charterId":    req.CharterID,
		"budgetId":     req.BudgetID,
		"capabilityId": capabilityIDFromRequest(req),
	})
	inputBytes, err := marshalGatewayInvocationInput(req.Input, nil)
	if err != nil {
		return nil, err
	}
	var laneVal any
	if laneID != "" {
		laneVal = laneID
	}
	res, err := g.db.ExecContext(ctx, `
INSERT INTO gateway_invocations(
  correlation_id, created_at, completed_at, tool_id, lane_id, job_id, packet_id,
  initiator, action, risk_class, write_intent, scope_json, input_json,
  status, denied_reason, permission_profile_id
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		req.CorrelationID, now, now, toolID, laneVal, req.JobID, req.PacketID,
		req.Initiator, req.Action, risk, 0,
		string(scopeBytes), string(inputBytes),
		StatusDenied, reason, profileID,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	_, _ = g.audit.Record(ctx, audit.CreateRequest{
		CorrelationID:       req.CorrelationID,
		Category:            "gateway",
		Action:              "tool.denied",
		Actor:               req.Initiator,
		SubjectType:         "tool",
		SubjectID:           toolID,
		JobID:               req.JobID,
		GatewayInvocationID: &id,
		RiskClass:           risk,
		Outcome:             "denied",
		Summary:             reason,
		Payload: gatewayAuditContextPayload(req, map[string]any{
			"paths": req.Paths,
			"lane":  laneID,
		}),
	})
	return &Result{
		InvocationID:   id,
		CorrelationID:  req.CorrelationID,
		TraceID:        req.TraceID,
		Status:         StatusDenied,
		PolicyOutcome:  OutcomeDeny,
		Allowed:        false,
		DeniedReason:   reason,
		RiskClass:      risk,
		ExecutionLevel: executionLevelFromRisk(risk),
		Lane:           laneID,
		Tool:           toolID,
		Domain:         toolDomainFromID(toolID),
		Action:         req.Action,
		CapabilityID:   capabilityIDFromRequest(req),
		ProfileID:      profileID,
		Message:        reason,
		Data:           map[string]any{},
	}, nil
}

// ExecuteTool executes a typed tool request contract and returns a typed tool result.
// This keeps AI-OS callers on explicit domain contracts while still using the
// existing gateway pipeline for enforcement and execution.
func (g *Gateway) ExecuteTool(ctx context.Context, req domain.ToolRequest) (domain.ToolResult, error) {
	startedAt := time.Now().UnixMilli()
	issues := req.Validate()
	if len(issues) > 0 {
		return domain.ToolResult{
			Success:       false,
			ToolID:        req.ToolID,
			RequestID:     req.ID,
			Status:        domain.ToolStatusDenied,
			Error:         &issues[0],
			StartedAt:     startedAt,
			CompletedAt:   time.Now().UnixMilli(),
			CorrelationID: req.CorrelationID,
			TraceID:       req.TraceID,
		}, nil
	}
	payload := map[string]any{}
	for key, value := range req.Payload {
		payload[key] = value
	}
	var paths []string
	if len(req.Scope.SelectedPaths) > 0 {
		paths = append(paths, req.Scope.SelectedPaths...)
	}
	if raw, ok := payload["paths"]; ok {
		if arr, ok := raw.([]string); ok {
			paths = append(paths, arr...)
		}
	}
	gwReq := Request{
		ToolID:              req.ToolID,
		LaneID:              req.Scope.LaneID,
		Domain:              toolDomainFromID(req.ToolID),
		Action:              "invoke",
		RiskClass:           "",
		ExecutionLevel:      "",
		CorrelationID:       req.CorrelationID,
		TraceID:             req.TraceID,
		Source:              string(req.Source),
		WorkspaceID:         req.Scope.WorkspaceID,
		IntentID:            req.IntentID,
		CharterID:           req.CharterID,
		BudgetID:            req.BudgetID,
		ApprovalID:          req.ApprovalID,
		ProvenanceActor:     req.Provenance.Actor,
		ProvenanceActorType: req.Provenance.ActorType,
		Paths:               paths,
		Input:               payload,
		Initiator:           req.Actor.ID,
		DryRun:              req.DryRun,
		Metadata:            req.Metadata,
	}
	result, err := g.Execute(ctx, gwReq)
	completedAt := time.Now().UnixMilli()
	if err != nil {
		execErr := domain.ToolExecutionError{
			Code:    domain.ToolErrExecutionFailed,
			Field:   "gateway",
			Message: err.Error(),
		}
		return domain.ToolResult{
			Success:       false,
			ToolID:        req.ToolID,
			RequestID:     req.ID,
			Status:        domain.ToolStatusFailed,
			Error:         &execErr,
			StartedAt:     startedAt,
			CompletedAt:   completedAt,
			CorrelationID: req.CorrelationID,
			TraceID:       req.TraceID,
		}, nil
	}
	toolArtifacts := make([]domain.ArtifactRef, 0, len(result.Artifacts))
	for i, item := range result.Artifacts {
		toolArtifacts = append(toolArtifacts, domain.ArtifactRef{
			ID:   fmt.Sprintf("%s:%d", req.ID, i),
			Type: strings.TrimSpace(item.Type),
			URI:  strings.TrimSpace(item.Path),
			Scope: domain.ForgeScope{
				WorkspaceID: req.Scope.WorkspaceID,
				LaneID:      req.Scope.LaneID,
			},
			CreatedAt: completedAt,
			Provenance: domain.Provenance{
				Actor:     req.Actor.ID,
				ActorType: req.Provenance.ActorType,
				Source:    req.Provenance.Source,
				TraceID:   req.TraceID,
			},
			Metadata: map[string]any{"summary": item.Summary},
		})
	}
	return domain.ToolResult{
		Success:       result.Status == StatusOK || result.Status == StatusDryRun,
		ToolID:        req.ToolID,
		RequestID:     req.ID,
		Status:        mapGatewayStatusToToolStatus(result.Status),
		Output:        cloneToolOutput(result.Data),
		Error:         toolErrorFromGatewayResult(result),
		Warnings:      warningsFromGatewayResult(result),
		Artifacts:     toolArtifacts,
		AuditID:       "",
		ResourceUsage: domain.ToolResourceUsage{},
		StartedAt:     startedAt,
		CompletedAt:   completedAt,
		CorrelationID: req.CorrelationID,
		TraceID:       req.TraceID,
		Metadata: map[string]any{
			"gatewayInvocationId": result.InvocationID,
			"policyOutcome":       result.PolicyOutcome,
			"capabilityId":        result.CapabilityID,
		},
	}, nil
}

func (g *Gateway) recordDryRun(ctx context.Context, req Request, lane *lanes.Lane, tool Tool, risk, level, profileID string) (*Result, error) {
	id, err := g.openInvocation(ctx, req, lane, tool, risk, level, profileID, StatusDryRun, nil)
	if err != nil {
		return nil, err
	}
	now := time.Now().UnixMilli()
	_, _ = g.db.ExecContext(ctx, `UPDATE gateway_invocations SET completed_at=? WHERE id=?`, now, id)
	_, _ = g.audit.Record(ctx, audit.CreateRequest{
		CorrelationID:       req.CorrelationID,
		Category:            "gateway",
		Action:              "tool.dry_run",
		Actor:               req.Initiator,
		SubjectType:         "tool",
		SubjectID:           tool.ID(),
		JobID:               req.JobID,
		GatewayInvocationID: &id,
		RiskClass:           risk,
		Outcome:             "dry_run",
		Summary:             fmt.Sprintf("dry-run of tool %q", tool.ID()),
		Payload:             gatewayAuditContextPayload(req, map[string]any{"dryRun": true}),
	})
	return &Result{
		InvocationID:   id,
		CorrelationID:  req.CorrelationID,
		TraceID:        req.TraceID,
		Status:         StatusDryRun,
		PolicyOutcome:  OutcomeAllow,
		Allowed:        true,
		RiskClass:      risk,
		ExecutionLevel: level,
		Lane:           lane.ID,
		Tool:           tool.ID(),
		Domain:         tool.Domain(),
		Action:         tool.Action(),
		CapabilityID:   capabilityIDFromRequest(req),
		ProfileID:      profileID,
		Data:           map[string]any{"dryRun": true},
	}, nil
}

func (g *Gateway) recordCapabilityTerminal(ctx context.Context, req Request, capability domain.ToolCapability, policy ToolPolicyDecision, risk string) (*Result, error) {
	now := time.Now().UnixMilli()
	scopeBytes, _ := json.Marshal(map[string]any{
		"paths":        req.Paths,
		"lane":         req.LaneID,
		"workspaceId":  req.WorkspaceID,
		"capabilityId": capability.ID,
		"source":       req.Source,
		"intentId":     req.IntentID,
		"charterId":    req.CharterID,
		"budgetId":     req.BudgetID,
		"traceId":      req.TraceID,
	})
	inputBytes, err := marshalGatewayInvocationInput(req.Input, nil)
	if err != nil {
		return nil, err
	}
	status := policy.Status
	if strings.TrimSpace(status) == "" {
		status = StatusDenied
	}
	deniedReason := explainToolPolicyDecision(policy)
	if strings.TrimSpace(deniedReason) == "" {
		deniedReason = "tool capability policy blocked request"
	}
	policyOutcome := OutcomeDeny
	if status == StatusNeedsApprov {
		policyOutcome = OutcomeRequireApproval
	}
	res, err := g.db.ExecContext(ctx, `
INSERT INTO gateway_invocations(
  correlation_id, created_at, completed_at, tool_id, lane_id, job_id, packet_id,
  initiator, action, risk_class, write_intent, scope_json, input_json,
  status, denied_reason, permission_profile_id
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		req.CorrelationID, now, now, req.ToolID, req.LaneID, req.JobID, req.PacketID,
		req.Initiator, nonEmpty(req.Action, capability.Name), nonEmpty(risk, gatewayRiskClassFromToolRisk(policy.Risk.Risk)), 0,
		string(scopeBytes), string(inputBytes), status, deniedReason, "",
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	auditAction := "tool.denied"
	outcome := "denied"
	if status == StatusNeedsApprov {
		auditAction = "tool.needs_approval"
		outcome = "needs_approval"
	} else if status == StatusUnsupported {
		auditAction = "tool.unsupported"
		outcome = "unsupported"
	} else if status == StatusDisabled {
		auditAction = "tool.disabled"
		outcome = "disabled"
	}
	payload := map[string]any{
		"capabilityId": capability.ID,
		"status":       status,
		"reason":       deniedReason,
		"warnings":     policy.Warnings,
	}
	if policy.Error != nil {
		payload["error"] = policy.Error
	}
	_, _ = g.audit.Record(ctx, audit.CreateRequest{
		CorrelationID:       req.CorrelationID,
		Category:            "gateway",
		Action:              auditAction,
		Actor:               req.Initiator,
		SubjectType:         "tool_capability",
		SubjectID:           capability.ID,
		JobID:               req.JobID,
		GatewayInvocationID: &id,
		RiskClass:           nonEmpty(risk, gatewayRiskClassFromToolRisk(policy.Risk.Risk)),
		Outcome:             outcome,
		Summary:             deniedReason,
		Payload:             gatewayAuditContextPayload(req, payload),
	})
	return &Result{
		InvocationID:   id,
		CorrelationID:  req.CorrelationID,
		TraceID:        req.TraceID,
		Status:         status,
		PolicyOutcome:  policyOutcome,
		Allowed:        false,
		DeniedReason:   deniedReason,
		RiskClass:      nonEmpty(risk, gatewayRiskClassFromToolRisk(policy.Risk.Risk)),
		ExecutionLevel: executionLevelFromRisk(nonEmpty(risk, gatewayRiskClassFromToolRisk(policy.Risk.Risk))),
		Lane:           req.LaneID,
		Tool:           req.ToolID,
		Domain:         capability.Domain,
		Action:         capability.Name,
		CapabilityID:   capability.ID,
		ProfileID:      "",
		Message:        deniedReason,
		Data: map[string]any{
			"toolError": policy.Error,
			"warnings":  policy.Warnings,
		},
	}, nil
}

// ListInvocations returns recent gateway invocations for the UI.
