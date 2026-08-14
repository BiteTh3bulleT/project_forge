package controllane

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// TestCanonicalCognitiveWritesStayBounded guards Phase 1 authority convergence:
// canonical AI-OS cognitive tables should not gain new direct write paths
// outside the syscall control-lane store (and bounded restore/migration paths).
func TestCanonicalCognitiveWritesStayBounded(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("failed to resolve caller path")
	}

	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", ".."))
	internalRoot := filepath.Join(repoRoot, "services", "core", "internal")

	allowed := map[string]struct{}{
		filepath.Join("services", "core", "internal", "aios", "controllane", "sqlite_store.go"):         {},
		filepath.Join("services", "core", "internal", "aios", "controllane", "sqlite_store_compat.go"):  {},
		filepath.Join("services", "core", "internal", "aios", "controllane", "sqlite_store_context.go"): {},
		filepath.Join("services", "core", "internal", "aios", "controllane", "sqlite_store_helpers.go"): {},
		filepath.Join("services", "core", "internal", "aios", "controllane", "sqlite_store_journal.go"): {},
		filepath.Join("services", "core", "internal", "aios", "controllane", "sqlite_store_notes.go"):   {},
		filepath.Join("services", "core", "internal", "aios", "controllane", "sqlite_store_objects.go"): {},
		filepath.Join("services", "core", "internal", "aios", "controllane", "sqlite_store_state.go"):   {},
		filepath.Join("services", "core", "internal", "backup", "service.go"):                           {},
		filepath.Join("services", "core", "internal", "backup", "section_mappings.go"):                  {},
		filepath.Join("services", "core", "internal", "store", "migrate_columns.go"):                    {},
		filepath.Join("services", "core", "internal", "store", "migrate.go"):                            {},
	}

	// Canonical cognitive filesystem tables that must remain kernel-governed.
	rx := regexp.MustCompile(`(?i)\b((?:insert(?:\s+or\s+(?:replace|ignore|abort|fail|rollback))?\s+into)|update|delete\s+from)\s+` +
		`(memory_notes|semantic_links|state_items|state_versions|open_loops|contradiction_records|` +
		`supersession_records|derived_models|context_packet_snapshots|artifact_refs|provenance_records|journal_events)\b`)

	violations := []string{}
	err := filepath.WalkDir(internalRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return err
		}
		rel = filepath.Clean(rel)
		if _, ok := allowed[rel]; ok {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		matches := rx.FindAllStringSubmatch(string(body), -1)
		for _, m := range matches {
			if len(m) < 3 {
				continue
			}
			violations = append(violations, rel+": "+strings.ToUpper(strings.TrimSpace(m[1]))+" "+strings.TrimSpace(m[2]))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("authority guard walk failed: %v", err)
	}

	if len(violations) > 0 {
		t.Fatalf("found direct canonical cognitive write paths outside allowed kernel/restore boundaries:\n%s", strings.Join(violations, "\n"))
	}
}

// TestKernelProcessorHasSingleConstructionSite guards the durable adapter
// invariant: the legacy Control Lane processor is assembled exactly once at
// daemon bootstrap, then selected behind the production FORGE-K authority.
// It must not become a parallel write root in feature code.
func TestKernelProcessorHasSingleConstructionSite(t *testing.T) {
	expected := []string{
		filepath.Join("services", "core", "internal", "api", "server.go"),
	}
	assertProductionCallSites(t, `\bcontrollane\.NewProcessor\b`, expected, "controllane.NewProcessor")
}

// TestGatewayHasSingleConstructionSite mirrors the kernel invariant for the
// tool execution write root. `gateway.New` must be called exactly once in
// production, from `api/server.go`. Tools register *into* that gateway; they
// do not stand up their own.
func TestGatewayHasSingleConstructionSite(t *testing.T) {
	expected := []string{
		filepath.Join("services", "core", "internal", "api", "server.go"),
	}
	assertProductionCallSites(t, `\bgateway\.New\(`, expected, "gateway.New")
}

func assertProductionCallSites(t *testing.T, pattern string, expected []string, label string) {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("failed to resolve caller path")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", ".."))
	internalRoot := filepath.Join(repoRoot, "services", "core", "internal")

	allowed := make(map[string]struct{}, len(expected))
	for _, p := range expected {
		allowed[filepath.Clean(p)] = struct{}{}
	}

	rx := regexp.MustCompile(pattern)

	found := map[string]struct{}{}
	err := filepath.WalkDir(internalRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return err
		}
		rel = filepath.Clean(rel)
		// The kernel and gateway packages may reference their own constructors
		// in docstrings and unexported helpers; we only police *external*
		// callers, since intra-package construction can't open a parallel
		// write root.
		if strings.HasPrefix(rel, filepath.Join("services", "core", "internal", "aios", "controllane")) ||
			strings.HasPrefix(rel, filepath.Join("services", "core", "internal", "gateway")) {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !rx.Match(body) {
			return nil
		}
		found[rel] = struct{}{}
		return nil
	})
	if err != nil {
		t.Fatalf("authority guard walk for %s failed: %v", label, err)
	}

	for site := range found {
		if _, ok := allowed[site]; !ok {
			t.Fatalf("%s called from unexpected production site %s; if this is intentional, update the allow-list in authority_guard_test.go and document the new write root in docs/status/duplicate_systems.md", label, site)
		}
	}
	for _, site := range expected {
		if _, ok := found[site]; !ok {
			t.Fatalf("%s expected at %s but no call site found; the pinned construction site has moved — update the allow-list to match the new bootstrap path", label, site)
		}
	}
}
