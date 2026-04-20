package api

import (
	"context"
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/gateway"
)

type gatewayAutonomyAuthorizer struct {
	loop *AutonomyMaintenanceLoop
}

func newGatewayAutonomyAuthorizer(loop *AutonomyMaintenanceLoop) gateway.ToolAutonomyAuthorizer {
	if loop == nil {
		return nil
	}
	return &gatewayAutonomyAuthorizer{loop: loop}
}

func (a *gatewayAutonomyAuthorizer) AuthorizeToolRequest(ctx context.Context, req gateway.ToolAutonomyRequest) (gateway.ToolAutonomyDecision, error) {
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

	charterID := strings.TrimSpace(req.Request.CharterID)
	if charterID == "" {
		return gateway.ToolAutonomyDecision{
			Allowed:          false,
			RequiresApproval: true,
			Reason:           "self-initiated tool request requires charterId",
		}, nil
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
	if budgetID != "" {
		budgets, err := a.loop.ListBudgets(ctx)
		if err != nil {
			return gateway.ToolAutonomyDecision{}, err
		}
		for _, budget := range budgets {
			if !strings.EqualFold(strings.TrimSpace(budget.ID), budgetID) {
				continue
			}
			if budget.Status != domain.BudgetStatusActive {
				return gateway.ToolAutonomyDecision{
					Allowed:          false,
					RequiresApproval: false,
					Reason:           "autonomy budget is not active",
				}, nil
			}
			if budget.MaxInternalToolCalls > 0 && budget.Usage.InternalToolCalls+1 > budget.MaxInternalToolCalls {
				return gateway.ToolAutonomyDecision{
					Allowed:          false,
					RequiresApproval: true,
					Reason:           "autonomy budget maxInternalToolCalls exceeded",
				}, nil
			}
			if budget.MaxExternalCallsWithoutApprove > 0 && budget.Usage.ExternalToolCallsNoAppr+1 > budget.MaxExternalCallsWithoutApprove {
				return gateway.ToolAutonomyDecision{
					Allowed:          false,
					RequiresApproval: true,
					Reason:           "autonomy budget external tool allowance exceeded",
				}, nil
			}
			break
		}
	}

	return gateway.ToolAutonomyDecision{
		Allowed:          true,
		RequiresApproval: false,
		Reason:           "autonomy charter and budget allow tool request",
	}, nil
}
