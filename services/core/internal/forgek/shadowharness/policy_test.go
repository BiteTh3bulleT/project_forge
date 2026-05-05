package shadowharness

import "testing"

func TestDefaultShadowHarnessPolicyForbidsAllSideEffects(t *testing.T) {
	policy := DefaultShadowHarnessPolicy()
	if err := ValidateShadowHarnessPolicy(policy); err != nil {
		t.Fatal(err)
	}
	if policy.AllowLiveMutation || policy.AllowToolExecution || policy.AllowModelRuntimeCalls ||
		policy.AllowRetrievalExecution || policy.AllowSearchExecution || policy.AllowEmbeddingCalls || policy.AllowMemoryWrites ||
		policy.AllowControllaneMutations ||
		policy.AllowUserVisibleOutput || policy.AllowPublicAPIChanges {
		t.Fatalf("default policy has side effects enabled: %#v", policy)
	}
	if !policy.ProduceConsensusReport || !policy.ProduceContextReport || !policy.ProduceRAGReport ||
		!policy.ProduceRuntimeReport || !policy.ProduceKVReport || !policy.ProduceLymphaticReport {
		t.Fatalf("default policy should enable diagnostic reporting: %#v", policy)
	}
}

func TestShadowHarnessPolicyRejectsSideEffects(t *testing.T) {
	cases := []struct {
		name string
		mut  func(*ShadowHarnessPolicy)
	}{
		{"live mutation", func(p *ShadowHarnessPolicy) { p.AllowLiveMutation = true }},
		{"tool execution", func(p *ShadowHarnessPolicy) { p.AllowToolExecution = true }},
		{"modelruntime call", func(p *ShadowHarnessPolicy) { p.AllowModelRuntimeCalls = true }},
		{"retrieval execution", func(p *ShadowHarnessPolicy) { p.AllowRetrievalExecution = true }},
		{"search execution", func(p *ShadowHarnessPolicy) { p.AllowSearchExecution = true }},
		{"embedding call", func(p *ShadowHarnessPolicy) { p.AllowEmbeddingCalls = true }},
		{"memory write", func(p *ShadowHarnessPolicy) { p.AllowMemoryWrites = true }},
		{"controllane mutation", func(p *ShadowHarnessPolicy) { p.AllowControllaneMutations = true }},
		{"user-visible output", func(p *ShadowHarnessPolicy) { p.AllowUserVisibleOutput = true }},
		{"public API change", func(p *ShadowHarnessPolicy) { p.AllowPublicAPIChanges = true }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			policy := DefaultShadowHarnessPolicy()
			tc.mut(&policy)
			if err := ValidateShadowHarnessPolicy(policy); err == nil {
				t.Fatal("expected side-effectful policy to be rejected")
			}
		})
	}
}
