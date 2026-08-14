package store

import (
	"database/sql"
	"strings"
	"testing"

	forgejournal "forge/projectforge/services/core/internal/forgekernel/journal"
	_ "modernc.org/sqlite"
)

func TestMigrateBackfillsLegacyJournalDeterministically(t *testing.T) {
	db := openLegacyJournalDB(t)
	defer db.Close()
	insertLegacyJournalFixture(t, db)
	if err := migrate(db); err != nil {
		t.Fatalf("migrate legacy database: %v", err)
	}

	rows, err := db.Query(`
SELECT id,journal_schema_version,chain_sequence,prior_hash,event_hash
FROM journal_events ORDER BY chain_sequence ASC`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	type chained struct {
		id, version, prior, hash string
		sequence                 uint64
	}
	got := []chained{}
	for rows.Next() {
		var row chained
		if err := rows.Scan(&row.id, &row.version, &row.sequence, &row.prior, &row.hash); err != nil {
			t.Fatal(err)
		}
		got = append(got, row)
	}
	if len(got) != 2 || got[0].id != "evt-a" || got[1].id != "evt-b" {
		t.Fatalf("legacy order was not frozen by created_at then id: %#v", got)
	}
	if got[0].version != forgejournal.SchemaVersion || got[0].sequence != 1 || got[0].prior != "" {
		t.Fatalf("unexpected genesis backfill: %#v", got[0])
	}
	if got[1].sequence != 2 || got[1].prior != got[0].hash {
		t.Fatalf("backfilled chain is not linked: %#v", got)
	}
	var head forgejournal.Head
	if err := db.QueryRow(`SELECT sequence,event_id,head_hash FROM forge_k_journal_head WHERE id=1`).Scan(&head.Sequence, &head.EventID, &head.Hash); err != nil {
		t.Fatal(err)
	}
	if head.Sequence != 2 || head.EventID != "evt-b" || head.Hash != got[1].hash {
		t.Fatalf("unexpected persisted head: %#v", head)
	}

	// Re-running migration verifies rather than rewriting the existing chain.
	if err := migrate(db); err != nil {
		t.Fatalf("repeat migration: %v", err)
	}
	var repeatedHash string
	if err := db.QueryRow(`SELECT event_hash FROM journal_events WHERE id='evt-b'`).Scan(&repeatedHash); err != nil {
		t.Fatal(err)
	}
	if repeatedHash != got[1].hash {
		t.Fatalf("repeat migration rewrote chain hash: %q != %q", repeatedHash, got[1].hash)
	}
}

func TestMigrateFailsClosedOnChainedPayloadTamper(t *testing.T) {
	db := openLegacyJournalDB(t)
	defer db.Close()
	insertLegacyJournalFixture(t, db)
	if err := migrate(db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DROP TRIGGER journal_events_no_update`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE journal_events SET payload_json='{"tampered":true}' WHERE id='evt-a'`); err != nil {
		t.Fatal(err)
	}
	if err := migrate(db); err == nil || !strings.Contains(err.Error(), "payload hash") {
		t.Fatalf("expected payload integrity migration failure, got %v", err)
	}
}

func TestMigrateAddsImmutableCommitProofTablesAndLegacyMarkers(t *testing.T) {
	db := openLegacyJournalDB(t)
	defer db.Close()
	if _, err := db.Exec(`
INSERT INTO semantic_idempotency_keys(idempotency_key,action,result_json,created_at,correlation_id)
VALUES('key-1','CREATE_NOTE','{}',100,'corr-1')`); err != nil {
		t.Fatal(err)
	}
	if err := migrate(db); err != nil {
		t.Fatal(err)
	}
	var requestFingerprint, idempotencyFingerprint, requestJSON, planJSON, sealJSON, receiptJSON, authproofJSON string
	if err := db.QueryRow(`
SELECT request_fingerprint,idempotency_fingerprint,request_json,plan_json,seal_json,receipt_json,authproof_json
FROM semantic_idempotency_keys WHERE idempotency_key='key-1'`).Scan(
		&requestFingerprint, &idempotencyFingerprint, &requestJSON, &planJSON, &sealJSON, &receiptJSON, &authproofJSON,
	); err != nil {
		t.Fatal(err)
	}
	if requestFingerprint != legacyUnboundRequestFingerprint || idempotencyFingerprint != legacyUnboundRequestFingerprint {
		t.Fatalf("legacy row was not marked non-replayable: request=%q idempotency=%q", requestFingerprint, idempotencyFingerprint)
	}
	if requestJSON != "{}" || planJSON != "{}" || sealJSON != "{}" || receiptJSON != "{}" || authproofJSON != "{}" {
		t.Fatalf("legacy replay proof defaults must fail closed: request=%q plan=%q seal=%q receipt=%q auth=%q", requestJSON, planJSON, sealJSON, receiptJSON, authproofJSON)
	}
	if _, err := db.Exec(`UPDATE semantic_idempotency_keys SET action='DELETE_FILE' WHERE idempotency_key='key-1'`); err == nil {
		t.Fatal("expected idempotency update to be rejected")
	}
	if _, err := db.Exec(`DELETE FROM semantic_idempotency_keys WHERE idempotency_key='key-1'`); err == nil {
		t.Fatal("expected idempotency delete to be rejected")
	}

	if _, err := db.Exec(`
INSERT INTO forge_k_audit_outbox(
 id,syscall_id,request_fingerprint,action,workspace_id,lane_id,correlation_id,
 trace_id,success,result_json,created_at,committed_by
) VALUES('out-1','sys-1','sha256:a','CREATE_NOTE','ws','control','corr','trace',1,'{}',200,'forge_k.kernel')`); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT request_json,receipt_json,authproof_json FROM forge_k_audit_outbox WHERE id='out-1'`).Scan(&requestJSON, &receiptJSON, &authproofJSON); err != nil || requestJSON != "{}" || receiptJSON != "{}" || authproofJSON != "{}" {
		t.Fatalf("audit outbox proof defaults: request=%q receipt=%q auth=%q err=%v", requestJSON, receiptJSON, authproofJSON, err)
	}
	if _, err := db.Exec(`UPDATE forge_k_audit_outbox SET success=0 WHERE id='out-1'`); err == nil {
		t.Fatal("expected audit outbox update to be rejected")
	}
	if _, err := db.Exec(`DELETE FROM forge_k_audit_outbox WHERE id='out-1'`); err == nil {
		t.Fatal("expected audit outbox delete to be rejected")
	}
}

func openLegacyJournalDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+t.TempDir()+"/legacy.sqlite?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`
CREATE TABLE provenance_records (
 id TEXT PRIMARY KEY, actor TEXT NOT NULL, actor_type TEXT NOT NULL,
 source TEXT NOT NULL DEFAULT '', trace_id TEXT NOT NULL DEFAULT '',
 workspace_id TEXT NOT NULL, lane_id TEXT NOT NULL DEFAULT '',
 selected_paths_json TEXT NOT NULL DEFAULT '[]', metadata_json TEXT NOT NULL DEFAULT '{}',
 created_at INTEGER NOT NULL, proposed_by TEXT NOT NULL DEFAULT '',
 committed_by TEXT NOT NULL DEFAULT 'forge_kernel', syscall_id TEXT NOT NULL DEFAULT '',
 correlation_id TEXT NOT NULL DEFAULT '', audit_id TEXT NOT NULL DEFAULT ''
);
CREATE TABLE journal_events (
 id TEXT PRIMARY KEY, type TEXT NOT NULL, source TEXT NOT NULL,
 actor TEXT NOT NULL DEFAULT '', workspace_id TEXT NOT NULL,
 lane_id TEXT NOT NULL DEFAULT '', selected_paths_json TEXT NOT NULL DEFAULT '[]',
 payload_json TEXT NOT NULL DEFAULT '{}', correlation_id TEXT NOT NULL DEFAULT '',
 trace_id TEXT NOT NULL DEFAULT '', provenance_id TEXT REFERENCES provenance_records(id) ON DELETE SET NULL,
 provenance_json TEXT NOT NULL DEFAULT '{}', created_at INTEGER NOT NULL,
 metadata_json TEXT NOT NULL DEFAULT '{}', proposed_by TEXT NOT NULL DEFAULT '',
 committed_by TEXT NOT NULL DEFAULT 'forge_kernel', syscall_id TEXT NOT NULL DEFAULT '',
 audit_id TEXT NOT NULL DEFAULT ''
);
CREATE TABLE semantic_idempotency_keys (
 idempotency_key TEXT PRIMARY KEY, action TEXT NOT NULL, result_json TEXT NOT NULL,
 created_at INTEGER NOT NULL DEFAULT 0, correlation_id TEXT NOT NULL DEFAULT ''
);`); err != nil {
		db.Close()
		t.Fatal(err)
	}
	return db
}

func insertLegacyJournalFixture(t *testing.T, db *sql.DB) {
	t.Helper()
	// Insert out of ID order to prove migration uses persisted created_at then id.
	for _, values := range [][]any{
		{"evt-b", "semantic_syscall.create_link", int64(200), `{"action":"CREATE_LINK"}`},
		{"evt-a", "semantic_syscall.create_note", int64(100), ` {"action": "CREATE_NOTE"} `},
	} {
		if _, err := db.Exec(`
INSERT INTO journal_events(
 id,type,source,actor,workspace_id,lane_id,selected_paths_json,payload_json,
 correlation_id,trace_id,provenance_json,created_at,metadata_json,committed_by
) VALUES(?,?, 'forge_kernel','operator','ws-main','control.semantic','["/notes"]',?,
 'corr-1','trace-1','{"actor":"operator"}',?,'{}','forge_kernel')`,
			values[0], values[1], values[3], values[2],
		); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`
CREATE TRIGGER journal_events_no_update BEFORE UPDATE ON journal_events
BEGIN SELECT RAISE(FAIL, 'journal_events are append-only'); END;
CREATE TRIGGER journal_events_no_delete BEFORE DELETE ON journal_events
BEGIN SELECT RAISE(FAIL, 'journal_events are append-only'); END;`); err != nil {
		t.Fatal(err)
	}
}
