package controllane

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/forgekernel/semanticdiff"
)

func (s *SQLiteSemanticStore) CreateSemanticDiff(req domain.SyscallRequest, decision semanticdiff.Decision) error {
	if s == nil || s.exec == nil {
		return fmt.Errorf("semantic diff store unavailable")
	}
	if err := semanticdiff.VerifyDecision(req, decision); err != nil {
		return err
	}
	currentInput, err := prepareSemanticDiffAuthorityInput(req, s)
	if err != nil {
		return err
	}
	currentDecision, issues := semanticdiff.Decide(req, currentInput)
	if len(issues) > 0 {
		return fmt.Errorf("semantic diff source revalidation failed: %s", issues[0].Message)
	}
	if !reflect.DeepEqual(currentDecision, decision) {
		return fmt.Errorf("semantic diff sources changed after Kernel decision")
	}
	operation, result, object := semanticDiffRecords(req, decision)
	provenanceID, err := s.ensureProvenance(s.background, req.Scope, req.Provenance, map[string]any{
		"object_type": "semantic_diff_operation", "operation_id": operation.ID,
	}, req.RequestedAt)
	if err != nil {
		return err
	}
	if provenanceID != operation.ProvenanceID || provenanceID != object.ProvenanceID {
		return fmt.Errorf("semantic diff provenance identity mismatch")
	}
	if _, err = s.exec.ExecContext(s.background, `
INSERT INTO forge_k_semantic_diff_operations(
 id,operator_version,workspace_id,lane_id,selected_paths_json,left_evidence_id,right_evidence_id,
 left_source_json,right_source_json,source_manifest_hash,provenance_id,provenance_json,created_at,
 syscall_id,correlation_id,trace_id,proposed_by,committed_by,transaction_id,journal_event_id,
 audit_outbox_id,idempotency_key,authorization_fingerprint
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		operation.ID, operation.OperatorVersion, operation.Scope.WorkspaceID, operation.Scope.LaneID,
		encodeStringSlice(operation.Scope.SelectedPaths), operation.Left.EvidenceID, operation.Right.EvidenceID,
		encodeJSON(operation.Left), encodeJSON(operation.Right), operation.SourceManifestHash,
		operation.ProvenanceID, encodeJSON(operation.Provenance), operation.CreatedAt, operation.SyscallID,
		operation.CorrelationID, operation.TraceID, operation.ProposedBy, operation.CommittedBy,
		operation.TransactionID, operation.JournalEventID, operation.AuditOutboxID,
		operation.IdempotencyKey, operation.AuthorizationFingerprint,
	); err != nil {
		return err
	}
	if _, err = s.exec.ExecContext(s.background, `
INSERT INTO forge_k_semantic_diff_results(
 id,operation_id,operator_version,tokens_json,content,content_hash,source_manifest_hash,
 created_at,syscall_id,committed_by
) VALUES(?,?,?,?,?,?,?,?,?,?)`, result.ID, result.OperationID, result.OperatorVersion,
		encodeStringSlice(result.Tokens), result.Content, result.ContentHash, result.SourceManifestHash,
		result.CreatedAt, result.SyscallID, result.CommittedBy); err != nil {
		return err
	}
	_, err = s.exec.ExecContext(s.background, `
INSERT INTO forge_k_semantic_derived_objects(
 id,operation_id,result_id,object_class,workspace_id,lane_id,selected_paths_json,
 source_evidence_ids_json,source_manifest_hash,content,content_hash,canonical_truth,
 provenance_id,provenance_json,created_at,syscall_id,correlation_id,trace_id,committed_by
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, object.ID, object.OperationID, object.ResultID,
		object.ObjectClass, object.Scope.WorkspaceID, object.Scope.LaneID, encodeStringSlice(object.Scope.SelectedPaths),
		encodeStringSlice(object.SourceEvidenceIDs), object.SourceManifestHash, object.Content, object.ContentHash, 0,
		object.ProvenanceID, encodeJSON(object.Provenance), object.CreatedAt, object.SyscallID,
		object.CorrelationID, object.TraceID, object.CommittedBy)
	return err
}

func (s *SQLiteSemanticStore) FindSemanticDiffOperation(id string, scope domain.ForgeScope) (SemanticDiffOperation, bool) {
	row := s.exec.QueryRowContext(s.background, `
SELECT id,operator_version,workspace_id,lane_id,selected_paths_json,left_source_json,right_source_json,
       source_manifest_hash,provenance_id,provenance_json,created_at,syscall_id,correlation_id,trace_id,
       proposed_by,committed_by,transaction_id,journal_event_id,audit_outbox_id,idempotency_key,authorization_fingerprint
FROM forge_k_semantic_diff_operations WHERE id=? AND workspace_id=? AND lane_id=?`,
		strings.TrimSpace(id), scope.WorkspaceID, scope.LaneID)
	var operation SemanticDiffOperation
	var selectedPathsJSON, leftJSON, rightJSON, provenanceJSON string
	if err := row.Scan(&operation.ID, &operation.OperatorVersion, &operation.Scope.WorkspaceID, &operation.Scope.LaneID,
		&selectedPathsJSON, &leftJSON, &rightJSON, &operation.SourceManifestHash, &operation.ProvenanceID,
		&provenanceJSON, &operation.CreatedAt, &operation.SyscallID, &operation.CorrelationID, &operation.TraceID,
		&operation.ProposedBy, &operation.CommittedBy, &operation.TransactionID, &operation.JournalEventID,
		&operation.AuditOutboxID, &operation.IdempotencyKey, &operation.AuthorizationFingerprint); err != nil {
		return SemanticDiffOperation{}, false
	}
	operation.Scope.SelectedPaths = decodeStringSlice(selectedPathsJSON)
	if json.Unmarshal([]byte(leftJSON), &operation.Left) != nil || json.Unmarshal([]byte(rightJSON), &operation.Right) != nil ||
		json.Unmarshal([]byte(provenanceJSON), &operation.Provenance) != nil || !exactUtilityScopeMatches(operation.Scope, scope) {
		return SemanticDiffOperation{}, false
	}
	return operation, true
}

func (s *SQLiteSemanticStore) FindSemanticDiffResult(id string) (SemanticDiffResult, bool) {
	row := s.exec.QueryRowContext(s.background, `
SELECT id,operation_id,operator_version,tokens_json,content,content_hash,source_manifest_hash,created_at,syscall_id,committed_by
FROM forge_k_semantic_diff_results WHERE id=?`, strings.TrimSpace(id))
	var result SemanticDiffResult
	var tokensJSON string
	if err := row.Scan(&result.ID, &result.OperationID, &result.OperatorVersion, &tokensJSON, &result.Content,
		&result.ContentHash, &result.SourceManifestHash, &result.CreatedAt, &result.SyscallID, &result.CommittedBy); err != nil {
		return SemanticDiffResult{}, false
	}
	result.Tokens = decodeStringSlice(tokensJSON)
	return result, true
}

func (s *SQLiteSemanticStore) FindSemanticDerivedObject(id string, scope domain.ForgeScope) (SemanticDerivedObject, bool) {
	row := s.exec.QueryRowContext(s.background, `
SELECT id,operation_id,result_id,object_class,workspace_id,lane_id,selected_paths_json,
       source_evidence_ids_json,source_manifest_hash,content,content_hash,canonical_truth,
       provenance_id,provenance_json,created_at,syscall_id,correlation_id,trace_id,committed_by
FROM forge_k_semantic_derived_objects WHERE id=? AND workspace_id=? AND lane_id=?`,
		strings.TrimSpace(id), scope.WorkspaceID, scope.LaneID)
	var object SemanticDerivedObject
	var selectedPathsJSON, sourceIDsJSON, provenanceJSON string
	var canonical int
	if err := row.Scan(&object.ID, &object.OperationID, &object.ResultID, &object.ObjectClass,
		&object.Scope.WorkspaceID, &object.Scope.LaneID, &selectedPathsJSON, &sourceIDsJSON,
		&object.SourceManifestHash, &object.Content, &object.ContentHash, &canonical,
		&object.ProvenanceID, &provenanceJSON, &object.CreatedAt, &object.SyscallID,
		&object.CorrelationID, &object.TraceID, &object.CommittedBy); err != nil {
		return SemanticDerivedObject{}, false
	}
	object.Scope.SelectedPaths = decodeStringSlice(selectedPathsJSON)
	object.SourceEvidenceIDs = decodeStringSlice(sourceIDsJSON)
	object.CanonicalTruth = canonical != 0
	if json.Unmarshal([]byte(provenanceJSON), &object.Provenance) != nil || !exactUtilityScopeMatches(object.Scope, scope) {
		return SemanticDerivedObject{}, false
	}
	return object, true
}
