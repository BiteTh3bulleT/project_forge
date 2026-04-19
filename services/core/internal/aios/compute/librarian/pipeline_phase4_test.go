package librarian

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"forge/projectforge/services/core/internal/aios/controllane"
	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/audit"
	"forge/projectforge/services/core/internal/store"
)

type pipelineHarness struct {
	t           *testing.T
	store       *store.Store
	kernel      *controllane.Processor
	pipeline    *IngestPipeline
	repos       CellReadRepositories
	nowMillis   int64
	workspaceID string
}

func newPipelineHarness(t *testing.T, semantic SemanticInferenceService, cells []RuntimeCell, featureFlags map[string]bool) *pipelineHarness {
	t.Helper()
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	txRunner := controllane.NewSQLiteTransactionRunner(st.DB)
	kernel := controllane.NewProcessor(controllane.ProcessorOptions{
		Registry:     controllane.NewStaticActionRegistry(),
		Validator:    controllane.NewDeterministicValidator(),
		Capabilities: controllane.NewStaticCapabilityService(),
		ApprovalGate: controllane.NewStaticApprovalGate(),
		TxRunner:     txRunner,
		AuditSink:    controllane.NewCoreAuditSink(audit.New(st.DB)),
		NowMillis:    func() int64 { return 1761000000000 },
	})
	repos := CellReadRepositories{
		Journal:        controllane.NewSQLiteJournalRepository(st.DB),
		Notes:          controllane.NewSQLiteMemoryNoteRepository(st.DB),
		Links:          controllane.NewSQLiteSemanticLinkRepository(st.DB),
		State:          controllane.NewSQLiteStateRepository(st.DB),
		Loops:          controllane.NewSQLiteOpenLoopRepository(st.DB),
		Artifacts:      controllane.NewSQLiteArtifactRefRepository(st.DB),
		Models:         controllane.NewSQLiteDerivedModelRepository(st.DB),
		Contradictions: controllane.NewSQLiteContradictionRepository(st.DB),
		Supersessions:  controllane.NewSQLiteSupersessionRepository(st.DB),
		ContextPackets: controllane.NewSQLiteContextPacketRepository(st.DB),
	}
	h := &pipelineHarness{
		t:           t,
		store:       st,
		kernel:      kernel,
		repos:       repos,
		nowMillis:   1761000010000,
		workspaceID: "ws-main",
	}
	h.pipeline = NewIngestPipeline(IngestPipelineOptions{
		Kernel:       kernel,
		Repositories: repos,
		Cells:        cells,
		Semantic:     semantic,
		NowMillis:    h.nextMillis,
		FeatureFlags: featureFlags,
	})
	return h
}

func (h *pipelineHarness) nextMillis() int64 {
	h.nowMillis += 11
	return h.nowMillis
}

func (h *pipelineHarness) ingest(ctx context.Context, content string, configure func(*domain.IngestRequest)) domain.IngestResult {
	req := domain.IngestRequest{
		ID:        fmt.Sprintf("ingest-%d", h.nextMillis()),
		InputKind: domain.IngestUserMessage,
		Content:   content,
		Actor: domain.ActorIdentity{
			ID:   "operator",
			Kind: string(domain.SourceUser),
		},
		Source: domain.SourceUser,
		Scope: domain.ForgeScope{
			WorkspaceID: h.workspaceID,
			LaneID:      "control.semantic",
		},
		Provenance: domain.Provenance{
			Actor:     "operator",
			ActorType: "user",
			Source:    "test",
		},
		CorrelationID: "corr-" + fmt.Sprintf("%d", h.nextMillis()),
		TraceID:       "trace-" + fmt.Sprintf("%d", h.nextMillis()),
		CommitMode:    domain.IngestCommitValid,
		RequestedAt:   h.nextMillis(),
	}
	if configure != nil {
		configure(&req)
	}
	res, err := h.pipeline.Run(ctx, req)
	if err != nil {
		h.t.Fatalf("pipeline run failed: %v", err)
	}
	return res
}

