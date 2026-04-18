package policy

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"forge/projectforge/services/core/internal/strategies"
)

type ApprovalPreset struct {
	ID          string          `json:"id"`
	CreatedAtMs int64           `json:"createdAtMs"`
	UpdatedAtMs int64           `json:"updatedAtMs"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Profile     json.RawMessage `json:"profile"`
	Editable    bool            `json:"editable"`
}

type DossierProfile struct {
	DossierID           int64           `json:"dossierId"`
	UpdatedAtMs         int64           `json:"updatedAtMs"`
	PreferredStrategies json.RawMessage `json:"preferredStrategies"`
	PreferredAdapters   json.RawMessage `json:"preferredAdapters"`
	ApprovalPresetID    *string         `json:"approvalPresetId"`
	RetrievalDefaults   json.RawMessage `json:"retrievalDefaults"`
	HighValueFiles      json.RawMessage `json:"highValueFiles"`
	NoisyFiles          json.RawMessage `json:"noisyFiles"`
	RoutingNotes        string          `json:"routingNotes"`
	AutomationBindings  json.RawMessage `json:"automationBindings"`
}

type RecommendRequest struct {
	TaskType   string  `json:"taskType"`
	DossierID  *int64  `json:"dossierId"`
	StrategyID *string `json:"strategyId"`
	Objective  string  `json:"objective"`
}

type Recommendation struct {
	ID                     int64           `json:"id"`
	CreatedAtMs            int64           `json:"createdAtMs"`
	DossierID              *int64          `json:"dossierId"`
	TaskType               string          `json:"taskType"`
	StrategyID             *string         `json:"strategyId"`
	TargetAdapter          string          `json:"targetAdapter"`
	RetrievalMode          string          `json:"retrievalMode"`
	PacketShape            json.RawMessage `json:"packetShape"`
	ApprovalPresetID       *string         `json:"approvalPresetId"`
	ApprovalRequired       bool            `json:"approvalRequired"`
	Confidence             float64         `json:"confidence"`
	Reasons                json.RawMessage `json:"reasons"`
	Evidence               json.RawMessage `json:"evidence"`
	Inferred               bool            `json:"inferred"`
	OperatorOverrideAllowed bool           `json:"operatorOverrideAllowed"`
}

type Service struct {
	db         *sql.DB
	strategies *strategies.Service
}

func New(db *sql.DB, strategySvc *strategies.Service) *Service {
	return &Service{db: db, strategies: strategySvc}
}

func (s *Service) EnsureDefaults(ctx context.Context) error {
	now := time.Now().UnixMilli()
	presets := []struct {
		ID          string
		Name        string
		Description string
		Profile     map[string]any
		Editable    bool
	}{
		{
			ID:          "conservative",
			Name:        "Conservative",
			Description: "Read-only automation by default. Human approval required for reasoning and all risky operations.",
			Profile: map[string]any{
				"autoRun": map[string]bool{
					"read_only":          true,
					"external_reasoning": false,
					"write_files":        false,
					"run_commands":       false,
				},
				"requireReviewBeforeRetry": true,
			},
			Editable: false,
		},
		{
			ID:          "balanced",
			Name:        "Balanced",
			Description: "Read-only and selected reasoning can run automatically, write/command actions stay gated.",
			Profile: map[string]any{
				"autoRun": map[string]bool{
					"read_only":          true,
					"external_reasoning": true,
					"write_files":        false,
					"run_commands":       false,
				},
				"requireReviewBeforeRetry": false,
			},
			Editable: false,
		},
		{
			ID:          "aggressive",
			Name:        "Aggressive",
			Description: "Max automation within policy boundaries; risky operations still require explicit approvals.",
			Profile: map[string]any{
				"autoRun": map[string]bool{
					"read_only":          true,
					"external_reasoning": true,
					"write_files":        false,
					"run_commands":       false,
				},
				"requireReviewBeforeRetry": false,
			},
			Editable: false,
		},
	}
	for _, p := range presets {
		profile, _ := json.Marshal(p.Profile)
		_, err := s.db.ExecContext(ctx, `
INSERT INTO approval_presets(id, created_at, updated_at, name, description, profile_json, editable)
VALUES(?,?,?,?,?,?,?)
ON CONFLICT(id)
DO UPDATE SET
  updated_at=excluded.updated_at,
  name=excluded.name,
  description=excluded.description,
  profile_json=excluded.profile_json`,
			p.ID, now, now, p.Name, p.Description, string(profile), boolToInt(p.Editable),
		)
		if err != nil {
			return err
		}
	}
	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO settings(key, value) VALUES('approval_preset_global', 'balanced') ON CONFLICT(key) DO NOTHING`,
	); err != nil {
		return err
	}
	return nil
}

