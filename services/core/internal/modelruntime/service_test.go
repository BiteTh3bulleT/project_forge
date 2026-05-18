package modelruntime

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

type auditRecorderStub struct {
	mu      sync.Mutex
	records []ModelRuntimeAuditRecord
	nextID  int
}

type generateOutcome struct {
	res GenerateResult
	err error
}

type contextAwareTestBackend struct {
	kind     ModelBackendKind
	generate func(context.Context, GenerateRequest) (GenerateResult, error)
}

func (b *contextAwareTestBackend) Name() string { return "context-aware-test" }

func (b *contextAwareTestBackend) Kind() ModelBackendKind {
	if b.kind == "" {
		return BackendFake
	}
	return b.kind
}

func (b *contextAwareTestBackend) Supports(_ ModelFormat, _ []ModelCapability) bool { return true }

func (b *contextAwareTestBackend) Load(_ context.Context, manifest ModelManifest) (LoadedModel, error) {
	return LoadedModel{
		ModelID:  manifest.ID,
		Backend:  b.Kind(),
		Status:   StatusLoaded,
		LoadedAt: time.Now().UTC(),
	}, nil
}

func (b *contextAwareTestBackend) Unload(_ context.Context, _ string) error { return nil }

func (b *contextAwareTestBackend) Generate(ctx context.Context, req GenerateRequest) (GenerateResult, error) {
	if b.generate != nil {
		return b.generate(ctx, req)
	}
	return GenerateResult{Content: "ok", ModelID: req.ModelID, Backend: b.Kind()}, nil
}

func (b *contextAwareTestBackend) Health(_ context.Context) (BackendHealth, error) {
	return BackendHealth{Name: b.Name(), Kind: b.Kind(), Healthy: true}, nil
}

func (b *contextAwareTestBackend) Inspect(_ context.Context, modelID string) (BackendInspectResult, error) {
	return BackendInspectResult{ModelID: modelID, Backend: b.Kind(), Found: true}, nil
}

func (a *auditRecorderStub) RecordModelRuntime(_ context.Context, record ModelRuntimeAuditRecord) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.records = append(a.records, record)
	a.nextID++
	return "audit-" + strconvItoa(a.nextID), nil
}

func (a *auditRecorderStub) Records() []ModelRuntimeAuditRecord {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]ModelRuntimeAuditRecord, len(a.records))
	copy(out, a.records)
	return out
}

func baseGenerateRequest(modelID string) GenerateRequest {
	return GenerateRequest{
		ModelID:     modelID,
		WorkspaceID: "ws-test",
		Actor:       "unit-test",
		Source:      "service_test",
		Prompt:      "hello world",
	}
}

func completionManifest(id string, backend ModelBackendKind) ModelManifest {
	return ModelManifest{
		ID:           id,
		Backend:      backend,
		Format:       ModelFormatGGUF,
		Capabilities: []ModelCapability{CapabilityCompletion, CapabilityChat},
	}
}

