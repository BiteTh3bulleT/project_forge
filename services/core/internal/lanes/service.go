package lanes

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// Lane is a bounded local operation that declares its action type, scope,
// write intent, risk, and expected artifacts. Lanes are how FORGE exposes
// "safe" read-only paths without letting callers freestyle. Every gateway
// invocation must bind to a lane.
type Lane struct {
	ID                string   `json:"id"`
	CreatedAtMs       int64    `json:"createdAtMs"`
	UpdatedAtMs       int64    `json:"updatedAtMs"`
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	ActionType        string   `json:"actionType"`
	AllowedPaths      []string `json:"allowedPaths"`
	ForbiddenPaths    []string `json:"forbiddenPaths"`
	WriteIntent       bool     `json:"writeIntent"`
	RequiresApproval  bool     `json:"requiresApproval"`
	RiskClass         string   `json:"riskClass"`
	MaxBytes          int64    `json:"maxBytes"`
	ExpectedArtifacts []string `json:"expectedArtifacts"`
	Builtin           bool     `json:"builtin"`
	Enabled           bool     `json:"enabled"`
}

type Service struct {
	db *sql.DB
}

func New(db *sql.DB) *Service { return &Service{db: db} }

// EnsureDefaults seeds the built-in, inspectable action lanes. These cannot
// be deleted; they can be disabled.
func (s *Service) EnsureDefaults(ctx context.Context, workspaceDir string) error {
	for _, l := range defaultBuiltins(workspaceDir, time.Now().UnixMilli()) {
		if err := s.upsert(ctx, l); err != nil {
			return err
		}
	}
	return nil
}

