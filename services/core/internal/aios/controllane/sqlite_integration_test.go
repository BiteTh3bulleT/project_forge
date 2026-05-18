package controllane

import (
	"context"
	"strings"
	"testing"
	"time"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/audit"
	"forge/projectforge/services/core/internal/store"
)

func newSQLiteKernel(t *testing.T, approval ApprovalGate) (*Processor, *SQLiteTransactionRunner, *store.Store) {
	t.Helper()
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	txRunner := NewSQLiteTransactionRunner(st.DB)
	if approval == nil {
		approval = allowAllApprovalGate{}
	}
	k := NewProcessor(ProcessorOptions{
		Registry:     NewStaticActionRegistry(),
		Validator:    NewDeterministicValidator(),
		Capabilities: NewStaticCapabilityService(),
		ApprovalGate: approval,
		TxRunner:     txRunner,
		AuditSink:    NewCoreAuditSink(audit.New(st.DB)),
		NowMillis:    func() int64 { return 1760001000000 + time.Now().UnixMilli()%1000000 },
	})
	return k, txRunner, st
}

func validSQLiteRequest(action domain.SemanticActionType, id, workspaceID string) domain.SyscallRequest {
	req := validBaseRequest(action)
	req.ID = id
	req.Scope = domain.ForgeScope{WorkspaceID: workspaceID, LaneID: "control.semantic"}
	req.CorrelationID = "corr-" + id
	req.TraceID = "trace-" + id
	req.Provenance.TraceID = req.TraceID
	return req
}

func TestSQLiteJournalImmutability(t *testing.T) {
	ctx := context.Background()
	_, txRunner, st := newSQLiteKernel(t, nil)
	read := txRunner.read
	evt := createTestJournalEvent("evt-immut", "ws-immut", "corr-immut", 1760001000000)
	if err := read.Append(ctx, evt); err != nil {
		t.Fatalf("append journal event: %v", err)
	}
	got, ok, err := read.GetByID(ctx, evt.ID)
	if err != nil || !ok {
		t.Fatalf("get journal event failed: err=%v ok=%v", err, ok)
	}
	if got.ID != evt.ID {
		t.Fatalf("unexpected event id: %s", got.ID)
	}
	if _, err := st.DB.ExecContext(ctx, `UPDATE journal_events SET type = 'mutated' WHERE id = ?`, evt.ID); err == nil {
		t.Fatalf("expected journal update to fail due append-only trigger")
	}
	if _, err := st.DB.ExecContext(ctx, `DELETE FROM journal_events WHERE id = ?`, evt.ID); err == nil {
		t.Fatalf("expected journal delete to fail due append-only trigger")
	}
}

