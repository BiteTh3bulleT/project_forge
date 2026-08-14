package controllane

import (
	"context"
	"errors"
	"testing"

	"forge/projectforge/services/core/internal/aios/domain"
	forgejournal "forge/projectforge/services/core/internal/forgekernel/journal"
	corestore "forge/projectforge/services/core/internal/store"
)

func TestSQLiteJournalAppendReturnsDurableChainEvidence(t *testing.T) {
	ctx := context.Background()
	_, txRunner, st := newSQLiteKernel(t, nil)
	journalStore := txRunner.read
	journalStore.SetCommitMetadata(CommitMetadata{
		SyscallID: "sys-1", CorrelationID: "corr-1", TraceID: "trace-1",
		Source: domain.SourceUser, CommittedBy: "forge_k.kernel",
	})
	first, err := journalStore.AppendWithEvidence(ctx, createTestJournalEvent("evt-chain-1", "ws-chain", "corr-1", 1760001000001))
	if err != nil {
		t.Fatal(err)
	}
	journalStore.SetCommitMetadata(CommitMetadata{
		SyscallID: "sys-2", CorrelationID: "corr-2", TraceID: "trace-2",
		Source: domain.SourceSystem, CommittedBy: "forge_k.kernel",
	})
	second, err := journalStore.AppendWithEvidence(ctx, createTestJournalEvent("evt-chain-2", "ws-chain", "corr-2", 1760001000002))
	if err != nil {
		t.Fatal(err)
	}

	if first.PreviousHead != (forgejournal.Head{}) || first.Entry.Sequence != 1 || first.Head.Hash != first.Entry.Hash {
		t.Fatalf("unexpected genesis evidence: %#v", first)
	}
	if second.PreviousHead != first.Head || second.Entry.Sequence != 2 || second.Entry.PriorHash != first.Entry.Hash {
		t.Fatalf("unexpected linked evidence: first=%#v second=%#v", first, second)
	}
	var storedSequence uint64
	var storedPrior, storedHash string
	if err := st.DB.QueryRow(`
SELECT chain_sequence,prior_hash,event_hash FROM journal_events WHERE id=?`, second.Entry.EventID).Scan(
		&storedSequence, &storedPrior, &storedHash,
	); err != nil {
		t.Fatal(err)
	}
	if storedSequence != second.Entry.Sequence || storedPrior != second.Entry.PriorHash || storedHash != second.Entry.Hash {
		t.Fatalf("typed evidence differs from storage: sequence=%d prior=%q hash=%q", storedSequence, storedPrior, storedHash)
	}
	report, err := journalStore.VerifyJournalChain(ctx)
	if err != nil || !report.Passed || report.Head != second.Head {
		t.Fatalf("stored chain verification failed: report=%#v err=%v", report, err)
	}
}

func TestSQLiteJournalAppendRollsBackWithCallerTransaction(t *testing.T) {
	ctx := context.Background()
	_, txRunner, _ := newSQLiteKernel(t, nil)
	wantRollback := errors.New("force semantic mutation rollback")
	err := txRunner.Run(ctx, func(uow UnitOfWork) error {
		journalStore, ok := uow.Store().(*SQLiteSemanticStore)
		if !ok {
			t.Fatalf("unexpected unit-of-work store %T", uow.Store())
		}
		journalStore.SetCommitMetadata(CommitMetadata{SyscallID: "sys-rollback", CommittedBy: "forge_k.kernel"})
		if _, err := journalStore.AppendWithEvidence(ctx, createTestJournalEvent("evt-rollback", "ws-chain", "corr-rollback", 1760001000100)); err != nil {
			return err
		}
		return wantRollback
	})
	if !errors.Is(err, wantRollback) {
		t.Fatalf("unexpected transaction error: %v", err)
	}
	if _, ok, err := txRunner.read.GetByID(ctx, "evt-rollback"); err != nil || ok {
		t.Fatalf("journal row escaped rollback: ok=%v err=%v", ok, err)
	}
	head, err := txRunner.read.JournalChainHead(ctx)
	if err != nil || head != (forgejournal.Head{}) {
		t.Fatalf("journal head escaped rollback: head=%#v err=%v", head, err)
	}
}

func TestSQLiteJournalVerificationBindsStoredJSON(t *testing.T) {
	ctx := context.Background()
	_, txRunner, st := newSQLiteKernel(t, nil)
	journalStore := txRunner.read
	journalStore.SetCommitMetadata(CommitMetadata{SyscallID: "sys-tamper", CommittedBy: "forge_k.kernel"})
	if _, err := journalStore.AppendWithEvidence(ctx, createTestJournalEvent("evt-tamper", "ws-chain", "corr-tamper", 1760001000200)); err != nil {
		t.Fatal(err)
	}
	if _, err := st.DB.Exec(`DROP TRIGGER journal_events_no_update`); err != nil {
		t.Fatal(err)
	}
	if _, err := st.DB.Exec(`UPDATE journal_events SET payload_json='{"tampered":true}' WHERE id='evt-tamper'`); err != nil {
		t.Fatal(err)
	}
	if _, err := journalStore.VerifyJournalChain(ctx); !errors.Is(err, ErrJournalContentHashMismatch) {
		t.Fatalf("expected stored JSON tamper detection, got %v", err)
	}
}

func TestSQLiteJournalAppendFailsClosedOnDivergentHead(t *testing.T) {
	ctx := context.Background()
	_, txRunner, st := newSQLiteKernel(t, nil)
	journalStore := txRunner.read
	journalStore.SetCommitMetadata(CommitMetadata{SyscallID: "sys-first", CommittedBy: "forge_k.kernel"})
	if _, err := journalStore.AppendWithEvidence(ctx, createTestJournalEvent("evt-first", "ws-chain", "corr-first", 1760001000300)); err != nil {
		t.Fatal(err)
	}
	// Simulate replacement of the independently retained head. The unique
	// sequence index prevents the next append from building a second sequence 1.
	if _, err := st.DB.Exec(`UPDATE forge_k_journal_head SET sequence=0,event_id='',head_hash='' WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	journalStore.SetCommitMetadata(CommitMetadata{SyscallID: "sys-second", CommittedBy: "forge_k.kernel"})
	if _, err := journalStore.AppendWithEvidence(ctx, createTestJournalEvent("evt-second", "ws-chain", "corr-second", 1760001000301)); err == nil {
		t.Fatal("expected append against divergent head to fail")
	}
	var count int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM journal_events`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("failed append was not atomic; journal rows=%d", count)
	}
}

func TestSQLiteJournalChainSurvivesRestart(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()
	firstStore, err := corestore.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	journalStore := NewSQLiteSemanticStore(firstStore.DB)
	journalStore.SetCommitMetadata(CommitMetadata{SyscallID: "sys-restart", CommittedBy: "forge_k.kernel"})
	evidence, err := journalStore.AppendWithEvidence(ctx, createTestJournalEvent("evt-restart", "ws-chain", "corr-restart", 1760001000400))
	if err != nil {
		t.Fatal(err)
	}
	if err := firstStore.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := corestore.Open(dataDir)
	if err != nil {
		t.Fatalf("restart migration rejected valid journal: %v", err)
	}
	defer reopened.Close()
	reopenedJournal := NewSQLiteSemanticStore(reopened.DB)
	report, err := reopenedJournal.VerifyJournalChain(ctx)
	if err != nil || !report.Passed || report.Head != evidence.Head {
		t.Fatalf("journal did not survive restart: report=%#v err=%v", report, err)
	}
}
