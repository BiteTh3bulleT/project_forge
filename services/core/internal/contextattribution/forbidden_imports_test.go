package contextattribution

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContextAttributionPackageDoesNotImportLiveAuthorityOrSimulator(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob package files: %v", err)
	}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		body, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		source := string(body)
		for _, forbidden := range []string{
			"internal/forgek",
			"internal/aios/controllane",
			"internal/modelruntime",
			"internal/gateway",
			"internal/memory",
			"internal/retrieval",
			"database/sql",
			"net/http",
		} {
			if strings.Contains(source, forbidden) {
				t.Fatalf("%s imports forbidden authority dependency %q", file, forbidden)
			}
		}
	}
}
