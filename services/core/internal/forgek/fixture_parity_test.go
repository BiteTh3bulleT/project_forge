package forgek

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const forgekFixtureRoot = "../../../../fixtures/forgek"

func TestForgeKFixtureParityWithRustGoldenHashes(t *testing.T) {
	golden := readGoldenFixtureHashes(t)

	hashFixtureNames := map[string]string{
		"context_block_hash":        "context_block.valid.json",
		"kv_manifest_identity_hash": "kv_cache_manifest.valid.json",
		"snapshot_shape_hash":       "snapshot_manifest.valid.json",
		"context_bundle_hash":       "context_bundle.valid.json",
		"runtime_manifest_hash":     "runtime_driver_manifest.valid.json",
	}

	for _, hashName := range []string{
		"context_block_hash",
		"context_bundle_hash",
		"kv_manifest_identity_hash",
		"runtime_manifest_hash",
		"snapshot_shape_hash",
	} {
		if _, ok := golden[hashName]; !ok {
			t.Fatalf("golden hash %q is missing from hashes.json", hashName)
		}
	}

	for _, fixtureName := range hashFixtureNames {
		fixturePath := filepath.Join(forgekFixtureRoot, "valid", fixtureName)
		value := readFixtureJSON[any](t, fixturePath)
		validateStableFixtureFields(t, fixtureName, value)
	}

	for hashName, expected := range golden {
		fixtureName, ok := hashFixtureNames[hashName]
		if !ok {
			t.Fatalf("golden hash %q has no Go parity fixture mapping", hashName)
		}

		t.Run(hashName, func(t *testing.T) {
			value := readFixtureJSON[any](t, filepath.Join(forgekFixtureRoot, "valid", fixtureName))
			actual := stableFixtureHash(t, value)
			if actual != expected {
				t.Fatalf("%s drift for %s: got %s, want %s", hashName, fixtureName, actual, expected)
			}
		})
	}
}

func TestForgeKFixtureGoldenCanonicalJSON(t *testing.T) {
	goldenFixtureNames := map[string]string{
		"canonical_context_block.json":     "context_block.valid.json",
		"canonical_kv_manifest.json":       "kv_cache_manifest.valid.json",
		"canonical_snapshot_manifest.json": "snapshot_manifest.valid.json",
	}

	for goldenName, fixtureName := range goldenFixtureNames {
		t.Run(goldenName, func(t *testing.T) {
			value := readFixtureJSON[any](t, filepath.Join(forgekFixtureRoot, "valid", fixtureName))
			actual := compactFixtureJSON(canonicalizeFixtureValue(value, ""))

			raw, err := os.ReadFile(filepath.Join(forgekFixtureRoot, "golden", goldenName))
			if err != nil {
				t.Fatalf("read %s: %v", goldenName, err)
			}
			expected := strings.TrimRight(string(raw), "\r\n")
			if actual != expected {
				t.Fatalf("canonical JSON drift for %s", fixtureName)
			}
		})
	}
}

func TestForgeKValidFixturesHaveStableSimulatorShape(t *testing.T) {
	entries, err := os.ReadDir(filepath.Join(forgekFixtureRoot, "valid"))
	if err != nil {
		t.Fatalf("read valid fixtures: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".valid.json") {
			continue
		}
		t.Run(entry.Name(), func(t *testing.T) {
			value := readFixtureJSON[any](t, filepath.Join(forgekFixtureRoot, "valid", entry.Name()))
			validateStableFixtureFields(t, entry.Name(), value)
			if got := stableFixtureHash(t, value); len(got) != 64 {
				t.Fatalf("stable hash length = %d, want 64", len(got))
			}
		})
	}
}

