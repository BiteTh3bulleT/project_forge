package neurons

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestProposalEnvelopePreservesProvenanceAndConfidence(t *testing.T) {
	createdAt := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	envelope, err := NewNeuronEnvelope(NeuronEnvelopeInput{
		EnvelopeID:    "env-1",
		NeuronID:      "neural.intent",
		CaseID:        "case-1",
		WorkspaceID:   "workspace-a",
		Lane:          LaneNeural,
		InputRefs:     []string{"input-1"},
		OutputType:    OutputProposal,
		OutputPayload: map[string]any{"summary": "open case"},
		Confidence:    0.82,
		CreatedAt:     createdAt,
		Metadata:      map[string]any{"source": "test"},
	})
	if err != nil {
		t.Fatalf("create envelope: %v", err)
	}

	if envelope.NeuronID() != "neural.intent" || envelope.CaseID() != "case-1" || envelope.WorkspaceID() != "workspace-a" {
		t.Fatalf("unexpected envelope identity: %#v", envelope)
	}
	if envelope.OutputType() != OutputProposal || envelope.Confidence() != 0.82 {
		t.Fatalf("unexpected proposal output: %#v", envelope)
	}
	if !reflect.DeepEqual(envelope.InputRefs(), []string{"input-1"}) {
		t.Fatalf("input refs not preserved: %#v", envelope.InputRefs())
	}

	encoded, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}
	if !json.Valid(encoded) {
		t.Fatal("envelope did not serialize as valid JSON")
	}
}

func TestValidationEnvelopeCarriesPassFailStatus(t *testing.T) {
	envelope, err := NewNeuronEnvelope(NeuronEnvelopeInput{
		EnvelopeID:  "env-2",
		NeuronID:    "rule.required",
		CaseID:      "case-1",
		WorkspaceID: "workspace-a",
		Lane:        LaneArterial,
		OutputType:  OutputValidation,
		PassStatus:  PassFailed,
		CreatedAt:   time.Date(2026, 5, 3, 12, 1, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("create validation envelope: %v", err)
	}

	if envelope.OutputType() != OutputValidation {
		t.Fatalf("expected validation output, got %s", envelope.OutputType())
	}
	if envelope.PassStatus() != PassFailed {
		t.Fatalf("expected failed status, got %s", envelope.PassStatus())
	}
}

func TestEnvelopeCopyOutPreventsMutationOfStoredPayload(t *testing.T) {
	envelope, err := NewNeuronEnvelope(NeuronEnvelopeInput{
		EnvelopeID:    "env-3",
		NeuronID:      "neural.intent",
		CaseID:        "case-1",
		WorkspaceID:   "workspace-a",
		Lane:          LaneNeural,
		OutputType:    OutputProposal,
		OutputPayload: map[string]any{"summary": "original"},
		InputRefs:     []string{"input-1"},
		RequestedSyscalls: []SyscallRequest{{
			RequestID:           "req-1",
			RequestedByNeuronID: "neural.intent",
			WorkspaceID:         "workspace-a",
			SyscallName:         "case.update",
			Payload:             map[string]any{"summary": "request"},
		}},
		Metadata:  map[string]any{"source": "test"},
		CreatedAt: time.Date(2026, 5, 3, 12, 2, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("create envelope: %v", err)
	}

	payload := envelope.OutputPayload()
	payload["summary"] = "tampered"
	refs := envelope.InputRefs()
	refs[0] = "tampered"
	requests := envelope.RequestedSyscalls()
	requests[0].Payload["summary"] = "tampered"
	metadata := envelope.Metadata()
	metadata["source"] = "tampered"

	if envelope.OutputPayload()["summary"] == "tampered" {
		t.Fatal("payload mutation escaped into envelope")
	}
	if envelope.InputRefs()[0] == "tampered" {
		t.Fatal("input refs mutation escaped into envelope")
	}
	if envelope.RequestedSyscalls()[0].Payload["summary"] == "tampered" {
		t.Fatal("requested syscall mutation escaped into envelope")
	}
	if envelope.Metadata()["source"] == "tampered" {
		t.Fatal("metadata mutation escaped into envelope")
	}
}
