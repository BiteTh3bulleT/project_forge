package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forge/projectforge/services/core/internal/config"
	"forge/projectforge/services/core/internal/store"
)

func TestForgeSystemStatusReadOnlySurface(t *testing.T) {
	dataDir := t.TempDir()
	workspaceDir := filepath.Join(dataDir, "workspace")
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	srv := NewServer(st, config.Config{DataDir: dataDir, WorkspaceDir: workspaceDir})
	req := httptest.NewRequest(http.MethodGet, "/forge/system/status", nil)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	shellSession := asMap(t, payload["shell_session"])
	for _, key := range []string{
		"host_mutation_disabled",
		"model_mutation_disabled",
		"semantic_memory_write_disabled",
		"forge_k_live_authority_disabled",
	} {
		if shellSession[key] != true {
			t.Fatalf("shell_session.%s=%v, want true", key, shellSession[key])
		}
	}

	core := asMap(t, payload["core"])
	if core["core_url"] != "http://example.com" {
		t.Fatalf("core.core_url=%v, want http://example.com", core["core_url"])
	}

	forgeh := asMap(t, payload["forgeh"])
	if forgeh["advisory_only"] != true {
		t.Fatalf("forgeh.advisory_only=%v, want true", forgeh["advisory_only"])
	}
	if forgeh["canonical_write_committed"] != false {
		t.Fatalf("forgeh.canonical_write_committed=%v, want false", forgeh["canonical_write_committed"])
	}
	executions := asMap(t, forgeh["executions"])
	if executions["available"] != false {
		t.Fatalf("forgeh.executions.available=%v, want false", executions["available"])
	}
	if items, ok := executions["items"].([]any); !ok || len(items) != 0 {
		t.Fatalf("forgeh.executions.items=%#v, want empty list", executions["items"])
	}

	kernel := asMap(t, payload["kernel_activation"])
	if kernel["status"] != "partial_live_validation_ready" {
		t.Fatalf("kernel_activation.status=%v, want partial_live_validation_ready", kernel["status"])
	}
	if kernel["live_kernel_authority"] != false ||
		kernel["simulator_authority"] != false ||
		kernel["live_authority_migration"] != false ||
		kernel["mutation_controls_available"] != false {
		t.Fatalf("kernel_activation claimed forbidden authority or mutation controls: %#v", kernel)
	}
	if kernel["authority_ready_gates"] != float64(4) || kernel["authority_blocked_gates"] != float64(3) {
		t.Fatalf("kernel_activation authority gate counts unexpected: %#v", kernel)
	}

	storage := asMap(t, payload["storage"])
	if storage["truth_authority"] != "sqlite" {
		t.Fatalf("storage.truth_authority=%v, want sqlite", storage["truth_authority"])
	}
	cutover := asMap(t, storage["cutover_readiness"])
	if cutover["status"] != "blocked" {
		t.Fatalf("storage.cutover_readiness.status=%v, want blocked", cutover["status"])
	}
	if cutover["canonical_default"] != "sqlite" || cutover["requested_backend"] != "sqlite" {
		t.Fatalf("storage cutover changed backend defaults: %#v", cutover)
	}
	if cutover["live_owner"] == "" || cutover["target_owner"] == "" {
		t.Fatalf("storage cutover readiness missing owner fields: %#v", cutover)
	}
	if redis := asMap(t, storage["redis"]); redis["truth_authority"] != false {
		t.Fatalf("storage.redis.truth_authority=%v, want false", redis["truth_authority"])
	}
	if qdrant := asMap(t, storage["qdrant"]); qdrant["truth_authority"] != false {
		t.Fatalf("storage.qdrant.truth_authority=%v, want false", qdrant["truth_authority"])
	}

	legacy := asMap(t, payload["legacy_retirement"])
	if legacy["status"] != "direct_mutation_retired" {
		t.Fatalf("legacy_retirement.status=%v, want direct_mutation_retired", legacy["status"])
	}
	if legacy["direct_mutation_disabled"] != true ||
		legacy["rollback_proof_required"] != true ||
		legacy["no_forge_k_simulator_authority"] != true {
		t.Fatalf("legacy retirement flags wrong: %#v", legacy)
	}
	entries, ok := legacy["entries"].([]any)
	if !ok || len(entries) != 2 {
		t.Fatalf("legacy retirement entries=%#v, want two entries", legacy["entries"])
	}
	byID := map[string]map[string]any{}
	for _, raw := range entries {
		entry := asMap(t, raw)
		id, ok := entry["id"].(string)
		if !ok || id == "" {
			t.Fatalf("legacy retirement entry missing id: %#v", entry)
		}
		for _, key := range []string{"live_owner", "target_forge_k_owner", "default_live_replacement", "rollback_proof"} {
			if entry[key] == "" {
				t.Fatalf("legacy retirement entry %s missing %s: %#v", id, key, entry)
			}
		}
		if entry["mutation_allowed"] != false {
			t.Fatalf("legacy retirement entry %s allows mutation: %#v", id, entry)
		}
		byID[id] = entry
	}
	if byID["legacy_adapter_direct_invoke"]["route_state"] != "unrouted" {
		t.Fatalf("legacy adapter route state wrong: %#v", byID["legacy_adapter_direct_invoke"])
	}
	if byID["legacy_memory_observation_mutation"]["route_state"] != "gone_audited" {
		t.Fatalf("legacy memory route state wrong: %#v", byID["legacy_memory_observation_mutation"])
	}
}

func TestForgeSystemStatusDoesNotExposeMutationMethod(t *testing.T) {
	dataDir := t.TempDir()
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	srv := NewServer(st, config.Config{DataDir: dataDir, WorkspaceDir: filepath.Join(dataDir, "workspace")})
	req := httptest.NewRequest(http.MethodPost, "/forge/system/status", nil)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST /forge/system/status status=%d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
}

func TestForgeSystemStatusWarnsWhenCapabilityOverrideStoreFallsBack(t *testing.T) {
	dataDir := t.TempDir()
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	srv := NewServer(st, config.Config{DataDir: dataDir, WorkspaceDir: filepath.Join(dataDir, "workspace")})
	srv.capStoreOK = false
	srv.capStoreErr = "load tool capability overrides: missing table"

	req := httptest.NewRequest(http.MethodGet, "/forge/system/status", nil)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	warnings, ok := payload["warnings"].([]any)
	if !ok {
		t.Fatalf("warnings=%#v, want list", payload["warnings"])
	}
	if !containsStringValue(warnings, "gateway capability override store unavailable; capability status overrides are in-memory only for this process") {
		t.Fatalf("expected gateway override fallback warning, got %#v", warnings)
	}
}

func TestForgeSystemStatusSourceContainsNoHostMutationCommands(t *testing.T) {
	body, err := os.ReadFile("system_status.go")
	if err != nil {
		t.Fatalf("read system_status.go: %v", err)
	}
	source := string(body)
	for _, forbidden := range []string{
		"exec.Command",
		"systemctl start",
		"systemctl stop",
		"systemctl restart",
		"nixos-rebuild",
		"modprobe",
		"rmmod",
		"reboot",
		"shutdown",
		"LoadModel(",
		"UnloadModel(",
		"GenerateStream(",
		"os.RemoveAll",
	} {
		if strings.Contains(source, forbidden) {
			t.Fatalf("system status route must not contain forbidden mutation text %q", forbidden)
		}
	}
}

func containsStringValue(values []any, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func asMap(t *testing.T, value any) map[string]any {
	t.Helper()
	out, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("value=%#v, want object", value)
	}
	return out
}
