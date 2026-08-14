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
	if kernel["status"] != "forge_k_authenticated_commit_integrity_live" {
		t.Fatalf("kernel_activation.status=%v, want forge_k_authenticated_commit_integrity_live", kernel["status"])
	}
	if kernel["live_kernel_authority"] != false ||
		kernel["simulator_authority"] != false ||
		kernel["live_kernel_ingress_authority"] != true ||
		kernel["live_durable_orchestration"] != true ||
		kernel["live_authority_migration"] != true ||
		kernel["mutation_controls_available"] != false {
		t.Fatalf("kernel_activation claimed forbidden authority or mutation controls: %#v", kernel)
	}
	if kernel["authority_ready_gates"] != float64(4) || kernel["authority_blocked_gates"] != float64(3) {
		t.Fatalf("kernel_activation authority gate counts unexpected: %#v", kernel)
	}
	integrity := asMap(t, kernel["no_effect"])
	if integrity["commitIntegrityAuthority"] != true ||
		integrity["sealedPreparedPlans"] != true ||
		integrity["typedCommitReceiptValidation"] != true ||
		integrity["atomicAuditOutboxEvidence"] != true ||
		integrity["verifiedIdempotentReplay"] != true ||
		integrity["externalAuditSinkDelivery"] != false ||
		integrity["auditIdBackfill"] != false {
		t.Fatalf("kernel_activation commit integrity posture unexpected: %#v", integrity)
	}

	operatorCockpit := asMap(t, payload["operator_cockpit"])
	if operatorCockpit["available"] != true || operatorCockpit["mutation_controls_available"] != false {
		t.Fatalf("operator_cockpit flags wrong: %#v", operatorCockpit)
	}
	rows, ok := operatorCockpit["rows"].([]any)
	if !ok || len(rows) == 0 {
		t.Fatalf("operator_cockpit.rows=%#v, want non-empty API-owned rows", operatorCockpit["rows"])
	}
	rowIDs := map[string]bool{}
	for _, raw := range rows {
		row := asMap(t, raw)
		id, ok := row["id"].(string)
		if !ok || id == "" {
			t.Fatalf("operator cockpit row missing id: %#v", row)
		}
		for _, key := range []string{"label", "status", "live_owner", "target_owner", "source"} {
			if row[key] == "" {
				t.Fatalf("operator cockpit row %s missing %s: %#v", id, key, row)
			}
		}
		if row["mutation_allowed"] != false {
			t.Fatalf("operator cockpit row %s allows mutation: %#v", id, row)
		}
		rowIDs[id] = true
	}
	for _, id := range []string{"authority_gates", "cases", "context_bundles", "proposals", "journal_replay", "lymphatic_reports"} {
		if !rowIDs[id] {
			t.Fatalf("operator cockpit missing row %s: %#v", id, rows)
		}
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

func TestForgeSystemHostReadOnlySettingsSurface(t *testing.T) {
	dataDir := t.TempDir()
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	srv := NewServer(st, config.Config{
		DataDir:              dataDir,
		WorkspaceDir:         filepath.Join(dataDir, "workspace"),
		SafeModeForceCPUOnly: true,
	})
	req := httptest.NewRequest(http.MethodGet, "/forge/system/host", nil)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if payload["read_only"] != true {
		t.Fatalf("read_only=%v, want true", payload["read_only"])
	}
	if payload["mutation_disabled"] != true {
		t.Fatalf("mutation_disabled=%v, want true", payload["mutation_disabled"])
	}
	if payload["live_owner"] != "forge.system.host" {
		t.Fatalf("live_owner=%v, want forge.system.host", payload["live_owner"])
	}

	host := asMap(t, payload["host"])
	if host["architecture"] == "" {
		t.Fatalf("host architecture not reported: %#v", host)
	}
	cpu := asMap(t, payload["cpu"])
	if cpu["count"] == float64(0) {
		t.Fatalf("cpu count not reported: %#v", cpu)
	}
	memory := asMap(t, payload["memory"])
	if memory["pressure_level"] == "" {
		t.Fatalf("memory pressure not reported: %#v", memory)
	}
	storage := asMap(t, payload["storage"])
	if storage["root"] == "" || storage["pressure_level"] == "" {
		t.Fatalf("storage not reported: %#v", storage)
	}

	for _, key := range []string{"display", "audio", "network", "power"} {
		section := asMap(t, payload[key])
		if section["read_only"] != true || section["mutation_disabled"] != true {
			t.Fatalf("%s does not preserve read-only mutation boundary: %#v", key, section)
		}
		if section["status"] == "" {
			t.Fatalf("%s status not reported: %#v", key, section)
		}
	}

	session := asMap(t, payload["session"])
	if session["safe_mode"] != true {
		t.Fatalf("session.safe_mode=%v, want true", session["safe_mode"])
	}
	if session["host_mutation_disabled"] != true {
		t.Fatalf("session host mutation not disabled: %#v", session)
	}
	configView := asMap(t, payload["config"])
	if configView["safe_mode_force_cpu_only"] != true {
		t.Fatalf("config safe mode not reported: %#v", configView)
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

func TestForgeSystemSurfacesSourceContainsNoHostMutationCommands(t *testing.T) {
	var source strings.Builder
	for _, path := range []string{"system_status.go", "system_host.go"} {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		source.Write(body)
	}
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
		if strings.Contains(source.String(), forbidden) {
			t.Fatalf("system surfaces must not contain forbidden mutation text %q", forbidden)
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
