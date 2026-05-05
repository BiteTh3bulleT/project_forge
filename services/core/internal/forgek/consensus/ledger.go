package consensus

import (
	"sort"
	"sync"
)

type ClaimLedger struct {
	mu       sync.RWMutex
	claims   map[string]Claim
	evidence map[string]EvidenceRef
	agentRun map[string]AgentRun
}

func NewClaimLedger() *ClaimLedger {
	return &ClaimLedger{
		claims:   make(map[string]Claim),
		evidence: make(map[string]EvidenceRef),
		agentRun: make(map[string]AgentRun),
	}
}

func (l *ClaimLedger) SubmitClaim(claim Claim) error {
	if err := ValidateClaim(claim); err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.claims[claim.ClaimID] = claim.Clone()
	return nil
}

func (l *ClaimLedger) SubmitEvidenceRef(ref EvidenceRef) error {
	if err := ValidateEvidenceRef(ref); err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.evidence[ref.EvidenceID] = ref.Clone()
	return nil
}

func (l *ClaimLedger) SubmitAgentRun(run AgentRun) error {
	if run.AgentRunID == "" || run.RequestID == "" || run.AgentID == "" || run.Status == "" {
		return ErrInvalidAgentRun
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.agentRun[run.AgentRunID] = run.Clone()
	return nil
}

func (l *ClaimLedger) GetClaim(claimID string) (Claim, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	claim, ok := l.claims[claimID]
	return claim.Clone(), ok
}

func (l *ClaimLedger) ListClaims(filter ClaimListFilter) []Claim {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make([]Claim, 0)
	for _, claim := range l.claims {
		if filter.RequestID != "" && claim.RequestID != filter.RequestID {
			continue
		}
		if filter.Status != "" && claim.Status != filter.Status {
			continue
		}
		out = append(out, claim.Clone())
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ClaimKey == out[j].ClaimKey {
			return out[i].ClaimID < out[j].ClaimID
		}
		return out[i].ClaimKey < out[j].ClaimKey
	})
	return out
}

func (l *ClaimLedger) ListEvidence() []EvidenceRef {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make([]EvidenceRef, 0, len(l.evidence))
	for _, ref := range l.evidence {
		out = append(out, ref.Clone())
	}
	sort.Slice(out, func(i, j int) bool { return out[i].EvidenceID < out[j].EvidenceID })
	return out
}

func (l *ClaimLedger) EvidenceMap() map[string]EvidenceRef {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make(map[string]EvidenceRef, len(l.evidence))
	for key, ref := range l.evidence {
		out[key] = ref.Clone()
	}
	return out
}

func (l *ClaimLedger) GroupByClaimKey(requestID string) map[string][]Claim {
	claims := l.ListClaims(ClaimListFilter{RequestID: requestID})
	out := make(map[string][]Claim)
	for _, claim := range claims {
		out[claim.ClaimKey] = append(out[claim.ClaimKey], claim.Clone())
	}
	for key := range out {
		sort.Slice(out[key], func(i, j int) bool { return out[key][i].ClaimID < out[key][j].ClaimID })
	}
	return out
}

func (l *ClaimLedger) UpdateClaimStatus(claimID string, status ClaimStatus) error {
	if !ValidClaimStatus(status) {
		return ErrInvalidClaim
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	claim, ok := l.claims[claimID]
	if !ok {
		return ErrObjectNotFound
	}
	claim.Status = status
	l.claims[claimID] = claim
	return nil
}
