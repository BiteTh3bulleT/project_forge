package neurons

import "time"

const (
	NeuronTypeNeural = "NEURAL"
	NeuronTypeRule   = "RULE"
)

const (
	LaneNeural    = "NEURAL"
	LaneArterial  = "ARTERIAL"
	LaneLymphatic = "LYMPHATIC"
	LaneHyperlane = "HYPERLANE"
)

const (
	OutputProposal       = "PROPOSAL"
	OutputValidation     = "VALIDATION"
	OutputSyscallRequest = "SYSCALL_REQUEST"
	OutputObservation    = "OBSERVATION"
	OutputError          = "ERROR"
)

const (
	PassUnset  = ""
	PassPassed = "PASS"
	PassFailed = "FAIL"
)

const (
	AuthorityProposal  = "PROPOSAL"
	AuthorityValidated = "VALIDATED"
)

type SyscallRequest struct {
	RequestID           string
	RequestedByNeuronID string
	CaseID              string
	WorkspaceID         string
	SyscallName         string
	Payload             map[string]any
	Reason              string
	RequiredCapability  string
	CreatedAt           time.Time
	Mutation            bool
}

func cloneSyscallRequest(request SyscallRequest) SyscallRequest {
	request.Payload = cloneMap(request.Payload)
	return request
}
