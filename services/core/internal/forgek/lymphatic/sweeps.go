package lymphatic

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"forge/projectforge/services/core/internal/forgek/court"
	"forge/projectforge/services/core/internal/forgek/kv"
	forgekRuntime "forge/projectforge/services/core/internal/forgek/runtime"
	"forge/projectforge/services/core/internal/forgek/snapshots"
)

func BuildReport(request LymphaticSweepRequest, policy LymphaticPolicy, sources SweepSources) MaintenanceReport {
	findings := make([]LymphaticFinding, 0)
	proposals := make([]CleanupProposal, 0)
	warnings := make([]string, 0)

	for _, kind := range request.SweepKinds {
		switch kind {
		case SweepSnapshotHygiene:
			f, p := RunSnapshotHygiene(request, policy, sources)
			findings = append(findings, f...)
			proposals = append(proposals, p...)
		case SweepKVHygiene:
			f, p := RunKVHygiene(request, policy, sources)
			findings = append(findings, f...)
			proposals = append(proposals, p...)
		case SweepRuntimeResult:
			f, p := RunRuntimeResultHygiene(request, policy, sources)
			findings = append(findings, f...)
			proposals = append(proposals, p...)
		case SweepContradiction:
			f, p := RunContradictionSweep(request, policy, sources)
			findings = append(findings, f...)
			proposals = append(proposals, p...)
		case SweepOrphanRef:
			f, p := RunOrphanRefSweep(request, policy, sources)
			findings = append(findings, f...)
			proposals = append(proposals, p...)
		default:
			warnings = append(warnings, string(kind)+" declared but not implemented in Phase 10")
		}
	}

	findings = SortFindings(findings)
	proposals = SortProposals(proposals)
	if policy.MaxReportItems > 0 {
		if len(findings) > policy.MaxReportItems {
			findings = findings[:policy.MaxReportItems]
			warnings = append(warnings, "findings truncated by max_report_items")
		}
		if len(proposals) > policy.MaxReportItems {
			proposals = proposals[:policy.MaxReportItems]
			warnings = append(warnings, "proposals truncated by max_report_items")
		}
	}
	report := MaintenanceReport{
		ReportID:         reportID(request, policy, findings, proposals),
		WorkspaceID:      request.WorkspaceID,
		CaseID:           request.CaseID,
		SweepKinds:       request.SweepKinds,
		PolicyID:         policy.PolicyID,
		DryRun:           true,
		Status:           ReportComplete,
		Findings:         findings,
		CleanupProposals: proposals,
		Warnings:         NormalizeRefs(warnings),
		Summary:          reportSummary(findings, proposals),
		CreatedBy:        request.RequestedBy,
		CreatedAt:        request.CreatedAt,
		Metadata: CloneMap(map[string]any{
			"simulator_only":       true,
			"executes_cleanup":     false,
			"is_canonical_truth":   false,
			"is_admitted_evidence": false,
		}),
	}
	return report.Clone()
}

func RunSnapshotHygiene(request LymphaticSweepRequest, policy LymphaticPolicy, sources SweepSources) ([]LymphaticFinding, []CleanupProposal) {
	findings := make([]LymphaticFinding, 0)
	proposals := make([]CleanupProposal, 0)
	for _, snapshot := range sources.Snapshots {
		if !matchesScope(snapshot.WorkspaceID, snapshot.CaseID, request.WorkspaceID, request.CaseID) {
			continue
		}
		switch snapshot.Status {
		case snapshots.StatusSuperseded:
			if policy.IncludeSupersededObjects {
				f := newFinding(request, FindingSupersededSnapshot, SeverityMedium, []string{snapshot.SnapshotID}, snapshot.AllRefs(), "snapshot is superseded and remains inspectable", ProposalNoOpReview)
				findings = append(findings, f)
				proposals = append(proposals, proposalForFinding(f, ProposalNoOpReview, "", nil))
			}
		case snapshots.StatusExpired:
			f := newFinding(request, FindingExpiredSnapshotCandidate, SeverityInfo, []string{snapshot.SnapshotID}, snapshot.AllRefs(), "snapshot is already expired and remains inspectable", ProposalNoOpReview)
			findings = append(findings, f)
			proposals = append(proposals, proposalForFinding(f, ProposalNoOpReview, "", nil))
		case snapshots.StatusSealed, snapshots.StatusRestoreSeedCreated:
			if olderThan(snapshotTime(snapshot), sources.Now, policy.SnapshotExpireAfterDurationMS) {
				f := newFinding(request, FindingExpiredSnapshotCandidate, SeverityLow, []string{snapshot.SnapshotID}, snapshot.AllRefs(), "sealed snapshot exceeds expiration candidate threshold", ProposalExpireSnapshot)
				findings = append(findings, f)
				proposals = append(proposals, proposalForFinding(f, ProposalExpireSnapshot, "snapshot.expire", map[string]any{"snapshot_id": snapshot.SnapshotID, "reason": "lymphatic_expiration_candidate"}))
			}
		}
	}
	return findings, proposals
}

