package autonomy

import (
	"context"
	"fmt"
	"strings"

	"forge/projectforge/services/core/internal/aios/controllane"
	"forge/projectforge/services/core/internal/aios/domain"
)

type SelfInitiatedRunnerOptions struct {
	Kernel    controllane.ForgeKernelProcessor
	Policy    *AutonomyPolicyEvaluator
	Intents   *IntentQueueService
	Decisions DecisionRepository
	Budgets   *FreedomBudgetService
	Approval  ApprovalGate
	NowMillis func() int64
}

type SelfInitiatedSyscallRunner struct {
	kernel    controllane.ForgeKernelProcessor
	policy    *AutonomyPolicyEvaluator
	intents   *IntentQueueService
	decisions DecisionRepository
	budgets   *FreedomBudgetService
	approval  ApprovalGate
	nowMillis func() int64
}

func NewSelfInitiatedSyscallRunner(opts SelfInitiatedRunnerOptions) *SelfInitiatedSyscallRunner {
	nowFn := opts.NowMillis
	if nowFn == nil {
		nowFn = domain.NowMillis
	}
	approval := opts.Approval
	if approval == nil {
		defaultApproval := NewStaticApprovalEscalator()
		approval = defaultApproval
	}
	return &SelfInitiatedSyscallRunner{
		kernel:    opts.Kernel,
		policy:    opts.Policy,
		intents:   opts.Intents,
		decisions: opts.Decisions,
		budgets:   opts.Budgets,
		approval:  approval,
		nowMillis: nowFn,
	}
}

