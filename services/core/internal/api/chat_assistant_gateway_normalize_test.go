package api

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"forge/projectforge/services/core/internal/gateway"
)

func TestNormalizeChatInvokeArgsInputString(t *testing.T) {
	paths, input := normalizeChatInvokeArgs(map[string]any{
		"input":  "open Konsole",
		"target": "Konsole",
	})
	if len(paths) != 0 {
		t.Fatalf("expected no paths, got %v", paths)
	}
	if input["query"] != "open Konsole" {
		t.Fatalf("expected query to be populated from string input, got %#v", input["query"])
	}
	if input["target"] != "Konsole" {
		t.Fatalf("expected target to remain in input map, got %#v", input["target"])
	}
}

func TestNormalizeChatInvokeArgsPathsStringSlice(t *testing.T) {
	paths, _ := normalizeChatInvokeArgs(map[string]any{
		"paths": []string{"one.txt", "two.txt"},
	})
	if len(paths) != 2 || paths[0] != "one.txt" || paths[1] != "two.txt" {
		t.Fatalf("unexpected paths: %v", paths)
	}
}

func TestNormalizeChatInvokeArgsDownloadsAlias(t *testing.T) {
	paths, _ := normalizeChatInvokeArgs(map[string]any{
		"path": "Downloads/RandomSVGs",
	})
	if len(paths) != 1 || paths[0] != "~/Downloads/RandomSVGs" {
		t.Fatalf("unexpected Downloads alias paths: %v", paths)
	}

	paths, _ = normalizeChatInvokeArgs(map[string]any{
		"paths": []any{"/Downloads/RandomSVGs/turtle.svg"},
	})
	if len(paths) != 1 || paths[0] != "~/Downloads/RandomSVGs/turtle.svg" {
		t.Fatalf("unexpected absolute Downloads alias paths: %v", paths)
	}
}

func TestEnrichDesktopOpenInputFromUser(t *testing.T) {
	got := enrichDesktopOpenInputFromUser("desktop.open", map[string]any{
		"application": "konsole",
	}, "open konsole and ping 10.100.1.5")
	if got["query"] != "open konsole and ping 10.100.1.5" {
		t.Fatalf("expected query fallback from user content, got %#v", got["query"])
	}
}

func TestEnrichDesktopOpenInputFromUserPreservesQuery(t *testing.T) {
	got := enrichDesktopOpenInputFromUser("desktop.open", map[string]any{
		"query": "open konsole",
	}, "open konsole and ping 10.100.1.5")
	if got["query"] != "open konsole" {
		t.Fatalf("expected existing query to be preserved, got %#v", got["query"])
	}
}

func TestPathAllowedRootWorkspaceAllowsSystemPaths(t *testing.T) {
	if !pathAllowed("/", "/home/rshort/test.txt") {
		t.Fatalf("expected root workspace to allow absolute system paths")
	}
	if !pathAllowed("/", "var/log") {
		t.Fatalf("expected root workspace to allow relative paths")
	}
}

func TestPathAllowedNonRootWorkspaceRejectsOutsidePath(t *testing.T) {
	if pathAllowed("/tmp/project", "/etc/passwd") {
		t.Fatalf("expected non-root workspace to reject outside paths")
	}
}

func TestDispatchSuppressesCompositeMkdirBeforeGateway(t *testing.T) {
	srv, _ := newBackupAuditHarness(t)
	home := t.TempDir()
	t.Setenv("HOME", home)

	var stages []string
	user := `Create a directory in Downloads called PeanutButterJellyTime. Inside that folder create an svg file of a flower.`
	got := srv.dispatchToolCall(
		context.Background(),
		"corr-composite-mkdir",
		0,
		gateway.ChatModelName("fs.mkdir"),
		`{"path":"~/Downloads/PeanutButterJellyTime"}`,
		user,
		func(stage string, _ map[string]any) {
			stages = append(stages, stage)
		},
	)

	if got.state != "ok" {
		t.Fatalf("expected suppressed mkdir to return ok for model continuation, got state=%q text=%q", got.state, got.text)
	}
	result, ok := got.executionResult.(map[string]any)
	if !ok || result["suppressed"] != true {
		t.Fatalf("expected suppressed execution result, got %#v", got.executionResult)
	}
	if !containsStage(stages, "composite_mkdir_suppressed") {
		t.Fatalf("expected composite_mkdir_suppressed stage, got %v", stages)
	}
	if _, err := os.Stat(filepath.Join(home, "Downloads", "PeanutButterJellyTime")); !os.IsNotExist(err) {
		t.Fatalf("expected suppressed mkdir to leave no directory, stat err=%v", err)
	}
}

func containsStage(stages []string, want string) bool {
	for _, stage := range stages {
		if stage == want {
			return true
		}
	}
	return false
}
