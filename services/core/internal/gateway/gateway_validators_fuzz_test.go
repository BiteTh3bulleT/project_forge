package gateway

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fuzzOutboundResolver struct{}

func (fuzzOutboundResolver) LookupIPAddr(context.Context, string) ([]net.IPAddr, error) {
	return []net.IPAddr{{IP: net.ParseIP("93.184.216.34")}}, nil
}

func FuzzValidateOutboundHTTPURL(f *testing.F) {
	for _, seed := range []string{
		"https://example.com",
		"http://example.com/path?q=1",
		"ftp://example.com",
		"https://user@example.com",
		"http://127.0.0.1",
		"http://10.0.0.1",
		"http://[::1]/",
		" https://example.com/trimmed ",
		"",
		strings.Repeat("a", maxOutboundHTTPURLBytes+1),
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, raw string) {
		parsed, err := validateOutboundHTTPURL(context.Background(), raw, fuzzOutboundResolver{})
		if err != nil {
			return
		}
		if parsed == nil {
			t.Fatalf("accepted URL returned nil parser")
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			t.Fatalf("accepted URL with unsafe scheme %q from %q", parsed.Scheme, raw)
		}
		if parsed.User != nil {
			t.Fatalf("accepted URL with userinfo from %q", raw)
		}
		if strings.TrimSpace(parsed.Hostname()) == "" {
			t.Fatalf("accepted URL without host from %q", raw)
		}
		if len(strings.TrimSpace(raw)) > maxOutboundHTTPURLBytes {
			t.Fatalf("accepted oversized URL with %d bytes", len(strings.TrimSpace(raw)))
		}
		if ip := net.ParseIP(parsed.Hostname()); ip != nil && blockedOutboundIP(ip) {
			t.Fatalf("accepted blocked IP literal %q from %q", parsed.Hostname(), raw)
		}
	})
}

func FuzzFirstWorkspacePath(f *testing.F) {
	workspace := f.TempDir()
	if err := os.MkdirAll(filepath.Join(workspace, "nested"), 0o755); err != nil {
		f.Fatalf("create workspace seed dir: %v", err)
	}
	for _, seed := range []string{
		"file.txt",
		"nested/file.txt",
		".",
		"",
		"../escape.txt",
		filepath.Join(workspace, "absolute-inside.txt"),
		filepath.Dir(workspace),
		" nested/trimmed.txt ",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, raw string) {
		got, err := firstWorkspacePath([]string{raw}, workspace)
		if err != nil {
			return
		}
		if !filepath.IsAbs(got) {
			t.Fatalf("accepted path is not absolute: %q", got)
		}
		if got != filepath.Clean(got) {
			t.Fatalf("accepted path is not clean: %q", got)
		}
		if !pathContains(workspace, got) {
			t.Fatalf("accepted path %q outside workspace %q from %q", got, workspace, raw)
		}
		if err := validateWorkspacePath(workspace, got); err != nil {
			t.Fatalf("accepted path failed workspace validation: %v", err)
		}
	})
}

func FuzzNormalizeChmodMode(f *testing.F) {
	for _, seed := range []string{"", "0", "0000", "644", "0755", "777", "1000", "7777", "888", "abc", " 0644 "} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, raw string) {
		modeRaw, mode, err := normalizeChmodMode(raw)
		if err != nil {
			return
		}
		if mode > 0o777 {
			t.Fatalf("accepted chmod mode with unsupported bits: raw=%q normalized=%q mode=%#o", raw, modeRaw, mode)
		}
		if strings.TrimSpace(raw) == "" {
			if modeRaw != "0644" || mode != 0o644 {
				t.Fatalf("empty mode default = (%q, %#o), want (0644, 0644)", modeRaw, mode)
			}
			return
		}
		if modeRaw != strings.TrimSpace(raw) {
			t.Fatalf("mode raw = %q, want trimmed input %q", modeRaw, strings.TrimSpace(raw))
		}
	})
}

func FuzzNormalizeGitCheckoutRef(f *testing.F) {
	for _, seed := range []string{
		"main",
		"feature/shell-surfaces",
		"release-2026.05.10",
		"abc123def456",
		"HEAD~1",
		"--detach",
		"feature/../main",
		"feature@{1}",
		"refs/heads/main.lock",
		`feature\main`,
		strings.Repeat("a", 257),
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, raw string) {
		ref, err := normalizeGitCheckoutRef(raw)
		if err != nil {
			return
		}
		if ref != strings.TrimSpace(raw) {
			t.Fatalf("ref = %q, want trimmed input %q", ref, strings.TrimSpace(raw))
		}
		if ref == "" || len(ref) > 256 {
			t.Fatalf("accepted empty or oversized ref %q", ref)
		}
		if strings.HasPrefix(ref, "-") ||
			strings.Contains(ref, "..") ||
			strings.Contains(ref, "@{") ||
			strings.Contains(ref, `\`) ||
			strings.HasSuffix(ref, ".lock") ||
			!gitCheckoutRefPattern.MatchString(ref) {
			t.Fatalf("accepted unsafe ref %q from %q", ref, raw)
		}
	})
}

func FuzzNormalizeTerminatePID(f *testing.F) {
	for _, seed := range []float64{-1, 0, 1, 2, 12345, float64(os.Getpid()), 1.5} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, raw float64) {
		pid, err := normalizeTerminatePID(raw)
		if err != nil {
			return
		}
		if raw != float64(int(raw)) {
			t.Fatalf("accepted non-integer pid %v as %d", raw, pid)
		}
		if pid <= 0 || pid == 1 || pid == os.Getpid() {
			t.Fatalf("accepted unsafe pid %d from %v", pid, raw)
		}
		if pid != int(raw) {
			t.Fatalf("pid = %d, want %d", pid, int(raw))
		}
	})
}
