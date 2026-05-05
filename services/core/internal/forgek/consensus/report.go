package consensus

import (
	"sort"
	"strconv"
	"time"
)

func BuildReport(reportID string, request ConsensusRequest, policy ConsensusPolicy, decisions []ConsensusDecision, createdAt time.Time) ConsensusReport {
	report := ConsensusReport{
		ReportID:    reportID,
		RequestID:   request.RequestID,
		WorkspaceID: request.WorkspaceID,
		CaseID:      request.CaseID,
		PolicyID:    policy.PolicyID,
		Decisions:   cloneDecisions(decisions),
		CreatedAt:   createdAt,
	}
	for _, decision := range decisions {
		report.AcceptedClaimIDs = append(report.AcceptedClaimIDs, decision.AcceptedClaimIDs...)
		report.UncertainClaimIDs = append(report.UncertainClaimIDs, decision.UncertainClaimIDs...)
		report.RejectedClaimIDs = append(report.RejectedClaimIDs, decision.RejectedClaimIDs...)
		report.ConflictedClaimIDs = append(report.ConflictedClaimIDs, decision.ConflictedClaimIDs...)
		if decision.Status == StatusNeedsMoreEvidence || decision.Status == StatusConflicted || decision.Status == StatusDeferred {
			report.Escalations = append(report.Escalations, decision.ClaimKey+":"+string(decision.Status))
		}
	}
	report.AcceptedClaimIDs = NormalizeRefs(report.AcceptedClaimIDs)
	report.UncertainClaimIDs = NormalizeRefs(report.UncertainClaimIDs)
	report.RejectedClaimIDs = NormalizeRefs(report.RejectedClaimIDs)
	report.ConflictedClaimIDs = NormalizeRefs(report.ConflictedClaimIDs)
	report.Escalations = NormalizeRefs(report.Escalations)
	report.Summary = reportSummary(report)
	return report
}

func cloneDecisions(decisions []ConsensusDecision) []ConsensusDecision {
	out := make([]ConsensusDecision, len(decisions))
	for i, decision := range decisions {
		out[i] = decision.Clone()
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ClaimKey < out[j].ClaimKey })
	return out
}

func reportSummary(report ConsensusReport) string {
	return "accepted=" + strconv.Itoa(len(report.AcceptedClaimIDs)) +
		" uncertain=" + strconv.Itoa(len(report.UncertainClaimIDs)) +
		" rejected=" + strconv.Itoa(len(report.RejectedClaimIDs)) +
		" conflicted=" + strconv.Itoa(len(report.ConflictedClaimIDs))
}
