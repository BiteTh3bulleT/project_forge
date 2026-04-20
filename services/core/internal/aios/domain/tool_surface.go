package domain

import (
	"fmt"
	"strings"
)

type ToolCapabilityStatus string

const (
	ToolCapabilityActive       ToolCapabilityStatus = "active"
	ToolCapabilityDisabled     ToolCapabilityStatus = "disabled"
	ToolCapabilityStubbed      ToolCapabilityStatus = "stubbed"
	ToolCapabilityApprovalOnly ToolCapabilityStatus = "approval_only"
	ToolCapabilityDeprecated   ToolCapabilityStatus = "deprecated"
)

type ToolLane string

const (
	ToolLaneControl ToolLane = "control"
	ToolLaneIO      ToolLane = "io"
	ToolLaneCompute ToolLane = "compute"
)

type ToolEffect string

const (
	ToolEffectRead        ToolEffect = "read"
	ToolEffectWrite       ToolEffect = "write"
	ToolEffectExecute     ToolEffect = "execute"
	ToolEffectNetwork     ToolEffect = "network"
	ToolEffectExternal    ToolEffect = "external"
	ToolEffectPrivileged  ToolEffect = "privileged"
	ToolEffectDestructive ToolEffect = "destructive"
)

type ToolRisk string

const (
	ToolRiskNone     ToolRisk = "none"
	ToolRiskLow      ToolRisk = "low"
	ToolRiskMedium   ToolRisk = "medium"
	ToolRiskHigh     ToolRisk = "high"
	ToolRiskCritical ToolRisk = "critical"
)

func (r ToolRisk) Rank() int {
	switch r {
	case ToolRiskNone:
		return 0
	case ToolRiskLow:
		return 1
	case ToolRiskMedium:
		return 2
	case ToolRiskHigh:
		return 3
	case ToolRiskCritical:
		return 4
	default:
		return 5
	}
}

type ToolAuditLevel string

const (
	ToolAuditMinimal ToolAuditLevel = "minimal"
	ToolAuditBasic   ToolAuditLevel = "basic"
	ToolAuditVerbose ToolAuditLevel = "verbose"
)

type ToolArtifactBehavior string

const (
	ToolArtifactNone     ToolArtifactBehavior = "none"
	ToolArtifactOptional ToolArtifactBehavior = "optional"
	ToolArtifactRequired ToolArtifactBehavior = "required"
)

type ToolResourceCost struct {
	CPU           int `json:"cpu,omitempty"`
	Memory        int `json:"memory,omitempty"`
	IO            int `json:"io,omitempty"`
	Network       int `json:"network,omitempty"`
	Duration      int `json:"duration,omitempty"`
	CostUnits     int `json:"costUnits,omitempty"`
	Concurrency   int `json:"concurrency,omitempty"`
	ExternalCalls int `json:"externalCalls,omitempty"`
}

type ToolResourceLimits struct {
	MaxDurationMs  int      `json:"maxDurationMs,omitempty"`
	MaxOutputBytes int      `json:"maxOutputBytes,omitempty"`
	MaxMemoryMB    int      `json:"maxMemoryMb,omitempty"`
	MaxCPUPercent  int      `json:"maxCpuPercent,omitempty"`
	MaxNetworkByte int      `json:"maxNetworkBytes,omitempty"`
	AllowedPaths   []string `json:"allowedPaths,omitempty"`
	DeniedPaths    []string `json:"deniedPaths,omitempty"`
	AllowedHosts   []string `json:"allowedHosts,omitempty"`
	DeniedHosts    []string `json:"deniedHosts,omitempty"`
	AllowedMethods []string `json:"allowedMethods,omitempty"`
	SandboxMode    string   `json:"sandboxMode,omitempty"`
}

