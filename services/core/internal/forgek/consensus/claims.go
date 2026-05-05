package consensus

func (c Claim) Clone() Claim {
	c.ValueJSON = cloneAny(c.ValueJSON)
	c.EvidenceRefs = CloneStrings(c.EvidenceRefs)
	c.RiskFlags = CloneStrings(c.RiskFlags)
	c.Metadata = CloneMap(c.Metadata)
	return c
}

func (c Claim) IsCanonicalTruth() bool {
	return false
}

func (c Claim) IsMemoryTruth() bool {
	return false
}

func (c Claim) IsActionExecution() bool {
	return false
}

func (e EvidenceRef) Clone() EvidenceRef {
	e.Metadata = CloneMap(e.Metadata)
	return e
}

func (r AgentRun) Clone() AgentRun {
	r.Metadata = CloneMap(r.Metadata)
	if r.CompletedAt != nil {
		completed := *r.CompletedAt
		r.CompletedAt = &completed
	}
	return r
}

func (r ConsensusRequest) Clone() ConsensusRequest {
	r.Metadata = CloneMap(r.Metadata)
	return r
}

func (p ConsensusPolicy) Clone() ConsensusPolicy {
	p.Metadata = CloneMap(p.Metadata)
	return p
}

func (d ConsensusDecision) Clone() ConsensusDecision {
	d.AcceptedClaimIDs = CloneStrings(d.AcceptedClaimIDs)
	d.RejectedClaimIDs = CloneStrings(d.RejectedClaimIDs)
	d.UncertainClaimIDs = CloneStrings(d.UncertainClaimIDs)
	d.ConflictedClaimIDs = CloneStrings(d.ConflictedClaimIDs)
	d.Metadata = CloneMap(d.Metadata)
	return d
}

func (r ConsensusReport) Clone() ConsensusReport {
	r.Decisions = make([]ConsensusDecision, len(r.Decisions))
	for i, decision := range r.Decisions {
		r.Decisions[i] = decision.Clone()
	}
	r.AcceptedClaimIDs = CloneStrings(r.AcceptedClaimIDs)
	r.UncertainClaimIDs = CloneStrings(r.UncertainClaimIDs)
	r.RejectedClaimIDs = CloneStrings(r.RejectedClaimIDs)
	r.ConflictedClaimIDs = CloneStrings(r.ConflictedClaimIDs)
	r.Escalations = CloneStrings(r.Escalations)
	r.JournalRefs = CloneStrings(r.JournalRefs)
	r.Metadata = CloneMap(r.Metadata)
	return r
}

func (r ConsensusReport) IsCanonicalTruth() bool {
	return false
}

func (r ConsensusReport) IsEvidenceAdmitted() bool {
	return false
}

func (input ResponseCompositionInput) Clone() ResponseCompositionInput {
	input.AcceptedClaims = cloneClaims(input.AcceptedClaims)
	input.UncertainClaims = cloneClaims(input.UncertainClaims)
	input.ApprovedActionProposals = cloneClaims(input.ApprovedActionProposals)
	input.MemoryUpdateProposals = cloneClaims(input.MemoryUpdateProposals)
	input.StyleConstraints = CloneStrings(input.StyleConstraints)
	input.ResponseTrace = CloneStrings(input.ResponseTrace)
	input.Metadata = CloneMap(input.Metadata)
	return input
}

func cloneClaims(claims []Claim) []Claim {
	out := make([]Claim, len(claims))
	for i, claim := range claims {
		out[i] = claim.Clone()
	}
	return out
}
