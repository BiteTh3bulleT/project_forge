package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"
)

func (s *Service) SaveSelectionReason(ctx context.Context, req SaveSelectionReasonRequest) error {
	if req.RetrievalResultID <= 0 {
		return fmt.Errorf("retrieval result id is required")
	}
	reasonJSON, _ := json.Marshal(req.Reason)
	_, err := s.db.ExecContext(ctx, `
INSERT INTO retrieval_result_selection(retrieval_result_id, reason_json, created_at)
VALUES(?,?,?)
ON CONFLICT(retrieval_result_id) DO UPDATE SET
  reason_json=excluded.reason_json,
  created_at=excluded.created_at`,
		req.RetrievalResultID,
		string(reasonJSON),
		time.Now().UnixMilli(),
	)
	return err
}

func (s *Service) SelectionReasonsForRun(ctx context.Context, runID int64) (map[int64]RetrievalSelection, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT rs.retrieval_result_id, rs.reason_json, rs.created_at
FROM retrieval_result_selection rs
JOIN retrieval_results rr ON rr.id = rs.retrieval_result_id
WHERE rr.retrieval_run_id = ?`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]RetrievalSelection{}
	for rows.Next() {
		var row RetrievalSelection
		var reason string
		if err := rows.Scan(&row.RetrievalResultID, &reason, &row.CreatedAtMs); err != nil {
			return nil, err
		}
		trimmed := strings.TrimSpace(reason)
		if trimmed == "" {
			trimmed = "{}"
		}
		row.Reason = json.RawMessage(trimmed)
		out[row.RetrievalResultID] = row
	}
	return out, rows.Err()
}

func (s *Service) LinkResultObservation(ctx context.Context, resultID, observationID int64, note string) error {
	if resultID <= 0 || observationID <= 0 {
		return nil
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO retrieval_result_observations(retrieval_result_id, observation_id, selection_note, created_at)
VALUES(?,?,?,?)
ON CONFLICT(retrieval_result_id, observation_id) DO UPDATE SET
  selection_note=excluded.selection_note,
  created_at=excluded.created_at`,
		resultID,
		observationID,
		strings.TrimSpace(note),
		time.Now().UnixMilli(),
	)
	return err
}

func (s *Service) ObservationByResult(ctx context.Context, resultID int64) (*int64, error) {
	var obsID sql.NullInt64
	err := s.db.QueryRowContext(ctx, `
SELECT observation_id
FROM retrieval_result_observations
WHERE retrieval_result_id = ?
ORDER BY created_at DESC, observation_id DESC
LIMIT 1`, resultID).Scan(&obsID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !obsID.Valid {
		return nil, nil
	}
	v := obsID.Int64
	return &v, nil
}

func (s *Service) MarkObservationUsefulness(ctx context.Context, req MarkUsefulnessRequest) error {
	if req.ObservationID <= 0 {
		return fmt.Errorf("observation id is required")
	}
	signal := strings.TrimSpace(strings.ToLower(req.Signal))
	if signal == "" {
		signal = "unknown"
	}
	weight := req.Weight
	if weight == 0 {
		weight = 1
	}
	weight = math.Max(-1, math.Min(1, weight))
	_, err := s.db.ExecContext(ctx, `
INSERT INTO memory_usefulness_events(
  created_at, observation_id, retrieval_result_id, retrieval_run_id, packet_id, job_id,
  signal, weight, note
) VALUES(?,?,?,?,?,?,?,?,?)`,
		time.Now().UnixMilli(),
		req.ObservationID,
		nullInt64(req.RetrievalResultID),
		nullInt64(req.RetrievalRunID),
		nullInt64(req.PacketID),
		req.JobID,
		signal,
		weight,
		strings.TrimSpace(req.Note),
	)
	if err != nil {
		return err
	}
	if err := s.refreshObservationUsefulness(ctx, req.ObservationID); err != nil {
		return err
	}
	_ = s.TouchVSAReliabilityFromUsefulness(ctx, req.ObservationID, signal, weight)
	return nil
}

func (s *Service) refreshObservationUsefulness(ctx context.Context, observationID int64) error {
	rows, err := s.db.QueryContext(ctx, `
SELECT signal, weight
FROM memory_usefulness_events
WHERE observation_id = ?`, observationID)
	if err != nil {
		return err
	}
	defer rows.Close()
	score := 0.0
	usefulCount := 0
	noiseCount := 0
	for rows.Next() {
		var signal string
		var weight float64
		if err := rows.Scan(&signal, &weight); err != nil {
			return err
		}
		signal = strings.TrimSpace(strings.ToLower(signal))
		switch signal {
		case "useful", "succeeded", "success":
			score += 1 * weight
			usefulCount++
		case "noisy", "not_useful", "misleading", "failed", "failure":
			score -= 1 * math.Abs(weight)
			noiseCount++
		case "insufficient":
			score -= 0.25 * math.Abs(weight)
		default:
			score += 0
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
UPDATE memory_observations
SET updated_at = ?, usefulness_score = ?, usefulness_count = ?, noise_count = ?
WHERE id = ?`,
		time.Now().UnixMilli(),
		score,
		usefulCount,
		noiseCount,
		observationID,
	)
	return err
}

func (s *Service) SelectionByRun(ctx context.Context, runID int64) ([]RetrievalSelection, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT rs.retrieval_result_id, rs.reason_json, rs.created_at
FROM retrieval_result_selection rs
JOIN retrieval_results rr ON rr.id = rs.retrieval_result_id
WHERE rr.retrieval_run_id = ?
ORDER BY rr.rank_index ASC`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []RetrievalSelection{}
	for rows.Next() {
		var item RetrievalSelection
		var reason string
		if err := rows.Scan(&item.RetrievalResultID, &reason, &item.CreatedAtMs); err != nil {
			return nil, err
		}
		item.Reason = json.RawMessage(strings.TrimSpace(reason))
		if len(item.Reason) == 0 {
			item.Reason = json.RawMessage("{}")
		}
		out = append(out, item)
	}
	return out, rows.Err()
}
