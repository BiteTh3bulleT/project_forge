package controllane

import (
	"strings"
	"testing"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/forgekernel"
	"forge/projectforge/services/core/internal/forgekernel/court"
	"forge/projectforge/services/core/internal/forgekernel/semanticdiff"
)

func TestSemanticDiffValidationAndPreparedPlanAreForgeKOnly(t *testing.T) {
	store, req := semanticDiffFixture(t)
	if issues := validateComputeSemanticDiff(req, store); len(issues) != 0 {
		t.Fatalf("valid semantic diff rejected: %+v", issues)
	}
	if !isForgeKOnlyAction(req.Action) {
		t.Fatal("semantic diff is not production-FORGE-K-only")
	}
	ids, err := expectedCommitObjectIDs(req, store)
	want := semanticDiffObjectIDs(req.ID)
	if err != nil || len(ids) != len(want) {
		t.Fatalf("plan IDs: got=%v want=%v err=%v", ids, want, err)
	}
	for i := range want {
		if ids[i] != want[i] {
			t.Fatalf("plan IDs: got=%v want=%v", ids, want)
		}
	}
	def, ok := NewStaticActionRegistry().Get(req.Action)
	if !ok || !def.Mutating || def.Capability != CapSemanticDiffCompute || def.ApprovalPossible {
		t.Fatalf("unexpected registry definition: ok=%t def=%+v", ok, def)
	}
}

func TestSemanticDiffRejectsCallerOutputAndUntrustedSources(t *testing.T) {
	store, req := semanticDiffFixture(t)
	for name, mutate := range map[string]func(*domain.SyscallRequest){
		"caller output":       func(r *domain.SyscallRequest) { r.Payload["content"] = "invented" },
		"missing paths":       func(r *domain.SyscallRequest) { r.Scope.SelectedPaths = nil },
		"missing idempotency": func(r *domain.SyscallRequest) { r.IdempotencyKey = "" },
		"adapter":             func(r *domain.SyscallRequest) { r.Source = domain.SourceAdapter },
		"future iris":         func(r *domain.SyscallRequest) { r.Source = domain.SourceFutureIRIS },
		"model actor":         func(r *domain.SyscallRequest) { r.Actor.Kind = "llm_model"; r.Source = domain.SourceAdapter },
		"wrong version":       func(r *domain.SyscallRequest) { r.Payload["operatorVersion"] = "semantic.diff.v2" },
	} {
		t.Run(name, func(t *testing.T) {
			candidate := req
			candidate.Payload = cloneMap(req.Payload)
			candidate.Scope.SelectedPaths = append([]string(nil), req.Scope.SelectedPaths...)
			mutate(&candidate)
			if issues := validateComputeSemanticDiff(candidate, store); len(issues) == 0 {
				t.Fatal("invalid request passed")
			}
		})
	}
}

func TestInMemorySemanticDiffPersistsImmutableNoncanonicalRecords(t *testing.T) {
	store, req := semanticDiffFixture(t)
	input, err := prepareSemanticDiffAuthorityInput(req, store)
	if err != nil {
		t.Fatal(err)
	}
	decision, issues := semanticdiff.Decide(req, input)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	req.Metadata = map[string]any{
		"forgeKIngressAuthority": true, "kernelAuthorityOwner": forgekernel.AuthorityOwnerForgeK,
		"forgeKAuthorizationProof":       semanticdiff.MaterialHash("auth"),
		semanticdiff.MetadataDecisionKey: decision,
	}
	ids, summary, _, applyIssues := applyComputeSemanticDiff(nil, store, req)
	if len(applyIssues) != 0 || len(ids) != 3 || summary["canonicalTruth"] != false {
		t.Fatalf("apply failed: ids=%v summary=%v issues=%v", ids, summary, applyIssues)
	}
	operation, ok := store.FindSemanticDiffOperation(ids[0], req.Scope)
	if !ok || operation.SourceManifestHash != decision.SourceManifestHash {
		t.Fatalf("operation missing: ok=%t operation=%+v", ok, operation)
	}
	object, ok := store.FindSemanticDerivedObject(ids[2], req.Scope)
	if !ok || object.CanonicalTruth || object.ObjectClass != semanticdiff.DerivedObjectClass {
		t.Fatalf("derived object authority drift: ok=%t object=%+v", ok, object)
	}
	if _, _, _, replayIssues := applyComputeSemanticDiff(nil, store, req); len(replayIssues) == 0 {
		t.Fatal("direct duplicate apply unexpectedly succeeded")
	}
}

