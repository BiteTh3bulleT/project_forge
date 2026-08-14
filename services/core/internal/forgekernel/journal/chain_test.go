package journal

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestPlanAppendProducesStableLinkedEntries(t *testing.T) {
	first, err := PlanAppend(Head{}, testInput("evt-1", 100))
	if err != nil {
		t.Fatal(err)
	}
	second, err := PlanAppend(headOf(first), testInput("evt-2", 200))
	if err != nil {
		t.Fatal(err)
	}
	if first.Sequence != 1 || first.PriorHash != "" {
		t.Fatalf("unexpected genesis: %#v", first)
	}
	if second.Sequence != 2 || second.PriorHash != first.Hash {
		t.Fatalf("second entry is not linked: %#v", second)
	}
	if second.Hash == first.Hash || !strings.HasPrefix(second.Hash, "sha256:") {
		t.Fatalf("unexpected second hash: %q", second.Hash)
	}
	if first.Hash != "sha256:e69828bc59d84f6d193cdb28dcd04b55eb547f3dd7cec13d7fe26ab6099f2979" {
		t.Fatalf("canonical v1 digest changed: %q", first.Hash)
	}
	replanned, err := PlanAppend(headOf(first), testInput("evt-2", 200))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(replanned, second) {
		t.Fatalf("append plan is not deterministic:\n%#v\n%#v", replanned, second)
	}

	report := Verify([]Entry{first, second}, ptrHead(headOf(second)))
	if !report.Passed || len(report.Issues) != 0 || report.Head != headOf(second) {
		t.Fatalf("valid chain rejected: %#v", report)
	}
}

func TestCanonicalJSONIsStableAndRejectsDuplicateKeys(t *testing.T) {
	left, err := HashJSON([]byte(` { "z": [3, 2, 1], "a": {"ok":true} } `))
	if err != nil {
		t.Fatal(err)
	}
	right, err := HashJSON([]byte(`{"a":{"ok":true},"z":[3,2,1]}`))
	if err != nil {
		t.Fatal(err)
	}
	if left != right {
		t.Fatalf("equivalent JSON hashes differ: %q vs %q", left, right)
	}
	if _, err := HashJSON([]byte(`{"action":"a","action":"b"}`)); !errors.Is(err, ErrDuplicateJSONKey) {
		t.Fatalf("expected duplicate key rejection, got %v", err)
	}
}

func TestVerifyDetectsPayloadTampering(t *testing.T) {
	chain := testChain(t, 3)
	chain[1].PayloadHash = mustHashJSON(t, `{"changed":true}`)

	report := Verify(chain, nil)
	assertIssue(t, report, IssueHashMismatch, 1)
}

func TestVerifyDetectsReorder(t *testing.T) {
	chain := testChain(t, 3)
	chain[1], chain[2] = chain[2], chain[1]

	report := Verify(chain, nil)
	assertIssue(t, report, IssueSequenceMismatch, 1)
	assertIssue(t, report, IssuePriorHashMismatch, 1)
}

func TestVerifyDetectsExplicitSequenceAndPriorHashForgery(t *testing.T) {
	chain := testChain(t, 3)
	chain[1].Sequence = 9
	chain[1].PriorHash = mustHashJSON(t, `{"not":"the prior entry"}`)
	chain[1].Hash, _ = HashEntry(chain[1])

	report := Verify(chain, nil)
	assertIssue(t, report, IssueSequenceMismatch, 1)
	assertIssue(t, report, IssuePriorHashMismatch, 1)
}

func TestVerifyDetectsDuplicateEvent(t *testing.T) {
	chain := testChain(t, 2)
	duplicate := chain[1]
	duplicate.Sequence = 3
	duplicate.PriorHash = chain[1].Hash
	duplicate.Hash, _ = HashEntry(duplicate)
	chain = append(chain, duplicate)

	report := Verify(chain, nil)
	assertIssue(t, report, IssueDuplicateEventID, 2)
}

