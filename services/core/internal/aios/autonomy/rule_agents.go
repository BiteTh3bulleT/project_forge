package autonomy

import (
	"context"
	"fmt"
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/aios/truth"
)

type RuleAgent interface {
	ID() string
	Evaluate(ctx context.Context, in RuleAgentInput) (RuleAgentResult, error)
}

type RuleAgentInput struct {
	Scope         domain.ForgeScope
	CorrelationID string
	TraceID       string
	NowMillis     int64
	Truth         truth.OpenLoopLifecycleService
	Depth         int
	Trigger       string
}

type RuleAgentResult struct {
	AgentID  string
	Intent   domain.AutonomyIntent
	Actions  []domain.SyscallRequest
	Warnings []string
}

type RuleAgentEvaluation struct {
	AgentID string
	Result  RuleAgentResult
	Error   error
}

type RuleAgentRuntime struct {
	agents       []RuleAgent
	runner       *SelfInitiatedSyscallRunner
	autonomyMode domain.AutonomyMode
	nowMillis    func() int64
}

func NewRuleAgentRuntime(agents []RuleAgent, runner *SelfInitiatedSyscallRunner, autonomyMode domain.AutonomyMode, nowMillis func() int64) *RuleAgentRuntime {
	if nowMillis == nil {
		nowMillis = domain.NowMillis
	}
	if autonomyMode == "" {
		autonomyMode = domain.AutonomyModePropose
	}
	return &RuleAgentRuntime{agents: append([]RuleAgent{}, agents...), runner: runner, autonomyMode: autonomyMode, nowMillis: nowMillis}
}

func (r *RuleAgentRuntime) EvaluateOnce(ctx context.Context, in RuleAgentInput) []RuleAgentEvaluation {
	out := make([]RuleAgentEvaluation, 0, len(r.agents))
	for _, agent := range r.agents {
		res, err := agent.Evaluate(ctx, in)
		out = append(out, RuleAgentEvaluation{
			AgentID: agent.ID(),
			Result:  res,
			Error:   err,
		})
	}
	return out
}

func (r *RuleAgentRuntime) RunOnce(ctx context.Context, in RuleAgentInput) ([]domain.AutonomyRunSummary, error) {
	out := []domain.AutonomyRunSummary{}
	if r.runner == nil || len(r.agents) == 0 {
		return out, nil
	}
	for _, eval := range r.EvaluateOnce(ctx, in) {
		if eval.Error != nil {
			out = append(out, domain.AutonomyRunSummary{
				IntentID:      "",
				Decision:      domain.DecisionDeny,
				Warnings:      []string{"rule agent failed: " + eval.Error.Error()},
				Errors:        []domain.AutonomyError{{Code: domain.AutonomyErrInvalidInput, Field: eval.AgentID, Message: eval.Error.Error()}},
				CorrelationID: in.CorrelationID,
				TraceID:       in.TraceID,
			})
			continue
		}
		res := eval.Result
		if len(res.Actions) == 0 {
			continue
		}
		run, runErr := r.runner.Run(ctx, res.Intent, res.Actions, domain.RunModeProposeOnly, r.autonomyMode)
		run.Warnings = append(run.Warnings, res.Warnings...)
		run.Warnings = append(run.Warnings, "rule-agent runtime is propose-only; commits require explicit downstream approval path")
		out = append(out, run)
		if runErr != nil {
			continue
		}
	}
	return out, nil
}

func (r *RuleAgentRuntime) PlanOnce(ctx context.Context, in RuleAgentInput) ([]RuleAgentResult, []domain.AutonomyRunSummary) {
	plans := []RuleAgentResult{}
	diagnostics := []domain.AutonomyRunSummary{}
	for _, agent := range r.agents {
		res, err := agent.Evaluate(ctx, in)
		if err != nil {
			diagnostics = append(diagnostics, domain.AutonomyRunSummary{
				IntentID:      "",
				Decision:      domain.DecisionDeny,
				Warnings:      []string{"rule agent failed: " + err.Error()},
				Errors:        []domain.AutonomyError{{Code: domain.AutonomyErrInvalidInput, Field: agent.ID(), Message: err.Error()}},
				CorrelationID: in.CorrelationID,
				TraceID:       in.TraceID,
			})
			continue
		}
		if len(res.Actions) == 0 {
			continue
		}
		plans = append(plans, res)
	}
	return plans, diagnostics
}

type OpenLoopStalenessAgent struct {
	StaleCutoffMillis int64
}

func (OpenLoopStalenessAgent) ID() string { return "OpenLoopStalenessAgent" }

