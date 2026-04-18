package dashboard

import (
	"context"
	"database/sql"
	"encoding/json"
)

type JobLite struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Status       string `json:"status"`
	TargetAdapter string `json:"targetAdapter"`
	CreatedAtMs  int64  `json:"createdAtMs"`
}

type ImportLite struct {
	ID          int64  `json:"id"`
	AdapterID   string `json:"adapterId"`
	Summary     string `json:"summary"`
	CreatedAtMs int64  `json:"createdAtMs"`
}

type ReviewLite struct {
	ID          int64  `json:"id"`
	TargetType  string `json:"targetType"`
	TargetID    string `json:"targetId"`
	Status      string `json:"status"`
	Summary     string `json:"summary"`
	UpdatedAtMs int64  `json:"updatedAtMs"`
}

type AutomationLite struct {
	ID          int64  `json:"id"`
	RuleID      *int64 `json:"ruleId"`
	Status      string `json:"status"`
	Message     string `json:"message"`
	CreatedAtMs int64  `json:"createdAtMs"`
}

type RecommendationLite struct {
	ID          int64           `json:"id"`
	TaskType    string          `json:"taskType"`
	Adapter     string          `json:"adapter"`
	Confidence  float64         `json:"confidence"`
	Reasons     json.RawMessage `json:"reasons"`
	CreatedAtMs int64           `json:"createdAtMs"`
}

type Summary struct {
	ActiveJobs             []JobLite             `json:"activeJobs"`
	ApprovalsPending       int                   `json:"approvalsPending"`
	ReviewsPending         int                   `json:"reviewsPending"`
	RecentFailures         []JobLite             `json:"recentFailures"`
	RecentImports          []ImportLite          `json:"recentImports"`
	DossierHealth          []map[string]any      `json:"dossierHealth"`
	AutomationActivity     []AutomationLite      `json:"automationActivity"`
	RoutingRecommendations []RecommendationLite  `json:"routingRecommendations"`
	SystemStatus           map[string]any        `json:"systemStatus"`
}

type Service struct {
	db *sql.DB
}

func New(db *sql.DB) *Service { return &Service{db: db} }

func (s *Service) Summary(ctx context.Context) (*Summary, error) {
	activeJobs, _ := s.jobs(ctx, `WHERE status IN ('queued','preparing','awaiting_approval','running')`, 20)
	recentFailures, _ := s.jobs(ctx, `WHERE status = 'failed'`, 20)
	approvalsPending := s.scalarInt(ctx, `SELECT COUNT(*) FROM approval_requests WHERE status = 'pending'`)
	reviewsPending := s.scalarInt(ctx, `SELECT COUNT(*) FROM review_records WHERE status = 'pending'`)
	recentImports, _ := s.imports(ctx, 12)
	dossierHealth, _ := s.dossierHealth(ctx, 20)
	automation, _ := s.automationActivity(ctx, 20)
	recommendations, _ := s.recommendations(ctx, 20)
	systemStatus := map[string]any{
		"activeJobs":       len(activeJobs),
		"approvalsPending": approvalsPending,
		"reviewsPending":   reviewsPending,
		"dossierCount":     s.scalarInt(ctx, `SELECT COUNT(*) FROM dossiers`),
		"strategyCount":    s.scalarInt(ctx, `SELECT COUNT(*) FROM execution_strategies WHERE enabled = 1`),
		"automationRules":  s.scalarInt(ctx, `SELECT COUNT(*) FROM automation_rules WHERE enabled = 1`),
	}

	return &Summary{
		ActiveJobs:             activeJobs,
		ApprovalsPending:       approvalsPending,
		ReviewsPending:         reviewsPending,
		RecentFailures:         recentFailures,
		RecentImports:          recentImports,
		DossierHealth:          dossierHealth,
		AutomationActivity:     automation,
		RoutingRecommendations: recommendations,
		SystemStatus:           systemStatus,
	}, nil
}

func (s *Service) jobs(ctx context.Context, where string, limit int) ([]JobLite, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, title, status, target_adapter, created_at
FROM jobs `+where+`
ORDER BY created_at DESC
LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []JobLite{}
	for rows.Next() {
		var r JobLite
		if err := rows.Scan(&r.ID, &r.Title, &r.Status, &r.TargetAdapter, &r.CreatedAtMs); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Service) imports(ctx context.Context, limit int) ([]ImportLite, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, adapter_id, summary, created_at
FROM imported_executions
ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ImportLite{}
	for rows.Next() {
		var r ImportLite
		if err := rows.Scan(&r.ID, &r.AdapterID, &r.Summary, &r.CreatedAtMs); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Service) dossierHealth(ctx context.Context, limit int) ([]map[string]any, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT
  d.id,
  d.name,
  IFNULL((SELECT COUNT(*) FROM dossier_jobs dj JOIN jobs j ON j.id = dj.job_id WHERE dj.dossier_id = d.id), 0) AS job_count,
  IFNULL((SELECT COUNT(*) FROM dossier_jobs dj JOIN jobs j ON j.id = dj.job_id WHERE dj.dossier_id = d.id AND j.status = 'failed'), 0) AS fail_count,
  IFNULL((SELECT COUNT(*) FROM review_records rr WHERE rr.dossier_id = d.id AND rr.status = 'pending'), 0) AS review_pending
FROM dossiers d
ORDER BY d.updated_at DESC
LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id int64
		var name string
		var jobs int
		var fails int
		var reviews int
		if err := rows.Scan(&id, &name, &jobs, &fails, &reviews); err != nil {
			return nil, err
		}
		health := "stable"
		if fails >= 3 {
			health = "attention"
		}
		if reviews > 0 {
			health = "review_pending"
		}
		out = append(out, map[string]any{
			"dossierId":      id,
			"name":           name,
			"jobCount":       jobs,
			"failureCount":   fails,
			"reviewPending":  reviews,
			"health":         health,
		})
	}
	return out, rows.Err()
}

func (s *Service) automationActivity(ctx context.Context, limit int) ([]AutomationLite, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, rule_id, status, message, created_at
FROM automation_history
ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AutomationLite{}
	for rows.Next() {
		var r AutomationLite
		var rid sql.NullInt64
		if err := rows.Scan(&r.ID, &rid, &r.Status, &r.Message, &r.CreatedAtMs); err != nil {
			return nil, err
		}
		if rid.Valid {
			v := rid.Int64
			r.RuleID = &v
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Service) recommendations(ctx context.Context, limit int) ([]RecommendationLite, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, task_type, target_adapter, confidence, reasons_json, created_at
FROM routing_policy_recommendations
ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []RecommendationLite{}
	for rows.Next() {
		var r RecommendationLite
		var reasons string
		if err := rows.Scan(&r.ID, &r.TaskType, &r.Adapter, &r.Confidence, &reasons, &r.CreatedAtMs); err != nil {
			return nil, err
		}
		r.Reasons = json.RawMessage(reasons)
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Service) scalarInt(ctx context.Context, query string) int {
	var v int
	_ = s.db.QueryRowContext(ctx, query).Scan(&v)
	return v
}