func (s *Service) Recommend(ctx context.Context, req RecommendRequest) (*Recommendation, error) {
	taskType := strings.TrimSpace(req.TaskType)
	if taskType == "" {
		taskType = "general"
	}
	profile, _ := s.DossierProfile(ctx, req.DossierID)

	strategy, err := s.selectStrategy(ctx, taskType, req.StrategyID, profile)
	if err != nil {
		return nil, err
	}

	targetAdapter := strategy.TargetAdapter
	if profile != nil {
		var preferredAdapters []string
		_ = json.Unmarshal(profile.PreferredAdapters, &preferredAdapters)
		if len(preferredAdapters) > 0 && strings.TrimSpace(preferredAdapters[0]) != "" {
			targetAdapter = preferredAdapters[0]
		}
	}

	retrievalMode := strategy.RetrievalMode
	if profile != nil {
		var defaults map[string]any
		if err := json.Unmarshal(profile.RetrievalDefaults, &defaults); err == nil {
			if mode, ok := defaults["mode"].(string); ok && strings.TrimSpace(mode) != "" {
				retrievalMode = strings.TrimSpace(mode)
			}
		}
	}

	approvalPresetID := strategy.ApprovalPresetID
	if profile != nil && profile.ApprovalPresetID != nil {
		approvalPresetID = profile.ApprovalPresetID
	}
	if approvalPresetID == nil {
		if global, _ := s.GlobalApprovalPreset(ctx); global != "" {
			approvalPresetID = &global
		}
	}

	packetShape, reasons, evidence, confidence, inferred := s.computeGuidance(ctx, strategy, req.DossierID, targetAdapter, retrievalMode)
	now := time.Now().UnixMilli()
	reasonsJSON, _ := json.Marshal(reasons)
	evidenceJSON, _ := json.Marshal(evidence)
	packetShapeJSON, _ := json.Marshal(packetShape)

	res, err := s.db.ExecContext(ctx, `
INSERT INTO routing_policy_recommendations(
  created_at, dossier_id, task_type, strategy_id, target_adapter, retrieval_mode,
  packet_shape_json, approval_preset_id, approval_required, confidence, reasons_json, evidence_json,
  inferred, operator_override_allowed
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		now,
		req.DossierID,
		taskType,
		strategyID(strategy),
		targetAdapter,
		retrievalMode,
		string(packetShapeJSON),
		approvalPresetID,
		boolToInt(strategy.ApprovalRequired),
		confidence,
		string(reasonsJSON),
		string(evidenceJSON),
		boolToInt(inferred),
		1,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.GetRecommendation(ctx, id)
}

func (s *Service) GetRecommendation(ctx context.Context, id int64) (*Recommendation, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, created_at, dossier_id, task_type, strategy_id, target_adapter, retrieval_mode,
       packet_shape_json, approval_preset_id, approval_required, confidence, reasons_json, evidence_json,
       inferred, operator_override_allowed
FROM routing_policy_recommendations WHERE id = ?`, id)
	return scanRecommendation(row)
}

