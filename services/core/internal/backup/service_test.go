package backup

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forge/projectforge/services/core/internal/store"
)

func TestFullBackupExportRestoreParityForHighValueSections(t *testing.T) {
	ctx := context.Background()
	sourceDir := t.TempDir()

	source, err := store.Open(sourceDir)
	if err != nil {
		t.Fatalf("open source store: %v", err)
	}
	t.Cleanup(func() { _ = source.Close() })

	seedHighValueBackupFixture(t, ctx, source.DB)

	srcSvc := New(source.DB, sourceDir)
	bundle, err := srcSvc.CreateBundle(ctx, CreateBundleRequest{
		Kind:  "full_backup",
		Label: "high-value",
	})
	if err != nil {
		t.Fatalf("create full backup bundle: %v", err)
	}

	raw, err := os.ReadFile(bundle.FilePath)
	if err != nil {
		t.Fatalf("read bundle file: %v", err)
	}
	var doc BundleDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("decode bundle doc: %v", err)
	}

	requiredSections := []string{
		"dossiers", "dossier_profiles",
		"approval_requests", "approval_decisions",
		"events", "journal_events",
		"jobs", "job_status_history", "job_events",
		"artifacts", "artifact_refs",
		"memory_notes", "semantic_links", "state_items", "state_versions", "open_loops",
		"contradiction_records", "supersession_records", "derived_models",
		"context_packet_snapshots", "dream_reports", "restore_outcome_events", "semantic_idempotency_keys", "provenance_records",
		"project_context_records", "evaluation_records", "gateway_invocations", "audit_records",
		"autonomy_settings",
		"permission_profiles", "approval_presets", "execution_strategies",
		"model_manifests", "model_registry_status", "model_runtime_loads",
		"chat_threads", "chat_messages", "canvas_boards", "canvas_notes", "tool_capability_overrides",
	}
	for _, section := range requiredSections {
		if _, ok := doc.Entities[section]; !ok {
			t.Fatalf("full_backup missing required section %q", section)
		}
		if !bundleManifestContains(doc.Manifest, section) {
			t.Fatalf("full_backup manifest missing required section %q", section)
		}
		if doc.Checksums[section] == "" {
			t.Fatalf("full_backup missing checksum for section %q", section)
		}
	}

	expectedExportCounts := map[string]int{
		"dossiers":                  1,
		"dossier_profiles":          1,
		"task_packets":              1,
		"project_context_records":   1,
		"jobs":                      1,
		"job_status_history":        1,
		"job_events":                1,
		"approval_requests":         1,
		"approval_decisions":        1,
		"events":                    1,
		"artifacts":                 1,
		"provenance_records":        1,
		"journal_events":            1,
		"memory_notes":              1,
		"artifact_refs":             1,
		"evaluation_records":        1,
		"gateway_invocations":       1,
		"audit_records":             1,
		"semantic_idempotency_keys": 1,
		"dream_reports":             1,
		"autonomy_settings":         1,
		"model_manifests":           1,
		"model_registry_status":     1,
		"model_runtime_loads":       1,
		"chat_threads":              1,
		"chat_messages":             1,
		"canvas_boards":             1,
		"canvas_notes":              1,
		"tool_capability_overrides": 1,
	}
	for section, want := range expectedExportCounts {
		if got := doc.EntityCounts[section]; got != want {
			t.Fatalf("export count mismatch for %s: got %d want %d", section, got, want)
		}
	}

	targetDir := t.TempDir()
	target, err := store.Open(targetDir)
	if err != nil {
		t.Fatalf("open target store: %v", err)
	}
	t.Cleanup(func() { _ = target.Close() })

	dstSvc := New(target.DB, targetDir)
	stagedBundlePath := stageBundleForRestore(t, dstSvc, bundle.FilePath)
	restore, err := dstSvc.RestoreBundle(ctx, RestoreBundleRequest{
		FilePath: stagedBundlePath,
		// Intentionally out-of-order; restore must normalize to FK-safe order.
		Sections: []string{
			"dossier_profiles", "dossiers",
			"approval_decisions", "approval_requests",
			"artifacts", "job_events", "job_status_history",
			"jobs", "task_packets",
			"journal_events", "provenance_records",
			"memory_notes", "artifact_refs",
			"events", "autonomy_settings",
			"gateway_invocations", "audit_records",
			"dream_reports", "semantic_idempotency_keys",
			"evaluation_records", "project_context_records",
			"model_runtime_loads", "model_registry_status", "model_manifests",
			"chat_messages", "chat_threads", "canvas_notes", "canvas_boards", "tool_capability_overrides",
		},
	})
	if err != nil {
		t.Fatalf("restore bundle: %v", err)
	}
	if len(restore.Errors) != 0 {
		t.Fatalf("restore returned errors: %+v", restore.Errors)
	}
	if len(restore.Unsupported) != 0 {
		t.Fatalf("restore unexpectedly reported unsupported sections: %+v", restore.Unsupported)
	}
	if len(restore.ExportOnly) != 0 {
		t.Fatalf("restore unexpectedly reported export-only sections: %+v", restore.ExportOnly)
	}
	if !restore.Atomic || !restore.Applied || restore.RolledBack {
		t.Fatalf("restore result flags mismatch: atomic=%t applied=%t rolledBack=%t", restore.Atomic, restore.Applied, restore.RolledBack)
	}
	if restore.AtomicScope != "db-supported-sections-only" {
		t.Fatalf("restore atomic scope mismatch: got %q", restore.AtomicScope)
	}
	if restore.GlobalAtomic {
		t.Fatalf("restore should not claim global atomicity for non-DB side effects")
	}
	if len(restore.Warnings) == 0 {
		t.Fatalf("expected restore warnings for non-db limitations")
	}
	if !strings.Contains(strings.Join(restore.Warnings, "\n"), "artifact file bytes are not imported") {
		t.Fatalf("expected artifact restore side-effect warning, got %+v", restore.Warnings)
	}
	if got := restore.NonDBSideEffects["artifacts"]; !strings.Contains(got, "artifact file bytes are not imported") {
		t.Fatalf("expected explicit non-db side-effect reporting for artifacts, got %q", got)
	}

	expectedRestoreCounts := map[string]int{
		"dossiers":                  1,
		"dossier_profiles":          1,
		"task_packets":              1,
		"project_context_records":   1,
		"jobs":                      1,
		"job_status_history":        1,
		"job_events":                1,
		"approval_requests":         1,
		"approval_decisions":        1,
		"events":                    1,
		"artifacts":                 1,
		"provenance_records":        1,
		"journal_events":            1,
		"memory_notes":              1,
		"artifact_refs":             1,
		"evaluation_records":        1,
		"gateway_invocations":       1,
		"audit_records":             1,
		"semantic_idempotency_keys": 1,
		"dream_reports":             1,
		"autonomy_settings":         1,
		"model_manifests":           1,
		"model_registry_status":     1,
		"model_runtime_loads":       1,
		"chat_threads":              1,
		"chat_messages":             1,
		"canvas_boards":             1,
		"canvas_notes":              1,
		"tool_capability_overrides": 1,
	}
	for section, want := range expectedRestoreCounts {
		if got := restore.Imported[section]; got != want {
			t.Fatalf("restore count mismatch for %s: got %d want %d", section, got, want)
		}
	}

	var jobStatus string
	if err := target.DB.QueryRowContext(ctx, `SELECT status FROM jobs WHERE id = ?`, "job-1").Scan(&jobStatus); err != nil {
		t.Fatalf("query restored job: %v", err)
	}
	if jobStatus != "running" {
		t.Fatalf("restored job status mismatch: got %q", jobStatus)
	}

	var decision string
	if err := target.DB.QueryRowContext(ctx, `SELECT decision FROM approval_decisions WHERE id = ?`, 302).Scan(&decision); err != nil {
		t.Fatalf("query restored approval decision: %v", err)
	}
	if decision != "approved" {
		t.Fatalf("restored approval decision mismatch: got %q", decision)
	}
	var expiresAt, expiredAt int64
	if err := target.DB.QueryRowContext(ctx, `SELECT expires_at, expired_at FROM approval_requests WHERE id = ?`, 301).Scan(&expiresAt, &expiredAt); err != nil {
		t.Fatalf("query restored approval expiry: %v", err)
	}
	if expiresAt != 1200+3600000 || expiredAt != 0 {
		t.Fatalf("restored approval expiry mismatch: expires=%d expired=%d", expiresAt, expiredAt)
	}

	var artifactPath string
	if err := target.DB.QueryRowContext(ctx, `SELECT file_path FROM artifacts WHERE id = ?`, 501).Scan(&artifactPath); err != nil {
		t.Fatalf("query restored artifact: %v", err)
	}
	if artifactPath != "/tmp/result.txt" {
		t.Fatalf("restored artifact path mismatch: got %q", artifactPath)
	}

	var provTrace, provCorr, provSyscall, provAudit string
	if err := target.DB.QueryRowContext(ctx, `
SELECT trace_id, correlation_id, syscall_id, audit_id
FROM provenance_records WHERE id = ?`, "prov-1").Scan(&provTrace, &provCorr, &provSyscall, &provAudit); err != nil {
		t.Fatalf("query restored provenance: %v", err)
	}
	if provTrace != "trace-123" || provCorr != "corr-123" || provSyscall != "sys-123" || provAudit != "audit-123" {
		t.Fatalf("restored provenance trace fields mismatch: trace=%q corr=%q syscall=%q audit=%q", provTrace, provCorr, provSyscall, provAudit)
	}

	var journalTrace, journalCorr, journalProv string
	if err := target.DB.QueryRowContext(ctx, `
SELECT trace_id, correlation_id, provenance_id
FROM journal_events WHERE id = ?`, "journal-1").Scan(&journalTrace, &journalCorr, &journalProv); err != nil {
		t.Fatalf("query restored journal event: %v", err)
	}
	if journalTrace != "trace-123" || journalCorr != "corr-123" || journalProv != "prov-1" {
		t.Fatalf("restored journal trace/provenance mismatch: trace=%q corr=%q provenance=%q", journalTrace, journalCorr, journalProv)
	}

	var noteTrace, noteCorr, noteProv string
	if err := target.DB.QueryRowContext(ctx, `
SELECT trace_id, correlation_id, provenance_id
FROM memory_notes WHERE id = ?`, "note-1").Scan(&noteTrace, &noteCorr, &noteProv); err != nil {
		t.Fatalf("query restored memory note: %v", err)
	}
	if noteTrace != "trace-123" || noteCorr != "corr-123" || noteProv != "prov-1" {
		t.Fatalf("restored memory note trace/provenance mismatch: trace=%q corr=%q provenance=%q", noteTrace, noteCorr, noteProv)
	}

	var dossierName string
	if err := target.DB.QueryRowContext(ctx, `SELECT name FROM dossiers WHERE id = ?`, 701).Scan(&dossierName); err != nil {
		t.Fatalf("query restored dossier: %v", err)
	}
	if dossierName != "Core Systems" {
		t.Fatalf("restored dossier name mismatch: got %q", dossierName)
	}

	var ctxVersion int
	var ctxSourcePath string
	if err := target.DB.QueryRowContext(ctx, `
SELECT context_version, source_path
FROM project_context_records WHERE id = ?`, 601).Scan(&ctxVersion, &ctxSourcePath); err != nil {
		t.Fatalf("query restored project context: %v", err)
	}
	if ctxVersion != 1 || ctxSourcePath != "/workspace/FORGE_CONTEXT.md" {
		t.Fatalf("restored project context mismatch: version=%d source=%q", ctxVersion, ctxSourcePath)
	}

	var evalJob string
	var evalDossier sql.NullInt64
	var evalNotes string
	if err := target.DB.QueryRowContext(ctx, `
SELECT job_id, dossier_id, notes
FROM evaluation_records WHERE id = ?`, 602).Scan(&evalJob, &evalDossier, &evalNotes); err != nil {
		t.Fatalf("query restored evaluation: %v", err)
	}
	if evalJob != "job-1" || !evalDossier.Valid || evalDossier.Int64 != 701 || evalNotes != "reviewed and routed" {
		t.Fatalf("restored evaluation mismatch: job=%q dossier=%v notes=%q", evalJob, evalDossier, evalNotes)
	}

	var gatewayStatus, gatewayTool, gatewayAction string
	var gatewayApproval sql.NullInt64
	if err := target.DB.QueryRowContext(ctx, `
SELECT status, tool_id, action, approval_request_id
FROM gateway_invocations WHERE id = ?`, 603).Scan(&gatewayStatus, &gatewayTool, &gatewayAction, &gatewayApproval); err != nil {
		t.Fatalf("query restored gateway invocation: %v", err)
	}
	if gatewayStatus != "completed" || gatewayTool != "tool.backup" || gatewayAction != "restore" || !gatewayApproval.Valid || gatewayApproval.Int64 != 301 {
		t.Fatalf("restored gateway invocation mismatch: status=%q tool=%q action=%q approval=%v", gatewayStatus, gatewayTool, gatewayAction, gatewayApproval)
	}

	var auditCategory, auditAction string
	var auditGateway sql.NullInt64
	if err := target.DB.QueryRowContext(ctx, `
SELECT category, action, gateway_invocation_id
FROM audit_records WHERE id = ?`, 604).Scan(&auditCategory, &auditAction, &auditGateway); err != nil {
		t.Fatalf("query restored audit record: %v", err)
	}
	if auditCategory != "gateway" || auditAction != "tool.completed" || !auditGateway.Valid || auditGateway.Int64 != 603 {
		t.Fatalf("restored audit record mismatch: category=%q action=%q gateway=%v", auditCategory, auditAction, auditGateway)
	}

	var routingNotes string
	if err := target.DB.QueryRowContext(ctx, `SELECT routing_notes FROM dossier_profiles WHERE dossier_id = ?`, 701).Scan(&routingNotes); err != nil {
		t.Fatalf("query restored dossier profile: %v", err)
	}
	if routingNotes != "prefer safety-first execution" {
		t.Fatalf("restored dossier profile routing notes mismatch: got %q", routingNotes)
	}

	var autonomyValue string
	if err := target.DB.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = ?`, "autonomy_repo.intent.intent-1").Scan(&autonomyValue); err != nil {
		t.Fatalf("query restored autonomy setting: %v", err)
	}
	if autonomyValue != `{"id":"intent-1","status":"proposed"}` {
		t.Fatalf("autonomy setting mismatch: got %q", autonomyValue)
	}

	var idempotencyAction string
	if err := target.DB.QueryRowContext(ctx, `SELECT action FROM semantic_idempotency_keys WHERE idempotency_key = ?`, "idem-1").Scan(&idempotencyAction); err != nil {
		t.Fatalf("query restored semantic idempotency key: %v", err)
	}
	if idempotencyAction != "CREATE_NOTE" {
		t.Fatalf("semantic idempotency action mismatch: got %q", idempotencyAction)
	}

	var dreamMode, dreamCommittedBy string
	var dreamDryRun int
	if err := target.DB.QueryRowContext(ctx, `SELECT mode, dry_run, committed_by FROM dream_reports WHERE id = ?`, "dream-report-1").Scan(&dreamMode, &dreamDryRun, &dreamCommittedBy); err != nil {
		t.Fatalf("query restored dream report: %v", err)
	}
	if dreamMode != "nap" || dreamDryRun != 1 || dreamCommittedBy != "" {
		t.Fatalf("restored dream report authority mismatch: mode=%q dryRun=%d committedBy=%q", dreamMode, dreamDryRun, dreamCommittedBy)
	}

	var modelStatus string
	if err := target.DB.QueryRowContext(ctx, `SELECT status FROM model_registry_status WHERE model_id = ?`, "local-chat").Scan(&modelStatus); err != nil {
		t.Fatalf("query restored model registry status: %v", err)
	}
	if modelStatus != "available" {
		t.Fatalf("model registry status mismatch: got %q", modelStatus)
	}

	var chatContent string
	if err := target.DB.QueryRowContext(ctx, `SELECT content FROM chat_messages WHERE id = ?`, 802).Scan(&chatContent); err != nil {
		t.Fatalf("query restored chat message: %v", err)
	}
	if chatContent != "hello restored chat" {
		t.Fatalf("chat message mismatch: got %q", chatContent)
	}

	var capabilityStatus string
	if err := target.DB.QueryRowContext(ctx, `SELECT status FROM tool_capability_overrides WHERE capability_id = ?`, "filesystem.write_file").Scan(&capabilityStatus); err != nil {
		t.Fatalf("query restored capability override: %v", err)
	}
	if capabilityStatus != "approval_only" {
		t.Fatalf("capability override status mismatch: got %q", capabilityStatus)
	}

	if verificationState := restore.Verification["schema"]; verificationState != "passed" {
		t.Fatalf("expected restore verification schema passed, got %#v", verificationState)
	}

	var ignored string
	err = target.DB.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = ?`, "ui.theme").Scan(&ignored)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected ui.theme to remain absent after restore, got err=%v value=%q", err, ignored)
	}

	secondRestore, err := dstSvc.RestoreBundle(ctx, RestoreBundleRequest{
		FilePath: stagedBundlePath,
		Sections: []string{"audit_records", "semantic_idempotency_keys"},
	})
	if err != nil {
		t.Fatalf("second restore bundle: %v", err)
	}
	if len(secondRestore.Errors) != 0 || secondRestore.RolledBack {
		t.Fatalf("second restore should be idempotent for immutable sections: result=%+v", secondRestore)
	}
}

