package dossiers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Dossier struct {
	ID                int64           `json:"id"`
	CreatedAtMs       int64           `json:"createdAtMs"`
	UpdatedAtMs       int64           `json:"updatedAtMs"`
	Name              string          `json:"name"`
	Description       string          `json:"description"`
	PrimaryPaths      json.RawMessage `json:"primaryPaths"`
	RelatedRepos      json.RawMessage `json:"relatedRepos"`
	Constraints       json.RawMessage `json:"constraints"`
	PreferredAdapters json.RawMessage `json:"preferredAdapters"`
	ImportantFiles    json.RawMessage `json:"importantFiles"`
	RoutingNotes      string          `json:"routingNotes"`
}

type Brief struct {
	ID              int64           `json:"id"`
	DossierID       int64           `json:"dossierId"`
	CreatedAtMs     int64           `json:"createdAtMs"`
	SummaryMarkdown string          `json:"summaryMarkdown"`
	Context         json.RawMessage `json:"context"`
	Notes           string          `json:"notes"`
}

type SourceLink struct {
	SourceID int64  `json:"sourceId"`
	Path     string `json:"path"`
}

type DossierDetail struct {
	Dossier    Dossier       `json:"dossier"`
	Sources    []SourceLink  `json:"sources"`
	RecentJobs []JobSnapshot `json:"recentJobs"`
	Briefs     []Brief       `json:"briefs"`
}

type JobSnapshot struct {
	JobID          string  `json:"jobId"`
	Title          string  `json:"title"`
	Status         string  `json:"status"`
	TargetAdapter  string  `json:"targetAdapter"`
	CreatedAtMs    int64   `json:"createdAtMs"`
	ResultSummary  *string `json:"resultSummary"`
	LastFailureCode *string `json:"lastFailureCode"`
}

type CreateRequest struct {
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	SourceIDs         []int64  `json:"sourceIds"`
	PrimaryPaths      []string `json:"primaryPaths"`
	RelatedRepos      []string `json:"relatedRepos"`
	Constraints       []string `json:"constraints"`
	PreferredAdapters []string `json:"preferredAdapters"`
	ImportantFiles    []string `json:"importantFiles"`
	RoutingNotes      string   `json:"routingNotes"`
}

type UpdateRequest struct {
	Description       *string   `json:"description"`
	SourceIDs         []int64   `json:"sourceIds"`
	PrimaryPaths      *[]string `json:"primaryPaths"`
	RelatedRepos      *[]string `json:"relatedRepos"`
	Constraints       *[]string `json:"constraints"`
	PreferredAdapters *[]string `json:"preferredAdapters"`
	ImportantFiles    *[]string `json:"importantFiles"`
	RoutingNotes      *string   `json:"routingNotes"`
}

type Service struct {
	db *sql.DB
}

func New(db *sql.DB) *Service { return &Service{db: db} }

func (s *Service) Create(ctx context.Context, req CreateRequest) (*Dossier, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("dossier name is required")
	}
	now := time.Now().UnixMilli()
	primary := choosePaths(req.PrimaryPaths)
	if len(primary) == 0 && len(req.SourceIDs) > 0 {
		paths, err := s.sourcePaths(ctx, req.SourceIDs)
		if err == nil {
			primary = paths
		}
	}
	primaryJSON, _ := json.Marshal(primary)
	relatedJSON, _ := json.Marshal(req.RelatedRepos)
	constraintsJSON, _ := json.Marshal(req.Constraints)
	adaptersJSON, _ := json.Marshal(req.PreferredAdapters)
	importantJSON, _ := json.Marshal(req.ImportantFiles)

	res, err := s.db.ExecContext(ctx, `
INSERT INTO dossiers(
  created_at, updated_at, name, description,
  primary_paths_json, related_repos_json, constraints_json,
  preferred_adapters_json, important_files_json, routing_notes
) VALUES(?,?,?,?,?,?,?,?,?,?)`,
		now, now, name, strings.TrimSpace(req.Description),
		string(primaryJSON), string(relatedJSON), string(constraintsJSON),
		string(adaptersJSON), string(importantJSON), strings.TrimSpace(req.RoutingNotes),
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()

	for _, sid := range req.SourceIDs {
		_, _ = s.db.ExecContext(ctx,
			`INSERT INTO dossier_sources(dossier_id, source_id, linked_at) VALUES(?,?,?) ON CONFLICT(dossier_id, source_id) DO NOTHING`,
			id, sid, now,
		)
	}

	return s.Get(ctx, id)
}

