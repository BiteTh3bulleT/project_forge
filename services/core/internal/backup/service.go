package backup

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Bundle is a portable snapshot of FORGE operator state. Bundles are
// designed to be importable by a future IRIS instance or by another FORGE
// deployment; they are not raw database dumps. Each bundle carries a schema
// version, a kind, and a counted inventory of entities.
type Bundle struct {
	ID           int64           `json:"id"`
	CreatedAtMs  int64           `json:"createdAtMs"`
	Kind         string          `json:"kind"`
	Label        string          `json:"label"`
	VersionTag   string          `json:"versionTag"`
	FilePath     string          `json:"filePath"`
	SizeBytes    int64           `json:"sizeBytes"`
	SHA256       string          `json:"sha256"`
	EntityCounts json.RawMessage `json:"entityCounts"`
	Notes        string          `json:"notes"`
	SourceVer    string          `json:"sourceVersion"`
}

const BundleSchemaVersion = 1

// BundleDoc is the on-disk JSON layout for a bundle.
type BundleDoc struct {
	Schema       int                `json:"schema"`
	GeneratedAt  int64              `json:"generatedAtMs"`
	Kind         string             `json:"kind"`
	Label        string             `json:"label"`
	VersionTag   string             `json:"versionTag"`
	SourceVer    string             `json:"sourceVersion"`
	Notes        string             `json:"notes"`
	EntityCounts map[string]int     `json:"entityCounts"`
	Entities     map[string][]any   `json:"entities"`
	ImportedFrom map[string]string  `json:"importedFrom,omitempty"`
	Checksums    map[string]string  `json:"checksums,omitempty"`
}

// KnownKinds lists the bundle kinds supported by the exporter/importer. The
// gateway uses these to validate "which slice of FORGE do you want to
// snapshot?"
var KnownKinds = []string{
	"dossiers",
	"packets",
	"project_context",
	"evaluations",
	"strategies",
	"automation_rules",
	"policy_profiles",
	"audit_history",
	"portable_snapshot",
	"full_backup",
}

type Service struct {
	db       *sql.DB
	dataDir  string
	backups  string
	exports  string
}

func New(db *sql.DB, dataDir string) *Service {
	s := &Service{
		db:      db,
		dataDir: dataDir,
		backups: filepath.Join(dataDir, "backups"),
		exports: filepath.Join(dataDir, "exports"),
	}
	_ = os.MkdirAll(s.backups, 0o755)
	_ = os.MkdirAll(s.exports, 0o755)
	return s
}

func (s *Service) Dirs() (string, string) { return s.backups, s.exports }

// CreateBundle builds a BundleDoc of the requested kind, writes it to disk,
// records the inventory, and returns the record.
type CreateBundleRequest struct {
	Kind       string `json:"kind"`
	Label      string `json:"label"`
	VersionTag string `json:"versionTag"`
	Notes      string `json:"notes"`
	SourceVer  string `json:"sourceVersion"`
}

