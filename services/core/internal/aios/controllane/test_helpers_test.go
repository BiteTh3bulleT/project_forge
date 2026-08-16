package controllane

import (
	"context"
	"fmt"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/forgekernel"
	"forge/projectforge/services/core/internal/forgekernel/authproof"
)

type testForgeKAuthorizationPort struct{}

func (testForgeKAuthorizationPort) ResolveAuthorization(_ context.Context, req domain.SyscallRequest) (authproof.Proof, error) {
	def, ok := NewStaticActionRegistry().Get(req.Action)
	if !ok {
		return authproof.Proof{}, fmt.Errorf("action %q missing", req.Action)
	}
	const credential = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	service := authproof.PrincipalRecord{
		RecordID: "principal:forge.core", Version: "service_identity.v1", SubjectID: "forge.core", SubjectKind: "service",
		Source: domain.SourceSystem, Issuer: "forge.test", CredentialFingerprint: credential,
		Status: authproof.StatusActive, AuthenticatedAt: 1,
	}
	proof := authproof.Proof{
		EvidenceSnapshotID: "test-control-lane-authorization:v1",
		ServicePrincipal:   service,
		Origin: &authproof.PrincipalRecord{
			RecordID: "authn:" + req.Actor.ID, Version: "test_identity.v1", SubjectID: req.Actor.ID, SubjectKind: req.Actor.Kind,
			Source: req.Source, Issuer: "forge.test", CredentialFingerprint: credential, Status: authproof.StatusActive, AuthenticatedAt: 1,
		},
		Registry: authproof.RegistryRecord{
			RecordID: "action:" + string(req.Action), Version: "test_registry.v1", Authority: "forge_k.registry",
			Action: req.Action, Capability: def.Capability, TargetObjectType: def.TargetObjectType,
			Mutating: def.Mutating, MutationPolicy: authproof.MutationNever, AuthorizedMutating: false,
			SupportsDryRun: def.SupportsDryRun, ApprovalPossible: def.ApprovalPossible, JournalEventType: def.AuditEventName,
		},
		Capability: authproof.CapabilityRecord{
			RecordID: "grant:" + req.Actor.ID + ":" + def.Capability, Version: "test_capability.v1", Authority: "forge.capabilities",
			SubjectID: req.Actor.ID, SubjectKind: req.Actor.Kind, Source: req.Source, Action: req.Action,
			Capability: def.Capability, Scope: req.Scope, Status: authproof.StatusActive, GrantedAt: 1,
		},
		Approval: authproof.ApprovalRecord{
			PolicyRecordID: "approval-policy:test", PolicyVersion: "test_approval.v1", Authority: "forge.approvals",
			Status: authproof.ApprovalNotNeeded,
		},
	}
	if req.Action == domain.ActionCompileContext {
		proof.Registry.MutationPolicy = authproof.MutationRequestDependent
		proof.Registry.AuthorizedMutating = mergeCompileContextOptions(req.Payload).PersistSnapshot
	}
	return authproof.BuildProof(req, proof)
}

func processContextThroughForgeK(ctx context.Context, processor *Processor, req domain.SyscallRequest) (domain.SyscallResult, error) {
	if mergeCompileContextOptions(req.Payload).PersistSnapshot && req.IdempotencyKey == "" {
		req.IdempotencyKey = "test-context:" + req.ID
	}
	selection, err := forgekernel.SelectAuthority(string(forgekernel.ModeForgeK), processor, testForgeKAuthorizationPort{})
	if err != nil {
		return domain.SyscallResult{}, err
	}
	return selection.Processor.Process(ctx, req)
}

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
