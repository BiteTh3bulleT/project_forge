package api

import "testing"

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
