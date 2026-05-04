package neurons

import (
	"errors"
	"testing"
	"time"
)

func TestSimpleIntentProposalNeuronEmitsProposalWithoutMutatingState(t *testing.T) {
	neuron := NewSimpleIntentProposalNeuron("neural.intent")
	ctx := NeuronContext{
		EnvelopeID:  "env-1",
		CaseID:      "case-1",
		WorkspaceID: "workspace-a",
		CreatedAt:   time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC),
	}

	envelope, err := neuron.Fire(ctx, map[string]any{"text": "Open a case for Phase 2"})
	if err != nil {
		t.Fatalf("fire neural neuron: %v", err)
	}

	if envelope.OutputType() != OutputProposal {
		t.Fatalf("expected proposal output, got %s", envelope.OutputType())
	}
	if envelope.Confidence() <= 0 {
		t.Fatalf("expected confidence, got %f", envelope.Confidence())
	}
	if len(envelope.RequestedSyscalls()) != 0 {
		t.Fatal("simple neural proposal should not request syscalls")
	}
	payload := envelope.OutputPayload()
	if payload["inferred_intent"] == "" || payload["summary"] == "" {
		t.Fatalf("missing proposal payload: %#v", payload)
	}
}

func TestRequiredFieldRuleNeuronEmitsDeterministicValidation(t *testing.T) {
	neuron := NewRequiredFieldRuleNeuron("rule.required_text", "text")
	ctx := NeuronContext{
		EnvelopeID:  "env-rule",
		CaseID:      "case-1",
		WorkspaceID: "workspace-a",
		CreatedAt:   time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC),
	}

	pass, err := neuron.Fire(ctx, map[string]any{"text": "present"})
	if err != nil {
		t.Fatalf("fire passing rule: %v", err)
	}
	fail, err := neuron.Fire(ctx, map[string]any{"text": ""})
	if err != nil {
		t.Fatalf("fire failing rule: %v", err)
	}
	failAgain, err := neuron.Fire(ctx, map[string]any{"text": ""})
	if err != nil {
		t.Fatalf("fire repeated failing rule: %v", err)
	}

	if pass.OutputType() != OutputValidation || pass.PassStatus() != PassPassed {
		t.Fatalf("expected passing validation, got %#v", pass)
	}
	if fail.OutputType() != OutputValidation || fail.PassStatus() != PassFailed {
		t.Fatalf("expected failing validation, got %#v", fail)
	}
	if fail.OutputPayload()["reason"] != failAgain.OutputPayload()["reason"] {
		t.Fatalf("rule output is not deterministic: %#v vs %#v", fail.OutputPayload(), failAgain.OutputPayload())
	}
}

func TestNeuronOutputValidationRejectsWrongOutputCategory(t *testing.T) {
	neuron := NewSimpleIntentProposalNeuron("neural.intent")
	ctx := NeuronContext{
		EnvelopeID:  "env-1",
		CaseID:      "case-1",
		WorkspaceID: "workspace-a",
		CreatedAt:   time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC),
	}

	_, err := neuron.produceEnvelope(ctx, OutputValidation, map[string]any{"bad": true}, 1, PassPassed, nil)
	if !errors.Is(err, ErrInvalidOutputType) {
		t.Fatalf("expected ErrInvalidOutputType, got %v", err)
	}
}
