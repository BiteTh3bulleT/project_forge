package lineage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Relation struct {
	ID            int64           `json:"id"`
	ParentJobID   string          `json:"parentJobId"`
	ChildJobID    string          `json:"childJobId"`
	RelationType  string          `json:"relationType"`
	CreatedAtMs   int64           `json:"createdAtMs"`
	ChangeSummary json.RawMessage `json:"changeSummary"`
}

type JobSummary struct {
	ID              string  `json:"id"`
	Title           string  `json:"title"`
	Status          string  `json:"status"`
	TargetAdapter   string  `json:"targetAdapter"`
	CreatedAtMs     int64   `json:"createdAtMs"`
	ResultSummary   *string `json:"resultSummary"`
	LastFailureCode *string `json:"lastFailureCode"`
}

type JobLineage struct {
	JobID    string       `json:"jobId"`
	Parents  []Relation   `json:"parents"`
	Children []Relation   `json:"children"`
	Related  []JobSummary `json:"relatedJobs"`
}

type Service struct {
	db *sql.DB
}

func New(db *sql.DB) *Service { return &Service{db: db} }

func (s *Service) Link(ctx context.Context, parentJobID, childJobID, relationType string, changeSummary map[string]any) (*Relation, error) {
	parentJobID = strings.TrimSpace(parentJobID)
	childJobID = strings.TrimSpace(childJobID)
	relationType = strings.TrimSpace(strings.ToLower(relationType))
	if parentJobID == "" || childJobID == "" {
		return nil, fmt.Errorf("parent and child job ids are required")
	}
	if relationType == "" {
		relationType = "retry"
	}
	if changeSummary == nil {
		changeSummary = map[string]any{}
	}
	payload, _ := json.Marshal(changeSummary)
	now := time.Now().UnixMilli()

	_, err := s.db.ExecContext(ctx, `
INSERT INTO job_lineage(parent_job_id, child_job_id, relation_type, created_at, change_summary_json)
VALUES(?,?,?,?,?)
ON CONFLICT(parent_job_id, child_job_id)
DO UPDATE SET relation_type = excluded.relation_type, created_at = excluded.created_at, change_summary_json = excluded.change_summary_json`,
		parentJobID, childJobID, relationType, now, string(payload),
	)
	if err != nil {
		return nil, err
	}
	return s.GetRelation(ctx, parentJobID, childJobID)
}

func (s *Service) GetRelation(ctx context.Context, parentJobID, childJobID string) (*Relation, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, parent_job_id, child_job_id, relation_type, created_at, change_summary_json
FROM job_lineage
WHERE parent_job_id = ? AND child_job_id = ?`, parentJobID, childJobID)
	var r Relation
	var payload string
	if err := row.Scan(&r.ID, &r.ParentJobID, &r.ChildJobID, &r.RelationType, &r.CreatedAtMs, &payload); err != nil {
		return nil, err
	}
	r.ChangeSummary = json.RawMessage(payload)
	return &r, nil
}

func (s *Service) ForJob(ctx context.Context, jobID string) (*JobLineage, error) {
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return nil, fmt.Errorf("jobId is required")
	}

	parents, err := s.listRelations(ctx, `SELECT id, parent_job_id, child_job_id, relation_type, created_at, change_summary_json FROM job_lineage WHERE child_job_id = ? ORDER BY id DESC`, jobID)
	if err != nil {
		return nil, err
	}
	children, err := s.listRelations(ctx, `SELECT id, parent_job_id, child_job_id, relation_type, created_at, change_summary_json FROM job_lineage WHERE parent_job_id = ? ORDER BY id DESC`, jobID)
	if err != nil {
		return nil, err
	}

	seen := map[string]struct{}{}
	relatedIDs := make([]string, 0, len(parents)+len(children))
	for _, p := range parents {
		if _, ok := seen[p.ParentJobID]; !ok {
			seen[p.ParentJobID] = struct{}{}
			relatedIDs = append(relatedIDs, p.ParentJobID)
		}
	}
	for _, c := range children {
		if _, ok := seen[c.ChildJobID]; !ok {
			seen[c.ChildJobID] = struct{}{}
			relatedIDs = append(relatedIDs, c.ChildJobID)
		}
	}
	related, err := s.jobSummaries(ctx, relatedIDs)
	if err != nil {
		return nil, err
	}

	return &JobLineage{
		JobID:    jobID,
		Parents:  parents,
		Children: children,
		Related:  related,
	}, nil
}

func (s *Service) listRelations(ctx context.Context, query string, arg any) ([]Relation, error) {
	rows, err := s.db.QueryContext(ctx, query, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Relation{}
	for rows.Next() {
		var r Relation
		var payload string
		if err := rows.Scan(&r.ID, &r.ParentJobID, &r.ChildJobID, &r.RelationType, &r.CreatedAtMs, &payload); err != nil {
			return nil, err
		}
		r.ChangeSummary = json.RawMessage(payload)
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Service) jobSummaries(ctx context.Context, ids []string) ([]JobSummary, error) {
	if len(ids) == 0 {
		return []JobSummary{}, nil
	}
	query := `
SELECT id, title, status, target_adapter, created_at, result_summary, last_failure_code
FROM jobs
WHERE id IN (` + placeholders(len(ids)) + `)
ORDER BY created_at DESC`
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []JobSummary{}
	for rows.Next() {
		var r JobSummary
		var result sql.NullString
		var fail sql.NullString
		if err := rows.Scan(&r.ID, &r.Title, &r.Status, &r.TargetAdapter, &r.CreatedAtMs, &result, &fail); err != nil {
			return nil, err
		}
		if result.Valid {
			v := result.String
			r.ResultSummary = &v
		}
		if fail.Valid {
			v := fail.String
			r.LastFailureCode = &v
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	var b strings.Builder
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString("?")
	}
	return b.String()
}
