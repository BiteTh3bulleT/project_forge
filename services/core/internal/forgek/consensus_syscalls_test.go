package forgek

import (
	"errors"
	"testing"

	"forge/projectforge/services/core/internal/forgek/consensus"
	"forge/projectforge/services/core/internal/forgek/contextcompiler"
)

func grantConsensusCapability(t *testing.T, kernel *Kernel, actorID, workspaceID, mutationScope string, syscalls ...string) {
	t.Helper()
	if err := kernel.Capabilities().Grant(Capability{
		CapabilityID:    "cap-consensus-" + actorID + "-" + workspaceID,
		SubjectID:       actorID,
		AllowedSyscalls: syscalls,
		WorkspaceScope:  []string{workspaceID},
		MutationScope:   mutationScope,
		AuditRequired:   true,
	}); err != nil {
		t.Fatalf("grant consensus capability: %v", err)
	}
}

func consensusOpen(t *testing.T, kernel *Kernel, actorID string) consensus.ConsensusRequest {
	t.Helper()
	result := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallConsensusOpen,
		ActorID:     actorID,
		WorkspaceID: "workspace-a",
		Input: map[string]any{
			"request_id":  "request-a",
			"policy_id":   "policy-a",
			"criticality": "low",
		},
	})
	if !result.Success {
		t.Fatalf("consensus.open failed: %#v", result)
	}
	return result.Output.(consensus.ConsensusRequest)
}

func consensusEvidenceInput(requestID, evidenceID string, tier consensus.EvidenceTier) map[string]any {
	evidenceType := string(consensus.EvidenceSourceCode)
	if tier == consensus.EvidenceTier3 {
		evidenceType = string(consensus.EvidenceModelInference)
	}
	return map[string]any{
		"request_id":        requestID,
		"evidence_id":       evidenceID,
		"evidence_type":     evidenceType,
		"tier":              string(tier),
		"source":            "repo",
		"locator":           "file.go:12",
		"freshness_score":   1.0,
		"reliability_score": 1.0,
		"source_hash":       "hash-a",
	}
}

func consensusClaimInput(requestID, claimID, evidenceID string, value any) map[string]any {
	return map[string]any{
		"request_id":    requestID,
		"claim_id":      claimID,
		"claim_type":    string(consensus.ClaimTypeFact),
		"subject":       "build",
		"predicate":     "status",
		"value_json":    value,
		"scope":         "workspace-a",
		"temporal":      "2026-05-05",
		"evidence_refs": []string{evidenceID},
		"confidence":    0.95,
		"agent_id":      "agent-a",
	}
}

func TestConsensusSyscallsAreRegistered(t *testing.T) {
	kernel := testKernel()
	for _, name := range []string{
		SyscallConsensusOpen,
		SyscallConsensusSubmitClaim,
		SyscallConsensusSubmitEvidence,
		SyscallConsensusEvaluate,
		SyscallConsensusGetReport,
		SyscallConsensusListReports,
		SyscallConsensusBuildComposerInput,
		SyscallConsensusRead,
	} {
		if _, ok := kernel.Syscalls().Lookup(name); !ok {
			t.Fatalf("expected consensus syscall %s to be registered", name)
		}
	}
	if kernel.Consensus() == nil {
		t.Fatal("kernel does not own consensus service")
	}
}