func TestRestoreDetectsMissingDreamReportsTable(t *testing.T) {
	ctx := context.Background()
	sourceDir := t.TempDir()
	source, err := store.Open(sourceDir)
	if err != nil {
		t.Fatalf("open source store: %v", err)
	}
	t.Cleanup(func() { _ = source.Close() })
	seedHighValueBackupFixture(t, ctx, source.DB)
	srcSvc := New(source.DB, sourceDir)
	bundle, err := srcSvc.CreateBundle(ctx, CreateBundleRequest{Kind: "full_backup", Label: "dream-schema"})
	if err != nil {
		t.Fatalf("create bundle: %v", err)
	}

	targetDir := t.TempDir()
	target, err := store.Open(targetDir)
	if err != nil {
		t.Fatalf("open target store: %v", err)
	}
	t.Cleanup(func() { _ = target.Close() })
	mustExec(t, ctx, target.DB, `DROP TABLE dream_reports`)

	targetSvc := New(target.DB, targetDir)
	result, err := targetSvc.RestoreBundle(ctx, RestoreBundleRequest{
		FilePath: stageBundleForRestore(t, targetSvc, bundle.FilePath),
		Sections: []string{"dream_reports"},
	})
	if err != nil {
		t.Fatalf("restore bundle: %v", err)
	}
	if result.Verification["schema"] != "failed" || len(result.Errors) == 0 {
		t.Fatalf("expected dream report schema failure, got verification=%+v errors=%+v", result.Verification, result.Errors)
	}
	if result.Applied || result.Atomic || result.RolledBack {
		t.Fatalf("schema validation should fail before transaction, got applied=%t atomic=%t rolledBack=%t", result.Applied, result.Atomic, result.RolledBack)
	}
}

