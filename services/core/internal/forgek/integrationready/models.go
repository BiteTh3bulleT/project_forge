package integrationready

import "time"

const Phase11F = "Phase 11F - Integration Readiness Contracts"

const (
	SubsystemKernel          = "Kernel"
	SubsystemNeuronFabric    = "Neuron Fabric"
	SubsystemCourthouse      = "Courthouse"
	SubsystemMemoryPalace    = "Memory Palace"
	SubsystemSemanticAlgebra = "Semantic Algebra"
	SubsystemSnapshots       = "Snapshots"
	SubsystemContextCompiler = "Context Compiler"
	SubsystemKVSystem        = "KV System"
	SubsystemRuntimeBoundary = "Runtime Boundary"
	SubsystemLymphaticLane   = "Lymphatic Lane"
	SubsystemConsensusMesh   = "Consensus Mesh"
	SubsystemRustValidator   = "Rust Validator"
)

type ReadinessStatus string

const (
	StatusReadyForShadow ReadinessStatus = "READY_FOR_SHADOW"
	StatusNeedsAdapter   ReadinessStatus = "NEEDS_ADAPTER"
	StatusNeedsTests     ReadinessStatus = "NEEDS_TESTS"
	StatusTooRisky       ReadinessStatus = "TOO_RISKY"
	StatusDeferred       ReadinessStatus = "DEFERRED"
)

type AdapterType string

const (
	AdapterReadOnlyLiveState        AdapterType = "READ_ONLY_LIVE_STATE"
	AdapterLiveEvidence             AdapterType = "LIVE_EVIDENCE"
	AdapterLiveMemoryMirror         AdapterType = "LIVE_MEMORY_MIRROR"
	AdapterLiveRetrievalMirror      AdapterType = "LIVE_RETRIEVAL_MIRROR"
	AdapterReadOnlyRAG              AdapterType = "READ_ONLY_RAG"
	AdapterLiveEmbeddingTrace       AdapterType = "LIVE_EMBEDDING_TRACE"
	AdapterLiveSearchTrace          AdapterType = "LIVE_SEARCH_TRACE"
	AdapterLiveGatewayTrace         AdapterType = "LIVE_GATEWAY_TRACE"
	AdapterLiveModelRuntimeTrace    AdapterType = "LIVE_MODELRUNTIME_TRACE"
	AdapterLiveAuditMirror          AdapterType = "LIVE_AUDIT_MIRROR"
	AdapterLiveContextCompileMirror AdapterType = "LIVE_CONTEXT_COMPILE_MIRROR"
	AdapterLiveConsensusShadow      AdapterType = "LIVE_CONSENSUS_SHADOW"
)

const (
	OperationObserveLiveMetadata            = "observe_live_metadata"
	OperationNormalizeSourceRefs            = "normalize_source_refs"
	OperationNormalizeEvidenceRefs          = "normalize_evidence_refs"
	OperationProduceDiagnostics             = "produce_diagnostics"
	OperationProduceShadowRAGReport         = "produce_shadow_rag_report"
	OperationExecuteRetrieval               = "execute_retrieval"
	OperationCallEmbeddingProvider          = "call_embedding_provider"
	OperationMutateLiveState                = "mutate_live_state"
	OperationExecuteTools                   = "execute_tools"
	OperationCallModelRuntime               = "call_modelruntime"
	OperationWriteLiveMemory                = "write_live_memory"
	OperationCompileContext                 = "compile_context"
	OperationAdmitEvidence                  = "admit_evidence"
	OperationAlterUserVisibleOutput         = "alter_user_visible_output"
	OperationPromoteRetrievedContentToTruth = "promote_retrieved_content_to_truth"
)

type SubsystemReadiness struct {
	Subsystem             string          `json:"subsystem"`
	SimulatorStatus       string          `json:"simulator_status"`
	ContractStability     string          `json:"contract_stability"`
	LiveEquivalent        string          `json:"live_equivalent"`
	AdapterNeeded         string          `json:"adapter_needed"`
	TestGaps              []string        `json:"test_gaps,omitempty"`
	IntegrationRisk       string          `json:"integration_risk"`
	RecommendedNextAction string          `json:"recommended_next_action"`
	Status                ReadinessStatus `json:"status"`
}

