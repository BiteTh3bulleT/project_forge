package forgek

import (
	"errors"
	"testing"

	"forge/projectforge/services/core/internal/forgek/court"
	"forge/projectforge/services/core/internal/forgek/palace"
	"forge/projectforge/services/core/internal/forgek/semantic"
	"forge/projectforge/services/core/internal/forgek/snapshots"
)

func grantSnapshotCapability(t *testing.T, kernel *Kernel, actorID string, workspaceID string, syscalls ...string) {
	t.Helper()
	if err := kernel.Capabilities().Grant(Capability{
		CapabilityID:    "cap-snapshot-" + actorID + "-" + workspaceID,
		SubjectID:       actorID,
		AllowedSyscalls: syscalls,
		WorkspaceScope:  []string{workspaceID},
		MutationScope:   MutationScopeCanonical,
		AuditRequired:   true,
	}); err != nil {
		t.Fatalf("grant snapshot capability: %v", err)
	}
}

func createSnapshot(t *testing.T, kernel *Kernel, actorID string, input map[string]any) snapshots.Snapshot {
	t.Helper()
	result := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallSnapshotCreate,
		ActorID:     actorID,
		WorkspaceID: "workspace-a",
		Input:       input,
	})
	if !result.Success {
		t.Fatalf("snapshot.create failed: %v", result.Error)
	}
	return result.Output.(snapshots.Snapshot)
}

func TestSnapshotSyscallsAreRegistered(t *testing.T) {
	kernel := testKernel()
	for _, name := range []string{
		SyscallSnapshotCreate,
		SyscallSnapshotGet,
		SyscallSnapshotList,
		SyscallSnapshotSeal,
		SyscallSnapshotSupersede,
		SyscallSnapshotExpire,
		SyscallSnapshotDiff,
		SyscallSnapshotRestoreSeed,
	} {
		if _, ok := kernel.Syscalls().Lookup(name); !ok {
			t.Fatalf("expected snapshot syscall %s to be registered", name)
		}
	}
	if kernel.Snapshots() == nil {
		t.Fatal("kernel does not own snapshot service")
	}
}

func TestSnapshotCreateRequiresCapabilityAndJournalsShape(t *testing.T) {
	kernel := testKernel()
	denied := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallSnapshotCreate,
		ActorID:     "denied",
		WorkspaceID: "workspace-a",
		Input: map[string]any{
			"snapshot_type":      string(snapshots.SnapshotTypeSemanticSnapshot),
			"source_object_refs": []string{"semantic-a"},
		},
	})
	if denied.Success || !errors.Is(denied.Error, ErrCapabilityDenied) {
		t.Fatalf("expected capability denial, got %#v", denied)
	}

	grantSnapshotCapability(t, kernel, "operator", "workspace-a", SyscallSnapshotCreate)
	created := createSnapshot(t, kernel, "operator", map[string]any{
		"snapshot_type":            string(snapshots.SnapshotTypeSemanticSnapshot),
		"source_object_refs":       []string{"semantic-b", "semantic-a", "semantic-a"},
		"semantic_operation_refs":  []string{"operation-a"},
		"source_refs":              []string{"doc-a"},
		"summary":                  "semantic shape",
		"seal":                     true,
		"workspace_id":             "workspace-a",
		"future_ignored_parameter": "ignored",
	})
	if created.Status != snapshots.StatusSealed || created.SealedAt == nil {
		t.Fatalf("sealed create did not seal snapshot: %#v", created)
	}
	if created.ShapeHash == "" || created.SourceHash == "" {
		t.Fatalf("snapshot hashes missing: %#v", created)
	}
	obj, ok := kernel.Objects().GetObject(created.SnapshotID)
	if !ok || obj.ObjectType != ObjectTypeSnapshot || obj.AuthorityLevel != AuthorityShape {
		t.Fatalf("snapshot object not registered as shape: %#v", obj)
	}
	if obj.State["is_canonical_truth"] != false || obj.State["is_context_block"] != false || obj.State["is_deterministic_kv_cache"] != false {
		t.Fatalf("snapshot object claimed wrong authority: %#v", obj.State)
	}
	events := kernel.Journal().ListEvents()
	if len(events) != 1 || events[0].EventType != JournalEventSnapshotCreated || events[0].SyscallName != SyscallSnapshotCreate {
		t.Fatalf("snapshot create was not journaled: %#v", events)
	}
}

