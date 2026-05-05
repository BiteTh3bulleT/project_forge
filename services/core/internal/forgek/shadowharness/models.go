package shadowharness

import "time"

type ShadowObservation struct {
	ObservationID  string         `json:"observation_id"`
	WorkspaceID    string         `json:"workspace_id"`
	RequestID      string         `json:"request_id"`
	ObservedAt     time.Time      `json:"observed_at"`
	LivePath       string         `json:"live_path"`
	RequestSummary string         `json:"request_summary"`
	InputRefs      []string       `json:"input_refs,omitempty"`
	EvidenceRefs   []string       `json:"evidence_refs,omitempty"`
	RetrievalRefs  []string       `json:"retrieval_refs,omitempty"`
	ContextRefs    []string       `json:"context_refs,omitempty"`
	ConsensusRefs  []string       `json:"consensus_refs,omitempty"`
	RuntimeRefs    []string       `json:"runtime_refs,omitempty"`
	KVRefs         []string       `json:"kv_refs,omitempty"`
	RiskFlags      []string       `json:"risk_flags,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

type ShadowHarnessPolicy struct {
	PolicyID                string         `json:"policy_id"`
	Mode                    string         `json:"mode"`
	AllowLiveMutation       bool           `json:"allow_live_mutation"`
	AllowToolExecution      bool           `json:"allow_tool_execution"`
	AllowModelRuntimeCalls  bool           `json:"allow_modelruntime_calls"`
	AllowRetrievalExecution bool           `json:"allow_retrieval_execution"`
	AllowEmbeddingCalls     bool           `json:"allow_embedding_calls"`
	AllowMemoryWrites       bool           `json:"allow_memory_writes"`
	AllowUserVisibleOutput  bool           `json:"allow_user_visible_output"`
	AllowPublicAPIChanges   bool           `json:"allow_public_api_changes"`
	ProduceConsensusReport  bool           `json:"produce_consensus_report"`
	ProduceContextReport    bool           `json:"produce_context_report"`
	ProduceRAGReport        bool           `json:"produce_rag_report"`
	ProduceRuntimeReport    bool           `json:"produce_runtime_report"`
	ProduceKVReport         bool           `json:"produce_kv_report"`
	ProduceLymphaticReport  bool           `json:"produce_lymphatic_report"`
	Metadata                map[string]any `json:"metadata,omitempty"`
}

type ShadowComparisonReport struct {
	ReportID         string                `json:"report_id"`
	WorkspaceID      string                `json:"workspace_id"`
	RequestID        string                `json:"request_id"`
	GeneratedAt      time.Time             `json:"generated_at"`
	ObservationRefs  []string              `json:"observation_refs,omitempty"`
	ConsensusShadow  ConsensusShadowReport `json:"consensus_shadow"`
	ContextShadow    ContextShadowReport   `json:"context_shadow"`
	RAGShadow        RAGShadowReport       `json:"rag_shadow"`
	RuntimeShadow    RuntimeShadowReport   `json:"runtime_shadow"`
	KVShadow         KVShadowReport        `json:"kv_shadow"`
	LymphaticShadow  LymphaticShadowReport `json:"lymphatic_shadow"`
	Divergences      []string              `json:"divergences,omitempty"`
	Warnings         []string              `json:"warnings,omitempty"`
	Blockers         []string              `json:"blockers,omitempty"`
	NoEffectVerified bool                  `json:"no_effect_verified"`
	Metadata         map[string]any        `json:"metadata,omitempty"`
}

type RAGShadowReport struct {
	ReportID                string         `json:"report_id"`
	RequestID               string         `json:"request_id"`
	RetrievalRefs           []string       `json:"retrieval_refs,omitempty"`
	EvidenceRefs            []string       `json:"evidence_refs,omitempty"`
	SourceRefs              []string       `json:"source_refs,omitempty"`
	NormalizedEvidenceCount int            `json:"normalized_evidence_count"`
	Tier1Count              int            `json:"tier1_count"`
	Tier2Count              int            `json:"tier2_count"`
	Tier3Count              int            `json:"tier3_count"`
	StaleRefs               []string       `json:"stale_refs,omitempty"`
	UnsupportedRefs         []string       `json:"unsupported_refs,omitempty"`
	Warnings                []string       `json:"warnings,omitempty"`
	NoExecutionVerified     bool           `json:"no_execution_verified"`
	Metadata                map[string]any `json:"metadata,omitempty"`
}

type ConsensusShadowReport struct {
	ReportID               string         `json:"report_id"`
	RequestID              string         `json:"request_id"`
	ConsensusReportRef     string         `json:"consensus_report_ref,omitempty"`
	AcceptedClaimCount     int            `json:"accepted_claim_count"`
	RejectedClaimCount     int            `json:"rejected_claim_count"`
	UncertainClaimCount    int            `json:"uncertain_claim_count"`
	ConflictedClaimCount   int            `json:"conflicted_claim_count"`
	UnsupportedFactCount   int            `json:"unsupported_fact_count"`
	ComposerGuardPassed    bool           `json:"composer_guard_passed"`
	Warnings               []string       `json:"warnings,omitempty"`
	DiagnosticOnlyVerified bool           `json:"diagnostic_only_verified"`
	Metadata               map[string]any `json:"metadata,omitempty"`
}

type ContextShadowReport struct {
	ReportID                     string         `json:"report_id"`
	RequestID                    string         `json:"request_id"`
	ContextBundleRef             string         `json:"context_bundle_ref,omitempty"`
	BlockCount                   int            `json:"block_count"`
	StablePrefixHash             string         `json:"stable_prefix_hash,omitempty"`
	VolatileSuffixHash           string         `json:"volatile_suffix_hash,omitempty"`
	CacheEligibleBlockCount      int            `json:"cache_eligible_block_count"`
	RejectedEvidenceLeakDetected bool           `json:"rejected_evidence_leak_detected"`
	Warnings                     []string       `json:"warnings,omitempty"`
	DiagnosticOnlyVerified       bool           `json:"diagnostic_only_verified"`
	Metadata                     map[string]any `json:"metadata,omitempty"`
}

type RuntimeShadowReport struct {
	ReportID             string         `json:"report_id"`
	RequestID            string         `json:"request_id"`
	RuntimeResultRefs    []string       `json:"runtime_result_refs,omitempty"`
	DriverRefs           []string       `json:"driver_refs,omitempty"`
	ModelIdentityRefs    []string       `json:"model_identity_refs,omitempty"`
	ProposalOnlyVerified bool           `json:"proposal_only_verified"`
	Warnings             []string       `json:"warnings,omitempty"`
	Metadata             map[string]any `json:"metadata,omitempty"`
}

type KVShadowReport struct {
	ReportID                      string         `json:"report_id"`
	RequestID                     string         `json:"request_id"`
	KVManifestRefs                []string       `json:"kv_manifest_refs,omitempty"`
	CacheHitCount                 int            `json:"cache_hit_count"`
	CacheMissCount                int            `json:"cache_miss_count"`
	InvalidatedCount              int            `json:"invalidated_count"`
	EvictedCount                  int            `json:"evicted_count"`
	AccelerationNotMemoryVerified bool           `json:"acceleration_not_memory_verified"`
	Warnings                      []string       `json:"warnings,omitempty"`
	Metadata                      map[string]any `json:"metadata,omitempty"`
}

type LymphaticShadowReport struct {
	ReportID                 string         `json:"report_id"`
	RequestID                string         `json:"request_id"`
	MaintenanceReportRefs    []string       `json:"maintenance_report_refs,omitempty"`
	CleanupProposalCount     int            `json:"cleanup_proposal_count"`
	UnsafeProposalCount      int            `json:"unsafe_proposal_count"`
	NoSilentMutationVerified bool           `json:"no_silent_mutation_verified"`
	ProposalsDoNotExecute    bool           `json:"proposals_do_not_execute"`
	Warnings                 []string       `json:"warnings,omitempty"`
	Metadata                 map[string]any `json:"metadata,omitempty"`
}
