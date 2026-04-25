package dream

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
	"unicode"
)

type Mode string

const (
	ModeMicrodream Mode = "microdream"
	ModeNap        Mode = "nap"
	ModeDeepDream  Mode = "deep_dream"
)

type Service struct {
	db    *sql.DB
	clock func() time.Time
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db, clock: time.Now}
}

func (s *Service) SetClock(clock func() time.Time) {
	if clock != nil {
		s.clock = clock
	}
}

func (s *Service) Close() error { return nil }

type RunRequest struct {
	Mode                             Mode   `json:"mode"`
	WorkspaceID                      string `json:"workspaceId"`
	LaneID                           string `json:"laneId,omitempty"`
	WindowHours                      int    `json:"windowHours,omitempty"`
	MaxCandidates                    int    `json:"maxCandidates,omitempty"`
	DryRun                           *bool  `json:"dryRun,omitempty"`
	AllowLongTermPromotion           bool   `json:"allowLongTermPromotion,omitempty"`
	RequireOperatorReviewForLongTerm bool   `json:"requireOperatorReviewForLongTerm"`
	AllowCommits                     bool   `json:"allowCommits,omitempty"`
	CorrelationID                    string `json:"correlationId,omitempty"`
	TraceID                          string `json:"traceId,omitempty"`
}

type DreamRun struct {
	RunID                string `json:"run_id"`
	Mode                 Mode   `json:"mode"`
	WorkspaceID          string `json:"workspace_id"`
	LaneID               string `json:"lane_id,omitempty"`
	WindowStart          int64  `json:"window_start"`
	WindowEnd            int64  `json:"window_end"`
	DryRun               bool   `json:"dry_run"`
	StartedAt            int64  `json:"started_at"`
	CompletedAt          int64  `json:"completed_at"`
	Status               string `json:"status"`
	CandidatesConsidered int    `json:"candidates_considered"`
	ProposalsGenerated   int    `json:"proposals_generated"`
	Summary              string `json:"summary"`
	CorrelationID        string `json:"correlation_id,omitempty"`
	TraceID              string `json:"trace_id,omitempty"`
}

type ReplayCandidate struct {
	CandidateID          string             `json:"candidate_id"`
	SourceType           string             `json:"source_type"`
	SourceIDs            []string           `json:"source_ids"`
	WorkspaceID          string             `json:"workspace_id"`
	LaneID               string             `json:"lane_id,omitempty"`
	StartTimestamp       int64              `json:"start_timestamp"`
	EndTimestamp         int64              `json:"end_timestamp"`
	ContentSummary       string             `json:"content_summary"`
	Tags                 []string           `json:"tags"`
	RelatedGoalIDs       []string           `json:"related_goal_ids"`
	RelatedLoopIDs       []string           `json:"related_loop_ids"`
	RelatedSnapshotIDs   []string           `json:"related_snapshot_ids"`
	RawImportanceSignals map[string]float64 `json:"raw_importance_signals"`
	Trace                map[string]string  `json:"trace"`
}

type SalienceScore struct {
	CandidateID           string   `json:"candidate_id"`
	NoveltyScore          float64  `json:"novelty_score"`
	RepetitionScore       float64  `json:"repetition_score"`
	GoalRelevanceScore    float64  `json:"goal_relevance_score"`
	CorrectionValueScore  float64  `json:"correction_value_score"`
	OutcomeImpactScore    float64  `json:"outcome_impact_score"`
	ContradictionScore    float64  `json:"contradiction_score"`
	RetrievalUtilityScore float64  `json:"retrieval_utility_score"`
	RecencyScore          float64  `json:"recency_score"`
	TotalSalience         float64  `json:"total_salience"`
	Confidence            float64  `json:"confidence"`
	Explain               []string `json:"explain"`
}

type TierDecision string