func TestSnapshotMutationCapabilitiesAndWorkspaceScopeAreEnforced(t *testing.T) {
	kernel := testKernel()
	grantSnapshotCapability(t, kernel, "operator", "workspace-a", SyscallSnapshotCreate)
	first := createSnapshot(t, kernel, "operator", map[string]any{
		"snapshot_type":      string(snapshots.SnapshotTypeCaseSnapshot),
		"source_object_refs": []string{"case-a"},
	})
	second := createSnapshot(t, kernel, "operator", map[string]any{
		"snapshot_type":      string(snapshots.SnapshotTypeCaseSnapshot),
		"source_object_refs": []string{"case-b"},
	})

	for _, attempt := range []SyscallRequest{
		{Name: SyscallSnapshotSeal, ActorID: "operator", WorkspaceID: "workspace-a", Input: map[string]any{"snapshot_id": first.SnapshotID}},
		{Name: SyscallSnapshotSupersede, ActorID: "operator", WorkspaceID: "workspace-a", Input: map[string]any{"old_snapshot_id": first.SnapshotID, "new_snapshot_id": second.SnapshotID}},
		{Name: SyscallSnapshotExpire, ActorID: "operator", WorkspaceID: "workspace-a", Input: map[string]any{"snapshot_id": first.SnapshotID}},
		{Name: SyscallSnapshotRestoreSeed, ActorID: "operator", WorkspaceID: "workspace-a", Input: map[string]any{"snapshot_id": first.SnapshotID}},
	} {
		result := kernel.DispatchSyscall(attempt)
		if result.Success || !errors.Is(result.Error, ErrCapabilityDenied) {
			t.Fatalf("expected capability denial for %s, got %#v", attempt.Name, result)
		}
	}

	grantSnapshotCapability(t, kernel, "scoped", "workspace-b", SyscallSnapshotCreate)
	wrongScope := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallSnapshotCreate,
		ActorID:     "scoped",
		WorkspaceID: "workspace-a",
		Input: map[string]any{
			"snapshot_type":      string(snapshots.SnapshotTypeSemanticSnapshot),
			"source_object_refs": []string{"semantic-a"},
		},
	})
	if wrongScope.Success || !errors.Is(wrongScope.Error, ErrCapabilityDenied) {
		t.Fatalf("expected workspace capability denial, got %#v", wrongScope)
	}
}

