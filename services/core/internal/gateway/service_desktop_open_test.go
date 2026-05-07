package gateway

import (
	"runtime"
	"strings"
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

func TestDesktopNormalizeAppHintPoliteDesktopRequests(t *testing.T) {
	cases := map[string]string{
		"Can you open file explorer please": "file explorer",
		"Open my file explorer":             "file explorer",
		"Open google chrome please":         "google chrome",
	}
	for in, want := range cases {
		if got := desktopNormalizeAppHint(in); got != want {
			t.Fatalf("desktopNormalizeAppHint(%q)=%q want %q", in, got, want)
		}
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

func TestDesktopLaunchCandidatesTerminalOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-specific launcher candidates")
	}
	candidates := desktopLaunchCandidates("can you open a terminal please")
	if len(candidates) == 0 {
		t.Fatalf("expected launch candidates for terminal")
	}
	for _, candidate := range candidates {
		if len(candidate) == 0 {
			t.Fatalf("empty candidate in terminal candidates: %#v", candidates)
		}
		switch candidate[0] {
		case "wt.exe", "powershell.exe", "cmd.exe":
		default:
			t.Fatalf("expected Windows terminal candidate, got %q in %#v", candidate[0], candidates)
		}
	}
}

func TestDesktopLaunchCandidatesExplorerAndChromeOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-specific launcher candidates")
	}
	explorerCandidates := desktopLaunchCandidates("file explorer")
	if len(explorerCandidates) == 0 || explorerCandidates[0][0] != "explorer.exe" {
		t.Fatalf("expected explorer.exe candidate, got %#v", explorerCandidates)
	}
	chromeCandidates := desktopLaunchCandidates("google chrome")
	if len(chromeCandidates) == 0 || chromeCandidates[0][0] != "chrome.exe" {
		t.Fatalf("expected chrome.exe candidate, got %#v", chromeCandidates)
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

func TestDesktopSplitAppAndCommandGenericTerminalRun(t *testing.T) {
	app, cmd := desktopSplitAppAndCommand("Open terminal and run sudo zypper refresh")
	if app != "terminal" {
		t.Fatalf("expected app terminal, got %q", app)
	}
	want := []string{"sudo", "zypper", "refresh"}
	if len(cmd) != len(want) {
		t.Fatalf("command length = %d, want %d: %v", len(cmd), len(want), cmd)
	}
	for i := range want {
		if cmd[i] != want[i] {
			t.Fatalf("cmd[%d] = %q, want %q (full %v)", i, cmd[i], want[i], cmd)
		}
	}
}

func TestDesktopSplitAppAndCommandSSHRemoteMkdir(t *testing.T) {
	app, cmd := desktopSplitAppAndCommand("Open terminall, ssh into robert@10.150.1.9 password redacted-secret. Create a directory labled SSH-AI-TEST")
	if app != "terminal" {
		t.Fatalf("expected app terminal, got %q", app)
	}
	want := []string{"ssh", "robert@10.150.1.9", "mkdir", "-p", "SSH-AI-TEST"}
	if len(cmd) != len(want) {
		t.Fatalf("command length = %d, want %d: %v", len(cmd), len(want), cmd)
	}
	for i := range want {
		if cmd[i] != want[i] {
			t.Fatalf("cmd[%d] = %q, want %q (full %v)", i, cmd[i], want[i], cmd)
		}
	}
	for _, arg := range cmd {
		if strings.Contains(arg, "redacted-secret") {
			t.Fatalf("password must not be placed into terminal command args: %v", cmd)
		}
	}
}

func TestDesktopSplitAppAndCommandSSHRemotePythonBanner(t *testing.T) {
	app, cmd := desktopSplitAppAndCommand(`Open terminall, ssh into robert@10.150.1.2 password redacted-secret. Create a directory labled Auto_Banner. Inside that directory create a python program called hello_world.py. I want it to be a scrolling flashing banner with the words "HELLO WORLD".`)
	if app != "terminal" {
		t.Fatalf("expected app terminal, got %q", app)
	}
	joined := strings.Join(cmd, " ")
	required := []string{
		"ssh robert@10.150.1.2",
		"mkdir -p 'Auto_Banner'",
		"cat > 'Auto_Banner/hello_world.py'",
		"HELLO WORLD",
		"python3 'Auto_Banner/hello_world.py'",
	}
	for _, want := range required {
		if !strings.Contains(joined, want) {
			t.Fatalf("command missing %q: %v", want, cmd)
		}
	}
	for _, arg := range cmd {
		if strings.Contains(arg, "redacted-secret") {
			t.Fatalf("password must not be placed into terminal command args: %v", cmd)
		}
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

func TestDesktopInlineCommandFromInputGenericTerminalRun(t *testing.T) {
	cmd := desktopInlineCommandFromInput(map[string]any{
		"application": "terminal",
		"query":       "Open terminal and run sudo zypper refresh",
	})
	want := []string{"sudo", "zypper", "refresh"}
	if len(cmd) != len(want) {
		t.Fatalf("command length = %d, want %d: %v", len(cmd), len(want), cmd)
	}
	for i := range want {
		if cmd[i] != want[i] {
			t.Fatalf("cmd[%d] = %q, want %q (full %v)", i, cmd[i], want[i], cmd)
		}
	}
}