type ToolCapability struct {
	ID                        string               `json:"id"`
	Domain                    string               `json:"domain"`
	Name                      string               `json:"name"`
	Description               string               `json:"description"`
	Status                    ToolCapabilityStatus `json:"status"`
	Lane                      ToolLane             `json:"lane"`
	Effect                    []ToolEffect         `json:"effect"`
	Risk                      ToolRisk             `json:"risk"`
	RequiresWorkspace         bool                 `json:"requiresWorkspace"`
	RequiresIntent            bool                 `json:"requiresIntent"`
	RequiresApprovalByDefault bool                 `json:"requiresApprovalByDefault"`
	AutonomyEligible          bool                 `json:"autonomyEligible"`
	AllowedInDryRun           bool                 `json:"allowedInDryRun"`
	RequiredCapabilities      []string             `json:"requiredCapabilities,omitempty"`
	PolicyTags                []string             `json:"policyTags,omitempty"`
	ResourceCost              ToolResourceCost     `json:"resourceCost"`
	ResourceLimits            ToolResourceLimits   `json:"resourceLimits"`
	InputSchema               map[string]any       `json:"inputSchema,omitempty"`
	OutputSchema              map[string]any       `json:"outputSchema,omitempty"`
	AuditLevel                ToolAuditLevel       `json:"auditLevel"`
	ArtifactBehavior          ToolArtifactBehavior `json:"artifactBehavior"`
	RollbackSupport           bool                 `json:"rollbackSupport"`
	AdapterID                 string               `json:"adapterId,omitempty"`
	Metadata                  map[string]any       `json:"metadata,omitempty"`
}

func (c ToolCapability) Validate() []ToolExecutionError {
	issues := []ToolExecutionError{}
	if strings.TrimSpace(c.ID) == "" {
		issues = append(issues, ToolExecutionError{Code: ToolErrInvalidPayload, Field: "id", Message: "capability id is required"})
	}
	if strings.TrimSpace(c.Domain) == "" {
		issues = append(issues, ToolExecutionError{Code: ToolErrInvalidPayload, Field: "domain", Message: "capability domain is required"})
	}
	if strings.TrimSpace(c.Name) == "" {
		issues = append(issues, ToolExecutionError{Code: ToolErrInvalidPayload, Field: "name", Message: "capability name is required"})
	}
	if strings.TrimSpace(string(c.Status)) == "" {
		issues = append(issues, ToolExecutionError{Code: ToolErrInvalidPayload, Field: "status", Message: "capability status is required"})
	}
	if strings.TrimSpace(string(c.Risk)) == "" {
		issues = append(issues, ToolExecutionError{Code: ToolErrInvalidPayload, Field: "risk", Message: "capability risk is required"})
	}
	return issues
}

type ToolResultStatus string

const (
	ToolStatusSucceeded        ToolResultStatus = "succeeded"
	ToolStatusFailed           ToolResultStatus = "failed"
	ToolStatusDenied           ToolResultStatus = "denied"
	ToolStatusApprovalRequired ToolResultStatus = "approval_required"
	ToolStatusDryRun           ToolResultStatus = "dry_run"
	ToolStatusUnsupported      ToolResultStatus = "unsupported"
	ToolStatusDisabled         ToolResultStatus = "disabled"
)