func defaultBuiltins(workspaceDir string, now int64) []Lane {
	builtins := []Lane{
		{
			ID:                "fs.read",
			Name:              "Read workspace files",
			Description:       "Read files inside the approved workspace scope. Read-only.",
			ActionType:        "fs.read",
			AllowedPaths:      []string{workspaceDir},
			WriteIntent:       false,
			RequiresApproval:  false,
			RiskClass:         "read_only",
			MaxBytes:          0,
			ExpectedArtifacts: []string{"fileContent"},
			Builtin:           true,
			Enabled:           true,
		},
		{
			ID:                "fs.list",
			Name:              "List workspace directory",
			Description:       "List files and directories inside the approved workspace scope.",
			ActionType:        "fs.list",
			AllowedPaths:      []string{workspaceDir},
			WriteIntent:       false,
			RequiresApproval:  false,
			RiskClass:         "read_only",
			ExpectedArtifacts: []string{"directoryListing"},
			Builtin:           true,
			Enabled:           true,
		},
		{
			ID:                "repo.inspect",
			Name:              "Repo inspection",
			Description:       "Run safe read-only repo inspection commands (ls, status, counts).",
			ActionType:        "repo.inspect",
			AllowedPaths:      []string{workspaceDir},
			WriteIntent:       false,
			RequiresApproval:  false,
			RiskClass:         "read_only",
			ExpectedArtifacts: []string{"inspectionReport"},
			Builtin:           true,
			Enabled:           true,
		},
		{
			ID:                "git.status",
			Name:              "Git status",
			Description:       "Report git status/branch for the workspace. Read-only.",
			ActionType:        "git.status",
			AllowedPaths:      []string{workspaceDir},
			WriteIntent:       false,
			RequiresApproval:  false,
			RiskClass:         "read_only",
			ExpectedArtifacts: []string{"gitStatus"},
			Builtin:           true,
			Enabled:           true,
		},
		{
			ID:                "git.diff",
			Name:              "Git diff",
			Description:       "Return unified diff for workspace changes. Read-only.",
			ActionType:        "git.diff",
			AllowedPaths:      []string{workspaceDir},
			WriteIntent:       false,
			RequiresApproval:  false,
			RiskClass:         "read_only",
			ExpectedArtifacts: []string{"diffPatch"},
			Builtin:           true,
			Enabled:           true,
		},
		{
			ID:                "git.branch",
			Name:              "Git branch list",
			Description:       "List branches for the workspace repository. Read-only.",
			ActionType:        "git.branch",
			AllowedPaths:      []string{workspaceDir},
			WriteIntent:       false,
			RequiresApproval:  false,
			RiskClass:         "read_only",
			ExpectedArtifacts: []string{"gitBranches"},
			Builtin:           true,
			Enabled:           true,
		},
		{
			ID:                "fs.write.bounded",
			Name:              "Bounded workspace write",
			Description:       "Write files inside approved write scope. Requires approval for non-low risk. Size-capped.",
			ActionType:        "fs.write",
			AllowedPaths:      []string{workspaceDir},
			WriteIntent:       true,
			RequiresApproval:  true,
			RiskClass:         "safe_write",
			MaxBytes:          512 * 1024,
			ExpectedArtifacts: []string{"writtenFile"},
			Builtin:           true,
			Enabled:           true,
		},
		{
			ID:                "fs.write",
			Name:              "Write file",
			Description:       "Write file content within approved workspace scope.",
			ActionType:        "fs.write",
			AllowedPaths:      []string{workspaceDir},
			WriteIntent:       true,
			RequiresApproval:  true,
			RiskClass:         "safe_write",
			MaxBytes:          512 * 1024,
			ExpectedArtifacts: []string{"writtenFile"},
			Builtin:           true,
			Enabled:           true,
		},
		{
			ID:                "fs.mkdir",
			Name:              "Make directory",
			Description:       "Create directory within approved workspace scope.",
			ActionType:        "fs.mkdir",
			AllowedPaths:      []string{workspaceDir},
			WriteIntent:       true,
			RequiresApproval:  false,
			RiskClass:         "safe_write",
			MaxBytes:          0,
			ExpectedArtifacts: []string{"directoryCreated"},
			Builtin:           true,
			Enabled:           true,
		},
		{
			ID:                "fs.rename",
			Name:              "Rename or move path",
			Description:       "Rename or move file/directory within approved workspace scope.",
			ActionType:        "fs.rename",
			AllowedPaths:      []string{workspaceDir},
			WriteIntent:       true,
			RequiresApproval:  true,
			RiskClass:         "safe_write",
			MaxBytes:          0,
			ExpectedArtifacts: []string{"pathRenamed"},
			Builtin:           true,
			Enabled:           true,
		},
		{
			ID:                "fs.copy",
			Name:              "Copy path",
			Description:       "Copy file from source to destination within approved workspace scope.",
			ActionType:        "fs.copy",
			AllowedPaths:      []string{workspaceDir},
			WriteIntent:       true,
			RequiresApproval:  true,
			RiskClass:         "safe_write",
			MaxBytes:          2 * 1024 * 1024,
			ExpectedArtifacts: []string{"pathCopied"},
			Builtin:           true,
			Enabled:           true,
		},
		{
			ID:                "fs.chmod",
			Name:              "Change file mode",
			Description:       "Modify mode bits for approved workspace paths.",
			ActionType:        "fs.chmod",
			AllowedPaths:      []string{workspaceDir},
			WriteIntent:       true,
			RequiresApproval:  true,
			RiskClass:         "privileged",
			MaxBytes:          0,
			ExpectedArtifacts: []string{"modeChanged"},
			Builtin:           true,
			Enabled:           true,
		},
		{
			ID:                "fs.delete",
			Name:              "Delete path",
			Description:       "Delete file/directory within approved workspace scope.",
			ActionType:        "fs.delete",
			AllowedPaths:      []string{workspaceDir},
			WriteIntent:       true,
			RequiresApproval:  true,
			RiskClass:         "dangerous",
			MaxBytes:          0,
			ExpectedArtifacts: []string{"pathDeleted"},
			Builtin:           true,
			Enabled:           true,
		},
		{
			ID:                "export.dossier",
			Name:              "Export dossier bundle",
			Description:       "Export a dossier as a portable bundle artifact.",
			ActionType:        "export.dossier",
			AllowedPaths:      []string{workspaceDir},
			WriteIntent:       true,
			RequiresApproval:  false,
			RiskClass:         "safe_write",
			MaxBytes:          4 * 1024 * 1024,
			ExpectedArtifacts: []string{"exportBundle"},
			Builtin:           true,
			Enabled:           true,
		},
		{
			ID:                "export.packet",
			Name:              "Export task packet",
			Description:       "Export a task packet as a portable JSON artifact.",
			ActionType:        "export.packet",
			AllowedPaths:      []string{workspaceDir},
			WriteIntent:       true,
			RequiresApproval:  false,
			RiskClass:         "safe_write",
			MaxBytes:          1 * 1024 * 1024,
			ExpectedArtifacts: []string{"exportBundle"},
			Builtin:           true,
			Enabled:           true,
		},
		{
			ID:                "export.audit",
			Name:              "Export audit log",
			Description:       "Export the audit trail as a portable JSON artifact for external review.",
			ActionType:        "export.audit",
			AllowedPaths:      []string{workspaceDir},
			WriteIntent:       true,
			RequiresApproval:  false,
			RiskClass:         "safe_write",
			MaxBytes:          4 * 1024 * 1024,
			ExpectedArtifacts: []string{"auditBundle"},
			Builtin:           true,
			Enabled:           true,
		},
		{
			ID:                "validate.project_context",
			Name:              "Validate project context",
			Description:       "Validate project context artifacts for completeness. Read-only.",
			ActionType:        "validate.project_context",
			AllowedPaths:      []string{workspaceDir},
			WriteIntent:       false,
			RequiresApproval:  false,
			RiskClass:         "read_only",
			ExpectedArtifacts: []string{"validationReport"},
			Builtin:           true,
			Enabled:           true,
		},
		{
			ID:               "proc.run",
			Name:             "Run bounded command",
			Description:      "Execute a scoped command with timeout and captured output.",
			ActionType:       "proc.run",
			AllowedPaths:     []string{workspaceDir},
			WriteIntent:      false,
			RequiresApproval: true,
			RiskClass:        "scoped_execute",
			Builtin:          true,
			Enabled:          true,
		},
		{
			ID:               "proc.terminate",
			Name:             "Terminate process",
			Description:      "Terminate a PID with controlled signal semantics.",
			ActionType:       "proc.terminate",
			AllowedPaths:     []string{workspaceDir},
			WriteIntent:      false,
			RequiresApproval: true,
			RiskClass:        "privileged",
			Builtin:          true,
			Enabled:          true,
		},
		{
			ID:               "git.write",
			Name:             "Git write operations",
			Description:      "Mutating git operations (commit/checkout/stash/apply).",
			ActionType:       "git.write",
			AllowedPaths:     []string{workspaceDir},
			WriteIntent:      true,
			RequiresApproval: true,
			RiskClass:        "dangerous",
			MaxBytes:         2 * 1024 * 1024,
			Builtin:          true,
			Enabled:          true,
		},
		{
			ID:               "git.commit",
			Name:             "Git commit",
			Description:      "Create commit with provided message.",
			ActionType:       "git.commit",
			AllowedPaths:     []string{workspaceDir},
			WriteIntent:      true,
			RequiresApproval: true,
			RiskClass:        "dangerous",
			Builtin:          true,
			Enabled:          true,
		},
		{
			ID:               "git.checkout",
			Name:             "Git checkout",
			Description:      "Checkout branch/ref in workspace repository.",
			ActionType:       "git.checkout",
			AllowedPaths:     []string{workspaceDir},
			WriteIntent:      true,
			RequiresApproval: true,
			RiskClass:        "dangerous",
			Builtin:          true,
			Enabled:          true,
		},
		{
			ID:               "git.stash",
			Name:             "Git stash",
			Description:      "Stash local changes with optional message.",
			ActionType:       "git.stash",
			AllowedPaths:     []string{workspaceDir},
			WriteIntent:      true,
			RequiresApproval: true,
			RiskClass:        "safe_write",
			Builtin:          true,
			Enabled:          true,
		},
		{
			ID:               "git.apply_patch",
			Name:             "Git apply patch",
			Description:      "Apply patch content through git apply.",
			ActionType:       "git.apply_patch",
			AllowedPaths:     []string{workspaceDir},
			WriteIntent:      true,
			RequiresApproval: true,
			RiskClass:        "dangerous",
			Builtin:          true,
			Enabled:          true,
		},
		{
			ID:               "system.privileged",
			Name:             "System privileged actions",
			Description:      "Service control and privileged system operations.",
			ActionType:       "system.privileged",
			AllowedPaths:     []string{workspaceDir},
			WriteIntent:      true,
			RequiresApproval: true,
			RiskClass:        "privileged",
			Builtin:          true,
			Enabled:          true,
		},
		{
			ID:               "system.service_status",
			Name:             "Service status",
			Description:      "Inspect service state through systemctl status.",
			ActionType:       "system.service_status",
			AllowedPaths:     []string{workspaceDir},
			WriteIntent:      false,
			RequiresApproval: false,
			RiskClass:        "read_only",
			Builtin:          true,
			Enabled:          true,
		},
		{
			ID:               "system.service_control",
			Name:             "Service control",
			Description:      "Start/stop/restart service operations.",
			ActionType:       "system.service_control",
			AllowedPaths:     []string{workspaceDir},
			WriteIntent:      true,
			RequiresApproval: true,
			RiskClass:        "privileged",
			Builtin:          true,
			Enabled:          true,
		},
		{
			ID:               "system.logs",
			Name:             "System logs",
			Description:      "Inspect recent service logs via journalctl.",
			ActionType:       "system.logs",
			AllowedPaths:     []string{workspaceDir},
			WriteIntent:      false,
			RequiresApproval: false,
			RiskClass:        "read_only",
			Builtin:          true,
			Enabled:          true,
		},
		{
			ID:               "network.inspect",
			Name:             "Network inspection",
			Description:      "Interface/dns/connectivity/fetch actions.",
			ActionType:       "network.inspect",
			AllowedPaths:     []string{workspaceDir},
			WriteIntent:      false,
			RequiresApproval: true,
			RiskClass:        "scoped_execute",
			Builtin:          true,
			Enabled:          true,
		},
		{
			ID:               "net.interfaces",
			Name:             "Network interfaces",
			Description:      "List local network interfaces.",
			ActionType:       "net.interfaces",
			AllowedPaths:     []string{workspaceDir},
			WriteIntent:      false,
			RequiresApproval: false,
			RiskClass:        "read_only",
			Builtin:          true,
			Enabled:          true,
		},
		{
			ID:               "net.connectivity",
			Name:             "Connectivity check",
			Description:      "Perform controlled network connectivity checks.",
			ActionType:       "net.connectivity",
			AllowedPaths:     []string{workspaceDir},
			WriteIntent:      false,
			RequiresApproval: true,
			RiskClass:        "scoped_execute",
			Builtin:          true,
			Enabled:          true,
		},
		{
			ID:               "net.dns_lookup",
			Name:             "DNS lookup",
			Description:      "Resolve hostname using configured DNS.",
			ActionType:       "net.dns_lookup",
			AllowedPaths:     []string{workspaceDir},
			WriteIntent:      false,
			RequiresApproval: false,
			RiskClass:        "read_only",
			Builtin:          true,
			Enabled:          true,
		},
		{
			ID:               "net.fetch",
			Name:             "HTTP fetch",
			Description:      "Fetch approved URL content over network.",
			ActionType:       "net.fetch",
			AllowedPaths:     []string{workspaceDir},
			WriteIntent:      false,
			RequiresApproval: true,
			RiskClass:        "scoped_execute",
			Builtin:          true,
			Enabled:          true,
		},
		{
			ID:               "time.now",
			Name:             "System clock read",
			Description:      "Read current system time without side effects.",
			ActionType:       "time.now",
			AllowedPaths:     []string{workspaceDir},
			WriteIntent:      false,
			RequiresApproval: false,
			RiskClass:        "read_only",
			Builtin:          true,
			Enabled:          true,
		},
		{
			ID:               "desktop.session",
			Name:             "Desktop session actions",
			Description:      "Desktop notification/open actions.",
			ActionType:       "desktop.session",
			AllowedPaths:     []string{workspaceDir},
			WriteIntent:      true,
			RequiresApproval: true,
			RiskClass:        "safe_write",
			Builtin:          true,
			Enabled:          true,
		},
		{
			ID:               "desktop.notify",
			Name:             "Desktop notification",
			Description:      "Send local desktop notification.",
			ActionType:       "desktop.notify",
			AllowedPaths:     []string{workspaceDir},
			WriteIntent:      true,
			RequiresApproval: false,
			RiskClass:        "safe_write",
			Builtin:          true,
			Enabled:          true,
		},
		{
			ID:               "desktop.open",
			Name:             "Desktop open path",
			Description:      "Open file/folder path in local desktop session.",
			ActionType:       "desktop.open",
			AllowedPaths:     []string{workspaceDir},
			WriteIntent:      true,
			RequiresApproval: true,
			RiskClass:        "safe_write",
			Builtin:          true,
			Enabled:          true,
		},
		{
			ID:               "secret.get",
			Name:             "Secret retrieval",
			Description:      "Retrieve secret reference by logical name.",
			ActionType:       "secret.get",
			AllowedPaths:     []string{workspaceDir},
			WriteIntent:      false,
			RequiresApproval: true,
			RiskClass:        "privileged",
			Builtin:          true,
			Enabled:          true,
		},
		{
			ID:                "legacy.adapter.invoke",
			Name:              "Legacy adapter invoke compatibility",
			Description:       "Deprecated gateway-only legacy adapter compatibility lane; network/model behavior requires profile and approval gates.",
			ActionType:        "legacy.adapter.invoke",
			AllowedPaths:      []string{filepath.Clean(string(filepath.Separator))},
			WriteIntent:       false,
			RequiresApproval:  false,
			RiskClass:         "scoped_execute",
			ExpectedArtifacts: []string{"legacyAdapterInvocation"},
			Builtin:           true,
			Enabled:           true,
		},
	}
	for i := range builtins {
		builtins[i].CreatedAtMs = now
		builtins[i].UpdatedAtMs = now
	}
	return builtins
}

