package controllane

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
)

func (s *SQLiteSemanticStore) FindRetrievalUsefulnessTarget(ctx context.Context, resultID int64) (RetrievalUsefulnessTarget, bool, error) {
	var target RetrievalUsefulnessTarget
	var selectedPathsJSON, sourceProvenanceJSON string
	var jobID sql.NullString
	var packetID sql.NullInt64
	err := s.exec.QueryRowContext(ctx, `
SELECT rr.id, rr.retrieval_run_id, rr.evidence_id,
       r.workspace_id, r.lane_id, r.selected_paths_json,
	       COALESCE(r.original_job_id,r.job_id), COALESCE(r.original_packet_id,r.packet_id),
	       r.syscall_id, r.provenance_id, r.provenance_json,
	       COALESCE(rup.label, 'unknown'), COALESCE(rup.note, '')
FROM retrieval_results rr
JOIN retrieval_runs r ON r.id = rr.retrieval_run_id
LEFT JOIN retrieval_usefulness_projection rup ON rup.retrieval_result_id = rr.id
WHERE rr.id = ?`, resultID).Scan(
		&target.ResultID, &target.RunID, &target.EvidenceID,
		&target.Scope.WorkspaceID, &target.Scope.LaneID, &selectedPathsJSON,
		&jobID, &packetID, &target.SourceSyscall, &target.SourceProvID, &sourceProvenanceJSON,
		&target.OriginalLabel, &target.OriginalNote,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return RetrievalUsefulnessTarget{}, false, nil
		}
		return RetrievalUsefulnessTarget{}, false, err
	}
	target.Scope.SelectedPaths = decodeStringSlice(selectedPathsJSON)
	target.SourceProvJSON = strings.TrimSpace(sourceProvenanceJSON)
	if jobID.Valid {
		value := jobID.String
		target.JobID = &value
	}
	if packetID.Valid {
		value := packetID.Int64
		target.PacketID = &value
	}
	return target, true, nil
}

