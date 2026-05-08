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
