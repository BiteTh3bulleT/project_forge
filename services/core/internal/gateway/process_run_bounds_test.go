package gateway

import (
	"context"
	"errors"
	"runtime"
	"strings"
	"testing"
)

func TestNormalizeProcessRunTimeoutMsIsBounded(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		input map[string]any
		want  int
	}{
		{name: "default", input: nil, want: defaultProcessRunTimeoutMs},
		{name: "negative", input: map[string]any{"timeoutMs": -1.0}, want: defaultProcessRunTimeoutMs},
		{name: "valid", input: map[string]any{"timeoutMs": 1500.0}, want: 1500},
		{name: "oversized", input: map[string]any{"timeoutMs": 9999999.0}, want: maxProcessRunTimeoutMs},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := normalizeProcessRunTimeoutMs(tc.input); got != tc.want {
				t.Fatalf("expected timeout %d, got %d", tc.want, got)
			}
		})
	}
}

func TestNormalizeProcessRunCommandIsBounded(t *testing.T) {
	t.Parallel()
	command, err := normalizeProcessRunCommand(map[string]any{"command": "  go test ./...  "})
	if err != nil {
		t.Fatalf("expected valid command, got %v", err)
	}
	if command != "go test ./..." {
		t.Fatalf("unexpected normalized command %q", command)
	}
	if _, err := normalizeProcessRunCommand(map[string]any{"command": strings.Repeat("x", maxProcessRunCommandBytes+1)}); !errors.Is(err, errProcessRunCommandTooLarge) {
		t.Fatalf("expected command size rejection, got %v", err)
	}
}

func TestBoundedOutputBufferCapsCapturedData(t *testing.T) {
	t.Parallel()
	buf := newBoundedOutputBuffer(8)
	if n, err := buf.Write([]byte(strings.Repeat("x", 20))); err != nil || n != 20 {
		t.Fatalf("write = (%d, %v), want full accepted write", n, err)
	}
	if got := buf.String(); got != "xxxxxxxx" {
		t.Fatalf("unexpected captured output %q", got)
	}
	if !buf.Truncated() {
		t.Fatalf("expected output to be marked truncated")
	}
}

func TestProcessRunReturnsStructuredUnsupportedOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows parity behavior")
	}
	tool := &processRunTool{workspace: t.TempDir()}
	result, err := tool.Execute(context.Background(), Request{Input: map[string]any{"command": "echo hi"}})
	if err != nil {
		t.Fatalf("expected structured unsupported result, got error %v", err)
	}
	if result.Data["unsupported"] != true || result.Data["ok"] != false {
		t.Fatalf("expected unsupported result, got %#v", result.Data)
	}
}
