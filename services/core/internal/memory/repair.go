package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// RunRepairPass is retained as a fail-closed compatibility seam. Live repair
// is proposal-only; it cannot rewrite evidence or projection rows.
func (s *Service) RunRepairPass(ctx context.Context, req RunRepairRequest) (*RepairRunDetail, error) {
	return nil, ErrMemoryEvidenceAuthorityRequired
}

// PreviewRepairPass selects the same candidates as RunRepairPass without
// creating a run, rewriting historical observations, or rebuilding VSA
// projections. Live callers use this while repair commit authority is being
// moved behind FORGE-K.
func (s *Service) PreviewRepairPass(ctx context.Context, req RunRepairRequest) (*MaintenancePreview, error) {
	if req.MaxAgeDays <= 0 || req.MaxAgeDays > 365 {
		req.MaxAgeDays = 14
	}
	if req.Limit <= 0 || req.Limit > 300 {
		req.Limit = 120
	}
	candidates, err := s.repairCandidates(ctx, req.DossierID, req.MaxAgeDays, req.Limit)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(candidates))
	for _, candidate := range candidates {
		ids = append(ids, candidate.ID)
	}
	return &MaintenancePreview{
		Kind:          "memory.repair",
		DryRun:        true,
		ProposalOnly:  true,
		DossierID:     req.DossierID,
		CandidateIDs:  ids,
		Candidates:    len(ids),
		WouldWrite:    []string{"memory_observations", "memory_repair_runs", "memory_repair_items", "memory_vsa_*"},
		RequiresOwner: "forge_k.kernel",
		Note:          strings.TrimSpace(req.Note),
	}, nil
}

func (s *Service) repairCandidates(ctx context.Context, dossierID *int64, maxAgeDays, limit int) ([]Observation, error) {
	cutoff := time.Now().Add(-time.Duration(maxAgeDays) * 24 * time.Hour).UnixMilli()
	query := `
SELECT id
FROM memory_observations
WHERE (stale = 1 OR usefulness_score < -0.5 OR noise_count > usefulness_count OR COALESCE(last_verified_at, 0) < ?)`
	args := []any{cutoff}
	if dossierID != nil && *dossierID > 0 {
		query += ` AND dossier_id = ?`
		args = append(args, *dossierID)
	}
	query += ` ORDER BY stale DESC, usefulness_score ASC, updated_at ASC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	ids := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	_ = rows.Close()

	out := make([]Observation, 0, len(ids))
	for _, id := range ids {
		detail, detailErr := s.GetObservation(ctx, id)
		if detailErr != nil {
			continue
		}
		out = append(out, detail.Observation)
	}
	return out, nil
}

func (s *Service) ListRepairRuns(ctx context.Context, limit int, dossierID *int64) ([]RepairRun, error) {
	if limit <= 0 || limit > 200 {
		limit = 80
	}
	query := `
SELECT id, created_at, started_at, completed_at, dossier_id, mode, max_age_days, candidates, repaired, skipped, failed, note
FROM memory_repair_runs`
	args := []any{}
	if dossierID != nil && *dossierID > 0 {
		query += ` WHERE dossier_id = ?`
		args = append(args, *dossierID)
	}
	query += ` ORDER BY id DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []RepairRun{}
	for rows.Next() {
		item, err := scanRepairRun(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Service) GetRepairRun(ctx context.Context, runID int64) (*RepairRunDetail, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, created_at, started_at, completed_at, dossier_id, mode, max_age_days, candidates, repaired, skipped, failed, note
FROM memory_repair_runs
WHERE id = ?`, runID)
	run, err := scanRepairRun(row)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, repair_run_id, observation_id, status, issue, before_json, after_json, note, created_at
FROM memory_repair_items
WHERE repair_run_id = ?
ORDER BY id DESC`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []RepairItem{}
	for rows.Next() {
		var item RepairItem
		var beforeJSON, afterJSON string
		if err := rows.Scan(&item.ID, &item.RepairRunID, &item.ObservationID, &item.Status, &item.Issue, &beforeJSON, &afterJSON, &item.Note, &item.CreatedAtMs); err != nil {
			return nil, err
		}
		item.Before = json.RawMessage(strings.TrimSpace(beforeJSON))
		if len(item.Before) == 0 {
			item.Before = json.RawMessage("{}")
		}
		item.After = json.RawMessage(strings.TrimSpace(afterJSON))
		if len(item.After) == 0 {
			item.After = json.RawMessage("{}")
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &RepairRunDetail{Run: run, Items: items}, nil
}

type repairRunScanner interface {
	Scan(dest ...any) error
}

func scanRepairRun(scanner repairRunScanner) (RepairRun, error) {
	var run RepairRun
	var completed sql.NullInt64
	var dossierID sql.NullInt64
	if err := scanner.Scan(&run.ID, &run.CreatedAtMs, &run.StartedAtMs, &completed, &dossierID, &run.Mode, &run.MaxAgeDays, &run.Candidates, &run.Repaired, &run.Skipped, &run.Failed, &run.Note); err != nil {
		return run, err
	}
	run.CompletedAtMs = scanNullableInt64(completed)
	run.DossierID = scanNullableInt64(dossierID)
	return run, nil
}

func (s *Service) RepairTicker(ctx context.Context, interval time.Duration, maxAgeDays int, limit int) {
	if interval <= 0 {
		interval = 20 * time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, _ = s.PreviewRepairPass(ctx, RunRepairRequest{
				Mode:       "scheduled",
				MaxAgeDays: maxAgeDays,
				Limit:      limit,
				Note:       fmt.Sprintf("scheduled repair proposal every %s", interval),
			})
		}
	}
}
