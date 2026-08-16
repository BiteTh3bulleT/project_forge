package commitproof

import (
	"errors"
	"testing"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/forgekernel/court"
	forgejournal "forge/projectforge/services/core/internal/forgekernel/journal"
	"forge/projectforge/services/core/internal/forgekernel/semanticdiff"
)

func TestRequestFingerprintAndPlanSealAreMapOrderStable(t *testing.T) {
	reqA := testRequest()
	reqA.Payload = map[string]any{
		"title":  "Forge",
		"nested": map[string]any{"z": true, "a": float64(2)},
	}
	reqA.Metadata = map[string]any{"second": "b", "first": "a"}
	reqB := testRequest()
	reqB.Payload = map[string]any{
		"nested": map[string]any{"a": float64(2), "z": true},
		"title":  "Forge",
	}
	reqB.Metadata = map[string]any{"first": "a", "second": "b"}

	fingerprintA, err := FingerprintRequest(reqA)
	if err != nil {
		t.Fatalf("fingerprint request A: %v", err)
	}
	fingerprintB, err := FingerprintRequest(reqB)
	if err != nil {
		t.Fatalf("fingerprint request B: %v", err)
	}
	if fingerprintA != fingerprintB {
		t.Fatalf("map insertion order changed request fingerprint:\nA=%s\nB=%s", fingerprintA, fingerprintB)
	}

	planA := testPlan()
	planA.Details = map[string]any{"write": map[string]any{"z": 2, "a": 1}, "kind": "note"}
	planA.ExpectedObjectIDs = []string{"note-2", "note-1"}
	planA.ExpectedProvenanceIDs = []string{"prov-2", "prov-1"}
	planB := testPlan()
	planB.Details = map[string]any{"kind": "note", "write": map[string]any{"a": 1, "z": 2}}
	planB.ExpectedObjectIDs = []string{"note-1", "note-2"}
	planB.ExpectedProvenanceIDs = []string{"prov-1", "prov-2"}
	// Journal v1 has exactly one provenance identity; use the same set-order
	// check on object ids and keep provenance singular.
	planA.ExpectedProvenanceIDs = []string{"prov-1"}
	planB.ExpectedProvenanceIDs = []string{"prov-1"}
	planA = mustBindTestPlan(t, reqA, planA)
	planB = mustBindTestPlan(t, reqB, planB)
	sealA, err := SealPreparedPlan(reqA, planA)
	if err != nil {
		t.Fatalf("seal plan A: %v", err)
	}
	sealB, err := SealPreparedPlan(reqB, planB)
	if err != nil {
		t.Fatalf("seal plan B: %v", err)
	}
	if sealA != sealB {
		t.Fatalf("map or evidence-set order changed seal:\nA=%+v\nB=%+v", sealA, sealB)
	}
}

func TestVerifyPreparedPlanRejectsRequestAndPlanTampering(t *testing.T) {
	req := testRequest()
	plan := mustBindTestPlan(t, req, testPlan())
	seal, err := SealPreparedPlan(req, plan)
	if err != nil {
		t.Fatalf("seal prepared plan: %v", err)
	}
	if err := VerifyPreparedPlan(req, plan, seal); err != nil {
		t.Fatalf("verify original prepared plan: %v", err)
	}

	tests := []struct {
		name string
		edit func(*domain.SyscallRequest, *PreparedPlan)
	}{
		{"action", func(r *domain.SyscallRequest, _ *PreparedPlan) { r.Action = domain.ActionOpenLoop }},
		{"scope", func(r *domain.SyscallRequest, _ *PreparedPlan) { r.Scope.WorkspaceID = "ws-other" }},
		{"actor", func(r *domain.SyscallRequest, _ *PreparedPlan) { r.Actor.ID = "operator-other" }},
		{"payload", func(r *domain.SyscallRequest, _ *PreparedPlan) { r.Payload["title"] = "tampered" }},
		{"provenance", func(r *domain.SyscallRequest, _ *PreparedPlan) { r.Provenance.Source = "tampered" }},
		{"plan", func(_ *domain.SyscallRequest, p *PreparedPlan) { p.Details["write"] = "tampered" }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			changedReq := cloneTestRequest(req)
			changedPlan := cloneTestPlan(plan)
			tc.edit(&changedReq, &changedPlan)
			if err := VerifyPreparedPlan(changedReq, changedPlan, seal); !errors.Is(err, ErrSealMismatch) && !errors.Is(err, ErrInvalidPreparedPlan) {
				t.Fatalf("tampering error = %v, want seal/plan failure", err)
			}
		})
	}
}

