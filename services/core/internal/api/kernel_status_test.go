package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/config"
	"forge/projectforge/services/core/internal/forgekernel"
	"forge/projectforge/services/core/internal/store"
)

type kernelStatusProcessor struct{}

func (kernelStatusProcessor) Process(context.Context, domain.SyscallRequest) (domain.SyscallResult, error) {
	return domain.SyscallResult{}, nil
}

func liveForgeKStatusSelection() forgekernel.Selection {
	return forgekernel.Selection{
		Processor:       kernelStatusProcessor{},
		Mode:            forgekernel.ModeForgeK,
		AuthorityOwner:  forgekernel.AuthorityOwnerForgeK,
		CommitAdapter:   forgekernel.DurableCommitAdapter,
		RollbackMode:    forgekernel.ModeLegacyV1,
		SingleAuthority: true,
	}
}

func TestForgeKernelStatusReadOnlyActivationReadiness(t *testing.T) {
	srv := &Server{kernelAuthority: liveForgeKStatusSelection()}
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
	if payload["status"] != "forge_k_durable_orchestration_live" {
		t.Fatalf("unexpected kernel status payload: %#v", payload)
	}
	if payload["live_kernel_authority"] != false ||
		payload["simulator_authority"] != false ||
		payload["live_kernel_ingress_authority"] != true ||
		payload["live_durable_orchestration"] != true ||
		payload["live_authority_migration"] != true ||
		payload["shadow_authoritative"] != false ||
		payload["mutation_controls_available"] != false {
		t.Fatalf("kernel status claimed forbidden authority or mutation controls: %#v", payload)
	}

	actions, ok := payload["validation_actions"].([]any)
	if !ok || len(actions) != 7 {
		t.Fatalf("expected seven validation actions, got %#v", payload["validation_actions"])
	}
	seenActions := map[string]bool{}
	for _, raw := range actions {
		action, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("unexpected action shape: %#v", raw)
		}
		name, _ := action["action"].(string)
		seenActions[name] = true
		if action["registered"] != true || action["mutating"] != false || action["approval_possible"] != false {
			t.Fatalf("validation action is not read-only closed: %#v", action)
		}
		if action["live_owner"] != forgekernel.AuthorityOwnerForgeK {
			t.Fatalf("validation action bypassed boot-selected ingress owner: %#v", action)
		}
	}
	if !seenActions["VALIDATE_ADMISSION_CANDIDATE"] {
		t.Fatalf("missing admission candidate validation action: %#v", actions)
	}
	if !seenActions["VALIDATE_CONTEXT_ATTRIBUTION"] {
		t.Fatalf("missing context attribution validation action: %#v", actions)
	}
	activationGates, ok := payload["gates"].([]any)
	if !ok {
		t.Fatalf("missing activation gates: %#v", payload["gates"])
	}
	seenFullAuthorityGate := false
	for _, raw := range activationGates {
		gate, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("unexpected activation gate shape: %#v", raw)
		}
		if gate["name"] == "full_kernel_authority_gated" {
			seenFullAuthorityGate = true
		}
		if gate["name"] == "live_kernel_authority_disabled" {
			t.Fatalf("live K20A status retained legacy disabled gate: %#v", gate)
		}
	}
	if !seenFullAuthorityGate {
		t.Fatalf("missing full authority migration gate: %#v", activationGates)
	}

	if payload["authority_ready_gates"] != float64(4) || payload["authority_blocked_gates"] != float64(3) {
		t.Fatalf("unexpected authority gate counts: %#v", payload)
	}
	gates, ok := payload["authority_gates"].([]any)
	if !ok || len(gates) != 7 {
		t.Fatalf("expected seven authority gates, got %#v", payload["authority_gates"])
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
		if subsystem == "Courthouse" && entry["current_status"] != "ADMISSION_CANDIDATE_ONLY" {
			t.Fatalf("courthouse must remain candidate-only, got %#v", entry)
		}
		if subsystem == "Kernel" && (entry["current_status"] != "FORGE_K_DURABLE_ORCHESTRATION_LIVE" || entry["live_owner"] != "forge_k.kernel") {
			t.Fatalf("kernel ingress authority not reported: %#v", entry)
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

func TestForgeKernelStatusReportsLegacyRollbackMode(t *testing.T) {
	srv := &Server{
		cfg: config.Config{ForgeKernelAuthorityMode: "legacy_v1"},
		kernelAuthority: forgekernel.Selection{
			Processor:       kernelStatusProcessor{},
			Mode:            forgekernel.ModeLegacyV1,
			AuthorityOwner:  forgekernel.AuthorityOwnerLegacyV1,
			CommitAdapter:   forgekernel.DurableCommitAdapter,
			RollbackMode:    forgekernel.ModeLegacyV1,
			SingleAuthority: true,
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/forge/kernel/status", nil)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["status"] != "partial_live_validation_ready" || payload["live_kernel_ingress_authority"] != false || payload["live_durable_orchestration"] != false || payload["live_authority_migration"] != false {
		t.Fatalf("legacy rollback posture incorrect: %#v", payload)
	}
}

func TestForgeKernelStatusReportsBootSelectionFailureClosed(t *testing.T) {
	dataDir := t.TempDir()
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	srv := NewServer(st, config.Config{
		DataDir:                  dataDir,
		WorkspaceDir:             filepath.Join(dataDir, "workspace"),
		ForgeKernelAuthorityMode: "both",
	})
	t.Cleanup(srv.ShutdownWatch)
	if srv.kernelAuthority.SingleAuthority || srv.kernelAuthority.Processor != nil || srv.kernelErr == "" || srv.autonomy != nil {
		t.Fatalf("invalid boot authority did not fail closed: selection=%#v err=%q autonomy=%#v", srv.kernelAuthority, srv.kernelErr, srv.autonomy)
	}
	req := httptest.NewRequest(http.MethodGet, "/forge/kernel/status", nil)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["status"] != "kernel_authority_unavailable" || payload["mode"] != "fail_closed" || payload["live_owner"] != "none" {
		t.Fatalf("boot selection failure posture incorrect: %#v", payload)
	}
	if payload["live_kernel_ingress_authority"] != false || payload["live_durable_orchestration"] != false || payload["live_authority_migration"] != false {
		t.Fatalf("boot selection failure claimed live authority: %#v", payload)
	}
}

func TestForgeKernelStatusDoesNotExposeMutationMethod(t *testing.T) {
	srv := &Server{kernelAuthority: liveForgeKStatusSelection()}
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
