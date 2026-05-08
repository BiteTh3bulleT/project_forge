package forgeh

import "time"

const (
	ResourcePressureUnavailable = "unavailable"
	ResourcePressureNormal      = "normal"
	ResourcePressureElevated    = "elevated"
	ResourcePressureConstrained = "constrained"
	ResourcePressureCritical    = "critical"
)

const (
	ResourcePostureNormal      = "normal"
	ResourcePostureDegraded    = "degraded"
	ResourcePostureConstrained = "constrained"
	ResourcePostureCritical    = "critical"
)

const (
	WorkloadLaneInteractive      = "interactive"
	WorkloadLaneBackgroundIngest = "background_ingest"
	WorkloadLaneEmbedding        = "embedding"
	WorkloadLaneModelLoad        = "model_load"
	WorkloadLaneModelInference   = "model_inference"
	WorkloadLaneMaintenance      = "maintenance"
	WorkloadLaneDesktopUI        = "desktop_ui"
)

const (
	PolicyDecisionAllow            = "allow"
	PolicyDecisionAllowWithWarning = "allow_with_warning"
	PolicyDecisionDefer            = "defer"
	PolicyDecisionDeny             = "deny"
	PolicyDecisionUnavailable      = "unavailable"
)

const (
	ModelLoadSmallLocalOK     = "small_local_ok"
	ModelLoadCurrentModelOnly = "current_model_only"
	ModelLoadDeferLargeModel  = "defer_large_model"
	ModelLoadCPUOnlySafeMode  = "cpu_only_safe_mode"
	ModelLoadDenyNewModelLoad = "deny_new_model_load"
	ModelLoadUnavailable      = "unavailable"
)

const (
	BackgroundWorkAllow = "allow_background_work"
	BackgroundWorkDefer = "defer_background_work"
	BackgroundWorkDeny  = "deny_background_work"
)

const (
	ProposalActionPauseBackgroundIngest  = "pause_background_ingest"
	ProposalActionDeferBackgroundIngest  = "defer_background_ingest"
	ProposalActionDeferEmbedding         = "defer_embedding"
	ProposalActionDenyNewModelLoad       = "deny_new_model_load"
	ProposalActionDeferLargeModelLoad    = "defer_large_model_load"
	ProposalActionPreferCurrentModelOnly = "prefer_current_model_only"
	ProposalActionPreferCPUSafeMode      = "prefer_cpu_safe_mode"
	ProposalActionWarnOperator           = "warn_operator"
	ProposalActionEnterDegradedMode      = "enter_degraded_mode"
	ProposalActionScheduleMaintenance    = "schedule_maintenance_later"
)

const (
	ProposalStatusProposed       = "proposed"
	ProposalStatusApproved       = "approved"
	ProposalStatusRejected       = "rejected"
	ProposalStatusExpired        = "expired"
	ProposalStatusSuperseded     = "superseded"
	ProposalStatusCommittedLater = "committed_later"
)

const (
	ProposalRiskLow      = "low"
	ProposalRiskModerate = "moderate"
	ProposalRiskHigh     = "high"
	ProposalRiskCritical = "critical"
)

const (
	ExecutionStatusPlanned  = "planned"
	ExecutionStatusExecuted = "executed"
	ExecutionStatusSkipped  = "skipped"
	ExecutionStatusRejected = "rejected"
	ExecutionStatusFailed   = "failed"
	ExecutionStatusBlocked  = "blocked"
)

const (
	ExecutionResultOperatorWarned            = "operator_warned"
	ExecutionResultBackgroundIngestDeferred  = "background_ingest_deferred"
	ExecutionResultBackgroundIngestPaused    = "background_ingest_paused"
	ExecutionResultEmbeddingDeferred         = "embedding_deferred"
	ExecutionResultNewModelLoadDenied        = "new_model_load_denied"
	ExecutionResultLargeModelLoadDeferred    = "large_model_load_deferred"
	ExecutionResultCurrentModelOnlyPreferred = "current_model_only_preferred"
	ExecutionResultCPUSafeModePreferred      = "cpu_safe_mode_preferred"
	ExecutionResultDegradedModeEntered       = "degraded_mode_entered"
	ExecutionResultMaintenanceScheduledLater = "maintenance_scheduled_later"
)

