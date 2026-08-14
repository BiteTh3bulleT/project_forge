package authproof

import (
	"encoding/json"
	"errors"
	"testing"

	"forge/projectforge/services/core/internal/aios/domain"
)

const testDigest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func TestBuildAndVerifyProofBindsAllAuthorityRecords(t *testing.T) {
	req := testRequest()
	proof := mustBuildProof(t, req, testProof(req))
	if err := VerifyProof(req, proof); err != nil {
		t.Fatalf("verify proof: %v", err)
	}
	if proof.AuthorizationFingerprint == "" || proof.ServicePrincipal.RecordHash == "" || proof.Origin == nil || proof.Origin.RecordHash == "" {
		t.Fatalf("proof was not fully bound: %#v", proof)
	}
	if err := VerifyPlanBinding(proof, PlanBinding{
		Action: req.Action, Capability: proof.Registry.Capability,
		TargetObjectType: proof.Registry.TargetObjectType, Mutating: proof.Registry.Mutating,
		JournalEventType: proof.Registry.JournalEventType,
	}); err != nil {
		t.Fatalf("verify plan binding: %v", err)
	}
}

func TestVerifyProofRejectsRequestAndAuthorityTampering(t *testing.T) {
	req := testRequest()
	base := mustBuildProof(t, req, testProof(req))
	tests := []struct {
		name string
		edit func(*domain.SyscallRequest, *Proof)
	}{
		{"action", func(r *domain.SyscallRequest, _ *Proof) { r.Action = domain.ActionOpenLoop }},
		{"actor", func(r *domain.SyscallRequest, _ *Proof) { r.Actor.ID = "mallory" }},
		{"source", func(r *domain.SyscallRequest, _ *Proof) { r.Source = domain.SourceAdapter }},
		{"scope", func(r *domain.SyscallRequest, _ *Proof) { r.Scope.WorkspaceID = "other" }},
		{"selected_path", func(r *domain.SyscallRequest, _ *Proof) { r.Scope.SelectedPaths[0] = "/other" }},
		{"payload", func(r *domain.SyscallRequest, _ *Proof) { r.Payload["content"] = "tampered" }},
		{"provenance", func(r *domain.SyscallRequest, _ *Proof) { r.Provenance.Source = "tampered" }},
		{"required_capability", func(r *domain.SyscallRequest, _ *Proof) { r.RequiredCapability = "memory.loop.open" }},
		{"caller_metadata", func(r *domain.SyscallRequest, _ *Proof) { r.Metadata["callerClaim"] = "tampered" }},
		{"service_principal", func(_ *domain.SyscallRequest, p *Proof) { p.ServicePrincipal.RecordID = "principal:other" }},
		{"service_credential", func(_ *domain.SyscallRequest, p *Proof) {
			p.ServicePrincipal.CredentialFingerprint = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
		}},
		{"origin", func(_ *domain.SyscallRequest, p *Proof) { p.Origin.SubjectID = "mallory" }},
		{"registry", func(_ *domain.SyscallRequest, p *Proof) { p.Registry.Capability = "memory.loop.open" }},
		{"grant", func(_ *domain.SyscallRequest, p *Proof) { p.Capability.RecordID = "grant:other" }},
		{"grant_scope", func(_ *domain.SyscallRequest, p *Proof) { p.Capability.Scope.LaneID = "other" }},
		{"approval_policy", func(_ *domain.SyscallRequest, p *Proof) { p.Approval.PolicyVersion = "v2" }},
		{"aggregate", func(_ *domain.SyscallRequest, p *Proof) { p.AuthorizationFingerprint = testDigest }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			requestClone := cloneRequest(req)
			proofClone := cloneProof(t, base)
			tc.edit(&requestClone, &proofClone)
			if err := VerifyProof(requestClone, proofClone); err == nil {
				t.Fatal("tampered evidence unexpectedly verified")
			}
		})
	}
}

