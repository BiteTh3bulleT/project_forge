package semantic

import (
	"testing"
	"time"
)

func testSemanticObject(id, summary string, refs ...string) SemanticObject {
	return SemanticObject{
		SemanticObjectID:  id,
		WorkspaceID:       "workspace-a",
		ObjectType:        ObjectTypeEvidence,
		SourceObjectRefs:  append([]string(nil), refs...),
		SourceRefs:        append([]string(nil), refs...),
		ContentSummary:    summary,
		NormalizedContent: summary,
		AuthorityLevel:    AuthorityAdmitted,
		ProvenanceRefs:    []string{"event-" + id},
		CreatedAt:         time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC),
	}
}

func TestMergePreservesProvenanceAndRejectsContradictions(t *testing.T) {
	service := NewSemanticAlgebraService()
	a := testSemanticObject("semantic-a", "kernel commits truth", "exhibit-a")
	b := testSemanticObject("semantic-b", "courthouse admits evidence", "exhibit-b")

	result, err := service.Merge(OperationRequest{
		OperationID:  "operation-1",
		ResultID:     "result-1",
		WorkspaceID:  "workspace-a",
		CaseID:       "case-1",
		InputObjects: []SemanticObject{a, b},
		CreatedBy:    "operator",
		CreatedAt:    time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC),
		NextObjectID: fixedObjectID("semantic-merged"),
	})
	if err != nil {
		t.Fatalf("merge failed: %v", err)
	}
	if len(result.OutputObjects) != 1 {
		t.Fatalf("expected one merged object, got %#v", result.OutputObjects)
	}
	merged := result.OutputObjects[0]
	if merged.SemanticObjectID != "semantic-merged" || !containsString(merged.SourceObjectRefs, "exhibit-a") || !containsString(merged.SourceObjectRefs, "exhibit-b") {
		t.Fatalf("merge did not preserve source refs: %#v", merged)
	}
	if len(service.ListOperations("workspace-a")) != 1 {
		t.Fatal("merge did not record operation")
	}

	contradicted := b
	contradicted.ContradictedBy = []string{"contradiction-1"}
	if _, err := service.Merge(OperationRequest{
		OperationID:  "operation-2",
		ResultID:     "result-2",
		WorkspaceID:  "workspace-a",
		InputObjects: []SemanticObject{a, contradicted},
		CreatedBy:    "operator",
		CreatedAt:    time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC),
		NextObjectID: fixedObjectID("semantic-blocked"),
	}); err == nil {
		t.Fatal("merge silently accepted contradicted input")
	}
}

func TestDiffIntersectCompressAndDeriveAreDeterministicAndProvenanceSafe(t *testing.T) {
	service := NewSemanticAlgebraService()
	a := testSemanticObject("semantic-a", "kernel commits truth courthouse admits evidence", "exhibit-a", "shared")
	b := testSemanticObject("semantic-b", "courthouse admits evidence context compiler shapes", "exhibit-b", "shared")
	request := OperationRequest{
		WorkspaceID:  "workspace-a",
		CaseID:       "case-1",
		InputObjects: []SemanticObject{a, b},
		CreatedBy:    "operator",
		CreatedAt:    time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC),
		NextObjectID: sequenceObjectIDs("semantic-1", "semantic-2", "semantic-3"),
	}

	diff, err := service.Diff(withIDs(request, "operation-diff", "result-diff"))
	if err != nil {
		t.Fatalf("diff failed: %v", err)
	}
	if len(diff.OutputObjects) != 1 || diff.OutputObjects[0].ContentSummary == "" {
		t.Fatalf("diff did not produce summary object: %#v", diff)
	}

	intersect, err := service.Intersect(withIDs(request, "operation-intersect", "result-intersect"))
	if err != nil {
		t.Fatalf("intersect failed: %v", err)
	}
	if !containsString(intersect.OutputObjects[0].SourceObjectRefs, "shared") {
		t.Fatalf("intersect did not preserve shared source ref: %#v", intersect.OutputObjects[0])
	}

	compress, err := service.Compress(withIDs(request, "operation-compress", "result-compress"))
	if err != nil {
		t.Fatalf("compress failed: %v", err)
	}
	if compress.OutputObjects[0].Metadata["derived"] != true || compress.OutputObjects[0].Metadata["compressed"] != true {
		t.Fatalf("compress did not mark derived compressed output: %#v", compress.OutputObjects[0].Metadata)
	}
	if !containsString(compress.Warnings, WarningCompressionCannotCreateTruth) {
		t.Fatalf("compress missing doctrine warning: %#v", compress.Warnings)
	}

	derive, err := service.Derive(withIDs(request, "operation-derive", "result-derive"))
	if err != nil {
		t.Fatalf("derive failed: %v", err)
	}
	if derive.OutputObjects[0].Metadata["derived"] != true || len(derive.OutputObjects[0].SourceObjectRefs) < 2 {
		t.Fatalf("derive did not cite source objects: %#v", derive.OutputObjects[0])
	}
}

func TestContradictSupersedePromoteDemoteExpireProduceExplicitResults(t *testing.T) {
	service := NewSemanticAlgebraService()
	a := testSemanticObject("semantic-a", "old evidence", "exhibit-a")
	b := testSemanticObject("semantic-b", "new evidence", "exhibit-b")
	request := OperationRequest{
		WorkspaceID:  "workspace-a",
		CaseID:       "case-1",
		InputObjects: []SemanticObject{a, b},
		Parameters: map[string]any{
			"exhibit_a_id": "exhibit-a",
			"exhibit_b_id": "exhibit-b",
			"reason":       "new evidence supersedes old evidence",
		},
		CreatedBy:    "operator",
		CreatedAt:    time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC),
		NextObjectID: sequenceObjectIDs("semantic-c", "semantic-s", "semantic-p", "semantic-d", "semantic-e"),
	}

	contradict, err := service.Contradict(withIDs(request, "operation-contradict", "result-contradict"))
	if err != nil {
		t.Fatalf("contradict failed: %v", err)
	}
	if len(contradict.RequestedSyscalls) != 1 || contradict.RequestedSyscalls[0].SyscallName != "court.register_contradiction" {
		t.Fatalf("contradict did not request courthouse syscall: %#v", contradict.RequestedSyscalls)
	}

	supersede, err := service.Supersede(withIDs(request, "operation-supersede", "result-supersede"))
	if err != nil {
		t.Fatalf("supersede failed: %v", err)
	}
	if len(supersede.RequestedSyscalls) != 1 || supersede.RequestedSyscalls[0].SyscallName != "court.register_supersession" {
		t.Fatalf("supersede did not request courthouse syscall: %#v", supersede.RequestedSyscalls)
	}

	for _, fn := range []func(OperationRequest) (SemanticTransformResult, error){
		service.Promote,
		service.Demote,
		service.Expire,
	} {
		result, err := fn(withIDs(request, "operation-state", "result-state"))
		if err != nil {
			t.Fatalf("state operator failed: %v", err)
		}
		if len(result.OutputObjects) != 1 || len(result.OutputObjects[0].ProvenanceRefs) == 0 {
			t.Fatalf("state operator did not preserve provenance: %#v", result)
		}
	}
}

func fixedObjectID(id string) func() string {
	return func() string { return id }
}

func sequenceObjectIDs(ids ...string) func() string {
	index := 0
	return func() string {
		if index >= len(ids) {
			return ids[len(ids)-1]
		}
		value := ids[index]
		index++
		return value
	}
}

func withIDs(request OperationRequest, operationID, resultID string) OperationRequest {
	request.OperationID = operationID
	request.ResultID = resultID
	return request
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
