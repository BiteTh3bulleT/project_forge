package forgek

import (
	"errors"
	"testing"

	"forge/projectforge/services/core/internal/forgek/kv"
	"forge/projectforge/services/core/internal/forgek/lymphatic"
	"forge/projectforge/services/core/internal/forgek/snapshots"
)

func grantLymphaticCapability(t *testing.T, kernel *Kernel, actorID, workspaceID, mutationScope string, syscalls ...string) {
	t.Helper()
	if err := kernel.Capabilities().Grant(Capability{
		CapabilityID:    "cap-lymph-" + actorID + "-" + workspaceID,
		SubjectID:       actorID,
		AllowedSyscalls: syscalls,
		WorkspaceScope:  []string{workspaceID},
		MutationScope:   mutationScope,
		AuditRequired:   true,
	}); err != nil {
		t.Fatalf("grant lymphatic capability: %v", err)
	}
}

func TestLymphaticSyscallsAreRegistered(t *testing.T) {
	kernel := testKernel()
	for _, name := range []string{
		SyscallLymphRunSweep,
		SyscallLymphGetReport,
		SyscallLymphListReports,
		SyscallLymphGetProposal,
		SyscallLymphListProposals,
		SyscallLymphCreateProposal,
		SyscallLymphRead,
	} {
		if _, ok := kernel.Syscalls().Lookup(name); !ok {
			t.Fatalf("expected lymphatic syscall %s to be registered", name)
		}
	}
	if kernel.Lymphatic() == nil {
		t.Fatal("kernel does not own lymphatic service")
	}
}

func TestLymphaticRunSweepRequiresCapabilityAndJournals(t *testing.T) {
	kernel := testKernel()
	denied := kernel.DispatchSyscall(SyscallRequest{Name: SyscallLymphRunSweep, ActorID: "denied", WorkspaceID: "workspace-a"})
	if denied.Success || !errors.Is(denied.Error, ErrCapabilityDenied) {
		t.Fatalf("expected sweep denial, got %#v", denied)
	}
	grantLymphaticCapability(t, kernel, "reader", "workspace-a", MutationScopeNone, SyscallLymphRead)
	readOnlyDenied := kernel.DispatchSyscall(SyscallRequest{Name: SyscallLymphRunSweep, ActorID: "reader", WorkspaceID: "workspace-a"})
	if readOnlyDenied.Success || !errors.Is(readOnlyDenied.Error, ErrCapabilityDenied) {
		t.Fatalf("expected read-only sweep denial, got %#v", readOnlyDenied)
	}

	grantSnapshotCapability(t, kernel, "snapshot-operator", "workspace-a", SyscallSnapshotCreate)
	snapshot := createSnapshot(t, kernel, "snapshot-operator", map[string]any{
		"snapshot_type":      string(snapshots.SnapshotTypeCaseSnapshot),
		"source_object_refs": []string{"missing-source"},
		"summary":            "orphan source ref",
	})
	grantLymphaticCapability(t, kernel, "operator", "workspace-a", MutationScopeCanonical, SyscallLymphRunSweep)
	result := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallLymphRunSweep,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
		Input:       map[string]any{"sweep_kinds": []string{string(lymphatic.SweepOrphanRef)}},
	})
	if !result.Success || result.JournalEvent == "" {
		t.Fatalf("lymph.run_sweep failed or did not journal: %#v", result)
	}
	report := result.Output.(lymphatic.MaintenanceReport)
	if report.WorkspaceID != "workspace-a" || !report.DryRun || report.IsCanonicalTruth() {
		t.Fatalf("unexpected report boundary: %#v", report)
	}
	if len(report.Findings) == 0 || report.Findings[0].FindingType != lymphatic.FindingOrphanedReference {
		t.Fatalf("expected orphan finding for snapshot %s, got %#v", snapshot.SnapshotID, report.Findings)
	}
	event := kernel.Journal().ListEvents()[len(kernel.Journal().ListEvents())-1]
	if event.EventType != JournalEventLymphaticSweepCompleted || event.SyscallName != SyscallLymphRunSweep {
		t.Fatalf("unexpected lymphatic journal event: %#v", event)
	}
	obj, ok := kernel.Objects().GetObject(report.ReportID)
	if !ok || obj.ObjectType != ObjectTypeMaintenanceReport || obj.State["executes_cleanup"] != false {
		t.Fatalf("report object not recorded as non-executing maintenance report: %#v ok=%v", obj, ok)
	}
}

