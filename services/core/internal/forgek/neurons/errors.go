package neurons

import "errors"

var (
	ErrInvalidManifest       = errors.New("invalid neuron manifest")
	ErrInvalidNeuronType     = errors.New("invalid neuron type")
	ErrInvalidLane           = errors.New("invalid neuron lane")
	ErrInvalidOutputType     = errors.New("invalid neuron output type")
	ErrInvalidEnvelope       = errors.New("invalid neuron envelope")
	ErrSideEffectsNotAllowed = errors.New("neuron side effects are not allowed")
	ErrNeuronNotFound        = errors.New("neuron not found")
	ErrInvalidInput          = errors.New("invalid neuron input")
)
