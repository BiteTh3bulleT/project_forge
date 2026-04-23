package permissions

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

// Profile is the unified security boundary for gateway-level tool execution.
// It separates read / write / execute permissions, carries explicit allow /
// forbid path lists, and declares which risk classes require approval.
type Profile struct {
	ID                   string   `json:"id"`
	CreatedAtMs          int64    `json:"createdAtMs"`
	UpdatedAtMs          int64    `json:"updatedAtMs"`
	Name                 string   `json:"name"`
	Description          string   `json:"description"`
	AllowedReadPaths     []string `json:"allowedReadPaths"`
	AllowedWritePaths    []string `json:"allowedWritePaths"`
	AllowedExecutePaths  []string `json:"allowedExecutePaths"`
	ForbiddenPaths       []string `json:"forbiddenPaths"`
	AllowedTools         []string `json:"allowedTools"`
	ApprovalRequiredRisk []string `json:"approvalRequiredRisks"`
	MaxBytesPerWrite     int64    `json:"maxBytesPerWrite"`
	AllowNetwork         bool     `json:"allowNetwork"`
	Editable             bool     `json:"editable"`
	Active               bool     `json:"active"`
}

// CheckRequest describes a single action a tool or lane wants to perform so
// the permission service can return a clear allow / deny decision with a
// reason.
type CheckRequest struct {
	ToolID      string
	LaneID      string
	Action      string
	Paths       []string
	Reads       bool
	Writes      bool
	Executes    bool
	WriteBytes  int64
	UsesNetwork bool
	RiskClass   string
	// JobID, when set, allows bypassing soft policy gates after the operator
	// has approved that job (see jobHasOperatorApproval). Hard denies
	// (forbidden paths) still apply.
	JobID *string
}

// Decision is the result of a permission check.
type Decision struct {
	Allowed          bool     `json:"allowed"`
	RequiresApproval bool     `json:"requiresApproval"`
	Reason           string   `json:"reason"`
	ProfileID        string   `json:"profileId"`
	MatchedDenies    []string `json:"matchedDenies"`
	MatchedAllows    []string `json:"matchedAllows"`
}

type Service struct {
	db *sql.DB
}

func New(db *sql.DB) *Service { return &Service{db: db} }