func (s *Service) Update(ctx context.Context, dossierID int64, req UpdateRequest) (*Dossier, error) {
	current, err := s.Get(ctx, dossierID)
	if err != nil {
		return nil, err
	}
	if req.Description != nil {
		current.Description = strings.TrimSpace(*req.Description)
	}
	if req.PrimaryPaths != nil {
		b, _ := json.Marshal(choosePaths(*req.PrimaryPaths))
		current.PrimaryPaths = b
	}
	if req.RelatedRepos != nil {
		b, _ := json.Marshal(*req.RelatedRepos)
		current.RelatedRepos = b
	}
	if req.Constraints != nil {
		b, _ := json.Marshal(*req.Constraints)
		current.Constraints = b
	}
	if req.PreferredAdapters != nil {
		b, _ := json.Marshal(*req.PreferredAdapters)
		current.PreferredAdapters = b
	}
	if req.ImportantFiles != nil {
		b, _ := json.Marshal(*req.ImportantFiles)
		current.ImportantFiles = b
	}
	if req.RoutingNotes != nil {
		current.RoutingNotes = strings.TrimSpace(*req.RoutingNotes)
	}

	now := time.Now().UnixMilli()
	_, err = s.db.ExecContext(ctx, `
UPDATE dossiers
SET updated_at = ?, description = ?, primary_paths_json = ?, related_repos_json = ?, constraints_json = ?,
    preferred_adapters_json = ?, important_files_json = ?, routing_notes = ?
WHERE id = ?`,
		now,
		current.Description,
		string(current.PrimaryPaths),
		string(current.RelatedRepos),
		string(current.Constraints),
		string(current.PreferredAdapters),
		string(current.ImportantFiles),
		current.RoutingNotes,
		dossierID,
	)
	if err != nil {
		return nil, err
	}

	if req.SourceIDs != nil {
		_, _ = s.db.ExecContext(ctx, `DELETE FROM dossier_sources WHERE dossier_id = ?`, dossierID)
		for _, sid := range req.SourceIDs {
			_, _ = s.db.ExecContext(ctx,
				`INSERT INTO dossier_sources(dossier_id, source_id, linked_at) VALUES(?,?,?) ON CONFLICT(dossier_id, source_id) DO NOTHING`,
				dossierID, sid, now,
			)
		}
	}

	return s.Get(ctx, dossierID)
}

