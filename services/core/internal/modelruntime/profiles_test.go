package modelruntime

import "testing"

func TestBackendProfileLabelsMatchM5Contract(t *testing.T) {
	expected := []BackendProfileLabel{
		BackendProfileCPUSafe,
		BackendProfileLocalLlamaCPP,
		BackendProfileLocalOllamaDev,
		BackendProfileInteractiveVLLM,
		BackendProfileEmbeddingTEI,
		BackendProfileOpenAICompatibleRemote,
	}
	if got := AllBackendProfileLabels(); len(got) != len(expected) {
		t.Fatalf("expected %d backend profiles, got %d: %v", len(expected), len(got), got)
	}
	for i, want := range expected {
		if got := AllBackendProfileLabels()[i]; got != want {
			t.Fatalf("profile %d mismatch: got %q want %q", i, got, want)
		}
		if !want.IsKnown() {
			t.Fatalf("expected profile %q to be known", want)
		}
	}
}

func TestBackendProfileLabelParsingIsBounded(t *testing.T) {
	if got := ParseBackendProfileLabel(" interactive_vllm "); got != BackendProfileInteractiveVLLM {
		t.Fatalf("expected trimmed interactive_vllm, got %q", got)
	}
	if got := ParseBackendProfileLabel("INTERACTIVE_VLLM"); got != BackendProfileUnknown {
		t.Fatalf("profile parsing should be exact after trim, got %q", got)
	}
	if got := ParseBackendProfileLabel("nixos_rebuild"); got != BackendProfileUnknown {
		t.Fatalf("unknown profile should stay unknown, got %q", got)
	}
}
