package neurons

import (
	"errors"
	"testing"
	"time"
)

func TestSchedulerRegistersListsAndDispatchesNeurons(t *testing.T) {
	scheduler := NewNeuronScheduler()
	neural := NewSimpleIntentProposalNeuron("neural.intent")
	rule := NewRequiredFieldRuleNeuron("rule.required_text", "text")

	if err := scheduler.Register(neural); err != nil {
		t.Fatalf("register neural: %v", err)
	}
	if err := scheduler.Register(rule); err != nil {
		t.Fatalf("register rule: %v", err)
	}

	if got := len(scheduler.List()); got != 2 {
		t.Fatalf("expected two neurons, got %d", got)
	}
	if got := len(scheduler.ListByLane(LaneNeural)); got != 1 {
		t.Fatalf("expected one neural lane neuron, got %d", got)
	}
	if _, ok := scheduler.Get("rule.required_text"); !ok {
		t.Fatal("registered rule neuron missing")
	}

	envelope, err := scheduler.Dispatch("neural.intent", NeuronContext{
		EnvelopeID:  "env-1",
		CaseID:      "case-1",
		WorkspaceID: "workspace-a",
		CreatedAt:   time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC),
	}, map[string]any{"text": "Open a case"})
	if err != nil {
		t.Fatalf("dispatch neural: %v", err)
	}
	if envelope.OutputType() != OutputProposal {
		t.Fatalf("expected proposal envelope, got %s", envelope.OutputType())
	}
}

func TestSchedulerRejectsUnknownNeuron(t *testing.T) {
	scheduler := NewNeuronScheduler()
	_, err := scheduler.Dispatch("missing", NeuronContext{}, map[string]any{})
	if !errors.Is(err, ErrNeuronNotFound) {
		t.Fatalf("expected ErrNeuronNotFound, got %v", err)
	}
}

func TestSchedulerDoesNotBypassKernelOrCapabilities(t *testing.T) {
	scheduler := NewNeuronScheduler()
	requestingRule := NewCaseUpdateRequestRuleNeuron("rule.request_update")
	if err := scheduler.Register(requestingRule); err != nil {
		t.Fatalf("register requesting rule: %v", err)
	}

	envelope, err := scheduler.Dispatch("rule.request_update", NeuronContext{
		EnvelopeID:  "env-req",
		CaseID:      "case-1",
		WorkspaceID: "workspace-a",
		CreatedAt:   time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC),
	}, map[string]any{"summary": "requested update"})
	if err != nil {
		t.Fatalf("dispatch requesting rule: %v", err)
	}
	if envelope.OutputType() != OutputSyscallRequest {
		t.Fatalf("expected syscall request output, got %s", envelope.OutputType())
	}
	if got := len(scheduler.JournalRefs()); got != 0 {
		t.Fatalf("scheduler should not journal directly, got refs %#v", got)
	}
}
