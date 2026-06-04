package retrieval

import "testing"

func TestSearchEvidencePacketRecordsTrustAndRejectedCandidates(t *testing.T) {
	packet := SearchEvidencePacket{
		ID:          "sep-1",
		WorkspaceID: "ws-main",
		Query:       "current gateway authority",
		RoutingMode: RoutingModeCodeContext,
		CreatedAtMs: 1760000000000,
		Candidates: []SearchEvidenceCandidate{
			{
				ID:            "cand-local",
				SourceRef:     "file:docs/architecture/tool_gateway.md",
				SourceKind:    SourceKindLocalLive,
				TrustTier:     TrustTierLocalLive,
				FreshnessMs:   1760000000000,
				Scope:         EvidenceScope{WorkspaceID: "ws-main", LaneID: "operator", Path: "docs/architecture/tool_gateway.md"},
				Summary:       "Gateway is the only tool boundary.",
				Citation:      Citation{Ref: "docs/architecture/tool_gateway.md:1"},
				Relevance:     0.98,
				Selected:      true,
				SelectionNote: "fresh local authority doc",
			},
		},
		RejectedCandidates: []RejectedSearchCandidate{
			{
				CandidateID:   "cand-web-old",
				SourceRef:     "web:https://example.invalid/old-forge",
				TrustTier:     TrustTierWeb,
				Reason:        RejectionReasonStale,
				ReplacedByRef: "file:docs/architecture/tool_gateway.md",
			},
		},
	}

	if issues := packet.Validate(); len(issues) != 0 {
		t.Fatalf("Validate() issues = %#v, want none", issues)
	}
	if packet.Candidates[0].TrustTier != TrustTierLocalLive {
		t.Fatalf("candidate trust tier = %q, want local_live", packet.Candidates[0].TrustTier)
	}
	if packet.RejectedCandidates[0].Reason != RejectionReasonStale {
		t.Fatalf("rejection reason = %q, want stale", packet.RejectedCandidates[0].Reason)
	}
}

func TestSearchEvidencePacketCannotAuthorizeExecutionOrMemoryWrites(t *testing.T) {
	packet := SearchEvidencePacket{ID: "sep-2", WorkspaceID: "ws-main", Query: "run tool", RoutingMode: RoutingModeAnswerContext}

	if packet.CanAuthorizeExecution() {
		t.Fatalf("SearchEvidencePacket must never authorize execution")
	}
	if packet.CanWriteCanonicalMemory() {
		t.Fatalf("SearchEvidencePacket must never write canonical memory")
	}
	if packet.CanBypassAuthorityPlane() {
		t.Fatalf("SearchEvidencePacket must never bypass Gateway, approvals, audit, Control Lane, or modelruntime")
	}
}

func TestContextPacketPreservesSelectedAndRejectedRefs(t *testing.T) {
	packet := ContextPacket{
		ID:              "ctx-axiom-1",
		WorkspaceID:     "ws-main",
		Query:           "explain gateway authority",
		RoutingMode:     RoutingModeCodeContext,
		SourcePacketIDs: []string{"sep-1"},
		SelectedRefs: []ContextSourceRef{
			{
				SourceRef:   "file:docs/architecture/tool_gateway.md",
				CandidateID: "cand-local",
				TrustTier:   TrustTierLocalLive,
				Citation:    Citation{Ref: "docs/architecture/tool_gateway.md:1"},
			},
		},
		RejectedRefs: []RejectedSearchCandidate{
			{CandidateID: "cand-web-old", SourceRef: "web:https://example.invalid/old-forge", TrustTier: TrustTierWeb, Reason: RejectionReasonStale},
		},
		TokenBudget: 4096,
		CreatedAtMs: 1760000000000,
	}

	if issues := packet.Validate(); len(issues) != 0 {
		t.Fatalf("Validate() issues = %#v, want none", issues)
	}
	if packet.SelectedRefs[0].TrustTier != TrustTierLocalLive {
		t.Fatalf("selected trust tier = %q, want local_live", packet.SelectedRefs[0].TrustTier)
	}
	if packet.RejectedRefs[0].Reason != RejectionReasonStale {
		t.Fatalf("rejected reason = %q, want stale", packet.RejectedRefs[0].Reason)
	}
}

func TestContextPacketCannotExecuteWriteMemoryOrBypassAuthority(t *testing.T) {
	packet := ContextPacket{ID: "ctx-axiom-2", WorkspaceID: "ws-main", Query: "answer", RoutingMode: RoutingModeAnswerContext, TokenBudget: 1024}

	if packet.CanAuthorizeExecution() {
		t.Fatalf("ContextPacket must never authorize execution")
	}
	if packet.CanWriteCanonicalMemory() {
		t.Fatalf("ContextPacket must never write canonical memory")
	}
	if packet.CanBypassAuthorityPlane() {
		t.Fatalf("ContextPacket must never bypass Gateway, approvals, audit, Control Lane, or modelruntime")
	}
}

func TestAxiomPacketsValidateRequiredScopeAndRouting(t *testing.T) {
	evidence := SearchEvidencePacket{ID: "sep-missing", Query: "missing scope"}
	if issues := evidence.Validate(); len(issues) < 2 {
		t.Fatalf("SearchEvidencePacket Validate() issues = %#v, want missing workspace and routing", issues)
	}

	context := ContextPacket{ID: "ctx-missing", WorkspaceID: "ws-main", Query: "missing budget", RoutingMode: RoutingModeAnswerContext}
	if issues := context.Validate(); len(issues) != 1 || issues[0].Field != "tokenBudget" {
		t.Fatalf("ContextPacket Validate() issues = %#v, want tokenBudget issue", issues)
	}
}
