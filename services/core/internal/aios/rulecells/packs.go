package rulecells

const (
	PackKernelAuthorityID = "forge.kernel.authority.v0"
	PackArterialRestoreID = "forge.arterial.restore.v0"
	PackLymphaticDreamID  = "forge.lymphatic.dream.v0"
	PackNeuralClassifyID  = "forge.neural.classification.v0"
	PackRuntimeRoutingID  = "forge.runtime.routing.v0"
	PackOperatorID        = "forge.operator.attention.v0"
	StaticPackVersion     = "0.1.0"
)

func StaticRulePacks() []RulePack {
	return []RulePack{
		kernelAuthorityPack(),
		neuralClassificationPack(),
		arterialRestorePack(),
		lymphaticDreamPack(),
		runtimeRoutingPack(),
		operatorAttentionPack(),
	}
}

func kernelAuthorityPack() RulePack {
	return RulePack{
		ID:           PackKernelAuthorityID,
		Version:      StaticPackVersion,
		Description:  "Deterministic authority-boundary warnings and stricter policy proposals.",
		MaxLatencyMs: 5,
		Rules: []RuleCell{
			{
				ID:          "kernel.reject_llm_direct_truth_write",
				Name:        "Reject direct LLM truth write",
				Description: "LLM/future semantic surfaces cannot directly mutate canonical truth.",
				Lane:        LaneKernel, Phase: PhaseSyscallValidation, Priority: 100, Enabled: true,
				InputType: "authority_precheck",
				Condition: Condition{Field: "direct_truth_mutation_attempt", Operator: OpBoolIs, Value: true},
				Action:    "reject", Version: "0.1.0",
				Explain: "direct truth mutation must go through semantic syscalls",
				Output:  RuleOutput{Type: OutputRejectDecision, Decision: "reject", Severity: "high", Explain: "direct truth mutation must go through semantic syscalls", Tags: []string{"authority", "truth"}},
			},
			{
				ID:   "kernel.require_approval_destructive_high_risk",
				Name: "Require approval for destructive high-risk work",
				Lane: LaneKernel, Phase: PhaseGatewayPrecheck, Priority: 90, Enabled: true,
				InputType: "gateway_precheck",
				Condition: Condition{Field: "risk_class", Operator: OpRiskClassMatch, Values: []string{"high", "critical", "destructive"}},
				Action:    "policy", Version: "0.1.0",
				Output: RuleOutput{Type: OutputPolicyDecision, Decision: "approval_required", Severity: "high", Explain: "destructive or high-risk action requires approval", Tags: []string{"approval", "risk"}},
			},
			{
				ID:   "kernel.warn_wrong_workspace_scope",
				Name: "Warn on wrong workspace or scope",
				Lane: LaneKernel, Phase: PhaseSyscallValidation, Priority: 85, Enabled: true,
				InputType: "authority_precheck",
				Condition: Condition{Field: "wrong_workspace", Operator: OpBoolIs, Value: true},
				Action:    "reject", Version: "0.1.0",
				Output: RuleOutput{Type: OutputRejectDecision, Decision: "reject", Severity: "high", Explain: "workspace scope mismatch cannot be accepted", Tags: []string{"workspace", "scope"}},
			},
			{
				ID:   "kernel.warn_missing_provenance",
				Name: "Warn on missing provenance",
				Lane: LaneKernel, Phase: PhaseSyscallValidation, Priority: 70, Enabled: true,
				InputType: "authority_precheck",
				Condition: Condition{Field: "missing_provenance", Operator: OpBoolIs, Value: true},
				Action:    "policy", Version: "0.1.0",
				Output: RuleOutput{Type: OutputPolicyDecision, Decision: "warn", Severity: "medium", Explain: "missing provenance weakens traceability", Tags: []string{"provenance"}},
			},
		},
	}
}