// EnsureDefaults seeds the baseline "standard" profile if no profile exists.
// The default profile is conservative: read-only of workspace, no writes,
// no execute, no network, and requires approval for anything non-low risk.
func (s *Service) EnsureDefaults(ctx context.Context, workspaceDir string) error {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM permission_profiles`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	now := time.Now().UnixMilli()
	defaultProfile := Profile{
		ID:                   "standard",
		Name:                 "Standard (read-only workspace)",
		Description:          "Read workspace files, no writes, no execute, no network. Approval required for medium and high risk actions.",
		AllowedReadPaths:     []string{workspaceDir},
		AllowedWritePaths:    []string{filepath.Join(workspaceDir, "scratch")},
		AllowedExecutePaths:  []string{},
		ForbiddenPaths:       []string{"/etc", "/var", "/usr", "/boot", "/root", "~/.ssh", "~/.aws", "~/.gnupg"},
		AllowedTools:         []string{"fs.read", "fs.list", "fs.mkdir", "repo.inspect", "git.status", "git.diff", "git.branch", "validate.project_context", "net.interfaces", "net.dns_lookup", "net.connectivity", "time.now"},
		ApprovalRequiredRisk: []string{"medium", "high"},
		MaxBytesPerWrite:     512 * 1024,
		AllowNetwork:         false,
		Editable:             true,
		Active:               true,
	}
	hardenedProfile := Profile{
		ID:                   "strict",
		Name:                 "Strict (audit-only)",
		Description:          "Read-only inspection only. All non-low risk requires approval. No writes, no network, no execute.",
		AllowedReadPaths:     []string{workspaceDir},
		AllowedWritePaths:    []string{},
		AllowedExecutePaths:  []string{},
		ForbiddenPaths:       []string{"/etc", "/var", "/usr", "/boot", "/root", "~/.ssh", "~/.aws", "~/.gnupg", "~/.config"},
		AllowedTools:         []string{"fs.read", "fs.list", "repo.inspect", "git.status", "git.diff", "git.branch", "validate.project_context", "net.interfaces", "time.now"},
		ApprovalRequiredRisk: []string{"low", "medium", "high"},
		MaxBytesPerWrite:     0,
		AllowNetwork:         false,
		Editable:             true,
		Active:               false,
	}
	workspaceProfile := Profile{
		ID:                   "workspace-write",
		Name:                 "Workspace Write",
		Description:          "Allows bounded writes within the configured workspace. Medium and high risk actions still require approval.",
		AllowedReadPaths:     []string{workspaceDir},
		AllowedWritePaths:    []string{filepath.Join(workspaceDir, "artifacts"), filepath.Join(workspaceDir, "exports"), filepath.Join(workspaceDir, "scratch")},
		AllowedExecutePaths:  []string{},
		ForbiddenPaths:       []string{"/etc", "/var", "/usr", "/boot", "/root", "~/.ssh", "~/.aws", "~/.gnupg"},
		AllowedTools:         []string{"fs.read", "fs.list", "fs.mkdir", "fs.rename", "fs.copy", "fs.delete", "fs.write", "fs.chmod", "git.status", "git.diff", "git.branch", "git.commit", "git.checkout", "git.stash", "git.apply_patch", "repo.inspect", "validate.project_context", "proc.run", "proc.terminate", "system.service_status", "system.service_control", "system.logs", "desktop.notify", "desktop.open", "net.interfaces", "net.dns_lookup", "net.connectivity", "net.fetch", "secret.get"},
		ApprovalRequiredRisk: []string{"medium", "high"},
		MaxBytesPerWrite:     2 * 1024 * 1024,
		AllowNetwork:         false,
		Editable:             true,
		Active:               false,
	}
	for _, p := range []Profile{defaultProfile, hardenedProfile, workspaceProfile} {
		p.CreatedAtMs = now
		p.UpdatedAtMs = now
		if _, err := s.insert(ctx, p); err != nil {
			return err
		}
	}
	return nil
}

// EnsureMkdirChatPolicy merges fs.mkdir + workspace scratch write scope into the
// built-in "standard" profile so chat tool execution works on existing installs.
func (s *Service) EnsureMkdirChatPolicy(ctx context.Context, workspaceDir string) error {
	p, err := s.Get(ctx, "standard")
	if err != nil || p == nil {
		return nil
	}
	scratch := filepath.Join(workspaceDir, "scratch")
	p.AllowedTools = mergeUniqueStrings(p.AllowedTools, []string{"fs.mkdir"})
	p.AllowedWritePaths = mergeUniqueStrings(p.AllowedWritePaths, []string{scratch})
	_, err = s.insert(ctx, *p)
	return err
}

// EnsureGatewayToolPolicy upgrades legacy profiles so governed chat/tools can execute
// real operations without getting trapped in allowlist/execute-scope dead ends.
// Safety boundaries (forbidden paths, risk-gate approvals, network policy) remain intact.
func (s *Service) EnsureGatewayToolPolicy(ctx context.Context, workspaceDir string) error {
	gatewayTools := []string{
		"fs.read", "fs.list", "fs.mkdir", "fs.write", "fs.rename", "fs.copy", "fs.delete", "fs.chmod",
		"repo.inspect", "validate.project_context",
		"git.status", "git.diff", "git.branch", "git.commit", "git.checkout", "git.stash", "git.apply_patch",
		"proc.run", "proc.terminate",
		"system.service_status", "system.service_control", "system.logs",
		"desktop.notify", "desktop.open",
		"net.interfaces", "net.dns_lookup", "net.connectivity", "net.fetch",
		"time.now",
		"secret.get",
		"export.dossier", "export.packet", "export.audit",
	}
	ids := []string{"standard", "workspace-write"}
	for _, id := range ids {
		p, err := s.Get(ctx, id)
		if err != nil || p == nil {
			continue
		}
		p.AllowedTools = mergeUniqueStrings(p.AllowedTools, gatewayTools)
		p.AllowedReadPaths = mergeUniqueStrings(p.AllowedReadPaths, []string{workspaceDir})
		p.AllowedExecutePaths = mergeUniqueStrings(p.AllowedExecutePaths, []string{workspaceDir})
		if id == "workspace-write" {
			p.AllowedWritePaths = mergeUniqueStrings(p.AllowedWritePaths, []string{
				workspaceDir,
				filepath.Join(workspaceDir, "artifacts"),
				filepath.Join(workspaceDir, "exports"),
				filepath.Join(workspaceDir, "scratch"),
			})
		}
		if _, err := s.insert(ctx, *p); err != nil {
			return err
		}
	}
	return nil
}

func mergeUniqueStrings(base []string, add []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(base)+len(add))
	for _, x := range append(append([]string{}, base...), add...) {
		x = strings.TrimSpace(x)
		if x == "" {
			continue
		}
		if _, ok := seen[x]; ok {
			continue
		}
		seen[x] = struct{}{}
		out = append(out, x)
	}
	return out
}

func (s *Service) insert(ctx context.Context, p Profile) (*Profile, error) {
	reads, _ := json.Marshal(nonNilStrings(p.AllowedReadPaths))
	writes, _ := json.Marshal(nonNilStrings(p.AllowedWritePaths))
	execs, _ := json.Marshal(nonNilStrings(p.AllowedExecutePaths))
	forbid, _ := json.Marshal(nonNilStrings(p.ForbiddenPaths))
	tools, _ := json.Marshal(nonNilStrings(p.AllowedTools))
	risks, _ := json.Marshal(nonNilStrings(p.ApprovalRequiredRisk))
	_, err := s.db.ExecContext(ctx, `
INSERT INTO permission_profiles(
  id, created_at, updated_at, name, description,
  allowed_read_paths_json, allowed_write_paths_json, allowed_execute_paths_json,
  forbidden_paths_json, allowed_tools_json, approval_required_risks_json,
  max_bytes_per_write, allow_network, editable, active
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  updated_at=excluded.updated_at,
  name=excluded.name,
  description=excluded.description,
  allowed_read_paths_json=excluded.allowed_read_paths_json,
  allowed_write_paths_json=excluded.allowed_write_paths_json,
  allowed_execute_paths_json=excluded.allowed_execute_paths_json,
  forbidden_paths_json=excluded.forbidden_paths_json,
  allowed_tools_json=excluded.allowed_tools_json,
  approval_required_risks_json=excluded.approval_required_risks_json,
  max_bytes_per_write=excluded.max_bytes_per_write,
  allow_network=excluded.allow_network,
  editable=excluded.editable,
  active=excluded.active
`,
		p.ID, p.CreatedAtMs, p.UpdatedAtMs, p.Name, p.Description,
		string(reads), string(writes), string(execs),
		string(forbid), string(tools), string(risks),
		p.MaxBytesPerWrite, boolToInt(p.AllowNetwork), boolToInt(p.Editable), boolToInt(p.Active),
	)
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, p.ID)
}

// Save upserts a profile. If a caller marks a profile active, any other
// active profile is cleared so exactly one is active at a time.
func (s *Service) Save(ctx context.Context, p Profile) (*Profile, error) {
	if strings.TrimSpace(p.ID) == "" {
		return nil, errors.New("profile id required")
	}
	if strings.TrimSpace(p.Name) == "" {
		return nil, errors.New("profile name required")
	}
	if p.CreatedAtMs == 0 {
		p.CreatedAtMs = time.Now().UnixMilli()
	}
	p.UpdatedAtMs = time.Now().UnixMilli()
	if p.Active {
		if _, err := s.db.ExecContext(ctx, `UPDATE permission_profiles SET active = 0 WHERE id <> ?`, p.ID); err != nil {
			return nil, err
		}
	}
	return s.insert(ctx, p)
}

func (s *Service) Get(ctx context.Context, id string) (*Profile, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, created_at, updated_at, name, description,
       allowed_read_paths_json, allowed_write_paths_json, allowed_execute_paths_json,
       forbidden_paths_json, allowed_tools_json, approval_required_risks_json,
       max_bytes_per_write, allow_network, editable, active
FROM permission_profiles WHERE id = ?`, id)
	return scanProfile(row)
}

