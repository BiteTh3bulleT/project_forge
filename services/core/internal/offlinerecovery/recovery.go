// Package offlinerecovery implements the daemon-stopped FORGE-K whole-store
// recovery boundary. It never opens or mutates the live restore API path.
package offlinerecovery

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"forge/projectforge/services/core/internal/aios/controllane"
	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/backup"
	"forge/projectforge/services/core/internal/forgekernel"
	"forge/projectforge/services/core/internal/forgekernel/authproof"
	"forge/projectforge/services/core/internal/forgekernel/commitproof"
	"forge/projectforge/services/core/internal/forgekernel/contextcompile"
	"forge/projectforge/services/core/internal/forgekernel/court"
	"forge/projectforge/services/core/internal/forgekernel/semanticdiff"
	"forge/projectforge/services/core/internal/store"

	_ "modernc.org/sqlite"
)

const ResultVersion = "forge_k.offline_recovery.v1"

type Request struct {
	DataDir    string
	BundlePath string
}

type Result struct {
	Version           string `json:"version"`
	Applied           bool   `json:"applied"`
	RolledBack        bool   `json:"rolledBack"`
	BundleSHA256      string `json:"bundleSha256"`
	InspectionPlan    string `json:"inspectionPlanDigest"`
	RestoredStorePath string `json:"restoredStorePath"`
	PriorStoreBackup  string `json:"priorStoreBackup"`
	JournalEntries    int    `json:"journalEntries"`
	IdempotencyProofs int    `json:"idempotencyProofs"`
	AuditProofs       int    `json:"auditProofs"`
	AuditAttempts     int    `json:"auditDeliveryAttempts"`
	CourtExhibits     int    `json:"courtExhibits"`
	CourtRulings      int    `json:"courtRulings"`
	CourtAppeals      int    `json:"courtAppeals"`
}

type recoveryHooks struct {
	afterSwap func() error
}

type verificationCounts struct {
	journal, idempotency, audit, attempts, exhibits, rulings, appeals int
}

func Recover(ctx context.Context, req Request) (*Result, error) {
	return recoverWithHooks(ctx, req, recoveryHooks{})
}

