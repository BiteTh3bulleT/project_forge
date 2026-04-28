package rulecells

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type Engine struct {
	packs []RulePack
	clock func() time.Time
}

func NewEngine(opts EngineOptions) (*Engine, error) {
	clock := opts.Clock
	if clock == nil {
		clock = time.Now
	}
	packs := append([]RulePack(nil), opts.Packs...)
	if len(packs) == 0 {
		packs = StaticRulePacks()
	}
	if err := validatePacks(packs); err != nil {
		return nil, err
	}
	return &Engine{packs: packs, clock: clock}, nil
}

func MustStaticEngine() *Engine {
	engine, err := NewEngine(EngineOptions{Packs: StaticRulePacks()})
	if err != nil {
		panic(err)
	}
	return engine
}

func (e *Engine) Run(ctx context.Context, in RunInput, opts RunOptions) (RunResult, error) {
	if e == nil {
		return RunResult{}, nil
	}
	start := e.clock().UnixMilli()
	trace := RuleTrace{
		TraceID:      stableTraceID(in),
		Lane:         in.Lane,
		Phase:        in.Phase,
		InputID:      strings.TrimSpace(in.InputID),
		StartedAt:    start,
		MatchedRules: []MatchedRuleTrace{},
		Outputs:      []RuleOutput{},
		Warnings:     []string{},
	}
	if opts.Disabled {
		trace.CompletedAt = e.clock().UnixMilli()
		trace.LatencyMs = maxInt64(0, trace.CompletedAt-trace.StartedAt)
		return RunResult{Trace: trace, Warnings: trace.Warnings}, nil
	}

	type candidate struct {
		pack RulePack
		rule RuleCell
	}
	candidates := []candidate{}
	for _, pack := range e.packs {
		for _, rule := range pack.Rules {
			if rule.Lane != in.Lane || rule.Phase != in.Phase {
				continue
			}
			if !hasPackRef(trace.RulePacks, pack.ID, pack.Version) {
				trace.RulePacks = append(trace.RulePacks, RulePackRef{ID: pack.ID, Version: pack.Version})
			}
			if !rule.Enabled {
				if opts.Debug {
					trace.NonMatches = append(trace.NonMatches, rule.ID+":disabled")
				}
				continue
			}
			candidates = append(candidates, candidate{pack: pack, rule: rule})
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].rule.Priority == candidates[j].rule.Priority {
			return candidates[i].rule.ID < candidates[j].rule.ID
		}
		return candidates[i].rule.Priority > candidates[j].rule.Priority
	})

	for _, candidate := range candidates {
		select {
		case <-ctx.Done():
			return RunResult{}, ctx.Err()
		default:
		}
		trace.RulesEvaluated++
		ruleStart := e.clock().UnixMilli()
		matched := conditionMatches(candidate.rule.Condition, in.Facts, e.clock().UnixMilli())
		ruleLatency := maxInt64(0, e.clock().UnixMilli()-ruleStart)
		if candidate.rule.MaxLatencyMs > 0 && ruleLatency > candidate.rule.MaxLatencyMs {
			trace.Warnings = append(trace.Warnings, fmt.Sprintf("rule %s exceeded latency budget: %dms > %dms", candidate.rule.ID, ruleLatency, candidate.rule.MaxLatencyMs))
		}
		if !matched {
			if opts.Debug {
				trace.NonMatches = append(trace.NonMatches, candidate.rule.ID)
			}
			continue
		}
		output := candidate.rule.Output
		if output.Type == "" {
			output.Type = outputTypeForAction(candidate.rule.Action)
		}
		if output.ScoreDelta == 0 {
			output.ScoreDelta = candidate.rule.ScoreDelta
		}
		if output.Weight == 0 {
			output.Weight = candidate.rule.Weight
		}
		if len(output.Tags) == 0 {
			output.Tags = append([]string(nil), candidate.rule.Tags...)
		}
		if output.Explain == "" {
			output.Explain = candidate.rule.Explain
		}
		if output.Metadata == nil {
			output.Metadata = map[string]any{}
		}
		output.Metadata["rule_id"] = candidate.rule.ID
		output.Metadata["rule_version"] = candidate.rule.Version
		output.Metadata["rule_pack_id"] = candidate.pack.ID
		output.Metadata["rule_pack_version"] = candidate.pack.Version
		output = applyAuthoritySafety(output, strings.TrimSpace(in.AuthoritativeDecision), &trace)
		trace.Outputs = append(trace.Outputs, output)
		trace.MatchedRules = append(trace.MatchedRules, MatchedRuleTrace{
			RuleID:      candidate.rule.ID,
			RuleVersion: candidate.rule.Version,
			PackID:      candidate.pack.ID,
			PackVersion: candidate.pack.Version,
			OutputTypes: []string{string(output.Type)},
			Explain:     output.Explain,
		})
	}
	completed := e.clock().UnixMilli()
	trace.CompletedAt = completed
	trace.LatencyMs = maxInt64(0, completed-start)
	if opts.MaxLatencyMs > 0 && trace.LatencyMs > opts.MaxLatencyMs {
		trace.Warnings = append(trace.Warnings, fmt.Sprintf("rule engine exceeded latency budget: %dms > %dms", trace.LatencyMs, opts.MaxLatencyMs))
	}
	for _, pack := range e.packs {
		if pack.MaxLatencyMs > 0 && trace.LatencyMs > pack.MaxLatencyMs {
			trace.Warnings = append(trace.Warnings, fmt.Sprintf("rule pack %s@%s exceeded latency budget: %dms > %dms", pack.ID, pack.Version, trace.LatencyMs, pack.MaxLatencyMs))
		}
	}
	return RunResult{Outputs: append([]RuleOutput(nil), trace.Outputs...), Trace: trace, Warnings: append([]string(nil), trace.Warnings...)}, nil
}

