package forgek

import (
	"errors"
	"testing"

	"forge/projectforge/services/core/internal/forgek/court"
)

func grantCourtCapability(t *testing.T, kernel *Kernel, actorID string, syscalls ...string) {
	t.Helper()
	if err := kernel.Capabilities().Grant(Capability{
		CapabilityID:    "cap-court-" + actorID,
		SubjectID:       actorID,
		AllowedSyscalls: syscalls,
		WorkspaceScope:  []string{"workspace-a"},
		MutationScope:   MutationScopeCanonical,
		AuditRequired:   true,
	}); err != nil {
		t.Fatalf("grant court capability: %v", err)
	}
}

func openCourtCase(t *testing.T, kernel *Kernel, actorID string) string {
	t.Helper()
	result := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallCaseOpen,
		ActorID:     actorID,
		WorkspaceID: "workspace-a",
		Input: map[string]any{
			"user_intent": "phase 3 courthouse case",
		},
	})
	if !result.Success {
		t.Fatalf("case.open failed: %v", result.Error)
	}
	return result.ObjectID
}

func submitExhibit(t *testing.T, kernel *Kernel, actorID, caseID, summary string) string {
	t.Helper()
	result := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallCourtSubmit,
		ActorID:     actorID,
		WorkspaceID: "workspace-a",
		Input: map[string]any{
			"case_id":          caseID,
			"source_object_id": "source-" + summary,
			"source_type":      court.SourceTypeManual,
			"source_refs":      []string{"source-ref-" + summary},
			"content_summary":  summary,
		},
	})
	if !result.Success {
		t.Fatalf("court.submit failed: %v", result.Error)
	}
	return result.ObjectID
}

func TestCourthouseSyscallsAreRegistered(t *testing.T) {
	kernel := testKernel()
	for _, name := range []string{
		SyscallCourtSubmit,
		SyscallCourtAdmit,
		SyscallCourtReject,
		SyscallCourtRule,
		SyscallCourtRegisterContradiction,
		SyscallCourtRegisterSupersession,
		SyscallCourtListExhibits,
		SyscallCourtListRulings,
		SyscallCourtListContradictions,
	} {
		if _, ok := kernel.Syscalls().Lookup(name); !ok {
			t.Fatalf("expected courthouse syscall %s to be registered", name)
		}
	}
}

func TestCourtSubmitRequiresCapabilityAndOpenCase(t *testing.T) {
	kernel := testKernel()
	grantCaseCapability(t, kernel, "operator", "workspace-a")
	caseID := openCourtCase(t, kernel, "operator")

	denied := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallCourtSubmit,
		ActorID:     "no-court-capability",
		WorkspaceID: "workspace-a",
		Input: map[string]any{
			"case_id":         caseID,
			"source_type":     court.SourceTypeManual,
			"content_summary": "candidate evidence",
		},
	})
	if denied.Success || !errors.Is(denied.Error, ErrCapabilityDenied) {
		t.Fatalf("expected capability denial, got %#v", denied)
	}

	grantCourtCapability(t, kernel, "operator", SyscallCourtSubmit)
	closeResult := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallCaseClose,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
		Input:       map[string]any{"case_id": caseID},
	})
	if !closeResult.Success {
		t.Fatalf("case.close failed: %v", closeResult.Error)
	}
	closedSubmit := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallCourtSubmit,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
		Input: map[string]any{
			"case_id":         caseID,
			"source_type":     court.SourceTypeManual,
			"content_summary": "late evidence",
		},
	})
	if closedSubmit.Success || !errors.Is(closedSubmit.Error, ErrInvalidStateTransition) {
		t.Fatalf("expected closed case rejection, got %#v", closedSubmit)
	}
}

