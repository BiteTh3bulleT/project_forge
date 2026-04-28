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

	"forge/projectforge/services/core/internal/aios/rulecells"
)

type Mode string

const (
	ModeMicrodream Mode = "microdream"
	ModeNap        Mode = "nap"
	ModeDeepDream  Mode = "deep_dream"
)

type Service struct {
	db         *sql.DB
	clock      func() time.Time
	ruleEngine RuleEngine
}

type RuleEngine interface {
	Run(ctx context.Context, in rulecells.RunInput, opts rulecells.RunOptions) (rulecells.RunResult, error)
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db, clock: time.Now}
}

func (s *Service) SetClock(clock func() time.Time) {
	if clock != nil {
		s.clock = clock
	}
}

func (s *Service) SetRuleEngine(engine RuleEngine) {
	s.ruleEngine = engine
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
	CandidateID            string         `json:"candidate_id"`
	NoveltyScore           float64        `json:"novelty_score"`
	RepetitionScore        float64        `json:"repetition_score"`
	GoalRelevanceScore     float64        `json:"goal_relevance_score"`
	CorrectionValueScore   float64        `json:"correction_value_score"`
	OutcomeImpactScore     float64        `json:"outcome_impact_score"`
	ContradictionScore     float64        `json:"contradiction_score"`
	RetrievalUtilityScore  float64        `json:"retrieval_utility_score"`
	RecencyScore           float64        `json:"recency_score"`
	PreRuleTotalSalience   float64        `json:"pre_rule_total_salience,omitempty"`
	RuleSalienceAdjustment float64        `json:"rule_salience_adjustment,omitempty"`
	RuleTrace              map[string]any `json:"rule_trace,omitempty"`
	TotalSalience          float64        `json:"total_salience"`
	Confidence             float64        `json:"confidence"`
	Explain                []string       `json:"explain"`
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
	Run                               DreamRun          `json:"run"`
	Candidates                        []ReplayCandidate `json:"candidates"`
	SalienceScores                    []SalienceScore   `json:"salience_scores"`
	ProposedTierRouting               []RoutingProposal `json:"proposed_tier_routing"`
	ProposedMemoryActions             []RoutingProposal `json:"proposed_memory_actions"`
	ProposedSnapshotHygieneActions    []RoutingProposal `json:"proposed_snapshot_hygiene_actions"`
	ProposedRestoreScoreUpdates       []RoutingProposal `json:"proposed_restore_score_updates"`
	ProposedEmbeddingRefreshActions   []RoutingProposal `json:"proposed_embedding_refresh_actions"`
	ProposedRepairActions             []RoutingProposal `json:"proposed_repair_actions"`
	RestoreOutcomeCandidates          []ReplayCandidate `json:"restore_outcome_candidates,omitempty"`
	MemoryGapProposals                []RoutingProposal `json:"memory_gap_proposals,omitempty"`
	StaleEvidenceReviewProposals      []RoutingProposal `json:"stale_evidence_review_proposals,omitempty"`
	HelpfulEvidencePromotionProposals []RoutingProposal `json:"helpful_evidence_promotion_proposals,omitempty"`
	ItemsRequiringReview              []RoutingProposal `json:"items_requiring_review"`
	NoOpReasons                       []string          `json:"no_op_reasons"`
	Warnings                          []string          `json:"warnings"`
	Trace                             map[string]any    `json:"trace"`
}

