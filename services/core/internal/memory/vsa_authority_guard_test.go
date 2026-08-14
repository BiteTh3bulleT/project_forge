package memory

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestLegacyMemoryPackageCannotWriteVSAProjection ensures the memory service
// remains a read/preview consumer. The sole projection writer is the scoped
// FORGE-K SQLite commit adapter in aios/controllane.
func TestLegacyMemoryPackageCannotWriteVSAProjection(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	forbidden := []string{
		"insert into memory_vsa_",
		"update memory_vsa_",
		"delete from memory_vsa_",
		"insert into memory_usefulness_events",
		"update memory_usefulness_events",
		"delete from memory_usefulness_events",
	}
	violations := []string{}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), entry.Name(), nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", entry.Name(), err)
		}
		ast.Inspect(parsed, func(node ast.Node) bool {
			literal, ok := node.(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				return true
			}
			value, err := strconv.Unquote(literal.Value)
			if err != nil {
				return true
			}
			normalized := strings.ToLower(strings.Join(strings.Fields(value), " "))
			for _, token := range forbidden {
				if strings.Contains(normalized, token) {
					violations = append(violations, entry.Name()+": "+token)
				}
			}
			return true
		})
	}
	if len(violations) != 0 {
		t.Fatalf("legacy memory projection/evidence writers detected: %v", violations)
	}
}
