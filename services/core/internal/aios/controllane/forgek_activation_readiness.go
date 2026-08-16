package controllane

import (
	"time"

	"forge/projectforge/services/core/internal/aios/domain"
)

const ForgeKActivationReadinessPhase = "19"

type ForgeKActivationReadinessReport struct {
	GeneratedAt                time.Time                         `json:"generated_at"`
	Phase                      string                            `json:"phase"`
	Status                     string                            `json:"status"`
	Summary                    string                            `json:"summary"`
	Mode                       string                            `json:"mode"`
	LiveOwner                  string                            `json:"live_owner"`
	PolicyVersion              string                            `json:"policy_version"`
	KernelRuntimeState         string                            `json:"kernel_runtime_state"`
	ClosedValidationLanes      int                               `json:"closed_validation_lanes"`
	TotalValidationLanes       int                               `json:"total_validation_lanes"`
	ValidationActions          []ForgeKActivationActionReadiness `json:"validation_actions"`
	AuthorityReadyGates        int                               `json:"authority_ready_gates"`
	AuthorityBlockedGates      int                               `json:"authority_blocked_gates"`
	AuthorityGates             []ForgeKAuthorityGateReadiness    `json:"authority_gates"`
	AuthorityMatrix            []ForgeKAuthorityGateMatrixEntry  `json:"authority_matrix"`
	Gates                      []ForgeKActivationReadinessGate   `json:"gates"`
	NoEffect                   map[string]any                    `json:"no_effect"`
	SimulatorAuthority         bool                              `json:"simulator_authority"`
	LiveKernelIngressAuthority bool                              `json:"live_kernel_ingress_authority"`
	LiveDurableOrchestration   bool                              `json:"live_durable_orchestration"`
	LiveKernelAuthority        bool                              `json:"live_kernel_authority"`
	LiveAuthorityMigration     bool                              `json:"live_authority_migration"`
	ShadowAuthoritative        bool                              `json:"shadow_authoritative"`
	MutationControlsAvailable  bool                              `json:"mutation_controls_available"`
	Notes                      []string                          `json:"notes"`
}

type ForgeKActivationActionReadiness struct {
	Action              domain.SemanticActionType `json:"action"`
	Capability          string                    `json:"capability"`
	Registered          bool                      `json:"registered"`
	Mutating            bool                      `json:"mutating"`
	ApprovalPossible    bool                      `json:"approval_possible"`
	SupportsDryRun      bool                      `json:"supports_dry_run"`
	AuditEventName      string                    `json:"audit_event_name"`
	Closed              bool                      `json:"closed"`
	Mode                string                    `json:"mode"`
	LiveOwner           string                    `json:"live_owner"`
	SimulatorAuthority  bool                      `json:"simulator_authority"`
	LiveKernelAuthority bool                      `json:"live_kernel_authority"`
}

type ForgeKActivationReadinessGate struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Reason string `json:"reason"`
}

type ForgeKAuthorityGateReadiness struct {
	Name                     string `json:"name"`
	Status                   string `json:"status"`
	LiveOwner                string `json:"live_owner"`
	RequiredForLiveAuthority bool   `json:"required_for_live_authority"`
	MutationAuthority        bool   `json:"mutation_authority"`
	Reason                   string `json:"reason"`
	NextStep                 string `json:"next_step"`
}

type ForgeKAuthorityGateMatrixEntry struct {
	Subsystem       string   `json:"subsystem"`
	CurrentStatus   string   `json:"current_status"`
	LiveOwner       string   `json:"live_owner"`
	TargetOwner     string   `json:"target_owner"`
	FeatureFlag     string   `json:"feature_flag"`
	RollbackPath    string   `json:"rollback_path"`
	TestsRequired   []string `json:"tests_required"`
	TestsPassing    []string `json:"tests_passing"`
	Blockers        []string `json:"blockers"`
	OperatorVisible bool     `json:"operator_visible"`
}