func neuralClassificationPack() RulePack {
	return RulePack{
		ID:           PackNeuralClassifyID,
		Version:      StaticPackVersion,
		Description:  "Fast deterministic event/input classification tags.",
		MaxLatencyMs: 5,
		Rules: []RuleCell{
			{
				ID: "neural.classify.user_correction", Name: "Correction tagging",
				Lane: LaneNeural, Phase: PhaseIngestClassification, Priority: 100, Enabled: true,
				InputType: "event", Condition: Condition{Field: "text", Operator: OpContains, Values: []string{"correct", "correction", "actually", "instead"}},
				Action: "route", Version: "0.1.0",
				Output: RuleOutput{Type: OutputRouteDecision, Decision: "tag_correction", Explain: "input looks like a correction", Tags: []string{"correction", "memory_candidate"}},
			},
			{
				ID: "neural.classify.failure", Name: "Failure tagging",
				Lane: LaneNeural, Phase: PhaseIngestClassification, Priority: 90, Enabled: true,
				InputType: "event", Condition: Condition{Field: "text", Operator: OpContains, Values: []string{"failed", "failure", "error", "timeout", "blocked"}},
				Action: "route", Version: "0.1.0",
				Output: RuleOutput{Type: OutputRouteDecision, Decision: "tag_failure", Explain: "input looks like a failure or blocked loop", Tags: []string{"failure", "blocked_loop"}},
			},
			{
				ID: "neural.classify.task", Name: "Task tagging",
				Lane: LaneNeural, Phase: PhaseIngestClassification, Priority: 80, Enabled: true,
				InputType: "event", Condition: Condition{Field: "text", Operator: OpContains, Values: []string{"todo", "task", "next action", "follow up"}},
				Action: "route", Version: "0.1.0",
				Output: RuleOutput{Type: OutputRouteDecision, Decision: "tag_task", Explain: "input looks like a task candidate", Tags: []string{"task"}},
			},
			{
				ID: "neural.classify.memory_candidate", Name: "Memory candidate tagging",
				Lane: LaneNeural, Phase: PhaseIngestClassification, Priority: 70, Enabled: true,
				InputType: "event", Condition: Condition{Field: "important", Operator: OpBoolIs, Value: true},
				Action: "route", Version: "0.1.0",
				Output: RuleOutput{Type: OutputMemoryProposal, Decision: "memory_candidate", Explain: "input was flagged as important", Tags: []string{"memory_candidate"}},
			},
		},
	}
}