func TestSQLiteSyscallPersistenceFlows(t *testing.T) {
	ctx := context.Background()
	k, txRunner, st := newSQLiteKernel(t, nil)
	read := txRunner.read
	scope := ScopeFilter{WorkspaceID: "ws-main", LaneID: "control.semantic"}

	t.Run("create note through syscall with audit linkage", func(t *testing.T) {
		req := validSQLiteRequest(domain.ActionCreateNote, "note-create-1", scope.WorkspaceID)
		req.Payload = map[string]any{
			"id":      "note-a",
			"type":    string(domain.NoteFact),
			"title":   "A",
			"content": "content-a",
		}
		res, err := k.Process(ctx, req)
		if err != nil || !res.Success {
			t.Fatalf("create note failed: err=%v res=%+v", err, res)
		}
		if res.AuditID == "" {
			t.Fatalf("expected audit id")
		}
		n, ok := read.FindNote("note-a")
		if !ok || n.ID != "note-a" {
			t.Fatalf("note not persisted")
		}
		var corr, auditID, proposedBy, committedBy string
		if err := st.DB.QueryRowContext(ctx, `SELECT correlation_id, audit_id, proposed_by, committed_by FROM memory_notes WHERE id = ?`, "note-a").Scan(&corr, &auditID, &proposedBy, &committedBy); err != nil {
			t.Fatalf("query persisted note metadata: %v", err)
		}
		if corr != req.CorrelationID || auditID == "" {
			t.Fatalf("expected correlation/audit linkage, got corr=%q audit=%q", corr, auditID)
		}
		if proposedBy == "" || committedBy != "forge_kernel" {
			t.Fatalf("expected proposed/committed metadata, got proposedBy=%q committedBy=%q", proposedBy, committedBy)
		}
		var provProposedBy, provCommittedBy, provCorr, provAuditID string
		if err := st.DB.QueryRowContext(ctx, `
SELECT p.proposed_by, p.committed_by, p.correlation_id, p.audit_id
FROM memory_notes n
JOIN provenance_records p ON p.id = n.provenance_id
WHERE n.id = ?`, "note-a").Scan(&provProposedBy, &provCommittedBy, &provCorr, &provAuditID); err != nil {
			t.Fatalf("query provenance linkage: %v", err)
		}
		if provProposedBy == "" || provCommittedBy != "forge_kernel" || provCorr != req.CorrelationID || provAuditID == "" {
			t.Fatalf("unexpected provenance trace fields: proposed=%q committed=%q corr=%q audit=%q", provProposedBy, provCommittedBy, provCorr, provAuditID)
		}
	})

	t.Run("create link and query source target neighborhood", func(t *testing.T) {
		mustCreateNote(ctx, k, "note-b", "B")
		mustCreateNote(ctx, k, "note-c", "C")
		req := validSQLiteRequest(domain.ActionCreateLink, "link-create-1", scope.WorkspaceID)
		req.Payload = map[string]any{
			"id":       "link-a-b",
			"type":     string(domain.LinkSupports),
			"sourceId": "note-a",
			"targetId": "note-b",
		}
		res, err := k.Process(ctx, req)
		if err != nil || !res.Success {
			t.Fatalf("create link failed: err=%v res=%+v", err, res)
		}
		bySource, err := read.ListBySource(ctx, "note-a", scope, 20)
		if err != nil || len(bySource) == 0 {
			t.Fatalf("list by source failed: err=%v count=%d", err, len(bySource))
		}
		byTarget, err := read.ListByTarget(ctx, "note-b", scope, 20)
		if err != nil || len(byTarget) == 0 {
			t.Fatalf("list by target failed: err=%v count=%d", err, len(byTarget))
		}
		nh, err := read.ListNeighborhood(ctx, "note-a", scope, 1, 50)
		if err != nil || len(nh) == 0 {
			t.Fatalf("list neighborhood failed: err=%v count=%d", err, len(nh))
		}
	})

	t.Run("update state preserves timeline", func(t *testing.T) {
		req1 := validSQLiteRequest(domain.ActionUpdateState, "state-upsert-1", scope.WorkspaceID)
		req1.Payload = map[string]any{
			"key":         "runtime.mode",
			"value":       map[string]any{"value": "alpha"},
			"derivedFrom": []string{"note-a"},
		}
		res1, err := k.Process(ctx, req1)
		if err != nil || !res1.Success {
			t.Fatalf("state update 1 failed: err=%v res=%+v", err, res1)
		}

		req2 := validSQLiteRequest(domain.ActionUpdateState, "state-upsert-2", scope.WorkspaceID)
		req2.Payload = map[string]any{
			"key":         "runtime.mode",
			"value":       map[string]any{"value": "beta"},
			"derivedFrom": []string{"note-b"},
		}
		res2, err := k.Process(ctx, req2)
		if err != nil || !res2.Success {
			t.Fatalf("state update 2 failed: err=%v res=%+v", err, res2)
		}
		cur, ok, err := read.GetCurrent(ctx, "runtime.mode", scope)
		if err != nil || !ok {
			t.Fatalf("get current state failed: err=%v ok=%v", err, ok)
		}
		if got := cur.Value["value"]; got != "beta" {
			t.Fatalf("expected beta state value, got %v", got)
		}
		timeline, err := read.GetTimeline(ctx, "runtime.mode", scope, 10)
		if err != nil {
			t.Fatalf("get timeline failed: %v", err)
		}
		if len(timeline) < 2 {
			t.Fatalf("expected >=2 timeline entries, got %d", len(timeline))
		}
		if timeline[0].ProposedBy == "" || timeline[0].CommittedBy != "forge_kernel" {
			t.Fatalf("expected state timeline proposer/committer metadata, got proposed=%q committed=%q", timeline[0].ProposedBy, timeline[0].CommittedBy)
		}
		if timeline[0].CorrelationID == "" || timeline[0].AuditID == "" {
			t.Fatalf("expected state timeline correlation/audit linkage, got corr=%q audit=%q", timeline[0].CorrelationID, timeline[0].AuditID)
		}
	})

	t.Run("open and close loop with transition enforcement", func(t *testing.T) {
		openReq := validSQLiteRequest(domain.ActionOpenLoop, "loop-open-1", scope.WorkspaceID)
		openReq.Payload = map[string]any{
			"id":       "loop-a",
			"title":    "loop a",
			"state":    string(domain.LoopOpen),
			"priority": "high",
		}
		openRes, err := k.Process(ctx, openReq)
		if err != nil || !openRes.Success {
			t.Fatalf("open loop failed: err=%v res=%+v", err, openRes)
		}

		closeReq := validSQLiteRequest(domain.ActionCloseLoop, "loop-close-1", scope.WorkspaceID)
		closeReq.Payload = map[string]any{"loopId": "loop-a", "reason": "done"}
		closeRes, err := k.Process(ctx, closeReq)
		if err != nil || !closeRes.Success {
			t.Fatalf("close loop failed: err=%v res=%+v", err, closeRes)
		}

		badClose := validSQLiteRequest(domain.ActionCloseLoop, "loop-close-2", scope.WorkspaceID)
		badClose.Payload = map[string]any{"loopId": "loop-a", "reason": "done-again"}
		badRes, err := k.Process(ctx, badClose)
		if err != nil {
			t.Fatalf("unexpected close loop error: %v", err)
		}
		if badRes.Success || badRes.DeterministicErrCode != domain.ErrInvalidStateTransition {
			t.Fatalf("expected invalid transition rejection, got %+v", badRes)
		}
		loop, ok := read.FindLoop("loop-a")
		if !ok || loop.State != domain.LoopResolved {
			t.Fatalf("expected resolved loop to remain queryable")
		}
	})

	t.Run("supersession preserves old and new", func(t *testing.T) {
		req := validSQLiteRequest(domain.ActionMarkSuperseded, "supersede-1", scope.WorkspaceID)
		req.Payload = map[string]any{
			"oldObjectId":   "note-a",
			"oldObjectKind": "memory_note",
			"newObjectId":   "note-b",
			"newObjectKind": "memory_note",
			"reason":        "new evidence",
		}
		res, err := k.Process(ctx, req)
		if err != nil || !res.Success {
			t.Fatalf("mark superseded failed: err=%v res=%+v", err, res)
		}
		succ, ok, err := read.GetCurrentSuccessor(ctx, "note-a", scope)
		if err != nil || !ok {
			t.Fatalf("get successor failed: err=%v ok=%v", err, ok)
		}
		if succ.NewID != "note-b" {
			t.Fatalf("unexpected successor: %+v", succ)
		}
		oldNote, oldOk := read.FindNote("note-a")
		newNote, newOk := read.FindNote("note-b")
		if !oldOk || !newOk {
			t.Fatalf("both notes must remain after supersession")
		}
		if oldNote.Status != domain.NoteSuperseded || newNote.Status == domain.NoteArchived {
			t.Fatalf("unexpected note statuses after supersession old=%s new=%s", oldNote.Status, newNote.Status)
		}
	})

	t.Run("contradiction preserves both sides", func(t *testing.T) {
		req := validSQLiteRequest(domain.ActionRegisterContradict, "contradict-1", scope.WorkspaceID)
		req.Payload = map[string]any{
			"leftObjectId":    "note-a",
			"leftObjectKind":  "memory_note",
			"rightObjectId":   "note-c",
			"rightObjectKind": "memory_note",
			"reason":          "contradiction",
			"severity":        "high",
			"confidence":      0.82,
		}
		res, err := k.Process(ctx, req)
		if err != nil || !res.Success {
			t.Fatalf("register contradiction failed: err=%v res=%+v", err, res)
		}
		list, err := read.ListByObject(ctx, "note-a", scope, 20)
		if err != nil || len(list) == 0 {
			t.Fatalf("expected contradiction by object, err=%v count=%d", err, len(list))
		}
		if _, ok := read.FindNote("note-a"); !ok {
			t.Fatalf("left note removed unexpectedly")
		}
		if _, ok := read.FindNote("note-c"); !ok {
			t.Fatalf("right note removed unexpectedly")
		}
	})

	t.Run("derive model keeps evidence and stays provisional", func(t *testing.T) {
		req := validSQLiteRequest(domain.ActionDeriveModel, "derive-1", scope.WorkspaceID)
		req.Payload = map[string]any{
			"id":           "model-a",
			"type":         "routing",
			"expression":   map[string]any{"formula": "score=evidence*confidence"},
			"derivedFrom":  []string{"note-a", "note-b"},
			"supportCount": 2,
			"confidence":   0.6,
		}
		res, err := k.Process(ctx, req)
		if err != nil || !res.Success {
			t.Fatalf("derive model failed: err=%v res=%+v", err, res)
		}
		model, ok := read.FindModel("model-a")
		if !ok {
			t.Fatalf("model missing")
		}
		if model.Status != domain.ModelProvisional {
			t.Fatalf("expected provisional model, got %s", model.Status)
		}
		if len(model.DerivedFrom) != 2 {
			t.Fatalf("expected derivedFrom evidence")
		}
		if _, ok := read.FindNote("note-a"); !ok {
			t.Fatalf("evidence note should remain")
		}
	})

	t.Run("archive note keeps record and removes from active list", func(t *testing.T) {
		req := validSQLiteRequest(domain.ActionArchiveNote, "archive-1", scope.WorkspaceID)
		req.Payload = map[string]any{
			"noteId": "note-c",
			"reason": "cleanup",
		}
		res, err := k.Process(ctx, req)
		if err != nil || !res.Success {
			t.Fatalf("archive note failed: err=%v res=%+v", err, res)
		}
		note, ok := read.FindNote("note-c")
		if !ok || note.Status != domain.NoteArchived {
			t.Fatalf("note should remain archived")
		}
		active, err := read.ListActive(ctx, scope)
		if err != nil {
			t.Fatalf("list active notes: %v", err)
		}
		for _, n := range active {
			if n.ID == "note-c" {
				t.Fatalf("archived note should not appear in active list")
			}
		}
	})

	t.Run("workspace isolation and linked neighborhood", func(t *testing.T) {
		ws2 := "ws-other"
		reqWs2 := validSQLiteRequest(domain.ActionCreateNote, "ws2-note-1", ws2)
		reqWs2.Payload = map[string]any{
			"id":      "note-ws2",
			"type":    string(domain.NoteFact),
			"title":   "ws2",
			"content": "isolated",
		}
		if res, err := k.Process(ctx, reqWs2); err != nil || !res.Success {
			t.Fatalf("create ws2 note failed: err=%v res=%+v", err, res)
		}

		ws1Notes, err := read.ListActive(ctx, scope)
		if err != nil {
			t.Fatalf("list ws1 notes: %v", err)
		}
		for _, n := range ws1Notes {
			if n.Scope.WorkspaceID != scope.WorkspaceID {
				t.Fatalf("workspace leak into ws1 query: note=%+v", n)
			}
		}

		ws2Notes, err := read.ListActive(ctx, ScopeFilter{WorkspaceID: ws2, LaneID: "control.semantic"})
		if err != nil || len(ws2Notes) == 0 {
			t.Fatalf("list ws2 notes failed: err=%v count=%d", err, len(ws2Notes))
		}
		for _, n := range ws2Notes {
			if n.Scope.WorkspaceID != ws2 {
				t.Fatalf("workspace leak into ws2 query: note=%+v", n)
			}
		}

		// two-hop neighborhood
		link2 := validSQLiteRequest(domain.ActionCreateLink, "link-b-c", scope.WorkspaceID)
		link2.Payload = map[string]any{
			"id":       "link-b-c",
			"type":     string(domain.LinkSupports),
			"sourceId": "note-b",
			"targetId": "note-c",
		}
		if res, err := k.Process(ctx, link2); err != nil || !res.Success {
			t.Fatalf("create link b-c failed: err=%v res=%+v", err, res)
		}
		nh, err := read.ListNeighborhood(ctx, "note-a", scope, 2, 50)
		if err != nil {
			t.Fatalf("list two-hop neighborhood: %v", err)
		}
		found := false
		for _, l := range nh {
			if l.ID == "link-b-c" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected two-hop neighborhood to include link-b-c")
		}
	})

	t.Run("dry-run does not persist", func(t *testing.T) {
		req := validSQLiteRequest(domain.ActionCreateNote, "dryrun-note-1", scope.WorkspaceID)
		req.DryRun = true
		req.Payload = map[string]any{
			"id":      "note-dryrun",
			"type":    string(domain.NoteFact),
			"title":   "dryrun",
			"content": "no write",
		}
		res, err := k.Process(ctx, req)
		if err != nil || !res.Success || !res.DryRun {
			t.Fatalf("dry-run failed: err=%v res=%+v", err, res)
		}
		if _, ok := read.FindNote("note-dryrun"); ok {
			t.Fatalf("dry-run wrote note unexpectedly")
		}
	})

	t.Run("transaction rollback on partial failure", func(t *testing.T) {
		// Force link conflict in MARK_SUPERSEDED handler.
		preLink := validSQLiteRequest(domain.ActionCreateLink, "txrb-1", scope.WorkspaceID)
		preLink.Payload = map[string]any{
			"id":       "txrb-1:supersedes_link",
			"type":     string(domain.LinkRelatesTo),
			"sourceId": "note-a",
			"targetId": "note-b",
		}
		if res, err := k.Process(ctx, preLink); err != nil || !res.Success {
			t.Fatalf("seed conflicting link failed: err=%v res=%+v", err, res)
		}
		req := validSQLiteRequest(domain.ActionMarkSuperseded, "txrb-1", scope.WorkspaceID)
		req.Payload = map[string]any{
			"oldObjectId": "note-a",
			"newObjectId": "note-b",
			"reason":      "rollback case",
		}
		res, err := k.Process(ctx, req)
		if err != nil {
			t.Fatalf("unexpected rollback error: %v", err)
		}
		if res.Success {
			t.Fatalf("expected rollback failure")
		}
		if _, ok, err := read.GetByIDSupersession(ctx, "txrb-1:supersession"); err != nil || ok {
			t.Fatalf("expected no supersession persisted after rollback, err=%v ok=%v", err, ok)
		}
	})

	t.Run("context packet snapshot repository", func(t *testing.T) {
		pkt := createTestContextPacketSnapshot("ctx-snap-1", scope.WorkspaceID, 1760001009999)
		if err := read.CreateSnapshot(ctx, pkt, "ctx-syscall-1", "corr-ctx-1", "trace-ctx-1", map[string]any{"source": "test"}); err != nil {
			t.Fatalf("create snapshot failed: %v", err)
		}
		got, ok, err := read.GetSnapshotByID(ctx, pkt.ID)
		if err != nil || !ok {
			t.Fatalf("get snapshot failed: err=%v ok=%v", err, ok)
		}
		if got.ID != pkt.ID || got.Query == "" {
			t.Fatalf("unexpected snapshot payload: %+v", got)
		}
		byScope, err := read.ListSnapshotsByScope(ctx, ScopeFilter{WorkspaceID: scope.WorkspaceID}, 10)
		if err != nil || len(byScope) == 0 {
			t.Fatalf("list snapshots by scope failed: err=%v count=%d", err, len(byScope))
		}
		byCorr, err := read.ListSnapshotsByCorrelation(ctx, "corr-ctx-1", 10)
		if err != nil || len(byCorr) == 0 {
			t.Fatalf("list snapshots by correlation failed: err=%v count=%d", err, len(byCorr))
		}
	})
}

