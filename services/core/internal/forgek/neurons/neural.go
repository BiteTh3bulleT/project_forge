package neurons

import (
	"strings"
	"time"
)

type SimpleIntentProposalNeuron struct {
	BaseNeuron
}

func NewSimpleIntentProposalNeuron(neuronID string) *SimpleIntentProposalNeuron {
	base, err := NewBaseNeuron(NeuronManifest{
		NeuronID:           neuronID,
		NeuronName:         "Simple intent proposal neuron",
		NeuronType:         NeuronTypeNeural,
		Lane:               LaneNeural,
		AuthorityLevel:     AuthorityProposal,
		InputType:          "text",
		OutputType:         OutputProposal,
		SideEffectsAllowed: false,
		Deterministic:      false,
		PolicyVersion:      "phase-2",
		Description:        "Produces proposal envelopes from operator text without model calls.",
	})
	if err != nil {
		panic(err)
	}
	return &SimpleIntentProposalNeuron{BaseNeuron: base}
}

func (n *SimpleIntentProposalNeuron) Fire(ctx NeuronContext, input map[string]any) (NeuronEnvelope, error) {
	if err := n.ValidateInput(input); err != nil {
		return NeuronEnvelope{}, err
	}
	text, ok := input["text"].(string)
	if !ok || strings.TrimSpace(text) == "" {
		return NeuronEnvelope{}, ErrInvalidInput
	}
	payload := map[string]any{
		"inferred_intent": strings.TrimSpace(text),
		"summary":         summarizeText(text),
		"assumptions":     []string{"Phase 2 example neuron; no runtime model call performed."},
	}
	return n.produceEnvelope(ctx, OutputProposal, payload, 0.75, PassUnset, nil)
}

func (n *SimpleIntentProposalNeuron) produceEnvelope(ctx NeuronContext, outputType string, payload map[string]any, confidence float64, passStatus string, requests []SyscallRequest) (NeuronEnvelope, error) {
	if outputType != n.Manifest().OutputType {
		return NeuronEnvelope{}, ErrInvalidOutputType
	}
	createdAt := ctx.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	envelopeID := ctx.EnvelopeID
	if envelopeID == "" {
		envelopeID = n.Manifest().NeuronID + ".envelope"
	}
	envelope, err := NewNeuronEnvelope(NeuronEnvelopeInput{
		EnvelopeID:         envelopeID,
		NeuronID:           n.Manifest().NeuronID,
		CaseID:             ctx.CaseID,
		WorkspaceID:        ctx.WorkspaceID,
		Lane:               n.Manifest().Lane,
		InputRefs:          ctx.InputRefs,
		OutputType:         outputType,
		OutputPayload:      payload,
		Confidence:         confidence,
		PassStatus:         passStatus,
		ValidationRequired: true,
		RequestedSyscalls:  requests,
		CreatedAt:          createdAt,
		Metadata:           ctx.Metadata,
	})
	if err != nil {
		return NeuronEnvelope{}, err
	}
	if err := n.ValidateOutput(envelope); err != nil {
		return NeuronEnvelope{}, err
	}
	return envelope, nil
}

func summarizeText(text string) string {
	text = strings.TrimSpace(text)
	if len(text) <= 80 {
		return text
	}
	return text[:80]
}
