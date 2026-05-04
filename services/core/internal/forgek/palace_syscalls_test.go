package forgek

import (
	"errors"
	"testing"

	"forge/projectforge/services/core/internal/forgek/court"
	"forge/projectforge/services/core/internal/forgek/palace"
)

func grantPalaceCapability(t *testing.T, kernel *Kernel, actorID string, syscalls ...string) {
	t.Helper()
	if err := kernel.Capabilities().Grant(Capability{
		CapabilityID:    "cap-palace-" + actorID,
		SubjectID:       actorID,
		AllowedSyscalls: syscalls,
		WorkspaceScope:  []string{"workspace-a"},
		MutationScope:   MutationScopeCanonical,
		AuditRequired:   true,
	}); err != nil {
		t.Fatalf("grant palace capability: %v", err)
	}
}

func createRoom(t *testing.T, kernel *Kernel, actorID, name string, tags []string) string {
	t.Helper()
	result := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallPalaceCreateRoom,
		ActorID:     actorID,
		WorkspaceID: "workspace-a",
		Input: map[string]any{
			"name":        name,
			"description": "room " + name,
			"domain_tags": tags,
		},
	})
	if !result.Success {
		t.Fatalf("palace.create_room failed: %v", result.Error)
	}
	return result.ObjectID
}

func createAnchor(t *testing.T, kernel *Kernel, actorID, roomID, label string, objectRefs, keywords, tags []string) string {
	t.Helper()
	result := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallPalaceCreateAnchor,
		ActorID:     actorID,
		WorkspaceID: "workspace-a",
		Input: map[string]any{
			"room_id":     roomID,
			"label":       label,
			"object_refs": objectRefs,
			"keywords":    keywords,
			"tags":        tags,
			"source_refs": []string{"doc:" + label},
		},
	})
	if !result.Success {
		t.Fatalf("palace.create_anchor failed: %v", result.Error)
	}
	return result.ObjectID
}

func TestMemoryPalaceSyscallsAreRegistered(t *testing.T) {
	kernel := testKernel()
	for _, name := range []string{
		SyscallPalaceCreateRoom,
		SyscallPalaceUpdateRoom,
		SyscallPalaceLinkRooms,
		SyscallPalaceCreateAnchor,
		SyscallPalaceUpdateAnchor,
		SyscallPalaceLinkAnchor,
		SyscallPalaceRoute,
		SyscallPalaceRecordRouteResult,
		SyscallPalaceListRooms,
		SyscallPalaceListAnchors,
		SyscallPalaceListRoutes,
		SyscallPalaceGetRoom,
		SyscallPalaceGetAnchor,
		SyscallPalaceGetRoute,
	} {
		if _, ok := kernel.Syscalls().Lookup(name); !ok {
			t.Fatalf("expected memory palace syscall %s to be registered", name)
		}
	}
}

func TestMemoryRoomSyscallsRequireCapabilityAndJournal(t *testing.T) {
	kernel := testKernel()

	denied := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallPalaceCreateRoom,
		ActorID:     "denied",
		WorkspaceID: "workspace-a",
		Input:       map[string]any{"name": "architecture"},
	})
	if denied.Success || !errors.Is(denied.Error, ErrCapabilityDenied) {
		t.Fatalf("expected capability denial, got %#v", denied)
	}

	grantPalaceCapability(t, kernel, "operator", SyscallPalaceCreateRoom, SyscallPalaceUpdateRoom, SyscallPalaceLinkRooms)
	roomA := createRoom(t, kernel, "operator", "architecture", []string{"architecture", "kernel"})
	roomB := createRoom(t, kernel, "operator", "evidence", []string{"evidence"})

	update := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallPalaceUpdateRoom,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
		Input: map[string]any{
			"room_id":     roomA,
			"description": "updated architecture room",
			"domain_tags": []string{"architecture", "microkernel"},
		},
	})
	if !update.Success {
		t.Fatalf("palace.update_room failed: %v", update.Error)
	}

	link := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallPalaceLinkRooms,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
		Input: map[string]any{
			"room_id":        roomA,
			"linked_room_id": roomB,
		},
	})
	if !link.Success {
		t.Fatalf("palace.link_rooms failed: %v", link.Error)
	}

	roomObj, _ := kernel.Objects().GetObject(roomA)
	if roomObj.ObjectType != ObjectTypeMemoryRoom || !containsStringState(roomObj.State, "linked_room_refs", roomB) {
		t.Fatalf("room object did not record link: %#v", roomObj)
	}
	if got := len(kernel.Journal().ListEvents()); got != 4 {
		t.Fatalf("expected create/create/update/link journal events, got %d", got)
	}
}

