package controllane

import (
	"context"
	"testing"

	"forge/projectforge/services/core/internal/aios/domain"
)

func TestValidateKVIdentityLiveSyscallSucceedsWithoutMemoryMutation(t *testing.T) {
	ctx := context.Background()
	k, _, auditSink := newTestKernel()
	req := validKVIdentityRequest()
	res, err := k.Process(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected success, got %#v", res)
	}
	if len(res.CommittedObjectIDs) != 0 {
		t.Fatalf("in-memory control lane has no journal store; expected no semantic commits, got %v", res.CommittedObjectIDs)
	}
	if got, ok := res.StateSummary["passed"].(bool); !ok || !got {
		t.Fatalf("expected passed summary, got %#v", res.StateSummary)
	}
	if res.StateSummary["memoryMutation"] != false || res.StateSummary["runtimeMutation"] != false || res.StateSummary["liveKVReuse"] != false {
		t.Fatalf("KV identity validation claimed mutation/reuse: %#v", res.StateSummary)
	}
	if res.AuditID == "" || len(auditSink.Records) == 0 || !auditSink.Records[len(auditSink.Records)-1].Success {
		t.Fatalf("expected successful audit record, auditID=%q records=%#v", res.AuditID, auditSink.Records)
	}
}

func TestValidateKVIdentityRejectsMismatchWithoutStateMutation(t *testing.T) {
	ctx := context.Background()
	k, store, auditSink := newTestKernel()
	req := validKVIdentityRequest()
	req.ID = "kv-identity-bad"
	req.IdempotencyKey = "kv-idem-bad"
	request := req.Payload["request"].(map[string]any)
	request["token_input_hash"] = "different-token"
	res, err := k.Process(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Success {
		t.Fatalf("expected rejection, got %#v", res)
	}
	if res.DeterministicErrCode != domain.ErrInvalidPayload {
		t.Fatalf("expected invalid payload, got %s", res.DeterministicErrCode)
	}
	if len(res.CommittedObjectIDs) != 0 {
		t.Fatalf("rejected request committed ids: %v", res.CommittedObjectIDs)
	}
	if _, ok := store.GetIdempotency(req.IdempotencyKey); ok {
		t.Fatal("failed validation must not persist idempotency state")
	}
	if res.AuditID == "" || len(auditSink.Records) == 0 || auditSink.Records[len(auditSink.Records)-1].Success {
		t.Fatalf("expected rejected audit record, auditID=%q records=%#v", res.AuditID, auditSink.Records)
	}
}

func TestValidateKVIdentityRejectsMissingPayloadBeforeCommit(t *testing.T) {
	ctx := context.Background()
	k, store, _ := newTestKernel()
	req := validBaseRequest(domain.ActionValidateKVIdentity)
	req.ID = "kv-identity-missing"
	req.IdempotencyKey = "kv-idem-missing"
	req.Payload = map[string]any{"manifest": map[string]any{}}
	res, err := k.Process(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Success {
		t.Fatalf("expected missing request payload to fail")
	}
	if len(res.CommittedObjectIDs) != 0 {
		t.Fatalf("invalid request committed ids: %v", res.CommittedObjectIDs)
	}
	if _, ok := store.GetIdempotency(req.IdempotencyKey); ok {
		t.Fatal("invalid request must not persist idempotency state")
	}
}

func TestValidateKVIdentityCapabilityDeniedForProposeOnlySource(t *testing.T) {
	ctx := context.Background()
	k, _, _ := newTestKernel()
	req := validKVIdentityRequest()
	req.ID = "kv-identity-future-iris"
	req.Source = domain.SourceFutureIRIS
	req.Actor.Kind = string(domain.SourceFutureIRIS)
	req.Provenance.ActorType = string(domain.SourceFutureIRIS)
	res, err := k.Process(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Success {
		t.Fatalf("expected propose-only source to be denied")
	}
	if res.DeterministicErrCode != domain.ErrCapabilityDenied {
		t.Fatalf("expected capability denied, got %s", res.DeterministicErrCode)
	}
}

func TestValidateKVIdentityIdempotentReplay(t *testing.T) {
	ctx := context.Background()
	k, _, _ := newTestKernel()
	req := validKVIdentityRequest()
	req.IdempotencyKey = "kv-idem-pass"
	first, err := k.Process(ctx, req)
	if err != nil || !first.Success {
		t.Fatalf("first request failed: err=%v res=%#v", err, first)
	}
	req.ID = "kv-identity-pass-replay"
	second, err := k.Process(ctx, req)
	if err != nil || !second.Success {
		t.Fatalf("replay failed: err=%v res=%#v", err, second)
	}
	if second.StateSummary["passed"] != true {
		t.Fatalf("replay lost state summary: %#v", second.StateSummary)
	}
	if !containsWarning(second.Warnings, "idempotent replay") {
		t.Fatalf("expected idempotent replay warning, got %v", second.Warnings)
	}
}

func validKVIdentityRequest() domain.SyscallRequest {
	req := validBaseRequest(domain.ActionValidateKVIdentity)
	req.ID = "kv-identity-pass"
	req.Payload = map[string]any{
		"manifest": validKVManifestPayload(),
		"request":  validKVRequestPayload(),
	}
	return req
}

func validKVManifestPayload() map[string]any {
	return map[string]any{
		"cache_id":             "cache-a",
		"cache_mode":           "STRICT_PREFIX",
		"workspace_id":         "ws-main",
		"bundle_id":            "bundle-a",
		"block_id":             "block-a",
		"bundle_hash":          "bundle-hash",
		"stable_prefix_hash":   "stable-hash",
		"volatile_suffix_hash": "volatile-hash",
		"model_id":             "model-a",
		"model_revision":       "rev-a",
		"tokenizer_id":         "tok-a",
		"tokenizer_revision":   "tok-rev-a",
		"chat_template_hash":   "chat-template",
		"prompt_layout_hash":   "layout",
		"policy_schema_hash":   "policy",
		"syscall_schema_hash":  "syscall",
		"token_input_hash":     "token-input",
		"runtime_backend":      "mock",
		"runtime_version":      "1",
		"attention_backend":    "attn",
		"rope_config_hash":     "rope",
		"kv_precision":         "fp16",
		"cache_salt":           "salt",
		"status":               "AVAILABLE",
	}
}

func validKVRequestPayload() map[string]any {
	manifest := validKVManifestPayload()
	out := make(map[string]any, len(manifest))
	for key, value := range manifest {
		if key != "cache_id" && key != "status" {
			out[key] = value
		}
	}
	out["request_id"] = "kv-request-a"
	return out
}

func containsWarning(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
