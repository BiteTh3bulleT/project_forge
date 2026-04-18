package release

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// BuildVersion is compiled into the forge-core binary at build time. Keep
// this in sync with release notes and packaging scripts.
const BuildVersion = "0.5.0"

// Service handles release readiness: recording release artifacts, tracking
// the pre-ship checklist, and reporting packaging status to the UI.
type Service struct {
	db           *sql.DB
	dataDir      string
	workspaceDir string
}

func New(db *sql.DB, dataDir, workspaceDir string) *Service {
	return &Service{db: db, dataDir: dataDir, workspaceDir: workspaceDir}
}

// ChecklistItem represents one pre-ship readiness check with its current
// state. The checklist is inspectable and not opinionated — it only reports
// what's true right now.
type ChecklistItem struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Status   string `json:"status"` // ok | warn | fail | pending
	Detail   string `json:"detail"`
	Category string `json:"category"`
}

type Checklist struct {
	Version     string          `json:"version"`
	GeneratedAt int64           `json:"generatedAtMs"`
	Platform    string          `json:"platform"`
	GoVersion   string          `json:"goVersion"`
	DataDir     string          `json:"dataDir"`
	Workspace   string          `json:"workspaceDir"`
	Items       []ChecklistItem `json:"items"`
	Ready       bool            `json:"ready"`
	FirstRun    bool            `json:"firstRun"`
}

// CheckReadiness inspects the local environment and returns a checklist.
func (s *Service) CheckReadiness(ctx context.Context) (*Checklist, error) {
	cl := &Checklist{
		Version:     BuildVersion,
		GeneratedAt: time.Now().UnixMilli(),
		Platform:    runtime.GOOS + "/" + runtime.GOARCH,
		GoVersion:   runtime.Version(),
		DataDir:     s.dataDir,
		Workspace:   s.workspaceDir,
	}
	items := []ChecklistItem{}

	items = append(items, dirItem("data.dir", "Data directory writable", s.dataDir, "storage"))
	items = append(items, dirItem("data.backups", "Backups directory present", filepath.Join(s.dataDir, "backups"), "storage"))
	items = append(items, dirItem("data.exports", "Exports directory present", filepath.Join(s.dataDir, "exports"), "storage"))
	items = append(items, dirItem("workspace", "Workspace directory accessible", s.workspaceDir, "workspace"))

	items = append(items, s.permissionsItem(ctx))
	items = append(items, s.sourcesItem(ctx))
	items = append(items, s.auditItem(ctx))
	items = append(items, s.laneItem(ctx))
	items = append(items, s.migrationItem(ctx))
	items = append(items, s.firstRunItem(ctx))

	ready := true
	firstRun := false
	for _, it := range items {
		if it.Status == "fail" {
			ready = false
		}
		if it.ID == "first_run" && it.Status == "pending" {
			firstRun = true
		}
	}
	cl.Items = items
	cl.Ready = ready
	cl.FirstRun = firstRun
	return cl, nil
}

// RecordArtifact captures a release artifact (e.g., a built binary or
// installer produced by the packaging script).
type ArtifactRequest struct {
	Kind       string   `json:"kind"`
	VersionTag string   `json:"versionTag"`
	Channel    string   `json:"channel"`
	Summary    string   `json:"summary"`
	Notes      string   `json:"notes"`
	Checklist  []string `json:"checklist"`
}

type Artifact struct {
	ID          int64           `json:"id"`
	CreatedAtMs int64           `json:"createdAtMs"`
	Kind        string          `json:"kind"`
	VersionTag  string          `json:"versionTag"`
	Channel     string          `json:"channel"`
	Status      string          `json:"status"`
	Summary     string          `json:"summary"`
	Checklist   json.RawMessage `json:"checklist"`
	Notes       string          `json:"notes"`
}

func (s *Service) RecordArtifact(ctx context.Context, req ArtifactRequest) (*Artifact, error) {
	kind := strings.TrimSpace(req.Kind)
	version := strings.TrimSpace(req.VersionTag)
	if kind == "" {
		return nil, errors.New("kind required")
	}
	if version == "" {
		version = BuildVersion
	}
	channel := req.Channel
	if strings.TrimSpace(channel) == "" {
		channel = "local"
	}
	checkBytes, _ := json.Marshal(nonNilStrings(req.Checklist))
	now := time.Now().UnixMilli()
	res, err := s.db.ExecContext(ctx, `
INSERT INTO release_artifacts(
  created_at, kind, version_tag, channel, status, summary, checklist_json, notes
) VALUES(?,?,?,?,?,?,?,?)`,
		now, kind, version, channel, "prepared", req.Summary, string(checkBytes), req.Notes,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.Get(ctx, id)
}

func (s *Service) List(ctx context.Context, limit int) ([]Artifact, error) {
	if limit <= 0 || limit > 300 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, created_at, kind, version_tag, channel, status, summary, checklist_json, notes
FROM release_artifacts ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Artifact{}
	for rows.Next() {
		var a Artifact
		var checklist string
		if err := rows.Scan(&a.ID, &a.CreatedAtMs, &a.Kind, &a.VersionTag, &a.Channel, &a.Status, &a.Summary, &checklist, &a.Notes); err != nil {
			return nil, err
		}
		a.Checklist = json.RawMessage(checklist)
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Service) Get(ctx context.Context, id int64) (*Artifact, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, created_at, kind, version_tag, channel, status, summary, checklist_json, notes
FROM release_artifacts WHERE id = ?`, id)
	var a Artifact
	var checklist string
	if err := row.Scan(&a.ID, &a.CreatedAtMs, &a.Kind, &a.VersionTag, &a.Channel, &a.Status, &a.Summary, &checklist, &a.Notes); err != nil {
		return nil, err
	}
	a.Checklist = json.RawMessage(checklist)
	return &a, nil
}

// FirstRunSummary reports whether the operator has completed the basic
// first-run setup steps. It's used by the UI to show or hide the onboarding
// guidance panel.
type FirstRunSummary struct {
	NeedsSetup       bool     `json:"needsSetup"`
	HasSources       bool     `json:"hasSources"`
	HasActiveProfile bool     `json:"hasActiveProfile"`
	HasDossiers      bool     `json:"hasDossiers"`
	NextSteps        []string `json:"nextSteps"`
}

func (s *Service) FirstRun(ctx context.Context) (*FirstRunSummary, error) {
	sum := &FirstRunSummary{}
	sources := 0
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sources`).Scan(&sources)
	dossiers := 0
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM dossiers`).Scan(&dossiers)
	profile := 0
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM permission_profiles WHERE active = 1`).Scan(&profile)

	sum.HasSources = sources > 0
	sum.HasDossiers = dossiers > 0
	sum.HasActiveProfile = profile > 0
	steps := []string{}
	if !sum.HasSources {
		steps = append(steps, "Add at least one source folder in Sources")
	}
	if !sum.HasActiveProfile {
		steps = append(steps, "Activate a permission profile in Permissions")
	}
	if !sum.HasDossiers {
		steps = append(steps, "Create a dossier in Dossiers")
	}
	sum.NextSteps = steps
	sum.NeedsSetup = len(steps) > 0
	return sum, nil
}