func TestSnapshotLifecycleSyscallsJournalAndPreserveInspectability(t *testing.T) {
	kernel := testKernel()
	grantSnapshotCapability(t, kernel, "operator", "workspace-a",
		SyscallSnapshotCreate,
		SyscallSnapshotSeal,
		SyscallSnapshotSupersede,
		SyscallSnapshotExpire,
		SyscallSnapshotRestoreSeed,
	)
	first := createSnapshot(t, kernel, "operator", map[string]any{
		"snapshot_type":      string(snapshots.SnapshotTypeCaseSnapshot),
		"source_object_refs": []string{"case-a"},
	})
	second := createSnapshot(t, kernel, "operator", map[string]any{
		"snapshot_type":      string(snapshots.SnapshotTypeCaseSnapshot),
		"source_object_refs": []string{"case-b"},
	})

	seal := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallSnapshotSeal,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
		Input:       map[string]any{"snapshot_id": first.SnapshotID},
	})
	if !seal.Success || seal.Output.(snapshots.Snapshot).Status != snapshots.StatusSealed {
		t.Fatalf("snapshot.seal failed: %#v", seal)
	}
	supersede := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallSnapshotSupersede,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
		Input: map[string]any{
			"old_snapshot_id": first.SnapshotID,
			"new_snapshot_id": second.SnapshotID,
		},
	})
	if !supersede.Success {
		t.Fatalf("snapshot.supersede failed: %v", supersede.Error)
	}
	superseded, ok := kernel.Snapshots().GetSnapshot(first.SnapshotID)
	if !ok || superseded.Status != snapshots.StatusSuperseded || superseded.SupersededBy != second.SnapshotID {
		t.Fatalf("superseded snapshot not inspectable: %#v ok=%v", superseded, ok)
	}
	if _, ok := kernel.Snapshots().GetSnapshot(second.SnapshotID); !ok {
		t.Fatal("superseding snapshot not inspectable")
	}

	third := createSnapshot(t, kernel, "operator", map[string]any{
		"snapshot_type":      string(snapshots.SnapshotTypeSemanticSnapshot),
		"source_object_refs": []string{"semantic-c"},
	})
	expire := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallSnapshotExpire,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
		Input:       map[string]any{"snapshot_id": third.SnapshotID},
	})
	if !expire.Success || expire.Output.(snapshots.Snapshot).Status != snapshots.StatusExpired {
		t.Fatalf("snapshot.expire failed: %#v", expire)
	}
	if _, ok := kernel.Snapshots().GetSnapshot(third.SnapshotID); !ok {
		t.Fatal("expired snapshot not inspectable")
	}

	fourth := createSnapshot(t, kernel, "operator", map[string]any{
		"snapshot_type":      string(snapshots.SnapshotTypeContextRestoreSnapshot),
		"source_object_refs": []string{"semantic-d"},
	})
	seed := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallSnapshotRestoreSeed,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
		Input:       map[string]any{"snapshot_id": fourth.SnapshotID},
	})
	if !seed.Success {
		t.Fatalf("snapshot.restore_seed failed: %v", seed.Error)
	}
	restoreSeed := seed.Output.(snapshots.RestoreSeed)
	if restoreSeed.IsCanonicalTruth() || restoreSeed.IsContextBlock() {
		t.Fatal("restore seed claimed truth or context authority")
	}
	seedObj, ok := kernel.Objects().GetObject(seed.ObjectID)
	if !ok || seedObj.ObjectType != ObjectTypeRestoreSeed || seedObj.State["is_context_block"] != false {
		t.Fatalf("restore seed object has wrong authority: %#v", seedObj)
	}

	eventTypes := map[string]bool{}
	for _, event := range kernel.Journal().ListEvents() {
		eventTypes[event.EventType] = true
	}
	for _, eventType := range []string{
		JournalEventSnapshotCreated,
		JournalEventSnapshotSealed,
		JournalEventSnapshotSuperseded,
		JournalEventSnapshotExpired,
		JournalEventSnapshotRestoreSeedCreated,
	} {
		if !eventTypes[eventType] {
			t.Fatalf("missing journal event %s in %#v", eventType, eventTypes)
		}
	}
}

func TestSnapshotReadOnlySyscallsDoNotRequireMutationCapabilityOrJournal(t *testing.T) {
	kernel := testKernel()
	grantSnapshotCapability(t, kernel, "operator", "workspace-a", SyscallSnapshotCreate)
	first := createSnapshot(t, kernel, "operator", map[string]any{
		"snapshot_type":      string(snapshots.SnapshotTypeSemanticSnapshot),
		"source_object_refs": []string{"semantic-a", "semantic-b"},
	})
	second := createSnapshot(t, kernel, "operator", map[string]any{
		"snapshot_type":      string(snapshots.SnapshotTypeSemanticSnapshot),
		"source_object_refs": []string{"semantic-b", "semantic-c"},
	})
	before := len(kernel.Journal().ListEvents())

	get := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallSnapshotGet,
		ActorID:     "reader",
		WorkspaceID: "workspace-a",
		Input:       map[string]any{"snapshot_id": first.SnapshotID},
	})
	list := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallSnapshotList,
		ActorID:     "reader",
		WorkspaceID: "workspace-a",
		Input:       map[string]any{"snapshot_type": string(snapshots.SnapshotTypeSemanticSnapshot)},
	})
	diff := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallSnapshotDiff,
		ActorID:     "reader",
		WorkspaceID: "workspace-a",
		Input: map[string]any{
			"left_snapshot_id":  first.SnapshotID,
			"right_snapshot_id": second.SnapshotID,
		},
	})
	if !get.Success || !list.Success || !diff.Success {
		t.Fatalf("read-only snapshot syscalls failed: get=%#v list=%#v diff=%#v", get, list, diff)
	}
	if len(kernel.Journal().ListEvents()) != before {
		t.Fatal("read-only snapshot syscalls journaled or mutated state")
	}
	snapshotDiff := diff.Output.(snapshots.SnapshotDiff)
	if !contains(snapshotDiff.AddedRefs, "semantic-c") || !contains(snapshotDiff.RemovedRefs, "semantic-a") {
		t.Fatalf("diff did not compare snapshot shape: %#v", snapshotDiff)
	}
}

