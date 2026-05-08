package kv

import (
	"time"

	"forge/projectforge/services/core/internal/kvidentity"
)

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
	WarningTokenInputHashPlaceholder        = kvidentity.WarningTokenInputHashPlaceholder
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
	shared := kvidentity.ValidateIdentity(resultID, manifestIdentity(manifest), requestIdentity(request), HitEligibleStatus(manifest.Status), createdAt)
	return ValidationResult{
		ResultID:         shared.ResultID,
		CandidateCacheID: shared.CandidateCacheID,
		RequestIdentity:  request,
		Passed:           shared.Passed,
		FailedGates:      NormalizeRefs(shared.FailedGates),
		Warnings:         NormalizeRefs(shared.Warnings),
		CreatedAt:        createdAt,
	}
}

func manifestIdentity(manifest KVCacheManifest) kvidentity.ManifestIdentity {
	return kvidentity.ManifestIdentity{
		CacheID:            manifest.CacheID,
		CacheMode:          string(manifest.CacheMode),
		WorkspaceID:        manifest.WorkspaceID,
		BundleID:           manifest.BundleID,
		BlockID:            manifest.BlockID,
		BundleHash:         manifest.BundleHash,
		StablePrefixHash:   manifest.StablePrefixHash,
		VolatileSuffixHash: manifest.VolatileSuffixHash,
		ModelID:            manifest.ModelID,
		ModelRevision:      manifest.ModelRevision,
		TokenizerID:        manifest.TokenizerID,
		TokenizerRevision:  manifest.TokenizerRevision,
		ChatTemplateHash:   manifest.ChatTemplateHash,
		PromptLayoutHash:   manifest.PromptLayoutHash,
		PolicySchemaHash:   manifest.PolicySchemaHash,
		SyscallSchemaHash:  manifest.SyscallSchemaHash,
		TokenInputHash:     manifest.TokenInputHash,
		FinalTokenIDsHash:  manifest.FinalTokenIDsHash,
		RuntimeBackend:     manifest.RuntimeBackend,
		RuntimeVersion:     manifest.RuntimeVersion,
		AttentionBackend:   manifest.AttentionBackend,
		RopeConfigHash:     manifest.RopeConfigHash,
		KVPrecision:        manifest.KVPrecision,
		CacheSalt:          manifest.CacheSalt,
		Status:             string(manifest.Status),
	}
}

func requestIdentity(request KVLookupRequest) kvidentity.RequestIdentity {
	return kvidentity.RequestIdentity{
		RequestID:          request.RequestID,
		CacheMode:          string(request.CacheMode),
		WorkspaceID:        request.WorkspaceID,
		BundleID:           request.BundleID,
		BlockID:            request.BlockID,
		BundleHash:         request.BundleHash,
		StablePrefixHash:   request.StablePrefixHash,
		VolatileSuffixHash: request.VolatileSuffixHash,
		ModelID:            request.ModelID,
		ModelRevision:      request.ModelRevision,
		TokenizerID:        request.TokenizerID,
		TokenizerRevision:  request.TokenizerRevision,
		ChatTemplateHash:   request.ChatTemplateHash,
		PromptLayoutHash:   request.PromptLayoutHash,
		PolicySchemaHash:   request.PolicySchemaHash,
		SyscallSchemaHash:  request.SyscallSchemaHash,
		TokenInputHash:     request.TokenInputHash,
		FinalTokenIDsHash:  request.FinalTokenIDsHash,
		RuntimeBackend:     request.RuntimeBackend,
		RuntimeVersion:     request.RuntimeVersion,
		AttentionBackend:   request.AttentionBackend,
		RopeConfigHash:     request.RopeConfigHash,
		KVPrecision:        request.KVPrecision,
		CacheSalt:          request.CacheSalt,
	}
}
