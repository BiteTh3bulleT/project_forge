package dream

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"forge/projectforge/services/core/internal/aios/rulecells"
	"forge/projectforge/services/core/internal/store"
)

func TestDreamReplaySelectorFindsRecentCandidatesAndRespectsScope(t *testing.T) {
	ctx := context.Background()
	st, svc, now := newDreamHarness(t)
	insertDreamFixtures(t, st, now)

	report, err := svc.Run(ctx, RunRequest{Mode: ModeNap, WorkspaceID: "ws-dream", LaneID: "control.semantic"})
	if err != nil {
		t.Fatalf("dream run: %v", err)
	}
	if len(report.Candidates) < 5 {
		t.Fatalf("expected recent journal/context/memory/loop/contradiction candidates, got %d", len(report.Candidates))
	}
	for _, c := range report.Candidates {
		if c.WorkspaceID != "ws-dream" {
			t.Fatalf("wrong workspace candidate was included: %+v", c)
		}
		if c.LaneID != "" && c.LaneID != "control.semantic" {
			t.Fatalf("wrong lane candidate was included: %+v", c)
		}
	}
	if !hasSource(report.Candidates, "journal_event") || !hasSource(report.Candidates, "context_snapshot") || !hasSource(report.Candidates, "memory_note") {
		t.Fatalf("expected journal/context/memory candidates, got %+v", report.Candidates)
	}
}

