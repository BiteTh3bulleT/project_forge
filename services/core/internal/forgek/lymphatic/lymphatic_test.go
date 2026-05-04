package lymphatic

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"forge/projectforge/services/core/internal/forgek/court"
	"forge/projectforge/services/core/internal/forgek/kv"
	forgekRuntime "forge/projectforge/services/core/internal/forgek/runtime"
	"forge/projectforge/services/core/internal/forgek/snapshots"
)

func TestDefaultPolicyIsDeterministicDryRunAndRejectsSecrets(t *testing.T) {
	policy := DefaultPolicy("workspace-a", testTime())
	if !policy.DryRun || policy.WorkspaceID != "workspace-a" || len(policy.SweepKindsEnabled) != len(AllSweepKinds()) {
		t.Fatalf("unexpected default policy: %#v", policy)
	}
	if err := ValidatePolicy(policy); err != nil {
		t.Fatalf("valid policy rejected: %v", err)
	}
	again := DefaultPolicy("workspace-a", testTime())
	if PolicyHashForTest(policy) != PolicyHashForTest(again) {
		t.Fatalf("policy hash changed for stable input")
	}
	policy.Metadata = map[string]any{"secret_token": "nope"}
	if !errors.Is(ValidatePolicy(policy), ErrInvalidPolicy) {
		t.Fatal("expected secret metadata rejection")
	}
}

func TestSnapshotHygieneDetectsSupersededExpiredAndOrphanRefsWithoutMutation(t *testing.T) {
	oldSnapshot := testSnapshot("snapshot-old", snapshots.StatusSuperseded)
	sealed := testSnapshot("snapshot-sealed", snapshots.StatusSealed)
	before := sealed.Clone()
	request := testRequest(SweepSnapshotHygiene, SweepOrphanRef)
	report, err := NewService().RunSweep(request, testPolicy(), SweepSources{
		Snapshots:       []snapshots.Snapshot{oldSnapshot, sealed},
		KnownObjectRefs: []string{"snapshot-old", "snapshot-sealed"},
		Now:             testTime(),
	})
	if err != nil {
		t.Fatalf("run sweep: %v", err)
	}
	if !hasFinding(report, FindingSupersededSnapshot) || !hasFinding(report, FindingExpiredSnapshotCandidate) || !hasFinding(report, FindingOrphanedReference) {
		t.Fatalf("expected snapshot findings, got %#v", report.Findings)
	}
	if !reflect.DeepEqual(before, sealed) {
		t.Fatal("snapshot hygiene mutated source snapshot")
	}
	for _, proposal := range report.CleanupProposals {
		if !proposal.RequiresReview || proposal.ExecutesCleanup() {
			t.Fatalf("proposal executed or skipped review: %#v", proposal)
		}
	}
}

func TestKVRuntimeAndContradictionSweepsAreProposalOnly(t *testing.T) {
	request := testRequest(SweepKVHygiene, SweepRuntimeResult, SweepContradiction)
	manifest := testManifest()
	result := forgekRuntime.RuntimeGenerateResult{
		ResultID:              "runtime-result-a",
		RequestID:             "runtime-request-a",
		WorkspaceID:           "workspace-a",
		CaseID:                "case-a",
		BundleID:              "bundle-a",
		Warnings:              []string{"proposal needs review"},
		CreatedAt:             testTime().Add(-48 * time.Hour),
		AuthorityLevel:        forgekRuntime.RuntimeAuthorityProposalOnly,
		IsCanonicalTruth:      false,
		IsAdmittedEvidence:    false,
		IsModelDriverProposal: true,
	}
	contradiction := court.Contradiction{
		ContradictionID: "contradiction-a",
		WorkspaceID:     "workspace-a",
		CaseID:          "case-a",
		ExhibitAID:      "exhibit-a",
		ExhibitBID:      "exhibit-b",
		Status:          court.ContradictionOpen,
		CreatedAt:       testTime(),
	}
	report, err := NewService().RunSweep(request, testPolicy(), SweepSources{
		KVManifests:     []kv.KVCacheManifest{manifest},
		RuntimeResults:  []forgekRuntime.RuntimeGenerateResult{result},
		Contradictions:  []court.Contradiction{contradiction},
		KnownObjectRefs: []string{"cache-a", "runtime-result-a", "contradiction-a", "bundle-a", "exhibit-a", "exhibit-b"},
		Now:             testTime(),
	})
	if err != nil {
		t.Fatalf("run sweep: %v", err)
	}
	for _, findingType := range []FindingType{FindingInvalidatedKVManifest, FindingStaleRuntimeResult, FindingOpenContradiction} {
		if !hasFinding(report, findingType) {
			t.Fatalf("missing finding %s in %#v", findingType, report.Findings)
		}
	}
	for _, proposal := range report.CleanupProposals {
		if proposal.ExecutesCleanup() || !proposal.RequiresReview {
			t.Fatalf("proposal crossed cleanup boundary: %#v", proposal)
		}
	}
}