func (h *pipelineHarness) seedAction(ctx context.Context, action domain.SemanticActionType, id string, payload map[string]any) domain.SyscallResult {
	req := domain.SyscallRequest{
		ID:     id,
		Action: action,
		Actor: domain.ActorIdentity{
			ID:   "seed",
			Kind: string(domain.SourceUser),
		},
		Source: domain.SourceUser,
		Scope: domain.ForgeScope{
			WorkspaceID: h.workspaceID,
			LaneID:      "control.semantic",
		},
		Payload: payload,
		Provenance: domain.Provenance{
			Actor:     "seed",
			ActorType: "user",
			Source:    "test",
			TraceID:   "trace-" + id,
		},
		CorrelationID: "corr-" + id,
		TraceID:       "trace-" + id,
		RequestedAt:   h.nextMillis(),
	}
	res, err := h.kernel.Process(ctx, req)
	if err != nil {
		h.t.Fatalf("seed action %s failed: %v", action, err)
	}
	if !res.Success {
		h.t.Fatalf("seed action %s rejected: %+v", action, res)
	}
	return res
}

func findDiag(result domain.IngestResult, cellName string) (domain.CellDiagnostic, bool) {
	for _, diag := range result.Diagnostics {
		if diag.CellName == cellName {
			return diag, true
		}
	}
	return domain.CellDiagnostic{}, false
}

func hasAction(actions []domain.SyscallRequest, action domain.SemanticActionType) bool {
	for _, item := range actions {
		if item.Action == action {
			return true
		}
	}
	return false
}

type fakeSemanticInference struct {
	candidates []domain.SyscallRequest
	err        error
}

func (f fakeSemanticInference) ExtractCandidates(_ context.Context, _ InferenceRequest) ([]domain.SyscallRequest, error) {
	return append([]domain.SyscallRequest{}, f.candidates...), f.err
}

func (f fakeSemanticInference) ClassifyCandidate(_ context.Context, candidate domain.SyscallRequest) (domain.SyscallRequest, error) {
	return candidate, nil
}

func (f fakeSemanticInference) SuggestLinks(_ context.Context, _ domain.SyscallRequest, _ InferenceNeighborhood) ([]domain.SyscallRequest, error) {
	return nil, nil
}

func (f fakeSemanticInference) DetectContradictions(_ context.Context, _ domain.SyscallRequest, _ InferenceNeighborhood) ([]domain.SyscallRequest, error) {
	return nil, nil
}

func (f fakeSemanticInference) ProposeModel(_ context.Context, _ domain.ForgeScope, _ InferenceNeighborhood) (*domain.SyscallRequest, error) {
	return nil, nil
}

func (f fakeSemanticInference) SynthesizeSummary(_ context.Context, _ domain.ForgeScope, _ InferenceNeighborhood) (string, error) {
	return "", nil
}

type runtimeCellStub struct {
	name string
	deps []string
}

func (c runtimeCellStub) Name() string                                        { return c.name }
func (runtimeCellStub) Version() string                                       { return "test-v1" }
func (runtimeCellStub) Lane() string                                          { return "compute" }
func (c runtimeCellStub) Dependencies() []string                              { return append([]string{}, c.deps...) }
func (runtimeCellStub) CanRun(context.Context, CellRunContext) (bool, string) { return true, "" }
func (c runtimeCellStub) Run(_ context.Context, _ CellRunContext) (CellRunResult, error) {
	return CellRunResult{CellName: c.name, CellVersion: "test-v1"}, nil
}

func TestPipelineConstructsWithDefaultsAndNoopInference(t *testing.T) {
	h := newPipelineHarness(t, nil, nil, nil)
	if h.pipeline == nil {
		t.Fatalf("expected pipeline")
	}
	if len(h.pipeline.cells) == 0 {
		t.Fatalf("expected default cells")
	}
	if _, ok := h.pipeline.semantic.(NoopSemanticInference); !ok {
		t.Fatalf("expected no-op semantic inference by default")
	}
	if errs := h.pipeline.validateCellDependencies(); len(errs) > 0 {
		t.Fatalf("unexpected dependency errors: %+v", errs)
	}
}

func TestPipelineRejectsDependencyCycles(t *testing.T) {
	cells := []RuntimeCell{
		runtimeCellStub{name: "A", deps: []string{"B"}},
		runtimeCellStub{name: "B", deps: []string{"A"}},
	}
	h := newPipelineHarness(t, nil, cells, nil)
	res := h.ingest(context.Background(), "hello", func(req *domain.IngestRequest) {
		req.CommitMode = domain.IngestValidateOnly
	})
	if res.Success {
		t.Fatalf("expected dependency-cycle rejection")
	}
	if len(res.Errors) == 0 || res.Errors[0].Code != domain.IngestErrCellDependency {
		t.Fatalf("expected dependency error, got %+v", res.Errors)
	}
}

