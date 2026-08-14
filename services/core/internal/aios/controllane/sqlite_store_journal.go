package controllane

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"forge/projectforge/services/core/internal/aios/domain"
	forgejournal "forge/projectforge/services/core/internal/forgekernel/journal"
)

var (
	ErrJournalHeadConflict        = errors.New("FORGE-K journal head compare-and-swap conflict")
	ErrJournalTransactionRequired = errors.New("FORGE-K journal append requires a SQLite transaction")
	ErrJournalContentHashMismatch = errors.New("FORGE-K journal content hash mismatch")
)

type JournalAppendEvidence struct {
	PreviousHead forgejournal.Head  `json:"previousHead"`
	Entry        forgejournal.Entry `json:"entry"`
	Head         forgejournal.Head  `json:"head"`
}

// JournalRepository
func (s *SQLiteSemanticStore) Append(ctx context.Context, evt domain.JournalEvent) error {
	_, err := s.AppendWithEvidence(ctx, evt)
	return err
}

// AppendWithEvidence atomically appends a journal row and advances the
// independently persisted head. Direct DB callers are wrapped in a transaction;
// kernel commits already arrive on the caller's semantic mutation transaction.
func (s *SQLiteSemanticStore) AppendWithEvidence(ctx context.Context, evt domain.JournalEvent) (JournalAppendEvidence, error) {
	switch exec := s.exec.(type) {
	case *sql.DB:
		tx, err := exec.BeginTx(ctx, nil)
		if err != nil {
			return JournalAppendEvidence{}, err
		}
		defer func() { _ = tx.Rollback() }()
		child := newSQLiteSemanticStore(tx)
		child.meta = s.meta
		evidence, err := child.appendWithEvidence(ctx, evt)
		if err != nil {
			return JournalAppendEvidence{}, err
		}
		if err := tx.Commit(); err != nil {
			return JournalAppendEvidence{}, err
		}
		return evidence, nil
	case *sql.Tx:
		return s.appendWithEvidence(ctx, evt)
	default:
		return JournalAppendEvidence{}, ErrJournalTransactionRequired
	}
}