func TestSQLiteFutureIrisCannotBypassKernel(t *testing.T) {
	ctx := context.Background()
	k, txRunner, _ := newSQLiteKernel(t, NewStaticApprovalGate())
	read := txRunner.read
	req := validSQLiteRequest(domain.ActionCreateNote, "iris-note-denied", "ws-main")
	req.Source = domain.SourceFutureIRIS
	req.Actor.Kind = string(domain.SourceFutureIRIS)
	req.Payload = map[string]any{
		"id":      "note-iris-denied",
		"type":    string(domain.NoteFact),
		"title":   "iris proposal",
		"content": "candidate",
	}
	res, err := k.Process(ctx, req)
	if err != nil {
		t.Fatalf("unexpected future_iris process error: %v", err)
	}
	if res.Success || res.DeterministicErrCode != domain.ErrApprovalRequired {
		t.Fatalf("expected approval-required rejection for future_iris mutating action, got %+v", res)
	}
	if _, ok := read.FindNote("note-iris-denied"); ok {
		t.Fatalf("future_iris should not bypass validation/approval and commit directly")
	}
}

func TestSQLiteCreateNoteIsLowRiskKernelStyleCommit(t *testing.T) {
	ctx := context.Background()
	k, txRunner, st := newSQLiteKernel(t, nil)
	req := validSQLiteRequest(domain.ActionCreateNote, "phase11-note-commit", "ws-phase11")
	req.Payload = map[string]any{
		"id":      "phase11-note",
		"type":    string(domain.NoteFact),
		"title":   "Phase 11 note",
		"content": "low-risk object committed through Control Lane",
	}

	res, err := k.Process(ctx, req)
	if err != nil || !res.Success {
		t.Fatalf("create note failed: err=%v res=%+v", err, res)
	}
	if !stringInSlice(res.CommittedObjectIDs, "phase11-note") || !stringInSlice(res.CommittedObjectIDs, "phase11-note-commit:journal_event") {
		t.Fatalf("expected note and journal event committed ids, got %v", res.CommittedObjectIDs)
	}
	note, ok := txRunner.read.FindNote("phase11-note")
	if !ok {
		t.Fatalf("expected note to be queryable through semantic read store")
	}
	if note.Scope.WorkspaceID != req.Scope.WorkspaceID || note.Provenance.TraceID != req.TraceID {
		t.Fatalf("unexpected note scope/provenance: %#v", note)
	}

	var eventType, eventSource, eventCorr string
	if err := st.DB.QueryRowContext(ctx, `SELECT type, source, correlation_id FROM journal_events WHERE id = ?`, "phase11-note-commit:journal_event").Scan(&eventType, &eventSource, &eventCorr); err != nil {
		t.Fatalf("query journal event: %v", err)
	}
	if eventType != "semantic_syscall.create_note" || eventSource != "forge_kernel" || eventCorr != req.CorrelationID {
		t.Fatalf("unexpected journal event type/source/corr: %q %q %q", eventType, eventSource, eventCorr)
	}

	var committedBy, syscallID, correlationID, auditID string
	if err := st.DB.QueryRowContext(ctx, `SELECT committed_by, syscall_id, correlation_id, audit_id FROM memory_notes WHERE id = ?`, "phase11-note").Scan(&committedBy, &syscallID, &correlationID, &auditID); err != nil {
		t.Fatalf("query note lineage: %v", err)
	}
	if committedBy != "forge_kernel" || syscallID != req.ID || correlationID != req.CorrelationID || auditID == "" {
		t.Fatalf("unexpected note lineage committed=%q syscall=%q corr=%q audit=%q", committedBy, syscallID, correlationID, auditID)
	}
}

