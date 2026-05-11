package controllane

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestControlLaneDoesNotImportForgeKSimulatorAuthority(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve test path")
	}
	dir := filepath.Dir(thisFile)
	forbidden := []string{
		"forge/projectforge/services/core/internal/forgek",
	}
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imp := range file.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)
			for _, bad := range forbidden {
				if importPath == bad || strings.HasPrefix(importPath, bad+"/") {
					t.Fatalf("%s imports forbidden simulator authority package %s", path, importPath)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk failed: %v", err)
	}
}