func TestBuildProofFailsClosedOnMissingEvidence(t *testing.T) {
	req := testRequest()
	tests := []struct {
		name string
		edit func(*Proof)
	}{
		{"snapshot", func(p *Proof) { p.EvidenceSnapshotID = "" }},
		{"service_record", func(p *Proof) { p.ServicePrincipal.RecordID = "" }},
		{"service_credential", func(p *Proof) { p.ServicePrincipal.CredentialFingerprint = "" }},
		{"origin", func(p *Proof) { p.Origin = nil }},
		{"registry", func(p *Proof) { p.Registry.RecordID = "" }},
		{"registry_capability", func(p *Proof) { p.Registry.Capability = "" }},
		{"grant", func(p *Proof) { p.Capability.RecordID = "" }},
		{"approval_policy", func(p *Proof) { p.Approval.PolicyRecordID = "" }},
		{"approval_status", func(p *Proof) { p.Approval.Status = "" }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			proof := testProof(req)
			tc.edit(&proof)
			if _, err := BuildProof(req, proof); !errors.Is(err, ErrInvalidProof) && !errors.Is(err, ErrProofMismatch) {
				t.Fatalf("error = %v, want closed proof failure", err)
			}
		})
	}
}

func TestRequestFingerprintIsDeterministicForMapOrderAndRetryTransport(t *testing.T) {
	left := testRequest()
	left.Payload = map[string]any{"z": map[string]any{"b": 2, "a": 1}, "a": "value"}
	left.Metadata = map[string]any{"z": 2, "a": map[string]any{"y": true, "x": false}}
	right := cloneRequest(left)
	right.Payload = map[string]any{"a": "value", "z": map[string]any{"a": 1, "b": 2}}
	right.Metadata = map[string]any{"a": map[string]any{"x": false, "y": true}, "z": 2}
	right.ID = "sys-retry"
	right.CorrelationID = "corr-retry"
	right.TraceID = "trace-retry"
	right.Provenance.TraceID = "trace-retry"
	right.RequestedAt++
	right.Metadata["forgeKIngressAuthority"] = true
	right.Metadata["kernelAuthorityOwner"] = "forge_k.kernel"
	right.Metadata["forgeKAuthorizationProof"] = testDigest

	leftHash, err := RequestFingerprint(left)
	if err != nil {
		t.Fatal(err)
	}
	rightHash, err := RequestFingerprint(right)
	if err != nil {
		t.Fatal(err)
	}
	if leftHash != rightHash {
		t.Fatalf("retry/map ordering changed semantic fingerprint: %s != %s", leftHash, rightHash)
	}
}

func TestSameAuthorizationRequiresSamePrincipalGrantAndApproval(t *testing.T) {
	originalReq := testRequest()
	original := mustBuildProof(t, originalReq, testProof(originalReq))
	retryReq := cloneRequest(originalReq)
	retryReq.ID = "sys-retry"
	retryReq.CorrelationID = "corr-retry"
	retryReq.TraceID = "trace-retry"
	retryReq.Provenance.TraceID = "trace-retry"
	retryReq.RequestedAt++
	retrySeed := testProof(originalReq)
	retry := mustBuildProof(t, retryReq, retrySeed)
	if err := SameAuthorization(original, retry); err != nil {
		t.Fatalf("same authority rejected: %v", err)
	}

	changed := testProof(retryReq)
	changed.Origin.RecordID = "authn:operator:new-session"
	changed = mustBuildProof(t, retryReq, changed)
	if err := SameAuthorization(original, changed); !errors.Is(err, ErrProofMismatch) {
		t.Fatalf("changed authenticated origin error = %v, want mismatch", err)
	}
}

func TestSystemAndInternalCallsBindServicePrincipalWithoutOrigin(t *testing.T) {
	for _, source := range []domain.ActionSource{domain.SourceSystem, domain.SourceInternal} {
		t.Run(string(source), func(t *testing.T) {
			req := testRequest()
			req.Source = source
			req.Actor = domain.ActorIdentity{ID: "forge.autonomy", Kind: "autonomy"}
			req.Provenance.Actor = req.Actor.ID
			req.Provenance.ActorType = "system"
			proof := testProof(req)
			proof.Origin = nil
			proof.Capability.SubjectID = proof.ServicePrincipal.SubjectID
			proof.Capability.SubjectKind = proof.ServicePrincipal.SubjectKind
			proof.Capability.Source = source
			bound := mustBuildProof(t, req, proof)
			if err := VerifyProof(req, bound); err != nil {
				t.Fatalf("verify system service proof: %v", err)
			}
		})
	}
}

