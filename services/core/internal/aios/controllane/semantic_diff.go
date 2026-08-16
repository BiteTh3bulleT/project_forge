package controllane

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/forgekernel"
	"forge/projectforge/services/core/internal/forgekernel/court"
	"forge/projectforge/services/core/internal/forgekernel/semanticdiff"
)

type SemanticDiffOperation struct {
	ID                       string                `json:"id"`
	OperatorVersion          string                `json:"operatorVersion"`
	Scope                    domain.ForgeScope     `json:"scope"`
	Left                     semanticdiff.Evidence `json:"left"`
	Right                    semanticdiff.Evidence `json:"right"`
	SourceManifestHash       string                `json:"sourceManifestHash"`
	ProvenanceID             string                `json:"provenanceId"`
	Provenance               domain.Provenance     `json:"provenance"`
	CreatedAt                int64                 `json:"createdAt"`
	SyscallID                string                `json:"syscallId"`
	CorrelationID            string                `json:"correlationId"`
	TraceID                  string                `json:"traceId"`
	ProposedBy               string                `json:"proposedBy"`
	CommittedBy              string                `json:"committedBy"`
	TransactionID            string                `json:"transactionId"`
	JournalEventID           string                `json:"journalEventId"`
	AuditOutboxID            string                `json:"auditOutboxId"`
	IdempotencyKey           string                `json:"idempotencyKey"`
	AuthorizationFingerprint string                `json:"authorizationFingerprint"`
}

type SemanticDiffResult struct {
	ID                 string   `json:"id"`
	OperationID        string   `json:"operationId"`
	OperatorVersion    string   `json:"operatorVersion"`
	Tokens             []string `json:"tokens"`
	Content            string   `json:"content"`
	ContentHash        string   `json:"contentHash"`
	SourceManifestHash string   `json:"sourceManifestHash"`
	CreatedAt          int64    `json:"createdAt"`
	SyscallID          string   `json:"syscallId"`
	CommittedBy        string   `json:"committedBy"`
}

type SemanticDerivedObject struct {
	ID                 string            `json:"id"`
	OperationID        string            `json:"operationId"`
	ResultID           string            `json:"resultId"`
	ObjectClass        string            `json:"objectClass"`
	Scope              domain.ForgeScope `json:"scope"`
	SourceEvidenceIDs  []string          `json:"sourceEvidenceIds"`
	SourceManifestHash string            `json:"sourceManifestHash"`
	Content            string            `json:"content"`
	ContentHash        string            `json:"contentHash"`
	CanonicalTruth     bool              `json:"canonicalTruth"`
	ProvenanceID       string            `json:"provenanceId"`
	Provenance         domain.Provenance `json:"provenance"`
	CreatedAt          int64             `json:"createdAt"`
	SyscallID          string            `json:"syscallId"`
	CorrelationID      string            `json:"correlationId"`
	TraceID            string            `json:"traceId"`
	CommittedBy        string            `json:"committedBy"`
}

type semanticDiffPayload struct {
	LeftEvidenceID  string `json:"leftEvidenceId"`
	RightEvidenceID string `json:"rightEvidenceId"`
	OperatorVersion string `json:"operatorVersion"`
}

