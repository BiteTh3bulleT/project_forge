package permissions

import (
	"context"
	"path/filepath"
	"testing"

	"forge/projectforge/services/core/internal/store"
)

func TestEnsureGatewayToolPolicyUpgradesStandardProfile(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	workspace := t.TempDir()
	dataDir := t.TempDir()

	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	svc := New(st.DB)
	if err := svc.EnsureDefaults(ctx, workspace); err != nil {
		t.Fatalf("ensure defaults: %v", err)
	}
	if err := svc.EnsureMkdirChatPolicy(ctx, workspace); err != nil {
		t.Fatalf("ensure mkdir policy: %v", err)
	}
	if err := svc.EnsureGatewayToolPolicy(ctx, workspace); err != nil {
		t.Fatalf("ensure gateway policy: %v", err)
	}

	standard, err := svc.Get(ctx, "standard")
	if err != nil {
		t.Fatalf("load standard profile: %v", err)
	}
	if standard == nil {
		t.Fatalf("standard profile missing")
	}
	if !contains(standard.AllowedExecutePaths, filepath.Clean(workspace)) {
		t.Fatalf("expected workspace execute scope in standard profile: %v", standard.AllowedExecutePaths)
	}
	if !contains(standard.AllowedReadPaths, filepath.Clean(workspace)) {
		t.Fatalf("expected workspace read scope in standard profile: %v", standard.AllowedReadPaths)
	}
	for _, tool := range []string{"proc.run", "git.status", "fs.write", "net.fetch"} {
		if !contains(standard.AllowedTools, tool) {
			t.Fatalf("expected tool %q in standard allowlist", tool)
		}
	}
}
