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
	"sort"
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
	FilePath string   `json:"filePath"`
	Sections []string `json:"sections"`
	DryRun   bool     `json:"dryRun"`
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
			"context_packet_snapshots", "dream_reports", "semantic_idempotency_keys", "autonomy_settings",
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

type insertMap struct {
	sql    string
	fields []string
}

var extractQueries = map[string]string{
	"sources":                       "SELECT * FROM sources ORDER BY id ASC",
	"files":                         "SELECT * FROM files ORDER BY id ASC",
	"chunks":                        "SELECT * FROM chunks ORDER BY id ASC",
	"embedding_records":             "SELECT * FROM embedding_records ORDER BY id ASC",
	"retrieval_runs":                "SELECT * FROM retrieval_runs ORDER BY id ASC",
	"retrieval_results":             "SELECT * FROM retrieval_results ORDER BY id ASC",
	"retrieval_result_selection":    "SELECT * FROM retrieval_result_selection ORDER BY retrieval_result_id ASC",
	"packet_retrieval_runs":         "SELECT * FROM packet_retrieval_runs ORDER BY packet_id ASC, retrieval_run_id ASC",
	"dossiers":                      "SELECT * FROM dossiers",
	"dossier_sources":               "SELECT * FROM dossier_sources ORDER BY dossier_id ASC, source_id ASC",
	"dossier_jobs":                  "SELECT * FROM dossier_jobs ORDER BY dossier_id ASC, job_id ASC",
	"dossier_packets":               "SELECT * FROM dossier_packets ORDER BY dossier_id ASC, packet_id ASC",
	"dossier_briefs":                "SELECT * FROM dossier_briefs ORDER BY id ASC",
	"context_evidence":              "SELECT * FROM context_evidence ORDER BY id ASC",
	"task_packets":                  "SELECT * FROM task_packets",
	"project_context_records":       "SELECT * FROM project_context_records",
	"events":                        "SELECT * FROM events ORDER BY id ASC",
	"jobs":                          "SELECT * FROM jobs ORDER BY created_at DESC",
	"job_status_history":            "SELECT * FROM job_status_history ORDER BY id ASC",
	"job_events":                    "SELECT * FROM job_events ORDER BY id ASC",
	"approval_requests":             "SELECT * FROM approval_requests ORDER BY id ASC",
	"approval_decisions":            "SELECT * FROM approval_decisions ORDER BY id ASC",
	"artifacts":                     "SELECT * FROM artifacts ORDER BY id ASC",
	"execution_strategies":          "SELECT * FROM execution_strategies",
	"approval_presets":              "SELECT * FROM approval_presets",
	"permission_profiles":           "SELECT * FROM permission_profiles",
	"dossier_profiles":              "SELECT * FROM dossier_profiles",
	"automation_rules":              "SELECT * FROM automation_rules",
	"evaluation_records":            "SELECT * FROM evaluation_records",
	"memory_observations":           "SELECT * FROM memory_observations ORDER BY id ASC",
	"memory_observation_links":      "SELECT * FROM memory_observation_links ORDER BY id ASC",
	"retrieval_result_observations": "SELECT * FROM retrieval_result_observations ORDER BY retrieval_result_id ASC, observation_id ASC",
	"memory_usefulness_events":      "SELECT * FROM memory_usefulness_events ORDER BY id ASC",
	"packet_alignment_notes":        "SELECT * FROM packet_alignment_notes ORDER BY id ASC",
	"memory_repair_runs":            "SELECT * FROM memory_repair_runs ORDER BY id ASC",
	"memory_repair_items":           "SELECT * FROM memory_repair_items ORDER BY id ASC",
	"audit_records":                 "SELECT * FROM audit_records ORDER BY id DESC LIMIT 5000",
	"gateway_invocations":           "SELECT * FROM gateway_invocations ORDER BY id DESC LIMIT 5000",
	"action_lanes":                  "SELECT * FROM action_lanes",
	"model_manifests":               "SELECT * FROM model_manifests ORDER BY id ASC",
	"model_registry_status":         "SELECT * FROM model_registry_status ORDER BY model_id ASC",
	"model_runtime_loads":           "SELECT * FROM model_runtime_loads ORDER BY id ASC",
	"provenance_records":            "SELECT * FROM provenance_records ORDER BY created_at DESC",
	"journal_events":                "SELECT * FROM journal_events ORDER BY created_at DESC",
	"memory_notes":                  "SELECT * FROM memory_notes ORDER BY updated_at DESC",
	"semantic_links":                "SELECT * FROM semantic_links ORDER BY created_at DESC",
	"state_items":                   "SELECT * FROM state_items ORDER BY updated_at DESC",
	"state_versions":                "SELECT * FROM state_versions ORDER BY id DESC",
	"open_loops":                    "SELECT * FROM open_loops ORDER BY updated_at DESC",
	"artifact_refs":                 "SELECT * FROM artifact_refs ORDER BY created_at DESC",
	"derived_models":                "SELECT * FROM derived_models ORDER BY updated_at DESC",
	"contradiction_records":         "SELECT * FROM contradiction_records ORDER BY created_at DESC",
	"supersession_records":          "SELECT * FROM supersession_records ORDER BY created_at DESC",
	"context_packet_snapshots":      "SELECT * FROM context_packet_snapshots ORDER BY created_at DESC",
	"dream_reports":                 "SELECT * FROM dream_reports ORDER BY created_at DESC",
	"semantic_idempotency_keys":     "SELECT * FROM semantic_idempotency_keys ORDER BY created_at DESC, idempotency_key ASC",
	"autonomy_settings":             "SELECT key, value FROM settings WHERE key LIKE 'autonomy_repo.%' ORDER BY key ASC",
	"memory_vsa_pointers":           "SELECT * FROM memory_vsa_pointers ORDER BY updated_at DESC",
	"memory_vsa_role_bindings":      "SELECT * FROM memory_vsa_role_bindings ORDER BY updated_at DESC",
	"memory_vsa_associations":       "SELECT * FROM memory_vsa_associations ORDER BY updated_at DESC",
	"retrieval_result_vsa_signals":  "SELECT * FROM retrieval_result_vsa_signals ORDER BY created_at DESC",
	"memory_vsa_reindex_runs":       "SELECT * FROM memory_vsa_reindex_runs ORDER BY id DESC",
	"memory_vsa_reindex_items":      "SELECT * FROM memory_vsa_reindex_items ORDER BY id DESC",
	"chat_threads":                  "SELECT * FROM chat_threads ORDER BY id ASC",
	"chat_messages":                 "SELECT * FROM chat_messages ORDER BY id ASC",
	"canvas_boards":                 "SELECT * FROM canvas_boards ORDER BY id ASC",
	"canvas_notes":                  "SELECT * FROM canvas_notes ORDER BY id ASC",
	"tool_capability_overrides":     "SELECT * FROM tool_capability_overrides ORDER BY capability_id ASC",
	"feature_flags":                 "SELECT * FROM feature_flags ORDER BY key ASC",
	"alert_rules":                   "SELECT * FROM alert_rules ORDER BY id ASC",
	"scheduled_tasks":               "SELECT * FROM scheduled_tasks ORDER BY id ASC",
}