func TestCourtSubmitAdmitRejectAndCaseReferencesAreJournaled(t *testing.T) {
	kernel := testKernel()
	grantCaseCapability(t, kernel, "operator", "workspace-a")
	grantCourtCapability(t, kernel, "operator", SyscallCourtSubmit, SyscallCourtAdmit, SyscallCourtReject)
	caseID := openCourtCase(t, kernel, "operator")

	admittedID := submitExhibit(t, kernel, "operator", caseID, "admit-me")
	submittedObj, _ := kernel.Objects().GetObject(admittedID)
	if submittedObj.ObjectType != ObjectTypeExhibit || submittedObj.State["admissibility_status"] != court.StatusSubmitted {
		t.Fatalf("unexpected submitted exhibit object: %#v", submittedObj)
	}

	admit := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallCourtAdmit,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
		Input: map[string]any{
			"case_id":          caseID,
			"exhibit_id":       admittedID,
			"admission_reason": "provenance is sufficient",
		},
	})
	if !admit.Success {
		t.Fatalf("court.admit failed: %v", admit.Error)
	}
	admittedObj, _ := kernel.Objects().GetObject(admittedID)
	if admittedObj.State["admissibility_status"] != court.StatusAdmitted {
		t.Fatalf("exhibit was not admitted: %#v", admittedObj.State)
	}

	rejectedID := submitExhibit(t, kernel, "operator", caseID, "reject-me")
	rejectNoReason := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallCourtReject,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
		Input:       map[string]any{"case_id": caseID, "exhibit_id": rejectedID},
	})
	if rejectNoReason.Success || !errors.Is(rejectNoReason.Error, ErrInvalidInput) {
		t.Fatalf("expected rejection reason requirement, got %#v", rejectNoReason)
	}
	reject := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallCourtReject,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
		Input: map[string]any{
			"case_id":          caseID,
			"exhibit_id":       rejectedID,
			"rejection_reason": "source is incomplete",
		},
	})
	if !reject.Success {
		t.Fatalf("court.reject failed: %v", reject.Error)
	}

	caseObj, _ := kernel.Objects().GetObject(caseID)
	if !containsStringState(caseObj.State, "submitted_exhibits", admittedID) ||
		!containsStringState(caseObj.State, "admitted_exhibits", admittedID) ||
		!containsStringState(caseObj.State, "rejected_exhibits", rejectedID) {
		t.Fatalf("case references not updated: %#v", caseObj.State)
	}
	if got := len(kernel.Journal().ListEvents()); got != 5 {
		t.Fatalf("expected case.open plus submit/admit/submit/reject events, got %d", got)
	}
}

func TestCourthouseMutationSyscallsRequireSpecificCapabilities(t *testing.T) {
	kernel := testKernel()
	grantCaseCapability(t, kernel, "operator", "workspace-a")
	grantCourtCapability(t, kernel, "operator", SyscallCourtSubmit)
	caseID := openCourtCase(t, kernel, "operator")
	exhibitA := submitExhibit(t, kernel, "operator", caseID, "a")
	exhibitB := submitExhibit(t, kernel, "operator", caseID, "b")

	attempts := []SyscallRequest{
		{Name: SyscallCourtAdmit, ActorID: "denied", WorkspaceID: "workspace-a", Input: map[string]any{"case_id": caseID, "exhibit_id": exhibitA, "admission_reason": "ok"}},
		{Name: SyscallCourtReject, ActorID: "denied", WorkspaceID: "workspace-a", Input: map[string]any{"case_id": caseID, "exhibit_id": exhibitA, "rejection_reason": "no"}},
		{Name: SyscallCourtRule, ActorID: "denied", WorkspaceID: "workspace-a", Input: map[string]any{"case_id": caseID, "ruling_type": court.RulingAdmission, "admitted_exhibit_refs": []string{exhibitA}}},
		{Name: SyscallCourtRegisterContradiction, ActorID: "denied", WorkspaceID: "workspace-a", Input: map[string]any{"case_id": caseID, "exhibit_a_id": exhibitA, "exhibit_b_id": exhibitB}},
		{Name: SyscallCourtRegisterSupersession, ActorID: "denied", WorkspaceID: "workspace-a", Input: map[string]any{"case_id": caseID, "old_object_id": exhibitA, "new_object_id": exhibitB, "reason": "newer"}},
	}
	for _, attempt := range attempts {
		result := kernel.DispatchSyscall(attempt)
		if result.Success || !errors.Is(result.Error, ErrCapabilityDenied) {
			t.Fatalf("expected capability denial for %s, got %#v", attempt.Name, result)
		}
	}
}