func (s *Service) CreateBundle(ctx context.Context, req CreateBundleRequest) (*Bundle, error) {
	kind := strings.ToLower(strings.TrimSpace(req.Kind))
	if kind == "" {
		return nil, errors.New("bundle kind required")
	}
	if !isKnownKind(kind) {
		return nil, fmt.Errorf("unknown bundle kind %q", kind)
	}
	doc := BundleDoc{
		Schema:       BundleSchemaVersion,
		GeneratedAt:  time.Now().UnixMilli(),
		Kind:         kind,
		Label:        req.Label,
		VersionTag:   strings.TrimSpace(req.VersionTag),
		SourceVer:    strings.TrimSpace(req.SourceVer),
		Notes:        req.Notes,
		EntityCounts: map[string]int{},
		Entities:     map[string][]any{},
	}

	sections, err := s.pickSections(kind)
	if err != nil {
		return nil, err
	}
	for _, sec := range sections {
		rows, err := s.extractSection(ctx, sec)
		if err != nil {
			return nil, fmt.Errorf("extract %s: %w", sec, err)
		}
		doc.Entities[sec] = rows
		doc.EntityCounts[sec] = len(rows)
	}

	targetDir := s.backups
	if kind != "full_backup" && kind != "portable_snapshot" {
		targetDir = s.exports
	}
	label := req.Label
	if label == "" {
		label = kind
	}
	fileName := fmt.Sprintf("%s-%s-%d.json", label, kind, doc.GeneratedAt)
	fileName = sanitizeName(fileName)
	outPath := filepath.Join(targetDir, fileName)

	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(outPath, raw, 0o644); err != nil {
		return nil, err
	}

	sum := sha256.Sum256(raw)
	sumHex := hex.EncodeToString(sum[:])

	countsBytes, _ := json.Marshal(doc.EntityCounts)
	now := time.Now().UnixMilli()
	res, err := s.db.ExecContext(ctx, `
INSERT INTO backup_bundles(
  created_at, kind, label, version_tag, file_path, size_bytes, sha256,
  entity_counts_json, notes, source_version
) VALUES(?,?,?,?,?,?,?,?,?,?)`,
		now, kind, label, doc.VersionTag, outPath, int64(len(raw)), sumHex,
		string(countsBytes), doc.Notes, doc.SourceVer,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return s.Get(ctx, id)
}

// RestoreBundleRequest imports a bundle JSON file into the current FORGE
// instance. Restores are conservative: existing records are only touched for
// the sections explicitly listed in sections (or all by default).
type RestoreBundleRequest struct {
	FilePath string   `json:"filePath"`
	Sections []string `json:"sections"`
	DryRun   bool     `json:"dryRun"`
}

type RestoreResult struct {
	Accepted   bool              `json:"accepted"`
	DryRun     bool              `json:"dryRun"`
	BundleKind string            `json:"bundleKind"`
	Imported   map[string]int    `json:"imported"`
	Skipped    map[string]int    `json:"skipped"`
	Errors     []string          `json:"errors"`
	Schema     int               `json:"schema"`
	Meta       map[string]string `json:"meta"`
}

func (s *Service) RestoreBundle(ctx context.Context, req RestoreBundleRequest) (*RestoreResult, error) {
	if strings.TrimSpace(req.FilePath) == "" {
		return nil, errors.New("file path required")
	}
	raw, err := os.ReadFile(req.FilePath)
	if err != nil {
		return nil, err
	}
	var doc BundleDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("invalid bundle JSON: %w", err)
	}
	if doc.Schema != BundleSchemaVersion {
		return nil, fmt.Errorf("unsupported bundle schema %d (want %d)", doc.Schema, BundleSchemaVersion)
	}
	result := &RestoreResult{
		Accepted:   true,
		DryRun:     req.DryRun,
		BundleKind: doc.Kind,
		Imported:   map[string]int{},
		Skipped:    map[string]int{},
		Schema:     doc.Schema,
		Meta: map[string]string{
			"label":         doc.Label,
			"versionTag":    doc.VersionTag,
			"sourceVersion": doc.SourceVer,
		},
	}
	sections := req.Sections
	if len(sections) == 0 {
		sections = knownSections(doc)
	}
	for _, sec := range sections {
		rows, ok := doc.Entities[sec]
		if !ok {
			result.Skipped[sec] = 0
			continue
		}
		if req.DryRun {
			result.Imported[sec] = len(rows)
			continue
		}
		n, err := s.restoreSection(ctx, sec, rows)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", sec, err))
			continue
		}
		result.Imported[sec] = n
	}
	return result, nil
}

func (s *Service) List(ctx context.Context, limit int) ([]Bundle, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, created_at, kind, label, version_tag, file_path, size_bytes, sha256,
       entity_counts_json, notes, source_version
FROM backup_bundles ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Bundle{}
	for rows.Next() {
		var b Bundle
		var counts string
		if err := rows.Scan(&b.ID, &b.CreatedAtMs, &b.Kind, &b.Label, &b.VersionTag, &b.FilePath, &b.SizeBytes, &b.SHA256, &counts, &b.Notes, &b.SourceVer); err != nil {
			return nil, err
		}
		b.EntityCounts = json.RawMessage(counts)
		out = append(out, b)
	}
	return out, rows.Err()
}

