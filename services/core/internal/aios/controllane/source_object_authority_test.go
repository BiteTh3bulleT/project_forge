package controllane

import (
	"context"
	"testing"

	"forge/projectforge/services/core/internal/aios/domain"
)

func TestValidateSourceObjectAuthoritySucceedsWithoutSemanticMutation(t *testing.T) {
	ctx := context.Background()
	k, _, auditSink := newTestKernel()
	mustCreateNote(ctx, k, "note-source-a", "Source A")

	req := validSourceObjectAuthorityRequest()
	res, err := k.Process(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected source object authority success, got %#v", res)
	}
	if len(res.CommittedObjectIDs) != 0 {
		t.Fatalf("source object authority validation must not commit semantic objects, got %v", res.CommittedObjectIDs)
	}
	if res.StateSummary["passed"] != true {
		t.Fatalf("expected passed state summary, got %#v", res.StateSummary)
	}
	if res.StateSummary["memoryMutation"] != false ||
		res.StateSummary["runtimeMutation"] != false ||
		res.StateSummary["modelRuntimeCall"] != false ||
		res.StateSummary["evidenceAdmission"] != false ||
		res.StateSummary["contextCompilation"] != false ||
		res.StateSummary["liveAuthorityMigration"] != false {
		t.Fatalf("source object validation claimed forbidden effects: %#v", res.StateSummary)
	}
	resolved, ok := res.StateSummary["resolvedRefs"].([]map[string]string)
	if !ok || len(resolved) != 1 || resolved[0]["ref_id"] != "note-source-a" || resolved[0]["object_kind"] != "memory_note" {
		t.Fatalf("unexpected resolved refs: %#v", res.StateSummary["resolvedRefs"])
	}
	assertForgeKValidationContract(t, string(domain.ActionValidateSourceObject), res.StateSummary)
	nested, ok := res.StateSummary["sourceObjectAuthorityValidation"].(map[string]any)
	if !ok {
		t.Fatalf("missing nested sourceObjectAuthorityValidation: %#v", res.StateSummary)
	}
	assertForgeKValidationContract(t, string(domain.ActionValidateSourceObject), nested)
	if nested["decision"] != SourceObjectDecisionAccepted || nested["resolvedRefCount"] != 1 {
		t.Fatalf("unexpected nested decision: %#v", nested)
	}
	if len(auditSink.Records) == 0 {
		t.Fatal("expected audit record")
	}
	auditDecision := auditSink.Records[len(auditSink.Records)-1].SourceObjectAuthority
	if auditDecision["decision"] != SourceObjectDecisionAccepted || auditDecision["liveAuthorityMigration"] != false {
		t.Fatalf("audit missing source object authority decision: %#v", auditDecision)
	}
	assertForgeKValidationContract(t, string(domain.ActionValidateSourceObject), auditDecision)
}

