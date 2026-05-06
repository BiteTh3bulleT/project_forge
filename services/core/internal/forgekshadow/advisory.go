package forgekshadow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

func buildShadowAdvisory(report DiagnosticReport, now time.Time) (ShadowAdvisoryReport, error) {
	refs := collectAdvisoryRefs(report)
	warnings := make([]string, 0, 4)
	riskFlags := make([]string, 0, 4)
	if len(refs.safeRefs) == 0 {
		warnings = append(warnings, "insufficient_safe_metadata")
		riskFlags = append(riskFlags, "insufficient_safe_metadata")
	}
	if report.RetrievalMetadata != nil {
		riskFlags = append(riskFlags, "retrieval_metadata_only")
	}
	consensus := buildConsensusAdvisory(len(refs.safeRefs))
	contextAdvisory := buildContextCompilerAdvisory(report, refs.safeRefs)
	warnings = append(warnings, consensus.Warnings...)
	warnings = append(warnings, contextAdvisory.Warnings...)
	warnings = normalizeAdvisoryStrings(warnings)
	riskFlags = normalizeAdvisoryStrings(riskFlags)
	metadata, err := safeMetadata(map[string]any{
		"advisory_type": "shadow_advisory",
		"scope":         "diagnostic_only",
	})
	if err != nil {
		return ShadowAdvisoryReport{}, err
	}
	return ShadowAdvisoryReport{
		ReportID:               fmt.Sprintf("shadow-advisory-%s", report.Observation.RequestID),
		GeneratedAt:            now,
		WorkspaceID:            report.Observation.WorkspaceID,
		RequestID:              report.Observation.RequestID,
		CorrelationID:          advisoryCorrelationID(report),
		SourceShadowReportRefs: refs.sourceShadowReportRefs,
		RouteMetadataRefs:      refs.routeMetadataRefs,
		ChatMetadataRefs:       refs.chatMetadataRefs,
		RetrievalMetadataRefs:  refs.retrievalMetadataRefs,
		EvidenceSummary: ShadowEvidenceSummary{
			RouteMetadataCount:     countPresent(report.RouteEnvelope),
			ChatMetadataCount:      countPresent(report.ChatMetadata),
			RetrievalMetadataCount: countPresent(report.RetrievalMetadata),
			SafeRefCount:           len(refs.safeRefs),
			MetadataOnly:           true,
			SafeRefs:               refs.safeRefs,
		},
		ConsensusAdvisory:       consensus,
		ContextCompilerAdvisory: contextAdvisory,
		RiskSummary: ShadowRiskSummary{
			RiskFlags:            riskFlags,
			WarningCount:         len(warnings),
			MetadataOnly:         true,
			NoRawContentVerified: true,
		},
		Warnings:         warnings,
		NoEffectVerified: report.Comparison.NoEffectVerified,
		Metadata:         metadata,
	}, nil
}

type advisoryRefs struct {
	sourceShadowReportRefs []string
	routeMetadataRefs      []string
	chatMetadataRefs       []string
	retrievalMetadataRefs  []string
	safeRefs               []string
}

func collectAdvisoryRefs(report DiagnosticReport) advisoryRefs {
	out := advisoryRefs{
		sourceShadowReportRefs: normalizeAdvisoryStrings([]string{
			report.Observation.ObservationID,
			report.Comparison.ReportID,
		}),
	}
	if report.RouteEnvelope != nil {
		out.routeMetadataRefs = normalizeAdvisoryStrings([]string{report.RouteEnvelope.ObservationID})
		out.safeRefs = append(out.safeRefs, safeAdvisoryRef(report.RouteEnvelope.RoutePattern))
	}
	if report.ChatMetadata != nil {
		out.chatMetadataRefs = normalizeAdvisoryStrings([]string{report.ChatMetadata.ObservationID})
		out.safeRefs = append(out.safeRefs, safeAdvisoryRef(report.ChatMetadata.ThreadID), safeAdvisoryRef(report.ChatMetadata.MessageID))
	}
	if report.RetrievalMetadata != nil {
		out.retrievalMetadataRefs = normalizeAdvisoryStrings([]string{report.RetrievalMetadata.ObservationID})
		out.safeRefs = append(out.safeRefs,
			safeAdvisoryRef(report.RetrievalMetadata.RetrievalRunID),
			safeAdvisoryRef(report.RetrievalMetadata.RetrievalResultID),
			safeAdvisoryRef(report.RetrievalMetadata.SourceRefID),
			safeAdvisoryRef(report.RetrievalMetadata.SourceHash),
		)
	}
	out.safeRefs = normalizeAdvisoryStrings(out.safeRefs)
	return out
}

func buildConsensusAdvisory(safeRefCount int) ShadowConsensusAdvisory {
	if safeRefCount == 0 {
		return ShadowConsensusAdvisory{
			Status:              "metadata_only_no_factual_claims",
			ProposedClaimCount:  0,
			AcceptedClaimCount:  0,
			RejectedClaimCount:  0,
			UncertainClaimCount: 0,
			Summary:             "Metadata is insufficient for factual claim acceptance.",
			Warnings:            []string{"metadata_only_no_factual_claims"},
		}
	}
	return ShadowConsensusAdvisory{
		Status:              "metadata_only_uncertain",
		ProposedClaimCount:  1,
		AcceptedClaimCount:  0,
		RejectedClaimCount:  0,
		UncertainClaimCount: 1,
		Summary:             "Safe metadata refs can form only an uncertain advisory claim.",
		Warnings:            []string{"no_canonical_truth_created"},
	}
}

func buildContextCompilerAdvisory(report DiagnosticReport, safeRefs []string) ShadowContextCompilerAdvisory {
	if len(safeRefs) == 0 {
		return ShadowContextCompilerAdvisory{
			Status:     "insufficient_safe_metadata",
			BlockCount: 0,
			Warnings:   []string{"shadow_context_not_compiled"},
		}
	}
	blockCount := 1
	if report.RetrievalMetadata != nil {
		blockCount++
	}
	if report.ChatMetadata != nil {
		blockCount++
	}
	return ShadowContextCompilerAdvisory{
		Status: "shadow_context_summary_created",
		BundleHash: advisoryHash(map[string]any{
			"request_id": report.Observation.RequestID,
			"safe_refs":  safeRefs,
			"blocks":     blockCount,
		}),
		BlockCount: blockCount,
		CacheEligibilitySummary: map[string]int{
			"cache_if_stable": blockCount,
			"do_not_cache":    0,
		},
	}
}

func advisoryCorrelationID(report DiagnosticReport) string {
	switch {
	case report.RouteEnvelope != nil:
		return report.RouteEnvelope.CorrelationID
	case report.ChatMetadata != nil:
		return report.ChatMetadata.CorrelationID
	case report.RetrievalMetadata != nil:
		return report.RetrievalMetadata.CorrelationID
	default:
		return ""
	}
}

func safeAdvisoryRef(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || containsUnsafeTerm(value) || containsRawContentTerm(value) || len(value) > maxMetadataStringLength {
		return ""
	}
	return value
}

func normalizeAdvisoryStrings(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, item := range in {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}

func countPresent[T any](value *T) int {
	if value == nil {
		return 0
	}
	return 1
}

func advisoryHash(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		data = []byte(fmt.Sprint(value))
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}
