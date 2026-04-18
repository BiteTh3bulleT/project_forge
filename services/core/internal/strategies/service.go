package strategies

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Strategy struct {
	ID                string          `json:"id"`
	CreatedAtMs       int64           `json:"createdAtMs"`
	UpdatedAtMs       int64           `json:"updatedAtMs"`
	Name              string          `json:"name"`
	TaskType          string          `json:"taskType"`
	TargetAdapter     string          `json:"targetAdapter"`
	RetrievalMode     string          `json:"retrievalMode"`
	PacketRules       json.RawMessage `json:"packetRules"`
	ApprovalRequired  bool            `json:"approvalRequired"`
	ApprovalPresetID  *string         `json:"approvalPresetId"`
	ExpectedArtifacts json.RawMessage `json:"expectedArtifacts"`
	SuccessCriteria   json.RawMessage `json:"successCriteria"`
	RetryGuidance     json.RawMessage `json:"retryGuidance"`
	Enabled           bool            `json:"enabled"`
}

type SaveRequest struct {
	ID                string         `json:"id"`
	Name              string         `json:"name"`
	TaskType          string         `json:"taskType"`
	TargetAdapter     string         `json:"targetAdapter"`
	RetrievalMode     string         `json:"retrievalMode"`
	PacketRules       map[string]any `json:"packetRules"`
	ApprovalRequired  bool           `json:"approvalRequired"`
	ApprovalPresetID  *string        `json:"approvalPresetId"`
	ExpectedArtifacts []string       `json:"expectedArtifacts"`
	SuccessCriteria   map[string]any `json:"successCriteria"`
	RetryGuidance     map[string]any `json:"retryGuidance"`
	Enabled           bool           `json:"enabled"`
}

type Service struct {
	db *sql.DB
}

func New(db *sql.DB) *Service { return &Service{db: db} }