func (s *SQLiteSemanticStore) appendWithEvidence(ctx context.Context, evt domain.JournalEvent) (JournalAppendEvidence, error) {
	scope := toScopeFilter(evt.Scope)
	provID, err := s.ensureProvenance(ctx, evt.Scope, evt.Provenance, nil, evt.Timestamp)
	if err != nil {
		return JournalAppendEvidence{}, err
	}
	selectedJSON := encodeStringSlice(evt.Scope.SelectedPaths)
	payloadJSON := encodeJSON(evt.Payload)
	provenanceJSON := encodeJSON(evt.Provenance)
	metadataJSON := encodeJSON(map[string]any{})
	payloadHash, err := forgejournal.HashJSON([]byte(payloadJSON))
	if err != nil {
		return JournalAppendEvidence{}, err
	}
	provenanceHash, err := forgejournal.HashJSON([]byte(provenanceJSON))
	if err != nil {
		return JournalAppendEvidence{}, err
	}
	metadataHash, err := forgejournal.HashJSON([]byte(metadataJSON))
	if err != nil {
		return JournalAppendEvidence{}, err
	}
	previousHead, err := s.JournalChainHead(ctx)
	if err != nil {
		return JournalAppendEvidence{}, err
	}
	traceID := nonEmpty(evt.Provenance.TraceID, s.meta.TraceID)
	syscallID := nonEmpty(s.meta.SyscallID, "legacy:"+evt.ID)
	committedBy := nonEmpty(s.meta.CommittedBy, "forge_kernel")
	planned, err := forgejournal.PlanAppend(previousHead, forgejournal.AppendInput{
		EventID: evt.ID, EventType: evt.Type, Source: evt.Source, Actor: evt.Provenance.Actor,
		WorkspaceID: scope.WorkspaceID, LaneID: scope.LaneID,
		SelectedPaths: append([]string{}, evt.Scope.SelectedPaths...),
		CorrelationID: evt.CorrelationID, TraceID: traceID, ProvenanceID: provID,
		ProvenanceHash: provenanceHash, PayloadHash: payloadHash, MetadataHash: metadataHash,
		ProposedBy: string(s.meta.Source), CommittedBy: committedBy, SyscallID: syscallID,
		CreatedAt: evt.Timestamp,
	})
	if err != nil {
		return JournalAppendEvidence{}, err
	}
	_, err = s.exec.ExecContext(ctx, `
INSERT INTO journal_events(
  id, type, source, actor, workspace_id, lane_id, selected_paths_json, payload_json,
  correlation_id, trace_id, provenance_id, provenance_json, created_at, metadata_json,
	  proposed_by, committed_by, syscall_id, audit_id, journal_schema_version,
	  chain_sequence, payload_hash, provenance_hash, journal_metadata_hash,
	  prior_hash, event_hash
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		evt.ID, evt.Type, evt.Source, evt.Provenance.Actor, scope.WorkspaceID, scope.LaneID, selectedJSON, payloadJSON,
		evt.CorrelationID, traceID, provID, provenanceJSON, evt.Timestamp, metadataJSON,
		string(s.meta.Source), committedBy, syscallID, "", planned.SchemaVersion,
		planned.Sequence, planned.PayloadHash, planned.ProvenanceHash, planned.MetadataHash,
		planned.PriorHash, planned.Hash,
	)
	if err != nil {
		return JournalAppendEvidence{}, err
	}
	nextHead := forgejournal.Head{Sequence: planned.Sequence, EventID: planned.EventID, Hash: planned.Hash}
	result, err := s.exec.ExecContext(ctx, `
UPDATE forge_k_journal_head
SET sequence=?,event_id=?,head_hash=?,updated_at=?
WHERE id=1 AND sequence=? AND event_id=? AND head_hash=?`,
		nextHead.Sequence, nextHead.EventID, nextHead.Hash, evt.Timestamp,
		previousHead.Sequence, previousHead.EventID, previousHead.Hash,
	)
	if err != nil {
		return JournalAppendEvidence{}, err
	}
	if affected, err := result.RowsAffected(); err != nil || affected != 1 {
		return JournalAppendEvidence{}, ErrJournalHeadConflict
	}
	return JournalAppendEvidence{PreviousHead: previousHead, Entry: planned, Head: nextHead}, nil
}

func (s *SQLiteSemanticStore) JournalChainHead(ctx context.Context) (forgejournal.Head, error) {
	var head forgejournal.Head
	if err := s.exec.QueryRowContext(ctx, `
SELECT sequence,event_id,head_hash FROM forge_k_journal_head WHERE id=1`).Scan(
		&head.Sequence, &head.EventID, &head.Hash,
	); err != nil {
		return forgejournal.Head{}, err
	}
	return head, nil
}

func (s *SQLiteSemanticStore) VerifyJournalChain(ctx context.Context) (forgejournal.VerificationReport, error) {
	entries, err := s.loadJournalChain(ctx)
	if err != nil {
		return forgejournal.VerificationReport{}, err
	}
	head, err := s.JournalChainHead(ctx)
	if err != nil {
		return forgejournal.VerificationReport{}, err
	}
	return forgejournal.Verify(entries, &head), nil
}

func (s *SQLiteSemanticStore) loadJournalChain(ctx context.Context) ([]forgejournal.Entry, error) {
	rows, err := s.exec.QueryContext(ctx, `
SELECT id,type,source,actor,workspace_id,lane_id,selected_paths_json,
       correlation_id,trace_id,COALESCE(provenance_id,''),provenance_json,
       payload_json,metadata_json,proposed_by,committed_by,syscall_id,audit_id,
       created_at,journal_schema_version,chain_sequence,payload_hash,
       provenance_hash,journal_metadata_hash,prior_hash,event_hash
FROM journal_events
ORDER BY chain_sequence ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := []forgejournal.Entry{}
	for rows.Next() {
		var entry forgejournal.Entry
		var selectedRaw, provenanceRaw, payloadRaw, metadataRaw string
		if err := rows.Scan(
			&entry.EventID, &entry.EventType, &entry.Source, &entry.Actor,
			&entry.WorkspaceID, &entry.LaneID, &selectedRaw, &entry.CorrelationID,
			&entry.TraceID, &entry.ProvenanceID, &provenanceRaw, &payloadRaw,
			&metadataRaw, &entry.ProposedBy, &entry.CommittedBy, &entry.SyscallID,
			&entry.AuditID, &entry.CreatedAt, &entry.SchemaVersion, &entry.Sequence,
			&entry.PayloadHash, &entry.ProvenanceHash, &entry.MetadataHash,
			&entry.PriorHash, &entry.Hash,
		); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(selectedRaw), &entry.SelectedPaths); err != nil {
			return nil, err
		}
		contentFields := []struct{ field, raw, stored string }{
			{"payload", payloadRaw, entry.PayloadHash},
			{"provenance", provenanceRaw, entry.ProvenanceHash},
			{"metadata", metadataRaw, entry.MetadataHash},
		}
		for _, candidate := range contentFields {
			actual, err := forgejournal.HashJSON([]byte(candidate.raw))
			if err != nil {
				return nil, fmt.Errorf("journal event %q %s JSON: %w", entry.EventID, candidate.field, err)
			}
			if actual != candidate.stored {
				return nil, fmt.Errorf("%w: event %q field %s", ErrJournalContentHashMismatch, entry.EventID, candidate.field)
			}
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

func (s *SQLiteSemanticStore) GetByID(ctx context.Context, id string) (domain.JournalEvent, bool, error) {
	row := s.exec.QueryRowContext(ctx, `
SELECT id, type, source, actor, workspace_id, lane_id, selected_paths_json, payload_json, correlation_id, trace_id, provenance_json, created_at
FROM journal_events WHERE id = ?`, id)
	var evt domain.JournalEvent
	var payloadRaw, provRaw, selected, actor, trace string
	if err := row.Scan(&evt.ID, &evt.Type, &evt.Source, &actor, &evt.Scope.WorkspaceID, &evt.Scope.LaneID, &selected, &payloadRaw, &evt.CorrelationID, &trace, &provRaw, &evt.Timestamp); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.JournalEvent{}, false, nil
		}
		return domain.JournalEvent{}, false, err
	}
	evt.Scope.SelectedPaths = decodeStringSlice(selected)
	_ = json.Unmarshal([]byte(payloadRaw), &evt.Payload)
	_ = json.Unmarshal([]byte(provRaw), &evt.Provenance)
	if evt.Provenance.Actor == "" {
		evt.Provenance.Actor = actor
	}
	if evt.Provenance.TraceID == "" {
		evt.Provenance.TraceID = trace
	}
	return evt, true, nil
}

func (s *SQLiteSemanticStore) ListByScope(ctx context.Context, scope ScopeFilter, limit int) ([]domain.JournalEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.exec.QueryContext(ctx, `
SELECT id, type, source, actor, workspace_id, lane_id, selected_paths_json, payload_json, correlation_id, trace_id, provenance_json, created_at
FROM journal_events
WHERE workspace_id = ? AND (? = '' OR lane_id = ?)
ORDER BY created_at DESC
LIMIT ?`, scope.WorkspaceID, scope.LaneID, scope.LaneID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanJournalRows(rows)
}

func (s *SQLiteSemanticStore) ListByCorrelation(ctx context.Context, correlationID string, limit int) ([]domain.JournalEvent, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.exec.QueryContext(ctx, `
SELECT id, type, source, actor, workspace_id, lane_id, selected_paths_json, payload_json, correlation_id, trace_id, provenance_json, created_at
FROM journal_events
WHERE correlation_id = ?
ORDER BY created_at DESC
LIMIT ?`, correlationID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanJournalRows(rows)
}

func (s *SQLiteSemanticStore) ListRecent(ctx context.Context, filter RecentFilter) ([]domain.JournalEvent, error) {
	if filter.Scope.WorkspaceID == "" {
		return nil, nil
	}
	return s.ListByScope(ctx, filter.Scope, filter.Limit)
}