func (r *SelfInitiatedSyscallRunner) Run(ctx context.Context, intent domain.AutonomyIntent, actions []domain.SyscallRequest, mode domain.AutonomyRunMode, autonomyMode domain.AutonomyMode) (domain.AutonomyRunSummary, error) {
	summary := domain.AutonomyRunSummary{
		IntentID:           intent.ID,
		Decision:           domain.DecisionDeny,
		CommittedObjectIDs: []string{},
		CommittedActions:   []domain.SyscallResult{},
		Approval:           domain.ApprovalEscalationResult{Status: domain.ApprovalAllowed},
		Warnings:           []string{},
		Errors:             []domain.AutonomyError{},
		CorrelationID:      intent.CorrelationID,
		TraceID:            intent.TraceID,
	}
	if r.policy == nil {
		summary.Errors = append(summary.Errors, domain.AutonomyError{Code: domain.AutonomyErrInvalidInput, Field: "policy", Message: "autonomy policy evaluator is required"})
		return summary, summary.Error()
	}
	if r.intents == nil {
		summary.Errors = append(summary.Errors, domain.AutonomyError{Code: domain.AutonomyErrInvalidInput, Field: "intents", Message: "intent queue service is required"})
		return summary, summary.Error()
	}
	if strings.TrimSpace(intent.ID) == "" {
		intent.ID = "intent-" + shortHash(string(intent.Type), fmt.Sprintf("%d", r.nowMillis()))
	}
	if strings.TrimSpace(intent.Scope.WorkspaceID) == "" {
		summary.Errors = append(summary.Errors, domain.AutonomyError{Code: domain.AutonomyErrInvalidScope, Field: "intent.scope.workspaceId", Message: "intent scope.workspaceId is required"})
		return summary, summary.Error()
	}
	if intent.CreatedAt <= 0 {
		intent.CreatedAt = r.nowMillis()
	}
	intent.UpdatedAt = intent.CreatedAt
	if strings.TrimSpace(string(intent.Status)) == "" {
		intent.Status = domain.IntentStatusProposed
	}
	intent.ProposedActions = append([]domain.SyscallRequest{}, actions...)

	if _, ok, err := r.intents.Get(ctx, intent.ID); err != nil {
		summary.Errors = append(summary.Errors, domain.AutonomyError{Code: domain.AutonomyErrInvalidInput, Field: "intent", Message: err.Error()})
		return summary, summary.Error()
	} else if !ok {
		if _, err := r.intents.Enqueue(ctx, intent); err != nil {
			summary.Errors = append(summary.Errors, domain.AutonomyError{Code: domain.AutonomyErrInvalidInput, Field: "intent", Message: err.Error()})
			return summary, summary.Error()
		}
	}

	eval, err := r.policy.Evaluate(ctx, EvaluationInput{Intent: intent, Actions: actions, Mode: autonomyMode, PreviewOnly: mode != domain.RunModeCommitIfAuthorized})
	if err != nil {
		summary.Errors = append(summary.Errors, domain.AutonomyError{Code: domain.AutonomyErrInvalidInput, Field: "policy", Message: err.Error()})
		_ = r.intents.MarkBlocked(ctx, intent.ID, "policy evaluation failed")
		return summary, summary.Error()
	}
	summary.DecisionID = eval.ID
	summary.Decision = eval.Decision
	if r.decisions != nil {
		_ = r.decisions.Create(ctx, eval)
	}

	if mode == domain.RunModeExplainOnly || mode == domain.RunModeProposeOnly {
		summary.Warnings = append(summary.Warnings, "runner mode does not allow commit")
		if eval.Decision == domain.DecisionAllowAutoCommit {
			eval.Decision = domain.DecisionAllowProposeOnly
			summary.Decision = eval.Decision
		}
		return summary, nil
	}

	if eval.Decision == domain.DecisionApprovalRequired {
		approval, err := r.approval.RequestApproval(ctx, intent, eval)
		if err != nil {
			summary.Errors = append(summary.Errors, domain.AutonomyError{Code: domain.AutonomyErrApprovalRequired, Field: "approval", Message: err.Error()})
			_ = r.intents.MarkBlocked(ctx, intent.ID, "approval escalation failed")
			return summary, summary.Error()
		}
		summary.Approval = approval
		intent.RequiredApproval = true
		intent.ApprovalID = approval.ApprovalID
		if approval.Status == domain.ApprovalAllowed {
			if err := r.intents.Approve(ctx, intent.ID); err != nil {
				summary.Errors = append(summary.Errors, domain.AutonomyError{Code: domain.AutonomyErrInvalidTransition, Field: "intent.status", Message: err.Error()})
				return summary, summary.Error()
			}
		} else if approval.Status == domain.ApprovalDenied {
			_ = r.intents.Reject(ctx, intent.ID, nonEmpty(approval.Reason, "autonomy approval denied"))
			return summary, nil
		} else {
			_ = r.intents.MarkBlocked(ctx, intent.ID, nonEmpty(approval.Reason, "approval required"))
			return summary, nil
		}
	}

	switch eval.Decision {
	case domain.DecisionAllowAutoCommit:
		if mode == domain.RunModeValidateOnly {
			results := r.validateOnly(ctx, intent, eval.AllowedActions)
			summary.CommittedActions = results
			return summary, nil
		}
		if err := r.intents.Approve(ctx, intent.ID); err != nil {
			summary.Errors = append(summary.Errors, domain.AutonomyError{Code: domain.AutonomyErrInvalidTransition, Field: "intent.status", Message: err.Error()})
			return summary, summary.Error()
		}
		if err := r.intents.MarkRunning(ctx, intent.ID); err != nil {
			summary.Errors = append(summary.Errors, domain.AutonomyError{Code: domain.AutonomyErrInvalidTransition, Field: "intent.status", Message: err.Error()})
			return summary, summary.Error()
		}

		reservationID := ""
		if r.budgets != nil && strings.TrimSpace(eval.BudgetID) != "" {
			reservation, err := r.budgets.ReserveBudget(ctx, eval.BudgetID, intent.ID, intent.Scope, string(intent.Type), len(eval.AllowedActions), map[string]any{"decisionId": eval.ID})
			if err != nil {
				summary.Errors = append(summary.Errors, domain.AutonomyError{Code: domain.AutonomyErrBudgetDenied, Field: "budget", Message: err.Error()})
				_ = r.intents.MarkBlocked(ctx, intent.ID, "budget reservation failed")
				return summary, summary.Error()
			}
			eval.BudgetReservationID = reservation.ID
			reservationID = reservation.ID
		}

		results, committedIDs, commitErrs := r.commitAllowedActions(ctx, intent, eval)
		summary.CommittedActions = results
		summary.CommittedObjectIDs = append(summary.CommittedObjectIDs, committedIDs...)
		if len(commitErrs) > 0 {
			summary.Errors = append(summary.Errors, commitErrs...)
			if reservationID != "" {
				_ = r.budgets.ReleaseBudget(ctx, reservationID, "syscall failure")
			}
			_ = r.intents.MarkBlocked(ctx, intent.ID, "commit failed")
			return summary, summary.Error()
		}
		if reservationID != "" {
			if _, err := r.budgets.ConsumeBudget(ctx, reservationID); err != nil {
				summary.Errors = append(summary.Errors, domain.AutonomyError{Code: domain.AutonomyErrBudgetDenied, Field: "budget", Message: err.Error()})
				_ = r.intents.MarkBlocked(ctx, intent.ID, "budget consume failed")
				return summary, summary.Error()
			}
		}
		if err := r.intents.MarkCompleted(ctx, intent.ID, results); err != nil {
			summary.Errors = append(summary.Errors, domain.AutonomyError{Code: domain.AutonomyErrInvalidTransition, Field: "intent.status", Message: err.Error()})
			return summary, summary.Error()
		}
		return summary, nil
	case domain.DecisionAllowProposeOnly:
		summary.Warnings = append(summary.Warnings, "policy returned propose_only")
		return summary, nil
	case domain.DecisionBlockedByBudget, domain.DecisionBlockedByCharter, domain.DecisionBlockedByKernel, domain.DecisionBlockedByRisk, domain.DecisionBlockedByScope:
		_ = r.intents.MarkBlocked(ctx, intent.ID, nonEmpty(strings.Join(eval.DeniedReasons, "; "), "autonomy policy blocked commit"))
		return summary, nil
	case domain.DecisionDeny:
		_ = r.intents.Reject(ctx, intent.ID, nonEmpty(strings.Join(eval.DeniedReasons, "; "), "autonomy denied"))
		return summary, nil
	default:
		_ = r.intents.MarkBlocked(ctx, intent.ID, "unsupported policy decision")
		return summary, nil
	}
}

