package integrationready

import (
	"cmp"
	"fmt"
	"slices"
	"sort"
	"strings"
)

func DefaultSubsystemStatuses() []SubsystemReadiness {
	values := []SubsystemReadiness{
		subsystem(SubsystemKernel, "implemented/tested in simulator", "stable simulator syscall contract", "aios/controllane", "ReadOnlyLiveStateAdapter", []string{"route inventory tests", "shadow comparison tests"}, "high", "Phase 12A live integration design", StatusNeedsAdapter),
		subsystem(SubsystemNeuronFabric, "implemented/tested in simulator", "stable proposal/validation envelopes", "lanes, permissions, gateway routing", "LiveEvidenceAdapter", []string{"live proposal mirror tests"}, "medium", "define read-only neuron shadow inputs", StatusNeedsAdapter),
		subsystem(SubsystemCourthouse, "implemented/tested in simulator", "stable evidence/admission model", "aios/controllane semantic admission", "LiveEvidenceAdapter", []string{"admission parity tests"}, "high", "map live evidence records before migration", StatusNeedsAdapter),
		subsystem(SubsystemMemoryPalace, "implemented/tested in simulator", "stable retrieval-shape refs", "memory/retrieval/search/embeddings", "ReadOnlyRAGAdapter", []string{"retrieval provenance mirror tests"}, "high", "keep retrieval evidence-only until Phase 12 design", StatusNeedsAdapter),
		subsystem(SubsystemSemanticAlgebra, "implemented/tested in simulator", "stable deterministic transform records", "aios/controllane semantic records", "LiveEvidenceAdapter", []string{"operation provenance parity tests"}, "medium", "define transform mirror fixtures", StatusNeedsTests),
		subsystem(SubsystemSnapshots, "implemented/tested in simulator", "stable shape-not-truth contract", "backup/release and live snapshot/restore surfaces", "LiveMemoryMirrorAdapter", []string{"restore non-execution shadow tests"}, "medium", "prepare restore-seed mirror reports", StatusNeedsAdapter),
		subsystem(SubsystemContextCompiler, "implemented/tested in simulator", "stable deterministic context blocks", "live COMPILE_CONTEXT path", "LiveContextCompileMirrorAdapter", []string{"context comparison tests"}, "high", "design read-only context shadow harness", StatusNeedsAdapter),
		subsystem(SubsystemKVSystem, "implemented/tested in simulator", "stable metadata-only validation gates", "modelruntime/cache metadata", "LiveModelRuntimeTraceAdapter", []string{"no live KV reuse tests"}, "medium", "keep KV metadata diagnostic-only", StatusReadyForShadow),
		subsystem(SubsystemRuntimeBoundary, "implemented/tested in simulator with mock driver", "stable driver-as-proposal boundary", "modelruntime", "LiveModelRuntimeTraceAdapter", []string{"real driver non-authority tests"}, "high", "defer real runtime drivers", StatusNeedsAdapter),
		subsystem(SubsystemLymphaticLane, "implemented/tested in simulator", "stable maintenance proposal model", "aios/dream and autonomy maintenance", "ReadOnlyLiveStateAdapter", []string{"no cleanup mutation shadow tests"}, "medium", "mirror live maintenance reports only", StatusReadyForShadow),
		subsystem(SubsystemConsensusMesh, "implemented/tested in simulator", "stable claim governance contract", "response/action composition planning", "LiveConsensusShadowAdapter", []string{"composer guard shadow tests"}, "medium", "compare consensus diagnostics without affecting output", StatusNeedsTests),
		subsystem(SubsystemRustValidator, "implemented/tested as research/tooling", "stable fixture validation for selected models", "CI/tooling only", "none", []string{"future consensus fixture parity"}, "low", "keep Rust validation out of live authority", StatusReadyForShadow),
	}
	sortSubsystems(values)
	return values
}