func TestService_LifecycleAndSchedulerBoundary(t *testing.T) {
	backend := NewFakeBackend(FakeBackendOptions{Healthy: true, MaxOutputTokens: 20})
	audit := &auditRecorderStub{}

	svc, err := NewService(ServiceOptions{
		Backends: []ModelBackend{backend},
		Models: []ModelManifest{
			completionManifest("model-a", BackendFake),
			completionManifest("model-b", BackendFake),
		},
		DefaultTimeout:  2 * time.Second,
		MaxOutputTokens: 10,
		Audit:           audit,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	list, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 models, got %d", len(list))
	}
	ids := []string{list[0].Manifest.ID, list[1].Manifest.ID}
	sort.Strings(ids)
	if ids[0] != "model-a" || ids[1] != "model-b" {
		t.Fatalf("unexpected list ordering/content: %+v", ids)
	}

	if _, err := svc.Load(context.Background(), "model-a"); err != nil {
		t.Fatalf("load model-a: %v", err)
	}

	genReq := baseGenerateRequest("model-a")
	genReq.Prompt = "one two three"
	genA, err := svc.Generate(context.Background(), genReq)
	if err != nil {
		t.Fatalf("generate model-a: %v", err)
	}
	if genA.ModelID != "model-a" || genA.Backend != BackendFake {
		t.Fatalf("unexpected generate result: %+v", genA)
	}
	if genA.AuditID == "" {
		t.Fatalf("expected audit id on successful generate")
	}
	assertProposalEnvelopeNoAuthority(t, genA.Proposal, genA, genReq)

	if _, err := svc.Load(context.Background(), "model-b"); err != nil {
		t.Fatalf("load model-b: %v", err)
	}
	notLoadedReq := baseGenerateRequest("model-a")
	notLoadedReq.Prompt = "still loaded?"
	if _, err := svc.Generate(context.Background(), notLoadedReq); !errors.Is(err, ErrModelNotLoaded) {
		t.Fatalf("expected model-a to be unloaded by scheduler boundary, got %v", err)
	}

	if err := svc.Unload(context.Background(), "model-b"); err != nil {
		t.Fatalf("unload model-b: %v", err)
	}
	if err := svc.Unload(context.Background(), "model-b"); !errors.Is(err, ErrModelNotLoaded) {
		t.Fatalf("expected deterministic second unload error, got %v", err)
	}

	health, err := svc.Health(context.Background())
	if err != nil {
		t.Fatalf("health: %v", err)
	}
	if !health.Healthy {
		t.Fatalf("expected service healthy, got %+v", health)
	}

	events := backend.Events()
	expected := []string{"load:model-a", "unload:model-a", "load:model-b", "unload:model-b"}
	if strings.Join(events, ",") != strings.Join(expected, ",") {
		t.Fatalf("unexpected backend events: got %v want %v", events, expected)
	}

	records := audit.Records()
	if len(records) < 5 {
		t.Fatalf("expected multiple audit records, got %d", len(records))
	}
}

func TestService_GenerateWrapsOutputAsProposalOnly(t *testing.T) {
	backend := NewFakeBackend(FakeBackendOptions{Healthy: true})
	svc, err := NewService(ServiceOptions{
		Backends:       []ModelBackend{backend},
		Models:         []ModelManifest{completionManifest("proposal-model", BackendFake)},
		DefaultTimeout: 2 * time.Second,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	if _, err := svc.Load(context.Background(), "proposal-model"); err != nil {
		t.Fatalf("load model: %v", err)
	}
	req := baseGenerateRequest("proposal-model")
	req.CorrelationID = "corr-proposal"
	req.TraceID = "trace-proposal"
	result, err := svc.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	assertProposalEnvelopeNoAuthority(t, result.Proposal, result, req)
	if result.Proposal.CorrelationID != "corr-proposal" || result.Proposal.TraceID != "trace-proposal" {
		t.Fatalf("proposal lost trace identity: %#v", result.Proposal)
	}
	if result.Proposal.OutputHash == "" || result.Proposal.OutputBytes != len([]byte(result.Content)) {
		t.Fatalf("proposal output metadata mismatch: proposal=%#v content=%q", result.Proposal, result.Content)
	}
}

func assertProposalEnvelopeNoAuthority(t *testing.T, proposal *ProposalEnvelope, result GenerateResult, req GenerateRequest) {
	t.Helper()
	if proposal == nil {
		t.Fatalf("expected proposal envelope")
	}
	if proposal.SchemaVersion != "modelruntime.proposal/v1" || proposal.ProposalKind != "model_runtime_output" {
		t.Fatalf("unexpected proposal identity: %#v", proposal)
	}
	if proposal.ModelID != result.ModelID || proposal.Backend != result.Backend || proposal.WorkspaceID != req.WorkspaceID {
		t.Fatalf("proposal lost runtime identity: proposal=%#v result=%#v req=%#v", proposal, result, req)
	}
	if !proposal.ProposalOnly ||
		proposal.CanonicalCommit ||
		proposal.TruthMutation ||
		proposal.MemoryMutation ||
		proposal.EvidenceAdmission ||
		proposal.GatewayExecution ||
		proposal.ModelOutputAuthority ||
		!proposal.RequiresKernelCommit ||
		!proposal.RequiresValidation ||
		proposal.LiveAuthorityMigration {
		t.Fatalf("proposal envelope claimed forbidden authority: %#v", proposal)
	}
}

func TestService_AutoLoadAndBoundedOutput(t *testing.T) {
	backend := NewFakeBackend(FakeBackendOptions{Healthy: true})
	svc, err := NewService(ServiceOptions{
		Backends: []ModelBackend{backend},
		Models: []ModelManifest{{
			ID:             "model-c",
			Backend:        BackendFake,
			Format:         ModelFormatGGUF,
			Capabilities:   []ModelCapability{CapabilityCompletion},
			DefaultRuntime: ModelRuntimeDefaults{MaxTokens: 6},
		}},
		AutoLoad:        true,
		MaxOutputTokens: 3,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	req := baseGenerateRequest("model-c")
	req.Prompt = "alpha beta gamma delta epsilon"
	req.MaxTokens = 10
	res, err := svc.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(strings.Fields(res.Content)) > 3 {
		t.Fatalf("expected output bounded to 3 tokens, got %q", res.Content)
	}
	if res.FinishReason != "length" {
		t.Fatalf("expected finish reason length, got %s", res.FinishReason)
	}

	inspect, err := svc.Inspect(context.Background(), "model-c")
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if inspect.Status != StatusLoaded {
		t.Fatalf("expected loaded status after autoload, got %s", inspect.Status)
	}
	if !inspect.BackendInspect.Found {
		t.Fatalf("expected backend inspect to find loaded model")
	}
}

func TestService_UnknownModel(t *testing.T) {
	backend := NewFakeBackend(FakeBackendOptions{Healthy: true})
	svc, err := NewService(ServiceOptions{
		Backends: []ModelBackend{backend},
		Models:   []ModelManifest{completionManifest("known", BackendFake)},
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	req := baseGenerateRequest("unknown")
	req.Prompt = "x"
	_, err = svc.Generate(context.Background(), req)
	if !errors.Is(err, ErrModelNotFound) {
		t.Fatalf("expected ErrModelNotFound, got %v", err)
	}
}

func TestService_MissingBackendReturnsDeterministicError(t *testing.T) {
	backend := NewFakeBackend(FakeBackendOptions{Healthy: true})
	svc, err := NewService(ServiceOptions{
		Backends: []ModelBackend{backend},
		Models:   []ModelManifest{completionManifest("orphan", BackendLlamaCpp)},
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	if _, err := svc.Inspect(context.Background(), "orphan"); !errors.Is(err, ErrBackendUnavailable) {
		t.Fatalf("inspect expected ErrBackendUnavailable, got %v", err)
	}
	if _, err := svc.Load(context.Background(), "orphan"); !errors.Is(err, ErrBackendUnavailable) {
		t.Fatalf("load expected ErrBackendUnavailable, got %v", err)
	}
	if err := svc.Unload(context.Background(), "orphan"); !errors.Is(err, ErrBackendUnavailable) {
		t.Fatalf("unload expected ErrBackendUnavailable, got %v", err)
	}
	req := baseGenerateRequest("orphan")
	req.Prompt = "ping"
	if _, err := svc.Generate(context.Background(), req); !errors.Is(err, ErrBackendUnavailable) {
		t.Fatalf("generate expected ErrBackendUnavailable, got %v", err)
	}
}

func TestService_ContextValidationAndHook(t *testing.T) {
	backend := NewFakeBackend(FakeBackendOptions{Healthy: true})
	hookErr := errors.New("hook blocked")

	svc, err := NewService(ServiceOptions{
		Backends: []ModelBackend{backend},
		Models:   []ModelManifest{completionManifest("ctx-model", BackendFake)},
		AutoLoad: true,
		RequestValidator: func(_ context.Context, req GenerateRequest) error {
			if strings.TrimSpace(req.WorkspaceID) == "" {
				return ErrWorkspaceRequired
			}
			if req.WorkspaceID == "deny" {
				return hookErr
			}
			return nil
		},
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	missingActor := baseGenerateRequest("ctx-model")
	missingActor.Actor = ""
	if _, err := svc.Generate(context.Background(), missingActor); !errors.Is(err, ErrActorRequired) {
		t.Fatalf("expected ErrActorRequired, got %v", err)
	}

	missingSource := baseGenerateRequest("ctx-model")
	missingSource.Source = ""
	if _, err := svc.Generate(context.Background(), missingSource); !errors.Is(err, ErrSourceRequired) {
		t.Fatalf("expected ErrSourceRequired, got %v", err)
	}

	missingWorkspace := baseGenerateRequest("ctx-model")
	missingWorkspace.WorkspaceID = ""
	if _, err := svc.Generate(context.Background(), missingWorkspace); !errors.Is(err, ErrWorkspaceRequired) {
		t.Fatalf("expected ErrWorkspaceRequired, got %v", err)
	}

	streamReq := baseGenerateRequest("ctx-model")
	streamReq.Stream = true
	if _, err := svc.Generate(context.Background(), streamReq); !errors.Is(err, ErrStreamingUnsupported) {
		t.Fatalf("expected ErrStreamingUnsupported, got %v", err)
	}

	hookBlocked := baseGenerateRequest("ctx-model")
	hookBlocked.WorkspaceID = "deny"
	if _, err := svc.Generate(context.Background(), hookBlocked); !errors.Is(err, hookErr) {
		t.Fatalf("expected hook error, got %v", err)
	}
}

func TestService_CapabilityChecks(t *testing.T) {
	backend := NewFakeBackend(FakeBackendOptions{Healthy: true})

	svc, err := NewService(ServiceOptions{
		Backends: []ModelBackend{backend},
		Models: []ModelManifest{{
			ID:           "embed-only",
			Backend:      BackendFake,
			Format:       ModelFormatGGUF,
			Capabilities: []ModelCapability{CapabilityEmbedding},
		}},
		AutoLoad: true,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	if _, err := svc.Generate(context.Background(), baseGenerateRequest("embed-only")); !errors.Is(err, ErrModelCapabilityUnsupported) {
		t.Fatalf("expected ErrModelCapabilityUnsupported, got %v", err)
	}

	svcChat, err := NewService(ServiceOptions{
		Backends: []ModelBackend{backend},
		Models: []ModelManifest{{
			ID:           "chat-only",
			Backend:      BackendFake,
			Format:       ModelFormatGGUF,
			Capabilities: []ModelCapability{CapabilityChat},
		}},
		AutoLoad: true,
	})
	if err != nil {
		t.Fatalf("new service chat: %v", err)
	}
	chatReq := baseGenerateRequest("chat-only")
	chatReq.Prompt = ""
	chatReq.Messages = []GenerateMessage{{Role: "user", Content: "hello"}}
	if _, err := svcChat.Generate(context.Background(), chatReq); err != nil {
		t.Fatalf("expected chat request to pass capability checks, got %v", err)
	}
}

func TestService_SchedulerAdmissionAndAuditAccounting(t *testing.T) {
	startedCh := make(chan struct{}, 4)
	releaseCh := make(chan struct{}, 4)
	backend := NewFakeBackend(FakeBackendOptions{
		Healthy: true,
		Generate: func(req GenerateRequest) (GenerateResult, error) {
			startedCh <- struct{}{}
			<-releaseCh
			return GenerateResult{Content: "scheduler output"}, nil
		},
	})
	audit := &auditRecorderStub{}

	svc, err := NewService(ServiceOptions{
		Backends:              []ModelBackend{backend},
		Models:                []ModelManifest{completionManifest("sched", BackendFake)},
		AutoLoad:              true,
		Audit:                 audit,
		MaxQueueDepth:         1,
		MaxConcurrentRequests: 1,
		CompletedHistoryLimit: 16,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	firstDone := make(chan generateOutcome, 1)
	secondDone := make(chan generateOutcome, 1)

	firstReq := baseGenerateRequest("sched")
	firstReq.Prompt = "first"
	go func() {
		res, err := svc.Generate(context.Background(), firstReq)
		firstDone <- generateOutcome{res: res, err: err}
	}()

	waitForSignal(t, startedCh, "first request did not start")

	secondReq := baseGenerateRequest("sched")
	secondReq.Prompt = "second"
	go func() {
		res, err := svc.Generate(context.Background(), secondReq)
		secondDone <- generateOutcome{res: res, err: err}
	}()

	waitForSchedulerState(t, svc, func(s SchedulerSnapshot) bool {
		return len(s.Running) == 1 && len(s.Queued) == 1
	}, "expected one running and one queued request")

	thirdReq := baseGenerateRequest("sched")
	thirdReq.Prompt = "third"
	_, err = svc.Generate(context.Background(), thirdReq)
	if !errors.Is(err, ErrRequestQueueFull) {
		t.Fatalf("expected ErrRequestQueueFull, got %v", err)
	}

	time.Sleep(25 * time.Millisecond)
	releaseCh <- struct{}{}
	waitForSignal(t, startedCh, "second request did not start after first completion")
	time.Sleep(25 * time.Millisecond)
	releaseCh <- struct{}{}

	first := waitForGenerateResult(t, firstDone, "first")
	if first.err != nil {
		t.Fatalf("first request error: %v", first.err)
	}
	second := waitForGenerateResult(t, secondDone, "second")
	if second.err != nil {
		t.Fatalf("second request error: %v", second.err)
	}

	snapshot := svc.SchedulerSnapshot()
	if len(snapshot.Queued) != 0 || len(snapshot.Running) != 0 {
		t.Fatalf("expected no queued/running requests after completion, got %+v", snapshot)
	}
	if len(snapshot.Completed) < 3 {
		t.Fatalf("expected completed history to include 3 records, got %d", len(snapshot.Completed))
	}

	var rejected int
	for _, rec := range snapshot.Completed {
		if rec.Outcome == "rejected" {
			rejected++
		}
	}
	if rejected != 1 {
		t.Fatalf("expected exactly one rejected request in completed history, got %d", rejected)
	}

	records := audit.Records()
	var generateRecords []ModelRuntimeAuditRecord
	for _, rec := range records {
		if rec.Operation == "generate" {
			generateRecords = append(generateRecords, rec)
		}
	}
	if len(generateRecords) < 3 {
		t.Fatalf("expected generate audit records, got %d", len(generateRecords))
	}

	hasQueueWait := false
	hasRequestID := true
	hasOutputBytes := false
	for _, rec := range generateRecords {
		if strings.TrimSpace(rec.RequestID) == "" {
			hasRequestID = false
		}
		if rec.QueueWaitMs > 0 {
			hasQueueWait = true
		}
		if rec.Outcome == "ok" && rec.OutputBytes > 0 {
			hasOutputBytes = true
		}
	}
	if !hasRequestID {
		t.Fatalf("expected request IDs on all generate audit records: %+v", generateRecords)
	}
	if !hasQueueWait {
		t.Fatalf("expected at least one queued request with non-zero queue wait: %+v", generateRecords)
	}
	if !hasOutputBytes {
		t.Fatalf("expected successful generate audit record to include output bytes: %+v", generateRecords)
	}
}

func TestService_RunningCancellationRecordsCanceledAccounting(t *testing.T) {
	backend := &contextAwareTestBackend{
		kind: BackendFake,
		generate: func(ctx context.Context, req GenerateRequest) (GenerateResult, error) {
			<-ctx.Done()
			return GenerateResult{}, ctx.Err()
		},
	}
	audit := &auditRecorderStub{}
	svc, err := NewService(ServiceOptions{
		Backends:              []ModelBackend{backend},
		Models:                []ModelManifest{completionManifest("cancel-model", BackendFake)},
		AutoLoad:              true,
		Audit:                 audit,
		CompletedHistoryLimit: 4,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan generateOutcome, 1)
	go func() {
		res, err := svc.Generate(ctx, baseGenerateRequest("cancel-model"))
		done <- generateOutcome{res: res, err: err}
	}()

	waitForSchedulerState(t, svc, func(s SchedulerSnapshot) bool {
		return len(s.Running) == 1
	}, "expected request to enter running state")
	cancel()
	out := waitForGenerateResult(t, done, "canceled request")
	if !errors.Is(out.err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", out.err)
	}

	snapshot := svc.SchedulerSnapshot()
	if len(snapshot.Running) != 0 || len(snapshot.Completed) != 1 {
		t.Fatalf("expected one completed cancellation and no running requests, got %+v", snapshot)
	}
	completed := snapshot.Completed[0]
	if completed.State != RequestStateCanceled || completed.Outcome != "canceled" {
		t.Fatalf("expected canceled scheduler accounting, got %+v", completed)
	}

	records := audit.Records()
	var generateAudit *ModelRuntimeAuditRecord
	for i := range records {
		if records[i].Operation == "generate" {
			generateAudit = &records[i]
		}
	}
	if generateAudit == nil {
		t.Fatalf("expected generate audit record, got %+v", records)
	}
	if generateAudit.Outcome != "canceled" || !strings.Contains(generateAudit.Error, context.Canceled.Error()) {
		t.Fatalf("expected canceled audit accounting, got %+v", *generateAudit)
	}
}

func TestService_SchedulerSnapshotShowsBackpressureReason(t *testing.T) {
	startedCh := make(chan struct{}, 1)
	releaseCh := make(chan struct{}, 1)
	backend := NewFakeBackend(FakeBackendOptions{
		Healthy: true,
		Generate: func(req GenerateRequest) (GenerateResult, error) {
			startedCh <- struct{}{}
			<-releaseCh
			return GenerateResult{Content: "ok"}, nil
		},
	})
	svc, err := NewService(ServiceOptions{
		Backends:              []ModelBackend{backend},
		Models:                []ModelManifest{completionManifest("pressure-visible", BackendFake)},
		AutoLoad:              true,
		GPUEnabled:            true,
		GPUBkgJobsEnabled:     true,
		GPUMaxBackgroundJobs:  1,
		MaxQueueDepth:         4,
		MaxConcurrentRequests: 1,
		SchedulingInteractivePriorityOverBackground: true,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	interactiveDone := make(chan generateOutcome, 1)
	interactiveReq := baseGenerateRequest("pressure-visible")
	interactiveReq.WorkloadClass = GPUWorkloadInteractiveInference
	go func() {
		res, err := svc.Generate(context.Background(), interactiveReq)
		interactiveDone <- generateOutcome{res: res, err: err}
	}()
	waitForSignal(t, startedCh, "interactive request did not start")

	backgroundDone := make(chan generateOutcome, 1)
	backgroundReq := baseGenerateRequest("pressure-visible")
	backgroundReq.WorkloadClass = GPUWorkloadBackgroundEmbedding
	go func() {
		res, err := svc.Generate(context.Background(), backgroundReq)
		backgroundDone <- generateOutcome{res: res, err: err}
	}()

	waitForSchedulerState(t, svc, func(s SchedulerSnapshot) bool {
		return len(s.Running) == 1 &&
			len(s.Queued) == 1 &&
			s.Queued[0].BackpressureReason == "concurrency_limit"
	}, "expected queued background request to expose concurrency backpressure")

	health, err := svc.Health(context.Background())
	if err != nil {
		t.Fatalf("health: %v", err)
	}
	if health.State != RuntimeHealthOverloaded {
		t.Fatalf("expected overloaded health while request is backpressured, got %+v", health)
	}

	releaseCh <- struct{}{}
	if out := waitForGenerateResult(t, interactiveDone, "interactive request"); out.err != nil {
		t.Fatalf("interactive request error: %v", out.err)
	}
	waitForSignal(t, startedCh, "background request did not start")
	releaseCh <- struct{}{}
	if out := waitForGenerateResult(t, backgroundDone, "background request"); out.err != nil {
		t.Fatalf("background request error: %v", out.err)
	}
}

func TestService_MaxOutputBytesBound(t *testing.T) {
	backend := NewFakeBackend(FakeBackendOptions{
		Healthy: true,
		Generate: func(req GenerateRequest) (GenerateResult, error) {
			return GenerateResult{Content: "123456789"}, nil
		},
	})
	svc, err := NewService(ServiceOptions{
		Backends:       []ModelBackend{backend},
		Models:         []ModelManifest{completionManifest("bytes", BackendFake)},
		AutoLoad:       true,
		MaxOutputBytes: 4,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	res, err := svc.Generate(context.Background(), baseGenerateRequest("bytes"))
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len([]byte(res.Content)) > 4 {
		t.Fatalf("expected bounded output bytes <= 4, got %d (%q)", len([]byte(res.Content)), res.Content)
	}
	if !containsString(res.Warnings, ErrOutputBytesBound.Error()) {
		t.Fatalf("expected output-byte warning, got %v", res.Warnings)
	}
	if res.FinishReason != "length" {
		t.Fatalf("expected finish reason length when output bytes are bounded, got %s", res.FinishReason)
	}
}

func TestService_LoadAndUnloadRejectLifecycleBusyAndUnavailable(t *testing.T) {
	backend := NewFakeBackend(FakeBackendOptions{Healthy: true})
	svc, err := NewService(ServiceOptions{
		Backends: []ModelBackend{backend},
		Models:   []ModelManifest{completionManifest("lifecycle", BackendFake)},
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	svc.setStatus("lifecycle", StatusLoading)
	if _, err := svc.Load(context.Background(), "lifecycle"); !errors.Is(err, ErrModelLifecycleBusy) {
		t.Fatalf("expected ErrModelLifecycleBusy on load while loading, got %v", err)
	}
	if err := svc.Unload(context.Background(), "lifecycle"); !errors.Is(err, ErrModelLifecycleBusy) {
		t.Fatalf("expected ErrModelLifecycleBusy on unload while loading, got %v", err)
	}

	svc.setStatus("lifecycle", StatusDisabled)
	if _, err := svc.Load(context.Background(), "lifecycle"); !errors.Is(err, ErrModelUnavailable) {
		t.Fatalf("expected ErrModelUnavailable, got %v", err)
	}
}

func TestService_MaxLoadedModelsLimit(t *testing.T) {
	backendA := NewFakeBackend(FakeBackendOptions{Healthy: true, Kind: BackendFake})
	backendB := NewFakeBackend(FakeBackendOptions{Healthy: true, Kind: BackendLlamaCpp})

	svc, err := NewService(ServiceOptions{
		Backends: []ModelBackend{backendA, backendB},
		Models: []ModelManifest{
			completionManifest("first", BackendFake),
			completionManifest("second", BackendLlamaCpp),
		},
		MaxLoadedModels: 1,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	if _, err := svc.Load(context.Background(), "first"); err != nil {
		t.Fatalf("load first: %v", err)
	}
	if _, err := svc.Load(context.Background(), "second"); !errors.Is(err, ErrLoadedModelLimit) {
		t.Fatalf("expected ErrLoadedModelLimit, got %v", err)
	}
}

func TestService_GPUBackgroundWorkloadDisabledByPolicy(t *testing.T) {
	backend := NewFakeBackend(FakeBackendOptions{Healthy: true, Kind: BackendFake})
	svc, err := NewService(ServiceOptions{
		Backends:          []ModelBackend{backend},
		Models:            []ModelManifest{completionManifest("policy-model", BackendFake)},
		AutoLoad:          true,
		GPUEnabled:        true,
		GPUBkgJobsEnabled: false,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	req := baseGenerateRequest("policy-model")
	req.WorkloadClass = GPUWorkloadBackgroundEmbedding
	_, err = svc.Generate(context.Background(), req)
	if !errors.Is(err, ErrBackgroundJobsDisabled) {
		t.Fatalf("expected ErrBackgroundJobsDisabled, got %v", err)
	}
}

func TestService_GPUInteractiveCanBeBlockedWhenGPURequired(t *testing.T) {
	backend := NewFakeBackend(FakeBackendOptions{Healthy: true, Kind: BackendFake})
	svc, err := NewService(ServiceOptions{
		Backends:                           []ModelBackend{backend},
		Models:                             []ModelManifest{completionManifest("gpu-required", BackendFake)},
		AutoLoad:                           true,
		GPUEnabled:                         false,
		GPURequiredForInteractiveInference: true,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	req := baseGenerateRequest("gpu-required")
	req.Messages = []GenerateMessage{{Role: "user", Content: "hello"}}
	req.Prompt = ""
	_, err = svc.Generate(context.Background(), req)
	if !errors.Is(err, ErrGPUNotAllowedForInteractive) {
		t.Fatalf("expected ErrGPUNotAllowedForInteractive, got %v", err)
	}
}

func TestService_HealthUnavailableWhenGPURequiredBackendsAreUnavailable(t *testing.T) {
	backend := NewFakeBackend(FakeBackendOptions{Healthy: false, Kind: BackendFake})
	svc, err := NewService(ServiceOptions{
		Backends:                []ModelBackend{backend},
		Models:                  []ModelManifest{completionManifest("health-model", BackendFake)},
		GPUEnabled:              true,
		DegradeOnUnavailableGPU: true,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	health, _ := svc.Health(context.Background())
	if health.State != RuntimeHealthUnavailable {
		t.Fatalf("expected runtime health state unavailable, got %s", health.State)
	}
	if health.Healthy {
		t.Fatalf("expected unhealthy runtime when gpu is enabled and unavailable")
	}
	if len(health.DegradedReasons) == 0 {
		t.Fatalf("expected degraded reasons to be populated")
	}
}

func TestService_HealthPreservesDegradedPayloadWhenBackendHealthErrors(t *testing.T) {
	backend := NewFakeBackend(FakeBackendOptions{
		Healthy:      false,
		Kind:         BackendFake,
		HealthDetail: "backend probe failed",
		HealthErr:    errors.New("probe failed"),
	})
	svc, err := NewService(ServiceOptions{
		Backends: []ModelBackend{backend},
		Models:   []ModelManifest{completionManifest("health-error-model", BackendFake)},
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	health, err := svc.Health(context.Background())
	if err != nil {
		t.Fatalf("health should return degraded payload without surfacing probe error: %v", err)
	}
	if health.Healthy || health.State != RuntimeHealthDegraded {
		t.Fatalf("expected degraded unhealthy payload, got %+v", health)
	}
	if len(health.DegradedReasons) == 0 {
		t.Fatalf("expected degraded reason for backend probe error")
	}
}

func TestService_HealthTracksBackendSupervisionFailures(t *testing.T) {
	now := time.Date(2026, 5, 14, 13, 40, 0, 0, time.UTC)
	backend := NewFakeBackend(FakeBackendOptions{
		Healthy:      false,
		Kind:         BackendFake,
		HealthDetail: "probe failed",
		HealthErr:    errors.New("probe failed"),
	})
	svc, err := NewService(ServiceOptions{
		Backends: []ModelBackend{backend},
		Models:   []ModelManifest{completionManifest("supervision-model", BackendFake)},
		Clock:    func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	if _, err := svc.Health(context.Background()); err != nil {
		t.Fatalf("first health should return degraded payload: %v", err)
	}
	health, err := svc.Health(context.Background())
	if err != nil {
		t.Fatalf("second health should return degraded payload: %v", err)
	}

	supervision := health.Backends[BackendFake].Supervision
	if supervision.ConsecutiveFailures != 2 || supervision.LastError != "probe failed" {
		t.Fatalf("unexpected supervision snapshot: %+v", supervision)
	}
	if !supervision.LastProbeAt.Equal(now) {
		t.Fatalf("last probe at=%v, want %v", supervision.LastProbeAt, now)
	}
	if supervision.RestartSupported || supervision.RestartAttempted {
		t.Fatalf("unmanaged backend must not claim restart support: %+v", supervision)
	}
}

func TestService_HealthSupervisionRecommendsOperatorRestartForRepeatedUnmanagedFailures(t *testing.T) {
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	backend := NewFakeBackend(FakeBackendOptions{
		Healthy:      false,
		Kind:         BackendFake,
		HealthDetail: "probe failed",
		HealthErr:    errors.New("probe failed"),
	})
	svc, err := NewService(ServiceOptions{
		Backends: []ModelBackend{backend},
		Models:   []ModelManifest{completionManifest("operator-restart-model", BackendFake)},
		Clock:    func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	if _, err := svc.Health(context.Background()); err != nil {
		t.Fatalf("first health probe should return degraded payload: %v", err)
	}
	health, err := svc.Health(context.Background())
	if err != nil {
		t.Fatalf("second health probe should return degraded payload: %v", err)
	}

	raw, err := json.Marshal(health.Backends[BackendFake].Supervision)
	if err != nil {
		t.Fatalf("marshal supervision: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("unmarshal supervision: %v", err)
	}
	if payload["state"] != "degraded" {
		t.Fatalf("expected degraded supervision state, got %#v", payload)
	}
	if payload["supervisionMode"] != "unmanaged_external_backend" {
		t.Fatalf("expected explicit unmanaged supervision mode, got %#v", payload)
	}
	if payload["restartPolicy"] != "operator_managed_restart" || payload["restartReason"] != "backend_health_probe_failed" {
		t.Fatalf("expected operator restart policy/reason, got %#v", payload)
	}
	if payload["restartRecommended"] != true || payload["requiresOperatorAction"] != true {
		t.Fatalf("expected restart recommendation requiring operator action, got %#v", payload)
	}
	if payload["restartSupported"] != false || payload["restartAttempted"] != false {
		t.Fatalf("unmanaged backend must not claim automated restart support, got %#v", payload)
	}
}

func TestService_HealthExposesResourceLimits(t *testing.T) {
	backend := NewFakeBackend(FakeBackendOptions{Healthy: true, Kind: BackendFake})
	svc, err := NewService(ServiceOptions{
		Backends:                           []ModelBackend{backend},
		Models:                             []ModelManifest{completionManifest("limits-model", BackendFake)},
		DefaultTimeout:                     4 * time.Second,
		MaxPromptTokens:                    8192,
		MaxOutputTokens:                    512,
		MaxOutputBytes:                     4096,
		MaxLoadedModels:                    3,
		LoadTimeout:                        1500 * time.Millisecond,
		UnloadTimeout:                      2500 * time.Millisecond,
		MaxQueueDepth:                      7,
		MaxConcurrentRequests:              2,
		CompletedHistoryLimit:              11,
		GPUEnabled:                         true,
		GPURequiredForInteractiveInference: true,
		GPUBkgJobsEnabled:                  true,
		GPUMaxBackgroundJobs:               2,
		GPUBkgIdleThreshold:                750 * time.Millisecond,
		GPUVRAMHeadroomFraction:            0.25,
		DegradeOnUnavailableGPU:            true,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	health, err := svc.Health(context.Background())
	if err != nil {
		t.Fatalf("health: %v", err)
	}
	raw, err := json.Marshal(health)
	if err != nil {
		t.Fatalf("marshal health: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("unmarshal health: %v", err)
	}
	limits, ok := payload["resourceLimits"].(map[string]any)
	if !ok {
		t.Fatalf("expected resourceLimits in health payload, got %#v", payload)
	}
	assertJSONNumber(t, limits, "maxLoadedModels", 3)
	assertJSONNumber(t, limits, "maxQueueDepth", 7)
	assertJSONNumber(t, limits, "maxConcurrentRequests", 2)
	assertJSONNumber(t, limits, "completedHistoryLimit", 11)
	assertJSONNumber(t, limits, "maxPromptTokens", 8192)
	assertJSONNumber(t, limits, "maxOutputTokens", 512)
	assertJSONNumber(t, limits, "maxOutputBytes", 4096)
	assertJSONNumber(t, limits, "defaultTimeoutMs", 4000)
	assertJSONNumber(t, limits, "loadTimeoutMs", 1500)
	assertJSONNumber(t, limits, "unloadTimeoutMs", 2500)
	assertJSONNumber(t, limits, "gpuMaxBackgroundJobs", 2)
	assertJSONNumber(t, limits, "gpuBackgroundIdleThresholdMs", 750)
	assertJSONNumber(t, limits, "gpuVRAMHeadroomPercent", 25)
	if limits["gpuEnabled"] != true || limits["gpuRequiredForInteractiveInference"] != true || limits["gpuBackgroundJobsEnabled"] != true || limits["degradeOnUnavailableGPU"] != true {
		t.Fatalf("expected GPU policy limits to be visible, got %#v", limits)
	}
}

func TestService_GPUTelemetryPressureBlocksBackgroundJobs(t *testing.T) {
	backend := NewFakeBackend(FakeBackendOptions{Healthy: true, Kind: BackendFake})
	svc, err := NewService(ServiceOptions{
		Backends:          []ModelBackend{backend},
		Models:            []ModelManifest{completionManifest("pressure-model", BackendFake)},
		AutoLoad:          true,
		GPUEnabled:        true,
		GPUBkgJobsEnabled: true,
		GPUTelemetry: func(context.Context) (GPUTelemetrySnapshot, error) {
			return GPUTelemetrySnapshot{
				Enabled:                 true,
				Available:               true,
				Healthy:                 true,
				State:                   "pressure",
				MemoryPressure:          0.95,
				MemoryPressureThreshold: 0.90,
				BackgroundAdmissionOK:   false,
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	req := baseGenerateRequest("pressure-model")
	req.WorkloadClass = GPUWorkloadBackgroundEmbedding
	_, err = svc.Generate(context.Background(), req)
	if !errors.Is(err, ErrBackgroundWorkloadDeferred) {
		t.Fatalf("expected ErrBackgroundWorkloadDeferred under GPU pressure, got %v", err)
	}
	health, _ := svc.Health(context.Background())
	if health.GPUTelemetry == nil || health.GPUTelemetry.BackgroundAdmissionOK {
		t.Fatalf("expected telemetry in health to block background admission: %+v", health.GPUTelemetry)
	}
}

func waitForSchedulerState(t *testing.T, svc *Service, check func(SchedulerSnapshot) bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		if check(svc.SchedulerSnapshot()) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s", msg)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func waitForSignal(t *testing.T, ch <-chan struct{}, msg string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatalf("%s", msg)
	}
}

func waitForGenerateResult(t *testing.T, ch <-chan generateOutcome, label string) generateOutcome {
	t.Helper()
	select {
	case out := <-ch:
		return out
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for %s result", label)
		return generateOutcome{}
	}
}

func assertJSONNumber(t *testing.T, payload map[string]any, key string, want float64) {
	t.Helper()
	got, ok := payload[key].(float64)
	if !ok || got != want {
		t.Fatalf("%s=%#v, want %v in %#v", key, payload[key], want, payload)
	}
}

func containsString(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}

func strconvItoa(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	buf := [20]byte{}
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + (v % 10))
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
