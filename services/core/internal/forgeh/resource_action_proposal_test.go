package forgeh

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"forge/projectforge/services/core/internal/hostbridge"
)

func TestGenerateResourceActionProposalsNormalPolicyProducesNone(t *testing.T) {
	policy := New(Options{Now: fixedNow}).Evaluate(normalSnapshot())
	proposals := GenerateResourceActionProposals(policy, ProposalOptions{Now: fixedNow})
	if len(proposals) != 0 {
		t.Fatalf("expected no proposals for normal policy, got %#v", proposals)
	}
}

func TestGenerateResourceActionProposalsBackgroundIngestDefer(t *testing.T) {
	policy := policyWith(func(s *testSnapshot) {
		s.memoryAvailable = 25
	})
	proposals := GenerateResourceActionProposals(policy, ProposalOptions{Now: fixedNow})
	proposal := requireProposal(t, proposals, ProposalActionDeferBackgroundIngest)
	if proposal.TargetLane != WorkloadLaneBackgroundIngest {
		t.Fatalf("target lane = %q", proposal.TargetLane)
	}
	if proposal.SourcePolicyID == "" || proposal.SourceSnapshotID == "" {
		t.Fatalf("missing source refs: %#v", proposal)
	}
	if proposal.Status != ProposalStatusProposed || !proposal.AdvisoryOnly || !proposal.RequiresOperatorApproval {
		t.Fatalf("invalid default governance fields: %#v", proposal)
	}
}

func TestGenerateResourceActionProposalsBackgroundIngestDenyPauses(t *testing.T) {
	policy := policyWith(func(s *testSnapshot) {
		s.memoryAvailable = 1
	})
	proposals := GenerateResourceActionProposals(policy, ProposalOptions{Now: fixedNow})
	proposal := requireProposal(t, proposals, ProposalActionPauseBackgroundIngest)
	if proposal.RiskLevel != ProposalRiskCritical {
		t.Fatalf("risk = %q", proposal.RiskLevel)
	}
}

func TestGenerateResourceActionProposalsEmbeddingDefer(t *testing.T) {
	policy := policyWith(func(s *testSnapshot) {
		s.diskFree = 15
	})
	proposal := requireProposal(t, GenerateResourceActionProposals(policy, ProposalOptions{Now: fixedNow}), ProposalActionDeferEmbedding)
	if proposal.TargetLane != WorkloadLaneEmbedding || proposal.RecommendedDecision != PolicyDecisionDefer {
		t.Fatalf("unexpected embedding proposal: %#v", proposal)
	}
}

func TestGenerateResourceActionProposalsModelLoadRecommendations(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*testSnapshot)
		action string
	}{
		{name: "current model only", mutate: func(s *testSnapshot) { s.gpuFree = 30 }, action: ProposalActionPreferCurrentModelOnly},
		{name: "defer large model", mutate: func(s *testSnapshot) { s.gpuFree = 12 }, action: ProposalActionDeferLargeModelLoad},
		{name: "cpu safe mode", mutate: func(s *testSnapshot) { s.gpuUnavailable = true }, action: ProposalActionPreferCPUSafeMode},
		{name: "deny new model load", mutate: func(s *testSnapshot) { s.memoryAvailable = 1 }, action: ProposalActionDenyNewModelLoad},
		{name: "unavailable", mutate: func(s *testSnapshot) { s.memoryTotal = 0; s.memoryAvailable = 0 }, action: ProposalActionWarnOperator},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy := policyWith(tt.mutate)
			proposal := requireProposal(t, GenerateResourceActionProposals(policy, ProposalOptions{Now: fixedNow}), tt.action)
			if proposal.TargetLane == WorkloadLaneModelLoad && proposal.ActionType == "" {
				t.Fatalf("invalid model proposal: %#v", proposal)
			}
		})
	}
}

func TestGenerateResourceActionProposalsPosture(t *testing.T) {
	degraded := policyWith(func(s *testSnapshot) { s.memoryAvailable = 25 })
	requireProposal(t, GenerateResourceActionProposals(degraded, ProposalOptions{Now: fixedNow}), ProposalActionWarnOperator)

	constrained := policyWith(func(s *testSnapshot) { s.memoryAvailable = 10 })
	constrainedProposal := requireProposal(t, GenerateResourceActionProposals(constrained, ProposalOptions{Now: fixedNow}), ProposalActionEnterDegradedMode)
	if constrainedProposal.RiskLevel != ProposalRiskHigh {
		t.Fatalf("constrained degraded-mode risk = %q", constrainedProposal.RiskLevel)
	}

	critical := policyWith(func(s *testSnapshot) { s.memoryAvailable = 1 })
	criticalProposal := requireProposal(t, GenerateResourceActionProposals(critical, ProposalOptions{Now: fixedNow}), ProposalActionEnterDegradedMode)
	if criticalProposal.RiskLevel != ProposalRiskCritical {
		t.Fatalf("critical degraded-mode risk = %q", criticalProposal.RiskLevel)
	}
}

func TestProposalIDsAreDeterministic(t *testing.T) {
	policy := policyWith(func(s *testSnapshot) { s.gpuFree = 12 })
	left := GenerateResourceActionProposals(policy, ProposalOptions{Now: fixedNow})
	right := GenerateResourceActionProposals(policy, ProposalOptions{Now: fixedNow})
	if len(left) != len(right) {
		t.Fatalf("length mismatch: %d vs %d", len(left), len(right))
	}
	for i := range left {
		if left[i].ProposalID == "" || left[i].ProposalID != right[i].ProposalID {
			t.Fatalf("proposal ids not deterministic: %#v %#v", left[i], right[i])
		}
	}
}

