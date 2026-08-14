package controllane

import (
	"strings"
	"testing"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/forgekernel"
	"forge/projectforge/services/core/internal/forgekernel/court"
)

func TestMemoryEvidenceValidationRequiresExactGovernedScopeIdempotencyAndTrustedSource(t *testing.T) {
	store := NewInMemorySemanticStore()
	req := memoryEvidenceValidationRequest()
	seedMemoryEvidenceCourt(t, store, req.Scope, "sha256:"+strings.Repeat("a", 64), "sha256:"+strings.Repeat("a", 64), court.DecisionAdmitted)
	if issues := validateMemoryEvidenceAction(req, store); len(issues) != 0 {
		t.Fatalf("valid request rejected: %#v", issues)
	}
	for name, mutate := range map[string]func(*domain.SyscallRequest){
		"workspace":   func(req *domain.SyscallRequest) { req.Scope.WorkspaceID = "" },
		"lane":        func(req *domain.SyscallRequest) { req.Scope.LaneID = "" },
		"paths":       func(req *domain.SyscallRequest) { req.Scope.SelectedPaths = nil },
		"idempotency": func(req *domain.SyscallRequest) { req.IdempotencyKey = "" },
		"adapter":     func(req *domain.SyscallRequest) { req.Source = domain.SourceAdapter },
		"future iris": func(req *domain.SyscallRequest) { req.Source = domain.SourceFutureIRIS },
	} {
		t.Run(name, func(t *testing.T) {
			tampered := req
			tampered.Scope.SelectedPaths = append([]string(nil), req.Scope.SelectedPaths...)
			mutate(&tampered)
			if issues := validateMemoryEvidenceAction(tampered, store); len(issues) == 0 {
				t.Fatal("invalid authority/scope request passed")
			}
		})
	}
	for _, source := range []domain.ActionSource{domain.SourceUser, domain.SourceSystem, domain.SourceInternal} {
		if policy, ok := productionCapabilityPolicy(source, req.Action); !ok || strings.TrimSpace(policy) == "" {
			t.Fatalf("production source %q lacks explicit policy: policy=%q ok=%v", source, policy, ok)
		}
	}
}

func TestMemoryEvidenceValidationRejectsCallerFieldsAndInconsistentCourtHash(t *testing.T) {
	req := memoryEvidenceValidationRequest()
	req.Payload["contentSummary"] = "caller supplied"
	if issues := validateMemoryEvidenceAction(req, nil); len(issues) == 0 || issues[0].Field != "payload" {
		t.Fatalf("unknown caller content was accepted: %#v", issues)
	}

	store := NewInMemorySemanticStore()
	req = memoryEvidenceValidationRequest()
	seedMemoryEvidenceCourt(t, store, req.Scope, "sha256:"+strings.Repeat("a", 64), "sha256:"+strings.Repeat("b", 64), court.DecisionAdmitted)
	if issues := validateMemoryEvidenceAction(req, store); len(issues) == 0 {
		t.Fatal("exhibit/ruling content hash mismatch was accepted")
	}
}

func TestMemoryEvidencePreparedPlanDeclaresExactImmutableObjects(t *testing.T) {
	if !isForgeKOnlyAction(domain.ActionMaterializeAdmittedEvidence) || !isForgeKOnlyAction(domain.ActionReviseMemoryEvidence) {
		t.Fatal("K20H actions are not production-FORGE-K-only")
	}
	req := memoryEvidenceValidationRequest()
	ids, err := expectedCommitObjectIDs(req, nil)
	if err != nil || len(ids) != 1 || ids[0] != req.ID+":memory_evidence" {
		t.Fatalf("materialize ids=%v err=%v", ids, err)
	}
	req.Action = domain.ActionReviseMemoryEvidence
	req.Payload["priorEvidenceId"] = "prior"
	ids, err = expectedCommitObjectIDs(req, nil)
	if err != nil || len(ids) != 2 || ids[0] != req.ID+":memory_evidence" || ids[1] != req.ID+":memory_supersession" {
		t.Fatalf("revision ids=%v err=%v", ids, err)
	}
}

func TestMemoryEvidenceCannotPredateCourtSourceOrPriorRevision(t *testing.T) {
	store := NewInMemorySemanticStore()
	req := memoryEvidenceValidationRequest()
	seedMemoryEvidenceCourt(t, store, req.Scope, "sha256:"+strings.Repeat("a", 64), "sha256:"+strings.Repeat("a", 64), court.DecisionAdmitted)
	req.RequestedAt = 0
	if issues := validateMemoryEvidenceAction(req, store); len(issues) == 0 {
		t.Fatal("materialization predating Court source passed")
	}

	req = memoryEvidenceValidationRequest()
	req.Action = domain.ActionReviseMemoryEvidence
	req.Payload["priorEvidenceId"] = "prior"
	prior := MemoryEvidence{EvidenceID: "prior", RootEvidenceID: "prior", Revision: 1, CourtCaseID: "case", Scope: req.Scope, CreatedAt: 2}
	store.state.memoryEvidence[prior.EvidenceID] = prior
	if issues := validateMemoryEvidenceAction(req, store); len(issues) == 0 {
		t.Fatal("revision predating prior evidence passed")
	}
}

