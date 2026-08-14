package controllane

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/forgekernel"
	"forge/projectforge/services/core/internal/forgekernel/court"
)

type MemoryEvidence struct {
	RowID                       int64             `json:"rowId"`
	EvidenceID                  string            `json:"evidenceId"`
	RootEvidenceID              string            `json:"rootEvidenceId"`
	Revision                    int               `json:"revision"`
	CourtCaseID                 string            `json:"courtCaseId"`
	CourtExhibitID              string            `json:"courtExhibitId"`
	CourtRulingID               string            `json:"courtRulingId"`
	AdmissionSyscallID          string            `json:"admissionSyscallId"`
	SourceObjectKind            string            `json:"sourceObjectKind"`
	SourceObjectID              string            `json:"sourceObjectId"`
	SourceObjectVersion         string            `json:"sourceObjectVersion"`
	SourceObjectHash            string            `json:"sourceObjectHash"`
	Scope                       domain.ForgeScope `json:"scope"`
	SourceType                  string            `json:"sourceType"`
	SourceRefs                  []string          `json:"sourceRefs"`
	ContentSummary              string            `json:"contentSummary"`
	RawRef                      string            `json:"rawRef"`
	ContentHash                 string            `json:"contentHash"`
	SourceProvenanceID          string            `json:"sourceProvenanceId"`
	SourceProvenance            domain.Provenance `json:"sourceProvenance"`
	MaterializationProvenanceID string            `json:"materializationProvenanceId"`
	MaterializationProvenance   domain.Provenance `json:"materializationProvenance"`
	CreatedAt                   int64             `json:"createdAt"`
	ProposedBy                  string            `json:"proposedBy"`
	CommittedBy                 string            `json:"committedBy"`
	SyscallID                   string            `json:"syscallId"`
	CorrelationID               string            `json:"correlationId"`
	TraceID                     string            `json:"traceId"`
	TransactionID               string            `json:"transactionId"`
	JournalEventID              string            `json:"journalEventId"`
	AuditOutboxID               string            `json:"auditOutboxId"`
	IdempotencyKey              string            `json:"idempotencyKey"`
	AuthorizationFingerprint    string            `json:"authorizationFingerprint"`
}

type MemoryEvidenceSupersession struct {
	ID                    string            `json:"id"`
	RootEvidenceID        string            `json:"rootEvidenceId"`
	SupersededEvidenceID  string            `json:"supersededEvidenceId"`
	ReplacementEvidenceID string            `json:"replacementEvidenceId"`
	Scope                 domain.ForgeScope `json:"scope"`
	ProvenanceID          string            `json:"provenanceId"`
	Provenance            domain.Provenance `json:"provenance"`
	CreatedAt             int64             `json:"createdAt"`
	SyscallID             string            `json:"syscallId"`
	CorrelationID         string            `json:"correlationId"`
	TraceID               string            `json:"traceId"`
	CommittedBy           string            `json:"committedBy"`
}

type memoryEvidencePayload struct {
	ExhibitID       string `json:"exhibitId"`
	RulingID        string `json:"rulingId"`
	PriorEvidenceID string `json:"priorEvidenceId,omitempty"`
}