func TestSnapshotCitesCourthouseObjectsWithoutChangingAdmissibility(t *testing.T) {
	kernel := testKernel()
	grantCaseCapability(t, kernel, "operator", "workspace-a")
	grantCourtCapability(t, kernel, "operator", SyscallCourtSubmit, SyscallCourtAdmit, SyscallCourtReject, SyscallCourtRule)
	grantSnapshotCapability(t, kernel, "operator", "workspace-a", SyscallSnapshotCreate, SyscallSnapshotRestoreSeed)
	caseID := openCourtCase(t, kernel, "operator")
	admittedID := submitExhibit(t, kernel, "operator", caseID, "admitted")
	rejectedID := submitExhibit(t, kernel, "operator", caseID, "rejected")
	if admit := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallCourtAdmit,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
		Input:       map[string]any{"case_id": caseID, "exhibit_id": admittedID, "admission_reason": "sourced"},
	}); !admit.Success {
		t.Fatalf("court.admit failed: %v", admit.Error)
	}
	if reject := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallCourtReject,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
		Input:       map[string]any{"case_id": caseID, "exhibit_id": rejectedID, "rejection_reason": "insufficient"},
	}); !reject.Success {
		t.Fatalf("court.reject failed: %v", reject.Error)
	}
	ruling := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallCourtRule,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
		Input: map[string]any{
			"case_id":               caseID,
			"ruling_type":           court.RulingAdmission,
			"admitted_exhibit_refs": []string{admittedID},
			"rejected_exhibit_refs": []string{rejectedID},
		},
	})
	if !ruling.Success {
		t.Fatalf("court.rule failed: %v", ruling.Error)
	}
	beforeCase, _ := kernel.Objects().GetObject(caseID)

	snapshot := createSnapshot(t, kernel, "operator", map[string]any{
		"snapshot_type":           string(snapshots.SnapshotTypeCaseSnapshot),
		"case_id":                 caseID,
		"source_object_refs":      []string{caseID},
		"submitted_object_refs":   []string{admittedID, rejectedID},
		"admitted_object_refs":    []string{admittedID},
		"rejected_object_refs":    []string{rejectedID},
		"summary":                 "case admissibility shape",
		"semantic_operation_refs": []string{ruling.ObjectID},
	})
	afterCase, _ := kernel.Objects().GetObject(caseID)
	if beforeCase.State["status"] != afterCase.State["status"] ||
		!containsStringState(afterCase.State, "admitted_exhibits", admittedID) ||
		!containsStringState(afterCase.State, "rejected_exhibits", rejectedID) {
		t.Fatalf("snapshot changed case/admissibility state: before=%#v after=%#v", beforeCase.State, afterCase.State)
	}
	admittedObj, _ := kernel.Objects().GetObject(admittedID)
	rejectedObj, _ := kernel.Objects().GetObject(rejectedID)
	if admittedObj.State["admissibility_status"] != court.StatusAdmitted || rejectedObj.State["admissibility_status"] != court.StatusRejected {
		t.Fatalf("snapshot changed exhibit admissibility: admitted=%#v rejected=%#v", admittedObj.State, rejectedObj.State)
	}
	if !contains(snapshot.AdmittedObjectRefs, admittedID) || !contains(snapshot.RejectedObjectRefs, rejectedID) {
		t.Fatalf("snapshot did not cite courthouse refs: %#v", snapshot)
	}
	seed := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallSnapshotRestoreSeed,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
		Input:       map[string]any{"snapshot_id": snapshot.SnapshotID},
	})
	if !seed.Success {
		t.Fatalf("restore seed failed: %v", seed.Error)
	}
	if seed.Output.(snapshots.RestoreSeed).IsContextBlock() {
		t.Fatal("restore seed became context")
	}
}

