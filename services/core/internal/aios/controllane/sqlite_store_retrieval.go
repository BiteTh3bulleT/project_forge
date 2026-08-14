package controllane

import (
	"context"
	"fmt"
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
)

func (s *SQLiteSemanticStore) RecordRetrievalEvidence(ctx context.Context, req domain.SyscallRequest, evidence RetrievalEvidence) (RetrievalEvidenceCommit, error) {
	if s == nil || s.exec == nil {
		return RetrievalEvidenceCommit{}, fmt.Errorf("retrieval evidence store is unavailable")
	}
	committedBy := strings.TrimSpace(readString(req.Metadata, "kernelAuthorityOwner"))
	authorizationFingerprint := strings.TrimSpace(readString(req.Metadata, "forgeKAuthorizationProof"))
	runResult, err := s.exec.ExecContext(ctx, `
INSERT INTO retrieval_runs(
  evidence_id,created_at,query,mode,dossier_id,packet_id,job_id,
  original_dossier_id,original_packet_id,original_job_id,weighting_json,notes,
  workspace_id,lane_id,selected_paths_json,syscall_id,correlation_id,trace_id,
  provenance_id,provenance_json,proposed_by,committed_by,transaction_id,journal_event_id,
  audit_outbox_id,idempotency_key,authorization_fingerprint
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		evidence.EvidenceID, evidence.CreatedAt, strings.TrimSpace(evidence.Query), evidence.Mode,
		evidence.DossierID, evidence.PacketID, evidence.JobID, evidence.DossierID, evidence.PacketID, evidence.JobID,
		encodeJSON(evidence.Weighting), evidence.Notes,
		req.Scope.WorkspaceID, req.Scope.LaneID, encodeJSON(req.Scope.SelectedPaths), req.ID,
		req.CorrelationID, req.TraceID, provenanceID(req.Scope, req.Provenance), encodeJSON(req.Provenance),
		req.Actor.ID, committedBy, req.ID+":transaction", req.ID+":journal_event", req.ID+":audit_outbox",
		req.IdempotencyKey, authorizationFingerprint,
	)
	if err != nil {
		return RetrievalEvidenceCommit{}, err
	}
	runID, err := runResult.LastInsertId()
	if err != nil || runID <= 0 {
		return RetrievalEvidenceCommit{}, fmt.Errorf("resolve retrieval run id: %w", err)
	}
	commit := RetrievalEvidenceCommit{RunID: runID, ResultIDs: make([]int64, 0, len(evidence.Results))}
	for _, result := range evidence.Results {
		inserted, insertErr := s.exec.ExecContext(ctx, `
INSERT INTO retrieval_results(
  evidence_id,retrieval_run_id,chunk_id,file_id,original_chunk_id,original_file_id,abs_path,rel_path,rank_index,
  keyword_score,semantic_score,hybrid_score,snippet,selected_for_packet,usefulness_label,usefulness_note
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			result.EvidenceID, runID, result.ChunkID, result.FileID, result.ChunkID, result.FileID, result.AbsPath, result.RelPath, result.RankIndex,
			result.KeywordScore, result.SemanticScore, result.HybridScore, result.Snippet, boolToInt(result.Selected), "unknown", "",
		)
		if insertErr != nil {
			return RetrievalEvidenceCommit{}, insertErr
		}
		resultID, idErr := inserted.LastInsertId()
		if idErr != nil || resultID <= 0 {
			return RetrievalEvidenceCommit{}, fmt.Errorf("resolve retrieval result id at rank %d: %w", result.RankIndex, idErr)
		}
		if _, reasonErr := s.exec.ExecContext(ctx, `
INSERT INTO retrieval_result_selection(retrieval_result_id,reason_json,created_at)
VALUES(?,?,?)`, resultID, encodeJSON(result.SelectionReason), evidence.CreatedAt); reasonErr != nil {
			return RetrievalEvidenceCommit{}, reasonErr
		}
		commit.ResultIDs = append(commit.ResultIDs, resultID)
	}
	if evidence.PacketID != nil {
		if _, err := s.exec.ExecContext(ctx, `
INSERT INTO packet_retrieval_runs(packet_id,retrieval_run_id,created_at)
VALUES(?,?,?)`, *evidence.PacketID, runID, evidence.CreatedAt); err != nil {
			return RetrievalEvidenceCommit{}, err
		}
	}
	return commit, nil
}
