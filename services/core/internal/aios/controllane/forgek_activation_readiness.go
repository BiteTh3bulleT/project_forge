package controllane

import (
	"time"

	"forge/projectforge/services/core/internal/aios/domain"
)

const ForgeKActivationReadinessPhase = "14K"

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
		domain.ActionValidateSemanticOperation,
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
