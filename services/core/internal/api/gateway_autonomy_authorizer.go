package api

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/gateway"
)

type gatewayAutonomyAuthorizer struct {
	loop *AutonomyMaintenanceLoop
	mu   sync.Mutex
}

func newGatewayAutonomyAuthorizer(loop *AutonomyMaintenanceLoop) gateway.ToolAutonomyAuthorizer {
	if loop == nil {
		return nil
	}
	return &gatewayAutonomyAuthorizer{loop: loop}
}

func (a *gatewayAutonomyAuthorizer) AuthorizeToolRequest(ctx context.Context, req gateway.ToolAutonomyRequest) (gateway.ToolAutonomyDecision, error) {
	if a == nil || a.loop == nil || a.loop.intents == nil {
		return gateway.ToolAutonomyDecision{
			Allowed:          false,
			RequiresApproval: true,
			Reason:           "autonomy loop is not configured",
		}, nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.authorizeToolRequestLocked(ctx, req)
}

func (a *gatewayAutonomyAuthorizer) authorizeToolRequestLocked(ctx context.Context, req gateway.ToolAutonomyRequest) (gateway.ToolAutonomyDecision, error) {
	if a == nil || a.loop == nil {
		return gateway.ToolAutonomyDecision{
			Allowed:          false,
			RequiresApproval: true,
			Reason:           "autonomy loop is not configured",
		}, nil
	}
	intentID := strings.TrimSpace(req.Request.IntentID)
	if intentID == "" {
		return gateway.ToolAutonomyDecision{
			Allowed:          false,
			RequiresApproval: false,
			Reason:           "self-initiated tool request requires intentId",
		}, nil
	}
	intent, ok, err := a.loop.intents.GetByID(ctx, intentID)
	if err != nil {
		return gateway.ToolAutonomyDecision{}, err
	}
	if !ok {
		return gateway.ToolAutonomyDecision{Allowed: false, Reason: "autonomy intent does not exist"}, nil
	}
	switch intent.Status {
	case domain.IntentStatusProposed, domain.IntentStatusApproved, domain.IntentStatusRunning:
	default:
		return gateway.ToolAutonomyDecision{Allowed: false, Reason: "autonomy intent is not active"}, nil
	}
	if strings.TrimSpace(req.Request.WorkspaceID) != strings.TrimSpace(intent.Scope.WorkspaceID) {
		return gateway.ToolAutonomyDecision{Allowed: false, Reason: "tool request scope does not match autonomy intent"}, nil
	}

	charterID := strings.TrimSpace(req.Request.CharterID)
	if charterID == "" {
		return gateway.ToolAutonomyDecision{
			Allowed:          false,
			RequiresApproval: true,
			Reason:           "self-initiated tool request requires charterId",
		}, nil
	}
	if charterID != strings.TrimSpace(intent.CharterID) {
		return gateway.ToolAutonomyDecision{Allowed: false, Reason: "tool request charter does not match autonomy intent"}, nil
	}

	now := domain.NowMillis()
	charters, err := a.loop.ListCharters(ctx, true)
	if err != nil {
		return gateway.ToolAutonomyDecision{}, err
	}
	var charter *domain.AutonomyCharter
	for i := range charters {
		if strings.EqualFold(strings.TrimSpace(charters[i].ID), charterID) {
			charter = &charters[i]
			break
		}
	}
	if charter == nil || !charter.IsActive(now) {
		return gateway.ToolAutonomyDecision{
			Allowed:          false,
			RequiresApproval: false,
			Reason:           "autonomy charter is missing or inactive",
		}, nil
	}
	capabilityID := strings.TrimSpace(strings.ToLower(req.Capability.ID))
	if capabilityID != "" {
		if charter.DeniesTool(capabilityID) {
			return gateway.ToolAutonomyDecision{
				Allowed:          false,
				RequiresApproval: false,
				Reason:           "autonomy charter denies capability",
			}, nil
		}
		if len(charter.AllowedTools) > 0 && !charter.AllowsTool(capabilityID) {
			return gateway.ToolAutonomyDecision{
				Allowed:          false,
				RequiresApproval: true,
				Reason:           "capability is outside charter allowedTools",
			}, nil
		}
		if charter.RequiresToolApproval(capabilityID) {
			return gateway.ToolAutonomyDecision{
				Allowed:          false,
				RequiresApproval: true,
				Reason:           "charter requires approval for capability",
			}, nil
		}
	}
	if charter.MaxToolRisk != "" && req.Risk.Risk.Rank() > charter.MaxToolRisk.Rank() {
		return gateway.ToolAutonomyDecision{
			Allowed:          false,
			RequiresApproval: true,
			Reason:           "capability risk exceeds charter maxToolRisk",
		}, nil
	}

	budgetID := strings.TrimSpace(req.Request.BudgetID)
	if budgetID == "" {
		return gateway.ToolAutonomyDecision{Allowed: false, Reason: "self-initiated tool request requires budgetId"}, nil
	}
	if budgetID != strings.TrimSpace(intent.BudgetID) {
		return gateway.ToolAutonomyDecision{Allowed: false, Reason: "tool request budget does not match autonomy intent"}, nil
	}
	if budgetID != "" {
		budgets, err := a.loop.ListBudgets(ctx)
		if err != nil {
			return gateway.ToolAutonomyDecision{}, err
		}
		foundBudget := false
		for _, budget := range budgets {
			if !strings.EqualFold(strings.TrimSpace(budget.ID), budgetID) {
				continue
			}
			foundBudget = true
			if budget.Status != domain.BudgetStatusActive {
				return gateway.ToolAutonomyDecision{
					Allowed:          false,
					RequiresApproval: false,
					Reason:           "autonomy budget is not active",
				}, nil
			}
			countsAsInternal := !req.UsesNetwork || strings.TrimSpace(req.Request.ApprovalID) != ""
			if countsAsInternal && budget.MaxInternalToolCalls > 0 && budget.Usage.InternalToolCalls+1 > budget.MaxInternalToolCalls {
				return gateway.ToolAutonomyDecision{
					Allowed:          false,
					RequiresApproval: true,
					Reason:           "autonomy budget maxInternalToolCalls exceeded",
				}, nil
			}
			if req.UsesNetwork && strings.TrimSpace(req.Request.ApprovalID) == "" &&
				budget.Usage.ExternalToolCallsNoAppr+1 > budget.MaxExternalCallsWithoutApprove {
				return gateway.ToolAutonomyDecision{
					Allowed:          false,
					RequiresApproval: true,
					Reason:           "autonomy budget external tool allowance exceeded",
				}, nil
			}
			break
		}
		if !foundBudget {
			return gateway.ToolAutonomyDecision{Allowed: false, Reason: "autonomy budget does not exist"}, nil
		}
	}

	return gateway.ToolAutonomyDecision{
		Allowed:          true,
		RequiresApproval: false,
		Reason:           "autonomy charter and budget allow tool request",
	}, nil
}

func (a *gatewayAutonomyAuthorizer) ConsumeAuthorizedToolRequest(ctx context.Context, req gateway.ToolAutonomyRequest) error {
	if a == nil || a.loop == nil || a.loop.budgets == nil {
		return fmt.Errorf("autonomy budget repository is not configured")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	decision, err := a.authorizeToolRequestLocked(ctx, req)
	if err != nil {
		return err
	}
	if !decision.Allowed {
		return fmt.Errorf("%s", decision.Reason)
	}
	budgetID := strings.TrimSpace(req.Request.BudgetID)
	budget, ok, err := a.loop.budgets.GetByID(ctx, budgetID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("autonomy budget does not exist")
	}
	if req.UsesNetwork && strings.TrimSpace(req.Request.ApprovalID) == "" {
		budget.Usage.ExternalToolCallsNoAppr++
	} else {
		budget.Usage.InternalToolCalls++
	}
	budget.UpdatedAt = domain.NowMillis()
	return a.loop.budgets.Update(ctx, budget)
}
