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
	"forge/projectforge/services/core/internal/forgekernel/commitproof"
	"forge/projectforge/services/core/internal/forgekernel/court"
	forgejournal "forge/projectforge/services/core/internal/forgekernel/journal"
	"forge/projectforge/services/core/internal/store"
)

type recordingProcessor struct {
	calls         int
	out           domain.SyscallResult
	err           error
	rejections    int
	prepares      int
	records       int
	observes      int
	events        []string
	disposition   Disposition
	plan          *commitproof.PreparedPlan
	receipt       *commitproof.CommitReceipt
	prepared      *PreparedSyscall
	committedPlan commitproof.PreparedPlan
}

func (p *recordingProcessor) Prepare(_ context.Context, req domain.SyscallRequest) (PreparedSyscall, error) {
	p.prepares++
	p.events = append(p.events, "prepare")
	if p.prepared != nil {
		prepared := *p.prepared
		prepared.Request = req
		return prepared, p.err
	}
	disposition := p.disposition
	if disposition == "" {
		disposition = DispositionCommit
	}
	plan := recordingPlan(req)
	if p.plan != nil {
		plan = *p.plan
	}
	return PreparedSyscall{Request: req, Result: p.out, Disposition: disposition, Plan: plan}, p.err
}

func (p *recordingProcessor) Commit(_ context.Context, prepared PreparedSyscall, seal commitproof.PreparedPlanSeal) (CommitOutcome, error) {
	p.calls++
	p.events = append(p.events, "commit")
	p.committedPlan = prepared.Plan
	receipt := recordingReceipt(prepared, seal)
	if p.receipt != nil {
		receipt = *p.receipt
	}
	result := p.out
	if result.Success {
		result.Action = prepared.Request.Action
		result.RequestID = prepared.Request.ID
		result.CorrelationID = prepared.Request.CorrelationID
		result.TraceID = prepared.Request.TraceID
		result.IdempotencyKey = prepared.Request.IdempotencyKey
		result.DryRun = prepared.Request.DryRun
		if result.CommittedObjectIDs == nil {
			result.CommittedObjectIDs = append([]string(nil), prepared.Plan.ExpectedObjectIDs...)
		}
	}
	return CommitOutcome{Result: result, Receipt: receipt}, p.err
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
	result, err := selection.Processor.Process(context.Background(), kernelTestRequest("sys-1"))
	if err != nil || !result.Success || delegate.prepares != 1 || delegate.calls != 1 || delegate.records != 1 || delegate.observes != 1 {
		t.Fatalf("result=%#v err=%v delegate=%#v", result, err, delegate)
	}
	if result.StateSummary["kernelAuthorityOwner"] != AuthorityOwnerForgeK || result.StateSummary["singleCommitAuthority"] != true {
		t.Fatalf("missing authority metadata: %#v", result.StateSummary)
	}
	if result.StateSummary["commitProofVerified"] != true || result.StateSummary["preparedPlanSeal"] == "" || result.StateSummary["transactionId"] == "" {
		t.Fatalf("missing commit-proof metadata: %#v", result.StateSummary)
	}
	if delegate.committedPlan.ExpectedJournalEventID != "sys-1:journal_event" || delegate.committedPlan.ExpectedJournalPayloadHash == "" {
		t.Fatalf("Kernel did not bind prepared plan before commit: %#v", delegate.committedPlan)
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

func TestForgeKRejectsMissingPreparedPlanBeforeCommit(t *testing.T) {
	missing := commitproof.PreparedPlan{}
	delegate := &recordingProcessor{
		out:  domain.SyscallResult{Success: true},
		plan: &missing,
	}
	selection, err := SelectAuthority(string(ModeForgeK), delegate)
	if err != nil {
		t.Fatalf("select authority: %v", err)
	}
	result, err := selection.Processor.Process(context.Background(), kernelTestRequest("sys-missing-plan"))
	if !errors.Is(err, ErrInvalidPreparedProof) {
		t.Fatalf("process error = %v, want ErrInvalidPreparedProof", err)
	}
	if result.Success || result.DeterministicErrCode != domain.ErrInternal || delegate.calls != 0 || delegate.records != 1 || delegate.observes != 1 {
		t.Fatalf("result=%#v delegate=%#v", result, delegate)
	}
	if result.StateSummary["commitProofVerified"] != false {
		t.Fatalf("missing failure proof metadata: %#v", result.StateSummary)
	}
	if !slices.Equal(delegate.events, []string{"prepare", "record", "observe"}) {
		t.Fatalf("invalid prepared plan reached commit: %v", delegate.events)
	}
}

func TestForgeKRejectsMissingOrTamperedReceiptAfterCommit(t *testing.T) {
	for _, tc := range []struct {
		name    string
		receipt commitproof.CommitReceipt
	}{
		{name: "missing", receipt: commitproof.CommitReceipt{}},
		{name: "tampered", receipt: commitproof.CommitReceipt{
			Version:                commitproof.CommitReceiptVersion,
			RequestFingerprint:     "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			PreparedPlanSeal:       "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			TransactionID:          "tx-tampered",
			JournalEventID:         "journal-tampered",
			JournalEventHash:       "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			ObjectIDs:              []string{"sys-tampered-receipt:note"},
			ProvenanceIDs:          []string{"sys-tampered-receipt:provenance"},
			AuditOutboxID:          "audit-tampered",
			IdempotencyFingerprint: "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			delegate := &recordingProcessor{
				out:     domain.SyscallResult{Success: true},
				receipt: &tc.receipt,
			}
			selection, err := SelectAuthority(string(ModeForgeK), delegate)
			if err != nil {
				t.Fatalf("select authority: %v", err)
			}
			result, err := selection.Processor.Process(context.Background(), kernelTestRequest("sys-"+tc.name+"-receipt"))
			if !errors.Is(err, ErrInvalidCommitReceipt) {
				t.Fatalf("process error = %v, want ErrInvalidCommitReceipt", err)
			}
			if result.Success || result.DeterministicErrCode != domain.ErrPersistenceUnavailable || delegate.calls != 1 || delegate.records != 1 || delegate.observes != 1 {
				t.Fatalf("result=%#v delegate=%#v", result, delegate)
			}
			if result.StateSummary["commitProofVerified"] != false {
				t.Fatalf("missing failure proof metadata: %#v", result.StateSummary)
			}
			if !slices.Equal(delegate.events, []string{"prepare", "commit", "record", "observe"}) {
				t.Fatalf("receipt failure reached wrong stages: %v", delegate.events)
			}
		})
	}
}

func TestForgeKValidatesBoundReplayWithoutCommittingAgain(t *testing.T) {
	original := authorizedKernelTestRequest("sys-original")
	plan := mustBindKernelPlan(t, original, recordingPlan(original))
	seal, err := commitproof.SealPreparedPlan(original, plan)
	if err != nil {
		t.Fatalf("seal original: %v", err)
	}
	stored := domain.SyscallResult{
		Success:            true,
		Action:             original.Action,
		RequestID:          original.ID,
		CorrelationID:      original.CorrelationID,
		TraceID:            original.TraceID,
		IdempotencyKey:     original.IdempotencyKey,
		ApprovalStatus:     domain.ApprovalAllowed,
		CommittedObjectIDs: append([]string(nil), plan.ExpectedObjectIDs...),
		Warnings:           []string{},
		ValidationDetails:  []domain.ValidationDetail{},
		StateSummary:       map[string]any{},
	}
	receipt := recordingReceipt(PreparedSyscall{Request: original, Plan: plan}, seal)
	current := cloneKernelRequest(original)
	current.ID = "sys-retry"
	current.CorrelationID = "corr-retry"
	current.TraceID = "trace-retry"
	current.Provenance.TraceID = "trace-retry"
	current.RequestedAt++
	delegate := &recordingProcessor{prepared: &PreparedSyscall{
		Result: stored, Disposition: DispositionReplay,
		ReplayRequest: original, ReplayPlan: plan, ReplaySeal: seal, ReplayReceipt: receipt,
	}}
	selection, err := SelectAuthority(string(ModeForgeK), delegate)
	if err != nil {
		t.Fatalf("select authority: %v", err)
	}
	result, err := selection.Processor.Process(context.Background(), current)
	if err != nil || !result.Success {
		t.Fatalf("replay: err=%v result=%#v", err, result)
	}
	if delegate.calls != 0 || result.RequestID != current.ID || result.CorrelationID != current.CorrelationID || result.TraceID != current.TraceID {
		t.Fatalf("replay committed or retained original transport identity: result=%#v delegate=%#v", result, delegate)
	}
	if !slices.Contains(result.Warnings, "idempotent replay") || result.StateSummary["commitProofVerified"] != true {
		t.Fatalf("replay lacks verified proof metadata: %#v", result)
	}
	if !slices.Equal(delegate.events, []string{"prepare", "record", "observe"}) {
		t.Fatalf("replay reached commit stage: %v", delegate.events)
	}
}

func TestForgeKRejectsLegacyUnboundOrTamperedReplay(t *testing.T) {
	original := authorizedKernelTestRequest("sys-replay-source")
	plan := mustBindKernelPlan(t, original, recordingPlan(original))
	seal, err := commitproof.SealPreparedPlan(original, plan)
	if err != nil {
		t.Fatalf("seal original: %v", err)
	}
	receipt := recordingReceipt(PreparedSyscall{Request: original, Plan: plan}, seal)
	result := storedKernelResult(original, plan.ExpectedObjectIDs)
	for _, tc := range []struct {
		name string
		edit func(*PreparedSyscall)
	}{
		{name: "legacy-unbound", edit: func(p *PreparedSyscall) { p.ReplayReceipt = commitproof.CommitReceipt{} }},
		{name: "tampered-plan", edit: func(p *PreparedSyscall) { p.ReplayPlan.Details["write"] = "tampered" }},
		{name: "tampered-receipt", edit: func(p *PreparedSyscall) {
			p.ReplayReceipt.PreparedPlanSeal = "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			prepared := PreparedSyscall{
				Result: result, Disposition: DispositionReplay,
				ReplayRequest: original, ReplayPlan: cloneCommitPlan(plan), ReplaySeal: seal, ReplayReceipt: receipt,
			}
			tc.edit(&prepared)
			delegate := &recordingProcessor{prepared: &prepared}
			selection, err := SelectAuthority(string(ModeForgeK), delegate)
			if err != nil {
				t.Fatalf("select authority: %v", err)
			}
			got, err := selection.Processor.Process(context.Background(), cloneKernelRequest(original))
			if !errors.Is(err, ErrInvalidCommitReceipt) && !errors.Is(err, ErrInvalidPreparedProof) {
				t.Fatalf("replay error = %v, want proof failure", err)
			}
			if got.Success || delegate.calls != 0 || delegate.records != 1 || delegate.observes != 1 {
				t.Fatalf("result=%#v delegate=%#v", got, delegate)
			}
		})
	}
}

