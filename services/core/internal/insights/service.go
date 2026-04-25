package insights

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
	DossierID      *int64          `json:"dossierId"`
	AdapterID      string          `json:"adapterId"`
	TaskType       string          `json:"taskType"`
	Recommendation string          `json:"recommendation"`
	Confidence     float64         `json:"confidence"`
	Reasons        json.RawMessage `json:"reasons"`
	Evidence       json.RawMessage `json:"evidence"`
	AdvisoryLevel  string          `json:"advisoryLevel"`
}

type GenerateRequest struct {
	DossierID *int64 `json:"dossierId"`
}

type Service struct {
	db *sql.DB
}

func New(db *sql.DB) *Service { return &Service{db: db} }

func (s *Service) Generate(ctx context.Context, req GenerateRequest) ([]Record, error) {
	adapterRows, err := s.adapterSignals(ctx, req.DossierID)
	if err != nil {
		return nil, err
	}
	out := []Record{}
	for _, row := range adapterRows {
		reasons := []string{
			fmt.Sprintf("success_rate=%.2f", row.SuccessRate),
			fmt.Sprintf("avg_quality=%.2f", row.AvgQuality),
			fmt.Sprintf("retry_rate=%.2f", row.RetryRate),
			fmt.Sprintf("runs=%d", row.Runs),
		}
		rec := fmt.Sprintf("%s is advisory-preferred for this workload.", row.Adapter)
		conf := clamp((row.SuccessRate*0.55)+(row.AvgQuality/5.0*0.30)+((1-row.RetryRate)*0.15), 0.05, 0.95)

		if row.Runs < 3 {
			rec = fmt.Sprintf("%s has limited evidence; keep operator override on.", row.Adapter)
			conf = clamp(conf*0.6, 0.05, 0.6)
		}
		if row.SuccessRate < 0.5 || row.AvgQuality < 2.8 {
			rec = fmt.Sprintf("%s underperforms in recent evaluations; use with tighter packet scope.", row.Adapter)
			conf = clamp(conf*0.85, 0.05, 0.8)
		}

		r, err := s.insert(ctx, Record{
			DossierID:      req.DossierID,
			AdapterID:      row.Adapter,
			TaskType:       row.TaskType,
			Recommendation: rec,
			Confidence:     conf,
			Reasons:        mustJSON(reasons),
			Evidence:       mustJSON(row),
			AdvisoryLevel:  "advisory",
		})
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}

	retrievalRec, err := s.retrievalSignal(ctx, req.DossierID)
	if err == nil && retrievalRec != nil {
		r, err := s.insert(ctx, *retrievalRec)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}

	reviewRec, err := s.reviewSignal(ctx, req.DossierID)
	if err == nil && reviewRec != nil {
		r, err := s.insert(ctx, *reviewRec)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return out, nil
}

func (s *Service) List(ctx context.Context, limit int, dossierID *int64) ([]Record, error) {
	if limit <= 0 || limit > 300 {
		limit = 120
	}
	query := `
SELECT id, created_at, dossier_id, adapter_id, task_type, recommendation, confidence, reasons_json, evidence_json, advisory_level
FROM routing_insights`
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
		r, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return out, rows.Err()
}

type adapterSignal struct {
	Adapter     string  `json:"adapter"`
	TaskType    string  `json:"taskType"`
	Runs        int     `json:"runs"`
	SuccessRate float64 `json:"successRate"`
	AvgQuality  float64 `json:"avgQuality"`
	RetryRate   float64 `json:"retryRate"`
}

func (s *Service) adapterSignals(ctx context.Context, dossierID *int64) ([]adapterSignal, error) {
	query := `
SELECT j.target_adapter,
       COALESCE(NULLIF(j.requested_action, ''), 'general') AS task_type,
       COUNT(er.id) AS runs,
       AVG(CASE WHEN er.success = 1 THEN 1.0 ELSE 0.0 END) AS success_rate,
       AVG(er.quality_rating) AS avg_quality,
       AVG(CASE WHEN er.retry_recommended = 1 THEN 1.0 ELSE 0.0 END) AS retry_rate
FROM evaluation_records er
JOIN jobs j ON j.id = er.job_id`
	args := []any{}
	if dossierID != nil {
		query += ` WHERE er.dossier_id = ?`
		args = append(args, *dossierID)
	}
	query += ` GROUP BY j.target_adapter, task_type ORDER BY runs DESC`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []adapterSignal{}
	for rows.Next() {
		var r adapterSignal
		if err := rows.Scan(&r.Adapter, &r.TaskType, &r.Runs, &r.SuccessRate, &r.AvgQuality, &r.RetryRate); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Service) retrievalSignal(ctx context.Context, dossierID *int64) (*Record, error) {
	query := `
SELECT
  SUM(CASE WHEN usefulness_label IN ('noisy','not_useful') THEN 1 ELSE 0 END) AS noisy_count,
  SUM(CASE WHEN usefulness_label = 'useful' THEN 1 ELSE 0 END) AS useful_count,
  COUNT(*) AS total
FROM retrieval_results rr
JOIN retrieval_runs r ON r.id = rr.retrieval_run_id`
	args := []any{}
	if dossierID != nil {
		query += ` WHERE r.dossier_id = ?`
		args = append(args, *dossierID)
	}
	var noisy sql.NullInt64
	var useful sql.NullInt64
	var total sql.NullInt64
	if err := s.db.QueryRowContext(ctx, query, args...).Scan(&noisy, &useful, &total); err != nil {
		return nil, err
	}
	if !total.Valid || total.Int64 == 0 {
		return nil, nil
	}
	noisyCount := int(noisy.Int64)
	usefulCount := int(useful.Int64)
	totalCount := int(total.Int64)
	noisyRate := float64(noisyCount) / float64(totalCount)
	usefulRate := float64(usefulCount) / float64(totalCount)

	recommendation := "Hybrid retrieval is stable for this dossier."
	if noisyRate >= 0.35 {
		recommendation = "Retrieved context is noisy; tighten dossier scope or lower keyword weight."
	}
	if usefulRate >= 0.55 {
		recommendation = "Hybrid retrieval shows strong useful-hit ratio; keep hybrid default."
	}
	confidence := clamp((usefulRate*0.7)+((1-noisyRate)*0.3), 0.05, 0.95)

	evidence := map[string]any{
		"noisyCount":  noisyCount,
		"usefulCount": usefulCount,
		"total":       totalCount,
		"noisyRate":   noisyRate,
		"usefulRate":  usefulRate,
	}
	reasons := []string{
		fmt.Sprintf("useful_rate=%.2f", usefulRate),
		fmt.Sprintf("noisy_rate=%.2f", noisyRate),
		fmt.Sprintf("total_results=%d", totalCount),
	}

	return &Record{
		DossierID:      dossierID,
		AdapterID:      "retrieval",
		TaskType:       "context_selection",
		Recommendation: recommendation,
		Confidence:     confidence,
		Reasons:        mustJSON(reasons),
		Evidence:       mustJSON(evidence),
		AdvisoryLevel:  "advisory",
	}, nil
}

func (s *Service) reviewSignal(ctx context.Context, dossierID *int64) (*Record, error) {
	query := `
SELECT
  SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) AS pending_count,
  SUM(CASE WHEN status = 'approved' THEN 1 ELSE 0 END) AS approved_count,
  SUM(CASE WHEN status = 'rejected' THEN 1 ELSE 0 END) AS rejected_count,
  SUM(CASE WHEN status = 'deferred' THEN 1 ELSE 0 END) AS deferred_count,
  COUNT(*) AS total_count
FROM review_records`
	args := []any{}
	if dossierID != nil {
		query += ` WHERE dossier_id = ?`
		args = append(args, *dossierID)
	}
	var pending sql.NullInt64
	var approved sql.NullInt64
	var rejected sql.NullInt64
	var deferred sql.NullInt64
	var total sql.NullInt64
	if err := s.db.QueryRowContext(ctx, query, args...).Scan(&pending, &approved, &rejected, &deferred, &total); err != nil {
		return nil, err
	}
	if !total.Valid || total.Int64 == 0 {
		return nil, nil
	}
	pendingCount := int(pending.Int64)
	approvedCount := int(approved.Int64)
	rejectedCount := int(rejected.Int64)
	deferredCount := int(deferred.Int64)
	totalCount := int(total.Int64)
	rejectedRate := float64(rejectedCount) / float64(totalCount)
	pendingRate := float64(pendingCount) / float64(totalCount)

	recommendation := "Review flow is stable."
	if pendingCount >= 3 {
		recommendation = "Review queue is accumulating; prioritize review workflow before next risky retries."
	}
	if rejectedRate >= 0.30 && rejectedCount >= 2 {
		recommendation = "Rejected outputs are elevated; tighten strategy scope and require conservative approvals."
	}
	if deferredCount >= 3 {
		recommendation = "Many deferred reviews; clarify expected deliverable type in task packets."
	}
	confidence := clamp((1.0-pendingRate)*0.4+(1.0-rejectedRate)*0.6, 0.05, 0.95)

	evidence := map[string]any{
		"pendingCount":  pendingCount,
		"approvedCount": approvedCount,
		"rejectedCount": rejectedCount,
		"deferredCount": deferredCount,
		"total":         totalCount,
		"pendingRate":   pendingRate,
		"rejectedRate":  rejectedRate,
	}
	reasons := []string{
		fmt.Sprintf("pending_count=%d", pendingCount),
		fmt.Sprintf("approved_count=%d", approvedCount),
		fmt.Sprintf("rejected_count=%d", rejectedCount),
		fmt.Sprintf("deferred_count=%d", deferredCount),
	}

	return &Record{
		DossierID:      dossierID,
		AdapterID:      "review",
		TaskType:       "review_workflow",
		Recommendation: recommendation,
		Confidence:     confidence,
		Reasons:        mustJSON(reasons),
		Evidence:       mustJSON(evidence),
		AdvisoryLevel:  "advisory",
	}, nil
}

func (s *Service) insert(ctx context.Context, rec Record) (*Record, error) {
	now := time.Now().UnixMilli()
	reasons := rec.Reasons
	if len(reasons) == 0 {
		reasons = mustJSON([]string{})
	}
	evidence := rec.Evidence
	if len(evidence) == 0 {
		evidence = mustJSON(map[string]any{})
	}
	res, err := s.db.ExecContext(ctx, `
INSERT INTO routing_insights(
  created_at, dossier_id, adapter_id, task_type, recommendation, confidence, reasons_json, evidence_json, advisory_level
) VALUES(?,?,?,?,?,?,?,?,?)`,
		now,
		rec.DossierID,
		strings.TrimSpace(rec.AdapterID),
		strings.TrimSpace(rec.TaskType),
		strings.TrimSpace(rec.Recommendation),
		rec.Confidence,
		string(reasons),
		string(evidence),
		nonEmpty(rec.AdvisoryLevel, "advisory"),
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	row := s.db.QueryRowContext(ctx, `
SELECT id, created_at, dossier_id, adapter_id, task_type, recommendation, confidence, reasons_json, evidence_json, advisory_level
FROM routing_insights WHERE id = ?`, id)
	return scan(row)
}

func scan(scanner interface{ Scan(dest ...any) error }) (*Record, error) {
	var r Record
	var dossierID sql.NullInt64
	var reasons string
	var evidence string
	if err := scanner.Scan(
		&r.ID, &r.CreatedAtMs, &dossierID, &r.AdapterID, &r.TaskType, &r.Recommendation, &r.Confidence, &reasons, &evidence, &r.AdvisoryLevel,
	); err != nil {
		return nil, err
	}
	if dossierID.Valid {
		v := dossierID.Int64
		r.DossierID = &v
	}
	r.Reasons = json.RawMessage(reasons)
	r.Evidence = json.RawMessage(evidence)
	return &r, nil
}

func mustJSON(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

func nonEmpty(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return strings.TrimSpace(v)
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
