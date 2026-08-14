package controllane

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// SQLite Transaction runner.
type SQLiteTransactionRunner struct {
	db   *sql.DB
	read *SQLiteSemanticStore
}

func NewSQLiteTransactionRunner(db *sql.DB) *SQLiteTransactionRunner {
	return &SQLiteTransactionRunner{
		db:   db,
		read: NewSQLiteSemanticStore(db),
	}
}

func (r *SQLiteTransactionRunner) ReadStore() SemanticReadStore {
	return r.read
}

func (r *SQLiteTransactionRunner) Run(ctx context.Context, fn func(uow UnitOfWork) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	txStore := newSQLiteSemanticStore(tx)
	uow := &txUnitOfWork{store: txStore}
	if err := fn(uow); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *SQLiteTransactionRunner) LinkAudit(ctx context.Context, correlationID, syscallID, auditID string) error {
	return linkAuditOnExecutor(ctx, r.db, correlationID, syscallID, auditID)
}

func linkAuditOnExecutor(ctx context.Context, exec sqlExecutor, correlationID, syscallID, auditID string) error {
	if strings.TrimSpace(auditID) == "" || strings.TrimSpace(syscallID) == "" {
		return nil
	}
	tables := []string{
		"provenance_records",
		"memory_notes",
		"semantic_links",
		"state_items",
		"state_versions",
		"open_loops",
		"artifact_refs",
		"derived_models",
		"contradiction_records",
		"supersession_records",
		"court_exhibits",
		"court_rulings",
		"court_appeals",
		"context_packet_snapshots",
		"restore_outcome_events",
	}
	for _, tbl := range tables {
		query := fmt.Sprintf(`UPDATE %s SET audit_id = ? WHERE audit_id = '' AND syscall_id = ? AND correlation_id = ?`, tbl)
		if _, err := exec.ExecContext(ctx, query, auditID, syscallID, correlationID); err != nil {
			return err
		}
	}
	return nil
}