func (a OpenLoopStalenessAgent) Evaluate(ctx context.Context, in RuleAgentInput) (RuleAgentResult, error) {
	if in.Truth == nil {
		return RuleAgentResult{}, nil
	}
	cutoff := a.StaleCutoffMillis
	if cutoff <= 0 {
		cutoff = in.NowMillis - int64(72*60*60*1000)
	}
	loops, err := in.Truth.ListStaleLoops(ctx, in.Scope, cutoff, 20)
	if err != nil {
		return RuleAgentResult{}, err
	}
	if len(loops) == 0 {
		return RuleAgentResult{}, nil
	}
	loop := loops[0]
	intent := domain.AutonomyIntent{
		ID:            "intent-stale-loop-" + shortHash(loop.ID, fmt.Sprintf("%d", in.NowMillis)),
		Type:          domain.IntentStaleLoopReview,
		Title:         "Review stale open loop",
		Description:   "Create a stale-loop attention note for " + loop.ID,
		Source:        domain.IntentSourceRuleAgent,
		ProposedBy:    "rule_agent.open_loop_staleness",
		Scope:         in.Scope,
		Status:        domain.IntentStatusProposed,
		Risk:          domain.AutonomyRiskLow,
		AutonomyLevel: domain.AutonomyLevelAutoCommitSafe,
		CharterID:     "charter_open_loop_review",
		BudgetID:      "budget_memory_maintenance",
		Evidence:      []string{loop.ID},
		Provenance:    domain.Provenance{Actor: "rule_agent.open_loop_staleness", ActorType: "rule_agent", Source: "autonomy.rule_agent", TraceID: in.TraceID},
		CorrelationID: in.CorrelationID,
		TraceID:       in.TraceID,
		CreatedAt:     in.NowMillis,
		UpdatedAt:     in.NowMillis,
	}
	action := domain.SyscallRequest{
		ID:     "action-stale-loop-note-" + shortHash(loop.ID, fmt.Sprintf("%d", in.NowMillis)),
		Action: domain.ActionCreateNote,
		Actor:  domain.ActorIdentity{ID: "rule_agent.open_loop_staleness", Kind: "rule_agent"},
		Source: domain.SourceSystem,
		Scope:  in.Scope,
		Payload: map[string]any{
			"id":         "note-stale-loop-" + shortHash(loop.ID),
			"type":       string(domain.NoteSystem),
			"title":      "Stale loop warning: " + loop.Title,
			"content":    "Open loop " + loop.ID + " appears stale and needs review.",
			"confidence": 0.7,
			"status":     string(domain.NoteActive),
		},
		Provenance:    domain.Provenance{Actor: "rule_agent.open_loop_staleness", ActorType: "rule_agent", Source: "autonomy.rule_agent", TraceID: in.TraceID},
		CorrelationID: in.CorrelationID,
		TraceID:       in.TraceID,
		RequestedAt:   in.NowMillis,
	}
	return RuleAgentResult{AgentID: "OpenLoopStalenessAgent", Intent: intent, Actions: []domain.SyscallRequest{action}}, nil
}

type CleanupProposalAgent struct{}

func (CleanupProposalAgent) ID() string { return "CleanupProposalAgent" }

func (CleanupProposalAgent) Evaluate(_ context.Context, in RuleAgentInput) (RuleAgentResult, error) {
	intent := domain.AutonomyIntent{
		ID:            "intent-cleanup-" + shortHash(in.Scope.WorkspaceID, fmt.Sprintf("%d", in.NowMillis)),
		Type:          domain.IntentMemoryCleanup,
		Title:         "Cleanup review proposal",
		Description:   "Propose safe cleanup operation for review",
		Source:        domain.IntentSourceRuleAgent,
		ProposedBy:    "rule_agent.cleanup_proposal",
		Scope:         in.Scope,
		Status:        domain.IntentStatusProposed,
		Risk:          domain.AutonomyRiskMedium,
		AutonomyLevel: domain.AutonomyLevelProposeOnly,
		CharterID:     "charter_memory_maintenance",
		BudgetID:      "budget_memory_maintenance",
		Provenance:    domain.Provenance{Actor: "rule_agent.cleanup_proposal", ActorType: "rule_agent", Source: "autonomy.rule_agent", TraceID: in.TraceID},
		CorrelationID: in.CorrelationID,
		TraceID:       in.TraceID,
		CreatedAt:     in.NowMillis,
		UpdatedAt:     in.NowMillis,
	}
	return RuleAgentResult{
		AgentID:  "CleanupProposalAgent",
		Intent:   intent,
		Actions:  nil,
		Warnings: []string{"cleanup proposal currently has no deterministic cleanup target; proposals are safe-only pending stronger signal"},
	}, nil
}

func BuildRuleAgentInput(scope domain.ForgeScope, correlationID, traceID string, engine truth.OpenLoopLifecycleService, depth int, trigger string, now int64) RuleAgentInput {
	if strings.TrimSpace(correlationID) == "" {
		correlationID = "corr-" + shortHash(scope.WorkspaceID, fmt.Sprintf("%d", now))
	}
	if strings.TrimSpace(traceID) == "" {
		traceID = correlationID
	}
	return RuleAgentInput{
		Scope:         scope,
		CorrelationID: correlationID,
		TraceID:       traceID,
		NowMillis:     now,
		Truth:         engine,
		Depth:         depth,
		Trigger:       trigger,
	}
}
