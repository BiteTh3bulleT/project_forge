package modelruntime

import "strings"

// BackendProfileLabel names an operator-facing runtime posture. Profiles are
// descriptive contracts; they do not load models or select backends by
// themselves.
type BackendProfileLabel string

const (
	BackendProfileUnknown                BackendProfileLabel = ""
	BackendProfileCPUSafe                BackendProfileLabel = "cpu_safe"
	BackendProfileLocalLlamaCPP          BackendProfileLabel = "local_llama_cpp"
	BackendProfileLocalOllamaDev         BackendProfileLabel = "local_ollama_dev"
	BackendProfileInteractiveVLLM        BackendProfileLabel = "interactive_vllm"
	BackendProfileEmbeddingTEI           BackendProfileLabel = "embedding_tei"
	BackendProfileOpenAICompatibleRemote BackendProfileLabel = "openai_compatible_remote"
)

var backendProfileLabels = []BackendProfileLabel{
	BackendProfileCPUSafe,
	BackendProfileLocalLlamaCPP,
	BackendProfileLocalOllamaDev,
	BackendProfileInteractiveVLLM,
	BackendProfileEmbeddingTEI,
	BackendProfileOpenAICompatibleRemote,
}

func AllBackendProfileLabels() []BackendProfileLabel {
	out := make([]BackendProfileLabel, len(backendProfileLabels))
	copy(out, backendProfileLabels)
	return out
}

func ParseBackendProfileLabel(raw string) BackendProfileLabel {
	label := BackendProfileLabel(strings.TrimSpace(raw))
	if label.IsKnown() {
		return label
	}
	return BackendProfileUnknown
}

func (p BackendProfileLabel) IsKnown() bool {
	for _, known := range backendProfileLabels {
		if p == known {
			return true
		}
	}
	return false
}
