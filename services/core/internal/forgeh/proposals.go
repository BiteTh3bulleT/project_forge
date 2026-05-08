package forgeh

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"
)

var (
	ErrProposalNotProposed      = errors.New("proposal is not in proposed status")
	ErrProposalExecutionBlocked = errors.New("resource proposal execution is not allowed in phase n5")
	ErrProposalExpired          = errors.New("proposal is expired")
	ErrInvalidProposalStatus    = errors.New("invalid proposal status")
)

type ProposalOptions struct {
	Now       func() time.Time
	TTL       time.Duration
	ExpiresAt time.Time
}

func GenerateResourceActionProposals(policy ResourcePolicySnapshot, opts ProposalOptions) []ResourceActionProposal {
	now := proposalNow(opts)
	expiresAt := opts.ExpiresAt
	if expiresAt.IsZero() {
		ttl := opts.TTL
		if ttl <= 0 {
			ttl = 24 * time.Hour
		}
		expiresAt = now.Add(ttl)
	}
	proposals := []ResourceActionProposal{}
	add := func(actionType, lane, decision, reason, risk string) {
		proposal := ResourceActionProposal{
			CreatedAt:                now,
			SourcePolicyID:           policy.PolicyID,
			SourceSnapshotID:         policy.SourceSnapshotID,
			ActionType:               actionType,
			TargetLane:               lane,
			RecommendedDecision:      decision,
			Reason:                   reason,
			RiskLevel:                risk,
			RequiresOperatorApproval: true,
			Status:                   ProposalStatusProposed,
			ExpiresAt:                expiresAt,
			AdvisoryOnly:             true,
		}
		proposal.ProposalID = proposalID(proposal)
		proposals = append(proposals, proposal)
	}

	if decision, ok := policy.LaneDecisions[WorkloadLaneBackgroundIngest]; ok {
		switch decision.Decision {
		case PolicyDecisionDefer:
			add(ProposalActionDeferBackgroundIngest, WorkloadLaneBackgroundIngest, PolicyDecisionDefer, firstReason(decision, "background ingest should be deferred"), ProposalRiskModerate)
		case PolicyDecisionDeny:
			add(ProposalActionPauseBackgroundIngest, WorkloadLaneBackgroundIngest, PolicyDecisionDeny, firstReason(decision, "background ingest should pause"), riskForPosture(policy.OverallPosture))
		case PolicyDecisionAllowWithWarning:
			add(ProposalActionWarnOperator, WorkloadLaneBackgroundIngest, PolicyDecisionAllowWithWarning, firstReason(decision, "background ingest allowed with warning"), ProposalRiskLow)
		}
	}

	if decision, ok := policy.LaneDecisions[WorkloadLaneEmbedding]; ok {
		switch decision.Decision {
		case PolicyDecisionDefer:
			add(ProposalActionDeferEmbedding, WorkloadLaneEmbedding, PolicyDecisionDefer, firstReason(decision, "embedding should be deferred"), ProposalRiskModerate)
		case PolicyDecisionDeny:
			add(ProposalActionDeferEmbedding, WorkloadLaneEmbedding, PolicyDecisionDeny, firstReason(decision, "embedding should not run under current pressure"), riskForPosture(policy.OverallPosture))
		}
	}

	switch policy.ModelLoadRecommendation {
	case ModelLoadCurrentModelOnly:
		add(ProposalActionPreferCurrentModelOnly, WorkloadLaneModelLoad, PolicyDecisionAllowWithWarning, "Prefer currently loaded models until pressure falls.", ProposalRiskLow)
	case ModelLoadDeferLargeModel:
		add(ProposalActionDeferLargeModelLoad, WorkloadLaneModelLoad, PolicyDecisionDefer, "Defer large model loads until resource pressure falls.", ProposalRiskModerate)
	case ModelLoadCPUOnlySafeMode:
		add(ProposalActionPreferCPUSafeMode, WorkloadLaneModelLoad, PolicyDecisionAllowWithWarning, "Prefer CPU-only safe mode because GPU diagnostics are unavailable.", ProposalRiskLow)
	case ModelLoadDenyNewModelLoad:
		add(ProposalActionDenyNewModelLoad, WorkloadLaneModelLoad, PolicyDecisionDeny, "Deny new model loads while model resources are critical.", riskForPosture(policy.OverallPosture))
	case ModelLoadUnavailable:
		add(ProposalActionWarnOperator, WorkloadLaneModelLoad, PolicyDecisionUnavailable, "Model-load recommendation is unavailable because required diagnostics are missing.", ProposalRiskModerate)
	}

	switch policy.OverallPosture {
	case ResourcePostureDegraded:
		add(ProposalActionWarnOperator, "", PolicyDecisionAllowWithWarning, "Resource posture is degraded.", ProposalRiskLow)
	case ResourcePostureConstrained:
		add(ProposalActionEnterDegradedMode, "", PolicyDecisionDefer, "Resource posture is constrained; degraded operation is recommended.", ProposalRiskHigh)
	case ResourcePostureCritical:
		add(ProposalActionEnterDegradedMode, "", PolicyDecisionDeny, "Resource posture is critical; degraded operation is recommended.", ProposalRiskCritical)
	}

	sort.Slice(proposals, func(i, j int) bool {
		if proposals[i].ActionType == proposals[j].ActionType {
			return proposals[i].TargetLane < proposals[j].TargetLane
		}
		return proposals[i].ActionType < proposals[j].ActionType
	})
	return proposals
}

