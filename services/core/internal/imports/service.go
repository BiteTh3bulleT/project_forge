package imports

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Record struct {
	ID             int64           `json:"id"`
	CreatedAtMs    int64           `json:"createdAtMs"`
	AdapterID      string          `json:"adapterId"`
	ExternalRunID  string          `json:"externalRunId"`
	OriginJobID    *string         `json:"originJobId"`
	OriginPacketID *int64          `json:"originPacketId"`
	DossierID      *int64          `json:"dossierId"`
	Summary        string          `json:"summary"`
	OutputRefs     json.RawMessage `json:"outputRefs"`
	DiffSummary    string          `json:"diffSummary"`
	ExecutionNotes string          `json:"executionNotes"`
	Evaluation     json.RawMessage `json:"evaluation"`
}

type CreateRequest struct {
	AdapterID      string         `json:"adapterId"`
	ExternalRunID  string         `json:"externalRunId"`
	OriginJobID    *string        `json:"originJobId"`
	OriginPacketID *int64         `json:"originPacketId"`
	DossierID      *int64         `json:"dossierId"`
	Summary        string         `json:"summary"`
	OutputRefs     []string       `json:"outputRefs"`
	DiffSummary    string         `json:"diffSummary"`
	ExecutionNotes string         `json:"executionNotes"`
	Evaluation     map[string]any `json:"evaluation"`
}

type Service struct {
	db *sql.DB
}

func New(db *sql.DB) *Service { return &Service{db: db} }

func (s *Service) Create(ctx context.Context, req CreateRequest) (*Record, error) {
	adapterID := strings.TrimSpace(req.AdapterID)
	if adapterID == "" {
		return nil, fmt.Errorf("adapterId is required")
	}
	summary := strings.TrimSpace(req.Summary)
	if summary == "" {
		return nil, fmt.Errorf("summary is required")
	}
	outputRefs, _ := json.Marshal(nonNilStringSlice(req.OutputRefs))
	evaluation, _ := json.Marshal(nonNilMap(req.Evaluation))
	now := time.Now().UnixMilli()

	res, err := s.db.ExecContext(ctx, `
INSERT INTO imported_executions(
  created_at, adapter_id, external_run_id,
  origin_job_id, origin_packet_id, dossier_id,
  summary, output_refs_json, diff_summary, execution_notes, evaluation_json
) VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
		now,
		adapterID,
		strings.TrimSpace(req.ExternalRunID),
		req.OriginJobID,
		req.OriginPacketID,
		req.DossierID,
		summary,
		string(outputRefs),
		strings.TrimSpace(req.DiffSummary),
		strings.TrimSpace(req.ExecutionNotes),
		string(evaluation),
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.Get(ctx, id)
}

func (s *Service) Get(ctx context.Context, id int64) (*Record, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, created_at, adapter_id, external_run_id,
       origin_job_id, origin_packet_id, dossier_id,
       summary, output_refs_json, diff_summary, execution_notes, evaluation_json
FROM imported_executions WHERE id = ?`, id)
	var r Record
	var originJob sql.NullString
	var originPacket sql.NullInt64
	var dossier sql.NullInt64
	var outputRefs string
	var evaluation string
	if err := row.Scan(
		&r.ID, &r.CreatedAtMs, &r.AdapterID, &r.ExternalRunID,
		&originJob, &originPacket, &dossier,
		&r.Summary, &outputRefs, &r.DiffSummary, &r.ExecutionNotes, &evaluation,
	); err != nil {
		return nil, err
	}
	if originJob.Valid {
		v := originJob.String
		r.OriginJobID = &v
	}
	if originPacket.Valid {
		v := originPacket.Int64
		r.OriginPacketID = &v
	}
	if dossier.Valid {
		v := dossier.Int64
		r.DossierID = &v
	}
	r.OutputRefs = json.RawMessage(outputRefs)
	r.Evaluation = json.RawMessage(evaluation)
	return &r, nil
}

func (s *Service) List(ctx context.Context, limit int, dossierID *int64) ([]Record, error) {
	if limit <= 0 || limit > 300 {
		limit = 100
	}
	query := `
SELECT id, created_at, adapter_id, external_run_id,
       origin_job_id, origin_packet_id, dossier_id,
       summary, output_refs_json, diff_summary, execution_notes, evaluation_json
FROM imported_executions`
	args := []any{}
	if dossierID != nil {
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
	out := []Record{}
	for rows.Next() {
		var r Record
		var originJob sql.NullString
		var originPacket sql.NullInt64
		var dossier sql.NullInt64
		var outputRefs string
		var evaluation string
		if err := rows.Scan(
			&r.ID, &r.CreatedAtMs, &r.AdapterID, &r.ExternalRunID,
			&originJob, &originPacket, &dossier,
			&r.Summary, &outputRefs, &r.DiffSummary, &r.ExecutionNotes, &evaluation,
		); err != nil {
			return nil, err
		}
		if originJob.Valid {
			v := originJob.String
			r.OriginJobID = &v
		}
		if originPacket.Valid {
			v := originPacket.Int64
			r.OriginPacketID = &v
		}
		if dossier.Valid {
			v := dossier.Int64
			r.DossierID = &v
		}
		r.OutputRefs = json.RawMessage(outputRefs)
		r.Evaluation = json.RawMessage(evaluation)
		out = append(out, r)
	}
	return out, rows.Err()
}

func nonNilMap(v map[string]any) map[string]any {
	if v == nil {
		return map[string]any{}
	}
	return v
}

func nonNilStringSlice(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}
