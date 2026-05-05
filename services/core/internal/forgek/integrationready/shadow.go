package integrationready

import "fmt"

func DefaultShadowModePolicy() ShadowModePolicy {
	return ShadowModePolicy{
		PolicyID:             "phase-11f-shadow-mode-policy",
		Mode:                 "DIAGNOSTIC_ONLY",
		ObserveLiveRequest:   true,
		MirrorInputs:         true,
		MirrorEvidence:       true,
		MirrorRetrieval:      true,
		ProduceShadowReports: true,
		CompareContext:       true,
		CompareConsensus:     true,
		CompareResponse:      true,
		Metadata: map[string]any{
			"phase": Phase11F,
			"scope": "SIMULATOR_ONLY / INTEGRATION_PREP_ONLY",
		},
	}
}

func ValidateShadowModePolicy(policy ShadowModePolicy) error {
	if policy.PolicyID == "" || policy.Mode == "" {
		return fmt.Errorf("%w: shadow mode policy", ErrMissingRequiredField)
	}
	if policy.AllowLiveMutation || policy.AllowToolExecution || policy.AllowModelRuntimeCalls ||
		policy.AllowRetrievalExecution || policy.AllowMemoryWrites || policy.AllowUserVisibleOutput {
		return fmt.Errorf("%w: shadow mode policy", ErrInvalidShadowPolicy)
	}
	return nil
}