func TestSQLiteContextCompileDryRunAndReadOnlyPath(t *testing.T) {
	ctx := context.Background()
	k, txRunner, _ := newSQLiteKernel(t, nil)
	read := txRunner.read
	req := validSQLiteRequest(domain.ActionCompileContext, "ctx-compile-1", "ws-main")
	req.DryRun = true
	req.Payload = map[string]any{
		"query":  "summarize",
		"budget": map[string]any{"maxTokens": 20, "maxEvents": 10, "maxNotes": 10},
	}
	res, err := k.Process(ctx, req)
	if err != nil || !res.Success || !res.DryRun {
		t.Fatalf("compile context dry-run failed: err=%v res=%+v", err, res)
	}
	rows, err := txRunner.db.QueryContext(ctx, `SELECT id FROM context_packet_snapshots`)
	if err != nil {
		t.Fatalf("query context snapshots: %v", err)
	}
	defer rows.Close()
	if rows.Next() {
		t.Fatalf("dry-run compile should not persist context snapshots by default")
	}

	// read-store BuildContext still deterministic and queryable.
	pkt := read.BuildContext("summarize", domain.ForgeScope{WorkspaceID: "ws-main", LaneID: "control.semantic"}, domain.ContextBudget{
		MaxTokens: 20,
		MaxEvents: 10,
		MaxNotes:  10,
	}, 1760001001234)
	if pkt.ID == "" || !strings.Contains(pkt.ID, "ctx-") {
		t.Fatalf("unexpected context packet id: %q", pkt.ID)
	}
}

