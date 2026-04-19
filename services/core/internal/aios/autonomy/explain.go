package autonomy

import (
	"context"
	"sort"

	"forge/projectforge/services/core/internal/aios/domain"
)

type ExplanationService struct {
	intents   IntentRepository
	decisions DecisionRepository
	budgets   BudgetRepository
	charters  CharterRepository
}

func NewExplanationService(intents IntentRepository, decisions DecisionRepository, budgets BudgetRepository, charters CharterRepository) *ExplanationService {
	return &ExplanationService{intents: intents, decisions: decisions, budgets: budgets, charters: charters}
}

func (s *ExplanationService) ExplainIntent(ctx context.Context, intentID string) (map[string]any, error) {
	if s.intents == nil {
		return map[string]any{"intentId": intentID, "warning": "intent repository is not configured"}, nil
	}
	intent, ok, err := s.intents.GetByID(ctx, intentID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, domain.AutonomyError{Code: domain.AutonomyErrNotFound, Field: "intentId", Message: "intent not found"}
	}
	d := []domain.AutonomyDecision{}
	if s.decisions != nil {
		d, _ = s.decisions.ListByIntent(ctx, intentID, 50)
	}
	sort.SliceStable(d, func(i, j int) bool { return d[i].CreatedAt < d[j].CreatedAt })
	return map[string]any{
		"intent":    intent,
		"decisions": d,
	}, nil
}

func (s *ExplanationService) ExplainAutonomyDecision(ctx context.Context, intentID string) (map[string]any, error) {
	if s.decisions == nil {
		return map[string]any{"intentId": intentID, "warning": "decision repository is not configured"}, nil
	}
	rows, err := s.decisions.ListByIntent(ctx, intentID, 50)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].CreatedAt < rows[j].CreatedAt })
	latest := domain.AutonomyDecision{}
	if len(rows) > 0 {
		latest = rows[len(rows)-1]
	}
	return map[string]any{
		"intentId": intentID,
		"latest":   latest,
		"history":  rows,
	}, nil
}

func (s *ExplanationService) ExplainBudgetUsage(ctx context.Context, budgetID string) (map[string]any, error) {
	if s.budgets == nil {
		return map[string]any{"budgetId": budgetID, "warning": "budget repository is not configured"}, nil
	}
	budget, ok, err := s.budgets.GetByID(ctx, budgetID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, domain.AutonomyError{Code: domain.AutonomyErrNotFound, Field: "budgetId", Message: "budget not found"}
	}
	return map[string]any{
		"budgetId":  budget.ID,
		"status":    budget.Status,
		"period":    budget.Period,
		"usage":     budget.Usage,
		"resetsAt":  budget.ResetsAt,
		"updatedAt": budget.UpdatedAt,
	}, nil
}

func (s *ExplanationService) ExplainCharterDecision(ctx context.Context, charterID string, action domain.SemanticActionType) (map[string]any, error) {
	if s.charters == nil {
		return map[string]any{"charterId": charterID, "warning": "charter repository is not configured"}, nil
	}
	charter, ok, err := s.charters.GetByID(ctx, charterID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, domain.AutonomyError{Code: domain.AutonomyErrNotFound, Field: "charterId", Message: "charter not found"}
	}
	return map[string]any{
		"charterId":        charterID,
		"status":           charter.Status,
		"action":           action,
		"allowed":          charter.AllowsAction(action),
		"denied":           charter.DeniesAction(action),
		"requiresApproval": charter.RequiresApproval(action),
		"scope":            charter.Scope,
	}, nil
}