func ForgeKActivationReadiness(reg ActionRegistry, now time.Time) ForgeKActivationReadinessReport {
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	if reg == nil {
		reg = NewStaticActionRegistry()
	}

	required := []domain.SemanticActionType{
		domain.ActionValidateKVIdentity,
		domain.ActionValidateRefShape,
		domain.ActionCompareRefShape,
		domain.ActionValidateSourceObject,
		domain.ActionValidateSemanticOperation,
		domain.ActionValidateAdmissionCandidate,
		domain.ActionValidateContextAttribution,
	}
	actions := make([]ForgeKActivationActionReadiness, 0, len(required))
	registered := true
	nonMutating := true
	noApprovals := true
	closed := 0
	for _, action := range required {
		def, ok := reg.Get(action)
		item := ForgeKActivationActionReadiness{
			Action:              action,
			Registered:          ok,
			Closed:              false,
			Mode:                ForgeKActivationModePartialLiveEnforcement,
			LiveOwner:           ForgeKActivationOwnerControlLane,
			SimulatorAuthority:  false,
			LiveKernelAuthority: false,
		}
		if ok {
			item.Capability = def.Capability
			item.Mutating = def.Mutating
			item.ApprovalPossible = def.ApprovalPossible
			item.SupportsDryRun = def.SupportsDryRun
			item.AuditEventName = def.AuditEventName
			item.Closed = !def.Mutating && !def.ApprovalPossible && def.Capability != "" && def.AuditEventName != ""
		}
		if !item.Registered {
			registered = false
		}
		if item.Mutating {
			nonMutating = false
		}
		if item.ApprovalPossible {
			noApprovals = false
		}
		if item.Closed {
			closed++
		}
		actions = append(actions, item)
	}

	ready := registered && nonMutating && noApprovals && closed == len(required)
	status := "partial_live_validation_ready"
	if !ready {
		status = "blocked"
	}
	authorityGates := forgeKAuthorityGateReadiness(ready)
	authorityReady, authorityBlocked := countForgeKAuthorityGates(authorityGates)

	return ForgeKActivationReadinessReport{
		GeneratedAt:           now,
		Phase:                 ForgeKActivationReadinessPhase,
		Status:                status,
		Summary:               "FORGE-K is active only as live Control Lane validation metadata; simulator services remain non-authoritative.",
		Mode:                  ForgeKActivationModePartialLiveEnforcement,
		LiveOwner:             ForgeKActivationOwnerControlLane,
		PolicyVersion:         ForgeKActivationPolicyVersion,
		KernelRuntimeState:    "partial_live_validation",
		ClosedValidationLanes: closed,
		TotalValidationLanes:  len(required),
		ValidationActions:     actions,
		AuthorityReadyGates:   authorityReady,
		AuthorityBlockedGates: authorityBlocked,
		AuthorityGates:        authorityGates,
		AuthorityMatrix:       forgeKAuthorityGateMatrixReadiness(ready),
		Gates: []ForgeKActivationReadinessGate{
			{Name: "live_owner_explicit", Passed: true, Reason: "live owner remains aios.controllane"},
			{Name: "validation_actions_registered", Passed: registered, Reason: "required validation actions are registered in the live Control Lane registry"},
			{Name: "validation_actions_non_mutating", Passed: nonMutating, Reason: "validation actions are read-only and commit no semantic object"},
			{Name: "validation_actions_no_approval_controls", Passed: noApprovals, Reason: "readiness surface exposes no approve, reject, or execution controls"},
			{Name: "simulator_authority_disabled", Passed: true, Reason: "FORGE-K simulator services are not imported as live authority"},
			{Name: "live_kernel_authority_disabled", Passed: true, Reason: "live Kernel authority migration is still disabled"},
			{Name: "shadow_authoritative_disabled", Passed: true, Reason: "shadow reports remain diagnostic only"},
			{Name: "mutation_controls_absent", Passed: true, Reason: "status surface is read-only"},
		},
		NoEffect:                  forgeKNoEffectSummary(),
		SimulatorAuthority:        false,
		LiveKernelAuthority:       false,
		LiveAuthorityMigration:    false,
		ShadowAuthoritative:       false,
		MutationControlsAvailable: false,
		Notes: []string{
			"partial live validation is running through the existing Control Lane owner",
			"this is not full FORGE-K Kernel live authority",
			"semantic writes, retrieval, gateway execution, modelruntime calls, and evidence admission remain outside this surface",
		},
	}
}