func TestInMemoryMemoryEvidencePreservesImmutableRevisionParity(t *testing.T) {
	store := NewInMemorySemanticStore()
	scope := domain.ForgeScope{WorkspaceID: "ws", LaneID: "memory", SelectedPaths: []string{"artifact:1"}}
	first := MemoryEvidence{EvidenceID: "evidence-1", RootEvidenceID: "evidence-1", Revision: 1, CourtExhibitID: "exhibit-1", CourtRulingID: "ruling-1", Scope: scope}
	if err := store.CreateMemoryEvidence(first, nil); err != nil {
		t.Fatal(err)
	}
	second := MemoryEvidence{EvidenceID: "evidence-2", RootEvidenceID: first.EvidenceID, Revision: 2, CourtExhibitID: "exhibit-2", CourtRulingID: "ruling-2", Scope: scope}
	edge := &MemoryEvidenceSupersession{ID: "edge-2", RootEvidenceID: first.EvidenceID, SupersededEvidenceID: first.EvidenceID, ReplacementEvidenceID: second.EvidenceID, Scope: scope}
	if err := store.CreateMemoryEvidence(second, edge); err != nil {
		t.Fatal(err)
	}
	if got, ok := store.FindMemoryEvidence(first.EvidenceID, scope); !ok || got.EvidenceID != first.EvidenceID || !store.HasMemoryEvidenceSupersession(first.EvidenceID) {
		t.Fatalf("initial immutable evidence/edge missing: ok=%v got=%#v", ok, got)
	}
	third := second
	third.EvidenceID = "evidence-3"
	third.CourtExhibitID = "exhibit-3"
	third.CourtRulingID = "ruling-3"
	conflict := *edge
	conflict.ID = "edge-3"
	conflict.ReplacementEvidenceID = third.EvidenceID
	if err := store.CreateMemoryEvidence(third, &conflict); err == nil {
		t.Fatal("in-memory store accepted a second replacement for one leaf")
	}
}

func memoryEvidenceValidationRequest() domain.SyscallRequest {
	return domain.SyscallRequest{
		ID: "memory-validation", Action: domain.ActionMaterializeAdmittedEvidence,
		Actor: domain.ActorIdentity{ID: "operator", Kind: "user"}, Source: domain.SourceUser,
		Scope:         domain.ForgeScope{WorkspaceID: "ws", LaneID: "memory", SelectedPaths: []string{"artifact:1"}},
		Payload:       map[string]any{"exhibitId": "exhibit", "rulingId": "ruling"},
		Provenance:    domain.Provenance{Actor: "operator", ActorType: "user", Source: "test", TraceID: "trace"},
		CorrelationID: "corr", TraceID: "trace", RequestedAt: 1, IdempotencyKey: "idem-memory-validation",
	}
}

func seedMemoryEvidenceCourt(t *testing.T, store *InMemorySemanticStore, scope domain.ForgeScope, exhibitHash, rulingHash, decision string) {
	t.Helper()
	provenance := domain.Provenance{Actor: "operator", ActorType: "user", Source: "test", TraceID: "trace-admit"}
	exhibit := court.Exhibit{
		ID: "exhibit", CaseID: "case", Scope: scope, SourceType: "artifact", SourceRefs: []string{"artifact:1"},
		ContentSummary: "persisted Court summary", RawRef: "artifact:1", ContentHash: exhibitHash,
		Status: decision, CurrentRulingID: "ruling", CreatedAt: 1, UpdatedAt: 1, Provenance: provenance,
		SyscallID: "admit", CorrelationID: "corr-admit", TraceID: "trace-admit", ProposedBy: "operator", CommittedBy: forgekernel.AuthorityOwnerForgeK,
	}
	ruling := court.Ruling{
		ID: "ruling", CaseID: "case", ExhibitID: "exhibit", Scope: scope, Decision: decision,
		PolicyVersion: court.PolicyVersion, ContentHash: rulingHash, CreatedAt: 1, Provenance: provenance,
		SyscallID: "admit", CorrelationID: "corr-admit", TraceID: "trace-admit", ProposedBy: "operator", CommittedBy: forgekernel.AuthorityOwnerForgeK,
	}
	if err := store.CreateCourtDecision(exhibit, ruling, nil); err != nil {
		t.Fatal(err)
	}
}