func arterialRestorePack() RulePack {
	return RulePack{
		ID:           PackArterialRestoreID,
		Version:      StaticPackVersion,
		Description:  "Advisory restore score adjustments and fresh-compile routing hints.",
		MaxLatencyMs: 5,
		Rules: []RuleCell{
			{
				ID: "arterial.restore.exact_query_boost", Name: "Exact query boost",
				Lane: LaneArterial, Phase: PhaseRestoreScoring, Priority: 100, Enabled: true,
				InputType: "restore_candidate", Condition: Condition{Field: "query_exact", Operator: OpBoolIs, Value: true},
				Action: "score_adjustment", ScoreDelta: 0.04, Version: "0.1.0",
				Output: RuleOutput{Type: OutputScoreAdjustment, ScoreDelta: 0.04, Explain: "exact restore query match", Tags: []string{"restore", "query"}},
			},
			{
				ID: "arterial.restore.partial_query_overlap_boost", Name: "Partial query overlap boost",
				Lane: LaneArterial, Phase: PhaseRestoreScoring, Priority: 90, Enabled: true,
				InputType: "restore_candidate", Condition: Condition{Field: "query_overlap", Operator: OpNumericGTE, Threshold: 0.5},
				Action: "score_adjustment", ScoreDelta: 0.02, Version: "0.1.0",
				Output: RuleOutput{Type: OutputScoreAdjustment, ScoreDelta: 0.02, Explain: "partial restore query overlap", Tags: []string{"restore", "query"}},
			},
			{
				ID: "arterial.restore.stale_snapshot_penalty", Name: "Stale snapshot penalty",
				Lane: LaneArterial, Phase: PhaseRestoreScoring, Priority: 85, Enabled: true,
				InputType: "restore_candidate", Condition: Condition{Field: "stale", Operator: OpBoolIs, Value: true},
				Action: "score_adjustment", ScoreDelta: -0.05, Version: "0.1.0",
				Output: RuleOutput{Type: OutputScoreAdjustment, ScoreDelta: -0.05, Explain: "stale restore snapshot penalty", Tags: []string{"restore", "stale"}},
			},
			{
				ID: "arterial.restore.contradiction_marker_penalty", Name: "Contradiction marker penalty",
				Lane: LaneArterial, Phase: PhaseRestoreScoring, Priority: 80, Enabled: true,
				InputType: "restore_candidate", Condition: Condition{Field: "contradiction_marker", Operator: OpBoolIs, Value: true},
				Action: "score_adjustment", ScoreDelta: -0.05, Version: "0.1.0",
				Output: RuleOutput{Type: OutputScoreAdjustment, ScoreDelta: -0.05, Explain: "contradiction marker restore penalty", Tags: []string{"restore", "contradiction"}},
			},
			{
				ID: "arterial.restore.low_score_fresh_compile", Name: "Low score requires fresh compile",
				Lane: LaneArterial, Phase: PhaseRestoreScoring, Priority: 75, Enabled: true,
				InputType: "restore_candidate", Condition: Condition{Field: "base_score", Operator: OpNumericLT, Threshold: 0.35},
				Action: "fresh_compile_required", Version: "0.1.0",
				Output: RuleOutput{Type: OutputFreshCompileRequired, Decision: "requires_fresh_compile", Explain: "restore score too low for direct resume", Tags: []string{"restore", "fresh_compile"}},
			},
			{
				ID: "arterial.restore.high_confidence_boost", Name: "High-confidence restore boost",
				Lane: LaneArterial, Phase: PhaseRestoreScoring, Priority: 70, Enabled: true,
				InputType: "restore_candidate", Condition: Condition{Field: "confidence", Operator: OpNumericGTE, Threshold: 0.8},
				Action: "score_adjustment", ScoreDelta: 0.03, Version: "0.1.0",
				Output: RuleOutput{Type: OutputScoreAdjustment, ScoreDelta: 0.03, Explain: "high-confidence restore candidate", Tags: []string{"restore", "confidence"}},
			},
			{
				ID: "arterial.restore.wrong_workspace_reject", Name: "Wrong workspace reject",
				Lane: LaneArterial, Phase: PhaseRestoreScoring, Priority: 110, Enabled: true,
				InputType: "restore_candidate", Condition: Condition{Field: "wrong_workspace", Operator: OpBoolIs, Value: true},
				Action: "reject", Version: "0.1.0",
				Output: RuleOutput{Type: OutputRejectDecision, Decision: "reject", Severity: "high", Explain: "wrong workspace candidates must not be restored", Tags: []string{"restore", "workspace"}},
			},
		},
	}
}