func DefaultLivePathMappings() []LivePathMapping {
	values := []LivePathMapping{
		mapping("api routes", "services/core/internal/api", "HTTP route composition", "Kernel ingress shell", "ReadOnlyLiveStateAdapter", "high", []string{"route inventory tests", "no route behavior change tests"}, "not_started", "API routes remain live and unchanged."),
		mapping("aios/controllane", "services/core/internal/aios/controllane", "live semantic syscall processor", "Kernel and semantic syscalls", "ReadOnlyLiveStateAdapter", "high", []string{"syscall parity tests", "journal boundary tests"}, "not_started", "Control lane remains live semantic authority."),
		mapping("gateway", "services/core/internal/gateway", "live tool execution authority", "Neuron Fabric and execution boundary", "LiveGatewayTraceAdapter", "high", []string{"approval gate parity tests", "no tool execution from shadow tests"}, "not_started", "Gateway remains live tool authority."),
		mapping("permissions", "services/core/internal/permissions", "live permission profile policy", "capability policy", "ReadOnlyLiveStateAdapter", "medium", []string{"capability parity tests"}, "not_started", "Permissions remain live policy authority."),
		mapping("lanes", "services/core/internal/lanes", "live lane contracts", "lane and syscall scope contracts", "ReadOnlyLiveStateAdapter", "medium", []string{"lane mirror tests"}, "not_started", "Lanes remain live execution scope."),
		mapping("audit", "services/core/internal/audit", "live append-only audit records", "journal/provenance evidence", "LiveAuditMirrorAdapter", "medium", []string{"audit mirror provenance tests"}, "not_started", "Audit remains live record authority."),
		mapping("modelruntime", "services/core/internal/modelruntime", "live model runtime driver service", "Runtime Boundary", "LiveModelRuntimeTraceAdapter", "high", []string{"no modelruntime call tests", "runtime trace mirror tests"}, "not_started", "Model runtime remains a live external driver."),
		mapping("memory", "services/core/internal/memory", "legacy memory and VSA evidence/projection service", "Memory Palace and Courthouse evidence", "LiveMemoryMirrorAdapter", "high", []string{"no memory write tests", "memory provenance tests"}, "not_started", "Memory writes remain outside FORGE-K."),
		mapping("retrieval", "services/core/internal/retrieval", "live retrieval run service", "Memory Palace", "LiveRetrievalMirrorAdapter", "high", []string{"no retrieval execution tests", "retrieval provenance tests"}, "not_started", "Retrieval results are evidence/routing signals only."),
		mapping("search", "services/core/internal/search", "live search index/query support", "Memory Palace", "LiveSearchTraceAdapter", "medium", []string{"search trace mirror tests"}, "not_started", "Search is observed only in Phase 11F."),
		mapping("embeddings", "services/core/internal/embeddings", "live embedding provider/index support", "Memory Palace and KV metadata", "LiveEmbeddingTraceAdapter", "high", []string{"no embedding provider call tests"}, "not_started", "Embeddings remain trace metadata only."),
		mapping("dream/autonomy", "services/core/internal/aios/dream and autonomy", "live dry-run/reporting and autonomy surfaces", "Lymphatic Lane and Consensus Mesh", "ReadOnlyLiveStateAdapter", "medium", []string{"non-canonical report tests"}, "not_started", "Dream reports remain non-canonical evidence."),
		mapping("backup/release", "services/core/internal/backup and services/core/internal/release", "portable bundle and release readiness services", "Snapshots and Lymphatic Lane", "LiveMemoryMirrorAdapter", "medium", []string{"restore non-execution tests"}, "not_started", "Backup/release remain operational surfaces."),
		mapping("settings/config", "services/core/internal/config", "environment-derived runtime policy", "Kernel policy substrate", "ReadOnlyLiveStateAdapter", "medium", []string{"config mirror tests"}, "not_started", "Config remains live boot policy."),
	}
	return normalizeMappings(values)
}

func ValidateReadinessStatus(status ReadinessStatus) bool {
	switch status {
	case StatusReadyForShadow, StatusNeedsAdapter, StatusNeedsTests, StatusTooRisky, StatusDeferred:
		return true
	default:
		return false
	}
}

func NormalizeStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func ValidateNoLiveMutation(contracts []AdapterContract, policy ShadowModePolicy, mappings []LivePathMapping) error {
	if err := ValidateShadowModePolicy(policy); err != nil {
		return err
	}
	for _, contract := range contracts {
		if err := ValidateAdapterContract(contract); err != nil {
			return err
		}
	}
	for _, mapping := range mappings {
		if err := ValidateLivePathMapping(mapping); err != nil {
			return err
		}
	}
	return nil
}

func ValidateLivePathMapping(mapping LivePathMapping) error {
	if strings.TrimSpace(mapping.LiveSystem) == "" || strings.TrimSpace(mapping.LiveAuthorityPath) == "" ||
		strings.TrimSpace(mapping.CurrentAuthorityOwner) == "" || strings.TrimSpace(mapping.ForgeKTargetComponent) == "" ||
		strings.TrimSpace(mapping.RequiredAdapter) == "" || strings.TrimSpace(mapping.IntegrationRisk) == "" ||
		strings.TrimSpace(mapping.MigrationStatus) == "" || len(mapping.RequiredTests) == 0 {
		return fmt.Errorf("%w: live path mapping", ErrMissingRequiredField)
	}
	if mapping.LiveMutationAllowed {
		return fmt.Errorf("%w: live path mapping %s", ErrLiveMutationAllowed, mapping.LiveSystem)
	}
	return nil
}

func normalizeSubsystems(values []SubsystemReadiness) []SubsystemReadiness {
	out := append([]SubsystemReadiness(nil), values...)
	for i := range out {
		out[i].TestGaps = NormalizeStrings(out[i].TestGaps)
	}
	sortSubsystems(out)
	return out
}

func normalizeMappings(values []LivePathMapping) []LivePathMapping {
	out := append([]LivePathMapping(nil), values...)
	for i := range out {
		out[i].RequiredTests = NormalizeStrings(out[i].RequiredTests)
	}
	slices.SortFunc(out, func(a, b LivePathMapping) int {
		return cmp.Compare(a.LiveSystem, b.LiveSystem)
	})
	return out
}

func sortSubsystems(values []SubsystemReadiness) {
	slices.SortFunc(values, func(a, b SubsystemReadiness) int {
		return cmp.Compare(a.Subsystem, b.Subsystem)
	})
}

func subsystem(name, simulatorStatus, stability, liveEquivalent, adapterNeeded string, testGaps []string, risk, next string, status ReadinessStatus) SubsystemReadiness {
	return SubsystemReadiness{
		Subsystem:             name,
		SimulatorStatus:       simulatorStatus,
		ContractStability:     stability,
		LiveEquivalent:        liveEquivalent,
		AdapterNeeded:         adapterNeeded,
		TestGaps:              NormalizeStrings(testGaps),
		IntegrationRisk:       risk,
		RecommendedNextAction: next,
		Status:                status,
	}
}

func mapping(system, path, owner, target, adapter, risk string, tests []string, status, notes string) LivePathMapping {
	return LivePathMapping{
		LiveSystem:            system,
		LiveAuthorityPath:     path,
		CurrentAuthorityOwner: owner,
		ForgeKTargetComponent: target,
		RequiredAdapter:       adapter,
		IntegrationRisk:       risk,
		RequiredTests:         NormalizeStrings(tests),
		MigrationStatus:       status,
		LiveMutationAllowed:   false,
		Notes:                 notes,
	}
}

func validateSubsystem(status SubsystemReadiness) error {
	if strings.TrimSpace(status.Subsystem) == "" || strings.TrimSpace(status.SimulatorStatus) == "" ||
		strings.TrimSpace(status.ContractStability) == "" || strings.TrimSpace(status.LiveEquivalent) == "" ||
		strings.TrimSpace(status.AdapterNeeded) == "" || strings.TrimSpace(status.IntegrationRisk) == "" ||
		strings.TrimSpace(status.RecommendedNextAction) == "" {
		return fmt.Errorf("%w: subsystem readiness", ErrMissingRequiredField)
	}
	if !ValidateReadinessStatus(status.Status) {
		return fmt.Errorf("%w: %s", ErrInvalidStatus, status.Status)
	}
	return nil
}
