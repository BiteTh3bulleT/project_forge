package api

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRoutesDoNotUseRawURLMiddlewareLogger(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller unavailable")
	}
	body, err := os.ReadFile(filepath.Join(filepath.Dir(file), "routes.go"))
	if err != nil {
		t.Fatalf("read routes source: %v", err)
	}
	if strings.Contains(string(body), "middleware.Logger") {
		t.Fatal("routes must not use middleware.Logger because it logs raw URLs including query strings")
	}
}
