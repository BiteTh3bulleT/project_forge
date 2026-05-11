package store

import (
	"database/sql"
	"fmt"
)

func ensureToolCapabilityOverrideColumns(db *sql.DB) error {
	rows, err := db.Query(`PRAGMA table_info(tool_capability_overrides)`)
	if err != nil {
		return err
	}
	defer rows.Close()
	existing := make(map[string]struct{})
	for rows.Next() {
		var cid int
		var name string
		var ctype string
		var notNull int
		var dflt any
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notNull, &dflt, &pk); err != nil {
			return err
		}
		existing[name] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	add := []struct {
		name string
		ddl  string
	}{
		{name: "actor_kind", ddl: "TEXT NOT NULL DEFAULT ''"},
		{name: "previous_status", ddl: "TEXT NOT NULL DEFAULT ''"},
		{name: "risk_class", ddl: "TEXT NOT NULL DEFAULT ''"},
		{name: "transition_risk", ddl: "TEXT NOT NULL DEFAULT ''"},
		{name: "approval_request_id", ddl: "INTEGER REFERENCES approval_requests(id) ON DELETE SET NULL"},
		{name: "correlation_id", ddl: "TEXT NOT NULL DEFAULT ''"},
		{name: "trace_id", ddl: "TEXT NOT NULL DEFAULT ''"},
	}
	for _, col := range add {
		if _, ok := existing[col.name]; ok {
			continue
		}
		if _, err := db.Exec(fmt.Sprintf("ALTER TABLE tool_capability_overrides ADD COLUMN %s %s", col.name, col.ddl)); err != nil {
			return err
		}
	}
	return nil
}

// ensureApprovalRequestExpiryColumns adds expires_at and auto-expired metadata
// to approval_requests for existing DBs so P0 safety fixes apply without a wipe.
func ensureApprovalRequestExpiryColumns(db *sql.DB) error {
	rows, err := db.Query(`PRAGMA table_info(approval_requests)`)
	if err != nil {
		return err
	}
	defer rows.Close()
	existing := make(map[string]struct{})
	for rows.Next() {
		var cid int
		var name string
		var ctype string
		var notnull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return err
		}
		existing[name] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	additions := []struct{ name, ddl string }{
		{name: "expires_at", ddl: "INTEGER NOT NULL DEFAULT 0"},
		{name: "expired_at", ddl: "INTEGER NOT NULL DEFAULT 0"},
	}
	for _, col := range additions {
		if _, ok := existing[col.name]; ok {
			continue
		}
		if _, err := db.Exec(fmt.Sprintf(
			"ALTER TABLE approval_requests ADD COLUMN %s %s",
			col.name, col.ddl,
		)); err != nil {
			return err
		}
	}
	// Backfill expires_at for legacy pending rows so the reaper has something to
	// act on; default TTL is 24 hours from row creation.
	if _, err := db.Exec(`
UPDATE approval_requests
SET expires_at = created_at + (24 * 3600 * 1000)
WHERE expires_at = 0`); err != nil {
		return err
	}
	return nil
}

func ensureContextPacketSnapshotColumns(db *sql.DB) error {
	rows, err := db.Query(`PRAGMA table_info(context_packet_snapshots)`)
	if err != nil {
		return err
	}
	defer rows.Close()

	existing := make(map[string]struct{})
	for rows.Next() {
		var cid, notNull, pk int
		var name, colType string
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &colType, &notNull, &defaultValue, &pk); err != nil {
			return err
		}
		existing[name] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	additions := []struct {
		name string
		ddl  string
	}{
		{name: "snapshot_kind", ddl: "TEXT NOT NULL DEFAULT ''"},
		{name: "snapshot_fingerprint", ddl: "TEXT NOT NULL DEFAULT ''"},
		{name: "parent_snapshot_id", ddl: "TEXT NOT NULL DEFAULT ''"},
		{name: "header_json", ddl: "TEXT NOT NULL DEFAULT '{}'"},
		{name: "graph_json", ddl: "TEXT NOT NULL DEFAULT '{}'"},
		{name: "delta_json", ddl: "TEXT NOT NULL DEFAULT '{}'"},
		{name: "restore_scores_json", ddl: "TEXT NOT NULL DEFAULT '{}'"},
		{name: "render_artifact_ref_id", ddl: "TEXT NOT NULL DEFAULT ''"},
		{name: "resume_hints_json", ddl: "TEXT NOT NULL DEFAULT '{}'"},
	}

	for _, col := range additions {
		if _, ok := existing[col.name]; ok {
			continue
		}
		if _, err := db.Exec(fmt.Sprintf(
			"ALTER TABLE context_packet_snapshots ADD COLUMN %s %s",
			col.name, col.ddl,
		)); err != nil {
			return err
		}
	}

	_, err = db.Exec(`
UPDATE context_packet_snapshots
SET snapshot_kind = COALESCE(snapshot_kind, ''),
    snapshot_fingerprint = COALESCE(snapshot_fingerprint, ''),
    parent_snapshot_id = COALESCE(parent_snapshot_id, ''),
    header_json = COALESCE(header_json, '{}'),
    graph_json = COALESCE(graph_json, '{}'),
    delta_json = COALESCE(delta_json, '{}'),
    restore_scores_json = COALESCE(restore_scores_json, '{}'),
    render_artifact_ref_id = COALESCE(render_artifact_ref_id, ''),
    resume_hints_json = COALESCE(resume_hints_json, '{}')`)
	return err
}