func recoverWithHooks(ctx context.Context, req Request, hooks recoveryHooks) (_ *Result, retErr error) {
	dataDir, targetPath, err := validateRequest(req)
	if err != nil {
		return nil, err
	}
	lock, err := store.AcquireOfflineRecoveryLock(dataDir)
	if err != nil {
		return nil, fmt.Errorf("offline recovery requires stopped daemon: %w", err)
	}
	defer lock.Close()

	bundleSvc := backup.New(nil, dataDir)
	doc, inspection, err := bundleSvc.LoadVerifiedFullBundle(ctx, req.BundlePath)
	if err != nil {
		return nil, fmt.Errorf("verify recovery bundle: %w", err)
	}
	result := &Result{
		Version: ResultVersion, BundleSHA256: inspection.BundleSHA256,
		InspectionPlan: inspection.PlanDigest, RestoredStorePath: targetPath,
	}

	stageDir, err := os.MkdirTemp(dataDir, ".forge-recovery-stage-")
	if err != nil {
		return nil, fmt.Errorf("create recovery staging directory: %w", err)
	}
	defer os.RemoveAll(stageDir)
	staged, err := store.Open(stageDir)
	if err != nil {
		return nil, fmt.Errorf("create migrated staging store: %w", err)
	}
	stageClosed := false
	defer func() {
		if !stageClosed {
			_ = staged.Close()
		}
	}()
	if err := importWholeBundle(ctx, staged.DB, doc); err != nil {
		return nil, fmt.Errorf("restore staging store: %w", err)
	}
	counts, err := verifyStore(ctx, staged.DB, doc)
	if err != nil {
		return nil, fmt.Errorf("verify staging store: %w", err)
	}
	if _, err := staged.DB.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		return nil, fmt.Errorf("checkpoint staging store: %w", err)
	}
	if err := staged.Close(); err != nil {
		return nil, fmt.Errorf("close staging store: %w", err)
	}
	stageClosed = true
	stagePath := filepath.Join(stageDir, "forge.sqlite")
	if err := prepareReplacementMetadata(targetPath, stagePath); err != nil {
		return nil, fmt.Errorf("preserve current store ownership and mode: %w", err)
	}
	if err := syncFile(stagePath); err != nil {
		return nil, fmt.Errorf("sync staging store: %w", err)
	}

	if err := checkpointStoppedStore(ctx, targetPath); err != nil {
		return nil, fmt.Errorf("checkpoint stopped current store: %w", err)
	}
	priorPath, err := preservePriorStore(dataDir, targetPath)
	if err != nil {
		return nil, fmt.Errorf("preserve prior store: %w", err)
	}
	result.PriorStoreBackup = priorPath
	for _, suffix := range []string{"-wal", "-shm"} {
		if err := os.Remove(targetPath + suffix); err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("remove checkpointed current store sidecar %s: %w", suffix, err)
		}
	}
	if err := os.Rename(stagePath, targetPath); err != nil {
		return nil, fmt.Errorf("atomic recovery store swap: %w", err)
	}
	if err := syncDir(dataDir); err != nil {
		if rollbackErr := rollbackStore(targetPath, priorPath); rollbackErr != nil {
			return nil, errors.Join(fmt.Errorf("sync recovered store directory: %w", err), rollbackErr)
		}
		result.RolledBack = true
		return result, fmt.Errorf("sync recovered store directory: %w", err)
	}
	if hooks.afterSwap != nil {
		if err := hooks.afterSwap(); err != nil {
			if rollbackErr := rollbackStore(targetPath, priorPath); rollbackErr != nil {
				return nil, errors.Join(err, rollbackErr)
			}
			result.RolledBack = true
			return result, err
		}
	}
	postDB, err := openExistingSQLite(targetPath)
	if err == nil {
		_, err = verifyStore(ctx, postDB, doc)
		closeErr := postDB.Close()
		err = errors.Join(err, closeErr)
	}
	if err != nil {
		if rollbackErr := rollbackStore(targetPath, priorPath); rollbackErr != nil {
			return nil, errors.Join(fmt.Errorf("post-swap verification failed: %w", err), rollbackErr)
		}
		result.RolledBack = true
		return result, fmt.Errorf("post-swap verification failed: %w", err)
	}
	result.Applied = true
	result.JournalEntries = counts.journal
	result.IdempotencyProofs = counts.idempotency
	result.AuditProofs = counts.audit
	result.AuditAttempts = counts.attempts
	result.CourtExhibits = counts.exhibits
	result.CourtRulings = counts.rulings
	result.CourtAppeals = counts.appeals
	return result, nil
}

func validateRequest(req Request) (string, string, error) {
	dataDir := strings.TrimSpace(req.DataDir)
	if dataDir == "" {
		return "", "", fmt.Errorf("data directory is required")
	}
	abs, err := filepath.Abs(dataDir)
	if err != nil {
		return "", "", err
	}
	if filepath.Clean(abs) == string(filepath.Separator) {
		return "", "", fmt.Errorf("filesystem root cannot be a recovery data directory")
	}
	target := filepath.Join(abs, "forge.sqlite")
	info, err := os.Lstat(target)
	if err != nil {
		return "", "", fmt.Errorf("current store is required: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "", "", fmt.Errorf("current store must be a regular non-symlink file")
	}
	if strings.TrimSpace(req.BundlePath) == "" {
		return "", "", fmt.Errorf("bundle path is required")
	}
	return abs, target, nil
}

func importWholeBundle(ctx context.Context, db *sql.DB, doc backup.BundleDoc) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `PRAGMA defer_foreign_keys=ON`); err != nil {
		return err
	}
	sections := make([]string, 0, len(doc.Entities))
	for section := range doc.Entities {
		sections = append(sections, section)
	}
	sort.SliceStable(sections, func(i, j int) bool {
		return recoveryPriority(sections[i]) < recoveryPriority(sections[j])
	})
	for _, section := range sections {
		if err := importSection(ctx, tx, section, doc.Entities[section]); err != nil {
			return fmt.Errorf("section %s: %w", section, err)
		}
	}
	if err := verifyForeignKeys(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}

