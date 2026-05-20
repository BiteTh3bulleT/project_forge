package lanes_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"forge/projectforge/services/core/internal/lanes"
	"forge/projectforge/services/core/internal/store"
)

func TestSaveRejectsMissingRequiredFields(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)

	tests := []struct {
		name    string
		lane    lanes.Lane
		wantErr string
	}{
		{
			name: "missing id",
			lane: lanes.Lane{
				Name:       "Custom lane",
				ActionType: "custom.action",
			},
			wantErr: "lane id required",
		},
		{
			name: "missing name",
			lane: lanes.Lane{
				ID:         "custom.action",
				ActionType: "custom.action",
			},
			wantErr: "lane name required",
		},
		{
			name: "blank id",
			lane: lanes.Lane{
				ID:         " \t",
				Name:       "Custom lane",
				ActionType: "custom.action",
			},
			wantErr: "lane id required",
		},
		{
			name: "blank name",
			lane: lanes.Lane{
				ID:         "custom.action",
				Name:       " \n",
				ActionType: "custom.action",
			},
			wantErr: "lane name required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Save(ctx, tt.lane)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Save() error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestEnsureDefaultsSeedsExpectedLaneInvariants(t *testing.T) {
	ctx := context.Background()
	workspaceDir := filepath.Join(t.TempDir(), "workspace")
	svc := newTestService(t)

	if err := svc.EnsureDefaults(ctx, workspaceDir); err != nil {
		t.Fatalf("EnsureDefaults() error = %v", err)
	}

	tests := []struct {
		id               string
		actionType       string
		writeIntent      bool
		requiresApproval bool
		riskClass        string
		maxBytes         int64
		artifacts        []string
		allowedPaths     []string
	}{
		{
			id:           "fs.read",
			actionType:   "fs.read",
			riskClass:    "read_only",
			artifacts:    []string{"fileContent"},
			allowedPaths: []string{workspaceDir},
		},
		{
			id:               "fs.write",
			actionType:       "fs.write",
			writeIntent:      true,
			requiresApproval: true,
			riskClass:        "safe_write",
			maxBytes:         512 * 1024,
			artifacts:        []string{"writtenFile"},
			allowedPaths:     []string{workspaceDir},
		},
		{
			id:               "fs.delete",
			actionType:       "fs.delete",
			writeIntent:      true,
			requiresApproval: true,
			riskClass:        "dangerous",
			artifacts:        []string{"pathDeleted"},
			allowedPaths:     []string{workspaceDir},
		},
		{
			id:               "legacy.adapter.invoke",
			actionType:       "legacy.adapter.invoke",
			riskClass:        "scoped_execute",
			artifacts:        []string{"legacyAdapterInvocation"},
			allowedPaths:     []string{filepath.Clean(string(filepath.Separator))},
		},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			got, err := svc.Get(ctx, tt.id)
			if err != nil {
				t.Fatalf("Get(%q) error = %v", tt.id, err)
			}
			if !got.Builtin || !got.Enabled {
				t.Fatalf("Get(%q) builtin/enabled = %v/%v, want true/true", tt.id, got.Builtin, got.Enabled)
			}
			if got.ActionType != tt.actionType {
				t.Fatalf("ActionType = %q, want %q", got.ActionType, tt.actionType)
			}
			if got.WriteIntent != tt.writeIntent {
				t.Fatalf("WriteIntent = %v, want %v", got.WriteIntent, tt.writeIntent)
			}
			if got.RequiresApproval != tt.requiresApproval {
				t.Fatalf("RequiresApproval = %v, want %v", got.RequiresApproval, tt.requiresApproval)
			}
			if got.RiskClass != tt.riskClass {
				t.Fatalf("RiskClass = %q, want %q", got.RiskClass, tt.riskClass)
			}
			if got.MaxBytes != tt.maxBytes {
				t.Fatalf("MaxBytes = %d, want %d", got.MaxBytes, tt.maxBytes)
			}
			if !reflect.DeepEqual(got.ExpectedArtifacts, tt.artifacts) {
				t.Fatalf("ExpectedArtifacts = %#v, want %#v", got.ExpectedArtifacts, tt.artifacts)
			}
			if !reflect.DeepEqual(got.AllowedPaths, tt.allowedPaths) {
				t.Fatalf("AllowedPaths = %#v, want %#v", got.AllowedPaths, tt.allowedPaths)
			}
		})
	}
}