type ToolRequest struct {
	ID            string         `json:"id"`
	ToolID        string         `json:"toolId"`
	Actor         ActorIdentity  `json:"actor"`
	Source        ActionSource   `json:"source"`
	Scope         ForgeScope     `json:"scope"`
	IntentID      string         `json:"intentId,omitempty"`
	CharterID     string         `json:"charterId,omitempty"`
	BudgetID      string         `json:"budgetId,omitempty"`
	ApprovalID    string         `json:"approvalId,omitempty"`
	Payload       map[string]any `json:"payload"`
	DryRun        bool           `json:"dryRun"`
	RequestedAt   int64          `json:"requestedAt"`
	CorrelationID string         `json:"correlationId,omitempty"`
	TraceID       string         `json:"traceId,omitempty"`
	Provenance    Provenance     `json:"provenance"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

func (r ToolRequest) Validate() []ToolExecutionError {
	issues := []ToolExecutionError{}
	if strings.TrimSpace(r.ID) == "" {
		issues = append(issues, ToolExecutionError{Code: ToolErrInvalidPayload, Field: "id", Message: "tool request id is required"})
	}
	if strings.TrimSpace(r.ToolID) == "" {
		issues = append(issues, ToolExecutionError{Code: ToolErrToolNotFound, Field: "toolId", Message: "tool id is required"})
	}
	if strings.TrimSpace(r.Actor.ID) == "" {
		issues = append(issues, ToolExecutionError{Code: ToolErrUnauthorized, Field: "actor.id", Message: "actor.id is required"})
	}
	if strings.TrimSpace(r.Actor.Kind) == "" {
		issues = append(issues, ToolExecutionError{Code: ToolErrUnauthorized, Field: "actor.kind", Message: "actor.kind is required"})
	}
	if strings.TrimSpace(r.Scope.WorkspaceID) == "" {
		issues = append(issues, ToolExecutionError{Code: ToolErrScopeDenied, Field: "scope.workspaceId", Message: "scope.workspaceId is required"})
	}
	if strings.TrimSpace(r.Provenance.Actor) == "" {
		issues = append(issues, ToolExecutionError{Code: ToolErrInvalidPayload, Field: "provenance.actor", Message: "provenance.actor is required"})
	}
	if strings.TrimSpace(r.Provenance.ActorType) == "" {
		issues = append(issues, ToolExecutionError{Code: ToolErrInvalidPayload, Field: "provenance.actorType", Message: "provenance.actorType is required"})
	}
	if r.RequestedAt <= 0 {
		issues = append(issues, ToolExecutionError{Code: ToolErrInvalidPayload, Field: "requestedAt", Message: "requestedAt must be a positive timestamp"})
	}
	return issues
}

type ToolResourceUsage struct {
	CPUPercent   float64 `json:"cpuPercent,omitempty"`
	MemoryMB     float64 `json:"memoryMb,omitempty"`
	DurationMs   int64   `json:"durationMs,omitempty"`
	OutputBytes  int64   `json:"outputBytes,omitempty"`
	NetworkBytes int64   `json:"networkBytes,omitempty"`
	CostUnits    int64   `json:"costUnits,omitempty"`
}

type ToolResult struct {
	Success       bool                `json:"success"`
	ToolID        string              `json:"toolId"`
	RequestID     string              `json:"requestId"`
	Status        ToolResultStatus    `json:"status"`
	Output        map[string]any      `json:"output,omitempty"`
	Error         *ToolExecutionError `json:"error,omitempty"`
	Warnings      []string            `json:"warnings,omitempty"`
	Artifacts     []ArtifactRef       `json:"artifacts,omitempty"`
	AuditID       string              `json:"auditId,omitempty"`
	ResourceUsage ToolResourceUsage   `json:"resourceUsage,omitempty"`
	StartedAt     int64               `json:"startedAt"`
	CompletedAt   int64               `json:"completedAt"`
	CorrelationID string              `json:"correlationId,omitempty"`
	TraceID       string              `json:"traceId,omitempty"`
	Metadata      map[string]any      `json:"metadata,omitempty"`
}

type ToolExecutionErrorCode string

const (
	ToolErrToolNotFound         ToolExecutionErrorCode = "TOOL_NOT_FOUND"
	ToolErrToolDisabled         ToolExecutionErrorCode = "TOOL_DISABLED"
	ToolErrInvalidPayload       ToolExecutionErrorCode = "INVALID_PAYLOAD"
	ToolErrUnauthorized         ToolExecutionErrorCode = "UNAUTHORIZED"
	ToolErrApprovalRequired     ToolExecutionErrorCode = "APPROVAL_REQUIRED"
	ToolErrBudgetExceeded       ToolExecutionErrorCode = "BUDGET_EXCEEDED"
	ToolErrPolicyDenied         ToolExecutionErrorCode = "POLICY_DENIED"
	ToolErrRiskTooHigh          ToolExecutionErrorCode = "RISK_TOO_HIGH"
	ToolErrScopeDenied          ToolExecutionErrorCode = "SCOPE_DENIED"
	ToolErrResourceLimit        ToolExecutionErrorCode = "RESOURCE_LIMIT_EXCEEDED"
	ToolErrAdapterUnavailable   ToolExecutionErrorCode = "ADAPTER_UNAVAILABLE"
	ToolErrExecutionFailed      ToolExecutionErrorCode = "EXECUTION_FAILED"
	ToolErrTimeout              ToolExecutionErrorCode = "TIMEOUT"
	ToolErrDryRunOnly           ToolExecutionErrorCode = "DRY_RUN_ONLY"
	ToolErrUnsupportedOperation ToolExecutionErrorCode = "UNSUPPORTED"
)

type ToolExecutionError struct {
	Code    ToolExecutionErrorCode `json:"code"`
	Field   string                 `json:"field,omitempty"`
	Message string                 `json:"message"`
}

func (e ToolExecutionError) Error() string {
	if strings.TrimSpace(e.Field) == "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("%s (%s): %s", e.Code, e.Field, e.Message)
}
