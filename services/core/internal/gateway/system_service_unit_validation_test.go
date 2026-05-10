package gateway

import (
	"context"
	"strings"
	"testing"
)

func TestNormalizeSystemdUnitNameRejectsUnsafeValues(t *testing.T) {
	t.Parallel()
	for _, name := range []string{
		"--system",
		"-u",
		"ssh.service --all",
		"ssh.service\nother.service",
		"../ssh.service",
		"ssh/service",
		strings.Repeat("a", 257) + ".service",
	} {
		if _, err := normalizeSystemdUnitName(name); err == nil {
			t.Fatalf("expected unsafe unit name %q to be rejected", name)
		}
	}
}

func TestSystemServiceToolsRejectUnsafeUnitNamesBeforeExecution(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		tool  Tool
		input map[string]any
	}{
		{
			name:  "service status",
			tool:  &serviceStatusTool{},
			input: map[string]any{"service": "--system"},
		},
		{
			name:  "service control",
			tool:  &serviceControlTool{},
			input: map[string]any{"service": "--system", "control": "restart"},
		},
		{
			name:  "journal unit",
			tool:  &journalTailTool{},
			input: map[string]any{"service": "ssh.service --all", "lines": 10},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, err := tc.tool.Execute(context.Background(), Request{Input: tc.input}); err == nil {
				t.Fatalf("expected %s to reject unsafe unit name", tc.name)
			}
		})
	}
}

func TestNormalizeSystemdUnitNameAllowsBoundedUnitNames(t *testing.T) {
	t.Parallel()
	for _, name := range []string{
		"ssh",
		"ssh.service",
		"forge-core.service",
		"getty@tty1.service",
		"user-runtime-dir@1000.service",
		"dev-disk-by\\x2duuid-abc.mount",
		"dbus-org.freedesktop.resolve1.service",
	} {
		got, err := normalizeSystemdUnitName(" " + name + " ")
		if err != nil {
			t.Fatalf("expected unit name %q to be accepted: %v", name, err)
		}
		if got != name {
			t.Fatalf("expected normalized unit %q, got %q", name, got)
		}
	}
}

func TestNormalizeJournalTailLinesIsBounded(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		input map[string]any
		want  int
	}{
		{name: "default", input: nil, want: defaultJournalTailLines},
		{name: "negative", input: map[string]any{"lines": -10.0}, want: defaultJournalTailLines},
		{name: "valid", input: map[string]any{"lines": 25.0}, want: 25},
		{name: "oversized", input: map[string]any{"lines": 50000.0}, want: maxJournalTailLines},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := normalizeJournalTailLines(tc.input); got != tc.want {
				t.Fatalf("expected %d lines, got %d", tc.want, got)
			}
		})
	}
}
