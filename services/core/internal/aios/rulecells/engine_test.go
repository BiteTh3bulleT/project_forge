package rulecells

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestRegistryAndRuleLoading(t *testing.T) {
	pack := testPack("pack.registration", RuleCell{
		ID:        "rule.registration",
		Name:      "registration",
		Lane:      LaneKernel,
		Phase:     PhaseSyscallValidation,
		Priority:  1,
		Enabled:   true,
		Condition: Condition{Field: "flag", Operator: OpBoolIs, Value: true},
		Version:   "0.1.0",
		Output:    RuleOutput{Type: OutputPolicyDecision, Decision: "warn"},
	})
	reg, err := NewRegistry(pack)
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	engine, err := NewEngine(EngineOptions{Packs: reg.Packs(), Clock: fixedClock(1760000000000)})
	if err != nil {
		t.Fatalf("engine: %v", err)
	}
	result, err := engine.Run(context.Background(), RunInput{
		Lane:    LaneKernel,
		Phase:   PhaseSyscallValidation,
		InputID: "in-1",
		Facts:   map[string]any{"flag": true},
	}, RunOptions{DryRun: true})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(result.Outputs) != 1 || result.Trace.RulePacks[0].ID != "pack.registration" || result.Trace.RulePacks[0].Version != "0.1.0" {
		t.Fatalf("expected registered pack output and versioned trace, got %+v", result)
	}
}

func TestDisabledRuleIsSkipped(t *testing.T) {
	engine := mustTestEngine(t, testPack("pack.disabled", RuleCell{
		ID:        "rule.disabled",
		Name:      "disabled",
		Lane:      LaneKernel,
		Phase:     PhaseSyscallValidation,
		Priority:  1,
		Enabled:   false,
		Condition: Condition{Field: "flag", Operator: OpBoolIs, Value: true},
		Version:   "0.1.0",
		Output:    RuleOutput{Type: OutputRejectDecision, Decision: "reject"},
	}))
	result, err := engine.Run(context.Background(), RunInput{Lane: LaneKernel, Phase: PhaseSyscallValidation, InputID: "in-1", Facts: map[string]any{"flag": true}}, RunOptions{Debug: true})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(result.Outputs) != 0 || result.Trace.RulesEvaluated != 0 || !containsString(result.Trace.NonMatches, "rule.disabled:disabled") {
		t.Fatalf("disabled rule should be skipped, got %+v", result.Trace)
	}
}

