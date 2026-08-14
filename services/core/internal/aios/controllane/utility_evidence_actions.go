package controllane

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/forgekernel"
)

const (
	maxUtilityNoteBytes       = 4096
	maxUtilityFeedbackBytes   = 16384
	maxUtilityCorrectionBytes = 8192
	maxUtilityMetadataBytes   = 16384
)

type recordRetrievalUsefulnessPayload struct {
	ResultID int64          `json:"resultId"`
	Label    string         `json:"label"`
	Note     string         `json:"note,omitempty"`
	JobID    *string        `json:"jobId,omitempty"`
	PacketID *int64         `json:"packetId,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type recordRestoreOutcomeFeedbackPayload struct {
	RestoreOutcomeID  string         `json:"restoreOutcomeId"`
	Outcome           RestoreOutcome `json:"outcome"`
	OutcomeConfidence float64        `json:"outcomeConfidence"`
	OperatorFeedback  string         `json:"operatorFeedback,omitempty"`
	CorrectionSummary string         `json:"correctionSummary,omitempty"`
	Metadata          map[string]any `json:"metadata,omitempty"`
}

type utilityEvidenceReadStore interface {
	FindRetrievalUsefulnessTarget(context.Context, int64) (RetrievalUsefulnessTarget, bool, error)
	FindRestoreOutcomeFeedbackTarget(context.Context, string) (RestoreOutcomeFeedbackTarget, bool, error)
}

type utilityEvidenceWriteStore interface {
	CreateRetrievalUsefulnessEvent(RetrievalUsefulnessEvent) error
	CreateRestoreOutcomeFeedbackEvent(RestoreOutcomeFeedbackEvent) error
}

func decodeStrictUtilityPayload(payload map[string]any, dst any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return fmt.Errorf("multiple payload values")
		}
		return err
	}
	return nil
}

func validateRecordRetrievalUsefulness(req domain.SyscallRequest, store SemanticReadStore) []domain.SyscallError {
	var payload recordRetrievalUsefulnessPayload
	if err := decodeStrictUtilityPayload(req.Payload, &payload); err != nil {
		return []domain.SyscallError{errField(domain.ErrInvalidPayload, "payload", "invalid retrieval usefulness payload: "+err.Error())}
	}
	issues := validateUtilityEnvelope(req)
	if payload.ResultID <= 0 {
		issues = append(issues, errField(domain.ErrInvalidPayload, "payload.resultId", "resultId must be positive"))
	}
	if !ValidateRetrievalUsefulnessLabel(payload.Label) {
		issues = append(issues, errField(domain.ErrInvalidPayload, "payload.label", "label must be useful|not_useful|noisy|insufficient|unknown"))
	}
	if len([]byte(strings.TrimSpace(payload.Note))) > maxUtilityNoteBytes {
		issues = append(issues, errField(domain.ErrInvalidPayload, "payload.note", "note exceeds 4096 bytes"))
	}
	if payload.JobID != nil && strings.TrimSpace(*payload.JobID) == "" {
		issues = append(issues, errField(domain.ErrInvalidPayload, "payload.jobId", "jobId must not be empty"))
	}
	if payload.PacketID != nil && *payload.PacketID <= 0 {
		issues = append(issues, errField(domain.ErrInvalidPayload, "payload.packetId", "packetId must be positive"))
	}
	issues = append(issues, validateUtilityMetadata(payload.Metadata)...)
	read, ok := store.(utilityEvidenceReadStore)
	if !ok || payload.ResultID <= 0 {
		if !ok {
			issues = append(issues, errField(domain.ErrPersistenceUnavailable, "payload.resultId", "utility evidence target store unavailable"))
		}
		return issues
	}
	target, found, err := read.FindRetrievalUsefulnessTarget(context.Background(), payload.ResultID)
	if err != nil {
		return append(issues, errField(domain.ErrPersistenceUnavailable, "payload.resultId", err.Error()))
	}
	if !found {
		return append(issues, errField(domain.ErrNotFound, "payload.resultId", "retrieval result evidence not found or legacy-unbound"))
	}
	if !exactUtilityScopeMatches(req.Scope, target.Scope) {
		issues = append(issues, errField(domain.ErrInvalidScope, "scope", "retrieval result workspace/lane/selectedPaths identity mismatch"))
	}
	if strings.TrimSpace(target.EvidenceID) == "" || strings.TrimSpace(target.SourceSyscall) == "" ||
		strings.TrimSpace(target.SourceProvID) == "" || strings.TrimSpace(target.SourceProvJSON) == "" {
		issues = append(issues, errField(domain.ErrInvalidProvenance, "payload.resultId", "retrieval result evidence is legacy-unbound"))
	}
	if payload.JobID != nil && !optionalStringEqual(payload.JobID, target.JobID) {
		issues = append(issues, errField(domain.ErrConflict, "payload.jobId", "jobId does not match retrieval run evidence"))
	}
	if payload.PacketID != nil && !optionalInt64Equal(payload.PacketID, target.PacketID) {
		issues = append(issues, errField(domain.ErrConflict, "payload.packetId", "packetId does not match retrieval run evidence"))
	}
	return issues
}

func validateRecordRestoreOutcomeFeedback(req domain.SyscallRequest, store SemanticReadStore) []domain.SyscallError {
	var payload recordRestoreOutcomeFeedbackPayload
	if err := decodeStrictUtilityPayload(req.Payload, &payload); err != nil {
		return []domain.SyscallError{errField(domain.ErrInvalidPayload, "payload", "invalid restore outcome feedback payload: "+err.Error())}
	}
	issues := validateUtilityEnvelope(req)
	payload.RestoreOutcomeID = strings.TrimSpace(payload.RestoreOutcomeID)
	if payload.RestoreOutcomeID == "" {
		issues = append(issues, errField(domain.ErrMissingRequiredField, "payload.restoreOutcomeId", "restoreOutcomeId is required"))
	}
	if !ValidateRestoreOutcome(payload.Outcome) || payload.Outcome == RestoreOutcomeUnknown {
		issues = append(issues, errField(domain.ErrInvalidPayload, "payload.outcome", "valid non-unknown outcome is required"))
	}
	if payload.OutcomeConfidence < 0 || payload.OutcomeConfidence > 1 {
		issues = append(issues, errField(domain.ErrInvalidPayload, "payload.outcomeConfidence", "outcomeConfidence must be between 0 and 1"))
	}
	if len([]byte(strings.TrimSpace(payload.OperatorFeedback))) > maxUtilityFeedbackBytes {
		issues = append(issues, errField(domain.ErrInvalidPayload, "payload.operatorFeedback", "operatorFeedback exceeds 16384 bytes"))
	}
	if len([]byte(strings.TrimSpace(payload.CorrectionSummary))) > maxUtilityCorrectionBytes {
		issues = append(issues, errField(domain.ErrInvalidPayload, "payload.correctionSummary", "correctionSummary exceeds 8192 bytes"))
	}
	issues = append(issues, validateUtilityMetadata(payload.Metadata)...)
	read, ok := store.(utilityEvidenceReadStore)
	if !ok || payload.RestoreOutcomeID == "" {
		if !ok {
			issues = append(issues, errField(domain.ErrPersistenceUnavailable, "payload.restoreOutcomeId", "utility evidence target store unavailable"))
		}
		return issues
	}
	target, found, err := read.FindRestoreOutcomeFeedbackTarget(context.Background(), payload.RestoreOutcomeID)
	if err != nil {
		return append(issues, errField(domain.ErrPersistenceUnavailable, "payload.restoreOutcomeId", err.Error()))
	}
	if !found {
		return append(issues, errField(domain.ErrNotFound, "payload.restoreOutcomeId", "restore outcome evidence not found or legacy-unbound"))
	}
	if !exactUtilityScopeMatches(req.Scope, target.Scope) {
		issues = append(issues, errField(domain.ErrInvalidScope, "scope", "restore outcome workspace/lane/selectedPaths identity mismatch"))
	}
	if strings.TrimSpace(target.SourceSyscall) == "" || strings.TrimSpace(target.CommittedBy) == "" {
		issues = append(issues, errField(domain.ErrInvalidProvenance, "payload.restoreOutcomeId", "restore outcome evidence is legacy-unbound"))
	}
	return issues
}

func validateUtilityEnvelope(req domain.SyscallRequest) []domain.SyscallError {
	issues := []domain.SyscallError{}
	issues = append(issues, validateUtilityForgeKIngress(req)...)
	if strings.TrimSpace(req.IdempotencyKey) == "" {
		issues = append(issues, errField(domain.ErrMissingRequiredField, "idempotencyKey", "utility evidence writes require idempotencyKey"))
	}
	if strings.TrimSpace(req.Scope.WorkspaceID) == "" || strings.TrimSpace(req.Scope.LaneID) == "" ||
		len(normalizedUtilityPaths(req.Scope.SelectedPaths)) == 0 {
		issues = append(issues, errField(domain.ErrInvalidScope, "scope", "workspaceId, laneId, and selectedPaths identity are required"))
	}
	if req.Source == domain.SourceAdapter || req.Source == domain.SourceFutureIRIS {
		issues = append(issues, errField(domain.ErrUnauthorized, "source", "external model/proposal sources cannot commit utility evidence"))
	}
	return issues
}

func validateUtilityMetadata(metadata map[string]any) []domain.SyscallError {
	allowed := map[string]bool{
		"reason": true, "sourceSurface": true, "operatorComment": true,
		"tags": true, "uiContext": true,
	}
	for key := range metadata {
		if !allowed[strings.TrimSpace(key)] {
			return []domain.SyscallError{errField(domain.ErrInvalidPayload, "payload.metadata."+strings.TrimSpace(key), "metadata key is not in the descriptive allowlist")}
		}
	}
	if forbidden := findForbiddenUtilityMetadataClaim(metadata); forbidden != "" {
		return []domain.SyscallError{errField(domain.ErrUnauthorized, "payload.metadata."+forbidden, "metadata cannot claim authority, proof, decision, or projection state")}
	}
	raw, err := json.Marshal(metadata)
	if err != nil {
		return []domain.SyscallError{errField(domain.ErrInvalidPayload, "payload.metadata", "metadata is not deterministic JSON")}
	}
	if len(raw) > maxUtilityMetadataBytes {
		return []domain.SyscallError{errField(domain.ErrInvalidPayload, "payload.metadata", "metadata exceeds 16384 bytes")}
	}
	return nil
}

func findForbiddenUtilityMetadataClaim(value any) string {
	forbidden := map[string]bool{
		"eventid": true, "decision": true, "projection": true, "committedby": true,
		"kernelauthorityowner": true, "forgekingressauthority": true,
		"auth": true, "authorization": true, "authorizationproof": true, "authproof": true,
		"proof": true, "receipt": true, "commitreceipt": true, "seal": true,
		"preparedplan": true, "preparedplanseal": true, "audit": true, "auditid": true,
		"auditoutbox": true, "outbox": true, "journal": true, "journalhead": true,
		"provenance": true, "provenanceid": true, "syscall": true, "syscallid": true,
		"kernelauthority": true, "forgekauthority": true,
	}
	var walk func(any, string) string
	walk = func(current any, path string) string {
		switch typed := current.(type) {
		case map[string]any:
			for key, child := range typed {
				normalized := strings.ToLower(strings.NewReplacer("_", "", "-", "", ".", "").Replace(strings.TrimSpace(key)))
				childPath := key
				if path != "" {
					childPath = path + "." + key
				}
				if forbidden[normalized] {
					return childPath
				}
				if found := walk(child, childPath); found != "" {
					return found
				}
			}
		case []any:
			for index, child := range typed {
				if found := walk(child, fmt.Sprintf("%s[%d]", path, index)); found != "" {
					return found
				}
			}
		}
		return ""
	}
	return walk(value, "")
}

func validateUtilityForgeKIngress(req domain.SyscallRequest) []domain.SyscallError {
	owner, _ := req.Metadata["kernelAuthorityOwner"].(string)
	ingress, _ := req.Metadata["forgeKIngressAuthority"].(bool)
	if !ingress || owner != forgekernel.AuthorityOwnerForgeK {
		return []domain.SyscallError{errField(domain.ErrUnauthorized, "metadata.kernelAuthorityOwner", "utility evidence mutation requires production FORGE-K ingress")}
	}
	return nil
}

func applyRecordRetrievalUsefulness(ctx context.Context, store SemanticStore, req domain.SyscallRequest) ([]string, map[string]any, []string, []domain.SyscallError) {
	if issues := validateUtilityForgeKIngress(req); len(issues) > 0 {
		return nil, nil, nil, issues
	}
	var payload recordRetrievalUsefulnessPayload
	if err := decodeStrictUtilityPayload(req.Payload, &payload); err != nil {
		return nil, nil, nil, []domain.SyscallError{errField(domain.ErrInvalidPayload, "payload", err.Error())}
	}
	read, readOK := store.(utilityEvidenceReadStore)
	write, writeOK := store.(utilityEvidenceWriteStore)
	if !readOK || !writeOK {
		return nil, nil, nil, []domain.SyscallError{errField(domain.ErrPersistenceUnavailable, "store", "utility evidence store unavailable")}
	}
	target, found, err := read.FindRetrievalUsefulnessTarget(ctx, payload.ResultID)
	if err != nil {
		return nil, nil, nil, []domain.SyscallError{errField(domain.ErrPersistenceUnavailable, "payload.resultId", err.Error())}
	}
	if !found {
		return nil, nil, nil, []domain.SyscallError{errField(domain.ErrNotFound, "payload.resultId", "retrieval result evidence not found")}
	}
	eventID := "retrieval-usefulness:" + strings.TrimSpace(req.ID)
	event := RetrievalUsefulnessEvent{
		ID: eventID, CreatedAt: req.RequestedAt, ResultID: target.ResultID, RunID: target.RunID,
		TargetEvidenceID: target.EvidenceID, Scope: req.Scope,
		Label: NormalizeRetrievalUsefulnessLabel(payload.Label), Note: strings.TrimSpace(payload.Note),
		JobID: payload.JobID, PacketID: payload.PacketID,
		CorrelationID: req.CorrelationID, TraceID: req.TraceID, SyscallID: req.ID,
		Provenance: req.Provenance, ProposedBy: string(req.Source), CommittedBy: "forge_k.kernel",
		Metadata: nonNilMap(payload.Metadata),
	}
	if err := write.CreateRetrievalUsefulnessEvent(event); err != nil {
		return nil, nil, nil, []domain.SyscallError{errField(domain.ErrPersistenceUnavailable, "utilityEvidence", err.Error())}
	}
	return []string{eventID}, map[string]any{
		"utilityEvidenceType": "retrieval_usefulness", "eventId": eventID,
		"retrievalResultId": target.ResultID, "retrievalRunId": target.RunID,
		"targetEvidenceId": target.EvidenceID, "projection": "noncanonical_rebuildable",
	}, nil, nil
}

func applyRecordRestoreOutcomeFeedback(ctx context.Context, store SemanticStore, req domain.SyscallRequest) ([]string, map[string]any, []string, []domain.SyscallError) {
	if issues := validateUtilityForgeKIngress(req); len(issues) > 0 {
		return nil, nil, nil, issues
	}
	var payload recordRestoreOutcomeFeedbackPayload
	if err := decodeStrictUtilityPayload(req.Payload, &payload); err != nil {
		return nil, nil, nil, []domain.SyscallError{errField(domain.ErrInvalidPayload, "payload", err.Error())}
	}
	read, readOK := store.(utilityEvidenceReadStore)
	write, writeOK := store.(utilityEvidenceWriteStore)
	if !readOK || !writeOK {
		return nil, nil, nil, []domain.SyscallError{errField(domain.ErrPersistenceUnavailable, "store", "utility evidence store unavailable")}
	}
	target, found, err := read.FindRestoreOutcomeFeedbackTarget(ctx, payload.RestoreOutcomeID)
	if err != nil {
		return nil, nil, nil, []domain.SyscallError{errField(domain.ErrPersistenceUnavailable, "payload.restoreOutcomeId", err.Error())}
	}
	if !found {
		return nil, nil, nil, []domain.SyscallError{errField(domain.ErrNotFound, "payload.restoreOutcomeId", "restore outcome evidence not found")}
	}
	eventID := "restore-outcome-feedback:" + strings.TrimSpace(req.ID)
	event := RestoreOutcomeFeedbackEvent{
		ID: eventID, CreatedAt: req.RequestedAt, RestoreOutcomeID: target.RestoreOutcomeID,
		Scope: req.Scope, OriginalOutcome: target.OriginalOutcome, Outcome: payload.Outcome,
		OutcomeConfidence: payload.OutcomeConfidence, OperatorFeedback: strings.TrimSpace(payload.OperatorFeedback),
		CorrectionSummary: strings.TrimSpace(payload.CorrectionSummary),
		CorrelationID:     req.CorrelationID, TraceID: req.TraceID, SyscallID: req.ID,
		Provenance: req.Provenance, ProposedBy: string(req.Source), CommittedBy: "forge_k.kernel",
		Metadata: nonNilMap(payload.Metadata),
	}
	if err := write.CreateRestoreOutcomeFeedbackEvent(event); err != nil {
		return nil, nil, nil, []domain.SyscallError{errField(domain.ErrPersistenceUnavailable, "utilityEvidence", err.Error())}
	}
	return []string{eventID}, map[string]any{
		"utilityEvidenceType": "restore_outcome_feedback", "eventId": eventID,
		"restoreOutcomeId": target.RestoreOutcomeID, "originalEvidencePreserved": true,
		"projection": "noncanonical_rebuildable",
	}, nil, nil
}