func lymphaticDreamPack() RulePack {
	return RulePack{
		ID:           PackLymphaticDreamID,
		Version:      StaticPackVersion,
		Description:  "Advisory Dream Mode salience and tier-routing hints.",
		MaxLatencyMs: 5,
		Rules: []RuleCell{
			{
				ID: "lymphatic.dream.user_correction_salience", Name: "User correction high salience",
				Lane: LaneLymphatic, Phase: PhaseSalienceScoring, Priority: 100, Enabled: true,
				InputType: "dream_candidate", Condition: Condition{Field: "tags", Operator: OpTagMatch, Values: []string{"correction", "preference", "user_correction"}},
				Action: "score_adjustment", ScoreDelta: 0.08, Version: "0.1.0",
				Output: RuleOutput{Type: OutputScoreAdjustment, ScoreDelta: 0.08, Explain: "user correction deserves high replay salience", Tags: []string{"dream", "correction"}},
			},
			{
				ID: "lymphatic.dream.unresolved_contradiction_salience", Name: "Unresolved contradiction high salience",
				Lane: LaneLymphatic, Phase: PhaseSalienceScoring, Priority: 95, Enabled: true,
				InputType: "dream_candidate", Condition: Condition{Field: "tags", Operator: OpTagMatch, Values: []string{"contradiction", "unresolved", "conflict"}},
				Action: "score_adjustment", ScoreDelta: 0.07, Version: "0.1.0",
				Output: RuleOutput{Type: OutputScoreAdjustment, ScoreDelta: 0.07, Explain: "unresolved contradiction requires replay attention", Tags: []string{"dream", "contradiction"}},
			},
			{
				ID: "lymphatic.dream.block_long_term_on_contradiction", Name: "Block long-term promotion on contradiction",
				Lane: LaneLymphatic, Phase: PhaseMemoryTierRouting, Priority: 100, Enabled: true,
				InputType: "dream_candidate", Condition: Condition{Field: "contradiction_score", Operator: OpNumericGT, Threshold: 0},
				Action: "policy", Version: "0.1.0",
				Output: RuleOutput{Type: OutputPolicyDecision, Decision: "block_long_term_promotion", Severity: "high", Explain: "unresolved contradiction blocks long-term promotion", Tags: []string{"dream", "promotion", "contradiction"}},
			},
			{
				ID: "lymphatic.dream.active_blocked_loop_salience", Name: "Active blocked loop salience",
				Lane: LaneLymphatic, Phase: PhaseSalienceScoring, Priority: 85, Enabled: true,
				InputType: "dream_candidate", Condition: Condition{Field: "tags", Operator: OpTagMatch, Values: []string{"blocked", "blocker", "active"}},
				Action: "score_adjustment", ScoreDelta: 0.05, Version: "0.1.0",
				Output: RuleOutput{Type: OutputScoreAdjustment, ScoreDelta: 0.05, Explain: "active blocked loop raises salience", Tags: []string{"dream", "blocked_loop"}},
			},
			{
				ID: "lymphatic.dream.repeated_failure_boost", Name: "Repeated failure salience boost",
				Lane: LaneLymphatic, Phase: PhaseSalienceScoring, Priority: 80, Enabled: true,
				InputType: "dream_candidate", Condition: Condition{Field: "failure_count", Operator: OpNumericGTE, Threshold: 2},
				Action: "score_adjustment", ScoreDelta: 0.04, Version: "0.1.0",
				Output: RuleOutput{Type: OutputScoreAdjustment, ScoreDelta: 0.04, Explain: "repeated failure should be replayed", Tags: []string{"dream", "failure"}},
			},
		},
	}
}

