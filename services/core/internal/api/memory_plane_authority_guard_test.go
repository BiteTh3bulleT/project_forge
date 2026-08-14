package api

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestMemoryPlaneDirectWritersStayContained prevents live feature code from
// reopening the K20F evidence/projection mutation paths. The memory package
// retains implementation methods for historical compatibility and focused
// tests, but production code may not invoke those mutating entry points.
func TestMemoryPlaneDirectWritersStayContained(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve guard path")
	}
	internalRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), ".."))
	forbidden := []string{
		".RunRepairPass(",
		".RunVSAReindex(",
		".LinkResultObservation(",
		".TouchVSAReliabilityFromUsefulness(",
	}
	violations := []string{}
	err := filepath.WalkDir(internalRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, token := range forbidden {
			if strings.Contains(string(body), token) {
				rel, _ := filepath.Rel(internalRoot, path)
				if token == ".TouchVSAReliabilityFromUsefulness(" && filepath.ToSlash(rel) == "memory/retrieval.go" {
					continue
				}
				violations = append(violations, filepath.ToSlash(rel)+": "+token)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk production sources: %v", err)
	}
	if len(violations) > 0 {
		t.Fatalf("direct memory evidence/projection writers escaped containment:\n%s", strings.Join(violations, "\n"))
	}
}

func TestRetrievalDoesNotCreateLegacyObservationGraph(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve guard path")
	}
	servicePath := filepath.Join(filepath.Dir(thisFile), "..", "retrieval", "service.go")
	body, err := os.ReadFile(servicePath)
	if err != nil {
		t.Fatalf("read retrieval service: %v", err)
	}
	for _, token := range []string{".RecordObservation(", ".LinkResultObservation(", "INSERT INTO memory_observations", "INSERT INTO retrieval_result_observations"} {
		if strings.Contains(string(body), token) {
			t.Fatalf("retrieval service reopened legacy observation mutation via %q", token)
		}
	}
}
