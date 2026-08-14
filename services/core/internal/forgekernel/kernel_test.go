package forgekernel_test

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"testing"

	"forge/projectforge/services/core/internal/aios/controllane"
	"forge/projectforge/services/core/internal/aios/domain"
	. "forge/projectforge/services/core/internal/forgekernel"
	"forge/projectforge/services/core/internal/store"
)

type recordingProcessor struct {
	calls       int
	out         domain.SyscallResult
	err         error
	rejections  int
	prepares    int
	records     int
	observes    int
	events      []string
	disposition Disposition
}

func (p *recordingProcessor) Prepare(_ context.Context, req domain.SyscallRequest) (PreparedSyscall, error) {
	p.prepares++
	p.events = append(p.events, "prepare")
	disposition := p.disposition
	if disposition == "" {
		disposition = DispositionCommit
	}
	return PreparedSyscall{Request: req, Result: p.out, Disposition: disposition}, p.err
}

func (p *recordingProcessor) Commit(_ context.Context, _ PreparedSyscall) (domain.SyscallResult, error) {
	p.calls++
	p.events = append(p.events, "commit")
	return p.out, p.err
}

func (p *recordingProcessor) RecordResult(_ context.Context, _ domain.SyscallRequest, result domain.SyscallResult) domain.SyscallResult {
	p.records++
	p.events = append(p.events, "record")
	if !result.Success {
		p.rejections++
	}
	result.AuditID = "audit-result"
	return result
}

func (p *recordingProcessor) ObserveResult(_ context.Context, _ domain.SyscallRequest, _ domain.SyscallResult) {
	p.observes++
	p.events = append(p.events, "observe")
}

func (p *recordingProcessor) Process(_ context.Context, _ domain.SyscallRequest) (domain.SyscallResult, error) {
	p.calls++
	return p.out, p.err
}

func TestSelectAuthorityDefaultsToForgeKWithOneCommitAuthority(t *testing.T) {
	delegate := &recordingProcessor{out: domain.SyscallResult{Success: true}}
	selection, err := SelectAuthority("", delegate)
	if err != nil {
		t.Fatalf("select authority: %v", err)
	}
	if selection.Mode != ModeForgeK || selection.AuthorityOwner != AuthorityOwnerForgeK || !selection.SingleAuthority {
		t.Fatalf("unexpected selection: %#v", selection)
	}
	result, err := selection.Processor.Process(context.Background(), domain.SyscallRequest{ID: "sys-1"})
	if err != nil || !result.Success || delegate.prepares != 1 || delegate.calls != 1 || delegate.records != 1 || delegate.observes != 1 {
		t.Fatalf("result=%#v err=%v delegate=%#v", result, err, delegate)
	}
	if result.StateSummary["kernelAuthorityOwner"] != AuthorityOwnerForgeK || result.StateSummary["singleCommitAuthority"] != true {
		t.Fatalf("missing authority metadata: %#v", result.StateSummary)
	}
	if !slices.Equal(delegate.events, []string{"prepare", "commit", "record", "observe"}) {
		t.Fatalf("FORGE-K did not own durable stage order: %v", delegate.events)
	}
}

func TestForgeKCompleteDispositionNeverCommits(t *testing.T) {
	delegate := &recordingProcessor{
		out:         domain.SyscallResult{Success: false, DeterministicErrCode: domain.ErrCapabilityDenied},
		disposition: DispositionComplete,
	}
	selection, err := SelectAuthority(string(ModeForgeK), delegate)
	if err != nil {
		t.Fatalf("select authority: %v", err)
	}
	result, err := selection.Processor.Process(context.Background(), domain.SyscallRequest{ID: "sys-complete"})
	if err != nil || result.Success || delegate.calls != 0 {
		t.Fatalf("result=%#v err=%v delegate=%#v", result, err, delegate)
	}
	if !slices.Equal(delegate.events, []string{"prepare", "record", "observe"}) {
		t.Fatalf("completed preflight reached wrong stages: %v", delegate.events)
	}
}

func TestSelectAuthorityLegacyRollbackBypassesForgeKFacade(t *testing.T) {
	delegate := &recordingProcessor{}
	selection, err := SelectAuthority(string(ModeLegacyV1), delegate)
	if err != nil {
		t.Fatalf("select authority: %v", err)
	}
	if selection.Mode != ModeLegacyV1 || selection.Processor != delegate || !selection.SingleAuthority {
		t.Fatalf("unexpected rollback selection: %#v", selection)
	}
}

