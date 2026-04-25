package evaluations

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type Record struct {
	ID                    int64  `json:"id"`
	CreatedAtMs           int64  `json:"createdAtMs"`
	JobID                 string `json:"jobId"`
	DossierID             *int64 `json:"dossierId"`
	Success               bool   `json:"success"`
	QualityRating         int    `json:"qualityRating"`
	UsefulnessRating      int    `json:"usefulnessRating"`
	CorrectnessConfidence int    `json:"correctnessConfidence"`
	PacketQualityRating   int    `json:"packetQualityRating"`
	AdapterSuitability    int    `json:"adapterSuitability"`
	RetryRecommended      bool   `json:"retryRecommended"`
	InfluenceRouting      bool   `json:"influenceRouting"`
	Notes                 string `json:"notes"`
	Scorer                string `json:"scorer"`
}

type SaveRequest struct {
	JobID                 string `json:"jobId"`
	DossierID             *int64 `json:"dossierId"`
	Success               bool   `json:"success"`
	QualityRating         int    `json:"qualityRating"`
	UsefulnessRating      int    `json:"usefulnessRating"`
	CorrectnessConfidence int    `json:"correctnessConfidence"`
	PacketQualityRating   int    `json:"packetQualityRating"`
	AdapterSuitability    int    `json:"adapterSuitability"`
	RetryRecommended      bool   `json:"retryRecommended"`
	InfluenceRouting      bool   `json:"influenceRouting"`
	Notes                 string `json:"notes"`
	Scorer                string `json:"scorer"`
}

type AdapterMetric struct {
	Adapter               string  `json:"adapter"`
	Runs                  int     `json:"runs"`
	SuccessRate           float64 `json:"successRate"`
	AvgQuality            float64 `json:"avgQuality"`
	AvgUsefulness         float64 `json:"avgUsefulness"`
	AvgAdapterSuitability float64 `json:"avgAdapterSuitability"`
	RetryRate             float64 `json:"retryRate"`
}

type Service struct {
	db *sql.DB
}

func New(db *sql.DB) *Service { return &Service{db: db} }

func (s *Service) Save(ctx context.Context, req SaveRequest) (*Record, error) {
	if req.JobID == "" {
		return nil, fmt.Errorf("jobId is required")
	}
	if !within(req.QualityRating) || !within(req.UsefulnessRating) || !within(req.CorrectnessConfidence) || !within(req.PacketQualityRating) || !within(req.AdapterSuitability) {
		return nil, fmt.Errorf("ratings must be between 1 and 5")
	}
	now := time.Now().UnixMilli()
	res, err := s.db.ExecContext(ctx, `
INSERT INTO evaluation_records(
  created_at, job_id, dossier_id, success,
  quality_rating, usefulness_rating, correctness_confidence,
  packet_quality_rating, adapter_suitability,
  retry_recommended, influence_routing, notes, scorer
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		now,
		req.JobID,
		req.DossierID,
		boolToInt(req.Success),
		req.QualityRating,
		req.UsefulnessRating,
		req.CorrectnessConfidence,
		req.PacketQualityRating,
		req.AdapterSuitability,
		boolToInt(req.RetryRecommended),
		boolToInt(req.InfluenceRouting),
		req.Notes,
		nonEmpty(req.Scorer, "operator"),
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.Get(ctx, id)
}

func (s *Service) Get(ctx context.Context, id int64) (*Record, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, created_at, job_id, dossier_id, success,
       quality_rating, usefulness_rating, correctness_confidence,
       packet_quality_rating, adapter_suitability,
       retry_recommended, influence_routing, notes, scorer
FROM evaluation_records WHERE id = ?`, id)
	return scanRecord(row)
}

func (s *Service) LatestByJob(ctx context.Context, jobID string) (*Record, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, created_at, job_id, dossier_id, success,
       quality_rating, usefulness_rating, correctness_confidence,
       packet_quality_rating, adapter_suitability,
       retry_recommended, influence_routing, notes, scorer
FROM evaluation_records WHERE job_id = ? ORDER BY id DESC LIMIT 1`, jobID)
	r, err := scanRecord(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return r, err
}

func (s *Service) List(ctx context.Context, limit int, dossierID *int64) ([]Record, error) {
	if limit <= 0 || limit > 300 {
		limit = 120
	}
	query := `
SELECT id, created_at, job_id, dossier_id, success,
       quality_rating, usefulness_rating, correctness_confidence,
       packet_quality_rating, adapter_suitability,
       retry_recommended, influence_routing, notes, scorer
FROM evaluation_records`
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
		r, err := scanRecord(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return out, rows.Err()
}

func (s *Service) AdapterMetrics(ctx context.Context, dossierID *int64) ([]AdapterMetric, error) {
	query := `
SELECT j.target_adapter,
       COUNT(er.id) AS runs,
       AVG(CASE WHEN er.success = 1 THEN 1.0 ELSE 0.0 END) AS success_rate,
       AVG(er.quality_rating) AS avg_quality,
       AVG(er.usefulness_rating) AS avg_usefulness,
       AVG(er.adapter_suitability) AS avg_adapter_suitability,
       AVG(CASE WHEN er.retry_recommended = 1 THEN 1.0 ELSE 0.0 END) AS retry_rate
FROM evaluation_records er
JOIN jobs j ON j.id = er.job_id`
	args := []any{}
	if dossierID != nil {
		query += ` WHERE er.dossier_id = ?`
		args = append(args, *dossierID)
	}
	query += ` GROUP BY j.target_adapter ORDER BY runs DESC`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AdapterMetric{}
	for rows.Next() {
		var m AdapterMetric
		if err := rows.Scan(&m.Adapter, &m.Runs, &m.SuccessRate, &m.AvgQuality, &m.AvgUsefulness, &m.AvgAdapterSuitability, &m.RetryRate); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func scanRecord(scanner interface{ Scan(dest ...any) error }) (*Record, error) {
	var r Record
	var dossierID sql.NullInt64
	var success, retry, influence int
	if err := scanner.Scan(
		&r.ID,
		&r.CreatedAtMs,
		&r.JobID,
		&dossierID,
		&success,
		&r.QualityRating,
		&r.UsefulnessRating,
		&r.CorrectnessConfidence,
		&r.PacketQualityRating,
		&r.AdapterSuitability,
		&retry,
		&influence,
		&r.Notes,
		&r.Scorer,
	); err != nil {
		return nil, err
	}
	if dossierID.Valid {
		v := dossierID.Int64
		r.DossierID = &v
	}
	r.Success = success == 1
	r.RetryRecommended = retry == 1
	r.InfluenceRouting = influence == 1
	return &r, nil
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func within(v int) bool { return v >= 1 && v <= 5 }

func nonEmpty(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return strings.TrimSpace(v)
}
