package controllane

import (
	"context"
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
	if persistedRes.StateSummary["snapshotFingerprint"] == "" {
		t.Fatalf("expected snapshot fingerprint in summary")
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
}
