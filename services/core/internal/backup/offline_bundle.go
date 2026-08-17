package backup

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// LoadVerifiedFullBundle loads a full backup for the standalone offline
// recovery path. It reuses the production inspection contract, then rereads
// and hash-binds the exact bytes returned to the recovery importer. It never
// applies rows to Service.db.
func (s *Service) LoadVerifiedFullBundle(ctx context.Context, filePath string) (BundleDoc, *RestoreResult, error) {
	inspection, err := s.InspectBundle(ctx, RestoreBundleRequest{FilePath: filePath, DryRun: true})
	if err != nil {
		return BundleDoc{}, inspection, err
	}
	if inspection == nil || !inspection.Accepted {
		details := ""
		if inspection != nil && len(inspection.Errors) > 0 {
			details = ": " + strings.Join(inspection.Errors, "; ")
		}
		return BundleDoc{}, inspection, fmt.Errorf("backup bundle failed deterministic inspection%s", details)
	}
	resolved, err := s.ResolveRestorePath(filePath)
	if err != nil {
		return BundleDoc{}, inspection, err
	}
	raw, err := readRestoreBundleFile(resolved)
	if err != nil {
		return BundleDoc{}, inspection, err
	}
	sum := sha256.Sum256(raw)
	digest := "sha256:" + hex.EncodeToString(sum[:])
	if digest != inspection.BundleSHA256 {
		return BundleDoc{}, inspection, fmt.Errorf("backup bundle changed after inspection")
	}
	doc, err := decodeBundleDocStrict(raw)
	if err != nil {
		return BundleDoc{}, inspection, err
	}
	if strings.ToLower(strings.TrimSpace(doc.Kind)) != "full_backup" {
		return BundleDoc{}, inspection, fmt.Errorf("offline recovery requires kind full_backup")
	}
	expected, err := s.pickSections("full_backup")
	if err != nil {
		return BundleDoc{}, inspection, err
	}
	missing := []string{}
	for _, section := range expected {
		if _, ok := doc.Entities[section]; !ok {
			missing = append(missing, section)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return BundleDoc{}, inspection, fmt.Errorf("full backup missing required sections: %s", strings.Join(missing, ","))
	}
	return doc, inspection, nil
}

// VerifyDatabaseSections proves that the staged database contains exactly the
// row counts and deterministic section checksums declared by the inspected
// bundle. It is intentionally unavailable through the live restore API.
func VerifyDatabaseSections(ctx context.Context, db *sql.DB, doc BundleDoc) error {
	if db == nil {
		return fmt.Errorf("staged database is required")
	}
	svc := &Service{db: db}
	for _, section := range knownSections(doc) {
		rows, err := svc.extractSection(ctx, section)
		if err != nil {
			return fmt.Errorf("verify restored section %s: %w", section, err)
		}
		if got, want := len(rows), doc.EntityCounts[section]; got != want {
			return fmt.Errorf("restored row count mismatch for %s: got=%d want=%d", section, got, want)
		}
		if got, want := checksumRows(rows), strings.TrimSpace(doc.Checksums[section]); !strings.EqualFold(got, want) {
			return fmt.Errorf("restored checksum mismatch for %s", section)
		}
	}
	return nil
}
