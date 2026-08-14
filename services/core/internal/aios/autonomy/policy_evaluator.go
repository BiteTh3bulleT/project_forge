package autonomy

import (
	"context"
	"fmt"
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/forgekernel"
)

type PolicyEvaluatorOptions struct {
	Charters                 CharterRepository
	Risk                     RiskClassifier
	Budgets                  *FreedomBudgetService
	Kernel                   forgekernel.Processor
	NowMillis                func() int64
	CharterService           CharterService
	HasActiveStateDependency func(scope domain.ForgeScope, objectID string) bool
}

type EvaluationInput struct {
	Intent      domain.AutonomyIntent
	Actions     []domain.SyscallRequest
	Mode        domain.AutonomyMode
	PreviewOnly bool
}

type AutonomyPolicyEvaluator struct {
	charters       CharterRepository
	risk           RiskClassifier
	budgets        *FreedomBudgetService
	kernel         forgekernel.Processor
	nowMillis      func() int64
	charterService CharterService
	hasStateDep    func(scope domain.ForgeScope, objectID string) bool
}

func NewAutonomyPolicyEvaluator(opts PolicyEvaluatorOptions) *AutonomyPolicyEvaluator {
	nowFn := opts.NowMillis
	if nowFn == nil {
		nowFn = domain.NowMillis
	}
	risk := opts.Risk
	if risk == nil {
		defaultRisk := NewDeterministicRiskClassifier()
		risk = defaultRisk
	}
	return &AutonomyPolicyEvaluator{
		charters:       opts.Charters,
		risk:           risk,
		budgets:        opts.Budgets,
		kernel:         opts.Kernel,
		nowMillis:      nowFn,
		charterService: opts.CharterService,
		hasStateDep:    opts.HasActiveStateDependency,
	}
}

