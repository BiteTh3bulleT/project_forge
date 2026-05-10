package gateway

import (
	"context"
	"net"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestReadFileToolRejectsNonRegularFiles(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix socket fixture is not portable to windows")
	}
	workspace := t.TempDir()
	socketPath := filepath.Join(workspace, "tool.sock")
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("listen on unix socket fixture: %v", err)
	}
	defer listener.Close()

	tool := &readFileTool{workspace: workspace}
	_, err = tool.Execute(context.Background(), Request{Paths: []string{"tool.sock"}})
	if err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("Execute error = %v, want non-regular file error", err)
	}
}
