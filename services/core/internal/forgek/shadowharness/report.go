package shadowharness

import (
	"fmt"
	"strings"
)

func NewShadowObservation(input ShadowObservation) (ShadowObservation, error) {
	if input.ObservationID == "" || input.WorkspaceID == "" || input.RequestID == "" ||
		input.ObservedAt.IsZero() || input.LivePath == "" || input.RequestSummary == "" {
		return ShadowObservation{}, fmt.Errorf("%w: shadow observation", ErrMissingRequiredField)
	}
	if hasSecretMetadata(input.Metadata) {
		return ShadowObservation{}, ErrSecretMetadata
	}
	input.InputRefs = normalizeStrings(input.InputRefs)
	input.EvidenceRefs = normalizeStrings(input.EvidenceRefs)
	input.RetrievalRefs = normalizeStrings(input.RetrievalRefs)
	input.ContextRefs = normalizeStrings(input.ContextRefs)
	input.ConsensusRefs = normalizeStrings(input.ConsensusRefs)
	input.RuntimeRefs = normalizeStrings(input.RuntimeRefs)
	input.KVRefs = normalizeStrings(input.KVRefs)
	input.RiskFlags = normalizeStrings(input.RiskFlags)
	return input, nil
}

func NewShadowComparisonReport(input ShadowComparisonReport) (ShadowComparisonReport, error) {
	if input.ReportID == "" || input.WorkspaceID == "" || input.RequestID == "" || input.GeneratedAt.IsZero() {
		return ShadowComparisonReport{}, fmt.Errorf("%w: shadow comparison report", ErrMissingRequiredField)
	}
	if hasSecretMetadata(input.Metadata) {
		return ShadowComparisonReport{}, ErrSecretMetadata
	}
	input.ObservationRefs = normalizeStrings(input.ObservationRefs)
	input.Divergences = normalizeStrings(input.Divergences)
	input.Warnings = normalizeStrings(input.Warnings)
	input.Blockers = normalizeStrings(input.Blockers)
	input.RAGShadow = normalizeRAGReport(input.RAGShadow)
	input.RuntimeShadow = normalizeRuntimeReport(input.RuntimeShadow)
	input.KVShadow = normalizeKVReport(input.KVShadow)
	input.LymphaticShadow = normalizeLymphaticReport(input.LymphaticShadow)
	return input, nil
}

func NewRAGShadowReport(input RAGShadowReport) (RAGShadowReport, error) {
	if input.ReportID == "" || input.RequestID == "" {
		return RAGShadowReport{}, fmt.Errorf("%w: rag shadow report", ErrMissingRequiredField)
	}
	if hasSecretMetadata(input.Metadata) {
		return RAGShadowReport{}, ErrSecretMetadata
	}
	return normalizeRAGReport(input), nil
}

func (o ShadowObservation) IsDiagnosticOnly() bool           { return true }
func (o ShadowObservation) CanMutateLiveState() bool         { return false }
func (o ShadowObservation) CanAffectUserVisibleOutput() bool { return false }

func (r ShadowComparisonReport) IsDiagnosticOnly() bool   { return true }
func (r ShadowComparisonReport) CanMutateLiveState() bool { return false }
func (r ShadowComparisonReport) CanExecuteActions() bool  { return false }
func (r ShadowComparisonReport) CanWriteMemory() bool     { return false }

func normalizeRAGReport(report RAGShadowReport) RAGShadowReport {
	report.RetrievalRefs = normalizeStrings(report.RetrievalRefs)
	report.EvidenceRefs = normalizeStrings(report.EvidenceRefs)
	report.SourceRefs = normalizeStrings(report.SourceRefs)
	report.StaleRefs = normalizeStrings(report.StaleRefs)
	report.UnsupportedRefs = normalizeStrings(report.UnsupportedRefs)
	report.Warnings = normalizeStrings(report.Warnings)
	return report
}

func normalizeRuntimeReport(report RuntimeShadowReport) RuntimeShadowReport {
	report.RuntimeResultRefs = normalizeStrings(report.RuntimeResultRefs)
	report.DriverRefs = normalizeStrings(report.DriverRefs)
	report.ModelIdentityRefs = normalizeStrings(report.ModelIdentityRefs)
	report.Warnings = normalizeStrings(report.Warnings)
	return report
}

func normalizeKVReport(report KVShadowReport) KVShadowReport {
	report.KVManifestRefs = normalizeStrings(report.KVManifestRefs)
	report.Warnings = normalizeStrings(report.Warnings)
	return report
}

func normalizeLymphaticReport(report LymphaticShadowReport) LymphaticShadowReport {
	report.MaintenanceReportRefs = normalizeStrings(report.MaintenanceReportRefs)
	report.Warnings = normalizeStrings(report.Warnings)
	return report
}

func hasSecretMetadata(metadata map[string]any) bool {
	for key := range metadata {
		normalized := strings.ToLower(key)
		if strings.Contains(normalized, "secret") || strings.Contains(normalized, "token") ||
			strings.Contains(normalized, "password") || strings.Contains(normalized, "credential") {
			return true
		}
	}
	return false
}
