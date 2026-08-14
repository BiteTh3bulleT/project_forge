package controllane

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/forgekernel/commitproof"
	"forge/projectforge/services/core/internal/store"
)

func TestInMemoryIntegrityRecordsAreConditionalAndImmutable(t *testing.T) {
	semantic := NewInMemorySemanticStore()
	assertIntegrityStorageContract(t, semantic)
}

func TestSQLiteIntegrityRecordsAreConditionalAndImmutable(t *testing.T) {
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	semantic := NewSQLiteSemanticStore(st.DB)
	assertIntegrityStorageContract(t, semantic)

	for _, statement := range []string{
		`UPDATE semantic_idempotency_keys SET request_fingerprint = 'changed' WHERE idempotency_key = 'idem-1'`,
		`DELETE FROM semantic_idempotency_keys WHERE idempotency_key = 'idem-1'`,
		`UPDATE forge_k_audit_outbox SET success = 0 WHERE id = 'audit-intent-1'`,
		`DELETE FROM forge_k_audit_outbox WHERE id = 'audit-intent-1'`,
	} {
		if _, err := st.DB.Exec(statement); err == nil {
			t.Fatalf("immutable integrity row accepted statement: %s", statement)
		}
	}
}

func TestSQLiteIntegrityRecordsRollbackWithUnitOfWork(t *testing.T) {
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	runner := NewSQLiteTransactionRunner(st.DB)
	rollback := errors.New("force rollback")
	err = runner.Run(context.Background(), func(uow UnitOfWork) error {
		if err := uow.Store().SetIdempotency("idem-rollback", testIdempotencyRecord("fp-rollback")); err != nil {
			return err
		}
		if err := uow.Store().CreateAuditOutbox(testAuditOutboxRecord("audit-rollback", "sys-rollback", "fp-rollback", 10)); err != nil {
			return err
		}
		return rollback
	})
	if !errors.Is(err, rollback) {
		t.Fatalf("unexpected rollback result: %v", err)
	}
	read := runner.ReadStore()
	if _, ok := read.GetIdempotency("idem-rollback"); ok {
		t.Fatal("rolled-back idempotency row remained visible")
	}
	if _, ok := read.GetAuditOutbox("audit-rollback"); ok {
		t.Fatal("rolled-back audit outbox row remained visible")
	}
}

func TestInMemoryJournalChainParityAndRollback(t *testing.T) {
	ctx := context.Background()
	base := NewInMemorySemanticStore()
	runner := NewInMemoryTransactionRunner(base)
	var first JournalAppendEvidence
	if err := runner.Run(ctx, func(uow UnitOfWork) error {
		store := uow.Store().(*TransactionalSemanticStore)
		store.SetCommitMetadata(CommitMetadata{SyscallID: "sys-1", Source: domain.SourceUser, CommittedBy: "forge_k.kernel"})
		var err error
		first, err = store.AppendWithEvidence(ctx, testJournalEvent("evt-1", "sys-1", 10))
		return err
	}); err != nil {
		t.Fatalf("append first in-memory journal entry: %v", err)
	}
	if first.PreviousHead.Sequence != 0 || first.Entry.Sequence != 1 || first.Head.Hash == "" {
		t.Fatalf("invalid first append evidence: %#v", first)
	}
	rollback := errors.New("rollback append")
	if err := runner.Run(ctx, func(uow UnitOfWork) error {
		store := uow.Store().(*TransactionalSemanticStore)
		store.SetCommitMetadata(CommitMetadata{SyscallID: "sys-rollback", Source: domain.SourceUser, CommittedBy: "forge_k.kernel"})
		if _, err := store.AppendWithEvidence(ctx, testJournalEvent("evt-rollback", "sys-rollback", 15)); err != nil {
			return err
		}
		return rollback
	}); !errors.Is(err, rollback) {
		t.Fatalf("unexpected rollback result: %v", err)
	}
	head, err := base.JournalChainHead(ctx)
	if err != nil || head != first.Head {
		t.Fatalf("rollback advanced journal head: head=%#v err=%v", head, err)
	}
	var second JournalAppendEvidence
	if err := runner.Run(ctx, func(uow UnitOfWork) error {
		store := uow.Store().(*TransactionalSemanticStore)
		store.SetCommitMetadata(CommitMetadata{SyscallID: "sys-2", Source: domain.SourceUser, CommittedBy: "forge_k.kernel"})
		var err error
		second, err = store.AppendWithEvidence(ctx, testJournalEvent("evt-2", "sys-2", 20))
		return err
	}); err != nil {
		t.Fatalf("append second in-memory journal entry: %v", err)
	}
	if second.PreviousHead != first.Head || second.Entry.PriorHash != first.Entry.Hash || second.Entry.Sequence != 2 {
		t.Fatalf("journal chain did not link: first=%#v second=%#v", first, second)
	}
	report, err := base.VerifyJournalChain(ctx)
	if err != nil || !report.Passed || report.Head != second.Head || report.EntryCount != 2 {
		t.Fatalf("in-memory journal verification failed: report=%#v err=%v", report, err)
	}
}