func forgeKAuthorityGateMatrixReadiness(validationReady bool) []ForgeKAuthorityGateMatrixEntry {
	validationStatus := "BLOCKED"
	validationTests := []string{}
	kernelStatus := "BLOCKED"
	kernelTests := []string{}
	kernelBlockers := []string{"required live validation actions are not all closed"}
	if validationReady {
		validationStatus = "KV_REUSE_CANARY_VALIDATION_ONLY"
		validationTests = []string{"Control Lane validation action registry tests", "kernel status read-only activation tests", "exact-identity KV reuse canary tests"}
		kernelStatus = "STATE_AND_LOOP_COMMIT_LIVE"
		kernelTests = []string{"Control Lane validation action registry tests", "kernel status read-only activation tests", "low-risk note kernel-style commit test", "state and loop kernel-style commit test"}
		kernelBlockers = []string{"FORGE-K Kernel simulator is not live authority", "links/tags/operator facades remain future bounded phases", "broader object families remain future bounded phases"}
	}

	return []ForgeKAuthorityGateMatrixEntry{
		{
			Subsystem:       "Kernel",
			CurrentStatus:   kernelStatus,
			LiveOwner:       ForgeKActivationOwnerControlLane,
			TargetOwner:     "forgek.kernel",
			FeatureFlag:     "n/a; CREATE_NOTE, UPDATE_STATE, OPEN_LOOP, and CLOSE_LOOP commit through existing Control Lane syscall transaction path",
			RollbackPath:    "keep existing Control Lane commit path or revert Phase 11/12 docs/tests/readiness metadata",
			TestsRequired:   []string{"Control Lane validation registry tests", "read-only kernel status API tests", "low-risk note commit tests", "state and loop commit tests", "forbidden simulator import tests"},
			TestsPassing:    kernelTests,
			Blockers:        kernelBlockers,
			OperatorVisible: true,
		},
		{
			Subsystem:       "Courthouse",
			CurrentStatus:   "DETERMINISTIC_ADMISSION_RULING_PARTIAL",
			LiveOwner:       "services/core/internal/forgekernel/court with the temporary aios/controllane SQLite durable adapter",
			TargetOwner:     "forgek.court",
			FeatureFlag:     "boot-selected forge_k authority; no separate Court mutation flag",
			RollbackPath:    "legacy_v1 cannot execute Court actions; retain immutable Court history and fail closed",
			TestsRequired:   []string{"deterministic admission and appeal tests", "atomic Court journal/provenance/receipt tests", "legacy/model/adapter denial tests"},
			TestsPassing:    []string{"K20C production Court integration tests", "K20D commit-proof and rollback tests", "Court scope/history query tests"},
			Blockers:        []string{"public governed Court ingress and whole-store recovery remain incomplete", "simulator Court remains non-authoritative"},
			OperatorVisible: true,
		},
		{
			Subsystem:     "Memory Palace",
			CurrentStatus: "ADMITTED_EVIDENCE_MATERIALIZATION_PARTIAL",
			LiveOwner:     "services/core/internal/forgekernel with the temporary aios/controllane SQLite durable adapter",
			TargetOwner:   "forgek.palace",
			FeatureFlag:   "boot-selected forge_k authority; no separate evidence write flag",
			RollbackPath:  "legacy_v1 cannot execute the K20H actions; retain immutable rows as historical evidence",
			TestsRequired: []string{"Court-bound materialization/revision tests", "atomic rollback/replay/concurrency tests", "legacy observation exclusion tests"},
			TestsPassing:  []string{"K20H production Kernel integration tests", "immutable evidence schema tests", "backup inspection tests"},
			Blockers: []string{
				"retrieval/search/embeddings are not FORGE-K-owned live authority",
				"legacy memory observations remain untrusted and excluded",
				"remaining Memory Palace read/compile and whole-store recovery cutovers are incomplete",
			},
			OperatorVisible: true,
		},
		{
			Subsystem:       "Semantic Algebra",
			CurrentStatus:   "DETERMINISTIC_DIFF_AUTHORITY_PARTIAL",
			LiveOwner:       "services/core/internal/forgekernel/semanticdiff with the temporary aios/controllane SQLite durable adapter",
			TargetOwner:     "forgek.semantic",
			FeatureFlag:     "boot-selected forge_k authority; COMPUTE_SEMANTIC_DIFF fails closed under legacy_v1",
			RollbackPath:    "select legacy_v1 to disable semantic diff commits; immutable operation/result/object evidence remains inspectable",
			TestsRequired:   []string{"deterministic semantic diff vectors", "exact admitted-source authority tests", "atomic journal/provenance/receipt tests", "replay and legacy/adapter denial tests"},
			TestsPassing:    []string{"K20I pure semanticdiff tests", "production Kernel semantic diff integration tests", "SQLite immutability and rollback tests"},
			Blockers:        []string{"only semantic.diff.v1 is production-owned; merge/intersect/compress/derive remain uncut", "no narrow authenticated operator ingress exists yet", "temporary Control Lane durable adapter remains"},
			OperatorVisible: true,
		},
		{
			Subsystem:       "Snapshots",
			CurrentStatus:   "SIMULATOR_ONLY",
			LiveOwner:       "services/core/internal/backup and existing snapshot/restore paths",
			TargetOwner:     "forgek.snapshots",
			FeatureFlag:     "not implemented",
			RollbackPath:    "keep backup/restore on existing live services",
			TestsRequired:   []string{"restore non-execution tests", "snapshot shape-not-truth tests", "rollback proof tests"},
			TestsPassing:    []string{},
			Blockers:        []string{"FORGE-K snapshots are not live truth or restore authority"},
			OperatorVisible: true,
		},
		{
			Subsystem:       "Context Compiler",
			CurrentStatus:   "CONTEXT_ATTRIBUTION_VALIDATION_ONLY",
			LiveOwner:       ForgeKActivationOwnerControlLane + " plus services/core/internal/forgekshadow and legacy COMPILE_CONTEXT paths",
			TargetOwner:     "forgek.contextcompiler",
			FeatureFlag:     "n/a; VALIDATE_CONTEXT_ATTRIBUTION is validation-only, while FORGE_K_SHADOW_MODE_ENABLED gates shadow observation",
			RollbackPath:    "remove VALIDATE_CONTEXT_ATTRIBUTION and keep live prompt/context assembly on existing paths",
			TestsRequired:   []string{"context attribution validation tests", "read-only context mirror tests", "prompt parity tests", "no modelruntime call tests"},
			TestsPassing:    []string{"context attribution validation tests", "ContextBundle shadow tests", "Control Lane admission-candidate observer tests", "forbidden simulator Context Compiler import tests"},
			Blockers:        []string{"live prompt/context assembly remains outside FORGE-K Context Compiler authority", "context attribution validates refs and reasons only and does not replace COMPILE_CONTEXT"},
			OperatorVisible: true,
		},
		{
			Subsystem:       "KV System",
			CurrentStatus:   validationStatus,
			LiveOwner:       ForgeKActivationOwnerControlLane,
			TargetOwner:     "forgek.kv",
			FeatureFlag:     "n/a; exact-identity canary requires explicit kvReuseCanary + canary_path=control_lane_validation_only",
			RollbackPath:    "remove KV reuse canary acceptance and keep VALIDATE_KV_IDENTITY validation-only without canary reuse",
			TestsRequired:   []string{"KV identity validation tests", "exact final-token canary tests", "no backend KV reuse tests", "cache-not-memory tests"},
			TestsPassing:    validationTests,
			Blockers:        []string{"backend KV reuse and runtime cache reuse remain disabled", "canary is validation-only and does not store or reuse live KV tensors"},
			OperatorVisible: true,
		},
		{
			Subsystem:       "Runtime Boundary",
			CurrentStatus:   "RUNTIME_PROPOSAL_BOUNDARY",
			LiveOwner:       "services/core/internal/modelruntime",
			TargetOwner:     "forgek.runtime",
			FeatureFlag:     "n/a; modelruntime output proposal envelope is always metadata on successful generation",
			RollbackPath:    "remove proposal envelope metadata and keep model execution and lifecycle inside existing modelruntime",
			TestsRequired:   []string{"runtime trace-only identity tests", "no backend behavior change tests", "proposal-only output tests"},
			TestsPassing:    []string{"modelruntime proposal envelope tests", "API bridge proposal preservation tests", "no simulator runtime import scan"},
			Blockers:        []string{"live modelruntime remains outside FORGE-K runtime driver authority", "model output is proposal-only and cannot commit truth"},
			OperatorVisible: true,
		},
		{
			Subsystem:       "Lymphatic Lane",
			CurrentStatus:   "LYMPHATIC_PROPOSAL_ONLY_ONLINE",
			LiveOwner:       "existing dream/autonomy/maintenance paths",
			TargetOwner:     "forgek.lymphatic",
			FeatureFlag:     "n/a; autonomy maintenance dry-run sweeps emit proposal-only cleanup metadata",
			RollbackPath:    "remove proposal-only lymphatic metadata and keep cleanup and maintenance on existing live paths",
			TestsRequired:   []string{"maintenance report mirror tests", "cleanup proposal no-execution tests", "no silent mutation tests"},
			TestsPassing:    []string{"autonomy maintenance dry-run proposal-only tests", "no simulator lymphatic import scan"},
			Blockers:        []string{"FORGE-K Lymphatic Lane does not run live cleanup or maintenance actions", "non-dry-run autonomy maintenance remains existing live autonomy authority, not Lymphatic Lane authority"},
			OperatorVisible: true,
		},
		{
			Subsystem:       "Consensus Mesh",
			CurrentStatus:   "CONSENSUS_GATE_MODEL_RUNTIME_ONLY",
			LiveOwner:       "existing API response, modelruntime proposal metadata, gateway, policy, and approval paths",
			TargetOwner:     "forgek.consensus",
			FeatureFlag:     "n/a; modelruntime-backed assistant final responses carry deterministic consensus gate metadata",
			RollbackPath:    "remove consensus gate metadata/final-response guard and keep live decisions on existing policy, approval, and operator gates",
			TestsRequired:   []string{"proposal-only consensus tests", "no decision authority tests", "approval separation tests"},
			TestsPassing:    []string{"pure consensus gate tests", "modelruntime chat final-response gate test", "no simulator Consensus Mesh import scan"},
			Blockers:        []string{"Consensus Mesh is not live decision authority", "gateway, Ollama, and streaming token surfaces are not fully consensus gated", "consensus accepted is not canonical truth or admitted evidence"},
			OperatorVisible: true,
		},
	}
}