// Section upserts used during restore.
var insertStatements = func() map[string]insertMap {
	m := map[string]insertMap{
		"dossiers": {
			sql: `INSERT INTO dossiers(
  id, created_at, updated_at, name, description, primary_paths_json, related_repos_json,
  constraints_json, preferred_adapters_json, important_files_json, routing_notes
) VALUES(?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  created_at=excluded.created_at,
  updated_at=excluded.updated_at,
  name=excluded.name,
  description=excluded.description,
  primary_paths_json=excluded.primary_paths_json,
  related_repos_json=excluded.related_repos_json,
  constraints_json=excluded.constraints_json,
  preferred_adapters_json=excluded.preferred_adapters_json,
  important_files_json=excluded.important_files_json,
  routing_notes=excluded.routing_notes`,
			fields: []string{
				"id", "created_at", "updated_at", "name", "description", "primary_paths_json", "related_repos_json",
				"constraints_json", "preferred_adapters_json", "important_files_json", "routing_notes",
			},
		},
		"task_packets": {
			sql: `INSERT INTO task_packets(
  id, packet_version, created_at, generated_at, title, user_request, objective,
  adapter_target, execution_mode, risk_class, expected_output_json, constraints_json,
  instructions, selected_paths_json, scope_snapshot_json, source_references_json,
  retrieved_context_json, project_notes, source_context_record_ids_json, request_payload_json
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  packet_version=excluded.packet_version,
  created_at=excluded.created_at,
  generated_at=excluded.generated_at,
  title=excluded.title,
  user_request=excluded.user_request,
  objective=excluded.objective,
  adapter_target=excluded.adapter_target,
  execution_mode=excluded.execution_mode,
  risk_class=excluded.risk_class,
  expected_output_json=excluded.expected_output_json,
  constraints_json=excluded.constraints_json,
  instructions=excluded.instructions,
  selected_paths_json=excluded.selected_paths_json,
  scope_snapshot_json=excluded.scope_snapshot_json,
  source_references_json=excluded.source_references_json,
  retrieved_context_json=excluded.retrieved_context_json,
  project_notes=excluded.project_notes,
  source_context_record_ids_json=excluded.source_context_record_ids_json,
  request_payload_json=excluded.request_payload_json`,
			fields: []string{
				"id", "packet_version", "created_at", "generated_at", "title", "user_request", "objective",
				"adapter_target", "execution_mode", "risk_class", "expected_output_json", "constraints_json",
				"instructions", "selected_paths_json", "scope_snapshot_json", "source_references_json",
				"retrieved_context_json", "project_notes", "source_context_record_ids_json", "request_payload_json",
			},
		},
		"project_context_records": {
			sql: `INSERT INTO project_context_records(
  id, context_version, created_at, generated_at, source_path, source_hash, source_size_bytes,
  normalized_summary_json, briefing_markdown, agents_markdown, claude_markdown, cursor_markdown,
  generated_agents_path, generated_claude_path, generated_briefing_path, generated_cursor_path, notes
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  context_version=excluded.context_version,
  created_at=excluded.created_at,
  generated_at=excluded.generated_at,
  source_path=excluded.source_path,
  source_hash=excluded.source_hash,
  source_size_bytes=excluded.source_size_bytes,
  normalized_summary_json=excluded.normalized_summary_json,
  briefing_markdown=excluded.briefing_markdown,
  agents_markdown=excluded.agents_markdown,
  claude_markdown=excluded.claude_markdown,
  cursor_markdown=excluded.cursor_markdown,
  generated_agents_path=excluded.generated_agents_path,
  generated_claude_path=excluded.generated_claude_path,
  generated_briefing_path=excluded.generated_briefing_path,
  generated_cursor_path=excluded.generated_cursor_path,
  notes=excluded.notes`,
			fields: []string{
				"id", "context_version", "created_at", "generated_at", "source_path", "source_hash", "source_size_bytes",
				"normalized_summary_json", "briefing_markdown", "agents_markdown", "claude_markdown", "cursor_markdown",
				"generated_agents_path", "generated_claude_path", "generated_briefing_path", "generated_cursor_path", "notes",
			},
		},
		"events": {
			sql: `INSERT INTO events(id, created_at, type, payload_json)
VALUES(?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  created_at=excluded.created_at,
  type=excluded.type,
  payload_json=excluded.payload_json`,
			fields: []string{"id", "created_at", "type", "payload_json"},
		},
		"jobs": {
			sql: `INSERT INTO jobs(
  id, created_at, updated_at, queued_at, started_at, completed_at, title,
  requested_action, target_adapter, initiating_source, execution_boundary,
  risk_class, status, approval_status, write_intent, cancel_requested,
  task_packet_id, result_summary, failure_info, last_failure_code, last_error, metadata_json
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  created_at=excluded.created_at,
  updated_at=excluded.updated_at,
  queued_at=excluded.queued_at,
  started_at=excluded.started_at,
  completed_at=excluded.completed_at,
  title=excluded.title,
  requested_action=excluded.requested_action,
  target_adapter=excluded.target_adapter,
  initiating_source=excluded.initiating_source,
  execution_boundary=excluded.execution_boundary,
  risk_class=excluded.risk_class,
  status=excluded.status,
  approval_status=excluded.approval_status,
  write_intent=excluded.write_intent,
  cancel_requested=excluded.cancel_requested,
  task_packet_id=excluded.task_packet_id,
  result_summary=excluded.result_summary,
  failure_info=excluded.failure_info,
  last_failure_code=excluded.last_failure_code,
  last_error=excluded.last_error,
  metadata_json=excluded.metadata_json`,
			fields: []string{
				"id", "created_at", "updated_at", "queued_at", "started_at", "completed_at", "title",
				"requested_action", "target_adapter", "initiating_source", "execution_boundary",
				"risk_class", "status", "approval_status", "write_intent", "cancel_requested",
				"task_packet_id", "result_summary", "failure_info", "last_failure_code", "last_error", "metadata_json",
			},
		},
		"job_status_history": {
			sql: `INSERT INTO job_status_history(id, job_id, created_at, from_status, to_status, reason)
VALUES(?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  job_id=excluded.job_id,
  created_at=excluded.created_at,
  from_status=excluded.from_status,
  to_status=excluded.to_status,
  reason=excluded.reason`,
			fields: []string{"id", "job_id", "created_at", "from_status", "to_status", "reason"},
		},
		"job_events": {
			sql: `INSERT INTO job_events(id, job_id, created_at, type, message, payload_json)
VALUES(?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  job_id=excluded.job_id,
  created_at=excluded.created_at,
  type=excluded.type,
  message=excluded.message,
  payload_json=excluded.payload_json`,
			fields: []string{"id", "job_id", "created_at", "type", "message", "payload_json"},
		},
		"approval_requests": {
			sql: `INSERT INTO approval_requests(
	id, job_id, created_at, status, requested_action, risk_class, requested_adapter,
  write_intent, scope_snapshot_json, task_packet_id, request_summary, expires_at, expired_at
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  job_id=excluded.job_id,
  created_at=excluded.created_at,
  status=excluded.status,
  requested_action=excluded.requested_action,
  risk_class=excluded.risk_class,
  requested_adapter=excluded.requested_adapter,
  write_intent=excluded.write_intent,
  scope_snapshot_json=excluded.scope_snapshot_json,
  task_packet_id=excluded.task_packet_id,
  request_summary=excluded.request_summary,
  expires_at=excluded.expires_at,
  expired_at=excluded.expired_at`,
			fields: []string{
				"id", "job_id", "created_at", "status", "requested_action", "risk_class",
				"requested_adapter", "write_intent", "scope_snapshot_json", "task_packet_id", "request_summary",
				"expires_at", "expired_at",
			},
		},
		"approval_decisions": {
			sql: `INSERT INTO approval_decisions(id, request_id, created_at, actor, decision, note)
VALUES(?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  request_id=excluded.request_id,
  created_at=excluded.created_at,
  actor=excluded.actor,
  decision=excluded.decision,
  note=excluded.note`,
			fields: []string{"id", "request_id", "created_at", "actor", "decision", "note"},
		},
		"artifacts": {
			sql: `INSERT INTO artifacts(
  id, created_at, job_id, packet_id, type, title, file_path, mime_type, metadata_json
) VALUES(?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  created_at=excluded.created_at,
  job_id=excluded.job_id,
  packet_id=excluded.packet_id,
  type=excluded.type,
  title=excluded.title,
  file_path=excluded.file_path,
  mime_type=excluded.mime_type,
  metadata_json=excluded.metadata_json`,
			fields: []string{"id", "created_at", "job_id", "packet_id", "type", "title", "file_path", "mime_type", "metadata_json"},
		},
		"evaluation_records": {
			sql: `INSERT INTO evaluation_records(
  id, created_at, job_id, dossier_id, success, quality_rating, usefulness_rating, correctness_confidence,
  packet_quality_rating, adapter_suitability, retry_recommended, influence_routing, notes, scorer
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  created_at=excluded.created_at,
  job_id=excluded.job_id,
  dossier_id=excluded.dossier_id,
  success=excluded.success,
  quality_rating=excluded.quality_rating,
  usefulness_rating=excluded.usefulness_rating,
  correctness_confidence=excluded.correctness_confidence,
  packet_quality_rating=excluded.packet_quality_rating,
  adapter_suitability=excluded.adapter_suitability,
  retry_recommended=excluded.retry_recommended,
  influence_routing=excluded.influence_routing,
  notes=excluded.notes,
  scorer=excluded.scorer`,
			fields: []string{
				"id", "created_at", "job_id", "dossier_id", "success", "quality_rating", "usefulness_rating", "correctness_confidence",
				"packet_quality_rating", "adapter_suitability", "retry_recommended", "influence_routing", "notes", "scorer",
			},
		},
		"autonomy_settings": {
			sql: `INSERT INTO settings(key, value)
VALUES(?, ?)
ON CONFLICT(key) DO UPDATE SET value=excluded.value`,
			fields: []string{"key", "value"},
		},
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
		"dossier_profiles": {
			sql: `INSERT INTO dossier_profiles(
  dossier_id, updated_at, preferred_strategies_json, preferred_adapters_json,
  approval_preset_id, retrieval_defaults_json, high_value_files_json,
  noisy_files_json, routing_notes, automation_bindings_json
) VALUES(?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(dossier_id) DO UPDATE SET
  updated_at=excluded.updated_at,
  preferred_strategies_json=excluded.preferred_strategies_json,
  preferred_adapters_json=excluded.preferred_adapters_json,
  approval_preset_id=excluded.approval_preset_id,
  retrieval_defaults_json=excluded.retrieval_defaults_json,
  high_value_files_json=excluded.high_value_files_json,
  noisy_files_json=excluded.noisy_files_json,
  routing_notes=excluded.routing_notes,
  automation_bindings_json=excluded.automation_bindings_json`,
			fields: []string{
				"dossier_id", "updated_at", "preferred_strategies_json", "preferred_adapters_json",
				"approval_preset_id", "retrieval_defaults_json", "high_value_files_json",
				"noisy_files_json", "routing_notes", "automation_bindings_json",
			},
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
		"automation_rules": {
			sql: `INSERT INTO automation_rules(
  id, created_at, updated_at, name, trigger, condition_json, action_json, scope_json, enabled, dry_run_default
) VALUES(?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  created_at=excluded.created_at,
  updated_at=excluded.updated_at,
  name=excluded.name,
  trigger=excluded.trigger,
  condition_json=excluded.condition_json,
  action_json=excluded.action_json,
  scope_json=excluded.scope_json,
  enabled=excluded.enabled,
  dry_run_default=excluded.dry_run_default`,
			fields: []string{
				"id", "created_at", "updated_at", "name", "trigger", "condition_json", "action_json", "scope_json", "enabled", "dry_run_default",
			},
		},
		"action_lanes": {
			sql: `INSERT INTO action_lanes(
  id, created_at, updated_at, name, description, action_type, allowed_paths_json,
  forbidden_paths_json, write_intent, requires_approval, risk_class, max_bytes,
  expected_artifacts_json, builtin, enabled
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  created_at=excluded.created_at,
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
  builtin=excluded.builtin,
  enabled=excluded.enabled`,
			fields: []string{
				"id", "created_at", "updated_at", "name", "description", "action_type", "allowed_paths_json",
				"forbidden_paths_json", "write_intent", "requires_approval", "risk_class", "max_bytes",
				"expected_artifacts_json", "builtin", "enabled",
			},
		},
		"provenance_records": {
			sql: `INSERT INTO provenance_records(
  id, actor, actor_type, source, trace_id, workspace_id, lane_id, selected_paths_json,
  metadata_json, created_at, proposed_by, committed_by, syscall_id, correlation_id, audit_id
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  actor=excluded.actor,
  actor_type=excluded.actor_type,
  source=excluded.source,
  trace_id=excluded.trace_id,
  workspace_id=excluded.workspace_id,
  lane_id=excluded.lane_id,
  selected_paths_json=excluded.selected_paths_json,
  metadata_json=excluded.metadata_json,
  created_at=excluded.created_at,
  proposed_by=excluded.proposed_by,
  committed_by=excluded.committed_by,
  syscall_id=excluded.syscall_id,
  correlation_id=excluded.correlation_id,
  audit_id=excluded.audit_id`,
			fields: []string{
				"id", "actor", "actor_type", "source", "trace_id", "workspace_id", "lane_id", "selected_paths_json",
				"metadata_json", "created_at", "proposed_by", "committed_by", "syscall_id", "correlation_id", "audit_id",
			},
		},
		"gateway_invocations": {
			sql: `INSERT INTO gateway_invocations(
  id, correlation_id, created_at, completed_at, tool_id, lane_id, job_id, packet_id,
  approval_request_id, initiator, action, risk_class, write_intent, scope_json, input_json,
  status, denied_reason, result_json, artifacts_json, permission_profile_id
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  correlation_id=excluded.correlation_id,
  created_at=excluded.created_at,
  completed_at=excluded.completed_at,
  tool_id=excluded.tool_id,
  lane_id=excluded.lane_id,
  job_id=excluded.job_id,
  packet_id=excluded.packet_id,
  approval_request_id=excluded.approval_request_id,
  initiator=excluded.initiator,
  action=excluded.action,
  risk_class=excluded.risk_class,
  write_intent=excluded.write_intent,
  scope_json=excluded.scope_json,
  input_json=excluded.input_json,
  status=excluded.status,
  denied_reason=excluded.denied_reason,
  result_json=excluded.result_json,
  artifacts_json=excluded.artifacts_json,
  permission_profile_id=excluded.permission_profile_id`,
			fields: []string{
				"id", "correlation_id", "created_at", "completed_at", "tool_id", "lane_id", "job_id", "packet_id",
				"approval_request_id", "initiator", "action", "risk_class", "write_intent", "scope_json", "input_json",
				"status", "denied_reason", "result_json", "artifacts_json", "permission_profile_id",
			},
		},
		"audit_records": {
			sql: `INSERT INTO audit_records(
  id, created_at, correlation_id, category, action, actor, subject_type, subject_id,
  job_id, gateway_invocation_id, approval_request_id, risk_class, outcome, summary, payload_json
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO NOTHING`,
			fields: []string{
				"id", "created_at", "correlation_id", "category", "action", "actor", "subject_type", "subject_id",
				"job_id", "gateway_invocation_id", "approval_request_id", "risk_class", "outcome", "summary", "payload_json",
			},
		},
		"journal_events": {
			sql: `INSERT INTO journal_events(
  id, type, source, actor, workspace_id, lane_id, selected_paths_json, payload_json,
  correlation_id, trace_id, provenance_id, provenance_json, created_at, metadata_json,
  proposed_by, committed_by, syscall_id, audit_id
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO NOTHING`,
			fields: []string{
				"id", "type", "source", "actor", "workspace_id", "lane_id", "selected_paths_json", "payload_json",
				"correlation_id", "trace_id", "provenance_id", "provenance_json", "created_at", "metadata_json",
				"proposed_by", "committed_by", "syscall_id", "audit_id",
			},
		},
		"memory_notes": {
			sql: `INSERT INTO memory_notes(
  id, type, title, content, workspace_id, lane_id, selected_paths_json, confidence, status,
  provenance_id, provenance_json, created_at, updated_at, archived_at, superseded_by, metadata_json,
  proposed_by, committed_by, syscall_id, correlation_id, trace_id, audit_id
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  type=excluded.type,
  title=excluded.title,
  content=excluded.content,
  workspace_id=excluded.workspace_id,
  lane_id=excluded.lane_id,
  selected_paths_json=excluded.selected_paths_json,
  confidence=excluded.confidence,
  status=excluded.status,
  provenance_id=excluded.provenance_id,
  provenance_json=excluded.provenance_json,
  created_at=excluded.created_at,
  updated_at=excluded.updated_at,
  archived_at=excluded.archived_at,
  superseded_by=excluded.superseded_by,
  metadata_json=excluded.metadata_json,
  proposed_by=excluded.proposed_by,
  committed_by=excluded.committed_by,
  syscall_id=excluded.syscall_id,
  correlation_id=excluded.correlation_id,
  trace_id=excluded.trace_id,
  audit_id=excluded.audit_id`,
			fields: []string{
				"id", "type", "title", "content", "workspace_id", "lane_id", "selected_paths_json", "confidence", "status",
				"provenance_id", "provenance_json", "created_at", "updated_at", "archived_at", "superseded_by", "metadata_json",
				"proposed_by", "committed_by", "syscall_id", "correlation_id", "trace_id", "audit_id",
			},
		},
		"semantic_links": {
			sql: `INSERT INTO semantic_links(
  id, type, source_id, source_kind, target_id, target_kind, confidence, provenance_id,
  provenance_json, workspace_id, lane_id, selected_paths_json, created_at, metadata_json,
  proposed_by, committed_by, syscall_id, correlation_id, trace_id, audit_id
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  type=excluded.type,
  source_id=excluded.source_id,
  source_kind=excluded.source_kind,
  target_id=excluded.target_id,
  target_kind=excluded.target_kind,
  confidence=excluded.confidence,
  provenance_id=excluded.provenance_id,
  provenance_json=excluded.provenance_json,
  workspace_id=excluded.workspace_id,
  lane_id=excluded.lane_id,
  selected_paths_json=excluded.selected_paths_json,
  created_at=excluded.created_at,
  metadata_json=excluded.metadata_json,
  proposed_by=excluded.proposed_by,
  committed_by=excluded.committed_by,
  syscall_id=excluded.syscall_id,
  correlation_id=excluded.correlation_id,
  trace_id=excluded.trace_id,
  audit_id=excluded.audit_id`,
			fields: []string{
				"id", "type", "source_id", "source_kind", "target_id", "target_kind", "confidence", "provenance_id",
				"provenance_json", "workspace_id", "lane_id", "selected_paths_json", "created_at", "metadata_json",
				"proposed_by", "committed_by", "syscall_id", "correlation_id", "trace_id", "audit_id",
			},
		},
		"state_items": {
			sql: `INSERT INTO state_items(
  id, key, value_json, workspace_id, lane_id, selected_paths_json, status,
  derived_from_json, current_version, updated_at, metadata_json, proposed_by,
  committed_by, syscall_id, correlation_id, trace_id, audit_id
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  key=excluded.key,
  value_json=excluded.value_json,
  workspace_id=excluded.workspace_id,
  lane_id=excluded.lane_id,
  selected_paths_json=excluded.selected_paths_json,
  status=excluded.status,
  derived_from_json=excluded.derived_from_json,
  current_version=excluded.current_version,
  updated_at=excluded.updated_at,
  metadata_json=excluded.metadata_json,
  proposed_by=excluded.proposed_by,
  committed_by=excluded.committed_by,
  syscall_id=excluded.syscall_id,
  correlation_id=excluded.correlation_id,
  trace_id=excluded.trace_id,
  audit_id=excluded.audit_id`,
			fields: []string{
				"id", "key", "value_json", "workspace_id", "lane_id", "selected_paths_json", "status",
				"derived_from_json", "current_version", "updated_at", "metadata_json", "proposed_by",
				"committed_by", "syscall_id", "correlation_id", "trace_id", "audit_id",
			},
		},
		"state_versions": {
			sql: `INSERT INTO state_versions(
  id, state_item_id, state_key, workspace_id, lane_id, previous_value_json, new_value_json,
  changed_by, derived_from_json, syscall_id, audit_id, correlation_id, trace_id, created_at,
  metadata_json, proposed_by, committed_by
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  state_item_id=excluded.state_item_id,
  state_key=excluded.state_key,
  workspace_id=excluded.workspace_id,
  lane_id=excluded.lane_id,
  previous_value_json=excluded.previous_value_json,
  new_value_json=excluded.new_value_json,
  changed_by=excluded.changed_by,
  derived_from_json=excluded.derived_from_json,
  syscall_id=excluded.syscall_id,
  audit_id=excluded.audit_id,
  correlation_id=excluded.correlation_id,
  trace_id=excluded.trace_id,
  created_at=excluded.created_at,
  metadata_json=excluded.metadata_json,
  proposed_by=excluded.proposed_by,
  committed_by=excluded.committed_by`,
			fields: []string{
				"id", "state_item_id", "state_key", "workspace_id", "lane_id", "previous_value_json", "new_value_json",
				"changed_by", "derived_from_json", "syscall_id", "audit_id", "correlation_id", "trace_id", "created_at",
				"metadata_json", "proposed_by", "committed_by",
			},
		},
		"open_loops": {
			sql: `INSERT INTO open_loops(
  id, title, state, priority, owner, blocker, next_action, related_notes_json, created_from,
  workspace_id, lane_id, selected_paths_json, created_at, updated_at, resolved_at, archived_at,
  metadata_json, proposed_by, committed_by, syscall_id, correlation_id, trace_id, audit_id
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  title=excluded.title,
  state=excluded.state,
  priority=excluded.priority,
  owner=excluded.owner,
  blocker=excluded.blocker,
  next_action=excluded.next_action,
  related_notes_json=excluded.related_notes_json,
  created_from=excluded.created_from,
  workspace_id=excluded.workspace_id,
  lane_id=excluded.lane_id,
  selected_paths_json=excluded.selected_paths_json,
  created_at=excluded.created_at,
  updated_at=excluded.updated_at,
  resolved_at=excluded.resolved_at,
  archived_at=excluded.archived_at,
  metadata_json=excluded.metadata_json,
  proposed_by=excluded.proposed_by,
  committed_by=excluded.committed_by,
  syscall_id=excluded.syscall_id,
  correlation_id=excluded.correlation_id,
  trace_id=excluded.trace_id,
  audit_id=excluded.audit_id`,
			fields: []string{
				"id", "title", "state", "priority", "owner", "blocker", "next_action", "related_notes_json", "created_from",
				"workspace_id", "lane_id", "selected_paths_json", "created_at", "updated_at", "resolved_at", "archived_at",
				"metadata_json", "proposed_by", "committed_by", "syscall_id", "correlation_id", "trace_id", "audit_id",
			},
		},
		"artifact_refs": {
			sql: `INSERT INTO artifact_refs(
  id, type, uri, content_hash, workspace_id, lane_id, selected_paths_json, provenance_id,
  provenance_json, created_at, metadata_json, proposed_by, committed_by, syscall_id,
  correlation_id, trace_id, audit_id
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  type=excluded.type,
  uri=excluded.uri,
  content_hash=excluded.content_hash,
  workspace_id=excluded.workspace_id,
  lane_id=excluded.lane_id,
  selected_paths_json=excluded.selected_paths_json,
  provenance_id=excluded.provenance_id,
  provenance_json=excluded.provenance_json,
  created_at=excluded.created_at,
  metadata_json=excluded.metadata_json,
  proposed_by=excluded.proposed_by,
  committed_by=excluded.committed_by,
  syscall_id=excluded.syscall_id,
  correlation_id=excluded.correlation_id,
  trace_id=excluded.trace_id,
  audit_id=excluded.audit_id`,
			fields: []string{
				"id", "type", "uri", "content_hash", "workspace_id", "lane_id", "selected_paths_json", "provenance_id",
				"provenance_json", "created_at", "metadata_json", "proposed_by", "committed_by", "syscall_id",
				"correlation_id", "trace_id", "audit_id",
			},
		},
		"derived_models": {
			sql: `INSERT INTO derived_models(
  id, type, expression_json, derived_from_json, support_count, confidence, status, workspace_id,
  lane_id, selected_paths_json, last_validated_at, created_at, updated_at, metadata_json,
  proposed_by, committed_by, syscall_id, correlation_id, trace_id, audit_id
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  type=excluded.type,
  expression_json=excluded.expression_json,
  derived_from_json=excluded.derived_from_json,
  support_count=excluded.support_count,
  confidence=excluded.confidence,
  status=excluded.status,
  workspace_id=excluded.workspace_id,
  lane_id=excluded.lane_id,
  selected_paths_json=excluded.selected_paths_json,
  last_validated_at=excluded.last_validated_at,
  created_at=excluded.created_at,
  updated_at=excluded.updated_at,
  metadata_json=excluded.metadata_json,
  proposed_by=excluded.proposed_by,
  committed_by=excluded.committed_by,
  syscall_id=excluded.syscall_id,
  correlation_id=excluded.correlation_id,
  trace_id=excluded.trace_id,
  audit_id=excluded.audit_id`,
			fields: []string{
				"id", "type", "expression_json", "derived_from_json", "support_count", "confidence", "status", "workspace_id",
				"lane_id", "selected_paths_json", "last_validated_at", "created_at", "updated_at", "metadata_json",
				"proposed_by", "committed_by", "syscall_id", "correlation_id", "trace_id", "audit_id",
			},
		},
		"contradiction_records": {
			sql: `INSERT INTO contradiction_records(
  id, left_object_id, left_object_kind, right_object_id, right_object_kind, reason, severity,
  confidence, provenance_id, provenance_json, workspace_id, lane_id, created_at, metadata_json,
  proposed_by, committed_by, syscall_id, correlation_id, trace_id, audit_id
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  left_object_id=excluded.left_object_id,
  left_object_kind=excluded.left_object_kind,
  right_object_id=excluded.right_object_id,
  right_object_kind=excluded.right_object_kind,
  reason=excluded.reason,
  severity=excluded.severity,
  confidence=excluded.confidence,
  provenance_id=excluded.provenance_id,
  provenance_json=excluded.provenance_json,
  workspace_id=excluded.workspace_id,
  lane_id=excluded.lane_id,
  created_at=excluded.created_at,
  metadata_json=excluded.metadata_json,
  proposed_by=excluded.proposed_by,
  committed_by=excluded.committed_by,
  syscall_id=excluded.syscall_id,
  correlation_id=excluded.correlation_id,
  trace_id=excluded.trace_id,
  audit_id=excluded.audit_id`,
			fields: []string{
				"id", "left_object_id", "left_object_kind", "right_object_id", "right_object_kind", "reason", "severity",
				"confidence", "provenance_id", "provenance_json", "workspace_id", "lane_id", "created_at", "metadata_json",
				"proposed_by", "committed_by", "syscall_id", "correlation_id", "trace_id", "audit_id",
			},
		},
		"supersession_records": {
			sql: `INSERT INTO supersession_records(
  id, old_object_id, old_object_kind, new_object_id, new_object_kind, reason, provenance_id,
  provenance_json, workspace_id, lane_id, created_at, metadata_json, proposed_by, committed_by,
  syscall_id, correlation_id, trace_id, audit_id
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  old_object_id=excluded.old_object_id,
  old_object_kind=excluded.old_object_kind,
  new_object_id=excluded.new_object_id,
  new_object_kind=excluded.new_object_kind,
  reason=excluded.reason,
  provenance_id=excluded.provenance_id,
  provenance_json=excluded.provenance_json,
  workspace_id=excluded.workspace_id,
  lane_id=excluded.lane_id,
  created_at=excluded.created_at,
  metadata_json=excluded.metadata_json,
  proposed_by=excluded.proposed_by,
  committed_by=excluded.committed_by,
  syscall_id=excluded.syscall_id,
  correlation_id=excluded.correlation_id,
  trace_id=excluded.trace_id,
  audit_id=excluded.audit_id`,
			fields: []string{
				"id", "old_object_id", "old_object_kind", "new_object_id", "new_object_kind", "reason", "provenance_id",
				"provenance_json", "workspace_id", "lane_id", "created_at", "metadata_json", "proposed_by", "committed_by",
				"syscall_id", "correlation_id", "trace_id", "audit_id",
			},
		},
		"context_packet_snapshots": {
			sql: `INSERT INTO context_packet_snapshots(
  id, query, workspace_id, lane_id, snapshot_kind, snapshot_fingerprint, parent_snapshot_id,
  selected_paths_json, included_state_json, included_open_loops_json, included_notes_json, included_links_json,
  included_models_json, included_artifacts_json, included_events_json, header_json, graph_json, delta_json,
  restore_scores_json, render_artifact_ref_id, resume_hints_json, budget_json, inclusion_reasons_json, created_at,
  correlation_id, trace_id, syscall_id, metadata_json, proposed_by, committed_by, audit_id
) VALUES(?,?,?,?,COALESCE(?,''),COALESCE(?,''),COALESCE(?,''),?,?,?,?,?,?,?,?,COALESCE(?, '{}'),COALESCE(?, '{}'),COALESCE(?, '{}'),COALESCE(?, '{}'),COALESCE(?,''),COALESCE(?, '{}'),?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  query=excluded.query,
  workspace_id=excluded.workspace_id,
  lane_id=excluded.lane_id,
  snapshot_kind=excluded.snapshot_kind,
  snapshot_fingerprint=excluded.snapshot_fingerprint,
  parent_snapshot_id=excluded.parent_snapshot_id,
  selected_paths_json=excluded.selected_paths_json,
  included_state_json=excluded.included_state_json,
  included_open_loops_json=excluded.included_open_loops_json,
  included_notes_json=excluded.included_notes_json,
  included_links_json=excluded.included_links_json,
  included_models_json=excluded.included_models_json,
  included_artifacts_json=excluded.included_artifacts_json,
  included_events_json=excluded.included_events_json,
  header_json=excluded.header_json,
  graph_json=excluded.graph_json,
  delta_json=excluded.delta_json,
  restore_scores_json=excluded.restore_scores_json,
  render_artifact_ref_id=excluded.render_artifact_ref_id,
  resume_hints_json=excluded.resume_hints_json,
  budget_json=excluded.budget_json,
  inclusion_reasons_json=excluded.inclusion_reasons_json,
  created_at=excluded.created_at,
  correlation_id=excluded.correlation_id,
  trace_id=excluded.trace_id,
  syscall_id=excluded.syscall_id,
  metadata_json=excluded.metadata_json,
  proposed_by=excluded.proposed_by,
  committed_by=excluded.committed_by,
  audit_id=excluded.audit_id`,
			fields: []string{
				"id", "query", "workspace_id", "lane_id", "snapshot_kind", "snapshot_fingerprint", "parent_snapshot_id",
				"selected_paths_json", "included_state_json", "included_open_loops_json", "included_notes_json", "included_links_json",
				"included_models_json", "included_artifacts_json", "included_events_json", "header_json", "graph_json", "delta_json",
				"restore_scores_json", "render_artifact_ref_id", "resume_hints_json", "budget_json", "inclusion_reasons_json", "created_at",
				"correlation_id", "trace_id", "syscall_id", "metadata_json", "proposed_by", "committed_by", "audit_id",
			},
		},
		"dream_reports": {
			sql: `INSERT INTO dream_reports(
  id, created_at, completed_at, workspace_id, lane_id, mode, dry_run, status,
  time_window_start, time_window_end, candidates_considered, proposals_generated,
  summary_json, candidates_json, salience_scores_json, memory_tier_proposals_json,
  repair_proposals_json, snapshot_hygiene_proposals_json, warnings_json, trace_json,
  correlation_id, trace_id, syscall_id, audit_id, proposed_by, committed_by, metadata_json
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  created_at=excluded.created_at,
  completed_at=excluded.completed_at,
  workspace_id=excluded.workspace_id,
  lane_id=excluded.lane_id,
  mode=excluded.mode,
  dry_run=excluded.dry_run,
  status=excluded.status,
  time_window_start=excluded.time_window_start,
  time_window_end=excluded.time_window_end,
  candidates_considered=excluded.candidates_considered,
  proposals_generated=excluded.proposals_generated,
  summary_json=excluded.summary_json,
  candidates_json=excluded.candidates_json,
  salience_scores_json=excluded.salience_scores_json,
  memory_tier_proposals_json=excluded.memory_tier_proposals_json,
  repair_proposals_json=excluded.repair_proposals_json,
  snapshot_hygiene_proposals_json=excluded.snapshot_hygiene_proposals_json,
  warnings_json=excluded.warnings_json,
  trace_json=excluded.trace_json,
  correlation_id=excluded.correlation_id,
  trace_id=excluded.trace_id,
  syscall_id=excluded.syscall_id,
  audit_id=excluded.audit_id,
  proposed_by=excluded.proposed_by,
  committed_by=excluded.committed_by,
  metadata_json=excluded.metadata_json`,
			fields: []string{
				"id", "created_at", "completed_at", "workspace_id", "lane_id", "mode", "dry_run", "status",
				"time_window_start", "time_window_end", "candidates_considered", "proposals_generated",
				"summary_json", "candidates_json", "salience_scores_json", "memory_tier_proposals_json",
				"repair_proposals_json", "snapshot_hygiene_proposals_json", "warnings_json", "trace_json",
				"correlation_id", "trace_id", "syscall_id", "audit_id", "proposed_by", "committed_by", "metadata_json",
			},
		},
		"semantic_idempotency_keys": {
			sql: `INSERT INTO semantic_idempotency_keys(
  idempotency_key, action, result_json, created_at, correlation_id
) VALUES(?,?,?,?,?)
ON CONFLICT(idempotency_key) DO NOTHING`,
			fields: []string{"idempotency_key", "action", "result_json", "created_at", "correlation_id"},
		},
	}
	addRestoreTable := func(table string, conflict []string, fields []string) {
		m[table] = buildRestoreInsert(table, conflict, fields, false)
	}
	addRestoreTable("sources", []string{"id"}, []string{"id", "path", "created_at", "last_scan_started_at", "last_scan_completed_at", "last_error"})
	addRestoreTable("files", []string{"id"}, []string{"id", "source_id", "rel_path", "abs_path", "size_bytes", "mtime_ns", "content_sha256", "indexed_at"})
	addRestoreTable("chunks", []string{"id"}, []string{"id", "file_id", "chunk_index", "content"})
	addRestoreTable("embedding_records", []string{"id"}, []string{"id", "chunk_id", "file_id", "source_id", "provider", "model", "vector_json", "dims", "norm", "content_sha256", "status", "error_message", "updated_at"})
	addRestoreTable("retrieval_runs", []string{"id"}, []string{"id", "created_at", "query", "mode", "dossier_id", "packet_id", "job_id", "weighting_json", "notes"})
	addRestoreTable("retrieval_results", []string{"id"}, []string{"id", "retrieval_run_id", "chunk_id", "file_id", "abs_path", "rel_path", "rank_index", "keyword_score", "semantic_score", "hybrid_score", "snippet", "selected_for_packet", "usefulness_label", "usefulness_note"})
	addRestoreTable("retrieval_result_selection", []string{"retrieval_result_id"}, []string{"retrieval_result_id", "reason_json", "created_at"})
	addRestoreTable("packet_retrieval_runs", []string{"packet_id", "retrieval_run_id"}, []string{"packet_id", "retrieval_run_id", "created_at"})
	addRestoreTable("dossier_sources", []string{"dossier_id", "source_id"}, []string{"dossier_id", "source_id", "linked_at"})
	addRestoreTable("dossier_jobs", []string{"dossier_id", "job_id"}, []string{"dossier_id", "job_id", "linked_at"})
	addRestoreTable("dossier_packets", []string{"dossier_id", "packet_id"}, []string{"dossier_id", "packet_id", "linked_at"})
	addRestoreTable("dossier_briefs", []string{"id"}, []string{"id", "dossier_id", "created_at", "summary_markdown", "context_json", "notes"})
	addRestoreTable("context_evidence", []string{"id"}, []string{"id", "created_at", "retrieval_result_id", "retrieval_run_id", "job_id", "packet_id", "chunk_id", "evidence_type", "weight", "note"})
	addRestoreTable("memory_observations", []string{"id"}, []string{"id", "created_at", "updated_at", "observed_at", "type", "raw_content", "summary", "embedding_ref", "dossier_id", "project_key", "source_path", "entities_json", "tags_json", "related_files_json", "task_type", "confidence", "verification_state", "lineage_json", "origin_kind", "origin_id", "stale", "last_verified_at", "usefulness_score", "usefulness_count", "noise_count"})
	addRestoreTable("memory_observation_links", []string{"id"}, []string{"id", "created_at", "from_observation_id", "to_observation_id", "relation_type", "note"})
	addRestoreTable("retrieval_result_observations", []string{"retrieval_result_id", "observation_id"}, []string{"retrieval_result_id", "observation_id", "selection_note", "created_at"})
	addRestoreTable("memory_usefulness_events", []string{"id"}, []string{"id", "created_at", "observation_id", "retrieval_result_id", "retrieval_run_id", "packet_id", "job_id", "signal", "weight", "note"})
	addRestoreTable("packet_alignment_notes", []string{"id"}, []string{"id", "packet_id", "observation_id", "retrieval_result_id", "note", "created_at"})
	addRestoreTable("memory_repair_runs", []string{"id"}, []string{"id", "created_at", "started_at", "completed_at", "dossier_id", "mode", "max_age_days", "candidates", "repaired", "skipped", "failed", "note"})
	addRestoreTable("memory_repair_items", []string{"id"}, []string{"id", "repair_run_id", "observation_id", "status", "issue", "before_json", "after_json", "note", "created_at"})
	addRestoreTable("model_manifests", []string{"id"}, []string{"id", "schema_version", "display_name", "family", "format", "backend", "model_path", "sha256", "size_bytes", "quantization", "context_length", "capabilities_json", "default_runtime_json", "license_json", "metadata_json", "discovered_at", "updated_at"})
	addRestoreTable("model_registry_status", []string{"model_id"}, []string{"model_id", "backend", "status", "updated_at", "last_error", "metadata_json"})
	addRestoreTable("model_runtime_loads", []string{"id"}, []string{"id", "model_id", "backend", "status", "loaded_at", "unloaded_at", "endpoint", "pid", "resource_usage_json", "metadata_json"})
	addRestoreTable("chat_threads", []string{"id"}, []string{"id", "title", "created_at", "updated_at", "dossier_id"})
	addRestoreTable("chat_messages", []string{"id"}, []string{"id", "thread_id", "role", "content", "created_at", "metadata_json"})
	addRestoreTable("canvas_boards", []string{"id"}, []string{"id", "title", "dossier_id", "created_at", "updated_at"})
	addRestoreTable("canvas_notes", []string{"id"}, []string{"id", "board_id", "title", "body", "x", "y", "width", "height", "pinned", "color", "links_json", "created_at", "updated_at"})
	addRestoreTable("tool_capability_overrides", []string{"capability_id"}, []string{"capability_id", "status", "reason", "actor", "actor_kind", "previous_status", "risk_class", "transition_risk", "approval_request_id", "correlation_id", "trace_id", "updated_at"})
	addRestoreTable("feature_flags", []string{"key"}, []string{"key", "value", "updated_at", "actor"})
	addRestoreTable("alert_rules", []string{"id"}, []string{"id", "name", "expression", "status", "silenced_until", "created_at", "updated_at"})
	addRestoreTable("scheduled_tasks", []string{"id"}, []string{"id", "kind", "payload_json", "status", "created_at", "updated_at"})
	return m
}()

func buildRestoreInsert(table string, conflictFields, fields []string, doNothing bool) insertMap {
	placeholders := make([]string, len(fields))
	for i := range placeholders {
		placeholders[i] = "?"
	}
	sqlText := fmt.Sprintf("INSERT INTO %s(%s) VALUES(%s)", table, strings.Join(fields, ","), strings.Join(placeholders, ","))
	if len(conflictFields) > 0 {
		sqlText += fmt.Sprintf(" ON CONFLICT(%s)", strings.Join(conflictFields, ","))
	}
	if doNothing {
		sqlText += " DO NOTHING"
	} else {
		assignments := make([]string, 0, len(fields))
		conflictSet := map[string]struct{}{}
		for _, field := range conflictFields {
			conflictSet[field] = struct{}{}
		}
		for _, field := range fields {
			if _, isConflict := conflictSet[field]; isConflict {
				continue
			}
			assignments = append(assignments, fmt.Sprintf("%s=excluded.%s", field, field))
		}
		if len(assignments) == 0 {
			sqlText += " DO NOTHING"
		} else {
			sqlText += " DO UPDATE SET " + strings.Join(assignments, ",")
		}
	}
	return insertMap{sql: sqlText, fields: fields}
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

func knownSections(doc BundleDoc) []string {
	out := make([]string, 0, len(doc.Entities))
	for k := range doc.Entities {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func normalizeSections(sections []string) []string {
	if len(sections) == 0 {
		return nil
	}
	out := make([]string, 0, len(sections))
	seen := make(map[string]struct{}, len(sections))
	for _, sec := range sections {
		normalized := strings.ToLower(strings.TrimSpace(sec))
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	return out
}

func orderSectionsForRestore(sections []string) []string {
	if len(sections) <= 1 {
		return sections
	}
	type orderedSection struct {
		name     string
		priority int
		index    int
	}
	ordered := make([]orderedSection, 0, len(sections))
	for i, sec := range sections {
		priority, ok := restoreSectionPriority[sec]
		if !ok {
			priority = 1000
		}
		ordered = append(ordered, orderedSection{
			name:     sec,
			priority: priority,
			index:    i,
		})
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].priority == ordered[j].priority {
			return ordered[i].index < ordered[j].index
		}
		return ordered[i].priority < ordered[j].priority
	})
	out := make([]string, 0, len(ordered))
	for _, item := range ordered {
		out = append(out, item.name)
	}
	return out
}

var restoreSectionPriority = map[string]int{
	"sources":                       1,
	"files":                         2,
	"chunks":                        3,
	"embedding_records":             4,
	"dossiers":                      8,
	"task_packets":                  10,
	"project_context_records":       11,
	"permission_profiles":           15,
	"approval_presets":              16,
	"dossier_profiles":              17,
	"execution_strategies":          18,
	"automation_rules":              19,
	"action_lanes":                  20,
	"jobs":                          21,
	"events":                        25,
	"dossier_sources":               26,
	"dossier_jobs":                  27,
	"dossier_packets":               28,
	"dossier_briefs":                29,
	"approval_requests":             30,
	"approval_decisions":            31,
	"job_status_history":            32,
	"job_events":                    33,
	"artifacts":                     34,
	"evaluation_records":            36,
	"provenance_records":            40,
	"gateway_invocations":           41,
	"audit_records":                 42,
	"retrieval_runs":                43,
	"retrieval_results":             44,
	"retrieval_result_selection":    45,
	"packet_retrieval_runs":         46,
	"context_evidence":              47,
	"journal_events":                50,
	"state_items":                   55,
	"state_versions":                56,
	"memory_observations":           57,
	"memory_observation_links":      58,
	"retrieval_result_observations": 59,
	"memory_notes":                  60,
	"semantic_links":                61,
	"open_loops":                    62,
	"artifact_refs":                 63,
	"derived_models":                64,
	"contradiction_records":         65,
	"supersession_records":          66,
	"context_packet_snapshots":      67,
	"dream_reports":                 68,
	"semantic_idempotency_keys":     69,
	"autonomy_settings":             70,
	"memory_usefulness_events":      71,
	"packet_alignment_notes":        72,
	"memory_repair_runs":            73,
	"memory_repair_items":           74,
	"model_manifests":               80,
	"model_registry_status":         81,
	"model_runtime_loads":           82,
	"chat_threads":                  90,
	"chat_messages":                 91,
	"canvas_boards":                 92,
	"canvas_notes":                  93,
	"tool_capability_overrides":     94,
	"feature_flags":                 95,
	"alert_rules":                   96,
	"scheduled_tasks":               97,
}

var restoreExportOnlyReasons = map[string]string{
	"memory_vsa_pointers":          "restore export-only by policy: VSA pointers are derived from observation lineage and fingerprint state",
	"memory_vsa_role_bindings":     "restore export-only by policy: VSA role bindings are derived from observation lineage and role reconciliation",
	"memory_vsa_associations":      "restore export-only by policy: VSA associations are derived graph edges and must be recomputed",
	"retrieval_result_vsa_signals": "restore export-only by policy: VSA signals are derived from retrieval runs/results and must be recomputed",
	"memory_vsa_reindex_runs":      "restore export-only by policy: reindex runs are operational maintenance history tied to live memory state",
	"memory_vsa_reindex_items":     "restore export-only by policy: reindex items are operational maintenance history tied to live memory state",
}

func restoreExportOnlyReason(section string) (string, bool) {
	reason, ok := restoreExportOnlyReasons[section]
	return reason, ok
}

func restoreUnsupportedReason(section string) string {
	if reason, ok := restoreExportOnlyReason(section); ok {
		return reason
	}
	return "restore mapping not implemented"
}

func restoreSectionWarning(section string) string {
	switch section {
	case "artifacts":
		return "restore limitation: artifact rows are restored, but artifact file bytes are not imported or rollback-managed"
	default:
		return ""
	}
}

func restoreSectionNonDBSideEffect(section string) string {
	return restoreSectionWarning(section)
}

func appendUniqueString(items []string, value string) []string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return items
	}
	for _, item := range items {
		if item == trimmed {
			return items
		}
	}
	return append(items, trimmed)
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
