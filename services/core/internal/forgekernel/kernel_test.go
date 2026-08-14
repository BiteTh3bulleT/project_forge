package forgekernel

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"forge/projectforge/services/core/internal/aios/controllane"
	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/store"
)

type recordingProcessor struct {
	calls      int
	out        domain.SyscallResult
	err        error
	rejections int
}

func (p *recordingProcessor) RecordKernelRejection(_ context.Context, _ domain.SyscallRequest, result domain.SyscallResult) domain.SyscallResult {
	p.rejections++
	result.AuditID = "audit-rejection"
	return result
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
	if err != nil || !result.Success || delegate.calls != 1 {
		t.Fatalf("result=%#v err=%v calls=%d", result, err, delegate.calls)
	}
	if result.StateSummary["kernelAuthorityOwner"] != AuthorityOwnerForgeK || result.StateSummary["singleCommitAuthority"] != true {
		t.Fatalf("missing authority metadata: %#v", result.StateSummary)
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
			if result.Success || result.DeterministicErrCode != domain.ErrUnauthorized || result.AuditID == "" || delegate.calls != 0 || delegate.rejections != 1 {
				t.Fatalf("result=%#v calls=%d", result, delegate.calls)
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
	if _, err := SelectAuthority(string(ModeForgeK), processorWithoutRejectionAudit{}); !errors.Is(err, ErrMissingRejectionAudit) {
		t.Fatalf("missing rejection audit error=%v", err)
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

type processorWithoutRejectionAudit struct{}

func (processorWithoutRejectionAudit) Process(context.Context, domain.SyscallRequest) (domain.SyscallResult, error) {
	return domain.SyscallResult{}, nil
}
