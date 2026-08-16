package backup

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"forge/projectforge/services/core/internal/store"
)

func TestRestoreBundleNonDryFailsClosedBeforeReadingPath(t *testing.T) {
	svc := New(nil, t.TempDir())
	result, err := svc.RestoreBundle(context.Background(), RestoreBundleRequest{
		FilePath: filepath.Join(t.TempDir(), "does-not-exist.json"),
		DryRun:   false,
	})
	if result != nil {
		t.Fatalf("disabled apply returned a result: %+v", result)
	}
	if !errors.Is(err, ErrForgeKRestoreApplyDisabled) {
		t.Fatalf("error=%v want=%v", err, ErrForgeKRestoreApplyDisabled)
	}
}

func TestInspectFullBackupProducesDeterministicNonMergeablePlan(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	svc := New(st.DB, dataDir)
	bundle, err := svc.CreateBundle(ctx, CreateBundleRequest{Kind: "full_backup", Label: "inspection"})
	if err != nil {
		t.Fatalf("create bundle: %v", err)
	}
	first, err := svc.InspectBundle(ctx, RestoreBundleRequest{FilePath: bundle.FilePath, DryRun: true})
	if err != nil {
		t.Fatalf("inspect bundle: %v", err)
	}
	second, err := svc.InspectBundle(ctx, RestoreBundleRequest{FilePath: bundle.FilePath, DryRun: true})
	if err != nil {
		t.Fatalf("inspect bundle again: %v", err)
	}
	if !first.Accepted || !first.DryRun || !first.InspectionOnly || first.Applied || first.Atomic || first.GlobalAtomic {
		t.Fatalf("inspection flags do not fail closed: %+v", first)
	}
	if first.AtomicScope != "none-inspection-only" {
		t.Fatalf("atomic scope=%q", first.AtomicScope)
	}
	if first.BundleSHA256 == "" || first.PlanDigest == "" || first.PlanDigest != second.PlanDigest {
		t.Fatalf("non-deterministic or missing proof: first=%q second=%q bundle=%q", first.PlanDigest, second.PlanDigest, first.BundleSHA256)
	}
	if !sort.StringsAreSorted(first.EffectiveSections) {
		t.Fatalf("effective sections are not normalized: %+v", first.EffectiveSections)
	}

	wantDisposition := map[string]string{
		"journal_events":                        "never_live_merge",
		"semantic_idempotency_keys":             "never_live_merge",
		"forge_k_journal_head":                  "offline_recovery_only",
		"forge_k_audit_outbox":                  "offline_recovery_only",
		"court_exhibits":                        "offline_recovery_only",
		"court_rulings":                         "offline_recovery_only",
		"court_appeals":                         "offline_recovery_only",
		"forge_k_memory_evidence":               "offline_recovery_only",
		"forge_k_memory_evidence_supersessions": "offline_recovery_only",
		"forge_k_semantic_diff_operations":      "offline_recovery_only",
		"forge_k_semantic_diff_results":         "offline_recovery_only",
		"forge_k_semantic_derived_objects":      "offline_recovery_only",
		"forge_k_context_bundles":               "offline_recovery_only",
		"forge_k_context_snapshot_heads":        "offline_recovery_only",
	}
	for section, want := range wantDisposition {
		inspection, ok := first.SectionInspections[section]
		if !ok {
			t.Fatalf("missing authority section %q from full backup inspection", section)
		}
		if inspection.Disposition != want {
			t.Fatalf("section %s disposition=%q want=%q", section, inspection.Disposition, want)
		}
		if len(inspection.Blockers) == 0 {
			t.Fatalf("section %s has no explicit blocker", section)
		}
		if inspection.ComputedCount != inspection.DeclaredCount || inspection.ComputedChecksum == "" || inspection.ComputedChecksum != inspection.DeclaredChecksum {
			t.Fatalf("section %s lacks count/checksum inspection proof: %#v", section, inspection)
		}
	}
}

