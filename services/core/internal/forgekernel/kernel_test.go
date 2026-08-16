package forgekernel_test

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"

	"forge/projectforge/services/core/internal/aios/controllane"
	"forge/projectforge/services/core/internal/aios/domain"
	. "forge/projectforge/services/core/internal/forgekernel"
	"forge/projectforge/services/core/internal/forgekernel/authproof"
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
	suppliedAuth  *authproof.Proof
	committedPlan commitproof.PreparedPlan
	committedAuth authproof.Proof
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
	prepared := PreparedSyscall{Request: req, Result: p.out, Disposition: disposition, Plan: plan}
	if p.suppliedAuth != nil {
		prepared.AuthorizationProof = *p.suppliedAuth
	}
	return prepared, p.err
}

func (p *recordingProcessor) Commit(_ context.Context, prepared PreparedSyscall, seal commitproof.PreparedPlanSeal) (CommitOutcome, error) {
	p.calls++
	p.events = append(p.events, "commit")
	p.committedPlan = prepared.Plan
	p.committedAuth = prepared.AuthorizationProof
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

type testAuthorizationPort struct {
	err    error
	mutate func(*authproof.Proof)
}

func (p testAuthorizationPort) ResolveAuthorization(_ context.Context, req domain.SyscallRequest) (authproof.Proof, error) {
	if p.err != nil {
		return authproof.Proof{}, p.err
	}
	if err := testProductionAuthorizationPolicy(req); err != nil {
		return authproof.Proof{}, err
	}
	proof, err := buildTestAuthorizationProof(req)
	if err != nil {
		return authproof.Proof{}, err
	}
	if p.mutate != nil {
		p.mutate(&proof)
	}
	return proof, nil
}

func testProductionAuthorizationPolicy(req domain.SyscallRequest) error {
	def, ok := controllane.NewStaticActionRegistry().Get(req.Action)
	if !ok {
		return errors.New("production authorization action missing")
	}
	allowed := req.Source == domain.SourceUser || req.Source == domain.SourceSystem || req.Source == domain.SourceInternal
	switch req.Source {
	case domain.SourceAdapter:
		switch req.Action {
		case domain.ActionCreateNote, domain.ActionCreateLink,
			domain.ActionValidateKVIdentity, domain.ActionValidateRefShape, domain.ActionCompareRefShape,
			domain.ActionValidateSourceObject, domain.ActionValidateSemanticOperation,
			domain.ActionValidateAdmissionCandidate, domain.ActionValidateContextAttribution:
			allowed = true
		}
	case domain.SourceFutureIRIS:
		switch req.Action {
		case domain.ActionCreateNote, domain.ActionCreateLink, domain.ActionRegisterContradict,
			domain.ActionDeriveModel:
			allowed = true
		}
	}
	if !allowed {
		return errors.New("production capability policy denied source/action")
	}
	mutating := def.Mutating || (req.Action == domain.ActionCompileContext && testCompileContextPersists(req.Payload))
	if mutating && (req.Source == domain.SourceAdapter || req.Source == domain.SourceFutureIRIS) {
		approved, _ := req.Metadata["testDurableApprovalVerified"].(bool)
		if !approved {
			return errors.New("production durable approval proof required")
		}
	}
	return nil
}

func TestSelectAuthorityDefaultsToForgeKWithOneCommitAuthority(t *testing.T) {
	delegate := &recordingProcessor{out: domain.SyscallResult{Success: true}}
	selection, err := SelectAuthority(delegate, testAuthorizationPort{})
	if err != nil {
		t.Fatalf("select authority: %v", err)
	}
	if selection.AuthorityOwner != AuthorityOwnerForgeK || !selection.SingleAuthority {
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
	if delegate.committedAuth.AuthorizationFingerprint == "" || result.StateSummary["authorizationProofVerified"] != true {
		t.Fatalf("Kernel did not pass verified authorization to commit: auth=%#v result=%#v", delegate.committedAuth, result.StateSummary)
	}
	if !slices.Equal(delegate.events, []string{"prepare", "commit", "record", "observe"}) {
		t.Fatalf("FORGE-K did not own durable stage order: %v", delegate.events)
	}
}

func TestKernelRequiresIdempotencyForPersistedContextCompile(t *testing.T) {
	delegate := &recordingProcessor{out: domain.SyscallResult{Success: true}}
	selection, err := SelectAuthority(delegate, testAuthorizationPort{})
	if err != nil {
		t.Fatalf("select authority: %v", err)
	}
	req := kernelTestRequest("compile-context-without-idempotency")
	req.Action = domain.ActionCompileContext
	req.RequiredCapability = "context.compile"
	req.Payload = map[string]any{
		"query": "forge context",
		"compileOptions": map[string]any{
			"persistSnapshot": true,
		},
	}
	req.IdempotencyKey = ""
	result, err := selection.Processor.Process(context.Background(), req)
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if result.Success || delegate.prepares != 0 || delegate.calls != 0 || delegate.records != 1 || delegate.observes != 1 {
		t.Fatalf("result=%#v delegate=%#v", result, delegate)
	}
	if len(result.RejectedReasons) != 1 || result.RejectedReasons[0].Code != domain.ErrMissingRequiredField || result.RejectedReasons[0].Field != "idempotencyKey" {
		t.Fatalf("errors=%#v", result.RejectedReasons)
	}

}

func TestForgeKCompleteDispositionNeverCommits(t *testing.T) {
	delegate := &recordingProcessor{
		out:         domain.SyscallResult{Success: false, DeterministicErrCode: domain.ErrCapabilityDenied},
		disposition: DispositionComplete,
	}
	selection, err := SelectAuthority(delegate, testAuthorizationPort{})
	if err != nil {
		t.Fatalf("select authority: %v", err)
	}
	result, err := selection.Processor.Process(context.Background(), kernelTestRequest("sys-complete"))
	if err != nil || result.Success || delegate.calls != 0 {
		t.Fatalf("result=%#v err=%v delegate=%#v", result, err, delegate)
	}
	if !slices.Equal(delegate.events, []string{"prepare", "record", "observe"}) {
		t.Fatalf("completed preflight reached wrong stages: %v", delegate.events)
	}
}

func TestForgeKRejectsMissingOrTamperedAuthorizationBeforePrepare(t *testing.T) {
	tests := []struct {
		name string
		port testAuthorizationPort
	}{
		{name: "resolver failure", port: testAuthorizationPort{err: errors.New("identity authority unavailable")}},
		{name: "tampered origin", port: testAuthorizationPort{mutate: func(proof *authproof.Proof) {
			proof.Origin.SubjectID = "mallory"
		}}},
		{name: "tampered capability", port: testAuthorizationPort{mutate: func(proof *authproof.Proof) {
			proof.Capability.Scope.WorkspaceID = "other"
		}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			delegate := &recordingProcessor{out: domain.SyscallResult{Success: true}}
			selection, err := SelectAuthority(delegate, tc.port)
			if err != nil {
				t.Fatalf("select authority: %v", err)
			}
			result, err := selection.Processor.Process(context.Background(), kernelTestRequest("sys-auth-fail"))
			if !errors.Is(err, ErrInvalidAuthorization) || result.Success || result.DeterministicErrCode != domain.ErrUnauthorized {
				t.Fatalf("result=%#v err=%v", result, err)
			}
			if delegate.prepares != 0 || delegate.calls != 0 || delegate.records != 1 || delegate.observes != 1 {
				t.Fatalf("authorization failure crossed prepare/commit boundary: %#v", delegate)
			}
			if result.StateSummary["authorizationProofVerified"] != false {
				t.Fatalf("authorization failure annotation missing: %#v", result.StateSummary)
			}
		})
	}
}

func TestForgeKRejectsAdapterSuppliedAuthorizationMismatch(t *testing.T) {
	req := authorizedKernelTestRequest("sys-adapter-auth")
	supplied := mustBuildTestAuthorizationProof(t, req)
	supplied.EvidenceSnapshotID = "adapter-injected-snapshot"
	var err error
	supplied, err = authproof.BuildProof(req, supplied)
	if err != nil {
		t.Fatalf("build mismatched proof: %v", err)
	}
	delegate := &recordingProcessor{out: domain.SyscallResult{Success: true}, suppliedAuth: &supplied}
	selection, err := SelectAuthority(delegate, testAuthorizationPort{})
	if err != nil {
		t.Fatalf("select authority: %v", err)
	}
	result, err := selection.Processor.Process(context.Background(), req)
	if !errors.Is(err, ErrInvalidAuthorization) || result.Success || delegate.calls != 0 {
		t.Fatalf("result=%#v err=%v delegate=%#v", result, err, delegate)
	}
}

func TestForgeKRejectsMissingPreparedPlanBeforeCommit(t *testing.T) {
	missing := commitproof.PreparedPlan{}
	delegate := &recordingProcessor{
		out:  domain.SyscallResult{Success: true},
		plan: &missing,
	}
	selection, err := SelectAuthority(delegate, testAuthorizationPort{})
	if err != nil {
		t.Fatalf("select authority: %v", err)
	}
	result, err := selection.Processor.Process(context.Background(), kernelTestRequest("sys-missing-plan"))
	if !errors.Is(err, ErrInvalidAuthorization) {
		t.Fatalf("process error = %v, want ErrInvalidAuthorization", err)
	}
	if result.Success || result.DeterministicErrCode != domain.ErrUnauthorized || delegate.calls != 0 || delegate.records != 1 || delegate.observes != 1 {
		t.Fatalf("result=%#v delegate=%#v", result, delegate)
	}
	if result.StateSummary["authorizationProofVerified"] != false {
		t.Fatalf("missing authorization failure metadata: %#v", result.StateSummary)
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
			selection, err := SelectAuthority(delegate, testAuthorizationPort{})
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
	originalAuthorization := mustBuildTestAuthorizationProof(t, original)
	original.Metadata["forgeKAuthorizationProof"] = originalAuthorization.AuthorizationFingerprint
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
		ReplayAuthorizationProof: originalAuthorization,
	}}
	selection, err := SelectAuthority(delegate, testAuthorizationPort{})
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
	originalAuthorization := mustBuildTestAuthorizationProof(t, original)
	original.Metadata["forgeKAuthorizationProof"] = originalAuthorization.AuthorizationFingerprint
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
		{name: "tampered-authorization", edit: func(p *PreparedSyscall) {
			p.ReplayAuthorizationProof.Capability.RecordID = "grant:tampered"
		}},
		{name: "missing-authorization", edit: func(p *PreparedSyscall) { p.ReplayAuthorizationProof = authproof.Proof{} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			prepared := PreparedSyscall{
				Result: result, Disposition: DispositionReplay,
				ReplayRequest: original, ReplayPlan: cloneCommitPlan(plan), ReplaySeal: seal, ReplayReceipt: receipt,
				ReplayAuthorizationProof: originalAuthorization,
			}
			tc.edit(&prepared)
			delegate := &recordingProcessor{prepared: &prepared}
			selection, err := SelectAuthority(delegate, testAuthorizationPort{})
			if err != nil {
				t.Fatalf("select authority: %v", err)
			}
			got, err := selection.Processor.Process(context.Background(), cloneKernelRequest(original))
			if !errors.Is(err, ErrInvalidCommitReceipt) && !errors.Is(err, ErrInvalidPreparedProof) && !errors.Is(err, ErrInvalidAuthorization) {
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
	originalAuthorization := mustBuildTestAuthorizationProof(t, original)
	original.Metadata["forgeKAuthorizationProof"] = originalAuthorization.AuthorizationFingerprint
	sealedRequest := cloneKernelRequest(original)
	sealedRequest.Metadata[court.MetadataDecisionKey] = decision
	plan := commitproof.PreparedPlan{
		Action: domain.ActionAdmitEvidence, Capability: controllane.CapEvidenceAdmit, TargetObjectType: "court_exhibit_ruling",
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
		ReplayAuthorizationProof: originalAuthorization,
	}}
	selection, err := SelectAuthority(delegate, testAuthorizationPort{})
	if err != nil {
		t.Fatalf("select authority: %v", err)
	}
	result, err := selection.Processor.Process(context.Background(), current)
	if err != nil || !result.Success || result.StateSummary["commitProofVerified"] != true || delegate.calls != 0 {
		t.Fatalf("Court replay: result=%#v err=%v delegate=%#v", result, err, delegate)
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
			selection, err := SelectAuthority(delegate, testAuthorizationPort{})
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
	if _, err := SelectAuthority(nil, testAuthorizationPort{}); !errors.Is(err, ErrMissingCommitAdapter) {
		t.Fatalf("missing adapter error=%v", err)
	}
	if _, err := SelectAuthority(processorWithoutDurablePort{}, testAuthorizationPort{}); !errors.Is(err, ErrMissingDurablePort) {
		t.Fatalf("missing durable port error=%v", err)
	}
	if _, err := SelectAuthority(&recordingProcessor{}, nil); !errors.Is(err, ErrMissingAuthorization) {
		t.Fatalf("missing authorization port error=%v", err)
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
	selection, err := SelectAuthority(commit, testAuthorizationPort{})
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
		req.Provenance.ActorType = req.Actor.Kind
		req.Payload = map[string]any{"noteId": "note-k20b-capability", "reason": "must not commit"}
		result, err := selection.Processor.Process(ctx, req)
		if !errors.Is(err, ErrInvalidAuthorization) || result.Success || result.DeterministicErrCode != domain.ErrUnauthorized {
			t.Fatalf("result=%#v err=%v", result, err)
		}
		assertNoNoteOrJournal(t, st, req)
	})

	t.Run("proposal source requires approval", func(t *testing.T) {
		selection, st, _ := newLiveSQLiteAuthority(t)
		req := liveNoteRequest("k20b-approval", "note-k20b-approval")
		req.Source = domain.SourceFutureIRIS
		req.Actor.Kind = string(domain.SourceFutureIRIS)
		req.Provenance.ActorType = req.Actor.Kind
		result, err := selection.Processor.Process(ctx, req)
		if !errors.Is(err, ErrInvalidAuthorization) || result.Success || result.DeterministicErrCode != domain.ErrUnauthorized {
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
	selection, err := SelectAuthority(adapter, testAuthorizationPort{})
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
		JournalEventType:      "semantic_syscall." + strings.ToLower(string(req.Action)),
		ExpectedObjectIDs:     []string{objectID},
		ExpectedProvenanceIDs: []string{req.ID + ":provenance"},
		Details:               map[string]any{"write": "create_note"},
	}
}

func buildTestAuthorizationProof(req domain.SyscallRequest) (authproof.Proof, error) {
	def, ok := controllane.NewStaticActionRegistry().Get(req.Action)
	if !ok {
		return authproof.Proof{}, errors.New("test authorization registry action missing")
	}
	const credential = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	service := authproof.PrincipalRecord{
		RecordID: "principal:forge.core", Version: "service_identity.v1",
		SubjectID: "forge.core", SubjectKind: "service", Source: domain.SourceSystem,
		Issuer: "forge.bootstrap", CredentialFingerprint: credential,
		Status: authproof.StatusActive, AuthenticatedAt: 1,
	}
	proof := authproof.Proof{
		EvidenceSnapshotID: "test-authorization-snapshot:v1",
		ServicePrincipal:   service,
		Registry: authproof.RegistryRecord{
			RecordID: "action:" + string(req.Action), Version: "test-registry.v1", Authority: "forge_k.registry",
			Action: req.Action, Capability: def.Capability, TargetObjectType: def.TargetObjectType,
			Mutating: def.Mutating, MutationPolicy: authproof.MutationNever, AuthorizedMutating: false,
			SupportsDryRun: def.SupportsDryRun, ApprovalPossible: def.ApprovalPossible,
			JournalEventType: "semantic_syscall." + strings.ToLower(string(req.Action)),
		},
		Capability: authproof.CapabilityRecord{
			RecordID: "grant:" + req.Actor.ID + ":" + def.Capability, Version: "test-capability.v1", Authority: "forge.capabilities",
			SubjectID: req.Actor.ID, SubjectKind: req.Actor.Kind, Source: req.Source,
			Action: req.Action, Capability: def.Capability, Scope: req.Scope,
			Status: authproof.StatusActive, GrantedAt: 1,
		},
		Approval: authproof.ApprovalRecord{
			PolicyRecordID: "approval-policy:test", PolicyVersion: "test-approval.v1", Authority: "forge.approvals",
			Status: authproof.ApprovalNotNeeded,
		},
	}
	if def.Mutating {
		proof.Registry.MutationPolicy = authproof.MutationAlways
		proof.Registry.AuthorizedMutating = true
	} else if req.Action == domain.ActionCompileContext {
		proof.Registry.MutationPolicy = authproof.MutationRequestDependent
		proof.Registry.AuthorizedMutating = testCompileContextPersists(req.Payload)
	}
	switch req.Source {
	case domain.SourceUser, domain.SourceAdapter, domain.SourceFutureIRIS:
		proof.Origin = &authproof.PrincipalRecord{
			RecordID: "authn:" + req.Actor.ID + ":session", Version: "test-identity.v1",
			SubjectID: req.Actor.ID, SubjectKind: req.Actor.Kind, Source: req.Source,
			Issuer: "forge.test.context", CredentialFingerprint: credential,
			Status: authproof.StatusActive, AuthenticatedAt: 1,
		}
	default:
		proof.Capability.SubjectID = service.SubjectID
		proof.Capability.SubjectKind = service.SubjectKind
	}
	if def.Mutating && (req.Source == domain.SourceAdapter || req.Source == domain.SourceFutureIRIS) {
		proof.Approval.Required = true
		proof.Approval.Status = authproof.ApprovalApproved
		proof.Approval.RequestID = "approval-request:test"
		proof.Approval.DecisionID = "approval-decision:test"
		proof.Approval.DecidedBy = "operator:reviewer"
		proof.Approval.DecisionAt = 1
	}
	return authproof.BuildProof(req, proof)
}

func testCompileContextPersists(payload map[string]any) bool {
	persist := false
	apply := func(values map[string]any) {
		if value, ok := values["persistSnapshot"].(bool); ok {
			persist = value
		}
	}
	apply(payload)
	if nested, ok := payload["restoreSnapshot"].(map[string]any); ok {
		apply(nested)
	}
	if nested, ok := payload["compileOptions"].(map[string]any); ok {
		apply(nested)
	}
	return persist
}

func mustBuildTestAuthorizationProof(t *testing.T, req domain.SyscallRequest) authproof.Proof {
	t.Helper()
	proof, err := buildTestAuthorizationProof(req)
	if err != nil {
		t.Fatalf("build test authorization proof: %v", err)
	}
	return proof
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
