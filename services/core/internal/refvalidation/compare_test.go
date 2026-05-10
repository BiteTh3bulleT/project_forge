package refvalidation

import "testing"

func TestCompareRefShapesReportsStableAddedRemovedAndUnchangedRefs(t *testing.T) {
	res := CompareRefShapes(ComparisonRequest{
		ResultID:    "compare-a",
		WorkspaceID: "ws-main",
		CandidateRefs: []ObjectRef{
			{RefType: "memory_note", RefID: "note-b"},
			{RefType: "memory_note", RefID: "note-a"},
		},
		ObservedRefs: []ObjectRef{
			{RefType: "memory_note", RefID: "note-b"},
			{RefType: "memory_note", RefID: "note-c"},
		},
	})
	if !res.Passed {
		t.Fatalf("expected comparison validation to pass, got %#v", res)
	}
	if res.Match {
		t.Fatalf("expected candidate and observed refs to differ")
	}
	if len(res.RemovedRefs) != 1 || res.RemovedRefs[0].RefID != "note-a" {
		t.Fatalf("removed refs mismatch: %#v", res.RemovedRefs)
	}
	if len(res.AddedRefs) != 1 || res.AddedRefs[0].RefID != "note-c" {
		t.Fatalf("added refs mismatch: %#v", res.AddedRefs)
	}
	if len(res.UnchangedRefs) != 1 || res.UnchangedRefs[0].RefID != "note-b" {
		t.Fatalf("unchanged refs mismatch: %#v", res.UnchangedRefs)
	}
	if res.MemoryMutation || res.RuntimeMutation || res.LiveAuthorityMigration {
		t.Fatalf("comparison claimed mutation or authority migration: %#v", res)
	}
}

func TestCompareRefShapesFailsClosedForInvalidObservedRefs(t *testing.T) {
	res := CompareRefShapes(ComparisonRequest{
		ResultID:    "compare-a",
		WorkspaceID: "ws-main",
		CandidateRefs: []ObjectRef{
			{RefType: "memory_note", RefID: "note-a"},
		},
		ObservedRefs: []ObjectRef{
			{RefType: "raw_prompt", RefID: "token=secret"},
		},
	})
	if res.Passed {
		t.Fatalf("expected invalid observed refs to fail")
	}
	if len(res.Failures) == 0 {
		t.Fatalf("expected comparison failures")
	}
}