func TestEnsureDefaultsSeedsAuthorityBoundaryMatrix(t *testing.T) {
	ctx := context.Background()
	workspaceDir := filepath.Join(t.TempDir(), "workspace")
	svc := newTestService(t)

	if err := svc.EnsureDefaults(ctx, workspaceDir); err != nil {
		t.Fatalf("EnsureDefaults() error = %v", err)
	}

	tests := []struct {
		id               string
		writeIntent      bool
		requiresApproval bool
		riskClass        string
		maxBytes         int64
		artifacts        []string
	}{
		{
			id:               "proc.run",
			requiresApproval: true,
			riskClass:        "scoped_execute",
		},
		{
			id:               "git.commit",
			writeIntent:      true,
			requiresApproval: true,
			riskClass:        "dangerous",
		},
		{
			id:               "git.write",
			writeIntent:      true,
			requiresApproval: true,
			riskClass:        "dangerous",
			maxBytes:         2 * 1024 * 1024,
		},
		{
			id:               "system.service_control",
			writeIntent:      true,
			requiresApproval: true,
			riskClass:        "privileged",
		},
		{
			id:        "system.logs",
			riskClass: "read_only",
		},
		{
			id:               "net.fetch",
			requiresApproval: true,
			riskClass:        "scoped_execute",
		},
		{
			id:        "net.dns_lookup",
			riskClass: "read_only",
		},
		{
			id:          "desktop.notify",
			writeIntent: true,
			riskClass:   "safe_write",
		},
		{
			id:               "desktop.open",
			writeIntent:      true,
			requiresApproval: true,
			riskClass:        "safe_write",
		},
		{
			id:               "secret.get",
			requiresApproval: true,
			riskClass:        "privileged",
		},
		{
			id:               "fs.copy",
			writeIntent:      true,
			requiresApproval: true,
			riskClass:        "safe_write",
			maxBytes:         2 * 1024 * 1024,
			artifacts:        []string{"pathCopied"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			got, err := svc.Get(ctx, tt.id)
			if err != nil {
				t.Fatalf("Get(%q) error = %v", tt.id, err)
			}
			if !got.Builtin || !got.Enabled {
				t.Fatalf("Get(%q) builtin/enabled = %v/%v, want true/true", tt.id, got.Builtin, got.Enabled)
			}
			if got.ActionType != tt.id {
				t.Fatalf("ActionType = %q, want %q", got.ActionType, tt.id)
			}
			if got.WriteIntent != tt.writeIntent {
				t.Fatalf("WriteIntent = %v, want %v", got.WriteIntent, tt.writeIntent)
			}
			if got.RequiresApproval != tt.requiresApproval {
				t.Fatalf("RequiresApproval = %v, want %v", got.RequiresApproval, tt.requiresApproval)
			}
			if got.RiskClass != tt.riskClass {
				t.Fatalf("RiskClass = %q, want %q", got.RiskClass, tt.riskClass)
			}
			if got.MaxBytes != tt.maxBytes {
				t.Fatalf("MaxBytes = %d, want %d", got.MaxBytes, tt.maxBytes)
			}
			if !reflect.DeepEqual(got.AllowedPaths, []string{workspaceDir}) {
				t.Fatalf("AllowedPaths = %#v, want workspace path %#v", got.AllowedPaths, []string{workspaceDir})
			}
			if tt.artifacts != nil && !reflect.DeepEqual(got.ExpectedArtifacts, tt.artifacts) {
				t.Fatalf("ExpectedArtifacts = %#v, want %#v", got.ExpectedArtifacts, tt.artifacts)
			}
		})
	}
}

func TestEnsureDefaultsIfEmptyPreservesExistingLaneState(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)

	created, err := svc.Save(ctx, lanes.Lane{
		ID:          "custom.inspect",
		Name:        "Custom inspect",
		Description: "operator-defined lane",
		ActionType:  "custom.inspect",
		RiskClass:   "read_only",
		Enabled:     true,
	})
	if err != nil {
		t.Fatalf("Save() custom lane error = %v", err)
	}

	if err := svc.EnsureDefaultsIfEmpty(ctx, filepath.Join(t.TempDir(), "workspace")); err != nil {
		t.Fatalf("EnsureDefaultsIfEmpty() error = %v", err)
	}

	got, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("List() returned %d lanes, want only existing custom lane", len(got))
	}
	if got[0].ID != created.ID || got[0].Description != created.Description {
		t.Fatalf("List()[0] = %#v, want preserved custom lane %#v", got[0], created)
	}
}

