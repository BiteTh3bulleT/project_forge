package api

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestMemoryPlaneDirectWritersStayContained prevents live feature code from
// reopening the retired legacy evidence/projection mutation paths.
func TestMemoryPlaneDirectWritersStayContained(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve guard path")
	}
	internalRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), ".."))
	forbidden := []string{
		".RecordObservation(",
		".UpdateObservation(",
		".AddLink(",
		".MarkObservationUsefulness(",
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

func TestProductionContainsNoLegacyMemoryEvidenceSQLWriter(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve guard path")
	}
	internalRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), ".."))
	forbidden := []string{
		"INSERT INTO memory_observations",
		"UPDATE memory_observations",
		"DELETE FROM memory_observations",
		"INSERT INTO memory_observation_links",
		"UPDATE memory_observation_links",
		"DELETE FROM memory_observation_links",
		"INSERT INTO memory_usefulness_events",
		"INSERT INTO memory_repair_runs",
		"INSERT INTO memory_repair_items",
	}
	violations := []string{}
	err := filepath.WalkDir(internalRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, token := range forbidden {
			if strings.Contains(string(body), token) {
				violations = append(violations, filepath.Base(path)+": "+token)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk production sources: %v", err)
	}
	if len(violations) > 0 {
		t.Fatalf("legacy memory SQL writers remain:\n%s", strings.Join(violations, "\n"))
	}
}

func TestLegacyMemoryMutationRoutesAreTerminalRetirementGates(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve guard path")
	}
	apiRoot := filepath.Dir(thisFile)
	routes, err := os.ReadFile(filepath.Join(apiRoot, "routes.go"))
	if err != nil {
		t.Fatal(err)
	}
	routeText := string(routes)
	for _, want := range []string{
		`s.legacyMemoryMutationRetired("legacy.memory.observation.create")`,
		`s.legacyMemoryMutationRetired("legacy.memory.observation.patch")`,
		`s.legacyMemoryMutationRetired("legacy.memory.observation.usefulness")`,
	} {
		if !strings.Contains(routeText, want) {
			t.Fatalf("missing terminal retirement route %q", want)
		}
	}
	for _, forbidden := range []string{
		"handleCreateMemoryObservation",
		"handlePatchMemoryObservation",
		"handleMarkMemoryObservationUsefulness",
		`/memory/observations/{id}/links`,
		`/memory/observation-links`,
	} {
		if strings.Contains(routeText, forbidden) {
			t.Fatalf("legacy memory mutation route remains via %q", forbidden)
		}
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