func TestDreamConsumesRestoreOutcomeFeedback(t *testing.T) {
	ctx := context.Background()
	st, svc, now := newDreamHarness(t)
	mustExec(t, st, `INSERT INTO restore_outcome_events(id,created_at,updated_at,workspace_id,lane_id,query,context_packet_id,snapshot_id,snapshot_kind,restore_score,requires_fresh_compile,selected_evidence_json,selected_state_keys_json,selected_loop_ids_json,selected_artifact_ids_json,outcome,outcome_confidence,operator_feedback,correction_summary,downstream_action_type,downstream_object_id,metadata_json) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		"outcome-corrected", now-1000, now-1000, "ws-dream", "control.semantic", "restore blockers", "ctx-corrected", "snap-corrected", "restore", 0.7, 0, `["note-a"]`, `[]`, `[]`, `[]`, "operator_corrected", 1.0, "wrong context", "operator corrected restore", "compile_context", "ctx-corrected", `{}`)
	mustExec(t, st, `INSERT INTO restore_outcome_events(id,created_at,updated_at,workspace_id,lane_id,query,context_packet_id,snapshot_id,snapshot_kind,restore_score,requires_fresh_compile,selected_evidence_json,selected_state_keys_json,selected_loop_ids_json,selected_artifact_ids_json,outcome,outcome_confidence,operator_feedback,correction_summary,downstream_action_type,downstream_object_id,metadata_json) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		"outcome-gap-a", now-2000, now-2000, "ws-dream", "control.semantic", "missing memory", "ctx-gap-a", "", "restore", 0.2, 1, `[]`, `[]`, `[]`, `[]`, "fresh_compile_required", 1.0, "", "", "compile_context", "ctx-gap-a", `{}`)
	mustExec(t, st, `INSERT INTO restore_outcome_events(id,created_at,updated_at,workspace_id,lane_id,query,context_packet_id,snapshot_id,snapshot_kind,restore_score,requires_fresh_compile,selected_evidence_json,selected_state_keys_json,selected_loop_ids_json,selected_artifact_ids_json,outcome,outcome_confidence,operator_feedback,correction_summary,downstream_action_type,downstream_object_id,metadata_json) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		"outcome-helpful", now-3000, now-3000, "ws-dream", "control.semantic", "good restore", "ctx-helpful", "snap-helpful", "restore", 0.9, 0, `["note-good"]`, `[]`, `[]`, `[]`, "helpful", 1.0, "", "", "compile_context", "ctx-helpful", `{}`)

	report, err := svc.Run(ctx, RunRequest{Mode: ModeNap, WorkspaceID: "ws-dream", LaneID: "control.semantic", MaxCandidates: 20})
	if err != nil {
		t.Fatalf("dream run: %v", err)
	}
	if len(report.RestoreOutcomeCandidates) != 3 {
		t.Fatalf("expected restore outcome candidates, got %+v", report.RestoreOutcomeCandidates)
	}
	correctedCandidateID := ""
	for _, candidate := range report.RestoreOutcomeCandidates {
		if containsString(candidate.SourceIDs, "outcome-corrected") {
			correctedCandidateID = candidate.CandidateID
			break
		}
	}
	if correctedCandidateID == "" {
		t.Fatalf("missing corrected restore outcome candidate: %+v", report.RestoreOutcomeCandidates)
	}
	correctedScore := findScore(t, report.SalienceScores, correctedCandidateID)
	if correctedScore.TotalSalience < 0.88 {
		t.Fatalf("operator corrected outcome should be very high salience, got %+v", correctedScore)
	}
	if len(report.MemoryGapProposals) == 0 {
		t.Fatalf("expected memory gap proposal from fresh compile outcome")
	}
	if len(report.StaleEvidenceReviewProposals) == 0 {
		t.Fatalf("expected review proposal from operator corrected outcome")
	}
	if len(report.HelpfulEvidencePromotionProposals) == 0 {
		t.Fatalf("expected helpful evidence promotion proposal")
	}
}

func TestDreamSalienceAndRoutingPolicy(t *testing.T) {
	now := int64(1760000000000)
	candidates := []ReplayCandidate{
		replayCandidate("journal_event", "correction", "ws", "lane", now-1000, "user correction corrected preference", []string{"correction", "preference"}, "", ""),
		replayCandidate("contradiction", "conflict", "ws", "lane", now-2000, "unresolved contradiction", []string{"contradiction", "unresolved"}, "", ""),
		replayCandidate("open_loop", "blocker", "ws", "lane", now-3000, "blocked active loop", []string{"active", "blocker", "blocked"}, "", ""),
		replayCandidate("memory_note", "noise-a", "ws", "lane", now-4000, "routine duplicate noise", nil, "", ""),
		replayCandidate("memory_note", "noise-b", "ws", "lane", now-5000, "routine duplicate noise", nil, "", ""),
	}
	candidates[2].RelatedLoopIDs = []string{"loop-1"}

	scores := ScoreCandidates(candidates, now)
	correction := findScore(t, scores, "correction")
	plain := findScore(t, scores, "noise-a")
	if correction.CorrectionValueScore <= plain.CorrectionValueScore || correction.TotalSalience <= plain.TotalSalience {
		t.Fatalf("expected user correction to outrank plain candidate: correction=%+v plain=%+v", correction, plain)
	}
	if contradiction := findScore(t, scores, "conflict"); contradiction.ContradictionScore == 0 {
		t.Fatalf("expected contradiction score: %+v", contradiction)
	}
	if blocker := findScore(t, scores, "blocker"); blocker.GoalRelevanceScore == 0 || blocker.OutcomeImpactScore == 0 {
		t.Fatalf("expected active blocker/open loop salience: %+v", blocker)
	}

	routes := RouteCandidates(candidates, scores, RunRequest{}, true)
	if route := findRoute(t, routes, "noise-a"); route.Decision != Demote && route.Decision != Discard && route.Decision != Noop {
		t.Fatalf("expected low-salience redundant candidate to avoid promotion, got %+v", route)
	}

	longTerm := RouteCandidates(candidates, []SalienceScore{{
		CandidateID:        candidates[0].CandidateID,
		TotalSalience:      0.9,
		Confidence:         0.9,
		ContradictionScore: 0,
	}}, RunRequest{AllowLongTermPromotion: true, RequireOperatorReviewForLongTerm: false}, true)
	if longTerm[0].Decision != PromoteLongTerm {
		t.Fatalf("expected high confidence non-contradictory long-term promotion, got %+v", longTerm[0])
	}
	needsReview := RouteCandidates(candidates, []SalienceScore{{
		CandidateID:        candidates[1].CandidateID,
		TotalSalience:      0.95,
		Confidence:         0.7,
		ContradictionScore: 1,
	}}, RunRequest{AllowLongTermPromotion: true}, true)
	if needsReview[0].Decision != NeedsReview {
		t.Fatalf("expected contradiction to require review, got %+v", needsReview[0])
	}
}

func TestDreamRunDryRunModesAndNoCanonicalCommits(t *testing.T) {
	ctx := context.Background()
	st, svc, now := newDreamHarness(t)
	insertDreamFixtures(t, st, now)
	before := tableCounts(t, st)

	micro, err := svc.Run(ctx, RunRequest{Mode: ModeMicrodream, WorkspaceID: "ws-dream", LaneID: "control.semantic"})
	if err != nil {
		t.Fatalf("microdream: %v", err)
	}
	nap, err := svc.Run(ctx, RunRequest{Mode: ModeNap, WorkspaceID: "ws-dream", LaneID: "control.semantic"})
	if err != nil {
		t.Fatalf("nap: %v", err)
	}
	deep, err := svc.Run(ctx, RunRequest{Mode: ModeDeepDream, WorkspaceID: "ws-dream", LaneID: "control.semantic", AllowCommits: true})
	if err != nil {
		t.Fatalf("deep dream: %v", err)
	}
	falseDryRun := false
	explicitNonDry, err := svc.Run(ctx, RunRequest{Mode: ModeMicrodream, WorkspaceID: "ws-dream", LaneID: "control.semantic", DryRun: &falseDryRun})
	if err != nil {
		t.Fatalf("explicit non-dry dream: %v", err)
	}
	if !micro.Run.DryRun || !nap.Run.DryRun || !deep.Run.DryRun || !explicitNonDry.Run.DryRun {
		t.Fatalf("Dream Mode v0 must be dry-run by default and with allowCommits")
	}
	if len(explicitNonDry.Warnings) == 0 {
		t.Fatalf("expected warning when Dream Mode v0 ignores dryRun=false")
	}
	if micro.Run.WindowEnd-micro.Run.WindowStart >= nap.Run.WindowEnd-nap.Run.WindowStart {
		t.Fatalf("expected microdream to use shorter window than nap")
	}
	if nap.Run.WindowEnd-nap.Run.WindowStart >= deep.Run.WindowEnd-deep.Run.WindowStart {
		t.Fatalf("expected nap to use shorter window than deep_dream")
	}
	if micro.Run.CandidatesConsidered > 8 || nap.Run.CandidatesConsidered > 24 || deep.Run.CandidatesConsidered > 80 {
		t.Fatalf("mode candidate limits not respected: micro=%d nap=%d deep=%d", micro.Run.CandidatesConsidered, nap.Run.CandidatesConsidered, deep.Run.CandidatesConsidered)
	}
	if len(deep.ProposedTierRouting) == 0 || len(deep.SalienceScores) == 0 || len(deep.Trace) == 0 {
		t.Fatalf("dry-run report missing candidates, scores, proposals, or trace: %+v", deep)
	}
	if got := tableCounts(t, st); got != before {
		t.Fatalf("Dream Mode dry-run committed canonical changes: before=%v after=%v", before, got)
	}
	if deep.Trace["gpu_required"] != false || deep.Trace["modelruntime_required"] != false {
		t.Fatalf("expected no GPU/modelruntime dependency trace, got %+v", deep.Trace)
	}
	if err := svc.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if err := svc.Close(); err != nil {
		t.Fatalf("second close should be idempotent: %v", err)
	}
}

func TestDreamRuleCellsBoostCorrectionAndTracePackVersion(t *testing.T) {
	ctx := context.Background()
	st, svc, now := newDreamHarness(t)
	insertDreamFixtures(t, st, now)
	svc.SetRuleEngine(rulecells.MustStaticEngine())

	report, err := svc.Run(ctx, RunRequest{Mode: ModeNap, WorkspaceID: "ws-dream", LaneID: "control.semantic"})
	if err != nil {
		t.Fatalf("dream run: %v", err)
	}
	correction := findScore(t, report.SalienceScores, "evt-correction")
	if correction.RuleSalienceAdjustment <= 0 || correction.TotalSalience <= correction.PreRuleTotalSalience {
		t.Fatalf("expected correction salience boost, got %+v", correction)
	}
	if correction.RuleTrace == nil {
		t.Fatalf("expected correction rule trace")
	}
	packs, ok := correction.RuleTrace["rule_packs"].([]map[string]any)
	if !ok || len(packs) != 1 || packs[0]["pack_id"] != rulecells.PackLymphaticDreamID || packs[0]["version"] != rulecells.StaticPackVersion {
		t.Fatalf("expected lymphatic pack version trace, got %#v", correction.RuleTrace["rule_packs"])
	}
	if _, ok := report.Trace["rule_cells"]; !ok {
		t.Fatalf("expected report rule cell trace, got %+v", report.Trace)
	}
}

func TestDreamRuleCellsBlockLongTermPromotionOnContradiction(t *testing.T) {
	now := int64(1760000000000)
	candidate := replayCandidate("memory_note", "contradictory", "ws", "lane", now-1000, "unresolved contradiction", []string{"contradiction", "unresolved"}, "", "")
	scores := []SalienceScore{{
		CandidateID:        candidate.CandidateID,
		ContradictionScore: 1,
		TotalSalience:      0.95,
		Confidence:         0.95,
	}}
	svc := NewService(nil)
	svc.SetRuleEngine(rulecells.MustStaticEngine())
	_, blocked, warnings := svc.applyRuleCells(context.Background(), []ReplayCandidate{candidate}, scores, now)
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %+v", warnings)
	}
	if !blocked[candidate.CandidateID] {
		t.Fatalf("expected contradiction to block long-term promotion")
	}
	routes := []RoutingProposal{{CandidateID: candidate.CandidateID, Decision: PromoteLongTerm, Confidence: 0.95, DryRun: true}}
	applyLongTermBlocks(routes, blocked)
	if routes[0].Decision != NeedsReview || !strings.Contains(routes[0].Reason, "rule cell") {
		t.Fatalf("expected stricter routing to needs_review, got %+v", routes[0])
	}
}

func TestDreamRuleCellsCapAndClampSalience(t *testing.T) {
	ctx := context.Background()
	st, svc, now := newDreamHarness(t)
	insertDreamFixtures(t, st, now)
	svc.SetRuleEngine(dreamStaticRuleEngine{result: rulecells.RunResult{
		Outputs: []rulecells.RuleOutput{
			{Type: rulecells.OutputScoreAdjustment, ScoreDelta: 2.0},
			{Type: rulecells.OutputScoreAdjustment, ScoreDelta: 2.0},
			{Type: rulecells.OutputScoreAdjustment, ScoreDelta: 2.0},
		},
		Trace: rulecells.RuleTrace{
			TraceID:      "trace-dream-cap",
			Lane:         rulecells.LaneLymphatic,
			Phase:        rulecells.PhaseSalienceScoring,
			InputID:      "candidate",
			RulePacks:    []rulecells.RulePackRef{{ID: "pack.dream.cap", Version: "0.1.0"}},
			MatchedRules: []rulecells.MatchedRuleTrace{{RuleID: "rule.dream.cap", PackID: "pack.dream.cap", PackVersion: "0.1.0"}},
			Outputs:      []rulecells.RuleOutput{{Type: rulecells.OutputScoreAdjustment, ScoreDelta: 2.0}},
		},
	}})

	report, err := svc.Run(ctx, RunRequest{Mode: ModeNap, WorkspaceID: "ws-dream", LaneID: "control.semantic"})
	if err != nil {
		t.Fatalf("dream run: %v", err)
	}
	if len(report.SalienceScores) == 0 {
		t.Fatalf("expected salience scores")
	}
	for _, score := range report.SalienceScores {
		if score.RuleSalienceAdjustment != maxDreamRuleSalienceAdjustment {
			t.Fatalf("expected capped salience adjustment %.2f, got %+v", maxDreamRuleSalienceAdjustment, score)
		}
		if score.TotalSalience < 0 || score.TotalSalience > 1 {
			t.Fatalf("expected final salience clamped to 0..1, got %+v", score)
		}
	}
}

func TestDreamRuleEngineErrorFallsBackWithWarning(t *testing.T) {
	ctx := context.Background()
	st, svc, now := newDreamHarness(t)
	insertDreamFixtures(t, st, now)
	before := tableCounts(t, st)
	svc.SetRuleEngine(dreamStaticRuleEngine{err: errors.New("boom")})

	report, err := svc.Run(ctx, RunRequest{Mode: ModeNap, WorkspaceID: "ws-dream", LaneID: "control.semantic"})
	if err != nil {
		t.Fatalf("dream run should not fail on rule engine error: %v", err)
	}
	if len(report.Warnings) == 0 || !strings.Contains(strings.Join(report.Warnings, "\n"), "dream rule engine failed") {
		t.Fatalf("expected explicit rule engine warning, got %+v", report.Warnings)
	}
	if got := tableCounts(t, st); got != before {
		t.Fatalf("Dream rule failure fallback committed canonical changes: before=%v after=%v", before, got)
	}
}

func TestDreamRuleEngineNilKeepsBaseSalience(t *testing.T) {
	ctx := context.Background()
	st, svc, now := newDreamHarness(t)
	insertDreamFixtures(t, st, now)

	report, err := svc.Run(ctx, RunRequest{Mode: ModeNap, WorkspaceID: "ws-dream", LaneID: "control.semantic"})
	if err != nil {
		t.Fatalf("dream run: %v", err)
	}
	for _, score := range report.SalienceScores {
		if score.RuleSalienceAdjustment != 0 || score.RuleTrace != nil {
			t.Fatalf("nil rule engine should preserve base salience, got %+v", score)
		}
	}
	if _, ok := report.Trace["rule_cells"]; ok {
		t.Fatalf("nil rule engine should not emit rule trace, got %+v", report.Trace)
	}
}

func TestDreamNoModelRuntimeVectorOrControlLaneBypass(t *testing.T) {
	raw, err := os.ReadFile("service.go")
	if err != nil {
		t.Fatalf("read service.go: %v", err)
	}
	text := string(raw)
	for _, forbidden := range []string{"internal/modelruntime", "internal/retrieval", "semantic_score", "vector"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("dream service must not depend on %s", forbidden)
		}
	}
	if strings.Contains(text, "INSERT INTO memory_notes") || strings.Contains(text, "UPDATE memory_notes") || strings.Contains(text, "DELETE FROM memory_notes") {
		t.Fatalf("dream service must not mutate canonical memory tables")
	}
}

func newDreamHarness(t *testing.T) (*store.Store, *Service, int64) {
	t.Helper()
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	now := int64(1760000000000)
	svc := NewService(st.DB)
	svc.SetClock(func() time.Time { return time.UnixMilli(now) })
	return st, svc, now
}

func insertDreamFixtures(t *testing.T, st *store.Store, now int64) {
	t.Helper()
	mustExec(t, st, `INSERT INTO journal_events(id,type,source,actor,workspace_id,lane_id,payload_json,created_at,metadata_json) VALUES(?,?,?,?,?,?,?,?,?)`, "evt-correction", "user_correction", "operator", "user", "ws-dream", "control.semantic", `{"text":"corrected preference"}`, now-1000, `{}`)
	mustExec(t, st, `INSERT INTO journal_events(id,type,source,actor,workspace_id,lane_id,payload_json,created_at,metadata_json) VALUES(?,?,?,?,?,?,?,?,?)`, "evt-other", "user_correction", "operator", "user", "ws-other", "control.semantic", `{"text":"wrong workspace"}`, now-1000, `{}`)
	mustExec(t, st, `INSERT INTO context_packet_snapshots(id,query,workspace_id,lane_id,snapshot_kind,restore_scores_json,resume_hints_json,created_at) VALUES(?,?,?,?,?,?,?,?)`, "ctx-restore", "restore blockers", "ws-dream", "control.semantic", "restore", `{"decision":"fresh_compile_below_threshold"}`, `{"requires_fresh_compile":true}`, now-2000)
	mustExec(t, st, `INSERT INTO memory_notes(id,type,title,content,workspace_id,lane_id,confidence,status,created_at,updated_at,metadata_json) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, "note-correction", "fact", "Correction", "user correction should be remembered", "ws-dream", "control.semantic", 0.9, "active", now-3000, now-3000, `{}`)
	mustExec(t, st, `INSERT INTO state_items(id,key,value_json,workspace_id,lane_id,status,updated_at,metadata_json) VALUES(?,?,?,?,?,?,?,?)`, "state-1", "runtime.safe_mode", `{"enabled":true}`, "ws-dream", "control.semantic", "active", now-3500, `{}`)
	mustExec(t, st, `INSERT INTO open_loops(id,title,state,priority,owner,blocker,next_action,workspace_id,lane_id,created_at,updated_at,metadata_json) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`, "loop-blocker", "Blocked loop", "open", "high", "forge", "blocked on restore", "score candidates", "ws-dream", "control.semantic", now-4000, now-4000, `{}`)
	mustExec(t, st, `INSERT INTO contradiction_records(id,left_object_id,right_object_id,reason,severity,confidence,workspace_id,lane_id,created_at,metadata_json) VALUES(?,?,?,?,?,?,?,?,?,?)`, "contradiction-1", "note-a", "note-b", "unresolved contradiction", "high", 0.8, "ws-dream", "control.semantic", now-5000, `{}`)
	mustExec(t, st, `INSERT INTO artifact_refs(id,type,uri,workspace_id,lane_id,created_at,metadata_json) VALUES(?,?,?,?,?,?,?)`, "artifact-1", "restore_report", "artifact://restore", "ws-dream", "control.semantic", now-6000, `{}`)
}