func (r *SelfInitiatedSyscallRunner) Preview(ctx context.Context, intent domain.AutonomyIntent, actions []domain.SyscallRequest, autonomyMode domain.AutonomyMode) (domain.AutonomyRunSummary, error) {
	summary := domain.AutonomyRunSummary{
		IntentID:           strings.TrimSpace(intent.ID),
		Decision:           domain.DecisionDeny,
		CommittedObjectIDs: []string{},
		CommittedActions:   []domain.SyscallResult{},
		Approval:           domain.ApprovalEscalationResult{Status: domain.ApprovalAllowed},
		Warnings:           []string{},
		Errors:             []domain.AutonomyError{},
		CorrelationID:      intent.CorrelationID,
		TraceID:            intent.TraceID,
	}
	if r.policy == nil {
		summary.Errors = append(summary.Errors, domain.AutonomyError{Code: domain.AutonomyErrInvalidInput, Field: "policy", Message: "autonomy policy evaluator is required"})
		return summary, summary.Error()
	}
	if strings.TrimSpace(intent.Scope.WorkspaceID) == "" {
		summary.Errors = append(summary.Errors, domain.AutonomyError{Code: domain.AutonomyErrInvalidScope, Field: "intent.scope.workspaceId", Message: "intent scope.workspaceId is required"})
		return summary, summary.Error()
	}
	eval, err := r.policy.Evaluate(ctx, EvaluationInput{
		Intent:      intent,
		Actions:     actions,
		Mode:        autonomyMode,
		PreviewOnly: true,
	})
	if err != nil {
		summary.Errors = append(summary.Errors, domain.AutonomyError{Code: domain.AutonomyErrInvalidInput, Field: "policy", Message: err.Error()})
		return summary, summary.Error()
	}
	summary.DecisionID = eval.ID
	summary.Decision = eval.Decision
	if len(eval.DeniedReasons) > 0 {
		summary.Warnings = append(summary.Warnings, strings.Join(eval.DeniedReasons, "; "))
	}
	switch eval.Decision {
	case domain.DecisionAllowAutoCommit:
		summary.Warnings = append(summary.Warnings, "preview only: commit suppressed")
	case domain.DecisionApprovalRequired:
		summary.Warnings = append(summary.Warnings, "preview only: approval would be required before commit")
	case domain.DecisionAllowProposeOnly:
		summary.Warnings = append(summary.Warnings, "preview only: propose_only decision")
	}
	return summary, nil
}