func RunKVHygiene(request LymphaticSweepRequest, policy LymphaticPolicy, sources SweepSources) ([]LymphaticFinding, []CleanupProposal) {
	findings := make([]LymphaticFinding, 0)
	proposals := make([]CleanupProposal, 0)
	for _, manifest := range sources.KVManifests {
		if !matchesScope(manifest.WorkspaceID, manifest.CaseID, request.WorkspaceID, request.CaseID) {
			continue
		}
		refs := manifest.AllRefs()
		switch manifest.Status {
		case kv.StatusInvalidated:
			if policy.IncludeInvalidatedKV {
				f := newFinding(request, FindingInvalidatedKVManifest, SeverityMedium, []string{manifest.CacheID}, refs, "KV manifest is invalidated and cannot be reused", ProposalEvictKV)
				findings = append(findings, f)
				proposals = append(proposals, proposalForFinding(f, ProposalEvictKV, "kv.evict", map[string]any{"cache_id": manifest.CacheID, "reason": "lymphatic_invalidated_manifest"}))
			}
		case kv.StatusEvicted, kv.StatusExpired:
			f := newFinding(request, FindingEvictableKVManifest, SeverityInfo, []string{manifest.CacheID}, refs, "KV manifest is not hit eligible and remains inspectable", ProposalNoOpReview)
			findings = append(findings, f)
			proposals = append(proposals, proposalForFinding(f, ProposalNoOpReview, "", nil))
		default:
			if manifest.ReuseCount <= policy.KvColdAfterReuseCount && (manifest.MemoryTier == kv.TierDiskCold || manifest.MemoryTier == kv.TierRemoteCold || manifest.MemoryTier == kv.TierNone) {
				f := newFinding(request, FindingEvictableKVManifest, SeverityLow, []string{manifest.CacheID}, refs, "KV manifest is cold and cleanup-eligible metadata", ProposalDemoteKVTier)
				findings = append(findings, f)
				proposals = append(proposals, proposalForFinding(f, ProposalDemoteKVTier, "kv.demote", map[string]any{"cache_id": manifest.CacheID}))
			}
		}
		if olderThan(manifest.CreatedAt, sources.Now, policy.KvExpireAfterDurationMS) {
			f := newFinding(request, FindingEvictableKVManifest, SeverityLow, []string{manifest.CacheID}, refs, "KV manifest exceeds expiration candidate threshold", ProposalEvictKV)
			findings = append(findings, f)
			proposals = append(proposals, proposalForFinding(f, ProposalEvictKV, "kv.evict", map[string]any{"cache_id": manifest.CacheID, "reason": "lymphatic_expiration_candidate"}))
		}
	}
	return findings, proposals
}

func RunRuntimeResultHygiene(request LymphaticSweepRequest, policy LymphaticPolicy, sources SweepSources) ([]LymphaticFinding, []CleanupProposal) {
	findings := make([]LymphaticFinding, 0)
	proposals := make([]CleanupProposal, 0)
	for _, result := range sources.RuntimeResults {
		if !matchesScope(result.WorkspaceID, result.CaseID, request.WorkspaceID, request.CaseID) {
			continue
		}
		refs := runtimeRefs(result)
		if olderThan(result.CreatedAt, sources.Now, policy.RuntimeResultStaleAfterMS) || result.Error != "" || len(result.Warnings) > 0 || result.FinishReason == forgekRuntime.FinishError {
			reason := "runtime result is proposal evidence requiring maintenance review"
			if olderThan(result.CreatedAt, sources.Now, policy.RuntimeResultStaleAfterMS) {
				reason = "runtime result exceeds stale-result threshold"
			}
			f := newFinding(request, FindingStaleRuntimeResult, SeverityLow, []string{result.ResultID}, refs, reason, ProposalMarkRuntimeResultStale)
			findings = append(findings, f)
			proposals = append(proposals, proposalForFinding(f, ProposalMarkRuntimeResultStale, "", nil))
		}
	}
	return findings, proposals
}