func ApproveProposal(proposal ResourceActionProposal, now time.Time, reason string) (ResourceActionProposal, error) {
	return transitionProposal(proposal, ProposalStatusApproved, now, reason, "")
}

func RejectProposal(proposal ResourceActionProposal, now time.Time, reason string) (ResourceActionProposal, error) {
	return transitionProposal(proposal, ProposalStatusRejected, now, reason, "")
}

func SupersedeProposal(proposal ResourceActionProposal, now time.Time, supersededBy string, reason string) (ResourceActionProposal, error) {
	return transitionProposal(proposal, ProposalStatusSuperseded, now, reason, supersededBy)
}

func ExpireProposal(proposal ResourceActionProposal, now time.Time) (ResourceActionProposal, error) {
	if proposal.Status != ProposalStatusProposed {
		return proposal, ErrProposalNotProposed
	}
	if !proposal.ExpiresAt.IsZero() && now.Before(proposal.ExpiresAt) {
		return proposal, nil
	}
	proposal.Status = ProposalStatusExpired
	proposal.DecidedAt = now.UTC()
	proposal.DecisionReason = "proposal expired"
	return proposal, nil
}

func transitionProposal(proposal ResourceActionProposal, status string, now time.Time, reason string, supersededBy string) (ResourceActionProposal, error) {
	if status == ProposalStatusCommittedLater {
		return proposal, ErrProposalExecutionBlocked
	}
	if status != ProposalStatusApproved && status != ProposalStatusRejected && status != ProposalStatusSuperseded {
		return proposal, ErrInvalidProposalStatus
	}
	if proposal.Status != ProposalStatusProposed {
		return proposal, ErrProposalNotProposed
	}
	if !proposal.ExpiresAt.IsZero() && !now.Before(proposal.ExpiresAt) {
		return proposal, ErrProposalExpired
	}
	proposal.Status = status
	proposal.DecidedAt = now.UTC()
	proposal.DecisionReason = strings.TrimSpace(reason)
	if status == ProposalStatusSuperseded {
		proposal.SupersededBy = strings.TrimSpace(supersededBy)
	}
	return proposal, nil
}

func proposalNow(opts ProposalOptions) time.Time {
	if opts.Now != nil {
		return opts.Now().UTC()
	}
	return time.Now().UTC()
}

func firstReason(decision LaneDecision, fallback string) string {
	if len(decision.Reasons) > 0 && strings.TrimSpace(decision.Reasons[0]) != "" {
		return strings.TrimSpace(decision.Reasons[0])
	}
	return fallback
}

func riskForPosture(posture string) string {
	switch posture {
	case ResourcePostureCritical:
		return ProposalRiskCritical
	case ResourcePostureConstrained:
		return ProposalRiskHigh
	case ResourcePostureDegraded:
		return ProposalRiskModerate
	default:
		return ProposalRiskLow
	}
}

func proposalID(proposal ResourceActionProposal) string {
	clone := proposal
	clone.ProposalID = ""
	body, _ := json.Marshal(clone)
	sum := sha256.Sum256(body)
	return "forgeh_proposal_" + hex.EncodeToString(sum[:8])
}