func stringInSlice(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func TestSQLiteContextCompilePersistsSnapshotEvidence(t *testing.T) {
	ctx := context.Background()
	k, txRunner, st := newSQLiteKernel(t, nil)
	read := txRunner.read
	scope := ScopeFilter{WorkspaceID: "ws-main", LaneID: "control.semantic"}

	mustCreateNote(ctx, k, "ctx-note-a", "ctx a")
	mustCreateNote(ctx, k, "ctx-note-b", "ctx b")
	loopReq := validSQLiteRequest(domain.ActionOpenLoop, "ctx-loop-1", scope.WorkspaceID)
	loopReq.Payload = map[string]any{
		"id":       "ctx-loop-a",
		"title":    "ctx loop",
		"state":    string(domain.LoopBlocked),
		"priority": "high",
		"blocker":  "awaiting evidence",
	}
	if res, err := k.Process(ctx, loopReq); err != nil || !res.Success {
		t.Fatalf("seed loop failed: err=%v res=%+v", err, res)
	}

	req := validSQLiteRequest(domain.ActionCompileContext, "ctx-compile-persist-1", scope.WorkspaceID)
	req.Payload = map[string]any{
		"query":              "summarize blockers",
		"budget":             map[string]any{"maxTokens": 50, "maxEvents": 20, "maxNotes": 20},
		"persistSnapshot":    true,
		"renderSnapshotCard": true,
		"snapshotKind":       "restore",
	}
	res, err := k.Process(ctx, req)
	if err != nil || !res.Success {
		t.Fatalf("persisting compile context failed: err=%v res=%+v", err, res)
	}

	packet, ok := read.FindLatestContextSnapshot(req.Scope, "summarize blockers", "restore")
	if !ok {
		t.Fatalf("expected persisted context snapshot")
	}
	if packet.RestoreSnapshot == nil || packet.CompileOptions == nil {
		t.Fatalf("expected restore snapshot metadata on packet")
	}
	if _, ok := packet.RestoreSnapshot.Metadata["restore_trace_json"].(map[string]any); !ok {
		t.Fatalf("expected restore_trace_json in restore snapshot metadata")
	}
	if got := readString(packet.RestoreSnapshot.Metadata, "rendered_card_artifact_id"); got == "" {
		t.Fatalf("expected rendered card artifact link")
	}
	if packet.RestoreSnapshot.SnapshotID != packet.ID {
		t.Fatalf("expected restore snapshot id to align with packet id, got packet=%q restore=%q", packet.ID, packet.RestoreSnapshot.SnapshotID)
	}

	var corr, traceID, syscallID, auditID, proposedBy, committedBy, metadataRaw string
	var snapshotKind, snapshotFingerprint, parentSnapshotID, headerJSON, graphJSON, deltaJSON, restoreScoresJSON, renderArtifactRefID, resumeHintsJSON string
	if err := st.DB.QueryRowContext(ctx, `
SELECT correlation_id, trace_id, syscall_id, audit_id, proposed_by, committed_by, metadata_json,
       snapshot_kind, snapshot_fingerprint, parent_snapshot_id, header_json, graph_json, delta_json,
       restore_scores_json, render_artifact_ref_id, resume_hints_json
FROM context_packet_snapshots
WHERE id = ?`, packet.ID).Scan(
		&corr,
		&traceID,
		&syscallID,
		&auditID,
		&proposedBy,
		&committedBy,
		&metadataRaw,
		&snapshotKind,
		&snapshotFingerprint,
		&parentSnapshotID,
		&headerJSON,
		&graphJSON,
		&deltaJSON,
		&restoreScoresJSON,
		&renderArtifactRefID,
		&resumeHintsJSON,
	); err != nil {
		t.Fatalf("query persisted snapshot metadata: %v", err)
	}
	if corr != req.CorrelationID || traceID != req.TraceID || syscallID != req.ID || auditID == "" {
		t.Fatalf("unexpected snapshot lineage corr=%q trace=%q syscall=%q audit=%q resultAudit=%q warnings=%v", corr, traceID, syscallID, auditID, res.AuditID, res.Warnings)
	}
	if proposedBy == "" || committedBy != "forge_kernel" {
		t.Fatalf("unexpected proposed/committed metadata proposed=%q committed=%q", proposedBy, committedBy)
	}
	if !strings.Contains(metadataRaw, `"snapshot_kind":"restore"`) {
		t.Fatalf("expected snapshot metadata to carry snapshot_kind, got %s", metadataRaw)
	}
	if snapshotKind != "restore" {
		t.Fatalf("expected snapshot_kind column to persist restore, got %q", snapshotKind)
	}
	if snapshotFingerprint == "" {
		t.Fatalf("expected snapshot_fingerprint column to be populated")
	}
	if parentSnapshotID != "" {
		t.Fatalf("first persisted snapshot should not set parent snapshot id, got %q", parentSnapshotID)
	}
	if headerJSON == "{}" || graphJSON == "{}" || deltaJSON == "{}" {
		t.Fatalf("expected header/graph/delta columns to persist non-empty snapshot evidence")
	}
	if restoreScoresJSON == "{}" {
		t.Fatalf("expected restore_scores_json to persist scored selection metadata, got %s", restoreScoresJSON)
	}
	if !strings.Contains(metadataRaw, `"restore_trace_json"`) {
		t.Fatalf("expected metadata_json to persist restore_trace_json, got %s", metadataRaw)
	}
	if renderArtifactRefID == "" {
		t.Fatalf("expected render_artifact_ref_id column to persist rendered card artifact")
	}
	if resumeHintsJSON == "{}" {
		t.Fatalf("expected resume_hints_json to persist resume contract metadata, got %s", resumeHintsJSON)
	}

	renderedArtifactID := readString(packet.RestoreSnapshot.Metadata, "rendered_card_artifact_id")
	var artifactCorr, artifactTrace, artifactSyscall, artifactAudit, artifactProposed, artifactCommitted string
	if err := st.DB.QueryRowContext(ctx, `
SELECT correlation_id, trace_id, syscall_id, audit_id, proposed_by, committed_by
FROM artifact_refs
WHERE id = ?`, renderedArtifactID).Scan(&artifactCorr, &artifactTrace, &artifactSyscall, &artifactAudit, &artifactProposed, &artifactCommitted); err != nil {
		t.Fatalf("query rendered card artifact: %v", err)
	}
	if artifactCorr != req.CorrelationID || artifactTrace != req.TraceID || artifactSyscall != req.ID || artifactAudit == "" {
		t.Fatalf("unexpected artifact lineage corr=%q trace=%q syscall=%q audit=%q", artifactCorr, artifactTrace, artifactSyscall, artifactAudit)
	}
	if artifactProposed == "" || artifactCommitted != "forge_kernel" {
		t.Fatalf("unexpected artifact proposed/committed metadata proposed=%q committed=%q", artifactProposed, artifactCommitted)
	}
	if res.StateSummary["snapshotFingerprint"] == "" {
		t.Fatalf("expected snapshot fingerprint in syscall summary")
	}
	if readString(res.StateSummary, "restoreDecision") == "" {
		t.Fatalf("expected restoreDecision summary field")
	}
}

func TestSQLiteContextCompileRepeatedFingerprintLinksParent(t *testing.T) {
	ctx := context.Background()
	k, txRunner, _ := newSQLiteKernel(t, nil)

	mustCreateNote(ctx, k, "ctx-repeat-a", "repeat a")
	mustCreateNote(ctx, k, "ctx-repeat-b", "repeat b")

	first := validSQLiteRequest(domain.ActionCompileContext, "ctx-repeat-compile-1", "ws-main")
	first.Payload = map[string]any{
		"query":           "summarize blockers",
		"budget":          map[string]any{"maxTokens": 50, "maxEvents": 20, "maxNotes": 20},
		"persistSnapshot": true,
		"snapshotKind":    "restore",
	}
	firstRes, err := k.Process(ctx, first)
	if err != nil || !firstRes.Success {
		t.Fatalf("first persisted compile failed: err=%v res=%+v", err, firstRes)
	}

	second := validSQLiteRequest(domain.ActionCompileContext, "ctx-repeat-compile-2", "ws-main")
	second.RequestedAt = first.RequestedAt + 1000
	second.Payload = first.Payload
	secondRes, err := k.Process(ctx, second)
	if err != nil || !secondRes.Success {
		t.Fatalf("second persisted compile failed: err=%v res=%+v", err, secondRes)
	}

	packet, ok := txRunner.read.FindLatestContextSnapshot(second.Scope, "summarize blockers", "restore")
	if !ok || packet.RestoreSnapshot == nil {
		t.Fatalf("expected latest repeated snapshot")
	}
	if parent := readString(packet.RestoreSnapshot.Metadata, "parent_snapshot_id"); parent == "" {
		t.Fatalf("expected parent snapshot linkage on repeated compile")
	}
	var parentSnapshotID string
	if err := txRunner.db.QueryRowContext(ctx, `
SELECT parent_snapshot_id
FROM context_packet_snapshots
WHERE id = ?`, packet.ID).Scan(&parentSnapshotID); err != nil {
		t.Fatalf("query parent_snapshot_id: %v", err)
	}
	if strings.TrimSpace(parentSnapshotID) == "" {
		t.Fatalf("expected parent_snapshot_id column to persist repeated lineage")
	}
	if reason, ok := packet.RestoreSnapshot.Metadata["restore_reason_json"].(map[string]any); !ok || reason["fingerprint_matched"] != true {
		t.Fatalf("expected fingerprint matched restore reason, got %#v", packet.RestoreSnapshot.Metadata["restore_reason_json"])
	}
	if trace, ok := packet.RestoreSnapshot.Metadata["restore_trace_json"].(map[string]any); !ok {
		t.Fatalf("expected restore_trace_json on repeated compile snapshot, got %#v", packet.RestoreSnapshot.Metadata["restore_trace_json"])
	} else if _, ok := trace["winner"]; !ok {
		t.Fatalf("expected winner section in restore trace")
	}
	scores, ok := packet.RestoreSnapshot.Metadata["restore_scores_json"].(map[string]any)
	if !ok || scores["decision"] != "selected" {
		t.Fatalf("expected selected restore_scores_json metadata, got %#v", packet.RestoreSnapshot.Metadata["restore_scores_json"])
	}
	if got := packet.RestoreSnapshot.Evidence["delta"]; got == nil {
		t.Fatalf("expected persisted delta evidence")
	}
}

func TestSQLiteListContextSnapshotsFiltersByScopeQueryAndKind(t *testing.T) {
	ctx := context.Background()
	k, txRunner, _ := newSQLiteKernel(t, nil)
	read := txRunner.read

	mustCreateNote(ctx, k, "ctx-list-note-a", "list a")

	first := validSQLiteRequest(domain.ActionCompileContext, "ctx-list-compile-1", "ws-main")
	first.Payload = map[string]any{
		"query":           "summarize blockers",
		"budget":          map[string]any{"maxTokens": 50, "maxEvents": 20, "maxNotes": 20},
		"persistSnapshot": true,
		"snapshotKind":    "restore",
	}
	if res, err := k.Process(ctx, first); err != nil || !res.Success {
		t.Fatalf("first compile failed: err=%v res=%+v", err, res)
	}

	second := validSQLiteRequest(domain.ActionCompileContext, "ctx-list-compile-2", "ws-main")
	second.RequestedAt = first.RequestedAt + 1000
	second.Payload = first.Payload
	if res, err := k.Process(ctx, second); err != nil || !res.Success {
		t.Fatalf("second compile failed: err=%v res=%+v", err, res)
	}

	otherKind := validSQLiteRequest(domain.ActionCompileContext, "ctx-list-compile-3", "ws-main")
	otherKind.RequestedAt = second.RequestedAt + 1000
	otherKind.Payload = map[string]any{
		"query":           "summarize blockers",
		"budget":          map[string]any{"maxTokens": 50, "maxEvents": 20, "maxNotes": 20},
		"persistSnapshot": true,
		"snapshotKind":    "review",
	}
	if res, err := k.Process(ctx, otherKind); err != nil || !res.Success {
		t.Fatalf("other-kind compile failed: err=%v res=%+v", err, res)
	}

	if _, err := txRunner.db.ExecContext(ctx, `
UPDATE context_packet_snapshots
SET created_at = ?
WHERE workspace_id = ? AND query = ? AND snapshot_kind = ?`, int64(1760001999000), "ws-main", "summarize blockers", "restore"); err != nil {
		t.Fatalf("force timestamp tie: %v", err)
	}
	list := read.ListContextSnapshots(domain.ForgeScope{WorkspaceID: "ws-main", LaneID: "control.semantic"}, "summarize blockers", "restore", 10)
	if len(list) != 2 {
		t.Fatalf("expected 2 restore snapshots, got %d", len(list))
	}
	if list[0].CreatedAt < list[1].CreatedAt {
		t.Fatalf("expected descending created_at order, got %d then %d", list[0].CreatedAt, list[1].CreatedAt)
	}
	if list[0].CreatedAt == list[1].CreatedAt && list[0].ID < list[1].ID {
		t.Fatalf("expected deterministic id-desc tie ordering, got %q then %q", list[0].ID, list[1].ID)
	}
}

func TestSQLiteContextCompileNoDirectPersistenceByReadStore(t *testing.T) {
	ctx := context.Background()
	_, txRunner, _ := newSQLiteKernel(t, nil)

	_ = txRunner.read.BuildContext("summarize blockers", domain.ForgeScope{WorkspaceID: "ws-main", LaneID: "control.semantic"}, domain.ContextBudget{
		MaxTokens: 50,
		MaxEvents: 20,
		MaxNotes:  20,
	}, 1760001007777)

	var snapshotCount int
	if err := txRunner.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM context_packet_snapshots`).Scan(&snapshotCount); err != nil {
		t.Fatalf("count context snapshots: %v", err)
	}
	if snapshotCount != 0 {
		t.Fatalf("read-store BuildContext must not persist snapshot rows, got %d", snapshotCount)
	}
	var artifactCount int
	if err := txRunner.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM artifact_refs`).Scan(&artifactCount); err != nil {
		t.Fatalf("count artifact refs: %v", err)
	}
	if artifactCount != 0 {
		t.Fatalf("read-store BuildContext must not persist artifact refs, got %d", artifactCount)
	}
}

