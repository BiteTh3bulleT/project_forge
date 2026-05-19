package automation

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const maxRulePayloadBytes = 64 << 10

var errRulePayloadTooLarge = errors.New("automation rule payload too large")

type Rule struct {
	ID            int64           `json:"id"`
	CreatedAtMs   int64           `json:"createdAtMs"`
	UpdatedAtMs   int64           `json:"updatedAtMs"`
	Name          string          `json:"name"`
	Trigger       string          `json:"trigger"`
	Condition     json.RawMessage `json:"condition"`
	Action        json.RawMessage `json:"action"`
	Scope         json.RawMessage `json:"scope"`
	Enabled       bool            `json:"enabled"`
	DryRunDefault bool            `json:"dryRunDefault"`
}

type HistoryEntry struct {
	ID          int64           `json:"id"`
	CreatedAtMs int64           `json:"createdAtMs"`
	RuleID      *int64          `json:"ruleId"`
	Trigger     string          `json:"trigger"`
	Matched     bool            `json:"matched"`
	DryRun      bool            `json:"dryRun"`
	Status      string          `json:"status"`
	Message     string          `json:"message"`
	Preview     json.RawMessage `json:"preview"`
	Result      json.RawMessage `json:"result"`
}

type SaveRuleRequest struct {
	ID            *int64         `json:"id"`
	Name          string         `json:"name"`
	Trigger       string         `json:"trigger"`
	Condition     map[string]any `json:"condition"`
	Action        map[string]any `json:"action"`
	Scope         map[string]any `json:"scope"`
	Enabled       bool           `json:"enabled"`
	DryRunDefault bool           `json:"dryRunDefault"`
}

type RunRequest struct {
	RuleID  int64          `json:"ruleId"`
	Trigger string         `json:"trigger"`
	Context map[string]any `json:"context"`
	DryRun  *bool          `json:"dryRun"`
}

type RunResult struct {
	Rule      Rule           `json:"rule"`
	Matched   bool           `json:"matched"`
	DryRun    bool           `json:"dryRun"`
	Preview   map[string]any `json:"preview"`
	Executed  bool           `json:"executed"`
	Execution map[string]any `json:"execution"`
	HistoryID int64          `json:"historyId"`
}

type Executor func(ctx context.Context, action map[string]any, scope map[string]any, dryRun bool) (map[string]any, error)

type Service struct {
	db *sql.DB
}

func New(db *sql.DB) *Service { return &Service{db: db} }