func (s *Service) ListRecommendations(ctx context.Context, limit int, dossierID *int64) ([]Recommendation, error) {
	if limit <= 0 || limit > 300 {
		limit = 120
	}
	query := `
SELECT id, created_at, dossier_id, task_type, strategy_id, target_adapter, retrieval_mode,
       packet_shape_json, approval_preset_id, approval_required, confidence, reasons_json, evidence_json,
       inferred, operator_override_allowed
FROM routing_policy_recommendations`
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
	out := []Recommendation{}
	for rows.Next() {
		r, err := scanRecommendation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return out, rows.Err()
}

func (s *Service) ListApprovalPresets(ctx context.Context, limit int) ([]ApprovalPreset, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, created_at, updated_at, name, description, profile_json, editable
FROM approval_presets
ORDER BY name ASC
LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ApprovalPreset{}
	for rows.Next() {
		var r ApprovalPreset
		var profile string
		var editable int
		if err := rows.Scan(&r.ID, &r.CreatedAtMs, &r.UpdatedAtMs, &r.Name, &r.Description, &profile, &editable); err != nil {
			return nil, err
		}
		r.Profile = json.RawMessage(profile)
		r.Editable = editable == 1
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Service) SaveApprovalPreset(ctx context.Context, id, name, description string, profile map[string]any, editable bool) (*ApprovalPreset, error) {
	id = strings.TrimSpace(id)
	name = strings.TrimSpace(name)
	if id == "" || name == "" {
		return nil, fmt.Errorf("preset id and name are required")
	}
	now := time.Now().UnixMilli()
	profileJSON, _ := json.Marshal(nonNilMap(profile))
	_, err := s.db.ExecContext(ctx, `
INSERT INTO approval_presets(id, created_at, updated_at, name, description, profile_json, editable)
VALUES(?,?,?,?,?,?,?)
ON CONFLICT(id)
DO UPDATE SET
  updated_at=excluded.updated_at,
  name=excluded.name,
  description=excluded.description,
  profile_json=excluded.profile_json,
  editable=excluded.editable`,
		id, now, now, name, strings.TrimSpace(description), string(profileJSON), boolToInt(editable),
	)
	if err != nil {
		return nil, err
	}
	row := s.db.QueryRowContext(ctx, `
SELECT id, created_at, updated_at, name, description, profile_json, editable
FROM approval_presets WHERE id = ?`, id)
	var r ApprovalPreset
	var profileRaw string
	var ed int
	if err := row.Scan(&r.ID, &r.CreatedAtMs, &r.UpdatedAtMs, &r.Name, &r.Description, &profileRaw, &ed); err != nil {
		return nil, err
	}
	r.Profile = json.RawMessage(profileRaw)
	r.Editable = ed == 1
	return &r, nil
}

func (s *Service) GlobalApprovalPreset(ctx context.Context) (string, error) {
	var value string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = 'approval_preset_global'`).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(value), nil
}

func (s *Service) SetGlobalApprovalPreset(ctx context.Context, presetID string) error {
	presetID = strings.TrimSpace(presetID)
	if presetID == "" {
		return fmt.Errorf("preset id is required")
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO settings(key, value) VALUES('approval_preset_global', ?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`,
		presetID,
	)
	return err
}

func (s *Service) DossierProfile(ctx context.Context, dossierID *int64) (*DossierProfile, error) {
	if dossierID == nil || *dossierID <= 0 {
		return nil, nil
	}
	row := s.db.QueryRowContext(ctx, `
SELECT dossier_id, updated_at, preferred_strategies_json, preferred_adapters_json, approval_preset_id,
       retrieval_defaults_json, high_value_files_json, noisy_files_json, routing_notes, automation_bindings_json
FROM dossier_profiles
WHERE dossier_id = ?`, *dossierID)
	var p DossierProfile
	var preferredStrategies string
	var preferredAdapters string
	var approvalPreset sql.NullString
	var retrievalDefaults string
	var highValueFiles string
	var noisyFiles string
	var automationBindings string
	if err := row.Scan(
		&p.DossierID, &p.UpdatedAtMs, &preferredStrategies, &preferredAdapters, &approvalPreset,
		&retrievalDefaults, &highValueFiles, &noisyFiles, &p.RoutingNotes, &automationBindings,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	p.PreferredStrategies = json.RawMessage(preferredStrategies)
	p.PreferredAdapters = json.RawMessage(preferredAdapters)
	p.RetrievalDefaults = json.RawMessage(retrievalDefaults)
	p.HighValueFiles = json.RawMessage(highValueFiles)
	p.NoisyFiles = json.RawMessage(noisyFiles)
	p.AutomationBindings = json.RawMessage(automationBindings)
	if approvalPreset.Valid {
		v := approvalPreset.String
		p.ApprovalPresetID = &v
	}
	return &p, nil
}

type SaveDossierProfileRequest struct {
	DossierID           int64          `json:"dossierId"`
	PreferredStrategies []string       `json:"preferredStrategies"`
	PreferredAdapters   []string       `json:"preferredAdapters"`
	ApprovalPresetID    *string        `json:"approvalPresetId"`
	RetrievalDefaults   map[string]any `json:"retrievalDefaults"`
	HighValueFiles      []string       `json:"highValueFiles"`
	NoisyFiles          []string       `json:"noisyFiles"`
	RoutingNotes        string         `json:"routingNotes"`
	AutomationBindings  []int64        `json:"automationBindings"`
}

func (s *Service) SaveDossierProfile(ctx context.Context, req SaveDossierProfileRequest) (*DossierProfile, error) {
	if req.DossierID <= 0 {
		return nil, fmt.Errorf("dossierId is required")
	}
	now := time.Now().UnixMilli()
	strategiesJSON, _ := json.Marshal(nonNilStrings(req.PreferredStrategies))
	adaptersJSON, _ := json.Marshal(nonNilStrings(req.PreferredAdapters))
	retrievalDefaultsJSON, _ := json.Marshal(nonNilMap(req.RetrievalDefaults))
	highValueJSON, _ := json.Marshal(nonNilStrings(req.HighValueFiles))
	noisyJSON, _ := json.Marshal(nonNilStrings(req.NoisyFiles))
	automationBindingsJSON, _ := json.Marshal(req.AutomationBindings)
	_, err := s.db.ExecContext(ctx, `
INSERT INTO dossier_profiles(
  dossier_id, updated_at, preferred_strategies_json, preferred_adapters_json, approval_preset_id,
  retrieval_defaults_json, high_value_files_json, noisy_files_json, routing_notes, automation_bindings_json
) VALUES(?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(dossier_id)
DO UPDATE SET
  updated_at=excluded.updated_at,
  preferred_strategies_json=excluded.preferred_strategies_json,
  preferred_adapters_json=excluded.preferred_adapters_json,
  approval_preset_id=excluded.approval_preset_id,
  retrieval_defaults_json=excluded.retrieval_defaults_json,
  high_value_files_json=excluded.high_value_files_json,
  noisy_files_json=excluded.noisy_files_json,
  routing_notes=excluded.routing_notes,
  automation_bindings_json=excluded.automation_bindings_json`,
		req.DossierID,
		now,
		string(strategiesJSON),
		string(adaptersJSON),
		req.ApprovalPresetID,
		string(retrievalDefaultsJSON),
		string(highValueJSON),
		string(noisyJSON),
		strings.TrimSpace(req.RoutingNotes),
		string(automationBindingsJSON),
	)
	if err != nil {
		return nil, err
	}
	id := req.DossierID
	return s.DossierProfile(ctx, &id)
}

func (s *Service) selectStrategy(ctx context.Context, taskType string, forcedID *string, profile *DossierProfile) (*strategies.Strategy, error) {
	if forcedID != nil && strings.TrimSpace(*forcedID) != "" {
		return s.strategies.Get(ctx, strings.TrimSpace(*forcedID))
	}
	if profile != nil {
		var preferred []string
		if err := json.Unmarshal(profile.PreferredStrategies, &preferred); err == nil {
			for _, id := range preferred {
				id = strings.TrimSpace(id)
				if id == "" {
					continue
				}
				if strat, err := s.strategies.Get(ctx, id); err == nil {
					return strat, nil
				}
			}
		}
	}
	all, err := s.strategies.List(ctx, true, 200)
	if err != nil {
		return nil, err
	}
	for _, strat := range all {
		if strings.EqualFold(strat.TaskType, taskType) {
			tmp := strat
			return &tmp, nil
		}
	}
	if len(all) == 0 {
		return nil, fmt.Errorf("no enabled strategies configured")
	}
	tmp := all[0]
	return &tmp, nil
}

func (s *Service) computeGuidance(ctx context.Context, strategy *strategies.Strategy, dossierID *int64, adapter, retrievalMode string) (map[string]any, []string, map[string]any, float64, bool) {
	packetShape := map[string]any{}
	_ = json.Unmarshal(strategy.PacketRules, &packetShape)
	if packetShape == nil {
		packetShape = map[string]any{}
	}

	query := `
SELECT COUNT(er.id) AS runs,
       AVG(CASE WHEN er.success = 1 THEN 1.0 ELSE 0.0 END) AS success_rate,
       AVG(er.quality_rating) AS quality
FROM evaluation_records er
JOIN jobs j ON j.id = er.job_id
WHERE j.target_adapter = ?`
	args := []any{adapter}
	if dossierID != nil {
		query += ` AND (er.dossier_id = ? OR er.dossier_id IS NULL)`
		args = append(args, *dossierID)
	}
	var runs sql.NullInt64
	var successRate sql.NullFloat64
	var quality sql.NullFloat64
	_ = s.db.QueryRowContext(ctx, query, args...).Scan(&runs, &successRate, &quality)

	reasons := []string{
		fmt.Sprintf("strategy=%s", strategy.Name),
		fmt.Sprintf("adapter=%s", adapter),
		fmt.Sprintf("retrieval_mode=%s", retrievalMode),
	}
	evidence := map[string]any{
		"evaluationRuns": coalesceInt(runs, 0),
		"successRate":    coalesceFloat(successRate, 0),
		"qualityAvg":     coalesceFloat(quality, 0),
	}
	confidence := 0.45
	if runs.Valid && runs.Int64 > 0 {
		confidence = (coalesceFloat(successRate, 0.5) * 0.6) + ((coalesceFloat(quality, 3.0) / 5.0) * 0.4)
	}
	inferred := !(runs.Valid && runs.Int64 >= 3)
	if inferred {
		reasons = append(reasons, "low direct evidence: using inferred recommendation")
	} else {
		reasons = append(reasons, "directly supported by prior evaluations")
	}
	if _, ok := packetShape["targetItems"]; !ok {
		packetShape["targetItems"] = 8
	}
	if _, ok := packetShape["maxItems"]; !ok {
		packetShape["maxItems"] = 14
	}
	packetShape["retrievalMode"] = retrievalMode
	packetShape["requiresManualReviewForRisky"] = true

	confidence = math.Max(0.05, math.Min(0.95, confidence))
	return packetShape, reasons, evidence, confidence, inferred
}

func scanRecommendation(scanner interface{ Scan(dest ...any) error }) (*Recommendation, error) {
	var r Recommendation
	var dossierID sql.NullInt64
	var strategyID sql.NullString
	var packetShape string
	var approvalPreset sql.NullString
	var approvalRequired int
	var reasons string
	var evidence string
	var inferred int
	var override int
	if err := scanner.Scan(
		&r.ID, &r.CreatedAtMs, &dossierID, &r.TaskType, &strategyID, &r.TargetAdapter, &r.RetrievalMode,
		&packetShape, &approvalPreset, &approvalRequired, &r.Confidence, &reasons, &evidence,
		&inferred, &override,
	); err != nil {
		return nil, err
	}
	if dossierID.Valid {
		v := dossierID.Int64
		r.DossierID = &v
	}
	if strategyID.Valid {
		v := strategyID.String
		r.StrategyID = &v
	}
	r.PacketShape = json.RawMessage(packetShape)
	if approvalPreset.Valid {
		v := approvalPreset.String
		r.ApprovalPresetID = &v
	}
	r.ApprovalRequired = approvalRequired == 1
	r.Reasons = json.RawMessage(reasons)
	r.Evidence = json.RawMessage(evidence)
	r.Inferred = inferred == 1
	r.OperatorOverrideAllowed = override == 1
	return &r, nil
}

func strategyID(s *strategies.Strategy) any {
	if s == nil {
		return nil
	}
	return s.ID
}

func coalesceInt(v sql.NullInt64, def int64) int64 {
	if v.Valid {
		return v.Int64
	}
	return def
}

func coalesceFloat(v sql.NullFloat64, def float64) float64 {
	if v.Valid {
		return v.Float64
	}
	return def
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