func validatePacks(packs []RulePack) error {
	seen := map[string]struct{}{}
	for _, pack := range packs {
		if strings.TrimSpace(pack.ID) == "" {
			return fmt.Errorf("rule pack id required")
		}
		if strings.TrimSpace(pack.Version) == "" {
			return fmt.Errorf("rule pack %s version required", pack.ID)
		}
		for _, rule := range pack.Rules {
			if strings.TrimSpace(rule.ID) == "" {
				return fmt.Errorf("rule id required in pack %s", pack.ID)
			}
			key := pack.ID + ":" + rule.ID
			if _, ok := seen[key]; ok {
				return fmt.Errorf("duplicate rule id %s in pack %s", rule.ID, pack.ID)
			}
			seen[key] = struct{}{}
		}
	}
	return nil
}

func conditionMatches(c Condition, facts map[string]any, nowMs int64) bool {
	if facts == nil {
		facts = map[string]any{}
	}
	value := facts[strings.TrimSpace(c.Field)]
	switch c.Operator {
	case OpEquals, OpStatusMatch, OpProviderStatusMatch, OpRiskClassMatch:
		return strings.EqualFold(strings.TrimSpace(fmt.Sprint(value)), strings.TrimSpace(fmt.Sprint(c.Value))) || stringInList(strings.TrimSpace(fmt.Sprint(value)), c.Values)
	case OpContains:
		haystack := strings.ToLower(fmt.Sprint(value))
		if len(c.Values) > 0 {
			for _, needle := range c.Values {
				if strings.Contains(haystack, strings.ToLower(strings.TrimSpace(needle))) {
					return true
				}
			}
			return false
		}
		return strings.Contains(haystack, strings.ToLower(strings.TrimSpace(fmt.Sprint(c.Value))))
	case OpNumericGT:
		return numberValue(value) > numberTarget(c)
	case OpNumericGTE:
		return numberValue(value) >= numberTarget(c)
	case OpNumericLT:
		return numberValue(value) < numberTarget(c)
	case OpNumericLTE:
		return numberValue(value) <= numberTarget(c)
	case OpAgeGTE:
		age := c.AgeMs
		if age <= 0 {
			age = int64(numberTarget(c))
		}
		ts := int64(numberValue(value))
		return ts > 0 && nowMs > ts && nowMs-ts >= age
	case OpTagMatch:
		return tagMatch(value, c.Value, c.Values)
	case OpTokenOverlapGTE:
		target := strings.TrimSpace(fmt.Sprint(c.Value))
		if len(c.Values) > 0 {
			target = strings.Join(c.Values, " ")
		}
		return tokenOverlap(fmt.Sprint(value), target) >= c.Threshold
	case OpBoolIs:
		want, _ := strconv.ParseBool(strings.TrimSpace(fmt.Sprint(c.Value)))
		return boolValue(value) == want
	default:
		return false
	}
}

func applyAuthoritySafety(output RuleOutput, authoritative string, trace *RuleTrace) RuleOutput {
	if !isHardAuthorityDecision(authoritative) || !isLooseningOutput(output) {
		return output
	}
	output.Decision = authoritative
	output.Type = OutputPolicyDecision
	if output.Metadata == nil {
		output.Metadata = map[string]any{}
	}
	output.Metadata["authority_conflict"] = true
	output.Metadata["authority_decision"] = authoritative
	output.Explain = strings.TrimSpace(output.Explain + "; authoritative denial wins")
	if trace != nil {
		trace.Warnings = append(trace.Warnings, "rule output could not loosen authoritative decision: "+authoritative)
	}
	return output
}