func TestForgeKInvalidFixturesAreRejectedByGoFixtureRules(t *testing.T) {
	tests := map[string]func(*testing.T, map[string]any){
		"snapshot_missing_workspace.invalid.json": func(t *testing.T, object map[string]any) {
			if hasString(object, "workspace_id") {
				t.Fatalf("invalid snapshot unexpectedly has workspace_id")
			}
		},
		"context_block_missing_source_refs.invalid.json": func(t *testing.T, object map[string]any) {
			if hasAnyNonEmptyArray(object,
				"source_object_refs", "source_refs", "admitted_exhibit_refs", "rejected_exhibit_refs",
				"ruling_refs", "contradiction_refs", "supersession_refs", "palace_route_refs",
				"semantic_operation_refs", "derived_object_refs",
			) {
				t.Fatalf("invalid context block unexpectedly has provenance refs")
			}
		},
		"kv_manifest_missing_token_hash.invalid.json": func(t *testing.T, object map[string]any) {
			if hasString(object, "token_input_hash") {
				t.Fatalf("invalid kv manifest unexpectedly has token_input_hash")
			}
		},
		"kv_manifest_bad_cache_mode.invalid.json": func(t *testing.T, object map[string]any) {
			if stringIn(stringField(object, "cache_mode"), []string{"STRICT_PREFIX", "SNAPSHOT_PREFIX", "BACKEND_COMPOSITIONAL"}) {
				t.Fatalf("invalid kv manifest unexpectedly has valid cache_mode")
			}
		},
		"runtime_driver_manifest_with_secret.invalid.json": func(t *testing.T, object map[string]any) {
			if !hasSecretLookingFixtureField(object) {
				t.Fatalf("invalid runtime manifest did not expose a secret-looking field")
			}
		},
	}

	for fixtureName, check := range tests {
		t.Run(fixtureName, func(t *testing.T) {
			value := readFixtureJSON[map[string]any](t, filepath.Join(forgekFixtureRoot, "invalid", fixtureName))
			check(t, value)
		})
	}
}

func readGoldenFixtureHashes(t *testing.T) map[string]string {
	t.Helper()

	raw := readFixtureJSON[map[string]any](t, filepath.Join(forgekFixtureRoot, "golden", "hashes.json"))
	hashes := make(map[string]string)
	for key, value := range raw {
		hashValue, ok := value.(string)
		if ok && strings.HasSuffix(key, "_hash") {
			hashes[key] = hashValue
		}
	}
	return hashes
}

func readFixtureJSON[T any](t *testing.T, path string) T {
	t.Helper()

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	var value T
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return value
}