func TestSaveRoundTripsCustomLaneFieldsAndListOrdering(t *testing.T) {
	ctx := context.Background()
	workspaceDir := filepath.Join(t.TempDir(), "workspace")
	svc := newTestService(t)

	if err := svc.EnsureDefaults(ctx, workspaceDir); err != nil {
		t.Fatalf("EnsureDefaults() error = %v", err)
	}

	created, err := svc.Save(ctx, lanes.Lane{
		ID:                "aaa.custom",
		Name:              "AAA Custom",
		Description:       "custom lane metadata",
		ActionType:        "custom.action",
		AllowedPaths:      []string{filepath.Join(workspaceDir, "allowed")},
		ForbiddenPaths:    []string{filepath.Join(workspaceDir, "forbidden")},
		WriteIntent:       true,
		RequiresApproval:  true,
		RiskClass:         "safe_write",
		MaxBytes:          4096,
		ExpectedArtifacts: []string{"customArtifact"},
		Enabled:           true,
	})
	if err != nil {
		t.Fatalf("Save() custom lane error = %v", err)
	}

	got, err := svc.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get(%q) error = %v", created.ID, err)
	}
	if got.Builtin {
		t.Fatal("Builtin = true, want custom lane to remain operator-defined")
	}
	if got.ActionType != "custom.action" || got.RiskClass != "safe_write" {
		t.Fatalf("custom lane action/risk = %q/%q, want custom.action/safe_write", got.ActionType, got.RiskClass)
	}
	if !got.WriteIntent || !got.RequiresApproval || !got.Enabled {
		t.Fatalf("custom lane booleans = write:%v approval:%v enabled:%v, want true/true/true", got.WriteIntent, got.RequiresApproval, got.Enabled)
	}
	if got.MaxBytes != 4096 {
		t.Fatalf("MaxBytes = %d, want 4096", got.MaxBytes)
	}
	if !reflect.DeepEqual(got.AllowedPaths, []string{filepath.Join(workspaceDir, "allowed")}) {
		t.Fatalf("AllowedPaths = %#v, want saved path", got.AllowedPaths)
	}
	if !reflect.DeepEqual(got.ForbiddenPaths, []string{filepath.Join(workspaceDir, "forbidden")}) {
		t.Fatalf("ForbiddenPaths = %#v, want saved path", got.ForbiddenPaths)
	}
	if !reflect.DeepEqual(got.ExpectedArtifacts, []string{"customArtifact"}) {
		t.Fatalf("ExpectedArtifacts = %#v, want saved artifact", got.ExpectedArtifacts)
	}

	listed, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	seenCustom := false
	for _, lane := range listed {
		if lane.ID == created.ID {
			seenCustom = true
		}
		if seenCustom && lane.Builtin {
			t.Fatalf("List() returned builtin lane %q after custom lanes", lane.ID)
		}
	}
	if !seenCustom {
		t.Fatalf("List() did not include custom lane %q", created.ID)
	}
}

func TestSavePreservesBuiltinAuthorityFields(t *testing.T) {
	ctx := context.Background()
	workspaceDir := filepath.Join(t.TempDir(), "workspace")
	svc := newTestService(t)

	if err := svc.EnsureDefaults(ctx, workspaceDir); err != nil {
		t.Fatalf("EnsureDefaults() error = %v", err)
	}
	before, err := svc.Get(ctx, "fs.read")
	if err != nil {
		t.Fatalf("Get(fs.read) before save error = %v", err)
	}

	updated, err := svc.Save(ctx, lanes.Lane{
		ID:               before.ID,
		Name:             "Renamed read lane",
		Description:      "operator-visible metadata update",
		ActionType:       "attempted.retype",
		AllowedPaths:     []string{filepath.Join(t.TempDir(), "other")},
		ForbiddenPaths:   []string{"/tmp/nope"},
		WriteIntent:      true,
		RequiresApproval: true,
		RiskClass:        "dangerous",
		Builtin:          false,
		Enabled:          false,
	})
	if err != nil {
		t.Fatalf("Save() builtin lane error = %v", err)
	}

	if updated.Name != "Renamed read lane" || updated.Description != "operator-visible metadata update" {
		t.Fatalf("editable fields were not updated: %#v", updated)
	}
	if !updated.Builtin {
		t.Fatal("Builtin = false, want built-in status preserved")
	}
	if updated.CreatedAtMs != before.CreatedAtMs {
		t.Fatalf("CreatedAtMs = %d, want preserved %d", updated.CreatedAtMs, before.CreatedAtMs)
	}
	if updated.ActionType != before.ActionType {
		t.Fatalf("ActionType = %q, want preserved %q", updated.ActionType, before.ActionType)
	}
	if !reflect.DeepEqual(updated.AllowedPaths, before.AllowedPaths) {
		t.Fatalf("AllowedPaths = %#v, want preserved %#v", updated.AllowedPaths, before.AllowedPaths)
	}
	if !reflect.DeepEqual(updated.ForbiddenPaths, before.ForbiddenPaths) {
		t.Fatalf("ForbiddenPaths = %#v, want preserved %#v", updated.ForbiddenPaths, before.ForbiddenPaths)
	}
	if updated.Enabled {
		t.Fatal("Enabled = true, want operator-disabled builtin lane to stay disabled")
	}
}

func TestDeleteRejectsBuiltinsAndDeletesCustomLanes(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)

	if err := svc.EnsureDefaults(ctx, filepath.Join(t.TempDir(), "workspace")); err != nil {
		t.Fatalf("EnsureDefaults() error = %v", err)
	}
	if err := svc.Delete(ctx, "fs.read"); err == nil || !strings.Contains(err.Error(), "built-in") {
		t.Fatalf("Delete(fs.read) error = %v, want built-in rejection", err)
	}

	if _, err := svc.Save(ctx, lanes.Lane{
		ID:         "custom.delete",
		Name:       "Custom delete",
		ActionType: "custom.delete",
		RiskClass:  "read_only",
		Enabled:    true,
	}); err != nil {
		t.Fatalf("Save() custom lane error = %v", err)
	}
	if err := svc.Delete(ctx, "custom.delete"); err != nil {
		t.Fatalf("Delete(custom.delete) error = %v", err)
	}
	if _, err := svc.Get(ctx, "custom.delete"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("Get(custom.delete) error = %v, want sql.ErrNoRows", err)
	}
}

func newTestService(t *testing.T) *lanes.Service {
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
	return lanes.New(st.DB)
}
