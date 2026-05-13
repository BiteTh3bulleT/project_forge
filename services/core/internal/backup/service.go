package backup

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

const (
	BundleSchemaVersion = 1

	restoreAtomicScopeDBSupported = "db-supported-sections-only"
	restoreExportOnlyPolicyWarn   = "restore policy: VSA-derived sections are export-only; rerun VSA reindex/signals after restore"
	maxRestoreBundleBytes         = 64 << 20
)

// BundleDoc is the on-disk JSON layout for a bundle.
type BundleDoc struct {
	Schema       int               `json:"schema"`
	GeneratedAt  int64             `json:"generatedAtMs"`
	Kind         string            `json:"kind"`
	Label        string            `json:"label"`
	VersionTag   string            `json:"versionTag"`
	SourceVer    string            `json:"sourceVersion"`
	Notes        string            `json:"notes"`
	EntityCounts map[string]int    `json:"entityCounts"`
	Entities     map[string][]any  `json:"entities"`
	Manifest     []SectionManifest `json:"manifest,omitempty"`
	ImportedFrom map[string]string `json:"importedFrom,omitempty"`
	Checksums    map[string]string `json:"checksums,omitempty"`
}

type SectionManifest struct {
	Name                   string `json:"name"`
	Purpose                string `json:"purpose"`
	AuthorityClass         string `json:"authorityClass"`
	BackupRequired         bool   `json:"backupRequired"`
	RestoreRequired        bool   `json:"restoreRequired"`
	ExportOnly             bool   `json:"exportOnly"`
	IntegrityCheckRequired bool   `json:"integrityCheckRequired"`
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
	db      *sql.DB
	dataDir string
	backups string
	exports string
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

// ResolveRestorePath validates that a restore bundle path is a regular file
// staged under this service's governed backup/export directories.
func (s *Service) ResolveRestorePath(filePath string) (string, error) {
	input := strings.TrimSpace(filePath)
	if input == "" {
		return "", errors.New("file path required")
	}
	targetAbs, err := filepath.Abs(input)
	if err != nil {
		return "", fmt.Errorf("resolve restore path: %w", err)
	}
	info, err := os.Lstat(targetAbs)
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("restore bundle path must not be a symlink: %s", targetAbs)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("restore bundle path must be a regular file: %s", targetAbs)
	}
	resolvedTarget, err := filepath.EvalSymlinks(targetAbs)
	if err != nil {
		return "", fmt.Errorf("resolve restore path symlinks: %w", err)
	}
	resolvedTarget, err = filepath.Abs(resolvedTarget)
	if err != nil {
		return "", fmt.Errorf("resolve restore path: %w", err)
	}
	for _, root := range []string{s.backups, s.exports} {
		ok, err := restorePathWithinRoot(resolvedTarget, root)
		if err != nil {
			return "", err
		}
		if ok {
			return resolvedTarget, nil
		}
	}
	return "", fmt.Errorf("restore bundle path must be under the backup or export directory")
}

func restorePathWithinRoot(target, root string) (bool, error) {
	rootAbs, err := filepath.Abs(strings.TrimSpace(root))
	if err != nil {
		return false, fmt.Errorf("resolve restore root: %w", err)
	}
	info, err := os.Lstat(rootAbs)
	if err != nil {
		return false, fmt.Errorf("restore root unavailable: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return false, fmt.Errorf("restore root must not be a symlink: %s", rootAbs)
	}
	if !info.IsDir() {
		return false, fmt.Errorf("restore root must be a directory: %s", rootAbs)
	}
	resolvedRoot, err := filepath.EvalSymlinks(rootAbs)
	if err != nil {
		return false, fmt.Errorf("resolve restore root symlinks: %w", err)
	}
	resolvedRoot, err = filepath.Abs(resolvedRoot)
	if err != nil {
		return false, fmt.Errorf("resolve restore root: %w", err)
	}
	rel, err := filepath.Rel(resolvedRoot, target)
	if err != nil {
		return false, nil
	}
	return rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)), nil
}

func readRestoreBundleFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if info, err := f.Stat(); err == nil && info.Size() > maxRestoreBundleBytes {
		return nil, fmt.Errorf("restore bundle too large: %d bytes exceeds %d byte limit", info.Size(), maxRestoreBundleBytes)
	}
	raw, err := io.ReadAll(io.LimitReader(f, maxRestoreBundleBytes+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > maxRestoreBundleBytes {
		return nil, fmt.Errorf("restore bundle too large: exceeds %d byte limit", maxRestoreBundleBytes)
	}
	return raw, nil
}

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
		Manifest:     []SectionManifest{},
		Checksums:    map[string]string{},
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
		doc.Manifest = append(doc.Manifest, backupSectionManifest(sec))
		doc.Checksums[sec] = checksumRows(rows)
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
	FilePath   string   `json:"filePath"`
	Sections   []string `json:"sections"`
	DryRun     bool     `json:"dryRun"`
	ApprovalID string   `json:"approvalId,omitempty"`
}

type RestoreResult struct {
	Accepted         bool              `json:"accepted"`
	DryRun           bool              `json:"dryRun"`
	Atomic           bool              `json:"atomic"`
	AtomicScope      string            `json:"atomicScope"`
	GlobalAtomic     bool              `json:"globalAtomic"`
	Applied          bool              `json:"applied"`
	RolledBack       bool              `json:"rolledBack"`
	BundleKind       string            `json:"bundleKind"`
	Imported         map[string]int    `json:"imported"`
	Skipped          map[string]int    `json:"skipped"`
	Unsupported      map[string]string `json:"unsupported,omitempty"`
	ExportOnly       map[string]string `json:"exportOnly,omitempty"`
	NonDBSideEffects map[string]string `json:"nonDbSideEffects,omitempty"`
	Warnings         []string          `json:"warnings,omitempty"`
	Errors           []string          `json:"errors"`
	Schema           int               `json:"schema"`
	Meta             map[string]string `json:"meta"`
	Verification     map[string]any    `json:"verification,omitempty"`
}

type restoreSectionPlan struct {
	name string
	rows []any
}

func (s *Service) RestoreBundle(ctx context.Context, req RestoreBundleRequest) (*RestoreResult, error) {
	filePath, err := s.ResolveRestorePath(req.FilePath)
	if err != nil {
		return nil, err
	}
	raw, err := readRestoreBundleFile(filePath)
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
		Accepted:         true,
		DryRun:           req.DryRun,
		AtomicScope:      restoreAtomicScopeDBSupported,
		GlobalAtomic:     false,
		BundleKind:       doc.Kind,
		Imported:         map[string]int{},
		Skipped:          map[string]int{},
		Unsupported:      map[string]string{},
		ExportOnly:       map[string]string{},
		NonDBSideEffects: map[string]string{},
		Warnings:         []string{},
		Schema:           doc.Schema,
		Meta: map[string]string{
			"label":         doc.Label,
			"versionTag":    doc.VersionTag,
			"sourceVersion": doc.SourceVer,
		},
		Verification: map[string]any{
			"schema":       "not_run",
			"rowCounts":    map[string]int{},
			"checksums":    map[string]string{},
			"fatalErrors":  []string{},
			"nonFatalGaps": []string{},
		},
	}
	sections := normalizeSections(req.Sections)
	if len(sections) == 0 {
		sections = knownSections(doc)
	}
	sections = orderSectionsForRestore(sections)
	planned := make([]restoreSectionPlan, 0, len(sections))
	for _, sec := range sections {
		rows, ok := doc.Entities[sec]
		if !ok {
			result.Unsupported[sec] = "section not found in bundle"
			result.Skipped[sec] = 0
			continue
		}
		if reason, exportOnly := restoreExportOnlyReason(sec); exportOnly {
			result.ExportOnly[sec] = reason
			result.Unsupported[sec] = reason
			result.Skipped[sec] = len(rows)
			result.Warnings = appendUniqueString(result.Warnings, restoreExportOnlyPolicyWarn)
			continue
		}
		if _, supported := insertStatements[sec]; !supported {
			result.Unsupported[sec] = restoreUnsupportedReason(sec)
			result.Skipped[sec] = len(rows)
			continue
		}
		if err := s.validateRestoreSchema(ctx, sec); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", sec, err))
			result.Verification["schema"] = "failed"
			result.Verification["fatalErrors"] = appendStringSlice(result.Verification["fatalErrors"], fmt.Sprintf("%s: %v", sec, err))
			return result, nil
		}
		planned = append(planned, restoreSectionPlan{name: sec, rows: rows})
		if limitation := restoreSectionNonDBSideEffect(sec); limitation != "" {
			result.NonDBSideEffects[sec] = limitation
			result.Warnings = appendUniqueString(result.Warnings, limitation)
		}
	}
	if req.DryRun {
		for _, sec := range planned {
			result.Imported[sec.name] = len(sec.rows)
		}
		result.Verification["schema"] = "passed"
		return result, nil
	}
	if len(planned) == 0 {
		return result, nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("begin restore transaction: %v", err))
		return result, err
	}
	result.Atomic = true
	applied := map[string]int{}
	for _, sec := range planned {
		n, err := s.restoreSection(ctx, tx, sec.name, sec.rows)
		if err != nil {
			_ = tx.Rollback()
			result.RolledBack = true
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", sec.name, err))
			return result, nil
		}
		applied[sec.name] = n
	}
	if err := tx.Commit(); err != nil {
		_ = tx.Rollback()
		result.RolledBack = true
		result.Errors = append(result.Errors, fmt.Sprintf("commit restore transaction: %v", err))
		return result, err
	}
	result.Applied = true
	result.Imported = applied
	s.verifyRestoreResult(ctx, doc, planned, result)
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
			"events", "jobs", "job_status_history", "job_events",
			"approval_requests", "approval_decisions",
			"artifacts",
			"sources", "files", "chunks", "embedding_records",
			"retrieval_runs", "retrieval_results", "retrieval_result_selection", "packet_retrieval_runs",
			"dossier_sources", "dossier_jobs", "dossier_packets", "dossier_briefs", "context_evidence",
			"memory_observations", "memory_observation_links", "retrieval_result_observations",
			"memory_usefulness_events", "packet_alignment_notes", "memory_repair_runs", "memory_repair_items",
			"execution_strategies", "approval_presets", "permission_profiles", "dossier_profiles",
			"automation_rules", "evaluation_records", "audit_records",
			"gateway_invocations", "action_lanes",
			"model_manifests", "model_registry_status", "model_runtime_loads",
			"provenance_records", "journal_events", "memory_notes", "semantic_links",
			"state_items", "state_versions", "open_loops", "artifact_refs",
			"derived_models", "contradiction_records", "supersession_records",
			"context_packet_snapshots", "dream_reports", "restore_outcome_events", "semantic_idempotency_keys", "autonomy_settings",
			"chat_threads", "chat_messages", "canvas_boards", "canvas_notes",
			"tool_capability_overrides", "feature_flags", "alert_rules", "scheduled_tasks",
			"memory_vsa_pointers", "memory_vsa_role_bindings", "memory_vsa_associations",
			"retrieval_result_vsa_signals", "memory_vsa_reindex_runs", "memory_vsa_reindex_items",
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

