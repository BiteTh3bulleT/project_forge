package backup

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
)

var ErrForgeKRestoreApplyDisabled = errors.New("FORGE_K_RESTORE_APPLY_DISABLED")

const restoreInspectionPlanVersion = "forge_k.restore_inspection.v1"

type RestoreSectionInspection struct {
	Name             string   `json:"name"`
	AuthorityClass   string   `json:"authorityClass"`
	Disposition      string   `json:"disposition"`
	ComputedCount    int      `json:"computedCount"`
	DeclaredCount    int      `json:"declaredCount"`
	ComputedChecksum string   `json:"computedChecksum"`
	DeclaredChecksum string   `json:"declaredChecksum"`
	Blockers         []string `json:"blockers"`
}

type restoreInspectionPlan struct {
	Version           string                     `json:"version"`
	BundleSHA256      string                     `json:"bundleSha256"`
	Schema            int                        `json:"schema"`
	Kind              string                     `json:"kind"`
	EffectiveSections []string                   `json:"effectiveSections"`
	Sections          []RestoreSectionInspection `json:"sections"`
}

// RestoreBundle is retained as a compatibility entrypoint for callers that
// previously used it for both preview and apply. Production restore is now
// inspection-only; every non-dry request fails before reading or mutating the
// store.
func (s *Service) RestoreBundle(ctx context.Context, req RestoreBundleRequest) (*RestoreResult, error) {
	if !req.DryRun {
		return nil, ErrForgeKRestoreApplyDisabled
	}
	return s.InspectBundle(ctx, req)
}

