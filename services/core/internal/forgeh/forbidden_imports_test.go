package forgeh

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestForbiddenAuthorityImports(t *testing.T) {
	forbidden := []string{
		"forge/projectforge/services/core/internal/api",
		"forge/projectforge/services/core/internal/aios",
		"forge/projectforge/services/core/internal/audit",
		"forge/projectforge/services/core/internal/gateway",
		"forge/projectforge/services/core/internal/forgek",
		"forge/projectforge/services/core/internal/lanes",
		"forge/projectforge/services/core/internal/memory",
		"forge/projectforge/services/core/internal/modelruntime",
		"forge/projectforge/services/core/internal/permissions",
	}
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
			for _, blocked := range forbidden {
				if path == blocked || strings.HasPrefix(path, blocked+"/") {
					t.Fatalf("%s imports forbidden authority package %s", file, path)
				}
			}
		}
	}
}

func TestNoMutationCommandText(t *testing.T) {
	forbidden := []string{
		"exec.Command",
		"nixos-rebuild",
		"systemctl restart",
		"systemctl stop",
		"systemctl start",
		"modprobe",
		"rmmod",
		"apt upgrade",
		"dnf upgrade",
		"pacman -Syu",
		"zypper",
		"RemoveAll",
		"LoadModel",
		"Unload",
		"GenerateStream",
		"WriteFile",
	}
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
		text := string(body)
		for _, blocked := range forbidden {
			if strings.Contains(text, blocked) {
				t.Fatalf("%s contains forbidden mutation/runtime text %q", file, blocked)
			}
		}
	}
}