func (s *Service) List(ctx context.Context, limit int) ([]Dossier, error) {
	if limit <= 0 || limit > 300 {
		limit = 120
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, created_at, updated_at, name, description,
       primary_paths_json, related_repos_json, constraints_json,
       preferred_adapters_json, important_files_json, routing_notes
FROM dossiers ORDER BY updated_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Dossier{}
	for rows.Next() {
		var d Dossier
		var p, r, c, a, i string
		if err := rows.Scan(&d.ID, &d.CreatedAtMs, &d.UpdatedAtMs, &d.Name, &d.Description, &p, &r, &c, &a, &i, &d.RoutingNotes); err != nil {
			return nil, err
		}
		d.PrimaryPaths = json.RawMessage(p)
		d.RelatedRepos = json.RawMessage(r)
		d.Constraints = json.RawMessage(c)
		d.PreferredAdapters = json.RawMessage(a)
		d.ImportantFiles = json.RawMessage(i)
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *Service) Get(ctx context.Context, dossierID int64) (*Dossier, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, created_at, updated_at, name, description,
       primary_paths_json, related_repos_json, constraints_json,
       preferred_adapters_json, important_files_json, routing_notes
FROM dossiers WHERE id = ?`, dossierID)
	var d Dossier
	var p, r, c, a, i string
	if err := row.Scan(&d.ID, &d.CreatedAtMs, &d.UpdatedAtMs, &d.Name, &d.Description, &p, &r, &c, &a, &i, &d.RoutingNotes); err != nil {
		return nil, err
	}
	d.PrimaryPaths = json.RawMessage(p)
	d.RelatedRepos = json.RawMessage(r)
	d.Constraints = json.RawMessage(c)
	d.PreferredAdapters = json.RawMessage(a)
	d.ImportantFiles = json.RawMessage(i)
	return &d, nil
}

func (s *Service) Detail(ctx context.Context, dossierID int64) (*DossierDetail, error) {
	d, err := s.Get(ctx, dossierID)
	if err != nil {
		return nil, err
	}
	sources, err := s.SourceLinks(ctx, dossierID)
	if err != nil {
		return nil, err
	}
	jobs, err := s.RecentJobs(ctx, dossierID, 30)
	if err != nil {
		return nil, err
	}
	briefs, err := s.Briefs(ctx, dossierID, 12)
	if err != nil {
		return nil, err
	}
	return &DossierDetail{Dossier: *d, Sources: sources, RecentJobs: jobs, Briefs: briefs}, nil
}

func (s *Service) SourceLinks(ctx context.Context, dossierID int64) ([]SourceLink, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT ds.source_id, s.path
FROM dossier_sources ds
JOIN sources s ON s.id = ds.source_id
WHERE ds.dossier_id = ?
ORDER BY ds.source_id ASC`, dossierID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SourceLink{}
	for rows.Next() {
		var r SourceLink
		if err := rows.Scan(&r.SourceID, &r.Path); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Service) RecentJobs(ctx context.Context, dossierID int64, limit int) ([]JobSnapshot, error) {
	if limit <= 0 || limit > 200 {
		limit = 40
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT j.id, j.title, j.status, j.target_adapter, j.created_at, j.result_summary, j.last_failure_code
FROM dossier_jobs dj
JOIN jobs j ON j.id = dj.job_id
WHERE dj.dossier_id = ?
ORDER BY dj.linked_at DESC
LIMIT ?`, dossierID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []JobSnapshot{}
	for rows.Next() {
		var r JobSnapshot
		var res sql.NullString
		var fail sql.NullString
		if err := rows.Scan(&r.JobID, &r.Title, &r.Status, &r.TargetAdapter, &r.CreatedAtMs, &res, &fail); err != nil {
			return nil, err
		}
		if res.Valid {
			v := res.String
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

func (s *Service) Briefs(ctx context.Context, dossierID int64, limit int) ([]Brief, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, dossier_id, created_at, summary_markdown, context_json, notes
FROM dossier_briefs
WHERE dossier_id = ?
ORDER BY id DESC LIMIT ?`, dossierID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Brief{}
	for rows.Next() {
		var b Brief
		var contextJSON string
		if err := rows.Scan(&b.ID, &b.DossierID, &b.CreatedAtMs, &b.SummaryMarkdown, &contextJSON, &b.Notes); err != nil {
			return nil, err
		}
		b.Context = json.RawMessage(contextJSON)
		out = append(out, b)
	}
	return out, rows.Err()
}

func (s *Service) AttachJob(ctx context.Context, dossierID int64, jobID string) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO dossier_jobs(dossier_id, job_id, linked_at)
VALUES(?,?,?) ON CONFLICT(dossier_id, job_id) DO NOTHING`, dossierID, jobID, time.Now().UnixMilli())
	return err
}

func (s *Service) AttachPacket(ctx context.Context, dossierID int64, packetID int64) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO dossier_packets(dossier_id, packet_id, linked_at)
VALUES(?,?,?) ON CONFLICT(dossier_id, packet_id) DO NOTHING`, dossierID, packetID, time.Now().UnixMilli())
	return err
}

func (s *Service) GenerateBrief(ctx context.Context, dossierID int64, notes string) (*Brief, error) {
	detail, err := s.Detail(ctx, dossierID)
	if err != nil {
		return nil, err
	}
	var sb strings.Builder
	sb.WriteString("# Dossier Brief\n\n")
	sb.WriteString("Dossier: ")
	sb.WriteString(detail.Dossier.Name)
	sb.WriteString("\n\n")
	if strings.TrimSpace(detail.Dossier.Description) != "" {
		sb.WriteString("## Description\n")
		sb.WriteString(detail.Dossier.Description)
		sb.WriteString("\n\n")
	}
	sb.WriteString("## Source Paths\n")
	if len(detail.Sources) == 0 {
		sb.WriteString("- (none linked)\n")
	} else {
		for _, src := range detail.Sources {
			sb.WriteString("- ")
			sb.WriteString(src.Path)
			sb.WriteString("\n")
		}
	}
	sb.WriteString("\n## Recent Jobs\n")
	if len(detail.RecentJobs) == 0 {
		sb.WriteString("- (none)\n")
	} else {
		for _, j := range detail.RecentJobs {
			sb.WriteString("- ")
			sb.WriteString(j.JobID)
			sb.WriteString(" · ")
			sb.WriteString(j.Status)
			sb.WriteString(" · ")
			sb.WriteString(j.TargetAdapter)
			sb.WriteString("\n")
		}
	}
	contextJSON, _ := json.Marshal(map[string]any{
		"dossierId": dossierID,
		"jobCount":  len(detail.RecentJobs),
		"sourceCount": len(detail.Sources),
	})
	now := time.Now().UnixMilli()
	res, err := s.db.ExecContext(ctx, `
INSERT INTO dossier_briefs(dossier_id, created_at, summary_markdown, context_json, notes)
VALUES(?,?,?,?,?)`, dossierID, now, sb.String(), string(contextJSON), strings.TrimSpace(notes))
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	row := s.db.QueryRowContext(ctx, `SELECT id, dossier_id, created_at, summary_markdown, context_json, notes FROM dossier_briefs WHERE id = ?`, id)
	var b Brief
	var contextRaw string
	if err := row.Scan(&b.ID, &b.DossierID, &b.CreatedAtMs, &b.SummaryMarkdown, &contextRaw, &b.Notes); err != nil {
		return nil, err
	}
	b.Context = json.RawMessage(contextRaw)
	_, _ = s.db.ExecContext(ctx, `UPDATE dossiers SET updated_at = ? WHERE id = ?`, now, dossierID)
	return &b, nil
}

func (s *Service) sourcePaths(ctx context.Context, sourceIDs []int64) ([]string, error) {
	if len(sourceIDs) == 0 {
		return nil, nil
	}
	q := `SELECT path FROM sources WHERE id IN (` + placeholders(len(sourceIDs)) + `) ORDER BY id ASC`
	args := make([]any, 0, len(sourceIDs))
	for _, id := range sourceIDs {
		args = append(args, id)
	}
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func choosePaths(in []string) []string {
	set := map[string]struct{}{}
	out := []string{}
	for _, p := range in {
		trim := strings.TrimSpace(p)
		if trim == "" {
			continue
		}
		if _, ok := set[trim]; ok {
			continue
		}
		set[trim] = struct{}{}
		out = append(out, trim)
	}
	return out
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