func (s *Service) EnsureDefaults(ctx context.Context) error {
	defaults := []SaveRuleRequest{
		{
			Name:    "Regenerate Guidance On Context Import",
			Trigger: "project_context.changed",
			Condition: map[string]any{
				"source": "project_context",
			},
			Action: map[string]any{
				"type":        "create_job",
				"templateId":  "normalize_project_context",
				"title":       "Auto context normalization",
				"userRequest": "Normalize updated project context",
			},
			Scope:         map[string]any{"dossierId": nil},
			Enabled:       false,
			DryRunDefault: true,
		},
		{
			Name:    "Refresh Dossier Brief On Key File Change",
			Trigger: "source.changed",
			Condition: map[string]any{
				"pathContains": "docs/",
			},
			Action: map[string]any{
				"type": "generate_dossier_brief",
			},
			Scope:         map[string]any{},
			Enabled:       false,
			DryRunDefault: true,
		},
		{
			Name:    "Queue Review After External Import",
			Trigger: "import.execution.created",
			Condition: map[string]any{
				"always": true,
			},
			Action: map[string]any{
				"type": "create_review",
			},
			Scope:         map[string]any{},
			Enabled:       true,
			DryRunDefault: true,
		},
	}
	for _, r := range defaults {
		if err := s.ensureDefaultRule(ctx, r); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) ensureDefaultRule(ctx context.Context, req SaveRuleRequest) error {
	name := strings.TrimSpace(req.Name)
	trigger := strings.TrimSpace(req.Trigger)
	if name == "" || trigger == "" {
		return fmt.Errorf("name and trigger are required")
	}
	condition, err := marshalRulePayload("condition", req.Condition)
	if err != nil {
		return err
	}
	action, err := marshalRulePayload("action", req.Action)
	if err != nil {
		return err
	}
	scope, err := marshalRulePayload("scope", req.Scope)
	if err != nil {
		return err
	}
	now := time.Now().UnixMilli()
	_, err = s.db.ExecContext(ctx, `
INSERT INTO automation_rules(created_at, updated_at, name, trigger, condition_json, action_json, scope_json, enabled, dry_run_default)
VALUES(?,?,?,?,?,?,?,?,?)
ON CONFLICT(name) DO NOTHING`,
		now, now, name, trigger, string(condition), string(action), string(scope), boolToInt(req.Enabled), boolToInt(req.DryRunDefault),
	)
	return err
}

func (s *Service) SaveRule(ctx context.Context, req SaveRuleRequest) (*Rule, error) {
	name := strings.TrimSpace(req.Name)
	trigger := strings.TrimSpace(req.Trigger)
	if name == "" || trigger == "" {
		return nil, fmt.Errorf("name and trigger are required")
	}
	condition, err := marshalRulePayload("condition", req.Condition)
	if err != nil {
		return nil, err
	}
	action, err := marshalRulePayload("action", req.Action)
	if err != nil {
		return nil, err
	}
	scope, err := marshalRulePayload("scope", req.Scope)
	if err != nil {
		return nil, err
	}
	now := time.Now().UnixMilli()

	if req.ID == nil || *req.ID <= 0 {
		res, err := s.db.ExecContext(ctx, `
INSERT INTO automation_rules(created_at, updated_at, name, trigger, condition_json, action_json, scope_json, enabled, dry_run_default)
VALUES(?,?,?,?,?,?,?,?,?)`,
			now, now, name, trigger, string(condition), string(action), string(scope), boolToInt(req.Enabled), boolToInt(req.DryRunDefault),
		)
		if err != nil {
			return nil, err
		}
		id, _ := res.LastInsertId()
		return s.GetRule(ctx, id)
	}
	_, err = s.db.ExecContext(ctx, `
UPDATE automation_rules
SET updated_at = ?, name = ?, trigger = ?, condition_json = ?, action_json = ?, scope_json = ?, enabled = ?, dry_run_default = ?
WHERE id = ?`,
		now, name, trigger, string(condition), string(action), string(scope), boolToInt(req.Enabled), boolToInt(req.DryRunDefault), *req.ID,
	)
	if err != nil {
		return nil, err
	}
	return s.GetRule(ctx, *req.ID)
}

func (s *Service) GetRule(ctx context.Context, id int64) (*Rule, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, created_at, updated_at, name, trigger, condition_json, action_json, scope_json, enabled, dry_run_default
FROM automation_rules WHERE id = ?`, id)
	return scanRule(row)
}

func (s *Service) ListRules(ctx context.Context, enabledOnly bool, limit int) ([]Rule, error) {
	if limit <= 0 || limit > 400 {
		limit = 120
	}
	query := `
SELECT id, created_at, updated_at, name, trigger, condition_json, action_json, scope_json, enabled, dry_run_default
FROM automation_rules`
	args := []any{}
	if enabledOnly {
		query += ` WHERE enabled = 1`
	}
	query += ` ORDER BY updated_at DESC, id DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Rule{}
	for rows.Next() {
		r, err := scanRule(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return out, rows.Err()
}

func (s *Service) ListHistory(ctx context.Context, limit int) ([]HistoryEntry, error) {
	if limit <= 0 || limit > 500 {
		limit = 120
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, created_at, rule_id, trigger, matched, dry_run, status, message, preview_json, result_json
FROM automation_history
ORDER BY id DESC
LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []HistoryEntry{}
	for rows.Next() {
		var h HistoryEntry
		var ruleID sql.NullInt64
		var matched int
		var dryRun int
		var preview string
		var result string
		if err := rows.Scan(&h.ID, &h.CreatedAtMs, &ruleID, &h.Trigger, &matched, &dryRun, &h.Status, &h.Message, &preview, &result); err != nil {
			return nil, err
		}
		if ruleID.Valid {
			v := ruleID.Int64
			h.RuleID = &v
		}
		h.Matched = matched == 1
		h.DryRun = dryRun == 1
		h.Preview = json.RawMessage(preview)
		h.Result = json.RawMessage(result)
		out = append(out, h)
	}
	return out, rows.Err()
}

func (s *Service) Run(ctx context.Context, req RunRequest, exec Executor) (*RunResult, error) {
	rule, err := s.GetRule(ctx, req.RuleID)
	if err != nil {
		return nil, err
	}
	trigger := strings.TrimSpace(req.Trigger)
	if trigger == "" {
		trigger = rule.Trigger
	}
	preview := map[string]any{
		"ruleId":   rule.ID,
		"name":     rule.Name,
		"trigger":  trigger,
		"action":   json.RawMessage(rule.Action),
		"scope":    json.RawMessage(rule.Scope),
		"context":  nonNilMap(req.Context),
		"matched":  false,
		"executed": false,
	}

	matched := strings.EqualFold(trigger, rule.Trigger) && rule.Enabled && conditionMatches(rule.Condition, req.Context)
	preview["matched"] = matched
	dryRun := rule.DryRunDefault
	if req.DryRun != nil {
		dryRun = *req.DryRun
	}
	preview["dryRun"] = dryRun

	resultPayload := map[string]any{}
	executed := false
	status := "skipped"
	message := "rule did not match or disabled"

	if matched {
		status = "preview_ready"
		message = "rule matched"
		if !dryRun && exec != nil {
			var action map[string]any
			var scope map[string]any
			_ = json.Unmarshal(rule.Action, &action)
			_ = json.Unmarshal(rule.Scope, &scope)
			res, runErr := exec(ctx, nonNilMap(action), nonNilMap(scope), false)
			if runErr != nil {
				status = "failed"
				message = runErr.Error()
				resultPayload["error"] = runErr.Error()
			} else {
				status = "executed"
				message = "automation action executed"
				executed = true
				resultPayload = res
			}
		}
	}

	previewJSON, _ := json.Marshal(preview)
	resultJSON, _ := json.Marshal(resultPayload)
	now := time.Now().UnixMilli()
	res, err := s.db.ExecContext(ctx, `
INSERT INTO automation_history(created_at, rule_id, trigger, matched, dry_run, status, message, preview_json, result_json)
VALUES(?,?,?,?,?,?,?,?,?)`,
		now, rule.ID, trigger, boolToInt(matched), boolToInt(dryRun), status, message, string(previewJSON), string(resultJSON),
	)
	if err != nil {
		return nil, err
	}
	hid, _ := res.LastInsertId()
	return &RunResult{
		Rule:      *rule,
		Matched:   matched,
		DryRun:    dryRun,
		Preview:   preview,
		Executed:  executed,
		Execution: resultPayload,
		HistoryID: hid,
	}, nil
}

func scanRule(scanner interface{ Scan(dest ...any) error }) (*Rule, error) {
	var r Rule
	var condition string
	var action string
	var scope string
	var enabled int
	var dryRun int
	if err := scanner.Scan(&r.ID, &r.CreatedAtMs, &r.UpdatedAtMs, &r.Name, &r.Trigger, &condition, &action, &scope, &enabled, &dryRun); err != nil {
		return nil, err
	}
	r.Condition = json.RawMessage(condition)
	r.Action = json.RawMessage(action)
	r.Scope = json.RawMessage(scope)
	r.Enabled = enabled == 1
	r.DryRunDefault = dryRun == 1
	return &r, nil
}

func nonNilMap(v map[string]any) map[string]any {
	if v == nil {
		return map[string]any{}
	}
	return v
}

func conditionMatches(raw json.RawMessage, ctx map[string]any) bool {
	condition := map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &condition); err != nil {
			return false
		}
	}
	if len(condition) == 0 {
		return true
	}
	if v, ok := condition["always"].(bool); ok && !v {
		return false
	}
	if want, ok := stringValue(condition["source"]); ok {
		got, ok := stringValue(nonNilMap(ctx)["source"])
		if !ok || got != want {
			return false
		}
	}
	if needle, ok := stringValue(condition["pathContains"]); ok {
		path, ok := stringValue(nonNilMap(ctx)["path"])
		if !ok || !strings.Contains(path, needle) {
			return false
		}
	}
	return true
}

func stringValue(v any) (string, bool) {
	s, ok := v.(string)
	if !ok {
		return "", false
	}
	s = strings.TrimSpace(s)
	return s, s != ""
}

func marshalRulePayload(label string, payload map[string]any) ([]byte, error) {
	body, err := json.Marshal(nonNilMap(payload))
	if err != nil {
		return nil, err
	}
	if len(body) > maxRulePayloadBytes {
		return nil, fmt.Errorf("%w: %s %d > %d bytes", errRulePayloadTooLarge, strings.TrimSpace(label), len(body), maxRulePayloadBytes)
	}
	return body, nil
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