func importSection(ctx context.Context, tx *sql.Tx, section string, rows []any) error {
	table := section
	if section == "autonomy_settings" {
		table = "settings"
	}
	columns, err := tableColumns(ctx, tx, table)
	if err != nil {
		return err
	}
	if section == "forge_k_journal_head" {
		if len(rows) != 1 {
			return fmt.Errorf("journal head section must contain exactly one row")
		}
		record, err := recordMap(rows[0])
		if err != nil {
			return err
		}
		if fmt.Sprint(record["id"]) != "1" {
			return fmt.Errorf("journal head id must be 1")
		}
		values := make([]any, 4)
		for i, field := range []string{"sequence", "event_id", "head_hash", "updated_at"} {
			values[i], err = sqliteValue(record[field])
			if err != nil {
				return fmt.Errorf("journal head column %q: %w", field, err)
			}
		}
		_, err = tx.ExecContext(ctx, `UPDATE forge_k_journal_head SET sequence=?,event_id=?,head_hash=?,updated_at=? WHERE id=1`, values...)
		return err
	}
	for _, row := range rows {
		record, err := recordMap(row)
		if err != nil {
			return err
		}
		fields := make([]string, 0, len(record))
		for field := range record {
			if _, ok := columns[field]; !ok {
				return fmt.Errorf("bundle column %q is absent from current schema table %q", field, table)
			}
			fields = append(fields, field)
		}
		sort.Strings(fields)
		placeholders := make([]string, len(fields))
		args := make([]any, len(fields))
		for i, field := range fields {
			placeholders[i] = "?"
			args[i], err = sqliteValue(record[field])
			if err != nil {
				return fmt.Errorf("column %q: %w", field, err)
			}
		}
		statement := fmt.Sprintf("INSERT INTO %s(%s) VALUES(%s)", quoteIdentifier(table), joinQuoted(fields), strings.Join(placeholders, ","))
		if _, err := tx.ExecContext(ctx, statement, args...); err != nil {
			return err
		}
	}
	return nil
}

func verifyStore(ctx context.Context, db *sql.DB, doc backup.BundleDoc) (verificationCounts, error) {
	if err := verifySQLiteIntegrity(ctx, db); err != nil {
		return verificationCounts{}, err
	}
	if err := backup.VerifyDatabaseSections(ctx, db, doc); err != nil {
		return verificationCounts{}, err
	}
	report, err := controllane.NewSQLiteSemanticStore(db).VerifyJournalChain(ctx)
	if err != nil {
		return verificationCounts{}, err
	}
	if !report.Passed {
		return verificationCounts{}, fmt.Errorf("FORGE-K journal chain/head verification failed: %+v", report.Issues)
	}
	counts, err := verifyCommitProofIdentities(ctx, db)
	if err != nil {
		return verificationCounts{}, err
	}
	counts.journal = report.EntryCount
	if err := verifyCourtIdentities(ctx, db, &counts); err != nil {
		return verificationCounts{}, err
	}
	return counts, nil
}

