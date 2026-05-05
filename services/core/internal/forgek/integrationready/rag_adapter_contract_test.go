package integrationready

import (
	"slices"
	"testing"
)

func TestReadOnlyRAGAdapterBoundary(t *testing.T) {
	contract := ReadOnlyRAGAdapterContract()
	if err := ValidateAdapterContract(contract); err != nil {
		t.Fatal(err)
	}
	if contract.AdapterType != AdapterReadOnlyRAG {
		t.Fatalf("unexpected adapter type: %s", contract.AdapterType)
	}
	if !contract.ReadOnly || !contract.PreservesProvenance {
		t.Fatal("ReadOnlyRAGAdapter must be read-only and preserve provenance")
	}
	if contract.RetrievalExecutionAllowed {
		t.Fatal("ReadOnlyRAGAdapter must not execute retrieval")
	}
	if contract.LiveMutationAllowed || contract.MemoryWriteAllowed || contract.ModelRuntimeCallAllowed || contract.UserVisibleOutputAllowed {
		t.Fatalf("ReadOnlyRAGAdapter has forbidden permissions: %#v", contract)
	}
	if !slices.Contains(contract.AllowedOperations, OperationNormalizeEvidenceRefs) {
		t.Fatalf("ReadOnlyRAGAdapter should allow evidence ref normalization: %#v", contract.AllowedOperations)
	}
	for _, forbidden := range []string{
		OperationExecuteRetrieval,
		OperationCallEmbeddingProvider,
		OperationWriteLiveMemory,
		OperationCompileContext,
		OperationAdmitEvidence,
		OperationPromoteRetrievedContentToTruth,
	} {
		if !slices.Contains(contract.ForbiddenOperations, forbidden) {
			t.Fatalf("ReadOnlyRAGAdapter missing forbidden operation %s", forbidden)
		}
	}
}
