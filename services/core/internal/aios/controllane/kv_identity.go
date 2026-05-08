package controllane

import (
	"strings"
	"time"

	"forge/projectforge/services/core/internal/kvidentity"
)

func kvManifestIdentityFromPayload(raw any) kvidentity.ManifestIdentity {
	m, _ := raw.(map[string]any)
	return kvidentity.ManifestIdentity{
		CacheID:            readString(m, "cache_id"),
		CacheMode:          readString(m, "cache_mode"),
		WorkspaceID:        readString(m, "workspace_id"),
		BundleID:           readString(m, "bundle_id"),
		BlockID:            readString(m, "block_id"),
		BundleHash:         readString(m, "bundle_hash"),
		StablePrefixHash:   readString(m, "stable_prefix_hash"),
		VolatileSuffixHash: readString(m, "volatile_suffix_hash"),
		ModelID:            readString(m, "model_id"),
		ModelRevision:      readString(m, "model_revision"),
		TokenizerID:        readString(m, "tokenizer_id"),
		TokenizerRevision:  readString(m, "tokenizer_revision"),
		ChatTemplateHash:   readString(m, "chat_template_hash"),
		PromptLayoutHash:   readString(m, "prompt_layout_hash"),
		PolicySchemaHash:   readString(m, "policy_schema_hash"),
		SyscallSchemaHash:  readString(m, "syscall_schema_hash"),
		TokenInputHash:     readString(m, "token_input_hash"),
		FinalTokenIDsHash:  readString(m, "final_token_ids_hash"),
		RuntimeBackend:     readString(m, "runtime_backend"),
		RuntimeVersion:     readString(m, "runtime_version"),
		AttentionBackend:   readString(m, "attention_backend"),
		RopeConfigHash:     readString(m, "rope_config_hash"),
		KVPrecision:        readString(m, "kv_precision"),
		CacheSalt:          readString(m, "cache_salt"),
		Status:             readString(m, "status"),
	}
}

func kvRequestIdentityFromPayload(raw any) kvidentity.RequestIdentity {
	m, _ := raw.(map[string]any)
	return kvidentity.RequestIdentity{
		RequestID:          readString(m, "request_id"),
		CacheMode:          readString(m, "cache_mode"),
		WorkspaceID:        readString(m, "workspace_id"),
		BundleID:           readString(m, "bundle_id"),
		BlockID:            readString(m, "block_id"),
		BundleHash:         readString(m, "bundle_hash"),
		StablePrefixHash:   readString(m, "stable_prefix_hash"),
		VolatileSuffixHash: readString(m, "volatile_suffix_hash"),
		ModelID:            readString(m, "model_id"),
		ModelRevision:      readString(m, "model_revision"),
		TokenizerID:        readString(m, "tokenizer_id"),
		TokenizerRevision:  readString(m, "tokenizer_revision"),
		ChatTemplateHash:   readString(m, "chat_template_hash"),
		PromptLayoutHash:   readString(m, "prompt_layout_hash"),
		PolicySchemaHash:   readString(m, "policy_schema_hash"),
		SyscallSchemaHash:  readString(m, "syscall_schema_hash"),
		TokenInputHash:     readString(m, "token_input_hash"),
		FinalTokenIDsHash:  readString(m, "final_token_ids_hash"),
		RuntimeBackend:     readString(m, "runtime_backend"),
		RuntimeVersion:     readString(m, "runtime_version"),
		AttentionBackend:   readString(m, "attention_backend"),
		RopeConfigHash:     readString(m, "rope_config_hash"),
		KVPrecision:        readString(m, "kv_precision"),
		CacheSalt:          readString(m, "cache_salt"),
	}
}

func liveKVManifestHitEligible(status string) bool {
	switch strings.TrimSpace(status) {
	case "AVAILABLE", "HIT_RECORDED":
		return true
	default:
		return false
	}
}

func millisToTime(millis int64) time.Time {
	if millis <= 0 {
		return time.Time{}
	}
	return time.UnixMilli(millis).UTC()
}