func TestSQLiteBuildContextPreFiltersSnapshotArtifactsAndCompileEvents(t *testing.T) {
	ctx := context.Background()
	_, txRunner, _ := newSQLiteKernel(t, nil)
	read := txRunner.read
	scope := domain.ForgeScope{WorkspaceID: "ws-main", LaneID: "control.semantic"}

	if err := read.CreateArtifact(ctx, domain.ArtifactRef{
		ID:          "artifact-regular",
		Type:        "document",
		URI:         "artifact://evidence/regular.txt",
		Scope:       scope,
		ContentHash: "sha1:regular",
		CreatedAt:   1760001000100,
		Metadata:    map[string]any{"kind": "evidence"},
	}); err != nil {
		t.Fatalf("seed regular artifact: %v", err)
	}
	if err := read.CreateArtifact(ctx, domain.ArtifactRef{
		ID:          "artifact-snapshot-card",
		Type:        "context_snapshot_card",
		URI:         "artifact://context_snapshot/card.svg",
		Scope:       scope,
		ContentHash: "sha1:snapshot",
		CreatedAt:   1760001000200,
		Metadata:    map[string]any{"kind": "context_snapshot_card"},
	}); err != nil {
		t.Fatalf("seed snapshot-card artifact: %v", err)
	}

	if err := read.Append(ctx, domain.JournalEvent{
		ID:            "evt-user",
		Type:          "memory.note.created",
		Source:        string(domain.SourceUser),
		Scope:         scope,
		Payload:       map[string]any{"ok": true},
		CorrelationID: "corr-buildctx-1",
		Provenance: domain.Provenance{
			Actor:     "tester",
			ActorType: "test",
			Source:    "test",
			TraceID:   "trace-buildctx-1",
		},
		Timestamp: 1760001000300,
	}); err != nil {
		t.Fatalf("seed non-compile event: %v", err)
	}
	if err := read.Append(ctx, domain.JournalEvent{
		ID:            "evt-compile",
		Type:          "semantic_syscall.compile_context",
		Source:        string(domain.SourceSystem),
		Scope:         scope,
		Payload:       map[string]any{"internal": true},
		CorrelationID: "corr-buildctx-2",
		Provenance: domain.Provenance{
			Actor:     "forge",
			ActorType: "system",
			Source:    "test",
			TraceID:   "trace-buildctx-2",
		},
		Timestamp: 1760001000400,
	}); err != nil {
		t.Fatalf("seed compile event: %v", err)
	}

	packet := read.BuildContext("filter-check", scope, domain.ContextBudget{
		MaxTokens: 100,
		MaxEvents: 1,
		MaxNotes:  1,
	}, 1760001000500)

	if len(packet.Artifacts) != 1 || packet.Artifacts[0].ID != "artifact-regular" {
		t.Fatalf("expected build context to keep non-snapshot artifacts in-budget, got %+v", packet.Artifacts)
	}
	if len(packet.RawEvents) != 1 || packet.RawEvents[0].ID != "evt-user" {
		t.Fatalf("expected build context to keep non-compile events in-budget, got %+v", packet.RawEvents)
	}
}
