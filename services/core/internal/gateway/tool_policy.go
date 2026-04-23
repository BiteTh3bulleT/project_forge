package gateway

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
)

type ToolRiskClassification struct {
	Risk             domain.ToolRisk
	RequiresApproval bool
	Reasons          []string
}

type ToolRiskClassifier interface {
	Classify(capability domain.ToolCapability, req Request) ToolRiskClassification
}

type DeterministicToolRiskClassifier struct{}

func (DeterministicToolRiskClassifier) Classify(capability domain.ToolCapability, req Request) ToolRiskClassification {
	risk := capability.Risk
	reasons := []string{}

	if strings.EqualFold(strings.TrimSpace(req.Source), string(domain.SourceFutureIRIS)) {
		if risk.Rank() < domain.ToolRiskMedium.Rank() {
			risk = domain.ToolRiskMedium
		}
		reasons = append(reasons, "future_iris source does not bypass tool policy")
	}
	if len(req.Paths) > 0 {
		for _, p := range req.Paths {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			if filepath.IsAbs(p) {
				reasons = append(reasons, "absolute path request increases risk")
				if risk.Rank() < domain.ToolRiskHigh.Rank() {
					risk = domain.ToolRiskHigh
				}
				break
			}
		}
	}
	if weakProvenance(req) {
		reasons = append(reasons, "weak provenance increases risk")
		if risk.Rank() < domain.ToolRiskMedium.Rank() {
			risk = domain.ToolRiskMedium
		}
	}
	requiresApproval := capability.RequiresApprovalByDefault || risk.Rank() >= domain.ToolRiskHigh.Rank()
	return ToolRiskClassification{
		Risk:             risk,
		RequiresApproval: requiresApproval,
		Reasons:          reasons,
	}
}

type ToolAutonomyRequest struct {
	Request    Request
	Capability domain.ToolCapability
	Risk       ToolRiskClassification
}

type ToolAutonomyDecision struct {
	Allowed          bool
	RequiresApproval bool
	Reason           string
	Warnings         []string
}

type ToolAutonomyAuthorizer interface {
	AuthorizeToolRequest(ctx context.Context, req ToolAutonomyRequest) (ToolAutonomyDecision, error)
}

type ToolPolicyInput struct {
	Request       Request
	Capability    domain.ToolCapability
	ResolvedPaths []string
	HasAdapter    bool
}

type ToolPolicyDecision struct {
	Allowed          bool
	Status           string
	Risk             ToolRiskClassification
	RequiresApproval bool
	Reason           string
	Error            *domain.ToolExecutionError
	Warnings         []string
}

type ToolPolicyEvaluator struct {
	riskClassifier ToolRiskClassifier
	autonomy       ToolAutonomyAuthorizer
	workspace      string
}

func NewToolPolicyEvaluator(workspace string, riskClassifier ToolRiskClassifier, autonomy ToolAutonomyAuthorizer) ToolPolicyEvaluator {
	if riskClassifier == nil {
		riskClassifier = DeterministicToolRiskClassifier{}
	}
	return ToolPolicyEvaluator{
		riskClassifier: riskClassifier,
		autonomy:       autonomy,
		workspace:      strings.TrimSpace(workspace),
	}
}

