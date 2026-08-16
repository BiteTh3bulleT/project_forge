package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"forge/projectforge/services/core/internal/aios/controllane"
	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/forgekernel"
	"forge/projectforge/services/core/internal/forgekernel/semanticdiff"
)

const (
	governedEvidenceBodyLimit = int64(1 << 20)
	retrievalAdmissionPolicy  = "forge-k-court-v1:governed-retrieval-evidence"
)

type governedEvidenceScopeRequest struct {
	WorkspaceID   string   `json:"workspaceId"`
	LaneID        string   `json:"laneId"`
	SelectedPaths []string `json:"selectedPaths"`
}

type courtAdmissionIntent struct {
	governedEvidenceScopeRequest
	RetrievalResultID int64 `json:"retrievalResultId"`
}

type evidenceRevisionIntent struct {
	governedEvidenceScopeRequest
	ExhibitID string `json:"exhibitId"`
}

type semanticDiffIntent struct {
	governedEvidenceScopeRequest
	LeftEvidenceID  string `json:"leftEvidenceId"`
	RightEvidenceID string `json:"rightEvidenceId"`
}

type governedRetrievalSource struct {
	EvidenceID string
	AbsPath    string
	RelPath    string
	Rank       int
	Snippet    string
	Scope      domain.ForgeScope
	SyscallID  string
	ProvID     string
}

func decodeGovernedEvidenceIntent(w http.ResponseWriter, r *http.Request, dst any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, governedEvidenceBodyLimit))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("request body must contain one JSON object")
		}
		return err
	}
	return nil
}

func governedIntentScope(in governedEvidenceScopeRequest) (domain.ForgeScope, error) {
	scope := domain.ForgeScope{
		WorkspaceID: strings.TrimSpace(in.WorkspaceID),
		LaneID:      strings.TrimSpace(in.LaneID),
	}
	if scope.WorkspaceID == "" || scope.LaneID == "" {
		return domain.ForgeScope{}, errors.New("workspaceId and laneId are required")
	}
	seen := map[string]struct{}{}
	for _, path := range in.SelectedPaths {
		path = strings.TrimSpace(path)
		if path == "" {
			return domain.ForgeScope{}, errors.New("selectedPaths must contain non-empty paths")
		}
		if _, exists := seen[path]; exists {
			continue
		}
		seen[path] = struct{}{}
		scope.SelectedPaths = append(scope.SelectedPaths, path)
	}
	if len(scope.SelectedPaths) == 0 {
		return domain.ForgeScope{}, errors.New("selectedPaths are required")
	}
	sort.Strings(scope.SelectedPaths)
	return scope, nil
}

func exactGovernedAPIScope(left, right domain.ForgeScope) bool {
	if strings.TrimSpace(left.WorkspaceID) != strings.TrimSpace(right.WorkspaceID) ||
		strings.TrimSpace(left.LaneID) != strings.TrimSpace(right.LaneID) {
		return false
	}
	l := append([]string(nil), left.SelectedPaths...)
	r := append([]string(nil), right.SelectedPaths...)
	sort.Strings(l)
	sort.Strings(r)
	if len(l) != len(r) {
		return false
	}
	for i := range l {
		if strings.TrimSpace(l[i]) != strings.TrimSpace(r[i]) {
			return false
		}
	}
	return len(l) > 0
}

func (s *Server) requireGovernedKernel(w http.ResponseWriter) bool {
	if s.kernelAuthority.Processor == nil || !s.kernelAuthorizationReady || s.kernelAuthority.Mode != forgekernel.ModeForgeK {
		writeAPIError(w, http.StatusServiceUnavailable, "kernel_authority_unavailable", "production FORGE-K authority is unavailable", nil)
		return false
	}
	return true
}

func requireIdempotencyHeader(w http.ResponseWriter, r *http.Request) (string, bool) {
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if key == "" {
		writeAPIError(w, http.StatusBadRequest, "request_failed", "Idempotency-Key header is required", nil)
		return "", false
	}
	return key, true
}

