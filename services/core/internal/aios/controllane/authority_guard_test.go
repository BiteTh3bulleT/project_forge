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
		filepath.Join("services", "core", "internal", "aios", "controllane", "sqlite_store.go"): {},
		filepath.Join("services", "core", "internal", "backup", "service.go"):                   {},
		filepath.Join("services", "core", "internal", "store", "migrate.go"):                    {},
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