// InspectBundle validates and seals a deterministic restore proposal without
// applying any row. It deliberately reports every section as non-live-mergeable
// until a Kernel-owned semantic migration or daemon-stopped recovery path exists.
func (s *Service) InspectBundle(_ context.Context, req RestoreBundleRequest) (*RestoreResult, error) {
	if !req.DryRun {
		return nil, ErrForgeKRestoreApplyDisabled
	}
	filePath, err := s.ResolveRestorePath(req.FilePath)
	if err != nil {
		return nil, err
	}
	raw, err := readRestoreBundleFile(filePath)
	if err != nil {
		return nil, err
	}
	rawSum := sha256.Sum256(raw)
	bundleDigest := "sha256:" + hex.EncodeToString(rawSum[:])

	duplicates, err := duplicateJSONKeys(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid bundle JSON: %w", err)
	}
	doc, err := decodeBundleDocStrict(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid bundle JSON: %w", err)
	}
	result := newRestoreInspectionResult(doc, bundleDigest)
	if doc.Schema != BundleSchemaVersion {
		result.Errors = append(result.Errors, fmt.Sprintf("unsupported bundle schema %d (want %d)", doc.Schema, BundleSchemaVersion))
	}
	if !isKnownKind(strings.ToLower(strings.TrimSpace(doc.Kind))) {
		result.Errors = append(result.Errors, fmt.Sprintf("unknown bundle kind %q", doc.Kind))
	}
	for _, duplicate := range duplicates {
		result.Errors = append(result.Errors, "duplicate JSON key: "+duplicate)
	}

	requested, requestedDuplicates := normalizeSectionsForInspection(req.Sections)
	for _, duplicate := range requestedDuplicates {
		result.Errors = append(result.Errors, "duplicate requested section: "+duplicate)
	}
	if len(requested) == 0 {
		requested = knownSections(doc)
	}
	result.EffectiveSections = append([]string{}, requested...)

	manifestByName := map[string]SectionManifest{}
	for _, manifest := range doc.Manifest {
		name := strings.ToLower(strings.TrimSpace(manifest.Name))
		if name == "" {
			result.Errors = append(result.Errors, "manifest contains an empty section name")
			continue
		}
		if _, exists := manifestByName[name]; exists {
			result.Errors = append(result.Errors, "duplicate manifest section: "+name)
			continue
		}
		manifest.Name = name
		manifestByName[name] = manifest
		if _, exists := doc.Entities[name]; !exists {
			result.Errors = append(result.Errors, "manifest section has no entity payload: "+name)
		}
	}

	allSections := knownSections(doc)
	for _, section := range sortedInspectionMapKeys(doc.EntityCounts) {
		if _, exists := doc.Entities[section]; !exists {
			result.Errors = append(result.Errors, "entity count has no section payload: "+section)
		}
	}
	for _, section := range sortedInspectionMapKeys(doc.Checksums) {
		if _, exists := doc.Entities[section]; !exists {
			result.Errors = append(result.Errors, "checksum has no section payload: "+section)
		}
	}
	computed := map[string]RestoreSectionInspection{}
	for _, section := range allSections {
		rows := doc.Entities[section]
		inspection := inspectRestoreSection(section, rows, doc.EntityCounts, doc.Checksums)
		expectedManifest := backupSectionManifest(section)
		manifest, exists := manifestByName[section]
		if !exists {
			result.Errors = append(result.Errors, "manifest missing section: "+section)
		} else {
			result.Errors = append(result.Errors, validateRestoreManifest(section, manifest, expectedManifest)...)
		}
		if _, exists := doc.EntityCounts[section]; !exists {
			result.Errors = append(result.Errors, "entity count missing for section: "+section)
		} else if inspection.DeclaredCount != inspection.ComputedCount {
			result.Errors = append(result.Errors, fmt.Sprintf("entity count mismatch for %s: declared=%d computed=%d", section, inspection.DeclaredCount, inspection.ComputedCount))
		}
		if strings.TrimSpace(inspection.DeclaredChecksum) == "" {
			result.Errors = append(result.Errors, "checksum missing for section: "+section)
		} else if !strings.EqualFold(inspection.DeclaredChecksum, inspection.ComputedChecksum) {
			result.Errors = append(result.Errors, fmt.Sprintf("checksum mismatch for %s", section))
		}
		if !knownBackupSection(section) {
			result.Errors = append(result.Errors, "unknown section authority: "+section)
			inspection.AuthorityClass = "unknown"
			inspection.Disposition = "unsupported"
			inspection.Blockers = []string{"section has no local authority contract", "live restore apply is disabled"}
		}
		computed[section] = inspection
	}

	planSections := make([]RestoreSectionInspection, 0, len(requested))
	for _, section := range requested {
		inspection, exists := computed[section]
		if !exists {
			inspection = RestoreSectionInspection{
				Name: section, AuthorityClass: "unknown", Disposition: "unsupported",
				Blockers: []string{"section not found in bundle", "live restore apply is disabled"},
			}
			result.Errors = append(result.Errors, "requested section not found in bundle: "+section)
			result.Unsupported[section] = "section not found in bundle"
		}
		result.SectionInspections[section] = inspection
		result.Planned[section] = inspection.ComputedCount
		planSections = append(planSections, inspection)
	}
	sort.Slice(planSections, func(i, j int) bool { return planSections[i].Name < planSections[j].Name })
	plan := restoreInspectionPlan{
		Version: restoreInspectionPlanVersion, BundleSHA256: bundleDigest, Schema: doc.Schema,
		Kind: strings.ToLower(strings.TrimSpace(doc.Kind)), EffectiveSections: append([]string{}, requested...), Sections: planSections,
	}
	planRaw, err := json.Marshal(plan)
	if err != nil {
		return nil, err
	}
	planSum := sha256.Sum256(planRaw)
	result.PlanDigest = "sha256:" + hex.EncodeToString(planSum[:])
	result.Verification["bundleSha256"] = bundleDigest
	result.Verification["schema"] = validationState(result.Errors, "schema")
	result.Verification["kind"] = validationState(result.Errors, "kind")
	result.Verification["manifest"] = validationState(result.Errors, "manifest")
	result.Verification["counts"] = validationState(result.Errors, "count")
	result.Verification["checksums"] = validationState(result.Errors, "checksum")
	result.Verification["duplicates"] = validationState(result.Errors, "duplicate")
	result.Verification["authority"] = validationState(result.Errors, "authority")
	result.Verification["planDigest"] = result.PlanDigest
	result.Accepted = len(result.Errors) == 0
	return result, nil
}

func sortedInspectionMapKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func newRestoreInspectionResult(doc BundleDoc, bundleDigest string) *RestoreResult {
	return &RestoreResult{
		DryRun: true, InspectionOnly: true, AtomicScope: "none-inspection-only", GlobalAtomic: false,
		BundleKind: doc.Kind, Imported: map[string]int{}, Planned: map[string]int{}, Skipped: map[string]int{},
		Unsupported: map[string]string{}, ExportOnly: map[string]string{}, NonDBSideEffects: map[string]string{},
		Warnings: []string{"live restore apply is disabled; inspection does not mutate FORGE"}, Errors: []string{}, Schema: doc.Schema,
		Meta:         map[string]string{"label": doc.Label, "versionTag": doc.VersionTag, "sourceVersion": doc.SourceVer},
		Verification: map[string]any{}, BundleSHA256: bundleDigest, EffectiveSections: []string{},
		SectionInspections: map[string]RestoreSectionInspection{},
	}
}

func inspectRestoreSection(section string, rows []any, counts map[string]int, checksums map[string]string) RestoreSectionInspection {
	manifest := backupSectionManifest(section)
	disposition, blockers := restoreInspectionDisposition(section, manifest.AuthorityClass)
	return RestoreSectionInspection{
		Name: section, AuthorityClass: manifest.AuthorityClass, Disposition: disposition,
		ComputedCount: len(rows), DeclaredCount: counts[section], ComputedChecksum: checksumRows(rows),
		DeclaredChecksum: strings.TrimSpace(checksums[section]), Blockers: blockers,
	}
}

