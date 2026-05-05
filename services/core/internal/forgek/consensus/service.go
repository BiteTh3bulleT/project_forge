package consensus

import (
	"sort"
	"sync"
	"time"
)

type Service struct {
	mu       sync.RWMutex
	requests map[string]ConsensusRequest
	policies map[string]ConsensusPolicy
	reports  map[string]ConsensusReport
	ledger   *ClaimLedger
}

func NewService() *Service {
	return &Service{
		requests: make(map[string]ConsensusRequest),
		policies: make(map[string]ConsensusPolicy),
		reports:  make(map[string]ConsensusReport),
		ledger:   NewClaimLedger(),
	}
}

func (s *Service) OpenRequest(request ConsensusRequest, policy ConsensusPolicy) (ConsensusRequest, error) {
	request = NormalizeRequest(request)
	if policy.PolicyID == "" {
		policy = DefaultPolicy(request.WorkspaceID, CriticalityLow, request.OpenedAt)
	}
	policy = NormalizePolicy(policy)
	if request.PolicyID == "" {
		request.PolicyID = policy.PolicyID
	}
	if err := ValidateRequest(request); err != nil {
		return ConsensusRequest{}, err
	}
	if policy.WorkspaceID != request.WorkspaceID {
		return ConsensusRequest{}, ErrInvalidPolicy
	}
	if err := ValidatePolicy(policy); err != nil {
		return ConsensusRequest{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests[request.RequestID] = request.Clone()
	s.policies[policy.PolicyID] = policy.Clone()
	return request.Clone(), nil
}

func (s *Service) SubmitClaim(input ClaimInput) (Claim, error) {
	claim, err := NewClaim(input)
	if err != nil {
		return Claim{}, err
	}
	if _, ok := s.GetRequest(claim.RequestID); !ok {
		return Claim{}, ErrObjectNotFound
	}
	if err := s.ledger.SubmitClaim(claim); err != nil {
		return Claim{}, err
	}
	return claim.Clone(), nil
}

func (s *Service) SubmitEvidence(ref EvidenceRef) (EvidenceRef, error) {
	created, err := NewEvidenceRef(ref)
	if err != nil {
		return EvidenceRef{}, err
	}
	if err := s.ledger.SubmitEvidenceRef(created); err != nil {
		return EvidenceRef{}, err
	}
	return created.Clone(), nil
}

func (s *Service) Evaluate(requestID, reportID string, createdAt time.Time) (ConsensusReport, error) {
	request, ok := s.GetRequest(requestID)
	if !ok {
		return ConsensusReport{}, ErrObjectNotFound
	}
	policy, ok := s.GetPolicy(request.PolicyID)
	if !ok {
		policy = DefaultPolicy(request.WorkspaceID, CriticalityLow, createdAt)
	}
	claims := s.ledger.ListClaims(ClaimListFilter{RequestID: requestID})
	evidence := s.ledger.EvidenceMap()
	decisions := EvaluateClaims(requestID, claims, evidence, policy, createdAt)
	report := BuildReport(reportID, request, policy, decisions, createdAt)
	for _, id := range report.AcceptedClaimIDs {
		_ = s.ledger.UpdateClaimStatus(id, StatusAccepted)
	}
	for _, id := range report.UncertainClaimIDs {
		_ = s.ledger.UpdateClaimStatus(id, StatusUncertain)
	}
	for _, id := range report.RejectedClaimIDs {
		_ = s.ledger.UpdateClaimStatus(id, StatusRejected)
	}
	for _, id := range report.ConflictedClaimIDs {
		_ = s.ledger.UpdateClaimStatus(id, StatusConflicted)
	}
	s.StoreReport(report)
	return report.Clone(), nil
}

func EvaluateClaims(requestID string, claims []Claim, evidence map[string]EvidenceRef, policy ConsensusPolicy, createdAt time.Time) []ConsensusDecision {
	policy = NormalizePolicy(policy)
	conflicts := DetectConflicts(claims)
	conflictedIDs := map[string]bool{}
	for _, conflict := range conflicts {
		for _, id := range conflict.ClaimIDs {
			conflictedIDs[id] = true
		}
	}
	byKey := make(map[string][]Claim)
	for _, claim := range claims {
		byKey[claim.ClaimKey] = append(byKey[claim.ClaimKey], claim)
	}
	keys := make([]string, 0, len(byKey))
	for key := range byKey {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	decisions := make([]ConsensusDecision, 0, len(keys))
	for _, key := range keys {
		group := byKey[key]
		sort.Slice(group, func(i, j int) bool { return group[i].ClaimID < group[j].ClaimID })
		claimType := group[0].ClaimType
		opposing := opposingClaims(group[0], claims)
		stats := ScoreClaims(group, opposing, evidence, nil)
		quorum := QuorumMet(policy, stats)
		evidencePassed := EvidencePolicyPassed(policy, claimType, stats, group, evidence)
		riskPassed := RiskPolicyPassed(policy, group)
		decision := ConsensusDecision{
			DecisionID:           StableHash(map[string]any{"request_id": requestID, "claim_key": key}),
			RequestID:            requestID,
			ClaimKey:             key,
			SupportWeight:        stats.SupportWeight,
			OpposingWeight:       stats.OpposingWeight,
			TotalEligibleWeight:  stats.TotalEligibleWeight,
			SupportRatio:         stats.SupportRatio,
			ConflictRatio:        stats.ConflictRatio,
			QuorumMet:            quorum,
			EvidencePolicyPassed: evidencePassed,
			RiskPolicyPassed:     riskPassed,
			CreatedAt:            createdAt,
		}
		groupIDs := claimIDs(group)
		switch {
		case anyConflicted(groupIDs, conflictedIDs) || stats.ConflictRatio > policy.MaxConflictRatio:
			decision.Status = StatusConflicted
			decision.ConflictedClaimIDs = groupIDs
			decision.DecisionReason = "conflict policy blocked acceptance"
		case claimType == ClaimTypeUncertainty:
			decision.Status = StatusUncertain
			decision.UncertainClaimIDs = groupIDs
			decision.DecisionReason = "uncertainty claim preserved as uncertainty"
		case !evidencePassed:
			decision.Status = StatusNeedsMoreEvidence
			decision.RejectedClaimIDs = groupIDs
			decision.DecisionReason = "evidence policy not satisfied"
		case !quorum:
			decision.Status = StatusNeedsMoreEvidence
			decision.UncertainClaimIDs = groupIDs
			decision.DecisionReason = "quorum not met"
		case !riskPassed:
			decision.Status = StatusDeferred
			decision.UncertainClaimIDs = groupIDs
			decision.DecisionReason = "risk policy requires confirmation"
		case stats.SupportRatio >= policy.MinSupportRatio && stats.SupportWeight > 0:
			decision.Status = StatusAccepted
			decision.AcceptedClaimIDs = groupIDs
			decision.DecisionReason = "support, quorum, evidence, and risk policies satisfied"
		default:
			decision.Status = StatusRejected
			decision.RejectedClaimIDs = groupIDs
			decision.DecisionReason = "support ratio below policy threshold"
		}
		decisions = append(decisions, decision)
	}
	return decisions
}

func (s *Service) StoreReport(report ConsensusReport) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reports[report.ReportID] = report.Clone()
}

func (s *Service) GetReport(reportID string) (ConsensusReport, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	report, ok := s.reports[reportID]
	return report.Clone(), ok
}

func (s *Service) ListReports(filter ReportListFilter) []ConsensusReport {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]ConsensusReport, 0)
	for _, report := range s.reports {
		if filter.WorkspaceID != "" && report.WorkspaceID != filter.WorkspaceID {
			continue
		}
		if filter.RequestID != "" && report.RequestID != filter.RequestID {
			continue
		}
		if filter.CaseID != "" && report.CaseID != filter.CaseID {
			continue
		}
		out = append(out, report.Clone())
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ReportID < out[j].ReportID })
	return out
}

