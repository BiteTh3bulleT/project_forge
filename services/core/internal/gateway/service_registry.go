package gateway

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
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
	register(&webSearchTool{})
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
			RequiresApprovalByDefault: requiresApprovalByDefault || gatewayToolIntrinsicApprovalReason(t, t.RiskClass(), t.ExecutionLevel()) != "",
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

type CapabilityStatusUpdateMetadata struct {
	Actor             string
	ActorKind         string
	Reason            string
	RiskClass         string
	TransitionRisk    string
	ApprovalRequestID *int64
	CorrelationID     string
	TraceID           string
}

func (g *Gateway) UpdateCapabilityStatus(id string, status domain.ToolCapabilityStatus) (domain.ToolCapability, domain.ToolCapability, bool, error) {
	return g.UpdateCapabilityStatusWithMetadata(context.Background(), id, status, CapabilityStatusUpdateMetadata{Actor: "operator"})
}

func (g *Gateway) UpdateCapabilityStatusWithMetadata(ctx context.Context, id string, status domain.ToolCapabilityStatus, meta CapabilityStatusUpdateMetadata) (domain.ToolCapability, domain.ToolCapability, bool, error) {
	if g.capabilities == nil {
		return domain.ToolCapability{}, domain.ToolCapability{}, false, fmt.Errorf("capability registry unavailable")
	}
	previous, found := g.capabilities.Get(id)
	if !found {
		return domain.ToolCapability{}, domain.ToolCapability{}, false, nil
	}
	if previous.Status != status && strings.TrimSpace(meta.Reason) == "" {
		return domain.ToolCapability{}, domain.ToolCapability{}, false, fmt.Errorf("capability status change reason is required")
	}
	if previous.Status != status && strings.TrimSpace(meta.Actor) == "" {
		return domain.ToolCapability{}, domain.ToolCapability{}, false, fmt.Errorf("capability status change actor is required")
	}
	transition := ClassifyCapabilityStatusTransition(previous, status)
	if transition.RequiresApproval && meta.ApprovalRequestID == nil {
		return domain.ToolCapability{}, domain.ToolCapability{}, false, fmt.Errorf("capability status transition requires approval: %s -> %s for %s", previous.Status, status, id)
	}
	overrideMeta := ToolCapabilityOverrideMetadata{
		PreviousStatus:    previous.Status,
		ActorKind:         meta.ActorKind,
		RiskClass:         meta.RiskClass,
		TransitionRisk:    meta.TransitionRisk,
		ApprovalRequestID: meta.ApprovalRequestID,
		CorrelationID:     meta.CorrelationID,
		TraceID:           meta.TraceID,
	}
	updated, ok, err := g.capabilities.UpdateStatusWithMetadata(ctx, id, status, meta.Actor, meta.Reason, overrideMeta)
	if err != nil {
		return domain.ToolCapability{}, domain.ToolCapability{}, false, err
	}
	if !ok {
		return domain.ToolCapability{}, domain.ToolCapability{}, false, nil
	}
	return previous, updated, true, nil
}

type CapabilityStatusTransition struct {
	RiskClass        string   `json:"riskClass"`
	RequiresApproval bool     `json:"requiresApproval"`
	Elevation        bool     `json:"elevation"`
	Dangerous        bool     `json:"dangerous"`
	Reasons          []string `json:"reasons"`
}

func ClassifyCapabilityStatusTransition(previous domain.ToolCapability, nextStatus domain.ToolCapabilityStatus) CapabilityStatusTransition {
	nextStatus = domain.ToolCapabilityStatus(strings.TrimSpace(strings.ToLower(string(nextStatus))))
	out := CapabilityStatusTransition{
		RiskClass: statusTransitionRiskLow,
		Reasons:   []string{},
	}
	prevRank := capabilityStatusFreedomRank(previous.Status)
	nextRank := capabilityStatusFreedomRank(nextStatus)
	out.Elevation = nextRank > prevRank
	out.Dangerous, out.Reasons = capabilityDangerReasons(previous)
	if previous.Risk.Rank() >= domain.ToolRiskHigh.Rank() {
		out.Dangerous = true
		out.Reasons = appendUniqueString(out.Reasons, "capability risk is "+string(previous.Risk))
	}
	if out.Elevation {
		out.RiskClass = statusTransitionRiskMedium
		out.Reasons = appendUniqueString(out.Reasons, "transition increases execution freedom")
	}
	if out.Elevation && (nextStatus == domain.ToolCapabilityActive || out.Dangerous) {
		out.RiskClass = statusTransitionRiskHigh
		out.RequiresApproval = true
	}
	if previous.Status == domain.ToolCapabilityApprovalOnly && nextStatus == domain.ToolCapabilityActive && out.Dangerous {
		out.RiskClass = statusTransitionRiskHigh
		out.RequiresApproval = true
		out.Reasons = appendUniqueString(out.Reasons, "approval_only to active removes approval gate")
	}
	if previous.Status == nextStatus {
		out.RiskClass = statusTransitionRiskLow
		out.RequiresApproval = false
		out.Elevation = false
		out.Reasons = appendUniqueString(out.Reasons, "status unchanged")
	}
	if previous.Risk.Rank() > domain.ToolRiskCritical.Rank() {
		out.RiskClass = statusTransitionRiskHigh
		out.RequiresApproval = true
		out.Dangerous = true
		out.Reasons = appendUniqueString(out.Reasons, "unknown capability risk is conservative")
	}
	return out
}

const (
	statusTransitionRiskLow    = "low"
	statusTransitionRiskMedium = "medium"
	statusTransitionRiskHigh   = "high"
)

func capabilityStatusFreedomRank(status domain.ToolCapabilityStatus) int {
	switch status {
	case domain.ToolCapabilityActive:
		return 3
	case domain.ToolCapabilityApprovalOnly:
		return 2
	case domain.ToolCapabilityDisabled, domain.ToolCapabilityStubbed, domain.ToolCapabilityDeferred, domain.ToolCapabilityDeprecated:
		return 1
	default:
		return 0
	}
}

func capabilityDangerReasons(capability domain.ToolCapability) (bool, []string) {
	reasons := []string{}
	for _, effect := range capability.Effect {
		switch effect {
		case domain.ToolEffectWrite, domain.ToolEffectExecute, domain.ToolEffectNetwork, domain.ToolEffectExternal, domain.ToolEffectPrivileged, domain.ToolEffectDestructive:
			reasons = appendUniqueString(reasons, "effect "+string(effect))
		}
	}
	switch strings.TrimSpace(strings.ToLower(capability.Domain)) {
	case "process", "network", "identity", "external", "config", "backup", "device", "code":
		reasons = appendUniqueString(reasons, "domain "+capability.Domain)
	}
	return len(reasons) > 0, reasons
}

func appendUniqueString(rows []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return rows
	}
	for _, row := range rows {
		if row == value {
			return rows
		}
	}
	return append(rows, value)
}

// Execute runs the full pipeline: lane resolution, permission check,
// invocation record, tool execution, artifact capture, audit write.
