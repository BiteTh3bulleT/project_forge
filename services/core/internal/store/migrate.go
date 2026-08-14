package store

import (
	"database/sql"
	"fmt"
)

func migrate(db *sql.DB) error {
	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	if err := ensureForgeKJournalChain(db); err != nil {
		return fmt.Errorf("migrate FORGE-K journal chain: %w", err)
	}
	if err := ensureMemoryVSAProjectionAuthority(db); err != nil {
		return fmt.Errorf("migrate memory VSA projection authority: %w", err)
	}
	if err := ensureContextPacketSnapshotColumns(db); err != nil {
		return fmt.Errorf("migrate context_packet_snapshots: %w", err)
	}
	if err := ensureApprovalRequestExpiryColumns(db); err != nil {
		return fmt.Errorf("migrate approval_requests expiry: %w", err)
	}
	if err := ensureToolCapabilityOverrideColumns(db); err != nil {
		return fmt.Errorf("migrate tool_capability_overrides: %w", err)
	}
	return nil
}
