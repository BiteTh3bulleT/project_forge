package controllane

import (
	"context"
	"strings"
	"testing"

	"forge/projectforge/services/core/internal/aios/domain"
)

type allowAllApprovalGate struct{}

func (allowAllApprovalGate) Evaluate(_ context.Context, _ domain.SyscallRequest, _ ActionDefinition) (ApprovalDecision, error) {
	return ApprovalDecision{Status: domain.ApprovalAllowed, Reason: "test override"}, nil
}

func TestProcessorConstructsAndWires(t *testing.T) {
	k, _, _ := newTestKernel()
	if k == nil {
		t.Fatalf("expected kernel processor instance")
	}
}

func TestProcessorSupportedSyscalls(t *testing.T) {
	ctx := context.Background()
	k, store, _ := newTestKernel()

	createNote := validBaseRequest(domain.ActionCreateNote)
	createNote.ID = "create-note-1"
	createNote.Payload = map[string]any{
		"id":      "note-1",
		"type":    string(domain.NoteFact),
		"title":   "Title",
		"content": "Body",
	}
	res, err := k.Process(ctx, createNote)
	if err != nil || !res.Success {
		t.Fatalf("CREATE_NOTE failed: err=%v res=%+v", err, res)
	}

	createNote2 := validBaseRequest(domain.ActionCreateNote)
	createNote2.ID = "create-note-2"
	createNote2.Payload = map[string]any{
		"id":      "note-2",
		"type":    string(domain.NoteFact),
		"title":   "Title2",
		"content": "Body2",
	}
	if res2, err := k.Process(ctx, createNote2); err != nil || !res2.Success {
		t.Fatalf("second CREATE_NOTE failed: err=%v res=%+v", err, res2)
	}

	createLink := validBaseRequest(domain.ActionCreateLink)
	createLink.ID = "create-link-1"
	createLink.Payload = map[string]any{
		"id":       "link-1",
		"type":     string(domain.LinkSupports),
		"sourceId": "note-1",
		"targetId": "note-2",
	}
	if res, err = k.Process(ctx, createLink); err != nil || !res.Success {
		t.Fatalf("CREATE_LINK failed: err=%v res=%+v", err, res)
	}

	updateState := validBaseRequest(domain.ActionUpdateState)
	updateState.ID = "update-state-1"
	updateState.Payload = map[string]any{
		"id":          "state-1",
		"key":         "runtime.mode",
		"value":       map[string]any{"value": "deterministic"},
		"derivedFrom": []string{"note-1"},
	}
	if res, err = k.Process(ctx, updateState); err != nil || !res.Success {
		t.Fatalf("UPDATE_STATE failed: err=%v res=%+v", err, res)
	}

	openLoop := validBaseRequest(domain.ActionOpenLoop)
	openLoop.ID = "open-loop-1"
	openLoop.Payload = map[string]any{
		"id":       "loop-1",
		"title":    "Do thing",
		"state":    string(domain.LoopOpen),
		"priority": "medium",
	}
	if res, err = k.Process(ctx, openLoop); err != nil || !res.Success {
		t.Fatalf("OPEN_LOOP failed: err=%v res=%+v", err, res)
	}

	closeLoop := validBaseRequest(domain.ActionCloseLoop)
	closeLoop.ID = "close-loop-1"
	closeLoop.Payload = map[string]any{
		"loopId": "loop-1",
		"reason": "completed",
	}
	if res, err = k.Process(ctx, closeLoop); err != nil || !res.Success {
		t.Fatalf("CLOSE_LOOP failed: err=%v res=%+v", err, res)
	}

	markSuperseded := validBaseRequest(domain.ActionMarkSuperseded)
	markSuperseded.ID = "mark-sup-1"
	markSuperseded.Payload = map[string]any{
		"oldObjectId": "note-1",
		"newObjectId": "note-2",
		"reason":      "new note supersedes old",
	}
	if res, err = k.Process(ctx, markSuperseded); err != nil || !res.Success {
		t.Fatalf("MARK_SUPERSEDED failed: err=%v res=%+v", err, res)
	}

	registerContradiction := validBaseRequest(domain.ActionRegisterContradict)
	registerContradiction.ID = "contradict-1"
	registerContradiction.Payload = map[string]any{
		"leftObjectId":  "note-1",
		"rightObjectId": "note-2",
		"reason":        "claims differ",
		"severity":      "medium",
		"confidence":    0.8,
	}
	if res, err = k.Process(ctx, registerContradiction); err != nil || !res.Success {
		t.Fatalf("REGISTER_CONTRADICTION failed: err=%v res=%+v", err, res)
	}

	deriveModel := validBaseRequest(domain.ActionDeriveModel)
	deriveModel.ID = "model-derive-1"
	deriveModel.Payload = map[string]any{
		"id":           "model-1",
		"type":         "routing",
		"expression":   "score = evidence * confidence",
		"derivedFrom":  []string{"note-1", "note-2"},
		"supportCount": 2,
		"confidence":   0.6,
	}
	if res, err = k.Process(ctx, deriveModel); err != nil || !res.Success {
		t.Fatalf("DERIVE_MODEL failed: err=%v res=%+v", err, res)
	}

	archiveNote := validBaseRequest(domain.ActionArchiveNote)
	archiveNote.ID = "archive-note-1"
	archiveNote.Payload = map[string]any{
		"noteId": "note-2",
		"reason": "cleanup",
	}
	if res, err = k.Process(ctx, archiveNote); err != nil || !res.Success {
		t.Fatalf("ARCHIVE_NOTE failed: err=%v res=%+v", err, res)
	}
	if note, ok := store.FindNote("note-2"); !ok || note.Status != domain.NoteArchived {
		t.Fatalf("expected note-2 archived and preserved, got exists=%v note=%+v", ok, note)
	}

	compile := validBaseRequest(domain.ActionCompileContext)
	compile.ID = "compile-context-1"
	compile.Payload = map[string]any{
		"query": "summarize state",
		"budget": map[string]any{
			"maxTokens": 1000,
			"maxEvents": 20,
			"maxNotes":  20,
		},
	}
	res, err = k.Process(ctx, compile)
	if err != nil || !res.Success {
		t.Fatalf("COMPILE_CONTEXT failed: err=%v res=%+v", err, res)
	}
	if len(res.CommittedObjectIDs) != 0 {
		t.Fatalf("compile context should be read-only")
	}
}

