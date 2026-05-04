package forgek

import (
	"errors"
	"testing"

	"forge/projectforge/services/core/internal/forgek/contextcompiler"
	"forge/projectforge/services/core/internal/forgek/snapshots"
)

func grantContextCapability(t *testing.T, kernel *Kernel, actorID string, workspaceID string, mutationScope string, syscalls ...string) {
	t.Helper()
	if err := kernel.Capabilities().Grant(Capability{
		CapabilityID:    "cap-context-" + actorID + "-" + workspaceID,
		SubjectID:       actorID,
		AllowedSyscalls: syscalls,
		WorkspaceScope:  []string{workspaceID},
		MutationScope:   mutationScope,
		AuditRequired:   true,
	}); err != nil {
		t.Fatalf("grant context capability: %v", err)
	}
}

func compileContext(t *testing.T, kernel *Kernel, actorID string, input map[string]any) contextcompiler.ContextCompileResult {
	t.Helper()
	result := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallContextCompile,
		ActorID:     actorID,
		WorkspaceID: "workspace-a",
		Input:       input,
	})
	if !result.Success {
		t.Fatalf("context.compile failed: %v", result.Error)
	}
	return result.Output.(contextcompiler.ContextCompileResult)
}

func TestContextSyscallsAreRegistered(t *testing.T) {
	kernel := testKernel()
	for _, name := range []string{
		SyscallContextCompile,
		SyscallContextCompileFromSnapshot,
		SyscallContextCompileFromRestoreSeed,
		SyscallContextGetBundle,
		SyscallContextListBundles,
		SyscallContextGetBlock,
		SyscallContextListBlocks,
		SyscallContextValidateLayout,
		SyscallContextHash,
	} {
		if _, ok := kernel.Syscalls().Lookup(name); !ok {
			t.Fatalf("expected context syscall %s to be registered", name)
		}
	}
	if kernel.ContextCompiler() == nil {
		t.Fatal("kernel does not own context compiler service")
	}
}

func TestContextCompileRequiresCapabilityAndJournals(t *testing.T) {
	kernel := testKernel()
	input := map[string]any{
		"case_id":               "case-a",
		"source_object_refs":    []string{"case-a"},
		"admitted_exhibit_refs": []string{"exhibit-a"},
		"current_task_summary":  "compile deterministic context",
	}
	denied := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallContextCompile,
		ActorID:     "denied",
		WorkspaceID: "workspace-a",
		Input:       input,
	})
	if denied.Success || !errors.Is(denied.Error, ErrCapabilityDenied) {
		t.Fatalf("expected capability denial, got %#v", denied)
	}

	grantContextCapability(t, kernel, "reader", "workspace-a", MutationScopeNone, SyscallContextRead)
	readOnlyDenied := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallContextCompile,
		ActorID:     "reader",
		WorkspaceID: "workspace-a",
		Input:       input,
	})
	if readOnlyDenied.Success || !errors.Is(readOnlyDenied.Error, ErrCapabilityDenied) {
		t.Fatalf("expected read-only actor denial, got %#v", readOnlyDenied)
	}

	grantContextCapability(t, kernel, "operator", "workspace-a", MutationScopeCanonical, SyscallContextCompile)
	compiled := compileContext(t, kernel, "operator", input)
	if compiled.Bundle.BundleHash == "" || compiled.Bundle.TokenInputHash == "" {
		t.Fatalf("compiled bundle missing hashes: %#v", compiled.Bundle)
	}
	obj, ok := kernel.Objects().GetObject(compiled.Bundle.BundleID)
	if !ok || obj.ObjectType != ObjectTypeContextBundle || obj.AuthorityLevel != AuthorityCompiled {
		t.Fatalf("context bundle object not registered as compiled shape: %#v", obj)
	}
	if obj.State["is_canonical_truth"] != false || obj.State["is_model_response"] != false || obj.State["is_deterministic_kv_cache"] != false {
		t.Fatalf("context bundle claimed wrong authority: %#v", obj.State)
	}
	events := kernel.Journal().ListEvents()
	if len(events) != 1 || events[0].EventType != JournalEventContextCompiled || events[0].SyscallName != SyscallContextCompile {
		t.Fatalf("context.compile was not journaled: %#v", events)
	}
}

