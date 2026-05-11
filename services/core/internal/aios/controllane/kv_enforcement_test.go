package controllane

import (
	"testing"

	"forge/projectforge/services/core/internal/aios/domain"
)

func TestKVIdentityEnforcementAcceptsValidClaim(t *testing.T) {
	decision := EnforceKVIdentity(validKVIdentityRequest())
	if !decision.Accepted {
		t.Fatalf("expected accepted decision, got %#v", decision)
	}
	if decision.Decision != KVIdentityDecisionAccepted {
		t.Fatalf("decision = %q, want %q", decision.Decision, KVIdentityDecisionAccepted)
	}
	if decision.AccelerationOnly != true || decision.MemoryMutation != false || decision.RuntimeMutation != false || decision.LiveKVReuse != false {
		t.Fatalf("unexpected authority flags: %#v", decision)
	}
	summary := decision.ToStateSummary()
	activation, ok := summary["forgeKActivation"].(map[string]any)
	if !ok {
		t.Fatalf("missing forgeKActivation summary: %#v", summary)
	}
	if activation["mode"] != ForgeKActivationModePartialLiveEnforcement ||
		activation["action"] != string(domain.ActionValidateKVIdentity) {
		t.Fatalf("unexpected activation summary: %#v", activation)
	}
	noEffect, ok := summary["forgeKNoEffect"].(map[string]any)
	if !ok || noEffect["memoryMutation"] != false || noEffect["runtimeMutation"] != false {
		t.Fatalf("unexpected no-effect summary: %#v", summary["forgeKNoEffect"])
	}
}

func TestKVIdentityEnforcementRejectsInvalidClaim(t *testing.T) {
	req := validKVIdentityRequest()
	req.Payload["request"].(map[string]any)["token_input_hash"] = "different-token"
	decision := EnforceKVIdentity(req)
	if decision.Accepted {
		t.Fatalf("expected rejected decision")
	}
	if decision.Decision != KVIdentityDecisionRejected {
		t.Fatalf("decision = %q, want %q", decision.Decision, KVIdentityDecisionRejected)
	}
	if !containsWarning(decision.FailedGates, "same_token_identity") {
		t.Fatalf("failed gates = %v", decision.FailedGates)
	}
}

func TestKVIdentityEnforcementRejectsMalformedClaim(t *testing.T) {
	req := validBaseRequest(domain.ActionValidateKVIdentity)
	req.Payload = map[string]any{"manifest": map[string]any{}}
	decision := EnforceKVIdentity(req)
	if decision.Accepted {
		t.Fatalf("expected malformed decision")
	}
	if decision.Decision != KVIdentityDecisionMalformed {
		t.Fatalf("decision = %q, want %q", decision.Decision, KVIdentityDecisionMalformed)
	}
}

func TestKVIdentityEnforcementRejectsUnsupportedLiveReuseClaim(t *testing.T) {
	req := validKVIdentityRequest()
	req.Payload["liveKVReuse"] = true
	decision := EnforceKVIdentity(req)
	if decision.Accepted {
		t.Fatalf("expected unsupported decision")
	}
	if decision.Decision != KVIdentityDecisionUnsupported {
		t.Fatalf("decision = %q, want %q", decision.Decision, KVIdentityDecisionUnsupported)
	}
	if decision.LiveKVReuse != false {
		t.Fatalf("policy must not allow live KV reuse: %#v", decision)
	}
}

func TestKVIdentityEnforcementRejectsAmbiguousReuseClaim(t *testing.T) {
	req := validKVIdentityRequest()
	req.Payload["request"].(map[string]any)["enable_live_reuse"] = "true"
	decision := EnforceKVIdentity(req)
	if decision.Accepted {
		t.Fatalf("expected unsupported ambiguous reuse decision")
	}
	if decision.Decision != KVIdentityDecisionUnsupported {
		t.Fatalf("decision = %q, want %q", decision.Decision, KVIdentityDecisionUnsupported)
	}
}

func TestKVIdentityEnforcementCountersRecordDecisions(t *testing.T) {
	counters := NewKVIdentityEnforcementCounters()
	counters.Record(KVIdentityEnforcementDecision{Decision: KVIdentityDecisionAccepted})
	counters.Record(KVIdentityEnforcementDecision{Decision: KVIdentityDecisionRejected})
	counters.Record(KVIdentityEnforcementDecision{Decision: KVIdentityDecisionMalformed})
	counters.Record(KVIdentityEnforcementDecision{Decision: KVIdentityDecisionUnsupported})
	counters.Record(KVIdentityEnforcementDecision{Decision: KVIdentityDecisionInternalError})
	snapshot := counters.Snapshot()
	if snapshot.Accepted != 1 || snapshot.Rejected != 1 || snapshot.Malformed != 1 || snapshot.UnsupportedLiveReuse != 1 || snapshot.InternalError != 1 {
		t.Fatalf("unexpected counter snapshot: %#v", snapshot)
	}
}