func verifyCommitProofIdentities(ctx context.Context, db *sql.DB) (verificationCounts, error) {
	rows, err := db.QueryContext(ctx, `SELECT idempotency_key,action,result_json,request_fingerprint,idempotency_fingerprint,request_json,plan_json,seal_json,receipt_json,authproof_json FROM semantic_idempotency_keys ORDER BY idempotency_key`)
	if err != nil {
		return verificationCounts{}, err
	}
	type proofRow struct {
		key, action, resultRaw, requestFingerprint, idempotencyFingerprint string
		requestRaw, planRaw, sealRaw, receiptRaw, authRaw                  string
	}
	records := []proofRow{}
	for rows.Next() {
		var record proofRow
		if err := rows.Scan(&record.key, &record.action, &record.resultRaw, &record.requestFingerprint, &record.idempotencyFingerprint, &record.requestRaw, &record.planRaw, &record.sealRaw, &record.receiptRaw, &record.authRaw); err != nil {
			_ = rows.Close()
			return verificationCounts{}, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return verificationCounts{}, err
	}
	if err := rows.Close(); err != nil {
		return verificationCounts{}, err
	}
	count := 0
	for _, record := range records {
		key, action := record.key, record.action
		resultRaw, requestFingerprint, idempotencyFingerprint := record.resultRaw, record.requestFingerprint, record.idempotencyFingerprint
		requestRaw, planRaw, sealRaw, receiptRaw, authRaw := record.requestRaw, record.planRaw, record.sealRaw, record.receiptRaw, record.authRaw
		var req domain.SyscallRequest
		var plan commitproof.PreparedPlan
		var seal commitproof.PreparedPlanSeal
		var receipt commitproof.CommitReceipt
		var proof authproof.Proof
		var result domain.SyscallResult
		for _, item := range []struct {
			name string
			raw  string
			out  any
		}{{"request", requestRaw, &req}, {"plan", planRaw, &plan}, {"seal", sealRaw, &seal}, {"receipt", receiptRaw, &receipt}, {"authproof", authRaw, &proof}, {"result", resultRaw, &result}} {
			if err := decodeProofJSON(item.raw, item.out); err != nil {
				return verificationCounts{}, fmt.Errorf("idempotency %q invalid %s JSON: %w", key, item.name, err)
			}
		}
		if err := hydrateKernelMetadata(&req); err != nil {
			return verificationCounts{}, fmt.Errorf("idempotency %q request metadata: %w", key, err)
		}
		if req.IdempotencyKey != key || string(req.Action) != action {
			return verificationCounts{}, fmt.Errorf("idempotency %q request identity mismatch", key)
		}
		computedRequest, err := commitproof.FingerprintRequest(req)
		if err != nil || computedRequest != requestFingerprint {
			return verificationCounts{}, fmt.Errorf("idempotency %q request fingerprint mismatch: stored=%q computed=%q error=%v", key, requestFingerprint, computedRequest, err)
		}
		computedIdempotency, err := commitproof.IdempotencyFingerprint(req)
		if err != nil || computedIdempotency != idempotencyFingerprint {
			return verificationCounts{}, fmt.Errorf("idempotency %q proof fingerprint mismatch", key)
		}
		if err := authproof.VerifyProof(req, proof); err != nil {
			return verificationCounts{}, fmt.Errorf("idempotency %q authorization proof: %w", key, err)
		}
		if err := authproof.VerifyPlanBinding(proof, authproof.PlanBinding{Action: plan.Action, Capability: plan.Capability, TargetObjectType: plan.TargetObjectType, Mutating: plan.Mutating, JournalEventType: plan.JournalEventType}); err != nil {
			return verificationCounts{}, fmt.Errorf("idempotency %q authorization plan binding: %w", key, err)
		}
		if err := commitproof.ValidateCommitReceipt(req, plan, seal, receipt, result); err != nil {
			return verificationCounts{}, fmt.Errorf("idempotency %q commit receipt: %w", key, err)
		}
		if err := verifyReceiptRows(ctx, db, req, receipt, requestFingerprint, requestRaw, receiptRaw, authRaw, resultRaw); err != nil {
			return verificationCounts{}, fmt.Errorf("idempotency %q durable proof identity: %w", key, err)
		}
		count++
	}
	auditCount, err := verifyAuditOutboxIdentities(ctx, db)
	if err != nil {
		return verificationCounts{}, err
	}
	var attemptCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM forge_k_audit_delivery_attempts attempt
WHERE NOT EXISTS(SELECT 1 FROM forge_k_audit_outbox outbox
  WHERE outbox.id=attempt.outbox_id AND outbox.request_fingerprint=attempt.request_fingerprint)`).Scan(&attemptCount); err != nil {
		return verificationCounts{}, err
	}
	if attemptCount != 0 {
		return verificationCounts{}, fmt.Errorf("audit delivery attempt/outbox proof identity mismatch for %d rows", attemptCount)
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM forge_k_audit_delivery_attempts`).Scan(&attemptCount); err != nil {
		return verificationCounts{}, err
	}
	return verificationCounts{idempotency: count, audit: auditCount, attempts: attemptCount}, nil
}