const (
	RetainShortTerm TierDecision = "retain_short_term"
	PromoteMidTerm  TierDecision = "promote_mid_term"
	PromoteLongTerm TierDecision = "promote_long_term"
	Demote          TierDecision = "demote"
	Merge           TierDecision = "merge"
	Discard         TierDecision = "discard"
	NeedsReview     TierDecision = "needs_review"
	RepairRequired  TierDecision = "repair_required"
	Noop            TierDecision = "no_op"
)

type RoutingProposal struct {
	CandidateID string       `json:"candidate_id"`
	SourceType  string       `json:"source_type,omitempty"`
	Decision    TierDecision `json:"decision"`
	Confidence  float64      `json:"confidence"`
	Reason      string       `json:"reason"`
	DryRun      bool         `json:"dry_run"`
}

type DreamReport struct {
	Run                             DreamRun          `json:"run"`
	Candidates                      []ReplayCandidate `json:"candidates"`
	SalienceScores                  []SalienceScore   `json:"salience_scores"`
	ProposedTierRouting             []RoutingProposal `json:"proposed_tier_routing"`
	ProposedMemoryActions           []RoutingProposal `json:"proposed_memory_actions"`
	ProposedSnapshotHygieneActions  []RoutingProposal `json:"proposed_snapshot_hygiene_actions"`
	ProposedRestoreScoreUpdates     []RoutingProposal `json:"proposed_restore_score_updates"`
	ProposedEmbeddingRefreshActions []RoutingProposal `json:"proposed_embedding_refresh_actions"`
	ProposedRepairActions           []RoutingProposal `json:"proposed_repair_actions"`
	ItemsRequiringReview            []RoutingProposal `json:"items_requiring_review"`
	NoOpReasons                     []string          `json:"no_op_reasons"`
	Warnings                        []string          `json:"warnings"`
	Trace                           map[string]any    `json:"trace"`
}

func (s *Service) Run(ctx context.Context, req RunRequest) (DreamReport, error) {
	if s == nil || s.db == nil {
		return DreamReport{}, fmt.Errorf("dream service requires sqlite store")
	}
	req = normalizeRequest(req)
	if strings.TrimSpace(req.WorkspaceID) == "" {
		return DreamReport{}, fmt.Errorf("workspaceId required")
	}
	now := s.clock().UnixMilli()
	windowStart := now - int64(req.WindowHours)*60*60*1000
	dryRun := true
	started := now
	candidates, err := s.SelectReplayCandidates(ctx, req, windowStart, now)
	if err != nil {
		return DreamReport{}, err
	}
	scores := ScoreCandidates(candidates, now)
	routing := RouteCandidates(candidates, scores, req, dryRun)
	report := DreamReport{
		Run: DreamRun{
			RunID:                stableID("dream", req.WorkspaceID, req.LaneID, string(req.Mode), fmt.Sprint(started)),
			Mode:                 req.Mode,
			WorkspaceID:          req.WorkspaceID,
			LaneID:               req.LaneID,
			WindowStart:          windowStart,
			WindowEnd:            now,
			DryRun:               dryRun,
			StartedAt:            started,
			CompletedAt:          s.clock().UnixMilli(),
			Status:               "completed",
			CandidatesConsidered: len(candidates),
			Summary:              fmt.Sprintf("%s considered %d replay candidate(s) and generated deterministic dry-run proposals", req.Mode, len(candidates)),
			CorrelationID:        req.CorrelationID,
			TraceID:              req.TraceID,
		},
		Candidates:            candidates,
		SalienceScores:        scores,
		ProposedTierRouting:   routing,
		ProposedMemoryActions: routing,
		Trace: map[string]any{
			"cpu_only":                  true,
			"modelruntime_required":     false,
			"gpu_required":              false,
			"canonical_write_committed": false,
			"window_hours":              req.WindowHours,
			"max_candidates":            req.MaxCandidates,
		},
	}
	for _, proposal := range routing {
		switch proposal.Decision {
		case NeedsReview:
			report.ItemsRequiringReview = append(report.ItemsRequiringReview, proposal)
		case RepairRequired:
			report.ProposedRepairActions = append(report.ProposedRepairActions, proposal)
		case Noop, Discard:
			report.NoOpReasons = append(report.NoOpReasons, proposal.CandidateID+": "+proposal.Reason)
		}
		if proposal.SourceType == "context_snapshot" {
			report.ProposedSnapshotHygieneActions = append(report.ProposedSnapshotHygieneActions, proposal)
			report.ProposedRestoreScoreUpdates = append(report.ProposedRestoreScoreUpdates, proposal)
		}
		if proposal.SourceType == "memory_note" || proposal.SourceType == "context_snapshot" {
			report.ProposedEmbeddingRefreshActions = append(report.ProposedEmbeddingRefreshActions, proposal)
		}
	}
	report.Run.ProposalsGenerated = len(routing)
	if req.AllowCommits {
		report.Warnings = append(report.Warnings, "allowCommits ignored in Dream Mode v0; report is dry-run only")
	}
	if req.DryRun != nil && !*req.DryRun {
		report.Warnings = append(report.Warnings, "dryRun=false ignored in Dream Mode v0; report is dry-run only")
	}
	return report, nil
}

