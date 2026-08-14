package retrieval

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestUtilityEvidenceHasNoProductionDirectWriter(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source")
	}
	internalDir := filepath.Dir(filepath.Dir(currentFile))
	utilityAdapter := filepath.Join(internalDir, "aios", "controllane", "sqlite_store_utility_evidence.go")
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
		if strings.Contains(text, "UPDATE retrieval_results SET usefulness") {
			violations = append(violations, path+": mutates immutable retrieval evidence projection columns")
		}
		if strings.Contains(text, "UpdateRestoreOutcomeFeedback(") || strings.Contains(text, "UPDATE restore_outcome_events") {
			violations = append(violations, path+": retains mutable restore outcome feedback authority")
		}
		if path != utilityAdapter && (strings.Contains(text, "INSERT INTO forge_k_retrieval_usefulness_events") ||
			strings.Contains(text, "INSERT INTO forge_k_restore_outcome_feedback_events") ||
			strings.Contains(text, "INSERT INTO retrieval_usefulness_projection") ||
			strings.Contains(text, "INSERT INTO restore_outcome_feedback_projection")) {
			violations = append(violations, path+": utility event/projection writer outside K adapter")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan production sources: %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("utility authority bypasses found: %+v", violations)
	}

	serviceSource, err := os.ReadFile(filepath.Join(internalDir, "retrieval", "service.go"))
	if err != nil {
		t.Fatalf("read retrieval service: %v", err)
	}
	serviceText := string(serviceSource)
	for _, forbidden := range []string{"MarkObservationUsefulness(", "INSERT INTO context_evidence"} {
		if strings.Contains(serviceText, forbidden) {
			t.Fatalf("retrieval service retains direct utility writer %q", forbidden)
		}
	}
}