func verifyAuditOutboxIdentities(ctx context.Context, db *sql.DB) (int, error) {
	rows, err := db.QueryContext(ctx, `SELECT id,syscall_id,request_fingerprint,action,workspace_id,lane_id,correlation_id,trace_id,success,result_json,request_json,receipt_json,authproof_json,created_at,committed_by FROM forge_k_audit_outbox ORDER BY id`)
	if err != nil {
		return 0, err
	}
	type auditRow struct {
		id, syscallID, requestFingerprint, action, workspaceID, laneID string
		correlation, trace, resultRaw, requestRaw, receiptRaw, authRaw string
		committedBy                                                    string
		success, createdAt                                             int64
	}
	records := []auditRow{}
	for rows.Next() {
		var record auditRow
		if err := rows.Scan(&record.id, &record.syscallID, &record.requestFingerprint, &record.action, &record.workspaceID, &record.laneID, &record.correlation, &record.trace, &record.success, &record.resultRaw, &record.requestRaw, &record.receiptRaw, &record.authRaw, &record.createdAt, &record.committedBy); err != nil {
			_ = rows.Close()
			return 0, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return 0, err
	}
	if err := rows.Close(); err != nil {
		return 0, err
	}
	count := 0
	for _, record := range records {
		id, requestFingerprint := record.id, record.requestFingerprint
		resultRaw, requestRaw, receiptRaw, authRaw := record.resultRaw, record.requestRaw, record.receiptRaw, record.authRaw
		var req domain.SyscallRequest
		var receipt commitproof.CommitReceipt
		var proof authproof.Proof
		var result domain.SyscallResult
		for _, item := range []struct {
			name string
			raw  string
			out  any
		}{{"request", requestRaw, &req}, {"receipt", receiptRaw, &receipt}, {"authproof", authRaw, &proof}, {"result", resultRaw, &result}} {
			if err := decodeProofJSON(item.raw, item.out); err != nil {
				return 0, fmt.Errorf("audit outbox %q invalid %s JSON: %w", id, item.name, err)
			}
		}
		if err := hydrateKernelMetadata(&req); err != nil {
			return 0, fmt.Errorf("audit outbox %q request metadata: %w", id, err)
		}
		computedIdempotency, err := commitproof.IdempotencyFingerprint(req)
		if err != nil || receipt.IdempotencyFingerprint != computedIdempotency {
			return 0, fmt.Errorf("audit outbox %q idempotency fingerprint mismatch", id)
		}
		auditRecord := controllane.AuditOutboxRecord{
			ID: id, SyscallID: record.syscallID, RequestFingerprint: requestFingerprint,
			Action: domain.SemanticActionType(record.action), WorkspaceID: record.workspaceID, LaneID: record.laneID,
			CorrelationID: record.correlation, TraceID: record.trace, Success: record.success != 0,
			Result: result, Request: req, Receipt: receipt, AuthorizationProof: proof,
			CreatedAt: record.createdAt, CommittedBy: record.committedBy,
		}
		if err := controllane.VerifyAuditOutboxRecord(auditRecord); err != nil {
			return 0, fmt.Errorf("audit outbox %q typed proof identity: %w", id, err)
		}
		var eventHash, eventSyscall string
		if err := db.QueryRowContext(ctx, `SELECT event_hash,syscall_id FROM journal_events WHERE id=?`, receipt.JournalEventID).Scan(&eventHash, &eventSyscall); err != nil {
			return 0, fmt.Errorf("audit outbox %q journal proof: %w", id, err)
		}
		if eventHash != receipt.JournalEventHash || eventSyscall != req.ID {
			return 0, fmt.Errorf("audit outbox %q journal receipt identity mismatch", id)
		}
		count++
	}
	return count, nil
}

func verifyReceiptRows(ctx context.Context, db *sql.DB, req domain.SyscallRequest, receipt commitproof.CommitReceipt, requestFingerprint, requestRaw, receiptRaw, authRaw, resultRaw string) error {
	var eventHash, syscallID string
	if err := db.QueryRowContext(ctx, `SELECT event_hash,syscall_id FROM journal_events WHERE id=?`, receipt.JournalEventID).Scan(&eventHash, &syscallID); err != nil {
		return err
	}
	if eventHash != receipt.JournalEventHash || syscallID != req.ID {
		return fmt.Errorf("journal receipt identity mismatch")
	}
	var outboxID, outboxSyscall, outboxRequestFingerprint, action, correlation, trace, outboxResult, outboxRequest, outboxReceipt, outboxAuth, committedBy string
	if err := db.QueryRowContext(ctx, `SELECT id,syscall_id,request_fingerprint,action,correlation_id,trace_id,result_json,request_json,receipt_json,authproof_json,committed_by FROM forge_k_audit_outbox WHERE id=?`, receipt.AuditOutboxID).Scan(
		&outboxID, &outboxSyscall, &outboxRequestFingerprint, &action, &correlation, &trace, &outboxResult, &outboxRequest, &outboxReceipt, &outboxAuth, &committedBy,
	); err != nil {
		return err
	}
	if outboxID != receipt.AuditOutboxID || outboxSyscall != req.ID || outboxRequestFingerprint != requestFingerprint || action != string(req.Action) || correlation != req.CorrelationID || trace != req.TraceID || committedBy != forgekernel.AuthorityOwnerForgeK {
		return fmt.Errorf("audit outbox scalar identity mismatch")
	}
	for name, pair := range map[string][2]string{"request": {requestRaw, outboxRequest}, "receipt": {receiptRaw, outboxReceipt}, "authproof": {authRaw, outboxAuth}, "result": {resultRaw, outboxResult}} {
		if !sameJSON(pair[0], pair[1]) {
			return fmt.Errorf("audit outbox %s proof mismatch", name)
		}
	}
	return nil
}

func verifyCourtIdentities(ctx context.Context, db *sql.DB, counts *verificationCounts) error {
	checks := []struct {
		name  string
		query string
	}{
		{"court exhibit", `SELECT COUNT(*) FROM court_exhibits e WHERE e.committed_by!='forge_k.kernel' OR NOT EXISTS(SELECT 1 FROM court_rulings r WHERE r.id=e.current_ruling_id AND r.exhibit_id=e.id AND r.case_id=e.case_id AND r.workspace_id=e.workspace_id AND r.lane_id=e.lane_id) OR NOT EXISTS(SELECT 1 FROM journal_events j WHERE j.syscall_id=e.syscall_id) OR NOT EXISTS(SELECT 1 FROM forge_k_audit_outbox a WHERE a.syscall_id=e.syscall_id)`},
		{"court ruling", `SELECT COUNT(*) FROM court_rulings r WHERE r.committed_by!='forge_k.kernel' OR NOT EXISTS(SELECT 1 FROM court_exhibits e WHERE e.id=r.exhibit_id AND e.case_id=r.case_id AND e.workspace_id=r.workspace_id AND e.lane_id=r.lane_id) OR (r.prior_ruling_id!='' AND NOT EXISTS(SELECT 1 FROM court_rulings p WHERE p.id=r.prior_ruling_id AND p.exhibit_id=r.exhibit_id)) OR (r.appeal_id!='' AND NOT EXISTS(SELECT 1 FROM court_appeals a WHERE a.id=r.appeal_id AND a.new_ruling_id=r.id)) OR NOT EXISTS(SELECT 1 FROM journal_events j WHERE j.syscall_id=r.syscall_id) OR NOT EXISTS(SELECT 1 FROM forge_k_audit_outbox a WHERE a.syscall_id=r.syscall_id)`},
		{"court appeal", `SELECT COUNT(*) FROM court_appeals a WHERE a.committed_by!='forge_k.kernel' OR NOT EXISTS(SELECT 1 FROM court_exhibits e WHERE e.id=a.exhibit_id AND e.case_id=a.case_id AND e.workspace_id=a.workspace_id AND e.lane_id=a.lane_id) OR NOT EXISTS(SELECT 1 FROM court_rulings p WHERE p.id=a.prior_ruling_id AND p.exhibit_id=a.exhibit_id) OR NOT EXISTS(SELECT 1 FROM court_rulings n WHERE n.id=a.new_ruling_id AND n.appeal_id=a.id) OR NOT EXISTS(SELECT 1 FROM journal_events j WHERE j.syscall_id=a.syscall_id) OR NOT EXISTS(SELECT 1 FROM forge_k_audit_outbox o WHERE o.syscall_id=a.syscall_id)`},
	}
	for _, check := range checks {
		var broken int
		if err := db.QueryRowContext(ctx, check.query).Scan(&broken); err != nil {
			return err
		}
		if broken != 0 {
			return fmt.Errorf("%s proof identity verification failed for %d rows", check.name, broken)
		}
	}
	for table, target := range map[string]*int{"court_exhibits": &counts.exhibits, "court_rulings": &counts.rulings, "court_appeals": &counts.appeals} {
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+quoteIdentifier(table)).Scan(target); err != nil {
			return err
		}
	}
	return nil
}