func TestProcessorOpenLoopUpdatePreservesTransitionRules(t *testing.T) {
	ctx := context.Background()
	k, store, _ := newTestKernel()
	mustCreateNote(ctx, k, "note-a", "note a")
	mustCreateNote(ctx, k, "note-b", "note b")
	mustCreateNote(ctx, k, "note-c", "note c")

	create := validBaseRequest(domain.ActionOpenLoop)
	create.ID = "open-loop-update-create"
	create.Payload = map[string]any{
		"id":           "loop-update-1",
		"title":        "Initial loop",
		"state":        string(domain.LoopOpen),
		"priority":     "low",
		"owner":        "operator-a",
		"nextAction":   "inspect",
		"relatedNotes": []string{"note-a"},
	}
	res, err := k.Process(ctx, create)
	if err != nil || !res.Success {
		t.Fatalf("create open loop failed: err=%v res=%+v", err, res)
	}

	update := validBaseRequest(domain.ActionOpenLoop)
	update.ID = "open-loop-update-valid"
	update.Payload = map[string]any{
		"id":           "loop-update-1",
		"title":        "Updated loop",
		"state":        string(domain.LoopInProgress),
		"priority":     "high",
		"owner":        "operator-b",
		"blocker":      "none",
		"nextAction":   "finish",
		"relatedNotes": []string{"note-b", "note-c"},
	}
	res, err = k.Process(ctx, update)
	if err != nil || !res.Success {
		t.Fatalf("update open loop failed: err=%v res=%+v", err, res)
	}
	if len(res.CommittedObjectIDs) != 2 || !stringInSlice(res.CommittedObjectIDs, "loop-update-1") ||
		!stringInSlice(res.CommittedObjectIDs, "open-loop-update-valid:journal_event") {
		t.Fatalf("committed ids = %v", res.CommittedObjectIDs)
	}
	loop, ok := store.FindLoop("loop-update-1")
	if !ok {
		t.Fatalf("expected loop-update-1 to exist")
	}
	if loop.Title != "Updated loop" || loop.State != domain.LoopInProgress || loop.Priority != "high" || loop.Owner != "operator-b" || loop.Blocker != "none" || loop.NextAction != "finish" {
		t.Fatalf("loop update did not persist expected fields: %+v", loop)
	}
	if len(loop.RelatedNotes) != 2 || loop.RelatedNotes[0] != "note-b" || loop.RelatedNotes[1] != "note-c" {
		t.Fatalf("related notes = %v", loop.RelatedNotes)
	}

	invalid := validBaseRequest(domain.ActionOpenLoop)
	invalid.ID = "open-loop-update-invalid"
	invalid.Payload = map[string]any{
		"id":    "loop-update-1",
		"state": string(domain.LoopOpen),
	}
	res, err = k.Process(ctx, invalid)
	if err != nil {
		t.Fatalf("invalid transition returned unexpected error: %v", err)
	}
	if res.Success {
		t.Fatalf("expected invalid transition to fail")
	}
	if res.DeterministicErrCode != domain.ErrInvalidStateTransition {
		t.Fatalf("deterministic error = %q, want %q", res.DeterministicErrCode, domain.ErrInvalidStateTransition)
	}
	loop, ok = store.FindLoop("loop-update-1")
	if !ok || loop.State != domain.LoopInProgress {
		t.Fatalf("invalid transition should not mutate loop, exists=%v loop=%+v", ok, loop)
	}
}