func decodeMemoryEvidencePayload(payload map[string]any) (memoryEvidencePayload, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return memoryEvidencePayload{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var decoded memoryEvidencePayload
	if err := decoder.Decode(&decoded); err != nil {
		return memoryEvidencePayload{}, err
	}
	return decoded, nil
}

func validateMemoryEvidenceAction(req domain.SyscallRequest, store SemanticReadStore) []domain.SyscallError {
	payload, err := decodeMemoryEvidencePayload(req.Payload)
	if err != nil {
		return []domain.SyscallError{errField(domain.ErrInvalidPayload, "payload", err.Error())}
	}
	var issues []domain.SyscallError
	if strings.TrimSpace(payload.ExhibitID) == "" {
		issues = append(issues, errField(domain.ErrMissingRequiredField, "payload.exhibitId", "exhibitId is required"))
	}
	if strings.TrimSpace(payload.RulingID) == "" {
		issues = append(issues, errField(domain.ErrMissingRequiredField, "payload.rulingId", "rulingId is required"))
	}
	if req.Action == domain.ActionMaterializeAdmittedEvidence && strings.TrimSpace(payload.PriorEvidenceID) != "" {
		issues = append(issues, errField(domain.ErrInvalidPayload, "payload.priorEvidenceId", "initial materialization cannot name prior evidence"))
	}
	if req.Action == domain.ActionReviseMemoryEvidence && strings.TrimSpace(payload.PriorEvidenceID) == "" {
		issues = append(issues, errField(domain.ErrMissingRequiredField, "payload.priorEvidenceId", "revision requires priorEvidenceId"))
	}
	if strings.TrimSpace(req.Scope.WorkspaceID) == "" {
		issues = append(issues, errField(domain.ErrMissingRequiredField, "scope.workspaceId", "workspaceId is required"))
	}
	if strings.TrimSpace(req.Scope.LaneID) == "" {
		issues = append(issues, errField(domain.ErrMissingRequiredField, "scope.laneId", "laneId is required"))
	}
	if len(req.Scope.SelectedPaths) == 0 {
		issues = append(issues, errField(domain.ErrMissingRequiredField, "scope.selectedPaths", "selectedPaths are required"))
	}
	if strings.TrimSpace(req.IdempotencyKey) == "" {
		issues = append(issues, errField(domain.ErrMissingRequiredField, "idempotencyKey", "idempotencyKey is required"))
	}
	if req.Source != domain.SourceUser && req.Source != domain.SourceSystem && req.Source != domain.SourceInternal {
		issues = append(issues, errField(domain.ErrUnauthorized, "source", "memory evidence materialization requires an authenticated user or forge.core service principal"))
	}
	if store != nil && len(issues) == 0 {
		if _, _, _, resolveErr := resolveMemoryEvidence(req, payload, store); resolveErr != nil {
			issues = append(issues, errField(domain.ErrConflict, "payload", resolveErr.Error()))
		}
	}
	return issues
}

func resolveMemoryEvidence(req domain.SyscallRequest, payload memoryEvidencePayload, store SemanticReadStore) (MemoryEvidence, *MemoryEvidenceSupersession, court.Ruling, error) {
	exhibit, ok := store.FindCourtExhibit(strings.TrimSpace(payload.ExhibitID), req.Scope)
	if !ok {
		return MemoryEvidence{}, nil, court.Ruling{}, fmt.Errorf("admitted Court exhibit not found in exact scope")
	}
	ruling, ok := store.FindCourtRuling(strings.TrimSpace(payload.RulingID), req.Scope)
	if !ok {
		return MemoryEvidence{}, nil, court.Ruling{}, fmt.Errorf("Court ruling not found in exact scope")
	}
	if !exactUtilityScopeMatches(req.Scope, exhibit.Scope) || !exactUtilityScopeMatches(req.Scope, ruling.Scope) {
		return MemoryEvidence{}, nil, court.Ruling{}, fmt.Errorf("request, exhibit, and ruling scopes must match exactly")
	}
	if exhibit.Status != court.DecisionAdmitted || exhibit.CurrentRulingID != ruling.ID ||
		ruling.Decision != court.DecisionAdmitted || ruling.ExhibitID != exhibit.ID || ruling.CaseID != exhibit.CaseID {
		return MemoryEvidence{}, nil, court.Ruling{}, fmt.Errorf("exhibit must have the named admitted current ruling")
	}
	if !validMemoryEvidenceHash(exhibit.ContentHash) || ruling.ContentHash != exhibit.ContentHash {
		return MemoryEvidence{}, nil, court.Ruling{}, fmt.Errorf("persisted exhibit and ruling content hashes must match")
	}
	if strings.TrimSpace(exhibit.SyscallID) == "" || strings.TrimSpace(ruling.SyscallID) == "" ||
		strings.TrimSpace(exhibit.CommittedBy) != forgekernel.AuthorityOwnerForgeK ||
		strings.TrimSpace(ruling.CommittedBy) != forgekernel.AuthorityOwnerForgeK {
		return MemoryEvidence{}, nil, court.Ruling{}, fmt.Errorf("Court source lacks production admission authority")
	}
	if req.RequestedAt < exhibit.CreatedAt || req.RequestedAt < ruling.CreatedAt {
		return MemoryEvidence{}, nil, court.Ruling{}, fmt.Errorf("materialization cannot predate its admitted Court source")
	}
	evidenceID := req.ID + ":memory_evidence"
	evidence := MemoryEvidence{
		EvidenceID: evidenceID, RootEvidenceID: evidenceID, Revision: 1,
		CourtCaseID: exhibit.CaseID, CourtExhibitID: exhibit.ID, CourtRulingID: ruling.ID,
		AdmissionSyscallID: ruling.SyscallID, SourceObjectKind: "court_exhibit", SourceObjectID: exhibit.ID,
		SourceObjectVersion: ruling.ID, SourceObjectHash: exhibit.ContentHash, Scope: req.Scope,
		SourceType: exhibit.SourceType, SourceRefs: append([]string(nil), exhibit.SourceRefs...),
		ContentSummary: exhibit.ContentSummary, RawRef: exhibit.RawRef, ContentHash: exhibit.ContentHash,
		SourceProvenanceID: provenanceID(exhibit.Scope, exhibit.Provenance), SourceProvenance: exhibit.Provenance,
		MaterializationProvenanceID: provenanceID(req.Scope, req.Provenance), MaterializationProvenance: req.Provenance,
		CreatedAt: req.RequestedAt, ProposedBy: req.Actor.ID, CommittedBy: forgekernel.AuthorityOwnerForgeK,
		SyscallID: req.ID, CorrelationID: req.CorrelationID, TraceID: req.TraceID,
		TransactionID: req.ID + ":transaction", JournalEventID: req.ID + ":journal_event",
		AuditOutboxID: req.ID + ":audit_outbox", IdempotencyKey: req.IdempotencyKey,
		AuthorizationFingerprint: strings.TrimSpace(readString(req.Metadata, "forgeKAuthorizationProof")),
	}
	if req.Action == domain.ActionMaterializeAdmittedEvidence {
		return evidence, nil, ruling, nil
	}
	prior, ok := store.FindMemoryEvidence(strings.TrimSpace(payload.PriorEvidenceID), req.Scope)
	if !ok || store.HasMemoryEvidenceSupersession(prior.EvidenceID) {
		return MemoryEvidence{}, nil, court.Ruling{}, fmt.Errorf("prior memory evidence must exist and remain the current leaf")
	}
	if prior.CourtCaseID != exhibit.CaseID || !exactUtilityScopeMatches(prior.Scope, req.Scope) {
		return MemoryEvidence{}, nil, court.Ruling{}, fmt.Errorf("revision must preserve exact case and scope")
	}
	if req.RequestedAt < prior.CreatedAt {
		return MemoryEvidence{}, nil, court.Ruling{}, fmt.Errorf("revision cannot predate its prior evidence")
	}
	if prior.CourtExhibitID == exhibit.ID && prior.CourtRulingID == ruling.ID {
		return MemoryEvidence{}, nil, court.Ruling{}, fmt.Errorf("revision must use a newly admitted Court version")
	}
	evidence.RootEvidenceID = prior.RootEvidenceID
	evidence.Revision = prior.Revision + 1
	edge := &MemoryEvidenceSupersession{
		ID: req.ID + ":memory_supersession", RootEvidenceID: prior.RootEvidenceID,
		SupersededEvidenceID: prior.EvidenceID, ReplacementEvidenceID: evidence.EvidenceID,
		Scope: req.Scope, ProvenanceID: evidence.MaterializationProvenanceID, Provenance: req.Provenance,
		CreatedAt: req.RequestedAt, SyscallID: req.ID, CorrelationID: req.CorrelationID,
		TraceID: req.TraceID, CommittedBy: forgekernel.AuthorityOwnerForgeK,
	}
	return evidence, edge, ruling, nil
}

func validMemoryEvidenceHash(value string) bool {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "sha256:") || len(value) != 71 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}

