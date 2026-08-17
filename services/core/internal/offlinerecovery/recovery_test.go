//go:build linux

package offlinerecovery

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forge/projectforge/services/core/internal/aios/controllane"
	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/backup"
	"forge/projectforge/services/core/internal/forgekernel"
	"forge/projectforge/services/core/internal/forgekernel/commitproof"
	"forge/projectforge/services/core/internal/store"
)

func TestRecoverRequiresStoppedDaemonAndRestartsVerifiedStore(t *testing.T) {
	ctx := context.Background()
	targetDir := t.TempDir()
	target := openRecoveryTestStore(t, targetDir, "prior-store")
	bundlePath := stageRecoveryTestBundle(t, targetDir, "recovered-store", true)

	if _, err := Recover(ctx, Request{DataDir: targetDir, BundlePath: bundlePath}); !errors.Is(err, store.ErrDataDirLocked) {
		t.Fatalf("recovery while daemon store is open error = %v, want ErrDataDirLocked", err)
	}
	if err := target.Close(); err != nil {
		t.Fatal(err)
	}

	result, err := Recover(ctx, Request{DataDir: targetDir, BundlePath: bundlePath})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Applied || result.RolledBack || result.PriorStoreBackup == "" || result.JournalEntries != 1 {
		t.Fatalf("unexpected recovery result: %#v", result)
	}
	if got := readOnlyThreadTitle(t, result.PriorStoreBackup); got != "prior-store" {
		t.Fatalf("preserved prior store title = %q", got)
	}

	restarted, err := store.Open(targetDir)
	if err != nil {
		t.Fatalf("restart recovered store: %v", err)
	}
	defer restarted.Close()
	if got := threadTitle(t, restarted); got != "recovered-store" {
		t.Fatalf("recovered title = %q", got)
	}
	report, err := controllane.NewSQLiteSemanticStore(restarted.DB).VerifyJournalChain(ctx)
	if err != nil || !report.Passed || report.EntryCount != 1 {
		t.Fatalf("restarted journal verification = %#v, %v", report, err)
	}
}

func TestRecoverRejectsManifestAndJournalTamperingBeforeSwap(t *testing.T) {
	ctx := context.Background()
	for _, test := range []struct {
		name   string
		tamper func(*testing.T, string)
		want   string
	}{
		{name: "section checksum", tamper: tamperWithoutChecksum, want: "checksum"},
		{name: "journal chain", tamper: tamperJournalAndRechecksum, want: "journal"},
	} {
		t.Run(test.name, func(t *testing.T) {
			targetDir := t.TempDir()
			target := openRecoveryTestStore(t, targetDir, "untouched")
			bundlePath := stageRecoveryTestBundle(t, targetDir, "candidate", true)
			if err := target.Close(); err != nil {
				t.Fatal(err)
			}
			test.tamper(t, bundlePath)
			if _, err := Recover(ctx, Request{DataDir: targetDir, BundlePath: bundlePath}); err == nil || !strings.Contains(strings.ToLower(err.Error()), test.want) {
				t.Fatalf("tampered recovery error = %v, want %q", err, test.want)
			}
			if got := readOnlyThreadTitle(t, filepath.Join(targetDir, "forge.sqlite")); got != "untouched" {
				t.Fatalf("rejected recovery changed current store: %q", got)
			}
		})
	}
}

func TestRecoverRollsBackFailedPostSwapVerificationAndPreservesBackup(t *testing.T) {
	ctx := context.Background()
	targetDir := t.TempDir()
	target := openRecoveryTestStore(t, targetDir, "rollback-prior")
	bundlePath := stageRecoveryTestBundle(t, targetDir, "rollback-candidate", false)
	if err := target.Close(); err != nil {
		t.Fatal(err)
	}

	result, err := recoverWithHooks(ctx, Request{DataDir: targetDir, BundlePath: bundlePath}, recoveryHooks{
		afterSwap: func() error { return errors.New("injected post-swap failure") },
	})
	if err == nil || result == nil || !result.RolledBack || result.Applied {
		t.Fatalf("rollback result = %#v, error = %v", result, err)
	}
	if got := readOnlyThreadTitle(t, filepath.Join(targetDir, "forge.sqlite")); got != "rollback-prior" {
		t.Fatalf("rollback current title = %q", got)
	}
	if got := readOnlyThreadTitle(t, result.PriorStoreBackup); got != "rollback-prior" {
		t.Fatalf("recoverable prior backup title = %q", got)
	}
	restarted, err := store.Open(targetDir)
	if err != nil {
		t.Fatalf("restart rolled-back store: %v", err)
	}
	defer restarted.Close()
}

