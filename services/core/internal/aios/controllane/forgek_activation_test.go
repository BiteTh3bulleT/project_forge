package controllane

import "testing"

func TestForgeKActivationSummaryMarksPartialLiveEnforcement(t *testing.T) {
	summary := forgeKActivationSummary("VALIDATE_REF_SHAPE")
	if summary["mode"] != ForgeKActivationModePartialLiveEnforcement {
		t.Fatalf("mode=%#v, want %q", summary["mode"], ForgeKActivationModePartialLiveEnforcement)
	}
	if summary["liveOwner"] != "aios.controllane" {
		t.Fatalf("live owner=%#v, want aios.controllane", summary["liveOwner"])
	}
	if summary["simulatorAuthority"] != false || summary["liveKernelAuthority"] != false {
		t.Fatalf("activation summary claimed simulator/live kernel authority: %#v", summary)
	}
}

func TestForgeKNoEffectSummaryRejectsAuthorityFlags(t *testing.T) {
	summary := forgeKNoEffectSummary()
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
		if summary[key] != false {
			t.Fatalf("%s=%#v, want false in %#v", key, summary[key], summary)
		}
	}
}
