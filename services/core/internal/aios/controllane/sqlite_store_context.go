package controllane

import (
	"context"
	"fmt"
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
)

// ContextPacketRepository
func (s *SQLiteSemanticStore) CreateSnapshot(ctx context.Context, pkt domain.ContextPacket, syscallID, correlationID, traceID string, metadata map[string]any) error {
	metadata = contextSnapshotMetadata(pkt, metadata)
	cols := buildContextSnapshotColumns(pkt, metadata)
	_, err := s.exec.ExecContext(ctx, `
INSERT INTO context_packet_snapshots(
  id, query, workspace_id, lane_id, snapshot_kind, snapshot_fingerprint, parent_snapshot_id, selected_paths_json,
  included_state_json, included_open_loops_json, included_notes_json, included_links_json, included_models_json,
  included_artifacts_json, included_events_json, header_json, graph_json, delta_json, restore_scores_json, render_artifact_ref_id, resume_hints_json, budget_json, inclusion_reasons_json,
  created_at, correlation_id, trace_id, syscall_id, metadata_json, proposed_by, committed_by, audit_id
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		pkt.ID, pkt.Query, pkt.Scope.WorkspaceID, pkt.Scope.LaneID, cols.SnapshotKind, cols.SnapshotFingerprint, cols.ParentSnapshotID, encodeStringSlice(pkt.Scope.SelectedPaths),
		encodeIDsState(pkt.ActiveState), encodeIDsLoops(pkt.OpenLoops), encodeIDsNotes(pkt.Notes), encodeIDsLinks(pkt.LinkedNotes), encodeIDsModels(pkt.Models),
		encodeIDsArtifacts(pkt.Artifacts), encodeIDsEvents(pkt.RawEvents), cols.HeaderJSON, cols.GraphJSON, cols.DeltaJSON, cols.RestoreScoresJSON, cols.RenderArtifactRefID, cols.ResumeHintsJSON, encodeJSON(pkt.Budget), encodeJSON(pkt.InclusionReasons),
		pkt.CreatedAt, correlationID, traceID, syscallID, encodeJSON(nonNilMap(metadata)), string(s.meta.Source), nonEmpty(s.meta.CommittedBy, "forge_kernel"), "",
	)
	return err
}

func (s *SQLiteSemanticStore) CreateContextSnapshot(pkt domain.ContextPacket) error {
	return s.CreateSnapshot(s.background, pkt, s.meta.SyscallID, s.meta.CorrelationID, s.meta.TraceID, nil)
}

func (s *SQLiteSemanticStore) GetSnapshotByID(ctx context.Context, id string) (domain.ContextPacket, bool, error) {
	rows, err := s.exec.QueryContext(ctx, `
SELECT id, query, workspace_id, lane_id, selected_paths_json, budget_json, inclusion_reasons_json, created_at,
       metadata_json, snapshot_kind, snapshot_fingerprint, parent_snapshot_id, header_json, graph_json, delta_json,
       restore_scores_json, render_artifact_ref_id, resume_hints_json
FROM context_packet_snapshots WHERE id = ?`, id)
	if err != nil {
		return domain.ContextPacket{}, false, err
	}
	defer rows.Close()
	out, err := scanContextPacketRows(rows)
	if err != nil {
		return domain.ContextPacket{}, false, err
	}
	if len(out) == 0 {
		return domain.ContextPacket{}, false, nil
	}
	return out[0], true, nil
}

func (s *SQLiteSemanticStore) ListSnapshotsByScope(ctx context.Context, scope ScopeFilter, limit int) ([]domain.ContextPacket, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.exec.QueryContext(ctx, `
SELECT id, query, workspace_id, lane_id, selected_paths_json, budget_json, inclusion_reasons_json, created_at,
       metadata_json, snapshot_kind, snapshot_fingerprint, parent_snapshot_id, header_json, graph_json, delta_json,
       restore_scores_json, render_artifact_ref_id, resume_hints_json
FROM context_packet_snapshots
WHERE workspace_id = ? AND (? = '' OR lane_id = ?)
ORDER BY created_at DESC
LIMIT ?`, scope.WorkspaceID, scope.LaneID, scope.LaneID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanContextPacketRows(rows)
}

func (s *SQLiteSemanticStore) ListSnapshotsByCorrelation(ctx context.Context, correlationID string, limit int) ([]domain.ContextPacket, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.exec.QueryContext(ctx, `
SELECT id, query, workspace_id, lane_id, selected_paths_json, budget_json, inclusion_reasons_json, created_at,
       metadata_json, snapshot_kind, snapshot_fingerprint, parent_snapshot_id, header_json, graph_json, delta_json,
       restore_scores_json, render_artifact_ref_id, resume_hints_json
FROM context_packet_snapshots
WHERE correlation_id = ?
ORDER BY created_at DESC
LIMIT ?`, correlationID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanContextPacketRows(rows)
}

func (s *SQLiteSemanticStore) FindLatestSnapshotByQueryAndKind(ctx context.Context, scope ScopeFilter, query, snapshotKind string) (domain.ContextPacket, bool, error) {
	rows, err := s.exec.QueryContext(ctx, `
SELECT id, query, workspace_id, lane_id, selected_paths_json, budget_json, inclusion_reasons_json, created_at,
       metadata_json, snapshot_kind, snapshot_fingerprint, parent_snapshot_id, header_json, graph_json, delta_json,
       restore_scores_json, render_artifact_ref_id, resume_hints_json
FROM context_packet_snapshots
WHERE workspace_id = ? AND (? = '' OR lane_id = ?) AND query = ?
  AND (? = '' OR snapshot_kind = ? OR (snapshot_kind = '' AND json_extract(metadata_json, '$.snapshot_kind') = ?))
ORDER BY created_at DESC
LIMIT 1`, scope.WorkspaceID, scope.LaneID, scope.LaneID, query, snapshotKind, snapshotKind, snapshotKind)
	if err != nil {
		return domain.ContextPacket{}, false, err
	}
	defer rows.Close()
	out, err := scanContextPacketRows(rows)
	if err != nil {
		return domain.ContextPacket{}, false, err
	}
	if len(out) == 0 {
		return domain.ContextPacket{}, false, nil
	}
	return out[0], true, nil
}

func (s *SQLiteSemanticStore) CreateRestoreOutcome(ctx context.Context, event RestoreOutcomeEvent) error {
	event = normalizeRestoreOutcomeEvent(event)
	if event.ID == "" {
		return fmt.Errorf("restore outcome id required")
	}
	_, err := s.exec.ExecContext(ctx, `
INSERT INTO restore_outcome_events(
  id, created_at, updated_at, workspace_id, lane_id, query, context_packet_id, snapshot_id, snapshot_kind,
  restore_score, requires_fresh_compile, selected_evidence_json, selected_state_keys_json, selected_loop_ids_json,
  selected_artifact_ids_json, outcome, outcome_confidence, operator_feedback, failure_reason, correction_summary,
  downstream_action_type, downstream_object_id, correlation_id, trace_id, syscall_id, audit_id, proposed_by,
  committed_by, metadata_json
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		event.ID, event.CreatedAt, event.UpdatedAt, event.WorkspaceID, event.LaneID, event.Query, event.ContextPacketID, event.SnapshotID, event.SnapshotKind,
		event.RestoreScore, boolToInt(event.RequiresFreshCompile), encodeStringSlice(event.SelectedEvidence), encodeStringSlice(event.SelectedStateKeys), encodeStringSlice(event.SelectedLoopIDs),
		encodeStringSlice(event.SelectedArtifactIDs), string(event.Outcome), event.OutcomeConfidence, event.OperatorFeedback, event.FailureReason, event.CorrectionSummary,
		event.DownstreamActionType, event.DownstreamObjectID, event.CorrelationID, event.TraceID, event.SyscallID, event.AuditID, event.ProposedBy,
		event.CommittedBy, encodeJSON(nonNilMap(event.Metadata)),
	)
	return err
}

func (s *SQLiteSemanticStore) GetRestoreOutcome(ctx context.Context, id string) (RestoreOutcomeEvent, bool, error) {
	rows, err := s.exec.QueryContext(ctx, restoreOutcomeSelectSQL(`id = ?`), strings.TrimSpace(id))
	if err != nil {
		return RestoreOutcomeEvent{}, false, err
	}
	defer rows.Close()
	out, err := scanRestoreOutcomeRows(rows)
	if err != nil {
		return RestoreOutcomeEvent{}, false, err
	}
	if len(out) == 0 {
		return RestoreOutcomeEvent{}, false, nil
	}
	return out[0], true, nil
}

func (s *SQLiteSemanticStore) ListRestoreOutcomes(ctx context.Context, filter RestoreOutcomeFilter) ([]RestoreOutcomeEvent, error) {
	filter = normalizeRestoreOutcomeFilter(filter)
	where := []string{"1=1"}
	args := []any{}
	if filter.WorkspaceID != "" {
		where = append(where, "workspace_id = ?")
		args = append(args, filter.WorkspaceID)
	}
	if filter.LaneID != "" {
		where = append(where, "(lane_id = '' OR lane_id = ?)")
		args = append(args, filter.LaneID)
	}
	if filter.Query != "" {
		where = append(where, "query = ?")
		args = append(args, filter.Query)
	}
	if filter.SnapshotID != "" {
		where = append(where, "snapshot_id = ?")
		args = append(args, filter.SnapshotID)
	}
	if filter.Outcome != "" {
		where = append(where, "outcome = ?")
		args = append(args, string(filter.Outcome))
	}
	if filter.Since > 0 {
		where = append(where, "created_at >= ?")
		args = append(args, filter.Since)
	}
	query := restoreOutcomeSelectSQL(strings.Join(where, " AND ")) + ` ORDER BY created_at DESC, id DESC LIMIT ?`
	args = append(args, filter.Limit)
	rows, err := s.exec.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRestoreOutcomeRows(rows)
}

// Extra read helpers used by tests and context compilation.
func (s *SQLiteSemanticStore) ListLinksByScope(ctx context.Context, scope ScopeFilter, limit int) ([]domain.SemanticLink, error) {
	return s.listLinks(ctx, "1=1", nil, scope, limit)
}

func (s *SQLiteSemanticStore) ListModelsByScope(ctx context.Context, scope ScopeFilter, limit int) ([]domain.AdaptivePolicyModel, error) {
	return s.listModels(ctx, "1=1", nil, scope, limit)
}
