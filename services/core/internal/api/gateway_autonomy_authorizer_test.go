package api

import (
	"context"
	"strings"
	"testing"

	"forge/projectforge/services/core/internal/aios/autonomy"
	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/gateway"
)

func TestGatewayAutonomyAuthorizerBindsIntentCharterBudgetAndConsumesUsage(t *testing.T) {
	ctx := context.Background()
	now := domain.NowMillis()
	scope := domain.ForgeScope{WorkspaceID: "workspace:test", LaneID: "control.semantic"}
	bundle := autonomy.NewInMemoryBundle()
	charter := autonomy.DefaultCharters(scope, now, "forge.autonomy")[0]
	charter.AllowedTools = []string{"filesystem.read_file", "network.http_request"}
	charter.RequiresApprovalTools = nil
	if err := bundle.Charters.Create(ctx, charter); err != nil {
		t.Fatalf("create charter: %v", err)
	}
	budget := autonomy.DefaultBudgets(scope, now, "forge.autonomy")[0]
	budget.MaxInternalToolCalls = 1
	budget.MaxExternalCallsWithoutApprove = 0
	if err := bundle.Budgets.Create(ctx, budget); err != nil {
		t.Fatalf("create budget: %v", err)
	}
	intent := domain.AutonomyIntent{
		ID:            "intent-tool-test",
		Type:          domain.IntentSelfMaintenance,
		Title:         "bounded tool test",
		Source:        domain.IntentSourceForge,
		ProposedBy:    "forge.autonomy",
		Scope:         scope,
		Status:        domain.IntentStatusRunning,
		Risk:          domain.AutonomyRiskLow,
		AutonomyLevel: domain.AutonomyLevelAutoCommitSafe,
		CharterID:     charter.ID,
		BudgetID:      budget.ID,
		Provenance:    domain.Provenance{Actor: "forge.autonomy", ActorType: "system", Source: "test"},
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := bundle.Intents.Enqueue(ctx, intent); err != nil {
		t.Fatalf("create intent: %v", err)
	}
	loop := NewAutonomyMaintenanceLoop(AutonomyMaintenanceLoopOptions{
		Scope: scope, Mode: domain.AutonomyModeMaintain,
		Charters: bundle.Charters, Intents: bundle.Intents, Budgets: bundle.Budgets,
	})
	authorizer := newGatewayAutonomyAuthorizer(loop).(*gatewayAutonomyAuthorizer)
	base := gateway.ToolAutonomyRequest{
		Request: gateway.Request{
			Source: string(domain.SourceSystem), WorkspaceID: scope.WorkspaceID,
			IntentID: intent.ID, CharterID: charter.ID, BudgetID: budget.ID,
		},
		Capability: domain.ToolCapability{ID: "filesystem.read_file", Risk: domain.ToolRiskLow},
	}

	unknownIntent := base
	unknownIntent.Request.IntentID = "intent-missing"
	if got, err := authorizer.AuthorizeToolRequest(ctx, unknownIntent); err != nil || got.Allowed || !strings.Contains(got.Reason, "does not exist") {
		t.Fatalf("unknown intent decision=%+v err=%v", got, err)
	}

	wrongBudget := base
	wrongBudget.Request.BudgetID = "budget-missing"
	if got, err := authorizer.AuthorizeToolRequest(ctx, wrongBudget); err != nil || got.Allowed || !strings.Contains(got.Reason, "does not match") {
		t.Fatalf("wrong budget decision=%+v err=%v", got, err)
	}

	if got, err := authorizer.AuthorizeToolRequest(ctx, base); err != nil || !got.Allowed {
		t.Fatalf("initial authorization=%+v err=%v", got, err)
	}
	if err := authorizer.ConsumeAuthorizedToolRequest(ctx, base); err != nil {
		t.Fatalf("consume tool budget: %v", err)
	}
	if got, err := authorizer.AuthorizeToolRequest(ctx, base); err != nil || got.Allowed || !strings.Contains(got.Reason, "maxInternalToolCalls") {
		t.Fatalf("exhausted budget decision=%+v err=%v", got, err)
	}

	external := base
	external.Capability.ID = "network.http_request"
	external.UsesNetwork = true
	if got, err := authorizer.AuthorizeToolRequest(ctx, external); err != nil || got.Allowed || !got.RequiresApproval {
		t.Fatalf("zero external allowance decision=%+v err=%v", got, err)
	}
}