func applyMemoryEvidenceAction(_ context.Context, store SemanticStore, req domain.SyscallRequest) ([]string, map[string]any, []string, []domain.SyscallError) {
	if ingress, _ := req.Metadata["forgeKIngressAuthority"].(bool); !ingress || readString(req.Metadata, "kernelAuthorityOwner") != forgekernel.AuthorityOwnerForgeK {
		return nil, nil, nil, []domain.SyscallError{{Code: domain.ErrUnauthorized, Field: "metadata.kernelAuthorityOwner", Message: "memory evidence requires production FORGE-K ingress"}}
	}
	payload, err := decodeMemoryEvidencePayload(req.Payload)
	if err != nil {
		return nil, nil, nil, []domain.SyscallError{{Code: domain.ErrInvalidPayload, Field: "payload", Message: err.Error()}}
	}
	evidence, supersession, _, err := resolveMemoryEvidence(req, payload, store)
	if err != nil {
		return nil, nil, nil, []domain.SyscallError{{Code: domain.ErrConflict, Field: "memoryEvidence", Message: err.Error()}}
	}
	if err := store.CreateMemoryEvidence(evidence, supersession); err != nil {
		return nil, nil, nil, []domain.SyscallError{{Code: domain.ErrConflict, Field: "memoryEvidence", Message: err.Error()}}
	}
	ids := []string{evidence.EvidenceID}
	if supersession != nil {
		ids = append(ids, supersession.ID)
	}
	return ids, map[string]any{"memoryEvidence": map[string]any{
		"evidenceId": evidence.EvidenceID, "rootEvidenceId": evidence.RootEvidenceID,
		"revision": evidence.Revision, "courtExhibitId": evidence.CourtExhibitID,
		"courtRulingId": evidence.CourtRulingID, "contentHash": evidence.ContentHash,
		"immutable": true, "canonicalTruth": false, "courtAdmitted": true,
	}}, nil, nil
}