func TestForgeKRejectsExternalAuthorityClaimBeforeCommit(t *testing.T) {
	for name, claim := range map[string]any{
		"boolean": true,
		"string":  "true",
		"numeric": float64(1),
	} {
		t.Run(name, func(t *testing.T) {
			delegate := &recordingProcessor{out: domain.SyscallResult{Success: true}}
			selection, err := SelectAuthority(string(ModeForgeK), delegate)
			if err != nil {
				t.Fatalf("select authority: %v", err)
			}
			result, err := selection.Processor.Process(context.Background(), domain.SyscallRequest{
				ID:       "sys-claim",
				Metadata: map[string]any{"modelToolSelectionAuthority": claim},
			})
			if err != nil {
				t.Fatalf("process: %v", err)
			}
			if result.Success || result.DeterministicErrCode != domain.ErrUnauthorized || result.AuditID == "" || delegate.prepares != 0 || delegate.calls != 0 || delegate.rejections != 1 || delegate.records != 1 || delegate.observes != 1 {
				t.Fatalf("result=%#v delegate=%#v", result, delegate)
			}
		})
	}
}

func TestSelectAuthorityFailsClosed(t *testing.T) {
	if _, err := SelectAuthority("both", &recordingProcessor{}); !errors.Is(err, ErrInvalidAuthorityMode) {
		t.Fatalf("invalid mode error=%v", err)
	}
	if _, err := SelectAuthority(string(ModeForgeK), nil); !errors.Is(err, ErrMissingCommitAdapter) {
		t.Fatalf("missing adapter error=%v", err)
	}
	if _, err := SelectAuthority(string(ModeForgeK), processorWithoutDurablePort{}); !errors.Is(err, ErrMissingDurablePort) {
		t.Fatalf("missing durable port error=%v", err)
	}
}

func TestForgeKLiveCommitCarriesAuthorityIntoAuditAndJournal(t *testing.T) {
	ctx := context.Background()
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	auditSink := controllane.NewInMemoryAuditSink()
	commit := controllane.NewProcessor(controllane.ProcessorOptions{
		Registry:     controllane.NewStaticActionRegistry(),
		Validator:    controllane.NewDeterministicValidator(),
		Capabilities: controllane.NewStaticCapabilityService(),
		ApprovalGate: controllane.NewStaticApprovalGate(),
		TxRunner:     controllane.NewSQLiteTransactionRunner(st.DB),
		AuditSink:    auditSink,
		NowMillis:    func() int64 { return 1760000000000 },
	})
	selection, err := SelectAuthority(string(ModeForgeK), commit)
	if err != nil {
		t.Fatalf("select authority: %v", err)
	}
	req := domain.SyscallRequest{
		ID: "k20a-note-commit", Action: domain.ActionCreateNote,
		Actor: domain.ActorIdentity{ID: "operator", Kind: string(domain.SourceUser)}, Source: domain.SourceUser,
		Scope:         domain.ForgeScope{WorkspaceID: "ws-k20a"},
		Payload:       map[string]any{"id": "k20a-note", "type": string(domain.NoteFact), "title": "K20A", "content": "FORGE-K ingress evidence"},
		Provenance:    domain.Provenance{Actor: "operator", ActorType: "user", Source: "test", TraceID: "trace-k20a"},
		CorrelationID: "corr-k20a", TraceID: "trace-k20a", RequestedAt: 1760000000000,
	}
	result, err := selection.Processor.Process(ctx, req)
	if err != nil || !result.Success {
		t.Fatalf("process: err=%v result=%#v", err, result)
	}
	if len(auditSink.Records) != 1 {
		t.Fatalf("audit records=%d", len(auditSink.Records))
	}
	authority, _ := auditSink.Records[0].SemanticSyscallEnvelope["authorityEffects"].(map[string]any)
	if authority["forgeKIngressOwned"] != true || authority["controlLaneOwned"] != false || authority["controlLaneCommitAdapter"] != true {
		t.Fatalf("audit authority mismatch: %#v", authority)
	}
	var payloadJSON string
	if err := st.DB.QueryRowContext(ctx, `SELECT payload_json FROM journal_events WHERE id = ?`, req.ID+":journal_event").Scan(&payloadJSON); err != nil {
		t.Fatalf("query journal: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		t.Fatalf("decode journal payload: %v", err)
	}
	if payload["kernelAuthorityOwner"] != AuthorityOwnerForgeK || payload["durableCommitAdapter"] != DurableCommitAdapter {
		t.Fatalf("journal authority mismatch: %#v", payload)
	}
}

func TestForgeKDurablePortGatesAndRollback(t *testing.T) {
	ctx := context.Background()
	t.Run("idempotent replay does not commit twice", func(t *testing.T) {
		selection, st, _ := newLiveSQLiteAuthority(t)
		req := liveNoteRequest("k20b-idempotent", "note-k20b-idempotent")
		req.IdempotencyKey = "idem-k20b"
		first, err := selection.Processor.Process(ctx, req)
		if err != nil || !first.Success {
			t.Fatalf("first process: err=%v result=%#v", err, first)
		}
		req.ID = "k20b-idempotent-replay"
		req.CorrelationID = "corr-k20b-idempotent-replay"
		replayed, err := selection.Processor.Process(ctx, req)
		if err != nil || !replayed.Success || !slices.Contains(replayed.Warnings, "idempotent replay") {
			t.Fatalf("replay: err=%v result=%#v", err, replayed)
		}
		var journals int
		if err := st.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM journal_events WHERE id IN (?, ?)`, "k20b-idempotent:journal_event", "k20b-idempotent-replay:journal_event").Scan(&journals); err != nil || journals != 1 {
			t.Fatalf("journal count=%d err=%v", journals, err)
		}
	})

	t.Run("capability denial never commits", func(t *testing.T) {
		selection, st, _ := newLiveSQLiteAuthority(t)
		req := liveNoteRequest("k20b-capability", "note-k20b-capability")
		req.Action = domain.ActionArchiveNote
		req.Source = domain.SourceAdapter
		req.Actor.Kind = string(domain.SourceAdapter)
		req.Payload = map[string]any{"noteId": "note-k20b-capability", "reason": "must not commit"}
		result, err := selection.Processor.Process(ctx, req)
		if err != nil || result.Success || result.DeterministicErrCode != domain.ErrCapabilityDenied {
			t.Fatalf("result=%#v err=%v", result, err)
		}
		assertNoNoteOrJournal(t, st, req)
	})

	t.Run("proposal source requires approval", func(t *testing.T) {
		selection, st, _ := newLiveSQLiteAuthority(t)
		req := liveNoteRequest("k20b-approval", "note-k20b-approval")
		req.Source = domain.SourceFutureIRIS
		req.Actor.Kind = string(domain.SourceFutureIRIS)
		result, err := selection.Processor.Process(ctx, req)
		if err != nil || result.Success || result.DeterministicErrCode != domain.ErrApprovalRequired {
			t.Fatalf("result=%#v err=%v", result, err)
		}
		assertNoNoteOrJournal(t, st, req)
	})

	t.Run("journal failure rolls back object", func(t *testing.T) {
		selection, st, _ := newLiveSQLiteAuthority(t)
		req := liveNoteRequest("k20b-journal-failure", "note-k20b-journal-failure")
		if _, err := st.DB.ExecContext(ctx, `INSERT INTO journal_events(id, type, source, workspace_id, created_at) VALUES(?,?,?,?,?)`, req.ID+":journal_event", "preexisting", "test", req.Scope.WorkspaceID, req.RequestedAt); err != nil {
			t.Fatalf("seed journal collision: %v", err)
		}
		result, err := selection.Processor.Process(ctx, req)
		if err != nil || result.Success || result.DeterministicErrCode != domain.ErrPersistenceUnavailable {
			t.Fatalf("result=%#v err=%v", result, err)
		}
		var notes int
		if err := st.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM memory_notes WHERE id = ?`, req.Payload["id"]).Scan(&notes); err != nil || notes != 0 {
			t.Fatalf("rolled-back note count=%d err=%v", notes, err)
		}
	})
}

