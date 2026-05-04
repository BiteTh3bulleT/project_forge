package neurons

import "time"

type NeuronContext struct {
	EnvelopeID  string
	CaseID      string
	WorkspaceID string
	InputRefs   []string
	CreatedAt   time.Time
	Metadata    map[string]any
}

type Neuron interface {
	Manifest() NeuronManifest
	ValidateInput(input map[string]any) error
	Fire(ctx NeuronContext, input map[string]any) (NeuronEnvelope, error)
	ValidateOutput(envelope NeuronEnvelope) error
}

type BaseNeuron struct {
	manifest NeuronManifest
}

func NewBaseNeuron(manifest NeuronManifest) (BaseNeuron, error) {
	normalized, err := manifest.Validated()
	if err != nil {
		return BaseNeuron{}, err
	}
	return BaseNeuron{manifest: normalized}, nil
}

func (n BaseNeuron) Manifest() NeuronManifest {
	return n.manifest
}

func (n BaseNeuron) ValidateInput(input map[string]any) error {
	if input == nil {
		return ErrInvalidInput
	}
	return nil
}

func (n BaseNeuron) ValidateOutput(envelope NeuronEnvelope) error {
	if envelope.NeuronID() != n.manifest.NeuronID {
		return ErrInvalidEnvelope
	}
	if envelope.OutputType() != n.manifest.OutputType {
		return ErrInvalidOutputType
	}
	_, err := n.manifest.ValidateRequestedSyscalls(envelope.RequestedSyscalls())
	return err
}
