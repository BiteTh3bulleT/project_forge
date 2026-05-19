package lanes

import (
	"path/filepath"
	"testing"
)

func TestDefaultBuiltinsHaveUniqueIDs(t *testing.T) {
	seen := map[string]bool{}
	for _, lane := range defaultBuiltins(filepath.Join(t.TempDir(), "workspace"), 1234) {
		if lane.ID == "" {
			t.Fatal("default builtin lane has empty id")
		}
		if seen[lane.ID] {
			t.Fatalf("duplicate default builtin lane id %q", lane.ID)
		}
		seen[lane.ID] = true
	}
}
