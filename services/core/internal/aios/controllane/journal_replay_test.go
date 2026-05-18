package controllane

import (
	"testing"

	"forge/projectforge/services/core/internal/aios/domain"
)

func TestBuildJournalReplayReportProducesStableHashChain(t *testing.T) {
	events := []domain.JournalEvent{
		testReplayEvent("evt-b", 200, map[string]any{"action": "CREATE_LINK", "committedObjectIds": []any{"link-a"}}),
		testReplayEvent("evt-a", 100, map[string]any{"action": "CREATE_NOTE", "committedObjectIds": []any{"note-a"}}),
	}

	report := BuildJournalReplayReport(events, "")
	if !report.Passed {
		t.Fatalf("expected replay report to pass: %#v", report.Mismatches)
	}
	if report.EventCount != 2 || len(report.Records) != 2 {
		t.Fatalf("unexpected record count: %#v", report)
	}
	if report.Records[0].EventID != "evt-a" || report.Records[1].EventID != "evt-b" {
		t.Fatalf("events not sorted deterministically: %#v", report.Records)
	}
	if report.Records[0].PreviousHash != "" {
		t.Fatalf("first record should not have previous hash: %#v", report.Records[0])
	}
	if report.Records[1].PreviousHash != report.Records[0].EventHash {
		t.Fatalf("hash chain not linked: %#v", report.Records)
	}
	if report.HeadHash != report.Records[1].EventHash {
		t.Fatalf("head hash mismatch: %#v", report)
	}

	reordered := BuildJournalReplayReport([]domain.JournalEvent{events[1], events[0]}, "")
	if reordered.HeadHash != report.HeadHash {
		t.Fatalf("same event set should produce same head hash, got %q vs %q", reordered.HeadHash, report.HeadHash)
	}
}

func TestBuildJournalReplayReportDetectsHeadMismatch(t *testing.T) {
	report := BuildJournalReplayReport([]domain.JournalEvent{
		testReplayEvent("evt-a", 100, map[string]any{"action": "CREATE_NOTE"}),
	}, "sha256:not-the-head")

	if report.Passed {
		t.Fatal("expected replay report to fail for head mismatch")
	}
	if len(report.Mismatches) != 1 || report.Mismatches[0].Field != "headHash" {
		t.Fatalf("expected headHash mismatch, got %#v", report.Mismatches)
	}
	if report.Mismatches[0].Actual != report.HeadHash {
		t.Fatalf("mismatch should include actual head hash: %#v", report.Mismatches[0])
	}
}

func TestBuildJournalReplayReportPayloadHashChangesWithPayloadOnly(t *testing.T) {
	base := testReplayEvent("evt-a", 100, map[string]any{"action": "CREATE_NOTE", "committedObjectIds": []any{"note-a"}})
	changed := base
	changed.Payload = map[string]any{"action": "CREATE_NOTE", "committedObjectIds": []any{"note-b"}}

	baseReport := BuildJournalReplayReport([]domain.JournalEvent{base}, "")
	changedReport := BuildJournalReplayReport([]domain.JournalEvent{changed}, "")
	if baseReport.Records[0].PayloadHash == changedReport.Records[0].PayloadHash {
		t.Fatal("payload hash should change when payload changes")
	}
	if baseReport.HeadHash == changedReport.HeadHash {
		t.Fatal("event hash should change when payload hash changes")
	}
}

func testReplayEvent(id string, timestamp int64, payload map[string]any) domain.JournalEvent {
	return domain.JournalEvent{
		ID:        id,
		Type:      "semantic_syscall.create_note",
		Timestamp: timestamp,
		Source:    "forge_kernel",
		Scope:     domain.ForgeScope{WorkspaceID: "ws-main", LaneID: "control.semantic"},
		Payload:   payload,
		Provenance: domain.Provenance{
			Actor:     "operator",
			ActorType: "user",
			Source:    "ui",
			TraceID:   "trace-a",
		},
		CorrelationID: "corr-a",
	}
}
