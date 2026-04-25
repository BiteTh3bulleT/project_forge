package failurepatterns

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Pattern struct {
	ID             int64           `json:"id"`
	CreatedAtMs    int64           `json:"createdAtMs"`
	DossierID      *int64          `json:"dossierId"`
	TargetAdapter  string          `json:"targetAdapter"`
	StrategyID     *string         `json:"strategyId"`
	RetrievalMode  string          `json:"retrievalMode"`
	PacketStyle    string          `json:"packetStyle"`
	FailureCode    string          `json:"failureCode"`
	FailureCount   int             `json:"failureCount"`
	Recommendation string          `json:"recommendation"`
	Evidence       json.RawMessage `json:"evidence"`
}

type AnalyzeRequest struct {
	DossierID *int64 `json:"dossierId"`
	Lookback  int    `json:"lookback"`
}

type Service struct {
	db *sql.DB
}

func New(db *sql.DB) *Service { return &Service{db: db} }

func (s *Service) Analyze(ctx context.Context, req AnalyzeRequest) ([]Pattern, error) {
	lookback := req.Lookback
	if lookback <= 0 || lookback > 1000 {
		lookback = 250
	}
	query := `
SELECT
  j.target_adapter,
  COALESCE(json_extract(j.metadata_json, '$.templateId'), '') AS strategy_id,
  COALESCE(json_extract(j.metadata_json, '$.requestPayload.retrievalMode'), 'hybrid') AS retrieval_mode,
  COALESCE(j.last_failure_code, 'execution_failure') AS failure_code,
  CASE
    WHEN IFNULL(json_array_length(p.retrieved_context_json), 0) >= 16 THEN 'oversized'
    WHEN IFNULL(json_array_length(p.retrieved_context_json), 0) <= 4 THEN 'minimal'
    ELSE 'balanced'
  END AS packet_style,
  COUNT(*) AS fail_count
FROM jobs j
LEFT JOIN task_packets p ON p.id = j.task_packet_id
WHERE j.status = 'failed'`
	args := []any{}
	if req.DossierID != nil {
		query += ` AND EXISTS (SELECT 1 FROM dossier_jobs dj WHERE dj.job_id = j.id AND dj.dossier_id = ?)`
		args = append(args, *req.DossierID)
	}
	query += `
GROUP BY j.target_adapter, strategy_id, retrieval_mode, failure_code, packet_style
ORDER BY fail_count DESC
LIMIT ?`
	args = append(args, lookback)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	patterns := []Pattern{}
	now := time.Now().UnixMilli()
	for rows.Next() {
		var adapter string
		var strategyID string
		var retrievalMode string
		var failureCode string
		var packetStyle string
		var failCount int
		if err := rows.Scan(&adapter, &strategyID, &retrievalMode, &failureCode, &packetStyle, &failCount); err != nil {
			return nil, err
		}
		recommendation := recommend(adapter, retrievalMode, packetStyle, failureCode, failCount)
		evidence := map[string]any{
			"targetAdapter": adapter,
			"retrievalMode": retrievalMode,
			"packetStyle":   packetStyle,
			"failureCode":   failureCode,
			"failureCount":  failCount,
		}
		evJSON, _ := json.Marshal(evidence)
		res, err := s.db.ExecContext(ctx, `
INSERT INTO failure_pattern_snapshots(
  created_at, dossier_id, target_adapter, strategy_id, retrieval_mode, packet_style, failure_code, failure_count, recommendation, evidence_json
) VALUES(?,?,?,?,?,?,?,?,?,?)`,
			now, req.DossierID, adapter, nullableString(strategyID), retrievalMode, packetStyle, failureCode, failCount, recommendation, string(evJSON),
		)
		if err != nil {
			return nil, err
		}
		id, _ := res.LastInsertId()
		p := Pattern{
			ID:             id,
			CreatedAtMs:    now,
			DossierID:      req.DossierID,
			TargetAdapter:  adapter,
			RetrievalMode:  retrievalMode,
			PacketStyle:    packetStyle,
			FailureCode:    failureCode,
			FailureCount:   failCount,
			Recommendation: recommendation,
			Evidence:       evJSON,
		}
		if strings.TrimSpace(strategyID) != "" {
			v := strategyID
			p.StrategyID = &v
		}
		patterns = append(patterns, p)
	}
	return patterns, rows.Err()
}

func (s *Service) List(ctx context.Context, limit int, dossierID *int64) ([]Pattern, error) {
	if limit <= 0 || limit > 400 {
		limit = 120
	}
	query := `
SELECT id, created_at, dossier_id, target_adapter, strategy_id, retrieval_mode, packet_style, failure_code, failure_count, recommendation, evidence_json
FROM failure_pattern_snapshots`
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
	out := []Pattern{}
	for rows.Next() {
		p, err := scanPattern(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

func scanPattern(scanner interface{ Scan(dest ...any) error }) (*Pattern, error) {
	var p Pattern
	var dossierID sql.NullInt64
	var strategyID sql.NullString
	var evidence string
	if err := scanner.Scan(
		&p.ID, &p.CreatedAtMs, &dossierID, &p.TargetAdapter, &strategyID, &p.RetrievalMode, &p.PacketStyle, &p.FailureCode, &p.FailureCount, &p.Recommendation, &evidence,
	); err != nil {
		return nil, err
	}
	if dossierID.Valid {
		v := dossierID.Int64
		p.DossierID = &v
	}
	if strategyID.Valid {
		v := strategyID.String
		p.StrategyID = &v
	}
	p.Evidence = json.RawMessage(evidence)
	return &p, nil
}

func recommend(adapter, retrievalMode, packetStyle, failureCode string, count int) string {
	switch {
	case packetStyle == "oversized":
		return "Trim packet context and prioritize high-signal files before retry."
	case packetStyle == "minimal":
		return "Expand packet context and consider hybrid retrieval."
	case failureCode == "adapter_timeout":
		return fmt.Sprintf("Adapter %s timed out repeatedly; tighten scope or switch adapter.", adapter)
	case failureCode == "adapter_unavailable":
		return fmt.Sprintf("Adapter %s unavailable; require fallback adapter in strategy.", adapter)
	case count >= 3 && retrievalMode == "keyword":
		return "Keyword-only failures repeating; switch strategy to hybrid retrieval."
	default:
		return "Require review before retry and adjust strategy scope."
	}
}

func nullableString(v string) any {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return v
}
