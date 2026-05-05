package integrationready

import "testing"

func TestDefaultAdapterContractsAreReadOnlyAndPreserveProvenance(t *testing.T) {
	contracts := DefaultAdapterContracts()
	if len(contracts) != 12 {
		t.Fatalf("expected 12 default adapter contracts, got %d", len(contracts))
	}
	for _, contract := range contracts {
		if err := ValidateAdapterContract(contract); err != nil {
			t.Fatalf("default contract %s should validate: %v", contract.AdapterID, err)
		}
		if !contract.ReadOnly {
			t.Fatalf("%s must be read-only", contract.AdapterID)
		}
		if contract.LiveMutationAllowed {
			t.Fatalf("%s must forbid live mutation", contract.AdapterID)
		}
		if contract.ToolExecutionAllowed {
			t.Fatalf("%s must forbid tool execution", contract.AdapterID)
		}
		if contract.ModelRuntimeCallAllowed {
			t.Fatalf("%s must forbid modelruntime calls", contract.AdapterID)
		}
		if contract.UserVisibleOutputAllowed {
			t.Fatalf("%s must forbid user-visible output", contract.AdapterID)
		}
		if !contract.PreservesProvenance {
			t.Fatalf("%s must preserve provenance", contract.AdapterID)
		}
	}
}

func TestAdapterContractRejectsMutationCapabilities(t *testing.T) {
	contract := ReadOnlyRAGAdapterContract()
	contract.LiveMutationAllowed = true
	if err := ValidateAdapterContract(contract); err == nil {
		t.Fatal("expected live mutation permission to be rejected")
	}
	contract = ReadOnlyRAGAdapterContract()
	contract.ModelRuntimeCallAllowed = true
	if err := ValidateAdapterContract(contract); err == nil {
		t.Fatal("expected modelruntime call permission to be rejected")
	}
	contract = ReadOnlyRAGAdapterContract()
	contract.UserVisibleOutputAllowed = true
	if err := ValidateAdapterContract(contract); err == nil {
		t.Fatal("expected user-visible output permission to be rejected")
	}
}
