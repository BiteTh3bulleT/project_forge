package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"forge/projectforge/services/core/internal/aios/autonomy"
	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/events"
	"forge/projectforge/services/core/internal/store"
)

func TestHandleAutonomyStatusWithoutLoop(t *testing.T) {
	t.Parallel()

	s := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/api/autonomy/status", nil)
	rr := httptest.NewRecorder()
	s.handleAutonomyStatus(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusOK)
	}

	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	available, _ := payload["available"].(bool)
	if available {
		t.Fatalf("expected autonomy unavailable when loop is not configured")
	}
}

func TestHandleAutonomyEndpointsReturnData(t *testing.T) {
	t.Parallel()

	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	loop, intentID := buildTestAutonomyLoop(t, st)
	s := &Server{st: st, autonomy: loop}
	now := int64(1_900_000_000_000)
	if _, err := st.DB.Exec(`INSERT INTO events(created_at, type, payload_json) VALUES(?,?,?)`, now, "autonomy.test", `{"ok":true}`); err != nil {
		t.Fatalf("insert autonomy event: %v", err)
	}

	{
		req := httptest.NewRequest(http.MethodGet, "/api/autonomy/status", nil)
		rr := httptest.NewRecorder()
		s.handleAutonomyStatus(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("autonomy status code = %d, want %d", rr.Code, http.StatusOK)
		}
	}
	{
		req := httptest.NewRequest(http.MethodGet, "/api/autonomy/intents?limit=10", nil)
		rr := httptest.NewRecorder()
		s.handleAutonomyIntents(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("autonomy intents code = %d, want %d", rr.Code, http.StatusOK)
		}
		var payload struct {
			Intents []domain.AutonomyIntent `json:"intents"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode intents response: %v", err)
		}
		if len(payload.Intents) != 1 || payload.Intents[0].ID != intentID {
			t.Fatalf("unexpected intents response: %+v", payload.Intents)
		}
	}
	{
		req := httptest.NewRequest(http.MethodGet, "/api/autonomy/decisions?limit=10", nil)
		rr := httptest.NewRecorder()
		s.handleAutonomyDecisions(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("autonomy decisions code = %d, want %d", rr.Code, http.StatusOK)
		}
		var payload struct {
			Decisions []domain.AutonomyDecision `json:"decisions"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode decisions response: %v", err)
		}
		if len(payload.Decisions) != 1 {
			t.Fatalf("expected one decision, got %d", len(payload.Decisions))
		}
	}
	{
		req := httptest.NewRequest(http.MethodGet, "/api/autonomy/budgets", nil)
		rr := httptest.NewRecorder()
		s.handleAutonomyBudgets(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("autonomy budgets code = %d, want %d", rr.Code, http.StatusOK)
		}
		var payload struct {
			Budgets []domain.FreedomBudget `json:"budgets"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode budgets response: %v", err)
		}
		if len(payload.Budgets) == 0 {
			t.Fatalf("expected at least one budget")
		}
	}
	{
		req := httptest.NewRequest(http.MethodGet, "/api/autonomy/charters", nil)
		rr := httptest.NewRecorder()
		s.handleAutonomyCharters(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("autonomy charters code = %d, want %d", rr.Code, http.StatusOK)
		}
		var payload struct {
			Charters []domain.AutonomyCharter `json:"charters"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode charters response: %v", err)
		}
		if len(payload.Charters) == 0 {
			t.Fatalf("expected at least one charter")
		}
	}
	{
		req := httptest.NewRequest(http.MethodGet, "/api/autonomy/events?limit=10", nil)
		rr := httptest.NewRecorder()
		s.handleAutonomyEvents(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("autonomy events code = %d, want %d", rr.Code, http.StatusOK)
		}
		var payload struct {
			Events []map[string]any `json:"events"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode events response: %v", err)
		}
		if len(payload.Events) != 1 {
			t.Fatalf("expected one autonomy event, got %d", len(payload.Events))
		}
	}
}

func TestHandleAutonomyIntentExplain(t *testing.T) {
	t.Parallel()

	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	loop, intentID := buildTestAutonomyLoop(t, st)
	s := &Server{st: st, autonomy: loop}
	req := withRouteParam(httptest.NewRequest(http.MethodGet, "/api/autonomy/intents/"+intentID+"/explain", nil), "id", intentID)
	rr := httptest.NewRecorder()
	s.handleAutonomyIntentExplain(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("intent explain code = %d, want %d", rr.Code, http.StatusOK)
	}

	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode explain response: %v", err)
	}
	intent, ok := payload["intent"].(map[string]any)
	if !ok {
		t.Fatalf("expected intent block in explain payload: %+v", payload)
	}
	if got := intent["id"]; got != intentID {
		t.Fatalf("intent id = %v, want %s", got, intentID)
	}
}

func buildTestAutonomyLoop(t *testing.T, st *store.Store) (*AutonomyMaintenanceLoop, string) {
	t.Helper()

	now := int64(1_900_000_000_000)
	nowFn := func() int64 { return now }
	scope := domain.ForgeScope{WorkspaceID: "workspace:test-autonomy-ui", LaneID: "control.semantic"}
	bundle := autonomy.NewInMemoryBundle()
	for _, budget := range autonomy.DefaultBudgets(scope, now, "test") {
		if err := bundle.Budgets.Create(context.Background(), budget); err != nil {
			t.Fatalf("create default budget: %v", err)
		}
	}
	for _, charter := range autonomy.DefaultCharters(scope, now, "test") {
		if err := bundle.Charters.Create(context.Background(), charter); err != nil {
			t.Fatalf("create default charter: %v", err)
		}
	}

	intentID := "intent-autonomy-ui-test"
	intent := domain.AutonomyIntent{
		ID:            intentID,
		Type:          domain.IntentSelfMaintenance,
		Title:         "Autonomy UI test intent",
		Description:   "intent created for API surface test",
		Source:        domain.IntentSourceForge,
		ProposedBy:    "forge.test",
		Scope:         scope,
		Status:        domain.IntentStatusProposed,
		Risk:          domain.AutonomyRiskLow,
		AutonomyLevel: domain.AutonomyLevelProposeOnly,
		CharterID:     "charter_memory_maintenance",
		BudgetID:      "budget_memory_maintenance",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := bundle.Intents.Enqueue(context.Background(), intent); err != nil {
		t.Fatalf("enqueue intent: %v", err)
	}

	decision := domain.AutonomyDecision{
		ID:            "decision-autonomy-ui-test",
		IntentID:      intentID,
		Decision:      domain.DecisionAllowProposeOnly,
		AutonomyLevel: domain.AutonomyLevelProposeOnly,
		Risk:          domain.AutonomyRiskLow,
		DeniedReasons: []string{},
		Warnings:      []string{},
		AllowedActions: []domain.SyscallRequest{
			{
				ID:     "action-autonomy-ui-test",
				Action: domain.ActionCreateNote,
				Scope:  scope,
			},
		},
		BlockedActions: []domain.SyscallRequest{},
		Explanation:    "test decision",
		CreatedAt:      now,
	}
	if err := bundle.Decisions.Create(context.Background(), decision); err != nil {
		t.Fatalf("create decision: %v", err)
	}

	explainer := autonomy.NewExplanationService(bundle.Intents, bundle.Decisions, bundle.Budgets, bundle.Charters)
	loop := NewAutonomyMaintenanceLoop(AutonomyMaintenanceLoopOptions{
		DB:             st.DB,
		Events:         events.New(st.DB),
		Scope:          scope,
		Mode:           domain.AutonomyModeMaintain,
		NowMillis:      nowFn,
		Charters:       bundle.Charters,
		Intents:        bundle.Intents,
		Budgets:        bundle.Budgets,
		Decisions:      bundle.Decisions,
		Explainer:      explainer,
		IdleAfter:      2,
		TickEvery:      1,
		RunMaintenance: func(context.Context, string) error { return nil },
		RunImprovement: func(context.Context, string) error { return nil },
	})
	return loop, intentID
}

func withRouteParam(req *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}