func (r *SelfInitiatedSyscallRunner) validateOnly(ctx context.Context, intent domain.AutonomyIntent, actions []domain.SyscallRequest) []domain.SyscallResult {
	out := make([]domain.SyscallResult, 0, len(actions))
	for _, action := range actions {
		call := annotateSelfAction(action, intent, "validate")
		call.DryRun = true
		res, err := r.kernel.Process(ctx, call)
		if err != nil {
			res = domain.SyscallResult{
				Success:              false,
				Action:               call.Action,
				RequestID:            call.ID,
				ApprovalStatus:       domain.ApprovalAllowed,
				RejectedReasons:      []domain.SyscallError{{Code: domain.ErrInternal, Field: "kernel", Message: err.Error()}},
				DeterministicErrCode: domain.ErrInternal,
			}
		}
		out = append(out, res)
	}
	return out
}

func (r *SelfInitiatedSyscallRunner) commitAllowedActions(ctx context.Context, intent domain.AutonomyIntent, decision domain.AutonomyDecision) ([]domain.SyscallResult, []string, []domain.AutonomyError) {
	results := make([]domain.SyscallResult, 0, len(decision.AllowedActions))
	committedIDs := []string{}
	errs := []domain.AutonomyError{}
	for _, action := range decision.AllowedActions {
		if guardErr := validateAutonomyCommitAction(intent, action); guardErr != nil {
			errs = append(errs, *guardErr)
			continue
		}
		call := annotateSelfAction(action, intent, decision.ID)
		call.DryRun = false
		res, err := r.kernel.Process(ctx, call)
		if err != nil {
			errs = append(errs, domain.AutonomyError{Code: domain.AutonomyErrKernelBlocked, Field: string(call.Action), Message: err.Error()})
			continue
		}
		results = append(results, res)
		if !res.Success {
			errs = append(errs, domain.AutonomyError{Code: domain.AutonomyErrKernelBlocked, Field: string(call.Action), Message: summarizeSyscallFailure(res)})
			continue
		}
		committedIDs = append(committedIDs, res.CommittedObjectIDs...)
	}
	return results, uniqueStrings(committedIDs), errs
}