func normalizeRequest(req RunRequest) RunRequest {
	if req.Mode == "" {
		req.Mode = ModeMicrodream
	}
	switch req.Mode {
	case ModeMicrodream:
		if req.WindowHours <= 0 {
			req.WindowHours = 6
		}
		if req.MaxCandidates <= 0 {
			req.MaxCandidates = 8
		}
	case ModeNap:
		if req.WindowHours <= 0 {
			req.WindowHours = 24
		}
		if req.MaxCandidates <= 0 {
			req.MaxCandidates = 24
		}
	case ModeDeepDream:
		if req.WindowHours <= 0 {
			req.WindowHours = 168
		}
		if req.MaxCandidates <= 0 {
			req.MaxCandidates = 80
		}
	default:
		req.Mode = ModeMicrodream
		if req.WindowHours <= 0 {
			req.WindowHours = 6
		}
		if req.MaxCandidates <= 0 {
			req.MaxCandidates = 8
		}
	}
	if !req.AllowLongTermPromotion {
		req.RequireOperatorReviewForLongTerm = true
	}
	return req
}

func (s *Service) SelectReplayCandidates(ctx context.Context, req RunRequest, since, until int64) ([]ReplayCandidate, error) {
	candidates := []ReplayCandidate{}
	loaders := []func(context.Context, RunRequest, int64, int64) ([]ReplayCandidate, error){
		s.loadJournalEvents,
		s.loadContextSnapshots,
		s.loadMemoryNotes,
		s.loadStateItems,
		s.loadOpenLoops,
		s.loadContradictions,
		s.loadArtifacts,
	}
	for _, load := range loaders {
		rows, err := load(ctx, req, since, until)
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, rows...)
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].EndTimestamp == candidates[j].EndTimestamp {
			return candidates[i].CandidateID < candidates[j].CandidateID
		}
		return candidates[i].EndTimestamp > candidates[j].EndTimestamp
	})
	if len(candidates) > req.MaxCandidates {
		candidates = candidates[:req.MaxCandidates]
	}
	return candidates, nil
}

