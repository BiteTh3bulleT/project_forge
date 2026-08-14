package controllane

import (
	"context"
	"errors"
	"testing"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/forgekernel"
	"forge/projectforge/services/core/internal/forgekernel/authproof"
	"forge/projectforge/services/core/internal/forgekernel/commitproof"
	"forge/projectforge/services/core/internal/store"
)

func TestProductionAuthorizationRequiresConstructedForgeCorePrincipal(t *testing.T) {
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	for _, principal := range []*ForgeCoreServicePrincipal{nil, {}} {
		if _, err := NewProductionAuthorizationService(ProductionAuthorizationOptions{
			Registry: NewStaticActionRegistry(), DB: st.DB, ServicePrincipal: principal,
		}); !errors.Is(err, ErrInvalidServicePrincipal) {
			t.Fatalf("principal=%#v error=%v", principal, err)
		}
	}
	first := NewForgeCoreServicePrincipal()
	second := NewForgeCoreServicePrincipal()
	if first == second || first.guard == second.guard || !validForgeCorePrincipal(first) || !validForgeCorePrincipal(second) {
		t.Fatal("service principal construction did not produce guarded unique instances")
	}
}

func TestProductionAuthorizationBindsServicePrincipalForSystemTaxonomy(t *testing.T) {
	svc, _ := newProductionAuthorizationHarness(t)
	req := productionAuthorizationRequest(domain.SourceInternal)
	req.Actor = domain.ActorIdentity{ID: "forge.autonomy", Kind: "autonomy"}
	req.Provenance = domain.Provenance{Actor: req.Actor.ID, ActorType: "system", Source: "maintenance"}
	proof, err := svc.ResolveAuthorization(context.Background(), req)
	if err != nil {
		t.Fatalf("resolve internal authorization: %v", err)
	}
	if proof.Origin != nil || proof.Capability.SubjectID != "forge.core" || proof.ServicePrincipal.RecordID != forgeCorePrincipalRecordID {
		t.Fatalf("internal call did not bind service principal: %#v", proof)
	}
	if err := authproof.VerifyProof(req, proof); err != nil {
		t.Fatalf("verify internal authorization: %v", err)
	}
}

func TestProductionAuthorizationRequiresTrustedOriginAndRejectsSourceSpoof(t *testing.T) {
	svc, _ := newProductionAuthorizationHarness(t)
	req := productionAuthorizationRequest(domain.SourceUser)
	if _, err := svc.ResolveAuthorization(context.Background(), req); !errors.Is(err, ErrAuthorizationDenied) {
		t.Fatalf("missing origin error=%v", err)
	}
	origin := productionTestOrigin(req.Actor, req.Source, "session-1")
	ctx := authproof.WithTrustedOrigin(context.Background(), origin)
	proof, err := svc.ResolveAuthorization(ctx, req)
	if err != nil {
		t.Fatalf("resolve user authorization: %v", err)
	}
	if proof.Origin == nil || proof.Origin.RecordID != origin.RecordID || proof.Capability.SubjectID != req.Actor.ID {
		t.Fatalf("user origin not bound: %#v", proof)
	}

	spoofed := req
	spoofed.Actor.ID = "mallory"
	spoofed.Provenance.Actor = "mallory"
	if _, err := svc.ResolveAuthorization(ctx, spoofed); err == nil {
		t.Fatal("caller actor spoof inherited trusted origin")
	}
	testSource := req
	testSource.Source = domain.SourceTest
	if _, err := svc.ResolveAuthorization(context.Background(), testSource); !errors.Is(err, ErrAuthorizationDenied) {
		t.Fatalf("production SourceTest error=%v", err)
	}
}

func TestProductionAuthorizationUsesExplicitAdapterPolicyAndDurableApproval(t *testing.T) {
	svc, st := newProductionAuthorizationHarness(t)
	req := productionAuthorizationRequest(domain.SourceAdapter)
	req.Actor = domain.ActorIdentity{ID: "adapter:librarian", Kind: "adapter"}
	req.Provenance.Actor = req.Actor.ID
	req.Provenance.ActorType = req.Actor.Kind
	ctx := authproof.WithTrustedOrigin(context.Background(), productionTestOrigin(req.Actor, req.Source, "adapter-session"))

	denied := req
	denied.Action = domain.ActionArchiveNote
	denied.RequiredCapability = CapMemoryNoteArchive
	if _, err := svc.ResolveAuthorization(ctx, denied); !errors.Is(err, ErrAuthorizationDenied) {
		t.Fatalf("ungranted adapter action error=%v", err)
	}
	if _, err := svc.ResolveAuthorization(ctx, req); !errors.Is(err, ErrApprovalProofRequired) {
		t.Fatalf("missing adapter approval error=%v", err)
	}

	fingerprint, err := authproof.RequestFingerprint(req)
	if err != nil {
		t.Fatal(err)
	}
	approvalID := seedAuthorizationApproval(t, st, req, fingerprint)
	req.Metadata["approvalRequestId"] = approvalID
	proof, err := svc.ResolveAuthorization(ctx, req)
	if err != nil {
		t.Fatalf("resolve approved adapter mutation: %v", err)
	}
	if !proof.Approval.Required || proof.Approval.Status != authproof.ApprovalApproved || proof.Approval.DecisionID == "" {
		t.Fatalf("durable approval not bound: %#v", proof.Approval)
	}

	retry := req
	retry.ID = "sys-auth-retry"
	retry.CorrelationID = "corr-auth-retry"
	retry.TraceID = "trace-auth-retry"
	retry.Provenance.TraceID = retry.TraceID
	retry.RequestedAt++
	retryProof, err := svc.ResolveAuthorization(ctx, retry)
	if err != nil {
		t.Fatalf("resolve approved retry: %v", err)
	}
	if err := authproof.SameAuthorization(proof, retryProof); err != nil {
		t.Fatalf("retry did not preserve authorization: %v", err)
	}
}