// EnsureDefaultsIfEmpty seeds built-in lanes only for a fresh store with no
// lane records. Existing lane state, including operator-disabled lanes, is
// left untouched.
func (s *Service) EnsureDefaultsIfEmpty(ctx context.Context, workspaceDir string) error {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM action_lanes`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return s.EnsureDefaults(ctx, workspaceDir)
}

func (s *Service) upsert(ctx context.Context, l Lane) error {
	allow, _ := json.Marshal(nonNilStrings(l.AllowedPaths))
	forbid, _ := json.Marshal(nonNilStrings(l.ForbiddenPaths))
	expected, _ := json.Marshal(nonNilStrings(l.ExpectedArtifacts))
	_, err := s.db.ExecContext(ctx, `
INSERT INTO action_lanes(
  id, created_at, updated_at, name, description, action_type,
  allowed_paths_json, forbidden_paths_json, write_intent, requires_approval,
  risk_class, max_bytes, expected_artifacts_json, builtin, enabled
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  updated_at=excluded.updated_at,
  name=excluded.name,
  description=excluded.description,
  action_type=excluded.action_type,
  allowed_paths_json=excluded.allowed_paths_json,
  forbidden_paths_json=excluded.forbidden_paths_json,
  write_intent=excluded.write_intent,
  requires_approval=excluded.requires_approval,
  risk_class=excluded.risk_class,
  max_bytes=excluded.max_bytes,
  expected_artifacts_json=excluded.expected_artifacts_json,
  enabled=excluded.enabled
`,
		l.ID, l.CreatedAtMs, l.UpdatedAtMs, l.Name, l.Description, l.ActionType,
		string(allow), string(forbid), boolToInt(l.WriteIntent), boolToInt(l.RequiresApproval),
		l.RiskClass, l.MaxBytes, string(expected), boolToInt(l.Builtin), boolToInt(l.Enabled),
	)
	return err
}

func (s *Service) Save(ctx context.Context, l Lane) (*Lane, error) {
	if strings.TrimSpace(l.ID) == "" {
		return nil, errors.New("lane id required")
	}
	if strings.TrimSpace(l.Name) == "" {
		return nil, errors.New("lane name required")
	}
	existing, _ := s.Get(ctx, l.ID)
	if existing != nil {
		if existing.Builtin {
			l.Builtin = true
			l.ActionType = existing.ActionType
			l.AllowedPaths = existing.AllowedPaths
			l.ForbiddenPaths = existing.ForbiddenPaths
		}
		l.CreatedAtMs = existing.CreatedAtMs
	}
	if l.CreatedAtMs == 0 {
		l.CreatedAtMs = time.Now().UnixMilli()
	}
	l.UpdatedAtMs = time.Now().UnixMilli()
	if err := s.upsert(ctx, l); err != nil {
		return nil, err
	}
	return s.Get(ctx, l.ID)
}

func (s *Service) Get(ctx context.Context, id string) (*Lane, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, created_at, updated_at, name, description, action_type,
       allowed_paths_json, forbidden_paths_json, write_intent, requires_approval,
       risk_class, max_bytes, expected_artifacts_json, builtin, enabled
FROM action_lanes WHERE id = ?`, id)
	return scanLane(row)
}