func TestValidateCommitReceiptRequiresCompleteEvidence(t *testing.T) {
	req := testRequest()
	plan := mustBindTestPlan(t, req, testPlan())
	seal, err := SealPreparedPlan(req, plan)
	if err != nil {
		t.Fatalf("seal prepared plan: %v", err)
	}
	receipt := testReceipt(t, req, seal)
	result := testResult(req, receipt.ObjectIDs)
	if err := ValidateCommitReceipt(req, plan, seal, receipt, result); err != nil {
		t.Fatalf("valid receipt rejected: %v", err)
	}

	tests := []struct {
		name string
		edit func(*CommitReceipt)
	}{
		{"version", func(r *CommitReceipt) { r.Version = "" }},
		{"request fingerprint", func(r *CommitReceipt) { r.RequestFingerprint = "" }},
		{"plan seal", func(r *CommitReceipt) { r.PreparedPlanSeal = "" }},
		{"transaction", func(r *CommitReceipt) { r.TransactionID = "" }},
		{"journal id", func(r *CommitReceipt) { r.JournalEventID = "" }},
		{"journal hash", func(r *CommitReceipt) { r.JournalEventHash = "" }},
		{"objects", func(r *CommitReceipt) { r.ObjectIDs = nil }},
		{"provenance", func(r *CommitReceipt) { r.ProvenanceIDs = nil }},
		{"audit outbox", func(r *CommitReceipt) { r.AuditOutboxID = "" }},
		{"idempotency", func(r *CommitReceipt) { r.IdempotencyFingerprint = "" }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			changed := receipt
			tc.edit(&changed)
			if err := ValidateCommitReceipt(req, plan, seal, changed, result); err == nil {
				t.Fatal("missing receipt evidence was accepted")
			}
		})
	}
}

func TestValidateCommitReceiptRejectsInconsistentEvidence(t *testing.T) {
	req := testRequest()
	plan := mustBindTestPlan(t, req, testPlan())
	seal, err := SealPreparedPlan(req, plan)
	if err != nil {
		t.Fatalf("seal prepared plan: %v", err)
	}
	receipt := testReceipt(t, req, seal)
	result := testResult(req, receipt.ObjectIDs)
	otherDigest, err := digest("other", map[string]any{"different": true})
	if err != nil {
		t.Fatalf("make other digest: %v", err)
	}

	tests := []struct {
		name string
		edit func(*CommitReceipt)
	}{
		{"request fingerprint", func(r *CommitReceipt) { r.RequestFingerprint = otherDigest }},
		{"plan seal", func(r *CommitReceipt) { r.PreparedPlanSeal = otherDigest }},
		{"idempotency fingerprint", func(r *CommitReceipt) { r.IdempotencyFingerprint = otherDigest }},
		{"object ids", func(r *CommitReceipt) { r.ObjectIDs = []string{"note-other"} }},
		{"provenance ids", func(r *CommitReceipt) { r.ProvenanceIDs = []string{"prov-other"} }},
		{"duplicate objects", func(r *CommitReceipt) { r.ObjectIDs = []string{"note-1", "note-1"} }},
		{"malformed journal hash", func(r *CommitReceipt) { r.JournalEventHash = "sha256:not-a-hash" }},
		{"transaction id", func(r *CommitReceipt) { r.TransactionID = "tx-other" }},
		{"journal event id", func(r *CommitReceipt) { r.JournalEventID = "journal-other" }},
		{"audit outbox id", func(r *CommitReceipt) { r.AuditOutboxID = "outbox-other" }},
		{"journal entry hash", func(r *CommitReceipt) { r.JournalEntry.Hash = otherDigest }},
		{"journal entry event", func(r *CommitReceipt) {
			r.JournalEntry.EventID = "event-other"
			rehashReceiptEntry(t, r)
		}},
		{"journal entry payload", func(r *CommitReceipt) {
			r.JournalEntry.PayloadHash = otherDigest
			rehashReceiptEntry(t, r)
		}},
		{"journal entry provenance", func(r *CommitReceipt) {
			r.JournalEntry.ProvenanceID = "prov-other"
			rehashReceiptEntry(t, r)
		}},
		{"journal entry scope", func(r *CommitReceipt) {
			r.JournalEntry.WorkspaceID = "ws-other"
			rehashReceiptEntry(t, r)
		}},
		{"journal entry selected paths", func(r *CommitReceipt) {
			r.JournalEntry.SelectedPaths = []string{"/other"}
			rehashReceiptEntry(t, r)
		}},
		{"journal entry committed by", func(r *CommitReceipt) {
			r.JournalEntry.CommittedBy = "other.kernel"
			rehashReceiptEntry(t, r)
		}},
		{"journal entry sequence shape", func(r *CommitReceipt) {
			r.JournalEntry.Sequence = 2
			r.JournalEntry.PriorHash = ""
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			changed := receipt
			tc.edit(&changed)
			if err := ValidateCommitReceipt(req, plan, seal, changed, result); err == nil {
				t.Fatal("inconsistent receipt evidence was accepted")
			}
		})
	}
}

