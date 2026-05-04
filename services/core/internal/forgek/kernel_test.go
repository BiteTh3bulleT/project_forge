package forgek

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

func testKernel() *Kernel {
	ids := NewSequenceIDProvider(map[string]int{
		"cap":    0,
		"case":   0,
		"event":  0,
		"object": 0,
	})
	clock := NewFixedClock(time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC))
	return NewKernel(KernelOptions{IDs: ids, Clock: clock})
}

func grantCaseCapability(t *testing.T, kernel *Kernel, actorID, workspaceID string) {
	t.Helper()

	capability := Capability{
		CapabilityID:      "cap-case",
		SubjectID:         actorID,
		AllowedSyscalls:   []string{SyscallCaseOpen, SyscallCaseUpdate, SyscallCaseClose},
		WorkspaceScope:    []string{workspaceID},
		MutationScope:     MutationScopeCanonical,
		DelegationAllowed: false,
		AuditRequired:     true,
	}
	if err := kernel.Capabilities().Grant(capability); err != nil {
		t.Fatalf("grant capability: %v", err)
	}
}

func TestKernelInitializesWithCoreComponentsAndSyscalls(t *testing.T) {
	kernel := testKernel()

	if kernel.Objects() == nil {
		t.Fatal("object registry missing")
	}
	if kernel.Syscalls() == nil {
		t.Fatal("syscall registry missing")
	}
	if kernel.Capabilities() == nil {
		t.Fatal("capability manager missing")
	}
	if kernel.Journal() == nil {
		t.Fatal("journal missing")
	}
	for _, name := range []string{SyscallCaseOpen, SyscallCaseUpdate, SyscallCaseClose, SyscallObjectGet, SyscallObjectList, SyscallCapabilityGrant} {
		if _, ok := kernel.Syscalls().Lookup(name); !ok {
			t.Fatalf("expected syscall %q to be registered", name)
		}
	}
}

func TestUnknownSyscallFails(t *testing.T) {
	kernel := testKernel()

	result := kernel.DispatchSyscall(SyscallRequest{
		Name:        "unknown.syscall",
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
	})

	if result.Success {
		t.Fatal("unknown syscall succeeded")
	}
	if !errors.Is(result.Error, ErrUnknownSyscall) {
		t.Fatalf("expected ErrUnknownSyscall, got %v", result.Error)
	}
	if len(kernel.Journal().ListEvents()) != 0 {
		t.Fatal("unknown syscall should not journal a transition")
	}
}

func TestMutatingSyscallMarkedJournalRequiredMustAppendJournalEvent(t *testing.T) {
	kernel := testKernel()
	const syscallName = "test.mutate_without_journal"
	if err := kernel.Syscalls().Register(SyscallDefinition{
		Name:            syscallName,
		Version:         "v1",
		SideEffects:     true,
		JournalRequired: true,
		Deterministic:   true,
		Replayable:      true,
		Handler: func(_ *Kernel, request SyscallRequest, _ []string) SyscallResult {
			return SyscallResult{Success: true, SyscallName: request.Name, Output: "mutated"}
		},
	}); err != nil {
		t.Fatalf("register test syscall: %v", err)
	}
	if err := kernel.Capabilities().Grant(Capability{
		CapabilityID:    "cap-test",
		SubjectID:       "operator",
		AllowedSyscalls: []string{syscallName},
		WorkspaceScope:  []string{"workspace-a"},
		MutationScope:   MutationScopeCanonical,
	}); err != nil {
		t.Fatalf("grant test capability: %v", err)
	}

	result := kernel.DispatchSyscall(SyscallRequest{
		Name:        syscallName,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
	})

	if result.Success {
		t.Fatal("journal-required mutating syscall succeeded without journaling")
	}
	if !errors.Is(result.Error, ErrJournalRequired) {
		t.Fatalf("expected ErrJournalRequired, got %v", result.Error)
	}
}

func TestActorWithoutCapabilityCannotMutateState(t *testing.T) {
	kernel := testKernel()

	result := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallCaseOpen,
		ActorID:     "model-driver",
		WorkspaceID: "workspace-a",
		Input: map[string]any{
			"user_intent": "propose a case",
		},
	})

	if result.Success {
		t.Fatal("case.open succeeded without capability")
	}
	if !errors.Is(result.Error, ErrCapabilityDenied) {
		t.Fatalf("expected ErrCapabilityDenied, got %v", result.Error)
	}
	if got := len(kernel.Objects().ListObjects()); got != 0 {
		t.Fatalf("expected no objects, got %d", got)
	}
	if got := len(kernel.Journal().ListEvents()); got != 0 {
		t.Fatalf("expected no journal events, got %d", got)
	}
}

