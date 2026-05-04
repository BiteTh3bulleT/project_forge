package neurons

import (
	"fmt"
	"strings"
	"time"
)

type RequiredFieldRuleNeuron struct {
	BaseNeuron
	field string
}

func NewRequiredFieldRuleNeuron(neuronID, field string) *RequiredFieldRuleNeuron {
	base, err := NewBaseNeuron(NeuronManifest{
		NeuronID:           neuronID,
		NeuronName:         "Required field rule neuron",
		NeuronType:         NeuronTypeRule,
		Lane:               LaneArterial,
		AuthorityLevel:     AuthorityValidated,
		InputType:          "map",
		OutputType:         OutputValidation,
		SideEffectsAllowed: false,
		Deterministic:      true,
		PolicyVersion:      "phase-2",
		Description:        "Validates that a required field is present.",
	})
	if err != nil {
		panic(err)
	}
	return &RequiredFieldRuleNeuron{BaseNeuron: base, field: field}
}

func (n *RequiredFieldRuleNeuron) Fire(ctx NeuronContext, input map[string]any) (NeuronEnvelope, error) {
	if err := n.ValidateInput(input); err != nil {
		return NeuronEnvelope{}, err
	}
	value, ok := input[n.field]
	passed := ok && strings.TrimSpace(fmt.Sprint(value)) != ""
	status := PassPassed
	reason := "required field present"
	if !passed {
		status = PassFailed
		reason = "required field missing: " + n.field
	}
	payload := map[string]any{
		"field":  n.field,
		"passed": passed,
		"reason": reason,
	}
	return n.produceEnvelope(ctx, OutputValidation, payload, 1.0, status, nil)
}

func (n *RequiredFieldRuleNeuron) produceEnvelope(ctx NeuronContext, outputType string, payload map[string]any, confidence float64, passStatus string, requests []SyscallRequest) (NeuronEnvelope, error) {
	if outputType != n.Manifest().OutputType {
		return NeuronEnvelope{}, ErrInvalidOutputType
	}
	return produceRuleEnvelope(n.BaseNeuron, ctx, outputType, payload, confidence, passStatus, requests)
}

type CaseUpdateRequestRuleNeuron struct {
	BaseNeuron
}

func NewCaseUpdateRequestRuleNeuron(neuronID string) *CaseUpdateRequestRuleNeuron {
	base, err := NewBaseNeuron(NeuronManifest{
		NeuronID:             neuronID,
		NeuronName:           "Case update request rule neuron",
		NeuronType:           NeuronTypeRule,
		Lane:                 LaneArterial,
		AuthorityLevel:       AuthorityValidated,
		InputType:            "map",
		OutputType:           OutputSyscallRequest,
		SideEffectsAllowed:   true,
		RequiredCapabilities: []string{"case.update"},
		Deterministic:        true,
		PolicyVersion:        "phase-2",
		Description:          "Produces explicit case.update syscall request envelopes.",
	})
	if err != nil {
		panic(err)
	}
	return &CaseUpdateRequestRuleNeuron{BaseNeuron: base}
}

func (n *CaseUpdateRequestRuleNeuron) Fire(ctx NeuronContext, input map[string]any) (NeuronEnvelope, error) {
	if err := n.ValidateInput(input); err != nil {
		return NeuronEnvelope{}, err
	}
	summary, _ := input["summary"].(string)
	request := SyscallRequest{
		RequestID:           n.Manifest().NeuronID + ".request",
		RequestedByNeuronID: n.Manifest().NeuronID,
		CaseID:              ctx.CaseID,
		WorkspaceID:         ctx.WorkspaceID,
		SyscallName:         "case.update",
		Payload: map[string]any{
			"case_id": ctx.CaseID,
			"summary": summary,
		},
		Reason:             "deterministic rule requested case metadata update",
		RequiredCapability: "case.update",
		CreatedAt:          ctx.CreatedAt,
		Mutation:           true,
	}
	payload := map[string]any{
		"syscall_name": request.SyscallName,
		"reason":       request.Reason,
	}
	return produceRuleEnvelope(n.BaseNeuron, ctx, OutputSyscallRequest, payload, 1.0, PassPassed, []SyscallRequest{request})
}

func produceRuleEnvelope(base BaseNeuron, ctx NeuronContext, outputType string, payload map[string]any, confidence float64, passStatus string, requests []SyscallRequest) (NeuronEnvelope, error) {
	createdAt := ctx.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	envelopeID := ctx.EnvelopeID
	if envelopeID == "" {
		envelopeID = base.Manifest().NeuronID + ".envelope"
	}
	envelope, err := NewNeuronEnvelope(NeuronEnvelopeInput{
		EnvelopeID:         envelopeID,
		NeuronID:           base.Manifest().NeuronID,
		CaseID:             ctx.CaseID,
		WorkspaceID:        ctx.WorkspaceID,
		Lane:               base.Manifest().Lane,
		InputRefs:          ctx.InputRefs,
		OutputType:         outputType,
		OutputPayload:      payload,
		Confidence:         confidence,
		PassStatus:         passStatus,
		ValidationRequired: false,
		RequestedSyscalls:  requests,
		CreatedAt:          createdAt,
		Metadata:           ctx.Metadata,
	})
	if err != nil {
		return NeuronEnvelope{}, err
	}
	if err := base.ValidateOutput(envelope); err != nil {
		return NeuronEnvelope{}, err
	}
	return envelope, nil
}
