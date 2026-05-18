package consensusgate

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConsensusGateDoesNotImportLiveOrSimulatorAuthority(t *testing.T) {
	root := "."
	forbidden := []string{
		"services/core/internal/forgek/consensus",
		"services/core/internal/forgek",
		"services/core/internal/aios/controllane",
		"services/core/internal/gateway",
		"services/core/internal/modelruntime",
		"services/core/internal/memory",
		"services/core/internal/retrieval",
	}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, needle := range forbidden {
			if bytes.Contains(data, []byte(needle)) {
				t.Fatalf("consensusgate must remain pure; forbidden import %q in %s", needle, path)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk consensusgate files: %v", err)
	}
}
