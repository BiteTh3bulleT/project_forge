package kv

import "time"

const (
	GateModel               = "same_model"
	GateModelRevision       = "same_model_revision"
	GateTokenizer           = "same_tokenizer"
	GateTokenizerRevision   = "same_tokenizer_revision"
	GateChatTemplate        = "same_chat_template"
	GatePromptLayout        = "same_prompt_layout"
	GatePolicySyscallSchema = "same_policy_syscall_schema"
	GateTokenIdentity       = "same_token_identity"
	GateRuntimeAssumptions  = "same_runtime_kv_assumptions"
	GateCacheSalt           = "same_cache_salt"
	GateContextBundle       = "same_context_bundle"
	GateCacheMode           = "same_cache_mode"
	GateManifestAvailable   = "manifest_available"
)

const (
	WarningTokenInputHashPlaceholder        = "token_input_hash_used_as_phase_8_identity_placeholder"
	WarningBackendCompositionalMetadataOnly = "backend_compositional_metadata_only"
)

type ValidationResult struct {
	ResultID         string          `json:"result_id"`
	CandidateCacheID string          `json:"candidate_cache_id"`
	RequestIdentity  KVLookupRequest `json:"request_identity"`
	Passed           bool            `json:"passed"`
	FailedGates      []string        `json:"failed_gates,omitempty"`
	Warnings         []string        `json:"warnings,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
}

func ValidateIdentity(resultID string, manifest KVCacheManifest, request KVLookupRequest, createdAt time.Time) ValidationResult {
	request = NormalizeLookupRequest(request)
	failed := make([]string, 0)
	warnings := make([]string, 0)
	check := func(gate string, passed bool) {
		if !passed {
			failed = appendUnique(failed, gate)
		}
	}
	check(GateManifestAvailable, HitEligibleStatus(manifest.Status))
	check(GateCacheMode, manifest.CacheMode == request.CacheMode)
	check(GateContextBundle, manifest.WorkspaceID == request.WorkspaceID &&
		manifest.BundleID == request.BundleID &&
		manifest.BlockID == request.BlockID &&
		(request.BundleHash == "" || manifest.BundleHash == "" || manifest.BundleHash == request.BundleHash) &&
		(request.StablePrefixHash == "" || manifest.StablePrefixHash == "" || manifest.StablePrefixHash == request.StablePrefixHash) &&
		(request.VolatileSuffixHash == "" || manifest.VolatileSuffixHash == "" || manifest.VolatileSuffixHash == request.VolatileSuffixHash))
	check(GateModel, manifest.ModelID == request.ModelID)
	check(GateModelRevision, manifest.ModelRevision == request.ModelRevision)
	check(GateTokenizer, manifest.TokenizerID == request.TokenizerID)
	check(GateTokenizerRevision, manifest.TokenizerRevision == request.TokenizerRevision)
	check(GateChatTemplate, manifest.ChatTemplateHash == request.ChatTemplateHash)
	check(GatePromptLayout, manifest.PromptLayoutHash == request.PromptLayoutHash)
	check(GatePolicySyscallSchema, manifest.PolicySchemaHash == request.PolicySchemaHash &&
		manifest.SyscallSchemaHash == request.SyscallSchemaHash)
	if manifest.FinalTokenIDsHash != "" && request.FinalTokenIDsHash != "" {
		check(GateTokenIdentity, manifest.FinalTokenIDsHash == request.FinalTokenIDsHash)
	} else {
		check(GateTokenIdentity, manifest.TokenInputHash == request.TokenInputHash)
		warnings = appendUnique(warnings, WarningTokenInputHashPlaceholder)
	}
	check(GateRuntimeAssumptions, manifest.RuntimeBackend == request.RuntimeBackend &&
		manifest.RuntimeVersion == request.RuntimeVersion &&
		manifest.AttentionBackend == request.AttentionBackend &&
		manifest.RopeConfigHash == request.RopeConfigHash &&
		manifest.KVPrecision == request.KVPrecision)
	check(GateCacheSalt, manifest.CacheSalt == request.CacheSalt)
	if manifest.CacheMode == ModeBackendCompositional {
		warnings = appendUnique(warnings, WarningBackendCompositionalMetadataOnly)
	}
	return ValidationResult{
		ResultID:         resultID,
		CandidateCacheID: manifest.CacheID,
		RequestIdentity:  request,
		Passed:           len(failed) == 0,
		FailedGates:      NormalizeRefs(failed),
		Warnings:         NormalizeRefs(warnings),
		CreatedAt:        createdAt,
	}
}

func appendUnique(values []string, value string) []string {
	if value == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