func isHardAuthorityDecision(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "deny", "denied", "reject", "rejected", "approval_required", "requires_approval", "degraded", "unavailable", "disabled", "scope_denied", "capability_denied":
		return true
	default:
		return false
	}
}

func isLooseningOutput(output RuleOutput) bool {
	decision := strings.ToLower(strings.TrimSpace(output.Decision))
	if decision == "" {
		return false
	}
	switch decision {
	case "allow", "allowed", "route", "proceed", "selected", "ok", "available":
		return true
	default:
		return false
	}
}

func outputTypeForAction(action string) OutputType {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "score_adjustment":
		return OutputScoreAdjustment
	case "fresh_compile_required":
		return OutputFreshCompileRequired
	case "reject":
		return OutputRejectDecision
	case "attention":
		return OutputAttentionSignal
	case "background_defer":
		return OutputBackgroundDefer
	case "model_routing_hint":
		return OutputModelRoutingHint
	case "policy":
		return OutputPolicyDecision
	default:
		return OutputRouteDecision
	}
}

func stableTraceID(in RunInput) string {
	base := strings.Join([]string{string(in.Lane), string(in.Phase), strings.TrimSpace(in.InputID)}, ":")
	sum := uint64(1469598103934665603)
	for _, b := range []byte(base) {
		sum ^= uint64(b)
		sum *= 1099511628211
	}
	return fmt.Sprintf("ruletrace:%x", sum)
}

func hasPackRef(refs []RulePackRef, id, version string) bool {
	for _, ref := range refs {
		if ref.ID == id && ref.Version == version {
			return true
		}
	}
	return false
}

func stringInList(value string, values []string) bool {
	for _, candidate := range values {
		if strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(candidate)) {
			return true
		}
	}
	return false
}

func numberTarget(c Condition) float64 {
	if c.Threshold != 0 {
		return c.Threshold
	}
	return numberValue(c.Value)
}

func numberValue(value any) float64 {
	switch v := value.(type) {
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case float64:
		return v
	case float32:
		return float64(v)
	case json.Number:
		f, _ := v.Float64()
		return f
	case string:
		f, _ := strconv.ParseFloat(strings.TrimSpace(v), 64)
		return f
	default:
		f, _ := strconv.ParseFloat(strings.TrimSpace(fmt.Sprint(value)), 64)
		return f
	}
}

func boolValue(value any) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		out, _ := strconv.ParseBool(strings.TrimSpace(v))
		return out
	default:
		out, _ := strconv.ParseBool(strings.TrimSpace(fmt.Sprint(value)))
		return out
	}
}

func tagMatch(value any, single any, many []string) bool {
	tags := map[string]struct{}{}
	switch v := value.(type) {
	case []string:
		for _, tag := range v {
			tags[strings.ToLower(strings.TrimSpace(tag))] = struct{}{}
		}
	case []any:
		for _, tag := range v {
			tags[strings.ToLower(strings.TrimSpace(fmt.Sprint(tag)))] = struct{}{}
		}
	default:
		for _, tag := range strings.Fields(strings.ReplaceAll(fmt.Sprint(value), ",", " ")) {
			tags[strings.ToLower(strings.TrimSpace(tag))] = struct{}{}
		}
	}
	if len(many) == 0 {
		many = []string{fmt.Sprint(single)}
	}
	for _, candidate := range many {
		if _, ok := tags[strings.ToLower(strings.TrimSpace(candidate))]; ok {
			return true
		}
	}
	return false
}

func tokenOverlap(a, b string) float64 {
	setA := tokenSet(a)
	setB := tokenSet(b)
	if len(setA) == 0 || len(setB) == 0 {
		return 0
	}
	intersection := 0
	for tok := range setA {
		if _, ok := setB[tok]; ok {
			intersection++
		}
	}
	union := len(setA) + len(setB) - intersection
	if union <= 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

func tokenSet(raw string) map[string]struct{} {
	raw = strings.ToLower(strings.TrimSpace(raw))
	raw = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsSpace(r) {
			return r
		}
		return ' '
	}, raw)
	out := map[string]struct{}{}
	for _, tok := range strings.Fields(raw) {
		if len(tok) > 1 {
			out[tok] = struct{}{}
		}
	}
	return out
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func Clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return math.Round(v*1000) / 1000
}