func (s *Server) loadGovernedRetrievalSource(r *http.Request, resultID int64, scope domain.ForgeScope) (governedRetrievalSource, error) {
	semantic := controllane.NewSQLiteSemanticStore(s.st.DB)
	target, found, err := semantic.FindRetrievalUsefulnessTarget(r.Context(), resultID)
	if err != nil {
		return governedRetrievalSource{}, err
	}
	if !found || !exactGovernedAPIScope(scope, target.Scope) || strings.TrimSpace(target.SourceSyscall) == "" || strings.TrimSpace(target.SourceProvID) == "" {
		return governedRetrievalSource{}, errors.New("governed retrieval evidence not found in exact scope")
	}
	var source governedRetrievalSource
	var committedBy, authorizationFingerprint string
	err = s.st.DB.QueryRowContext(r.Context(), `
SELECT rr.evidence_id,rr.abs_path,rr.rel_path,rr.rank_index,rr.snippet,
       r.syscall_id,r.provenance_id,r.committed_by,r.authorization_fingerprint
FROM retrieval_results rr
JOIN retrieval_runs r ON r.id=rr.retrieval_run_id
WHERE rr.id=?`, resultID).Scan(
		&source.EvidenceID, &source.AbsPath, &source.RelPath, &source.Rank, &source.Snippet,
		&source.SyscallID, &source.ProvID, &committedBy, &authorizationFingerprint,
	)
	if err != nil {
		return governedRetrievalSource{}, err
	}
	if source.EvidenceID != target.EvidenceID || committedBy != forgekernel.AuthorityOwnerForgeK || !strings.HasPrefix(authorizationFingerprint, "sha256:") {
		return governedRetrievalSource{}, errors.New("retrieval evidence lacks production FORGE-K authority")
	}
	source.Scope = scope
	return source, nil
}

func governedRetrievalContentHash(source governedRetrievalSource) (string, error) {
	raw, err := json.Marshal(struct {
		Version    string `json:"version"`
		EvidenceID string `json:"evidenceId"`
		AbsPath    string `json:"absPath"`
		RelPath    string `json:"relPath"`
		Rank       int    `json:"rank"`
		Snippet    string `json:"snippet"`
		SyscallID  string `json:"syscallId"`
		ProvID     string `json:"provenanceId"`
	}{"forge.retrieval_evidence.court_source.v1", source.EvidenceID, source.AbsPath, source.RelPath, source.Rank, source.Snippet, source.SyscallID, source.ProvID})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func (s *Server) governedUserSyscall(r *http.Request, action domain.SemanticActionType, requestID, idempotency string, scope domain.ForgeScope, payload map[string]any) domain.SyscallRequest {
	actor := authenticatedActorName(r)
	traceID := requestID + ":trace"
	return domain.SyscallRequest{
		ID: requestID, Action: action, Actor: domain.ActorIdentity{ID: actor, Kind: "user"}, Source: domain.SourceUser,
		Scope: scope, Payload: payload,
		Provenance:    domain.Provenance{Actor: actor, ActorType: "user", Source: authenticatedActorSource(r), TraceID: traceID},
		CorrelationID: requestID + ":correlation", TraceID: traceID, IdempotencyKey: idempotency,
		RequestedAt: time.Now().UnixMilli(),
	}
}

func writeGovernedSyscallResult(w http.ResponseWriter, result domain.SyscallResult, err error) {
	if err != nil {
		writeAPIRequestError(w, http.StatusBadRequest, err)
		return
	}
	if !result.Success {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"result": result})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"result": result})
}

func (s *Server) handleAdmitRetrievalEvidence(w http.ResponseWriter, r *http.Request) {
	if !s.requireGovernedKernel(w) {
		return
	}
	key, ok := requireIdempotencyHeader(w, r)
	if !ok {
		return
	}
	var body courtAdmissionIntent
	if err := decodeGovernedEvidenceIntent(w, r, &body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "request_failed", "invalid request body", nil)
		return
	}
	scope, err := governedIntentScope(body.governedEvidenceScopeRequest)
	if err != nil || body.RetrievalResultID <= 0 {
		writeAPIError(w, http.StatusBadRequest, "request_failed", "retrievalResultId and exact scope are required", nil)
		return
	}
	caseID := strings.TrimSpace(chi.URLParam(r, "caseId"))
	if caseID == "" {
		writeAPIError(w, http.StatusBadRequest, "request_failed", "caseId is required", nil)
		return
	}
	source, err := s.loadGovernedRetrievalSource(r, body.RetrievalResultID, scope)
	if err != nil {
		writeAPIError(w, http.StatusNotFound, "request_failed", err.Error(), nil)
		return
	}
	contentHash, err := governedRetrievalContentHash(source)
	if err != nil {
		writeAPIRequestError(w, http.StatusInternalServerError, err)
		return
	}
	requestID := utilitySyscallRequestID("court-admit", key, caseID+"\x00"+source.EvidenceID)
	payload := map[string]any{
		"caseId": caseID, "sourceType": "governed_retrieval_result",
		"sourceRefs": []string{"retrieval-evidence:" + source.EvidenceID}, "contentSummary": source.Snippet,
		"rawRef": source.RelPath, "contentHash": contentHash, "policyRefs": []string{retrievalAdmissionPolicy},
	}
	result, err := s.kernelAuthority.Processor.Process(r.Context(), s.governedUserSyscall(r, domain.ActionAdmitEvidence, requestID, key, scope, payload))
	writeGovernedSyscallResult(w, result, err)
}