func (s *SQLiteSemanticStore) CreateRetrievalUsefulnessEvent(event RetrievalUsefulnessEvent) error {
	ctx := s.background
	target, ok, err := s.FindRetrievalUsefulnessTarget(ctx, event.ResultID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("retrieval result %d not found", event.ResultID)
	}
	if err := validateRetrievalUsefulnessEvent(event, target); err != nil {
		return err
	}
	event.Label = NormalizeRetrievalUsefulnessLabel(event.Label)
	event.PriorProjection = map[string]any{"label": target.OriginalLabel, "note": target.OriginalNote, "nonCanonical": true}
	var sourceProvenance map[string]any
	if err := json.Unmarshal([]byte(target.SourceProvJSON), &sourceProvenance); err != nil || len(sourceProvenance) == 0 {
		return fmt.Errorf("retrieval usefulness target provenance invalid")
	}
	event.SourceProvenance = sourceProvenance
	provenanceID, err := s.ensureProvenance(ctx, event.Scope, event.Provenance, map[string]any{
		"utilityEvidenceType": "retrieval_usefulness", "targetEvidenceId": event.TargetEvidenceID,
		"retrievalResultId": event.ResultID, "retrievalRunId": event.RunID,
	}, event.CreatedAt)
	if err != nil {
		return err
	}
	event.ProvenanceID = provenanceID
	if event.JobID == nil {
		event.JobID = target.JobID
	}
	if event.PacketID == nil {
		event.PacketID = target.PacketID
	}
	if !optionalStringEqual(event.JobID, target.JobID) || !optionalInt64Equal(event.PacketID, target.PacketID) {
		return fmt.Errorf("retrieval usefulness related-object binding mismatch")
	}
	_, err = s.exec.ExecContext(ctx, `
INSERT INTO forge_k_retrieval_usefulness_events(
  id, created_at, retrieval_result_id, retrieval_run_id, target_evidence_id,
  workspace_id, lane_id, selected_paths_json, label, note, job_id, packet_id,
  prior_projection_json, source_provenance_json, metadata_json,
  correlation_id, trace_id, syscall_id, provenance_id, provenance_json, proposed_by, committed_by
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		event.ID, event.CreatedAt, event.ResultID, event.RunID, event.TargetEvidenceID,
		event.Scope.WorkspaceID, event.Scope.LaneID, encodeStringSlice(event.Scope.SelectedPaths), event.Label, strings.TrimSpace(event.Note),
		nullableStringPointer(event.JobID), nullableInt64Pointer(event.PacketID),
		encodeJSON(event.PriorProjection), encodeJSON(event.SourceProvenance), encodeJSON(nonNilMap(event.Metadata)),
		event.CorrelationID, event.TraceID, event.SyscallID, event.ProvenanceID, encodeJSON(event.Provenance), event.ProposedBy, event.CommittedBy,
	)
	if err != nil {
		return err
	}
	// Projection is deliberately separate from immutable retrieval evidence.
	// It can be dropped and rebuilt in event order without rewriting the source.
	_, err = s.exec.ExecContext(ctx, `
INSERT INTO retrieval_usefulness_projection(
  retrieval_result_id, latest_event_id, label, note, updated_at, noncanonical
) VALUES(?,?,?,?,?,1)
ON CONFLICT(retrieval_result_id) DO UPDATE SET
  latest_event_id=excluded.latest_event_id,
  label=excluded.label,
  note=excluded.note,
  updated_at=excluded.updated_at,
  noncanonical=1`, event.ResultID, event.ID, event.Label, strings.TrimSpace(event.Note), event.CreatedAt)
	return err
}

func (s *SQLiteSemanticStore) FindRestoreOutcomeFeedbackTarget(ctx context.Context, id string) (RestoreOutcomeFeedbackTarget, bool, error) {
	var target RestoreOutcomeFeedbackTarget
	var selectedPathsJSON string
	err := s.exec.QueryRowContext(ctx, `
SELECT roe.id, roe.workspace_id, roe.lane_id, cps.selected_paths_json,
       roe.outcome, roe.syscall_id, roe.committed_by
FROM restore_outcome_events roe
JOIN context_packet_snapshots cps
  ON cps.id = roe.context_packet_id
 AND cps.workspace_id = roe.workspace_id
 AND cps.lane_id = roe.lane_id
WHERE roe.id = ?`, strings.TrimSpace(id)).Scan(
		&target.RestoreOutcomeID, &target.Scope.WorkspaceID, &target.Scope.LaneID,
		&selectedPathsJSON,
		&target.OriginalOutcome, &target.SourceSyscall, &target.CommittedBy,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return RestoreOutcomeFeedbackTarget{}, false, nil
		}
		return RestoreOutcomeFeedbackTarget{}, false, err
	}
	target.Scope.SelectedPaths = decodeStringSlice(selectedPathsJSON)
	return target, true, nil
}

func (s *SQLiteSemanticStore) GetRestoreOutcomeFeedbackProjection(ctx context.Context, id string, scope domain.ForgeScope) (RestoreOutcomeFeedbackProjection, bool, error) {
	var projection RestoreOutcomeFeedbackProjection
	var metadataJSON, selectedPathsJSON string
	var nonCanonical int
	err := s.exec.QueryRowContext(ctx, `
SELECT restore_outcome_id, latest_event_id, workspace_id, lane_id, selected_paths_json,
       outcome, outcome_confidence, operator_feedback, correction_summary,
       updated_by, updated_at, metadata_json, noncanonical
FROM restore_outcome_feedback_projection
WHERE restore_outcome_id = ? AND workspace_id = ? AND lane_id = ?`,
		strings.TrimSpace(id), strings.TrimSpace(scope.WorkspaceID), strings.TrimSpace(scope.LaneID),
	).Scan(
		&projection.RestoreOutcomeID, &projection.LatestEventID,
		&projection.Scope.WorkspaceID, &projection.Scope.LaneID,
		&selectedPathsJSON,
		&projection.Outcome, &projection.OutcomeConfidence, &projection.OperatorFeedback,
		&projection.CorrectionSummary, &projection.UpdatedBy, &projection.UpdatedAt,
		&metadataJSON, &nonCanonical,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return RestoreOutcomeFeedbackProjection{}, false, nil
		}
		return RestoreOutcomeFeedbackProjection{}, false, err
	}
	projection.Metadata = map[string]any{}
	_ = json.Unmarshal([]byte(metadataJSON), &projection.Metadata)
	projection.Scope.SelectedPaths = decodeStringSlice(selectedPathsJSON)
	projection.NonCanonical = nonCanonical == 1
	return projection, true, nil
}

func (s *SQLiteSemanticStore) CreateRestoreOutcomeFeedbackEvent(event RestoreOutcomeFeedbackEvent) error {
	ctx := s.background
	target, ok, err := s.FindRestoreOutcomeFeedbackTarget(ctx, event.RestoreOutcomeID)
	if err != nil {
		return err
	}
	if !ok {
		return restoreOutcomeNotFound(event.RestoreOutcomeID)
	}
	if err := validateRestoreOutcomeFeedbackEvent(event, target); err != nil {
		return err
	}
	prior, priorOK, err := s.GetRestoreOutcomeFeedbackProjection(ctx, event.RestoreOutcomeID, event.Scope)
	if err != nil {
		return err
	}
	event.PriorProjection = map[string]any{"present": priorOK, "nonCanonical": true}
	if priorOK {
		event.PriorProjection["latestEventId"] = prior.LatestEventID
		event.PriorProjection["outcome"] = prior.Outcome
		event.PriorProjection["outcomeConfidence"] = prior.OutcomeConfidence
		event.PriorProjection["operatorFeedback"] = prior.OperatorFeedback
		event.PriorProjection["correctionSummary"] = prior.CorrectionSummary
		event.PriorProjection["updatedAt"] = prior.UpdatedAt
	}
	provenanceID, err := s.ensureProvenance(ctx, event.Scope, event.Provenance, map[string]any{
		"utilityEvidenceType": "restore_outcome_feedback", "restoreOutcomeId": event.RestoreOutcomeID,
		"sourceSyscallId": target.SourceSyscall,
	}, event.CreatedAt)
	if err != nil {
		return err
	}
	event.ProvenanceID = provenanceID
	feedback := normalizeRestoreOutcomeFeedback(RestoreOutcomeFeedback{
		Outcome: event.Outcome, OutcomeConfidence: event.OutcomeConfidence,
		OperatorFeedback: event.OperatorFeedback, CorrectionSummary: event.CorrectionSummary,
		Metadata: event.Metadata, CorrelationID: event.CorrelationID, TraceID: event.TraceID,
		UpdatedBy: event.Provenance.Actor, UpdatedAt: event.CreatedAt,
	})
	event.ProjectionSnapshot = feedback
	_, err = s.exec.ExecContext(ctx, `
INSERT INTO forge_k_restore_outcome_feedback_events(
  id, created_at, restore_outcome_id, original_outcome,
  workspace_id, lane_id, selected_paths_json,
  outcome, outcome_confidence, operator_feedback, correction_summary,
  prior_projection_json, projection_snapshot_json, metadata_json,
  correlation_id, trace_id, syscall_id, provenance_id, provenance_json, proposed_by, committed_by
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		event.ID, event.CreatedAt, event.RestoreOutcomeID, string(event.OriginalOutcome),
		event.Scope.WorkspaceID, event.Scope.LaneID, encodeStringSlice(event.Scope.SelectedPaths),
		string(event.Outcome), event.OutcomeConfidence, strings.TrimSpace(event.OperatorFeedback), strings.TrimSpace(event.CorrectionSummary),
		encodeJSON(event.PriorProjection), encodeJSON(event.ProjectionSnapshot), encodeJSON(nonNilMap(event.Metadata)),
		event.CorrelationID, event.TraceID, event.SyscallID, event.ProvenanceID, encodeJSON(event.Provenance), event.ProposedBy, event.CommittedBy,
	)
	if err != nil {
		return err
	}
	_, err = s.exec.ExecContext(ctx, `
INSERT INTO restore_outcome_feedback_projection(
  restore_outcome_id, latest_event_id, workspace_id, lane_id, selected_paths_json,
  outcome, outcome_confidence, operator_feedback, correction_summary,
  updated_by, updated_at, metadata_json, noncanonical
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,1)
ON CONFLICT(restore_outcome_id) DO UPDATE SET
  latest_event_id=excluded.latest_event_id,
  workspace_id=excluded.workspace_id,
  lane_id=excluded.lane_id,
  selected_paths_json=excluded.selected_paths_json,
  outcome=excluded.outcome,
  outcome_confidence=excluded.outcome_confidence,
  operator_feedback=excluded.operator_feedback,
  correction_summary=excluded.correction_summary,
  updated_by=excluded.updated_by,
  updated_at=excluded.updated_at,
  metadata_json=excluded.metadata_json,
  noncanonical=1`,
		event.RestoreOutcomeID, event.ID, event.Scope.WorkspaceID, event.Scope.LaneID, encodeStringSlice(event.Scope.SelectedPaths),
		string(feedback.Outcome), feedback.OutcomeConfidence, feedback.OperatorFeedback, feedback.CorrectionSummary,
		feedback.UpdatedBy, feedback.UpdatedAt, encodeJSON(nonNilMap(feedback.Metadata)),
	)
	return err
}

func optionalStringEqual(left, right *string) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return strings.TrimSpace(*left) == strings.TrimSpace(*right)
}

func optionalInt64Equal(left, right *int64) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func nullableStringPointer(value *string) any {
	if value == nil {
		return nil
	}
	return strings.TrimSpace(*value)
}

func nullableInt64Pointer(value *int64) any {
	if value == nil {
		return nil
	}
	return *value
}