func TestRawEventIngestPreservesCorrelationAndProvenance(t *testing.T) {
	h := newPipelineHarness(t, nil, nil, nil)
	res := h.ingest(context.Background(), "hello", nil)
	if res.EventID == "" || strings.HasPrefix(res.EventID, "virtual-") {
		t.Fatalf("expected persisted event id, got %q", res.EventID)
	}
	evt, ok, err := h.repos.Journal.GetByID(context.Background(), res.EventID)
	if err != nil || !ok {
		t.Fatalf("expected journal event, err=%v ok=%v", err, ok)
	}
	if evt.CorrelationID != res.CorrelationID {
		t.Fatalf("correlation mismatch event=%q result=%q", evt.CorrelationID, res.CorrelationID)
	}
	if evt.Provenance.Actor != "operator" {
		t.Fatalf("unexpected event provenance actor: %q", evt.Provenance.Actor)
	}
}

func TestPreferenceExtractionCommitsNoteAndState(t *testing.T) {
	h := newPipelineHarness(t, nil, nil, nil)
	res := h.ingest(context.Background(), "I prefer concise project snapshots over transcript replay", nil)
	if !res.Success {
		t.Fatalf("expected ingest success, got errors %+v rejected=%d", res.Errors, len(res.RejectedActions))
	}
	if !hasAction(res.ProposedActions, domain.ActionCreateNote) {
		t.Fatalf("expected proposed CREATE_NOTE action")
	}
	scope := controllane.ScopeFilter{WorkspaceID: h.workspaceID, LaneID: "control.semantic"}
	notes, err := h.repos.Notes.ListByType(context.Background(), domain.NotePreference, scope)
	if err != nil {
		t.Fatalf("list preference notes: %v", err)
	}
	if len(notes) == 0 {
		t.Fatalf("expected committed preference note")
	}
	state, ok, err := h.repos.State.GetCurrent(context.Background(), "context_policy", scope)
	if err != nil || !ok {
		t.Fatalf("expected context_policy state update err=%v ok=%v", err, ok)
	}
	if state.Value["value"] != "structured_snapshots" {
		t.Fatalf("expected structured_snapshots state value, got %v", state.Value["value"])
	}
}

func TestLowValueGreetingCreatesNoDurableSemanticAction(t *testing.T) {
	h := newPipelineHarness(t, nil, nil, nil)
	res := h.ingest(context.Background(), "hello", nil)
	if len(res.ProposedActions) != 0 {
		t.Fatalf("expected no proposed actions for greeting, got %d", len(res.ProposedActions))
	}
	scope := controllane.ScopeFilter{WorkspaceID: h.workspaceID, LaneID: "control.semantic"}
	notes, err := h.repos.Notes.ListByScope(context.Background(), scope)
	if err != nil {
		t.Fatalf("list notes: %v", err)
	}
	if len(notes) != 0 {
		t.Fatalf("expected zero notes, got %d", len(notes))
	}
}

