package neurons

import "sync"

type NeuronScheduler struct {
	mu      sync.RWMutex
	neurons map[string]Neuron
}

func NewNeuronScheduler() *NeuronScheduler {
	return &NeuronScheduler{neurons: make(map[string]Neuron)}
}

func (s *NeuronScheduler) Register(neuron Neuron) error {
	manifest, err := neuron.Manifest().Validated()
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.neurons[manifest.NeuronID] = neuron
	return nil
}

func (s *NeuronScheduler) List() []NeuronManifest {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]NeuronManifest, 0, len(s.neurons))
	for _, neuron := range s.neurons {
		out = append(out, neuron.Manifest())
	}
	return out
}

func (s *NeuronScheduler) ListByLane(lane string) []NeuronManifest {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]NeuronManifest, 0)
	for _, neuron := range s.neurons {
		if neuron.Manifest().Lane == lane {
			out = append(out, neuron.Manifest())
		}
	}
	return out
}

func (s *NeuronScheduler) Get(neuronID string) (Neuron, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	neuron, ok := s.neurons[neuronID]
	return neuron, ok
}

func (s *NeuronScheduler) Dispatch(neuronID string, ctx NeuronContext, input map[string]any) (NeuronEnvelope, error) {
	neuron, ok := s.Get(neuronID)
	if !ok {
		return NeuronEnvelope{}, ErrNeuronNotFound
	}
	return neuron.Fire(ctx, input)
}

func (s *NeuronScheduler) JournalRefs() []string {
	return nil
}
