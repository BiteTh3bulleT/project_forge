package controllane

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestControlLaneDoesNotImportSimulatorKVService(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		body, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), file, body, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range parsed.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			if path == "forge/projectforge/services/core/internal/forgek/kv" || strings.HasPrefix(path, "forge/projectforge/services/core/internal/forgek/kv/") {
				t.Fatalf("%s imports simulator KV service package %s", file, path)
			}
		}
	}
}
