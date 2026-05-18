package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestForgeKernelStatusReadOnlyActivationReadiness(t *testing.T) {
	srv := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/forge/kernel/status", nil)
	rr := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d body=%s", rr.Code, http.StatusOK, rr.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["status"] != "partial_live_validation_ready" {
		t.Fatalf("unexpected kernel status payload: %#v", payload)
	}
	if payload["live_kernel_authority"] != false ||
		payload["simulator_authority"] != false ||
		payload["live_authority_migration"] != false ||
		payload["shadow_authoritative"] != false ||
		payload["mutation_controls_available"] != false {
		t.Fatalf("kernel status claimed forbidden authority or mutation controls: %#v", payload)
	}

	actions, ok := payload["validation_actions"].([]any)
	if !ok || len(actions) != 5 {
		t.Fatalf("expected five validation actions, got %#v", payload["validation_actions"])
	}
	for _, raw := range actions {
		action, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("unexpected action shape: %#v", raw)
		}
		if action["registered"] != true || action["mutating"] != false || action["approval_possible"] != false {
			t.Fatalf("validation action is not read-only closed: %#v", action)
		}
	}

	if payload["authority_ready_gates"] != float64(2) || payload["authority_blocked_gates"] != float64(4) {
		t.Fatalf("unexpected authority gate counts: %#v", payload)
	}
	gates, ok := payload["authority_gates"].([]any)
	if !ok || len(gates) != 6 {
		t.Fatalf("expected six authority gates, got %#v", payload["authority_gates"])
	}
	for _, raw := range gates {
		gate, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("unexpected authority gate shape: %#v", raw)
		}
		if gate["mutation_authority"] != false {
			t.Fatalf("authority readiness gate granted mutation authority: %#v", gate)
		}
		if gate["status"] == "blocked" && gate["next_step"] == "" {
			t.Fatalf("blocked authority gate lacks next step: %#v", gate)
		}
	}

	matrix, ok := payload["authority_matrix"].([]any)
	if !ok || len(matrix) != 10 {
		t.Fatalf("expected ten authority matrix entries, got %#v", payload["authority_matrix"])
	}
	seen := map[string]bool{}
	for _, raw := range matrix {
		entry, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("unexpected authority matrix entry shape: %#v", raw)
		}
		subsystem, _ := entry["subsystem"].(string)
		seen[subsystem] = true
		if entry["operator_visible"] != true {
			t.Fatalf("authority matrix entry must be operator visible: %#v", entry)
		}
		if entry["live_owner"] == "" || entry["target_owner"] == "" || entry["rollback_path"] == "" {
			t.Fatalf("authority matrix entry missing owner/rollback fields: %#v", entry)
		}
		if _, ok := entry["tests_required"].([]any); !ok {
			t.Fatalf("authority matrix entry missing tests_required: %#v", entry)
		}
	}
	for _, subsystem := range []string{
		"Kernel",
		"Courthouse",
		"Memory Palace",
		"Semantic Algebra",
		"Snapshots",
		"Context Compiler",
		"KV System",
		"Runtime Boundary",
		"Lymphatic Lane",
		"Consensus Mesh",
	} {
		if !seen[subsystem] {
			t.Fatalf("missing authority matrix subsystem %q in %#v", subsystem, matrix)
		}
	}
}

func TestForgeKernelStatusDoesNotExposeMutationMethod(t *testing.T) {
	srv := &Server{}
	req := httptest.NewRequest(http.MethodPost, "/forge/kernel/status", nil)
	rr := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST /forge/kernel/status status=%d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
}

func TestForgeKernelStatusSourceContainsNoMutationCommands(t *testing.T) {
	body, err := os.ReadFile("kernel_status.go")
	if err != nil {
		t.Fatalf("read kernel_status.go: %v", err)
	}
	source := string(body)
	for _, forbidden := range []string{
		"exec.Command",
		"systemctl",
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
			t.Fatalf("kernel status route must not contain forbidden mutation text %q", forbidden)
		}
	}
}