func TestRecoverPreservesKernelIdempotencyAuditAndDeliveryProofIdentities(t *testing.T) {
	ctx := context.Background()
	targetDir := t.TempDir()
	target := openRecoveryTestStore(t, targetDir, "proof-prior")
	bundlePath := stageProofRecoveryBundle(t, targetDir)
	if err := target.Close(); err != nil {
		t.Fatal(err)
	}

	result, err := Recover(ctx, Request{DataDir: targetDir, BundlePath: bundlePath})
	if err != nil {
		t.Fatal(err)
	}
	if result.IdempotencyProofs != 2 || result.AuditProofs != 2 || result.AuditAttempts != 1 || result.CourtExhibits != 1 || result.CourtRulings != 1 {
		t.Fatalf("proof counts not preserved: %#v", result)
	}
	restarted, err := store.Open(targetDir)
	if err != nil {
		t.Fatal(err)
	}
	defer restarted.Close()
	var attemptID, auditOutboxID string
	if err := restarted.DB.QueryRow(`SELECT id,outbox_id FROM forge_k_audit_delivery_attempts`).Scan(&attemptID, &auditOutboxID); err != nil {
		t.Fatal(err)
	}
	if attemptID != "attempt-proof-1" || auditOutboxID != "sys-proof:audit_outbox" {
		t.Fatalf("delivery identity changed: attempt=%q outbox=%q", attemptID, auditOutboxID)
	}
	var projectedOutbox string
	if err := restarted.DB.QueryRow(`SELECT forge_k_outbox_id FROM audit_records WHERE id=77`).Scan(&projectedOutbox); err != nil {
		t.Fatal(err)
	}
	if projectedOutbox != auditOutboxID {
		t.Fatalf("audit projection identity = %q, want %q", projectedOutbox, auditOutboxID)
	}
	var exhibitID, rulingID string
	if err := restarted.DB.QueryRow(`SELECT id,current_ruling_id FROM court_exhibits`).Scan(&exhibitID, &rulingID); err != nil {
		t.Fatal(err)
	}
	if exhibitID != "exhibit-proof" || rulingID != "ruling-proof" {
		t.Fatalf("Court identity changed: exhibit=%q ruling=%q", exhibitID, rulingID)
	}
}