func verifySQLiteIntegrity(ctx context.Context, db *sql.DB) error {
	var status string
	if err := db.QueryRowContext(ctx, `PRAGMA integrity_check`).Scan(&status); err != nil {
		return err
	}
	if status != "ok" {
		return fmt.Errorf("SQLite integrity_check failed: %s", status)
	}
	return verifyForeignKeys(ctx, db)
}

type queryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func verifyForeignKeys(ctx context.Context, q queryer) error {
	rows, err := q.QueryContext(ctx, `PRAGMA foreign_key_check`)
	if err != nil {
		return err
	}
	defer rows.Close()
	if rows.Next() {
		return fmt.Errorf("SQLite foreign_key_check failed")
	}
	return rows.Err()
}

func tableColumns(ctx context.Context, tx *sql.Tx, table string) (map[string]struct{}, error) {
	rows, err := tx.QueryContext(ctx, "PRAGMA table_info("+quoteIdentifier(table)+")")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	columns := map[string]struct{}{}
	for rows.Next() {
		var cid, notNull, pk int
		var name, typ string
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); err != nil {
			return nil, err
		}
		columns[name] = struct{}{}
	}
	if len(columns) == 0 {
		return nil, fmt.Errorf("current schema table %q is missing", table)
	}
	return columns, rows.Err()
}