type PersistReportRequest struct {
	Report      DreamReport    `json:"report"`
	SyscallID   string         `json:"syscallId,omitempty"`
	AuditID     string         `json:"auditId,omitempty"`
	ProposedBy  string         `json:"proposedBy,omitempty"`
	CommittedBy string         `json:"committedBy,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type ReportRecord struct {
	ID                       string            `json:"id"`
	CreatedAt                int64             `json:"createdAt"`
	CompletedAt              int64             `json:"completedAt"`
	WorkspaceID              string            `json:"workspaceId"`
	LaneID                   string            `json:"laneId,omitempty"`
	Mode                     Mode              `json:"mode"`
	DryRun                   bool              `json:"dryRun"`
	Status                   string            `json:"status"`
	TimeWindowStart          int64             `json:"timeWindowStart"`
	TimeWindowEnd            int64             `json:"timeWindowEnd"`
	CandidatesConsidered     int               `json:"candidatesConsidered"`
	ProposalsGenerated       int               `json:"proposalsGenerated"`
	Summary                  json.RawMessage   `json:"summary"`
	Candidates               []ReplayCandidate `json:"candidates"`
	SalienceScores           []SalienceScore   `json:"salienceScores"`
	MemoryTierProposals      []RoutingProposal `json:"memoryTierProposals"`
	RepairProposals          []RoutingProposal `json:"repairProposals"`
	SnapshotHygieneProposals []RoutingProposal `json:"snapshotHygieneProposals"`
	Warnings                 []string          `json:"warnings"`
	Trace                    json.RawMessage   `json:"trace"`
	CorrelationID            string            `json:"correlationId,omitempty"`
	TraceID                  string            `json:"traceId,omitempty"`
	SyscallID                string            `json:"syscallId,omitempty"`
	AuditID                  string            `json:"auditId,omitempty"`
	ProposedBy               string            `json:"proposedBy"`
	CommittedBy              string            `json:"committedBy,omitempty"`
	Metadata                 json.RawMessage   `json:"metadata"`
	EvidenceClass            string            `json:"evidenceClass"`
	NonCanonicalEvidence     bool              `json:"nonCanonicalEvidence"`
	CanonicalWriteCommitted  bool              `json:"canonicalWriteCommitted"`
}

func (r ReportRecord) ReviewItems() []RoutingProposal {
	out := []RoutingProposal{}
	for _, proposal := range r.MemoryTierProposals {
		if proposal.Decision == NeedsReview || proposal.Decision == RepairRequired {
			out = append(out, proposal)
		}
	}
	for _, proposal := range r.RepairProposals {
		if proposal.Decision == NeedsReview || proposal.Decision == RepairRequired {
			out = append(out, proposal)
		}
	}
	return out
}

type ListReportsRequest struct {
	WorkspaceID string
	LaneID      string
	Mode        Mode
	Limit       int
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
	ruleTrace, blockedLongTerm, ruleWarnings := s.applyRuleCells(ctx, candidates, scores, now)
	routing := RouteCandidates(candidates, scores, req, dryRun)
	if len(blockedLongTerm) > 0 {
		applyLongTermBlocks(routing, blockedLongTerm)
	}
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
	if len(ruleTrace) > 0 {
		report.Trace["rule_cells"] = ruleTrace
	}
	if len(ruleWarnings) > 0 {
		report.Warnings = append(report.Warnings, ruleWarnings...)
	}
	candidateByID := map[string]ReplayCandidate{}
	for _, candidate := range candidates {
		candidateByID[candidate.CandidateID] = candidate
		if candidate.SourceType == "restore_outcome" {
			report.RestoreOutcomeCandidates = append(report.RestoreOutcomeCandidates, candidate)
		}
	}
	for _, proposal := range routing {
		candidate := candidateByID[proposal.CandidateID]
		switch proposal.Decision {
		case NeedsReview:
			report.ItemsRequiringReview = append(report.ItemsRequiringReview, proposal)
		case RepairRequired:
			report.ProposedRepairActions = append(report.ProposedRepairActions, proposal)
		case Noop, Discard:
			report.NoOpReasons = append(report.NoOpReasons, proposal.CandidateID+": "+proposal.Reason)
		}
		if proposal.SourceType == "context_snapshot" || proposal.SourceType == "restore_outcome" {
			report.ProposedSnapshotHygieneActions = append(report.ProposedSnapshotHygieneActions, proposal)
			report.ProposedRestoreScoreUpdates = append(report.ProposedRestoreScoreUpdates, proposal)
		}
		if proposal.SourceType == "restore_outcome" {
			switch {
			case hasAnyTag(candidate, "fresh_compile_required", "no_candidate", "memory_gap"):
				report.MemoryGapProposals = append(report.MemoryGapProposals, proposal)
			case hasAnyTag(candidate, "harmful", "stale", "contradictory", "not_helpful", "operator_corrected"):
				report.StaleEvidenceReviewProposals = append(report.StaleEvidenceReviewProposals, proposal)
			case hasAnyTag(candidate, "helpful"):
				report.HelpfulEvidencePromotionProposals = append(report.HelpfulEvidencePromotionProposals, proposal)
			}
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

func (s *Service) PersistReport(ctx context.Context, req PersistReportRequest) (string, error) {
	if s == nil || s.db == nil {
		return "", fmt.Errorf("dream service requires sqlite store")
	}
	report := req.Report
	if strings.TrimSpace(report.Run.RunID) == "" {
		return "", fmt.Errorf("dream report id required")
	}
	if strings.TrimSpace(report.Run.WorkspaceID) == "" {
		return "", fmt.Errorf("dream report workspace_id required")
	}
	proposedBy := strings.TrimSpace(req.ProposedBy)
	if proposedBy == "" {
		proposedBy = "forge.dream"
	}
	summaryJSON := mustJSON(map[string]any{
		"summary":                     report.Run.Summary,
		"itemsRequiringReview":        report.ItemsRequiringReview,
		"proposedRestoreScoreUpdates": report.ProposedRestoreScoreUpdates,
		"proposedEmbeddingRefresh":    report.ProposedEmbeddingRefreshActions,
		"proposedMemoryActions":       report.ProposedMemoryActions,
		"noOpReasons":                 report.NoOpReasons,
		"nonCanonicalEvidence":        true,
		"canonicalWriteCommitted":     false,
	})
	_, err := s.db.ExecContext(ctx, `INSERT INTO dream_reports(
  id, created_at, completed_at, workspace_id, lane_id, mode, dry_run, status,
  time_window_start, time_window_end, candidates_considered, proposals_generated,
  summary_json, candidates_json, salience_scores_json, memory_tier_proposals_json,
  repair_proposals_json, snapshot_hygiene_proposals_json, warnings_json, trace_json,
  correlation_id, trace_id, syscall_id, audit_id, proposed_by, committed_by, metadata_json
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  completed_at=excluded.completed_at,
  status=excluded.status,
  candidates_considered=excluded.candidates_considered,
  proposals_generated=excluded.proposals_generated,
  summary_json=excluded.summary_json,
  candidates_json=excluded.candidates_json,
  salience_scores_json=excluded.salience_scores_json,
  memory_tier_proposals_json=excluded.memory_tier_proposals_json,
  repair_proposals_json=excluded.repair_proposals_json,
  snapshot_hygiene_proposals_json=excluded.snapshot_hygiene_proposals_json,
  warnings_json=excluded.warnings_json,
  trace_json=excluded.trace_json,
  correlation_id=excluded.correlation_id,
  trace_id=excluded.trace_id,
  syscall_id=excluded.syscall_id,
  audit_id=excluded.audit_id,
  proposed_by=excluded.proposed_by,
  committed_by=excluded.committed_by,
  metadata_json=excluded.metadata_json`,
		report.Run.RunID,
		report.Run.StartedAt,
		report.Run.CompletedAt,
		report.Run.WorkspaceID,
		report.Run.LaneID,
		string(report.Run.Mode),
		boolInt(report.Run.DryRun),
		report.Run.Status,
		report.Run.WindowStart,
		report.Run.WindowEnd,
		report.Run.CandidatesConsidered,
		report.Run.ProposalsGenerated,
		summaryJSON,
		mustJSON(report.Candidates),
		mustJSON(report.SalienceScores),
		mustJSON(report.ProposedTierRouting),
		mustJSON(report.ProposedRepairActions),
		mustJSON(report.ProposedSnapshotHygieneActions),
		mustJSON(report.Warnings),
		mustJSON(report.Trace),
		report.Run.CorrelationID,
		report.Run.TraceID,
		nullableString(req.SyscallID),
		nullableString(req.AuditID),
		proposedBy,
		strings.TrimSpace(req.CommittedBy),
		mustJSON(req.Metadata),
	)
	if err != nil {
		return "", err
	}
	return report.Run.RunID, nil
}

func (s *Service) GetReport(ctx context.Context, id, workspaceID, laneID string) (ReportRecord, error) {
	if strings.TrimSpace(workspaceID) == "" {
		return ReportRecord{}, fmt.Errorf("workspaceId required")
	}
	query := `SELECT id,created_at,completed_at,workspace_id,lane_id,mode,dry_run,status,
time_window_start,time_window_end,candidates_considered,proposals_generated,summary_json,
candidates_json,salience_scores_json,memory_tier_proposals_json,repair_proposals_json,
snapshot_hygiene_proposals_json,warnings_json,trace_json,correlation_id,trace_id,
COALESCE(syscall_id,''),COALESCE(audit_id,''),proposed_by,committed_by,metadata_json
FROM dream_reports WHERE id=? AND workspace_id=? AND (?='' OR lane_id=?)`
	row := s.db.QueryRowContext(ctx, query, strings.TrimSpace(id), strings.TrimSpace(workspaceID), strings.TrimSpace(laneID), strings.TrimSpace(laneID))
	return scanReport(row)
}

func (s *Service) ListReports(ctx context.Context, req ListReportsRequest) ([]ReportRecord, error) {
	if strings.TrimSpace(req.WorkspaceID) == "" {
		return nil, fmt.Errorf("workspaceId required")
	}
	limit := req.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id,created_at,completed_at,workspace_id,lane_id,mode,dry_run,status,
time_window_start,time_window_end,candidates_considered,proposals_generated,summary_json,
candidates_json,salience_scores_json,memory_tier_proposals_json,repair_proposals_json,
snapshot_hygiene_proposals_json,warnings_json,trace_json,correlation_id,trace_id,
COALESCE(syscall_id,''),COALESCE(audit_id,''),proposed_by,committed_by,metadata_json
FROM dream_reports
WHERE workspace_id=? AND (?='' OR lane_id=?) AND (?='' OR mode=?)
ORDER BY created_at DESC LIMIT ?`, strings.TrimSpace(req.WorkspaceID), strings.TrimSpace(req.LaneID), strings.TrimSpace(req.LaneID), strings.TrimSpace(string(req.Mode)), strings.TrimSpace(string(req.Mode)), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ReportRecord{}
	for rows.Next() {
		rec, err := scanReport(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
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
		s.loadRestoreOutcomeEvents,
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

func (s *Service) loadRestoreOutcomeEvents(ctx context.Context, req RunRequest, since, until int64) ([]ReplayCandidate, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,query,workspace_id,lane_id,context_packet_id,snapshot_id,restore_score,requires_fresh_compile,outcome,outcome_confidence,operator_feedback,correction_summary,correlation_id,trace_id,created_at FROM restore_outcome_events WHERE workspace_id=? AND (?='' OR lane_id='' OR lane_id=?) AND created_at BETWEEN ? AND ? ORDER BY created_at DESC LIMIT ?`, req.WorkspaceID, req.LaneID, req.LaneID, since, until, req.MaxCandidates)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ReplayCandidate{}
	for rows.Next() {
		var id, query, ws, lane, packetID, snapshotID, outcome, feedback, correction, corr, trace string
		var restoreScore, confidence float64
		var requiresFresh int
		var ts int64
		if err := rows.Scan(&id, &query, &ws, &lane, &packetID, &snapshotID, &restoreScore, &requiresFresh, &outcome, &confidence, &feedback, &correction, &corr, &trace, &ts); err != nil {
			return nil, err
		}
		tags := tagsFromText(strings.Join([]string{"restore_outcome", outcome, feedback, correction}, " "))
		tags = append(tags, "restore_outcome", outcome)
		if requiresFresh != 0 || outcome == "fresh_compile_required" || outcome == "no_candidate" {
			tags = append(tags, "memory_gap")
		}
		if outcome == "operator_corrected" {
			tags = append(tags, "correction", "corrected", "user_correction")
		}
		if outcome == "harmful" || outcome == "stale" || outcome == "contradictory" || outcome == "not_helpful" || outcome == "failed_execution" {
			tags = append(tags, "failed", "review")
		}
		c := replayCandidate("restore_outcome", id, ws, lane, ts, outcome+" restore outcome for "+query, normalizeTags(tags), corr, trace)
		c.SourceIDs = normalizeStrings([]string{id, packetID, snapshotID})
		if snapshotID != "" {
			c.RelatedSnapshotIDs = []string{snapshotID}
		}
		c.RawImportanceSignals["confidence"] = confidence
		c.RawImportanceSignals["restore_score"] = restoreScore
		c.RawImportanceSignals["restore_outcome"] = 1
		if outcome == "operator_corrected" {
			c.RawImportanceSignals["correction"] = 1
		}
		if hasAnyTag(c, "harmful", "stale", "contradictory", "not_helpful", "failed_execution") {
			c.RawImportanceSignals["outcome_impact"] = 1
		}
		if hasAnyTag(c, "memory_gap") {
			c.RawImportanceSignals["memory_gap"] = 1
		}
		if outcome == "helpful" {
			c.RawImportanceSignals["helpful"] = 1
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
			CorrectionValueScore:  boolScore(hasAnyTag(c, "correction", "corrected", "preference", "user_correction", "operator_corrected") || c.RawImportanceSignals["correction"] > 0),
			OutcomeImpactScore:    boolScore(hasAnyTag(c, "failed", "failure", "blocked", "error", "timeout", "harmful", "stale", "not_helpful", "failed_execution") || c.RawImportanceSignals["failed_restore"] > 0 || c.RawImportanceSignals["outcome_impact"] > 0),
			ContradictionScore:    boolScore(c.SourceType == "contradiction" || hasAnyTag(c, "contradiction", "conflict", "unresolved")),
			RetrievalUtilityScore: boolScore(c.SourceType == "context_snapshot" || c.SourceType == "restore_outcome" || len(c.RelatedSnapshotIDs) > 0),
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
		switch {
		case hasAnyTag(c, "operator_corrected"):
			score.TotalSalience = math.Max(score.TotalSalience, 0.88)
		case hasAnyTag(c, "harmful", "stale", "contradictory"):
			score.TotalSalience = math.Max(score.TotalSalience, 0.78)
		case hasAnyTag(c, "fresh_compile_required", "no_candidate", "memory_gap"):
			score.TotalSalience = math.Max(score.TotalSalience, 0.70)
		case hasAnyTag(c, "helpful") && score.RepetitionScore > 0:
			score.TotalSalience = math.Max(score.TotalSalience, 0.72)
		case hasAnyTag(c, "helpful"):
			score.TotalSalience = math.Max(score.TotalSalience, 0.55)
		}
		score.TotalSalience = clamp01(score.TotalSalience)
		score.PreRuleTotalSalience = score.TotalSalience
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
		candidate := byID[score.CandidateID]
		switch {
		case candidate.SourceType == "restore_outcome" && hasAnyTag(candidate, "operator_corrected", "contradictory"):
			decision = NeedsReview
			reason = "restore outcome feedback requires governed review before memory promotion"
		case candidate.SourceType == "restore_outcome" && hasAnyTag(candidate, "harmful", "stale", "not_helpful", "failed_execution"):
			decision = NeedsReview
			reason = "restore outcome marks selected evidence as stale or harmful"
		case candidate.SourceType == "restore_outcome" && hasAnyTag(candidate, "fresh_compile_required", "no_candidate", "memory_gap"):
			decision = NeedsReview
			reason = "repeated restore miss is a memory gap proposal"
		case candidate.SourceType == "restore_outcome" && hasAnyTag(candidate, "helpful") && score.TotalSalience >= 0.55:
			decision = PromoteMidTerm
			reason = "helpful restore evidence is a promotion candidate"
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
		sourceType := candidate.SourceType
		if c := candidate; c.SourceType == "memory_note" && decision == RetainShortTerm && score.TotalSalience < 0.35 {
			decision = Demote
			reason = "low-salience derived memory note can be demoted without touching journal truth"
		}
		out = append(out, RoutingProposal{CandidateID: score.CandidateID, SourceType: sourceType, Decision: decision, Confidence: score.Confidence, Reason: reason, DryRun: dryRun})
	}
	return out
}

const (
	maxDreamRuleSalienceAdjustment       = 0.15
	maxDreamIndividualSalienceAdjustment = 0.08
)

func (s *Service) applyRuleCells(ctx context.Context, candidates []ReplayCandidate, scores []SalienceScore, now int64) ([]map[string]any, map[string]bool, []string) {
	if s == nil || s.ruleEngine == nil {
		return nil, nil, nil
	}
	byID := map[string]ReplayCandidate{}
	for _, candidate := range candidates {
		byID[candidate.CandidateID] = candidate
	}
	traces := []map[string]any{}
	warnings := []string{}
	blockedLongTerm := map[string]bool{}
	for idx := range scores {
		candidate := byID[scores[idx].CandidateID]
		failureCount := 0
		if hasAnyTag(candidate, "failed", "failure", "error", "timeout", "blocked") || candidate.RawImportanceSignals["failed_restore"] > 0 {
			failureCount = 1
		}
		if scores[idx].RepetitionScore > 0 {
			failureCount++
		}
		facts := map[string]any{
			"tags":                append([]string(nil), candidate.Tags...),
			"candidate_id":        candidate.CandidateID,
			"source_type":         candidate.SourceType,
			"contradiction_score": scores[idx].ContradictionScore,
			"failure_count":       failureCount,
			"total_salience":      scores[idx].TotalSalience,
			"created_at":          candidate.EndTimestamp,
			"age_ms":              now - candidate.EndTimestamp,
		}
		result, err := s.ruleEngine.Run(ctx, rulecells.RunInput{
			Lane:      rulecells.LaneLymphatic,
			Phase:     rulecells.PhaseSalienceScoring,
			InputID:   candidate.CandidateID,
			InputType: "dream_candidate",
			Facts:     facts,
		}, rulecells.RunOptions{DryRun: true, MaxLatencyMs: 5})
		if err != nil {
			warnings = append(warnings, "dream rule engine failed during salience scoring: "+err.Error())
		} else {
			adjustment := boundedDreamAdjustment(result.Outputs)
			scores[idx].RuleSalienceAdjustment = adjustment
			scores[idx].TotalSalience = clamp01(scores[idx].TotalSalience + adjustment)
			scores[idx].Confidence = clamp01((scores[idx].TotalSalience + candidate.RawImportanceSignals["confidence"] + 0.5) / 2)
			if len(result.Trace.MatchedRules) > 0 || len(result.Trace.Warnings) > 0 {
				scores[idx].RuleTrace = dreamRuleTraceMap(result.Trace)
			}
			if adjustment != 0 {
				scores[idx].Explain = append(scores[idx].Explain, fmt.Sprintf("rule cell salience adjustment %.3f", adjustment))
			}
			warnings = append(warnings, result.Warnings...)
			if len(result.Trace.MatchedRules) > 0 || len(result.Trace.Warnings) > 0 {
				traces = append(traces, dreamRuleTraceMap(result.Trace))
			}
		}

		tierResult, err := s.ruleEngine.Run(ctx, rulecells.RunInput{
			Lane:      rulecells.LaneLymphatic,
			Phase:     rulecells.PhaseMemoryTierRouting,
			InputID:   candidate.CandidateID,
			InputType: "dream_candidate",
			Facts:     facts,
		}, rulecells.RunOptions{DryRun: true, MaxLatencyMs: 5})
		if err != nil {
			warnings = append(warnings, "dream rule engine failed during tier routing: "+err.Error())
			continue
		}
		for _, output := range tierResult.Outputs {
			if output.Type == rulecells.OutputPolicyDecision && strings.EqualFold(output.Decision, "block_long_term_promotion") {
				blockedLongTerm[candidate.CandidateID] = true
			}
		}
		warnings = append(warnings, tierResult.Warnings...)
		if len(tierResult.Trace.MatchedRules) > 0 {
			traces = append(traces, dreamRuleTraceMap(tierResult.Trace))
		}
	}
	sort.Slice(scores, func(i, j int) bool {
		if scores[i].TotalSalience == scores[j].TotalSalience {
			return scores[i].CandidateID < scores[j].CandidateID
		}
		return scores[i].TotalSalience > scores[j].TotalSalience
	})
	return traces, blockedLongTerm, warnings
}

func boundedDreamAdjustment(outputs []rulecells.RuleOutput) float64 {
	total := 0.0
	for _, output := range outputs {
		if output.Type != rulecells.OutputScoreAdjustment {
			continue
		}
		delta := output.ScoreDelta
		if delta > maxDreamIndividualSalienceAdjustment {
			delta = maxDreamIndividualSalienceAdjustment
		}
		if delta < -maxDreamIndividualSalienceAdjustment {
			delta = -maxDreamIndividualSalienceAdjustment
		}
		total += delta
	}
	if total > maxDreamRuleSalienceAdjustment {
		return maxDreamRuleSalienceAdjustment
	}
	if total < -maxDreamRuleSalienceAdjustment {
		return -maxDreamRuleSalienceAdjustment
	}
	return math.Round(total*1000) / 1000
}

func applyLongTermBlocks(routes []RoutingProposal, blocked map[string]bool) {
	for idx := range routes {
		if !blocked[routes[idx].CandidateID] || routes[idx].Decision != PromoteLongTerm {
			continue
		}
		routes[idx].Decision = NeedsReview
		routes[idx].Reason = "rule cell blocked long-term promotion; operator review required"
	}
}

func dreamRuleTraceMap(trace rulecells.RuleTrace) map[string]any {
	packs := make([]map[string]any, 0, len(trace.RulePacks))
	for _, pack := range trace.RulePacks {
		packs = append(packs, map[string]any{"pack_id": pack.ID, "version": pack.Version})
	}
	matched := make([]map[string]any, 0, len(trace.MatchedRules))
	for _, rule := range trace.MatchedRules {
		matched = append(matched, map[string]any{
			"rule_id":      rule.RuleID,
			"rule_version": rule.RuleVersion,
			"pack_id":      rule.PackID,
			"pack_version": rule.PackVersion,
			"output_types": rule.OutputTypes,
			"explain":      rule.Explain,
		})
	}
	return map[string]any{
		"trace_id":        trace.TraceID,
		"lane":            trace.Lane,
		"phase":           trace.Phase,
		"input_id":        trace.InputID,
		"latency_ms":      trace.LatencyMs,
		"rules_evaluated": trace.RulesEvaluated,
		"rule_packs":      packs,
		"matched_rules":   matched,
		"warnings":        trace.Warnings,
	}
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

func normalizeTags(tags []string) []string {
	seen := map[string]struct{}{}
	for _, tag := range tags {
		trimmed := strings.TrimSpace(strings.ToLower(tag))
		if trimmed == "" {
			continue
		}
		seen[trimmed] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for tag := range seen {
		out = append(out, tag)
	}
	sort.Strings(out)
	return out
}

func normalizeStrings(values []string) []string {
	seen := map[string]struct{}{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		seen[trimmed] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for value := range seen {
		out = append(out, value)
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

type reportScanner interface {
	Scan(dest ...any) error
}

func scanReport(row reportScanner) (ReportRecord, error) {
	var rec ReportRecord
	var mode string
	var dryRun int
	var summaryJSON, candidatesJSON, scoresJSON, tierJSON, repairJSON, snapshotJSON, warningsJSON, traceJSON, metadataJSON string
	if err := row.Scan(
		&rec.ID, &rec.CreatedAt, &rec.CompletedAt, &rec.WorkspaceID, &rec.LaneID, &mode, &dryRun, &rec.Status,
		&rec.TimeWindowStart, &rec.TimeWindowEnd, &rec.CandidatesConsidered, &rec.ProposalsGenerated,
		&summaryJSON, &candidatesJSON, &scoresJSON, &tierJSON, &repairJSON, &snapshotJSON, &warningsJSON, &traceJSON,
		&rec.CorrelationID, &rec.TraceID, &rec.SyscallID, &rec.AuditID, &rec.ProposedBy, &rec.CommittedBy, &metadataJSON,
	); err != nil {
		return ReportRecord{}, err
	}
	rec.Mode = Mode(mode)
	rec.DryRun = dryRun != 0
	rec.Summary = json.RawMessage(nonEmptyJSON(summaryJSON, "{}"))
	rec.Trace = json.RawMessage(nonEmptyJSON(traceJSON, "{}"))
	rec.Metadata = json.RawMessage(nonEmptyJSON(metadataJSON, "{}"))
	_ = json.Unmarshal([]byte(nonEmptyJSON(candidatesJSON, "[]")), &rec.Candidates)
	_ = json.Unmarshal([]byte(nonEmptyJSON(scoresJSON, "[]")), &rec.SalienceScores)
	_ = json.Unmarshal([]byte(nonEmptyJSON(tierJSON, "[]")), &rec.MemoryTierProposals)
	_ = json.Unmarshal([]byte(nonEmptyJSON(repairJSON, "[]")), &rec.RepairProposals)
	_ = json.Unmarshal([]byte(nonEmptyJSON(snapshotJSON, "[]")), &rec.SnapshotHygieneProposals)
	_ = json.Unmarshal([]byte(nonEmptyJSON(warningsJSON, "[]")), &rec.Warnings)
	rec.EvidenceClass = "non_canonical_evidence"
	rec.NonCanonicalEvidence = true
	rec.CanonicalWriteCommitted = false
	return rec, nil
}

func mustJSON(v any) string {
	if v == nil {
		return "{}"
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(raw)
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func nullableString(v string) any {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return v
}

func nonEmptyJSON(raw, fallback string) string {
	if strings.TrimSpace(raw) == "" {
		return fallback
	}
	return raw
}
