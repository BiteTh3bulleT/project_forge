package forgekernel_test

import (
	"context"
	"strings"
	"testing"

	"forge/projectforge/services/core/internal/aios/controllane"
	"forge/projectforge/services/core/internal/aios/domain"
	. "forge/projectforge/services/core/internal/forgekernel"
	"forge/projectforge/services/core/internal/forgekernel/court"
	"forge/projectforge/services/core/internal/store"
)

func TestForgeKCourthousePersistsCurrentAndImmutableHistory(t *testing.T) {
	ctx := context.Background()
	selection, st, _ := newLiveSQLiteAuthority(t)
	req := liveCourtRequest(domain.ActionAdmitEvidence, "court-admit-1")
	result, err := selection.Processor.Process(ctx, req)
	if err != nil || !result.Success {
		t.Fatalf("admit evidence: err=%v result=%#v", err, result)
	}
	summary, _ := result.StateSummary["courthouse"].(map[string]any)
	if summary["decision"] != court.DecisionAdmitted || summary["kernelDecisionAuthority"] != true || summary["modelAdmissionAuthority"] != false {
		t.Fatalf("unexpected Courthouse summary: %#v", summary)
	}
	read := controllane.NewSQLiteSemanticStore(st.DB)
	exhibit, ok := read.FindCourtExhibit("exhibit-1", req.Scope)
	if !ok || exhibit.Status != court.DecisionAdmitted || exhibit.CurrentRulingID != "ruling-1" || exhibit.AuditID == "" {
		t.Fatalf("unexpected current exhibit: ok=%v exhibit=%#v", ok, exhibit)
	}
	rulings := read.ListCourtRulings(req.Scope, "case-1", "exhibit-1")
	if len(rulings) != 1 || rulings[0].ID != "ruling-1" || rulings[0].PriorRulingID != "" || rulings[0].AuditID == "" {
		t.Fatalf("unexpected ruling history: %#v", rulings)
	}
	var journals, provenance int
	if err := st.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM journal_events WHERE id = ?`, req.ID+":journal_event").Scan(&journals); err != nil {
		t.Fatal(err)
	}
	if err := st.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM provenance_records WHERE syscall_id = ?`, req.ID).Scan(&provenance); err != nil {
		t.Fatal(err)
	}
	if journals != 1 || provenance == 0 {
		t.Fatalf("missing atomic lineage: journals=%d provenance=%d", journals, provenance)
	}
	if _, err := st.DB.ExecContext(ctx, `UPDATE court_rulings SET decision = 'rejected' WHERE id = ?`, "ruling-1"); err == nil {
		t.Fatal("immutable ruling accepted an update")
	}
	if _, err := st.DB.ExecContext(ctx, `DELETE FROM court_rulings WHERE id = ?`, "ruling-1"); err == nil {
		t.Fatal("append-only ruling accepted a delete")
	}
}

