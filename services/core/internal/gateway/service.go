package gateway

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
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

	OutcomeAllow           = "allow"
	OutcomeRequireApproval = "require_approval"
	OutcomeDeny            = "deny"
)

func New(opts Options) *Gateway {
	registry := opts.CapabilityRegistry
	if registry == nil {
		registry = NewToolCapabilityRegistry()
	}
	g := &Gateway{
		db:           opts.DB,
		perms:        opts.Permissions,
		lanes:        opts.Lanes,
		approvals:    opts.Approvals,
		audit:        opts.Audit,
		workspace:    opts.WorkspaceDir,
		dataDir:      opts.DataDir,
		tools:        map[string]Tool{},
		capabilities: registry,
		policy:       NewToolPolicyEvaluator(opts.WorkspaceDir, opts.RiskClassifier, opts.AutonomyPolicy),
		defaultMaxB:  2 * 1024 * 1024,
	}
	g.registerBuiltinTools()
	return g
}

func (g *Gateway) registerBuiltinTools() {
	register := func(t Tool) { g.tools[t.ID()] = t }
	register(&readFileTool{workspace: g.workspace})
	register(&listDirTool{workspace: g.workspace})
	register(&mkdirTool{workspace: g.workspace})
	register(&renameTool{workspace: g.workspace})
	register(&deleteTool{workspace: g.workspace})
	register(&copyTool{workspace: g.workspace})
	register(&chmodTool{workspace: g.workspace})
	register(&repoInspectTool{workspace: g.workspace})
	register(&gitStatusTool{workspace: g.workspace})
	register(&gitDiffTool{workspace: g.workspace})
	register(&gitBranchTool{workspace: g.workspace})
	register(&gitCommitTool{workspace: g.workspace})
	register(&gitCheckoutTool{workspace: g.workspace})
	register(&gitStashTool{workspace: g.workspace})
	register(&gitApplyPatchTool{workspace: g.workspace})
	register(&processRunTool{workspace: g.workspace})
	register(&processTerminateTool{})
	register(&serviceStatusTool{})
	register(&serviceControlTool{})
	register(&journalTailTool{})
	register(&desktopNotifyTool{})
	register(&desktopOpenTool{workspace: g.workspace})
	register(&networkInterfacesTool{})
	register(&networkConnectivityTool{})
	register(&networkDNSLookupTool{})
	register(&networkFetchTool{})
	register(&timeNowTool{})
	register(&secretGetTool{db: g.db})
	register(&writeFileTool{workspace: g.workspace})
	register(&validateContextTool{workspace: g.workspace, dataDir: g.dataDir})
	g.registerCapabilityBackings()
}

func (g *Gateway) registerCapabilityBackings() {
	if g.capabilities == nil {
		return
	}
	for _, capability := range g.capabilities.List() {
		toolID := metadataString(capability.Metadata, "gatewayToolId")
		if toolID == "" {
			continue
		}
		if _, exists := g.tools[toolID]; exists {
			continue
		}
		g.tools[toolID] = &capabilityBackingTool{
			capability: capability,
			toolID:     toolID,
			workspace:  g.workspace,
			dataDir:    g.dataDir,
			db:         g.db,
		}
	}
}

// RegisterTool adds a non-builtin tool to the gateway registry.
// It is intended for compatibility shims and bounded extension points.
func (g *Gateway) RegisterTool(t Tool) error {
	if t == nil {
		return errors.New("tool is nil")
	}
	id := strings.TrimSpace(t.ID())
	if id == "" {
		return errors.New("tool id required")
	}
	if _, exists := g.tools[id]; exists {
		return fmt.Errorf("tool %q already registered in gateway", id)
	}
	g.tools[id] = t
	return nil
}