func bundleManifestContains(manifest []SectionManifest, section string) bool {
	for _, entry := range manifest {
		if entry.Name == section {
			return true
		}
	}
	return false
}

func TestRestoreBundleContextPacketSnapshotColumns(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()

	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	doc := BundleDoc{
		Schema:      BundleSchemaVersion,
		GeneratedAt: 1234,
		Kind:        "full_backup",
		Label:       "snapshot-columns",
		EntityCounts: map[string]int{
			"context_packet_snapshots": 2,
		},
		Entities: map[string][]any{
			"context_packet_snapshots": {
				map[string]any{
					"id":                       "snap-old",
					"query":                    "restore the old-style snapshot",
					"workspace_id":             "workspace-1",
					"lane_id":                  "lane-a",
					"selected_paths_json":      `["/workspace/a.go"]`,
					"included_state_json":      `[]`,
					"included_open_loops_json": `[]`,
					"included_notes_json":      `[]`,
					"included_links_json":      `[]`,
					"included_models_json":     `[]`,
					"included_artifacts_json":  `[]`,
					"included_events_json":     `[]`,
					"budget_json":              `{"tokens":1000}`,
					"inclusion_reasons_json":   `{"paths":"root"}`,
					"created_at":               2000,
					"correlation_id":           "corr-old",
					"trace_id":                 "trace-old",
					"syscall_id":               "sys-old",
					"metadata_json":            `{"source":"old"}`,
					"proposed_by":              "worker-1",
					"committed_by":             "forge_kernel",
					"audit_id":                 "audit-old",
				},
				map[string]any{
					"id":                       "snap-new",
					"query":                    "restore the new-style snapshot",
					"workspace_id":             "workspace-1",
					"lane_id":                  "lane-a",
					"snapshot_kind":            "restore",
					"snapshot_fingerprint":     "fp-123",
					"parent_snapshot_id":       "snap-old",
					"selected_paths_json":      `["/workspace/b.go"]`,
					"included_state_json":      `["state-1"]`,
					"included_open_loops_json": `["loop-1"]`,
					"included_notes_json":      `["note-1"]`,
					"included_links_json":      `["link-1"]`,
					"included_models_json":     `["model-1"]`,
					"included_artifacts_json":  `["artifact-1"]`,
					"included_events_json":     `["event-1"]`,
					"header_json":              `{"title":"restore snapshot"}`,
					"graph_json":               `{"nodes":["root"]}`,
					"delta_json":               `{"added":["task_packets"]}`,
					"restore_scores_json":      `{"coverage":0.9}`,
					"render_artifact_ref_id":   "aref-render-1",
					"resume_hints_json":        `{"next":"resume at packet"}`,
					"budget_json":              `{"tokens":250}`,
					"inclusion_reasons_json":   `{"graph":"needed"}`,
					"created_at":               2001,
					"correlation_id":           "corr-new",
					"trace_id":                 "trace-new",
					"syscall_id":               "sys-new",
					"metadata_json":            `{"source":"new"}`,
					"proposed_by":              "worker-2",
					"committed_by":             "forge_kernel",
					"audit_id":                 "audit-new",
				},
			},
		},
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal bundle doc: %v", err)
	}
	svc := New(st.DB, dataDir)
	filePath := restoreStagingPath(t, svc, "snapshots.json")
	if err := os.WriteFile(filePath, raw, 0o644); err != nil {
		t.Fatalf("write bundle doc: %v", err)
	}

	result, err := svc.RestoreBundle(ctx, RestoreBundleRequest{
		FilePath: filePath,
		Sections: []string{"context_packet_snapshots"},
	})
	if err != nil {
		t.Fatalf("restore bundle: %v", err)
	}
	if len(result.Errors) != 0 {
		t.Fatalf("restore returned errors: %+v", result.Errors)
	}
	if got := result.Imported["context_packet_snapshots"]; got != 2 {
		t.Fatalf("restore count mismatch: got %d want 2", got)
	}

	var oldKind, oldFingerprint, oldParent, oldHeader, oldGraph, oldDelta, oldScores, oldRender, oldHints string
	if err := st.DB.QueryRowContext(ctx, `
SELECT snapshot_kind, snapshot_fingerprint, parent_snapshot_id, header_json, graph_json, delta_json,
       restore_scores_json, render_artifact_ref_id, resume_hints_json
FROM context_packet_snapshots WHERE id = ?`, "snap-old").Scan(
		&oldKind, &oldFingerprint, &oldParent, &oldHeader, &oldGraph, &oldDelta, &oldScores, &oldRender, &oldHints,
	); err != nil {
		t.Fatalf("query restored old snapshot: %v", err)
	}
	if oldKind != "" || oldFingerprint != "" || oldParent != "" || oldHeader != "{}" || oldGraph != "{}" || oldDelta != "{}" || oldScores != "{}" || oldRender != "" || oldHints != "{}" {
		t.Fatalf("old snapshot defaults mismatch: kind=%q fingerprint=%q parent=%q header=%q graph=%q delta=%q scores=%q render=%q hints=%q",
			oldKind, oldFingerprint, oldParent, oldHeader, oldGraph, oldDelta, oldScores, oldRender, oldHints)
	}

	var newKind, newFingerprint, newParent, newHeader, newGraph, newDelta, newScores, newRender, newHints string
	if err := st.DB.QueryRowContext(ctx, `
SELECT snapshot_kind, snapshot_fingerprint, parent_snapshot_id, header_json, graph_json, delta_json,
       restore_scores_json, render_artifact_ref_id, resume_hints_json
FROM context_packet_snapshots WHERE id = ?`, "snap-new").Scan(
		&newKind, &newFingerprint, &newParent, &newHeader, &newGraph, &newDelta, &newScores, &newRender, &newHints,
	); err != nil {
		t.Fatalf("query restored new snapshot: %v", err)
	}
	if newKind != "restore" || newFingerprint != "fp-123" || newParent != "snap-old" || newHeader != `{"title":"restore snapshot"}` || newGraph != `{"nodes":["root"]}` || newDelta != `{"added":["task_packets"]}` || newScores != `{"coverage":0.9}` || newRender != "aref-render-1" || newHints != `{"next":"resume at packet"}` {
		t.Fatalf("new snapshot fields mismatch: kind=%q fingerprint=%q parent=%q header=%q graph=%q delta=%q scores=%q render=%q hints=%q",
			newKind, newFingerprint, newParent, newHeader, newGraph, newDelta, newScores, newRender, newHints)
	}
}