func stageRecoveryTestBundle(t *testing.T, targetDir, title string, withJournal bool) string {
	t.Helper()
	ctx := context.Background()
	sourceDir := t.TempDir()
	source := openRecoveryTestStore(t, sourceDir, title)
	if withJournal {
		journal := controllane.NewSQLiteSemanticStore(source.DB)
		journal.SetCommitMetadata(controllane.CommitMetadata{
			SyscallID: "sys-recovery", CorrelationID: "corr-recovery", TraceID: "trace-recovery",
			Source: domain.SourceUser, CommittedBy: "forge_k.kernel",
		})
		_, err := journal.AppendWithEvidence(ctx, domain.JournalEvent{
			ID: "evt-recovery", Type: "semantic_syscall.test", Timestamp: 1760001000001,
			Source: "forge_kernel", Scope: domain.ForgeScope{WorkspaceID: "ws-recovery", LaneID: "control.semantic"},
			Payload: map[string]any{"action": "TEST_RECOVERY"}, CorrelationID: "corr-recovery",
			Provenance: domain.Provenance{Actor: "operator", ActorType: "user", Source: "test", TraceID: "trace-recovery"},
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	bundleRecord, err := backup.New(source.DB, sourceDir).CreateBundle(ctx, backup.CreateBundleRequest{Kind: "full_backup", Label: "recovery-test", SourceVer: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if err := source.Close(); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(bundleRecord.FilePath)
	if err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(targetDir, "backups", filepath.Base(bundleRecord.FilePath))
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destination, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	return destination
}

func stageProofRecoveryBundle(t *testing.T, targetDir string) string {
	t.Helper()
	ctx := context.Background()
	sourceDir := t.TempDir()
	source := openRecoveryTestStore(t, sourceDir, "proof-candidate")
	authorization, err := controllane.NewProductionAuthorizationService(controllane.ProductionAuthorizationOptions{
		Registry: controllane.NewStaticActionRegistry(), DB: source.DB,
		ServicePrincipal: controllane.NewForgeCoreServicePrincipal(),
	})
	if err != nil {
		t.Fatal(err)
	}
	adapter := controllane.NewProcessor(controllane.ProcessorOptions{
		TxRunner:  controllane.NewSQLiteTransactionRunner(source.DB),
		AuditSink: controllane.NewInMemoryAuditSink(), NowMillis: func() int64 { return 1760002000000 },
	})
	selection, err := forgekernel.SelectAuthority(adapter, authorization)
	if err != nil {
		t.Fatal(err)
	}
	req := domain.SyscallRequest{
		ID: "sys-proof", Action: domain.ActionCreateNote,
		Actor: domain.ActorIdentity{ID: "forge.core", Kind: string(domain.SourceSystem)}, Source: domain.SourceSystem,
		Scope:         domain.ForgeScope{WorkspaceID: "ws-proof"},
		Payload:       map[string]any{"id": "note-proof", "type": string(domain.NoteFact), "title": "proof", "content": "recovery proof"},
		Provenance:    domain.Provenance{Actor: "forge.core", ActorType: "system", Source: "recovery-test", TraceID: "trace-proof"},
		CorrelationID: "corr-proof", TraceID: "trace-proof", IdempotencyKey: "idem-proof", RequestedAt: 1760002000000,
	}
	result, err := selection.Processor.Process(ctx, req)
	if err != nil || !result.Success {
		t.Fatalf("create kernel proof: result=%#v err=%v", result, err)
	}
	courtReq := domain.SyscallRequest{
		ID: "sys-court-proof", Action: domain.ActionAdmitEvidence,
		Actor: domain.ActorIdentity{ID: "forge.core", Kind: string(domain.SourceSystem)}, Source: domain.SourceSystem,
		Scope: domain.ForgeScope{WorkspaceID: "ws-proof", LaneID: "control.semantic"},
		Payload: map[string]any{
			"caseId": "case-proof", "exhibitId": "exhibit-proof", "rulingId": "ruling-proof",
			"sourceType": "artifact", "sourceRefs": []string{"artifact:proof"}, "contentSummary": "recovery evidence",
			"rawRef": "artifact:proof", "contentHash": "sha256:" + strings.Repeat("a", 64), "policyRefs": []string{"policy:court-v1"},
		},
		Provenance:    domain.Provenance{Actor: "forge.core", ActorType: "system", Source: "recovery-test", TraceID: "trace-court-proof"},
		CorrelationID: "corr-court-proof", TraceID: "trace-court-proof", IdempotencyKey: "idem-court-proof", RequestedAt: 1760002000001,
	}
	courtResult, err := selection.Processor.Process(ctx, courtReq)
	if err != nil || !courtResult.Success {
		t.Fatalf("create Court proof: result=%#v err=%v", courtResult, err)
	}
	var courtRequestRaw, courtFingerprint string
	if err := source.DB.QueryRow(`SELECT request_json,request_fingerprint FROM semantic_idempotency_keys WHERE idempotency_key=?`, courtReq.IdempotencyKey).Scan(&courtRequestRaw, &courtFingerprint); err != nil {
		t.Fatal(err)
	}
	var persistedCourtRequest domain.SyscallRequest
	if err := decodeProofJSON(courtRequestRaw, &persistedCourtRequest); err != nil {
		t.Fatal(err)
	}
	if err := hydrateKernelMetadata(&persistedCourtRequest); err != nil {
		t.Fatal(err)
	}
	computedCourtFingerprint, err := commitproof.FingerprintRequest(persistedCourtRequest)
	if err != nil || computedCourtFingerprint != courtFingerprint {
		t.Fatalf("source Court proof is not self-consistent: stored=%s computed=%s err=%v request=%s", courtFingerprint, computedCourtFingerprint, err, courtRequestRaw)
	}
	var fingerprint string
	if err := source.DB.QueryRow(`SELECT request_fingerprint FROM forge_k_audit_outbox WHERE id=?`, "sys-proof:audit_outbox").Scan(&fingerprint); err != nil {
		t.Fatal(err)
	}
	if _, err := source.DB.Exec(`INSERT INTO audit_records(id,forge_k_outbox_id,created_at,correlation_id,category,action,actor,outcome,summary,payload_json)
VALUES(77,?,?,?,?,?,?,?,?,?)`, "sys-proof:audit_outbox", 1760002000001, "corr-proof", "forge_k", string(req.Action), "forge.core", "success", "projected", `{}`); err != nil {
		t.Fatal(err)
	}
	if _, err := source.DB.Exec(`INSERT INTO forge_k_audit_delivery_attempts(id,outbox_id,request_fingerprint,attempt_number,status,audit_id,created_at)
VALUES(?,?,?,?,?,?,?)`, "attempt-proof-1", "sys-proof:audit_outbox", fingerprint, 1, "delivered", "77", 1760002000001); err != nil {
		t.Fatal(err)
	}
	bundleRecord, err := backup.New(source.DB, sourceDir).CreateBundle(ctx, backup.CreateBundleRequest{Kind: "full_backup", Label: "proof-recovery", SourceVer: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if err := source.Close(); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(bundleRecord.FilePath)
	if err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(targetDir, "backups", filepath.Base(bundleRecord.FilePath))
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destination, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	return destination
}

func openRecoveryTestStore(t *testing.T, dataDir, title string) *store.Store {
	t.Helper()
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.DB.Exec(`INSERT INTO chat_threads(title,created_at,updated_at) VALUES(?,?,?)`, title, 1, 1); err != nil {
		_ = st.Close()
		t.Fatal(err)
	}
	return st
}

func threadTitle(t *testing.T, st *store.Store) string {
	t.Helper()
	var title string
	if err := st.DB.QueryRow(`SELECT title FROM chat_threads ORDER BY id LIMIT 1`).Scan(&title); err != nil {
		t.Fatal(err)
	}
	return title
}

func readOnlyThreadTitle(t *testing.T, path string) string {
	t.Helper()
	db, err := openExistingSQLite(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var title string
	if err := db.QueryRow(`SELECT title FROM chat_threads ORDER BY id LIMIT 1`).Scan(&title); err != nil {
		t.Fatal(err)
	}
	return title
}

func tamperWithoutChecksum(t *testing.T, path string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	tampered := strings.Replace(string(raw), `"title": "candidate"`, `"title": "tampered"`, 1)
	if tampered == string(raw) {
		t.Fatal("candidate title not found in bundle")
	}
	if err := os.WriteFile(path, []byte(tampered), 0o600); err != nil {
		t.Fatal(err)
	}
}

func tamperJournalAndRechecksum(t *testing.T, path string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc backup.BundleDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	rows := doc.Entities["journal_events"]
	if len(rows) != 1 {
		t.Fatalf("journal row count = %d", len(rows))
	}
	record := rows[0].(map[string]any)
	record["payload_json"] = `{"action":"TAMPERED"}`
	canonical, err := json.Marshal(rows)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(canonical)
	doc.Checksums["journal_events"] = hex.EncodeToString(sum[:])
	updated, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, updated, 0o600); err != nil {
		t.Fatal(err)
	}
}