func TestPriorityOrderingIsDeterministic(t *testing.T) {
	engine := mustTestEngine(t, testPack("pack.order", RuleCell{
		ID:        "rule.low",
		Name:      "low",
		Lane:      LaneArterial,
		Phase:     PhaseRestoreScoring,
		Priority:  10,
		Enabled:   true,
		Condition: Condition{Field: "flag", Operator: OpBoolIs, Value: true},
		Version:   "0.1.0",
		Output:    RuleOutput{Type: OutputScoreAdjustment, ScoreDelta: 0.01},
	}, RuleCell{
		ID:        "rule.high",
		Name:      "high",
		Lane:      LaneArterial,
		Phase:     PhaseRestoreScoring,
		Priority:  20,
		Enabled:   true,
		Condition: Condition{Field: "flag", Operator: OpBoolIs, Value: true},
		Version:   "0.1.0",
		Output:    RuleOutput{Type: OutputScoreAdjustment, ScoreDelta: 0.02},
	}, RuleCell{
		ID:        "rule.alpha",
		Name:      "alpha",
		Lane:      LaneArterial,
		Phase:     PhaseRestoreScoring,
		Priority:  20,
		Enabled:   true,
		Condition: Condition{Field: "flag", Operator: OpBoolIs, Value: true},
		Version:   "0.1.0",
		Output:    RuleOutput{Type: OutputScoreAdjustment, ScoreDelta: 0.03},
	}))
	result, err := engine.Run(context.Background(), RunInput{Lane: LaneArterial, Phase: PhaseRestoreScoring, InputID: "in-1", Facts: map[string]any{"flag": true}}, RunOptions{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	got := matchedIDs(result.Trace)
	want := []string{"rule.alpha", "rule.high", "rule.low"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("priority ordering mismatch: got %v want %v", got, want)
	}
}

func TestLanePhaseFilteringRunsOnlyRelevantRules(t *testing.T) {
	engine := mustTestEngine(t, testPack("pack.filter", RuleCell{
		ID:        "rule.restore",
		Name:      "restore",
		Lane:      LaneArterial,
		Phase:     PhaseRestoreScoring,
		Priority:  1,
		Enabled:   true,
		Condition: Condition{Field: "flag", Operator: OpBoolIs, Value: true},
		Version:   "0.1.0",
		Output:    RuleOutput{Type: OutputScoreAdjustment, ScoreDelta: 0.01},
	}, RuleCell{
		ID:        "rule.runtime",
		Name:      "runtime",
		Lane:      LaneRuntime,
		Phase:     PhaseModelRouting,
		Priority:  1,
		Enabled:   true,
		Condition: Condition{Field: "flag", Operator: OpBoolIs, Value: true},
		Version:   "0.1.0",
		Output:    RuleOutput{Type: OutputModelRoutingHint, Decision: "no_model_needed"},
	}))
	result, err := engine.Run(context.Background(), RunInput{Lane: LaneArterial, Phase: PhaseRestoreScoring, InputID: "in-1", Facts: map[string]any{"flag": true}}, RunOptions{})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if result.Trace.RulesEvaluated != 1 || len(result.Outputs) != 1 || result.Trace.MatchedRules[0].RuleID != "rule.restore" {
		t.Fatalf("expected only restore rule, got %+v", result.Trace)
	}
}

func TestRepeatedInputProducesIdenticalOutput(t *testing.T) {
	engine := mustTestEngine(t, testPack("pack.repeat", RuleCell{
		ID:        "rule.repeat",
		Name:      "repeat",
		Lane:      LaneRuntime,
		Phase:     PhaseProviderCooldown,
		Priority:  1,
		Enabled:   true,
		Condition: Condition{Field: "provider_status", Operator: OpProviderStatusMatch, Value: "cooldown"},
		Version:   "0.1.0",
		Output:    RuleOutput{Type: OutputPolicyDecision, Decision: "reject"},
	}))
	input := RunInput{Lane: LaneRuntime, Phase: PhaseProviderCooldown, InputID: "provider-a", Facts: map[string]any{"provider_status": "cooldown"}}
	a, err := engine.Run(context.Background(), input, RunOptions{DryRun: true})
	if err != nil {
		t.Fatalf("run a: %v", err)
	}
	b, err := engine.Run(context.Background(), input, RunOptions{DryRun: true})
	if err != nil {
		t.Fatalf("run b: %v", err)
	}
	if !reflect.DeepEqual(a.Outputs, b.Outputs) || !reflect.DeepEqual(a.Trace, b.Trace) {
		t.Fatalf("expected identical deterministic output\na=%+v\nb=%+v", a, b)
	}
}

func TestLatencyBudgetWarning(t *testing.T) {
	ticks := []time.Time{
		time.UnixMilli(1000), time.UnixMilli(1000), time.UnixMilli(1012), time.UnixMilli(1012), time.UnixMilli(1020),
	}
	i := 0
	clock := func() time.Time {
		if i >= len(ticks) {
			return ticks[len(ticks)-1]
		}
		out := ticks[i]
		i++
		return out
	}
	engine, err := NewEngine(EngineOptions{Clock: clock, Packs: []RulePack{testPack("pack.latency", RuleCell{
		ID:           "rule.slow",
		Name:         "slow",
		Lane:         LaneKernel,
		Phase:        PhaseSyscallValidation,
		Priority:     1,
		Enabled:      true,
		Condition:    Condition{Field: "flag", Operator: OpBoolIs, Value: true},
		Version:      "0.1.0",
		MaxLatencyMs: 1,
		Output:       RuleOutput{Type: OutputPolicyDecision, Decision: "warn"},
	})}})
	if err != nil {
		t.Fatalf("engine: %v", err)
	}
	result, err := engine.Run(context.Background(), RunInput{Lane: LaneKernel, Phase: PhaseSyscallValidation, InputID: "in-1", Facts: map[string]any{"flag": true}}, RunOptions{MaxLatencyMs: 1})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(result.Warnings) == 0 || !strings.Contains(strings.Join(result.Warnings, "\n"), "latency budget") {
		t.Fatalf("expected latency warning, got %+v", result.Warnings)
	}
}

func TestTraceIncludesMatchedRuleOutputAndPackVersion(t *testing.T) {
	engine := MustStaticEngine()
	result, err := engine.Run(context.Background(), RunInput{
		Lane:    LaneKernel,
		Phase:   PhaseSyscallValidation,
		InputID: "truth-write",
		Facts:   map[string]any{"direct_truth_mutation_attempt": true},
	}, RunOptions{DryRun: true})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(result.Trace.MatchedRules) == 0 || len(result.Trace.Outputs) == 0 || len(result.Trace.RulePacks) != 1 {
		t.Fatalf("trace missing matched rules, outputs, or pack refs: %+v", result.Trace)
	}
	if result.Trace.RulePacks[0].ID != PackKernelAuthorityID || result.Trace.RulePacks[0].Version != StaticPackVersion {
		t.Fatalf("unexpected pack trace: %+v", result.Trace.RulePacks)
	}
}

func TestStaticKernelTruthWriteReject(t *testing.T) {
	result, err := MustStaticEngine().Run(context.Background(), RunInput{
		Lane:    LaneKernel,
		Phase:   PhaseSyscallValidation,
		InputID: "truth-write",
		Facts:   map[string]any{"direct_truth_mutation_attempt": true},
	}, RunOptions{DryRun: true})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !hasOutput(result.Outputs, OutputRejectDecision, "reject") {
		t.Fatalf("expected truth-write reject, got %+v", result.Outputs)
	}
}

func TestStaticRuntimeProviderCooldownBlocksRetry(t *testing.T) {
	result, err := MustStaticEngine().Run(context.Background(), RunInput{
		Lane:    LaneRuntime,
		Phase:   PhaseProviderCooldown,
		InputID: "provider-a",
		Facts:   map[string]any{"provider_status": "cooldown"},
	}, RunOptions{DryRun: true})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !hasOutput(result.Outputs, OutputPolicyDecision, "reject") {
		t.Fatalf("expected cooldown reject, got %+v", result.Outputs)
	}
}

func TestStaticNeuralClassificationTagsCorrection(t *testing.T) {
	result, err := MustStaticEngine().Run(context.Background(), RunInput{
		Lane:    LaneNeural,
		Phase:   PhaseIngestClassification,
		InputID: "event-a",
		Facts:   map[string]any{"text": "Actually, correct that preference"},
	}, RunOptions{DryRun: true})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !hasOutput(result.Outputs, OutputRouteDecision, "tag_correction") {
		t.Fatalf("expected correction tag route, got %+v", result.Outputs)
	}
	if len(result.Trace.RulePacks) != 1 || result.Trace.RulePacks[0].ID != PackNeuralClassifyID {
		t.Fatalf("expected neural classification pack trace, got %+v", result.Trace.RulePacks)
	}
}

func TestStaticOperatorBlockedLoopAttention(t *testing.T) {
	result, err := MustStaticEngine().Run(context.Background(), RunInput{
		Lane:    LaneOperator,
		Phase:   PhaseAttentionRouting,
		InputID: "loop-a",
		Facts:   map[string]any{"blocked_loop": true},
	}, RunOptions{DryRun: true})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !hasOutput(result.Outputs, OutputAttentionSignal, "raise_attention") {
		t.Fatalf("expected attention signal, got %+v", result.Outputs)
	}
}

func TestDryRunAndOutputsHaveNoCanonicalMutationCapability(t *testing.T) {
	result, err := MustStaticEngine().Run(context.Background(), RunInput{
		Lane:    LaneArterial,
		Phase:   PhaseRestoreScoring,
		InputID: "restore-a",
		Facts:   map[string]any{"query_exact": true, "base_score": 0.5},
	}, RunOptions{DryRun: true})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	text := strings.ToLower(string(raw))
	for _, forbidden := range []string{"commit", "mutate", "insert", "update", "delete"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("rule output should not expose canonical mutation capability %q in %s", forbidden, text)
		}
	}
}

func TestAuthorityConflictHardDenialWins(t *testing.T) {
	engine := mustTestEngine(t, testPack("pack.conflict", RuleCell{
		ID:        "rule.allow",
		Name:      "allow",
		Lane:      LaneKernel,
		Phase:     PhaseSyscallValidation,
		Priority:  1,
		Enabled:   true,
		Condition: Condition{Field: "flag", Operator: OpBoolIs, Value: true},
		Version:   "0.1.0",
		Output:    RuleOutput{Type: OutputRouteDecision, Decision: "allow", Explain: "rule would allow"},
	}))
	result, err := engine.Run(context.Background(), RunInput{
		Lane:                  LaneKernel,
		Phase:                 PhaseSyscallValidation,
		InputID:               "conflict",
		Facts:                 map[string]any{"flag": true},
		AuthoritativeDecision: "approval_required",
	}, RunOptions{DryRun: true})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(result.Outputs) != 1 || result.Outputs[0].Decision != "approval_required" || result.Outputs[0].Metadata["authority_conflict"] != true {
		t.Fatalf("authority denial should win over allow output, got %+v", result.Outputs)
	}
}

func TestStaticArterialWrongWorkspaceAndFreshCompileOutputs(t *testing.T) {
	engine := MustStaticEngine()
	wrong, err := engine.Run(context.Background(), RunInput{
		Lane:    LaneArterial,
		Phase:   PhaseRestoreScoring,
		InputID: "wrong-workspace",
		Facts:   map[string]any{"wrong_workspace": true, "base_score": 0.8},
	}, RunOptions{DryRun: true})
	if err != nil {
		t.Fatalf("wrong workspace run: %v", err)
	}
	if !hasOutput(wrong.Outputs, OutputRejectDecision, "reject") {
		t.Fatalf("expected wrong workspace reject, got %+v", wrong.Outputs)
	}
	low, err := engine.Run(context.Background(), RunInput{
		Lane:    LaneArterial,
		Phase:   PhaseRestoreScoring,
		InputID: "low-score",
		Facts:   map[string]any{"base_score": 0.2},
	}, RunOptions{DryRun: true})
	if err != nil {
		t.Fatalf("low score run: %v", err)
	}
	if !hasType(low.Outputs, OutputFreshCompileRequired) {
		t.Fatalf("expected fresh compile required, got %+v", low.Outputs)
	}
}

func mustTestEngine(t *testing.T, packs ...RulePack) *Engine {
	t.Helper()
	engine, err := NewEngine(EngineOptions{Packs: packs, Clock: fixedClock(1760000000000)})
	if err != nil {
		t.Fatalf("engine: %v", err)
	}
	return engine
}

func testPack(id string, rules ...RuleCell) RulePack {
	return RulePack{ID: id, Version: "0.1.0", Rules: rules}
}

func fixedClock(ms int64) func() time.Time {
	return func() time.Time { return time.UnixMilli(ms) }
}

func matchedIDs(trace RuleTrace) []string {
	out := make([]string, 0, len(trace.MatchedRules))
	for _, matched := range trace.MatchedRules {
		out = append(out, matched.RuleID)
	}
	return out
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func hasOutput(outputs []RuleOutput, typ OutputType, decision string) bool {
	for _, output := range outputs {
		if output.Type == typ && output.Decision == decision {
			return true
		}
	}
	return false
}

func hasType(outputs []RuleOutput, typ OutputType) bool {
	for _, output := range outputs {
		if output.Type == typ {
			return true
		}
	}
	return false
}