func (s *Service) GetRequest(requestID string) (ConsensusRequest, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	request, ok := s.requests[requestID]
	return request.Clone(), ok
}

func (s *Service) GetPolicy(policyID string) (ConsensusPolicy, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	policy, ok := s.policies[policyID]
	return policy.Clone(), ok
}

func (s *Service) ListClaims(filter ClaimListFilter) []Claim {
	return s.ledger.ListClaims(filter)
}

func (s *Service) ListEvidence() []EvidenceRef {
	return s.ledger.ListEvidence()
}

func NormalizeRequest(request ConsensusRequest) ConsensusRequest {
	request.RequestID = trim(request.RequestID)
	request.WorkspaceID = trim(request.WorkspaceID)
	request.CaseID = trim(request.CaseID)
	request.PolicyID = trim(request.PolicyID)
	request.OpenedBy = trim(request.OpenedBy)
	request.Metadata = CloneMap(request.Metadata)
	return request
}

func ValidateRequest(request ConsensusRequest) error {
	request = NormalizeRequest(request)
	if request.RequestID == "" || request.WorkspaceID == "" || request.OpenedBy == "" || containsSecretMetadata(request.Metadata) {
		return ErrInvalidRequest
	}
	return nil
}

func opposingClaims(claim Claim, all []Claim) []Claim {
	out := make([]Claim, 0)
	for _, candidate := range all {
		if ClaimsConflict(claim, candidate) {
			out = append(out, candidate)
		}
	}
	return out
}

func claimIDs(claims []Claim) []string {
	ids := make([]string, 0, len(claims))
	for _, claim := range claims {
		ids = append(ids, claim.ClaimID)
	}
	return NormalizeRefs(ids)
}

func anyConflicted(ids []string, conflicted map[string]bool) bool {
	for _, id := range ids {
		if conflicted[id] {
			return true
		}
	}
	return false
}
