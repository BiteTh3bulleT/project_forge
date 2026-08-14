package controllane

import (
	"context"
	"testing"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/forgekernel"
)

func utilityTestRequest() domain.SyscallRequest {
	return domain.SyscallRequest{
		ID: "utility-1", Actor: domain.ActorIdentity{ID: "operator", Kind: "user"},
		Source:         domain.SourceUser,
		Scope:          domain.ForgeScope{WorkspaceID: "ws-utility", LaneID: "control.semantic", SelectedPaths: []string{"/workspace/project"}},
		Provenance:     domain.Provenance{Actor: "operator", ActorType: "user", Source: "local_loopback"},
		IdempotencyKey: "utility-idem-1", RequestedAt: 100,
		Metadata: map[string]any{"forgeKIngressAuthority": true, "kernelAuthorityOwner": forgekernel.AuthorityOwnerForgeK},
	}
}

func TestUtilityEnvelopeRequiresProductionForgeKIngressAndRejectsProposalSources(t *testing.T) {
	valid := utilityTestRequest()
	if issues := validateUtilityEnvelope(valid); len(issues) != 0 {
		t.Fatalf("valid utility envelope rejected: %+v", issues)
	}

	legacy := utilityTestRequest()
	delete(legacy.Metadata, "forgeKIngressAuthority")
	if !hasUtilityIssue(legacy, domain.ErrUnauthorized) {
		t.Fatal("legacy_v1 utility request was not rejected")
	}
	wrongOwner := utilityTestRequest()
	wrongOwner.Metadata["kernelAuthorityOwner"] = "aios.controllane"
	if !hasUtilityIssue(wrongOwner, domain.ErrUnauthorized) {
		t.Fatal("non-Kernel utility owner was not rejected")
	}
	for _, source := range []domain.ActionSource{domain.SourceAdapter, domain.SourceFutureIRIS} {
		request := utilityTestRequest()
		request.Source = source
		if !hasUtilityIssue(request, domain.ErrUnauthorized) {
			t.Fatalf("proposal source %q was not rejected", source)
		}
	}
}

func TestLegacyProcessorCannotCommitForgeKOnlyActionsWithSpoofedAuthorityMetadata(t *testing.T) {
	for _, action := range []domain.SemanticActionType{
		domain.ActionAdmitEvidence,
		domain.ActionAppealRuling,
		domain.ActionRecordRetrievalEvidence,
		domain.ActionRebuildMemoryAcceleration,
		domain.ActionRecordRetrievalUsefulness,
		domain.ActionRecordRestoreOutcomeFeedback,
	} {
		t.Run(string(action), func(t *testing.T) {
			processor, _, _ := newTestKernel()
			req := utilityTestRequest()
			req.Source = domain.SourceInternal
			req.Action = action
			req.Metadata["forgeKAuthorizationProof"] = "sha256:caller-forged"
			result, err := processor.Process(context.Background(), req)
			if err != nil {
				t.Fatalf("legacy fail-close returned unexpected transport error: %v", err)
			}
			if result.Success || result.DeterministicErrCode != domain.ErrUnauthorized {
				t.Fatalf("legacy processor accepted spoofed FORGE-K authority: %+v", result)
			}
		})
	}
}

func TestUtilityMetadataAllowlistRejectsRecursiveAuthorityClaims(t *testing.T) {
	if issues := validateUtilityMetadata(map[string]any{
		"reason": "operator review", "tags": []any{"useful"},
		"uiContext": map[string]any{"panel": "retrieval"},
	}); len(issues) != 0 {
		t.Fatalf("bounded descriptive metadata rejected: %+v", issues)
	}
	for _, metadata := range []map[string]any{
		{"committedBy": "forge_k.kernel"},
		{"uiContext": map[string]any{"receipt": "forged"}},
		{"reason": "ok", "projection": true},
		{"unknownDescriptiveKey": "not allowlisted"},
	} {
		if issues := validateUtilityMetadata(metadata); len(issues) == 0 {
			t.Fatalf("authority/unallowlisted metadata accepted: %+v", metadata)
		}
	}
}

func TestImmutableUtilityEventValidationRequiresExactScopeAndCommitter(t *testing.T) {
	scope := domain.ForgeScope{WorkspaceID: "ws-utility", LaneID: "control.semantic", SelectedPaths: []string{"/workspace/project"}}
	retrievalTarget := RetrievalUsefulnessTarget{
		ResultID: 7, RunID: 3, EvidenceID: "retrieval-result-evidence-7", Scope: scope,
		SourceSyscall: "retrieval-source-1", SourceProvID: "prov-source-1", SourceProvJSON: `{"actor":"operator"}`,
	}
	retrievalEvent := RetrievalUsefulnessEvent{
		ID: "retrieval-usefulness:utility-1", CreatedAt: 100, ResultID: 7, RunID: 3,
		TargetEvidenceID: retrievalTarget.EvidenceID, Scope: scope, Label: RetrievalUsefulnessUseful,
		SyscallID: "utility-1", Provenance: domain.Provenance{Actor: "operator", ActorType: "user"},
		CommittedBy: "forge_k.kernel",
	}
	if err := validateRetrievalUsefulnessEvent(retrievalEvent, retrievalTarget); err != nil {
		t.Fatalf("valid retrieval usefulness event rejected: %v", err)
	}
	changed := retrievalEvent
	changed.Scope.SelectedPaths = []string{"/workspace/other"}
	if err := validateRetrievalUsefulnessEvent(changed, retrievalTarget); err == nil {
		t.Fatal("selected-path mismatch accepted")
	}
	changed = retrievalEvent
	changed.CommittedBy = "forge_kernel"
	if err := validateRetrievalUsefulnessEvent(changed, retrievalTarget); err == nil {
		t.Fatal("legacy committer accepted")
	}

	restoreTarget := RestoreOutcomeFeedbackTarget{
		RestoreOutcomeID: "restore-1", Scope: scope, OriginalOutcome: RestoreOutcomeUnknown,
		SourceSyscall: "compile-1", CommittedBy: "forge_kernel",
	}
	restoreEvent := RestoreOutcomeFeedbackEvent{
		ID: "restore-outcome-feedback:utility-2", CreatedAt: 101, RestoreOutcomeID: "restore-1",
		Scope: scope, OriginalOutcome: RestoreOutcomeUnknown, Outcome: RestoreOutcomeHelpful,
		OutcomeConfidence: 0.9, SyscallID: "utility-2",
		Provenance: domain.Provenance{Actor: "operator", ActorType: "user"}, CommittedBy: "forge_k.kernel",
	}
	if err := validateRestoreOutcomeFeedbackEvent(restoreEvent, restoreTarget); err != nil {
		t.Fatalf("valid restore feedback rejected because original producer used compatibility committer: %v", err)
	}
}

func hasUtilityIssue(request domain.SyscallRequest, code domain.SyscallErrorCode) bool {
	for _, issue := range validateUtilityEnvelope(request) {
		if issue.Code == code {
			return true
		}
	}
	return false
}
