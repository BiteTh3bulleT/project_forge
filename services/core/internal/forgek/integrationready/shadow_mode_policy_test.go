package integrationready

import "testing"

func TestDefaultShadowModePolicyAllowsDiagnosticsOnly(t *testing.T) {
	policy := DefaultShadowModePolicy()
	if err := ValidateShadowModePolicy(policy); err != nil {
		t.Fatal(err)
	}
	if !policy.ObserveLiveRequest || !policy.MirrorInputs || !policy.ProduceShadowReports {
		t.Fatalf("shadow diagnostics should be enabled: %#v", policy)
	}
	if policy.AllowLiveMutation || policy.AllowToolExecution || policy.AllowModelRuntimeCalls ||
		policy.AllowRetrievalExecution || policy.AllowMemoryWrites || policy.AllowUserVisibleOutput {
		t.Fatalf("shadow mode has forbidden permissions: %#v", policy)
	}
}

func TestShadowModePolicyRejectsLiveSideEffects(t *testing.T) {
	cases := []struct {
		name string
		mut  func(*ShadowModePolicy)
	}{
		{"live mutation", func(p *ShadowModePolicy) { p.AllowLiveMutation = true }},
		{"tool execution", func(p *ShadowModePolicy) { p.AllowToolExecution = true }},
		{"modelruntime call", func(p *ShadowModePolicy) { p.AllowModelRuntimeCalls = true }},
		{"retrieval execution", func(p *ShadowModePolicy) { p.AllowRetrievalExecution = true }},
		{"memory write", func(p *ShadowModePolicy) { p.AllowMemoryWrites = true }},
		{"user-visible output", func(p *ShadowModePolicy) { p.AllowUserVisibleOutput = true }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			policy := DefaultShadowModePolicy()
			tc.mut(&policy)
			if err := ValidateShadowModePolicy(policy); err == nil {
				t.Fatal("expected side-effectful shadow mode policy to be rejected")
			}
		})
	}
}