func TestActorWithCapabilityCanOpenCaseAndJournalTransition(t *testing.T) {
	kernel := testKernel()
	grantCaseCapability(t, kernel, "operator", "workspace-a")

	result := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallCaseOpen,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
		Input: map[string]any{
			"user_intent": "establish kernel simulator",
			"summary":     "Phase 1 authority skeleton",
		},
	})

	if !result.Success {
		t.Fatalf("case.open failed: %v", result.Error)
	}
	if result.ObjectID != "case-0001" {
		t.Fatalf("unexpected case id %q", result.ObjectID)
	}

	obj, ok := kernel.Objects().GetObject(result.ObjectID)
	if !ok {
		t.Fatal("case object missing")
	}
	if obj.ObjectType != ObjectTypeCasePacket || obj.AuthorityLevel != AuthorityCommitted {
		t.Fatalf("unexpected object authority: %#v", obj)
	}

	casePacket, ok := result.Output.(CasePacket)
	if !ok {
		t.Fatalf("expected CasePacket output, got %T", result.Output)
	}
	if casePacket.Status != CaseStatusOpen {
		t.Fatalf("expected open case, got %s", casePacket.Status)
	}
	if !reflect.DeepEqual(casePacket.JournalRefs, []string{"event-0001"}) {
		t.Fatalf("unexpected case journal refs: %#v", casePacket.JournalRefs)
	}

	events := kernel.Journal().ListEvents()
	if len(events) != 1 {
		t.Fatalf("expected one journal event, got %d", len(events))
	}
	if events[0].EventType != JournalEventCaseOpened || events[0].SyscallName != SyscallCaseOpen {
		t.Fatalf("unexpected journal event: %#v", events[0])
	}
	if events[0].PriorHash != "" || events[0].EventHash == "" {
		t.Fatalf("expected first hash chain event, got prior=%q hash=%q", events[0].PriorHash, events[0].EventHash)
	}
}

func TestCapabilityScopeAndExpirationAreEnforced(t *testing.T) {
	kernel := testKernel()
	expiredAt := time.Date(2026, 5, 3, 11, 0, 0, 0, time.UTC)
	if err := kernel.Capabilities().Grant(Capability{
		CapabilityID:    "cap-expired",
		SubjectID:       "operator",
		AllowedSyscalls: []string{SyscallCaseOpen},
		WorkspaceScope:  []string{"workspace-a"},
		MutationScope:   MutationScopeCanonical,
		Expiration:      &expiredAt,
	}); err != nil {
		t.Fatalf("grant expired capability: %v", err)
	}
	if err := kernel.Capabilities().Grant(Capability{
		CapabilityID:    "cap-other-workspace",
		SubjectID:       "operator",
		AllowedSyscalls: []string{SyscallCaseOpen},
		WorkspaceScope:  []string{"workspace-b"},
		MutationScope:   MutationScopeCanonical,
	}); err != nil {
		t.Fatalf("grant scoped capability: %v", err)
	}

	result := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallCaseOpen,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
		Input: map[string]any{
			"user_intent": "should be denied",
		},
	})

	if result.Success {
		t.Fatal("case.open succeeded with expired or wrong-scope capability")
	}
	if !errors.Is(result.Error, ErrCapabilityDenied) {
		t.Fatalf("expected ErrCapabilityDenied, got %v", result.Error)
	}
}

func TestObjectReadSyscallsReturnCopiesWithoutJournalMutation(t *testing.T) {
	kernel := testKernel()
	grantCaseCapability(t, kernel, "operator", "workspace-a")
	open := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallCaseOpen,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
		Input: map[string]any{
			"user_intent": "read objects",
		},
	})
	if !open.Success {
		t.Fatalf("case.open failed: %v", open.Error)
	}

	get := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallObjectGet,
		ActorID:     "reader",
		WorkspaceID: "workspace-a",
		Input: map[string]any{
			"object_id": open.ObjectID,
		},
	})
	if !get.Success {
		t.Fatalf("object.get failed: %v", get.Error)
	}
	obj := get.Output.(KernelObject)
	obj.State["status"] = "tampered"

	list := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallObjectList,
		ActorID:     "reader",
		WorkspaceID: "workspace-a",
	})
	if !list.Success {
		t.Fatalf("object.list failed: %v", list.Error)
	}
	objects := list.Output.([]KernelObject)
	if len(objects) != 1 {
		t.Fatalf("expected one listed object, got %d", len(objects))
	}
	if objects[0].State["status"] == "tampered" {
		t.Fatal("object.get returned mutable registry state")
	}
	if got := len(kernel.Journal().ListEvents()); got != 1 {
		t.Fatalf("read syscalls should not append journal events, got %d", got)
	}
}