func TestProductionAuthorizationCompileMutationPolicy(t *testing.T) {
	svc, _ := newProductionAuthorizationHarness(t)
	for _, persist := range []bool{false, true} {
		req := productionAuthorizationRequest(domain.SourceInternal)
		req.Action = domain.ActionCompileContext
		req.RequiredCapability = CapContextCompile
		req.Payload = map[string]any{"query": "forge", "compileOptions": map[string]any{"persistSnapshot": persist}}
		proof, err := svc.ResolveAuthorization(context.Background(), req)
		if err != nil {
			t.Fatalf("persist=%v resolve: %v", persist, err)
		}
		if proof.Registry.MutationPolicy != authproof.MutationRequestDependent || proof.Registry.AuthorizedMutating != persist {
			t.Fatalf("persist=%v registry=%#v", persist, proof.Registry)
		}
	}
}

func TestProductionAuditOutboxBindsExactRequestAndAuthorizationProof(t *testing.T) {
	svc, st := newProductionAuthorizationHarness(t)
	req := productionAuthorizationRequest(domain.SourceUser)
	req.Metadata["forgeKIngressAuthority"] = true
	req.Metadata["kernelAuthorityOwner"] = forgekernel.AuthorityOwnerForgeK
	req.Metadata["durableCommitAdapter"] = forgekernel.DurableCommitAdapter
	ctx := authproof.WithTrustedOrigin(context.Background(), productionTestOrigin(req.Actor, req.Source, "session-a"))
	proof, err := svc.ResolveAuthorization(ctx, req)
	if err != nil {
		t.Fatalf("resolve authorization: %v", err)
	}
	req.Metadata["forgeKAuthorizationProof"] = proof.AuthorizationFingerprint
	requestFingerprint, err := commitproof.FingerprintRequest(req)
	if err != nil {
		t.Fatalf("fingerprint request: %v", err)
	}
	rec := AuditOutboxRecord{
		ID: req.ID + ":audit_outbox", SyscallID: req.ID, RequestFingerprint: requestFingerprint,
		Action: req.Action, WorkspaceID: req.Scope.WorkspaceID, LaneID: req.Scope.LaneID,
		CorrelationID: req.CorrelationID, TraceID: req.TraceID, Success: true,
		Result:  domain.SyscallResult{Success: true, Action: req.Action, RequestID: req.ID},
		Request: req, AuthorizationProof: proof, CreatedAt: req.RequestedAt,
		CommittedBy: forgekernel.AuthorityOwnerForgeK,
		Receipt: commitproof.CommitReceipt{
			Version: commitproof.CommitReceiptVersion, RequestFingerprint: requestFingerprint,
			AuditOutboxID: req.ID + ":audit_outbox",
		},
	}
	if err := validateAuditOutboxRecord(rec); err != nil {
		t.Fatalf("valid production audit intent rejected: %v", err)
	}
	for name, semantic := range map[string]integrityStore{
		"memory": NewInMemorySemanticStore(),
		"sqlite": NewSQLiteSemanticStore(st.DB),
	} {
		if err := semantic.CreateAuditOutbox(rec); err != nil {
			t.Fatalf("%s create audit intent: %v", name, err)
		}
		stored, ok := semantic.GetAuditOutbox(rec.ID)
		if !ok || stored.Request.ID != req.ID || stored.AuthorizationProof.AuthorizationFingerprint != proof.AuthorizationFingerprint {
			t.Fatalf("%s lost self-verifying audit evidence: %#v", name, stored)
		}
	}

	otherReq := req
	otherReq.Actor = domain.ActorIdentity{ID: "other-operator", Kind: "user"}
	otherReq.Provenance.Actor = otherReq.Actor.ID
	otherReq.Metadata = make(map[string]any, len(req.Metadata))
	for key, value := range req.Metadata {
		otherReq.Metadata[key] = value
	}
	otherCtx := authproof.WithTrustedOrigin(context.Background(), productionTestOrigin(otherReq.Actor, otherReq.Source, "session-b"))
	delete(otherReq.Metadata, "forgeKAuthorizationProof")
	otherProof, err := svc.ResolveAuthorization(otherCtx, otherReq)
	if err != nil {
		t.Fatalf("resolve second authorization: %v", err)
	}

	tests := map[string]func(*AuditOutboxRecord){
		"swapped proof": func(candidate *AuditOutboxRecord) { candidate.AuthorizationProof = otherProof },
		"tampered payload": func(candidate *AuditOutboxRecord) {
			candidate.Request.Payload["content"] = "tampered"
		},
		"tampered scope row":           func(candidate *AuditOutboxRecord) { candidate.WorkspaceID = "ws-other" },
		"tampered receipt fingerprint": func(candidate *AuditOutboxRecord) { candidate.Receipt.RequestFingerprint = "sha256:wrong" },
		"legacy committer downgrade":   func(candidate *AuditOutboxRecord) { candidate.CommittedBy = "forge_kernel" },
		"stripped kernel owner": func(candidate *AuditOutboxRecord) {
			delete(candidate.Request.Metadata, "kernelAuthorityOwner")
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			candidate := cloneAuditOutboxRecord(rec)
			mutate(&candidate)
			if err := validateAuditOutboxRecord(candidate); !errors.Is(err, ErrInvalidAuditOutboxRecord) {
				t.Fatalf("tampered audit intent error=%v", err)
			}
		})
	}
}