func TestLymphaticReadSyscallsRequireReadCapabilityAndDoNotJournal(t *testing.T) {
	kernel := testKernel()
	grantLymphaticCapability(t, kernel, "operator", "workspace-a", MutationScopeCanonical, SyscallLymphRunSweep)
	reportResult := kernel.DispatchSyscall(SyscallRequest{Name: SyscallLymphRunSweep, ActorID: "operator", WorkspaceID: "workspace-a"})
	if !reportResult.Success {
		t.Fatalf("setup sweep failed: %#v", reportResult)
	}
	reportID := reportResult.Output.(lymphatic.MaintenanceReport).ReportID
	before := len(kernel.Journal().ListEvents())

	withoutRead := kernel.DispatchSyscall(SyscallRequest{Name: SyscallLymphGetReport, ActorID: "reader", WorkspaceID: "workspace-a", Input: map[string]any{"report_id": reportID}})
	if withoutRead.Success || !errors.Is(withoutRead.Error, ErrCapabilityDenied) {
		t.Fatalf("expected read denial, got %#v", withoutRead)
	}
	grantLymphaticCapability(t, kernel, "reader", "workspace-a", MutationScopeNone, SyscallLymphRead)
	getReport := kernel.DispatchSyscall(SyscallRequest{Name: SyscallLymphGetReport, ActorID: "reader", WorkspaceID: "workspace-a", Input: map[string]any{"report_id": reportID}})
	listReports := kernel.DispatchSyscall(SyscallRequest{Name: SyscallLymphListReports, ActorID: "reader", WorkspaceID: "workspace-a"})
	readAll := kernel.DispatchSyscall(SyscallRequest{Name: SyscallLymphRead, ActorID: "reader", WorkspaceID: "workspace-a"})
	if !getReport.Success || !listReports.Success || !readAll.Success {
		t.Fatalf("read syscalls failed: get=%#v list=%#v read=%#v", getReport, listReports, readAll)
	}
	if len(kernel.Journal().ListEvents()) != before {
		t.Fatal("read-only lymphatic syscalls journaled state")
	}
}

func TestLymphaticCreateProposalRequiresCapabilityAndDoesNotExecute(t *testing.T) {
	kernel := testKernel()
	grantCaseCapability(t, kernel, "case-operator", "workspace-a")
	open := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallCaseOpen,
		ActorID:     "case-operator",
		WorkspaceID: "workspace-a",
		Input:       map[string]any{"user_intent": "lymphatic proposal boundary", "summary": "lymphatic proposal boundary"},
	})
	if !open.Success {
		t.Fatalf("case.open failed: %#v", open)
	}
	beforeCase := open.Output.(CasePacket)

	denied := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallLymphCreateProposal,
		ActorID:     "denied",
		WorkspaceID: "workspace-a",
		Input:       map[string]any{"target_object_refs": []string{beforeCase.CaseID}, "reason": "review"},
	})
	if denied.Success || !errors.Is(denied.Error, ErrCapabilityDenied) {
		t.Fatalf("expected proposal denial, got %#v", denied)
	}
	grantLymphaticCapability(t, kernel, "operator", "workspace-a", MutationScopeCanonical, SyscallLymphCreateProposal)
	result := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallLymphCreateProposal,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
		CaseID:      beforeCase.CaseID,
		Input: map[string]any{
			"proposal_type":      string(lymphatic.ProposalNoOpReview),
			"target_object_refs": []string{beforeCase.CaseID},
			"reason":             "manual maintenance review",
		},
	})
	if !result.Success || result.JournalEvent == "" {
		t.Fatalf("lymph.create_proposal failed or did not journal: %#v", result)
	}
	proposal := result.Output.(lymphatic.CleanupProposal)
	if proposal.ExecutesCleanup() || !proposal.RequiresReview {
		t.Fatalf("proposal crossed execution boundary: %#v", proposal)
	}
	obj, ok := kernel.Objects().GetObject(proposal.ProposalID)
	if !ok || obj.ObjectType != ObjectTypeCleanupProposal || obj.State["executes_cleanup"] != false {
		t.Fatalf("proposal object missing or executable: %#v ok=%v", obj, ok)
	}
	afterCase, ok := kernel.objects.getCase(beforeCase.CaseID)
	if !ok || afterCase.Status != beforeCase.Status || len(afterCase.AdmittedExhibitRefs) != len(beforeCase.AdmittedExhibitRefs) {
		t.Fatalf("lymphatic proposal mutated case packet: before=%#v after=%#v ok=%v", beforeCase, afterCase, ok)
	}
}