func TestCaseUpdateAndCloseOnlyThroughSyscalls(t *testing.T) {
	kernel := testKernel()
	grantCaseCapability(t, kernel, "operator", "workspace-a")

	open := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallCaseOpen,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
		Input: map[string]any{
			"user_intent": "track kernel work",
		},
	})
	if !open.Success {
		t.Fatalf("case.open failed: %v", open.Error)
	}

	update := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallCaseUpdate,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
		Input: map[string]any{
			"case_id":     open.ObjectID,
			"summary":     "updated only through syscall",
			"object_refs": []string{"obj-a", "obj-b"},
		},
	})
	if !update.Success {
		t.Fatalf("case.update failed: %v", update.Error)
	}
	updatedCase := update.Output.(CasePacket)
	if updatedCase.Status != CaseStatusUpdated {
		t.Fatalf("expected updated status, got %s", updatedCase.Status)
	}
	if !reflect.DeepEqual(updatedCase.ObjectRefs, []string{"obj-a", "obj-b"}) {
		t.Fatalf("unexpected object refs: %#v", updatedCase.ObjectRefs)
	}
	if !reflect.DeepEqual(updatedCase.JournalRefs, []string{"event-0001", "event-0002"}) {
		t.Fatalf("unexpected journal refs after update: %#v", updatedCase.JournalRefs)
	}

	closeResult := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallCaseClose,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
		Input: map[string]any{
			"case_id": open.ObjectID,
		},
	})
	if !closeResult.Success {
		t.Fatalf("case.close failed: %v", closeResult.Error)
	}
	closedCase := closeResult.Output.(CasePacket)
	if closedCase.Status != CaseStatusClosed || closedCase.ClosedAt == nil {
		t.Fatalf("expected closed case, got %#v", closedCase)
	}

	blockedUpdate := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallCaseUpdate,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
		Input: map[string]any{
			"case_id": open.ObjectID,
			"summary": "should fail",
		},
	})
	if blockedUpdate.Success {
		t.Fatal("closed case update succeeded")
	}
	if !errors.Is(blockedUpdate.Error, ErrInvalidStateTransition) {
		t.Fatalf("expected ErrInvalidStateTransition, got %v", blockedUpdate.Error)
	}

	if got := len(kernel.Journal().ListEvents()); got != 3 {
		t.Fatalf("expected three journaled transitions, got %d", got)
	}
}

func TestPublicObjectRegistryCopiesPreventDirectMutation(t *testing.T) {
	kernel := testKernel()
	grantCaseCapability(t, kernel, "operator", "workspace-a")

	result := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallCaseOpen,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
		Input: map[string]any{
			"user_intent": "protect registry",
		},
	})
	if !result.Success {
		t.Fatalf("case.open failed: %v", result.Error)
	}

	obj, ok := kernel.Objects().GetObject(result.ObjectID)
	if !ok {
		t.Fatal("object missing")
	}
	obj.State["status"] = "tampered"
	obj.JournalRefs = append(obj.JournalRefs, "fake-event")

	again, _ := kernel.Objects().GetObject(result.ObjectID)
	if again.State["status"] == "tampered" {
		t.Fatal("public object copy mutated registry state")
	}
	if reflect.DeepEqual(again.JournalRefs, obj.JournalRefs) {
		t.Fatal("public journal refs mutation affected registry state")
	}

	listed := kernel.Objects().ListObjects()
	listed[0].State["status"] = "tampered-list"
	fromRegistry, _ := kernel.Objects().GetObject(result.ObjectID)
	if fromRegistry.State["status"] == "tampered-list" {
		t.Fatal("public list result mutated registry state")
	}
}

func TestJournalIsAppendOnlyThroughPublicAPI(t *testing.T) {
	kernel := testKernel()
	grantCaseCapability(t, kernel, "operator", "workspace-a")

	open := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallCaseOpen,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
		Input: map[string]any{
			"user_intent": "protect journal",
		},
	})
	if !open.Success {
		t.Fatalf("case.open failed: %v", open.Error)
	}

	events := kernel.Journal().ListEvents()
	events[0].Result = "tampered"
	events = append(events[:0], events[1:]...)

	event, ok := kernel.Journal().GetEvent("event-0001")
	if !ok {
		t.Fatal("event missing after public slice mutation")
	}
	if event.Result != SyscallResultCommitted {
		t.Fatalf("journal event was mutated through public API: %#v", event)
	}
	if got := len(kernel.Journal().ListEvents()); got != 1 {
		t.Fatalf("journal deletion through public API changed event count to %d", got)
	}
}

func TestDirectNeuralProposalHasNoCanonicalAuthority(t *testing.T) {
	kernel := testKernel()

	proposal := NeuralProposal{
		ProposalID:  "proposal-1",
		ActorID:     "model-driver",
		WorkspaceID: "workspace-a",
		Summary:     "open a canonical case",
	}

	result := kernel.SubmitNeuralProposal(proposal)
	if result.Success {
		t.Fatal("neural proposal received canonical authority")
	}
	if !errors.Is(result.Error, ErrProposalOnly) {
		t.Fatalf("expected ErrProposalOnly, got %v", result.Error)
	}
	if len(kernel.Objects().ListObjects()) != 0 {
		t.Fatal("neural proposal mutated object registry")
	}
	if len(kernel.Journal().ListEvents()) != 0 {
		t.Fatal("neural proposal appended journal event")
	}
}
