package integrationready

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
)

func DefaultAdapterContracts() []AdapterContract {
	values := []AdapterContract{
		defaultAdapter("adapter-read-only-live-state", AdapterReadOnlyLiveState, "live daemon state", "Kernel"),
		defaultAdapter("adapter-live-evidence", AdapterLiveEvidence, "live evidence records", "Courthouse"),
		defaultAdapter("adapter-live-memory-mirror", AdapterLiveMemoryMirror, "live memory metadata", "Memory Palace"),
		defaultAdapter("adapter-live-retrieval-mirror", AdapterLiveRetrievalMirror, "live retrieval metadata", "Memory Palace"),
		ReadOnlyRAGAdapterContract(),
		defaultAdapter("adapter-live-embedding-trace", AdapterLiveEmbeddingTrace, "live embedding trace metadata", "Memory Palace"),
		defaultAdapter("adapter-live-search-trace", AdapterLiveSearchTrace, "live search result metadata", "Memory Palace"),
		defaultAdapter("adapter-live-gateway-trace", AdapterLiveGatewayTrace, "live gateway invocation traces", "Neuron Fabric"),
		defaultAdapter("adapter-live-modelruntime-trace", AdapterLiveModelRuntimeTrace, "live modelruntime traces", "Runtime Boundary"),
		defaultAdapter("adapter-live-audit-mirror", AdapterLiveAuditMirror, "live audit records", "Kernel"),
		defaultAdapter("adapter-live-context-compile-mirror", AdapterLiveContextCompileMirror, "live context compile metadata", "Context Compiler"),
		defaultAdapter("adapter-live-consensus-shadow", AdapterLiveConsensusShadow, "live response/action proposal traces", "Consensus Mesh"),
	}
	return normalizeContracts(values)
}

func ReadOnlyRAGAdapterContract() AdapterContract {
	contract := defaultAdapter("adapter-read-only-rag", AdapterReadOnlyRAG, "existing live retrieval, search, embedding, and VSA trace metadata", "Memory Palace")
	contract.Purpose = "Observe existing live retrieval/search/memory outputs and normalize them into FORGE-K EvidenceRefs for shadow-mode diagnostics."
	contract.AllowedOperations = NormalizeStrings([]string{
		OperationObserveLiveMetadata,
		OperationNormalizeSourceRefs,
		OperationNormalizeEvidenceRefs,
		OperationProduceDiagnostics,
		OperationProduceShadowRAGReport,
	})
	contract.ForbiddenOperations = NormalizeStrings(append(contract.ForbiddenOperations,
		OperationExecuteRetrieval,
		OperationCallEmbeddingProvider,
		OperationWriteLiveMemory,
		OperationCompileContext,
		OperationAdmitEvidence,
		OperationPromoteRetrievedContentToTruth,
	))
	return contract
}

func ValidateAdapterContract(contract AdapterContract) error {
	if strings.TrimSpace(contract.AdapterID) == "" || strings.TrimSpace(contract.Purpose) == "" ||
		strings.TrimSpace(contract.SourceSystem) == "" || strings.TrimSpace(contract.TargetForgeKComponent) == "" {
		return fmt.Errorf("%w: adapter contract", ErrMissingRequiredField)
	}
	if !ValidAdapterType(contract.AdapterType) {
		return fmt.Errorf("%w: %s", ErrInvalidAdapterType, contract.AdapterType)
	}
	if !contract.ReadOnly || !contract.PreservesProvenance {
		return fmt.Errorf("%w: adapter must be read-only and preserve provenance", ErrLiveSideEffect)
	}
	if contract.LiveMutationAllowed || contract.ToolExecutionAllowed || contract.ModelRuntimeCallAllowed ||
		contract.RetrievalExecutionAllowed || contract.MemoryWriteAllowed || contract.UserVisibleOutputAllowed {
		return fmt.Errorf("%w: adapter %s", ErrLiveSideEffect, contract.AdapterID)
	}
	return nil
}

func ValidAdapterType(value AdapterType) bool {
	switch value {
	case AdapterReadOnlyLiveState, AdapterLiveEvidence, AdapterLiveMemoryMirror, AdapterLiveRetrievalMirror,
		AdapterReadOnlyRAG, AdapterLiveEmbeddingTrace, AdapterLiveSearchTrace, AdapterLiveGatewayTrace,
		AdapterLiveModelRuntimeTrace, AdapterLiveAuditMirror, AdapterLiveContextCompileMirror, AdapterLiveConsensusShadow:
		return true
	default:
		return false
	}
}

func defaultAdapter(id string, adapterType AdapterType, source, target string) AdapterContract {
	return AdapterContract{
		AdapterID:             id,
		AdapterType:           adapterType,
		Purpose:               "Observe, normalize, mirror, and report without live mutation.",
		AllowedOperations:     NormalizeStrings([]string{OperationObserveLiveMetadata, OperationNormalizeSourceRefs, OperationNormalizeEvidenceRefs, OperationProduceDiagnostics}),
		ForbiddenOperations:   NormalizeStrings([]string{OperationMutateLiveState, OperationExecuteTools, OperationCallModelRuntime, OperationExecuteRetrieval, OperationCallEmbeddingProvider, OperationWriteLiveMemory, OperationAlterUserVisibleOutput}),
		SourceSystem:          source,
		TargetForgeKComponent: target,
		PreservesProvenance:   true,
		ReadOnly:              true,
	}
}

func normalizeContracts(values []AdapterContract) []AdapterContract {
	out := append([]AdapterContract(nil), values...)
	for i := range out {
		out[i].AllowedOperations = NormalizeStrings(out[i].AllowedOperations)
		out[i].ForbiddenOperations = NormalizeStrings(out[i].ForbiddenOperations)
	}
	slices.SortFunc(out, func(a, b AdapterContract) int {
		return cmp.Compare(a.AdapterID, b.AdapterID)
	})
	return out
}
