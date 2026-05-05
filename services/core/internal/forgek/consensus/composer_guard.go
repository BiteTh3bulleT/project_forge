package consensus

import "time"

func BuildComposerInput(inputID string, report ConsensusReport, claims []Claim, styleConstraints []string, userCurrentTurnText string, createdAt time.Time) (ResponseCompositionInput, error) {
	byID := make(map[string]Claim, len(claims))
	for _, claim := range claims {
		byID[claim.ClaimID] = claim.Clone()
	}
	out := ResponseCompositionInput{
		InputID:             trim(inputID),
		ReportID:            report.ReportID,
		RequestID:           report.RequestID,
		WorkspaceID:         report.WorkspaceID,
		StyleConstraints:    NormalizeRefs(styleConstraints),
		UserCurrentTurnText: userCurrentTurnText,
		CreatedAt:           createdAt,
		ResponseTrace:       []string{"accepted_claims_only", report.ReportID},
	}
	if out.InputID == "" || out.ReportID == "" || out.WorkspaceID == "" {
		return ResponseCompositionInput{}, ErrComposerInputRejected
	}
	for _, id := range report.AcceptedClaimIDs {
		claim, ok := byID[id]
		if !ok {
			continue
		}
		switch claim.ClaimType {
		case ClaimTypeActionProposal:
			out.ApprovedActionProposals = append(out.ApprovedActionProposals, claim.Clone())
		case ClaimTypeMemoryUpdateProposal:
			out.MemoryUpdateProposals = append(out.MemoryUpdateProposals, claim.Clone())
		default:
			out.AcceptedClaims = append(out.AcceptedClaims, claim.Clone())
		}
	}
	for _, id := range report.UncertainClaimIDs {
		if claim, ok := byID[id]; ok {
			out.UncertainClaims = append(out.UncertainClaims, claim.Clone())
		}
	}
	if err := ValidateComposerInput(out, report); err != nil {
		return ResponseCompositionInput{}, err
	}
	return out.Clone(), nil
}

func ValidateComposerInput(input ResponseCompositionInput, report ConsensusReport) error {
	rejected := make(map[string]bool)
	for _, id := range report.RejectedClaimIDs {
		rejected[id] = true
	}
	for _, id := range report.ConflictedClaimIDs {
		rejected[id] = true
	}
	for _, claim := range append(append(append([]Claim{}, input.AcceptedClaims...), input.ApprovedActionProposals...), input.MemoryUpdateProposals...) {
		if rejected[claim.ClaimID] || claim.Status != StatusAccepted {
			return ErrRejectedClaimReference
		}
	}
	for _, claim := range input.UncertainClaims {
		if rejected[claim.ClaimID] || claim.Status != StatusUncertain {
			return ErrRejectedClaimReference
		}
	}
	return nil
}