func TestStoreOpenMigratesContextPacketSnapshotColumns(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()

	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}

	mustExec(t, ctx, st.DB, `DROP TABLE context_packet_snapshots`)
	mustExec(t, ctx, st.DB, `
CREATE TABLE context_packet_snapshots (
  id TEXT PRIMARY KEY,
  query TEXT NOT NULL,
  workspace_id TEXT NOT NULL,
  lane_id TEXT NOT NULL DEFAULT '',
  selected_paths_json TEXT NOT NULL DEFAULT '[]',
  included_state_json TEXT NOT NULL DEFAULT '[]',
  included_open_loops_json TEXT NOT NULL DEFAULT '[]',
  included_notes_json TEXT NOT NULL DEFAULT '[]',
  included_links_json TEXT NOT NULL DEFAULT '[]',
  included_models_json TEXT NOT NULL DEFAULT '[]',
  included_artifacts_json TEXT NOT NULL DEFAULT '[]',
  included_events_json TEXT NOT NULL DEFAULT '[]',
  budget_json TEXT NOT NULL DEFAULT '{}',
  inclusion_reasons_json TEXT NOT NULL DEFAULT '{}',
  created_at INTEGER NOT NULL,
  correlation_id TEXT NOT NULL DEFAULT '',
  trace_id TEXT NOT NULL DEFAULT '',
  syscall_id TEXT NOT NULL DEFAULT '',
  metadata_json TEXT NOT NULL DEFAULT '{}',
  proposed_by TEXT NOT NULL DEFAULT '',
  committed_by TEXT NOT NULL DEFAULT 'forge_kernel',
  audit_id TEXT NOT NULL DEFAULT ''
)`)
	mustExec(t, ctx, st.DB, `
INSERT INTO context_packet_snapshots(
  id, query, workspace_id, lane_id, selected_paths_json, included_state_json, included_open_loops_json,
  included_notes_json, included_links_json, included_models_json, included_artifacts_json, included_events_json,
  budget_json, inclusion_reasons_json, created_at, correlation_id, trace_id, syscall_id, metadata_json,
  proposed_by, committed_by, audit_id
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		"legacy-snap", "legacy query", "workspace-legacy", "lane-legacy", `["/workspace/legacy.go"]`, `[]`, `[]`,
		`[]`, `[]`, `[]`, `[]`, `[]`, `{"tokens":10}`, `{"paths":"legacy"}`, 1900, "corr-legacy", "trace-legacy",
		"sys-legacy", `{"source":"legacy"}`, "worker-legacy", "forge_kernel", "audit-legacy",
	)

	if err := st.Close(); err != nil {
		t.Fatalf("close pre-migration store: %v", err)
	}

	st, err = store.Open(dataDir)
	if err != nil {
		t.Fatalf("reopen migrated store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	rows, err := st.DB.QueryContext(ctx, `PRAGMA table_info(context_packet_snapshots)`)
	if err != nil {
		t.Fatalf("inspect snapshot columns: %v", err)
	}
	defer rows.Close()

	columns := map[string]struct{}{}
	for rows.Next() {
		var cid, notNull, pk int
		var name, colType string
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &colType, &notNull, &defaultValue, &pk); err != nil {
			t.Fatalf("scan table info: %v", err)
		}
		columns[name] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate table info: %v", err)
	}

	for _, name := range []string{
		"snapshot_kind", "snapshot_fingerprint", "parent_snapshot_id", "header_json", "graph_json",
		"delta_json", "restore_scores_json", "render_artifact_ref_id", "resume_hints_json",
	} {
		if _, ok := columns[name]; !ok {
			t.Fatalf("migrated table missing column %q", name)
		}
	}

	var snapshotKind, snapshotFingerprint, parentSnapshotID, headerJSON, graphJSON, deltaJSON, restoreScoresJSON, renderArtifactRefID, resumeHintsJSON string
	if err := st.DB.QueryRowContext(ctx, `
SELECT snapshot_kind, snapshot_fingerprint, parent_snapshot_id, header_json, graph_json, delta_json,
       restore_scores_json, render_artifact_ref_id, resume_hints_json
FROM context_packet_snapshots WHERE id = ?`, "legacy-snap").Scan(
		&snapshotKind, &snapshotFingerprint, &parentSnapshotID, &headerJSON, &graphJSON, &deltaJSON, &restoreScoresJSON, &renderArtifactRefID, &resumeHintsJSON,
	); err != nil {
		t.Fatalf("query migrated snapshot row: %v", err)
	}
	if snapshotKind != "" || snapshotFingerprint != "" || parentSnapshotID != "" || headerJSON != "{}" || graphJSON != "{}" || deltaJSON != "{}" || restoreScoresJSON != "{}" || renderArtifactRefID != "" || resumeHintsJSON != "{}" {
		t.Fatalf("migrated defaults mismatch: kind=%q fingerprint=%q parent=%q header=%q graph=%q delta=%q scores=%q render=%q hints=%q",
			snapshotKind, snapshotFingerprint, parentSnapshotID, headerJSON, graphJSON, deltaJSON, restoreScoresJSON, renderArtifactRefID, resumeHintsJSON)
	}
}

func TestRestoreBundleExplicitlyReportsUnsupportedSections(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()

	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	doc := BundleDoc{
		Schema:      BundleSchemaVersion,
		GeneratedAt: 1234,
		Kind:        "full_backup",
		Label:       "unsupported-sections",
		EntityCounts: map[string]int{
			"memory_vsa_pointers":          1,
			"retrieval_result_vsa_signals": 1,
		},
		Entities: map[string][]any{
			"memory_vsa_pointers": {
				map[string]any{"id": 1},
			},
			"retrieval_result_vsa_signals": {
				map[string]any{"id": 2},
			},
		},
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal bundle doc: %v", err)
	}
	svc := New(st.DB, dataDir)
	filePath := restoreStagingPath(t, svc, "unsupported.json")
	if err := os.WriteFile(filePath, raw, 0o644); err != nil {
		t.Fatalf("write bundle doc: %v", err)
	}

	result, err := svc.RestoreBundle(ctx, RestoreBundleRequest{
		FilePath: filePath,
		Sections: []string{"memory_vsa_pointers", "retrieval_result_vsa_signals", "missing_section"},
	})
	if err != nil {
		t.Fatalf("restore bundle: %v", err)
	}
	if len(result.Errors) != 0 {
		t.Fatalf("expected no hard restore errors, got %+v", result.Errors)
	}
	if result.Atomic || result.Applied || result.RolledBack {
		t.Fatalf("unexpected restore flags for unsupported-only restore: atomic=%t applied=%t rolledBack=%t", result.Atomic, result.Applied, result.RolledBack)
	}
	if result.AtomicScope != "db-supported-sections-only" {
		t.Fatalf("restore atomic scope mismatch: got %q", result.AtomicScope)
	}
	if result.GlobalAtomic {
		t.Fatalf("unsupported-only restore should not claim global atomicity")
	}
	if len(result.NonDBSideEffects) != 0 {
		t.Fatalf("unsupported-only restore should not report non-db side effects, got %+v", result.NonDBSideEffects)
	}
	if !strings.Contains(strings.Join(result.Warnings, "\n"), "export-only") {
		t.Fatalf("expected export-only policy warning, got %+v", result.Warnings)
	}

	if got := result.Unsupported["memory_vsa_pointers"]; !strings.Contains(got, "observation lineage") {
		t.Fatalf("memory_vsa_pointers should be explicitly flagged, got %q", got)
	}
	if got := result.Unsupported["retrieval_result_vsa_signals"]; !strings.Contains(got, "retrieval runs/results") {
		t.Fatalf("retrieval_result_vsa_signals should be explicitly flagged, got %q", got)
	}
	if got := result.Unsupported["missing_section"]; !strings.Contains(got, "not found") {
		t.Fatalf("missing_section should be explicitly flagged, got %q", got)
	}
	if got := result.ExportOnly["memory_vsa_pointers"]; !strings.Contains(got, "export-only by policy") {
		t.Fatalf("expected memory_vsa_pointers export-only policy reason, got %q", got)
	}
	if got := result.ExportOnly["retrieval_result_vsa_signals"]; !strings.Contains(got, "export-only by policy") {
		t.Fatalf("expected retrieval_result_vsa_signals export-only policy reason, got %q", got)
	}
	if _, exists := result.ExportOnly["missing_section"]; exists {
		t.Fatalf("missing section must not be marked export-only")
	}

	if got := result.Skipped["memory_vsa_pointers"]; got != 1 {
		t.Fatalf("expected memory_vsa_pointers to report skipped row count, got %d", got)
	}
	if got := result.Skipped["retrieval_result_vsa_signals"]; got != 1 {
		t.Fatalf("expected retrieval_result_vsa_signals to report skipped row count, got %d", got)
	}
	if got := result.Skipped["missing_section"]; got != 0 {
		t.Fatalf("expected missing section to report 0 skipped rows, got %d", got)
	}
}

func TestRestoreBundleIntegrityDetectsMissingCriticalTable(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()

	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	mustExec(t, ctx, st.DB, `DROP TABLE chat_messages`)

	doc := BundleDoc{
		Schema:      BundleSchemaVersion,
		GeneratedAt: 1234,
		Kind:        "full_backup",
		Label:       "missing-critical-table",
		EntityCounts: map[string]int{
			"chat_messages": 1,
		},
		Entities: map[string][]any{
			"chat_messages": {
				map[string]any{
					"id":            1,
					"thread_id":     1,
					"role":          "user",
					"content":       "must not import",
					"created_at":    100,
					"metadata_json": `{}`,
				},
			},
		},
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal bundle doc: %v", err)
	}
	svc := New(st.DB, dataDir)
	filePath := restoreStagingPath(t, svc, "missing-critical-table.json")
	if err := os.WriteFile(filePath, raw, 0o644); err != nil {
		t.Fatalf("write bundle doc: %v", err)
	}

	result, err := svc.RestoreBundle(ctx, RestoreBundleRequest{FilePath: filePath})
	if err != nil {
		t.Fatalf("restore bundle: %v", err)
	}
	if len(result.Errors) == 0 {
		t.Fatalf("expected schema integrity error")
	}
	if result.Applied || result.Atomic || result.RolledBack {
		t.Fatalf("schema validation should fail before transaction, got applied=%t atomic=%t rolledBack=%t", result.Applied, result.Atomic, result.RolledBack)
	}
	if got := strings.Join(result.Errors, "\n"); !strings.Contains(got, `critical table "chat_messages" is missing`) {
		t.Fatalf("missing table error should be deterministic, got %q", got)
	}
	if result.Verification["schema"] != "failed" {
		t.Fatalf("expected verification schema failed, got %#v", result.Verification["schema"])
	}
}

func TestRestoreBundleRollsBackOnLateSectionFailure(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()

	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	doc := BundleDoc{
		Schema:      BundleSchemaVersion,
		GeneratedAt: 1234,
		Kind:        "full_backup",
		Label:       "rollback-late-failure",
		EntityCounts: map[string]int{
			"dossiers":          1,
			"approval_requests": 1,
		},
		Entities: map[string][]any{
			"dossiers": {
				map[string]any{
					"id":                      9001,
					"created_at":              100,
					"updated_at":              101,
					"name":                    "Rollback Dossier",
					"description":             "Should not persist",
					"primary_paths_json":      `[]`,
					"related_repos_json":      `[]`,
					"constraints_json":        `[]`,
					"preferred_adapters_json": `[]`,
					"important_files_json":    `[]`,
					"routing_notes":           "rollback check",
				},
			},
			"approval_requests": {
				map[string]any{
					"id":                  9101,
					"job_id":              "missing-job",
					"created_at":          200,
					"status":              "pending",
					"requested_action":    "restore",
					"risk_class":          "medium",
					"requested_adapter":   "adapter.x",
					"write_intent":        0,
					"scope_snapshot_json": `{"paths":[]}`,
					"task_packet_id":      11,
					"request_summary":     "expected to fail",
				},
			},
		},
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal bundle doc: %v", err)
	}
	svc := New(st.DB, dataDir)
	filePath := restoreStagingPath(t, svc, "rollback.json")
	if err := os.WriteFile(filePath, raw, 0o644); err != nil {
		t.Fatalf("write bundle doc: %v", err)
	}

	result, err := svc.RestoreBundle(ctx, RestoreBundleRequest{
		FilePath: filePath,
		Sections: []string{"approval_requests", "dossiers"},
	})
	if err != nil {
		t.Fatalf("restore bundle: %v", err)
	}
	if !result.Atomic || result.Applied || !result.RolledBack {
		t.Fatalf("unexpected restore flags after rollback: atomic=%t applied=%t rolledBack=%t", result.Atomic, result.Applied, result.RolledBack)
	}
	if result.AtomicScope != "db-supported-sections-only" {
		t.Fatalf("restore atomic scope mismatch: got %q", result.AtomicScope)
	}
	if result.GlobalAtomic {
		t.Fatalf("rollback case should not claim global atomicity")
	}
	if len(result.Errors) == 0 {
		t.Fatalf("expected restore errors after late section failure")
	}
	if len(result.Imported) != 0 {
		t.Fatalf("expected no imported counts to be committed after rollback, got %+v", result.Imported)
	}

	var count int
	if err := st.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM dossiers WHERE id = ?`, 9001).Scan(&count); err != nil {
		t.Fatalf("query rolled back dossier: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected rolled back dossier to be absent, got count=%d", count)
	}
	if err := st.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM approval_requests WHERE id = ?`, 9101).Scan(&count); err != nil {
		t.Fatalf("query rolled back approval request: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected rolled back approval request to be absent, got count=%d", count)
	}
}

func TestRestoreBundleRejectsFilesOutsideBundleDirs(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	outside := filepath.Join(t.TempDir(), "outside.json")
	if err := os.WriteFile(outside, []byte(`{"schema":1}`), 0o644); err != nil {
		t.Fatalf("write outside bundle: %v", err)
	}

	_, err = New(st.DB, dataDir).RestoreBundle(ctx, RestoreBundleRequest{FilePath: outside, DryRun: true})
	if err == nil {
		t.Fatalf("expected outside restore path to be rejected")
	}
	if !strings.Contains(err.Error(), "backup or export directory") {
		t.Fatalf("expected bundle dir rejection, got %v", err)
	}
}

func TestRestoreBundleRejectsSymlinkEscapingBundleDirs(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	svc := New(st.DB, dataDir)
	backupDir, _ := svc.Dirs()

	outside := filepath.Join(t.TempDir(), "outside.json")
	if err := os.WriteFile(outside, []byte(`{"schema":1}`), 0o644); err != nil {
		t.Fatalf("write outside bundle: %v", err)
	}
	linkPath := filepath.Join(backupDir, "outside-link.json")
	if err := os.Symlink(outside, linkPath); err != nil {
		t.Skipf("skipping symlink escape test because symlink creation is unavailable: %v", err)
	}

	_, err = svc.RestoreBundle(ctx, RestoreBundleRequest{FilePath: linkPath, DryRun: true})
	if err == nil {
		t.Fatalf("expected symlink restore path to be rejected")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected symlink rejection, got %v", err)
	}
}

func TestRestoreBundleRejectsOversizeBundleFile(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	svc := New(st.DB, dataDir)
	backupDir, _ := svc.Dirs()
	bundlePath := filepath.Join(backupDir, "oversize.json")
	f, err := os.Create(bundlePath)
	if err != nil {
		t.Fatalf("create oversized bundle: %v", err)
	}
	if err := f.Truncate(maxRestoreBundleBytes + 1); err != nil {
		_ = f.Close()
		t.Fatalf("truncate oversized bundle: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close oversized bundle: %v", err)
	}

	_, err = svc.RestoreBundle(ctx, RestoreBundleRequest{FilePath: bundlePath, DryRun: true})
	if err == nil || !strings.Contains(err.Error(), "restore bundle too large") {
		t.Fatalf("RestoreBundle error = %v, want size error", err)
	}
}

func seedHighValueBackupFixture(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()

	mustExec(t, ctx, db, `INSERT INTO settings(key, value) VALUES(?, ?)`, "autonomy_repo.intent.intent-1", `{"id":"intent-1","status":"proposed"}`)
	mustExec(t, ctx, db, `INSERT INTO settings(key, value) VALUES(?, ?)`, "ui.theme", "dark")

	mustExec(t, ctx, db, `