func TestVerifyPlanBindingHandlesRequestDependentCompileAndCourtMutation(t *testing.T) {
	for _, persist := range []bool{false, true} {
		t.Run(map[bool]string{false: "compile_read_only", true: "compile_persisted"}[persist], func(t *testing.T) {
			req := testRequest()
			req.Action = domain.ActionCompileContext
			req.RequiredCapability = "context.compile"
			req.Payload = map[string]any{
				"query": "forge-k", "compileOptions": map[string]any{"persistSnapshot": persist},
			}
			proof := testProof(req)
			proof.Registry.Action = req.Action
			proof.Registry.Capability = req.RequiredCapability
			proof.Registry.TargetObjectType = "context_packet"
			proof.Registry.Mutating = false
			proof.Registry.MutationPolicy = MutationRequestDependent
			proof.Registry.AuthorizedMutating = persist
			proof.Registry.ApprovalPossible = false
			proof.Registry.JournalEventType = "semantic_syscall.compile_context"
			proof.Capability.Action = req.Action
			proof.Capability.Capability = req.RequiredCapability
			bound := mustBuildProof(t, req, proof)
			if err := VerifyPlanBinding(bound, PlanBinding{
				Action: req.Action, Capability: req.RequiredCapability, TargetObjectType: "context_packet",
				Mutating: persist, JournalEventType: "semantic_syscall.compile_context",
			}); err != nil {
				t.Fatalf("compile plan binding: %v", err)
			}
		})
	}

	t.Run("court_always_mutating", func(t *testing.T) {
		req := testRequest()
		req.Action = domain.ActionAdmitEvidence
		req.RequiredCapability = "court.evidence.admit"
		proof := testProof(req)
		proof.Registry.Action = req.Action
		proof.Registry.Capability = req.RequiredCapability
		proof.Registry.TargetObjectType = "court_exhibit_ruling"
		proof.Registry.Mutating = true
		proof.Registry.MutationPolicy = MutationAlways
		proof.Registry.AuthorizedMutating = true
		proof.Registry.JournalEventType = "semantic_syscall.admit_evidence"
		proof.Capability.Action = req.Action
		proof.Capability.Capability = req.RequiredCapability
		bound := mustBuildProof(t, req, proof)
		if err := VerifyPlanBinding(bound, PlanBinding{
			Action: req.Action, Capability: req.RequiredCapability, TargetObjectType: "court_exhibit_ruling",
			Mutating: true, JournalEventType: "semantic_syscall.admit_evidence",
		}); err != nil {
			t.Fatalf("court plan binding: %v", err)
		}
	})
}

func TestMutatingProposerRequiresSeparateDurableApproval(t *testing.T) {
	req := testRequest()
	req.Source = domain.SourceAdapter
	req.Actor = domain.ActorIdentity{ID: "adapter:librarian", Kind: "adapter"}
	req.Provenance.Actor = req.Actor.ID
	req.Provenance.ActorType = req.Actor.Kind
	proof := testProof(req)
	proof.Origin.SubjectID = req.Actor.ID
	proof.Origin.SubjectKind = req.Actor.Kind
	proof.Origin.Source = req.Source
	proof.Capability.SubjectID = req.Actor.ID
	proof.Capability.SubjectKind = req.Actor.Kind
	proof.Capability.Source = req.Source
	if _, err := BuildProof(req, proof); !errors.Is(err, ErrProofMismatch) {
		t.Fatalf("unapproved proposer mutation error = %v, want mismatch", err)
	}

	proof.Approval.Required = true
	proof.Approval.Status = ApprovalApproved
	proof.Approval.RequestID = "approval-request:1"
	proof.Approval.DecisionID = "approval-decision:1"
	proof.Approval.DecidedBy = "operator:reviewer"
	proof.Approval.DecisionAt = req.RequestedAt - 10
	proof.Approval.ExpiresAt = req.RequestedAt + 1000
	bound := mustBuildProof(t, req, proof)
	if err := VerifyProof(req, bound); err != nil {
		t.Fatalf("approved proposer proof: %v", err)
	}

	selfApproved := testProof(req)
	selfApproved.Origin = clonePrincipal(proof.Origin)
	selfApproved.Capability = proof.Capability
	selfApproved.Approval = proof.Approval
	selfApproved.Approval.DecidedBy = req.Actor.ID
	if _, err := BuildProof(req, selfApproved); !errors.Is(err, ErrProofMismatch) {
		t.Fatalf("self approval error = %v, want mismatch", err)
	}
}