func (e *AutonomyPolicyEvaluator) Evaluate(ctx context.Context, in EvaluationInput) (domain.AutonomyDecision, error) {
	now := e.nowMillis()
	decision := domain.AutonomyDecision{
		ID:             "decision-" + shortHash(in.Intent.ID, fmt.Sprintf("%d", now)),
		IntentID:       in.Intent.ID,
		Decision:       domain.DecisionDeny,
		AutonomyLevel:  in.Intent.AutonomyLevel,
		Risk:           domain.AutonomyRiskNone,
		DeniedReasons:  []string{},
		Warnings:       []string{},
		AllowedActions: []domain.SyscallRequest{},
		BlockedActions: []domain.SyscallRequest{},
		KernelPreview:  []domain.PolicyKernelPreview{},
		CorrelationID:  in.Intent.CorrelationID,
		TraceID:        in.Intent.TraceID,
		CreatedAt:      now,
		Metadata:       map[string]any{},
	}

	if errs := in.Intent.Validate(); len(errs) > 0 {
		decision.Decision = domain.DecisionDeny
		decision.DeniedReasons = append(decision.DeniedReasons, errs[0].Error())
		decision.Explanation = "invalid intent envelope"
		return decision, nil
	}
	if strings.TrimSpace(in.Intent.Scope.WorkspaceID) == "" {
		decision.Decision = domain.DecisionBlockedByScope
		decision.DeniedReasons = append(decision.DeniedReasons, "intent scope is required")
		decision.Explanation = "scope validation failed"
		return decision, nil
	}
	if len(in.Actions) == 0 {
		decision.Decision = domain.DecisionAllowProposeOnly
		decision.AutonomyLevel = domain.AutonomyLevelObserveOnly
		decision.Explanation = "no actions provided"
		return decision, nil
	}

	for _, action := range in.Actions {
		if !scopeMatch(in.Intent.Scope, action.Scope) {
			decision.Decision = domain.DecisionBlockedByScope
			decision.DeniedReasons = append(decision.DeniedReasons, "action scope does not match intent scope")
			decision.BlockedActions = append(decision.BlockedActions, action)
		}
	}
	if decision.Decision == domain.DecisionBlockedByScope {
		decision.Explanation = "scope mismatch blocks autonomous commit"
		return decision, nil
	}

	riskSummary := e.risk.ClassifyIntent(in.Intent, in.Actions)
	decision.Risk = riskSummary.MaxRisk
	decision.AutonomyLevel = recommendedLevelForRisk(riskSummary.MaxRisk)
	if riskSummary.ContainsGuardrail {
		decision.Warnings = append(decision.Warnings, riskSummary.GuardrailViolations...)
	}

	if in.Mode == domain.AutonomyModeOff {
		decision.Decision = domain.DecisionAllowProposeOnly
		decision.AutonomyLevel = domain.AutonomyLevelObserveOnly
		decision.DeniedReasons = append(decision.DeniedReasons, "autonomy mode is off")
		decision.Explanation = "self-initiated commits are disabled"
		decision.AllowedActions = append(decision.AllowedActions, in.Actions...)
		return decision, nil
	}
	if in.Mode == domain.AutonomyModeObserve || in.Mode == domain.AutonomyModePropose {
		decision.Decision = domain.DecisionAllowProposeOnly
		decision.AllowedActions = append(decision.AllowedActions, in.Actions...)
		decision.Explanation = "mode allows intent/proposal recording without commits"
		return decision, nil
	}
	if strings.TrimSpace(in.Intent.CharterID) == "" {
		decision.AllowedActions = append(decision.AllowedActions, in.Actions...)
		if riskSummary.ApprovalRequired {
			decision.Decision = domain.DecisionApprovalRequired
			decision.RequiredApprovalReason = "no charter specified and action risk requires approval"
			decision.Explanation = "missing charter for autonomous commit"
		} else {
			decision.Decision = domain.DecisionAllowProposeOnly
			decision.Explanation = "missing charter; proposal-only permitted"
		}
		return decision, nil
	}
	if strings.TrimSpace(in.Intent.CharterID) != "" && e.charters != nil {
		explicit, ok, err := e.charters.GetByID(ctx, in.Intent.CharterID)
		if err != nil {
			return decision, err
		}
		if !ok {
			decision.Decision = domain.DecisionBlockedByCharter
			decision.DeniedReasons = append(decision.DeniedReasons, "specified charter not found")
			decision.Explanation = "explicit charter is required but missing"
			decision.BlockedActions = append(decision.BlockedActions, in.Actions...)
			return decision, nil
		}
		if !explicit.IsActive(now) {
			decision.Decision = domain.DecisionBlockedByCharter
			decision.DeniedReasons = append(decision.DeniedReasons, "specified charter is inactive")
			decision.Explanation = "explicit charter is not active"
			decision.CharterID = explicit.ID
			decision.BlockedActions = append(decision.BlockedActions, in.Actions...)
			return decision, nil
		}
	}

	charter, hasCharter, charterErr := e.resolveCharter(ctx, in.Intent, in.Actions, now)
	if charterErr != nil {
		return decision, charterErr
	}
	if !hasCharter {
		decision.Decision = domain.DecisionBlockedByCharter
		decision.DeniedReasons = append(decision.DeniedReasons, "specified charter is not applicable")
		decision.Explanation = "explicit charter cannot authorize requested actions"
		decision.BlockedActions = append(decision.BlockedActions, in.Actions...)
		return decision, nil
	}
	decision.CharterID = charter.ID
	if strings.TrimSpace(in.Intent.CharterID) == "" {
		in.Intent.CharterID = charter.ID
	}

	if in.Mode == domain.AutonomyModeMission && !isMissionCharter(charter) {
		decision.Decision = domain.DecisionBlockedByCharter
		decision.DeniedReasons = append(decision.DeniedReasons, "mission mode requires mission charter")
		decision.Explanation = "active charter is not mission-class"
		decision.BlockedActions = append(decision.BlockedActions, in.Actions...)
		return decision, nil
	}

	approvalRequired := false
	for _, action := range in.Actions {
		classification := e.risk.ClassifyAction(in.Intent, action)
		eval := e.charterService.EvaluateAction(charter, classification.Risk, CharterConditionContext{
			Action:    action,
			Intent:    in.Intent,
			NowMillis: now,
			HasActiveStateDependency: func(objectID string) bool {
				if e.hasStateDep == nil {
					return false
				}
				return e.hasStateDep(in.Intent.Scope, objectID)
			},
		})
		decision.Warnings = append(decision.Warnings, eval.Warnings...)
		if !eval.Allowed {
			decision.BlockedActions = append(decision.BlockedActions, action)
			decision.DeniedReasons = append(decision.DeniedReasons, eval.Reasons...)
			continue
		}
		decision.AllowedActions = append(decision.AllowedActions, action)
		if eval.ApprovalRequired || classification.RequiresApproval {
			approvalRequired = true
		}
	}

	if len(decision.AllowedActions) == 0 {
		decision.Decision = domain.DecisionBlockedByCharter
		if len(decision.DeniedReasons) == 0 {
			decision.DeniedReasons = append(decision.DeniedReasons, "all actions blocked by charter")
		}
		decision.Explanation = "charter evaluation blocked all actions"
		return decision, nil
	}

	if approvalRequired || riskSummary.ApprovalRequired {
		decision.Decision = domain.DecisionApprovalRequired
		decision.RequiredApprovalReason = "risk level or charter policy requires approval"
		decision.Explanation = "approval required before commit"
		return decision, nil
	}

	if e.kernel != nil {
		preview, kernelBlocked := e.previewKernel(ctx, in.Intent, decision.AllowedActions)
		decision.KernelPreview = preview
		if kernelBlocked {
			decision.Decision = domain.DecisionBlockedByKernel
			decision.Explanation = "kernel dry-run validation rejected autonomous actions"
			return decision, nil
		}
	}

	if e.budgets != nil {
		budgetReq := BudgetCheckRequest{
			Scope:        in.Intent.Scope,
			BudgetID:     chooseBudgetID(in.Intent, charter),
			IntentID:     in.Intent.ID,
			RequestedFor: string(in.Intent.Type),
			Actions:      decision.AllowedActions,
			DryRun:       in.PreviewOnly,
			Mode:         in.Mode,
		}
		budgetRes, err := e.budgets.CheckBudget(ctx, budgetReq)
		if err != nil {
			return decision, err
		}
		if budgetRes.Budget != nil {
			decision.BudgetID = budgetRes.Budget.ID
		}
		if !budgetRes.Allowed {
			decision.Decision = domain.DecisionBlockedByBudget
			decision.DeniedReasons = append(decision.DeniedReasons, budgetRes.Reasons...)
			decision.Explanation = "budget limits block autonomous commit"
			return decision, nil
		}
	}

	if SupportsSelfCommit(in.Mode) && !e.hasDurableSelfCommitBacking() {
		decision.Decision = domain.DecisionAllowProposeOnly
		decision.DeniedReasons = append(decision.DeniedReasons, "durable charter+budget backing required for maintain/mission auto-commit")
		decision.Warnings = append(decision.Warnings, "autonomy self-commit is quarantined while autonomy repositories are in-memory")
		decision.Explanation = "persistence gate blocks auto-commit in maintain/mission modes"
		return decision, nil
	}

	decision.Decision = domain.DecisionAllowAutoCommit
	decision.Explanation = "charter, kernel preview, and budget authorize auto-commit"
	return decision, nil
}

