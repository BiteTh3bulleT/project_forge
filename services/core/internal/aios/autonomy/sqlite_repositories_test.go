package autonomy

import (
	"context"
	"testing"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/store"
)

func TestSQLiteBundlePersistsAcrossStoreReopen(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()

	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}

	scope := domain.ForgeScope{WorkspaceID: "ws-persist", LaneID: "control.semantic"}
	now := int64(1764000000000)

	bundle := NewSQLiteBundle(st.DB)
	charter := domain.AutonomyCharter{
		ID:             "charter-persist-1",
		Name:           "Persistence Charter",
		Description:    "verify sqlite durability",
		Scope:          scope,
		Status:         domain.CharterActive,
		Purpose:        "durability test",
		AllowedSources: []domain.IntentSource{domain.IntentSourceForge},
		RiskLimits:     []domain.AutonomyRisk{domain.AutonomyRiskLow, domain.AutonomyRiskMedium},
		EffectiveFrom:  now,
		CreatedBy:      "test",
		Provenance:     domain.Provenance{Actor: "test", ActorType: "test", Source: "test", TraceID: "trace-charter"},
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := bundle.Charters.Create(ctx, charter); err != nil {
		t.Fatalf("create charter: %v", err)
	}

	budget := domain.FreedomBudget{
		ID:                           "budget-persist-1",
		Name:                         "Persistence Budget",
		Scope:                        scope,
		Status:                       domain.BudgetStatusActive,
		Period:                       domain.BudgetPeriodDaily,
		MaxSelfActionsPerRun:         4,
		MaxRunsPerPeriod:             10,
		MaxCommittedActionsPerPeriod: 20,
		MaxProposedActionsPerPeriod:  40,
		MaxInternalToolCalls:         20,
		MaxArchiveActions:            5,
		MaxContextPrecompilations:    3,
		MaxProjectionRebuilds:        2,
		MaxRuleAgentRuns:             12,
		CreatedAt:                    now + 1,
		UpdatedAt:                    now + 1,
	}
	if err := bundle.Budgets.Create(ctx, budget); err != nil {
		t.Fatalf("create budget: %v", err)
	}

	intent := domain.AutonomyIntent{
		ID:            "intent-persist-1",
		Type:          domain.IntentSelfMaintenance,
		Title:         "Persistence Intent",
		Description:   "verify intent durability",
		Source:        domain.IntentSourceForge,
		ProposedBy:    "test",
		Scope:         scope,
		Status:        domain.IntentStatusProposed,
		Risk:          domain.AutonomyRiskLow,
		AutonomyLevel: domain.AutonomyLevelProposeOnly,
		CharterID:     charter.ID,
		BudgetID:      budget.ID,
		Provenance:    domain.Provenance{Actor: "test", ActorType: "test", Source: "test", TraceID: "trace-intent"},
		CorrelationID: "corr-persist-1",
		TraceID:       "trace-intent",
		CreatedAt:     now + 2,
		UpdatedAt:     now + 2,
	}
	if err := bundle.Intents.Enqueue(ctx, intent); err != nil {
		t.Fatalf("enqueue intent: %v", err)
	}

	reservation := domain.BudgetReservation{
		ID:           "reservation-persist-1",
		BudgetID:     budget.ID,
		IntentID:     intent.ID,
		Scope:        scope,
		RequestedFor: "syscall_actions",
		Units:        2,
		CreatedAt:    now + 3,
	}
	if err := bundle.Reservations.Create(ctx, reservation); err != nil {
		t.Fatalf("create reservation: %v", err)
	}

	decision := domain.AutonomyDecision{
		ID:            "decision-persist-1",
		IntentID:      intent.ID,
		Decision:      domain.DecisionAllowProposeOnly,
		AutonomyLevel: domain.AutonomyLevelProposeOnly,
		Risk:          domain.AutonomyRiskLow,
		CharterID:     charter.ID,
		BudgetID:      budget.ID,
		AllowedActions: []domain.SyscallRequest{{
			ID:     "syscall-persist-1",
			Action: domain.ActionCreateNote,
			Actor:  domain.ActorIdentity{ID: "forge.autonomy", Kind: "autonomy"},
			Source: domain.SourceSystem,
			Scope:  scope,
			Provenance: domain.Provenance{
				Actor: "forge.autonomy", ActorType: "system", Source: "autonomy", TraceID: "trace-syscall",
			},
		}},
		CorrelationID: intent.CorrelationID,
		TraceID:       "trace-decision",
		CreatedAt:     now + 4,
	}
	if err := bundle.Decisions.Create(ctx, decision); err != nil {
		t.Fatalf("create decision: %v", err)
	}

	curiosity := domain.CuriosityItem{
		ID:        "curiosity-persist-1",
		Title:     "Persistence Curiosity",
		Question:  "Is autonomy data durable?",
		Source:    domain.IntentSourceSystem,
		Scope:     scope,
		Priority:  "normal",
		Status:    domain.CuriosityOpen,
		CreatedAt: now + 5,
		UpdatedAt: now + 5,
	}
	if err := bundle.Curiosity.Create(ctx, curiosity); err != nil {
		t.Fatalf("create curiosity: %v", err)
	}

	if err := st.Close(); err != nil {
		t.Fatalf("close store: %v", err)
	}

	st2, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	t.Cleanup(func() { _ = st2.Close() })

	reloaded := NewSQLiteBundle(st2.DB)
	if got, ok, err := reloaded.Charters.GetByID(ctx, charter.ID); err != nil || !ok || got.Name != charter.Name {
		t.Fatalf("charter not persisted: ok=%v err=%v got=%+v", ok, err, got)
	}
	if got, ok, err := reloaded.Budgets.GetByID(ctx, budget.ID); err != nil || !ok || got.Name != budget.Name {
		t.Fatalf("budget not persisted: ok=%v err=%v got=%+v", ok, err, got)
	}
	if got, ok, err := reloaded.Intents.GetByID(ctx, intent.ID); err != nil || !ok || got.Title != intent.Title {
		t.Fatalf("intent not persisted: ok=%v err=%v got=%+v", ok, err, got)
	}
	if got, ok, err := reloaded.Reservations.GetByID(ctx, reservation.ID); err != nil || !ok || got.BudgetID != reservation.BudgetID {
		t.Fatalf("reservation not persisted: ok=%v err=%v got=%+v", ok, err, got)
	}
	if got, ok, err := reloaded.Curiosity.GetByID(ctx, curiosity.ID); err != nil || !ok || got.Question != curiosity.Question {
		t.Fatalf("curiosity not persisted: ok=%v err=%v got=%+v", ok, err, got)
	}

	byIntent, err := reloaded.Decisions.ListByIntent(ctx, intent.ID, 10)
	if err != nil {
		t.Fatalf("list decisions by intent: %v", err)
	}
	if len(byIntent) != 1 || byIntent[0].ID != decision.ID {
		t.Fatalf("decision by intent mismatch: %+v", byIntent)
	}

	byScope, err := reloaded.Decisions.ListByScope(ctx, scope, 10)
	if err != nil {
		t.Fatalf("list decisions by scope: %v", err)
	}
	if len(byScope) != 1 || byScope[0].ID != decision.ID {
		t.Fatalf("decision by scope mismatch: %+v", byScope)
	}

	byCorrelation, err := reloaded.Decisions.ListByCorrelation(ctx, intent.CorrelationID, 10)
	if err != nil {
		t.Fatalf("list decisions by correlation: %v", err)
	}
	if len(byCorrelation) != 1 || byCorrelation[0].ID != decision.ID {
		t.Fatalf("decision by correlation mismatch: %+v", byCorrelation)
	}
}
