package api

import (
	"os"
	"strings"
	"testing"
)

func TestRuntimeProposalBoundaryOwnsEveryModelVisibilitySurface(t *testing.T) {
	t.Parallel()

	exactCalls := map[string]int{
		"chat_assistant_modelruntime.go": 2,
		"chat_assistant_ollama.go":       2,
		"chat_assistant_gateway.go":      1,
		"model_runtime.go":               1,
	}
	for name, want := range exactCalls {
		raw, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if got := strings.Count(string(raw), "decideRuntimeProposal("); got != want {
			t.Fatalf("%s runtime decision calls=%d want=%d", name, got, want)
		}
	}

	gatewayRaw, err := os.ReadFile("chat_assistant_gateway.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{
		`"model_raw_response"`,
		`"arguments": argsForStage`,
		`"arguments": argsStr`,
	} {
		if strings.Contains(string(gatewayRaw), forbidden) {
			t.Fatalf("gateway stream exposes pre-decision model bytes via %q", forbidden)
		}
	}

	for _, name := range []string{"chat_assistant_prompt.go", "chat_post.go"} {
		raw, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if strings.Contains(string(raw), "buildMemoryObservationContext") {
			t.Fatalf("%s reintroduced legacy observation prompt authority", name)
		}
	}
}
