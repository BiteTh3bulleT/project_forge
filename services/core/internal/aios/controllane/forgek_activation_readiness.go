package controllane

import (
	"time"

	"forge/projectforge/services/core/internal/aios/domain"
)

const ForgeKActivationReadinessPhase = "14M"

type ForgeKActivationReadinessReport struct {
	GeneratedAt               time.Time                         `json:"generated_at"`
	Phase                     string                            `json:"phase"`
	Status                    string                            `json:"status"`
	Summary                   string                            `json:"summary"`
	Mode                      string                            `json:"mode"`
	LiveOwner                 string                            `json:"live_owner"`
	PolicyVersion             string                            `json:"policy_version"`
	KernelRuntimeState        string                            `json:"kernel_runtime_state"`
	ClosedValidationLanes     int                               `json:"closed_validation_lanes"`
	TotalValidationLanes      int                               `json:"total_validation_lanes"`
	ValidationActions         []ForgeKActivationActionReadiness `json:"validation_actions"`
	AuthorityReadyGates       int                               `json:"authority_ready_gates"`
	AuthorityBlockedGates     int                               `json:"authority_blocked_gates"`
	AuthorityGates            []ForgeKAuthorityGateReadiness    `json:"authority_gates"`
	AuthorityMatrix           []ForgeKAuthorityGateMatrixEntry  `json:"authority_matrix"`
	Gates                     []ForgeKActivationReadinessGate   `json:"gates"`
	NoEffect                  map[string]any                    `json:"no_effect"`
	SimulatorAuthority        bool                              `json:"simulator_authority"`
	LiveKernelAuthority       bool                              `json:"live_kernel_authority"`
	LiveAuthorityMigration    bool                              `json:"live_authority_migration"`
	ShadowAuthoritative       bool                              `json:"shadow_authoritative"`
	MutationControlsAvailable bool                              `json:"mutation_controls_available"`
	Notes                     []string                          `json:"notes"`
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
	validationBlockers := []string{"required live validation actions are not all closed"}
	if validationReady {
		validationStatus = "PARTIAL_LIVE_VALIDATION"
		validationTests = []string{"Control Lane validation action registry tests", "kernel status read-only activation tests"}
		validationBlockers = []string{"full Kernel mutation authority remains disabled"}
	}

	return []ForgeKAuthorityGateMatrixEntry{
		{
			Subsystem:       "Kernel",
			CurrentStatus:   validationStatus,
			LiveOwner:       ForgeKActivationOwnerControlLane,
			TargetOwner:     "forgek.kernel",
			FeatureFlag:     "n/a; live surface is validation-only Control Lane metadata",
			RollbackPath:    "remove validation readiness exposure and keep existing Control Lane behavior",
			TestsRequired:   []string{"Control Lane validation registry tests", "read-only kernel status API tests", "forbidden simulator import tests"},
			TestsPassing:    validationTests,
			Blockers:        validationBlockers,
			OperatorVisible: true,
		},
		{
			Subsystem: "Courthouse",
			CurrentStatus: func() string {
				if validationReady {
					return "ADMISSION_CANDIDATE_ONLY"
				}
				return "BLOCKED"
			}(),
			LiveOwner:       ForgeKActivationOwnerControlLane,
			TargetOwner:     "forgek.court",
			FeatureFlag:     "n/a; admission candidate validation only",
			RollbackPath:    "remove VALIDATE_ADMISSION_CANDIDATE and leave live evidence handling on existing non-FORGE-K paths",
			TestsRequired:   []string{"admission candidate validation tests", "no evidence admission tests", "no simulator service live import tests"},
			TestsPassing:    validationTests,
			Blockers:        []string{"live evidence admission and ruling authority remain disabled"},
			OperatorVisible: true,
		},
		{
			Subsystem:     "Memory Palace",
			CurrentStatus: "MEMORY_PALACE_MIRROR_ONLY",
			LiveOwner:     "services/core/internal/forgekshadow plus existing memory/retrieval owners",
			TargetOwner:   "forgek.palace",
			FeatureFlag:   "FORGE_K_SHADOW_MODE_ENABLED + FORGE_K_SHADOW_RETRIEVAL_METADATA_ENABLED for persisted observation",
			RollbackPath:  "remove memory palace mirror projection and keep memory/retrieval on existing live stores",
			TestsRequired: []string{"memory mirror read-only tests", "retrieval no-execution tests", "provenance preservation tests"},
			TestsPassing:  []string{"Memory Palace mirror tests", "retrieval metadata shadow observer tests", "forbidden simulator Palace import tests"},
			Blockers: []string{
				"live memory writes remain outside FORGE-K",
				"retrieval/search/embeddings are not FORGE-K-owned live authority",
				"Memory Palace mirror is diagnostic metadata only and does not admit evidence",
			},
			OperatorVisible: true,
		},
		{
			Subsystem:       "Semantic Algebra",
			CurrentStatus:   validationStatus,
			LiveOwner:       ForgeKActivationOwnerControlLane,
			TargetOwner:     "forgek.semantic",
			FeatureFlag:     "n/a; validation-only semantic operation shape checks",
			RollbackPath:    "remove semantic operation validation action and keep existing Control Lane mutation paths",
			TestsRequired:   []string{"semantic operation validation tests", "forbidden authority claim tests", "no semantic commit tests"},
			TestsPassing:    validationTests,
			Blockers:        []string{"semantic algebra does not execute live semantic operations"},
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
			CurrentStatus:   "BLOCKED",
			LiveOwner:       "services/core/internal/aios/controllane legacy COMPILE_CONTEXT paths",
			TargetOwner:     "forgek.contextcompiler",
			FeatureFlag:     "not implemented",
			RollbackPath:    "keep live prompt/context assembly on existing paths",
			TestsRequired:   []string{"read-only context mirror tests", "prompt parity tests", "no modelruntime call tests"},
			TestsPassing:    []string{},
			Blockers:        []string{"live prompt/context assembly remains outside FORGE-K Context Compiler authority"},
			OperatorVisible: true,
		},
		{
			Subsystem:       "KV System",
			CurrentStatus:   validationStatus,
			LiveOwner:       ForgeKActivationOwnerControlLane,
			TargetOwner:     "forgek.kv",
			FeatureFlag:     "n/a; KV identity validation only",
			RollbackPath:    "remove VALIDATE_KV_IDENTITY live validation and keep no live KV reuse",
			TestsRequired:   []string{"KV identity validation tests", "no live KV reuse tests", "cache-not-memory tests"},
			TestsPassing:    validationTests,
			Blockers:        []string{"live KV reuse and runtime cache reuse remain disabled"},
			OperatorVisible: true,
		},
		{
			Subsystem:       "Runtime Boundary",
			CurrentStatus:   "BLOCKED",
			LiveOwner:       "services/core/internal/modelruntime",
			TargetOwner:     "forgek.runtime",
			FeatureFlag:     "not implemented",
			RollbackPath:    "keep model execution and lifecycle inside existing modelruntime",
			TestsRequired:   []string{"runtime trace-only identity tests", "no backend behavior change tests", "proposal-only output tests"},
			TestsPassing:    []string{},
			Blockers:        []string{"live modelruntime remains outside FORGE-K runtime driver authority"},
			OperatorVisible: true,
		},
		{
			Subsystem:       "Lymphatic Lane",
			CurrentStatus:   "SIMULATOR_ONLY",
			LiveOwner:       "existing dream/autonomy/maintenance paths",
			TargetOwner:     "forgek.lymphatic",
			FeatureFlag:     "not implemented",
			RollbackPath:    "keep cleanup and maintenance on existing live paths",
			TestsRequired:   []string{"maintenance report mirror tests", "cleanup proposal no-execution tests", "no silent mutation tests"},
			TestsPassing:    []string{},
			Blockers:        []string{"FORGE-K Lymphatic Lane does not run live cleanup or maintenance actions"},
			OperatorVisible: true,
		},
		{
			Subsystem:       "Consensus Mesh",
			CurrentStatus:   "SIMULATOR_ONLY",
			LiveOwner:       "existing response/policy/approval paths",
			TargetOwner:     "forgek.consensus",
			FeatureFlag:     "not implemented",
			RollbackPath:    "keep live decisions on existing policy, approval, and operator gates",
			TestsRequired:   []string{"proposal-only consensus tests", "no decision authority tests", "approval separation tests"},
			TestsPassing:    []string{},
			Blockers:        []string{"Consensus Mesh is not live decision authority"},
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
	if validationReady {
		courthouseStatus = "ready"
		courthouseReason = "admission candidate validation is connected through the live Control Lane and does not admit evidence"
		courthouseNext = "keep admission candidate validation non-authoritative while evidence admission and ruling authority remain disabled"
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
