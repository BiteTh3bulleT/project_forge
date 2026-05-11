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
	if !ok || len(actions) != 4 {
		t.Fatalf("expected four validation actions, got %#v", payload["validation_actions"])
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

	if payload["authority_ready_gates"] != float64(1) || payload["authority_blocked_gates"] != float64(5) {
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
