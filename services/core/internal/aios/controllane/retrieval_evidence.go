package controllane

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"path/filepath"
	"strconv"
	"strings"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/forgekernel"
)

type RetrievalEvidence struct {
	EvidenceID string                    `json:"evidenceId"`
	CreatedAt  int64                     `json:"createdAt"`
	Query      string                    `json:"query"`
	Mode       string                    `json:"mode"`
	DossierID  *int64                    `json:"dossierId,omitempty"`
	PacketID   *int64                    `json:"packetId,omitempty"`
	JobID      *string                   `json:"jobId,omitempty"`
	Weighting  map[string]any            `json:"weighting"`
	Notes      string                    `json:"notes,omitempty"`
	Results    []RetrievalResultEvidence `json:"results"`
}

type RetrievalResultEvidence struct {
	EvidenceID      string         `json:"evidenceId"`
	ChunkID         *int64         `json:"chunkId,omitempty"`
	FileID          *int64         `json:"fileId,omitempty"`
	AbsPath         string         `json:"absPath"`
	RelPath         string         `json:"relPath"`
	RankIndex       int            `json:"rankIndex"`
	KeywordScore    float64        `json:"keywordScore"`
	SemanticScore   float64        `json:"semanticScore"`
	HybridScore     float64        `json:"hybridScore"`
	Snippet         string         `json:"snippet"`
	Selected        bool           `json:"selectedForPacket"`
	SelectionReason map[string]any `json:"selectionReason"`
}

type RetrievalEvidenceCommit struct {
	RunID     int64
	ResultIDs []int64
}

