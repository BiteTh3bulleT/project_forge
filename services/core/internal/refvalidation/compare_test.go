package refvalidation

import "testing"

func TestCompareRefShapesReportsDeterministicMatch(t *testing.T) {
	res := CompareRefShapes(ComparisonRequest{
		ResultID:    "compare-match",
		WorkspaceID: "ws-main",
		CandidateRefs: []ObjectRef{
			{RefType: "memory_note", RefID: "note-b"},
			{RefType: "memory_note", RefID: "note-a"},
		},
		ObservedRefs: []ObjectRef{
			{RefType: "memory_note", RefID: "note-a"},
			{RefType: "memory_note", RefID: "note-b"},
		},
	})
	if !res.Passed || !res.Match {
		t.Fatalf("expected comparison match, got %#v", res)
	}
	if len(res.AddedRefs) != 0 || len(res.RemovedRefs) != 0 {
		t.Fatalf("matched comparison reported drift: %#v", res)
	}
	if len(res.UnchangedRefs) != 2 || res.UnchangedRefs[0].RefID != "note-a" || res.UnchangedRefs[1].RefID != "note-b" {
		t.Fatalf("unchanged refs not deterministic: %#v", res.UnchangedRefs)
	}
	if res.MemoryMutation || res.RuntimeMutation || res.LiveAuthorityMigration {
		t.Fatalf("comparison claimed mutation or authority migration: %#v", res)
	}
}

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

func TestCompareRefShapesFailsClosedForInvalidCandidateRefs(t *testing.T) {
	res := CompareRefShapes(ComparisonRequest{
		ResultID:    "compare-bad-candidate",
		WorkspaceID: "ws-main",
		CandidateRefs: []ObjectRef{
			{RefType: "raw_prompt", RefID: "prompt-a"},
		},
		ObservedRefs: []ObjectRef{
			{RefType: "memory_note", RefID: "note-a"},
		},
	})
	if res.Passed {
		t.Fatalf("expected invalid candidate refs to fail")
	}
	if !hasFailure(res.Failures, GateCandidateRefs) {
		t.Fatalf("missing candidate failure gate: %#v", res.Failures)
	}
	if len(res.AddedRefs) != 0 || len(res.RemovedRefs) != 0 || len(res.UnchangedRefs) != 0 {
		t.Fatalf("failed comparison should not emit drift sets: %#v", res)
	}
	if res.MemoryMutation || res.RuntimeMutation || res.LiveAuthorityMigration {
		t.Fatalf("failed comparison claimed mutation or authority migration: %#v", res)
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
	if !hasFailure(res.Failures, GateObservedRefs) {
		t.Fatalf("missing observed failure gate: %#v", res.Failures)
	}
	if len(res.AddedRefs) != 0 || len(res.RemovedRefs) != 0 || len(res.UnchangedRefs) != 0 {
		t.Fatalf("failed comparison should not emit drift sets: %#v", res)
	}
	if res.MemoryMutation || res.RuntimeMutation || res.LiveAuthorityMigration {
		t.Fatalf("failed comparison claimed mutation or authority migration: %#v", res)
	}
}