func RunContradictionSweep(request LymphaticSweepRequest, _ LymphaticPolicy, sources SweepSources) ([]LymphaticFinding, []CleanupProposal) {
	findings := make([]LymphaticFinding, 0)
	proposals := make([]CleanupProposal, 0)
	for _, contradiction := range sources.Contradictions {
		if !matchesScope(contradiction.WorkspaceID, contradiction.CaseID, request.WorkspaceID, request.CaseID) || contradiction.Status != court.ContradictionOpen {
			continue
		}
		refs := NormalizeRefs([]string{contradiction.ExhibitAID, contradiction.ExhibitBID, contradiction.ClaimAID, contradiction.ClaimBID})
		f := newFinding(request, FindingOpenContradiction, SeverityHigh, []string{contradiction.ContradictionID}, refs, "open contradiction requires review; sweep does not resolve or merge it", ProposalRegisterContradictionReview)
		findings = append(findings, f)
		proposals = append(proposals, proposalForFinding(f, ProposalRegisterContradictionReview, "", nil))
	}
	return findings, proposals
}

func RunOrphanRefSweep(request LymphaticSweepRequest, _ LymphaticPolicy, sources SweepSources) ([]LymphaticFinding, []CleanupProposal) {
	known := knownRefs(sources)
	candidates := referencedObjectCandidates(sources)
	findings := make([]LymphaticFinding, 0)
	proposals := make([]CleanupProposal, 0)
	for owner, refs := range candidates {
		for _, ref := range refs {
			if _, ok := known[ref]; ok {
				continue
			}
			f := newFinding(request, FindingOrphanedReference, SeverityMedium, []string{owner, ref}, []string{owner}, "referenced object is not present in simulator registry", ProposalRepairOrphanRef)
			findings = append(findings, f)
			proposals = append(proposals, proposalForFinding(f, ProposalRepairOrphanRef, "", nil))
		}
	}
	return findings, proposals
}

func newFinding(request LymphaticSweepRequest, findingType FindingType, severity FindingSeverity, objectRefs, sourceRefs []string, reason string, action ProposalType) LymphaticFinding {
	finding := LymphaticFinding{
		FindingType:       findingType,
		Severity:          severity,
		WorkspaceID:       request.WorkspaceID,
		CaseID:            request.CaseID,
		ObjectRefs:        NormalizeRefs(objectRefs),
		SourceRefs:        NormalizeRefs(sourceRefs),
		Reason:            reason,
		RecommendedAction: action,
		CreatedAt:         request.CreatedAt,
	}
	finding.FindingID = "lymph-finding-" + StableHash(map[string]any{
		"finding_type": finding.FindingType,
		"workspace_id": finding.WorkspaceID,
		"case_id":      finding.CaseID,
		"object_refs":  finding.ObjectRefs,
		"source_refs":  finding.SourceRefs,
		"reason":       finding.Reason,
		"action":       finding.RecommendedAction,
	})[:16]
	return finding
}

func proposalForFinding(finding LymphaticFinding, proposalType ProposalType, syscallName string, payload map[string]any) CleanupProposal {
	proposal := CleanupProposal{
		ProposalType:        proposalType,
		WorkspaceID:         finding.WorkspaceID,
		CaseID:              finding.CaseID,
		TargetObjectRefs:    finding.ObjectRefs,
		ProposedSyscallName: syscallName,
		ProposedPayload:     CloneMap(payload),
		Reason:              finding.Reason,
		SafetyNotes: NormalizeRefs([]string{
			"dry_run_only",
			"requires_explicit_followup_syscall",
			"does_not_mutate_source_objects",
		}),
		RequiresReview: true,
		CreatedAt:      finding.CreatedAt,
		Metadata: map[string]any{
			"finding_id": finding.FindingID,
		},
	}
	proposal.ProposalID = "lymph-proposal-" + StableHash(map[string]any{
		"proposal_type": proposal.ProposalType,
		"workspace_id":  proposal.WorkspaceID,
		"case_id":       proposal.CaseID,
		"targets":       proposal.TargetObjectRefs,
		"syscall":       proposal.ProposedSyscallName,
		"payload":       proposal.ProposedPayload,
		"reason":        proposal.Reason,
	})[:16]
	return proposal
}

func reportID(request LymphaticSweepRequest, policy LymphaticPolicy, findings []LymphaticFinding, proposals []CleanupProposal) string {
	return "lymph-report-" + StableHash(map[string]any{
		"request_id": request.RequestID,
		"workspace":  request.WorkspaceID,
		"case_id":    request.CaseID,
		"sweeps":     request.SweepKinds,
		"policy":     policy.PolicyID,
		"findings":   findingIDs(findings),
		"proposals":  proposalIDs(proposals),
	})[:16]
}

