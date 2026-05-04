package forgek

import (
	"errors"
	"testing"

	"forge/projectforge/services/core/internal/forgek/palace"
	"forge/projectforge/services/core/internal/forgek/semantic"
)

func grantSemanticCapability(t *testing.T, kernel *Kernel, actorID string, syscalls ...string) {
	t.Helper()
	if err := kernel.Capabilities().Grant(Capability{
		CapabilityID:    "cap-semantic-" + actorID,
		SubjectID:       actorID,
		AllowedSyscalls: syscalls,
		WorkspaceScope:  []string{"workspace-a"},
		MutationScope:   MutationScopeCanonical,
		AuditRequired:   true,
	}); err != nil {
		t.Fatalf("grant semantic capability: %v", err)
	}
}

func semanticObjectInput(id, objectType, summary string, sourceRefs ...string) semantic.SemanticObject {
	return semantic.SemanticObject{
		SemanticObjectID:  id,
		WorkspaceID:       "workspace-a",
		ObjectType:        objectType,
		SourceObjectRefs:  append([]string(nil), sourceRefs...),
		SourceRefs:        append([]string(nil), sourceRefs...),
		ContentSummary:    summary,
		NormalizedContent: summary,
		AuthorityLevel:    semantic.AuthorityAdmitted,
		ProvenanceRefs:    []string{"event-" + id},
	}
}

func TestSemanticSyscallsAreRegistered(t *testing.T) {
	kernel := testKernel()
	for _, name := range []string{
		SyscallSemanticApply,
		SyscallSemanticMerge,
		SyscallSemanticDiff,
		SyscallSemanticIntersect,
		SyscallSemanticContradict,
		SyscallSemanticSupersede,
		SyscallSemanticCompress,
		SyscallSemanticDerive,
		SyscallSemanticPromote,
		SyscallSemanticDemote,
		SyscallSemanticExpire,
		SyscallSemanticListOperations,
		SyscallSemanticGetOperation,
	} {
		if _, ok := kernel.Syscalls().Lookup(name); !ok {
			t.Fatalf("expected semantic syscall %s to be registered", name)
		}
	}
}

func TestSemanticSyscallsRequireCapabilitiesAndJournalOperations(t *testing.T) {
	kernel := testKernel()
	inputs := []semantic.SemanticObject{
		semanticObjectInput("semantic-a", semantic.ObjectTypeEvidence, "kernel commits truth", "exhibit-a"),
		semanticObjectInput("semantic-b", semantic.ObjectTypeEvidence, "courthouse admits evidence", "exhibit-b"),
	}

	denied := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallSemanticMerge,
		ActorID:     "denied",
		WorkspaceID: "workspace-a",
		Input:       map[string]any{"objects": inputs},
	})
	if denied.Success || !errors.Is(denied.Error, ErrCapabilityDenied) {
		t.Fatalf("expected semantic capability denial, got %#v", denied)
	}

	grantSemanticCapability(t, kernel, "operator", SyscallSemanticMerge)
	merge := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallSemanticMerge,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
		Input:       map[string]any{"objects": inputs, "case_id": "case-1"},
	})
	if !merge.Success {
		t.Fatalf("semantic.merge failed: %v", merge.Error)
	}
	result := merge.Output.(semantic.SemanticTransformResult)
	if len(result.OutputObjects) != 1 || result.OutputObjects[0].AuthorityLevel != semantic.AuthorityProposal {
		t.Fatalf("merge output has wrong authority: %#v", result.OutputObjects)
	}
	operationObj, ok := kernel.Objects().GetObject(merge.ObjectID)
	if !ok || operationObj.ObjectType != ObjectTypeSemanticOperation {
		t.Fatalf("semantic operation not registered: %#v", operationObj)
	}
	if got := len(kernel.Journal().ListEvents()); got != 1 {
		t.Fatalf("expected one semantic journal event, got %d", got)
	}
	if kernel.Journal().ListEvents()[0].EventType != JournalEventSemanticMergeApplied {
		t.Fatalf("unexpected journal event: %#v", kernel.Journal().ListEvents()[0])
	}
}

func TestSemanticMutationSyscallsRequireSpecificCapabilities(t *testing.T) {
	kernel := testKernel()
	inputs := []semantic.SemanticObject{
		semanticObjectInput("semantic-a", semantic.ObjectTypeEvidence, "kernel commits truth", "exhibit-a"),
		semanticObjectInput("semantic-b", semantic.ObjectTypeEvidence, "courthouse admits evidence", "exhibit-b"),
	}
	for _, syscallName := range []string{
		SyscallSemanticApply,
		SyscallSemanticMerge,
		SyscallSemanticDiff,
		SyscallSemanticIntersect,
		SyscallSemanticContradict,
		SyscallSemanticSupersede,
		SyscallSemanticCompress,
		SyscallSemanticDerive,
		SyscallSemanticPromote,
		SyscallSemanticDemote,
		SyscallSemanticExpire,
	} {
		input := map[string]any{"objects": inputs}
		if syscallName == SyscallSemanticApply {
			input["operation_type"] = semantic.OperationMerge
		}
		result := kernel.DispatchSyscall(SyscallRequest{
			Name:        syscallName,
			ActorID:     "denied",
			WorkspaceID: "workspace-a",
			Input:       input,
		})
		if result.Success || !errors.Is(result.Error, ErrCapabilityDenied) {
			t.Fatalf("expected capability denial for %s, got %#v", syscallName, result)
		}
	}
}

