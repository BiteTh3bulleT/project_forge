package packetopt

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Guidance struct {
	ID              int64           `json:"id"`
	CreatedAtMs     int64           `json:"createdAtMs"`
	PacketID        *int64          `json:"packetId"`
	JobID           *string         `json:"jobId"`
	DossierID       *int64          `json:"dossierId"`
	GuidanceScore   float64         `json:"guidanceScore"`
	Issues          json.RawMessage `json:"issues"`
	Recommendations json.RawMessage `json:"recommendations"`
	Evidence        json.RawMessage `json:"evidence"`
}

type AnalyzeRequest struct {
	PacketID  int64  `json:"packetId"`
	JobID     *string `json:"jobId"`
	DossierID *int64 `json:"dossierId"`
}

type Service struct {
	db *sql.DB
}

func New(db *sql.DB) *Service { return &Service{db: db} }

func (s *Service) AnalyzePacket(ctx context.Context, req AnalyzeRequest) (*Guidance, error) {
	if req.PacketID <= 0 {
		return nil, fmt.Errorf("packetId is required")
	}
	var retrievedRaw string
	var riskClass string
	if err := s.db.QueryRowContext(ctx, `
SELECT retrieved_context_json, risk_class
FROM task_packets
WHERE id = ?`, req.PacketID).Scan(&retrievedRaw, &riskClass); err != nil {
		return nil, err
	}
	var retrieved []map[string]any
	_ = json.Unmarshal([]byte(retrievedRaw), &retrieved)
	contextCount := len(retrieved)
	issues := []string{}
	reco := []string{}
	score := 1.0
	if contextCount > 16 {
		issues = append(issues, "packet_oversized")
		reco = append(reco, "trim low-signal chunks and reduce packet target size")
		score -= 0.25
	}
	if contextCount < 4 {
		issues = append(issues, "packet_under_scoped")
		reco = append(reco, "expand retrieval scope or switch to hybrid mode")
		score -= 0.2
	}
	if strings.TrimSpace(riskClass) == "write_files" && contextCount < 6 {
		issues = append(issues, "insufficient_context_for_write_intent")
		reco = append(reco, "expand high-signal context before write-intent execution")
		score -= 0.2
	}

	noiseRate := s.estimateNoiseRate(ctx, req.PacketID)
	if noiseRate >= 0.35 {
		issues = append(issues, "likely_noise")
		reco = append(reco, "tighten retrieval filters and prefer dossier high-value files")
		score -= 0.2
	}
	if score < 0 {
		score = 0
	}

	evidence := map[string]any{
		"contextCount": contextCount,
		"noiseRate":    noiseRate,
		"riskClass":    riskClass,
	}
	issuesJSON, _ := json.Marshal(issues)
	recoJSON, _ := json.Marshal(reco)
	evidenceJSON, _ := json.Marshal(evidence)
	now := time.Now().UnixMilli()
	res, err := s.db.ExecContext(ctx, `
INSERT INTO packet_guidance_records(created_at, packet_id, job_id, dossier_id, guidance_score, issues_json, recommendations_json, evidence_json)
VALUES(?,?,?,?,?,?,?,?)`,
		now, req.PacketID, req.JobID, req.DossierID, score, string(issuesJSON), string(recoJSON), string(evidenceJSON),
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.Get(ctx, id)
}

func (s *Service) List(ctx context.Context, limit int, packetID *int64) ([]Guidance, error) {
	if limit <= 0 || limit > 300 {
		limit = 120
	}
	query := `
SELECT id, created_at, packet_id, job_id, dossier_id, guidance_score, issues_json, recommendations_json, evidence_json
FROM packet_guidance_records`
	args := []any{}
	if packetID != nil {
		query += ` WHERE packet_id = ?`
		args = append(args, *packetID)
	}
	query += ` ORDER BY id DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Guidance{}
	for rows.Next() {
		g, err := scanGuidance(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *g)
	}
	return out, rows.Err()
}

func (s *Service) Get(ctx context.Context, id int64) (*Guidance, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, created_at, packet_id, job_id, dossier_id, guidance_score, issues_json, recommendations_json, evidence_json
FROM packet_guidance_records WHERE id = ?`, id)
	return scanGuidance(row)
}

func (s *Service) estimateNoiseRate(ctx context.Context, packetID int64) float64 {
	var noisy sql.NullInt64
	var total sql.NullInt64
	_ = s.db.QueryRowContext(ctx, `
SELECT
  SUM(CASE WHEN rr.usefulness_label IN ('noisy','not_useful') THEN 1 ELSE 0 END) AS noisy_count,
  COUNT(rr.id) AS total_count
FROM packet_retrieval_runs pr
JOIN retrieval_results rr ON rr.retrieval_run_id = pr.retrieval_run_id
WHERE pr.packet_id = ?`, packetID).Scan(&noisy, &total)
	if !total.Valid || total.Int64 == 0 {
		return 0
	}
	return float64(noisy.Int64) / float64(total.Int64)
}

func scanGuidance(scanner interface{ Scan(dest ...any) error }) (*Guidance, error) {
	var g Guidance
	var packetID sql.NullInt64
	var jobID sql.NullString
	var dossierID sql.NullInt64
	var issues string
	var reco string
	var evidence string
	if err := scanner.Scan(
		&g.ID, &g.CreatedAtMs, &packetID, &jobID, &dossierID, &g.GuidanceScore, &issues, &reco, &evidence,
	); err != nil {
		return nil, err
	}
	if packetID.Valid {
		v := packetID.Int64
		g.PacketID = &v
	}
	if jobID.Valid {
		v := jobID.String
		g.JobID = &v
	}
	if dossierID.Valid {
		v := dossierID.Int64
		g.DossierID = &v
	}
	g.Issues = json.RawMessage(issues)
	g.Recommendations = json.RawMessage(reco)
	g.Evidence = json.RawMessage(evidence)
	return &g, nil
}
