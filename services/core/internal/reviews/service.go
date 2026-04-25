package reviews

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Status string

const (
	StatusPending  Status = "pending"
	StatusApproved Status = "approved"
	StatusRejected Status = "rejected"
	StatusDeferred Status = "deferred"
)

type Record struct {
	ID          int64           `json:"id"`
	CreatedAtMs int64           `json:"createdAtMs"`
	UpdatedAtMs int64           `json:"updatedAtMs"`
	TargetType  string          `json:"targetType"`
	TargetID    string          `json:"targetId"`
	DossierID   *int64          `json:"dossierId"`
	Status      Status          `json:"status"`
	Summary     string          `json:"summary"`
	Notes       string          `json:"notes"`
	Annotations json.RawMessage `json:"annotations"`
	Reviewer    string          `json:"reviewer"`
}

type CreateRequest struct {
	TargetType  string   `json:"targetType"`
	TargetID    string   `json:"targetId"`
	DossierID   *int64   `json:"dossierId"`
	Status      Status   `json:"status"`
	Summary     string   `json:"summary"`
	Notes       string   `json:"notes"`
	Annotations []string `json:"annotations"`
	Reviewer    string   `json:"reviewer"`
}

type UpdateRequest struct {
	Status      *Status   `json:"status"`
	Summary     *string   `json:"summary"`
	Notes       *string   `json:"notes"`
	Annotations *[]string `json:"annotations"`
	Reviewer    *string   `json:"reviewer"`
}

type Service struct {
	db *sql.DB
}

func New(db *sql.DB) *Service { return &Service{db: db} }

func (s *Service) Create(ctx context.Context, req CreateRequest) (*Record, error) {
	targetType := strings.TrimSpace(req.TargetType)
	targetID := strings.TrimSpace(req.TargetID)
	if targetType == "" || targetID == "" {
		return nil, fmt.Errorf("targetType and targetId are required")
	}
	status := req.Status
	if status == "" {
		status = StatusPending
	}
	notes := strings.TrimSpace(req.Notes)
	summary := strings.TrimSpace(req.Summary)
	annotations, _ := json.Marshal(nonNilStrings(req.Annotations))
	now := time.Now().UnixMilli()
	res, err := s.db.ExecContext(ctx, `
INSERT INTO review_records(created_at, updated_at, target_type, target_id, dossier_id, status, summary, notes, annotations_json, reviewer)
VALUES(?,?,?,?,?,?,?,?,?,?)`,
		now,
		now,
		targetType,
		targetID,
		req.DossierID,
		string(status),
		summary,
		notes,
		string(annotations),
		nonEmpty(req.Reviewer, "operator"),
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.Get(ctx, id)
}

func (s *Service) Get(ctx context.Context, id int64) (*Record, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, created_at, updated_at, target_type, target_id, dossier_id, status, summary, notes, annotations_json, reviewer
FROM review_records WHERE id = ?`, id)
	return scanRecord(row)
}

func (s *Service) List(ctx context.Context, status string, limit int) ([]Record, error) {
	if limit <= 0 || limit > 400 {
		limit = 120
	}
	query := `
SELECT id, created_at, updated_at, target_type, target_id, dossier_id, status, summary, notes, annotations_json, reviewer
FROM review_records`
	args := []any{}
	status = strings.TrimSpace(strings.ToLower(status))
	if status != "" {
		query += ` WHERE status = ?`
		args = append(args, status)
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

func (s *Service) Update(ctx context.Context, id int64, req UpdateRequest) (*Record, error) {
	rec, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Status != nil && *req.Status != "" {
		rec.Status = *req.Status
	}
	if req.Summary != nil {
		rec.Summary = strings.TrimSpace(*req.Summary)
	}
	if req.Notes != nil {
		rec.Notes = strings.TrimSpace(*req.Notes)
	}
	if req.Annotations != nil {
		raw, _ := json.Marshal(nonNilStrings(*req.Annotations))
		rec.Annotations = json.RawMessage(raw)
	}
	if req.Reviewer != nil {
		rec.Reviewer = nonEmpty(*req.Reviewer, rec.Reviewer)
	}
	_, err = s.db.ExecContext(ctx, `
UPDATE review_records
SET updated_at = ?, status = ?, summary = ?, notes = ?, annotations_json = ?, reviewer = ?
WHERE id = ?`,
		time.Now().UnixMilli(),
		string(rec.Status),
		rec.Summary,
		rec.Notes,
		string(rec.Annotations),
		rec.Reviewer,
		id,
	)
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

func scanRecord(scanner interface{ Scan(dest ...any) error }) (*Record, error) {
	var r Record
	var dossierID sql.NullInt64
	var annotations string
	if err := scanner.Scan(
		&r.ID, &r.CreatedAtMs, &r.UpdatedAtMs, &r.TargetType, &r.TargetID, &dossierID, &r.Status, &r.Summary, &r.Notes, &annotations, &r.Reviewer,
	); err != nil {
		return nil, err
	}
	if dossierID.Valid {
		v := dossierID.Int64
		r.DossierID = &v
	}
	r.Annotations = json.RawMessage(annotations)
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

func nonEmpty(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return strings.TrimSpace(v)
}