func TestContextReadSyscallsRequireReadCapabilityAndDoNotJournal(t *testing.T) {
	kernel := testKernel()
	grantContextCapability(t, kernel, "operator", "workspace-a", MutationScopeCanonical, SyscallContextCompile)
	compiled := compileContext(t, kernel, "operator", map[string]any{
		"case_id":                   "case-a",
		"source_object_refs":        []string{"case-a"},
		"admitted_exhibit_refs":     []string{"exhibit-a"},
		"semantic_operation_refs":   []string{"operation-a"},
		"include_contradictions":    true,
		"contradiction_refs":        []string{"contradiction-a"},
		"current_task_summary":      "read deterministic context",
		"include_restore_seed":      false,
		"layout_version":            "phase-7-test",
		"syscall_schema_version":    "v1",
		"include_rejected_evidence": false,
	})
	before := len(kernel.Journal().ListEvents())

	withoutRead := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallContextGetBundle,
		ActorID:     "reader",
		WorkspaceID: "workspace-a",
		Input:       map[string]any{"bundle_id": compiled.Bundle.BundleID},
	})
	if withoutRead.Success || !errors.Is(withoutRead.Error, ErrCapabilityDenied) {
		t.Fatalf("expected read capability denial, got %#v", withoutRead)
	}

	grantContextCapability(t, kernel, "reader", "workspace-a", MutationScopeNone, SyscallContextRead)
	getBundle := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallContextGetBundle,
		ActorID:     "reader",
		WorkspaceID: "workspace-a",
		Input:       map[string]any{"bundle_id": compiled.Bundle.BundleID},
	})
	listBundles := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallContextListBundles,
		ActorID:     "reader",
		WorkspaceID: "workspace-a",
		Input:       map[string]any{"case_id": "case-a"},
	})
	getBlock := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallContextGetBlock,
		ActorID:     "reader",
		WorkspaceID: "workspace-a",
		Input:       map[string]any{"block_id": compiled.Blocks[0].BlockID},
	})
	listBlocks := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallContextListBlocks,
		ActorID:     "reader",
		WorkspaceID: "workspace-a",
		Input:       map[string]any{"bundle_id": compiled.Bundle.BundleID},
	})
	validateLayout := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallContextValidateLayout,
		ActorID:     "reader",
		WorkspaceID: "workspace-a",
		Input:       map[string]any{"layout_version": "phase-7-test"},
	})
	hash := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallContextHash,
		ActorID:     "reader",
		WorkspaceID: "workspace-a",
		Input:       map[string]any{"canonical_text": compiled.Bundle.CanonicalPromptText},
	})
	if !getBundle.Success || !listBundles.Success || !getBlock.Success || !listBlocks.Success || !validateLayout.Success || !hash.Success {
		t.Fatalf("read context syscalls failed: bundle=%#v list=%#v block=%#v blocks=%#v layout=%#v hash=%#v", getBundle, listBundles, getBlock, listBlocks, validateLayout, hash)
	}
	if len(kernel.Journal().ListEvents()) != before {
		t.Fatal("read-only context syscalls journaled or mutated state")
	}
}

func TestContextWorkspaceScopeIsEnforced(t *testing.T) {
	kernel := testKernel()
	grantContextCapability(t, kernel, "scoped", "workspace-b", MutationScopeCanonical, SyscallContextCompile)
	result := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallContextCompile,
		ActorID:     "scoped",
		WorkspaceID: "workspace-a",
		Input: map[string]any{
			"case_id":            "case-a",
			"source_object_refs": []string{"case-a"},
		},
	})
	if result.Success || !errors.Is(result.Error, ErrCapabilityDenied) {
		t.Fatalf("expected workspace capability denial, got %#v", result)
	}
}

