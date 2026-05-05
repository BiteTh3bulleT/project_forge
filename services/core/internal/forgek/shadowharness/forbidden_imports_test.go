package shadowharness

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestShadowHarnessPackageDoesNotImportLiveDaemonPackages(t *testing.T) {
	forbidden := []string{
		"forge/projectforge/services/core/internal/api",
		"forge/projectforge/services/core/internal/aios",
		"forge/projectforge/services/core/internal/gateway",
		"forge/projectforge/services/core/internal/permissions",
		"forge/projectforge/services/core/internal/lanes",
		"forge/projectforge/services/core/internal/audit",
		"forge/projectforge/services/core/internal/modelruntime",
		"forge/projectforge/services/core/internal/memory",
		"forge/projectforge/services/core/internal/retrieval",
		"forge/projectforge/services/core/internal/search",
		"forge/projectforge/services/core/internal/embeddings",
	}
	err := filepath.WalkDir(".", func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imp := range file.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)
			for _, blocked := range forbidden {
				if importPath == blocked || strings.HasPrefix(importPath, blocked+"/") {
					t.Fatalf("%s imports forbidden live daemon package %s", path, importPath)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