func validateStableFixtureFields(t *testing.T, fixtureName string, value any) {
	t.Helper()

	object, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("%s root is %T, want JSON object", fixtureName, value)
	}

	switch {
	case hasKeys(object, "snapshot_id", "snapshot_type"):
		requireStringField(t, object, "snapshot_id")
		requireStringField(t, object, "workspace_id")
		requireEnumField(t, object, "snapshot_type", []string{
			"SEMANTIC_SNAPSHOT", "CASE_SNAPSHOT", "CONTEXT_RESTORE_SNAPSHOT", "PALACE_ROUTE_SNAPSHOT",
			"WORKSPACE_SNAPSHOT", "DECISION_SNAPSHOT", "KV_SHAPE_SNAPSHOT", "RUNTIME_SNAPSHOT",
		})
		requireEnumField(t, object, "status", []string{
			"DRAFT", "SEALED", "SUPERSEDED", "EXPIRED", "RESTORE_SEED_CREATED",
		})
		if stringField(object, "snapshot_type") != "WORKSPACE_SNAPSHOT" && !hasAnyNonEmptyArray(object,
			"source_object_refs", "source_refs", "palace_route_refs", "submitted_object_refs",
			"admitted_object_refs", "rejected_object_refs", "semantic_operation_refs",
			"contradiction_refs", "supersession_refs", "derived_object_refs",
			"context_block_refs", "token_hash_refs", "kv_manifest_refs",
		) {
			t.Fatalf("%s snapshot requires provenance refs", fixtureName)
		}
	case hasKeys(object, "block_id", "block_type"):
		requireContextBlockFixture(t, fixtureName, object)
	case hasKeys(object, "bundle_id", "blocks"):
		requireStringField(t, object, "bundle_id")
		requireStringField(t, object, "workspace_id")
		requireStringField(t, object, "layout_version")
		blocks, ok := object["blocks"].([]any)
		if !ok || len(blocks) == 0 {
			t.Fatalf("%s requires non-empty blocks", fixtureName)
		}
		for index, block := range blocks {
			blockObject, ok := block.(map[string]any)
			if !ok {
				t.Fatalf("%s block %d is %T, want JSON object", fixtureName, index, block)
			}
			requireContextBlockFixture(t, fmt.Sprintf("%s block %d", fixtureName, index), blockObject)
		}
	case hasKeys(object, "cache_id"):
		for _, field := range []string{
			"cache_id", "workspace_id", "model_id", "model_revision", "tokenizer_id", "tokenizer_revision",
			"chat_template_hash", "prompt_layout_hash", "policy_schema_hash", "syscall_schema_hash",
			"token_input_hash", "runtime_backend", "runtime_version", "attention_backend", "rope_config_hash",
			"kv_precision", "cache_salt",
		} {
			requireStringField(t, object, field)
		}
		if !hasString(object, "bundle_id") && !hasString(object, "block_id") {
			t.Fatalf("%s kv manifest requires bundle_id or block_id", fixtureName)
		}
		requireEnumField(t, object, "cache_mode", []string{"STRICT_PREFIX", "SNAPSHOT_PREFIX", "BACKEND_COMPOSITIONAL"})
		requireEnumField(t, object, "memory_tier", []string{"GPU_HOT", "CPU_WARM", "DISK_COLD", "REMOTE_COLD", "NONE"})
		requireEnumField(t, object, "status", []string{"AVAILABLE", "HIT_RECORDED", "INVALIDATED", "EVICTED", "EXPIRED"})
	case hasKeys(object, "driver_id", "driver_kind"):
		requireStringField(t, object, "driver_id")
		requireStringField(t, object, "runtime_backend")
		requireStringField(t, object, "runtime_version")
		requireEnumField(t, object, "driver_kind", []string{
			"MOCK", "LOCAL_DEV", "VLLM", "SGLANG", "TENSORRT_LLM", "OLLAMA", "LLAMA_CPP", "REMOTE_API", "FUTURE",
		})
		if authority := stringField(object, "authority_level"); authority != "" && authority != "PROPOSAL_ONLY" {
			t.Fatalf("%s authority_level = %q, want PROPOSAL_ONLY", fixtureName, authority)
		}
	default:
		t.Fatalf("%s has unknown FORGE-K fixture kind", fixtureName)
	}
}

func requireContextBlockFixture(t *testing.T, name string, object map[string]any) {
	t.Helper()

	requireStringField(t, object, "block_id")
	requireStringField(t, object, "workspace_id")
	requireEnumField(t, object, "block_type", []string{
		"KERNEL_DOCTRINE", "POLICY_BOUNDARY", "TOOL_CONTRACTS", "WORKSPACE_IDENTITY",
		"GOVERNING_PRECEDENT", "CASE_SUMMARY", "PALACE_ROUTE_SUMMARY", "ADMITTED_EVIDENCE",
		"REJECTED_EVIDENCE_SUMMARY", "CONTRADICTION_SUMMARY", "SEMANTIC_OPERATION_SUMMARY",
		"SNAPSHOT_RESTORE_SEED", "ACTIVE_CONSTRAINTS", "CURRENT_TASK", "VOLATILE_DETAIL",
		"USER_MESSAGE", "FUTURE_TOKEN_PLACEHOLDER", "FUTURE_KV_PLACEHOLDER",
	})
	requireEnumField(t, object, "cache_eligibility", []string{
		"CACHE_ALWAYS", "CACHE_IF_STABLE", "CACHE_EPHEMERAL", "DO_NOT_CACHE",
	})
	if !hasAnyNonEmptyArray(object,
		"source_object_refs", "source_refs", "admitted_exhibit_refs", "rejected_exhibit_refs",
		"ruling_refs", "contradiction_refs", "supersession_refs", "palace_route_refs",
		"semantic_operation_refs", "derived_object_refs",
	) {
		t.Fatalf("%s requires provenance refs", name)
	}
	if !hasString(object, "canonical_text") && !hasString(object, "content_hash") {
		t.Fatalf("%s requires canonical_text or content_hash", name)
	}
}

