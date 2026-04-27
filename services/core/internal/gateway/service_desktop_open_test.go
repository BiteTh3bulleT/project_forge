package gateway

import (
	"runtime"
	"testing"
)

func TestDesktopInputCandidate(t *testing.T) {
	input := map[string]any{
		"query":       "open konsole",
		"application": "konsole",
	}
	got := desktopInputCandidate(input)
	if got != "konsole" {
		t.Fatalf("expected application candidate, got %q", got)
	}
}

func TestDesktopNormalizeAppHint(t *testing.T) {
	got := desktopNormalizeAppHint("Open the Software Center.")
	if got != "software center" {
		t.Fatalf("expected normalized hint %q, got %q", "software center", got)
	}
}

func TestDesktopLaunchCandidates(t *testing.T) {
	candidates := desktopLaunchCandidates("software center")
	if len(candidates) == 0 {
		t.Fatalf("expected launch candidates for software center")
	}
	if candidates[0][0] != "plasma-discover" {
		t.Fatalf("expected first candidate plasma-discover, got %q", candidates[0][0])
	}
}

func TestDesktopLaunchCandidatesMinecraftOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-specific launcher candidates")
	}
	candidates := desktopLaunchCandidates("minecraft")
	if len(candidates) == 0 {
		t.Fatalf("expected launch candidates for minecraft")
	}
	if candidates[0][0] != "minecraft:" {
		t.Fatalf("expected first Minecraft candidate to use URI launcher, got %q", candidates[0][0])
	}
}

func TestDesktopSafeCommandToken(t *testing.T) {
	if !desktopSafeCommandToken("konsole") {
		t.Fatalf("expected konsole token to be safe")
	}
	if desktopSafeCommandToken("konsole;rm") {
		t.Fatalf("expected token with shell metacharacters to be unsafe")
	}
}

func TestDesktopLooksLikeURLAndPath(t *testing.T) {
	if !desktopLooksLikeURL("https://example.com") {
		t.Fatalf("expected https URL to match")
	}
	if desktopLooksLikeURL("konsole") {
		t.Fatalf("expected plain app hint to not match URL")
	}
	if !desktopLooksLikePath("./notes/todo.md") {
		t.Fatalf("expected relative path to match")
	}
	if desktopLooksLikePath("software center") {
		t.Fatalf("expected app hint to not match path")
	}
}

func TestDesktopSplitAppAndCommand(t *testing.T) {
	app, cmd := desktopSplitAppAndCommand("open konsole and ping 10.100.1.5")
	if app != "konsole" {
		t.Fatalf("expected app konsole, got %q", app)
	}
	if len(cmd) != 2 || cmd[0] != "ping" || cmd[1] != "10.100.1.5" {
		t.Fatalf("unexpected inline command: %v", cmd)
	}
}

func TestDesktopTerminalLaunchArgs(t *testing.T) {
	args, ok := desktopTerminalLaunchArgs("konsole", []string{"ping", "10.100.1.5"})
	if !ok {
		t.Fatalf("expected konsole to support inline command args")
	}
	if len(args) != 4 || args[0] != "--noclose" || args[1] != "-e" || args[2] != "ping" || args[3] != "10.100.1.5" {
		t.Fatalf("unexpected konsole args: %v", args)
	}
}

func TestDesktopSafePingTarget(t *testing.T) {
	if !desktopSafePingTarget("10.100.1.5") {
		t.Fatalf("expected ipv4 target to be safe")
	}
	if desktopSafePingTarget("10.100.1.5;rm") {
		t.Fatalf("expected shell-tainted target to be unsafe")
	}
}

func TestDesktopInlineCommandFromInput(t *testing.T) {
	cmd := desktopInlineCommandFromInput(map[string]any{
		"application": "konsole",
		"query":       "open konsole and ping 10.100.1.5",
	})
	if len(cmd) != 2 || cmd[0] != "ping" || cmd[1] != "10.100.1.5" {
		t.Fatalf("expected ping command from query fallback, got %v", cmd)
	}
}