INSERT INTO dossiers(
  id, created_at, updated_at, name, description, primary_paths_json, related_repos_json,
  constraints_json, preferred_adapters_json, important_files_json, routing_notes
) VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
		701, 900, 901, "Core Systems", "Core lane dossier", `["/workspace/core"]`, `["forge/projectforge"]`,
		`["no destructive writes"]`, `["adapter.x"]`, `["services/core/internal/api/server.go"]`, "focus on control-lane integrity",
	)

	mustExec(t, ctx, db, `
INSERT INTO dossier_profiles(
  dossier_id, updated_at, preferred_strategies_json, preferred_adapters_json,
  approval_preset_id, retrieval_defaults_json, high_value_files_json,
  noisy_files_json, routing_notes, automation_bindings_json
) VALUES(?,?,?,?,?,?,?,?,?,?)`,
		701, 902, `["safe-defaults"]`, `["adapter.x"]`,
		nil, `{"mode":"hybrid"}`, `["services/core/internal/backup/service.go"]`,
		`[]`, "prefer safety-first execution", `["backup_sweep"]`,
	)

	mustExec(t, ctx, db, `
INSERT INTO task_packets(
  id, packet_version, created_at, generated_at, title, user_request, objective,
  adapter_target, execution_mode, risk_class, expected_output_json, constraints_json,
  instructions, selected_paths_json, scope_snapshot_json, source_references_json,
  retrieved_context_json, project_notes, source_context_record_ids_json, request_payload_json
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		11, 1, 1000, 1001, "Packet Title", "Do thing", "Complete task",
		"adapter.x", "sync", "medium", `{"artifact":"report"}`, `[]`,
		"follow instructions", `[]`, `{"scope":"limited"}`, `[]`,
		`[]`, "notes", `[]`, `{"request":"payload"}`,
	)

	mustExec(t, ctx, db, `
INSERT INTO project_context_records(
  id, context_version, created_at, generated_at, source_path, source_hash, source_size_bytes,
  normalized_summary_json, briefing_markdown, agents_markdown, claude_markdown, cursor_markdown,
  generated_agents_path, generated_claude_path, generated_briefing_path, generated_cursor_path, notes
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		601, 1, 1800, 1801, "/workspace/FORGE_CONTEXT.md", "ctx-hash-1", 2222,
		`{"title":"Project Forge","headings":["Phase 2"]}`, "# briefing", "# agents",
		"# claude", "# cursor", "/workspace/AGENTS.md", "/workspace/CLAUDE.md",
		"/workspace/docs/FORGE_PROJECT_BRIEFING.md", "/workspace/.cursor/rules/forge-context.mdc",
		"normalized from workspace context",
	)

	mustExec(t, ctx, db, `
INSERT INTO jobs(
  id, created_at, updated_at, queued_at, started_at, completed_at, title,
  requested_action, target_adapter, initiating_source, execution_boundary,
  risk_class, status, approval_status, write_intent, cancel_requested,
  task_packet_id, result_summary, failure_info, last_failure_code, last_error, metadata_json
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		"job-1", 1100, 1101, 1102, 1103, nil, "Job Title",
		"run backup fixture", "adapter.x", "operator", "write_proposal",
		"medium", "running", "pending", 1, 0,
		11, "", "", "", "", `{"trace":"trace-123"}`,
	)

	mustExec(t, ctx, db, `INSERT INTO job_status_history(id, job_id, created_at, from_status, to_status, reason) VALUES(?,?,?,?,?,?)`,
		201, "job-1", 1104, "queued", "running", "worker started")
	mustExec(t, ctx, db, `INSERT INTO job_events(id, job_id, created_at, type, message, payload_json) VALUES(?,?,?,?,?,?)`,
		202, "job-1", 1105, "log", "started", `{"ok":true}`)

	mustExec(t, ctx, db, `
INSERT INTO evaluation_records(
  id, created_at, job_id, dossier_id, success, quality_rating, usefulness_rating, correctness_confidence,
  packet_quality_rating, adapter_suitability, retry_recommended, influence_routing, notes, scorer
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		602, 2100, "job-1", 701, 1, 5, 4, 5, 5, 5, 0, 1, "reviewed and routed", "reviewer-a")

	mustExec(t, ctx, db, `
INSERT INTO approval_requests(
  id, job_id, created_at, status, requested_action, risk_class, requested_adapter,
  write_intent, scope_snapshot_json, task_packet_id, request_summary, expires_at, expired_at
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		301, "job-1", 1200, "approved", "write files", "high", "adapter.x",
		1, `{"paths":["/tmp/result.txt"]}`, 11, "need approval", 1200+3600000, 0)
	mustExec(t, ctx, db, `INSERT INTO approval_decisions(id, request_id, created_at, actor, decision, note) VALUES(?,?,?,?,?,?)`,
		302, 301, 1201, "operator", "approved", "looks good")

	mustExec(t, ctx, db, `INSERT INTO events(id, created_at, type, payload_json) VALUES(?,?,?,?)`,
		401, 1300, "job.started", `{"jobId":"job-1"}`)

	mustExec(t, ctx, db, `
INSERT INTO artifacts(
  id, created_at, job_id, packet_id, type, title, file_path, mime_type, metadata_json
) VALUES(?,?,?,?,?,?,?,?,?)`,
		501, 1400, "job-1", 11, "report", "Result", "/tmp/result.txt", "text/plain", `{"sha":"abc"}`)

	mustExec(t, ctx, db, `
INSERT INTO gateway_invocations(
  id, correlation_id, created_at, completed_at, tool_id, lane_id, job_id, packet_id,
  approval_request_id, initiator, action, risk_class, write_intent, scope_json, input_json,
  status, denied_reason, result_json, artifacts_json, permission_profile_id
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		603, "corr-123", 2200, 2201, "tool.backup", "lane-a", "job-1", 11,
		301, "operator", "restore", "medium", 1,
		`{"section":"full_backup"}`, `{"dryRun":false}`, "completed", "", `{"ok":true}`, `["artifact-1"]`, "perm-1")

	mustExec(t, ctx, db, `
INSERT INTO audit_records(
  id, created_at, correlation_id, category, action, actor, subject_type, subject_id,
  job_id, gateway_invocation_id, approval_request_id, risk_class, outcome, summary, payload_json
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		604, 2300, "corr-123", "gateway", "tool.completed", "operator", "tool", "tool.backup",
		"job-1", 603, 301, "medium", "allowed", "tool completed", `{"result":"ok"}`)

	mustExec(t, ctx, db, `
INSERT INTO semantic_idempotency_keys(idempotency_key, action, result_json, created_at, correlation_id)
VALUES(?,?,?,?,?)`,
		"idem-1", "CREATE_NOTE", `{"success":true}`, 2400, "corr-123")

	mustExec(t, ctx, db, `
INSERT INTO dream_reports(
  id, created_at, completed_at, workspace_id, lane_id, mode, dry_run, status,
  time_window_start, time_window_end, candidates_considered, proposals_generated,
  summary_json, candidates_json, salience_scores_json, memory_tier_proposals_json,
  repair_proposals_json, snapshot_hygiene_proposals_json, warnings_json, trace_json,
  correlation_id, trace_id, syscall_id, audit_id, proposed_by, committed_by, metadata_json
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		"dream-report-1", 2450, 2451, "workspace-1", "lane-a", "nap", 1, "completed",
		2000, 2450, 2, 2,
		`{"summary":"dry-run report","nonCanonicalEvidence":true,"canonicalWriteCommitted":false}`,
		`[{"candidate_id":"cand-1","source_type":"memory_note","source_ids":["note-1"],"workspace_id":"workspace-1","lane_id":"lane-a","start_timestamp":2400,"end_timestamp":2400,"content_summary":"candidate","tags":["correction"],"related_goal_ids":[],"related_loop_ids":[],"related_snapshot_ids":[],"raw_importance_signals":{},"trace":{}}]`,
		`[{"candidate_id":"cand-1","total_salience":0.8,"confidence":0.8}]`,
		`[{"candidate_id":"cand-1","source_type":"memory_note","decision":"promote_mid_term","confidence":0.8,"reason":"dry run","dry_run":true}]`,
		`[]`, `[]`, `[]`,
		`{"cpu_only":true,"canonical_write_committed":false}`,
		"corr-123", "trace-123", nil, nil, "forge.dream", "", `{"evidence":"non_canonical"}`)

	mustExec(t, ctx, db, `
INSERT INTO provenance_records(
  id, actor, actor_type, source, trace_id, workspace_id, lane_id, selected_paths_json,
  metadata_json, created_at, proposed_by, committed_by, syscall_id, correlation_id, audit_id
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		"prov-1", "worker-1", "agent", "backup.test", "trace-123", "workspace-1", "lane-a", `["/tmp/result.txt"]`,
		`{"stage":"test"}`, 1500, "worker-1", "forge_kernel", "sys-123", "corr-123", "audit-123")

	mustExec(t, ctx, db, `
INSERT INTO journal_events(
  id, type, source, actor, workspace_id, lane_id, selected_paths_json, payload_json,
  correlation_id, trace_id, provenance_id, provenance_json, created_at, metadata_json,
  proposed_by, committed_by, syscall_id, audit_id
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		"journal-1", "state.updated", "backup.test", "worker-1", "workspace-1", "lane-a", `["/tmp/result.txt"]`, `{"state":"running"}`,
		"corr-123", "trace-123", "prov-1", `{"id":"prov-1"}`, 1501, `{"scope":"test"}`,
		"worker-1", "forge_kernel", "sys-123", "audit-123")

	mustExec(t, ctx, db, `
INSERT INTO memory_notes(
  id, type, title, content, workspace_id, lane_id, selected_paths_json, confidence, status,
  provenance_id, provenance_json, created_at, updated_at, archived_at, superseded_by, metadata_json,
  proposed_by, committed_by, syscall_id, correlation_id, trace_id, audit_id
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		"note-1", "fact", "Backup note", "Restored section parity", "workspace-1", "lane-a", `["/tmp/result.txt"]`, 0.91, "active",
		"prov-1", `{"id":"prov-1"}`, 1600, 1601, nil, nil, `{"topic":"backup"}`,
		"worker-1", "forge_kernel", "sys-123", "corr-123", "trace-123", "audit-123")

	mustExec(t, ctx, db, `
INSERT INTO artifact_refs(
  id, type, uri, content_hash, workspace_id, lane_id, selected_paths_json, provenance_id,
  provenance_json, created_at, metadata_json, proposed_by, committed_by, syscall_id,
  correlation_id, trace_id, audit_id
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		"aref-1", "file", "file:///tmp/result.txt", "sha256:abc", "workspace-1", "lane-a", `["/tmp/result.txt"]`, "prov-1",
		`{"id":"prov-1"}`, 1700, `{"kind":"result"}`, "worker-1", "forge_kernel", "sys-123", "corr-123", "trace-123", "audit-123")

	mustExec(t, ctx, db, `
INSERT INTO model_manifests(
  id, schema_version, display_name, family, format, backend, model_path, sha256, size_bytes,
  quantization, context_length, capabilities_json, default_runtime_json, license_json,
  metadata_json, discovered_at, updated_at
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		"local-chat", "forge.model/v1", "Local Chat", "llama", "gguf", "llama_cpp", "model.gguf", "sha-model", 12345,
		"q4", 4096, `["chat","completion"]`, `{"maxTokens":256}`, `{"name":"unknown"}`,
		`{"managed":true}`, 2500, 2501)
	mustExec(t, ctx, db, `
INSERT INTO model_registry_status(model_id, backend, status, updated_at, last_error, metadata_json)
VALUES(?,?,?,?,?,?)`,
		"local-chat", "llama_cpp", "available", 2502, "", `{"verified":true}`)
	mustExec(t, ctx, db, `
INSERT INTO model_runtime_loads(
  id, model_id, backend, status, loaded_at, unloaded_at, endpoint, pid, resource_usage_json, metadata_json
) VALUES(?,?,?,?,?,?,?,?,?,?)`,
		801, "local-chat", "llama_cpp", "loaded", 2503, nil, "http://127.0.0.1:8080", 0, `{"vram":0}`, `{"source":"test"}`)

	mustExec(t, ctx, db, `INSERT INTO chat_threads(id, title, created_at, updated_at, dossier_id) VALUES(?,?,?,?,?)`,
		801, "Restored Chat", 2600, 2601, 701)
	mustExec(t, ctx, db, `INSERT INTO chat_messages(id, thread_id, role, content, created_at, metadata_json) VALUES(?,?,?,?,?,?)`,
		802, 801, "user", "hello restored chat", 2602, `{"traceId":"trace-123"}`)

	mustExec(t, ctx, db, `INSERT INTO canvas_boards(id, title, dossier_id, created_at, updated_at) VALUES(?,?,?,?,?)`,
		803, "Restore Board", 701, 2700, 2701)
	mustExec(t, ctx, db, `
INSERT INTO canvas_notes(id, board_id, title, body, x, y, width, height, pinned, color, links_json, created_at, updated_at)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		804, 803, "Restore Note", "canvas state", 10, 20, 260, 180, 1, "blue", `[]`, 2702, 2703)

	mustExec(t, ctx, db, `
INSERT INTO tool_capability_overrides(capability_id, status, reason, actor, updated_at)
VALUES(?,?,?,?,?)`,
		"filesystem.write_file", "approval_only", "backup parity fixture", "operator", 2800)
}

func mustExec(t *testing.T, ctx context.Context, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.ExecContext(ctx, query, args...); err != nil {
		t.Fatalf("exec query failed: %v\nquery=%s", err, query)
	}
}

func stageBundleForRestore(t *testing.T, svc *Service, sourcePath string) string {
	t.Helper()
	backupDir, _ := svc.Dirs()
	raw, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("read source bundle: %v", err)
	}
	dst := filepath.Join(backupDir, filepath.Base(sourcePath))
	if err := os.WriteFile(dst, raw, 0o644); err != nil {
		t.Fatalf("stage bundle: %v", err)
	}
	return dst
}

func restoreStagingPath(t *testing.T, svc *Service, name string) string {
	t.Helper()
	backupDir, _ := svc.Dirs()
	return filepath.Join(backupDir, name)
}