func newProductionAuthorizationHarness(t *testing.T) (*ProductionAuthorizationService, *store.Store) {
	t.Helper()
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	svc, err := NewProductionAuthorizationService(ProductionAuthorizationOptions{
		Registry: NewStaticActionRegistry(), DB: st.DB, ServicePrincipal: NewForgeCoreServicePrincipal(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return svc, st
}

func productionAuthorizationRequest(source domain.ActionSource) domain.SyscallRequest {
	return domain.SyscallRequest{
		ID: "sys-auth", Action: domain.ActionCreateNote,
		Actor: domain.ActorIdentity{ID: "operator", Kind: "user"}, Source: source,
		Scope:         domain.ForgeScope{WorkspaceID: "ws-auth", LaneID: "control.semantic"},
		Payload:       map[string]any{"id": "note-auth", "type": string(domain.NoteFact), "title": "Auth", "content": "proof"},
		Provenance:    domain.Provenance{Actor: "operator", ActorType: "user", Source: "test", TraceID: "trace-auth"},
		CorrelationID: "corr-auth", TraceID: "trace-auth", IdempotencyKey: "idem-auth",
		RequestedAt: 1760000000000, RequiredCapability: CapMemoryNoteCreate,
		Metadata: map[string]any{},
	}
}

func productionTestOrigin(actor domain.ActorIdentity, source domain.ActionSource, session string) authproof.PrincipalRecord {
	return authproof.PrincipalRecord{
		RecordID: "test-origin:" + session, Version: "test-origin.v1",
		SubjectID: actor.ID, SubjectKind: actor.Kind, Source: source, Issuer: "test.trusted",
		CredentialFingerprint: authproof.CredentialFingerprint(session), Status: authproof.StatusActive, AuthenticatedAt: 1,
	}
}

func seedAuthorizationApproval(t *testing.T, st *store.Store, req domain.SyscallRequest, fingerprint string) int64 {
	t.Helper()
	if _, err := st.DB.Exec(`
INSERT INTO jobs(id,created_at,updated_at,title,requested_action,target_adapter,initiating_source,
 execution_boundary,risk_class,status,approval_status,write_intent,metadata_json)
VALUES('job-auth',1,1,'auth','CREATE_NOTE','adapter','adapter','write','high','awaiting_approval','pending',1,'{}')`); err != nil {
		t.Fatal(err)
	}
	scopeJSON := `{"authorizationRequestFingerprint":"` + fingerprint + `"}`
	result, err := st.DB.Exec(`
INSERT INTO approval_requests(job_id,created_at,status,requested_action,risk_class,requested_adapter,
 write_intent,scope_snapshot_json,request_summary,expires_at)
VALUES('job-auth',1,'resolved',?,'high','adapter',1,?,'auth',?)`, string(req.Action), scopeJSON, req.RequestedAt+1000)
	if err != nil {
		t.Fatal(err)
	}
	requestID, _ := result.LastInsertId()
	if _, err := st.DB.Exec(`INSERT INTO approval_decisions(request_id,created_at,actor,decision,note) VALUES(?,?,?,?,?)`,
		requestID, req.RequestedAt-10, "operator:reviewer", "approved", "approved"); err != nil {
		t.Fatal(err)
	}
	return requestID
}