type execContext interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func (s *Service) restoreSection(ctx context.Context, execer execContext, table string, rows []any) (int, error) {
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
		normalizeRestoreRecord(table, rec)
		args := make([]any, 0, len(insert.fields))
		for _, f := range insert.fields {
			args = append(args, rec[f])
		}
		res, err := execer.ExecContext(ctx, insert.sql, args...)
		if err != nil {
			return n, err
		}
		affected, err := res.RowsAffected()
		if err != nil || affected < 0 {
			n++
			continue
		}
		n += int(affected)
	}
	return n, nil
}

func (s *Service) validateRestoreSchema(ctx context.Context, table string) error {
	insert, ok := insertStatements[table]
	if !ok {
		return fmt.Errorf("no import mapping for %q", table)
	}
	schemaTable := table
	if table == "autonomy_settings" {
		schemaTable = "settings"
	}
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(`PRAGMA table_info(%s)`, schemaTable))
	if err != nil {
		return err
	}
	defer rows.Close()
	columns := map[string]struct{}{}
	for rows.Next() {
		var cid, notNull, pk int
		var name, colType string
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &colType, &notNull, &defaultValue, &pk); err != nil {
			return err
		}
		columns[name] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(columns) == 0 {
		return fmt.Errorf("critical table %q is missing", schemaTable)
	}
	for _, field := range insert.fields {
		if _, ok := columns[field]; !ok {
			return fmt.Errorf("critical table %q missing required column %q", table, field)
		}
	}
	return nil
}

