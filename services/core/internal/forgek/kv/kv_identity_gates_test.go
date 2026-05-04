package kv

import "testing"

func TestNineGateValidationPassesAndUsesTokenInputPlaceholder(t *testing.T) {
	manifest, err := NewManifest(validManifestInput())
	if err != nil {
		t.Fatalf("manifest: %v", err)
	}
	result := ValidateIdentity("result-1", manifest, validLookupRequest(), testKVTime())
	if !result.Passed || len(result.FailedGates) != 0 {
		t.Fatalf("expected all gates to pass: %#v", result)
	}
	if !contains(result.Warnings, WarningTokenInputHashPlaceholder) {
		t.Fatalf("expected token input placeholder warning: %#v", result.Warnings)
	}
}

func TestNineGateValidationRecordsEachFailure(t *testing.T) {
	manifest, err := NewManifest(validManifestInput())
	if err != nil {
		t.Fatalf("manifest: %v", err)
	}
	cases := []struct {
		name string
		edit func(*KVLookupRequest)
		gate string
	}{
		{"model", func(r *KVLookupRequest) { r.ModelID = "other" }, GateModel},
		{"model revision", func(r *KVLookupRequest) { r.ModelRevision = "other" }, GateModelRevision},
		{"tokenizer", func(r *KVLookupRequest) { r.TokenizerID = "other" }, GateTokenizer},
		{"tokenizer revision", func(r *KVLookupRequest) { r.TokenizerRevision = "other" }, GateTokenizerRevision},
		{"chat template", func(r *KVLookupRequest) { r.ChatTemplateHash = "other" }, GateChatTemplate},
		{"prompt layout", func(r *KVLookupRequest) { r.PromptLayoutHash = "other" }, GatePromptLayout},
		{"policy", func(r *KVLookupRequest) { r.PolicySchemaHash = "other" }, GatePolicySyscallSchema},
		{"syscall", func(r *KVLookupRequest) { r.SyscallSchemaHash = "other" }, GatePolicySyscallSchema},
		{"token input", func(r *KVLookupRequest) { r.TokenInputHash = "other" }, GateTokenIdentity},
		{"runtime backend", func(r *KVLookupRequest) { r.RuntimeBackend = "other" }, GateRuntimeAssumptions},
		{"runtime version", func(r *KVLookupRequest) { r.RuntimeVersion = "other" }, GateRuntimeAssumptions},
		{"attention", func(r *KVLookupRequest) { r.AttentionBackend = "other" }, GateRuntimeAssumptions},
		{"rope", func(r *KVLookupRequest) { r.RopeConfigHash = "other" }, GateRuntimeAssumptions},
		{"precision", func(r *KVLookupRequest) { r.KVPrecision = "int8" }, GateRuntimeAssumptions},
		{"salt", func(r *KVLookupRequest) { r.CacheSalt = "other" }, GateCacheSalt},
	}
	for _, tc := range cases {
		request := validLookupRequest()
		tc.edit(&request)
		result := ValidateIdentity("result-"+tc.name, manifest, request, testKVTime())
		if result.Passed || !contains(result.FailedGates, tc.gate) {
			t.Fatalf("%s: expected gate %s failure, got %#v", tc.name, tc.gate, result)
		}
	}
}

func TestNineGateValidationRejectsUnavailableManifest(t *testing.T) {
	input := validManifestInput()
	input.Status = StatusInvalidated
	manifest, err := NewManifest(input)
	if err != nil {
		t.Fatalf("manifest: %v", err)
	}
	result := ValidateIdentity("result-1", manifest, validLookupRequest(), testKVTime())
	if result.Passed || !contains(result.FailedGates, GateManifestAvailable) {
		t.Fatalf("expected manifest availability failure: %#v", result)
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
