package controllane

import (
	"context"
	"testing"

	"forge/projectforge/services/core/internal/aios/domain"
)

func TestRegistryIncludesStarterActions(t *testing.T) {
	reg := NewStaticActionRegistry()
	required := []domain.SemanticActionType{
		domain.ActionCreateNote,
		domain.ActionCreateLink,
		domain.ActionUpdateState,
		domain.ActionOpenLoop,
		domain.ActionCloseLoop,
		domain.ActionMarkSuperseded,
		domain.ActionRegisterContradict,
		domain.ActionDeriveModel,
		domain.ActionArchiveNote,
		domain.ActionCompileContext,
	}
	for _, action := range required {
		def, ok := reg.Get(action)
		if !ok {
			t.Fatalf("missing action %s in registry", action)
		}
		if def.Capability == "" {
			t.Fatalf("missing capability for action %s", action)
		}
		if def.AuditEventName == "" {
			t.Fatalf("missing audit event for action %s", action)
		}
	}
}

func TestUnsupportedActionRejectedDeterministically(t *testing.T) {
	store := NewInMemorySemanticStore()
	tx := NewInMemoryTransactionRunner(store)
	auditSink := NewInMemoryAuditSink()
	kernel := NewProcessor(ProcessorOptions{
		Registry:  NewStaticActionRegistry(),
		Validator: NewDeterministicValidator(),
		TxRunner:  tx,
		AuditSink: auditSink,
	})

	req := validBaseRequest(domain.SemanticActionType("NOPE_ACTION"))
	res, err := kernel.Process(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Success {
		t.Fatalf("expected unsupported action to fail")
	}
	if res.DeterministicErrCode != domain.ErrUnsupportedAction {
		t.Fatalf("expected ErrUnsupportedAction, got %s", res.DeterministicErrCode)
	}
}