func (s *Service) loadJournalEvents(ctx context.Context, req RunRequest, since, until int64) ([]ReplayCandidate, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,type,source,actor,workspace_id,lane_id,payload_json,metadata_json,correlation_id,trace_id,created_at FROM journal_events WHERE workspace_id=? AND (?='' OR lane_id='' OR lane_id=?) AND created_at BETWEEN ? AND ? ORDER BY created_at DESC LIMIT ?`, req.WorkspaceID, req.LaneID, req.LaneID, since, until, req.MaxCandidates)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ReplayCandidate{}
	for rows.Next() {
		var id, typ, source, actor, ws, lane, payload, meta, corr, trace string
		var ts int64
		if err := rows.Scan(&id, &typ, &source, &actor, &ws, &lane, &payload, &meta, &corr, &trace, &ts); err != nil {
			return nil, err
		}
		tags := tagsFromText(typ + " " + source + " " + payload + " " + meta)
		out = append(out, replayCandidate("journal_event", id, ws, lane, ts, typ+" "+summaryJSON(payload), tags, corr, trace))
	}
	return out, rows.Err()
}

func (s *Service) loadContextSnapshots(ctx context.Context, req RunRequest, since, until int64) ([]ReplayCandidate, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,query,workspace_id,lane_id,snapshot_kind,restore_scores_json,resume_hints_json,correlation_id,trace_id,created_at FROM context_packet_snapshots WHERE workspace_id=? AND (?='' OR lane_id='' OR lane_id=?) AND created_at BETWEEN ? AND ? ORDER BY created_at DESC LIMIT ?`, req.WorkspaceID, req.LaneID, req.LaneID, since, until, req.MaxCandidates)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ReplayCandidate{}
	for rows.Next() {
		var id, query, ws, lane, kind, scores, hints, corr, trace string
		var ts int64
		if err := rows.Scan(&id, &query, &ws, &lane, &kind, &scores, &hints, &corr, &trace, &ts); err != nil {
			return nil, err
		}
		tags := tagsFromText(kind + " " + scores + " " + hints)
		c := replayCandidate("context_snapshot", id, ws, lane, ts, query, tags, corr, trace)
		c.RelatedSnapshotIDs = []string{id}
		if strings.Contains(scores+hints, "fresh_compile") || strings.Contains(scores+hints, "below_threshold") {
			c.RawImportanceSignals["failed_restore"] = 1
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Service) loadMemoryNotes(ctx context.Context, req RunRequest, since, until int64) ([]ReplayCandidate, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,type,title,content,workspace_id,lane_id,confidence,status,metadata_json,correlation_id,trace_id,updated_at FROM memory_notes WHERE workspace_id=? AND (?='' OR lane_id='' OR lane_id=?) AND updated_at BETWEEN ? AND ? ORDER BY updated_at DESC LIMIT ?`, req.WorkspaceID, req.LaneID, req.LaneID, since, until, req.MaxCandidates)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ReplayCandidate{}
	for rows.Next() {
		var id, typ, title, content, ws, lane, status, meta, corr, trace string
		var confidence float64
		var ts int64
		if err := rows.Scan(&id, &typ, &title, &content, &ws, &lane, &confidence, &status, &meta, &corr, &trace, &ts); err != nil {
			return nil, err
		}
		c := replayCandidate("memory_note", id, ws, lane, ts, title+" "+content, tagsFromText(typ+" "+status+" "+title+" "+content+" "+meta), corr, trace)
		c.RawImportanceSignals["confidence"] = clamp01(confidence)
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Service) loadStateItems(ctx context.Context, req RunRequest, since, until int64) ([]ReplayCandidate, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,key,value_json,workspace_id,lane_id,status,metadata_json,correlation_id,trace_id,updated_at FROM state_items WHERE workspace_id=? AND (?='' OR lane_id='' OR lane_id=?) AND updated_at BETWEEN ? AND ? ORDER BY updated_at DESC LIMIT ?`, req.WorkspaceID, req.LaneID, req.LaneID, since, until, req.MaxCandidates)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ReplayCandidate{}
	for rows.Next() {
		var id, key, value, ws, lane, status, meta, corr, trace string
		var ts int64
		if err := rows.Scan(&id, &key, &value, &ws, &lane, &status, &meta, &corr, &trace, &ts); err != nil {
			return nil, err
		}
		out = append(out, replayCandidate("state_item", id, ws, lane, ts, key+" "+summaryJSON(value), tagsFromText(status+" "+key+" "+value+" "+meta), corr, trace))
	}
	return out, rows.Err()
}