func TestProcessorInvalidAndUnauthorizedPaths(t *testing.T) {
	ctx := context.Background()
	k, store, _ := newTestKernel()

	invalidNote := validBaseRequest(domain.ActionCreateNote)
	invalidNote.ID = "invalid-note-1"
	invalidNote.Payload = map[string]any{
		"type":  string(domain.NoteFact),
		"title": "missing content",
	}
	res, err := k.Process(ctx, invalidNote)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Success {
		t.Fatalf("expected invalid create note to fail")
	}
	if res.DeterministicErrCode == "" {
		t.Fatalf("expected deterministic error code")
	}
	if res.AuditID == "" {
		t.Fatalf("expected audit id for rejected syscall")
	}

	unauthorized := validBaseRequest(domain.ActionUpdateState)
	unauthorized.ID = "unauthorized-1"
	unauthorized.Source = domain.SourceAdapter
	unauthorized.Actor.Kind = string(domain.SourceAdapter)
	unauthorized.Payload = map[string]any{
		"key":         "runtime.mode",
		"value":       map[string]any{"value": "mutate"},
		"derivedFrom": []string{"seed"},
	}
	res, err = k.Process(ctx, unauthorized)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Success {
		t.Fatalf("expected unauthorized action to fail")
	}
	if res.DeterministicErrCode != domain.ErrCapabilityDenied {
		t.Fatalf("expected capability denied, got %s", res.DeterministicErrCode)
	}
	if res.AuditID == "" {
		t.Fatalf("expected audit id for unauthorized syscall")
	}

	approvalReq := validBaseRequest(domain.ActionCreateNote)
	approvalReq.ID = "approval-required-1"
	approvalReq.Source = domain.SourceFutureIRIS
	approvalReq.Actor.Kind = string(domain.SourceFutureIRIS)
	approvalReq.Payload = map[string]any{
		"type":    string(domain.NoteFact),
		"title":   "iris",
		"content": "proposal",
	}
	res, err = k.Process(ctx, approvalReq)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Success {
		t.Fatalf("expected approval required path to fail commit")
	}
	if res.DeterministicErrCode != domain.ErrApprovalRequired {
		t.Fatalf("expected approval required code, got %s", res.DeterministicErrCode)
	}
	if _, ok := store.FindNote("approval-required-1:note"); ok {
		t.Fatalf("approval-required syscall must not commit state")
	}
	if res.AuditID == "" {
		t.Fatalf("expected audit id for approval-required syscall")
	}
}

func TestFutureIrisUsesSameKernelPath(t *testing.T) {
	store := NewInMemorySemanticStore()
	auditSink := NewInMemoryAuditSink()
	k := NewProcessor(ProcessorOptions{
		Registry:     NewStaticActionRegistry(),
		Validator:    NewDeterministicValidator(),
		Capabilities: NewStaticCapabilityService(),
		ApprovalGate: allowAllApprovalGate{},
		TxRunner:     NewInMemoryTransactionRunner(store),
		AuditSink:    auditSink,
		NowMillis:    func() int64 { return 1760000001000 },
	})
	req := validBaseRequest(domain.ActionCreateNote)
	req.ID = "iris-proposal-1"
	req.Source = domain.SourceFutureIRIS
	req.Actor.Kind = string(domain.SourceFutureIRIS)
	req.Payload = map[string]any{
		"id":      "iris-note-1",
		"type":    string(domain.NoteFact),
		"title":   "IRIS Proposal",
		"content": "Proposed note",
	}
	res, err := k.Process(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected future_iris proposal to commit via kernel path: %+v", res)
	}
	if res.AuditID == "" {
		t.Fatalf("expected audit id for committed syscall")
	}
	if _, ok := store.FindNote("iris-note-1"); !ok {
		t.Fatalf("expected note to exist only via kernel commit")
	}
}

func TestDryRunAndIdempotencyBehavior(t *testing.T) {
	ctx := context.Background()
	k, store, _ := newTestKernel()

	dryRun := validBaseRequest(domain.ActionCreateNote)
	dryRun.ID = "dry-run-1"
	dryRun.DryRun = true
	dryRun.Payload = map[string]any{
		"id":      "note-dry",
		"type":    string(domain.NoteFact),
		"title":   "Dry",
		"content": "Run",
	}
	res, err := k.Process(ctx, dryRun)
	if err != nil || !res.Success {
		t.Fatalf("dry-run should succeed: err=%v res=%+v", err, res)
	}
	if res.AuditID == "" {
		t.Fatalf("expected audit id for dry-run")
	}
	if len(res.CommittedObjectIDs) != 0 {
		t.Fatalf("dry-run should not commit ids")
	}
	if _, ok := store.FindNote("note-dry"); ok {
		t.Fatalf("dry-run must not write note")
	}

	idem := validBaseRequest(domain.ActionCreateNote)
	idem.ID = "idempotent-1"
	idem.IdempotencyKey = "idem-key-1"
	idem.Payload = map[string]any{
		"id":      "note-idem",
		"type":    string(domain.NoteFact),
		"title":   "Idem",
		"content": "Once",
	}
	res, err = k.Process(ctx, idem)
	if err != nil || !res.Success {
		t.Fatalf("first idempotent call failed: err=%v res=%+v", err, res)
	}
	replay, err := k.Process(ctx, idem)
	if err != nil || !replay.Success {
		t.Fatalf("idempotent replay failed: err=%v res=%+v", err, replay)
	}
	if replay.AuditID == "" {
		t.Fatalf("expected audit id for idempotent replay")
	}
	if len(replay.Warnings) == 0 {
		t.Fatalf("expected idempotent replay warning")
	}

	conflict := validBaseRequest(domain.ActionCompileContext)
	conflict.ID = "idempotent-2"
	conflict.IdempotencyKey = "idem-key-1"
	conflict.Payload = map[string]any{
		"query":  "test",
		"budget": map[string]any{"maxTokens": 10, "maxEvents": 10, "maxNotes": 10},
	}
	conflictRes, err := k.Process(ctx, conflict)
	if err != nil {
		t.Fatalf("unexpected error for conflicting idempotency key: %v", err)
	}
	if conflictRes.Success {
		t.Fatalf("expected conflicting idempotency key to fail")
	}
	if conflictRes.DeterministicErrCode != domain.ErrDuplicate {
		t.Fatalf("expected duplicate code, got %s", conflictRes.DeterministicErrCode)
	}
	if conflictRes.AuditID == "" {
		t.Fatalf("expected audit id for idempotency conflict")
	}
}