func (e ToolPolicyEvaluator) Evaluate(ctx context.Context, in ToolPolicyInput) ToolPolicyDecision {
	risk := e.riskClassifier.Classify(in.Capability, in.Request)
	decision := ToolPolicyDecision{
		Allowed:          false,
		Status:           StatusDenied,
		Risk:             risk,
		RequiresApproval: false,
		Warnings:         append([]string{}, risk.Reasons...),
	}

	if strings.TrimSpace(in.Capability.ID) == "" {
		decision.Error = &domain.ToolExecutionError{Code: domain.ToolErrToolNotFound, Field: "toolId", Message: "capability not found"}
		decision.Reason = decision.Error.Message
		return decision
	}
	if !domain.IsKnownToolCapabilityStatus(in.Capability.Status) {
		decision.Status = StatusUnsupported
		decision.Error = &domain.ToolExecutionError{Code: domain.ToolErrUnsupportedOperation, Field: "status", Message: "capability status is unsupported"}
		decision.Reason = decision.Error.Message
		return decision
	}
	if in.Capability.Status == domain.ToolCapabilityDeferred {
		decision.Status = StatusUnsupported
		decision.Error = &domain.ToolExecutionError{Code: domain.ToolErrUnsupportedOperation, Field: "status", Message: "capability is deferred"}
		decision.Reason = decision.Error.Message
		return decision
	}
	if in.Capability.Status == domain.ToolCapabilityDisabled || in.Capability.Status == domain.ToolCapabilityDeprecated {
		decision.Status = StatusDisabled
		decision.Error = &domain.ToolExecutionError{Code: domain.ToolErrToolDisabled, Field: "status", Message: "capability is disabled"}
		decision.Reason = decision.Error.Message
		return decision
	}
	if in.Capability.Status == domain.ToolCapabilityStubbed && !in.HasAdapter {
		decision.Status = StatusUnsupported
		decision.Error = &domain.ToolExecutionError{Code: domain.ToolErrUnsupportedOperation, Field: "status", Message: "capability is stubbed and has no adapter"}
		decision.Reason = decision.Error.Message
		return decision
	}
	if !in.HasAdapter {
		decision.Status = StatusUnsupported
		decision.Error = &domain.ToolExecutionError{Code: domain.ToolErrAdapterUnavailable, Field: "adapterId", Message: "no adapter is available for capability"}
		decision.Reason = decision.Error.Message
		return decision
	}
	if in.Request.DryRun && !in.Capability.AllowedInDryRun {
		decision.Error = &domain.ToolExecutionError{Code: domain.ToolErrDryRunOnly, Field: "dryRun", Message: "capability does not support dry-run execution"}
		decision.Reason = decision.Error.Message
		return decision
	}
	if in.Capability.RequiresWorkspace {
		workspaceID := strings.TrimSpace(in.Request.WorkspaceID)
		if workspaceID == "" {
			decision.Error = &domain.ToolExecutionError{Code: domain.ToolErrScopeDenied, Field: "workspaceId", Message: "workspace scope is required"}
			decision.Reason = decision.Error.Message
			return decision
		}
	}
	if scopeDenied(in.ResolvedPaths, in.Capability.ResourceLimits.AllowedPaths, in.Capability.ResourceLimits.DeniedPaths) {
		decision.Error = &domain.ToolExecutionError{Code: domain.ToolErrScopeDenied, Field: "paths", Message: "requested paths violate capability resource limits"}
		decision.Reason = decision.Error.Message
		return decision
	}

	selfInitiated := isSelfInitiated(in.Request)
	if in.Capability.RequiresIntent || selfInitiated {
		if strings.TrimSpace(in.Request.IntentID) == "" {
			decision.Error = &domain.ToolExecutionError{Code: domain.ToolErrPolicyDenied, Field: "intentId", Message: "self-initiated tool calls require an intent"}
			decision.Reason = decision.Error.Message
			return decision
		}
		if e.autonomy != nil {
			autonomyDecision, err := e.autonomy.AuthorizeToolRequest(ctx, ToolAutonomyRequest{
				Request:    in.Request,
				Capability: in.Capability,
				Risk:       risk,
			})
			if err != nil {
				decision.Error = &domain.ToolExecutionError{Code: domain.ToolErrPolicyDenied, Field: "autonomy", Message: err.Error()}
				decision.Reason = decision.Error.Message
				return decision
			}
			decision.Warnings = append(decision.Warnings, autonomyDecision.Warnings...)
			if !autonomyDecision.Allowed && !autonomyDecision.RequiresApproval {
				decision.Error = &domain.ToolExecutionError{Code: domain.ToolErrPolicyDenied, Field: "autonomy", Message: nonEmpty(autonomyDecision.Reason, "autonomy policy denied tool request")}
				decision.Reason = decision.Error.Message
				return decision
			}
			if autonomyDecision.RequiresApproval {
				decision.Status = StatusNeedsApprov
				decision.RequiresApproval = true
				decision.Error = createToolExecutionError(domain.ToolErrApprovalRequired, "autonomy", "autonomy policy requires approval")
				decision.Reason = nonEmpty(autonomyDecision.Reason, "autonomy policy requires approval")
				return decision
			}
		} else if strings.TrimSpace(in.Request.CharterID) == "" || strings.TrimSpace(in.Request.BudgetID) == "" {
			decision.Status = StatusNeedsApprov
			decision.RequiresApproval = true
			decision.Error = createToolExecutionError(domain.ToolErrApprovalRequired, "autonomy", "self-initiated tool call is missing charter or budget context")
			decision.Reason = "self-initiated tool call is missing charter or budget context"
			return decision
		}
	}

	if in.Capability.Status == domain.ToolCapabilityApprovalOnly {
		decision.Status = StatusNeedsApprov
		decision.RequiresApproval = true
		decision.Error = createToolExecutionError(domain.ToolErrApprovalRequired, "status", "capability requires approval")
		decision.Reason = "tool risk or policy requires approval"
		return decision
	}
	if (risk.RequiresApproval || in.Capability.RequiresApprovalByDefault) && selfInitiated {
		decision.Status = StatusNeedsApprov
		decision.RequiresApproval = true
		decision.Error = createToolExecutionError(domain.ToolErrApprovalRequired, "risk", "self-initiated high-risk capability requires approval")
		decision.Reason = "self-initiated high-risk capability requires approval"
		return decision
	}

	decision.Allowed = true
	decision.Status = StatusOK
	decision.Reason = "tool policy allows execution"
	return decision
}