func TestValidateSourceObjectAuthorityRejectsMissingObjectFailClosed(t *testing.T) {
	ctx := context.Background()
	k, store, auditSink := newTestKernel()
	req := validSourceObjectAuthorityRequest()
	req.ID = "source-object-missing"
	req.IdempotencyKey = "source-object-missing"
	req.Payload["refs"] = []any{
		map[string]any{"ref_type": "memory_note", "ref_id": "missing-note", "workspace_id": "ws-main"},
	}

	res, err := k.Process(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Success {
		t.Fatal("expected missing source object to fail")
	}
	if res.DeterministicErrCode != domain.ErrNotFound {
		t.Fatalf("expected not found, got %s", res.DeterministicErrCode)
	}
	if len(res.CommittedObjectIDs) != 0 {
		t.Fatalf("failed validation committed ids: %v", res.CommittedObjectIDs)
	}
	if _, ok := store.GetIdempotency(req.IdempotencyKey); ok {
		t.Fatal("failed source object validation must not persist idempotency state")
	}
	assertSourceObjectRejectedWithoutEffects(t, res, auditSink)
}

func TestValidateSourceObjectAuthorityRejectsWrongWorkspace(t *testing.T) {
	ctx := context.Background()
	k, _, auditSink := newTestKernel()
	req := validBaseRequest(domain.ActionCreateNote)
	req.ID = "seed-other-workspace-note"
	req.Scope = domain.ForgeScope{WorkspaceID: "ws-other"}
	req.Payload = map[string]any{
		"id":      "note-other-workspace",
		"type":    string(domain.NoteFact),
		"title":   "Other",
		"content": "Other workspace",
		"status":  string(domain.NoteActive),
	}
	if res, err := k.Process(ctx, req); err != nil || !res.Success {
		t.Fatalf("seed note failed: res=%#v err=%v", res, err)
	}

	check := validSourceObjectAuthorityRequest()
	check.ID = "source-object-wrong-workspace"
	check.Payload["refs"] = []any{
		map[string]any{"ref_type": "memory_note", "ref_id": "note-other-workspace", "workspace_id": "ws-main"},
	}
	res, err := k.Process(ctx, check)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Success {
		t.Fatal("expected wrong workspace to fail")
	}
	if res.DeterministicErrCode != domain.ErrInvalidScope {
		t.Fatalf("expected invalid scope, got %s", res.DeterministicErrCode)
	}
	assertSourceObjectRejectedWithoutEffects(t, res, auditSink)
}

func TestValidateSourceObjectAuthorityRejectsUnsupportedRefType(t *testing.T) {
	ctx := context.Background()
	k, _, auditSink := newTestKernel()
	req := validSourceObjectAuthorityRequest()
	req.ID = "source-object-unsupported-type"
	req.Payload["refs"] = []any{
		map[string]any{"ref_type": "runtime_manifest", "ref_id": "runtime-manifest-a", "workspace_id": "ws-main"},
	}

	res, err := k.Process(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Success {
		t.Fatal("expected unsupported ref type to fail closed")
	}
	if res.DeterministicErrCode != domain.ErrNotFound {
		t.Fatalf("expected not found, got %s", res.DeterministicErrCode)
	}
	assertSourceObjectRejectedWithoutEffects(t, res, auditSink)
}

func TestValidateSourceObjectAuthorityCapabilityDeniedForProposeOnlySource(t *testing.T) {
	ctx := context.Background()
	k, _, _ := newTestKernel()
	req := validSourceObjectAuthorityRequest()
	req.ID = "source-object-future-iris"
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

func validSourceObjectAuthorityRequest() domain.SyscallRequest {
	req := validBaseRequest(domain.ActionValidateSourceObject)
	req.ID = "source-object-pass"
	req.Payload = map[string]any{
		"workspace_id": "ws-main",
		"refs": []any{
			map[string]any{"ref_type": "memory_note", "ref_id": "note-source-a", "workspace_id": "ws-main"},
		},
	}
	return req
}

func assertSourceObjectRejectedWithoutEffects(t *testing.T, res domain.SyscallResult, auditSink *InMemoryAuditSink) {
	t.Helper()
	if res.StateSummary["memoryMutation"] != false ||
		res.StateSummary["runtimeMutation"] != false ||
		res.StateSummary["modelRuntimeCall"] != false ||
		res.StateSummary["evidenceAdmission"] != false ||
		res.StateSummary["contextCompilation"] != false ||
		res.StateSummary["liveAuthorityMigration"] != false {
		t.Fatalf("rejected source object validation claimed forbidden effects: %#v", res.StateSummary)
	}
	assertForgeKValidationContract(t, string(domain.ActionValidateSourceObject), res.StateSummary)
	nested, ok := res.StateSummary["sourceObjectAuthorityValidation"].(map[string]any)
	if !ok {
		t.Fatalf("missing nested sourceObjectAuthorityValidation summary: %#v", res.StateSummary)
	}
	assertForgeKValidationContract(t, string(domain.ActionValidateSourceObject), nested)
	if nested["accepted"] != false || nested["decision"] != SourceObjectDecisionRejected {
		t.Fatalf("expected rejected nested decision, got %#v", nested)
	}
	if len(auditSink.Records) == 0 {
		t.Fatal("expected rejected validation audit record")
	}
	auditRecord := auditSink.Records[len(auditSink.Records)-1]
	if auditRecord.Success {
		t.Fatalf("expected rejected audit record, got %#v", auditRecord)
	}
	auditDecision := auditRecord.SourceObjectAuthority
	if auditDecision == nil {
		t.Fatalf("audit missing source object authority summary: %#v", auditRecord)
	}
	assertForgeKValidationContract(t, string(domain.ActionValidateSourceObject), auditDecision)
	if auditDecision["accepted"] != false || auditDecision["decision"] != SourceObjectDecisionRejected {
		t.Fatalf("expected rejected audit decision, got %#v", auditDecision)
	}
}
