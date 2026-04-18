package reconciliation

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Record struct {
	ID                 int64           `json:"id"`
	ImportID           int64           `json:"importId"`
	CreatedAtMs        int64           `json:"createdAtMs"`
	UpdatedAtMs        int64           `json:"updatedAtMs"`
	ChangedFiles       json.RawMessage `json:"changedFiles"`
	FailureReasons     json.RawMessage `json:"failureReasons"`
	UnresolvedIssues   json.RawMessage `json:"unresolvedIssues"`
	SuggestedNextSteps json.RawMessage `json:"suggestedNextSteps"`
	AgentNotes         string          `json:"agentNotes"`
	PatchSummary       string          `json:"patchSummary"`
	ReviewStatus       string          `json:"reviewStatus"`
}

type SaveRequest struct {
	ImportID           int64    `json:"importId"`
	ChangedFiles       []string `json:"changedFiles"`
	FailureReasons     []string `json:"failureReasons"`
	UnresolvedIssues   []string `json:"unresolvedIssues"`
	SuggestedNextSteps []string `json:"suggestedNextSteps"`
	AgentNotes         string   `json:"agentNotes"`
	PatchSummary       string   `json:"patchSummary"`
	ReviewStatus       string   `json:"reviewStatus"`
}

type Service struct {
	db *sql.DB
}

func New(db *sql.DB) *Service { return &Service{db: db} }

func (s *Service) Save(ctx context.Context, req SaveRequest) (*Record, error) {
	if req.ImportID <= 0 {
		return nil, fmt.Errorf("importId is required")
	}
	changedFiles, _ := json.Marshal(nonNilStrings(req.ChangedFiles))
	failureReasons, _ := json.Marshal(nonNilStrings(req.FailureReasons))
	unresolved, _ := json.Marshal(nonNilStrings(req.UnresolvedIssues))
	nextSteps, _ := json.Marshal(nonNilStrings(req.SuggestedNextSteps))
	now := time.Now().UnixMilli()
	status := strings.TrimSpace(strings.ToLower(req.ReviewStatus))
	if status == "" {
		status = "pending"
	}

	_, err := s.db.ExecContext(ctx, `
INSERT INTO imported_execution_reconciliations(
  import_id, created_at, updated_at, changed_files_json, failure_reasons_json,
  unresolved_issues_json, suggested_next_steps_json, agent_notes, patch_summary, review_status
) VALUES(?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(import_id)
DO UPDATE SET
  updated_at=excluded.updated_at,
  changed_files_json=excluded.changed_files_json,
  failure_reasons_json=excluded.failure_reasons_json,
  unresolved_issues_json=excluded.unresolved_issues_json,
  suggested_next_steps_json=excluded.suggested_next_steps_json,
  agent_notes=excluded.agent_notes,
  patch_summary=excluded.patch_summary,
  review_status=excluded.review_status`,
		req.ImportID,
		now,
		now,
		string(changedFiles),
		string(failureReasons),
		string(unresolved),
		string(nextSteps),
		strings.TrimSpace(req.AgentNotes),
		strings.TrimSpace(req.PatchSummary),
		status,
	)
	if err != nil {
		return nil, err
	}
	return s.ByImport(ctx, req.ImportID)
}

func (s *Service) ByImport(ctx context.Context, importID int64) (*Record, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, import_id, created_at, updated_at, changed_files_json, failure_reasons_json,
       unresolved_issues_json, suggested_next_steps_json, agent_notes, patch_summary, review_status
FROM imported_execution_reconciliations
WHERE import_id = ?`, importID)
	return scanRecord(row)
}

func (s *Service) List(ctx context.Context, limit int, reviewStatus string) ([]Record, error) {
	if limit <= 0 || limit > 300 {
		limit = 100
	}
	query := `
SELECT id, import_id, created_at, updated_at, changed_files_json, failure_reasons_json,
       unresolved_issues_json, suggested_next_steps_json, agent_notes, patch_summary, review_status
FROM imported_execution_reconciliations`
	args := []any{}
	reviewStatus = strings.TrimSpace(strings.ToLower(reviewStatus))
	if reviewStatus != "" {
		query += ` WHERE review_status = ?`
		args = append(args, reviewStatus)
	}
	query += ` ORDER BY updated_at DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Record{}
	for rows.Next() {
		r, err := scanRecord(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return out, rows.Err()
}

func scanRecord(scanner interface{ Scan(dest ...any) error }) (*Record, error) {
	var r Record
	var changedFiles string
	var failureReasons string
	var unresolved string
	var nextSteps string
	if err := scanner.Scan(
		&r.ID,
		&r.ImportID,
		&r.CreatedAtMs,
		&r.UpdatedAtMs,
		&changedFiles,
		&failureReasons,
		&unresolved,
		&nextSteps,
		&r.AgentNotes,
		&r.PatchSummary,
		&r.ReviewStatus,
	); err != nil {
		return nil, err
	}
	r.ChangedFiles = json.RawMessage(changedFiles)
	r.FailureReasons = json.RawMessage(failureReasons)
	r.UnresolvedIssues = json.RawMessage(unresolved)
	r.SuggestedNextSteps = json.RawMessage(nextSteps)
	return &r, nil
}

func nonNilStrings(v []string) []string {
	if v == nil {
		return []string{}
	}
	out := make([]string, 0, len(v))
	for _, item := range v {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		out = append(out, item)
	}
	return out
}