func TestConsensusMutatingSyscallsRequireCapabilityAndJournal(t *testing.T) {
	kernel := testKernel()
	denied := kernel.DispatchSyscall(SyscallRequest{Name: SyscallConsensusOpen, ActorID: "denied", WorkspaceID: "workspace-a"})
	if denied.Success || !errors.Is(denied.Error, ErrCapabilityDenied) {
		t.Fatalf("expected consensus.open denial, got %#v", denied)
	}
	grantConsensusCapability(t, kernel, "reader", "workspace-a", MutationScopeNone, SyscallConsensusRead)
	readOnlyDenied := kernel.DispatchSyscall(SyscallRequest{Name: SyscallConsensusOpen, ActorID: "reader", WorkspaceID: "workspace-a"})
	if readOnlyDenied.Success || !errors.Is(readOnlyDenied.Error, ErrCapabilityDenied) {
		t.Fatalf("expected read-only mutation denial, got %#v", readOnlyDenied)
	}
	grantConsensusCapability(t, kernel, "operator", "workspace-a", MutationScopeCanonical,
		SyscallConsensusOpen, SyscallConsensusSubmitEvidence, SyscallConsensusSubmitClaim, SyscallConsensusEvaluate)
	opened := consensusOpen(t, kernel, "operator")
	evidence := kernel.DispatchSyscall(SyscallRequest{Name: SyscallConsensusSubmitEvidence, ActorID: "operator", WorkspaceID: "workspace-a", Input: consensusEvidenceInput(opened.RequestID, "evidence-a", consensus.EvidenceTier1)})
	if !evidence.Success || evidence.JournalEvent == "" {
		t.Fatalf("consensus.submit_evidence failed or did not journal: %#v", evidence)
	}
	claim := kernel.DispatchSyscall(SyscallRequest{Name: SyscallConsensusSubmitClaim, ActorID: "operator", WorkspaceID: "workspace-a", Input: consensusClaimInput(opened.RequestID, "claim-a", "evidence-a", "green")})
	if !claim.Success || claim.JournalEvent == "" {
		t.Fatalf("consensus.submit_claim failed or did not journal: %#v", claim)
	}
	evaluate := kernel.DispatchSyscall(SyscallRequest{Name: SyscallConsensusEvaluate, ActorID: "operator", WorkspaceID: "workspace-a", Input: map[string]any{"request_id": opened.RequestID, "report_id": "report-a"}})
	if !evaluate.Success || evaluate.JournalEvent == "" {
		t.Fatalf("consensus.evaluate failed or did not journal: %#v", evaluate)
	}
	report := evaluate.Output.(consensus.ConsensusReport)
	if len(report.AcceptedClaimIDs) != 1 || report.IsCanonicalTruth() || report.IsEvidenceAdmitted() {
		t.Fatalf("unexpected report boundary: %#v", report)
	}
	for _, eventType := range []string{JournalEventConsensusOpened, JournalEventConsensusEvidenceSubmitted, JournalEventConsensusClaimSubmitted, JournalEventConsensusEvaluated} {
		if !hasJournalEvent(kernel.Journal().ListEvents(), eventType) {
			t.Fatalf("missing consensus journal event %s", eventType)
		}
	}
}

func TestConsensusReadSyscallsRequireReadCapabilityAndDoNotJournal(t *testing.T) {
	kernel := testKernel()
	grantConsensusCapability(t, kernel, "operator", "workspace-a", MutationScopeCanonical,
		SyscallConsensusOpen, SyscallConsensusSubmitEvidence, SyscallConsensusSubmitClaim, SyscallConsensusEvaluate)
	opened := consensusOpen(t, kernel, "operator")
	kernel.DispatchSyscall(SyscallRequest{Name: SyscallConsensusSubmitEvidence, ActorID: "operator", WorkspaceID: "workspace-a", Input: consensusEvidenceInput(opened.RequestID, "evidence-a", consensus.EvidenceTier1)})
	kernel.DispatchSyscall(SyscallRequest{Name: SyscallConsensusSubmitClaim, ActorID: "operator", WorkspaceID: "workspace-a", Input: consensusClaimInput(opened.RequestID, "claim-a", "evidence-a", "green")})
	evaluate := kernel.DispatchSyscall(SyscallRequest{Name: SyscallConsensusEvaluate, ActorID: "operator", WorkspaceID: "workspace-a", Input: map[string]any{"request_id": opened.RequestID, "report_id": "report-a"}})
	report := evaluate.Output.(consensus.ConsensusReport)
	beforeJournal := len(kernel.Journal().ListEvents())
	beforeObjects := len(kernel.Objects().ListObjects())

	withoutRead := kernel.DispatchSyscall(SyscallRequest{Name: SyscallConsensusGetReport, ActorID: "reader", WorkspaceID: "workspace-a", Input: map[string]any{"report_id": report.ReportID}})
	if withoutRead.Success || !errors.Is(withoutRead.Error, ErrCapabilityDenied) {
		t.Fatalf("expected read denial, got %#v", withoutRead)
	}
	grantConsensusCapability(t, kernel, "reader", "workspace-a", MutationScopeNone, SyscallConsensusRead)
	getReport := kernel.DispatchSyscall(SyscallRequest{Name: SyscallConsensusGetReport, ActorID: "reader", WorkspaceID: "workspace-a", Input: map[string]any{"report_id": report.ReportID}})
	listReports := kernel.DispatchSyscall(SyscallRequest{Name: SyscallConsensusListReports, ActorID: "reader", WorkspaceID: "workspace-a"})
	composer := kernel.DispatchSyscall(SyscallRequest{Name: SyscallConsensusBuildComposerInput, ActorID: "reader", WorkspaceID: "workspace-a", Input: map[string]any{"report_id": report.ReportID, "input_id": "composer-a"}})
	readAll := kernel.DispatchSyscall(SyscallRequest{Name: SyscallConsensusRead, ActorID: "reader", WorkspaceID: "workspace-a", Input: map[string]any{"request_id": opened.RequestID}})
	if !getReport.Success || !listReports.Success || !composer.Success || !readAll.Success {
		t.Fatalf("read syscalls failed: get=%#v list=%#v composer=%#v read=%#v", getReport, listReports, composer, readAll)
	}
	payload := composer.Output.(consensus.ResponseCompositionInput)
	if len(payload.AcceptedClaims) != 1 || len(payload.UncertainClaims) != 0 {
		t.Fatalf("composer payload did not enforce accepted-claims-only: %#v", payload)
	}
	if len(kernel.Journal().ListEvents()) != beforeJournal || len(kernel.Objects().ListObjects()) != beforeObjects {
		t.Fatal("read-only consensus syscalls journaled or mutated kernel objects")
	}
}