func decodeSemanticDiffPayload(payload map[string]any) (semanticDiffPayload, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return semanticDiffPayload{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var decoded semanticDiffPayload
	if err := decoder.Decode(&decoded); err != nil {
		return semanticDiffPayload{}, err
	}
	return decoded, nil
}

func validateComputeSemanticDiff(req domain.SyscallRequest, store SemanticReadStore) []domain.SyscallError {
	payload, err := decodeSemanticDiffPayload(req.Payload)
	if err != nil {
		return []domain.SyscallError{errField(domain.ErrInvalidPayload, "payload", err.Error())}
	}
	var issues []domain.SyscallError
	if strings.TrimSpace(payload.LeftEvidenceID) == "" {
		issues = append(issues, errField(domain.ErrMissingRequiredField, "payload.leftEvidenceId", "leftEvidenceId is required"))
	}
	if strings.TrimSpace(payload.RightEvidenceID) == "" {
		issues = append(issues, errField(domain.ErrMissingRequiredField, "payload.rightEvidenceId", "rightEvidenceId is required"))
	}
	if strings.TrimSpace(payload.OperatorVersion) != semanticdiff.OperatorVersion {
		issues = append(issues, errField(domain.ErrInvalidPayload, "payload.operatorVersion", "operatorVersion must be semantic.diff.v1"))
	}
	if strings.TrimSpace(payload.LeftEvidenceID) != "" && strings.TrimSpace(payload.LeftEvidenceID) == strings.TrimSpace(payload.RightEvidenceID) {
		issues = append(issues, errField(domain.ErrInvalidPayload, "payload.rightEvidenceId", "semantic diff requires distinct evidence objects"))
	}
	if strings.TrimSpace(req.Scope.WorkspaceID) == "" || strings.TrimSpace(req.Scope.LaneID) == "" || len(req.Scope.SelectedPaths) == 0 {
		issues = append(issues, errField(domain.ErrInvalidScope, "scope", "workspace, lane, and selectedPaths are required"))
	}
	if strings.TrimSpace(req.IdempotencyKey) == "" {
		issues = append(issues, errField(domain.ErrMissingRequiredField, "idempotencyKey", "idempotencyKey is required"))
	}
	if req.Source != domain.SourceUser && req.Source != domain.SourceSystem && req.Source != domain.SourceInternal {
		issues = append(issues, errField(domain.ErrUnauthorized, "source", "semantic algebra requires an authenticated user or forge.core service principal"))
	}
	if store != nil && len(issues) == 0 {
		if _, resolveErr := prepareSemanticDiffAuthorityInput(req, store); resolveErr != nil {
			issues = append(issues, errField(domain.ErrConflict, "semanticDiff.sources", resolveErr.Error()))
		}
	}
	return issues
}

func prepareSemanticDiffAuthorityInput(req domain.SyscallRequest, store SemanticReadStore) (semanticdiff.AuthorityInput, error) {
	payload, err := decodeSemanticDiffPayload(req.Payload)
	if err != nil {
		return semanticdiff.AuthorityInput{}, err
	}
	resolve := func(id string) (semanticdiff.Evidence, error) {
		evidence, ok := store.FindMemoryEvidence(strings.TrimSpace(id), req.Scope)
		if !ok || store.HasMemoryEvidenceSupersession(evidence.EvidenceID) {
			return semanticdiff.Evidence{}, fmt.Errorf("memory evidence %q is missing, out of scope, or superseded", id)
		}
		exhibit, ok := store.FindCourtExhibit(evidence.CourtExhibitID, req.Scope)
		if !ok {
			return semanticdiff.Evidence{}, fmt.Errorf("memory evidence %q lacks its Court exhibit", id)
		}
		ruling, ok := store.FindCourtRuling(evidence.CourtRulingID, req.Scope)
		if !ok || exhibit.Status != court.DecisionAdmitted || ruling.Decision != court.DecisionAdmitted ||
			exhibit.CurrentRulingID != ruling.ID || ruling.ExhibitID != exhibit.ID || ruling.CaseID != exhibit.CaseID ||
			ruling.ContentHash != evidence.ContentHash || exhibit.ContentHash != evidence.ContentHash {
			return semanticdiff.Evidence{}, fmt.Errorf("memory evidence %q is not backed by the current admitted Court ruling", id)
		}
		return semanticdiff.Evidence{
			RowID: evidence.RowID, EvidenceID: evidence.EvidenceID, Scope: evidence.Scope,
			Content: evidence.ContentSummary, MaterialHash: semanticdiff.MaterialHash(evidence.ContentSummary),
			EvidenceHash: evidence.ContentHash, CourtCaseID: evidence.CourtCaseID,
			CourtExhibitID: evidence.CourtExhibitID, CourtRulingID: evidence.CourtRulingID,
			AdmissionSyscallID:          evidence.AdmissionSyscallID,
			SourceProvenanceID:          evidence.SourceProvenanceID,
			MaterializationProvenanceID: evidence.MaterializationProvenanceID,
			CreatedAt:                   evidence.CreatedAt, CommittedBy: evidence.CommittedBy,
			Current: true, Admitted: true,
		}, nil
	}
	left, err := resolve(payload.LeftEvidenceID)
	if err != nil {
		return semanticdiff.AuthorityInput{}, err
	}
	right, err := resolve(payload.RightEvidenceID)
	if err != nil {
		return semanticdiff.AuthorityInput{}, err
	}
	return semanticdiff.AuthorityInput{Left: left, Right: right}, nil
}

func applyComputeSemanticDiff(_ context.Context, store SemanticStore, req domain.SyscallRequest) ([]string, map[string]any, []string, []domain.SyscallError) {
	if ingress, _ := req.Metadata["forgeKIngressAuthority"].(bool); !ingress || readString(req.Metadata, "kernelAuthorityOwner") != forgekernel.AuthorityOwnerForgeK {
		return nil, nil, nil, []domain.SyscallError{{Code: domain.ErrUnauthorized, Field: "metadata.kernelAuthorityOwner", Message: "semantic algebra requires production FORGE-K ingress"}}
	}
	decision, ok := semanticdiff.DecisionFromMetadata(req.Metadata)
	if !ok {
		return nil, nil, nil, []domain.SyscallError{{Code: domain.ErrUnauthorized, Field: "metadata." + semanticdiff.MetadataDecisionKey, Message: "missing production FORGE-K semantic decision"}}
	}
	if err := semanticdiff.VerifyDecision(req, decision); err != nil {
		return nil, nil, nil, []domain.SyscallError{{Code: domain.ErrConflict, Field: "semanticDiff.decision", Message: err.Error()}}
	}
	if err := store.CreateSemanticDiff(req, decision); err != nil {
		return nil, nil, nil, []domain.SyscallError{{Code: domain.ErrConflict, Field: "semanticDiff", Message: err.Error()}}
	}
	ids := semanticDiffObjectIDs(req.ID)
	return ids, map[string]any{
		"semanticOperationId": ids[0], "semanticResultId": ids[1], "semanticObjectId": ids[2],
		"operatorVersion": decision.OperatorVersion, "contentHash": decision.ContentHash,
		"sourceManifestHash": decision.SourceManifestHash, "objectClass": decision.ObjectClass,
		"canonicalTruth": false,
	}, []string{"semantic diff output is non-canonical derived evidence and requires separate Courthouse admission"}, nil
}

func semanticDiffObjectIDs(requestID string) []string {
	requestID = strings.TrimSpace(requestID)
	return []string{requestID + ":semantic_operation", requestID + ":semantic_result", requestID + ":semantic_object"}
}

func semanticDiffRecords(req domain.SyscallRequest, decision semanticdiff.Decision) (SemanticDiffOperation, SemanticDiffResult, SemanticDerivedObject) {
	ids := semanticDiffObjectIDs(req.ID)
	provID := provenanceID(req.Scope, req.Provenance)
	operation := SemanticDiffOperation{
		ID: ids[0], OperatorVersion: decision.OperatorVersion, Scope: req.Scope,
		Left: decision.Left, Right: decision.Right, SourceManifestHash: decision.SourceManifestHash,
		ProvenanceID: provID, Provenance: req.Provenance, CreatedAt: req.RequestedAt,
		SyscallID: req.ID, CorrelationID: req.CorrelationID, TraceID: req.TraceID,
		ProposedBy: req.Actor.ID, CommittedBy: forgekernel.AuthorityOwnerForgeK,
		TransactionID: req.ID + ":transaction", JournalEventID: req.ID + ":journal_event",
		AuditOutboxID: req.ID + ":audit_outbox", IdempotencyKey: req.IdempotencyKey,
		AuthorizationFingerprint: readString(req.Metadata, "forgeKAuthorizationProof"),
	}
	result := SemanticDiffResult{
		ID: ids[1], OperationID: operation.ID, OperatorVersion: decision.OperatorVersion,
		Tokens: append([]string(nil), decision.Tokens...), Content: decision.Content,
		ContentHash: decision.ContentHash, SourceManifestHash: decision.SourceManifestHash,
		CreatedAt: req.RequestedAt, SyscallID: req.ID, CommittedBy: forgekernel.AuthorityOwnerForgeK,
	}
	object := SemanticDerivedObject{
		ID: ids[2], OperationID: operation.ID, ResultID: result.ID, ObjectClass: decision.ObjectClass,
		Scope: req.Scope, SourceEvidenceIDs: []string{decision.Left.EvidenceID, decision.Right.EvidenceID},
		SourceManifestHash: decision.SourceManifestHash, Content: decision.Content, ContentHash: decision.ContentHash,
		CanonicalTruth: false, ProvenanceID: provID, Provenance: req.Provenance,
		CreatedAt: req.RequestedAt, SyscallID: req.ID, CorrelationID: req.CorrelationID,
		TraceID: req.TraceID, CommittedBy: forgekernel.AuthorityOwnerForgeK,
	}
	return operation, result, object
}

func (s *InMemorySemanticStore) FindSemanticDiffOperation(id string, scope domain.ForgeScope) (SemanticDiffOperation, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, ok := s.state.semanticDiffOperations[strings.TrimSpace(id)]
	return cloneIntegrityValue(record), ok && exactUtilityScopeMatches(record.Scope, scope)
}

func (s *InMemorySemanticStore) FindSemanticDiffResult(id string) (SemanticDiffResult, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, ok := s.state.semanticDiffResults[strings.TrimSpace(id)]
	return cloneIntegrityValue(record), ok
}

func (s *InMemorySemanticStore) FindSemanticDerivedObject(id string, scope domain.ForgeScope) (SemanticDerivedObject, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, ok := s.state.semanticDerivedObjects[strings.TrimSpace(id)]
	return cloneIntegrityValue(record), ok && exactUtilityScopeMatches(record.Scope, scope)
}

func (s *InMemorySemanticStore) CreateSemanticDiff(req domain.SyscallRequest, decision semanticdiff.Decision) error {
	input, err := prepareSemanticDiffAuthorityInput(req, s)
	if err != nil {
		return err
	}
	current, issues := semanticdiff.Decide(req, input)
	if len(issues) > 0 || !semanticDiffDecisionsEqual(current, decision) {
		return fmt.Errorf("semantic diff sources changed after Kernel decision")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return createSemanticDiffInState(&s.state, req, decision)
}

func (s *TransactionalSemanticStore) FindSemanticDiffOperation(id string, scope domain.ForgeScope) (SemanticDiffOperation, bool) {
	record, ok := s.state.semanticDiffOperations[strings.TrimSpace(id)]
	return cloneIntegrityValue(record), ok && exactUtilityScopeMatches(record.Scope, scope)
}

func (s *TransactionalSemanticStore) FindSemanticDiffResult(id string) (SemanticDiffResult, bool) {
	record, ok := s.state.semanticDiffResults[strings.TrimSpace(id)]
	return cloneIntegrityValue(record), ok
}

func (s *TransactionalSemanticStore) FindSemanticDerivedObject(id string, scope domain.ForgeScope) (SemanticDerivedObject, bool) {
	record, ok := s.state.semanticDerivedObjects[strings.TrimSpace(id)]
	return cloneIntegrityValue(record), ok && exactUtilityScopeMatches(record.Scope, scope)
}

func (s *TransactionalSemanticStore) CreateSemanticDiff(req domain.SyscallRequest, decision semanticdiff.Decision) error {
	input, err := prepareSemanticDiffAuthorityInput(req, s)
	if err != nil {
		return err
	}
	current, issues := semanticdiff.Decide(req, input)
	if len(issues) > 0 || !semanticDiffDecisionsEqual(current, decision) {
		return fmt.Errorf("semantic diff sources changed after Kernel decision")
	}
	return createSemanticDiffInState(s.state, req, decision)
}

func semanticDiffDecisionsEqual(left, right semanticdiff.Decision) bool {
	leftJSON, leftErr := json.Marshal(left)
	rightJSON, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftJSON, rightJSON)
}

func createSemanticDiffInState(state *memoryState, req domain.SyscallRequest, decision semanticdiff.Decision) error {
	if state == nil {
		return fmt.Errorf("semantic diff state unavailable")
	}
	if err := semanticdiff.VerifyDecision(req, decision); err != nil {
		return err
	}
	operation, result, object := semanticDiffRecords(req, decision)
	if _, ok := state.semanticDiffOperations[operation.ID]; ok {
		return fmt.Errorf("semantic diff operation already exists")
	}
	if _, ok := state.semanticDiffResults[result.ID]; ok {
		return fmt.Errorf("semantic diff result already exists")
	}
	if _, ok := state.semanticDerivedObjects[object.ID]; ok {
		return fmt.Errorf("semantic derived object already exists")
	}
	state.semanticDiffOperations[operation.ID] = cloneIntegrityValue(operation)
	state.semanticDiffResults[result.ID] = cloneIntegrityValue(result)
	state.semanticDerivedObjects[object.ID] = cloneIntegrityValue(object)
	return nil
}