func TestMemoryAnchorRouteAndRouteResultAreJournaled(t *testing.T) {
	kernel := testKernel()
	grantCaseCapability(t, kernel, "operator", "workspace-a")
	grantCourtCapability(t, kernel, "operator", SyscallCourtSubmit, SyscallCourtAdmit)
	grantPalaceCapability(t, kernel, "operator",
		SyscallPalaceCreateRoom,
		SyscallPalaceLinkRooms,
		SyscallPalaceCreateAnchor,
		SyscallPalaceLinkAnchor,
		SyscallPalaceRoute,
		SyscallPalaceRecordRouteResult,
	)
	caseID := openCourtCase(t, kernel, "operator")
	roomA := createRoom(t, kernel, "operator", "architecture", []string{"architecture", "kernel"})
	roomB := createRoom(t, kernel, "operator", "evidence", []string{"evidence"})
	linkRooms := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallPalaceLinkRooms,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
		Input:       map[string]any{"room_id": roomA, "linked_room_id": roomB},
	})
	if !linkRooms.Success {
		t.Fatalf("palace.link_rooms failed: %v", linkRooms.Error)
	}
	sourceExhibit := submitExhibit(t, kernel, "operator", caseID, "courthouse admission doctrine")
	anchorID := createAnchor(t, kernel, "operator", roomA, "courthouse admission", []string{sourceExhibit}, []string{"courthouse", "admission"}, []string{"architecture"})

	linkAnchor := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallPalaceLinkAnchor,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
		Input: map[string]any{
			"anchor_id":   anchorID,
			"object_refs": []string{sourceExhibit, "ruling-future"},
			"source_refs": []string{"doc:courthouse"},
		},
	})
	if !linkAnchor.Success {
		t.Fatalf("palace.link_anchor failed: %v", linkAnchor.Error)
	}

	route := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallPalaceRoute,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
		Input: map[string]any{
			"case_id":       caseID,
			"query_text":    "courthouse admission architecture",
			"start_room_id": roomA,
			"tags":          []string{"architecture"},
		},
	})
	if !route.Success {
		t.Fatalf("palace.route failed: %v", route.Error)
	}
	palaceRoute := route.Output.(palace.PalaceRoute)
	if len(palaceRoute.CandidateObjects) == 0 {
		t.Fatal("route returned no candidates")
	}
	candidate := palaceRoute.CandidateObjects[0]
	if candidate.SourceObjectID != sourceExhibit || candidate.IsExhibit() || candidate.IsAdmittedEvidence() {
		t.Fatalf("candidate has wrong authority or source: %#v", candidate)
	}

	caseObj, _ := kernel.Objects().GetObject(caseID)
	if !containsStringState(caseObj.State, "palace_route_refs", route.ObjectID) {
		t.Fatalf("case does not reference palace route: %#v", caseObj.State)
	}

	record := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallPalaceRecordRouteResult,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
		Input: map[string]any{
			"route_id":      route.ObjectID,
			"candidate_id":  candidate.CandidateID,
			"result_status": palace.RouteResultAdmitted,
		},
	})
	if !record.Success {
		t.Fatalf("palace.record_route_result failed: %v", record.Error)
	}

	roomObj, _ := kernel.Objects().GetObject(roomA)
	stats := roomObj.State["route_stats"].(map[string]int)
	if stats["success_count"] != 1 {
		t.Fatalf("route result did not update room stats: %#v", stats)
	}
}