func decodeRetrievalEvidence(payload map[string]any) (RetrievalEvidence, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return RetrievalEvidence{}, fmt.Errorf("encode retrieval evidence: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var evidence RetrievalEvidence
	if err := decoder.Decode(&evidence); err != nil {
		return RetrievalEvidence{}, fmt.Errorf("decode retrieval evidence: %w", err)
	}
	if decoder.More() {
		return RetrievalEvidence{}, fmt.Errorf("decode retrieval evidence: trailing value")
	}
	return evidence, nil
}

func validateRecordRetrievalEvidence(req domain.SyscallRequest) []domain.SyscallError {
	evidence, err := decodeRetrievalEvidence(req.Payload)
	if err != nil {
		return []domain.SyscallError{errField(domain.ErrInvalidPayload, "payload", err.Error())}
	}
	issues := make([]domain.SyscallError, 0)
	if len(req.Scope.SelectedPaths) == 0 {
		issues = append(issues, errField(domain.ErrMissingRequiredField, "scope.selectedPaths", "at least one resolved source path is required"))
	} else {
		seenPaths := make(map[string]struct{}, len(req.Scope.SelectedPaths))
		previous := ""
		for index, path := range req.Scope.SelectedPaths {
			trimmed := strings.TrimSpace(path)
			if trimmed == "" {
				issues = append(issues, errField(domain.ErrInvalidScope, "scope.selectedPaths["+strconv.Itoa(index)+"]", "source path must be non-empty"))
				continue
			}
			if !filepath.IsAbs(trimmed) || filepath.Clean(trimmed) != trimmed {
				issues = append(issues, errField(domain.ErrInvalidScope, "scope.selectedPaths["+strconv.Itoa(index)+"]", "source path must be an absolute canonical path"))
			}
			if _, duplicate := seenPaths[trimmed]; duplicate {
				issues = append(issues, errField(domain.ErrInvalidScope, "scope.selectedPaths["+strconv.Itoa(index)+"]", "source paths must be unique"))
			}
			if previous != "" && trimmed < previous {
				issues = append(issues, errField(domain.ErrInvalidScope, "scope.selectedPaths", "source paths must use deterministic lexical ordering"))
			}
			seenPaths[trimmed] = struct{}{}
			previous = trimmed
		}
	}
	if evidence.EvidenceID != req.ID+":retrieval_run" {
		issues = append(issues, errField(domain.ErrInvalidPayload, "payload.evidenceId", "must be deterministically derived from request id"))
	}
	if evidence.CreatedAt != req.RequestedAt {
		issues = append(issues, errField(domain.ErrInvalidPayload, "payload.createdAt", "must equal request requestedAt"))
	}
	if query := strings.TrimSpace(evidence.Query); query == "" || len(query) > 32768 {
		issues = append(issues, errField(domain.ErrInvalidPayload, "payload.query", "query must be non-empty and at most 32768 bytes"))
	}
	switch evidence.Mode {
	case "keyword", "semantic", "hybrid":
	default:
		issues = append(issues, errField(domain.ErrInvalidPayload, "payload.mode", "mode must be keyword, semantic, or hybrid"))
	}
	if evidence.Weighting == nil {
		issues = append(issues, errField(domain.ErrMissingRequiredField, "payload.weighting", "weighting object is required"))
	}
	if len(evidence.Notes) > 32768 {
		issues = append(issues, errField(domain.ErrInvalidPayload, "payload.notes", "notes must be at most 32768 bytes"))
	}
	for field, value := range map[string]*int64{"payload.dossierId": evidence.DossierID, "payload.packetId": evidence.PacketID} {
		if value != nil && *value <= 0 {
			issues = append(issues, errField(domain.ErrInvalidPayload, field, "must be a positive integer"))
		}
	}
	if evidence.JobID != nil && strings.TrimSpace(*evidence.JobID) == "" {
		issues = append(issues, errField(domain.ErrInvalidPayload, "payload.jobId", "must be non-empty when supplied"))
	}
	if len(evidence.Results) > 100 {
		issues = append(issues, errField(domain.ErrInvalidPayload, "payload.results", "at most 100 ordered results are allowed"))
	}
	for index, result := range evidence.Results {
		prefix := "payload.results[" + strconv.Itoa(index) + "]"
		if result.RankIndex != index {
			issues = append(issues, errField(domain.ErrInvalidPayload, prefix+".rankIndex", "ranks must be contiguous and ordered from zero"))
		}
		if result.EvidenceID != req.ID+":retrieval_result:"+strconv.Itoa(index) {
			issues = append(issues, errField(domain.ErrInvalidPayload, prefix+".evidenceId", "must be deterministically derived from request id and rank"))
		}
		if result.ChunkID != nil && *result.ChunkID <= 0 {
			issues = append(issues, errField(domain.ErrInvalidPayload, prefix+".chunkId", "must be positive when supplied"))
		}
		if result.FileID != nil && *result.FileID <= 0 {
			issues = append(issues, errField(domain.ErrInvalidPayload, prefix+".fileId", "must be positive when supplied"))
		}
		if strings.TrimSpace(result.AbsPath) == "" && strings.TrimSpace(result.RelPath) == "" {
			issues = append(issues, errField(domain.ErrInvalidPayload, prefix+".relPath", "absPath or relPath is required"))
		}
		if !retrievalPathWithinScope(result.AbsPath, req.Scope.SelectedPaths) {
			issues = append(issues, errField(domain.ErrInvalidScope, prefix+".absPath", "absolute result path must be contained by a selected source root"))
		}
		if trimmed := strings.TrimSpace(result.AbsPath); trimmed != "" && filepath.Clean(trimmed) != trimmed {
			issues = append(issues, errField(domain.ErrInvalidScope, prefix+".absPath", "result path must be canonical"))
		}
		if len(result.AbsPath) > 32768 || len(result.RelPath) > 32768 || len(result.Snippet) > 65536 {
			issues = append(issues, errField(domain.ErrInvalidPayload, prefix, "path or snippet exceeds evidence bounds"))
		}
		for field, score := range map[string]float64{
			"keywordScore": result.KeywordScore, "semanticScore": result.SemanticScore, "hybridScore": result.HybridScore,
		} {
			if !finiteBounded(score, -10, 10) {
				issues = append(issues, errField(domain.ErrInvalidPayload, prefix+"."+field, "score must be finite and within [-10,10]"))
			}
		}
		if result.SelectionReason == nil {
			issues = append(issues, errField(domain.ErrMissingRequiredField, prefix+".selectionReason", "selection reason object is required"))
		} else if raw, marshalErr := json.Marshal(result.SelectionReason); marshalErr != nil || len(raw) > 65536 {
			issues = append(issues, errField(domain.ErrInvalidPayload, prefix+".selectionReason", "selection reason must be valid bounded JSON"))
		}
	}
	return issues
}

func retrievalPathWithinScope(rawPath string, roots []string) bool {
	path := filepath.Clean(strings.TrimSpace(rawPath))
	if path == "" || path == "." || !filepath.IsAbs(path) {
		return false
	}
	for _, rawRoot := range roots {
		root := filepath.Clean(strings.TrimSpace(rawRoot))
		if root == "" || root == "." || !filepath.IsAbs(root) {
			continue
		}
		relative, err := filepath.Rel(root, path)
		if err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func finiteBounded(value, minimum, maximum float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= minimum && value <= maximum
}

func applyRecordRetrievalEvidence(ctx context.Context, store SemanticStore, req domain.SyscallRequest) ([]string, map[string]any, []string, []domain.SyscallError) {
	owner, _ := req.Metadata["kernelAuthorityOwner"].(string)
	ingress, _ := req.Metadata["forgeKIngressAuthority"].(bool)
	if !ingress || owner != forgekernel.AuthorityOwnerForgeK {
		return nil, nil, nil, []domain.SyscallError{{Code: domain.ErrUnauthorized, Field: "metadata.kernelAuthorityOwner", Message: "retrieval evidence admission requires production FORGE-K ingress"}}
	}
	evidence, err := decodeRetrievalEvidence(req.Payload)
	if err != nil {
		return nil, nil, nil, []domain.SyscallError{{Code: domain.ErrInvalidPayload, Field: "payload", Message: err.Error()}}
	}
	commit, err := store.RecordRetrievalEvidence(ctx, req, evidence)
	if err != nil {
		return nil, nil, nil, []domain.SyscallError{{Code: domain.ErrConflict, Field: "retrievalEvidence", Message: err.Error()}}
	}
	ids := make([]string, 0, 1+len(evidence.Results))
	ids = append(ids, evidence.EvidenceID)
	for _, result := range evidence.Results {
		ids = append(ids, result.EvidenceID)
	}
	return ids, map[string]any{
		"retrievalEvidence": map[string]any{
			"evidenceId": evidence.EvidenceID, "runId": commit.RunID,
			"resultCount": len(commit.ResultIDs), "atomic": true,
			"selectionReasonsCommitted": len(commit.ResultIDs),
			"memoryObservationsCreated": 0, "modelCanonicalAuthority": false,
		},
	}, nil, nil
}

func (s *InMemorySemanticStore) RecordRetrievalEvidence(_ context.Context, req domain.SyscallRequest, evidence RetrievalEvidence) (RetrievalEvidenceCommit, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.state.retrievalEvidence[evidence.EvidenceID]; exists {
		return RetrievalEvidenceCommit{}, fmt.Errorf("retrieval evidence %q already exists", evidence.EvidenceID)
	}
	s.state.retrievalEvidence[evidence.EvidenceID] = cloneIntegrityValue(evidence)
	resultIDs := make([]int64, len(evidence.Results))
	firstResultID := int64(1)
	for resultID := range s.state.retrievalTargets {
		if resultID >= firstResultID {
			firstResultID = resultID + 1
		}
	}
	runID := int64(len(s.state.retrievalEvidence))
	provenanceJSON, _ := json.Marshal(req.Provenance)
	for index := range resultIDs {
		resultIDs[index] = firstResultID + int64(index)
		s.state.retrievalTargets[resultIDs[index]] = RetrievalUsefulnessTarget{
			ResultID: resultIDs[index], RunID: runID, EvidenceID: evidence.Results[index].EvidenceID,
			Scope: req.Scope, JobID: evidence.JobID, PacketID: evidence.PacketID,
			SourceSyscall: req.ID, SourceProvID: provenanceID(req.Scope, req.Provenance),
			SourceProvJSON: string(provenanceJSON),
		}
	}
	return RetrievalEvidenceCommit{RunID: runID, ResultIDs: resultIDs}, nil
}

func (s *TransactionalSemanticStore) RecordRetrievalEvidence(_ context.Context, req domain.SyscallRequest, evidence RetrievalEvidence) (RetrievalEvidenceCommit, error) {
	if _, exists := s.state.retrievalEvidence[evidence.EvidenceID]; exists {
		return RetrievalEvidenceCommit{}, fmt.Errorf("retrieval evidence %q already exists", evidence.EvidenceID)
	}
	s.state.retrievalEvidence[evidence.EvidenceID] = cloneIntegrityValue(evidence)
	resultIDs := make([]int64, len(evidence.Results))
	firstResultID := int64(1)
	for resultID := range s.state.retrievalTargets {
		if resultID >= firstResultID {
			firstResultID = resultID + 1
		}
	}
	runID := int64(len(s.state.retrievalEvidence))
	provenanceJSON, _ := json.Marshal(req.Provenance)
	for index := range resultIDs {
		resultIDs[index] = firstResultID + int64(index)
		s.state.retrievalTargets[resultIDs[index]] = RetrievalUsefulnessTarget{
			ResultID: resultIDs[index], RunID: runID, EvidenceID: evidence.Results[index].EvidenceID,
			Scope: req.Scope, JobID: evidence.JobID, PacketID: evidence.PacketID,
			SourceSyscall: req.ID, SourceProvID: provenanceID(req.Scope, req.Provenance),
			SourceProvJSON: string(provenanceJSON),
		}
	}
	return RetrievalEvidenceCommit{RunID: runID, ResultIDs: resultIDs}, nil
}