func TestSnapshotCitesPalaceCandidatesWithoutAdmittingEvidence(t *testing.T) {
	kernel := testKernel()
	grantCaseCapability(t, kernel, "operator", "workspace-a")
	grantCourtCapability(t, kernel, "operator", SyscallCourtSubmit, SyscallCourtAdmit)
	grantPalaceCapability(t, kernel, "operator", SyscallPalaceCreateRoom, SyscallPalaceCreateAnchor, SyscallPalaceRoute)
	grantSnapshotCapability(t, kernel, "operator", "workspace-a", SyscallSnapshotCreate)
	caseID := openCourtCase(t, kernel, "operator")
	roomID := createRoom(t, kernel, "operator", "snapshot routes", []string{"snapshot"})
	sourceExhibit := submitExhibit(t, kernel, "operator", caseID, "candidate source")
	createAnchor(t, kernel, "operator", roomID, "candidate source", []string{sourceExhibit}, []string{"candidate"}, []string{"snapshot"})
	route := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallPalaceRoute,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
		Input:       map[string]any{"case_id": caseID, "query_text": "candidate", "start_room_id": roomID},
	})
	if !route.Success {
		t.Fatalf("palace.route failed: %v", route.Error)
	}
	candidate := route.Output.(palace.PalaceRoute).CandidateObjects[0]
	snapshot := createSnapshot(t, kernel, "operator", map[string]any{
		"snapshot_type":      string(snapshots.SnapshotTypePalaceRouteSnapshot),
		"case_id":            caseID,
		"palace_route_refs":  []string{route.ObjectID},
		"source_object_refs": []string{candidate.CandidateID},
		"summary":            "retrieval route shape",
	})
	if !contains(snapshot.PalaceRouteRefs, route.ObjectID) || !contains(snapshot.SourceObjectRefs, candidate.CandidateID) {
		t.Fatalf("snapshot did not cite palace route/candidate refs: %#v", snapshot)
	}
	if _, ok := kernel.Court().GetExhibit(candidate.CandidateID); ok {
		t.Fatal("snapshot converted candidate into exhibit")
	}
	candidateObject, ok := kernel.Objects().GetObject(candidate.CandidateID)
	if !ok || candidateObject.AuthorityLevel != AuthorityProposal {
		t.Fatalf("candidate authority changed: %#v", candidateObject)
	}
}

func TestSnapshotCitesSemanticOperationsWithoutPromotingDerivedObjects(t *testing.T) {
	kernel := testKernel()
	grantSemanticCapability(t, kernel, "operator", SyscallSemanticCompress)
	grantSnapshotCapability(t, kernel, "operator", "workspace-a", SyscallSnapshotCreate)
	compress := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallSemanticCompress,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
		Input: map[string]any{
			"case_id": "case-semantic",
			"objects": []semantic.SemanticObject{
				semanticObjectInput("semantic-a", semantic.ObjectTypeEvidence, "semantic source", "exhibit-a"),
			},
		},
	})
	if !compress.Success {
		t.Fatalf("semantic.compress failed: %v", compress.Error)
	}
	transform := compress.Output.(semantic.SemanticTransformResult)
	derivedID := transform.OutputObjects[0].SemanticObjectID
	derivedBefore, _ := kernel.Objects().GetObject(derivedID)

	snapshot := createSnapshot(t, kernel, "operator", map[string]any{
		"snapshot_type":           string(snapshots.SnapshotTypeSemanticSnapshot),
		"case_id":                 "case-semantic",
		"semantic_operation_refs": []string{compress.ObjectID},
		"derived_object_refs":     []string{derivedID},
		"source_object_refs":      []string{"semantic-a"},
		"summary":                 "semantic transform shape",
	})
	derivedAfter, _ := kernel.Objects().GetObject(derivedID)
	if derivedBefore.AuthorityLevel != derivedAfter.AuthorityLevel ||
		derivedAfter.AuthorityLevel != AuthorityProposal ||
		derivedAfter.State["semantic_authority"] != semantic.AuthorityProposal {
		t.Fatalf("snapshot promoted derived object: before=%#v after=%#v", derivedBefore, derivedAfter)
	}
	operation, ok := kernel.Semantic().GetOperation(compress.ObjectID)
	if !ok || !contains(operation.ProvenanceRefs, "semantic-a") || !contains(operation.ProvenanceRefs, "event-semantic-a") {
		t.Fatalf("semantic operation provenance not preserved: %#v", operation)
	}
	if !contains(snapshot.SemanticOperationRefs, compress.ObjectID) || !contains(snapshot.DerivedObjectRefs, derivedID) {
		t.Fatalf("snapshot did not cite semantic refs: %#v", snapshot)
	}
}