func TestMemoryPalaceCapabilitiesAndWorkspaceScopeAreEnforced(t *testing.T) {
	kernel := testKernel()
	grantPalaceCapability(t, kernel, "operator", SyscallPalaceCreateRoom)
	roomID := createRoom(t, kernel, "operator", "architecture", []string{"architecture"})

	deniedAnchor := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallPalaceCreateAnchor,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
		Input: map[string]any{
			"room_id": roomID,
			"label":   "missing anchor capability",
		},
	})
	if deniedAnchor.Success || !errors.Is(deniedAnchor.Error, ErrCapabilityDenied) {
		t.Fatalf("expected create_anchor capability denial, got %#v", deniedAnchor)
	}

	deniedRoute := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallPalaceRoute,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
		Input: map[string]any{
			"query_text":    "architecture",
			"start_room_id": roomID,
		},
	})
	if deniedRoute.Success || !errors.Is(deniedRoute.Error, ErrCapabilityDenied) {
		t.Fatalf("expected route capability denial, got %#v", deniedRoute)
	}

	if err := kernel.Capabilities().Grant(Capability{
		CapabilityID:    "cap-palace-other-workspace",
		SubjectID:       "other",
		AllowedSyscalls: []string{SyscallPalaceCreateRoom},
		WorkspaceScope:  []string{"workspace-b"},
		MutationScope:   MutationScopeCanonical,
	}); err != nil {
		t.Fatalf("grant other workspace capability: %v", err)
	}
	wrongScope := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallPalaceCreateRoom,
		ActorID:     "other",
		WorkspaceID: "workspace-a",
		Input:       map[string]any{"name": "wrong-scope"},
	})
	if wrongScope.Success || !errors.Is(wrongScope.Error, ErrCapabilityDenied) {
		t.Fatalf("expected workspace-scope denial, got %#v", wrongScope)
	}
}

func TestPalaceRouteCandidateRequiresCourthouseSubmitAndAdmit(t *testing.T) {
	kernel := testKernel()
	grantCaseCapability(t, kernel, "operator", "workspace-a")
	grantCourtCapability(t, kernel, "operator", SyscallCourtSubmit, SyscallCourtAdmit)
	grantPalaceCapability(t, kernel, "operator", SyscallPalaceCreateRoom, SyscallPalaceCreateAnchor, SyscallPalaceRoute)
	caseID := openCourtCase(t, kernel, "operator")
	roomID := createRoom(t, kernel, "operator", "architecture", []string{"architecture"})
	sourceExhibit := submitExhibit(t, kernel, "operator", caseID, "candidate source")
	createAnchor(t, kernel, "operator", roomID, "candidate source", []string{sourceExhibit}, []string{"candidate"}, []string{"architecture"})

	route := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallPalaceRoute,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
		Input: map[string]any{
			"case_id":       caseID,
			"query_text":    "candidate",
			"start_room_id": roomID,
		},
	})
	if !route.Success {
		t.Fatalf("palace.route failed: %v", route.Error)
	}
	candidate := route.Output.(palace.PalaceRoute).CandidateObjects[0]

	if _, ok := kernel.Court().GetExhibit(candidate.CandidateID); ok {
		t.Fatal("route candidate became an exhibit automatically")
	}
	candidateObject, ok := kernel.Objects().GetObject(candidate.CandidateID)
	if !ok {
		t.Fatal("candidate object was not registered for provenance")
	}
	if candidateObject.AuthorityLevel != AuthorityProposal {
		t.Fatalf("candidate should remain proposal authority, got %#v", candidateObject)
	}

	submit := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallCourtSubmit,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
		Input: map[string]any{
			"case_id":          caseID,
			"source_object_id": candidate.CandidateID,
			"source_type":      court.SourceTypeKernelObject,
			"source_refs":      []string{candidate.CandidateID, route.ObjectID},
			"content_summary":  candidate.CandidateSummary,
		},
	})
	if !submit.Success {
		t.Fatalf("court.submit candidate failed: %v", submit.Error)
	}
	exhibitObj, _ := kernel.Objects().GetObject(submit.ObjectID)
	if exhibitObj.State["admissibility_status"] != court.StatusSubmitted {
		t.Fatalf("submitted candidate was not SUBMITTED: %#v", exhibitObj.State)
	}

	admit := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallCourtAdmit,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
		Input: map[string]any{
			"case_id":          caseID,
			"exhibit_id":       submit.ObjectID,
			"admission_reason": "candidate submitted through Courthouse",
		},
	})
	if !admit.Success {
		t.Fatalf("court.admit submitted candidate failed: %v", admit.Error)
	}
}