type ResourcePolicySnapshot struct {
	PolicyID                     string                  `json:"policy_id"`
	CapturedAt                   time.Time               `json:"captured_at"`
	SourceSnapshotID             string                  `json:"source_snapshot_id"`
	OverallPosture               string                  `json:"overall_posture"`
	RAMPressure                  string                  `json:"ram_pressure"`
	SwapPressure                 string                  `json:"swap_pressure"`
	DiskPressure                 string                  `json:"disk_pressure"`
	VRAMPressure                 string                  `json:"vram_pressure"`
	ThermalPressure              string                  `json:"thermal_pressure"`
	LaneDecisions                map[string]LaneDecision `json:"lane_decisions"`
	ModelLoadRecommendation      string                  `json:"model_load_recommendation"`
	BackgroundWorkRecommendation string                  `json:"background_work_recommendation"`
	Warnings                     []string                `json:"warnings,omitempty"`
	OperatorActions              []string                `json:"operator_actions,omitempty"`
	SourceErrors                 []ResourcePolicyError   `json:"source_errors,omitempty"`
	AdvisoryOnly                 bool                    `json:"advisory_only"`
}

type LaneDecision struct {
	Decision string   `json:"decision"`
	Reasons  []string `json:"reasons,omitempty"`
}

type ResourcePolicyError struct {
	Source string `json:"source"`
	Error  string `json:"error"`
}

type ResourceActionProposal struct {
	ProposalID               string    `json:"proposal_id"`
	CreatedAt                time.Time `json:"created_at"`
	SourcePolicyID           string    `json:"source_policy_id"`
	SourceSnapshotID         string    `json:"source_snapshot_id"`
	ActionType               string    `json:"action_type"`
	TargetLane               string    `json:"target_lane,omitempty"`
	RecommendedDecision      string    `json:"recommended_decision"`
	Reason                   string    `json:"reason"`
	RiskLevel                string    `json:"risk_level"`
	RequiresOperatorApproval bool      `json:"requires_operator_approval"`
	Status                   string    `json:"status"`
	ExpiresAt                time.Time `json:"expires_at"`
	AdvisoryOnly             bool      `json:"advisory_only"`
	SupersededBy             string    `json:"superseded_by,omitempty"`
	Supersedes               string    `json:"supersedes,omitempty"`
	DecisionReason           string    `json:"decision_reason,omitempty"`
	DecidedAt                time.Time `json:"decided_at,omitempty"`
}

type ResourceActionExecution struct {
	ExecutionID              string    `json:"execution_id"`
	ProposalID               string    `json:"proposal_id"`
	SourcePolicyID           string    `json:"source_policy_id"`
	SourceSnapshotID         string    `json:"source_snapshot_id"`
	ActionType               string    `json:"action_type"`
	TargetLane               string    `json:"target_lane,omitempty"`
	Status                   string    `json:"status"`
	StartedAt                time.Time `json:"started_at"`
	FinishedAt               time.Time `json:"finished_at,omitempty"`
	Result                   string    `json:"result,omitempty"`
	Reason                   string    `json:"reason,omitempty"`
	SideEffects              []string  `json:"side_effects"`
	OperatorApprovalRequired bool      `json:"operator_approval_required"`
	ApprovedBeforeExecution  bool      `json:"approved_before_execution"`
	Bounded                  bool      `json:"bounded"`
	HostMutation             bool      `json:"host_mutation"`
	SemanticMemoryWrite      bool      `json:"semantic_memory_write"`
	ModelruntimeMutation     bool      `json:"modelruntime_mutation"`
	Errors                   []string  `json:"errors"`
}
