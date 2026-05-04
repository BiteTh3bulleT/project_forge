package neurons

import "fmt"

type NeuronManifest struct {
	NeuronID             string
	NeuronName           string
	NeuronType           string
	Lane                 string
	AuthorityLevel       string
	InputType            string
	OutputType           string
	SideEffectsAllowed   bool
	RequiredCapabilities []string
	Deterministic        bool
	ModelID              string
	PolicyVersion        string
	AuditRequired        bool
	Description          string
}

func (m NeuronManifest) Validated() (NeuronManifest, error) {
	if m.NeuronID == "" || m.NeuronName == "" || m.PolicyVersion == "" {
		return NeuronManifest{}, ErrInvalidManifest
	}
	if !validNeuronType(m.NeuronType) {
		return NeuronManifest{}, fmt.Errorf("%w: %s", ErrInvalidNeuronType, m.NeuronType)
	}
	if !validLane(m.Lane) {
		return NeuronManifest{}, fmt.Errorf("%w: %s", ErrInvalidLane, m.Lane)
	}

	if m.NeuronType == NeuronTypeRule {
		m.Deterministic = true
		if m.OutputType == "" {
			m.OutputType = OutputValidation
		}
		if m.AuthorityLevel == "" {
			m.AuthorityLevel = AuthorityValidated
		}
	}
	if m.NeuronType == NeuronTypeNeural {
		if m.OutputType == "" {
			m.OutputType = OutputProposal
		}
		if m.AuthorityLevel == "" {
			m.AuthorityLevel = AuthorityProposal
		}
	}
	if !validOutputType(m.OutputType) {
		return NeuronManifest{}, fmt.Errorf("%w: %s", ErrInvalidOutputType, m.OutputType)
	}
	m.RequiredCapabilities = append([]string(nil), m.RequiredCapabilities...)
	return m, nil
}

func (m NeuronManifest) ValidateRequestedSyscalls(requests []SyscallRequest) (NeuronManifest, error) {
	normalized, err := m.Validated()
	if err != nil {
		return NeuronManifest{}, err
	}
	if !normalized.SideEffectsAllowed {
		for _, request := range requests {
			if request.Mutation {
				return NeuronManifest{}, ErrSideEffectsNotAllowed
			}
		}
	}
	return normalized, nil
}

func validNeuronType(value string) bool {
	return value == NeuronTypeNeural || value == NeuronTypeRule
}

func validLane(value string) bool {
	switch value {
	case LaneNeural, LaneArterial, LaneLymphatic, LaneHyperlane:
		return true
	default:
		return false
	}
}

func validOutputType(value string) bool {
	switch value {
	case OutputProposal, OutputValidation, OutputSyscallRequest, OutputObservation, OutputError:
		return true
	default:
		return false
	}
}
