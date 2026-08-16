package controllane

import (
	"testing"
	"time"

	"forge/projectforge/services/core/internal/aios/domain"
)

func TestForgeKActivationReadinessReportsClosedValidationSurface(t *testing.T) {
	report := ForgeKActivationReadiness(NewStaticActionRegistry(), time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC))

	if report.Phase != "19" {
		t.Fatalf("phase=%q, want 19", report.Phase)
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
	if report.ClosedValidationLanes != 7 || report.TotalValidationLanes != 7 {
		t.Fatalf("validation lane counts = %d/%d, want 7/7", report.ClosedValidationLanes, report.TotalValidationLanes)
	}
	if report.AuthorityReadyGates != 4 || report.AuthorityBlockedGates != 3 {
		t.Fatalf("authority gate counts = ready %d blocked %d, want 4/3", report.AuthorityReadyGates, report.AuthorityBlockedGates)
	}
	if len(report.AuthorityMatrix) != 10 {
		t.Fatalf("authority matrix entries=%d, want 10: %#v", len(report.AuthorityMatrix), report.AuthorityMatrix)
	}

	actions := map[domain.SemanticActionType]ForgeKActivationActionReadiness{}
	for _, action := range report.ValidationActions {
		actions[action.Action] = action
	}
	for _, action := range []domain.SemanticActionType{
		domain.ActionValidateKVIdentity,
		domain.ActionValidateRefShape,
		domain.ActionCompareRefShape,
		domain.ActionValidateSourceObject,
		domain.ActionValidateSemanticOperation,
		domain.ActionValidateAdmissionCandidate,
		domain.ActionValidateContextAttribution,
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
	if authorityGates["source_object_authority_lookup"].Status != "ready" {
		t.Fatalf("source object authority gate not ready: %#v", authorityGates["source_object_authority_lookup"])
	}
	if authorityGates["courthouse_admission_integration"].Status != "ready" {
		t.Fatalf("courthouse admission candidate gate not ready: %#v", authorityGates["courthouse_admission_integration"])
	}
	if authorityGates["context_attribution_validation"].Status != "ready" {
		t.Fatalf("context attribution validation gate not ready: %#v", authorityGates["context_attribution_validation"])
	}
	for _, name := range []string{
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

	matrix := map[string]ForgeKAuthorityGateMatrixEntry{}
	for _, entry := range report.AuthorityMatrix {
		matrix[entry.Subsystem] = entry
		if !entry.OperatorVisible {
			t.Fatalf("authority matrix entry is not operator visible: %#v", entry)
		}
		if entry.LiveOwner == "" || entry.TargetOwner == "" || entry.RollbackPath == "" {
			t.Fatalf("authority matrix entry missing required ownership/rollback fields: %#v", entry)
		}
		if len(entry.TestsRequired) == 0 {
			t.Fatalf("authority matrix entry missing tests_required: %#v", entry)
		}
	}
	for _, subsystem := range []string{
		"Kernel",
		"Courthouse",
		"Memory Palace",
		"Semantic Algebra",
		"Snapshots",
		"Context Compiler",
		"KV System",
		"Runtime Boundary",
		"Lymphatic Lane",
		"Consensus Mesh",
	} {
		if _, ok := matrix[subsystem]; !ok {
			t.Fatalf("missing authority matrix subsystem %q in %#v", subsystem, report.AuthorityMatrix)
		}
	}
	if matrix["Kernel"].CurrentStatus != "STATE_AND_LOOP_COMMIT_LIVE" {
		t.Fatalf("kernel matrix status=%q, want STATE_AND_LOOP_COMMIT_LIVE", matrix["Kernel"].CurrentStatus)
	}
	if matrix["KV System"].CurrentStatus != "KV_REUSE_CANARY_VALIDATION_ONLY" {
		t.Fatalf("kv matrix status=%q, want KV_REUSE_CANARY_VALIDATION_ONLY", matrix["KV System"].CurrentStatus)
	}
	if matrix["Courthouse"].CurrentStatus != "DETERMINISTIC_ADMISSION_RULING_PARTIAL" {
		t.Fatalf("courthouse matrix status=%q, want DETERMINISTIC_ADMISSION_RULING_PARTIAL", matrix["Courthouse"].CurrentStatus)
	}
	if matrix["Memory Palace"].CurrentStatus != "ADMITTED_EVIDENCE_MATERIALIZATION_PARTIAL" {
		t.Fatalf("memory palace matrix status=%q, want ADMITTED_EVIDENCE_MATERIALIZATION_PARTIAL", matrix["Memory Palace"].CurrentStatus)
	}
	if matrix["Semantic Algebra"].CurrentStatus != "DETERMINISTIC_DIFF_AUTHORITY_PARTIAL" {
		t.Fatalf("semantic algebra matrix status=%q, want DETERMINISTIC_DIFF_AUTHORITY_PARTIAL", matrix["Semantic Algebra"].CurrentStatus)
	}
	if matrix["Context Compiler"].CurrentStatus != "FORGE_K_INGRESS_ONLY_ADAPTER_DECISION" {
		t.Fatalf("context compiler matrix status=%q, want FORGE_K_INGRESS_ONLY_ADAPTER_DECISION", matrix["Context Compiler"].CurrentStatus)
	}
	if matrix["Runtime Boundary"].CurrentStatus != "RUNTIME_PROPOSAL_BOUNDARY" {
		t.Fatalf("runtime boundary status=%q, want RUNTIME_PROPOSAL_BOUNDARY", matrix["Runtime Boundary"].CurrentStatus)
	}
	if matrix["Lymphatic Lane"].CurrentStatus != "LYMPHATIC_PROPOSAL_ONLY_ONLINE" {
		t.Fatalf("lymphatic lane status=%q, want LYMPHATIC_PROPOSAL_ONLY_ONLINE", matrix["Lymphatic Lane"].CurrentStatus)
	}
	if matrix["Consensus Mesh"].CurrentStatus != "CONSENSUS_GATE_MODEL_RUNTIME_ONLY" {
		t.Fatalf("consensus mesh status=%q, want CONSENSUS_GATE_MODEL_RUNTIME_ONLY", matrix["Consensus Mesh"].CurrentStatus)
	}
}

func TestForgeKActivationReadinessFailsClosedWhenActionMissing(t *testing.T) {
	report := ForgeKActivationReadiness(emptyActionRegistry{}, time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC))

	if report.Status != "blocked" {
		t.Fatalf("status=%q, want blocked", report.Status)
	}
	if report.ClosedValidationLanes != 0 || report.TotalValidationLanes != 7 {
		t.Fatalf("validation lane counts = %d/%d, want 0/7", report.ClosedValidationLanes, report.TotalValidationLanes)
	}
	if report.AuthorityReadyGates != 0 || report.AuthorityBlockedGates != 7 {
		t.Fatalf("authority gate counts = ready %d blocked %d, want 0/7", report.AuthorityReadyGates, report.AuthorityBlockedGates)
	}
	if len(report.AuthorityMatrix) != 10 {
		t.Fatalf("authority matrix entries=%d, want 10", len(report.AuthorityMatrix))
	}
	for _, entry := range report.AuthorityMatrix {
		if entry.Subsystem == "Kernel" && entry.CurrentStatus != "BLOCKED" {
			t.Fatalf("blocked report kernel matrix status=%q, want BLOCKED", entry.CurrentStatus)
		}
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