func TestSweepIsDeterministicForStableInputs(t *testing.T) {
	service := NewService()
	request := testRequest(SweepSnapshotHygiene, SweepOrphanRef)
	sources := SweepSources{
		Snapshots:       []snapshots.Snapshot{testSnapshot("snapshot-b", snapshots.StatusSealed), testSnapshot("snapshot-a", snapshots.StatusSuperseded)},
		KnownObjectRefs: []string{"snapshot-a", "snapshot-b"},
		Now:             testTime(),
	}
	first, err := service.RunSweep(request, testPolicy(), sources)
	if err != nil {
		t.Fatalf("first sweep: %v", err)
	}
	sources.Snapshots[0], sources.Snapshots[1] = sources.Snapshots[1], sources.Snapshots[0]
	second, err := service.RunSweep(request, testPolicy(), sources)
	if err != nil {
		t.Fatalf("second sweep: %v", err)
	}
	if first.ReportID != second.ReportID || !reflect.DeepEqual(first.Findings, second.Findings) || !reflect.DeepEqual(first.CleanupProposals, second.CleanupProposals) {
		t.Fatalf("sweep was not deterministic:\nfirst=%#v\nsecond=%#v", first, second)
	}
}

func TestServiceStoresClonesAndManualProposal(t *testing.T) {
	service := NewService()
	report, err := service.RunSweep(testRequest(SweepKVHygiene), testPolicy(), SweepSources{KVManifests: []kv.KVCacheManifest{testManifest()}, KnownObjectRefs: []string{"cache-a", "bundle-a"}, Now: testTime()})
	if err != nil {
		t.Fatalf("run sweep: %v", err)
	}
	report.Findings[0].Reason = "mutated outside service"
	stored, ok := service.GetReport(report.ReportID)
	if !ok || stored.Findings[0].Reason == "mutated outside service" {
		t.Fatalf("service did not clone report storage: %#v", stored)
	}
	proposal, err := service.CreateProposal(CleanupProposal{
		ProposalID:       "proposal-manual",
		ProposalType:     ProposalNoOpReview,
		WorkspaceID:      "workspace-a",
		CaseID:           "case-a",
		TargetObjectRefs: []string{"cache-a"},
		Reason:           "manual review",
		RequiresReview:   true,
		CreatedAt:        testTime(),
	})
	if err != nil {
		t.Fatalf("create proposal: %v", err)
	}
	proposal.Reason = "mutated proposal"
	storedProposal, ok := service.GetProposal("proposal-manual")
	if !ok || storedProposal.Reason == "mutated proposal" {
		t.Fatalf("service did not clone proposal storage: %#v", storedProposal)
	}
}

func TestInvalidRequestRejected(t *testing.T) {
	request := testRequest(SweepSnapshotHygiene)
	request.WorkspaceID = ""
	if !errors.Is(ValidateSweepRequest(request, testPolicy()), ErrInvalidSweepRequest) {
		t.Fatal("expected missing workspace rejection")
	}
	request = testRequest("UNKNOWN")
	if !errors.Is(ValidateSweepRequest(request, testPolicy()), ErrInvalidSweepKind) {
		t.Fatal("expected invalid sweep kind rejection")
	}
}

func testRequest(kinds ...SweepKind) LymphaticSweepRequest {
	return LymphaticSweepRequest{
		RequestID:   "sweep-a",
		WorkspaceID: "workspace-a",
		CaseID:      "case-a",
		SweepKinds:  kinds,
		DryRun:      true,
		RequestedBy: "unit-test",
		CreatedAt:   testTime(),
	}
}

func testPolicy() LymphaticPolicy {
	policy := DefaultPolicy("workspace-a", testTime())
	policy.SnapshotExpireAfterDurationMS = int64(time.Hour / time.Millisecond)
	policy.RuntimeResultStaleAfterMS = int64(time.Hour / time.Millisecond)
	return policy
}

func testSnapshot(id string, status snapshots.SnapshotStatus) snapshots.Snapshot {
	sealedAt := testTime().Add(-2 * time.Hour)
	s := snapshots.Snapshot{
		SnapshotID:       id,
		SnapshotType:     snapshots.SnapshotTypeCaseSnapshot,
		WorkspaceID:      "workspace-a",
		CaseID:           "case-a",
		Status:           status,
		SourceObjectRefs: []string{"missing-source"},
		ShapeHash:        "shape-" + id,
		SourceHash:       "source-" + id,
		CreatedBy:        "unit-test",
		CreatedAt:        sealedAt,
		SealedAt:         &sealedAt,
	}
	return s
}

func testManifest() kv.KVCacheManifest {
	return kv.KVCacheManifest{
		CacheID:           "cache-a",
		CacheMode:         kv.ModeStrictPrefix,
		WorkspaceID:       "workspace-a",
		CaseID:            "case-a",
		BundleID:          "bundle-a",
		ModelID:           "model-a",
		ModelRevision:     "rev-a",
		TokenizerID:       "tokenizer-a",
		TokenizerRevision: "tok-rev-a",
		ChatTemplateHash:  "template-a",
		PromptLayoutHash:  "layout-a",
		PolicySchemaHash:  "policy-a",
		SyscallSchemaHash: "syscall-a",
		TokenInputHash:    "token-a",
		RuntimeBackend:    "simulator",
		RuntimeVersion:    "v1",
		AttentionBackend:  "attention-a",
		RopeConfigHash:    "rope-a",
		KVPrecision:       "fp16",
		MemoryTier:        kv.TierDiskCold,
		CacheSalt:         "salt-a",
		Status:            kv.StatusInvalidated,
		CreatedAt:         testTime().Add(-2 * time.Hour),
	}
}

func hasFinding(report MaintenanceReport, findingType FindingType) bool {
	for _, finding := range report.Findings {
		if finding.FindingType == findingType {
			return true
		}
	}
	return false
}

func PolicyHashForTest(policy LymphaticPolicy) string {
	return StableHash(NormalizePolicy(policy))
}

func testTime() time.Time {
	return time.Date(2026, 5, 4, 15, 0, 0, 0, time.UTC)
}