func TestAuditAndTransactionRollback(t *testing.T) {
	ctx := context.Background()
	k, store, auditSink := newTestKernel()
	mustCreateNote(ctx, k, "old-note", "old")
	mustCreateNote(ctx, k, "new-note", "new")

	// Force partial failure inside MARK_SUPERSEDED: link id already exists.
	preLinkReq := validBaseRequest(domain.ActionCreateLink)
	preLinkReq.ID = "tx-fail-1"
	preLinkReq.Payload = map[string]any{
		"id":       "tx-fail-1:supersedes_link",
		"type":     string(domain.LinkRelatesTo),
		"sourceId": "old-note",
		"targetId": "new-note",
	}
	if res, err := k.Process(ctx, preLinkReq); err != nil || !res.Success {
		t.Fatalf("failed to seed conflict link: err=%v res=%+v", err, res)
	}

	txFailReq := validBaseRequest(domain.ActionMarkSuperseded)
	txFailReq.ID = "tx-fail-1"
	txFailReq.Payload = map[string]any{
		"oldObjectId": "old-note",
		"newObjectId": "new-note",
		"reason":      "conflict test",
	}
	res, err := k.Process(ctx, txFailReq)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Success {
		t.Fatalf("expected supersession commit to fail")
	}
	if store.ExistsObject("tx-fail-1:supersession") {
		t.Fatalf("partial commit detected: supersession should have rolled back")
	}

	if len(auditSink.Records) == 0 {
		t.Fatalf("expected audit records to be emitted")
	}
}