func countForgeKAuthorityGates(gates []ForgeKAuthorityGateReadiness) (ready int, blocked int) {
	for _, gate := range gates {
		switch gate.Status {
		case "ready":
			ready++
		case "blocked":
			blocked++
		default:
			blocked++
		}
	}
	return ready, blocked
}

func forgeKAuthorityGateReadiness(validationReady bool) []ForgeKAuthorityGateReadiness {
	controlLaneStatus := "blocked"
	controlLaneReason := "required live validation actions are not all closed"
	controlLaneNext := "close missing live Control Lane validation actions before considering broader authority migration"
	if validationReady {
		controlLaneStatus = "ready"
		controlLaneReason = "validation-only Control Lane enforcement is connected and non-mutating"
		controlLaneNext = "keep validation-only enforcement in the live Control Lane while later gates are designed"
	}
	courthouseStatus := "blocked"
	courthouseReason := "admission candidate validation is not fully closed"
	courthouseNext := "connect admission candidate validation without importing simulator Courthouse services"
	contextAttributionStatus := "blocked"
	contextAttributionReason := "context attribution validation is not fully closed"
	contextAttributionNext := "connect context attribution validation without importing simulator Context Compiler services"
	if validationReady {
		courthouseStatus = "ready"
		courthouseReason = "admission candidate validation is connected through the live Control Lane and does not admit evidence"
		courthouseNext = "keep admission candidate validation non-authoritative while evidence admission and ruling authority remain disabled"
		contextAttributionStatus = "ready"
		contextAttributionReason = "context attribution validation is connected through the live Control Lane and does not compile prompts"
		contextAttributionNext = "keep attribution validation non-authoritative while live Context Compiler prompt authority remains blocked"
	}

	return []ForgeKAuthorityGateReadiness{
		{
			Name:                     "control_lane_validation_enforcement",
			Status:                   controlLaneStatus,
			LiveOwner:                ForgeKActivationOwnerControlLane,
			RequiredForLiveAuthority: true,
			MutationAuthority:        false,
			Reason:                   controlLaneReason,
			NextStep:                 controlLaneNext,
		},
		{
			Name:                     "source_object_authority_lookup",
			Status:                   controlLaneStatus,
			LiveOwner:                ForgeKActivationOwnerControlLane,
			RequiredForLiveAuthority: true,
			MutationAuthority:        false,
			Reason: func() string {
				if validationReady {
					return "source object authority lookup is connected through the live Control Lane read store and fails closed"
				}
				return "source object authority lookup validation is not fully closed"
			}(),
			NextStep: func() string {
				if validationReady {
					return "keep source-object authority lookup read-only while evidence admission and mutation routing gates are designed"
				}
				return "close source-object authority validation with fail-closed tests and no simulator import"
			}(),
		},
		{
			Name:                     "courthouse_admission_integration",
			Status:                   courthouseStatus,
			LiveOwner:                ForgeKActivationOwnerControlLane,
			RequiredForLiveAuthority: true,
			MutationAuthority:        false,
			Reason:                   courthouseReason,
			NextStep:                 courthouseNext,
		},
		{
			Name:                     "live_context_compiler_authority",
			Status:                   "blocked",
			LiveOwner:                "future.live_authority_owner",
			RequiredForLiveAuthority: true,
			MutationAuthority:        false,
			Reason:                   "live prompt/context assembly remains outside FORGE-K Context Compiler authority",
			NextStep:                 "add a read-only mirror and parity tests before any context compilation authority migration",
		},
		{
			Name:                     "context_attribution_validation",
			Status:                   contextAttributionStatus,
			LiveOwner:                ForgeKActivationOwnerControlLane,
			RequiredForLiveAuthority: true,
			MutationAuthority:        false,
			Reason:                   contextAttributionReason,
			NextStep:                 contextAttributionNext,
		},
		{
			Name:                     "governed_semantic_mutation_routing",
			Status:                   "blocked",
			LiveOwner:                "future.live_authority_owner",
			RequiredForLiveAuthority: true,
			MutationAuthority:        false,
			Reason:                   "semantic operation validation does not execute governed semantic mutations",
			NextStep:                 "design explicit semantic mutation routing through existing syscall, audit, approval, and rollback boundaries",
		},
		{
			Name:                     "runtime_driver_authority_boundary",
			Status:                   "blocked",
			LiveOwner:                "future.live_authority_owner",
			RequiredForLiveAuthority: true,
			MutationAuthority:        false,
			Reason:                   "live modelruntime remains outside FORGE-K runtime driver authority",
			NextStep:                 "add trace-only runtime driver identity capture before any modelruntime authority migration",
		},
	}
}
