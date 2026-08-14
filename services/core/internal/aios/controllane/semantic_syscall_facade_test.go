package controllane

import (
	"context"
	"testing"

	"forge/projectforge/services/core/internal/aios/domain"
)

func TestBuildSemanticSyscallFacadeNormalizesWriteEnvelope(t *testing.T) {
	reg := NewStaticActionRegistry()
	def, ok := reg.Get(domain.ActionCreateLink)
	if !ok {
		t.Fatal("missing action definition")
	}
	req := validBaseRequest(domain.ActionCreateLink)
	req.ID = "syscall-link"
	req.Scope = domain.ForgeScope{WorkspaceID: "ws-main", LaneID: "control.semantic", SelectedPaths: []string{"docs/a.md"}}
	req.IdempotencyKey = "idem-link"
	req.Payload = map[string]any{
		"type":     string(domain.LinkSupports),
		"sourceId": "note-b",
		"targetId": "note-a",
	}

	facade := BuildSemanticSyscallFacade(req, def)
	if facade.SchemaVersion != SemanticSyscallFacadeSchemaVersion {
		t.Fatalf("unexpected schema version %q", facade.SchemaVersion)
	}
	if facade.ExpectedEffect != "commit_semantic_link" || !facade.Mutating {
		t.Fatalf("unexpected effect/mutation: %#v", facade)
	}
	if facade.RequiredCapability != CapMemoryLinkCreate {
		t.Fatalf("unexpected capability %q", facade.RequiredCapability)
	}
	if len(facade.Refs) != 2 || facade.Refs[0] != "note-a" || facade.Refs[1] != "note-b" {
		t.Fatalf("refs not deterministically normalized: %#v", facade.Refs)
	}
	if facade.RollbackMetadata["required"] != true || facade.RollbackMetadata["strategy"] != "revert_journaled_commit" {
		t.Fatalf("unexpected rollback metadata: %#v", facade.RollbackMetadata)
	}
	if facade.AuthorityEffects["callsModelRuntime"] || facade.AuthorityEffects["executesGatewayTool"] || facade.AuthorityEffects["importsForgeK"] {
		t.Fatalf("facade must not claim external authority effects: %#v", facade.AuthorityEffects)
	}
}

func TestBuildSemanticSyscallFacadeRedactsUnsafeRefs(t *testing.T) {
	def, _ := NewStaticActionRegistry().Get(domain.ActionUpdateState)
	req := validBaseRequest(domain.ActionUpdateState)
	req.Payload = map[string]any{
		"key":         "k",
		"value":       map[string]any{"ok": true},
		"derivedFrom": []string{"note-a", "token=secret", "note-a"},
	}

	facade := BuildSemanticSyscallFacade(req, def)
	if len(facade.Refs) != 1 || facade.Refs[0] != "note-a" {
		t.Fatalf("expected only safe deduped refs, got %#v", facade.Refs)
	}
}

func TestProcessorAuditIncludesSemanticSyscallFacade(t *testing.T) {
	ctx := context.Background()
	kernel, _, auditSink := newTestKernel()
	req := validBaseRequest(domain.ActionCreateNote)
	req.ID = "syscall-note"
	req.Payload = map[string]any{
		"id":      "note-facade",
		"type":    string(domain.NoteFact),
		"title":   "Facade",
		"content": "semantic syscall facade evidence",
	}

	res, err := kernel.Process(ctx, req)
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected success: %+v", res)
	}
	if len(auditSink.Records) == 0 {
		t.Fatal("expected audit record")
	}
	facade := auditSink.Records[len(auditSink.Records)-1].SemanticSyscallEnvelope
	if facade["schemaVersion"] != SemanticSyscallFacadeSchemaVersion {
		t.Fatalf("missing facade schema: %#v", facade)
	}
	if facade["expectedEffect"] != "commit_memory_note" {
		t.Fatalf("unexpected expected effect: %#v", facade)
	}
	authority, ok := facade["authorityEffects"].(map[string]any)
	if !ok {
		t.Fatalf("missing authority effects: %#v", facade)
	}
	if authority["callsModelRuntime"] != false || authority["executesGatewayTool"] != false || authority["importsForgeKSimulator"] != false {
		t.Fatalf("facade must preserve authority boundaries: %#v", authority)
	}
}
