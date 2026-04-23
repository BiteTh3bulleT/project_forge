package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func (s *Service) RunRepairPass(ctx context.Context, req RunRepairRequest) (*RepairRunDetail, error) {
	now := time.Now().UnixMilli()
	mode := strings.TrimSpace(req.Mode)
	if mode == "" {
		mode = "manual"
	}
	if req.MaxAgeDays <= 0 || req.MaxAgeDays > 365 {
		req.MaxAgeDays = 14
	}
	if req.Limit <= 0 || req.Limit > 300 {
		req.Limit = 120
	}
	res, err := s.db.ExecContext(ctx, `
INSERT INTO memory_repair_runs(created_at, started_at, dossier_id, mode, max_age_days, note)
VALUES(?,?,?,?,?,?)`,
		now,
		now,
		nullInt64(req.DossierID),
		mode,
		req.MaxAgeDays,
		strings.TrimSpace(req.Note),
	)
	if err != nil {
		return nil, err
	}
	runID, _ := res.LastInsertId()
	var vsaRunID *int64
	vsaIndexed := 0
	vsaSkipped := 0
	vsaFailed := 0
	if id, runErr := s.createVSAReindexRun(ctx, RunVSAReindexRequest{
		DossierID:   req.DossierID,
		Mode:        "repair",
		Limit:       req.Limit,
		TriggeredBy: "memory_repair",
		Reason:      "repair_flow",
		Note:        fmt.Sprintf("repair_run_id=%d %s", runID, strings.TrimSpace(req.Note)),
	}); runErr == nil {
		vsaRunID = &id
	}

	candidates, err := s.repairCandidates(ctx, req.DossierID, req.MaxAgeDays, req.Limit)
	if err != nil {
		return nil, err
	}
	repaired := 0
	skipped := 0
	failed := 0
	items := make([]RepairItem, 0, len(candidates))
	for _, obs := range candidates {
		item, action, vsaAction, repairErr := s.repairObservation(ctx, runID, obs, vsaRunID)
		if repairErr != nil {
			failed++
			if vsaRunID != nil {
				vsaFailed++
			}
			items = append(items, RepairItem{
				RepairRunID:   runID,
				ObservationID: obs.ID,
				Status:        "failed",
				Issue:         "repair_error",
				Before:        json.RawMessage("{}"),
				After:         json.RawMessage("{}"),
				Note:          repairErr.Error(),
				CreatedAtMs:   time.Now().UnixMilli(),
			})
			continue
		}
		switch action {
		case "repaired":
			repaired++
		case "skipped":
			skipped++
		default:
			skipped++
		}
		switch vsaAction {
		case "indexed":
			vsaIndexed++
		case "failed":
			vsaFailed++
		default:
			vsaSkipped++
		}
		items = append(items, *item)
	}
	completed := time.Now().UnixMilli()
	_, _ = s.db.ExecContext(ctx, `
UPDATE memory_repair_runs
SET completed_at = ?, candidates = ?, repaired = ?, skipped = ?, failed = ?
WHERE id = ?`,
		completed,
		len(candidates),
		repaired,
		skipped,
		failed,
		runID,
	)
	if vsaRunID != nil {
		s.completeVSAReindexRun(ctx, *vsaRunID, len(candidates), vsaIndexed, vsaSkipped, vsaFailed)
	}
	return s.GetRepairRun(ctx, runID)
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

func (s *Service) repairObservation(ctx context.Context, runID int64, obs Observation, vsaRunID *int64) (*RepairItem, string, string, error) {
	beforeState := map[string]any{
		"summary":           obs.Summary,
		"verificationState": obs.VerificationState,
		"stale":             obs.Stale,
		"lastVerifiedAtMs":  obs.LastVerifiedAtMs,
		"rawContent":        obs.RawContent,
	}
	before, _ := json.Marshal(beforeState)
	now := time.Now().UnixMilli()
	newRaw := strings.TrimSpace(obs.RawContent)
	newSummary := strings.TrimSpace(obs.Summary)
	changed := false
	note := ""
	if strings.TrimSpace(obs.OriginKind) == "retrieval_result" {
		resultID, parseErr := strconv.ParseInt(strings.TrimSpace(obs.OriginID), 10, 64)
		if parseErr == nil && resultID > 0 {
			var snippet sql.NullString
			if err := s.db.QueryRowContext(ctx, `SELECT snippet FROM retrieval_results WHERE id = ?`, resultID).Scan(&snippet); err == nil && snippet.Valid {
				fresh := strings.TrimSpace(snippet.String)
				if fresh != "" && fresh != newRaw {
					newRaw = fresh
					changed = true
					note = "raw content refreshed from retrieval result snippet"
				}
			}
		}
	}
	if newSummary == "" || obs.Stale || obs.UsefulnessScore < -0.5 {
		auto := summarizeRawContent(newRaw)
		if auto != "" && auto != newSummary {
			newSummary = auto
			changed = true
			if note == "" {
				note = "summary refreshed from current raw content"
			}
		}
	}
	newVerification := "checked"
	if changed {
		newVerification = "refreshed"
	}
	stale := 0
	_, err := s.db.ExecContext(ctx, `
UPDATE memory_observations
SET updated_at = ?, raw_content = ?, summary = ?, verification_state = ?, stale = ?, last_verified_at = ?
WHERE id = ?`,
		now,
		newRaw,
		newSummary,
		newVerification,
		stale,
		now,
		obs.ID,
	)
	if err != nil {
		return nil, "failed", "failed", err
	}
	afterState := map[string]any{
		"summary":           newSummary,
		"verificationState": newVerification,
		"stale":             false,
		"lastVerifiedAtMs":  now,
		"rawContent":        newRaw,
	}
	after, _ := json.Marshal(afterState)
	status := "skipped"
	issue := "no_change"
	if changed {
		status = "repaired"
		issue = "updated"
	}
	if note == "" {
		note = "observation verified"
	}
	res, err := s.db.ExecContext(ctx, `
INSERT INTO memory_repair_items(repair_run_id, observation_id, status, issue, before_json, after_json, note, created_at)
VALUES(?,?,?,?,?,?,?,?)`,
		runID,
		obs.ID,
		status,
		issue,
		string(before),
		string(after),
		note,
		now,
	)
	if err != nil {
		return nil, "failed", "failed", err
	}
	itemID, _ := res.LastInsertId()
	item := &RepairItem{
		ID:            itemID,
		RepairRunID:   runID,
		ObservationID: obs.ID,
		Status:        status,
		Issue:         issue,
		Before:        before,
		After:         after,
		Note:          note,
		CreatedAtMs:   now,
	}
	vsaStatus := "skipped"
	if vsaRunID != nil {
		_, _, vsaStatusOut, reindexErr := s.reindexObservationVSAWithEvidence(
			ctx,
			obs.ID,
			"repair_flow",
			vsaRunID,
			false,
			beforeState,
			afterState,
			fmt.Sprintf("repair_run_id=%d repair_item_id=%d status=%s", runID, itemID, status),
		)
		if reindexErr != nil {
			vsaStatus = "failed"
			item.Note = strings.TrimSpace(item.Note + "; vsa reindex failed: " + reindexErr.Error())
			_, _ = s.db.ExecContext(ctx, `UPDATE memory_repair_items SET note = ? WHERE id = ?`, item.Note, itemID)
		} else {
			vsaStatus = vsaStatusOut
		}
	}
	return item, status, vsaStatus, nil
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
			_, _ = s.RunRepairPass(ctx, RunRepairRequest{
				Mode:       "scheduled",
				MaxAgeDays: maxAgeDays,
				Limit:      limit,
				Note:       fmt.Sprintf("scheduled repair every %s", interval),
			})
		}
	}
}
