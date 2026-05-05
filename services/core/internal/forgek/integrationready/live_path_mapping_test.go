package integrationready

import "testing"

func TestDefaultLivePathMappingsDefineAuthorityAndForbidMutation(t *testing.T) {
	mappings := DefaultLivePathMappings()
	if len(mappings) != 14 {
		t.Fatalf("expected 14 live path mappings, got %d", len(mappings))
	}
	for _, mapping := range mappings {
		if err := ValidateLivePathMapping(mapping); err != nil {
			t.Fatalf("mapping %s should validate: %v", mapping.LiveSystem, err)
		}
		if mapping.CurrentAuthorityOwner == "" || mapping.ForgeKTargetComponent == "" || mapping.RequiredAdapter == "" || len(mapping.RequiredTests) == 0 {
			t.Fatalf("mapping missing required integration readiness fields: %#v", mapping)
		}
		if mapping.LiveMutationAllowed {
			t.Fatalf("%s must not allow live mutation in Phase 11F", mapping.LiveSystem)
		}
	}
}

func TestReadinessMatrixCoversAllSubsystems(t *testing.T) {
	statuses := DefaultSubsystemStatuses()
	want := map[string]bool{
		SubsystemKernel:          false,
		SubsystemNeuronFabric:    false,
		SubsystemCourthouse:      false,
		SubsystemMemoryPalace:    false,
		SubsystemSemanticAlgebra: false,
		SubsystemSnapshots:       false,
		SubsystemContextCompiler: false,
		SubsystemKVSystem:        false,
		SubsystemRuntimeBoundary: false,
		SubsystemLymphaticLane:   false,
		SubsystemConsensusMesh:   false,
		SubsystemRustValidator:   false,
	}
	for _, status := range statuses {
		seen, ok := want[status.Subsystem]
		if !ok {
			t.Fatalf("unexpected subsystem %q", status.Subsystem)
		}
		if seen {
			t.Fatalf("duplicate subsystem %q", status.Subsystem)
		}
		want[status.Subsystem] = true
		if status.Status == "" || status.AdapterNeeded == "" || status.IntegrationRisk == "" || status.RecommendedNextAction == "" {
			t.Fatalf("subsystem readiness incomplete: %#v", status)
		}
	}
	for subsystem, seen := range want {
		if !seen {
			t.Fatalf("missing subsystem %q", subsystem)
		}
	}
}