func recordMap(value any) (map[string]any, error) {
	record, ok := value.(map[string]any)
	if !ok || len(record) == 0 {
		return nil, fmt.Errorf("bundle row is not a non-empty object")
	}
	return record, nil
}

func sqliteValue(value any) (any, error) {
	number, ok := value.(json.Number)
	if !ok {
		return value, nil
	}
	if integer, err := number.Int64(); err == nil {
		return integer, nil
	}
	floating, err := number.Float64()
	if err != nil {
		return nil, fmt.Errorf("invalid JSON number %q", number)
	}
	return floating, nil
}

func sameJSON(left, right string) bool {
	var a, b any
	return json.Unmarshal([]byte(left), &a) == nil && json.Unmarshal([]byte(right), &b) == nil && reflect.DeepEqual(a, b)
}

func decodeProofJSON(raw string, target any) error {
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("trailing proof JSON")
		}
		return err
	}
	return nil
}

func hydrateKernelMetadata(req *domain.SyscallRequest) error {
	if req == nil || req.Metadata == nil {
		return nil
	}
	for _, binding := range []struct {
		key    string
		target func() any
	}{
		{court.MetadataDecisionKey, func() any { return new(court.Decision) }},
		{semanticdiff.MetadataDecisionKey, func() any { return new(semanticdiff.Decision) }},
		{contextcompile.MetadataInputKey, func() any { return new(contextcompile.Input) }},
		{contextcompile.MetadataDecisionKey, func() any { return new(contextcompile.Decision) }},
	} {
		value, ok := req.Metadata[binding.key]
		if !ok {
			continue
		}
		raw, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("%s: %w", binding.key, err)
		}
		target := binding.target()
		if err := json.Unmarshal(raw, target); err != nil {
			return fmt.Errorf("%s: %w", binding.key, err)
		}
		switch typed := target.(type) {
		case *court.Decision:
			req.Metadata[binding.key] = *typed
		case *semanticdiff.Decision:
			req.Metadata[binding.key] = *typed
		case *contextcompile.Input:
			req.Metadata[binding.key] = *typed
		case *contextcompile.Decision:
			req.Metadata[binding.key] = *typed
		}
	}
	return nil
}