func restoreInspectionDisposition(section, authorityClass string) (string, []string) {
	global := "live restore apply is disabled until daemon-stopped, whole-store, chain-verified recovery exists"
	switch section {
	case "journal_events":
		return "never_live_merge", []string{"journal events must be produced by the local FORGE-K hash chain", "import cannot advance or replace the live journal head", global}
	case "semantic_idempotency_keys":
		return "never_live_merge", []string{"idempotency proof is local immutable replay authority", "legacy or foreign proof must fail closed", global}
	case "forge_k_journal_head", "forge_k_audit_outbox":
		return "offline_recovery_only", []string{"FORGE-K commit proof cannot be merged into a running store", global}
	case "court_exhibits", "court_rulings", "court_appeals":
		return "offline_recovery_only", []string{"Courthouse current state and immutable history require whole-store consistency", global}
	case "provenance_records", "audit_records", "approval_requests", "approval_decisions", "gateway_invocations", "events", "job_events", "job_status_history":
		return "quarantine_evidence_only", []string{"foreign history cannot overwrite local immutable lineage", global}
	}
	switch authorityClass {
	case "canonical_or_historical_truth", "historical_truth":
		return "kernel_semantic_migration_required", []string{"current and historical truth require object-specific Kernel transitions", global}
	case "retrieval_index":
		return "rebuild_required", []string{"derived retrieval state must be rebuilt or reconciled", global}
	case "kernel_commit_proof":
		return "offline_recovery_only", []string{"Kernel proof state requires whole-store recovery verification", global}
	case "non_canonical_evidence":
		return "quarantine_evidence_only", []string{"foreign evidence requires immutable namespaced import", global}
	default:
		return "dedicated_reconciliation_required", []string{"operational state requires its owning authority and policy", global}
	}
}

func validateRestoreManifest(section string, got, want SectionManifest) []string {
	issues := []string{}
	if got.AuthorityClass != want.AuthorityClass {
		issues = append(issues, fmt.Sprintf("manifest authority mismatch for %s: declared=%q expected=%q", section, got.AuthorityClass, want.AuthorityClass))
	}
	if got.BackupRequired != want.BackupRequired || got.RestoreRequired != want.RestoreRequired || got.ExportOnly != want.ExportOnly || got.IntegrityCheckRequired != want.IntegrityCheckRequired {
		issues = append(issues, "manifest policy mismatch for section: "+section)
	}
	return issues
}

func knownBackupSection(section string) bool {
	_, extractable := extractQueries[section]
	return extractable
}

func normalizeSectionsForInspection(sections []string) ([]string, []string) {
	seen := map[string]bool{}
	out := []string{}
	duplicates := []string{}
	for _, raw := range sections {
		section := strings.ToLower(strings.TrimSpace(raw))
		if section == "" {
			continue
		}
		if seen[section] {
			duplicates = append(duplicates, section)
			continue
		}
		seen[section] = true
		out = append(out, section)
	}
	sort.Strings(out)
	sort.Strings(duplicates)
	return out, duplicates
}

func validationState(issues []string, token string) string {
	for _, issue := range issues {
		if strings.Contains(strings.ToLower(issue), strings.ToLower(token)) {
			return "failed"
		}
	}
	return "passed"
}

func decodeBundleDocStrict(raw []byte) (BundleDoc, error) {
	var doc BundleDoc
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&doc); err != nil {
		return BundleDoc{}, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return BundleDoc{}, errors.New("multiple JSON values")
		}
		return BundleDoc{}, err
	}
	return doc, nil
}

func duplicateJSONKeys(raw []byte) ([]string, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	duplicates := []string{}
	var walk func(string) error
	walk = func(path string) error {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		delim, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch delim {
		case '{':
			seen := map[string]bool{}
			for decoder.More() {
				keyToken, err := decoder.Token()
				if err != nil {
					return err
				}
				key, ok := keyToken.(string)
				if !ok {
					return errors.New("object key is not a string")
				}
				childPath := key
				if path != "" {
					childPath = path + "." + key
				}
				if seen[key] {
					duplicates = append(duplicates, childPath)
				}
				seen[key] = true
				if err := walk(childPath); err != nil {
					return err
				}
			}
			_, err = decoder.Token()
			return err
		case '[':
			index := 0
			for decoder.More() {
				if err := walk(fmt.Sprintf("%s[%d]", path, index)); err != nil {
					return err
				}
				index++
			}
			_, err = decoder.Token()
			return err
		default:
			return fmt.Errorf("unexpected JSON delimiter %q", delim)
		}
	}
	if err := walk(""); err != nil {
		return nil, err
	}
	sort.Strings(duplicates)
	return duplicates, nil
}