func TestVerifyPlanBindingRejectsRegistryDrift(t *testing.T) {
	req := testRequest()
	proof := mustBuildProof(t, req, testProof(req))
	plan := PlanBinding{
		Action: req.Action, Capability: proof.Registry.Capability,
		TargetObjectType: proof.Registry.TargetObjectType, Mutating: proof.Registry.Mutating,
		JournalEventType: proof.Registry.JournalEventType,
	}
	plan.Capability = "memory.note.archive"
	if err := VerifyPlanBinding(proof, plan); !errors.Is(err, ErrProofMismatch) {
		t.Fatalf("plan drift error = %v, want mismatch", err)
	}
}

func testRequest() domain.SyscallRequest {
	return domain.SyscallRequest{
		ID: "sys-1", Action: domain.ActionCreateNote,
		Actor:         domain.ActorIdentity{ID: "operator", Kind: "user"},
		Source:        domain.SourceUser,
		Scope:         domain.ForgeScope{WorkspaceID: "ws-1", LaneID: "semantic", SelectedPaths: []string{"/notes"}},
		Payload:       map[string]any{"id": "note-1", "content": "hello"},
		Provenance:    domain.Provenance{Actor: "operator", ActorType: "user", Source: "desktop", TraceID: "trace-1"},
		CorrelationID: "corr-1", TraceID: "trace-1", IdempotencyKey: "idem-1",
		RequestedAt: 2000, RequiredCapability: "memory.note.create",
		Metadata: map[string]any{"callerClaim": "bound"},
	}
}

func testProof(req domain.SyscallRequest) Proof {
	origin := &PrincipalRecord{
		RecordID: "authn:operator:session-1", Version: "identity.v1",
		SubjectID: req.Actor.ID, SubjectKind: req.Actor.Kind, Source: req.Source,
		Issuer: "forge.api.bearer", CredentialFingerprint: testDigest,
		Status: StatusActive, AuthenticatedAt: req.RequestedAt - 100, ExpiresAt: req.RequestedAt + 1000,
	}
	return Proof{
		EvidenceSnapshotID: "authorization-snapshot:1",
		ServicePrincipal: PrincipalRecord{
			RecordID: "principal:forge.core", Version: "service_identity.v1",
			SubjectID: "forge.core", SubjectKind: "service", Source: domain.SourceSystem,
			Issuer: "forge.bootstrap", CredentialFingerprint: testDigest,
			Status: StatusActive, AuthenticatedAt: 1,
		},
		Origin: origin,
		Registry: RegistryRecord{
			RecordID: "action:CREATE_NOTE", Version: "registry.v1", Authority: "forge_k.registry",
			Action: req.Action, Capability: "memory.note.create", TargetObjectType: "memory_note",
			Mutating: true, MutationPolicy: MutationAlways, AuthorizedMutating: true,
			SupportsDryRun: true, ApprovalPossible: true,
			JournalEventType: "semantic_syscall_committed",
		},
		Capability: CapabilityRecord{
			RecordID: "grant:operator:memory.note.create", Version: "capability.v1", Authority: "forge.capabilities",
			SubjectID: req.Actor.ID, SubjectKind: req.Actor.Kind, Source: req.Source,
			Action: req.Action, Capability: "memory.note.create", Scope: req.Scope,
			Status: StatusActive, GrantedAt: req.RequestedAt - 100, ExpiresAt: req.RequestedAt + 1000,
		},
		Approval: ApprovalRecord{
			PolicyRecordID: "approval-policy:user", PolicyVersion: "approval.v1", Authority: "forge.approvals",
			Required: false, Status: ApprovalNotNeeded,
		},
	}
}

func mustBuildProof(t *testing.T, req domain.SyscallRequest, proof Proof) Proof {
	t.Helper()
	bound, err := BuildProof(req, proof)
	if err != nil {
		t.Fatalf("build proof: %v", err)
	}
	return bound
}

func cloneRequest(req domain.SyscallRequest) domain.SyscallRequest {
	var out domain.SyscallRequest
	raw, _ := json.Marshal(req)
	_ = json.Unmarshal(raw, &out)
	return out
}

func cloneProof(t *testing.T, proof Proof) Proof {
	t.Helper()
	var out Proof
	raw, err := json.Marshal(proof)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func clonePrincipal(principal *PrincipalRecord) *PrincipalRecord {
	if principal == nil {
		return nil
	}
	clone := *principal
	return &clone
}