func scopeDenied(paths, allowed, denied []string) bool {
	if len(paths) == 0 {
		return false
	}
	for _, p := range paths {
		p = filepath.Clean(strings.TrimSpace(p))
		if p == "" {
			continue
		}
		for _, blocked := range denied {
			if pathWithinScope(p, blocked) {
				return true
			}
		}
		if len(allowed) > 0 {
			ok := false
			for _, allowedPath := range allowed {
				if pathWithinScope(p, allowedPath) {
					ok = true
					break
				}
			}
			if !ok {
				return true
			}
		}
	}
	return false
}

func pathWithinScope(path, scope string) bool {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return false
	}
	path = filepath.Clean(path)
	if !filepath.IsAbs(path) {
		if abs, err := filepath.Abs(path); err == nil {
			path = abs
		}
	}
	if !filepath.IsAbs(scope) {
		if abs, err := filepath.Abs(scope); err == nil {
			scope = abs
		}
	}
	scope = filepath.Clean(scope)
	rel, err := filepath.Rel(scope, path)
	if err != nil {
		return false
	}
	rel = filepath.Clean(rel)
	return rel == "." || !strings.HasPrefix(rel, "..")
}

func weakProvenance(req Request) bool {
	return strings.TrimSpace(req.ProvenanceActor) == "" || strings.TrimSpace(req.ProvenanceActorType) == ""
}

func isSelfInitiated(req Request) bool {
	source := strings.TrimSpace(strings.ToLower(req.Source))
	if source == "" {
		return false
	}
	switch source {
	case string(domain.SourceUser), "operator":
		return false
	case string(domain.SourceFutureIRIS), string(domain.SourceInternal), string(domain.SourceSystem), string(domain.SourceAdapter):
		return true
	default:
		return source != "user"
	}
}

func gatewayRiskClassFromToolRisk(risk domain.ToolRisk) string {
	switch risk {
	case domain.ToolRiskNone, domain.ToolRiskLow:
		return "read_only"
	case domain.ToolRiskMedium:
		return "safe_write"
	case domain.ToolRiskHigh:
		return "privileged"
	case domain.ToolRiskCritical:
		return "dangerous"
	default:
		return "scoped_execute"
	}
}

func toolExecutionErrMessage(err *domain.ToolExecutionError, fallback string) string {
	if err == nil {
		return fallback
	}
	if strings.TrimSpace(err.Message) != "" {
		return err.Message
	}
	return fallback
}

func createToolExecutionError(code domain.ToolExecutionErrorCode, field, message string) *domain.ToolExecutionError {
	return &domain.ToolExecutionError{
		Code:    code,
		Field:   strings.TrimSpace(field),
		Message: strings.TrimSpace(message),
	}
}

func explainToolPolicyDecision(decision ToolPolicyDecision) string {
	if decision.Allowed {
		return "allowed"
	}
	parts := []string{decision.Reason}
	if decision.Error != nil {
		parts = append(parts, fmt.Sprintf("code=%s", decision.Error.Code))
	}
	if len(decision.Warnings) > 0 {
		parts = append(parts, "warnings="+strings.Join(decision.Warnings, "; "))
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}