func (s *Service) loadOpenLoops(ctx context.Context, req RunRequest, since, until int64) ([]ReplayCandidate, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,title,state,priority,blocker,next_action,workspace_id,lane_id,metadata_json,correlation_id,trace_id,updated_at FROM open_loops WHERE workspace_id=? AND (?='' OR lane_id='' OR lane_id=?) AND updated_at BETWEEN ? AND ? ORDER BY updated_at DESC LIMIT ?`, req.WorkspaceID, req.LaneID, req.LaneID, since, until, req.MaxCandidates)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ReplayCandidate{}
	for rows.Next() {
		var id, title, state, priority, blocker, next, ws, lane, meta, corr, trace string
		var ts int64
		if err := rows.Scan(&id, &title, &state, &priority, &blocker, &next, &ws, &lane, &meta, &corr, &trace, &ts); err != nil {
			return nil, err
		}
		c := replayCandidate("open_loop", id, ws, lane, ts, title+" "+blocker+" "+next, tagsFromText(state+" "+priority+" "+blocker+" "+meta), corr, trace)
		c.RelatedLoopIDs = []string{id}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Service) loadContradictions(ctx context.Context, req RunRequest, since, until int64) ([]ReplayCandidate, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,left_object_id,right_object_id,reason,severity,confidence,workspace_id,lane_id,metadata_json,correlation_id,trace_id,created_at FROM contradiction_records WHERE workspace_id=? AND (?='' OR lane_id='' OR lane_id=?) AND created_at BETWEEN ? AND ? ORDER BY created_at DESC LIMIT ?`, req.WorkspaceID, req.LaneID, req.LaneID, since, until, req.MaxCandidates)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ReplayCandidate{}
	for rows.Next() {
		var id, left, right, reason, severity, ws, lane, meta, corr, trace string
		var confidence float64
		var ts int64
		if err := rows.Scan(&id, &left, &right, &reason, &severity, &confidence, &ws, &lane, &meta, &corr, &trace, &ts); err != nil {
			return nil, err
		}
		c := replayCandidate("contradiction", id, ws, lane, ts, reason+" "+left+" "+right, tagsFromText("contradiction unresolved "+severity+" "+meta), corr, trace)
		c.RawImportanceSignals["confidence"] = clamp01(confidence)
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Service) loadArtifacts(ctx context.Context, req RunRequest, since, until int64) ([]ReplayCandidate, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,type,uri,workspace_id,lane_id,metadata_json,correlation_id,trace_id,created_at FROM artifact_refs WHERE workspace_id=? AND (?='' OR lane_id='' OR lane_id=?) AND created_at BETWEEN ? AND ? ORDER BY created_at DESC LIMIT ?`, req.WorkspaceID, req.LaneID, req.LaneID, since, until, req.MaxCandidates)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ReplayCandidate{}
	for rows.Next() {
		var id, typ, uri, ws, lane, meta, corr, trace string
		var ts int64
		if err := rows.Scan(&id, &typ, &uri, &ws, &lane, &meta, &corr, &trace, &ts); err != nil {
			return nil, err
		}
		out = append(out, replayCandidate("artifact_ref", id, ws, lane, ts, typ+" "+uri, tagsFromText(typ+" "+meta), corr, trace))
	}
	return out, rows.Err()
}

func ScoreCandidates(candidates []ReplayCandidate, now int64) []SalienceScore {
	keys := map[string]int{}
	for _, c := range candidates {
		keys[semanticKey(c.ContentSummary)]++
	}
	out := make([]SalienceScore, 0, len(candidates))
	for _, c := range candidates {
		rep := clamp01(float64(keys[semanticKey(c.ContentSummary)]-1) / 3)
		score := SalienceScore{
			CandidateID:           c.CandidateID,
			NoveltyScore:          clamp01(1 - rep),
			RepetitionScore:       rep,
			GoalRelevanceScore:    boolScore(hasAnyTag(c, "open", "active", "blocker", "critical", "high")),
			CorrectionValueScore:  boolScore(hasAnyTag(c, "correction", "corrected", "preference", "user_correction")),
			OutcomeImpactScore:    boolScore(hasAnyTag(c, "failed", "failure", "blocked", "error", "timeout") || c.RawImportanceSignals["failed_restore"] > 0),
			ContradictionScore:    boolScore(c.SourceType == "contradiction" || hasAnyTag(c, "contradiction", "conflict", "unresolved")),
			RetrievalUtilityScore: boolScore(c.SourceType == "context_snapshot" || len(c.RelatedSnapshotIDs) > 0),
			RecencyScore:          recency(now, c.EndTimestamp),
		}
		score.TotalSalience = clamp01(
			0.10*score.NoveltyScore +
				0.10*score.RepetitionScore +
				0.15*score.GoalRelevanceScore +
				0.20*score.CorrectionValueScore +
				0.15*score.OutcomeImpactScore +
				0.15*score.ContradictionScore +
				0.05*score.RetrievalUtilityScore +
				0.10*score.RecencyScore,
		)
		score.Confidence = clamp01((score.TotalSalience + c.RawImportanceSignals["confidence"] + 0.5) / 2)
		score.Explain = salienceExplain(score)
		out = append(out, score)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].TotalSalience == out[j].TotalSalience {
			return out[i].CandidateID < out[j].CandidateID
		}
		return out[i].TotalSalience > out[j].TotalSalience
	})
	return out
}

func RouteCandidates(candidates []ReplayCandidate, scores []SalienceScore, req RunRequest, dryRun bool) []RoutingProposal {
	byID := map[string]ReplayCandidate{}
	for _, c := range candidates {
		byID[c.CandidateID] = c
	}
	out := make([]RoutingProposal, 0, len(scores))
	for _, score := range scores {
		decision := RetainShortTerm
		reason := "recent replay candidate retained in short-term tier"
		switch {
		case score.ContradictionScore > 0 && score.Confidence < 0.85:
			decision = NeedsReview
			reason = "unresolved contradiction requires operator review before promotion"
		case score.ContradictionScore > 0.75 && score.OutcomeImpactScore > 0:
			decision = RepairRequired
			reason = "contradictory failed outcome should be repaired through governed semantic action"
		case score.TotalSalience >= 0.82 && score.ContradictionScore == 0 && req.AllowLongTermPromotion && !req.RequireOperatorReviewForLongTerm:
			decision = PromoteLongTerm
			reason = "high confidence durable pattern without contradiction risk"
		case score.TotalSalience >= 0.72 && score.ContradictionScore == 0:
			if req.AllowLongTermPromotion && req.RequireOperatorReviewForLongTerm {
				decision = NeedsReview
				reason = "long-term promotion candidate requires operator review"
			} else {
				decision = PromoteMidTerm
				reason = "high salience candidate suitable for mid-term consolidation"
			}
		case score.TotalSalience >= 0.50:
			decision = PromoteMidTerm
			reason = "moderate salience candidate suitable for mid-term summary"
		case score.RepetitionScore > 0.65 && score.TotalSalience < 0.35:
			decision = Discard
			reason = "low-salience redundant derived candidate; raw journal truth remains intact"
		case score.TotalSalience < 0.25:
			decision = Noop
			reason = "insufficient salience for memory tier action"
		}
		sourceType := byID[score.CandidateID].SourceType
		if c := byID[score.CandidateID]; c.SourceType == "memory_note" && decision == RetainShortTerm && score.TotalSalience < 0.35 {
			decision = Demote
			reason = "low-salience derived memory note can be demoted without touching journal truth"
		}
		out = append(out, RoutingProposal{CandidateID: score.CandidateID, SourceType: sourceType, Decision: decision, Confidence: score.Confidence, Reason: reason, DryRun: dryRun})
	}
	return out
}

func replayCandidate(sourceType, id, ws, lane string, ts int64, summary string, tags []string, corr, trace string) ReplayCandidate {
	return ReplayCandidate{
		CandidateID:          stableID("dream", sourceType, id),
		SourceType:           sourceType,
		SourceIDs:            []string{id},
		WorkspaceID:          ws,
		LaneID:               lane,
		StartTimestamp:       ts,
		EndTimestamp:         ts,
		ContentSummary:       truncate(summary, 180),
		Tags:                 tags,
		RawImportanceSignals: map[string]float64{},
		Trace: map[string]string{
			"correlation_id": corr,
			"trace_id":       trace,
			"source_id":      id,
		},
	}
}

func stableID(parts ...string) string {
	joined := strings.Join(parts, ":")
	sum := uint64(1469598103934665603)
	for _, b := range []byte(joined) {
		sum ^= uint64(b)
		sum *= 1099511628211
	}
	return fmt.Sprintf("%s:%x", strings.TrimSpace(parts[0]), sum)
}

func tagsFromText(raw string) []string {
	norm := normalize(raw)
	seen := map[string]struct{}{}
	for _, token := range strings.Fields(norm) {
		switch token {
		case "correction", "corrected", "correct", "preference":
			seen["correction"] = struct{}{}
			if token == "preference" {
				seen["preference"] = struct{}{}
			}
		case "contradiction", "contradicts", "conflict", "unresolved":
			seen["contradiction"] = struct{}{}
			if token == "unresolved" {
				seen["unresolved"] = struct{}{}
			}
		case "blocked", "blocker", "open", "active", "failed", "failure", "error", "timeout", "critical", "high":
			seen[token] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for tag := range seen {
		out = append(out, tag)
	}
	sort.Strings(out)
	return out
}

func hasAnyTag(c ReplayCandidate, tags ...string) bool {
	set := map[string]struct{}{}
	for _, tag := range c.Tags {
		set[tag] = struct{}{}
	}
	for _, tag := range tags {
		if _, ok := set[tag]; ok {
			return true
		}
	}
	return false
}

func salienceExplain(score SalienceScore) []string {
	out := []string{}
	if score.CorrectionValueScore > 0 {
		out = append(out, "user correction or preference signal")
	}
	if score.ContradictionScore > 0 {
		out = append(out, "unresolved contradiction/conflict signal")
	}
	if score.GoalRelevanceScore > 0 {
		out = append(out, "active loop or blocker signal")
	}
	if score.OutcomeImpactScore > 0 {
		out = append(out, "failed or blocked outcome signal")
	}
	if score.RepetitionScore > 0 {
		out = append(out, "repeated similar replay signal")
	}
	if score.RetrievalUtilityScore > 0 {
		out = append(out, "restore/context retrieval utility signal")
	}
	if len(out) == 0 {
		out = append(out, "deterministic low-salience replay candidate")
	}
	return out
}

func summaryJSON(raw string) string {
	var decoded any
	if json.Unmarshal([]byte(raw), &decoded) == nil {
		encoded, _ := json.Marshal(decoded)
		return truncate(string(encoded), 120)
	}
	return truncate(raw, 120)
}

func normalize(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	raw = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsSpace(r) || r == '_' {
			return r
		}
		return ' '
	}, raw)
	return strings.Join(strings.Fields(raw), " ")
}

func semanticKey(raw string) string {
	parts := strings.Fields(normalize(raw))
	if len(parts) > 8 {
		parts = parts[:8]
	}
	return strings.Join(parts, " ")
}

func truncate(raw string, n int) string {
	raw = strings.Join(strings.Fields(raw), " ")
	if len(raw) <= n {
		return raw
	}
	return raw[:n]
}

func boolScore(ok bool) float64 {
	if ok {
		return 1
	}
	return 0
}

func recency(now, ts int64) float64 {
	if ts <= 0 || now <= ts {
		return 1
	}
	halfLife := float64(12 * 60 * 60 * 1000)
	return math.Round((1/(1+float64(now-ts)/halfLife))*1000) / 1000
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return math.Round(v*1000) / 1000
}