func TestProposalJSONIncludesGovernanceFields(t *testing.T) {
	policy := policyWith(func(s *testSnapshot) { s.gpuFree = 12 })
	proposal := requireProposal(t, GenerateResourceActionProposals(policy, ProposalOptions{Now: fixedNow}), ProposalActionDeferLargeModelLoad)
	body, err := json.Marshal(proposal)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"proposal_id", "created_at", "source_policy_id", "source_snapshot_id", "action_type", "target_lane", "recommended_decision", "reason", "risk_level", "requires_operator_approval", "status", "expires_at", "advisory_only"} {
		if _, ok := decoded[key]; !ok {
			t.Fatalf("missing key %q in %s", key, string(body))
		}
	}
}

func TestProposalLifecycleTransitions(t *testing.T) {
	proposal := sampleProposal(t)
	for _, transition := range []struct {
		name string
		fn   func(ResourceActionProposal) (ResourceActionProposal, error)
		want string
	}{
		{name: "approve", fn: func(p ResourceActionProposal) (ResourceActionProposal, error) {
			return ApproveProposal(p, fixedNow().Add(time.Hour), "approved for review")
		}, want: ProposalStatusApproved},
		{name: "reject", fn: func(p ResourceActionProposal) (ResourceActionProposal, error) {
			return RejectProposal(p, fixedNow().Add(time.Hour), "not now")
		}, want: ProposalStatusRejected},
		{name: "supersede", fn: func(p ResourceActionProposal) (ResourceActionProposal, error) {
			return SupersedeProposal(p, fixedNow().Add(time.Hour), "replacement", "newer policy")
		}, want: ProposalStatusSuperseded},
	} {
		t.Run(transition.name, func(t *testing.T) {
			got, err := transition.fn(proposal)
			if err != nil {
				t.Fatal(err)
			}
			if got.Status != transition.want || got.DecidedAt.IsZero() {
				t.Fatalf("bad transition: %#v", got)
			}
		})
	}
}

func TestProposalLifecycleRejectsExecutionStatus(t *testing.T) {
	proposal := sampleProposal(t)
	got, err := transitionProposal(proposal, ProposalStatusCommittedLater, fixedNow().Add(time.Hour), "execute", "")
	if !errors.Is(err, ErrProposalExecutionBlocked) {
		t.Fatalf("expected execution blocked, got %v", err)
	}
	if got.Status != ProposalStatusProposed {
		t.Fatalf("status changed on forbidden transition: %#v", got)
	}
}

func TestProposalExpiration(t *testing.T) {
	proposal := sampleProposal(t)
	unchanged, err := ExpireProposal(proposal, fixedNow().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if unchanged.Status != ProposalStatusProposed {
		t.Fatalf("proposal expired before deadline: %#v", unchanged)
	}
	expired, err := ExpireProposal(proposal, proposal.ExpiresAt.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if expired.Status != ProposalStatusExpired {
		t.Fatalf("not expired: %#v", expired)
	}
	approved, err := ApproveProposal(proposal, fixedNow().Add(time.Hour), "approved")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ExpireProposal(approved, proposal.ExpiresAt.Add(time.Second)); !errors.Is(err, ErrProposalNotProposed) {
		t.Fatalf("expected terminal proposal not to expire, got %v", err)
	}
}

func TestProposalAdvisoryOnlyDefault(t *testing.T) {
	proposal := sampleProposal(t)
	if !proposal.AdvisoryOnly {
		t.Fatalf("proposal must be advisory only: %#v", proposal)
	}
}

type testSnapshot struct {
	memoryTotal     uint64
	memoryAvailable uint64
	diskFree        uint64
	gpuFree         float64
	gpuUnavailable  bool
}

func policyWith(mutate func(*testSnapshot)) ResourcePolicySnapshot {
	cfg := testSnapshot{memoryTotal: 100, memoryAvailable: 40, diskFree: 40, gpuFree: 50}
	if mutate != nil {
		mutate(&cfg)
	}
	snapshot := normalSnapshot()
	snapshot.Memory.TotalBytes = cfg.memoryTotal
	snapshot.Memory.AvailableBytes = cfg.memoryAvailable
	snapshot.Disk.FreeBytes = cfg.diskFree
	if cfg.gpuUnavailable {
		snapshot.GPU = zeroGPU()
	} else {
		snapshot.GPU = gpuWithFree(cfg.gpuFree)
	}
	return New(Options{Now: fixedNow}).Evaluate(snapshot)
}

func zeroGPU() hostbridge.GPUDiagnostics {
	return hostbridge.GPUDiagnostics{}
}

func requireProposal(t *testing.T, proposals []ResourceActionProposal, action string) ResourceActionProposal {
	t.Helper()
	for _, proposal := range proposals {
		if proposal.ActionType == action {
			return proposal
		}
	}
	t.Fatalf("missing proposal action %q in %#v", action, proposals)
	return ResourceActionProposal{}
}

func sampleProposal(t *testing.T) ResourceActionProposal {
	t.Helper()
	policy := policyWith(func(s *testSnapshot) {
		s.gpuFree = 12
	})
	proposals := GenerateResourceActionProposals(policy, ProposalOptions{
		Now: fixedNow,
		TTL: 2 * time.Hour,
	})
	return requireProposal(t, proposals, ProposalActionDeferLargeModelLoad)
}