func (s *Service) EnsureDefaults(ctx context.Context) error {
	defs := []SaveRequest{
		{
			ID:               "local_summarize",
			Name:             "Local Summarize",
			TaskType:         "local_summarize",
			TargetAdapter:    "ollama",
			RetrievalMode:    "hybrid",
			PacketRules:      map[string]any{"targetItems": 8, "maxItems": 12},
			ApprovalRequired: true,
			ApprovalPresetID: strPtr("balanced"),
			ExpectedArtifacts: []string{
				"task_packet", "adapter_output", "job_result",
			},
			SuccessCriteria: map[string]any{"requiresSummary": true},
			RetryGuidance:   map[string]any{"maxRetries": 2, "adjustment": "tighten_scope"},
			Enabled:         true,
		},
		{
			ID:               "repo_analysis",
			Name:             "Repository Analysis",
			TaskType:         "repo_analysis",
			TargetAdapter:    "ollama",
			RetrievalMode:    "hybrid",
			PacketRules:      map[string]any{"targetItems": 10, "maxItems": 18},
			ApprovalRequired: true,
			ApprovalPresetID: strPtr("balanced"),
			ExpectedArtifacts: []string{
				"task_packet", "adapter_output", "job_result",
			},
			SuccessCriteria: map[string]any{"requiresRiskList": true},
			RetryGuidance:   map[string]any{"maxRetries": 2, "adjustment": "switch_retrieval_mode"},
			Enabled:         true,
		},
		{
			ID:               "codex_implementation_handoff",
			Name:             "Codex Implementation Handoff",
			TaskType:         "codex_implementation",
			TargetAdapter:    "codex",
			RetrievalMode:    "hybrid",
			PacketRules:      map[string]any{"targetItems": 10, "maxItems": 16, "requirePathScope": true},
			ApprovalRequired: true,
			ApprovalPresetID: strPtr("conservative"),
			ExpectedArtifacts: []string{
				"task_packet", "agent_guidance", "adapter_output",
			},
			SuccessCriteria: map[string]any{"requiresDeliverableType": "patch_or_plan"},
			RetryGuidance:   map[string]any{"maxRetries": 2, "adjustment": "add_review_gate"},
			Enabled:         true,
		},
		{
			ID:               "claude_refactor_planning",
			Name:             "Claude Refactor Planning",
			TaskType:         "claude_refactor_planning",
			TargetAdapter:    "claude_code",
			RetrievalMode:    "hybrid",
			PacketRules:      map[string]any{"targetItems": 12, "maxItems": 20},
			ApprovalRequired: true,
			ApprovalPresetID: strPtr("conservative"),
			ExpectedArtifacts: []string{
				"task_packet", "agent_guidance", "adapter_output",
			},
			SuccessCriteria: map[string]any{"requiresPlan": true},
			RetryGuidance:   map[string]any{"maxRetries": 2, "adjustment": "split_scope"},
			Enabled:         true,
		},
		{
			ID:               "context_regeneration",
			Name:             "Context Regeneration",
			TaskType:         "context_regeneration",
			TargetAdapter:    "forge",
			RetrievalMode:    "keyword",
			PacketRules:      map[string]any{"targetItems": 4, "maxItems": 8},
			ApprovalRequired: true,
			ApprovalPresetID: strPtr("balanced"),
			ExpectedArtifacts: []string{
				"context_normalization", "agent_guidance", "job_result",
			},
			SuccessCriteria: map[string]any{"requiresUpdatedGuides": true},
			RetryGuidance:   map[string]any{"maxRetries": 1, "adjustment": "verify_source_context"},
			Enabled:         true,
		},
		{
			ID:               "review_workflow",
			Name:             "Review Workflow",
			TaskType:         "review_workflow",
			TargetAdapter:    "forge",
			RetrievalMode:    "keyword",
			PacketRules:      map[string]any{"targetItems": 6, "maxItems": 10},
			ApprovalRequired: false,
			ApprovalPresetID: strPtr("balanced"),
			ExpectedArtifacts: []string{
				"job_result",
			},
			SuccessCriteria: map[string]any{"requiresReviewDecision": true},
			RetryGuidance:   map[string]any{"maxRetries": 1, "adjustment": "request_operator_annotation"},
			Enabled:         true,
		},
	}
	for _, d := range defs {
		if _, err := s.Save(ctx, d); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) Save(ctx context.Context, req SaveRequest) (*Strategy, error) {
	id := strings.TrimSpace(req.ID)
	if id == "" {
		return nil, fmt.Errorf("strategy id is required")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("strategy name is required")
	}
	taskType := strings.TrimSpace(req.TaskType)
	if taskType == "" {
		return nil, fmt.Errorf("taskType is required")
	}
	targetAdapter := strings.TrimSpace(req.TargetAdapter)
	if targetAdapter == "" {
		return nil, fmt.Errorf("targetAdapter is required")
	}
	retrievalMode := strings.TrimSpace(req.RetrievalMode)
	if retrievalMode == "" {
		retrievalMode = "hybrid"
	}
	now := time.Now().UnixMilli()

	packetRules, _ := json.Marshal(nonNilMap(req.PacketRules))
	expectedArtifacts, _ := json.Marshal(nonNilStrings(req.ExpectedArtifacts))
	successCriteria, _ := json.Marshal(nonNilMap(req.SuccessCriteria))
	retryGuidance, _ := json.Marshal(nonNilMap(req.RetryGuidance))

	_, err := s.db.ExecContext(ctx, `
INSERT INTO execution_strategies(
  id, created_at, updated_at, name, task_type, target_adapter, retrieval_mode,
  packet_rules_json, approval_required, approval_preset_id, expected_artifacts_json,
  success_criteria_json, retry_guidance_json, enabled
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id)
DO UPDATE SET
  updated_at=excluded.updated_at,
  name=excluded.name,
  task_type=excluded.task_type,
  target_adapter=excluded.target_adapter,
  retrieval_mode=excluded.retrieval_mode,
  packet_rules_json=excluded.packet_rules_json,
  approval_required=excluded.approval_required,
  approval_preset_id=excluded.approval_preset_id,
  expected_artifacts_json=excluded.expected_artifacts_json,
  success_criteria_json=excluded.success_criteria_json,
  retry_guidance_json=excluded.retry_guidance_json,
  enabled=excluded.enabled`,
		id,
		now,
		now,
		name,
		taskType,
		targetAdapter,
		retrievalMode,
		string(packetRules),
		boolToInt(req.ApprovalRequired),
		req.ApprovalPresetID,
		string(expectedArtifacts),
		string(successCriteria),
		string(retryGuidance),
		boolToInt(req.Enabled),
	)
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

func (s *Service) Get(ctx context.Context, id string) (*Strategy, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, created_at, updated_at, name, task_type, target_adapter, retrieval_mode,
       packet_rules_json, approval_required, approval_preset_id, expected_artifacts_json,
       success_criteria_json, retry_guidance_json, enabled
FROM execution_strategies WHERE id = ?`, strings.TrimSpace(id))
	return scanStrategy(row)
}

func (s *Service) List(ctx context.Context, enabledOnly bool, limit int) ([]Strategy, error) {
	if limit <= 0 || limit > 300 {
		limit = 120
	}
	query := `
SELECT id, created_at, updated_at, name, task_type, target_adapter, retrieval_mode,
       packet_rules_json, approval_required, approval_preset_id, expected_artifacts_json,
       success_criteria_json, retry_guidance_json, enabled
FROM execution_strategies`
	args := []any{}
	if enabledOnly {
		query += ` WHERE enabled = 1`
	}
	query += ` ORDER BY name ASC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Strategy{}
	for rows.Next() {
		r, err := scanStrategy(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return out, rows.Err()
}

func scanStrategy(scanner interface{ Scan(dest ...any) error }) (*Strategy, error) {
	var r Strategy
	var packetRules string
	var expectedArtifacts string
	var successCriteria string
	var retryGuidance string
	var approvalRequired int
	var enabled int
	var approvalPreset sql.NullString
	if err := scanner.Scan(
		&r.ID, &r.CreatedAtMs, &r.UpdatedAtMs, &r.Name, &r.TaskType, &r.TargetAdapter, &r.RetrievalMode,
		&packetRules, &approvalRequired, &approvalPreset, &expectedArtifacts, &successCriteria, &retryGuidance, &enabled,
	); err != nil {
		return nil, err
	}
	r.PacketRules = json.RawMessage(packetRules)
	r.ExpectedArtifacts = json.RawMessage(expectedArtifacts)
	r.SuccessCriteria = json.RawMessage(successCriteria)
	r.RetryGuidance = json.RawMessage(retryGuidance)
	r.ApprovalRequired = approvalRequired == 1
	r.Enabled = enabled == 1
	if approvalPreset.Valid {
		v := approvalPreset.String
		r.ApprovalPresetID = &v
	}
	return &r, nil
}

func strPtr(v string) *string {
	x := strings.TrimSpace(v)
	return &x
}

func nonNilMap(v map[string]any) map[string]any {
	if v == nil {
		return map[string]any{}
	}
	return v
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

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