func newLiveSQLiteAuthority(t *testing.T) (Selection, *store.Store, *controllane.InMemoryAuditSink) {
	t.Helper()
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	auditSink := controllane.NewInMemoryAuditSink()
	adapter := controllane.NewProcessor(controllane.ProcessorOptions{
		TxRunner:  controllane.NewSQLiteTransactionRunner(st.DB),
		AuditSink: auditSink,
		NowMillis: func() int64 { return 1760000000000 },
	})
	selection, err := SelectAuthority(string(ModeForgeK), adapter)
	if err != nil {
		t.Fatalf("select authority: %v", err)
	}
	return selection, st, auditSink
}

func liveNoteRequest(id, noteID string) domain.SyscallRequest {
	return domain.SyscallRequest{
		ID: id, Action: domain.ActionCreateNote,
		Actor: domain.ActorIdentity{ID: "operator", Kind: string(domain.SourceUser)}, Source: domain.SourceUser,
		Scope:         domain.ForgeScope{WorkspaceID: "ws-k20b"},
		Payload:       map[string]any{"id": noteID, "type": string(domain.NoteFact), "title": "K20B", "content": "durable port evidence"},
		Provenance:    domain.Provenance{Actor: "operator", ActorType: "user", Source: "test", TraceID: "trace-" + id},
		CorrelationID: "corr-" + id, TraceID: "trace-" + id, RequestedAt: 1760000000000,
	}
}

func assertNoNoteOrJournal(t *testing.T, st *store.Store, req domain.SyscallRequest) {
	t.Helper()
	var notes, journals int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM memory_notes WHERE id = ?`, req.Payload["id"]).Scan(&notes); err != nil {
		t.Fatalf("query notes: %v", err)
	}
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM journal_events WHERE id = ?`, req.ID+":journal_event").Scan(&journals); err != nil {
		t.Fatalf("query journals: %v", err)
	}
	if notes != 0 || journals != 0 {
		t.Fatalf("denied request committed notes=%d journals=%d", notes, journals)
	}
}

type processorWithoutDurablePort struct{}

func (processorWithoutDurablePort) Process(context.Context, domain.SyscallRequest) (domain.SyscallResult, error) {
	return domain.SyscallResult{}, nil
}
