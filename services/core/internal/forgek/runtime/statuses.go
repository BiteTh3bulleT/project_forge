package runtime

type DriverKind string

const (
	DriverKindMock        DriverKind = "MOCK"
	DriverKindLocalDev    DriverKind = "LOCAL_DEV"
	DriverKindVLLM        DriverKind = "VLLM"
	DriverKindSGLang      DriverKind = "SGLANG"
	DriverKindTensorRTLLM DriverKind = "TENSORRT_LLM"
	DriverKindOllama      DriverKind = "OLLAMA"
	DriverKindLlamaCPP    DriverKind = "LLAMA_CPP"
	DriverKindRemoteAPI   DriverKind = "REMOTE_API"
	DriverKindFuture      DriverKind = "FUTURE"
)

const (
	RuntimeAuthorityProposalOnly = "PROPOSAL_ONLY"
)

type RuntimeHealthStatus string

const (
	HealthAvailable   RuntimeHealthStatus = "AVAILABLE"
	HealthUnavailable RuntimeHealthStatus = "UNAVAILABLE"
)

type FinishReason string

const (
	FinishStop   FinishReason = "STOP"
	FinishLength FinishReason = "LENGTH"
	FinishError  FinishReason = "ERROR"
)

func ValidDriverKind(kind DriverKind) bool {
	switch kind {
	case DriverKindMock,
		DriverKindLocalDev,
		DriverKindVLLM,
		DriverKindSGLang,
		DriverKindTensorRTLLM,
		DriverKindOllama,
		DriverKindLlamaCPP,
		DriverKindRemoteAPI,
		DriverKindFuture:
		return true
	default:
		return false
	}
}