func (s *Service) List(ctx context.Context) ([]Lane, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, created_at, updated_at, name, description, action_type,
       allowed_paths_json, forbidden_paths_json, write_intent, requires_approval,
       risk_class, max_bytes, expected_artifacts_json, builtin, enabled
FROM action_lanes ORDER BY builtin DESC, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Lane{}
	for rows.Next() {
		l, err := scanLane(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *l)
	}
	return out, rows.Err()
}

func (s *Service) Delete(ctx context.Context, id string) error {
	l, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if l.Builtin {
		return fmt.Errorf("lane %q is built-in and cannot be deleted", id)
	}
	_, err = s.db.ExecContext(ctx, `DELETE FROM action_lanes WHERE id = ?`, id)
	return err
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanLane(row rowScanner) (*Lane, error) {
	var l Lane
	var allow, forbid, expected string
	var writeInt, reqApproval, builtin, enabled int
	if err := row.Scan(
		&l.ID, &l.CreatedAtMs, &l.UpdatedAtMs, &l.Name, &l.Description, &l.ActionType,
		&allow, &forbid, &writeInt, &reqApproval, &l.RiskClass, &l.MaxBytes, &expected, &builtin, &enabled,
	); err != nil {
		return nil, err
	}
	_ = json.Unmarshal([]byte(allow), &l.AllowedPaths)
	_ = json.Unmarshal([]byte(forbid), &l.ForbiddenPaths)
	_ = json.Unmarshal([]byte(expected), &l.ExpectedArtifacts)
	l.WriteIntent = writeInt == 1
	l.RequiresApproval = reqApproval == 1
	l.Builtin = builtin == 1
	l.Enabled = enabled == 1
	return &l, nil
}

func nonNilStrings(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
