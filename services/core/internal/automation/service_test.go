package automation

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"forge/projectforge/services/core/internal/store"
)

func TestEnsureDefaultsIsIdempotent(t *testing.T) {
	ctx := context.Background()
	svc, cleanup := newTestService(t)
	defer cleanup()

	if err := svc.EnsureDefaults(ctx); err != nil {
		t.Fatalf("first EnsureDefaults failed: %v", err)
	}
	if err := svc.EnsureDefaults(ctx); err != nil {
		t.Fatalf("second EnsureDefaults failed: %v", err)
	}
	rules, err := svc.ListRules(ctx, false, 10)
	if err != nil {
		t.Fatalf("ListRules failed: %v", err)
	}
	if len(rules) != 3 {
		t.Fatalf("default rules = %d, want 3", len(rules))
	}
}

func TestSaveUpdateAndListRules(t *testing.T) {
	ctx := context.Background()
	svc, cleanup := newTestService(t)
	defer cleanup()

	disabled, err := svc.SaveRule(ctx, SaveRuleRequest{
		Name:      "  disabled rule  ",
		Trigger:   " source.changed ",
		Condition: map[string]any{"pathContains": "docs/"},
		Action:    map[string]any{"type": "noop"},
		Enabled:   false,
	})
	if err != nil {
		t.Fatalf("SaveRule disabled failed: %v", err)
	}
	enabled, err := svc.SaveRule(ctx, SaveRuleRequest{
		Name:          "enabled rule",
		Trigger:       "project_context.changed",
		Condition:     map[string]any{"always": true},
		Action:        map[string]any{"type": "create_job"},
		Scope:         map[string]any{"workspace": "main"},
		Enabled:       true,
		DryRunDefault: true,
	})
	if err != nil {
		t.Fatalf("SaveRule enabled failed: %v", err)
	}
	if disabled.Name != "disabled rule" || disabled.Trigger != "source.changed" {
		t.Fatalf("SaveRule did not trim name/trigger: %+v", disabled)
	}

	renamed := "renamed enabled rule"
	updated, err := svc.SaveRule(ctx, SaveRuleRequest{
		ID:            &enabled.ID,
		Name:          renamed,
		Trigger:       enabled.Trigger,
		Condition:     map[string]any{"always": true},
		Action:        map[string]any{"type": "create_review"},
		Scope:         map[string]any{},
		Enabled:       true,
		DryRunDefault: false,
	})
	if err != nil {
		t.Fatalf("SaveRule update failed: %v", err)
	}
	if updated.Name != renamed || updated.DryRunDefault {
		t.Fatalf("updated rule = %+v, want renamed with dryRunDefault false", updated)
	}

	const sameUpdatedAt = int64(987654321)
	if _, err := svc.db.ExecContext(ctx, `UPDATE automation_rules SET updated_at = ? WHERE id IN (?, ?)`, sameUpdatedAt, disabled.ID, enabled.ID); err != nil {
		t.Fatalf("force same updated_at failed: %v", err)
	}
	all, err := svc.ListRules(ctx, false, 10)
	if err != nil {
		t.Fatalf("ListRules all failed: %v", err)
	}
	if got := ruleIDs(all); !reflect.DeepEqual(got, []int64{enabled.ID, disabled.ID}) {
		t.Fatalf("all rule order = %v, want id desc tie-breaker", got)
	}
	enabledOnly, err := svc.ListRules(ctx, true, 10)
	if err != nil {
		t.Fatalf("ListRules enabled failed: %v", err)
	}
	if got := ruleIDs(enabledOnly); !reflect.DeepEqual(got, []int64{enabled.ID}) {
		t.Fatalf("enabled rule ids = %v, want [%d]", got, enabled.ID)
	}
}

