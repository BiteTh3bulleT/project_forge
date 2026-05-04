package neurons

import (
	"encoding/json"
	"time"
)

type NeuronEnvelopeInput struct {
	EnvelopeID         string
	NeuronID           string
	CaseID             string
	WorkspaceID        string
	Lane               string
	InputRefs          []string
	OutputType         string
	OutputPayload      map[string]any
	Confidence         float64
	PassStatus         string
	ValidationRequired bool
	RequestedSyscalls  []SyscallRequest
	JournalRef         string
	CreatedAt          time.Time
	Metadata           map[string]any
}

type NeuronEnvelope struct {
	envelopeID         string
	neuronID           string
	caseID             string
	workspaceID        string
	lane               string
	inputRefs          []string
	outputType         string
	outputPayload      map[string]any
	confidence         float64
	passStatus         string
	validationRequired bool
	requestedSyscalls  []SyscallRequest
	journalRef         string
	createdAt          time.Time
	metadata           map[string]any
}

func NewNeuronEnvelope(input NeuronEnvelopeInput) (NeuronEnvelope, error) {
	if input.EnvelopeID == "" || input.NeuronID == "" || input.WorkspaceID == "" || input.OutputType == "" {
		return NeuronEnvelope{}, ErrInvalidEnvelope
	}
	if !validLane(input.Lane) {
		return NeuronEnvelope{}, ErrInvalidLane
	}
	if !validOutputType(input.OutputType) {
		return NeuronEnvelope{}, ErrInvalidOutputType
	}
	requests := make([]SyscallRequest, len(input.RequestedSyscalls))
	for i, request := range input.RequestedSyscalls {
		requests[i] = cloneSyscallRequest(request)
	}
	return NeuronEnvelope{
		envelopeID:         input.EnvelopeID,
		neuronID:           input.NeuronID,
		caseID:             input.CaseID,
		workspaceID:        input.WorkspaceID,
		lane:               input.Lane,
		inputRefs:          append([]string(nil), input.InputRefs...),
		outputType:         input.OutputType,
		outputPayload:      cloneMap(input.OutputPayload),
		confidence:         input.Confidence,
		passStatus:         input.PassStatus,
		validationRequired: input.ValidationRequired,
		requestedSyscalls:  requests,
		journalRef:         input.JournalRef,
		createdAt:          input.CreatedAt,
		metadata:           cloneMap(input.Metadata),
	}, nil
}

func (e NeuronEnvelope) EnvelopeID() string       { return e.envelopeID }
func (e NeuronEnvelope) NeuronID() string         { return e.neuronID }
func (e NeuronEnvelope) CaseID() string           { return e.caseID }
func (e NeuronEnvelope) WorkspaceID() string      { return e.workspaceID }
func (e NeuronEnvelope) Lane() string             { return e.lane }
func (e NeuronEnvelope) OutputType() string       { return e.outputType }
func (e NeuronEnvelope) Confidence() float64      { return e.confidence }
func (e NeuronEnvelope) PassStatus() string       { return e.passStatus }
func (e NeuronEnvelope) JournalRef() string       { return e.journalRef }
func (e NeuronEnvelope) CreatedAt() time.Time     { return e.createdAt }
func (e NeuronEnvelope) ValidationRequired() bool { return e.validationRequired }

func (e NeuronEnvelope) InputRefs() []string {
	return append([]string(nil), e.inputRefs...)
}

func (e NeuronEnvelope) OutputPayload() map[string]any {
	return cloneMap(e.outputPayload)
}

func (e NeuronEnvelope) RequestedSyscalls() []SyscallRequest {
	out := make([]SyscallRequest, len(e.requestedSyscalls))
	for i, request := range e.requestedSyscalls {
		out[i] = cloneSyscallRequest(request)
	}
	return out
}

func (e NeuronEnvelope) Metadata() map[string]any {
	return cloneMap(e.metadata)
}

func (e NeuronEnvelope) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		EnvelopeID         string           `json:"envelope_id"`
		NeuronID           string           `json:"neuron_id"`
		CaseID             string           `json:"case_id,omitempty"`
		WorkspaceID        string           `json:"workspace_id"`
		Lane               string           `json:"lane"`
		InputRefs          []string         `json:"input_refs,omitempty"`
		OutputType         string           `json:"output_type"`
		OutputPayload      map[string]any   `json:"output_payload,omitempty"`
		Confidence         float64          `json:"confidence,omitempty"`
		PassStatus         string           `json:"pass_status,omitempty"`
		ValidationRequired bool             `json:"validation_required,omitempty"`
		RequestedSyscalls  []SyscallRequest `json:"requested_syscalls,omitempty"`
		JournalRef         string           `json:"journal_ref,omitempty"`
		CreatedAt          time.Time        `json:"created_at"`
		Metadata           map[string]any   `json:"metadata,omitempty"`
	}{
		EnvelopeID:         e.envelopeID,
		NeuronID:           e.neuronID,
		CaseID:             e.caseID,
		WorkspaceID:        e.workspaceID,
		Lane:               e.lane,
		InputRefs:          e.InputRefs(),
		OutputType:         e.outputType,
		OutputPayload:      e.OutputPayload(),
		Confidence:         e.confidence,
		PassStatus:         e.passStatus,
		ValidationRequired: e.validationRequired,
		RequestedSyscalls:  e.RequestedSyscalls(),
		JournalRef:         e.journalRef,
		CreatedAt:          e.createdAt,
		Metadata:           e.Metadata(),
	})
}

func cloneMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
