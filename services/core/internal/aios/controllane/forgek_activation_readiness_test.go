package controllane

import (
	"testing"
	"time"

	"forge/projectforge/services/core/internal/aios/domain"
)

func TestForgeKActivationReadinessReportsClosedValidationSurface(t *testing.T) {
	report := ForgeKActivationReadiness(NewStaticActionRegistry(), time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC))

	if report.Phase != "14L" {
		t.Fatalf("phase=%q, want 14L", report.Phase)
	}
	if report.Status != "partial_live_validation_ready" {
		t.Fatalf("status=%q, want partial_live_validation_ready", report.Status)
	}
	if report.LiveOwner != ForgeKActivationOwnerControlLane {
		t.Fatalf("live owner=%q, want %q", report.LiveOwner, ForgeKActivationOwnerControlLane)
	}
	if report.Mode != ForgeKActivationModePartialLiveEnforcement {
		t.Fatalf("mode=%q, want %q", report.Mode, ForgeKActivationModePartialLiveEnforcement)
	}
	if report.SimulatorAuthority || report.LiveKernelAuthority || report.LiveAuthorityMigration || report.ShadowAuthoritative {
		t.Fatalf("readiness report claimed forbidden authority: %#v", report)
	}
	if report.MutationControlsAvailable {
		t.Fatalf("readiness report exposed mutation controls: %#v", report)
	}
	if report.ClosedValidationLanes != 4 || report.TotalValidationLanes != 4 {
		t.Fatalf("validation lane counts = %d/%d, want 4/4", report.ClosedValidationLanes, report.TotalValidationLanes)
	}
	if report.AuthorityReadyGates != 1 || report.AuthorityBlockedGates != 5 {
		t.Fatalf("authority gate counts = ready %d blocked %d, want 1/5", report.AuthorityReadyGates, report.AuthorityBlockedGates)
	}

	actions := map[domain.SemanticActionType]ForgeKActivationActionReadiness{}
	for _, action := range report.ValidationActions {
		actions[action.Action] = action
	}
	for _, action := range []domain.SemanticActionType{
		domain.ActionValidateKVIdentity,
		domain.ActionValidateRefShape,
		domain.ActionCompareRefShape,
		domain.ActionValidateSemanticOperation,
	} {
		got, ok := actions[action]
		if !ok {
			t.Fatalf("missing readiness action %s in %#v", action, report.ValidationActions)
		}
		if !got.Registered || got.Mutating || got.ApprovalPossible || got.Capability == "" {
			t.Fatalf("unexpected readiness for %s: %#v", action, got)
		}
	}

	for _, gate := range report.Gates {
		if !gate.Passed {
			t.Fatalf("gate %s failed unexpectedly: %#v", gate.Name, gate)
		}
	}
	authorityGates := map[string]ForgeKAuthorityGateReadiness{}
	for _, gate := range report.AuthorityGates {
		authorityGates[gate.Name] = gate
		if gate.MutationAuthority {
			t.Fatalf("readiness gate must not grant mutation authority: %#v", gate)
		}
		if gate.Status == "blocked" && gate.NextStep == "" {
			t.Fatalf("blocked gate lacks next step: %#v", gate)
		}
	}
	if authorityGates["control_lane_validation_enforcement"].Status != "ready" {
		t.Fatalf("control lane validation gate not ready: %#v", authorityGates["control_lane_validation_enforcement"])
	}
	for _, name := range []string{
		"source_object_authority_lookup",
		"courthouse_admission_integration",
		"live_context_compiler_authority",
		"governed_semantic_mutation_routing",
		"runtime_driver_authority_boundary",
	} {
		if authorityGates[name].Status != "blocked" {
			t.Fatalf("authority gate %s status=%q, want blocked", name, authorityGates[name].Status)
		}
		if !authorityGates[name].RequiredForLiveAuthority {
			t.Fatalf("authority gate %s should be required for live authority: %#v", name, authorityGates[name])
		}
	}
	for _, key := range []string{
		"memoryMutation",
		"runtimeMutation",
		"modelRuntimeCall",
		"evidenceAdmission",
		"contextCompilation",
		"gatewayExecution",
		"retrievalExecution",
		"liveAuthorityMigration",
	} {
		if report.NoEffect[key] != false {
			t.Fatalf("no-effect key %s=%#v, want false", key, report.NoEffect[key])
		}
	}
}

func TestForgeKActivationReadinessFailsClosedWhenActionMissing(t *testing.T) {
	report := ForgeKActivationReadiness(emptyActionRegistry{}, time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC))

	if report.Status != "blocked" {
		t.Fatalf("status=%q, want blocked", report.Status)
	}
	if report.ClosedValidationLanes != 0 || report.TotalValidationLanes != 4 {
		t.Fatalf("validation lane counts = %d/%d, want 0/4", report.ClosedValidationLanes, report.TotalValidationLanes)
	}
	if report.AuthorityReadyGates != 0 || report.AuthorityBlockedGates != 6 {
		t.Fatalf("authority gate counts = ready %d blocked %d, want 0/6", report.AuthorityReadyGates, report.AuthorityBlockedGates)
	}
	if report.LiveKernelAuthority || report.LiveAuthorityMigration || report.SimulatorAuthority {
		t.Fatalf("blocked report claimed forbidden authority: %#v", report)
	}
	foundFailedGate := false
	for _, gate := range report.Gates {
		if gate.Name == "validation_actions_registered" && !gate.Passed {
			foundFailedGate = true
		}
	}
	if !foundFailedGate {
		t.Fatalf("missing failed validation_actions_registered gate: %#v", report.Gates)
	}
}

type emptyActionRegistry struct{}

func (emptyActionRegistry) Get(domain.SemanticActionType) (ActionDefinition, bool) {
	return ActionDefinition{}, false
}

func (emptyActionRegistry) List() []ActionDefinition {
	return nil
}
