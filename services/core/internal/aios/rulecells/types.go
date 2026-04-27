package rulecells

import "time"

type Lane string

const (
	LaneKernel    Lane = "kernel"
	LaneNeural    Lane = "neural"
	LaneArterial  Lane = "arterial"
	LaneLymphatic Lane = "lymphatic"
	LaneOperator  Lane = "operator"
	LaneRuntime   Lane = "runtime"
)

type Phase string

const (
	PhaseSyscallValidation      Phase = "syscall_validation"
	PhaseIngestClassification   Phase = "ingest_classification"
	PhaseEventAdmission         Phase = "event_admission"
	PhaseRestoreScoring         Phase = "restore_scoring"
	PhaseWorkingMemoryAdmission Phase = "working_memory_admission"
	PhaseModelRouting           Phase = "model_routing"
	PhaseGatewayPrecheck        Phase = "gateway_precheck"
	PhaseProviderCooldown       Phase = "provider_cooldown"
	PhaseReplaySelection        Phase = "replay_selection"
	PhaseSalienceScoring        Phase = "salience_scoring"
	PhaseMemoryTierRouting      Phase = "memory_tier_routing"
	PhaseRepairSelection        Phase = "repair_selection"
	PhaseAttentionRouting       Phase = "attention_routing"
)

type OutputType string

const (
	OutputRouteDecision        OutputType = "RouteDecision"
	OutputScoreAdjustment      OutputType = "ScoreAdjustment"
	OutputPolicyDecision       OutputType = "PolicyDecision"
	OutputMemoryProposal       OutputType = "MemoryProposal"
	OutputAttentionSignal      OutputType = "AttentionSignal"
	OutputRepairProposal       OutputType = "RepairProposal"
	OutputModelRoutingHint     OutputType = "ModelRoutingHint"
	OutputFreshCompileRequired OutputType = "FreshCompileRequired"
	OutputVerifierRequired     OutputType = "VerifierRequired"
	OutputBackgroundDefer      OutputType = "BackgroundDefer"
	OutputRejectDecision       OutputType = "RejectDecision"
)

type ConditionOperator string

const (
	OpEquals              ConditionOperator = "equals"
	OpContains            ConditionOperator = "contains"
	OpNumericGT           ConditionOperator = "numeric_gt"
	OpNumericGTE          ConditionOperator = "numeric_gte"
	OpNumericLT           ConditionOperator = "numeric_lt"
	OpNumericLTE          ConditionOperator = "numeric_lte"
	OpAgeGTE              ConditionOperator = "age_gte"
	OpStatusMatch         ConditionOperator = "status_match"
	OpTagMatch            ConditionOperator = "tag_match"
	OpTokenOverlapGTE     ConditionOperator = "token_overlap_gte"
	OpBoolIs              ConditionOperator = "bool_is"
	OpProviderStatusMatch ConditionOperator = "provider_status_match"
	OpRiskClassMatch      ConditionOperator = "risk_class_match"
)

type Condition struct {
	Field     string            `json:"field"`
	Operator  ConditionOperator `json:"operator"`
	Value     any               `json:"value,omitempty"`
	Values    []string          `json:"values,omitempty"`
	Threshold float64           `json:"threshold,omitempty"`
	AgeMs     int64             `json:"ageMs,omitempty"`
}

type RuleOutput struct {
	Type       OutputType     `json:"type"`
	Decision   string         `json:"decision,omitempty"`
	ScoreDelta float64        `json:"score_delta,omitempty"`
	Weight     float64        `json:"weight,omitempty"`
	Tags       []string       `json:"tags,omitempty"`
	Explain    string         `json:"explain,omitempty"`
	Severity   string         `json:"severity,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

type RuleCell struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Description  string         `json:"description,omitempty"`
	Lane         Lane           `json:"lane"`
	Phase        Phase          `json:"phase"`
	Priority     int            `json:"priority"`
	Enabled      bool           `json:"enabled"`
	InputType    string         `json:"input_type,omitempty"`
	Condition    Condition      `json:"condition"`
	Action       string         `json:"action,omitempty"`
	ScoreDelta   float64        `json:"score_delta,omitempty"`
	Weight       float64        `json:"weight,omitempty"`
	Tags         []string       `json:"tags,omitempty"`
	Explain      string         `json:"explain,omitempty"`
	Version      string         `json:"version"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	MaxLatencyMs int64          `json:"max_latency_ms,omitempty"`
	Output       RuleOutput     `json:"output"`
}

type RulePack struct {
	ID           string     `json:"pack_id"`
	Version      string     `json:"version"`
	Description  string     `json:"description,omitempty"`
	MaxLatencyMs int64      `json:"max_latency_ms,omitempty"`
	Rules        []RuleCell `json:"rules"`
}

type RulePackRef struct {
	ID      string `json:"pack_id"`
	Version string `json:"version"`
}

type RunInput struct {
	Lane                  Lane           `json:"lane"`
	Phase                 Phase          `json:"phase"`
	InputID               string         `json:"input_id"`
	InputType             string         `json:"input_type,omitempty"`
	Facts                 map[string]any `json:"facts,omitempty"`
	AuthoritativeDecision string         `json:"authoritative_decision,omitempty"`
}

type RunOptions struct {
	DryRun       bool  `json:"dry_run,omitempty"`
	Debug        bool  `json:"debug,omitempty"`
	Disabled     bool  `json:"disabled,omitempty"`
	MaxLatencyMs int64 `json:"max_latency_ms,omitempty"`
}

type MatchedRuleTrace struct {
	RuleID      string   `json:"rule_id"`
	RuleVersion string   `json:"rule_version,omitempty"`
	PackID      string   `json:"pack_id"`
	PackVersion string   `json:"pack_version"`
	OutputTypes []string `json:"output_types,omitempty"`
	Explain     string   `json:"explain,omitempty"`
}

type RuleTrace struct {
	TraceID        string             `json:"trace_id"`
	Lane           Lane               `json:"lane"`
	Phase          Phase              `json:"phase"`
	InputID        string             `json:"input_id"`
	StartedAt      int64              `json:"started_at"`
	CompletedAt    int64              `json:"completed_at"`
	LatencyMs      int64              `json:"latency_ms"`
	RulesEvaluated int                `json:"rules_evaluated"`
	RulePacks      []RulePackRef      `json:"rule_packs,omitempty"`
	MatchedRules   []MatchedRuleTrace `json:"matched_rules"`
	Outputs        []RuleOutput       `json:"outputs"`
	Warnings       []string           `json:"warnings"`
	NonMatches     []string           `json:"non_matches,omitempty"`
}

type RunResult struct {
	Outputs  []RuleOutput `json:"outputs"`
	Trace    RuleTrace    `json:"trace"`
	Warnings []string     `json:"warnings"`
}

type EngineOptions struct {
	Packs []RulePack
	Clock func() time.Time
}