func TestForgeKCourthousePersistsRejectionAndAppealWithoutRewritingPriorRuling(t *testing.T) {
	ctx := context.Background()
	selection, st, _ := newLiveSQLiteAuthority(t)
	initial := liveCourtRequest(domain.ActionAdmitEvidence, "court-reject-1")
	initial.Payload["sourceRefs"] = []string{}
	initial.Payload["contentHash"] = ""
	first, err := selection.Processor.Process(ctx, initial)
	if err != nil || !first.Success {
		t.Fatalf("persist rejection: err=%v result=%#v", err, first)
	}
	firstSummary := first.StateSummary["courthouse"].(map[string]any)
	if firstSummary["decision"] != court.DecisionRejected {
		t.Fatalf("policy rejection was not persisted: %#v", firstSummary)
	}

	appeal := liveCourtRequest(domain.ActionAppealRuling, "court-appeal-1")
	appeal.Payload = map[string]any{
		"caseId": "case-1", "exhibitId": "exhibit-1", "priorRulingId": "ruling-1",
		"appealId": "appeal-1", "rulingId": "ruling-2", "grounds": "stable artifact supplied",
		"newSourceRefs": []string{"artifact:corrected"}, "newContentHash": "sha256:" + strings.Repeat("d", 64),
		"policyRefs": []string{"policy:court-v1"},
	}
	second, err := selection.Processor.Process(ctx, appeal)
	if err != nil || !second.Success {
		t.Fatalf("appeal ruling: err=%v result=%#v", err, second)
	}
	read := controllane.NewSQLiteSemanticStore(st.DB)
	exhibit, ok := read.FindCourtExhibit("exhibit-1", appeal.Scope)
	if !ok || exhibit.Status != court.DecisionAdmitted || exhibit.CurrentRulingID != "ruling-2" {
		t.Fatalf("current truth not advanced: %#v", exhibit)
	}
	rulings := read.ListCourtRulings(appeal.Scope, "case-1", "exhibit-1")
	if len(rulings) != 2 || rulings[0].ID != "ruling-1" || rulings[0].Decision != court.DecisionRejected || rulings[1].PriorRulingID != "ruling-1" || rulings[1].Decision != court.DecisionAdmitted {
		t.Fatalf("historical truth was not preserved: %#v", rulings)
	}
	var appealRows int
	if err := st.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM court_appeals WHERE id = ? AND prior_ruling_id = ? AND new_ruling_id = ?`, "appeal-1", "ruling-1", "ruling-2").Scan(&appealRows); err != nil || appealRows != 1 {
		t.Fatalf("appeal history missing: rows=%d err=%v", appealRows, err)
	}
	otherScope := domain.ForgeScope{WorkspaceID: "ws-other", LaneID: appeal.Scope.LaneID}
	if _, visible := read.FindCourtExhibit("exhibit-1", otherScope); visible || len(read.ListCourtRulings(otherScope, "case-1", "exhibit-1")) != 0 {
		t.Fatal("Courthouse records leaked across workspace scope")
	}
}

func TestCourthouseFailsClosedOutsideKernelAndForProposerActors(t *testing.T) {
	ctx := context.Background()
	for name, source := range map[string]domain.ActionSource{
		"adapter":     domain.SourceAdapter,
		"future_iris": domain.SourceFutureIRIS,
	} {
		t.Run(name, func(t *testing.T) {
			selection, st, _ := newLiveSQLiteAuthority(t)
			req := liveCourtRequest(domain.ActionAdmitEvidence, "court-source-"+name)
			req.Source = source
			req.Actor.Kind = string(source)
			result, err := selection.Processor.Process(ctx, req)
			if err != nil || result.Success || result.DeterministicErrCode != domain.ErrCapabilityDenied {
				t.Fatalf("proposer source did not fail closed: err=%v result=%#v", err, result)
			}
			assertNoCourtRows(t, st, req.ID)
		})
	}
	t.Run("model actor", func(t *testing.T) {
		selection, st, _ := newLiveSQLiteAuthority(t)
		req := liveCourtRequest(domain.ActionAdmitEvidence, "court-model")
		req.Actor.Kind = "llm_model"
		result, err := selection.Processor.Process(ctx, req)
		if err != nil || result.Success || result.DeterministicErrCode != domain.ErrUnauthorized {
			t.Fatalf("model actor did not fail closed: err=%v result=%#v", err, result)
		}
		assertNoCourtRows(t, st, req.ID)
	})
	t.Run("legacy rollback cannot rule", func(t *testing.T) {
		_, st, _ := newLiveSQLiteAuthority(t)
		adapter := controllane.NewProcessor(controllane.ProcessorOptions{TxRunner: controllane.NewSQLiteTransactionRunner(st.DB)})
		legacy, err := SelectAuthority(string(ModeLegacyV1), adapter)
		if err != nil {
			t.Fatal(err)
		}
		req := liveCourtRequest(domain.ActionAdmitEvidence, "court-legacy")
		result, err := legacy.Processor.Process(ctx, req)
		if err != nil || result.Success || result.DeterministicErrCode != domain.ErrUnauthorized {
			t.Fatalf("legacy authority admitted evidence: err=%v result=%#v", err, result)
		}
		assertNoCourtRows(t, st, req.ID)
	})
}

func TestForgeKCourthouseJournalFailureRollsBackAllRows(t *testing.T) {
	ctx := context.Background()
	selection, st, _ := newLiveSQLiteAuthority(t)
	req := liveCourtRequest(domain.ActionAdmitEvidence, "court-journal-failure")
	if _, err := st.DB.ExecContext(ctx, `INSERT INTO journal_events(id, type, source, workspace_id, created_at) VALUES(?,?,?,?,?)`, req.ID+":journal_event", "preexisting", "test", req.Scope.WorkspaceID, req.RequestedAt); err != nil {
		t.Fatal(err)
	}
	result, err := selection.Processor.Process(ctx, req)
	if err != nil || result.Success || result.DeterministicErrCode != domain.ErrPersistenceUnavailable {
		t.Fatalf("journal failure did not fail closed: err=%v result=%#v", err, result)
	}
	assertNoCourtRows(t, st, req.ID)
}

func liveCourtRequest(action domain.SemanticActionType, id string) domain.SyscallRequest {
	return domain.SyscallRequest{
		ID: id, Action: action,
		Actor: domain.ActorIdentity{ID: "operator", Kind: string(domain.SourceUser)}, Source: domain.SourceUser,
		Scope: domain.ForgeScope{WorkspaceID: "ws-court", LaneID: "control.semantic"},
		Payload: map[string]any{
			"caseId": "case-1", "exhibitId": "exhibit-1", "rulingId": "ruling-1",
			"sourceType": "artifact", "sourceRefs": []string{"artifact:1"}, "contentSummary": "bounded evidence",
			"rawRef": "artifact:1", "contentHash": "sha256:" + strings.Repeat("a", 64), "policyRefs": []string{"policy:court-v1"},
		},
		Provenance:    domain.Provenance{Actor: "operator", ActorType: "user", Source: "test", TraceID: "trace-" + id},
		CorrelationID: "corr-" + id, TraceID: "trace-" + id, RequestedAt: 1760000000000,
	}
}

func assertNoCourtRows(t *testing.T, st *store.Store, syscallID string) {
	t.Helper()
	for _, table := range []string{"court_exhibits", "court_rulings", "court_appeals"} {
		var count int
		if err := st.DB.QueryRow(`SELECT COUNT(*) FROM `+table+` WHERE syscall_id = ?`, syscallID).Scan(&count); err != nil {
			t.Fatalf("query %s: %v", table, err)
		}
		if count != 0 {
			t.Fatalf("failed request persisted %d rows in %s", count, table)
		}
	}
}
