package api

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestAPIHandlersUseStructuredErrors(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller unavailable")
	}
	apiDir := filepath.Dir(file)
	var offenders []string
	err := filepath.WalkDir(apiDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(body), "http.Error(") {
			offenders = append(offenders, filepath.Base(path))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan api handlers: %v", err)
	}
	if len(offenders) > 0 {
		t.Fatalf("api handlers must use structured error helpers instead of raw http.Error: %s", strings.Join(offenders, ", "))
	}
}