func (e *AutonomyPolicyEvaluator) resolveCharter(ctx context.Context, intent domain.AutonomyIntent, actions []domain.SyscallRequest, now int64) (domain.AutonomyCharter, bool, error) {
	if e.charters == nil {
		return domain.AutonomyCharter{}, false, nil
	}
	if strings.TrimSpace(intent.CharterID) == "" || len(actions) == 0 {
		return domain.AutonomyCharter{}, false, nil
	}
	charter, ok, err := e.charters.GetByID(ctx, strings.TrimSpace(intent.CharterID))
	if err != nil {
		return domain.AutonomyCharter{}, false, err
	}
	if !ok || !charter.IsActive(now) {
		return domain.AutonomyCharter{}, false, nil
	}
	return charter, true, nil
}

func (e *AutonomyPolicyEvaluator) previewKernel(ctx context.Context, intent domain.AutonomyIntent, actions []domain.SyscallRequest) ([]domain.PolicyKernelPreview, bool) {
	out := make([]domain.PolicyKernelPreview, 0, len(actions))
	blocked := false
	for _, action := range actions {
		previewReq := action
		previewReq.DryRun = true
		previewReq.ID = action.ID + ":autonomy_preview"
		previewReq.Metadata = mergeMeta(previewReq.Metadata, map[string]any{
			"autonomyPreview": true,
			"intentId":        intent.ID,
		})
		res, err := e.kernel.Process(ctx, previewReq)
		if err != nil {
			blocked = true
			out = append(out, domain.PolicyKernelPreview{
				Action: previewReq,
				Result: domain.SyscallResult{
					Success:              false,
					Action:               previewReq.Action,
					RequestID:            previewReq.ID,
					ApprovalStatus:       domain.ApprovalAllowed,
					RejectedReasons:      []domain.SyscallError{{Code: domain.ErrInternal, Field: "kernel", Message: err.Error()}},
					DeterministicErrCode: domain.ErrInternal,
				},
				Blocked: true,
			})
			continue
		}
		if !res.Success {
			blocked = true
		}
		out = append(out, domain.PolicyKernelPreview{Action: previewReq, Result: res, Blocked: !res.Success})
	}
	return out, blocked
}

func chooseBudgetID(intent domain.AutonomyIntent, charter domain.AutonomyCharter) string {
	if strings.TrimSpace(intent.BudgetID) != "" {
		return strings.TrimSpace(intent.BudgetID)
	}
	return strings.TrimSpace(charter.FreedomBudgetID)
}

func isMissionCharter(charter domain.AutonomyCharter) bool {
	if v, ok := charter.Metadata["mission"]; ok {
		flag := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", v)))
		if flag == "true" || flag == "1" || flag == "yes" {
			return true
		}
	}
	name := strings.ToLower(strings.TrimSpace(charter.Name + " " + charter.Purpose))
	return strings.Contains(name, "mission")
}

func (e *AutonomyPolicyEvaluator) hasDurableSelfCommitBacking() bool {
	if e == nil || e.charters == nil || e.budgets == nil {
		return false
	}
	if isInMemoryCharterRepository(e.charters) {
		return false
	}
	return e.budgets.HasDurableBacking()
}

func isInMemoryCharterRepository(repo CharterRepository) bool {
	switch repo.(type) {
	case *InMemoryCharterRepository:
		return true
	default:
		return false
	}
}