func TestAuditRecordShapesAcrossOutcomes(t *testing.T) {
	ctx := context.Background()
	k, _, auditSink := newTestKernel()

	valid := validBaseRequest(domain.ActionCreateNote)
	valid.ID = "audit-valid-1"
	valid.Payload = map[string]any{
		"id":      "audit-note-1",
		"type":    string(domain.NoteFact),
		"title":   "a",
		"content": "b",
	}
	validRes, err := k.Process(ctx, valid)
	if err != nil || !validRes.Success {
		t.Fatalf("expected valid commit, err=%v res=%+v", err, validRes)
	}

	invalid := validBaseRequest(domain.ActionCreateNote)
	invalid.ID = "audit-invalid-1"
	invalid.Payload = map[string]any{"type": string(domain.NoteFact), "title": "missing content"}
	invalidRes, err := k.Process(ctx, invalid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if invalidRes.Success {
		t.Fatalf("expected invalid syscall to fail")
	}

	unauthorized := validBaseRequest(domain.ActionUpdateState)
	unauthorized.ID = "audit-unauthorized-1"
	unauthorized.Source = domain.SourceAdapter
	unauthorized.Actor.Kind = string(domain.SourceAdapter)
	unauthorized.Payload = map[string]any{
		"key":         "runtime.mode",
		"value":       map[string]any{"value": "x"},
		"derivedFrom": []string{"audit-note-1"},
	}
	unauthRes, err := k.Process(ctx, unauthorized)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if unauthRes.Success {
		t.Fatalf("expected unauthorized syscall to fail")
	}

	dryRun := validBaseRequest(domain.ActionCompileContext)
	dryRun.ID = "audit-dryrun-1"
	dryRun.DryRun = true
	dryRun.Payload = map[string]any{
		"query": "audit",
		"budget": map[string]any{
			"maxTokens": 10,
			"maxEvents": 5,
			"maxNotes":  5,
		},
	}
	dryRes, err := k.Process(ctx, dryRun)
	if err != nil || !dryRes.Success || !dryRes.DryRun {
		t.Fatalf("expected dry-run success, err=%v res=%+v", err, dryRes)
	}

	if len(auditSink.Records) < 4 {
		t.Fatalf("expected >=4 audit records, got %d", len(auditSink.Records))
	}
	committed := auditSink.Records[len(auditSink.Records)-4]
	rejected := auditSink.Records[len(auditSink.Records)-3]
	denied := auditSink.Records[len(auditSink.Records)-2]
	dry := auditSink.Records[len(auditSink.Records)-1]

	if !committed.Success || committed.DryRun {
		t.Fatalf("expected committed record success=true dryRun=false, got %+v", committed)
	}
	if rejected.Success || rejected.ErrorCode == "" {
		t.Fatalf("expected rejected record with error code, got %+v", rejected)
	}
	if denied.Success || denied.ErrorCode != domain.ErrCapabilityDenied {
		t.Fatalf("expected capability-denied record, got %+v", denied)
	}
	if !dry.Success || !dry.DryRun {
		t.Fatalf("expected dry-run record success=true dryRun=true, got %+v", dry)
	}
}

func TestCompileContextSnapshotPersistenceOptIn(t *testing.T) {
	ctx := context.Background()
	k, store, _ := newTestKernel()
	mustCreateNote(ctx, k, "note-ctx-a", "ctx a")
	mustCreateNote(ctx, k, "note-ctx-b", "ctx b")

	readOnly := validBaseRequest(domain.ActionCompileContext)
	readOnly.ID = "compile-context-optout-1"
	readOnly.Payload = map[string]any{
		"query":  "summarize blockers",
		"budget": map[string]any{"maxTokens": 50, "maxEvents": 10, "maxNotes": 10},
	}
	readOnlyRes, err := k.Process(ctx, readOnly)
	if err != nil || !readOnlyRes.Success {
		t.Fatalf("compile context opt-out failed: err=%v res=%+v", err, readOnlyRes)
	}
	if len(readOnlyRes.CommittedObjectIDs) != 0 {
		t.Fatalf("persistSnapshot=false should keep committed ids empty, got %v", readOnlyRes.CommittedObjectIDs)
	}
	if len(store.snapshot().contextSnapshots) != 0 || len(store.snapshot().artifacts) != 0 {
		t.Fatalf("persistSnapshot=false should not write snapshot evidence")
	}
	if len(store.snapshot().restoreOutcomes) != 0 {
		t.Fatalf("persistSnapshot=false should not write restore outcome evidence")
	}
	if draft, ok := readOnlyRes.StateSummary["restoreOutcome"].(map[string]any); !ok || draft["outcome"] != string(RestoreOutcomeUnknown) {
		t.Fatalf("expected non-persisted restore outcome draft, got %#v", readOnlyRes.StateSummary["restoreOutcome"])
	}
	if len(readOnlyRes.Warnings) != 1 || readOnlyRes.Warnings[0] != "compile_context is deterministic Phase 2 stub" {
		t.Fatalf("expected unchanged warning for persistSnapshot=false, got %v", readOnlyRes.Warnings)
	}

	persisted := validBaseRequest(domain.ActionCompileContext)
	persisted.ID = "compile-context-optin-1"
	persisted.Payload = map[string]any{
		"query":              "summarize blockers",
		"budget":             map[string]any{"maxTokens": 50, "maxEvents": 10, "maxNotes": 10},
		"persistSnapshot":    true,
		"renderSnapshotCard": true,
		"snapshotKind":       "restore",
	}
	persistedRes, err := k.Process(ctx, persisted)
	if err != nil || !persistedRes.Success {
		t.Fatalf("compile context opt-in failed: err=%v res=%+v", err, persistedRes)
	}
	state := store.snapshot()
	if len(state.contextSnapshots) != 1 {
		t.Fatalf("expected one persisted context snapshot, got %d", len(state.contextSnapshots))
	}
	if len(state.restoreOutcomes) != 1 {
		t.Fatalf("expected one persisted restore outcome event, got %d", len(state.restoreOutcomes))
	}
	if len(state.artifacts) != 1 {
		t.Fatalf("expected one persisted artifact ref, got %d", len(state.artifacts))
	}
	packetID := persistedRes.StateSummary["contextPacketId"]
	packet, ok := state.contextSnapshots[packetID.(string)]
	if !ok {
		t.Fatalf("missing persisted packet %v", packetID)
	}
	if packet.RestoreSnapshot == nil || packet.CompileOptions == nil {
		t.Fatalf("expected persisted restore snapshot metadata")
	}
	if !packet.CompileOptions.PersistSnapshot || !packet.CompileOptions.RenderSnapshotCard {
		t.Fatalf("expected persisted compile options, got %+v", packet.CompileOptions)
	}
	if got := readString(packet.RestoreSnapshot.Metadata, "rendered_card_artifact_id"); got == "" {
		t.Fatalf("expected snapshot evidence to link rendered card artifact")
	}
	if scores, ok := packet.RestoreSnapshot.Metadata["restore_scores_json"].(map[string]any); !ok || len(scores) == 0 {
		t.Fatalf("expected non-empty restore_scores_json metadata, got %#v", packet.RestoreSnapshot.Metadata["restore_scores_json"])
	}
	if _, ok := packet.RestoreSnapshot.Metadata["restore_trace_json"].(map[string]any); !ok {
		t.Fatalf("expected restore_trace_json metadata, got %#v", packet.RestoreSnapshot.Metadata["restore_trace_json"])
	}
	if hints, ok := packet.RestoreSnapshot.Metadata["resume_hints_json"].(map[string]any); !ok || len(hints) == 0 {
		t.Fatalf("expected non-empty resume_hints_json metadata, got %#v", packet.RestoreSnapshot.Metadata["resume_hints_json"])
	}
	reason, ok := packet.RestoreSnapshot.Metadata["restore_reason_json"].(map[string]any)
	if !ok {
		t.Fatalf("expected restore_reason_json metadata")
	}
	if v := strings.TrimSpace(readString(reason, "decision_reason")); v == "" {
		t.Fatalf("expected restore decision reason in restore reason json")
	}
	if decision := readString(persistedRes.StateSummary, "restoreDecision"); decision == "" {
		t.Fatalf("expected restoreDecision in state summary")
	}
	if persistedRes.StateSummary["snapshotFingerprint"] == "" {
		t.Fatalf("expected snapshot fingerprint in summary")
	}
	if decision := readString(persistedRes.StateSummary, "restoreDecision"); decision != "fresh_compile_no_candidates" {
		t.Fatalf("expected first persisted compile to report no candidates decision, got %q", decision)
	}
	outcomeSummary, ok := persistedRes.StateSummary["restoreOutcome"].(map[string]any)
	if !ok || outcomeSummary["outcome"] != string(RestoreOutcomeNoCandidate) {
		t.Fatalf("expected no-candidate restore outcome summary, got %#v", persistedRes.StateSummary["restoreOutcome"])
	}
	for _, outcome := range state.restoreOutcomes {
		if outcome.Outcome != RestoreOutcomeNoCandidate || !outcome.RequiresFreshCompile || outcome.ContextPacketID == "" {
			t.Fatalf("unexpected restore outcome event: %+v", outcome)
		}
	}
}

func TestCompileContextSnapshotResumeHintsCanForceFreshCompile(t *testing.T) {
	ctx := context.Background()
	k, store, _ := newTestKernel()

	mustCreateNote(ctx, k, "note-resume-a", "resume a")

	first := validBaseRequest(domain.ActionCompileContext)
	first.ID = "compile-context-resume-first"
	first.Payload = map[string]any{
		"query":           "summarize blockers",
		"budget":          map[string]any{"maxTokens": 50, "maxEvents": 10, "maxNotes": 10},
		"persistSnapshot": true,
		"snapshotKind":    "restore",
	}
	firstRes, err := k.Process(ctx, first)
	if err != nil || !firstRes.Success {
		t.Fatalf("first compile failed: err=%v res=%+v", err, firstRes)
	}
	if firstPacketID := readString(firstRes.StateSummary, "contextPacketId"); firstPacketID == "" {
		t.Fatalf("expected first contextPacketId")
	}

	second := validBaseRequest(domain.ActionCompileContext)
	second.ID = "compile-context-resume-second"
	second.RequestedAt = first.RequestedAt + 1000
	second.Payload = map[string]any{
		"query":           "summarize blockers",
		"budget":          map[string]any{"maxTokens": 50, "maxEvents": 10, "maxNotes": 10},
		"persistSnapshot": true,
		"snapshotKind":    "restore",
		"resumeHints": map[string]any{
			"freshCompileOnly": true,
		},
	}
	secondRes, err := k.Process(ctx, second)
	if err != nil || !secondRes.Success {
		t.Fatalf("second compile failed: err=%v res=%+v", err, secondRes)
	}
	if decision := readString(secondRes.StateSummary, "restoreDecision"); decision != "fresh_compile_forced" {
		t.Fatalf("expected forced fresh compile decision, got %q", decision)
	}
	if parent := readString(secondRes.StateSummary, "parentSnapshotId"); parent != "" {
		t.Fatalf("forced fresh compile should not set parentSnapshotId, got %q", parent)
	}
	state := store.snapshot()
	packetID := readString(secondRes.StateSummary, "contextPacketId")
	packet, ok := state.contextSnapshots[packetID]
	if !ok {
		t.Fatalf("missing second persisted packet %q", packetID)
	}
	if parent := readString(packet.RestoreSnapshot.Metadata, "parent_snapshot_id"); parent != "" {
		t.Fatalf("forced fresh compile should not persist parent snapshot id, got %q", parent)
	}
	if source := readString(packet.RestoreSnapshot.Metadata, "restore_source_snapshot_id"); source != "" {
		t.Fatalf("forced fresh compile should not persist restore source snapshot id, got %q", source)
	}
	if reason, ok := packet.RestoreSnapshot.Metadata["restore_reason_json"].(map[string]any); !ok || strings.TrimSpace(readString(reason, "decision_reason")) == "" {
		t.Fatalf("expected restore_reason_json with decision_reason, got %#v", packet.RestoreSnapshot.Metadata["restore_reason_json"])
	}
	if sourceSummary := readString(secondRes.StateSummary, "restoreSourceSnapshotId"); sourceSummary != "" {
		t.Fatalf("forced fresh compile should not set restoreSourceSnapshotId summary, got %q", sourceSummary)
	}
}

func TestCompileContextSnapshotBelowThresholdFallsBackToFreshCompile(t *testing.T) {
	ctx := context.Background()
	k, store, _ := newTestKernel()

	mustCreateNote(ctx, k, "note-threshold-a", "threshold a")

	first := validBaseRequest(domain.ActionCompileContext)
	first.ID = "compile-context-threshold-first"
	first.Payload = map[string]any{
		"query":           "summarize blockers",
		"budget":          map[string]any{"maxTokens": 50, "maxEvents": 10, "maxNotes": 10},
		"persistSnapshot": true,
		"snapshotKind":    "restore",
	}
	if res, err := k.Process(ctx, first); err != nil || !res.Success {
		t.Fatalf("first compile failed: err=%v res=%+v", err, res)
	}
	// Shift semantic input so restore scoring is no longer a perfect fingerprint match.
	mustCreateNote(ctx, k, "note-threshold-b", "threshold b")

	second := validBaseRequest(domain.ActionCompileContext)
	second.ID = "compile-context-threshold-second"
	second.RequestedAt = first.RequestedAt + (10 * 24 * 60 * 60 * 1000)
	second.Payload = map[string]any{
		"query":           "summarize blockers",
		"budget":          map[string]any{"maxTokens": 50, "maxEvents": 10, "maxNotes": 10},
		"persistSnapshot": true,
		"snapshotKind":    "restore",
		"resumeHints": map[string]any{
			"minimumScore": 0.99,
		},
	}
	secondRes, err := k.Process(ctx, second)
	if err != nil || !secondRes.Success {
		t.Fatalf("second compile failed: err=%v res=%+v", err, secondRes)
	}
	if decision := readString(secondRes.StateSummary, "restoreDecision"); decision != "fresh_compile_below_threshold" {
		t.Fatalf("expected below-threshold fallback decision, got %q", decision)
	}
	if parent := readString(secondRes.StateSummary, "parentSnapshotId"); parent != "" {
		t.Fatalf("below-threshold fallback should not set parentSnapshotId, got %q", parent)
	}
	state := store.snapshot()
	packetID := readString(secondRes.StateSummary, "contextPacketId")
	packet, ok := state.contextSnapshots[packetID]
	if !ok {
		t.Fatalf("missing second persisted packet %q", packetID)
	}
	if parent := readString(packet.RestoreSnapshot.Metadata, "parent_snapshot_id"); parent != "" {
		t.Fatalf("below-threshold fallback should not persist parent snapshot id, got %q", parent)
	}
	if reason, ok := packet.RestoreSnapshot.Metadata["restore_reason_json"].(map[string]any); !ok || reason["decision_reason"] != "top candidate score below threshold" {
		t.Fatalf("expected restore reason decision_reason, got %#v", packet.RestoreSnapshot.Metadata["restore_reason_json"])
	}
	foundFreshOutcome := false
	for _, outcome := range state.restoreOutcomes {
		if outcome.ContextPacketID == packetID {
			foundFreshOutcome = true
			if outcome.Outcome != RestoreOutcomeFreshCompileRequired || !outcome.RequiresFreshCompile {
				t.Fatalf("expected fresh-compile restore outcome for below-threshold packet, got %+v", outcome)
			}
		}
	}
	if !foundFreshOutcome {
		t.Fatalf("missing restore outcome for below-threshold packet %q", packetID)
	}
}

func TestCompileContextSnapshotDryRunDoesNotWrite(t *testing.T) {
	ctx := context.Background()
	k, store, _ := newTestKernel()

	req := validBaseRequest(domain.ActionCompileContext)
	req.ID = "compile-context-dryrun-snapshot-1"
	req.DryRun = true
	req.Payload = map[string]any{
		"query":              "summarize blockers",
		"budget":             map[string]any{"maxTokens": 50, "maxEvents": 10, "maxNotes": 10},
		"persistSnapshot":    true,
		"renderSnapshotCard": true,
		"snapshotKind":       "restore",
	}
	res, err := k.Process(ctx, req)
	if err != nil || !res.Success || !res.DryRun {
		t.Fatalf("compile context dry-run failed: err=%v res=%+v", err, res)
	}
	state := store.snapshot()
	if len(state.contextSnapshots) != 0 || len(state.artifacts) != 0 {
		t.Fatalf("dry-run must not write snapshot rows or artifact refs")
	}
	if len(state.restoreOutcomes) != 0 {
		t.Fatalf("dry-run must not write restore outcome rows")
	}
}

func TestInMemoryBuildContextScopeAndEvidenceFiltering(t *testing.T) {
	store := NewInMemorySemanticStore()
	scopeMain := domain.ForgeScope{WorkspaceID: "ws-main", LaneID: "control.semantic"}
	scopeOther := domain.ForgeScope{WorkspaceID: "ws-other", LaneID: "control.semantic"}

	if err := store.CreateNote(domain.MemoryNote{
		ID:        "note-main-active",
		Type:      domain.NoteFact,
		Title:     "main active",
		Content:   "main active content",
		Scope:     scopeMain,
		Status:    domain.NoteActive,
		CreatedAt: 100,
		UpdatedAt: 100,
	}); err != nil {
		t.Fatalf("seed active main note: %v", err)
	}
	if err := store.CreateNote(domain.MemoryNote{
		ID:        "note-main-archived",
		Type:      domain.NoteFact,
		Title:     "main archived",
		Content:   "archived content",
		Scope:     scopeMain,
		Status:    domain.NoteArchived,
		CreatedAt: 101,
		UpdatedAt: 101,
	}); err != nil {
		t.Fatalf("seed archived main note: %v", err)
	}
	if err := store.CreateNote(domain.MemoryNote{
		ID:        "note-other-active",
		Type:      domain.NoteFact,
		Title:     "other active",
		Content:   "other content",
		Scope:     scopeOther,
		Status:    domain.NoteActive,
		CreatedAt: 102,
		UpdatedAt: 102,
	}); err != nil {
		t.Fatalf("seed other workspace note: %v", err)
	}

	if err := store.CreateLoop(domain.OpenLoop{
		ID:        "loop-main-open",
		Title:     "main open",
		State:     domain.LoopOpen,
		Priority:  "medium",
		Scope:     scopeMain,
		CreatedAt: 103,
		UpdatedAt: 103,
	}); err != nil {
		t.Fatalf("seed open main loop: %v", err)
	}
	if err := store.CreateLoop(domain.OpenLoop{
		ID:        "loop-main-resolved",
		Title:     "main resolved",
		State:     domain.LoopResolved,
		Priority:  "medium",
		Scope:     scopeMain,
		CreatedAt: 104,
		UpdatedAt: 104,
	}); err != nil {
		t.Fatalf("seed resolved main loop: %v", err)
	}
	if err := store.CreateLoop(domain.OpenLoop{
		ID:        "loop-other-open",
		Title:     "other open",
		State:     domain.LoopOpen,
		Priority:  "medium",
		Scope:     scopeOther,
		CreatedAt: 105,
		UpdatedAt: 105,
	}); err != nil {
		t.Fatalf("seed open other loop: %v", err)
	}

	if err := store.CreateState(domain.StateItem{
		ID:        "state-main-active",
		Key:       "runtime.mode",
		Value:     map[string]any{"value": "deterministic"},
		Scope:     scopeMain,
		Status:    domain.StateActive,
		UpdatedAt: 109,
	}); err != nil {
		t.Fatalf("seed active main state: %v", err)
	}
	if err := store.CreateState(domain.StateItem{
		ID:        "state-main-archived",
		Key:       "runtime.mode.old",
		Value:     map[string]any{"value": "legacy"},
		Scope:     scopeMain,
		Status:    domain.StateArchived,
		UpdatedAt: 110,
	}); err != nil {
		t.Fatalf("seed archived main state: %v", err)
	}
	if err := store.CreateState(domain.StateItem{
		ID:        "state-other-active",
		Key:       "runtime.mode",
		Value:     map[string]any{"value": "other"},
		Scope:     scopeOther,
		Status:    domain.StateActive,
		UpdatedAt: 111,
	}); err != nil {
		t.Fatalf("seed active other state: %v", err)
	}

	if err := store.CreateArtifactRef(domain.ArtifactRef{
		ID:          "artifact-main-evidence",
		Type:        "document",
		URI:         "artifact://evidence/main.txt",
		Scope:       scopeMain,
		ContentHash: "sha1:main",
		CreatedAt:   106,
		Metadata:    map[string]any{"kind": "evidence"},
	}); err != nil {
		t.Fatalf("seed main evidence artifact: %v", err)
	}
	if err := store.CreateArtifactRef(domain.ArtifactRef{
		ID:          "artifact-main-card",
		Type:        "context_snapshot_card",
		URI:         "artifact://snapshot/main.svg",
		Scope:       scopeMain,
		ContentHash: "sha1:card",
		CreatedAt:   107,
		Metadata:    map[string]any{"kind": "context_snapshot_card"},
	}); err != nil {
		t.Fatalf("seed main snapshot card artifact: %v", err)
	}
	if err := store.CreateArtifactRef(domain.ArtifactRef{
		ID:          "artifact-other-evidence",
		Type:        "document",
		URI:         "artifact://evidence/other.txt",
		Scope:       scopeOther,
		ContentHash: "sha1:other",
		CreatedAt:   108,
		Metadata:    map[string]any{"kind": "evidence"},
	}); err != nil {
		t.Fatalf("seed other evidence artifact: %v", err)
	}

	packet := store.BuildContext("scope-check", scopeMain, domain.ContextBudget{
		MaxTokens: 100,
		MaxEvents: 10,
		MaxNotes:  10,
	}, 1760003000000)

	if len(packet.Notes) != 1 || packet.Notes[0].ID != "note-main-active" {
		t.Fatalf("expected only active scoped notes, got %+v", packet.Notes)
	}
	if len(packet.OpenLoops) != 1 || packet.OpenLoops[0].ID != "loop-main-open" {
		t.Fatalf("expected only active scoped loops, got %+v", packet.OpenLoops)
	}
	if len(packet.ActiveState) != 1 || packet.ActiveState[0].ID != "state-main-active" {
		t.Fatalf("expected only scoped non-archived state items, got %+v", packet.ActiveState)
	}
	if len(packet.Artifacts) != 1 || packet.Artifacts[0].ID != "artifact-main-evidence" {
		t.Fatalf("expected scoped non-snapshot artifacts only, got %+v", packet.Artifacts)
	}
}