func mustExec(t *testing.T, st *store.Store, query string, args ...any) {
	t.Helper()
	if _, err := st.DB.Exec(query, args...); err != nil {
		t.Fatalf("exec %s: %v", query, err)
	}
}

func tableCounts(t *testing.T, st *store.Store) string {
	t.Helper()
	tables := []string{"journal_events", "context_packet_snapshots", "memory_notes", "state_items", "open_loops", "contradiction_records", "artifact_refs"}
	parts := make([]string, 0, len(tables))
	for _, table := range tables {
		var count int
		if err := st.DB.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		parts = append(parts, table+"="+strconv.Itoa(count))
	}
	return strings.Join(parts, ",")
}

func hasSource(candidates []ReplayCandidate, source string) bool {
	for _, c := range candidates {
		if c.SourceType == source {
			return true
		}
	}
	return false
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func findScore(t *testing.T, scores []SalienceScore, idPart string) SalienceScore {
	t.Helper()
	for _, score := range scores {
		if strings.Contains(score.CandidateID, stableID("dream", "journal_event", idPart)) || strings.Contains(score.CandidateID, stableID("dream", "memory_note", idPart)) || strings.Contains(score.CandidateID, stableID("dream", "contradiction", idPart)) || strings.Contains(score.CandidateID, stableID("dream", "open_loop", idPart)) {
			return score
		}
	}
	for _, score := range scores {
		if strings.Contains(score.CandidateID, idPart) {
			return score
		}
	}
	t.Fatalf("score containing %q not found in %+v", idPart, scores)
	return SalienceScore{}
}

func findRoute(t *testing.T, routes []RoutingProposal, idPart string) RoutingProposal {
	t.Helper()
	for _, route := range routes {
		if strings.Contains(route.CandidateID, stableID("dream", "memory_note", idPart)) || strings.Contains(route.CandidateID, stableID("dream", "journal_event", idPart)) {
			return route
		}
	}
	t.Fatalf("route containing %q not found in %+v", idPart, routes)
	return RoutingProposal{}
}

type dreamStaticRuleEngine struct {
	result rulecells.RunResult
	err    error
}

func (s dreamStaticRuleEngine) Run(context.Context, rulecells.RunInput, rulecells.RunOptions) (rulecells.RunResult, error) {
	return s.result, s.err
}
