package forgekshadow

import (
	"time"

	"forge/projectforge/services/core/internal/forgek/shadowharness"
	"forge/projectforge/services/core/internal/refvalidation"
)

type ObservationInput struct {
	WorkspaceID    string
	RequestID      string
	LivePath       string
	Method         string
	Path           string
	RequestSummary string
	Metadata       map[string]any
}

type DiagnosticReport struct {
	Observation           shadowharness.ShadowObservation
	Comparison            shadowharness.ShadowComparisonReport
	RouteEnvelope         *RouteEnvelopeObservation
	ChatMetadata          *ChatMetadataObservation
	RetrievalMetadata     *RetrievalMetadataObservation
	MemoryPalaceMirror    *MemoryPalaceMirrorReport
	ControlLaneValidation *ControlLaneValidationObservation
	ContextBundleShadow   *ContextBundleShadowReport
	Advisory              *ShadowAdvisoryReport
	StoredAt              time.Time
}

type RouteEnvelopeInput struct {
	WorkspaceID   string
	RequestID     string
	CorrelationID string
	Method        string
	Path          string
	RoutePattern  string
	RouteClass    string
	StatusCode    int
	Duration      time.Duration
	Warnings      []string
	Metadata      map[string]any
}

type RouteEnvelopeObservation struct {
	ObservationID string         `json:"observation_id"`
	ObservedAt    time.Time      `json:"observed_at"`
	Method        string         `json:"method"`
	Path          string         `json:"path,omitempty"`
	RoutePattern  string         `json:"route_pattern,omitempty"`
	RouteClass    string         `json:"route_class"`
	StatusCode    int            `json:"status_code,omitempty"`
	DurationMS    int64          `json:"duration_ms"`
	RequestID     string         `json:"request_id,omitempty"`
	WorkspaceID   string         `json:"workspace_id,omitempty"`
	CorrelationID string         `json:"correlation_id,omitempty"`
	Warnings      []string       `json:"warnings,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

type ChatMetadataInput struct {
	WorkspaceID   string
	RequestID     string
	CorrelationID string
	OperationKind string
	ThreadID      string
	MessageID     string
	RoleClass     string
	StreamClass   string
	ModelID       string
	ProviderID    string
	MessageCount  int
	Duration      time.Duration
	Warnings      []string
	Metadata      map[string]any
}

type ChatMetadataObservation struct {
	ObservationID string         `json:"observation_id"`
	ObservedAt    time.Time      `json:"observed_at"`
	OperationKind string         `json:"operation_kind"`
	ThreadID      string         `json:"thread_id,omitempty"`
	MessageID     string         `json:"message_id,omitempty"`
	RoleClass     string         `json:"role_class,omitempty"`
	StreamClass   string         `json:"stream_class,omitempty"`
	ModelID       string         `json:"model_id,omitempty"`
	ProviderID    string         `json:"provider_id,omitempty"`
	MessageCount  int            `json:"message_count,omitempty"`
	DurationMS    int64          `json:"duration_ms"`
	RequestID     string         `json:"request_id,omitempty"`
	WorkspaceID   string         `json:"workspace_id,omitempty"`
	CorrelationID string         `json:"correlation_id,omitempty"`
	Warnings      []string       `json:"warnings,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

type RetrievalMetadataInput struct {
	WorkspaceID       string
	RequestID         string
	CorrelationID     string
	RetrievalRunID    string
	RetrievalResultID string
	SourceType        string
	SourceRefID       string
	SourceHash        string
	ResultCount       int
	SelectedCount     int
	ScoreSummary      string
	RankingPosition   int
	RetrievalStrategy string
	IndexType         string
	EmbeddingModelID  string
	FreshnessStatus   string
	Duration          time.Duration
	Warnings          []string
	Metadata          map[string]any
}

type RetrievalMetadataObservation struct {
	ObservationID     string         `json:"observation_id"`
	ObservedAt        time.Time      `json:"observed_at"`
	WorkspaceID       string         `json:"workspace_id,omitempty"`
	RequestID         string         `json:"request_id,omitempty"`
	CorrelationID     string         `json:"correlation_id,omitempty"`
	RetrievalRunID    string         `json:"retrieval_run_id,omitempty"`
	RetrievalResultID string         `json:"retrieval_result_id,omitempty"`
	SourceType        string         `json:"source_type,omitempty"`
	SourceRefID       string         `json:"source_ref_id,omitempty"`
	SourceHash        string         `json:"source_hash,omitempty"`
	ResultCount       int            `json:"result_count,omitempty"`
	SelectedCount     int            `json:"selected_count,omitempty"`
	ScoreSummary      string         `json:"score_summary,omitempty"`
	RankingPosition   int            `json:"ranking_position,omitempty"`
	RetrievalStrategy string         `json:"retrieval_strategy,omitempty"`
	IndexType         string         `json:"index_type,omitempty"`
	EmbeddingModelID  string         `json:"embedding_model_id,omitempty"`
	FreshnessStatus   string         `json:"freshness_status,omitempty"`
	DurationMS        int64          `json:"duration_ms"`
	Warnings          []string       `json:"warnings,omitempty"`
	Metadata          map[string]any `json:"metadata,omitempty"`
}

type ControlLaneValidationInput struct {
	WorkspaceID            string
	RequestID              string
	CorrelationID          string
	Action                 string
	ValidationKind         string
	Decision               string
	Passed                 bool
	Match                  bool
	OperationType          string
	NormalizedRefCount     int
	AddedRefCount          int
	RemovedRefCount        int
	UnchangedRefCount      int
	FailureCount           int
	WarningCount           int
	MemoryMutation         bool
	RuntimeMutation        bool
	ModelRuntimeCall       bool
	EvidenceAdmission      bool
	ContextCompilation     bool
	UserVisibleOutput      bool
	LiveAuthorityMigration bool
	Duration               time.Duration
	Warnings               []string
	NormalizedRefs         []refvalidation.ObjectRef
	Metadata               map[string]any
}

type ControlLaneValidationObservation struct {
	ObservationID          string                    `json:"observation_id"`
	ObservedAt             time.Time                 `json:"observed_at"`
	WorkspaceID            string                    `json:"workspace_id,omitempty"`
	RequestID              string                    `json:"request_id,omitempty"`
	CorrelationID          string                    `json:"correlation_id,omitempty"`
	Action                 string                    `json:"action"`
	ValidationKind         string                    `json:"validation_kind"`
	Decision               string                    `json:"decision"`
	Passed                 bool                      `json:"passed"`
	Match                  bool                      `json:"match,omitempty"`
	OperationType          string                    `json:"operation_type,omitempty"`
	NormalizedRefCount     int                       `json:"normalized_ref_count,omitempty"`
	AddedRefCount          int                       `json:"added_ref_count,omitempty"`
	RemovedRefCount        int                       `json:"removed_ref_count,omitempty"`
	UnchangedRefCount      int                       `json:"unchanged_ref_count,omitempty"`
	FailureCount           int                       `json:"failure_count,omitempty"`
	WarningCount           int                       `json:"warning_count,omitempty"`
	MemoryMutation         bool                      `json:"memory_mutation"`
	RuntimeMutation        bool                      `json:"runtime_mutation"`
	ModelRuntimeCall       bool                      `json:"model_runtime_call"`
	EvidenceAdmission      bool                      `json:"evidence_admission"`
	ContextCompilation     bool                      `json:"context_compilation"`
	UserVisibleOutput      bool                      `json:"user_visible_output"`
	LiveAuthorityMigration bool                      `json:"live_authority_migration"`
	DurationMS             int64                     `json:"duration_ms"`
	Warnings               []string                  `json:"warnings,omitempty"`
	NormalizedRefs         []refvalidation.ObjectRef `json:"normalized_refs,omitempty"`
	Metadata               map[string]any            `json:"metadata,omitempty"`
}

type ShadowAdvisoryReport struct {
	ReportID                string                        `json:"report_id"`
	GeneratedAt             time.Time                     `json:"generated_at"`
	WorkspaceID             string                        `json:"workspace_id,omitempty"`
	RequestID               string                        `json:"request_id,omitempty"`
	CorrelationID           string                        `json:"correlation_id,omitempty"`
	SourceShadowReportRefs  []string                      `json:"source_shadow_report_refs,omitempty"`
	RouteMetadataRefs       []string                      `json:"route_metadata_refs,omitempty"`
	ChatMetadataRefs        []string                      `json:"chat_metadata_refs,omitempty"`
	RetrievalMetadataRefs   []string                      `json:"retrieval_metadata_refs,omitempty"`
	EvidenceSummary         ShadowEvidenceSummary         `json:"evidence_summary"`
	ConsensusAdvisory       ShadowConsensusAdvisory       `json:"consensus_advisory"`
	ContextCompilerAdvisory ShadowContextCompilerAdvisory `json:"context_compiler_advisory"`
	RiskSummary             ShadowRiskSummary             `json:"risk_summary"`
	Warnings                []string                      `json:"warnings,omitempty"`
	NoEffectVerified        bool                          `json:"no_effect_verified"`
	Metadata                map[string]any                `json:"metadata,omitempty"`
}

type ShadowEvidenceSummary struct {
	RouteMetadataCount     int      `json:"route_metadata_count"`
	ChatMetadataCount      int      `json:"chat_metadata_count"`
	RetrievalMetadataCount int      `json:"retrieval_metadata_count"`
	SafeRefCount           int      `json:"safe_ref_count"`
	MetadataOnly           bool     `json:"metadata_only"`
	SafeRefs               []string `json:"safe_refs,omitempty"`
}

type ShadowConsensusAdvisory struct {
	Status              string   `json:"status"`
	ProposedClaimCount  int      `json:"proposed_claim_count"`
	AcceptedClaimCount  int      `json:"accepted_claim_count"`
	RejectedClaimCount  int      `json:"rejected_claim_count"`
	UncertainClaimCount int      `json:"uncertain_claim_count"`
	Summary             string   `json:"summary"`
	Warnings            []string `json:"warnings,omitempty"`
}

type ShadowContextCompilerAdvisory struct {
	Status                  string         `json:"status"`
	BundleHash              string         `json:"bundle_hash,omitempty"`
	BlockCount              int            `json:"block_count"`
	CacheEligibilitySummary map[string]int `json:"cache_eligibility_summary,omitempty"`
	Warnings                []string       `json:"warnings,omitempty"`
}

type ShadowRiskSummary struct {
	RiskFlags            []string `json:"risk_flags,omitempty"`
	WarningCount         int      `json:"warning_count"`
	MetadataOnly         bool     `json:"metadata_only"`
	NoRawContentVerified bool     `json:"no_raw_content_verified"`
}
