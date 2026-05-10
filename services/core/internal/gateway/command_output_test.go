package gateway

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestBoundedCombinedOutputTruncatesLargeOutput(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=TestBoundedCombinedOutputHelperProcess")
	cmd.Env = append(os.Environ(), "FORGE_GATEWAY_EMIT_LARGE_OUTPUT=1")

	out, err := boundedCombinedOutput(cmd)
	if err != nil {
		t.Fatalf("boundedCombinedOutput returned error: %v", err)
	}
	if !strings.Contains(out, "command output truncated") {
		t.Fatalf("expected truncation marker in output")
	}
	if len(out) > gatewayCommandOutputLimit+128 {
		t.Fatalf("bounded output too large: got %d, limit %d", len(out), gatewayCommandOutputLimit)
	}
}

func TestBoundedCombinedOutputHelperProcess(t *testing.T) {
	if os.Getenv("FORGE_GATEWAY_EMIT_LARGE_OUTPUT") != "1" {
		return
	}
	chunk := strings.Repeat("x", 8192)
	for written := 0; written < gatewayCommandOutputLimit+len(chunk); written += len(chunk) {
		_, _ = os.Stdout.WriteString(chunk)
	}
	os.Exit(0)
}