func TestForgeKReplayReconstructsOriginalCourtDecision(t *testing.T) {
	original := authorizedKernelTestRequest("sys-court-original")
	original.Action = domain.ActionAdmitEvidence
	original.Payload = map[string]any{
		"caseId": "case-1", "exhibitId": "exhibit-1", "rulingId": "ruling-1",
		"sourceRefs":  []any{"artifact:source-1"},
		"policyRefs":  []any{"policy:admission-v1"},
		"contentHash": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}
	decision, issues := court.Decide(original)
	if len(issues) > 0 {
		t.Fatalf("original Court decision: %#v", issues)
	}
	sealedRequest := cloneKernelRequest(original)
	sealedRequest.Metadata[court.MetadataDecisionKey] = decision
	plan := commitproof.PreparedPlan{
		Action: domain.ActionAdmitEvidence, Capability: "memory.evidence.admit", TargetObjectType: "court_exhibit",
		Mutating: true, JournalEventType: "semantic_syscall.admit_evidence",
		ExpectedObjectIDs: []string{"exhibit-1", "ruling-1"}, ExpectedProvenanceIDs: []string{"prov-court-1"},
		Details: map[string]any{"decision": court.DecisionAdmitted},
	}
	plan = mustBindKernelPlan(t, sealedRequest, plan)
	seal, err := commitproof.SealPreparedPlan(sealedRequest, plan)
	if err != nil {
		t.Fatalf("seal Court request: %v", err)
	}
	receipt := recordingReceipt(PreparedSyscall{Request: sealedRequest, Plan: plan}, seal)
	receipt.ProvenanceIDs = []string{"prov-court-1"}
	stored := storedKernelResult(sealedRequest, plan.ExpectedObjectIDs)
	current := cloneKernelRequest(original)
	current.ID = "sys-court-retry"
	current.CorrelationID = "corr-court-retry"
	current.TraceID = "trace-court-retry"
	current.Provenance.TraceID = "trace-court-retry"
	delegate := &recordingProcessor{prepared: &PreparedSyscall{
		Result: stored, Disposition: DispositionReplay,
		ReplayRequest: original, ReplayPlan: plan, ReplaySeal: seal, ReplayReceipt: receipt,
	}}
	selection, err := SelectAuthority(string(ModeForgeK), delegate)
	if err != nil {
		t.Fatalf("select authority: %v", err)
	}
	result, err := selection.Processor.Process(context.Background(), current)
	if err != nil || !result.Success || result.StateSummary["commitProofVerified"] != true || delegate.calls != 0 {
		t.Fatalf("Court replay: result=%#v err=%v delegate=%#v", result, err, delegate)
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
	if result.StateSummary["commitProofVerified"] != true {
		t.Fatalf("commit proof was not exposed on successful result: %#v", result.StateSummary)
	}
	var requestFingerprint, receiptJSON, eventHash string
	if err := st.DB.QueryRowContext(ctx, `
SELECT request_fingerprint,receipt_json
FROM forge_k_audit_outbox WHERE id=?`, req.ID+":audit_outbox").Scan(&requestFingerprint, &receiptJSON); err != nil {
		t.Fatalf("query atomic audit outbox: %v", err)
	}
	var receipt commitproof.CommitReceipt
	if err := json.Unmarshal([]byte(receiptJSON), &receipt); err != nil {
		t.Fatalf("decode atomic commit receipt: %v", err)
	}
	if err := st.DB.QueryRowContext(ctx, `SELECT event_hash FROM journal_events WHERE id=?`, req.ID+":journal_event").Scan(&eventHash); err != nil {
		t.Fatalf("query journal hash: %v", err)
	}
	if requestFingerprint == "" || requestFingerprint != receipt.RequestFingerprint ||
		eventHash == "" || eventHash != receipt.JournalEventHash || receipt.AuditOutboxID != req.ID+":audit_outbox" {
		t.Fatalf("atomic proof evidence mismatch: request=%q event=%q receipt=%#v", requestFingerprint, eventHash, receipt)
	}
	report, err := controllane.NewSQLiteSemanticStore(st.DB).VerifyJournalChain(ctx)
	if err != nil || !report.Passed || report.EntryCount != 1 {
		t.Fatalf("journal chain verification: report=%#v err=%v", report, err)
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

	t.Run("audit outbox failure rolls back object and journal head", func(t *testing.T) {
		selection, st, _ := newLiveSQLiteAuthority(t)
		req := liveNoteRequest("k20d-outbox-failure", "note-k20d-outbox-failure")
		if _, err := st.DB.ExecContext(ctx, `
INSERT INTO forge_k_audit_outbox(
 id,syscall_id,request_fingerprint,action,workspace_id,lane_id,
 correlation_id,trace_id,success,result_json,receipt_json,created_at,committed_by
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			req.ID+":audit_outbox", req.ID, "sha256:wrong", string(req.Action), req.Scope.WorkspaceID, "",
			req.CorrelationID, req.TraceID, 1, `{}`, `{}`, req.RequestedAt, AuthorityOwnerForgeK,
		); err != nil {
			t.Fatalf("seed outbox collision: %v", err)
		}
		result, err := selection.Processor.Process(ctx, req)
		if err != nil || result.Success || result.DeterministicErrCode != domain.ErrPersistenceUnavailable {
			t.Fatalf("result=%#v err=%v", result, err)
		}
		assertNoNoteOrJournal(t, st, req)
		var sequence int64
		if err := st.DB.QueryRowContext(ctx, `SELECT sequence FROM forge_k_journal_head WHERE id=1`).Scan(&sequence); err != nil || sequence != 0 {
			t.Fatalf("rolled-back journal head sequence=%d err=%v", sequence, err)
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

func kernelTestRequest(id string) domain.SyscallRequest {
	return domain.SyscallRequest{
		ID: id, Action: domain.ActionCreateNote,
		Actor:          domain.ActorIdentity{ID: "operator", Kind: string(domain.SourceUser)},
		Source:         domain.SourceUser,
		Scope:          domain.ForgeScope{WorkspaceID: "ws-kernel-test"},
		Payload:        map[string]any{"id": id + ":note"},
		Provenance:     domain.Provenance{Actor: "operator", ActorType: "user", Source: "test", TraceID: "trace-" + id},
		CorrelationID:  "corr-" + id,
		TraceID:        "trace-" + id,
		IdempotencyKey: "idem-" + id,
		RequestedAt:    1760000000000,
	}
}

func authorizedKernelTestRequest(id string) domain.SyscallRequest {
	req := kernelTestRequest(id)
	req.Metadata = map[string]any{
		"forgeKIngressAuthority": true,
		"kernelAuthorityOwner":   AuthorityOwnerForgeK,
		"durableCommitAdapter":   DurableCommitAdapter,
	}
	return req
}

func recordingPlan(req domain.SyscallRequest) commitproof.PreparedPlan {
	objectID, _ := req.Payload["id"].(string)
	return commitproof.PreparedPlan{
		Action:                req.Action,
		Capability:            "memory.note.create",
		TargetObjectType:      "memory_note",
		Mutating:              true,
		JournalEventType:      "semantic_syscall_committed",
		ExpectedObjectIDs:     []string{objectID},
		ExpectedProvenanceIDs: []string{req.ID + ":provenance"},
		Details:               map[string]any{"write": "create_note"},
	}
}

func recordingReceipt(prepared PreparedSyscall, seal commitproof.PreparedPlanSeal) commitproof.CommitReceipt {
	idempotency, _ := commitproof.IdempotencyFingerprint(prepared.Request)
	provenance := commitproof.BuildJournalProvenance(prepared.Request)
	entry, _ := forgejournal.PlanAppend(forgejournal.Head{}, forgejournal.AppendInput{
		EventID: prepared.Plan.ExpectedJournalEventID, EventType: prepared.Plan.JournalEventType,
		Source: prepared.Plan.ExpectedJournalSource, Actor: provenance.Actor,
		WorkspaceID: prepared.Request.Scope.WorkspaceID, LaneID: prepared.Request.Scope.LaneID,
		SelectedPaths: append([]string(nil), prepared.Request.Scope.SelectedPaths...),
		CorrelationID: prepared.Request.CorrelationID, TraceID: provenance.TraceID,
		ProvenanceID: prepared.Plan.ExpectedProvenanceIDs[0], ProvenanceHash: prepared.Plan.ExpectedJournalProvenanceHash,
		PayloadHash: prepared.Plan.ExpectedJournalPayloadHash, MetadataHash: prepared.Plan.ExpectedJournalMetadataHash,
		ProposedBy: string(prepared.Request.Source), CommittedBy: prepared.Plan.ExpectedJournalCommittedBy,
		SyscallID: prepared.Request.ID, CreatedAt: prepared.Request.RequestedAt,
	})
	return commitproof.CommitReceipt{
		Version:                commitproof.CommitReceiptVersion,
		RequestFingerprint:     seal.RequestFingerprint,
		PreparedPlanSeal:       seal.SealDigest,
		TransactionID:          prepared.Plan.ExpectedTransactionID,
		JournalEventID:         prepared.Plan.ExpectedJournalEventID,
		JournalEventHash:       entry.Hash,
		ObjectIDs:              append([]string(nil), prepared.Plan.ExpectedObjectIDs...),
		ProvenanceIDs:          append([]string(nil), prepared.Plan.ExpectedProvenanceIDs...),
		AuditOutboxID:          prepared.Plan.ExpectedAuditOutboxID,
		IdempotencyFingerprint: idempotency,
		JournalEntry:           entry,
	}
}

func mustBindKernelPlan(t *testing.T, req domain.SyscallRequest, plan commitproof.PreparedPlan) commitproof.PreparedPlan {
	t.Helper()
	bound, err := commitproof.BindPreparedPlan(req, plan)
	if err != nil {
		t.Fatalf("bind prepared plan: %v", err)
	}
	return bound
}

func storedKernelResult(req domain.SyscallRequest, objectIDs []string) domain.SyscallResult {
	return domain.SyscallResult{
		Success: true, Action: req.Action, RequestID: req.ID,
		CorrelationID: req.CorrelationID, TraceID: req.TraceID,
		IdempotencyKey: req.IdempotencyKey, DryRun: req.DryRun,
		ApprovalStatus:     domain.ApprovalAllowed,
		CommittedObjectIDs: append([]string(nil), objectIDs...),
		Warnings:           []string{}, ValidationDetails: []domain.ValidationDetail{}, StateSummary: map[string]any{},
	}
}

func cloneKernelRequest(req domain.SyscallRequest) domain.SyscallRequest {
	clone := req
	clone.Scope.SelectedPaths = append([]string(nil), req.Scope.SelectedPaths...)
	clone.CapabilityHints = append([]string(nil), req.CapabilityHints...)
	clone.Payload = make(map[string]any, len(req.Payload))
	for key, value := range req.Payload {
		clone.Payload[key] = value
	}
	clone.Metadata = make(map[string]any, len(req.Metadata))
	for key, value := range req.Metadata {
		clone.Metadata[key] = value
	}
	return clone
}

func cloneCommitPlan(plan commitproof.PreparedPlan) commitproof.PreparedPlan {
	clone := plan
	clone.ExpectedObjectIDs = append([]string(nil), plan.ExpectedObjectIDs...)
	clone.ExpectedProvenanceIDs = append([]string(nil), plan.ExpectedProvenanceIDs...)
	clone.Details = make(map[string]any, len(plan.Details))
	for key, value := range plan.Details {
		clone.Details[key] = value
	}
	return clone
}

type processorWithoutDurablePort struct{}

func (processorWithoutDurablePort) Process(context.Context, domain.SyscallRequest) (domain.SyscallResult, error) {
	return domain.SyscallResult{}, nil
}
