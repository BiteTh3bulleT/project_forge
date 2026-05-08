package kv

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
)

func SHA256Text(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func StableJSON(value any) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

func StableHash(value any) string {
	return SHA256Text(StableJSON(value))
}

func NormalizeWhitespace(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func NormalizeRefs(values []string) []string {
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

func CloneStrings(values []string) []string {
	if values == nil {
		return nil
	}
	return append([]string(nil), values...)
}

func CloneMap(values map[string]any) map[string]any {
	if values == nil {
		return nil
	}
	out := make(map[string]any, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func IdentityHash(manifest KVCacheManifest) string {
	return StableHash(map[string]any{
		"cache_mode":           manifest.CacheMode,
		"workspace_id":         manifest.WorkspaceID,
		"case_id":              manifest.CaseID,
		"bundle_id":            manifest.BundleID,
		"block_id":             manifest.BlockID,
		"snapshot_id":          manifest.SnapshotID,
		"restore_seed_id":      manifest.RestoreSeedID,
		"model_id":             manifest.ModelID,
		"model_revision":       manifest.ModelRevision,
		"tokenizer_id":         manifest.TokenizerID,
		"tokenizer_revision":   manifest.TokenizerRevision,
		"chat_template_hash":   manifest.ChatTemplateHash,
		"prompt_layout_hash":   manifest.PromptLayoutHash,
		"policy_schema_hash":   manifest.PolicySchemaHash,
		"syscall_schema_hash":  manifest.SyscallSchemaHash,
		"token_input_hash":     manifest.TokenInputHash,
		"final_token_ids_hash": manifest.FinalTokenIDsHash,
		"runtime_backend":      manifest.RuntimeBackend,
		"runtime_version":      manifest.RuntimeVersion,
		"attention_backend":    manifest.AttentionBackend,
		"rope_config_hash":     manifest.RopeConfigHash,
		"kv_precision":         manifest.KVPrecision,
		"cache_salt":           manifest.CacheSalt,
	})
}
