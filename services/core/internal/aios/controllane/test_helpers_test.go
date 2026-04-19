package controllane

import (
	"context"

	"forge/projectforge/services/core/internal/aios/domain"
)

func validBaseRequest(action domain.SemanticActionType) domain.SyscallRequest {
	return domain.SyscallRequest{
		ID:     "req-1",
		Action: action,
		Actor: domain.ActorIdentity{
			ID:   "operator",
			Kind: string(domain.SourceUser),
		},
		Source:  domain.SourceUser,
		Scope:   domain.ForgeScope{WorkspaceID: "ws-main"},
		Payload: map[string]any{},
		Provenance: domain.Provenance{
			Actor:     "operator",
			ActorType: "user",
			Source:    "ui",
			TraceID:   "trace-1",
		},
		CorrelationID: "corr-1",
		TraceID:       "trace-1",
		RequestedAt:   1760000000000,
	}
}

func newTestKernel() (*Processor, *InMemorySemanticStore, *InMemoryAuditSink) {
	store := NewInMemorySemanticStore()
	auditSink := NewInMemoryAuditSink()
	kernel := NewProcessor(ProcessorOptions{
		Registry:     NewStaticActionRegistry(),
		Validator:    NewDeterministicValidator(),
		Capabilities: NewStaticCapabilityService(),
		ApprovalGate: NewStaticApprovalGate(),
		TxRunner:     NewInMemoryTransactionRunner(store),
		AuditSink:    auditSink,
		NowMillis:    func() int64 { return 1760000000000 },
	})
	return kernel, store, auditSink
}

func mustCreateNote(ctx context.Context, k *Processor, id, title string) {
	req := validBaseRequest(domain.ActionCreateNote)
	req.ID = "seed-" + id
	req.Payload = map[string]any{
		"id":      id,
		"type":    string(domain.NoteFact),
		"title":   title,
		"content": "seed content",
		"status":  string(domain.NoteActive),
	}
	_, _ = k.Process(ctx, req)
}
