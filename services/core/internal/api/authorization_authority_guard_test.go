package api

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestProductionAuthorizationConstructionAndOriginAuthorityAreUnique(t *testing.T) {
	coreRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve core root: %v", err)
	}
	wanted := map[string][]string{
		"NewForgeCoreServicePrincipal":      nil,
		"NewProductionAuthorizationService": nil,
		"WithTrustedOrigin":                 nil,
	}
	fset := token.NewFileSet()
	err = filepath.WalkDir(coreRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		parsed, parseErr := parser.ParseFile(fset, path, nil, 0)
		if parseErr != nil {
			return parseErr
		}
		ast.Inspect(parsed, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			name := ""
			switch fn := call.Fun.(type) {
			case *ast.Ident:
				name = fn.Name
			case *ast.SelectorExpr:
				name = fn.Sel.Name
			}
			if _, tracked := wanted[name]; tracked {
				relative, relErr := filepath.Rel(coreRoot, path)
				if relErr != nil {
					relative = path
				}
				wanted[name] = append(wanted[name], filepath.ToSlash(relative))
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("scan production Go sources: %v", err)
	}

	for _, constructor := range []string{"NewForgeCoreServicePrincipal", "NewProductionAuthorizationService"} {
		calls := wanted[constructor]
		if len(calls) != 1 || calls[0] != "internal/api/server.go" {
			t.Fatalf("%s production callsites=%v, want exactly [internal/api/server.go]", constructor, calls)
		}
	}
	for _, callsite := range wanted["WithTrustedOrigin"] {
		if callsite != "internal/api/auth.go" {
			t.Fatalf("trusted origin minted outside API authentication: %v", wanted["WithTrustedOrigin"])
		}
	}
	if len(wanted["WithTrustedOrigin"]) != 2 {
		t.Fatalf("trusted-origin production callsites=%v, want bearer and loopback paths in api/auth.go", wanted["WithTrustedOrigin"])
	}

	serverPath := filepath.Join(coreRoot, "internal", "api", "server.go")
	source, err := os.ReadFile(serverPath)
	if err != nil {
		t.Fatalf("read server assembly: %v", err)
	}
	serverSource := string(source)
	registryBindings := regexp.MustCompile(`Registry:\s+actionRegistry`).FindAllString(serverSource, -1)
	if strings.Count(serverSource, "controllane.NewStaticActionRegistry()") != 1 ||
		len(registryBindings) != 2 {
		t.Fatalf("production commit adapter and authorization resolver must share one actionRegistry construction")
	}
}
