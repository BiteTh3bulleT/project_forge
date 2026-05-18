package policy

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	"forge/projectforge/services/core/internal/store"
	"forge/projectforge/services/core/internal/strategies"
)

func TestEnsureDefaultsCreatesStableSystemPresetsAndGlobalDefault(t *testing.T) {
	ctx := context.Background()
	svc, db := newTestService(t, ctx)

	if err := svc.EnsureDefaults(ctx); err != nil {
		t.Fatalf("EnsureDefaults: %v", err)
	}
	if err := svc.EnsureDefaults(ctx); err != nil {
		t.Fatalf("EnsureDefaults second call: %v", err)
	}

	presets, err := svc.ListApprovalPresets(ctx, 10)
	if err != nil {
		t.Fatalf("ListApprovalPresets: %v", err)
	}

	got := map[string]ApprovalPreset{}
	for _, preset := range presets {
		got[preset.ID] = preset
	}

	for _, tc := range []struct {
		id                   string
		requireReviewOnRetry bool
		externalReasoning    bool
	}{
		{id: "aggressive", requireReviewOnRetry: false, externalReasoning: true},
		{id: "balanced", requireReviewOnRetry: false, externalReasoning: true},
		{id: "conservative", requireReviewOnRetry: true, externalReasoning: false},
	} {
		t.Run(tc.id, func(t *testing.T) {
			preset, ok := got[tc.id]
			if !ok {
				t.Fatalf("missing preset %q in %#v", tc.id, got)
			}
			if preset.Editable {
				t.Fatalf("default preset %q should not be editable", tc.id)
			}
			profile := decodeJSON[map[string]any](t, preset.Profile)
			autoRun := profile["autoRun"].(map[string]any)
			if got := profile["requireReviewBeforeRetry"].(bool); got != tc.requireReviewOnRetry {
				t.Fatalf("requireReviewBeforeRetry=%v, want %v", got, tc.requireReviewOnRetry)
			}
			if got := autoRun["external_reasoning"].(bool); got != tc.externalReasoning {
				t.Fatalf("external_reasoning=%v, want %v", got, tc.externalReasoning)
			}
			for _, denied := range []string{"write_files", "run_commands"} {
				if got := autoRun[denied].(bool); got {
					t.Fatalf("%s should remain approval-gated", denied)
				}
			}
		})
	}

	global, err := svc.GlobalApprovalPreset(ctx)
	if err != nil {
		t.Fatalf("GlobalApprovalPreset: %v", err)
	}
	if global != "balanced" {
		t.Fatalf("global preset=%q, want balanced", global)
	}

	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM approval_presets`).Scan(&count); err != nil {
		t.Fatalf("count presets: %v", err)
	}
	if count != 3 {
		t.Fatalf("preset count=%d, want idempotent count 3", count)
	}
}

func TestSaveApprovalPresetValidationAndPersistence(t *testing.T) {
	ctx := context.Background()
	svc, _ := newTestService(t, ctx)

	tests := []struct {
		name        string
		id          string
		displayName string
		wantErr     bool
	}{
		{name: "blank id rejected", id: "  ", displayName: "Needs Name", wantErr: true},
		{name: "blank name rejected", id: "custom", displayName: "  ", wantErr: true},
		{name: "trimmed fields and nil profile persisted", id: " custom ", displayName: " Custom Preset "},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			preset, err := svc.SaveApprovalPreset(ctx, tc.id, tc.displayName, " description ", nil, true)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("SaveApprovalPreset: %v", err)
			}
			if preset.ID != "custom" || preset.Name != "Custom Preset" || preset.Description != "description" {
				t.Fatalf("unexpected persisted preset: %#v", preset)
			}
			if !preset.Editable {
				t.Fatalf("custom preset should persist editable=true")
			}
			profile := decodeJSON[map[string]any](t, preset.Profile)
			if len(profile) != 0 {
				t.Fatalf("nil profile should persist as empty object, got %#v", profile)
			}
		})
	}
}

func TestSaveDossierProfileNormalizesAndRetrievesStructuredFields(t *testing.T) {
	ctx := context.Background()
	svc, db := newTestService(t, ctx)
	dossierID := insertDossier(t, ctx, db)
	presetID := "balanced"

	profile, err := svc.SaveDossierProfile(ctx, SaveDossierProfileRequest{
		DossierID:           dossierID,
		PreferredStrategies: []string{" repo_analysis ", "", " local_summarize "},
		PreferredAdapters:   []string{" codex ", " "},
		ApprovalPresetID:    &presetID,
		RetrievalDefaults:   map[string]any{"mode": "keyword"},
		HighValueFiles:      []string{" AGENTS.md ", ""},
		NoisyFiles:          nil,
		RoutingNotes:        " keep tight ",
		AutomationBindings:  []int64{11, 12},
	})
	if err != nil {
		t.Fatalf("SaveDossierProfile: %v", err)
	}

	if profile.DossierID != dossierID {
		t.Fatalf("DossierID=%d, want %d", profile.DossierID, dossierID)
	}
	if profile.ApprovalPresetID == nil || *profile.ApprovalPresetID != "balanced" {
		t.Fatalf("ApprovalPresetID=%v, want balanced", profile.ApprovalPresetID)
	}
	if profile.RoutingNotes != "keep tight" {
		t.Fatalf("RoutingNotes=%q, want trimmed note", profile.RoutingNotes)
	}
	assertJSONEqual(t, profile.PreferredStrategies, []string{"repo_analysis", "local_summarize"})
	assertJSONEqual(t, profile.PreferredAdapters, []string{"codex"})
	assertJSONEqual(t, profile.RetrievalDefaults, map[string]any{"mode": "keyword"})
	assertJSONEqual(t, profile.HighValueFiles, []string{"AGENTS.md"})
	assertJSONEqual(t, profile.NoisyFiles, []string{})
	assertJSONEqual(t, profile.AutomationBindings, []float64{11, 12})

	missingID := int64(0)
	missing, err := svc.DossierProfile(ctx, &missingID)
	if err != nil {
		t.Fatalf("DossierProfile invalid id: %v", err)
	}
	if missing != nil {
		t.Fatalf("invalid dossier id should return nil profile, got %#v", missing)
	}
}

func TestRecommendAppliesStrategyProfileOverridesEvidenceAndListFilters(t *testing.T) {
	ctx := context.Background()
	svc, db := newTestService(t, ctx)
	dossierID := insertDossier(t, ctx, db)
	presetID := "conservative"

	if _, err := svc.SaveDossierProfile(ctx, SaveDossierProfileRequest{
		DossierID:           dossierID,
		PreferredStrategies: []string{"repo_analysis"},
		PreferredAdapters:   []string{"codex"},
		ApprovalPresetID:    &presetID,
		RetrievalDefaults:   map[string]any{"mode": "keyword"},
	}); err != nil {
		t.Fatalf("SaveDossierProfile: %v", err)
	}
	insertEvaluationRuns(t, ctx, db, dossierID, "codex", []evaluationRun{
		{success: true, quality: 5},
		{success: true, quality: 4},
		{success: false, quality: 3},
	})

	recommendation, err := svc.Recommend(ctx, RecommendRequest{TaskType: "unknown", DossierID: &dossierID})
	if err != nil {
		t.Fatalf("Recommend: %v", err)
	}
	if recommendation.TaskType != "unknown" {
		t.Fatalf("TaskType=%q, want request task type persisted", recommendation.TaskType)
	}
	if recommendation.StrategyID == nil || *recommendation.StrategyID != "repo_analysis" {
		t.Fatalf("StrategyID=%v, want preferred repo_analysis", recommendation.StrategyID)
	}
	if recommendation.TargetAdapter != "codex" {
		t.Fatalf("TargetAdapter=%q, want profile adapter override", recommendation.TargetAdapter)
	}
	if recommendation.RetrievalMode != "keyword" {
		t.Fatalf("RetrievalMode=%q, want profile retrieval override", recommendation.RetrievalMode)
	}
	if recommendation.ApprovalPresetID == nil || *recommendation.ApprovalPresetID != "conservative" {
		t.Fatalf("ApprovalPresetID=%v, want profile preset override", recommendation.ApprovalPresetID)
	}
	if recommendation.Inferred {
		t.Fatalf("three direct evaluation runs should not be inferred")
	}
	if !recommendation.OperatorOverrideAllowed {
		t.Fatalf("operator override should remain allowed")
	}
	if !recommendation.ApprovalRequired {
		t.Fatalf("repo_analysis strategy should require approval")
	}
	if recommendation.Confidence < 0.65 || recommendation.Confidence > 0.75 {
		t.Fatalf("Confidence=%v, want evidence-weighted confidence around 0.7", recommendation.Confidence)
	}

	packetShape := decodeJSON[map[string]any](t, recommendation.PacketShape)
	if packetShape["retrievalMode"] != "keyword" {
		t.Fatalf("packet retrievalMode=%v, want keyword", packetShape["retrievalMode"])
	}
	if packetShape["requiresManualReviewForRisky"] != true {
		t.Fatalf("requiresManualReviewForRisky=%v, want true", packetShape["requiresManualReviewForRisky"])
	}
	if packetShape["targetItems"] != float64(10) || packetShape["maxItems"] != float64(18) {
		t.Fatalf("packet strategy sizing = %#v, want repo_analysis packet rules preserved", packetShape)
	}

	evidence := decodeJSON[map[string]any](t, recommendation.Evidence)
	if evidence["evaluationRuns"] != float64(3) {
		t.Fatalf("evaluationRuns=%v, want 3", evidence["evaluationRuns"])
	}

	all, err := svc.ListRecommendations(ctx, 0, nil)
	if err != nil {
		t.Fatalf("ListRecommendations all: %v", err)
	}
	if len(all) != 1 || all[0].ID != recommendation.ID {
		t.Fatalf("all recommendations=%#v, want only recommendation %d", all, recommendation.ID)
	}
	otherID := dossierID + 1
	filtered, err := svc.ListRecommendations(ctx, 10, &otherID)
	if err != nil {
		t.Fatalf("ListRecommendations filtered: %v", err)
	}
	if len(filtered) != 0 {
		t.Fatalf("filtered recommendations=%#v, want none for other dossier", filtered)
	}
}

func TestRecommendDefaultsAndValidation(t *testing.T) {
	ctx := context.Background()

	t.Run("global preset fills strategy without preset", func(t *testing.T) {
		svc, db := newTestService(t, ctx)
		if err := svc.SetGlobalApprovalPreset(ctx, " custom-global "); err != nil {
			t.Fatalf("SetGlobalApprovalPreset: %v", err)
		}
		strategySvc := strategies.New(db)
		if _, err := strategySvc.Save(ctx, strategies.SaveRequest{
			ID:               "no_preset",
			Name:             "No Preset",
			TaskType:         "blank_task",
			TargetAdapter:    "forge",
			RetrievalMode:    "keyword",
			PacketRules:      map[string]any{},
			ApprovalRequired: false,
			Enabled:          true,
		}); err != nil {
			t.Fatalf("Save strategy: %v", err)
		}

		recommendation, err := svc.Recommend(ctx, RecommendRequest{TaskType: "  ", StrategyID: stringPtr("no_preset")})
		if err != nil {
			t.Fatalf("Recommend: %v", err)
		}
		if recommendation.TaskType != "general" {
			t.Fatalf("blank task type persisted as %q, want general", recommendation.TaskType)
		}
		if recommendation.ApprovalPresetID == nil || *recommendation.ApprovalPresetID != "custom-global" {
			t.Fatalf("ApprovalPresetID=%v, want trimmed global preset", recommendation.ApprovalPresetID)
		}
		if !recommendation.Inferred {
			t.Fatalf("recommendation without evaluation history should be inferred")
		}
		if recommendation.Confidence != 0.45 {
			t.Fatalf("confidence=%v, want default 0.45", recommendation.Confidence)
		}
		packetShape := decodeJSON[map[string]any](t, recommendation.PacketShape)
		if packetShape["targetItems"] != float64(8) || packetShape["maxItems"] != float64(14) {
			t.Fatalf("default packet sizing=%#v, want targetItems=8 maxItems=14", packetShape)
		}
	})

	t.Run("empty global preset rejected", func(t *testing.T) {
		svc, _ := newTestService(t, ctx)
		if err := svc.SetGlobalApprovalPreset(ctx, "  "); err == nil {
			t.Fatalf("expected blank preset id error")
		}
	})
}

type evaluationRun struct {
	success bool
	quality int
}

func newTestService(t *testing.T, ctx context.Context) (*Service, *sql.DB) {
	t.Helper()

	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() {
		if err := st.Close(); err != nil {
			t.Fatalf("close store: %v", err)
		}
	})

	strategySvc := strategies.New(st.DB)
	if err := strategySvc.EnsureDefaults(ctx); err != nil {
		t.Fatalf("ensure strategy defaults: %v", err)
	}
	svc := New(st.DB, strategySvc)
	if err := svc.EnsureDefaults(ctx); err != nil {
		t.Fatalf("ensure policy defaults: %v", err)
	}
	return svc, st.DB
}

func insertDossier(t *testing.T, ctx context.Context, db *sql.DB) int64 {
	t.Helper()

	res, err := db.ExecContext(ctx, `
INSERT INTO dossiers(
  created_at, updated_at, name, description, primary_paths_json, related_repos_json,
  constraints_json, preferred_adapters_json, important_files_json, routing_notes
) VALUES(1,1,?,?,?,?,?,'[]','[]','')`,
		"dossier", "", "[]", "[]", "[]",
	)
	if err != nil {
		t.Fatalf("insert dossier: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("dossier id: %v", err)
	}
	return id
}

func insertEvaluationRuns(t *testing.T, ctx context.Context, db *sql.DB, dossierID int64, adapter string, runs []evaluationRun) {
	t.Helper()

	for idx, run := range runs {
		jobID := "job-" + string(rune('a'+idx))
		_, err := db.ExecContext(ctx, `
INSERT INTO jobs(
  id, created_at, updated_at, title, requested_action, target_adapter, initiating_source,
  execution_boundary, risk_class, status, approval_status, metadata_json
) VALUES(?,1,1,'job','action',?,'test','bounded','low','completed','approved','{}')`, jobID, adapter)
		if err != nil {
			t.Fatalf("insert job %d: %v", idx, err)
		}
		_, err = db.ExecContext(ctx, `
INSERT INTO evaluation_records(
  created_at, job_id, dossier_id, success, quality_rating, usefulness_rating,
  correctness_confidence, packet_quality_rating, adapter_suitability
) VALUES(1,?,?,?,?,5,5,5,5)`, jobID, dossierID, boolToInt(run.success), run.quality)
		if err != nil {
			t.Fatalf("insert evaluation %d: %v", idx, err)
		}
	}
}

func stringPtr(v string) *string {
	return &v
}

func decodeJSON[T any](t *testing.T, raw json.RawMessage) T {
	t.Helper()

	var out T
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode JSON %s: %v", string(raw), err)
	}
	return out
}

func assertJSONEqual(t *testing.T, raw json.RawMessage, want any) {
	t.Helper()

	var got any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("decode JSON %s: %v", string(raw), err)
	}
	wantJSON, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("marshal want: %v", err)
	}
	var normalizedWant any
	if err := json.Unmarshal(wantJSON, &normalizedWant); err != nil {
		t.Fatalf("decode want: %v", err)
	}
	if !jsonEqual(got, normalizedWant) {
		t.Fatalf("JSON=%#v, want %#v", got, normalizedWant)
	}
}

func jsonEqual(got, want any) bool {
	gotJSON, _ := json.Marshal(got)
	wantJSON, _ := json.Marshal(want)
	return string(gotJSON) == string(wantJSON)
}