func (s *Server) handleMaterializeAdmittedEvidence(w http.ResponseWriter, r *http.Request) {
	if !s.requireGovernedKernel(w) {
		return
	}
	key, ok := requireIdempotencyHeader(w, r)
	if !ok {
		return
	}
	var body governedEvidenceScopeRequest
	if err := decodeGovernedEvidenceIntent(w, r, &body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "request_failed", "invalid request body", nil)
		return
	}
	scope, err := governedIntentScope(body)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "request_failed", err.Error(), nil)
		return
	}
	exhibitID := strings.TrimSpace(chi.URLParam(r, "exhibitId"))
	exhibit, found := controllane.NewSQLiteSemanticStore(s.st.DB).FindCourtExhibit(exhibitID, scope)
	if !found || exhibit.CurrentRulingID == "" {
		writeAPIError(w, http.StatusNotFound, "request_failed", "admitted Court exhibit not found in exact scope", nil)
		return
	}
	requestID := utilitySyscallRequestID("memory-materialize", key, exhibitID)
	req := s.governedUserSyscall(r, domain.ActionMaterializeAdmittedEvidence, requestID, key, scope, map[string]any{
		"exhibitId": exhibit.ID, "rulingId": exhibit.CurrentRulingID,
	})
	result, err := s.kernelAuthority.Processor.Process(r.Context(), req)
	writeGovernedSyscallResult(w, result, err)
}

func (s *Server) handleReviseAdmittedEvidence(w http.ResponseWriter, r *http.Request) {
	if !s.requireGovernedKernel(w) {
		return
	}
	key, ok := requireIdempotencyHeader(w, r)
	if !ok {
		return
	}
	var body evidenceRevisionIntent
	if err := decodeGovernedEvidenceIntent(w, r, &body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "request_failed", "invalid request body", nil)
		return
	}
	scope, err := governedIntentScope(body.governedEvidenceScopeRequest)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "request_failed", err.Error(), nil)
		return
	}
	priorID := strings.TrimSpace(chi.URLParam(r, "priorEvidenceId"))
	store := controllane.NewSQLiteSemanticStore(s.st.DB)
	if _, found := store.FindMemoryEvidence(priorID, scope); !found {
		writeAPIError(w, http.StatusNotFound, "request_failed", "prior memory evidence not found in exact scope", nil)
		return
	}
	exhibit, found := store.FindCourtExhibit(strings.TrimSpace(body.ExhibitID), scope)
	if !found || exhibit.CurrentRulingID == "" {
		writeAPIError(w, http.StatusNotFound, "request_failed", "replacement admitted exhibit not found in exact scope", nil)
		return
	}
	requestID := utilitySyscallRequestID("memory-revise", key, priorID+"\x00"+exhibit.ID)
	req := s.governedUserSyscall(r, domain.ActionReviseMemoryEvidence, requestID, key, scope, map[string]any{
		"exhibitId": exhibit.ID, "rulingId": exhibit.CurrentRulingID, "priorEvidenceId": priorID,
	})
	result, err := s.kernelAuthority.Processor.Process(r.Context(), req)
	writeGovernedSyscallResult(w, result, err)
}

func (s *Server) handleComputeSemanticDiff(w http.ResponseWriter, r *http.Request) {
	if !s.requireGovernedKernel(w) {
		return
	}
	key, ok := requireIdempotencyHeader(w, r)
	if !ok {
		return
	}
	var body semanticDiffIntent
	if err := decodeGovernedEvidenceIntent(w, r, &body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "request_failed", "invalid request body", nil)
		return
	}
	scope, err := governedIntentScope(body.governedEvidenceScopeRequest)
	if err != nil || strings.TrimSpace(body.LeftEvidenceID) == "" || strings.TrimSpace(body.RightEvidenceID) == "" {
		writeAPIError(w, http.StatusBadRequest, "request_failed", "leftEvidenceId, rightEvidenceId, and exact scope are required", nil)
		return
	}
	target := strings.TrimSpace(body.LeftEvidenceID) + "\x00" + strings.TrimSpace(body.RightEvidenceID)
	requestID := utilitySyscallRequestID("semantic-diff", key, target)
	req := s.governedUserSyscall(r, domain.ActionComputeSemanticDiff, requestID, key, scope, map[string]any{
		"leftEvidenceId": strings.TrimSpace(body.LeftEvidenceID), "rightEvidenceId": strings.TrimSpace(body.RightEvidenceID),
		"operatorVersion": semanticdiff.OperatorVersion,
	})
	result, err := s.kernelAuthority.Processor.Process(r.Context(), req)
	writeGovernedSyscallResult(w, result, err)
}