func TestRunRecordsDryRunExecutionAndFailureHistory(t *testing.T) {
	ctx := context.Background()
	svc, cleanup := newTestService(t)
	defer cleanup()

	rule, err := svc.SaveRule(ctx, SaveRuleRequest{
		Name:          "run rule",
		Trigger:       "source.changed",
		Condition:     map[string]any{"always": true},
		Action:        map[string]any{"type": "create_job"},
		Scope:         map[string]any{"dossierId": float64(7)},
		Enabled:       true,
		DryRunDefault: true,
	})
	if err != nil {
		t.Fatalf("SaveRule failed: %v", err)
	}

	execCalls := 0
	dryRun, err := svc.Run(ctx, RunRequest{RuleID: rule.ID, Trigger: "source.changed"}, func(context.Context, map[string]any, map[string]any, bool) (map[string]any, error) {
		execCalls++
		return map[string]any{"unexpected": true}, nil
	})
	if err != nil {
		t.Fatalf("Run dry-run failed: %v", err)
	}
	if !dryRun.Matched || !dryRun.DryRun || dryRun.Executed || execCalls != 0 {
		t.Fatalf("dry-run result = %+v execCalls=%d, want matched dry-run without execution", dryRun, execCalls)
	}

	runForReal := false
	executed, err := svc.Run(ctx, RunRequest{RuleID: rule.ID, Trigger: "source.changed", DryRun: &runForReal}, func(_ context.Context, action map[string]any, scope map[string]any, dryRun bool) (map[string]any, error) {
		execCalls++
		if action["type"] != "create_job" || scope["dossierId"] != float64(7) || dryRun {
			t.Fatalf("executor inputs action=%v scope=%v dryRun=%v", action, scope, dryRun)
		}
		return map[string]any{"jobId": float64(123)}, nil
	})
	if err != nil {
		t.Fatalf("Run execute failed: %v", err)
	}
	if !executed.Executed || executed.Execution["jobId"] != float64(123) {
		t.Fatalf("executed result = %+v, want job id", executed)
	}

	_, err = svc.Run(ctx, RunRequest{RuleID: rule.ID, Trigger: "source.changed", DryRun: &runForReal}, func(context.Context, map[string]any, map[string]any, bool) (map[string]any, error) {
		return nil, errors.New("executor failed")
	})
	if err != nil {
		t.Fatalf("Run failed executor should record history without returning error: %v", err)
	}
	skipped, err := svc.Run(ctx, RunRequest{RuleID: rule.ID, Trigger: "other.trigger"}, nil)
	if err != nil {
		t.Fatalf("Run skipped failed: %v", err)
	}
	if skipped.Matched || skipped.Executed {
		t.Fatalf("skipped result = %+v, want unmatched and unexecuted", skipped)
	}

	history, err := svc.ListHistory(ctx, 10)
	if err != nil {
		t.Fatalf("ListHistory failed: %v", err)
	}
	statuses := make([]string, 0, len(history))
	for _, h := range history {
		statuses = append(statuses, h.Status)
		if len(h.Preview) == 0 || !json.Valid(h.Preview) {
			t.Fatalf("history preview is not valid json: %q", string(h.Preview))
		}
		if len(h.Result) == 0 || !json.Valid(h.Result) {
			t.Fatalf("history result is not valid json: %q", string(h.Result))
		}
	}
	wantStatuses := []string{"skipped", "failed", "executed", "preview_ready"}
	if !reflect.DeepEqual(statuses, wantStatuses) {
		t.Fatalf("history statuses = %v, want newest-first %v", statuses, wantStatuses)
	}
}

func TestRunEvaluatesPersistedConditions(t *testing.T) {
	ctx := context.Background()
	svc, cleanup := newTestService(t)
	defer cleanup()

	tests := []struct {
		name      string
		condition map[string]any
		runCtx    map[string]any
		wantMatch bool
	}{
		{
			name:      "always false blocks match",
			condition: map[string]any{"always": false},
			runCtx:    map[string]any{"path": "docs/readme.md", "source": "project_context"},
			wantMatch: false,
		},
		{
			name:      "path contains matches",
			condition: map[string]any{"pathContains": "docs/"},
			runCtx:    map[string]any{"path": "docs/readme.md"},
			wantMatch: true,
		},
		{
			name:      "path contains rejects other path",
			condition: map[string]any{"pathContains": "docs/"},
			runCtx:    map[string]any{"path": "src/main.go"},
			wantMatch: false,
		},
		{
			name:      "source matches",
			condition: map[string]any{"source": "project_context"},
			runCtx:    map[string]any{"source": "project_context"},
			wantMatch: true,
		},
		{
			name:      "source rejects mismatch",
			condition: map[string]any{"source": "project_context"},
			runCtx:    map[string]any{"source": "manual"},
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule, err := svc.SaveRule(ctx, SaveRuleRequest{
				Name:          tt.name,
				Trigger:       "source.changed",
				Condition:     tt.condition,
				Action:        map[string]any{"type": "create_job"},
				Enabled:       true,
				DryRunDefault: true,
			})
			if err != nil {
				t.Fatalf("SaveRule failed: %v", err)
			}
			result, err := svc.Run(ctx, RunRequest{RuleID: rule.ID, Trigger: "source.changed", Context: tt.runCtx}, nil)
			if err != nil {
				t.Fatalf("Run failed: %v", err)
			}
			if result.Matched != tt.wantMatch {
				t.Fatalf("matched = %v, want %v", result.Matched, tt.wantMatch)
			}
		})
	}
}

func ruleIDs(rules []Rule) []int64 {
	out := make([]int64, 0, len(rules))
	for _, rule := range rules {
		out = append(out, rule.ID)
	}
	return out
}

func newTestService(t *testing.T) (*Service, func()) {
	t.Helper()
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	return New(st.DB), func() {
		if err := st.Close(); err != nil {
			t.Fatalf("close store: %v", err)
		}
	}
}