type integrityStore interface {
	SetIdempotency(string, IdempotencyRecord) error
	GetIdempotency(string) (IdempotencyRecord, bool)
	CreateAuditOutbox(AuditOutboxRecord) error
	GetAuditOutbox(string) (AuditOutboxRecord, bool)
	ListAuditOutbox(int) []AuditOutboxRecord
}

func assertIntegrityStorageContract(t *testing.T, semantic integrityStore) {
	t.Helper()
	idem := testIdempotencyRecord("fp-1")
	if err := semantic.SetIdempotency("idem-1", idem); err != nil {
		t.Fatalf("insert idempotency: %v", err)
	}
	replay := idem
	replay.Result.Warnings = []string{"must not overwrite immutable result"}
	if err := semantic.SetIdempotency("idem-1", replay); err != nil {
		t.Fatalf("same fingerprint must be replay/no-op: %v", err)
	}
	got, ok := semantic.GetIdempotency("idem-1")
	if !ok || got.RequestFingerprint != "fp-1" || got.IdempotencyFingerprint != "idem-fp-1" ||
		got.Request.ID != "sys-1" || got.Plan.Action != domain.ActionCreateNote ||
		got.Seal.Version != commitproof.PreparedPlanVersion || got.Receipt.Version != commitproof.CommitReceiptVersion ||
		len(got.Result.Warnings) != 0 {
		t.Fatalf("idempotency row changed during replay: %#v", got)
	}
	got.Result.Warnings = append(got.Result.Warnings, "caller mutation")
	again, _ := semantic.GetIdempotency("idem-1")
	if len(again.Result.Warnings) != 0 {
		t.Fatalf("caller mutated immutable idempotency row through read alias: %#v", again)
	}
	conflict := idem
	conflict.RequestFingerprint = "fp-2"
	conflict.Seal.RequestFingerprint = "fp-2"
	conflict.Receipt.RequestFingerprint = "fp-2"
	if err := semantic.SetIdempotency("idem-1", conflict); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("fingerprint conflict did not fail closed: %v", err)
	}
	conflict = idem
	conflict.IdempotencyFingerprint = "idem-fp-2"
	conflict.Receipt.IdempotencyFingerprint = "idem-fp-2"
	if err := semantic.SetIdempotency("idem-1", conflict); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("idempotency fingerprint conflict did not fail closed: %v", err)
	}
	conflict = idem
	conflict.Action = domain.ActionCreateLink
	conflict.Request.Action = domain.ActionCreateLink
	conflict.Plan.Action = domain.ActionCreateLink
	if err := semantic.SetIdempotency("idem-1", conflict); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("action conflict did not fail closed: %v", err)
	}
	invalid := idem
	invalid.RequestFingerprint = ""
	if err := semantic.SetIdempotency("idem-invalid", invalid); !errors.Is(err, ErrInvalidIdempotencyRecord) {
		t.Fatalf("missing fingerprint accepted: %v", err)
	}

	first := testAuditOutboxRecord("audit-intent-1", "sys-1", "fp-1", 20)
	second := testAuditOutboxRecord("audit-intent-2", "sys-2", "fp-2", 10)
	if err := semantic.CreateAuditOutbox(first); err != nil {
		t.Fatalf("create audit intent: %v", err)
	}
	if err := semantic.CreateAuditOutbox(second); err != nil {
		t.Fatalf("create second audit intent: %v", err)
	}
	if err := semantic.CreateAuditOutbox(first); err != nil {
		t.Fatalf("exact audit-intent replay must be no-op: %v", err)
	}
	mutated := first
	mutated.Result.Success = false
	mutated.Success = false
	if err := semantic.CreateAuditOutbox(mutated); !errors.Is(err, ErrAuditOutboxConflict) {
		t.Fatalf("audit-intent mutation did not fail closed: %v", err)
	}
	duplicateSyscall := first
	duplicateSyscall.ID = "audit-intent-other"
	if err := semantic.CreateAuditOutbox(duplicateSyscall); !errors.Is(err, ErrAuditOutboxConflict) {
		t.Fatalf("duplicate syscall audit intent did not fail closed: %v", err)
	}
	stored, ok := semantic.GetAuditOutbox(first.ID)
	if !ok || stored.RequestFingerprint != first.RequestFingerprint || stored.Receipt.AuditOutboxID != first.ID || stored.CommittedBy != "forge_k.kernel" {
		t.Fatalf("audit intent read mismatch: %#v", stored)
	}
	stored.Result.Warnings = append(stored.Result.Warnings, "caller mutation")
	storedAgain, _ := semantic.GetAuditOutbox(first.ID)
	if len(storedAgain.Result.Warnings) != 0 {
		t.Fatalf("caller mutated immutable audit intent through read alias: %#v", storedAgain)
	}
	listed := semantic.ListAuditOutbox(1)
	if len(listed) != 1 || listed[0].ID != second.ID {
		t.Fatalf("audit outbox delivery order mismatch: %#v", listed)
	}
}

