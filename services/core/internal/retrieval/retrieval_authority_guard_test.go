package retrieval

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

func TestRetrievalEvidenceHasOneProductionWriter(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source")
	}
	internalDir := filepath.Dir(filepath.Dir(currentFile))
	adapter := filepath.Join(internalDir, "aios", "controllane", "sqlite_store_retrieval.go")
	patterns := map[string]*regexp.Regexp{
		"retrieval_runs":             regexp.MustCompile(`(?i)INSERT\s+INTO\s+retrieval_runs\s*\(`),
		"retrieval_results":          regexp.MustCompile(`(?i)INSERT\s+INTO\s+retrieval_results\s*\(`),
		"retrieval_result_selection": regexp.MustCompile(`(?i)INSERT\s+INTO\s+retrieval_result_selection\s*\(`),
		"packet_retrieval_runs":      regexp.MustCompile(`(?i)INSERT\s+INTO\s+packet_retrieval_runs\s*\(`),
	}
	counts := make(map[string]int, len(patterns))
	var violations []string
	err := filepath.WalkDir(internalDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		text := string(raw)
		for table, pattern := range patterns {
			matches := len(pattern.FindAllStringIndex(text, -1))
			if matches == 0 {
				continue
			}
			counts[table] += matches
			if path != adapter {
				violations = append(violations, path+": "+table)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan production sources: %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("retrieval authority bypasses found: %+v", violations)
	}
	for table := range patterns {
		if counts[table] != 1 {
			t.Fatalf("writer %q count=%d, want exactly one", table, counts[table])
		}
	}
}