func reportSummary(findings []LymphaticFinding, proposals []CleanupProposal) string {
	return fmt.Sprintf("dry-run lymphatic sweep found %d finding(s) and %d cleanup proposal(s)", len(findings), len(proposals))
}

func knownRefs(sources SweepSources) map[string]struct{} {
	known := make(map[string]struct{})
	for _, ref := range sources.KnownObjectRefs {
		if ref != "" {
			known[ref] = struct{}{}
		}
	}
	for _, snapshot := range sources.Snapshots {
		known[snapshot.SnapshotID] = struct{}{}
	}
	for _, manifest := range sources.KVManifests {
		known[manifest.CacheID] = struct{}{}
	}
	for _, result := range sources.RuntimeResults {
		known[result.ResultID] = struct{}{}
	}
	for _, contradiction := range sources.Contradictions {
		known[contradiction.ContradictionID] = struct{}{}
	}
	for _, bundle := range sources.ContextBundles {
		known[bundle.BundleID] = struct{}{}
	}
	for _, block := range sources.ContextBlocks {
		known[block.BlockID] = struct{}{}
	}
	return known
}

func referencedObjectCandidates(sources SweepSources) map[string][]string {
	candidates := make(map[string][]string)
	for _, snapshot := range sources.Snapshots {
		candidates[snapshot.SnapshotID] = NormalizeRefs(append(append(append(append(append(append(append(append(append([]string{}, snapshot.SourceObjectRefs...), snapshot.PalaceRouteRefs...), snapshot.SubmittedObjectRefs...), snapshot.AdmittedObjectRefs...), snapshot.RejectedObjectRefs...), snapshot.SemanticOperationRefs...), snapshot.ContradictionRefs...), snapshot.ContextBlockRefs...), snapshot.KVManifestRefs...))
	}
	for _, manifest := range sources.KVManifests {
		candidates[manifest.CacheID] = NormalizeRefs([]string{manifest.BundleID, manifest.BlockID, manifest.SnapshotID, manifest.RestoreSeedID})
	}
	for _, result := range sources.RuntimeResults {
		candidates[result.ResultID] = NormalizeRefs(append([]string{result.BundleID, result.KVCacheID}, result.ProvenanceRefs...))
	}
	for _, bundle := range sources.ContextBundles {
		refs := []string{bundle.SnapshotID, bundle.RestoreSeedID}
		for _, block := range bundle.Blocks {
			refs = append(refs, block.BlockID)
		}
		candidates[bundle.BundleID] = NormalizeRefs(refs)
	}
	for _, block := range sources.ContextBlocks {
		candidates[block.BlockID] = NormalizeRefs(block.AllRefs())
	}
	return candidates
}

func runtimeRefs(result forgekRuntime.RuntimeGenerateResult) []string {
	return NormalizeRefs(append([]string{result.RequestID, result.DriverID, result.BundleID, result.KVLookupID, result.KVCacheID, result.ProposalObjectRef}, result.ProvenanceRefs...))
}

func snapshotTime(snapshot snapshots.Snapshot) time.Time {
	if snapshot.SealedAt != nil {
		return *snapshot.SealedAt
	}
	return snapshot.CreatedAt
}

func olderThan(createdAt, now time.Time, thresholdMS int64) bool {
	if thresholdMS <= 0 || createdAt.IsZero() || now.IsZero() {
		return false
	}
	return !createdAt.After(now.Add(-time.Duration(thresholdMS) * time.Millisecond))
}

func matchesScope(workspaceID, caseID, requestWorkspaceID, requestCaseID string) bool {
	if workspaceID != requestWorkspaceID {
		return false
	}
	if requestCaseID != "" && caseID != "" && caseID != requestCaseID {
		return false
	}
	return true
}

func findingIDs(findings []LymphaticFinding) []string {
	out := make([]string, 0, len(findings))
	for _, finding := range findings {
		out = append(out, finding.FindingID)
	}
	sort.Strings(out)
	return out
}

func proposalIDs(proposals []CleanupProposal) []string {
	out := make([]string, 0, len(proposals))
	for _, proposal := range proposals {
		out = append(out, proposal.ProposalID)
	}
	sort.Strings(out)
	return out
}

func stringsTrimSpace(value string) string {
	return strings.TrimSpace(value)
}