func testIdempotencyRecord(fingerprint string) IdempotencyRecord {
	return IdempotencyRecord{
		Action: domain.ActionCreateNote, RequestFingerprint: fingerprint, IdempotencyFingerprint: "idem-" + fingerprint,
		Result:    domain.SyscallResult{Success: true, Action: domain.ActionCreateNote, RequestID: "sys-1", CorrelationID: "corr-1"},
		Request:   domain.SyscallRequest{ID: "sys-1", Action: domain.ActionCreateNote},
		Plan:      commitproof.PreparedPlan{Action: domain.ActionCreateNote},
		Seal:      commitproof.PreparedPlanSeal{Version: commitproof.PreparedPlanVersion, RequestFingerprint: fingerprint},
		Receipt:   commitproof.CommitReceipt{Version: commitproof.CommitReceiptVersion, RequestFingerprint: fingerprint, IdempotencyFingerprint: "idem-" + fingerprint},
		CreatedAt: 10, CorrelationID: "corr-1",
	}
}

func testAuditOutboxRecord(id, syscallID, fingerprint string, createdAt int64) AuditOutboxRecord {
	rec := AuditOutboxRecord{
		ID: id, SyscallID: syscallID, RequestFingerprint: fingerprint, Action: domain.ActionCreateNote,
		WorkspaceID: "ws-1", LaneID: "control.semantic", CorrelationID: "corr-" + syscallID,
		TraceID: "trace-" + syscallID, Success: true,
		Result:    domain.SyscallResult{Success: true, Action: domain.ActionCreateNote, RequestID: syscallID},
		CreatedAt: createdAt, CommittedBy: "forge_k.kernel",
	}
	rec.Receipt = commitproof.CommitReceipt{Version: commitproof.CommitReceiptVersion, RequestFingerprint: fingerprint, AuditOutboxID: id}
	return rec
}

func testJournalEvent(id, syscallID string, createdAt int64) domain.JournalEvent {
	return domain.JournalEvent{
		ID: id, Type: "semantic_syscall.create_note", Source: "forge_k.kernel", Timestamp: createdAt,
		Scope:         domain.ForgeScope{WorkspaceID: "ws-1", LaneID: "control.semantic"},
		Payload:       map[string]any{"action": domain.ActionCreateNote, "syscallId": syscallID},
		CorrelationID: "corr-" + syscallID,
		Provenance:    domain.Provenance{Actor: "operator", ActorType: "user", Source: "test", TraceID: "trace-" + syscallID},
	}
}

func ExampleAuditOutboxRecord() {
	rec := testAuditOutboxRecord("audit-intent-1", "sys-1", "sha256:request", 10)
	fmt.Println(rec.SyscallID, rec.RequestFingerprint)
	// Output: sys-1 sha256:request
}