// Tools returns a sorted snapshot of the registered tools for UI inspection.
func (g *Gateway) Tools() []ToolInfo {
	out := make([]ToolInfo, 0, len(g.tools))
	for _, t := range g.tools {
		capabilityID := ""
		capabilityStatus := ""
		capabilityRisk := ""
		adapterID := ""
		requiresApprovalByDefault := false
		autonomyEligible := false
		allowedInDryRun := true
		if g.capabilities != nil {
			if capability, ok := g.capabilities.Resolve(t.ID()); ok {
				capabilityID = capability.ID
				capabilityStatus = string(capability.Status)
				capabilityRisk = string(capability.Risk)
				adapterID = capability.AdapterID
				requiresApprovalByDefault = capability.RequiresApprovalByDefault
				autonomyEligible = capability.AutonomyEligible
				allowedInDryRun = capability.AllowedInDryRun
			}
		}
		out = append(out, ToolInfo{
			ID:                        t.ID(),
			Domain:                    t.Domain(),
			Action:                    t.Action(),
			Description:               t.Description(),
			RiskClass:                 t.RiskClass(),
			ExecutionLevel:            t.ExecutionLevel(),
			Executes:                  t.Executes(),
			UsesNetwork:               t.UsesNetwork(),
			WriteIntent:               t.WriteIntent(),
			CapabilityID:              capabilityID,
			CapabilityStatus:          capabilityStatus,
			CapabilityRisk:            capabilityRisk,
			AdapterID:                 adapterID,
			RequiresApprovalByDefault: requiresApprovalByDefault,
			AutonomyEligible:          autonomyEligible,
			AllowedInDryRun:           allowedInDryRun,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

type ToolInfo struct {
	ID                        string `json:"id"`
	Domain                    string `json:"domain"`
	Action                    string `json:"action"`
	Description               string `json:"description"`
	RiskClass                 string `json:"riskClass"`
	ExecutionLevel            string `json:"executionLevel"`
	Executes                  bool   `json:"executes"`
	UsesNetwork               bool   `json:"usesNetwork"`
	WriteIntent               bool   `json:"writeIntent"`
	CapabilityID              string `json:"capabilityId,omitempty"`
	CapabilityStatus          string `json:"capabilityStatus,omitempty"`
	CapabilityRisk            string `json:"capabilityRisk,omitempty"`
	AdapterID                 string `json:"adapterId,omitempty"`
	RequiresApprovalByDefault bool   `json:"requiresApprovalByDefault"`
	AutonomyEligible          bool   `json:"autonomyEligible"`
	AllowedInDryRun           bool   `json:"allowedInDryRun"`
}

func (g *Gateway) Capabilities() []domain.ToolCapability {
	if g.capabilities == nil {
		return []domain.ToolCapability{}
	}
	return g.capabilities.List()
}

func (g *Gateway) Capability(id string) (domain.ToolCapability, bool) {
	if g.capabilities == nil {
		return domain.ToolCapability{}, false
	}
	return g.capabilities.Get(id)
}

func (g *Gateway) UpdateCapabilityStatus(id string, status domain.ToolCapabilityStatus) (domain.ToolCapability, domain.ToolCapability, bool, error) {
	if g.capabilities == nil {
		return domain.ToolCapability{}, domain.ToolCapability{}, false, fmt.Errorf("capability registry unavailable")
	}
	previous, found := g.capabilities.Get(id)
	if !found {
		return domain.ToolCapability{}, domain.ToolCapability{}, false, nil
	}
	updated, ok, err := g.capabilities.UpdateStatus(id, status)
	if err != nil {
		return domain.ToolCapability{}, domain.ToolCapability{}, false, err
	}
	if !ok {
		return domain.ToolCapability{}, domain.ToolCapability{}, false, nil
	}
	return previous, updated, true, nil
}

// Execute runs the full pipeline: lane resolution, permission check,
// invocation record, tool execution, artifact capture, audit write.
func (g *Gateway) Execute(ctx context.Context, req Request) (*Result, error) {
	if strings.TrimSpace(req.CorrelationID) == "" {
		req.CorrelationID = newCorrelationID()
	}
	if strings.TrimSpace(req.Initiator) == "" {
		req.Initiator = "operator"
	}
	if strings.TrimSpace(req.Source) == "" {
		req.Source = "user"
	}
	if strings.TrimSpace(req.WorkspaceID) == "" {
		req.WorkspaceID = workspaceIDFromPath(g.workspace)
	}
	if strings.TrimSpace(req.ProvenanceActor) == "" {
		req.ProvenanceActor = req.Initiator
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

	requiresApproval := decision.RequiresApproval || lane.RequiresApproval || metadataBool(req.Metadata, "policyApprovalRequired")
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
		return g.recordNeedsApproval(ctx, req, lane, tool, risk, level, profileID, needsApprovalReason)
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
	inputPayload := cloneToolOutput(nonNilMap(req.Input))
	inputPayload["_metadata"] = mergeGatewayMetadata(req.Metadata, map[string]any{
		"approvalId":          req.ApprovalID,
		"provenanceActor":     req.ProvenanceActor,
		"provenanceActorType": req.ProvenanceActorType,
		"capabilityId":        capabilityIDFromRequest(req),
	})
	inputBytes, _ := json.Marshal(inputPayload)
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
	inputBytes, _ := json.Marshal(nonNilMap(req.Input))
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

func (g *Gateway) recordNeedsApproval(ctx context.Context, req Request, lane *lanes.Lane, tool Tool, risk, level, profileID, reason string) (*Result, error) {
	var approvalReqID *int64
	effectiveJobID, jobIDPtr, err := g.resolveApprovalJobID(ctx, req, lane, tool, risk, level)
	if err != nil {
		return nil, err
	}
	reqForInv := req
	reqForInv.JobID = jobIDPtr

	if g.approvals != nil && strings.TrimSpace(effectiveJobID) != "" {
		fingerprintHash, fingerprintFields := g.approvalFingerprint(reqForInv, lane, tool, risk, level, resolvePaths(g.workspace, reqForInv.Paths))
		scopeSnapshot := map[string]any{
			"laneId":                     lane.ID,
			"toolId":                     tool.ID(),
			"paths":                      req.Paths,
			"executionLevel":             level,
			"domain":                     tool.Domain(),
			"action":                     tool.Action(),
			"correlationId":              req.CorrelationID,
			"initiator":                  req.Initiator,
			"permissionNotes":            reason,
			"approvalFingerprintVersion": "gateway.v1",
			"approvalShapeHash":          fingerprintHash,
			"approvalFingerprintHash":    fingerprintHash,
			"approvalFingerprintFields":  fingerprintFields,
		}
		ar, err := g.approvals.OpenRequestForJob(ctx, effectiveJobID, approvals.CreateRequestInput{
			JobID:            effectiveJobID,
			RequestedAction:  fmt.Sprintf("%s.%s", tool.Domain(), tool.Action()),
			RiskClass:        risk,
			RequestedAdapter: "gateway",
			WriteIntent:      tool.WriteIntent(),
			ScopeSnapshot:    scopeSnapshot,
			TaskPacketID:     req.PacketID,
			RequestSummary:   fmt.Sprintf("Gateway action %s via lane %s", tool.ID(), lane.ID),
		})
		if err == nil && ar != nil {
			v := ar.ID
			approvalReqID = &v
			grantHash, grantFields := g.approvalFingerprintForRequestID(reqForInv, lane, tool, risk, level, resolvePaths(g.workspace, reqForInv.Paths), ar.ID)
			scopeSnapshot["approvalRequestId"] = ar.ID
			scopeSnapshot["approvalFingerprintHash"] = grantHash
			scopeSnapshot["approvalFingerprintFields"] = grantFields
			if err := g.updateApprovalRequestScopeSnapshot(ctx, ar.ID, scopeSnapshot); err != nil {
				return nil, fmt.Errorf("store approval fingerprint: %w", err)
			}
		}
	}
	id, err := g.openInvocation(ctx, reqForInv, lane, tool, risk, level, profileID, StatusNeedsApprov, approvalReqID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UnixMilli()
	_, _ = g.db.ExecContext(ctx, `UPDATE gateway_invocations SET completed_at=?, denied_reason=? WHERE id=?`, now, reason, id)
	_, _ = g.audit.Record(ctx, audit.CreateRequest{
		CorrelationID:       req.CorrelationID,
		Category:            "gateway",
		Action:              "tool.needs_approval",
		Actor:               req.Initiator,
		SubjectType:         "tool",
		SubjectID:           tool.ID(),
		JobID:               reqForInv.JobID,
		GatewayInvocationID: &id,
		ApprovalRequestID:   approvalReqID,
		RiskClass:           risk,
		Outcome:             "needs_approval",
		Summary:             reason,
		Payload:             gatewayAuditContextPayload(req, map[string]any{"reason": reason}),
	})
	data := map[string]any{
		"jobId": effectiveJobID,
	}
	if approvalReqID != nil {
		data["approvalRequestId"] = *approvalReqID
	}
	return &Result{
		InvocationID:   id,
		CorrelationID:  req.CorrelationID,
		TraceID:        req.TraceID,
		Status:         StatusNeedsApprov,
		PolicyOutcome:  OutcomeRequireApproval,
		Allowed:        false,
		DeniedReason:   reason,
		RiskClass:      risk,
		ExecutionLevel: level,
		Lane:           lane.ID,
		Tool:           tool.ID(),
		Domain:         tool.Domain(),
		Action:         tool.Action(),
		CapabilityID:   capabilityIDFromRequest(req),
		ProfileID:      profileID,
		Message:        reason,
		Data:           data,
	}, nil
}

// resolveApprovalJobID returns the job id used for approval_requests FK and a pointer for gateway_invocations.
// Chat and other callers often omit JobID; we insert a minimal gateway_action job row (not enqueued) so approvals can be recorded.
func (g *Gateway) resolveApprovalJobID(ctx context.Context, req Request, lane *lanes.Lane, tool Tool, risk, level string) (effectiveJobID string, jobIDPtr *string, err error) {
	if req.JobID != nil {
		if jid := strings.TrimSpace(*req.JobID); jid != "" {
			return jid, req.JobID, nil
		}
	}
	jid, err := g.insertChatGatewayApprovalJob(ctx, req, lane, tool, risk, level)
	if err != nil {
		return "", nil, err
	}
	return jid, &jid, nil
}

func (g *Gateway) insertChatGatewayApprovalJob(ctx context.Context, req Request, lane *lanes.Lane, tool Tool, risk, level string) (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("job id entropy: %w", err)
	}
	id := "chat-gw-" + hex.EncodeToString(b)
	now := time.Now().UnixMilli()
	writeIntent := 0
	if tool.WriteIntent() {
		writeIntent = 1
	}
	userRequest := fmt.Sprintf("Chat gateway: %s (correlation %s)", tool.ID(), req.CorrelationID)
	if raw := strings.TrimSpace(metadataString(req.Metadata, "chatUserRequest")); raw != "" {
		userRequest = raw
	}
	approvalInput := cloneToolOutput(nonNilMap(req.Input))
	if tool.ID() == "desktop.open" {
		if _, ok := approvalInput["query"]; !ok || strings.TrimSpace(fmt.Sprintf("%v", approvalInput["query"])) == "" {
			if raw := strings.TrimSpace(metadataString(req.Metadata, "chatUserRequest")); raw != "" {
				approvalInput["query"] = raw
			}
		}
	}
	meta := map[string]any{
		"templateId":    "gateway_action",
		"userRequest":   userRequest,
		"objective":     fmt.Sprintf("Execute %s after operator approval", tool.ID()),
		"executionMode": "governed_tool",
		"createdBy":     nonEmpty(req.Initiator, "chat"),
		"requestPayload": map[string]any{
			"toolId":         tool.ID(),
			"laneId":         lane.ID,
			"action":         nonEmpty(req.Action, tool.Action()),
			"domain":         tool.Domain(),
			"riskClass":      risk,
			"executionLevel": level,
			"correlationId":  req.CorrelationID,
			"paths":          req.Paths,
			"input":          approvalInput,
			"dryRun":         req.DryRun,
		},
	}
	metaJSON, _ := json.Marshal(meta)
	_, err := g.db.ExecContext(ctx, `
INSERT INTO jobs(
  id, created_at, updated_at, queued_at,
  title, requested_action, target_adapter, initiating_source,
  execution_boundary, risk_class, status, approval_status, write_intent,
  metadata_json
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		id,
		now,
		now,
		nil,
		"Chat gateway approval",
		"gateway.action",
		"forge",
		nonEmpty(req.Initiator, "chat"),
		"command_execution",
		risk,
		"awaiting_approval",
		"pending",
		writeIntent,
		string(metaJSON),
	)
	if err != nil {
		return "", fmt.Errorf("insert chat gateway job: %w", err)
	}
	return id, nil
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
	inputBytes, _ := json.Marshal(nonNilMap(req.Input))
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
type InvocationRecord struct {
	ID                  int64           `json:"id"`
	CorrelationID       string          `json:"correlationId"`
	TraceID             string          `json:"traceId,omitempty"`
	CreatedAtMs         int64           `json:"createdAtMs"`
	CompletedAtMs       *int64          `json:"completedAtMs,omitempty"`
	ToolID              string          `json:"toolId"`
	CapabilityID        string          `json:"capabilityId,omitempty"`
	LaneID              *string         `json:"laneId,omitempty"`
	JobID               *string         `json:"jobId,omitempty"`
	PacketID            *int64          `json:"packetId,omitempty"`
	Initiator           string          `json:"initiator"`
	Action              string          `json:"action"`
	Domain              string          `json:"domain"`
	RiskClass           string          `json:"riskClass"`
	ExecutionLevel      string          `json:"executionLevel"`
	PolicyOutcome       string          `json:"policyOutcome"`
	WriteIntent         bool            `json:"writeIntent"`
	Scope               json.RawMessage `json:"scope"`
	Input               json.RawMessage `json:"input"`
	Status              string          `json:"status"`
	DeniedReason        string          `json:"deniedReason"`
	Result              json.RawMessage `json:"result"`
	Artifacts           json.RawMessage `json:"artifacts"`
	PermissionProfileID string          `json:"permissionProfileId"`
	ApprovalRequestID   *int64          `json:"approvalRequestId,omitempty"`
}

func (g *Gateway) ListInvocations(ctx context.Context, limit int, status string) ([]InvocationRecord, error) {
	if limit <= 0 || limit > 500 {
		limit = 150
	}
	where := ""
	args := []any{}
	if strings.TrimSpace(status) != "" {
		where = "WHERE status = ?"
		args = append(args, status)
	}
	args = append(args, limit)
	query := fmt.Sprintf(`
SELECT id, correlation_id, created_at, completed_at, tool_id, lane_id, job_id, packet_id,
       initiator, action, risk_class, write_intent, scope_json, input_json,
       status, denied_reason, result_json, artifacts_json, permission_profile_id, approval_request_id
FROM gateway_invocations %s ORDER BY id DESC LIMIT ?`, where)
	rows, err := g.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanInvocationRows(rows)
}

// ListInvocationsByCorrelation returns invocation rows for one correlation id.
func (g *Gateway) ListInvocationsByCorrelation(ctx context.Context, correlationID string, limit int) ([]InvocationRecord, error) {
	correlationID = strings.TrimSpace(correlationID)
	if correlationID == "" {
		return []InvocationRecord{}, nil
	}
	if limit <= 0 || limit > 1000 {
		limit = 500
	}
	rows, err := g.db.QueryContext(ctx, `
SELECT id, correlation_id, created_at, completed_at, tool_id, lane_id, job_id, packet_id,
       initiator, action, risk_class, write_intent, scope_json, input_json,
       status, denied_reason, result_json, artifacts_json, permission_profile_id, approval_request_id
FROM gateway_invocations
WHERE correlation_id = ?
ORDER BY id ASC LIMIT ?`, correlationID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanInvocationRows(rows)
}

func scanInvocationRows(rows *sql.Rows) ([]InvocationRecord, error) {
	out := []InvocationRecord{}
	for rows.Next() {
		var r InvocationRecord
		var completed sql.NullInt64
		var lane sql.NullString
		var job sql.NullString
		var packet sql.NullInt64
		var approvalReq sql.NullInt64
		var profile sql.NullString
		var writeInt int
		var scope, input, result, artifacts string
		if err := rows.Scan(
			&r.ID, &r.CorrelationID, &r.CreatedAtMs, &completed, &r.ToolID, &lane, &job, &packet,
			&r.Initiator, &r.Action, &r.RiskClass, &writeInt, &scope, &input,
			&r.Status, &r.DeniedReason, &result, &artifacts, &profile, &approvalReq,
		); err != nil {
			return nil, err
		}
		if completed.Valid {
			v := completed.Int64
			r.CompletedAtMs = &v
		}
		if lane.Valid {
			v := lane.String
			r.LaneID = &v
		}
		if job.Valid {
			v := job.String
			r.JobID = &v
		}
		if packet.Valid {
			v := packet.Int64
			r.PacketID = &v
		}
		if profile.Valid {
			r.PermissionProfileID = profile.String
		}
		r.Domain = toolDomainFromID(r.ToolID)
		r.ExecutionLevel = executionLevelFromRisk(r.RiskClass)
		switch r.Status {
		case StatusDenied:
			r.PolicyOutcome = OutcomeDeny
		case StatusNeedsApprov:
			r.PolicyOutcome = OutcomeRequireApproval
		default:
			r.PolicyOutcome = OutcomeAllow
		}
		if approvalReq.Valid {
			v := approvalReq.Int64
			r.ApprovalRequestID = &v
		}
		r.WriteIntent = writeInt == 1
		r.Scope = json.RawMessage(scope)
		r.Input = json.RawMessage(input)
		r.Result = json.RawMessage(result)
		r.Artifacts = json.RawMessage(artifacts)
		scopeObj := map[string]any{}
		if err := json.Unmarshal([]byte(scope), &scopeObj); err == nil {
			r.TraceID = metadataString(scopeObj, "traceId")
			r.CapabilityID = metadataString(scopeObj, "capabilityId")
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// --- Built-in tools ---

type readFileTool struct{ workspace string }

func (t *readFileTool) ID() string             { return "fs.read" }
func (t *readFileTool) Domain() string         { return "filesystem" }
func (t *readFileTool) Action() string         { return "read_file" }
func (t *readFileTool) RiskClass() string      { return "read_only" }
func (t *readFileTool) ExecutionLevel() string { return "L0" }
func (t *readFileTool) Executes() bool         { return false }
func (t *readFileTool) UsesNetwork() bool      { return false }
func (t *readFileTool) WriteIntent() bool      { return false }
func (t *readFileTool) Description() string    { return "Read a file from the workspace" }
func (t *readFileTool) Execute(ctx context.Context, req Request) (Result, error) {
	target, err := firstPath(req.Paths, t.workspace)
	if err != nil {
		return Result{}, err
	}
	info, err := os.Stat(target)
	if err != nil {
		return Result{}, err
	}
	if info.IsDir() {
		return Result{}, fmt.Errorf("target %q is a directory, use fs.list", target)
	}
	if info.Size() > 2*1024*1024 {
		return Result{}, fmt.Errorf("file too large (%d bytes)", info.Size())
	}
	f, err := os.Open(target)
	if err != nil {
		return Result{}, err
	}
	defer f.Close()
	buf := bytes.Buffer{}
	if _, err := io.Copy(&buf, f); err != nil {
		return Result{}, err
	}
	return Result{
		Data: map[string]any{
			"path":  target,
			"size":  info.Size(),
			"bytes": buf.Len(),
			"text":  buf.String(),
		},
		Message: fmt.Sprintf("read %d bytes from %s", buf.Len(), target),
	}, nil
}

type listDirTool struct{ workspace string }

func (t *listDirTool) ID() string             { return "fs.list" }
func (t *listDirTool) Domain() string         { return "filesystem" }
func (t *listDirTool) Action() string         { return "list_directory" }
func (t *listDirTool) RiskClass() string      { return "read_only" }
func (t *listDirTool) ExecutionLevel() string { return "L0" }
func (t *listDirTool) Executes() bool         { return false }
func (t *listDirTool) UsesNetwork() bool      { return false }
func (t *listDirTool) WriteIntent() bool      { return false }
func (t *listDirTool) Description() string    { return "List a directory inside the workspace" }
func (t *listDirTool) Execute(ctx context.Context, req Request) (Result, error) {
	target, err := firstPath(req.Paths, t.workspace)
	if err != nil {
		return Result{}, err
	}
	entries, err := os.ReadDir(target)
	if err != nil {
		return Result{}, err
	}
	type entry struct {
		Name  string `json:"name"`
		IsDir bool   `json:"isDir"`
		Size  int64  `json:"size"`
	}
	out := []entry{}
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, entry{Name: e.Name(), IsDir: e.IsDir(), Size: info.Size()})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return Result{
		Data: map[string]any{
			"path":    target,
			"entries": out,
			"count":   len(out),
		},
		Message: fmt.Sprintf("listed %d entries in %s", len(out), target),
	}, nil
}

type repoInspectTool struct{ workspace string }

func (t *repoInspectTool) ID() string             { return "repo.inspect" }
func (t *repoInspectTool) Domain() string         { return "filesystem" }
func (t *repoInspectTool) Action() string         { return "inspect_repo" }
func (t *repoInspectTool) RiskClass() string      { return "read_only" }
func (t *repoInspectTool) ExecutionLevel() string { return "L0" }
func (t *repoInspectTool) Executes() bool         { return false }
func (t *repoInspectTool) UsesNetwork() bool      { return false }
func (t *repoInspectTool) WriteIntent() bool      { return false }
func (t *repoInspectTool) Description() string    { return "Return a shallow workspace inspection report" }
func (t *repoInspectTool) Execute(ctx context.Context, req Request) (Result, error) {
	target := t.workspace
	if len(req.Paths) > 0 {
		if p, err := firstPath(req.Paths, t.workspace); err == nil {
			target = p
		}
	}
	entries, err := os.ReadDir(target)
	if err != nil {
		return Result{}, err
	}
	files := 0
	dirs := 0
	topFiles := []string{}
	topDirs := []string{}
	for _, e := range entries {
		if e.IsDir() {
			dirs++
			if len(topDirs) < 20 {
				topDirs = append(topDirs, e.Name())
			}
		} else {
			files++
			if len(topFiles) < 20 {
				topFiles = append(topFiles, e.Name())
			}
		}
	}
	return Result{
		Data: map[string]any{
			"path":     target,
			"files":    files,
			"dirs":     dirs,
			"topFiles": topFiles,
			"topDirs":  topDirs,
		},
		Message: fmt.Sprintf("inspected %s: %d files, %d dirs", target, files, dirs),
	}, nil
}

type gitStatusTool struct{ workspace string }

func (t *gitStatusTool) ID() string             { return "git.status" }
func (t *gitStatusTool) Domain() string         { return "git" }
func (t *gitStatusTool) Action() string         { return "status" }
func (t *gitStatusTool) RiskClass() string      { return "read_only" }
func (t *gitStatusTool) ExecutionLevel() string { return "L0" }
func (t *gitStatusTool) Executes() bool         { return true }
func (t *gitStatusTool) UsesNetwork() bool      { return false }
func (t *gitStatusTool) WriteIntent() bool      { return false }
func (t *gitStatusTool) Description() string    { return "Return git status --short for the workspace" }
func (t *gitStatusTool) Execute(ctx context.Context, req Request) (Result, error) {
	dir := t.workspace
	if len(req.Paths) > 0 {
		if p, err := firstPath(req.Paths, t.workspace); err == nil {
			dir = p
		}
	}
	cmd := exec.CommandContext(ctx, "git", "status", "--short", "--branch")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return Result{Data: map[string]any{"path": dir, "available": false, "error": err.Error(), "output": string(out)}}, nil
	}
	return Result{
		Data: map[string]any{
			"path":      dir,
			"available": true,
			"output":    string(out),
		},
		Message: "git status captured",
	}, nil
}

type gitDiffTool struct{ workspace string }

func (t *gitDiffTool) ID() string             { return "git.diff" }
func (t *gitDiffTool) Domain() string         { return "git" }
func (t *gitDiffTool) Action() string         { return "diff" }
func (t *gitDiffTool) RiskClass() string      { return "read_only" }
func (t *gitDiffTool) ExecutionLevel() string { return "L0" }
func (t *gitDiffTool) Executes() bool         { return true }
func (t *gitDiffTool) UsesNetwork() bool      { return false }
func (t *gitDiffTool) WriteIntent() bool      { return false }
func (t *gitDiffTool) Description() string    { return "Return `git diff --stat` for the workspace" }
func (t *gitDiffTool) Execute(ctx context.Context, req Request) (Result, error) {
	dir := t.workspace
	if len(req.Paths) > 0 {
		if p, err := firstPath(req.Paths, t.workspace); err == nil {
			dir = p
		}
	}
	cmd := exec.CommandContext(ctx, "git", "diff", "--stat")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return Result{Data: map[string]any{"path": dir, "available": false, "error": err.Error(), "output": string(out)}}, nil
	}
	return Result{
		Data: map[string]any{
			"path":      dir,
			"available": true,
			"output":    string(out),
		},
		Message: "git diff --stat captured",
	}, nil
}

type writeFileTool struct{ workspace string }

func (t *writeFileTool) ID() string             { return "fs.write" }
func (t *writeFileTool) Domain() string         { return "filesystem" }
func (t *writeFileTool) Action() string         { return "write_file" }
func (t *writeFileTool) RiskClass() string      { return "safe_write" }
func (t *writeFileTool) ExecutionLevel() string { return "L1" }
func (t *writeFileTool) Executes() bool         { return false }
func (t *writeFileTool) UsesNetwork() bool      { return false }
func (t *writeFileTool) WriteIntent() bool      { return true }
func (t *writeFileTool) Description() string    { return "Write content to a file inside approved scope" }
func (t *writeFileTool) Execute(ctx context.Context, req Request) (Result, error) {
	target, err := firstPath(req.Paths, t.workspace)
	if err != nil {
		return Result{}, err
	}
	contents, _ := req.Input["contents"].(string)
	if contents == "" {
		return Result{}, errors.New("fs.write requires input.contents")
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return Result{}, err
	}
	if err := os.WriteFile(target, []byte(contents), 0o644); err != nil {
		return Result{}, err
	}
	return Result{
		Data: map[string]any{
			"path":  target,
			"bytes": len(contents),
		},
		Artifacts: []ResultArtifact{{Type: "writtenFile", Path: target, Summary: fmt.Sprintf("%d bytes", len(contents))}},
		Message:   fmt.Sprintf("wrote %d bytes to %s", len(contents), target),
	}, nil
}

type validateContextTool struct {
	workspace string
	dataDir   string
}

func (t *validateContextTool) ID() string             { return "validate.project_context" }
func (t *validateContextTool) Domain() string         { return "filesystem" }
func (t *validateContextTool) Action() string         { return "validate_context" }
func (t *validateContextTool) RiskClass() string      { return "read_only" }
func (t *validateContextTool) ExecutionLevel() string { return "L0" }
func (t *validateContextTool) Executes() bool         { return false }
func (t *validateContextTool) UsesNetwork() bool      { return false }
func (t *validateContextTool) WriteIntent() bool      { return false }
func (t *validateContextTool) Description() string {
	return "Check that project context artifacts exist and are non-empty"
}
func (t *validateContextTool) Execute(ctx context.Context, req Request) (Result, error) {
	want := []string{"AGENTS.md", "CLAUDE.md", "docs/FORGE_PROJECT_BRIEFING.md"}
	report := map[string]any{}
	missing := []string{}
	for _, rel := range want {
		p := filepath.Join(t.workspace, rel)
		info, err := os.Stat(p)
		switch {
		case err != nil:
			report[rel] = map[string]any{"exists": false}
			missing = append(missing, rel)
		case info.Size() == 0:
			report[rel] = map[string]any{"exists": true, "sizeBytes": 0, "empty": true}
			missing = append(missing, rel)
		default:
			report[rel] = map[string]any{"exists": true, "sizeBytes": info.Size()}
		}
	}
	status := "ok"
	if len(missing) > 0 {
		status = "incomplete"
	}
	return Result{
		Data: map[string]any{
			"status":  status,
			"report":  report,
			"missing": missing,
		},
		Message: fmt.Sprintf("project context validation: %s", status),
	}, nil
}

type mkdirTool struct{ workspace string }

func (t *mkdirTool) ID() string             { return "fs.mkdir" }
func (t *mkdirTool) Domain() string         { return "filesystem" }
func (t *mkdirTool) Action() string         { return "make_directory" }
func (t *mkdirTool) RiskClass() string      { return "low_write" }
func (t *mkdirTool) ExecutionLevel() string { return "L1" }
func (t *mkdirTool) Executes() bool         { return false }
func (t *mkdirTool) UsesNetwork() bool      { return false }
func (t *mkdirTool) WriteIntent() bool      { return true }
func (t *mkdirTool) Description() string    { return "Create a directory" }
func (t *mkdirTool) Execute(ctx context.Context, req Request) (Result, error) {
	target, err := firstPath(req.Paths, t.workspace)
	if err != nil {
		return Result{}, err
	}
	if err := os.MkdirAll(target, 0o755); err != nil {
		return Result{}, err
	}
	return Result{Data: map[string]any{"path": target}, Message: "directory created"}, nil
}

type renameTool struct{ workspace string }

func (t *renameTool) ID() string             { return "fs.rename" }
func (t *renameTool) Domain() string         { return "filesystem" }
func (t *renameTool) Action() string         { return "rename_move" }
func (t *renameTool) RiskClass() string      { return "safe_write" }
func (t *renameTool) ExecutionLevel() string { return "L1" }
func (t *renameTool) Executes() bool         { return false }
func (t *renameTool) UsesNetwork() bool      { return false }
func (t *renameTool) WriteIntent() bool      { return true }
func (t *renameTool) Description() string    { return "Rename or move a file/directory" }
func (t *renameTool) Execute(ctx context.Context, req Request) (Result, error) {
	if len(req.Paths) < 2 {
		return Result{}, errors.New("fs.rename requires source and destination paths")
	}
	src, err := firstPath(req.Paths[:1], t.workspace)
	if err != nil {
		return Result{}, err
	}
	dst, err := firstPath(req.Paths[1:], t.workspace)
	if err != nil {
		return Result{}, err
	}
	if err := os.Rename(src, dst); err != nil {
		return Result{}, err
	}
	return Result{Data: map[string]any{"from": src, "to": dst}, Message: "rename completed"}, nil
}

type deleteTool struct{ workspace string }

func (t *deleteTool) ID() string             { return "fs.delete" }
func (t *deleteTool) Domain() string         { return "filesystem" }
func (t *deleteTool) Action() string         { return "delete_path" }
func (t *deleteTool) RiskClass() string      { return "dangerous" }
func (t *deleteTool) ExecutionLevel() string { return "L4" }
func (t *deleteTool) Executes() bool         { return false }
func (t *deleteTool) UsesNetwork() bool      { return false }
func (t *deleteTool) WriteIntent() bool      { return true }
func (t *deleteTool) Description() string    { return "Delete a file or directory recursively" }
func (t *deleteTool) Execute(ctx context.Context, req Request) (Result, error) {
	target, err := firstPath(req.Paths, t.workspace)
	if err != nil {
		return Result{}, err
	}
	if err := os.RemoveAll(target); err != nil {
		return Result{}, err
	}
	return Result{Data: map[string]any{"path": target}, Message: "path deleted"}, nil
}

type copyTool struct{ workspace string }

func (t *copyTool) ID() string             { return "fs.copy" }
func (t *copyTool) Domain() string         { return "filesystem" }
func (t *copyTool) Action() string         { return "copy_file" }
func (t *copyTool) RiskClass() string      { return "safe_write" }
func (t *copyTool) ExecutionLevel() string { return "L1" }
func (t *copyTool) Executes() bool         { return false }
func (t *copyTool) UsesNetwork() bool      { return false }
func (t *copyTool) WriteIntent() bool      { return true }
func (t *copyTool) Description() string    { return "Copy a file from source to destination" }
func (t *copyTool) Execute(ctx context.Context, req Request) (Result, error) {
	if len(req.Paths) < 2 {
		return Result{}, errors.New("fs.copy requires source and destination paths")
	}
	src, err := firstPath(req.Paths[:1], t.workspace)
	if err != nil {
		return Result{}, err
	}
	dst, err := firstPath(req.Paths[1:], t.workspace)
	if err != nil {
		return Result{}, err
	}
	in, err := os.Open(src)
	if err != nil {
		return Result{}, err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return Result{}, err
	}
	out, err := os.Create(dst)
	if err != nil {
		return Result{}, err
	}
	defer out.Close()
	written, err := io.Copy(out, in)
	if err != nil {
		return Result{}, err
	}
	return Result{Data: map[string]any{"from": src, "to": dst, "bytes": written}, Message: "copy completed"}, nil
}

type chmodTool struct{ workspace string }

func (t *chmodTool) ID() string             { return "fs.chmod" }
func (t *chmodTool) Domain() string         { return "filesystem" }
func (t *chmodTool) Action() string         { return "chmod" }
func (t *chmodTool) RiskClass() string      { return "privileged" }
func (t *chmodTool) ExecutionLevel() string { return "L3" }
func (t *chmodTool) Executes() bool         { return false }
func (t *chmodTool) UsesNetwork() bool      { return false }
func (t *chmodTool) WriteIntent() bool      { return true }
func (t *chmodTool) Description() string    { return "Change file mode for a path" }
func (t *chmodTool) Execute(ctx context.Context, req Request) (Result, error) {
	target, err := firstPath(req.Paths, t.workspace)
	if err != nil {
		return Result{}, err
	}
	modeRaw, _ := req.Input["mode"].(string)
	modeRaw = strings.TrimSpace(modeRaw)
	if modeRaw == "" {
		modeRaw = "0644"
	}
	v, err := strconv.ParseUint(modeRaw, 8, 32)
	if err != nil {
		return Result{}, fmt.Errorf("invalid mode %q", modeRaw)
	}
	if err := os.Chmod(target, os.FileMode(v)); err != nil {
		return Result{}, err
	}
	return Result{Data: map[string]any{"path": target, "mode": modeRaw}, Message: "chmod applied"}, nil
}

type processRunTool struct{ workspace string }

func (t *processRunTool) ID() string             { return "proc.run" }
func (t *processRunTool) Domain() string         { return "process" }
func (t *processRunTool) Action() string         { return "run_command" }
func (t *processRunTool) RiskClass() string      { return "scoped_execute" }
func (t *processRunTool) ExecutionLevel() string { return "L2" }
func (t *processRunTool) Executes() bool         { return true }
func (t *processRunTool) UsesNetwork() bool      { return false }
func (t *processRunTool) WriteIntent() bool      { return false }
func (t *processRunTool) Description() string {
	return "Run a command with timeout and captured stdout/stderr"
}
func (t *processRunTool) Execute(ctx context.Context, req Request) (Result, error) {
	command, _ := req.Input["command"].(string)
	command = strings.TrimSpace(command)
	if command == "" {
		return Result{}, errors.New("proc.run requires input.command")
	}
	timeoutMs := 30_000
	if v, ok := req.Input["timeoutMs"].(float64); ok && v > 0 {
		timeoutMs = int(v)
	}
	runCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()
	cmd := exec.CommandContext(runCtx, "bash", "-lc", command)
	if len(req.Paths) > 0 {
		cwd, err := firstPath(req.Paths, t.workspace)
		if err == nil {
			cmd.Dir = cwd
		}
	} else {
		cmd.Dir = t.workspace
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}
	return Result{
		Data: map[string]any{
			"command":     command,
			"cwd":         cmd.Dir,
			"timeoutMs":   timeoutMs,
			"exitCode":    exitCode,
			"ok":          err == nil,
			"stdout":      stdout.String(),
			"stderr":      stderr.String(),
			"timedOut":    errors.Is(runCtx.Err(), context.DeadlineExceeded),
			"startedAtMs": time.Now().Add(-time.Duration(timeoutMs) * time.Millisecond).UnixMilli(),
			"endedAtMs":   time.Now().UnixMilli(),
		},
		Message: "process execution completed",
	}, nil
}

type processTerminateTool struct{}

func (t *processTerminateTool) ID() string             { return "proc.terminate" }
func (t *processTerminateTool) Domain() string         { return "process" }
func (t *processTerminateTool) Action() string         { return "terminate_process" }
func (t *processTerminateTool) RiskClass() string      { return "privileged" }
func (t *processTerminateTool) ExecutionLevel() string { return "L3" }
func (t *processTerminateTool) Executes() bool         { return true }
func (t *processTerminateTool) UsesNetwork() bool      { return false }
func (t *processTerminateTool) WriteIntent() bool      { return true }
func (t *processTerminateTool) Description() string    { return "Terminate a process by PID (SIGTERM)" }
func (t *processTerminateTool) Execute(ctx context.Context, req Request) (Result, error) {
	pid := int(readFloat(req.Input, "pid", 0))
	if pid <= 0 {
		return Result{}, errors.New("proc.terminate requires input.pid")
	}
	cmd := exec.CommandContext(ctx, "kill", strconv.Itoa(pid))
	out, err := cmd.CombinedOutput()
	return Result{Data: map[string]any{"pid": pid, "output": string(out), "ok": err == nil}, Message: "terminate attempted"}, nil
}

type gitBranchTool struct{ workspace string }

func (t *gitBranchTool) ID() string             { return "git.branch" }
func (t *gitBranchTool) Domain() string         { return "git" }
func (t *gitBranchTool) Action() string         { return "branch" }
func (t *gitBranchTool) RiskClass() string      { return "read_only" }
func (t *gitBranchTool) ExecutionLevel() string { return "L0" }
func (t *gitBranchTool) Executes() bool         { return true }
func (t *gitBranchTool) UsesNetwork() bool      { return false }
func (t *gitBranchTool) WriteIntent() bool      { return false }
func (t *gitBranchTool) Description() string    { return "List git branches" }
func (t *gitBranchTool) Execute(ctx context.Context, req Request) (Result, error) {
	dir := t.workspace
	if len(req.Paths) > 0 {
		if p, err := firstPath(req.Paths, t.workspace); err == nil {
			dir = p
		}
	}
	out, err := runCmd(ctx, dir, "git", "branch", "--all", "--verbose")
	return Result{Data: map[string]any{"path": dir, "output": out, "ok": err == nil}, Message: "git branch captured"}, nil
}

type gitCommitTool struct{ workspace string }

func (t *gitCommitTool) ID() string             { return "git.commit" }
func (t *gitCommitTool) Domain() string         { return "git" }
func (t *gitCommitTool) Action() string         { return "commit" }
func (t *gitCommitTool) RiskClass() string      { return "dangerous" }
func (t *gitCommitTool) ExecutionLevel() string { return "L4" }
func (t *gitCommitTool) Executes() bool         { return true }
func (t *gitCommitTool) UsesNetwork() bool      { return false }
func (t *gitCommitTool) WriteIntent() bool      { return true }
func (t *gitCommitTool) Description() string    { return "Create git commit with provided message" }
func (t *gitCommitTool) Execute(ctx context.Context, req Request) (Result, error) {
	dir := t.workspace
	if len(req.Paths) > 0 {
		if p, err := firstPath(req.Paths, t.workspace); err == nil {
			dir = p
		}
	}
	message, _ := req.Input["message"].(string)
	message = strings.TrimSpace(message)
	if message == "" {
		message = "FORGE gateway commit"
	}
	out, err := runCmd(ctx, dir, "git", "commit", "-m", message)
	return Result{Data: map[string]any{"path": dir, "message": message, "output": out, "ok": err == nil}, Message: "git commit executed"}, nil
}

type gitCheckoutTool struct{ workspace string }

func (t *gitCheckoutTool) ID() string             { return "git.checkout" }
func (t *gitCheckoutTool) Domain() string         { return "git" }
func (t *gitCheckoutTool) Action() string         { return "checkout" }
func (t *gitCheckoutTool) RiskClass() string      { return "dangerous" }
func (t *gitCheckoutTool) ExecutionLevel() string { return "L4" }
func (t *gitCheckoutTool) Executes() bool         { return true }
func (t *gitCheckoutTool) UsesNetwork() bool      { return false }
func (t *gitCheckoutTool) WriteIntent() bool      { return true }
func (t *gitCheckoutTool) Description() string    { return "Git checkout branch/ref" }
func (t *gitCheckoutTool) Execute(ctx context.Context, req Request) (Result, error) {
	dir := t.workspace
	if len(req.Paths) > 0 {
		if p, err := firstPath(req.Paths, t.workspace); err == nil {
			dir = p
		}
	}
	ref, _ := req.Input["ref"].(string)
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return Result{}, errors.New("git.checkout requires input.ref")
	}
	out, err := runCmd(ctx, dir, "git", "checkout", ref)
	return Result{Data: map[string]any{"path": dir, "ref": ref, "output": out, "ok": err == nil}, Message: "git checkout executed"}, nil
}

type gitStashTool struct{ workspace string }

func (t *gitStashTool) ID() string             { return "git.stash" }
func (t *gitStashTool) Domain() string         { return "git" }
func (t *gitStashTool) Action() string         { return "stash" }
func (t *gitStashTool) RiskClass() string      { return "safe_write" }
func (t *gitStashTool) ExecutionLevel() string { return "L1" }
func (t *gitStashTool) Executes() bool         { return true }
func (t *gitStashTool) UsesNetwork() bool      { return false }
func (t *gitStashTool) WriteIntent() bool      { return true }
func (t *gitStashTool) Description() string    { return "Git stash push/pop/list" }
func (t *gitStashTool) Execute(ctx context.Context, req Request) (Result, error) {
	dir := t.workspace
	if len(req.Paths) > 0 {
		if p, err := firstPath(req.Paths, t.workspace); err == nil {
			dir = p
		}
	}
	mode, _ := req.Input["mode"].(string)
	mode = strings.TrimSpace(strings.ToLower(mode))
	if mode == "" {
		mode = "push"
	}
	args := []string{"stash", mode}
	if mode == "push" {
		if msg, _ := req.Input["message"].(string); strings.TrimSpace(msg) != "" {
			args = append(args, "-m", strings.TrimSpace(msg))
		}
	}
	out, err := runCmd(ctx, dir, append([]string{"git"}, args...)...)
	return Result{Data: map[string]any{"path": dir, "mode": mode, "output": out, "ok": err == nil}, Message: "git stash executed"}, nil
}

type gitApplyPatchTool struct{ workspace string }

func (t *gitApplyPatchTool) ID() string             { return "git.apply_patch" }
func (t *gitApplyPatchTool) Domain() string         { return "git" }
func (t *gitApplyPatchTool) Action() string         { return "apply_patch" }
func (t *gitApplyPatchTool) RiskClass() string      { return "dangerous" }
func (t *gitApplyPatchTool) ExecutionLevel() string { return "L4" }
func (t *gitApplyPatchTool) Executes() bool         { return true }
func (t *gitApplyPatchTool) UsesNetwork() bool      { return false }
func (t *gitApplyPatchTool) WriteIntent() bool      { return true }
func (t *gitApplyPatchTool) Description() string    { return "Apply git patch from input.patch" }
func (t *gitApplyPatchTool) Execute(ctx context.Context, req Request) (Result, error) {
	dir := t.workspace
	if len(req.Paths) > 0 {
		if p, err := firstPath(req.Paths, t.workspace); err == nil {
			dir = p
		}
	}
	patch, _ := req.Input["patch"].(string)
	patch = strings.TrimSpace(patch)
	if patch == "" {
		return Result{}, errors.New("git.apply_patch requires input.patch")
	}
	cmd := exec.CommandContext(ctx, "git", "apply", "-")
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(patch)
	out, err := cmd.CombinedOutput()
	return Result{Data: map[string]any{"path": dir, "output": string(out), "ok": err == nil}, Message: "git apply executed"}, nil
}

type serviceStatusTool struct{}

func (t *serviceStatusTool) ID() string             { return "system.service_status" }
func (t *serviceStatusTool) Domain() string         { return "system" }
func (t *serviceStatusTool) Action() string         { return "service_status" }
func (t *serviceStatusTool) RiskClass() string      { return "read_only" }
func (t *serviceStatusTool) ExecutionLevel() string { return "L0" }
func (t *serviceStatusTool) Executes() bool         { return true }
func (t *serviceStatusTool) UsesNetwork() bool      { return false }
func (t *serviceStatusTool) WriteIntent() bool      { return false }
func (t *serviceStatusTool) Description() string    { return "Inspect system service status (systemctl)" }
func (t *serviceStatusTool) Execute(ctx context.Context, req Request) (Result, error) {
	name, _ := req.Input["service"].(string)
	name = strings.TrimSpace(name)
	if name == "" {
		return Result{}, errors.New("system.service_status requires input.service")
	}
	out, err := runCmd(ctx, "", "systemctl", "status", "--no-pager", name)
	return Result{Data: map[string]any{"service": name, "output": out, "ok": err == nil}, Message: "service status checked"}, nil
}

type serviceControlTool struct{}

func (t *serviceControlTool) ID() string             { return "system.service_control" }
func (t *serviceControlTool) Domain() string         { return "system" }
func (t *serviceControlTool) Action() string         { return "service_control" }
func (t *serviceControlTool) RiskClass() string      { return "privileged" }
func (t *serviceControlTool) ExecutionLevel() string { return "L3" }
func (t *serviceControlTool) Executes() bool         { return true }
func (t *serviceControlTool) UsesNetwork() bool      { return false }
func (t *serviceControlTool) WriteIntent() bool      { return true }
func (t *serviceControlTool) Description() string    { return "Start/stop/restart a service (systemctl)" }
func (t *serviceControlTool) Execute(ctx context.Context, req Request) (Result, error) {
	name, _ := req.Input["service"].(string)
	action, _ := req.Input["control"].(string)
	name = strings.TrimSpace(name)
	action = strings.TrimSpace(strings.ToLower(action))
	if name == "" || (action != "start" && action != "stop" && action != "restart") {
		return Result{}, errors.New("system.service_control requires input.service and control=start|stop|restart")
	}
	out, err := runCmd(ctx, "", "systemctl", action, name)
	return Result{Data: map[string]any{"service": name, "control": action, "output": out, "ok": err == nil}, Message: "service control executed"}, nil
}

type journalTailTool struct{}

func (t *journalTailTool) ID() string             { return "system.logs" }
func (t *journalTailTool) Domain() string         { return "system" }
func (t *journalTailTool) Action() string         { return "logs" }
func (t *journalTailTool) RiskClass() string      { return "read_only" }
func (t *journalTailTool) ExecutionLevel() string { return "L0" }
func (t *journalTailTool) Executes() bool         { return true }
func (t *journalTailTool) UsesNetwork() bool      { return false }
func (t *journalTailTool) WriteIntent() bool      { return false }
func (t *journalTailTool) Description() string    { return "Tail journal/service logs" }
func (t *journalTailTool) Execute(ctx context.Context, req Request) (Result, error) {
	service, _ := req.Input["service"].(string)
	lines := int(readFloat(req.Input, "lines", 100))
	if lines <= 0 {
		lines = 100
	}
	args := []string{"-n", strconv.Itoa(lines), "--no-pager"}
	if strings.TrimSpace(service) != "" {
		args = append(args, "-u", strings.TrimSpace(service))
	}
	out, err := runCmd(ctx, "", append([]string{"journalctl"}, args...)...)
	return Result{Data: map[string]any{"service": strings.TrimSpace(service), "lines": lines, "output": out, "ok": err == nil}, Message: "logs fetched"}, nil
}

type desktopNotifyTool struct{}

func (t *desktopNotifyTool) ID() string             { return "desktop.notify" }
func (t *desktopNotifyTool) Domain() string         { return "desktop" }
func (t *desktopNotifyTool) Action() string         { return "notify" }
func (t *desktopNotifyTool) RiskClass() string      { return "safe_write" }
func (t *desktopNotifyTool) ExecutionLevel() string { return "L1" }
func (t *desktopNotifyTool) Executes() bool         { return true }
func (t *desktopNotifyTool) UsesNetwork() bool      { return false }
func (t *desktopNotifyTool) WriteIntent() bool      { return true }
func (t *desktopNotifyTool) Description() string {
	return "Send local desktop notification (notify-send)"
}
func (t *desktopNotifyTool) Execute(ctx context.Context, req Request) (Result, error) {
	title, _ := req.Input["title"].(string)
	body, _ := req.Input["body"].(string)
	if strings.TrimSpace(title) == "" {
		title = "FORGE"
	}
	out, err := runCmd(ctx, "", "notify-send", strings.TrimSpace(title), strings.TrimSpace(body))
	return Result{Data: map[string]any{"title": title, "body": body, "output": out, "ok": err == nil}, Message: "notification attempted"}, nil
}

type desktopOpenTool struct{ workspace string }

func (t *desktopOpenTool) ID() string             { return "desktop.open" }
func (t *desktopOpenTool) Domain() string         { return "desktop" }
func (t *desktopOpenTool) Action() string         { return "open_path" }
func (t *desktopOpenTool) RiskClass() string      { return "safe_write" }
func (t *desktopOpenTool) ExecutionLevel() string { return "L1" }
func (t *desktopOpenTool) Executes() bool         { return true }
func (t *desktopOpenTool) UsesNetwork() bool      { return false }
func (t *desktopOpenTool) WriteIntent() bool      { return true }
func (t *desktopOpenTool) Description() string {
	return "Open file/folder/URL or launch desktop app using desktop session"
}
func (t *desktopOpenTool) Execute(ctx context.Context, req Request) (Result, error) {
	if len(req.Paths) > 0 {
		target, err := firstPath(req.Paths, t.workspace)
		if err != nil {
			return Result{}, err
		}
		out, err := runCmd(ctx, "", "xdg-open", target)
		return Result{
			Data:    map[string]any{"mode": "path", "path": target, "output": out, "ok": err == nil},
			Message: "open attempted",
		}, nil
	}

	candidate := desktopInputCandidate(req.Input)
	if candidate == "" {
		return Result{}, errors.New("desktop.open requires either paths[] or input.path|input.url|input.application")
	}
	appHint, inlineCommand := desktopSplitAppAndCommand(candidate)
	if len(inlineCommand) == 0 {
		inlineCommand = desktopInlineCommandFromInput(req.Input)
	}

	if desktopLooksLikeURL(appHint) {
		out, err := runCmd(ctx, "", "xdg-open", appHint)
		return Result{
			Data:    map[string]any{"mode": "url", "target": appHint, "output": out, "ok": err == nil},
			Message: "open attempted",
		}, nil
	}

	if desktopLooksLikePath(appHint) {
		target, err := firstPath([]string{appHint}, t.workspace)
		if err != nil {
			return Result{}, err
		}
		if !pathContains(t.workspace, target) {
			return Result{}, errors.New("desktop.open input.path must resolve inside workspace")
		}
		out, err := runCmd(ctx, "", "xdg-open", target)
		return Result{
			Data:    map[string]any{"mode": "path", "path": target, "output": out, "ok": err == nil},
			Message: "open attempted",
		}, nil
	}

	command, args, appName, err := desktopResolveAppLaunch(appHint)
	if err != nil {
		return Result{}, err
	}
	if len(inlineCommand) > 0 {
		terminalArgs, ok := desktopTerminalLaunchArgs(command, inlineCommand)
		if !ok {
			return Result{}, fmt.Errorf("application %q does not support inline command execution", appName)
		}
		args = append(args, terminalArgs...)
	}
	parts := append([]string{command}, args...)
	pid, runErr := runDetachedCmd("", parts...)
	out := ""
	return Result{
		Data: map[string]any{
			"mode":        "application",
			"application": appName,
			"command":     command,
			"args":        args,
			"inlineCmd":   inlineCommand,
			"pid":         pid,
			"output":      out,
			"ok":          runErr == nil,
		},
		Message: "application launch attempted",
	}, nil
}

func desktopInputCandidate(input map[string]any) string {
	if input == nil {
		return ""
	}
	keys := []string{"path", "url", "uri", "application", "app", "target", "query", "request", "text", "name", "input"}
	for _, key := range keys {
		raw, ok := input[key]
		if !ok {
			continue
		}
		switch value := raw.(type) {
		case string:
			s := strings.TrimSpace(value)
			if s != "" {
				return s
			}
		case fmt.Stringer:
			s := strings.TrimSpace(value.String())
			if s != "" {
				return s
			}
		}
	}
	return ""
}

func desktopInlineCommandFromInput(input map[string]any) []string {
	if input == nil {
		return nil
	}
	keys := []string{"query", "request", "text", "input", "target", "application", "app", "name"}
	for _, key := range keys {
		raw, ok := input[key]
		if !ok {
			continue
		}
		text := ""
		switch value := raw.(type) {
		case string:
			text = strings.TrimSpace(value)
		case fmt.Stringer:
			text = strings.TrimSpace(value.String())
		}
		if text == "" {
			continue
		}
		_, cmd := desktopSplitAppAndCommand(text)
		if len(cmd) > 0 {
			return cmd
		}
	}
	return nil
}

func desktopLooksLikeURL(v string) bool {
	v = strings.TrimSpace(strings.ToLower(v))
	return strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "https://") || strings.HasPrefix(v, "mailto:")
}

func desktopLooksLikePath(v string) bool {
	s := strings.TrimSpace(v)
	if s == "" {
		return false
	}
	if strings.HasPrefix(s, "/") || strings.HasPrefix(s, "./") || strings.HasPrefix(s, "../") || strings.HasPrefix(s, "~") {
		return true
	}
	return strings.Contains(s, "/")
}

func desktopSplitAppAndCommand(raw string) (appHint string, command []string) {
	normalized := desktopNormalizeAppHint(raw)
	if normalized == "" {
		return "", nil
	}
	if strings.HasPrefix(normalized, "ping ") {
		target := desktopExtractPingTarget(strings.TrimSpace(strings.TrimPrefix(normalized, "ping ")))
		if target != "" {
			return "terminal", []string{"ping", target}
		}
		return normalized, nil
	}
	delims := []string{
		" and run ping ",
		" and ping ",
		" to run ping ",
		" to ping ",
		" then ping ",
	}
	for _, delim := range delims {
		idx := strings.Index(normalized, delim)
		if idx <= 0 {
			continue
		}
		target := desktopExtractPingTarget(strings.TrimSpace(normalized[idx+len(delim):]))
		if target == "" {
			return normalized, nil
		}
		app := strings.TrimSpace(normalized[:idx])
		if app == "" {
			app = "terminal"
		}
		return app, []string{"ping", target}
	}
	return normalized, nil
}

func desktopExtractPingTarget(raw string) string {
	fields := strings.Fields(strings.TrimSpace(raw))
	if len(fields) == 0 {
		return ""
	}
	token := strings.Trim(fields[0], `"'`)
	if !desktopSafePingTarget(token) {
		return ""
	}
	return token
}

func desktopSafePingTarget(token string) bool {
	if token == "" {
		return false
	}
	for _, r := range token {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case r == '.', r == '-', r == ':':
		default:
			return false
		}
	}
	return true
}

func desktopTerminalLaunchArgs(command string, cmd []string) ([]string, bool) {
	if len(cmd) == 0 {
		return nil, false
	}
	switch command {
	case "konsole":
		return append([]string{"--noclose", "-e"}, cmd...), true
	case "x-terminal-emulator", "xfce4-terminal", "alacritty":
		return append([]string{"-e"}, cmd...), true
	case "gnome-terminal":
		return append([]string{"--"}, cmd...), true
	case "kitty":
		return append([]string{}, cmd...), true
	default:
		return nil, false
	}
}

func desktopResolveAppLaunch(raw string) (command string, args []string, appName string, err error) {
	hint := desktopNormalizeAppHint(raw)
	if hint == "" {
		return "", nil, "", errors.New("desktop.open input.application cannot be empty")
	}

	candidates := desktopLaunchCandidates(hint)
	for _, candidate := range candidates {
		if len(candidate) == 0 {
			continue
		}
		if _, lookErr := exec.LookPath(candidate[0]); lookErr == nil {
			return candidate[0], candidate[1:], hint, nil
		}
	}
	return "", nil, "", fmt.Errorf("no launcher found for %q on this system", hint)
}

func desktopNormalizeAppHint(raw string) string {
	s := strings.TrimSpace(strings.ToLower(raw))
	if s == "" {
		return ""
	}
	s = strings.Trim(s, `"'`)
	s = strings.TrimSuffix(s, ".")
	for _, prefix := range []string{"open ", "launch ", "start "} {
		if strings.HasPrefix(s, prefix) {
			s = strings.TrimSpace(strings.TrimPrefix(s, prefix))
		}
		if strings.HasPrefix(s, "the "+prefix) {
			s = strings.TrimSpace(strings.TrimPrefix(s, "the "+prefix))
		}
	}
	s = strings.TrimPrefix(s, "the ")
	s = strings.Join(strings.Fields(s), " ")
	return s
}

func desktopLaunchCandidates(hint string) [][]string {
	normalized := strings.TrimSpace(strings.ToLower(hint))
	switch {
	case normalized == "konsole" || strings.Contains(normalized, "konsole"):
		return [][]string{
			{"konsole"},
			{"gtk-launch", "org.kde.konsole.desktop"},
		}
	case strings.Contains(normalized, "terminal"):
		return [][]string{
			{"x-terminal-emulator"},
			{"konsole"},
			{"gnome-terminal"},
			{"xfce4-terminal"},
			{"kitty"},
			{"alacritty"},
		}
	case strings.Contains(normalized, "software center"),
		strings.Contains(normalized, "software manager"),
		strings.Contains(normalized, "app store"),
		strings.Contains(normalized, "discover"):
		return [][]string{
			{"plasma-discover"},
			{"discover"},
			{"gnome-software"},
			{"snap-store"},
			{"gtk-launch", "org.kde.discover.desktop"},
			{"gtk-launch", "org.gnome.Software.desktop"},
		}
	default:
		fields := strings.Fields(normalized)
		if len(fields) > 0 && desktopSafeCommandToken(fields[0]) {
			return [][]string{{fields[0]}}
		}
	}
	return nil
}

func desktopSafeCommandToken(token string) bool {
	if token == "" {
		return false
	}
	for _, r := range token {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case r == '-', r == '_', r == '.', r == '+':
		default:
			return false
		}
	}
	return true
}

type networkInterfacesTool struct{}

func (t *networkInterfacesTool) ID() string             { return "net.interfaces" }
func (t *networkInterfacesTool) Domain() string         { return "network" }
func (t *networkInterfacesTool) Action() string         { return "inspect_interfaces" }
func (t *networkInterfacesTool) RiskClass() string      { return "read_only" }
func (t *networkInterfacesTool) ExecutionLevel() string { return "L0" }
func (t *networkInterfacesTool) Executes() bool         { return false }
func (t *networkInterfacesTool) UsesNetwork() bool      { return false }
func (t *networkInterfacesTool) WriteIntent() bool      { return false }
func (t *networkInterfacesTool) Description() string    { return "Inspect local network interfaces" }
func (t *networkInterfacesTool) Execute(ctx context.Context, req Request) (Result, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return Result{}, err
	}
	out := make([]map[string]any, 0, len(ifaces))
	for _, iface := range ifaces {
		addrs, _ := iface.Addrs()
		a := []string{}
		for _, addr := range addrs {
			a = append(a, addr.String())
		}
		out = append(out, map[string]any{"name": iface.Name, "mtu": iface.MTU, "flags": iface.Flags.String(), "addrs": a})
	}
	return Result{Data: map[string]any{"interfaces": out, "count": len(out)}, Message: "interfaces listed"}, nil
}

type networkConnectivityTool struct{}

func (t *networkConnectivityTool) ID() string             { return "net.connectivity" }
func (t *networkConnectivityTool) Domain() string         { return "network" }
func (t *networkConnectivityTool) Action() string         { return "test_connectivity" }
func (t *networkConnectivityTool) RiskClass() string      { return "scoped_execute" }
func (t *networkConnectivityTool) ExecutionLevel() string { return "L2" }
func (t *networkConnectivityTool) Executes() bool         { return false }
func (t *networkConnectivityTool) UsesNetwork() bool      { return true }
func (t *networkConnectivityTool) WriteIntent() bool      { return false }
func (t *networkConnectivityTool) Description() string    { return "Check TCP connectivity to host:port" }
func (t *networkConnectivityTool) Execute(ctx context.Context, req Request) (Result, error) {
	target, _ := req.Input["target"].(string)
	target = strings.TrimSpace(target)
	if target == "" {
		target = "1.1.1.1:53"
	}
	timeoutMs := int(readFloat(req.Input, "timeoutMs", 5000))
	if timeoutMs <= 0 {
		timeoutMs = 5000
	}
	conn, err := net.DialTimeout("tcp", target, time.Duration(timeoutMs)*time.Millisecond)
	ok := err == nil
	if conn != nil {
		_ = conn.Close()
	}
	return Result{Data: map[string]any{"target": target, "ok": ok, "error": errString(err)}, Message: "connectivity test complete"}, nil
}

type networkDNSLookupTool struct{}

func (t *networkDNSLookupTool) ID() string             { return "net.dns_lookup" }
func (t *networkDNSLookupTool) Domain() string         { return "network" }
func (t *networkDNSLookupTool) Action() string         { return "dns_lookup" }
func (t *networkDNSLookupTool) RiskClass() string      { return "read_only" }
func (t *networkDNSLookupTool) ExecutionLevel() string { return "L0" }
func (t *networkDNSLookupTool) Executes() bool         { return false }
func (t *networkDNSLookupTool) UsesNetwork() bool      { return true }
func (t *networkDNSLookupTool) WriteIntent() bool      { return false }
func (t *networkDNSLookupTool) Description() string    { return "Resolve DNS name to IP addresses" }
func (t *networkDNSLookupTool) Execute(ctx context.Context, req Request) (Result, error) {
	host, _ := req.Input["host"].(string)
	host = strings.TrimSpace(host)
	if host == "" {
		return Result{}, errors.New("net.dns_lookup requires input.host")
	}
	addrs, err := net.LookupHost(host)
	return Result{Data: map[string]any{"host": host, "addresses": addrs, "ok": err == nil, "error": errString(err)}, Message: "dns lookup complete"}, nil
}

type networkFetchTool struct{}

func (t *networkFetchTool) ID() string             { return "net.fetch" }
func (t *networkFetchTool) Domain() string         { return "network" }
func (t *networkFetchTool) Action() string         { return "fetch_url" }
func (t *networkFetchTool) RiskClass() string      { return "scoped_execute" }
func (t *networkFetchTool) ExecutionLevel() string { return "L2" }
func (t *networkFetchTool) Executes() bool         { return false }
func (t *networkFetchTool) UsesNetwork() bool      { return true }
func (t *networkFetchTool) WriteIntent() bool      { return false }
func (t *networkFetchTool) Description() string    { return "Fetch approved URL content (GET)" }
func (t *networkFetchTool) Execute(ctx context.Context, req Request) (Result, error) {
	raw, _ := req.Input["url"].(string)
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Result{}, errors.New("net.fetch requires input.url")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return Result{}, err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return Result{}, errors.New("only http/https URLs are allowed")
	}
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Get(parsed.String())
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	return Result{
		Data: map[string]any{
			"url":        parsed.String(),
			"statusCode": resp.StatusCode,
			"ok":         resp.StatusCode >= 200 && resp.StatusCode < 300,
			"body":       string(body),
		},
		Message: "url fetched",
	}, nil
}

type secretGetTool struct{ db *sql.DB }

type timeNowTool struct{}

func (t *timeNowTool) ID() string             { return "time.now" }
func (t *timeNowTool) Domain() string         { return "time" }
func (t *timeNowTool) Action() string         { return "get_system_time" }
func (t *timeNowTool) RiskClass() string      { return "read_only" }
func (t *timeNowTool) ExecutionLevel() string { return "L0" }
func (t *timeNowTool) Executes() bool         { return false }
func (t *timeNowTool) UsesNetwork() bool      { return false }
func (t *timeNowTool) WriteIntent() bool      { return false }
func (t *timeNowTool) Description() string {
	return "Read current system clock in UTC and local timezone"
}
func (t *timeNowTool) Execute(_ context.Context, _ Request) (Result, error) {
	now := time.Now()
	_, offset := now.Zone()
	return Result{
		Data: map[string]any{
			"unixMs":      now.UnixMilli(),
			"iso8601":     now.Format(time.RFC3339Nano),
			"utcIso8601":  now.UTC().Format(time.RFC3339Nano),
			"zoneOffsetS": offset,
		},
		Message: "system time captured",
	}, nil
}

func (t *secretGetTool) ID() string             { return "secret.get" }
func (t *secretGetTool) Domain() string         { return "secrets" }
func (t *secretGetTool) Action() string         { return "get_secret_ref" }
func (t *secretGetTool) RiskClass() string      { return "privileged" }
func (t *secretGetTool) ExecutionLevel() string { return "L3" }
func (t *secretGetTool) Executes() bool         { return false }
func (t *secretGetTool) UsesNetwork() bool      { return false }
func (t *secretGetTool) WriteIntent() bool      { return false }
func (t *secretGetTool) Description() string {
	return "Resolve secret logical name and return masked metadata only"
}
func (t *secretGetTool) Execute(ctx context.Context, req Request) (Result, error) {
	name, _ := req.Input["name"].(string)
	name = strings.TrimSpace(name)
	if name == "" {
		return Result{}, errors.New("secret.get requires input.name")
	}
	var raw string
	err := t.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = ?`, "secret."+name).Scan(&raw)
	if err != nil {
		return Result{}, fmt.Errorf("secret %q not found", name)
	}
	return Result{
		Data: map[string]any{
			"name":     name,
			"exists":   true,
			"length":   len(strings.TrimSpace(raw)),
			"revealed": false,
			"masked":   maskSecret(raw),
		},
		Message: "secret metadata resolved",
	}, nil
}

// --- helpers ---

func laneCovers(lane *lanes.Lane, target string) bool {
	if len(lane.AllowedPaths) == 0 {
		return false
	}
	for _, s := range lane.ForbiddenPaths {
		if pathContains(s, target) {
			return false
		}
	}
	for _, a := range lane.AllowedPaths {
		if pathContains(a, target) {
			return true
		}
	}
	return false
}

func pathContains(scope, target string) bool {
	if scope == "" || target == "" {
		return false
	}
	absScope, err := filepath.Abs(scope)
	if err != nil {
		absScope = scope
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		absTarget = target
	}
	absScope = filepath.Clean(absScope)
	absTarget = filepath.Clean(absTarget)
	if absTarget == absScope {
		return true
	}
	rel, err := filepath.Rel(absScope, absTarget)
	if err != nil {
		return false
	}
	rel = filepath.Clean(rel)
	if rel == "." {
		return true
	}
	return !strings.HasPrefix(rel, "..")
}

func resolvePaths(workspace string, paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if !filepath.IsAbs(p) {
			p = filepath.Join(workspace, p)
		}
		abs, err := filepath.Abs(p)
		if err != nil {
			abs = p
		}
		out = append(out, filepath.Clean(abs))
	}
	return out
}

func firstPath(paths []string, workspace string) (string, error) {
	if len(paths) == 0 {
		return "", errors.New("this tool requires at least one path")
	}
	p := strings.TrimSpace(paths[0])
	if !filepath.IsAbs(p) {
		p = filepath.Join(workspace, p)
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	return filepath.Clean(abs), nil
}

func writeBytesFromInput(input map[string]any) int64 {
	if input == nil {
		return 0
	}
	if v, ok := input["contents"].(string); ok {
		return int64(len(v))
	}
	if v, ok := input["bytes"].(float64); ok {
		return int64(v)
	}
	return 0
}

func nonNilMap(v map[string]any) map[string]any {
	if v == nil {
		return map[string]any{}
	}
	return v
}

func mergeGatewayMetadata(base map[string]any, add map[string]any) map[string]any {
	out := map[string]any{}
	for key, value := range base {
		out[key] = value
	}
	for key, value := range add {
		if strings.TrimSpace(fmt.Sprintf("%v", value)) == "" {
			continue
		}
		out[key] = value
	}
	return out
}

func gatewayAuditContextPayload(req Request, payload map[string]any) map[string]any {
	out := map[string]any{
		"correlationId": req.CorrelationID,
	}
	if strings.TrimSpace(req.TraceID) != "" {
		out["traceId"] = req.TraceID
	}
	if strings.TrimSpace(req.WorkspaceID) != "" {
		out["workspaceId"] = req.WorkspaceID
	}
	for key, value := range payload {
		out[key] = value
	}
	return out
}

func summarizeResultArtifacts(items []ResultArtifact) []map[string]any {
	if len(items) == 0 {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		entry := map[string]any{
			"type":    strings.TrimSpace(item.Type),
			"path":    strings.TrimSpace(item.Path),
			"summary": strings.TrimSpace(item.Summary),
		}
		out = append(out, entry)
	}
	return out
}

func capabilityIDFromRequest(req Request) string {
	if req.Metadata == nil {
		return ""
	}
	return metadataString(req.Metadata, "toolCapabilityId")
}

func metadataBool(meta map[string]any, key string) bool {
	if meta == nil {
		return false
	}
	v, ok := meta[key]
	if !ok {
		return false
	}
	switch x := v.(type) {
	case bool:
		return x
	case string:
		x = strings.TrimSpace(strings.ToLower(x))
		return x == "true" || x == "1" || x == "yes"
	case int:
		return x != 0
	case int64:
		return x != 0
	case float64:
		return x != 0
	default:
		return false
	}
}

func workspaceIDFromPath(path string) string {
	base := strings.TrimSpace(filepath.Base(filepath.Clean(path)))
	if base == "" || base == "." {
		return "workspace:default"
	}
	base = strings.ToLower(strings.ReplaceAll(base, " ", "_"))
	return "workspace:" + base
}

func mapGatewayStatusToToolStatus(status string) domain.ToolResultStatus {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case StatusOK:
		return domain.ToolStatusSucceeded
	case StatusDryRun:
		return domain.ToolStatusDryRun
	case StatusNeedsApprov:
		return domain.ToolStatusApprovalRequired
	case StatusUnsupported:
		return domain.ToolStatusUnsupported
	case StatusDisabled:
		return domain.ToolStatusDisabled
	case StatusDenied:
		return domain.ToolStatusDenied
	default:
		return domain.ToolStatusFailed
	}
}

func cloneToolOutput(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func toolErrorFromGatewayResult(result *Result) *domain.ToolExecutionError {
	if result == nil {
		return &domain.ToolExecutionError{Code: domain.ToolErrExecutionFailed, Message: "gateway result is nil"}
	}
	msg := strings.TrimSpace(result.DeniedReason)
	if msg == "" && strings.TrimSpace(result.Message) != "" && strings.TrimSpace(strings.ToLower(result.Status)) != StatusOK {
		msg = strings.TrimSpace(result.Message)
	}
	if msg == "" {
		return nil
	}
	code := domain.ToolErrPolicyDenied
	switch strings.TrimSpace(strings.ToLower(result.Status)) {
	case StatusNeedsApprov:
		code = domain.ToolErrApprovalRequired
	case StatusUnsupported:
		code = domain.ToolErrUnsupportedOperation
	case StatusDisabled:
		code = domain.ToolErrToolDisabled
	case StatusError:
		code = domain.ToolErrExecutionFailed
	}
	return &domain.ToolExecutionError{
		Code:    code,
		Message: msg,
	}
}

func warningsFromGatewayResult(result *Result) []string {
	if result == nil {
		return nil
	}
	raw, ok := result.Data["warnings"]
	if !ok {
		return nil
	}
	out := []string{}
	switch rows := raw.(type) {
	case []string:
		return rows
	case []any:
		for _, item := range rows {
			s := strings.TrimSpace(fmt.Sprintf("%v", item))
			if s != "" {
				out = append(out, s)
			}
		}
	}
	return out
}

func appendStringWarning(existing any, warning string) []string {
	out := []string{}
	switch rows := existing.(type) {
	case []string:
		out = append(out, rows...)
	case []any:
		for _, row := range rows {
			text := strings.TrimSpace(fmt.Sprintf("%v", row))
			if text != "" {
				out = append(out, text)
			}
		}
	}
	warning = strings.TrimSpace(warning)
	if warning != "" {
		out = append(out, warning)
	}
	return out
}

func newCorrelationID() string {
	var buf [8]byte
	_, _ = rand.Read(buf[:])
	return "corr-" + hex.EncodeToString(buf[:])
}

func nullInt64(v *int64) any {
	if v == nil {
		return nil
	}
	return *v
}

func nonEmpty(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return strings.TrimSpace(v)
}

func (g *Gateway) approvalGrantPresent(ctx context.Context, req Request, lane *lanes.Lane, tool Tool, risk, level string, resolvedPaths []string) (bool, error) {
	if strings.TrimSpace(req.ApprovalID) != "" && g.approvals != nil {
		requestID, err := strconv.ParseInt(strings.TrimSpace(req.ApprovalID), 10, 64)
		if err != nil {
			return false, fmt.Errorf("invalid approval id %q", req.ApprovalID)
		}
		approvalReq, err := g.approvals.GetRequest(ctx, requestID)
		if err != nil {
			return false, err
		}
		if req.JobID != nil && strings.TrimSpace(*req.JobID) != "" && strings.TrimSpace(approvalReq.JobID) != strings.TrimSpace(*req.JobID) {
			return false, fmt.Errorf("approval request %d belongs to job %q, not %q", requestID, approvalReq.JobID, strings.TrimSpace(*req.JobID))
		}
		reqForFingerprint := req
		if reqForFingerprint.JobID == nil || strings.TrimSpace(*reqForFingerprint.JobID) == "" {
			jid := strings.TrimSpace(approvalReq.JobID)
			reqForFingerprint.JobID = &jid
		}
		actualHash, _ := g.approvalFingerprintForRequestID(reqForFingerprint, lane, tool, risk, level, resolvedPaths, requestID)
		expectedHash := approvalFingerprintHashFromScope(approvalReq.ScopeSnapshot)
		if expectedHash == "" {
			return false, fmt.Errorf("approval request %d is missing gateway approval fingerprint", requestID)
		}
		if actualHash != expectedHash {
			return false, fmt.Errorf("approval request %d fingerprint mismatch", requestID)
		}
		if approvalReq.Decision != nil && strings.EqualFold(strings.TrimSpace(approvalReq.Decision.Decision), "approved") {
			return true, nil
		}
		return false, nil
	}
	return g.jobApprovalGranted(ctx, req.JobID)
}

func (g *Gateway) approvalFingerprint(req Request, lane *lanes.Lane, tool Tool, risk, level string, resolvedPaths []string) (string, map[string]any) {
	return g.approvalFingerprintForRequestID(req, lane, tool, risk, level, resolvedPaths, 0)
}

func (g *Gateway) approvalFingerprintForRequestID(req Request, lane *lanes.Lane, tool Tool, risk, level string, resolvedPaths []string, approvalRequestID int64) (string, map[string]any) {
	jobID := ""
	if req.JobID != nil {
		jobID = strings.TrimSpace(*req.JobID)
	}
	fields := map[string]any{
		"version":          "gateway.v1",
		"actorId":          nonEmpty(req.ProvenanceActor, req.Initiator),
		"actorKind":        nonEmpty(req.ProvenanceActorType, req.Source),
		"initiator":        nonEmpty(req.Initiator, "operator"),
		"source":           nonEmpty(req.Source, "user"),
		"workspaceId":      nonEmpty(req.WorkspaceID, workspaceIDFromPath(g.workspace)),
		"laneId":           lane.ID,
		"toolId":           tool.ID(),
		"capabilityId":     capabilityIDFromRequest(req),
		"riskClass":        strings.TrimSpace(risk),
		"executionLevel":   strings.TrimSpace(level),
		"writeIntent":      tool.WriteIntent(),
		"jobId":            jobID,
		"domain":           nonEmpty(req.Domain, tool.Domain()),
		"action":           nonEmpty(req.Action, tool.Action()),
		"requestedPaths":   normalizedApprovalPaths(req.Paths),
		"resolvedPaths":    normalizedApprovalPaths(resolvedPaths),
		"inputActionShape": normalizeApprovalFingerprintValue(nonNilMap(req.Input)),
	}
	if approvalRequestID > 0 {
		fields["approvalRequestId"] = approvalRequestID
	}
	body, _ := json.Marshal(fields)
	sum := sha256.Sum256(body)
	return "sha256:" + hex.EncodeToString(sum[:]), fields
}

func (g *Gateway) updateApprovalRequestScopeSnapshot(ctx context.Context, requestID int64, scope map[string]any) error {
	scopeJSON, err := json.Marshal(scope)
	if err != nil {
		return err
	}
	_, err = g.db.ExecContext(ctx, `UPDATE approval_requests SET scope_snapshot_json = ? WHERE id = ?`, string(scopeJSON), requestID)
	return err
}

func approvalFingerprintHashFromScope(raw json.RawMessage) string {
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

func normalizedApprovalPaths(paths []string) []string {
	out := make([]string, 0, len(paths))
	seen := map[string]struct{}{}
	for _, p := range paths {
		p = filepath.Clean(strings.TrimSpace(p))
		if p == "." || p == "" {
			continue
		}
		key := strings.ToLower(filepath.ToSlash(p))
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, filepath.ToSlash(p))
	}
	sort.Strings(out)
	return out
}

func normalizeApprovalFingerprintValue(v any) any {
	switch typed := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for k, item := range typed {
			out[k] = normalizeApprovalFingerprintValue(item)
		}
		return out
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, normalizeApprovalFingerprintValue(item))
		}
		return out
	case []string:
		out := append([]string(nil), typed...)
		sort.Strings(out)
		return out
	default:
		return typed
	}
}

func (g *Gateway) jobApprovalGranted(ctx context.Context, jobID *string) (bool, error) {
	if !g.jobApprovalStatusGranted(ctx, jobID) {
		return false, nil
	}
	return true, nil
}

func (g *Gateway) jobApprovalStatusGranted(ctx context.Context, jobID *string) bool {
	if jobID == nil {
		return false
	}
	id := strings.TrimSpace(*jobID)
	if id == "" {
		return false
	}
	var status string
	err := g.db.QueryRowContext(ctx, `SELECT approval_status FROM jobs WHERE id = ?`, id).Scan(&status)
	if err != nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(status), "granted")
}

func (g *Gateway) jobApprovalFingerprintGranted(ctx context.Context, req Request, lane *lanes.Lane, tool Tool, risk, level string, resolvedPaths []string) (bool, error) {
	if req.JobID == nil || strings.TrimSpace(*req.JobID) == "" {
		return false, nil
	}
	jobID := strings.TrimSpace(*req.JobID)
	row := g.db.QueryRowContext(ctx, `
SELECT ar.id, ar.scope_snapshot_json
FROM approval_requests ar
JOIN approval_decisions ad ON ad.request_id = ar.id
WHERE ar.job_id = ? AND ar.status = 'resolved' AND lower(ad.decision) = 'approved'
ORDER BY ad.id DESC
LIMIT 1`, jobID)
	var requestID int64
	var scope string
	if err := row.Scan(&requestID, &scope); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	expectedHash := approvalFingerprintHashFromScope(json.RawMessage(scope))
	if expectedHash == "" {
		return false, fmt.Errorf("approval request %d is missing gateway approval fingerprint", requestID)
	}
	actualHash, _ := g.approvalFingerprintForRequestID(req, lane, tool, risk, level, resolvedPaths, requestID)
	if actualHash != expectedHash {
		return false, fmt.Errorf("approval request %d fingerprint mismatch", requestID)
	}
	return true, nil
}

func toolDomainFromID(id string) string {
	id = strings.TrimSpace(strings.ToLower(id))
	if strings.Contains(id, ".") {
		return strings.Split(id, ".")[0]
	}
	return "unknown"
}

func normalizeExecutionLevel(level string) string {
	switch strings.ToUpper(strings.TrimSpace(level)) {
	case "L0":
		return "L0"
	case "L1":
		return "L1"
	case "L2":
		return "L2"
	case "L3":
		return "L3"
	case "L4":
		return "L4"
	default:
		return ""
	}
}

func executionLevelFromRisk(risk string) string {
	switch strings.TrimSpace(strings.ToLower(risk)) {
	case "read_only":
		return "L0"
	case "low_write":
		return "L1"
	case "safe_write":
		return "L1"
	case "scoped_execute":
		return "L2"
	case "privileged":
		return "L3"
	case "dangerous":
		return "L4"
	case "low":
		return "L0"
	case "medium":
		return "L1"
	case "high":
		return "L3"
	default:
		return "L0"
	}
}

func legacyRiskClass(risk string) string {
	switch strings.TrimSpace(strings.ToLower(risk)) {
	case "read_only":
		return "low"
	case "low_write":
		return "low"
	case "safe_write":
		return "medium"
	case "scoped_execute":
		return "medium"
	case "privileged":
		return "high"
	case "dangerous":
		return "high"
	default:
		return strings.TrimSpace(strings.ToLower(risk))
	}
}

func effectiveRiskClass(requested, lane, tool string, extra ...string) string {
	best := strings.TrimSpace(strings.ToLower(tool))
	bestRank := levelRank(executionLevelFromRisk(best))
	candidates := []string{lane, requested}
	candidates = append(candidates, extra...)
	for _, candidate := range candidates {
		risk := strings.TrimSpace(strings.ToLower(candidate))
		if risk == "" {
			continue
		}
		rank := levelRank(executionLevelFromRisk(risk))
		if rank > bestRank {
			best = risk
			bestRank = rank
		}
	}
	if best == "" {
		return "read_only"
	}
	return best
}

func levelRank(level string) int {
	switch normalizeExecutionLevel(level) {
	case "L0":
		return 0
	case "L1":
		return 1
	case "L2":
		return 2
	case "L3":
		return 3
	case "L4":
		return 4
	default:
		return -1
	}
}

func runCmd(ctx context.Context, dir string, parts ...string) (string, error) {
	if len(parts) == 0 {
		return "", errors.New("command required")
	}
	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	if strings.TrimSpace(dir) != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func runDetachedCmd(dir string, parts ...string) (int, error) {
	if len(parts) == 0 {
		return 0, errors.New("command required")
	}
	cmd := exec.Command(parts[0], parts[1:]...)
	if strings.TrimSpace(dir) != "" {
		cmd.Dir = dir
	}
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	pid := 0
	if cmd.Process != nil {
		pid = cmd.Process.Pid
		_ = cmd.Process.Release()
	}
	return pid, nil
}

func readFloat(in map[string]any, key string, def float64) float64 {
	if in == nil {
		return def
	}
	v, ok := in[key]
	if !ok {
		return def
	}
	switch x := v.(type) {
	case float64:
		return x
	case int:
		return float64(x)
	case int64:
		return float64(x)
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(x), 64)
		if err == nil {
			return f
		}
	}
	return def
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func maskSecret(v string) string {
	s := strings.TrimSpace(v)
	if s == "" {
		return ""
	}
	if len(s) <= 4 {
		return "****"
	}
	return s[:2] + strings.Repeat("*", len(s)-4) + s[len(s)-2:]
}