func (s *InMemorySemanticStore) FindMemoryEvidence(id string, scope domain.ForgeScope) (MemoryEvidence, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	evidence, ok := s.state.memoryEvidence[strings.TrimSpace(id)]
	return cloneIntegrityValue(evidence), ok && exactUtilityScopeMatches(evidence.Scope, scope)
}

func (s *InMemorySemanticStore) HasMemoryEvidenceSupersession(id string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, edge := range s.state.memoryEvidenceSupersession {
		if edge.SupersededEvidenceID == strings.TrimSpace(id) {
			return true
		}
	}
	return false
}

func (s *InMemorySemanticStore) CreateMemoryEvidence(evidence MemoryEvidence, edge *MemoryEvidenceSupersession) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return createMemoryEvidenceInState(&s.state, evidence, edge)
}

func (s *TransactionalSemanticStore) FindMemoryEvidence(id string, scope domain.ForgeScope) (MemoryEvidence, bool) {
	evidence, ok := s.state.memoryEvidence[strings.TrimSpace(id)]
	return cloneIntegrityValue(evidence), ok && exactUtilityScopeMatches(evidence.Scope, scope)
}

func (s *TransactionalSemanticStore) HasMemoryEvidenceSupersession(id string) bool {
	for _, edge := range s.state.memoryEvidenceSupersession {
		if edge.SupersededEvidenceID == strings.TrimSpace(id) {
			return true
		}
	}
	return false
}

func (s *TransactionalSemanticStore) CreateMemoryEvidence(evidence MemoryEvidence, edge *MemoryEvidenceSupersession) error {
	return createMemoryEvidenceInState(s.state, evidence, edge)
}

func createMemoryEvidenceInState(state *memoryState, evidence MemoryEvidence, edge *MemoryEvidenceSupersession) error {
	if state == nil {
		return fmt.Errorf("memory evidence state unavailable")
	}
	if _, exists := state.memoryEvidence[evidence.EvidenceID]; exists {
		return fmt.Errorf("memory evidence already exists")
	}
	for _, existing := range state.memoryEvidence {
		if existing.CourtExhibitID == evidence.CourtExhibitID && existing.CourtRulingID == evidence.CourtRulingID {
			return fmt.Errorf("Court exhibit/ruling is already materialized")
		}
	}
	if edge != nil {
		prior, exists := state.memoryEvidence[edge.SupersededEvidenceID]
		if !exists || prior.RootEvidenceID != evidence.RootEvidenceID || prior.Revision+1 != evidence.Revision {
			return fmt.Errorf("invalid memory evidence revision lineage")
		}
		for _, existing := range state.memoryEvidenceSupersession {
			if existing.SupersededEvidenceID == edge.SupersededEvidenceID || existing.ReplacementEvidenceID == edge.ReplacementEvidenceID {
				return fmt.Errorf("memory evidence leaf was concurrently superseded")
			}
		}
		state.memoryEvidenceSupersession[edge.ID] = cloneIntegrityValue(*edge)
	}
	state.memoryEvidence[evidence.EvidenceID] = cloneIntegrityValue(evidence)
	return nil
}