func TestCourtRulingContradictionAndSupersessionAreJournaledAndInspectable(t *testing.T) {
	kernel := testKernel()
	grantCaseCapability(t, kernel, "operator", "workspace-a")
	grantCourtCapability(t, kernel, "operator",
		SyscallCourtSubmit,
		SyscallCourtRule,
		SyscallCourtRegisterContradiction,
		SyscallCourtRegisterSupersession,
	)
	caseID := openCourtCase(t, kernel, "operator")
	exhibitA := submitExhibit(t, kernel, "operator", caseID, "a")
	exhibitB := submitExhibit(t, kernel, "operator", caseID, "b")

	ruling := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallCourtRule,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
		Input: map[string]any{
			"case_id":               caseID,
			"ruling_type":           court.RulingAdmission,
			"admitted_exhibit_refs": []string{exhibitA},
			"rejected_exhibit_refs": []string{exhibitB},
			"reasoning_summary":     "explicit ruling over submitted exhibits",
		},
	})
	if !ruling.Success {
		t.Fatalf("court.rule failed: %v", ruling.Error)
	}

	contradiction := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallCourtRegisterContradiction,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
		Input: map[string]any{
			"case_id":            caseID,
			"exhibit_a_id":       exhibitA,
			"exhibit_b_id":       exhibitB,
			"contradiction_type": "factual_conflict",
			"description":        "evidence conflicts",
			"severity":           "medium",
		},
	})
	if !contradiction.Success {
		t.Fatalf("court.register_contradiction failed: %v", contradiction.Error)
	}

	supersession := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallCourtRegisterSupersession,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
		Input: map[string]any{
			"case_id":       caseID,
			"old_object_id": exhibitA,
			"new_object_id": exhibitB,
			"reason":        "newer evidence replaces older exhibit",
		},
	})
	if !supersession.Success {
		t.Fatalf("court.register_supersession failed: %v", supersession.Error)
	}

	oldExhibit, oldOK := kernel.Objects().GetObject(exhibitA)
	newExhibit, newOK := kernel.Objects().GetObject(exhibitB)
	if !oldOK || !newOK {
		t.Fatal("supersession deleted exhibit")
	}
	if oldExhibit.State["admissibility_status"] != court.StatusSuperseded {
		t.Fatalf("old exhibit not marked superseded: %#v", oldExhibit.State)
	}
	if newExhibit.State["admissibility_status"] == nil {
		t.Fatalf("new exhibit not inspectable: %#v", newExhibit.State)
	}

	caseObj, _ := kernel.Objects().GetObject(caseID)
	if !containsStringState(caseObj.State, "ruling_refs", ruling.ObjectID) ||
		!containsStringState(caseObj.State, "contradiction_refs", contradiction.ObjectID) ||
		!containsStringState(caseObj.State, "supersession_refs", supersession.ObjectID) {
		t.Fatalf("case references missing: %#v", caseObj.State)
	}
}

func TestNeuronEnvelopeCanBeSubmittedButIsNotAutomaticallyAdmitted(t *testing.T) {
	kernel := testKernel()
	grantCaseCapability(t, kernel, "operator", "workspace-a")
	grantCourtCapability(t, kernel, "operator", SyscallCourtSubmit, SyscallCourtAdmit)
	caseID := openCourtCase(t, kernel, "operator")

	if len(kernel.Journal().ListEvents()) != 1 {
		t.Fatal("creating a candidate envelope reference should not journal or submit evidence")
	}

	submit := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallCourtSubmit,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
		Input: map[string]any{
			"case_id":          caseID,
			"source_object_id": "env-1",
			"source_type":      court.SourceTypeNeuronEnvelope,
			"source_refs":      []string{"env-1"},
			"content_summary":  "proposal summary",
		},
	})
	if !submit.Success {
		t.Fatalf("court.submit neuron envelope failed: %v", submit.Error)
	}
	exhibitObj, _ := kernel.Objects().GetObject(submit.ObjectID)
	if exhibitObj.State["admissibility_status"] != court.StatusSubmitted {
		t.Fatalf("neuron exhibit automatically admitted: %#v", exhibitObj.State)
	}
}

func containsStringState(state map[string]any, key, value string) bool {
	values, ok := state[key].([]string)
	if !ok {
		return false
	}
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
