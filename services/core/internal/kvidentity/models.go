package kvidentity

import (
	"sort"
	"strings"
	"time"
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
	GateRequiredFields      = "required_identity_fields_present"
)

const (
	WarningTokenInputHashPlaceholder        = "token_input_hash_used_as_identity_placeholder"
	WarningBackendCompositionalMetadataOnly = "backend_compositional_metadata_only"
)

const (
	CacheModeBackendCompositional = "BACKEND_COMPOSITIONAL"
)

type ManifestIdentity struct {
	CacheID            string `json:"cache_id"`
	CacheMode          string `json:"cache_mode"`
	WorkspaceID        string `json:"workspace_id"`
	BundleID           string `json:"bundle_id"`
	BlockID            string `json:"block_id,omitempty"`
	BundleHash         string `json:"bundle_hash,omitempty"`
	StablePrefixHash   string `json:"stable_prefix_hash,omitempty"`
	VolatileSuffixHash string `json:"volatile_suffix_hash,omitempty"`
	ModelID            string `json:"model_id"`
	ModelRevision      string `json:"model_revision"`
	TokenizerID        string `json:"tokenizer_id"`
	TokenizerRevision  string `json:"tokenizer_revision"`
	ChatTemplateHash   string `json:"chat_template_hash"`
	PromptLayoutHash   string `json:"prompt_layout_hash"`
	PolicySchemaHash   string `json:"policy_schema_hash"`
	SyscallSchemaHash  string `json:"syscall_schema_hash"`
	TokenInputHash     string `json:"token_input_hash"`
	FinalTokenIDsHash  string `json:"final_token_ids_hash,omitempty"`
	RuntimeBackend     string `json:"runtime_backend"`
	RuntimeVersion     string `json:"runtime_version"`
	AttentionBackend   string `json:"attention_backend"`
	RopeConfigHash     string `json:"rope_config_hash"`
	KVPrecision        string `json:"kv_precision"`
	CacheSalt          string `json:"cache_salt"`
	Status             string `json:"status"`
}

type RequestIdentity struct {
	RequestID          string `json:"request_id,omitempty"`
	CacheMode          string `json:"cache_mode"`
	WorkspaceID        string `json:"workspace_id"`
	BundleID           string `json:"bundle_id"`
	BlockID            string `json:"block_id,omitempty"`
	BundleHash         string `json:"bundle_hash,omitempty"`
	StablePrefixHash   string `json:"stable_prefix_hash,omitempty"`
	VolatileSuffixHash string `json:"volatile_suffix_hash,omitempty"`
	ModelID            string `json:"model_id"`
	ModelRevision      string `json:"model_revision"`
	TokenizerID        string `json:"tokenizer_id"`
	TokenizerRevision  string `json:"tokenizer_revision"`
	ChatTemplateHash   string `json:"chat_template_hash"`
	PromptLayoutHash   string `json:"prompt_layout_hash"`
	PolicySchemaHash   string `json:"policy_schema_hash"`
	SyscallSchemaHash  string `json:"syscall_schema_hash"`
	TokenInputHash     string `json:"token_input_hash"`
	FinalTokenIDsHash  string `json:"final_token_ids_hash,omitempty"`
	RuntimeBackend     string `json:"runtime_backend"`
	RuntimeVersion     string `json:"runtime_version"`
	AttentionBackend   string `json:"attention_backend"`
	RopeConfigHash     string `json:"rope_config_hash"`
	KVPrecision        string `json:"kv_precision"`
	CacheSalt          string `json:"cache_salt"`
}

