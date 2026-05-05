package shadowharness

import "fmt"

func DefaultShadowHarnessPolicy() ShadowHarnessPolicy {
	return ShadowHarnessPolicy{
		PolicyID:               "phase-11g-shadow-harness-policy",
		Mode:                   "SHADOW_DESIGN_ONLY",
		ProduceConsensusReport: true,
		ProduceContextReport:   true,
		ProduceRAGReport:       true,
		ProduceRuntimeReport:   true,
		ProduceKVReport:        true,
		ProduceLymphaticReport: true,
		Metadata: map[string]any{
			"phase": "Phase 11G",
			"scope": "SIMULATOR_ONLY / SHADOW_DESIGN_ONLY",
		},
	}
}

func ValidateShadowHarnessPolicy(policy ShadowHarnessPolicy) error {
	if policy.PolicyID == "" || policy.Mode == "" {
		return fmt.Errorf("%w: shadow harness policy", ErrMissingRequiredField)
	}
	if policy.AllowLiveMutation || policy.AllowToolExecution || policy.AllowModelRuntimeCalls ||
		policy.AllowRetrievalExecution || policy.AllowSearchExecution || policy.AllowEmbeddingCalls ||
		policy.AllowMemoryWrites || policy.AllowControllaneMutations || policy.AllowUserVisibleOutput ||
		policy.AllowPublicAPIChanges {
		return fmt.Errorf("%w: shadow harness policy", ErrSideEffectAllowed)
	}
	if hasSecretMetadata(policy.Metadata) {
		return ErrSecretMetadata
	}
	return nil
}
