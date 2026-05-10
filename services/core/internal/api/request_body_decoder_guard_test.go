package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProductionAPIHandlersDoNotDecodeRequestBodyDirectly(t *testing.T) {
	t.Parallel()

	var offenders []string
	err := filepath.WalkDir(".", func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		text := string(raw)
		for _, pattern := range []string{
			"json.NewDecoder(r.Body)",
			"NewDecoder(r.Body)",
			"io.ReadAll(r.Body)",
			"ReadAll(r.Body)",
		} {
			if strings.Contains(text, pattern) {
				offenders = append(offenders, path+" contains "+pattern)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan api sources: %v", err)
	}
	if len(offenders) > 0 {
		t.Fatalf("production API handlers must use bounded request body helpers:\n%s", strings.Join(offenders, "\n"))
	}
}