func stableFixtureHash(t *testing.T, value any) string {
	t.Helper()

	projected := removeStableExcludedFields(canonicalizeFixtureValue(value, ""))
	canonical := canonicalFixtureJSON(t, projected)
	sum := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(sum[:])
}

func canonicalizeFixtureValue(value any, parentKey string) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, child := range typed {
			out[key] = canonicalizeFixtureValue(child, key)
		}
		return out
	case []any:
		out := make([]any, 0, len(typed))
		for _, child := range typed {
			out = append(out, canonicalizeFixtureValue(child, parentKey))
		}
		if isUnorderedFixtureArray(parentKey) {
			sort.Slice(out, func(i, j int) bool {
				return compactFixtureJSON(out[i]) < compactFixtureJSON(out[j])
			})
		}
		return out
	case string:
		return strings.Join(strings.Fields(typed), " ")
	default:
		return typed
	}
}

func removeStableExcludedFields(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, child := range typed {
			if isStableExcludedFixtureField(key) {
				continue
			}
			out[key] = removeStableExcludedFields(child)
		}
		return out
	case []any:
		out := make([]any, 0, len(typed))
		for _, child := range typed {
			out = append(out, removeStableExcludedFields(child))
		}
		return out
	default:
		return typed
	}
}

func canonicalFixtureJSON(t *testing.T, value any) string {
	t.Helper()

	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("canonical marshal: %v", err)
	}
	return string(raw)
}

func compactFixtureJSON(value any) string {
	raw, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(raw)
}

func isUnorderedFixtureArray(key string) bool {
	return strings.HasSuffix(key, "_refs") ||
		key == "source_refs" ||
		key == "blocks_refs" ||
		key == "supported_models" ||
		key == "supported_capabilities" ||
		key == "allowed_syscalls" ||
		key == "workspace_scope" ||
		key == "provenance_refs" ||
		key == "context_block_refs"
}

func isStableExcludedFixtureField(key string) bool {
	switch key {
	case "created_at", "updated_at", "sealed_at", "expired_at", "last_used_at", "invalidated_at",
		"journal_refs", "shape_hash", "source_hash", "content_hash", "token_input_hash",
		"bundle_hash", "stable_prefix_hash", "volatile_suffix_hash", "cache_id", "snapshot_id",
		"block_id", "bundle_id", "driver_id", "reuse_count":
		return true
	default:
		return false
	}
}

func hasKeys(object map[string]any, keys ...string) bool {
	for _, key := range keys {
		if _, ok := object[key]; !ok {
			return false
		}
	}
	return true
}

func requireStringField(t *testing.T, object map[string]any, field string) {
	t.Helper()

	if !hasString(object, field) {
		t.Fatalf("field %q is missing or not a non-empty string", field)
	}
}

func requireEnumField(t *testing.T, object map[string]any, field string, allowed []string) {
	t.Helper()

	value := stringField(object, field)
	if stringIn(value, allowed) {
		return
	}
	t.Fatalf("field %q = %q, want one of %v", field, value, allowed)
}

func stringIn(value string, allowed []string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

func hasString(object map[string]any, field string) bool {
	return stringField(object, field) != ""
}

func stringField(object map[string]any, field string) string {
	value, ok := object[field].(string)
	if !ok {
		return ""
	}
	return value
}

func hasAnyNonEmptyArray(object map[string]any, fields ...string) bool {
	for _, field := range fields {
		values, ok := object[field].([]any)
		if ok && len(values) > 0 {
			return true
		}
	}
	return false
}

func hasSecretLookingFixtureField(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			lowerKey := strings.ToLower(key)
			if strings.Contains(lowerKey, "secret") ||
				strings.Contains(lowerKey, "password") ||
				strings.Contains(lowerKey, "api_key") ||
				strings.Contains(lowerKey, "token") {
				return true
			}
			if hasSecretLookingFixtureField(child) {
				return true
			}
		}
	case []any:
		for _, child := range typed {
			if hasSecretLookingFixtureField(child) {
				return true
			}
		}
	}
	return false
}