func TestGoalDecisionExtractionCanUpdateTestModeState(t *testing.T) {
	h := newPipelineHarness(t, nil, nil, nil)
	res := h.ingest(context.Background(), "We should test this end to end with just FORGE", nil)
	if !res.Success {
		t.Fatalf("expected success, got %+v", res.Errors)
	}
	scope := controllane.ScopeFilter{WorkspaceID: h.workspaceID, LaneID: "control.semantic"}
	state, ok, err := h.repos.State.GetCurrent(context.Background(), "current_test_mode", scope)
	if err != nil || !ok {
		t.Fatalf("expected current_test_mode state err=%v ok=%v", err, ok)
	}
	if state.Value["value"] != "forge_only" {
		t.Fatalf("unexpected state value: %v", state.Value["value"])
	}
	notes, err := h.repos.Notes.ListByScope(context.Background(), scope)
	if err != nil {
		t.Fatalf("list notes: %v", err)
	}
	found := false
	for _, note := range notes {
		if note.Type == domain.NoteGoal || note.Type == domain.NoteDecision {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected goal/decision note from intake")
	}
}

func TestBlockerOpensLoopAndResolutionClosesLoop(t *testing.T) {
	h := newPipelineHarness(t, nil, nil, nil)
	openRes := h.ingest(context.Background(), "The blocker is missing schema support", nil)
	if !openRes.Success {
		t.Fatalf("expected blocker ingest success")
	}
	scope := controllane.ScopeFilter{WorkspaceID: h.workspaceID, LaneID: "control.semantic"}
	loops, err := h.repos.Loops.ListActive(context.Background(), scope, 10)
	if err != nil {
		t.Fatalf("list loops: %v", err)
	}
	if len(loops) == 0 {
		t.Fatalf("expected open loop")
	}
	loopID := loops[0].ID

	closeRes := h.ingest(context.Background(), "Schema support is resolved", nil)
	if !closeRes.Success {
		t.Fatalf("expected resolution ingest success, rejected=%d", len(closeRes.RejectedActions))
	}
	loop, ok, err := h.repos.Loops.GetByID(context.Background(), loopID)
	if err != nil || !ok {
		t.Fatalf("expected loop by id err=%v ok=%v", err, ok)
	}
	if loop.State != domain.LoopResolved {
		t.Fatalf("expected loop resolved, got %s", loop.State)
	}
}

func TestLinkerProposesAndCommitsArtifactLink(t *testing.T) {
	h := newPipelineHarness(t, nil, nil, nil)
	err := h.repos.Artifacts.Create(context.Background(), domain.ArtifactRef{
		ID:          "artifact-schema-1",
		Type:        "document",
		URI:         "docs/schema.md",
		Scope:       domain.ForgeScope{WorkspaceID: h.workspaceID, LaneID: "control.semantic"},
		ContentHash: "sha256:schema",
		CreatedAt:   h.nextMillis(),
		Provenance:  domain.Provenance{Actor: "seed", ActorType: "system", Source: "test"},
		Metadata:    map[string]any{"seed": true},
	})
	if err != nil {
		t.Fatalf("seed artifact: %v", err)
	}

	res := h.ingest(context.Background(), "Remember that artifact-schema-1 is the canonical schema artifact", nil)
	if !res.Success {
		t.Fatalf("expected success, rejected=%d", len(res.RejectedActions))
	}
	scope := controllane.ScopeFilter{WorkspaceID: h.workspaceID, LaneID: "control.semantic"}
	links, err := h.repos.Links.ListByTarget(context.Background(), "artifact-schema-1", scope, 20)
	if err != nil {
		t.Fatalf("list links by target: %v", err)
	}
	found := false
	for _, link := range links {
		if link.Type == domain.LinkAbout {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected about link to artifact")
	}
}

func TestContradictionCellProposesSupersessionAndPreservesOldRecord(t *testing.T) {
	h := newPipelineHarness(t, nil, nil, nil)
	h.seedAction(context.Background(), domain.ActionCreateNote, "seed-note-transcript", map[string]any{
		"id":      "note-transcript",
		"type":    string(domain.NotePreference),
		"title":   "Preference: transcript replay",
		"content": "I prefer transcript replay for recall",
		"status":  string(domain.NoteActive),
	})

	res := h.ingest(context.Background(), "I prefer concise project snapshots instead of transcript replay", nil)
	if !res.Success {
		t.Fatalf("expected supersession ingest success, rejected=%d errors=%+v", len(res.RejectedActions), res.Errors)
	}
	scope := controllane.ScopeFilter{WorkspaceID: h.workspaceID, LaneID: "control.semantic"}
	supers, err := h.repos.Supersessions.ListByOldObject(context.Background(), "note-transcript", scope, 10)
	if err != nil {
		t.Fatalf("list supersessions: %v", err)
	}
	if len(supers) == 0 {
		t.Fatalf("expected supersession record")
	}
	oldNote, ok, err := h.repos.Notes.GetByID(context.Background(), "note-transcript")
	if err != nil || !ok {
		t.Fatalf("expected old note to remain, err=%v ok=%v", err, ok)
	}
	if oldNote.Status != domain.NoteSuperseded {
		t.Fatalf("expected old note superseded status, got %s", oldNote.Status)
	}
}

func TestPatternProposalThresholdAndContraryEvidenceBehavior(t *testing.T) {
	h := newPipelineHarness(t, nil, nil, nil)
	h.ingest(context.Background(), "I prefer concise project snapshots over transcript replay", nil)
	scope := controllane.ScopeFilter{WorkspaceID: h.workspaceID, LaneID: "control.semantic"}
	modelsBefore, err := h.repos.Models.ListByType(context.Background(), "context_policy_preference", scope, 20)
	if err != nil {
		t.Fatalf("list models before: %v", err)
	}
	if len(modelsBefore) != 0 {
		t.Fatalf("expected no model before threshold")
	}

	res := h.ingest(context.Background(), "Remember that structured memory snapshots are preferred for project context", nil)
	if !res.Success {
		t.Fatalf("expected second ingest success")
	}
	modelsAfter, err := h.repos.Models.ListByType(context.Background(), "context_policy_preference", scope, 20)
	if err != nil {
		t.Fatalf("list models after: %v", err)
	}
	if len(modelsAfter) == 0 {
		t.Fatalf("expected derived model after repeated evidence")
	}
	if modelsAfter[0].Status != domain.ModelProvisional {
		t.Fatalf("expected provisional model status")
	}

	// Unit-level contrary-evidence check for deterministic suppression.
	cell := PatternRuntimeCell{SupportThreshold: 2}
	in := CellRunContext{
		Request: domain.IngestRequest{RequestedAt: h.nextMillis()},
		Event:   domain.JournalEvent{ID: "evt-contrary"},
		Scope:   domain.ForgeScope{WorkspaceID: h.workspaceID},
		ActiveNotes: []domain.MemoryNote{
			{ID: "n1", Title: "snapshots", Content: "structured memory snapshots"},
			{ID: "n2", Title: "replay", Content: "transcript replay is preferred"},
		},
	}
	out, err := cell.Run(context.Background(), in)
	if err != nil {
		t.Fatalf("pattern cell run: %v", err)
	}
	if len(out.ProposedActions) != 0 {
		t.Fatalf("expected contrary evidence to suppress model proposal")
	}
}

func TestRecallHintsDoNotMutateByDefault(t *testing.T) {
	h := newPipelineHarness(t, nil, nil, nil)
	h.seedAction(context.Background(), domain.ActionUpdateState, "seed-state-policy", map[string]any{
		"id":          "state-policy",
		"key":         "context_policy",
		"value":       map[string]any{"value": "structured_snapshots"},
		"derivedFrom": []string{"seed"},
	})
	h.seedAction(context.Background(), domain.ActionOpenLoop, "seed-loop-schema", map[string]any{
		"id":       "loop-schema",
		"title":    "Schema support backlog",
		"state":    string(domain.LoopOpen),
		"priority": "high",
	})

	res := h.ingest(context.Background(), "Need context policy and open blockers", nil)
	if !res.Success {
		t.Fatalf("expected success")
	}
	if hasAction(res.ProposedActions, domain.ActionCompileContext) {
		t.Fatalf("did not expect compile context action without explicit request")
	}
	diag, ok := findDiag(res, cellRecall)
	if !ok {
		t.Fatalf("expected recall diagnostic")
	}
	hints, ok := diag.Metadata["hints"].(map[string]any)
	if !ok {
		t.Fatalf("expected hints metadata")
	}
	keys, ok := hints["stateKeys"].([]string)
	if !ok {
		// SQLite JSON map decode in metadata can materialize []any depending path.
		if asAny, okAny := hints["stateKeys"].([]any); okAny {
			keys = make([]string, 0, len(asAny))
			for _, item := range asAny {
				keys = append(keys, fmt.Sprintf("%v", item))
			}
		}
	}
	if len(keys) == 0 {
		t.Fatalf("expected recall state key hints")
	}
}

func TestCleanupIsConservativeAndNoHardArchiveWithoutExplicitRule(t *testing.T) {
	h := newPipelineHarness(t, nil, nil, nil)
	h.seedAction(context.Background(), domain.ActionCreateNote, "seed-note-active", map[string]any{
		"id":      "note-active",
		"type":    string(domain.NoteFact),
		"title":   "Active",
		"content": "active note",
		"status":  string(domain.NoteActive),
	})
	res := h.ingest(context.Background(), "hello", nil)
	if !res.Success {
		t.Fatalf("expected success")
	}
	if hasAction(res.ProposedActions, domain.ActionArchiveNote) {
		t.Fatalf("cleanup must not archive without explicit metadata rule")
	}
	note, ok, err := h.repos.Notes.GetByID(context.Background(), "note-active")
	if err != nil || !ok {
		t.Fatalf("expected seeded note, err=%v ok=%v", err, ok)
	}
	if note.Status != domain.NoteActive {
		t.Fatalf("expected active note status unchanged")
	}
}

func TestValidateOnlyAndDryRunPerformNoSemanticWrites(t *testing.T) {
	h := newPipelineHarness(t, nil, nil, nil)
	validateOnly := h.ingest(context.Background(), "I prefer deterministic kernels", func(req *domain.IngestRequest) {
		req.CommitMode = domain.IngestValidateOnly
	})
	if validateOnly.Success && len(validateOnly.RejectedActions) > 0 {
		t.Fatalf("validate_only should report rejections deterministically without commit")
	}
	if !strings.HasPrefix(validateOnly.EventID, "virtual-") {
		t.Fatalf("expected virtual event in validate_only mode")
	}
	scope := controllane.ScopeFilter{WorkspaceID: h.workspaceID, LaneID: "control.semantic"}
	notes, err := h.repos.Notes.ListByScope(context.Background(), scope)
	if err != nil {
		t.Fatalf("list notes after validate_only: %v", err)
	}
	if len(notes) != 0 {
		t.Fatalf("validate_only should not persist notes")
	}

	dryRun := h.ingest(context.Background(), "I prefer concise artifacts", func(req *domain.IngestRequest) {
		req.DryRun = true
		req.CommitMode = domain.IngestCommitValid
	})
	if !dryRun.DryRun {
		t.Fatalf("expected dryrun result flag")
	}
	notes, err = h.repos.Notes.ListByScope(context.Background(), scope)
	if err != nil {
		t.Fatalf("list notes after dryrun: %v", err)
	}
	if len(notes) != 0 {
		t.Fatalf("dryrun should not persist notes")
	}
}

func TestCommitValidCommitsValidAndReportsInvalidActions(t *testing.T) {
	semantic := fakeSemanticInference{
		candidates: []domain.SyscallRequest{
			{
				ID:     "bad-note",
				Action: domain.ActionCreateNote,
				Actor:  domain.ActorIdentity{ID: "semantic.fake", Kind: string(domain.SourceInternal)},
				Source: domain.SourceInternal,
				Payload: map[string]any{
					"id":    "note-invalid",
					"type":  string(domain.NoteFact),
					"title": "missing content",
				},
			},
		},
	}
	h := newPipelineHarness(t, semantic, nil, nil)
	res := h.ingest(context.Background(), "I prefer concise project snapshots", nil)
	if len(res.AcceptedActions) == 0 {
		t.Fatalf("expected valid actions to commit")
	}
	if len(res.RejectedActions) == 0 {
		t.Fatalf("expected invalid injected action to be rejected")
	}
}

func TestIdempotentRepeatedIngestDoesNotDuplicateCommittedNotes(t *testing.T) {
	h := newPipelineHarness(t, nil, nil, nil)
	ctx := context.Background()
	baseID := "ingest-idem-1"
	res1 := h.ingest(ctx, "I prefer deterministic summaries", func(req *domain.IngestRequest) {
		req.ID = baseID
		req.CorrelationID = "corr-idem-1"
		req.TraceID = "trace-idem-1"
		req.IdempotencyKey = "idem-ingest-1"
	})
	if !res1.Success {
		t.Fatalf("first ingest should succeed")
	}
	res2 := h.ingest(ctx, "I prefer deterministic summaries", func(req *domain.IngestRequest) {
		req.ID = baseID
		req.CorrelationID = "corr-idem-1"
		req.TraceID = "trace-idem-1"
		req.IdempotencyKey = "idem-ingest-1"
	})
	if !res2.Success {
		t.Fatalf("second ingest replay should succeed, errors=%+v", res2.Errors)
	}
	scope := controllane.ScopeFilter{WorkspaceID: h.workspaceID, LaneID: "control.semantic"}
	notes, err := h.repos.Notes.ListByScope(ctx, scope)
	if err != nil {
		t.Fatalf("list notes: %v", err)
	}
	if len(notes) != 1 {
		t.Fatalf("expected one note after idempotent replay, got %d", len(notes))
	}
	if !strings.Contains(strings.Join(res2.Warnings, " | "), "reused existing journal event") {
		t.Fatalf("expected journal replay warning, got %v", res2.Warnings)
	}
}

func TestFutureIrisSemanticInjectionCannotBypassKernel(t *testing.T) {
	semantic := fakeSemanticInference{
		candidates: []domain.SyscallRequest{
			{
				ID:     "iris-note-candidate",
				Action: domain.ActionCreateNote,
				Actor:  domain.ActorIdentity{ID: "iris.service", Kind: string(domain.SourceFutureIRIS)},
				Source: domain.SourceFutureIRIS,
				Payload: map[string]any{
					"id":      "note-iris-candidate",
					"type":    string(domain.NoteFact),
					"title":   "IRIS candidate",
					"content": "candidate content",
				},
			},
		},
	}
	h := newPipelineHarness(t, semantic, nil, nil)
	res := h.ingest(context.Background(), "hello", nil)
	if res.Success {
		t.Fatalf("expected rejection due approval gate for future_iris mutation")
	}
	if len(res.RejectedActions) == 0 {
		t.Fatalf("expected rejected action")
	}
	gotCode := res.RejectedActions[0].Result.DeterministicErrCode
	if gotCode != domain.ErrApprovalRequired {
		t.Fatalf("expected approval required code, got %s", gotCode)
	}
	_, ok, err := h.repos.Notes.GetByID(context.Background(), "note-iris-candidate")
	if err != nil {
		t.Fatalf("get note: %v", err)
	}
	if ok {
		t.Fatalf("future_iris action must not commit directly")
	}
}

func TestWorkspaceIsolationAcrossIngestRuns(t *testing.T) {
	h := newPipelineHarness(t, nil, nil, nil)
	h.workspaceID = "ws-alpha"
	h.ingest(context.Background(), "I prefer concise snapshots", nil)
	h.workspaceID = "ws-beta"
	h.ingest(context.Background(), "I prefer concise snapshots", nil)

	wsAlpha, err := h.repos.Notes.ListByScope(context.Background(), controllane.ScopeFilter{WorkspaceID: "ws-alpha", LaneID: "control.semantic"})
	if err != nil {
		t.Fatalf("list ws-alpha notes: %v", err)
	}
	wsBeta, err := h.repos.Notes.ListByScope(context.Background(), controllane.ScopeFilter{WorkspaceID: "ws-beta", LaneID: "control.semantic"})
	if err != nil {
		t.Fatalf("list ws-beta notes: %v", err)
	}
	if len(wsAlpha) == 0 || len(wsBeta) == 0 {
		t.Fatalf("expected notes in both workspaces")
	}
	for _, note := range wsAlpha {
		if note.Scope.WorkspaceID != "ws-alpha" {
			t.Fatalf("workspace leak in alpha query: %+v", note.Scope)
		}
	}
	for _, note := range wsBeta {
		if note.Scope.WorkspaceID != "ws-beta" {
			t.Fatalf("workspace leak in beta query: %+v", note.Scope)
		}
	}
}

func TestAuditAndProvenanceTraceFromEventToCommittedAndRejectedActions(t *testing.T) {
	semantic := fakeSemanticInference{
		candidates: []domain.SyscallRequest{
			{
				ID:     "invalid-note-candidate",
				Action: domain.ActionCreateNote,
				Actor:  domain.ActorIdentity{ID: "semantic.fake", Kind: string(domain.SourceInternal)},
				Source: domain.SourceInternal,
				Payload: map[string]any{
					"id":    "invalid-note-candidate",
					"type":  string(domain.NoteFact),
					"title": "missing content",
				},
			},
		},
	}
	h := newPipelineHarness(t, semantic, nil, nil)
	res := h.ingest(context.Background(), "I prefer concise project snapshots", nil)
	if len(res.AuditIDs) == 0 {
		t.Fatalf("expected syscall audit ids")
	}
	if len(res.AcceptedActions) == 0 || len(res.RejectedActions) == 0 {
		t.Fatalf("expected both accepted and rejected outcomes")
	}
	accepted := res.AcceptedActions[0]
	if accepted.CellName == "" || accepted.CandidateBatch == "" {
		t.Fatalf("expected accepted action to include cell trace metadata: %+v", accepted)
	}
	rejected := res.RejectedActions[0]
	if rejected.CellName == "" || rejected.Result.DeterministicErrCode == "" {
		t.Fatalf("expected rejected action with provenance and deterministic reason: %+v", rejected)
	}
	if accepted.Result.CorrelationID != res.CorrelationID {
		t.Fatalf("expected end-to-end correlation trace, accepted=%q ingest=%q", accepted.Result.CorrelationID, res.CorrelationID)
	}
}

func TestIngestResultIncludesTruthDiagnosticsApplySummaries(t *testing.T) {
	h := newPipelineHarness(t, nil, nil, nil)
	res := h.ingest(context.Background(), "I prefer concise project snapshots over transcript replay", nil)
	if len(res.AcceptedActions) == 0 {
		t.Fatalf("expected accepted actions")
	}
	rawApply, ok := res.TruthDiagnostics["apply"]
	if !ok {
		t.Fatalf("expected truth diagnostics apply entries")
	}
	apply, ok := rawApply.([]domain.TruthApplySummary)
	if !ok {
		t.Fatalf("expected []domain.TruthApplySummary, got %T", rawApply)
	}
	if len(apply) != len(res.AcceptedActions)+len(res.RejectedActions) {
		t.Fatalf("expected truth apply entries to track action outcomes, apply=%d accepted=%d rejected=%d",
			len(apply), len(res.AcceptedActions), len(res.RejectedActions))
	}
	if apply[0].Action == "" || apply[0].RequestID == "" {
		t.Fatalf("expected structured truth apply summary metadata")
	}
}

func TestRepeatedSamePreferenceDoesNotChurnStateTimeline(t *testing.T) {
	h := newPipelineHarness(t, nil, nil, nil)
	ctx := context.Background()
	message := "I prefer concise project snapshots over transcript replay"
	first := h.ingest(ctx, message, nil)
	if !first.Success {
		t.Fatalf("first ingest failed: %+v", first.Errors)
	}
	scope := controllane.ScopeFilter{WorkspaceID: h.workspaceID, LaneID: "control.semantic"}
	timelineAfterFirst, err := h.repos.State.GetTimeline(ctx, "context_policy", scope, 20)
	if err != nil {
		t.Fatalf("get timeline after first: %v", err)
	}
	if len(timelineAfterFirst) == 0 {
		t.Fatalf("expected timeline entry after first ingest")
	}
	second := h.ingest(ctx, message, nil)
	if !second.Success {
		t.Fatalf("second ingest failed: %+v", second.Errors)
	}
	timelineAfterSecond, err := h.repos.State.GetTimeline(ctx, "context_policy", scope, 20)
	if err != nil {
		t.Fatalf("get timeline after second: %v", err)
	}
	if len(timelineAfterSecond) != len(timelineAfterFirst) {
		t.Fatalf("expected no state churn for same current value, first=%d second=%d",
			len(timelineAfterFirst), len(timelineAfterSecond))
	}
}

func TestOrderAndDedupeMergesProvenanceAndEnforcesActionOrdering(t *testing.T) {
	h := newPipelineHarness(t, nil, nil, nil)
	actions := []domain.SyscallRequest{
		{
			ID:     "b-link",
			Action: domain.ActionCreateLink,
			Scope:  domain.ForgeScope{WorkspaceID: "ws"},
			Payload: map[string]any{
				"type":     string(domain.LinkRelatesTo),
				"sourceId": "note-a",
				"targetId": "note-b",
			},
			Metadata: map[string]any{"proposedByCells": []string{"LinkerCell"}},
		},
		{
			ID:     "a-note",
			Action: domain.ActionCreateNote,
			Scope:  domain.ForgeScope{WorkspaceID: "ws"},
			Payload: map[string]any{
				"type":    string(domain.NoteFact),
				"title":   "Same",
				"content": "same",
			},
			Metadata: map[string]any{"proposedByCells": []string{"IntakeCell"}},
		},
		{
			ID:     "a-note-duplicate",
			Action: domain.ActionCreateNote,
			Scope:  domain.ForgeScope{WorkspaceID: "ws"},
			Payload: map[string]any{
				"type":    string(domain.NoteFact),
				"title":   "Same",
				"content": "same",
			},
			Metadata: map[string]any{"proposedByCells": []string{"CatalogCell"}},
		},
	}
	out := h.pipeline.orderAndDedupe(actions)
	if len(out) != 2 {
		t.Fatalf("expected dedupe to collapse duplicate note, got %d", len(out))
	}
	if out[0].Action != domain.ActionCreateNote || out[1].Action != domain.ActionCreateLink {
		t.Fatalf("expected note before link ordering, got %s then %s", out[0].Action, out[1].Action)
	}
	cells := readStringSliceAny(out[0].Metadata, "proposedByCells")
	if len(cells) < 2 {
		t.Fatalf("expected merged provenance cells in deduped action, got %v", cells)
	}
}

func TestCommitAllOrFailReturnsDeterministicUnsupported(t *testing.T) {
	h := newPipelineHarness(t, nil, nil, nil)
	res := h.ingest(context.Background(), "I prefer concise snapshots", func(req *domain.IngestRequest) {
		req.CommitMode = domain.IngestCommitAllOrFail
	})
	if res.Success {
		t.Fatalf("expected unsupported commit_all_or_fail result")
	}
	found := false
	for _, err := range res.Errors {
		if err.Code == domain.IngestErrUnsupported {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected unsupported mode error, got %+v", res.Errors)
	}
}
