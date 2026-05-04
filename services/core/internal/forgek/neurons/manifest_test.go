package neurons

import (
	"errors"
	"testing"
)

func TestValidNeuralManifestDefaultsProposalOriented(t *testing.T) {
	manifest := NeuronManifest{
		NeuronID:      "neural.intent",
		NeuronName:    "Simple intent proposal",
		NeuronType:    NeuronTypeNeural,
		Lane:          LaneNeural,
		InputType:     "operator_text",
		OutputType:    OutputProposal,
		PolicyVersion: "phase-2",
		Description:   "Produces proposal envelopes from operator text.",
	}

	normalized, err := manifest.Validated()
	if err != nil {
		t.Fatalf("valid neural manifest failed: %v", err)
	}
	if normalized.OutputType != OutputProposal {
		t.Fatalf("expected proposal output, got %s", normalized.OutputType)
	}
	if normalized.AuthorityLevel != AuthorityProposal {
		t.Fatalf("expected proposal authority, got %s", normalized.AuthorityLevel)
	}
	if normalized.SideEffectsAllowed {
		t.Fatal("neural manifest should default to side-effect free")
	}
}

func TestValidRuleManifestDefaultsDeterministicValidation(t *testing.T) {
	manifest := NeuronManifest{
		NeuronID:      "rule.required_field",
		NeuronName:    "Required field rule",
		NeuronType:    NeuronTypeRule,
		Lane:          LaneArterial,
		InputType:     "map",
		PolicyVersion: "phase-2",
		Description:   "Validates required fields.",
	}

	normalized, err := manifest.Validated()
	if err != nil {
		t.Fatalf("valid rule manifest failed: %v", err)
	}
	if !normalized.Deterministic {
		t.Fatal("rule manifest should default deterministic")
	}
	if normalized.OutputType != OutputValidation {
		t.Fatalf("expected validation output, got %s", normalized.OutputType)
	}
}

func TestInvalidNeuronManifestFailsClearly(t *testing.T) {
	cases := []struct {
		name     string
		manifest NeuronManifest
		wantErr  error
	}{
		{
			name: "invalid type",
			manifest: NeuronManifest{
				NeuronID:      "bad",
				NeuronName:    "Bad",
				NeuronType:    "OTHER",
				Lane:          LaneNeural,
				PolicyVersion: "phase-2",
			},
			wantErr: ErrInvalidNeuronType,
		},
		{
			name: "invalid lane",
			manifest: NeuronManifest{
				NeuronID:      "bad",
				NeuronName:    "Bad",
				NeuronType:    NeuronTypeRule,
				Lane:          "OTHER",
				PolicyVersion: "phase-2",
			},
			wantErr: ErrInvalidLane,
		},
		{
			name: "missing identity",
			manifest: NeuronManifest{
				NeuronType:    NeuronTypeRule,
				Lane:          LaneArterial,
				PolicyVersion: "phase-2",
			},
			wantErr: ErrInvalidManifest,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.manifest.Validated()
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected %v, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestSideEffectFreeManifestRejectsMutatingSyscallRequests(t *testing.T) {
	manifest := NeuronManifest{
		NeuronID:           "rule.no_side_effects",
		NeuronName:         "No side effects",
		NeuronType:         NeuronTypeRule,
		Lane:               LaneArterial,
		SideEffectsAllowed: false,
		PolicyVersion:      "phase-2",
	}

	_, err := manifest.ValidateRequestedSyscalls([]SyscallRequest{{
		RequestID:           "req-1",
		RequestedByNeuronID: "rule.no_side_effects",
		WorkspaceID:         "workspace-a",
		SyscallName:         "case.update",
		Mutation:            true,
	}})
	if !errors.Is(err, ErrSideEffectsNotAllowed) {
		t.Fatalf("expected ErrSideEffectsNotAllowed, got %v", err)
	}
}
