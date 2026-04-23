package memory

import (
	"context"
	"database/sql"
)

func (s *Service) DossierView(ctx context.Context, dossierID int64, limit int) (*DossierMemoryView, error) {
	if dossierID <= 0 {
		return &DossierMemoryView{DossierID: dossierID}, nil
	}
	if limit <= 0 || limit > 120 {
		limit = 40
	}
	view := &DossierMemoryView{DossierID: dossierID}
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM memory_observations WHERE dossier_id = ?`, dossierID).Scan(&view.ObservationCount)
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM memory_observations WHERE dossier_id = ? AND stale = 1`, dossierID).Scan(&view.StaleObservationCount)
	obs, err := s.ListObservations(ctx, ListObservationsRequest{DossierID: &dossierID, Limit: limit})
	if err != nil {
		return nil, err
	}
	view.RecentObservations = obs

	signalRows, err := s.db.QueryContext(ctx, `
SELECT m.id, m.created_at, m.observation_id, m.retrieval_result_id, m.retrieval_run_id, m.packet_id, m.job_id, m.signal, m.weight, m.note
FROM memory_usefulness_events m
JOIN memory_observations o ON o.id = m.observation_id
WHERE o.dossier_id = ?
ORDER BY m.id DESC
LIMIT ?`, dossierID, limit)
	if err != nil {
		return nil, err
	}
	defer signalRows.Close()
	signals := []UsefulnessEvent{}
	for signalRows.Next() {
		var item UsefulnessEvent
		var resultID, runID, packetID sql.NullInt64
		var jobID sql.NullString
		if err := signalRows.Scan(
			&item.ID,
			&item.CreatedAtMs,
			&item.ObservationID,
			&resultID,
			&runID,
			&packetID,
			&jobID,
			&item.Signal,
			&item.Weight,
			&item.Note,
		); err != nil {
			return nil, err
		}
		item.RetrievalResultID = scanNullableInt64(resultID)
		item.RetrievalRunID = scanNullableInt64(runID)
		item.PacketID = scanNullableInt64(packetID)
		item.JobID = scanNullableString(jobID)
		signals = append(signals, item)
	}
	if err := signalRows.Err(); err != nil {
		return nil, err
	}
	view.RecentSignals = signals

	alignRows, err := s.db.QueryContext(ctx, `
SELECT pan.id, pan.packet_id, pan.observation_id, pan.retrieval_result_id, pan.note, pan.created_at
FROM packet_alignment_notes pan
LEFT JOIN memory_observations mo ON mo.id = pan.observation_id
LEFT JOIN retrieval_results rr ON rr.id = pan.retrieval_result_id
LEFT JOIN retrieval_runs run ON run.id = rr.retrieval_run_id
WHERE mo.dossier_id = ? OR run.dossier_id = ?
ORDER BY pan.id DESC
LIMIT ?`, dossierID, dossierID, limit)
	if err != nil {
		return nil, err
	}
	defer alignRows.Close()
	alignments := []PacketAlignmentNote{}
	for alignRows.Next() {
		var item PacketAlignmentNote
		var obsID, resultID sql.NullInt64
		if err := alignRows.Scan(&item.ID, &item.PacketID, &obsID, &resultID, &item.Note, &item.CreatedAtMs); err != nil {
			return nil, err
		}
		item.ObservationID = scanNullableInt64(obsID)
		item.RetrievalResultID = scanNullableInt64(resultID)
		alignments = append(alignments, item)
	}
	if err := alignRows.Err(); err != nil {
		return nil, err
	}
	view.RecentAlignmentNotes = alignments
	if summary, summaryErr := s.DossierVSASummary(ctx, dossierID); summaryErr == nil {
		view.VSASummary = summary
	}
	return view, nil
}