type LivePathMapping struct {
	LiveSystem            string   `json:"live_system"`
	LiveAuthorityPath     string   `json:"live_authority_path"`
	CurrentAuthorityOwner string   `json:"current_authority_owner"`
	ForgeKTargetComponent string   `json:"forge_k_target_component"`
	RequiredAdapter       string   `json:"required_adapter"`
	IntegrationRisk       string   `json:"integration_risk"`
	RequiredTests         []string `json:"required_tests"`
	MigrationStatus       string   `json:"migration_status"`
	LiveMutationAllowed   bool     `json:"live_mutation_allowed"`
	Notes                 string   `json:"notes,omitempty"`
}

type AdapterContract struct {
	AdapterID                 string         `json:"adapter_id"`
	AdapterType               AdapterType    `json:"adapter_type"`
	Purpose                   string         `json:"purpose"`
	AllowedOperations         []string       `json:"allowed_operations"`
	ForbiddenOperations       []string       `json:"forbidden_operations"`
	SourceSystem              string         `json:"source_system"`
	TargetForgeKComponent     string         `json:"target_forge_k_component"`
	PreservesProvenance       bool           `json:"preserves_provenance"`
	ReadOnly                  bool           `json:"read_only"`
	LiveMutationAllowed       bool           `json:"live_mutation_allowed"`
	ToolExecutionAllowed      bool           `json:"tool_execution_allowed"`
	ModelRuntimeCallAllowed   bool           `json:"modelruntime_call_allowed"`
	RetrievalExecutionAllowed bool           `json:"retrieval_execution_allowed"`
	MemoryWriteAllowed        bool           `json:"memory_write_allowed"`
	UserVisibleOutputAllowed  bool           `json:"user_visible_output_allowed"`
	Metadata                  map[string]any `json:"metadata,omitempty"`
}

type ShadowModePolicy struct {
	PolicyID                string         `json:"policy_id"`
	Mode                    string         `json:"mode"`
	ObserveLiveRequest      bool           `json:"observe_live_request"`
	MirrorInputs            bool           `json:"mirror_inputs"`
	MirrorEvidence          bool           `json:"mirror_evidence"`
	MirrorRetrieval         bool           `json:"mirror_retrieval"`
	ProduceShadowReports    bool           `json:"produce_shadow_reports"`
	CompareContext          bool           `json:"compare_context"`
	CompareConsensus        bool           `json:"compare_consensus"`
	CompareResponse         bool           `json:"compare_response"`
	AllowLiveMutation       bool           `json:"allow_live_mutation"`
	AllowToolExecution      bool           `json:"allow_tool_execution"`
	AllowModelRuntimeCalls  bool           `json:"allow_modelruntime_calls"`
	AllowRetrievalExecution bool           `json:"allow_retrieval_execution"`
	AllowMemoryWrites       bool           `json:"allow_memory_writes"`
	AllowUserVisibleOutput  bool           `json:"allow_user_visible_output"`
	Metadata                map[string]any `json:"metadata,omitempty"`
}

type IntegrationReadinessReport struct {
	ReportID          string               `json:"report_id"`
	GeneratedAt       time.Time            `json:"generated_at"`
	Phase             string               `json:"phase"`
	SubsystemStatuses []SubsystemReadiness `json:"subsystem_statuses"`
	LivePathMappings  []LivePathMapping    `json:"live_path_mappings"`
	AdapterContracts  []AdapterContract    `json:"adapter_contracts"`
	ShadowPolicy      ShadowModePolicy     `json:"shadow_policy"`
	MissingContracts  []string             `json:"missing_contracts,omitempty"`
	MissingTests      []string             `json:"missing_tests,omitempty"`
	Blockers          []string             `json:"blockers,omitempty"`
	Warnings          []string             `json:"warnings,omitempty"`
	ReadinessScore    float64              `json:"readiness_score"`
	Metadata          map[string]any       `json:"metadata,omitempty"`
}

type ReportInput struct {
	ReportID          string
	GeneratedAt       time.Time
	Phase             string
	SubsystemStatuses []SubsystemReadiness
	LivePathMappings  []LivePathMapping
	AdapterContracts  []AdapterContract
	ShadowPolicy      ShadowModePolicy
	MissingContracts  []string
	MissingTests      []string
	Blockers          []string
	Warnings          []string
	Metadata          map[string]any
}