type ValidationResult struct {
	ResultID         string          `json:"result_id"`
	CandidateCacheID string          `json:"candidate_cache_id"`
	RequestIdentity  RequestIdentity `json:"request_identity"`
	Passed           bool            `json:"passed"`
	FailedGates      []string        `json:"failed_gates,omitempty"`
	Warnings         []string        `json:"warnings,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
}

func ValidateIdentity(resultID string, manifest ManifestIdentity, request RequestIdentity, manifestHitEligible bool, createdAt time.Time) ValidationResult {
	manifest = NormalizeManifestIdentity(manifest)
	request = NormalizeRequestIdentity(request)
	failed := make([]string, 0)
	warnings := make([]string, 0)
	check := func(gate string, passed bool) {
		if !passed {
			failed = appendUnique(failed, gate)
		}
	}
	check(GateRequiredFields, requiredManifestFieldsPresent(manifest) && requiredRequestFieldsPresent(request))
	check(GateManifestAvailable, manifestHitEligible)
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
	if manifest.CacheMode == CacheModeBackendCompositional {
		warnings = appendUnique(warnings, WarningBackendCompositionalMetadataOnly)
	}
	return ValidationResult{
		ResultID:         strings.TrimSpace(resultID),
		CandidateCacheID: manifest.CacheID,
		RequestIdentity:  request,
		Passed:           len(failed) == 0,
		FailedGates:      NormalizeStrings(failed),
		Warnings:         NormalizeStrings(warnings),
		CreatedAt:        createdAt,
	}
}

func requiredManifestFieldsPresent(manifest ManifestIdentity) bool {
	return manifest.CacheID != "" &&
		manifest.CacheMode != "" &&
		manifest.WorkspaceID != "" &&
		manifest.BundleID != "" &&
		manifest.ModelID != "" &&
		manifest.ModelRevision != "" &&
		manifest.TokenizerID != "" &&
		manifest.TokenizerRevision != "" &&
		manifest.ChatTemplateHash != "" &&
		manifest.PromptLayoutHash != "" &&
		manifest.PolicySchemaHash != "" &&
		manifest.SyscallSchemaHash != "" &&
		manifest.TokenInputHash != "" &&
		manifest.RuntimeBackend != "" &&
		manifest.RuntimeVersion != "" &&
		manifest.AttentionBackend != "" &&
		manifest.RopeConfigHash != "" &&
		manifest.KVPrecision != "" &&
		manifest.CacheSalt != "" &&
		manifest.Status != ""
}

func requiredRequestFieldsPresent(request RequestIdentity) bool {
	return request.CacheMode != "" &&
		request.WorkspaceID != "" &&
		request.BundleID != "" &&
		request.ModelID != "" &&
		request.ModelRevision != "" &&
		request.TokenizerID != "" &&
		request.TokenizerRevision != "" &&
		request.ChatTemplateHash != "" &&
		request.PromptLayoutHash != "" &&
		request.PolicySchemaHash != "" &&
		request.SyscallSchemaHash != "" &&
		request.TokenInputHash != "" &&
		request.RuntimeBackend != "" &&
		request.RuntimeVersion != "" &&
		request.AttentionBackend != "" &&
		request.RopeConfigHash != "" &&
		request.KVPrecision != "" &&
		request.CacheSalt != ""
}

func NormalizeManifestIdentity(manifest ManifestIdentity) ManifestIdentity {
	manifest.CacheID = strings.TrimSpace(manifest.CacheID)
	manifest.CacheMode = strings.TrimSpace(manifest.CacheMode)
	manifest.WorkspaceID = strings.TrimSpace(manifest.WorkspaceID)
	manifest.BundleID = strings.TrimSpace(manifest.BundleID)
	manifest.BlockID = strings.TrimSpace(manifest.BlockID)
	manifest.BundleHash = strings.TrimSpace(manifest.BundleHash)
	manifest.StablePrefixHash = strings.TrimSpace(manifest.StablePrefixHash)
	manifest.VolatileSuffixHash = strings.TrimSpace(manifest.VolatileSuffixHash)
	manifest.ModelID = strings.TrimSpace(manifest.ModelID)
	manifest.ModelRevision = strings.TrimSpace(manifest.ModelRevision)
	manifest.TokenizerID = strings.TrimSpace(manifest.TokenizerID)
	manifest.TokenizerRevision = strings.TrimSpace(manifest.TokenizerRevision)
	manifest.ChatTemplateHash = strings.TrimSpace(manifest.ChatTemplateHash)
	manifest.PromptLayoutHash = strings.TrimSpace(manifest.PromptLayoutHash)
	manifest.PolicySchemaHash = strings.TrimSpace(manifest.PolicySchemaHash)
	manifest.SyscallSchemaHash = strings.TrimSpace(manifest.SyscallSchemaHash)
	manifest.TokenInputHash = strings.TrimSpace(manifest.TokenInputHash)
	manifest.FinalTokenIDsHash = strings.TrimSpace(manifest.FinalTokenIDsHash)
	manifest.RuntimeBackend = strings.TrimSpace(manifest.RuntimeBackend)
	manifest.RuntimeVersion = strings.TrimSpace(manifest.RuntimeVersion)
	manifest.AttentionBackend = strings.TrimSpace(manifest.AttentionBackend)
	manifest.RopeConfigHash = strings.TrimSpace(manifest.RopeConfigHash)
	manifest.KVPrecision = strings.TrimSpace(manifest.KVPrecision)
	manifest.CacheSalt = strings.TrimSpace(manifest.CacheSalt)
	manifest.Status = strings.TrimSpace(manifest.Status)
	return manifest
}

func NormalizeRequestIdentity(request RequestIdentity) RequestIdentity {
	request.RequestID = strings.TrimSpace(request.RequestID)
	request.CacheMode = strings.TrimSpace(request.CacheMode)
	request.WorkspaceID = strings.TrimSpace(request.WorkspaceID)
	request.BundleID = strings.TrimSpace(request.BundleID)
	request.BlockID = strings.TrimSpace(request.BlockID)
	request.BundleHash = strings.TrimSpace(request.BundleHash)
	request.StablePrefixHash = strings.TrimSpace(request.StablePrefixHash)
	request.VolatileSuffixHash = strings.TrimSpace(request.VolatileSuffixHash)
	request.ModelID = strings.TrimSpace(request.ModelID)
	request.ModelRevision = strings.TrimSpace(request.ModelRevision)
	request.TokenizerID = strings.TrimSpace(request.TokenizerID)
	request.TokenizerRevision = strings.TrimSpace(request.TokenizerRevision)
	request.ChatTemplateHash = strings.TrimSpace(request.ChatTemplateHash)
	request.PromptLayoutHash = strings.TrimSpace(request.PromptLayoutHash)
	request.PolicySchemaHash = strings.TrimSpace(request.PolicySchemaHash)
	request.SyscallSchemaHash = strings.TrimSpace(request.SyscallSchemaHash)
	request.TokenInputHash = strings.TrimSpace(request.TokenInputHash)
	request.FinalTokenIDsHash = strings.TrimSpace(request.FinalTokenIDsHash)
	request.RuntimeBackend = strings.TrimSpace(request.RuntimeBackend)
	request.RuntimeVersion = strings.TrimSpace(request.RuntimeVersion)
	request.AttentionBackend = strings.TrimSpace(request.AttentionBackend)
	request.RopeConfigHash = strings.TrimSpace(request.RopeConfigHash)
	request.KVPrecision = strings.TrimSpace(request.KVPrecision)
	request.CacheSalt = strings.TrimSpace(request.CacheSalt)
	return request
}

func NormalizeStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
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