func TestSemanticOperatorsPreserveCourtAndPalaceBoundaries(t *testing.T) {
	kernel := testKernel()
	grantCaseCapability(t, kernel, "operator", "workspace-a")
	grantCourtCapability(t, kernel, "operator", SyscallCourtSubmit, SyscallCourtAdmit, SyscallCourtReject)
	grantPalaceCapability(t, kernel, "operator", SyscallPalaceCreateRoom, SyscallPalaceCreateAnchor, SyscallPalaceRoute)
	grantSemanticCapability(t, kernel, "operator", SyscallSemanticCompress, SyscallSemanticMerge)
	caseID := openCourtCase(t, kernel, "operator")
	admittedID := submitExhibit(t, kernel, "operator", caseID, "admitted evidence")
	if admit := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallCourtAdmit,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
		Input:       map[string]any{"case_id": caseID, "exhibit_id": admittedID, "admission_reason": "source is inspectable"},
	}); !admit.Success {
		t.Fatalf("court.admit failed: %v", admit.Error)
	}
	rejectedID := submitExhibit(t, kernel, "operator", caseID, "rejected evidence")
	if reject := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallCourtReject,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
		Input:       map[string]any{"case_id": caseID, "exhibit_id": rejectedID, "rejection_reason": "not admissible"},
	}); !reject.Success {
		t.Fatalf("court.reject failed: %v", reject.Error)
	}

	compress := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallSemanticCompress,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
		Input: map[string]any{
			"case_id": caseID,
			"objects": []semantic.SemanticObject{
				semanticObjectInput("semantic-admitted", semantic.ObjectTypeExhibitRef, "admitted evidence", admittedID),
			},
		},
	})
	if !compress.Success {
		t.Fatalf("semantic.compress failed: %v", compress.Error)
	}
	compressed := compress.Output.(semantic.SemanticTransformResult).OutputObjects[0]
	if compressed.Metadata["compressed"] != true || !containsStringState(map[string]any{"refs": compressed.SourceObjectRefs}, "refs", admittedID) {
		t.Fatalf("compressed object lost admitted exhibit provenance: %#v", compressed)
	}

	mergeRejected := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallSemanticMerge,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
		Input: map[string]any{
			"case_id": caseID,
			"objects": []semantic.SemanticObject{
				semanticObjectInput("semantic-rejected", semantic.ObjectTypeExhibitRef, "rejected evidence", rejectedID),
			},
		},
	})
	if mergeRejected.Success {
		t.Fatal("semantic.merge silently treated rejected exhibit as admitted input")
	}

	roomID := createRoom(t, kernel, "operator", "semantic candidates", []string{"semantic"})
	createAnchor(t, kernel, "operator", roomID, "candidate source", []string{admittedID}, []string{"candidate"}, []string{"semantic"})
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
	candidateTransform := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallSemanticCompress,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
		Input: map[string]any{
			"case_id": caseID,
			"objects": []semantic.SemanticObject{
				semanticObjectInput("semantic-candidate", semantic.ObjectTypeCandidateRef, candidate.CandidateSummary, candidate.CandidateID),
			},
		},
	})
	if !candidateTransform.Success {
		t.Fatalf("semantic.compress candidate failed: %v", candidateTransform.Error)
	}
	if _, ok := kernel.Court().GetExhibit(candidate.CandidateID); ok {
		t.Fatal("semantic transform converted candidate into exhibit")
	}
	candidateObj, _ := kernel.Objects().GetObject(candidate.CandidateID)
	if candidateObj.AuthorityLevel != AuthorityProposal {
		t.Fatalf("candidate authority changed after semantic transform: %#v", candidateObj)
	}
}

func TestSemanticContradictAndSupersedeRequestCourthouseButDoNotExecute(t *testing.T) {
	kernel := testKernel()
	grantSemanticCapability(t, kernel, "operator", SyscallSemanticContradict, SyscallSemanticSupersede)
	inputs := []semantic.SemanticObject{
		semanticObjectInput("semantic-a", semantic.ObjectTypeExhibitRef, "first exhibit", "exhibit-a"),
		semanticObjectInput("semantic-b", semantic.ObjectTypeExhibitRef, "second exhibit", "exhibit-b"),
	}

	contradict := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallSemanticContradict,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
		Input: map[string]any{
			"case_id": caseIDForSemanticRequest,
			"objects": inputs,
			"parameters": map[string]any{
				"exhibit_a_id": "exhibit-a",
				"exhibit_b_id": "exhibit-b",
			},
		},
	})
	if !contradict.Success {
		t.Fatalf("semantic.contradict failed: %v", contradict.Error)
	}
	if got := len(kernel.Court().ListCaseContradictions(caseIDForSemanticRequest)); got != 0 {
		t.Fatalf("semantic.contradict executed courthouse syscall directly, contradictions=%d", got)
	}
	result := contradict.Output.(semantic.SemanticTransformResult)
	if len(result.RequestedSyscalls) != 1 || result.RequestedSyscalls[0].SyscallName != SyscallCourtRegisterContradiction {
		t.Fatalf("semantic.contradict did not produce syscall request: %#v", result.RequestedSyscalls)
	}

	supersede := kernel.DispatchSyscall(SyscallRequest{
		Name:        SyscallSemanticSupersede,
		ActorID:     "operator",
		WorkspaceID: "workspace-a",
		Input: map[string]any{
			"case_id": caseIDForSemanticRequest,
			"objects": inputs,
			"parameters": map[string]any{
				"old_object_id": "exhibit-a",
				"new_object_id": "exhibit-b",
				"reason":        "newer evidence",
			},
		},
	})
	if !supersede.Success {
		t.Fatalf("semantic.supersede failed: %v", supersede.Error)
	}
	if len(supersede.Output.(semantic.SemanticTransformResult).RequestedSyscalls) != 1 {
		t.Fatalf("semantic.supersede did not produce syscall request: %#v", supersede.Output)
	}
}

const caseIDForSemanticRequest = "case-semantic-request"
