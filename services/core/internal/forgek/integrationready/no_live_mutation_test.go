package integrationready

import "testing"

func TestValidateNoLiveMutationPassesForDefaults(t *testing.T) {
	if err := ValidateNoLiveMutation(DefaultAdapterContracts(), DefaultShadowModePolicy(), DefaultLivePathMappings()); err != nil {
		t.Fatal(err)
	}
}

func TestValidateNoLiveMutationRejectsAnyLiveMutationSurface(t *testing.T) {
	contracts := DefaultAdapterContracts()
	contracts[0].LiveMutationAllowed = true
	if err := ValidateNoLiveMutation(contracts, DefaultShadowModePolicy(), DefaultLivePathMappings()); err == nil {
		t.Fatal("expected adapter live mutation to be rejected")
	}
	mappings := DefaultLivePathMappings()
	mappings[0].LiveMutationAllowed = true
	if err := ValidateNoLiveMutation(DefaultAdapterContracts(), DefaultShadowModePolicy(), mappings); err == nil {
		t.Fatal("expected mapping live mutation to be rejected")
	}
	policy := DefaultShadowModePolicy()
	policy.AllowMemoryWrites = true
	if err := ValidateNoLiveMutation(DefaultAdapterContracts(), policy, DefaultLivePathMappings()); err == nil {
		t.Fatal("expected shadow memory write to be rejected")
	}
}