func runtimeRoutingPack() RulePack {
	return RulePack{
		ID:           PackRuntimeRoutingID,
		Version:      StaticPackVersion,
		Description:  "Runtime advisory routing hints only.",
		MaxLatencyMs: 5,
		Rules: []RuleCell{
			{
				ID: "runtime.provider_cooldown_blocks_retry", Name: "Provider cooldown blocks retry",
				Lane: LaneRuntime, Phase: PhaseProviderCooldown, Priority: 100, Enabled: true,
				InputType: "runtime_status", Condition: Condition{Field: "provider_status", Operator: OpProviderStatusMatch, Value: "cooldown"},
				Action: "policy", Version: "0.1.0",
				Output: RuleOutput{Type: OutputPolicyDecision, Decision: "reject", Severity: "high", Explain: "provider cooldown blocks retry/model call", Tags: []string{"runtime", "cooldown"}},
			},
			{
				ID: "runtime.unavailable_blocks_inference", Name: "Modelruntime unavailable blocks inference",
				Lane: LaneRuntime, Phase: PhaseModelRouting, Priority: 95, Enabled: true,
				InputType: "runtime_status", Condition: Condition{Field: "runtime_status", Operator: OpStatusMatch, Value: "unavailable"},
				Action: "policy", Version: "0.1.0",
				Output: RuleOutput{Type: OutputPolicyDecision, Decision: "unavailable", Severity: "high", Explain: "modelruntime unavailable blocks inference-required action", Tags: []string{"runtime", "unavailable"}},
			},
			{
				ID: "runtime.background_gpu_defer_overloaded", Name: "Background GPU defer when overloaded",
				Lane: LaneRuntime, Phase: PhaseModelRouting, Priority: 80, Enabled: true,
				InputType: "runtime_status", Condition: Condition{Field: "runtime_overloaded", Operator: OpBoolIs, Value: true},
				Action: "background_defer", Version: "0.1.0",
				Output: RuleOutput{Type: OutputBackgroundDefer, Decision: "defer", Explain: "background GPU work should defer while runtime is overloaded", Tags: []string{"runtime", "background"}},
			},
			{
				ID: "runtime.status_only_no_model", Name: "Status-only requests need no model",
				Lane: LaneRuntime, Phase: PhaseModelRouting, Priority: 60, Enabled: true,
				InputType: "runtime_status", Condition: Condition{Field: "status_only", Operator: OpBoolIs, Value: true},
				Action: "model_routing_hint", Version: "0.1.0",
				Output: RuleOutput{Type: OutputModelRoutingHint, Decision: "no_model_needed", Explain: "status-only request can be answered without modelruntime", Tags: []string{"runtime", "no_model"}},
			},
		},
	}
}

func operatorAttentionPack() RulePack {
	return RulePack{
		ID:           PackOperatorID,
		Version:      StaticPackVersion,
		Description:  "Operator attention signals.",
		MaxLatencyMs: 5,
		Rules: []RuleCell{
			{
				ID: "operator.blocked_loop_attention", Name: "Blocked loop raises attention",
				Lane: LaneOperator, Phase: PhaseAttentionRouting, Priority: 100, Enabled: true,
				InputType: "operator_summary", Condition: Condition{Field: "blocked_loop", Operator: OpBoolIs, Value: true},
				Action: "attention", Version: "0.1.0",
				Output: RuleOutput{Type: OutputAttentionSignal, Decision: "raise_attention", Severity: "high", Explain: "blocked loop needs operator attention", Tags: []string{"operator", "blocked_loop"}},
			},
			{
				ID: "operator.failed_job_attention", Name: "Failed job raises attention",
				Lane: LaneOperator, Phase: PhaseAttentionRouting, Priority: 90, Enabled: true,
				InputType: "operator_summary", Condition: Condition{Field: "failed_job", Operator: OpBoolIs, Value: true},
				Action: "attention", Version: "0.1.0",
				Output: RuleOutput{Type: OutputAttentionSignal, Decision: "raise_attention", Severity: "medium", Explain: "failed job needs inspection", Tags: []string{"operator", "failed_job"}},
			},
			{
				ID: "operator.degraded_runtime_attention", Name: "Degraded runtime raises attention",
				Lane: LaneOperator, Phase: PhaseAttentionRouting, Priority: 85, Enabled: true,
				InputType: "operator_summary", Condition: Condition{Field: "runtime_status", Operator: OpStatusMatch, Value: "degraded"},
				Action: "attention", Version: "0.1.0",
				Output: RuleOutput{Type: OutputAttentionSignal, Decision: "raise_attention", Severity: "medium", Explain: "degraded modelruntime should be surfaced", Tags: []string{"operator", "runtime"}},
			},
			{
				ID: "operator.dream_review_attention", Name: "Dream review needed attention",
				Lane: LaneOperator, Phase: PhaseAttentionRouting, Priority: 80, Enabled: true,
				InputType: "operator_summary", Condition: Condition{Field: "dream_review_needed", Operator: OpBoolIs, Value: true},
				Action: "attention", Version: "0.1.0",
				Output: RuleOutput{Type: OutputAttentionSignal, Decision: "raise_attention", Severity: "medium", Explain: "Dream report review item needs operator attention", Tags: []string{"operator", "dream"}},
			},
		},
	}
}
