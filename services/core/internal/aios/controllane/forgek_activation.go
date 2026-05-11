package controllane

const (
	ForgeKActivationModePartialLiveEnforcement = "partial-live-enforcement"
	ForgeKActivationOwnerControlLane           = "aios.controllane"
	ForgeKActivationPolicyVersion              = "phase-14f-control-lane-enforcement-v1"
)

func forgeKActivationSummary(action string) map[string]any {
	return map[string]any{
		"mode":                ForgeKActivationModePartialLiveEnforcement,
		"liveOwner":           ForgeKActivationOwnerControlLane,
		"action":              action,
		"policyVersion":       ForgeKActivationPolicyVersion,
		"simulatorAuthority":  false,
		"liveKernelAuthority": false,
		"shadowAuthoritative": false,
	}
}

func forgeKNoEffectSummary() map[string]any {
	return map[string]any{
		"memoryMutation":         false,
		"runtimeMutation":        false,
		"modelRuntimeCall":       false,
		"evidenceAdmission":      false,
		"contextCompilation":     false,
		"gatewayExecution":       false,
		"retrievalExecution":     false,
		"liveAuthorityMigration": false,
	}
}