func semanticDiffFixture(t *testing.T) (*InMemorySemanticStore, domain.SyscallRequest) {
	t.Helper()
	store := NewInMemorySemanticStore()
	scope := domain.ForgeScope{WorkspaceID: "ws-diff", LaneID: "memory.semantic", SelectedPaths: []string{"artifact:diff"}}
	seed := func(suffix, summary, hashChar string) string {
		exhibitID := "exhibit-" + suffix
		rulingID := "ruling-" + suffix
		evidenceID := "evidence-" + suffix
		contentHash := "sha256:" + strings.Repeat(hashChar, 64)
		prov := domain.Provenance{Actor: "operator", ActorType: "user", Source: "test", TraceID: "trace-" + suffix}
		exhibit := court.Exhibit{
			ID: exhibitID, CaseID: "case-diff", Scope: scope, SourceType: "artifact", SourceRefs: []string{"artifact:" + suffix},
			ContentSummary: summary, RawRef: "artifact:" + suffix, ContentHash: contentHash,
			Status: court.DecisionAdmitted, CurrentRulingID: rulingID, CreatedAt: 1, UpdatedAt: 1,
			Provenance: prov, SyscallID: "admit-" + suffix, CorrelationID: "corr-" + suffix,
			TraceID: "trace-" + suffix, ProposedBy: "operator", CommittedBy: forgekernel.AuthorityOwnerForgeK,
		}
		ruling := court.Ruling{
			ID: rulingID, CaseID: "case-diff", ExhibitID: exhibitID, Scope: scope,
			Decision: court.DecisionAdmitted, PolicyVersion: court.PolicyVersion, ContentHash: contentHash,
			CreatedAt: 1, Provenance: prov, SyscallID: "admit-" + suffix,
			CorrelationID: "corr-" + suffix, TraceID: "trace-" + suffix,
			ProposedBy: "operator", CommittedBy: forgekernel.AuthorityOwnerForgeK,
		}
		if err := store.CreateCourtDecision(exhibit, ruling, nil); err != nil {
			t.Fatal(err)
		}
		evidence := MemoryEvidence{
			EvidenceID: evidenceID, RootEvidenceID: evidenceID, Revision: 1,
			CourtCaseID: "case-diff", CourtExhibitID: exhibitID, CourtRulingID: rulingID,
			AdmissionSyscallID: "admit-" + suffix, Scope: scope, ContentSummary: summary,
			ContentHash: contentHash, SourceProvenanceID: "source-prov-" + suffix,
			MaterializationProvenanceID: "materialize-prov-" + suffix, CreatedAt: 2,
			CommittedBy: forgekernel.AuthorityOwnerForgeK,
		}
		if err := store.CreateMemoryEvidence(evidence, nil); err != nil {
			t.Fatal(err)
		}
		return evidenceID
	}
	left := seed("left", "alpha beta", "a")
	right := seed("right", "alpha", "b")
	req := domain.SyscallRequest{
		ID: "semantic-diff", Action: domain.ActionComputeSemanticDiff,
		Actor: domain.ActorIdentity{ID: "operator", Kind: string(domain.SourceUser)}, Source: domain.SourceUser,
		Scope: scope, Payload: map[string]any{
			"leftEvidenceId": left, "rightEvidenceId": right, "operatorVersion": semanticdiff.OperatorVersion,
		},
		Provenance:    domain.Provenance{Actor: "operator", ActorType: "user", Source: "test", TraceID: "trace-diff"},
		CorrelationID: "corr-diff", TraceID: "trace-diff", RequestedAt: 3, IdempotencyKey: "idem-diff",
	}
	return store, req
}