func (s *Service) verifyRestoreResult(ctx context.Context, doc BundleDoc, planned []restoreSectionPlan, result *RestoreResult) {
	if result == nil {
		return
	}
	rowCounts := map[string]int{}
	checksums := map[string]string{}
	nonFatal := []string{}
	for _, sec := range planned {
		count, err := s.countTableRows(ctx, sec.name)
		if err != nil {
			nonFatal = append(nonFatal, fmt.Sprintf("%s row count unavailable: %v", sec.name, err))
			continue
		}
		rowCounts[sec.name] = count
		if rows, err := s.extractSection(ctx, sec.name); err == nil {
			checksums[sec.name] = checksumRows(rows)
		} else {
			nonFatal = append(nonFatal, fmt.Sprintf("%s checksum unavailable: %v", sec.name, err))
		}
	}
	if len(result.ExportOnly) > 0 {
		nonFatal = append(nonFatal, "export-only/rebuildable sections were intentionally not restored")
	}
	result.Verification["schema"] = "passed"
	result.Verification["rowCounts"] = rowCounts
	result.Verification["checksums"] = checksums
	result.Verification["nonFatalGaps"] = nonFatal
	_ = doc
}

func (s *Service) countTableRows(ctx context.Context, table string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, fmt.Sprintf(`SELECT COUNT(1) FROM %s`, table)).Scan(&count)
	return count, err
}

func normalizeRestoreRecord(table string, rec map[string]any) {
	if table != "approval_requests" || rec == nil {
		return
	}
	if _, ok := rec["expires_at"]; !ok || rec["expires_at"] == nil {
		createdAt := restoreInt64(rec["created_at"])
		rec["expires_at"] = createdAt + int64(24*time.Hour/time.Millisecond)
	}
	if _, ok := rec["expired_at"]; !ok || rec["expired_at"] == nil {
		rec["expired_at"] = int64(0)
	}
}

func restoreInt64(v any) int64 {
	switch t := v.(type) {
	case int64:
		return t
	case int:
		return int64(t)
	case float64:
		return int64(t)
	case json.Number:
		n, _ := t.Int64()
		return n
	default:
		return 0
	}
}

func checksumRows(rows []any) string {
	raw, err := json.Marshal(rows)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func backupSectionManifest(section string) SectionManifest {
	entry := SectionManifest{
		Name:                   section,
		Purpose:                "durable FORGE section",
		AuthorityClass:         "operational_projection",
		BackupRequired:         true,
		RestoreRequired:        true,
		ExportOnly:             false,
		IntegrityCheckRequired: true,
	}
	switch section {
	case "memory_notes", "semantic_links", "state_items", "state_versions", "open_loops", "derived_models", "contradiction_records", "supersession_records":
		entry.AuthorityClass = "canonical_or_historical_truth"
		entry.Purpose = "cognitive filesystem durable truth"
	case "journal_events":
		entry.AuthorityClass = "historical_truth"
		entry.Purpose = "append-only semantic syscall journal"
	case "context_packet_snapshots":
		entry.AuthorityClass = "non_canonical_evidence"
		entry.Purpose = "context restore snapshot evidence and scoring metadata"
	case "dream_reports":
		entry.AuthorityClass = "non_canonical_evidence"
		entry.Purpose = "Dream Mode dry-run replay, salience, memory-tier, repair, and snapshot hygiene report evidence"
	case "artifact_refs", "artifacts", "gateway_invocations", "audit_records", "approval_requests", "approval_decisions", "provenance_records":
		entry.AuthorityClass = "non_canonical_evidence"
		entry.Purpose = "audit, approval, provenance, gateway, or artifact evidence"
	case "model_manifests", "model_registry_status", "model_runtime_loads":
		entry.AuthorityClass = "operational_projection"
		entry.Purpose = "governed modelruntime registry and lifecycle state"
	case "memory_vsa_pointers", "memory_vsa_role_bindings", "memory_vsa_associations", "retrieval_result_vsa_signals", "memory_vsa_reindex_runs", "memory_vsa_reindex_items", "embedding_records":
		entry.AuthorityClass = "retrieval_index"
		entry.Purpose = "retrieval support/index state; rebuild after restore when exact semantics are not guaranteed"
		if _, exportOnly := restoreExportOnlyReason(section); exportOnly {
			entry.RestoreRequired = false
			entry.ExportOnly = true
		}
	case "secrets_vault", "identity_tokens":
		entry.AuthorityClass = "secret_operational_state"
		entry.Purpose = "sensitive local credential state; intentionally not exported by full_backup"
		entry.BackupRequired = false
		entry.RestoreRequired = false
		entry.ExportOnly = false
	}
	return entry
}

func appendStringSlice(v any, item string) []string {
	out, _ := v.([]string)
	return append(out, item)
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
