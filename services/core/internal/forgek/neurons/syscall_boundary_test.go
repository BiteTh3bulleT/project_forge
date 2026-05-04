package neurons

import (
	"errors"
	"testing"
	"time"

	"forge/projectforge/services/core/internal/forgek"
)

func testKernel() *forgek.Kernel {
	ids := forgek.NewSequenceIDProvider(map[string]int{"cap": 0, "case": 0, "event": 0})
	clock := forgek.NewFixedClock(time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC))
	return forgek.NewKernel(forgek.KernelOptions{IDs: ids, Clock: clock})
}

func grantCaseCapability(t *testing.T, kernel *forgek.Kernel, actorID string) {
	t.Helper()
	if err := kernel.Capabilities().Grant(forgek.Capability{
		CapabilityID:    "cap-" + actorID,
		SubjectID:       actorID,
		AllowedSyscalls: []string{forgek.SyscallCaseOpen, forgek.SyscallCaseUpdate},
		WorkspaceScope:  []string{"workspace-a"},
		MutationScope:   forgek.MutationScopeCanonical,
		AuditRequired:   true,
	}); err != nil {
		t.Fatalf("grant capability: %v", err)
	}
}

func openCase(t *testing.T, kernel *forgek.Kernel, actorID string) string {
	t.Helper()
	result := kernel.DispatchSyscall(forgek.SyscallRequest{
		Name:        forgek.SyscallCaseOpen,
		ActorID:     actorID,
		WorkspaceID: "workspace-a",
		Input: map[string]any{
			"user_intent": "phase 2 case",
		},
	})
	if !result.Success {
		t.Fatalf("case.open failed: %v", result.Error)
	}
	return result.ObjectID
}

func TestNeuronCreatedSyscallRequestDoesNotExecuteByItself(t *testing.T) {
	kernel := testKernel()
	grantCaseCapability(t, kernel, "operator")
	caseID := openCase(t, kernel, "operator")

	request := SyscallRequest{
		RequestID:           "req-1",
		RequestedByNeuronID: "rule.request_update",
		CaseID:              caseID,
		WorkspaceID:         "workspace-a",
		SyscallName:         forgek.SyscallCaseUpdate,
		Payload:             map[string]any{"case_id": caseID, "summary": "requested only"},
		Reason:              "rule requested update",
		RequiredCapability:  forgek.SyscallCaseUpdate,
		CreatedAt:           time.Date(2026, 5, 3, 12, 1, 0, 0, time.UTC),
		Mutation:            true,
	}

	if got := len(kernel.Journal().ListEvents()); got != 1 {
		t.Fatalf("expected only case.open event before request, got %d", got)
	}
	if request.SyscallName != forgek.SyscallCaseUpdate {
		t.Fatal("request setup failed")
	}
	if got := len(kernel.Journal().ListEvents()); got != 1 {
		t.Fatalf("syscall request executed itself, journal count %d", got)
	}
}

func TestNeuronPathCannotMutateWithoutKernelCapability(t *testing.T) {
	kernel := testKernel()
	grantCaseCapability(t, kernel, "operator")
	caseID := openCase(t, kernel, "operator")
	client := NewKernelSyscallClient(kernel, "neuron-actor")

	result := client.Submit(SyscallRequest{
		RequestID:           "req-1",
		RequestedByNeuronID: "rule.request_update",
		CaseID:              caseID,
		WorkspaceID:         "workspace-a",
		SyscallName:         forgek.SyscallCaseUpdate,
		Payload:             map[string]any{"case_id": caseID, "summary": "denied"},
		Mutation:            true,
	})

	if result.Success {
		t.Fatal("neuron path mutated without actor capability")
	}
	if !errors.Is(result.Error, forgek.ErrCapabilityDenied) {
		t.Fatalf("expected capability denial, got %v", result.Error)
	}
	if got := len(kernel.Journal().ListEvents()); got != 1 {
		t.Fatalf("denied neuron path should not append mutation journal, got %d", got)
	}
}

func TestNeuronPathCanRequestKernelDispatchWithCapability(t *testing.T) {
	kernel := testKernel()
	grantCaseCapability(t, kernel, "operator")
	grantCaseCapability(t, kernel, "rule-actor")
	caseID := openCase(t, kernel, "operator")
	client := NewKernelSyscallClient(kernel, "rule-actor")

	result := client.Submit(SyscallRequest{
		RequestID:           "req-1",
		RequestedByNeuronID: "rule.request_update",
		CaseID:              caseID,
		WorkspaceID:         "workspace-a",
		SyscallName:         forgek.SyscallCaseUpdate,
		Payload:             map[string]any{"case_id": caseID, "summary": "kernel dispatched"},
		Mutation:            true,
	})

	if !result.Success {
		t.Fatalf("kernel-dispatched neuron request failed: %v", result.Error)
	}
	if got := len(kernel.Journal().ListEvents()); got != 2 {
		t.Fatalf("expected case.open and case.update journal events, got %d", got)
	}
	obj, _ := kernel.Objects().GetObject(caseID)
	if obj.State["summary"] != "kernel dispatched" {
		t.Fatalf("expected kernel-updated case summary, got %#v", obj.State)
	}
}

func TestCaseIntegrationNeuronOutputsDoNotChangeCaseWithoutKernelDispatch(t *testing.T) {
	kernel := testKernel()
	grantCaseCapability(t, kernel, "operator")
	caseID := openCase(t, kernel, "operator")

	scheduler := NewNeuronScheduler()
	if err := scheduler.Register(NewSimpleIntentProposalNeuron("neural.intent")); err != nil {
		t.Fatalf("register neural: %v", err)
	}
	if err := scheduler.Register(NewRequiredFieldRuleNeuron("rule.required_text", "text")); err != nil {
		t.Fatalf("register rule: %v", err)
	}

	ctx := NeuronContext{
		CaseID:      caseID,
		WorkspaceID: "workspace-a",
		CreatedAt:   time.Date(2026, 5, 3, 12, 2, 0, 0, time.UTC),
	}
	if _, err := scheduler.Dispatch("neural.intent", ctx, map[string]any{"text": "update the case"}); err != nil {
		t.Fatalf("dispatch neural: %v", err)
	}
	if _, err := scheduler.Dispatch("rule.required_text", ctx, map[string]any{"text": "update the case"}); err != nil {
		t.Fatalf("dispatch rule: %v", err)
	}

	obj, _ := kernel.Objects().GetObject(caseID)
	if obj.State["summary"] != "" {
		t.Fatalf("case changed without kernel syscall: %#v", obj.State)
	}
	if got := len(kernel.Journal().ListEvents()); got != 1 {
		t.Fatalf("neuron outputs should not journal mutations, got %d events", got)
	}
}