func TestVerifyDetectsExactDuplicateEntry(t *testing.T) {
	chain := testChain(t, 2)
	chain = append(chain, chain[1])

	report := Verify(chain, nil)
	assertIssue(t, report, IssueDuplicateEventID, 2)
	assertIssue(t, report, IssueDuplicateHash, 2)
}

func TestVerifyDetectsRehashedDivergenceAgainstIndependentHead(t *testing.T) {
	chain := testChain(t, 3)
	expected := headOf(chain[2])

	chain[1].PayloadHash = mustHashJSON(t, `{"fork":true}`)
	chain[1].Hash, _ = HashEntry(chain[1])
	chain[2].PriorHash = chain[1].Hash
	chain[2].Hash, _ = HashEntry(chain[2])

	internalOnly := Verify(chain, nil)
	if !internalOnly.Passed {
		t.Fatalf("fully rehashed fork should require external head evidence: %#v", internalOnly.Issues)
	}
	report := Verify(chain, &expected)
	assertIssue(t, report, IssueHeadDivergence, len(chain))
}

func TestVerifyDetectsTruncationAgainstIndependentHead(t *testing.T) {
	chain := testChain(t, 3)
	expected := headOf(chain[2])

	report := Verify(chain[:2], &expected)
	assertIssue(t, report, IssueHeadDivergence, 2)
}

func TestPlanAppendRejectsInvalidAndAmbiguousInputs(t *testing.T) {
	input := testInput("evt-1", 100)
	input.PayloadHash = "abc"
	if _, err := PlanAppend(Head{}, input); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid digest rejection, got %v", err)
	}
	input = testInput("evt-1", 100)
	input.SelectedPaths = []string{"/safe", "/safe"}
	if _, err := PlanAppend(Head{}, input); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected duplicate path rejection, got %v", err)
	}
	if _, err := PlanAppend(Head{Sequence: 2}, testInput("evt-2", 200)); !errors.Is(err, ErrInvalidHead) {
		t.Fatalf("expected partial head rejection, got %v", err)
	}
}

func testChain(t *testing.T, count int) []Entry {
	t.Helper()
	entries := make([]Entry, 0, count)
	head := Head{}
	for i := 1; i <= count; i++ {
		entry, err := PlanAppend(head, testInput("evt-"+string(rune('0'+i)), int64(i*100)))
		if err != nil {
			t.Fatal(err)
		}
		entries = append(entries, entry)
		head = headOf(entry)
	}
	return entries
}

func testInput(id string, createdAt int64) AppendInput {
	return AppendInput{
		EventID: id, EventType: "semantic_syscall.create_note", Source: "forge_k.kernel",
		Actor: "operator", WorkspaceID: "ws-main", LaneID: "control.semantic",
		SelectedPaths: []string{"/notes"}, CorrelationID: "corr-1", TraceID: "trace-1",
		ProvenanceID: "prov-1", ProvenanceHash: mustTestHash(`{"actor":"operator"}`),
		PayloadHash: mustTestHash(`{"action":"CREATE_NOTE"}`), MetadataHash: mustTestHash(`{}`),
		ProposedBy: "user", CommittedBy: "forge_k.kernel", SyscallID: "syscall-" + id,
		AuditID: "audit-" + id, CreatedAt: createdAt,
	}
}

func mustTestHash(raw string) string {
	hash, err := HashJSON([]byte(raw))
	if err != nil {
		panic(err)
	}
	return hash
}

func mustHashJSON(t *testing.T, raw string) string {
	t.Helper()
	hash, err := HashJSON([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	return hash
}

func headOf(entry Entry) Head {
	return Head{Sequence: entry.Sequence, EventID: entry.EventID, Hash: entry.Hash}
}

func ptrHead(head Head) *Head { return &head }

func assertIssue(t *testing.T, report VerificationReport, code IssueCode, index int) {
	t.Helper()
	if report.Passed {
		t.Fatalf("expected verification failure for %s", code)
	}
	for _, issue := range report.Issues {
		if issue.Code == code && issue.Index == index {
			return
		}
	}
	t.Fatalf("missing issue %s at index %d: %#v", code, index, report.Issues)
}