func (s *Service) Get(ctx context.Context, id int64) (*Bundle, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, created_at, kind, label, version_tag, file_path, size_bytes, sha256,
       entity_counts_json, notes, source_version
FROM backup_bundles WHERE id = ?`, id)
	var b Bundle
	var counts string
	if err := row.Scan(&b.ID, &b.CreatedAtMs, &b.Kind, &b.Label, &b.VersionTag, &b.FilePath, &b.SizeBytes, &b.SHA256, &counts, &b.Notes, &b.SourceVer); err != nil {
		return nil, err
	}
	b.EntityCounts = json.RawMessage(counts)
	return &b, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	b, err := s.Get(ctx, id)
	if err != nil {
		return err
	}
	if b.FilePath != "" {
		_ = os.Remove(b.FilePath)
	}
	_, err = s.db.ExecContext(ctx, `DELETE FROM backup_bundles WHERE id = ?`, id)
	return err
}

// --- section extraction ---

func (s *Service) pickSections(kind string) ([]string, error) {
	switch kind {
	case "dossiers":
		return []string{"dossiers"}, nil
	case "packets":
		return []string{"task_packets"}, nil
	case "project_context":
		return []string{"project_context_records"}, nil
	case "evaluations":
		return []string{"evaluation_records"}, nil
	case "strategies":
		return []string{"execution_strategies"}, nil
	case "automation_rules":
		return []string{"automation_rules"}, nil
	case "policy_profiles":
		return []string{"permission_profiles", "approval_presets", "dossier_profiles"}, nil
	case "audit_history":
		return []string{"audit_records", "gateway_invocations"}, nil
	case "portable_snapshot":
		return []string{
			"dossiers", "task_packets", "project_context_records",
			"execution_strategies", "approval_presets", "permission_profiles",
			"automation_rules",
		}, nil
	case "full_backup":
		return []string{
			"dossiers", "task_packets", "project_context_records",
			"execution_strategies", "approval_presets", "permission_profiles",
			"automation_rules", "evaluation_records", "audit_records",
			"gateway_invocations", "action_lanes",
			"provenance_records", "journal_events", "memory_notes", "semantic_links",
			"state_items", "state_versions", "open_loops", "artifact_refs",
			"derived_models", "contradiction_records", "supersession_records",
			"context_packet_snapshots",
		}, nil
	}
	return nil, fmt.Errorf("unknown bundle kind %q", kind)
}

func (s *Service) extractSection(ctx context.Context, table string) ([]any, error) {
	query, ok := extractQueries[table]
	if !ok {
		return nil, fmt.Errorf("no extract query for %q", table)
	}
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	out := []any{}
	for rows.Next() {
		scanSlots := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range scanSlots {
			ptrs[i] = &scanSlots[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		record := map[string]any{}
		for i, col := range cols {
			record[col] = normalizeValue(scanSlots[i])
		}
		out = append(out, record)
	}
	return out, rows.Err()
}

func (s *Service) restoreSection(ctx context.Context, table string, rows []any) (int, error) {
	insert, ok := insertStatements[table]
	if !ok {
		return 0, fmt.Errorf("no import mapping for %q", table)
	}
	n := 0
	for _, r := range rows {
		rec, ok := r.(map[string]any)
		if !ok {
			continue
		}
		args := make([]any, 0, len(insert.fields))
		for _, f := range insert.fields {
			args = append(args, rec[f])
		}
		if _, err := s.db.ExecContext(ctx, insert.sql, args...); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

type insertMap struct {
	sql    string
	fields []string
}

var extractQueries = map[string]string{
	"dossiers":                "SELECT * FROM dossiers",
	"task_packets":            "SELECT * FROM task_packets",
	"project_context_records": "SELECT * FROM project_context_records",
	"execution_strategies":    "SELECT * FROM execution_strategies",
	"approval_presets":        "SELECT * FROM approval_presets",
	"permission_profiles":     "SELECT * FROM permission_profiles",
	"dossier_profiles":        "SELECT * FROM dossier_profiles",
	"automation_rules":        "SELECT * FROM automation_rules",
	"evaluation_records":      "SELECT * FROM evaluation_records",
	"audit_records":           "SELECT * FROM audit_records ORDER BY id DESC LIMIT 5000",
	"gateway_invocations":     "SELECT * FROM gateway_invocations ORDER BY id DESC LIMIT 5000",
	"action_lanes":            "SELECT * FROM action_lanes",
	"provenance_records":      "SELECT * FROM provenance_records ORDER BY created_at DESC",
	"journal_events":          "SELECT * FROM journal_events ORDER BY created_at DESC",
	"memory_notes":            "SELECT * FROM memory_notes ORDER BY updated_at DESC",
	"semantic_links":          "SELECT * FROM semantic_links ORDER BY created_at DESC",
	"state_items":             "SELECT * FROM state_items ORDER BY updated_at DESC",
	"state_versions":          "SELECT * FROM state_versions ORDER BY id DESC",
	"open_loops":              "SELECT * FROM open_loops ORDER BY updated_at DESC",
	"artifact_refs":           "SELECT * FROM artifact_refs ORDER BY created_at DESC",
	"derived_models":          "SELECT * FROM derived_models ORDER BY updated_at DESC",
	"contradiction_records":   "SELECT * FROM contradiction_records ORDER BY created_at DESC",
	"supersession_records":    "SELECT * FROM supersession_records ORDER BY created_at DESC",
	"context_packet_snapshots": "SELECT * FROM context_packet_snapshots ORDER BY created_at DESC",
}

// Only policy-shaped tables are wired for re-import in this pass. Other
// sections can be restored by the operator via the UI after review.
var insertStatements = map[string]insertMap{
	"permission_profiles": {
		sql: `INSERT INTO permission_profiles(
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
  active=excluded.active`,
		fields: []string{
			"id", "created_at", "updated_at", "name", "description",
			"allowed_read_paths_json", "allowed_write_paths_json", "allowed_execute_paths_json",
			"forbidden_paths_json", "allowed_tools_json", "approval_required_risks_json",
			"max_bytes_per_write", "allow_network", "editable", "active",
		},
	},
	"approval_presets": {
		sql: `INSERT INTO approval_presets(id, created_at, updated_at, name, description, profile_json, editable)
VALUES(?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  updated_at=excluded.updated_at,
  name=excluded.name,
  description=excluded.description,
  profile_json=excluded.profile_json,
  editable=excluded.editable`,
		fields: []string{"id", "created_at", "updated_at", "name", "description", "profile_json", "editable"},
	},
	"execution_strategies": {
		sql: `INSERT INTO execution_strategies(
  id, created_at, updated_at, name, task_type, target_adapter, retrieval_mode,
  packet_rules_json, approval_required, approval_preset_id, expected_artifacts_json,
  success_criteria_json, retry_guidance_json, enabled
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
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
		fields: []string{
			"id", "created_at", "updated_at", "name", "task_type", "target_adapter", "retrieval_mode",
			"packet_rules_json", "approval_required", "approval_preset_id", "expected_artifacts_json",
			"success_criteria_json", "retry_guidance_json", "enabled",
		},
	},
}

func knownSections(doc BundleDoc) []string {
	out := make([]string, 0, len(doc.Entities))
	for k := range doc.Entities {
		out = append(out, k)
	}
	return out
}

func normalizeValue(v any) any {
	if v == nil {
		return nil
	}
	switch t := v.(type) {
	case []byte:
		return string(t)
	default:
		return t
	}
}

func sanitizeName(in string) string {
	out := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			return r
		case r == '.' || r == '-' || r == '_':
			return r
		}
		return '_'
	}, in)
	return out
}

func isKnownKind(k string) bool {
	for _, v := range KnownKinds {
		if k == v {
			return true
		}
	}
	return false
}