func TestInspectBundleRejectsCountAndChecksumTampering(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	svc := New(st.DB, dataDir)
	bundle, err := svc.CreateBundle(ctx, CreateBundleRequest{Kind: "dossiers", Label: "tamper"})
	if err != nil {
		t.Fatalf("create bundle: %v", err)
	}
	raw, err := os.ReadFile(bundle.FilePath)
	if err != nil {
		t.Fatalf("read bundle: %v", err)
	}
	var doc BundleDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("decode bundle: %v", err)
	}
	doc.Entities["dossiers"] = append(doc.Entities["dossiers"], map[string]any{"id": "injected"})
	tampered, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatalf("encode tampered bundle: %v", err)
	}
	if err := os.WriteFile(bundle.FilePath, tampered, 0o600); err != nil {
		t.Fatalf("write tampered bundle: %v", err)
	}

	result, err := svc.InspectBundle(ctx, RestoreBundleRequest{FilePath: bundle.FilePath, DryRun: true})
	if err != nil {
		t.Fatalf("inspect tampered bundle: %v", err)
	}
	if result.Accepted {
		t.Fatalf("tampered bundle was accepted: %+v", result)
	}
	joined := strings.Join(result.Errors, "\n")
	if !strings.Contains(joined, "entity count mismatch for dossiers") || !strings.Contains(joined, "checksum mismatch for dossiers") {
		t.Fatalf("missing integrity failures: %s", joined)
	}
	if result.Verification["counts"] != "failed" || result.Verification["checksums"] != "failed" {
		t.Fatalf("verification did not expose failures: %+v", result.Verification)
	}
}

func TestProductionRestoreHasNoRawApplyCallsite(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current source path")
	}
	internalDir := filepath.Dir(filepath.Dir(currentFile))
	var rawApplyCallsites []string
	err := filepath.WalkDir(internalDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		text := string(raw)
		if strings.Contains(text, ".restoreBundleLegacyForTest(") {
			rawApplyCallsites = append(rawApplyCallsites, path)
		}
		if strings.Contains(text, ".RestoreBundle(") {
			rawApplyCallsites = append(rawApplyCallsites, path+": public RestoreBundle call")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk production sources: %v", err)
	}
	if len(rawApplyCallsites) != 0 {
		t.Fatalf("production restore apply callsites found: %+v", rawApplyCallsites)
	}

	apiSource, err := os.ReadFile(filepath.Join(internalDir, "api", "phase5.go"))
	if err != nil {
		t.Fatalf("read restore API source: %v", err)
	}
	if !strings.Contains(string(apiSource), ".InspectBundle(") {
		t.Fatal("restore API is not bound to inspection")
	}
	for _, retired := range []string{
		"evaluateBackupRestoreGovernance",
		"backupRestoreApprovalFingerprint",
		"ensureBackupRestoreApprovalJob",
		"auditBackupRestoreGovernance",
	} {
		if strings.Contains(string(apiSource), retired) {
			t.Fatalf("restore API retains retired apply/approval authority %q", retired)
		}
	}
	disabledAt := strings.Index(string(apiSource), "FORGE_K_RESTORE_APPLY_DISABLED")
	pathAt := strings.Index(string(apiSource), ".ResolveRestorePath(")
	inspectAt := strings.Index(string(apiSource), ".InspectBundle(")
	if disabledAt < 0 || pathAt < 0 || inspectAt < 0 || !(disabledAt < pathAt && pathAt < inspectAt) {
		t.Fatalf("restore API fail-close order is not disabled -> governed path -> inspection: disabled=%d path=%d inspect=%d", disabledAt, pathAt, inspectAt)
	}
	iolaneSource, err := os.ReadFile(filepath.Join(internalDir, "aios", "iolane", "interfaces.go"))
	if err != nil {
		t.Fatalf("read I/O lane source: %v", err)
	}
	if strings.Contains(string(iolaneSource), "RestoreBundle(") {
		t.Fatal("live I/O lane still exposes restore apply authority")
	}
}