func TestBindPreparedPlanOwnsJournalSchemaAfterFinalRequest(t *testing.T) {
	req := testRequest()
	req.Metadata[court.MetadataDecisionKey] = court.Decision{
		Action: domain.ActionAdmitEvidence, Decision: court.DecisionAdmitted, PolicyVersion: court.PolicyVersion,
	}
	plan := mustBindTestPlan(t, req, testPlan())
	if plan.ExpectedTransactionID != req.ID+":transaction" || plan.ExpectedJournalEventID != req.ID+":journal_event" ||
		plan.ExpectedAuditOutboxID != req.ID+":audit_outbox" || plan.ExpectedJournalSource != JournalSource ||
		plan.ExpectedJournalCommittedBy != "forge_k.kernel" {
		t.Fatalf("binder did not derive typed journal identity: %#v", plan)
	}
	if !validDigest(plan.ExpectedJournalPayloadHash) || !validDigest(plan.ExpectedJournalProvenanceHash) || !validDigest(plan.ExpectedJournalMetadataHash) {
		t.Fatalf("binder did not derive journal content hashes: %#v", plan)
	}
	payload, err := BuildJournalPayload(req, plan)
	if err != nil {
		t.Fatalf("build journal payload: %v", err)
	}
	if _, circular := payload["preparedPlanSeal"]; circular {
		t.Fatalf("journal payload contains circular plan seal: %#v", payload)
	}
	if payload["transactionId"] != plan.ExpectedTransactionID || payload["auditOutboxId"] != plan.ExpectedAuditOutboxID {
		t.Fatalf("journal payload differs from typed plan: %#v", payload)
	}

	tampered := plan
	tampered.ExpectedJournalPayloadHash = "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	if _, err := BindPreparedPlan(req, tampered); !errors.Is(err, ErrInvalidPreparedPlan) {
		t.Fatalf("binder repaired tampered content hash: %v", err)
	}
}

func TestValidateCommitReceiptRejectsReportedObjectMismatch(t *testing.T) {
	req := testRequest()
	plan := mustBindTestPlan(t, req, testPlan())
	seal, err := SealPreparedPlan(req, plan)
	if err != nil {
		t.Fatalf("seal prepared plan: %v", err)
	}
	receipt := testReceipt(t, req, seal)
	result := testResult(req, []string{"note-other"})
	if err := ValidateCommitReceipt(req, plan, seal, receipt, result); !errors.Is(err, ErrReceiptMismatch) {
		t.Fatalf("reported-object mismatch error = %v, want ErrReceiptMismatch", err)
	}
}