func recoveryPriority(section string) int {
	priorities := map[string]int{
		"sources": 1, "files": 2, "chunks": 3, "embedding_records": 4,
		"provenance_records": 10, "journal_events": 11, "court_exhibits": 20,
		"court_rulings": 21, "court_appeals": 22, "forge_k_memory_evidence": 23,
		"forge_k_memory_evidence_supersessions": 24, "forge_k_semantic_diff_operations": 25,
		"forge_k_semantic_diff_results": 26, "forge_k_semantic_derived_objects": 27,
		"forge_k_context_bundles": 28, "forge_k_context_snapshot_heads": 29,
		"semantic_idempotency_keys": 90, "forge_k_journal_head": 91, "forge_k_audit_outbox": 92,
		"forge_k_audit_delivery_attempts": 93,
	}
	if priority, ok := priorities[section]; ok {
		return priority
	}
	return 50
}

func quoteIdentifier(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

func joinQuoted(values []string) string {
	out := make([]string, len(values))
	for i, value := range values {
		out[i] = quoteIdentifier(value)
	}
	return strings.Join(out, ",")
}

func checkpointStoppedStore(ctx context.Context, path string) error {
	db, err := openExistingSQLite(path)
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`)
	return err
}

func openExistingSQLite(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=rw&_pragma=foreign_keys(1)", path))
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	db.SetMaxOpenConns(1)
	return db, nil
}

func preservePriorStore(dataDir, source string) (string, error) {
	dir := filepath.Join(dataDir, "recovery-prior")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	target := filepath.Join(dir, fmt.Sprintf("forge-before-recovery-%d.sqlite", time.Now().UnixNano()))
	if err := copyFile(source, target, 0o600); err != nil {
		return "", err
	}
	return target, syncDir(dir)
}

func rollbackStore(target, prior string) error {
	dir := filepath.Dir(target)
	temp, err := os.CreateTemp(dir, ".forge-recovery-rollback-")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	if err := temp.Close(); err != nil {
		return err
	}
	defer os.Remove(tempPath)
	if err := os.Remove(tempPath); err != nil {
		return err
	}
	if err := copyFile(prior, tempPath, 0o600); err != nil {
		return err
	}
	if err := prepareReplacementMetadata(prior, tempPath); err != nil {
		return err
	}
	if err := os.Rename(tempPath, target); err != nil {
		return err
	}
	return syncDir(dir)
}

func copyFile(source, target string, mode os.FileMode) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	remove := true
	defer func() {
		_ = out.Close()
		if remove {
			_ = os.Remove(target)
		}
	}()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	if err := out.Sync(); err != nil {
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	remove = false
	return nil
}

func syncFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return f.Sync()
}

func syncDir(path string) error {
	d, err := os.Open(path)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}