func TestConsensusWorkspaceScopeAndNoSecondAuthority(t *testing.T) {
	kernel := testKernel()
	grantCaseCapability(t, kernel, "case-operator", "workspace-a")
	openCase := kernel.DispatchSyscall(SyscallRequest{Name: SyscallCaseOpen, ActorID: "case-operator", WorkspaceID: "workspace-a", Input: map[string]any{"user_intent": "consensus", "summary": "consensus"}})
	if !openCase.Success {
		t.Fatalf("case.open failed: %#v", openCase)
	}
	beforeCase := openCase.Output.(CasePacket)
	grantConsensusCapability(t, kernel, "operator", "workspace-a", MutationScopeCanonical,
		SyscallConsensusOpen, SyscallConsensusSubmitEvidence, SyscallConsensusSubmitClaim, SyscallConsensusEvaluate)
	grantConsensusCapability(t, kernel, "wrong-scope", "workspace-b", MutationScopeCanonical, SyscallConsensusEvaluate)
	opened := consensusOpen(t, kernel, "operator")
	wrongScope := kernel.DispatchSyscall(SyscallRequest{Name: SyscallConsensusEvaluate, ActorID: "wrong-scope", WorkspaceID: "workspace-a", Input: map[string]any{"request_id": opened.RequestID}})
	if wrongScope.Success || !errors.Is(wrongScope.Error, ErrCapabilityDenied) {
		t.Fatalf("expected workspace scope denial, got %#v", wrongScope)
	}
	kernel.DispatchSyscall(SyscallRequest{Name: SyscallConsensusSubmitEvidence, ActorID: "operator", WorkspaceID: "workspace-a", Input: consensusEvidenceInput(opened.RequestID, "evidence-a", consensus.EvidenceTier1)})
	kernel.DispatchSyscall(SyscallRequest{Name: SyscallConsensusSubmitClaim, ActorID: "operator", WorkspaceID: "workspace-a", Input: consensusClaimInput(opened.RequestID, "claim-a", "evidence-a", "green")})
	result := kernel.DispatchSyscall(SyscallRequest{Name: SyscallConsensusEvaluate, ActorID: "operator", WorkspaceID: "workspace-a", Input: map[string]any{"request_id": opened.RequestID, "report_id": "report-a"}})
	if !result.Success {
		t.Fatalf("consensus.evaluate failed: %#v", result)
	}
	afterCase, ok := kernel.objects.getCase(beforeCase.CaseID)
	if !ok || afterCase.Status != beforeCase.Status || len(afterCase.AdmittedExhibitRefs) != len(beforeCase.AdmittedExhibitRefs) || len(afterCase.SubmittedExhibitRefs) != len(beforeCase.SubmittedExhibitRefs) {
		t.Fatalf("consensus mutated case/court state: before=%#v after=%#v ok=%v", beforeCase, afterCase, ok)
	}
	if len(kernel.ContextCompiler().ListBlocks(contextcompiler.BlockListFilter{WorkspaceID: "workspace-a"})) != 0 {
		t.Fatal("consensus created context blocks")
	}
	if len(kernel.Runtime().ListResults("workspace-a")) != 0 {
		t.Fatal("consensus called or stored runtime results")
	}
	obj, ok := kernel.Objects().GetObject("report-a")
	if !ok || obj.ObjectType != ObjectTypeConsensusReport || obj.State["is_canonical_truth"] != false || obj.State["is_admitted_evidence"] != false || obj.State["executes_action"] != false {
		t.Fatalf("consensus report object claimed forbidden authority: %#v ok=%v", obj, ok)
	}
}