func (s *Service) List(ctx context.Context) ([]Profile, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, created_at, updated_at, name, description,
       allowed_read_paths_json, allowed_write_paths_json, allowed_execute_paths_json,
       forbidden_paths_json, allowed_tools_json, approval_required_risks_json,
       max_bytes_per_write, allow_network, editable, active
FROM permission_profiles ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Profile{}
	for rows.Next() {
		p, err := scanProfile(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

func (s *Service) Active(ctx context.Context) (*Profile, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, created_at, updated_at, name, description,
       allowed_read_paths_json, allowed_write_paths_json, allowed_execute_paths_json,
       forbidden_paths_json, allowed_tools_json, approval_required_risks_json,
       max_bytes_per_write, allow_network, editable, active
FROM permission_profiles WHERE active = 1 ORDER BY updated_at DESC LIMIT 1`)
	p, err := scanProfile(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return s.Get(ctx, "standard")
		}
		return nil, err
	}
	return p, nil
}

func (s *Service) Activate(ctx context.Context, id string) (*Profile, error) {
	if _, err := s.db.ExecContext(ctx, `UPDATE permission_profiles SET active = 0`); err != nil {
		return nil, err
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE permission_profiles SET active = 1, updated_at = ? WHERE id = ?`, time.Now().UnixMilli(), id); err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	p, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if !p.Editable {
		return fmt.Errorf("profile %q is not editable", id)
	}
	if p.Active {
		return fmt.Errorf("profile %q is active and cannot be deleted", id)
	}
	_, err = s.db.ExecContext(ctx, `DELETE FROM permission_profiles WHERE id = ?`, id)
	return err
}

// Check resolves a tool request against the active profile. Forbidden paths are
// always hard denies. Tool allowlists, path scopes, network, size limits, and
// risk class map to RequiresApproval (human gate) instead of silent deny.
func (s *Service) Check(ctx context.Context, req CheckRequest) (*Decision, *Profile, error) {
	profile, err := s.Active(ctx)
	if err != nil {
		return nil, nil, err
	}
	if profile == nil {
		return &Decision{Allowed: false, Reason: "no active permission profile"}, nil, nil
	}

	if jid := strings.TrimSpace(derefJobID(req.JobID)); jid != "" {
		granted, err := s.jobHasOperatorApproval(ctx, jid)
		if err != nil {
			return nil, nil, err
		}
		if granted {
			return s.checkApprovedJob(ctx, profile, req)
		}
	}

	decision := Decision{Allowed: true, ProfileID: profile.ID}
	var pending []string

	if len(profile.AllowedTools) > 0 && !contains(profile.AllowedTools, req.ToolID) {
		pending = append(pending, fmt.Sprintf("tool %q not on profile allowlist (operator approval required)", req.ToolID))
	}
	if req.UsesNetwork && !profile.AllowNetwork {
		pending = append(pending, "network access not enabled in profile (operator approval required)")
	}
	if req.Writes && profile.MaxBytesPerWrite > 0 && req.WriteBytes > profile.MaxBytesPerWrite {
		pending = append(pending, fmt.Sprintf("write of %d bytes exceeds profile limit %d (operator approval required)", req.WriteBytes, profile.MaxBytesPerWrite))
	}

	for _, p := range req.Paths {
		abs := normalizePath(p)
		if abs == "" {
			continue
		}
		for _, forbid := range profile.ForbiddenPaths {
			if pathScopeMatch(abs, forbid) {
				return &Decision{
					Allowed:       false,
					Reason:        fmt.Sprintf("path %q is inside forbidden scope %q", abs, forbid),
					ProfileID:     profile.ID,
					MatchedDenies: []string{forbid},
				}, profile, nil
			}
		}
		if req.Reads {
			if len(profile.AllowedReadPaths) == 0 {
				pending = append(pending, fmt.Sprintf("read path %q — profile has no read scope (operator approval required)", abs))
			} else if !anyMatch(profile.AllowedReadPaths, abs) {
				pending = append(pending, fmt.Sprintf("read path %q outside profile read scope (operator approval required)", abs))
			}
		}
		if req.Writes {
			if len(profile.AllowedWritePaths) == 0 {
				pending = append(pending, fmt.Sprintf("write path %q — profile has no write scope (operator approval required)", abs))
			} else if !anyMatch(profile.AllowedWritePaths, abs) {
				pending = append(pending, fmt.Sprintf("write path %q outside profile write scope (operator approval required)", abs))
			}
		}
		if req.Executes {
			if len(profile.AllowedExecutePaths) == 0 {
				pending = append(pending, fmt.Sprintf("execute path %q — profile has no execute scope (operator approval required)", abs))
			} else if !anyMatch(profile.AllowedExecutePaths, abs) {
				pending = append(pending, fmt.Sprintf("execute path %q outside profile execute scope (operator approval required)", abs))
			}
		}
		decision.MatchedAllows = append(decision.MatchedAllows, abs)
	}

	if len(req.Paths) == 0 {
		if req.Writes && len(profile.AllowedWritePaths) == 0 {
			pending = append(pending, "profile has no write paths configured (operator approval required)")
		}
		if req.Executes && len(profile.AllowedExecutePaths) == 0 {
			pending = append(pending, "profile has no execute paths configured (operator approval required)")
		}
	}

	risk := strings.ToLower(strings.TrimSpace(req.RiskClass))
	if risk == "" {
		risk = "low"
	}
	if contains(profile.ApprovalRequiredRisk, risk) {
		pending = append(pending, fmt.Sprintf("risk class %q requires approval per active profile", risk))
	}

	if len(pending) > 0 {
		decision.RequiresApproval = true
		decision.Reason = strings.Join(pending, "; ")
	} else {
		decision.Reason = "within active permission profile"
	}
	return &decision, profile, nil
}

func derefJobID(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func (s *Service) jobHasOperatorApproval(ctx context.Context, jobID string) (bool, error) {
	var st string
	err := s.db.QueryRowContext(ctx, `SELECT approval_status FROM jobs WHERE id = ?`, jobID).Scan(&st)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return strings.EqualFold(strings.TrimSpace(st), "granted"), nil
}

// checkApprovedJob enforces only forbidden-path rules after the job was approved.
func (s *Service) checkApprovedJob(ctx context.Context, profile *Profile, req CheckRequest) (*Decision, *Profile, error) {
	decision := Decision{Allowed: true, ProfileID: profile.ID, Reason: "operator approved this job; soft policy gates lifted for execution"}
	for _, p := range req.Paths {
		abs := normalizePath(p)
		if abs == "" {
			continue
		}
		for _, forbid := range profile.ForbiddenPaths {
			if pathScopeMatch(abs, forbid) {
				return &Decision{
					Allowed:       false,
					Reason:        fmt.Sprintf("path %q is inside forbidden scope %q", abs, forbid),
					ProfileID:     profile.ID,
					MatchedDenies: []string{forbid},
				}, profile, nil
			}
		}
	}
	return &decision, profile, nil
}

// Summary is a compact view used by UI surfaces to explain the active profile
// without exposing internal wiring.
type Summary struct {
	Active          *Profile `json:"active"`
	ProfilesCount   int      `json:"profilesCount"`
	WriteEnabled    bool     `json:"writeEnabled"`
	ExecuteEnabled  bool     `json:"executeEnabled"`
	NetworkEnabled  bool     `json:"networkEnabled"`
	ForbiddenCount  int      `json:"forbiddenCount"`
	AllowedToolsN   int      `json:"allowedToolsCount"`
	RiskGatesLabels []string `json:"riskGatesLabels"`
}

func (s *Service) Summary(ctx context.Context) (*Summary, error) {
	active, err := s.Active(ctx)
	if err != nil {
		return nil, err
	}
	list, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	sum := &Summary{
		Active:        active,
		ProfilesCount: len(list),
	}
	if active != nil {
		sum.WriteEnabled = len(active.AllowedWritePaths) > 0
		sum.ExecuteEnabled = len(active.AllowedExecutePaths) > 0
		sum.NetworkEnabled = active.AllowNetwork
		sum.ForbiddenCount = len(active.ForbiddenPaths)
		sum.AllowedToolsN = len(active.AllowedTools)
		sum.RiskGatesLabels = active.ApprovalRequiredRisk
	}
	return sum, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanProfile(row rowScanner) (*Profile, error) {
	var p Profile
	var reads, writes, execs, forbid, tools, risks string
	var allowNet, editable, active int
	if err := row.Scan(
		&p.ID, &p.CreatedAtMs, &p.UpdatedAtMs, &p.Name, &p.Description,
		&reads, &writes, &execs, &forbid, &tools, &risks,
		&p.MaxBytesPerWrite, &allowNet, &editable, &active,
	); err != nil {
		return nil, err
	}
	_ = json.Unmarshal([]byte(reads), &p.AllowedReadPaths)
	_ = json.Unmarshal([]byte(writes), &p.AllowedWritePaths)
	_ = json.Unmarshal([]byte(execs), &p.AllowedExecutePaths)
	_ = json.Unmarshal([]byte(forbid), &p.ForbiddenPaths)
	_ = json.Unmarshal([]byte(tools), &p.AllowedTools)
	_ = json.Unmarshal([]byte(risks), &p.ApprovalRequiredRisk)
	p.AllowNetwork = allowNet == 1
	p.Editable = editable == 1
	p.Active = active == 1
	return &p, nil
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

func contains(list []string, v string) bool {
	for _, e := range list {
		if strings.EqualFold(strings.TrimSpace(e), strings.TrimSpace(v)) {
			return true
		}
	}
	return false
}

func anyMatch(scopes []string, target string) bool {
	for _, s := range scopes {
		if pathScopeMatch(target, s) {
			return true
		}
	}
	return false
}

func normalizePath(p string) string {
	trimmed := strings.TrimSpace(p)
	if trimmed == "" {
		return trimmed
	}
	if abs, err := filepath.Abs(trimmed); err == nil {
		return filepath.Clean(abs)
	}
	return filepath.Clean(trimmed)
}

// pathScopeMatch returns true if target is equal to scope or is a descendant
// of scope. Scope entries are treated as absolute paths; a target is
// considered "inside" a scope when a clean relative path does not start with
// "..".
func pathScopeMatch(target, scope string) bool {
	t := strings.TrimSpace(target)
	s := strings.TrimSpace(scope)
	if t == "" || s == "" {
		return false
	}
	absScope := s
	if a, err := filepath.Abs(s); err == nil {
		absScope = filepath.Clean(a)
	}
	absTarget := t
	if a, err := filepath.Abs(t); err == nil {
		absTarget = filepath.Clean(a)
	}
	if absTarget == absScope {
		return true
	}
	rel, err := filepath.Rel(absScope, absTarget)
	if err != nil {
		return false
	}
	rel = filepath.Clean(rel)
	if rel == "." {
		return true
	}
	if strings.HasPrefix(rel, "..") {
		return false
	}
	return !strings.HasPrefix(rel, string(filepath.Separator))
}
