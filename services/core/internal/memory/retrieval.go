package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
)

var ErrMemoryEvidenceAuthorityRequired = errors.New("memory evidence writes require governed FORGE-K syscall")

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
	return ErrMemoryEvidenceAuthorityRequired
}

func (s *Service) ObservationByResult(ctx context.Context, resultID int64) (*int64, error) {
	var obsID sql.NullInt64
	err := s.db.QueryRowContext(ctx, `
SELECT observation_id
FROM retrieval_result_observations
WHERE retrieval_result_id = ?
ORDER BY created_at DESC, observation_id DESC
LIMIT 1`, resultID).Scan(&obsID)
	if errors.Is(err, sql.ErrNoRows) {
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
	return ErrMemoryEvidenceAuthorityRequired
}

func (s *Service) refreshObservationUsefulness(ctx context.Context, observationID int64) error {
	return ErrMemoryEvidenceAuthorityRequired
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