func TestContextCompileFromSnapshotAndRestoreSeedJournalByReference(t *testing.T) {
	kernel := testKernel()
	grantSnapshotCapability(t, kernel, "operator", "workspace-a", SyscallSnapshotCreate, SyscallSnapshotRestoreSeed)
	snapshot := createSnapshot(t, kernel, "operator", map[string]any{
		"snapshot_type":           string(snapshots.SnapshotTypeContextRestoreSnapshot),
		"case_id":                 "case-a",
		"source_object_refs":      []string{"case-a", "semantic-a"},
		"source_refs":             []string{"source-a"},
		"admitted_object_refs":    []string{"exhibit-a"},
		"rejected_object_refs":    []string{"exhibit-rejected"},
		"semantic_operation_refs": []string{"operation-a"},
		"palace_route_refs":       []string{"route-a"},
		"contradiction_refs":      []string{"contradiction-a"},
		"summary":                 "snapshot shape",
	})
	seedResult := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallSnapshotRestoreSeed,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
		Input:       map[string]any{"snapshot_id": snapshot.SnapshotID},
	})
	if !seedResult.Success {
		t.Fatalf("snapshot.restore_seed failed: %v", seedResult.Error)
	}
	restoreSeed := seedResult.Output.(snapshots.RestoreSeed)
	beforeCompileSnapshot, ok := kernel.Snapshots().GetSnapshot(snapshot.SnapshotID)
	if !ok {
		t.Fatal("snapshot missing before context compile")
	}

	grantContextCapability(t, kernel, "operator", "workspace-a", MutationScopeCanonical,
		SyscallContextCompileFromSnapshot,
		SyscallContextCompileFromRestoreSeed,
	)
	fromSnapshot := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallContextCompileFromSnapshot,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
		Input: map[string]any{
			"snapshot_id":                       snapshot.SnapshotID,
			"include_rejected_evidence_summary": true,
			"include_contradictions":            true,
		},
	})
	if !fromSnapshot.Success {
		t.Fatalf("context.compile_from_snapshot failed: %v", fromSnapshot.Error)
	}
	compiledSnapshot := fromSnapshot.Output.(contextcompiler.ContextCompileResult)
	if compiledSnapshot.Bundle.SnapshotID != snapshot.SnapshotID ||
		!contains(compiledSnapshot.Bundle.SourceRefs, snapshot.SnapshotID) ||
		!contains(compiledSnapshot.Bundle.SourceRefs, "operation-a") {
		t.Fatalf("compiled bundle did not cite snapshot refs: %#v", compiledSnapshot.Bundle)
	}

	fromSeed := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallContextCompileFromRestoreSeed,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
		Input: map[string]any{
			"restore_seed_id":      restoreSeed.RestoreSeedID,
			"include_restore_seed": true,
		},
	})
	if !fromSeed.Success {
		t.Fatalf("context.compile_from_restore_seed failed: %v", fromSeed.Error)
	}
	compiledSeed := fromSeed.Output.(contextcompiler.ContextCompileResult)
	if compiledSeed.Bundle.RestoreSeedID != restoreSeed.RestoreSeedID ||
		!contains(compiledSeed.Bundle.SourceRefs, restoreSeed.RestoreSeedID) ||
		!contains(compiledSeed.Bundle.SourceRefs, restoreSeed.SnapshotID) {
		t.Fatalf("compiled bundle did not cite restore seed refs: %#v", compiledSeed.Bundle)
	}
	afterSnapshot, ok := kernel.Snapshots().GetSnapshot(snapshot.SnapshotID)
	if !ok || afterSnapshot.Status != beforeCompileSnapshot.Status {
		t.Fatalf("context compile mutated snapshot status: before=%s after=%#v", beforeCompileSnapshot.Status, afterSnapshot)
	}

	eventTypes := map[string]bool{}
	for _, event := range kernel.Journal().ListEvents() {
		eventTypes[event.EventType] = true
	}
	for _, eventType := range []string{
		JournalEventContextCompiledFromSnapshot,
		JournalEventContextCompiledFromRestoreSeed,
	} {
		if !eventTypes[eventType] {
			t.Fatalf("missing journal event %s in %#v", eventType, eventTypes)
		}
	}
}

func TestContextCompileDoesNotAdmitEvidenceOrCreateKVCache(t *testing.T) {
	kernel := testKernel()
	grantCaseCapability(t, kernel, "operator", "workspace-a")
	grantCourtCapability(t, kernel, "operator", SyscallCourtSubmit, SyscallCourtAdmit)
	caseID := openCourtCase(t, kernel, "operator")
	admittedID := submitExhibit(t, kernel, "operator", caseID, "admitted")
	rejectedID := submitExhibit(t, kernel, "operator", caseID, "not admitted")
	if admit := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallCourtAdmit,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
		Input:       map[string]any{"case_id": caseID, "exhibit_id": admittedID, "admission_reason": "sourced"},
	}); !admit.Success {
		t.Fatalf("court.admit failed: %v", admit.Error)
	}
	rejectedBefore, _ := kernel.Objects().GetObject(rejectedID)
	caseBefore, _ := kernel.Objects().GetObject(caseID)

	grantContextCapability(t, kernel, "operator", "workspace-a", MutationScopeCanonical, SyscallContextCompile)
	compiled := compileContext(t, kernel, "operator", map[string]any{
		"case_id":                            caseID,
		"source_object_refs":                 []string{caseID},
		"admitted_exhibit_refs":              []string{admittedID},
		"rejected_exhibit_refs":              []string{rejectedID},
		"include_rejected_evidence_summary":  true,
		"include_contradictions":             false,
		"semantic_operation_refs":            []string{"operation-a"},
		"current_task_summary":               "compile admitted evidence only",
		"future_kv_placeholder_requested_by": "ignored",
	})
	rejectedAfter, _ := kernel.Objects().GetObject(rejectedID)
	caseAfter, _ := kernel.Objects().GetObject(caseID)
	if rejectedBefore.State["admissibility_status"] != rejectedAfter.State["admissibility_status"] ||
		caseBefore.State["status"] != caseAfter.State["status"] {
		t.Fatalf("context compile changed court state: rejected before=%#v after=%#v case before=%#v after=%#v", rejectedBefore.State, rejectedAfter.State, caseBefore.State, caseAfter.State)
	}
	if compiled.Bundle.IsCanonicalTruth() || compiled.Bundle.IsModelResponse() || compiled.Bundle.IsKVCache() {
		t.Fatalf("context bundle claimed forbidden authority: %#v", compiled.Bundle)
	}
	for _, block := range compiled.Blocks {
		if block.IsCanonicalTruth() || block.IsKVCache() {
			t.Fatalf("context block claimed forbidden authority: %#v", block)
		}
		if block.BlockType == contextcompiler.BlockAdmittedEvidence && contains(block.AdmittedExhibitRefs, rejectedID) {
			t.Fatalf("rejected exhibit compiled as admitted evidence: %#v", block)
		}
	}
}