func annotateSelfAction(action domain.SyscallRequest, intent domain.AutonomyIntent, decisionID string) domain.SyscallRequest {
	next := action
	if strings.TrimSpace(next.ID) == "" {
		next.ID = "autonomy-action-" + shortHash(intent.ID, string(next.Action), decisionID)
	}
	if strings.TrimSpace(next.Actor.ID) == "" {
		next.Actor.ID = "forge.autonomy"
	}
	if strings.TrimSpace(next.Actor.Kind) == "" {
		next.Actor.Kind = "autonomy"
	}
	if strings.TrimSpace(string(next.Source)) == "" {
		next.Source = domain.SourceSystem
	}
	if strings.TrimSpace(next.Scope.WorkspaceID) == "" {
		next.Scope = intent.Scope
	}
	if strings.TrimSpace(next.CorrelationID) == "" {
		next.CorrelationID = intent.CorrelationID
	}
	if strings.TrimSpace(next.TraceID) == "" {
		next.TraceID = intent.TraceID
	}
	if next.RequestedAt <= 0 {
		next.RequestedAt = domain.NowMillis()
	}
	if strings.TrimSpace(next.Provenance.Actor) == "" {
		next.Provenance.Actor = nonEmpty(intent.Provenance.Actor, "forge.autonomy")
	}
	if strings.TrimSpace(next.Provenance.ActorType) == "" {
		next.Provenance.ActorType = nonEmpty(intent.Provenance.ActorType, "autonomy")
	}
	if strings.TrimSpace(next.Provenance.Source) == "" {
		next.Provenance.Source = nonEmpty(intent.Provenance.Source, "autonomy.runner")
	}
	if strings.TrimSpace(next.Provenance.TraceID) == "" {
		next.Provenance.TraceID = nonEmpty(next.TraceID, intent.TraceID)
	}
	next.Metadata = mergeMeta(next.Metadata, map[string]any{
		"autonomy":   true,
		"intentId":   intent.ID,
		"charterId":  intent.CharterID,
		"budgetId":   intent.BudgetID,
		"decisionId": decisionID,
		"source":     intent.Source,
	})
	if strings.TrimSpace(next.IdempotencyKey) == "" {
		next.IdempotencyKey = "autonomy:" + intent.ID + ":" + shortHash(string(next.Action), next.ID)
	}
	return next
}

func summarizeSyscallFailure(res domain.SyscallResult) string {
	if len(res.RejectedReasons) > 0 {
		first := res.RejectedReasons[0]
		if strings.TrimSpace(first.Field) != "" {
			return fmt.Sprintf("%s (%s): %s", first.Code, first.Field, first.Message)
		}
		return fmt.Sprintf("%s: %s", first.Code, first.Message)
	}
	if res.DeterministicErrCode != "" {
		return string(res.DeterministicErrCode)
	}
	return "kernel rejected action"
}

func nonEmpty(v, fallback string) string {
	if strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return strings.TrimSpace(fallback)
}

func uniqueStrings(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, item := range in {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, exists := seen[item]; exists {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func validateAutonomyCommitAction(intent domain.AutonomyIntent, action domain.SyscallRequest) *domain.AutonomyError {
	if intent.Source == domain.IntentSourceRuleAgent && isDestructiveAutonomyAction(action.Action) {
		err := domain.AutonomyError{
			Code:    domain.AutonomyErrKernelBlocked,
			Field:   string(action.Action),
			Message: "rule-agent intents cannot directly commit destructive actions",
		}
		return &err
	}
	if action.Action == domain.ActionArchiveNote && hasPlaceholderArchiveTarget(action.Payload) {
		err := domain.AutonomyError{
			Code:    domain.AutonomyErrKernelBlocked,
			Field:   "payload.noteId",
			Message: "cleanup placeholder target cannot be committed",
		}
		return &err
	}
	return nil
}

func isDestructiveAutonomyAction(action domain.SemanticActionType) bool {
	name := strings.ToUpper(strings.TrimSpace(string(action)))
	if name == "" {
		return false
	}
	switch action {
	case domain.ActionArchiveNote:
		return true
	}
	for _, token := range []string{"DELETE", "DESTROY", "PURGE", "RESTORE", "KILL"} {
		if strings.Contains(name, token) {
			return true
		}
	}
	return false
}

func hasPlaceholderArchiveTarget(payload map[string]any) bool {
	if len(payload) == 0 {
		return false
	}
	ids := []string{
		fmt.Sprintf("%v", payload["noteId"]),
		fmt.Sprintf("%v", payload["id"]),
		fmt.Sprintf("%v", payload["targetId"]),
		fmt.Sprintf("%v", payload["objectId"]),
	}
	for _, raw := range ids {
		id := strings.ToLower(strings.TrimSpace(raw))
		if id == "" {
			continue
		}
		if id == "candidate-note" || strings.HasPrefix(id, "candidate-") || strings.HasPrefix(id, "fake-") || strings.HasPrefix(id, "placeholder-") {
			return true
		}
	}
	return false
}