func TestIdempotencyFingerprintExcludesOnlyDerivedKernelDecisions(t *testing.T) {
	before := testRequest()
	after := cloneTestRequest(before)
	after.Metadata[court.MetadataDecisionKey] = court.Decision{
		Action: domain.ActionAdmitEvidence, CaseID: "case-1", RulingID: "ruling-1", Decision: court.DecisionAdmitted,
	}

	beforeFingerprint, err := IdempotencyFingerprint(before)
	if err != nil {
		t.Fatalf("fingerprint before Court decision: %v", err)
	}
	afterFingerprint, err := IdempotencyFingerprint(after)
	if err != nil {
		t.Fatalf("fingerprint after Court decision: %v", err)
	}
	if beforeFingerprint != afterFingerprint {
		t.Fatalf("deterministic Court metadata changed idempotency identity:\nbefore=%s\nafter=%s", beforeFingerprint, afterFingerprint)
	}
	semanticAfter := cloneTestRequest(after)
	semanticAfter.Metadata[semanticdiff.MetadataDecisionKey] = map[string]any{
		"operatorVersion": semanticdiff.OperatorVersion,
		"contentHash":     "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}
	semanticFingerprint, err := IdempotencyFingerprint(semanticAfter)
	if err != nil {
		t.Fatalf("fingerprint after semantic decision: %v", err)
	}
	if semanticFingerprint != afterFingerprint {
		t.Fatalf("deterministic semantic decision changed idempotency identity:\nbase=%s\nsemantic=%s", afterFingerprint, semanticFingerprint)
	}
	retry := cloneTestRequest(after)
	retry.ID = "syscall-retry"
	retry.CorrelationID = "corr-retry"
	retry.TraceID = "trace-retry"
	retry.RequestedAt++
	retry.Provenance.TraceID = "trace-retry"
	retryFingerprint, err := IdempotencyFingerprint(retry)
	if err != nil {
		t.Fatalf("fingerprint retry-local envelope: %v", err)
	}
	if retryFingerprint != afterFingerprint {
		t.Fatalf("retry-local identity changed idempotency fingerprint:\noriginal=%s\nretry=%s", afterFingerprint, retryFingerprint)
	}

	changed := cloneTestRequest(after)
	changed.Metadata["durableCommitAdapter"] = "tampered"
	changedFingerprint, err := IdempotencyFingerprint(changed)
	if err != nil {
		t.Fatalf("fingerprint after unrelated metadata mutation: %v", err)
	}
	if changedFingerprint == afterFingerprint {
		t.Fatal("idempotency fingerprint ignored metadata other than the Court decision")
	}

	planBefore := mustBindTestPlan(t, before, testPlan())
	sealBefore, err := SealPreparedPlan(before, planBefore)
	if err != nil {
		t.Fatalf("seal before Court decision: %v", err)
	}
	planAfter := mustBindTestPlan(t, after, testPlan())
	sealAfter, err := SealPreparedPlan(after, planAfter)
	if err != nil {
		t.Fatalf("seal after Court decision: %v", err)
	}
	if sealBefore.RequestFingerprint == sealAfter.RequestFingerprint || sealBefore.SealDigest == sealAfter.SealDigest {
		t.Fatal("prepared-plan seal did not bind the derived Court decision")
	}
}

func TestIdempotencyFingerprintStillBindsSemanticAuthority(t *testing.T) {
	base := testRequest()
	fingerprint, err := IdempotencyFingerprint(base)
	if err != nil {
		t.Fatalf("base idempotency fingerprint: %v", err)
	}
	for _, tc := range []struct {
		name string
		edit func(*domain.SyscallRequest)
	}{
		{"action", func(req *domain.SyscallRequest) { req.Action = domain.ActionOpenLoop }},
		{"actor", func(req *domain.SyscallRequest) { req.Actor.ID = "other-operator" }},
		{"scope", func(req *domain.SyscallRequest) { req.Scope.WorkspaceID = "ws-other" }},
		{"payload", func(req *domain.SyscallRequest) { req.Payload["content"] = "other content" }},
		{"provenance", func(req *domain.SyscallRequest) { req.Provenance.Source = "other source" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			changed := cloneTestRequest(base)
			tc.edit(&changed)
			got, err := IdempotencyFingerprint(changed)
			if err != nil {
				t.Fatalf("changed fingerprint: %v", err)
			}
			if got == fingerprint {
				t.Fatalf("%s mutation did not change idempotency identity", tc.name)
			}
		})
	}
}

func testRequest() domain.SyscallRequest {
	return domain.SyscallRequest{
		ID:             "syscall-1",
		Action:         domain.ActionCreateNote,
		Actor:          domain.ActorIdentity{ID: "operator-1", Kind: "local_operator"},
		Source:         domain.SourceUser,
		Scope:          domain.ForgeScope{WorkspaceID: "ws-main", LaneID: "control", SelectedPaths: []string{"/forge"}},
		Payload:        map[string]any{"id": "note-1", "title": "Forge", "content": "canonical"},
		Provenance:     domain.Provenance{Actor: "operator-1", ActorType: "local_operator", Source: "test", TraceID: "trace-1"},
		CorrelationID:  "corr-1",
		TraceID:        "trace-1",
		IdempotencyKey: "idem-1",
		RequestedAt:    1_760_000_000_000,
		Metadata:       map[string]any{"kernelAuthorityOwner": "forge_k.kernel"},
	}
}

func testPlan() PreparedPlan {
	return PreparedPlan{
		Action:                domain.ActionCreateNote,
		Capability:            "memory.note.create",
		TargetObjectType:      "memory_note",
		Mutating:              true,
		JournalEventType:      "semantic_syscall_committed",
		ExpectedObjectIDs:     []string{"note-1"},
		ExpectedProvenanceIDs: []string{"prov-1"},
		Details:               map[string]any{"write": "create_note"},
	}
}

func mustBindTestPlan(t *testing.T, req domain.SyscallRequest, plan PreparedPlan) PreparedPlan {
	t.Helper()
	bound, err := BindPreparedPlan(req, plan)
	if err != nil {
		t.Fatalf("bind prepared plan: %v", err)
	}
	return bound
}

func testReceipt(t *testing.T, req domain.SyscallRequest, seal PreparedPlanSeal) CommitReceipt {
	t.Helper()
	idempotency, err := IdempotencyFingerprint(req)
	if err != nil {
		t.Fatalf("idempotency fingerprint: %v", err)
	}
	plan := mustBindTestPlan(t, req, testPlan())
	entry, err := forgejournal.PlanAppend(forgejournal.Head{}, forgejournal.AppendInput{
		EventID: plan.ExpectedJournalEventID, EventType: plan.JournalEventType,
		Source: plan.ExpectedJournalSource, Actor: req.Provenance.Actor,
		WorkspaceID: req.Scope.WorkspaceID, LaneID: req.Scope.LaneID,
		SelectedPaths: append([]string(nil), req.Scope.SelectedPaths...),
		CorrelationID: req.CorrelationID, TraceID: BuildJournalProvenance(req).TraceID,
		ProvenanceID: plan.ExpectedProvenanceIDs[0], ProvenanceHash: plan.ExpectedJournalProvenanceHash,
		PayloadHash: plan.ExpectedJournalPayloadHash, MetadataHash: plan.ExpectedJournalMetadataHash,
		ProposedBy: string(req.Source), CommittedBy: plan.ExpectedJournalCommittedBy,
		SyscallID: req.ID, CreatedAt: req.RequestedAt,
	})
	if err != nil {
		t.Fatalf("journal entry: %v", err)
	}
	return CommitReceipt{
		Version:                CommitReceiptVersion,
		RequestFingerprint:     seal.RequestFingerprint,
		PreparedPlanSeal:       seal.SealDigest,
		TransactionID:          plan.ExpectedTransactionID,
		JournalEventID:         plan.ExpectedJournalEventID,
		JournalEventHash:       entry.Hash,
		ObjectIDs:              append([]string(nil), plan.ExpectedObjectIDs...),
		ProvenanceIDs:          []string{"prov-1"},
		AuditOutboxID:          plan.ExpectedAuditOutboxID,
		IdempotencyFingerprint: idempotency,
		JournalEntry:           entry,
	}
}

func testResult(req domain.SyscallRequest, objectIDs []string) domain.SyscallResult {
	return domain.SyscallResult{
		Success: true, Action: req.Action, RequestID: req.ID,
		CorrelationID: req.CorrelationID, TraceID: req.TraceID,
		IdempotencyKey: req.IdempotencyKey, DryRun: req.DryRun,
		CommittedObjectIDs: append([]string(nil), objectIDs...),
	}
}

func rehashReceiptEntry(t *testing.T, receipt *CommitReceipt) {
	t.Helper()
	hash, err := forgejournal.HashEntry(receipt.JournalEntry)
	if err != nil {
		t.Fatalf("rehash tampered journal entry: %v", err)
	}
	receipt.JournalEntry.Hash = hash
	receipt.JournalEventHash = hash
}

func cloneTestRequest(req domain.SyscallRequest) domain.SyscallRequest {
	clone := req
	clone.Payload = make(map[string]any, len(req.Payload))
	for key, value := range req.Payload {
		clone.Payload[key] = value
	}
	clone.Metadata = make(map[string]any, len(req.Metadata))
	for key, value := range req.Metadata {
		clone.Metadata[key] = value
	}
	clone.Scope.SelectedPaths = append([]string(nil), req.Scope.SelectedPaths...)
	return clone
}

func cloneTestPlan(plan PreparedPlan) PreparedPlan {
	clone := plan
	clone.ExpectedObjectIDs = append([]string(nil), plan.ExpectedObjectIDs...)
	clone.ExpectedProvenanceIDs = append([]string(nil), plan.ExpectedProvenanceIDs...)
	clone.Details = make(map[string]any, len(plan.Details))
	for key, value := range plan.Details {
		clone.Details[key] = value
	}
	return clone
}