func TestLymphaticSweepDoesNotMutateSnapshotsKVRuntimeOrCourt(t *testing.T) {
	kernel := testKernel()
	grantSnapshotCapability(t, kernel, "snapshot-operator", "workspace-a", SyscallSnapshotCreate)
	snapshot := createSnapshot(t, kernel, "snapshot-operator", map[string]any{
		"snapshot_type":      string(snapshots.SnapshotTypeCaseSnapshot),
		"source_object_refs": []string{"case-a"},
		"summary":            "stable snapshot",
	})
	compiled := compileBundleForKV(t, kernel, "compiler")
	grantKVCapability(t, kernel, "kv-operator", "workspace-a", MutationScopeCanonical, SyscallKVRegister, SyscallKVInvalidate)
	manifest := registerKVManifest(t, kernel, "kv-operator", compiled.Bundle)
	invalidate := kernel.DispatchSyscall(SyscallRequest{Name: SyscallKVInvalidate, ActorID: "kv-operator", WorkspaceID: "workspace-a", Input: map[string]any{"cache_id": manifest.CacheID, "reason": "test invalidation"}})
	if !invalidate.Success {
		t.Fatalf("kv.invalidate failed: %#v", invalidate)
	}

	beforeSnapshot, _ := kernel.Snapshots().GetSnapshot(snapshot.SnapshotID)
	beforeManifest, _ := kernel.KV().GetManifest(manifest.CacheID)
	beforeJournal := len(kernel.Journal().ListEvents())

	grantLymphaticCapability(t, kernel, "operator", "workspace-a", MutationScopeCanonical, SyscallLymphRunSweep)
	result := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallLymphRunSweep,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
		Input:       map[string]any{"sweep_kinds": []string{string(lymphatic.SweepKVHygiene), string(lymphatic.SweepSnapshotHygiene)}},
	})
	if !result.Success {
		t.Fatalf("lymph.run_sweep failed: %#v", result)
	}
	afterSnapshot, _ := kernel.Snapshots().GetSnapshot(snapshot.SnapshotID)
	afterManifest, _ := kernel.KV().GetManifest(manifest.CacheID)
	if beforeSnapshot.Status != afterSnapshot.Status || beforeManifest.Status != afterManifest.Status || beforeManifest.ReuseCount != afterManifest.ReuseCount {
		t.Fatalf("lymphatic sweep mutated sources: before snapshot=%#v after=%#v before kv=%#v after=%#v", beforeSnapshot, afterSnapshot, beforeManifest, afterManifest)
	}
	if len(kernel.Journal().ListEvents()) != beforeJournal+1 {
		t.Fatal("lymphatic sweep should add only its own journal event")
	}
	report := result.Output.(lymphatic.MaintenanceReport)
	if len(report.CleanupProposals) == 0 || report.CleanupProposals[0].ProposedSyscallName != "kv.evict" {
		t.Fatalf("expected KV cleanup proposal without mutation, got %#v", report.CleanupProposals)
	}
	if beforeManifest.Status != kv.StatusInvalidated {
		t.Fatalf("setup did not create invalidated manifest: %#v", beforeManifest)
	}
}
