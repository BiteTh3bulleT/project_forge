package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/forgekernel"
)

type kernelStatusProcessor struct{}

func (kernelStatusProcessor) Process(context.Context, domain.SyscallRequest) (domain.SyscallResult, error) {
	return domain.SyscallResult{}, nil
}

func liveForgeKStatusSelection() forgekernel.Selection {
	return forgekernel.Selection{
		Processor:       kernelStatusProcessor{},
		AuthorityOwner:  forgekernel.AuthorityOwnerForgeK,
		CommitAdapter:   forgekernel.DurableCommitAdapter,
		SingleAuthority: true,
	}
}

func liveForgeKStatusServer() *Server {
	return &Server{kernelAuthority: liveForgeKStatusSelection(), kernelAuthorizationReady: true}
}

func TestForgeKernelStatusReadOnlyActivationReadiness(t *testing.T) {
	srv := liveForgeKStatusServer()
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
	if payload["status"] != "forge_k_sole_live_authority" {
		t.Fatalf("unexpected kernel status payload: %#v", payload)
	}
	if payload["phase"] != "K20J" || payload["policy_version"] != "forge-k-sole-authority-k20j-v1" {
		t.Fatalf("unexpected kernel cutover identity: %#v", payload)
	}
	if payload["live_kernel_authority"] != true ||
		payload["simulator_authority"] != false ||
		payload["live_kernel_ingress_authority"] != true ||
		payload["live_durable_orchestration"] != true ||
		payload["live_authority_migration"] != false ||
		payload["shadow_authoritative"] != false ||
		payload["mutation_controls_available"] != false {
		t.Fatalf("kernel status claimed forbidden authority or mutation controls: %#v", payload)
	}
	integrity := asMap(t, payload["no_effect"])
	for _, key := range []string{
		"commitIntegrityAuthority",
		"sealedPreparedPlans",
		"typedCommitReceiptValidation",
		"atomicAuditOutboxEvidence",
		"verifiedIdempotentReplay",
		"authenticatedAuthorizationProof",
		"durableAuthorizationReplayProof",
		"uniqueForgeCoreServicePrincipal",
		"authenticatedUserOriginRequired",
	} {
		if integrity[key] != true {
			t.Fatalf("kernel status integrity flag %s=%v, want true: %#v", key, integrity[key], integrity)
		}
	}
	if integrity["externalAuditSinkDelivery"] != false || integrity["auditIdBackfill"] != false {
		t.Fatalf("kernel status overclaimed external audit projection completion: %#v", integrity)
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
		if action["live_kernel_authority"] != true {
			t.Fatalf("validation action retained pre-cutover authority metadata: %#v", action)
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
	seenSoleAuthorityGate := false
	for _, raw := range activationGates {
		gate, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("unexpected activation gate shape: %#v", raw)
		}
		if gate["name"] == "sole_live_kernel_authority" && gate["passed"] == true {
			seenSoleAuthorityGate = true
		}
		if gate["name"] == "live_kernel_authority_disabled" {
			t.Fatalf("live K20A status retained legacy disabled gate: %#v", gate)
		}
	}
	if !seenSoleAuthorityGate {
		t.Fatalf("missing sole live authority gate: %#v", activationGates)
	}

	if payload["authority_ready_gates"] != float64(7) || payload["authority_blocked_gates"] != float64(0) {
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
		if subsystem == "Courthouse" && (entry["current_status"] != "FORGE_K_ADMISSION_AND_RULING_LIVE" || entry["live_owner"] != "forge_k.kernel") {
			t.Fatalf("courthouse live authority not reported, got %#v", entry)
		}
		if subsystem == "Kernel" && (entry["current_status"] != "FORGE_K_SOLE_LIVE_AUTHORITY" || entry["live_owner"] != "forge_k.kernel") {
			t.Fatalf("sole kernel authority not reported: %#v", entry)
		}
		if subsystem == "Semantic Algebra" && (entry["current_status"] != "FORGE_K_DETERMINISTIC_DIFF_LIVE" || entry["live_owner"] != "forge_k.kernel") {
			t.Fatalf("semantic diff authority not reported: %#v", entry)
		}
		if subsystem == "Context Compiler" && (entry["current_status"] != "FORGE_K_CONTEXT_COMPILER_LIVE" || entry["live_owner"] != "forge_k.kernel") {
			t.Fatalf("context compiler authority not reported: %#v", entry)
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

func TestForgeKernelStatusReportsUnavailableAuthorityFailClosed(t *testing.T) {
	srv := &Server{kernelErr: forgekernel.ErrMissingAuthorization.Error()}
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
	if payload["status"] != "kernel_authority_unavailable" || payload["live_kernel_ingress_authority"] != false || payload["live_durable_orchestration"] != false || payload["live_authority_migration"] != false {
		t.Fatalf("unavailable authority fail-closed posture incorrect: %#v", payload)
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