// --- check helpers ---

func dirItem(id, title, path, category string) ChecklistItem {
	if strings.TrimSpace(path) == "" {
		return ChecklistItem{ID: id, Title: title, Status: "fail", Detail: "path not set", Category: category}
	}
	info, err := os.Stat(path)
	if err != nil {
		return ChecklistItem{ID: id, Title: title, Status: "fail", Detail: err.Error(), Category: category}
	}
	if !info.IsDir() {
		return ChecklistItem{ID: id, Title: title, Status: "fail", Detail: "exists but not a directory", Category: category}
	}
	tmp := filepath.Join(path, ".forge-readiness-probe")
	if err := os.WriteFile(tmp, []byte("ok"), 0o644); err != nil {
		return ChecklistItem{ID: id, Title: title, Status: "warn", Detail: "directory not writable: " + err.Error(), Category: category}
	}
	_ = os.Remove(tmp)
	return ChecklistItem{ID: id, Title: title, Status: "ok", Detail: path, Category: category}
}

func (s *Service) permissionsItem(ctx context.Context) ChecklistItem {
	var count int
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM permission_profiles WHERE active = 1`).Scan(&count)
	if count == 0 {
		return ChecklistItem{ID: "permissions.active", Title: "Active permission profile", Status: "fail", Detail: "no profile active — activate one in Permissions", Category: "security"}
	}
	return ChecklistItem{ID: "permissions.active", Title: "Active permission profile", Status: "ok", Detail: fmt.Sprintf("%d active", count), Category: "security"}
}

func (s *Service) sourcesItem(ctx context.Context) ChecklistItem {
	var count int
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sources`).Scan(&count)
	if count == 0 {
		return ChecklistItem{ID: "sources.present", Title: "Sources configured", Status: "warn", Detail: "no sources — retrieval and memory are empty", Category: "workspace"}
	}
	return ChecklistItem{ID: "sources.present", Title: "Sources configured", Status: "ok", Detail: fmt.Sprintf("%d sources", count), Category: "workspace"}
}

func (s *Service) auditItem(ctx context.Context) ChecklistItem {
	var count int
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_records`).Scan(&count)
	return ChecklistItem{ID: "audit.history", Title: "Audit trail available", Status: "ok", Detail: fmt.Sprintf("%d records", count), Category: "audit"}
}

func (s *Service) laneItem(ctx context.Context) ChecklistItem {
	var count int
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM action_lanes WHERE enabled = 1`).Scan(&count)
	if count == 0 {
		return ChecklistItem{ID: "lanes.enabled", Title: "Action lanes enabled", Status: "fail", Detail: "no enabled lanes — gateway cannot execute", Category: "gateway"}
	}
	return ChecklistItem{ID: "lanes.enabled", Title: "Action lanes enabled", Status: "ok", Detail: fmt.Sprintf("%d enabled", count), Category: "gateway"}
}

func (s *Service) migrationItem(ctx context.Context) ChecklistItem {
	tables := []string{"jobs", "audit_records", "permission_profiles", "action_lanes", "gateway_invocations", "backup_bundles", "release_artifacts"}
	for _, t := range tables {
		var name string
		err := s.db.QueryRowContext(ctx, `SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, t).Scan(&name)
		if err != nil {
			return ChecklistItem{ID: "migrations", Title: "Database migrations", Status: "fail", Detail: "missing table " + t, Category: "storage"}
		}
	}
	return ChecklistItem{ID: "migrations", Title: "Database migrations", Status: "ok", Detail: "all Phase 5 tables present", Category: "storage"}
}

func (s *Service) firstRunItem(ctx context.Context) ChecklistItem {
	summary, err := s.FirstRun(ctx)
	if err != nil {
		return ChecklistItem{ID: "first_run", Title: "First-run setup", Status: "warn", Detail: err.Error(), Category: "onboarding"}
	}
	if summary.NeedsSetup {
		return ChecklistItem{ID: "first_run", Title: "First-run setup", Status: "pending", Detail: strings.Join(summary.NextSteps, "; "), Category: "onboarding"}
	}
	return ChecklistItem{ID: "first_run", Title: "First-run setup", Status: "ok", Detail: "operator setup complete", Category: "onboarding"}
}

func nonNilStrings(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}
