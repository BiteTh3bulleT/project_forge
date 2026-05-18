package forgekshadow

import (
	"errors"
	"strings"
	"testing"
)

func TestMemoryPalaceMirrorBuildsEvidenceRefsWithoutExecution(t *testing.T) {
	observation, _, err := normalizeRetrievalMetadataInput(RetrievalMetadataInput{
		WorkspaceID:       "ws-main",
		RequestID:         "request-a",
		CorrelationID:     "corr-a",
		RetrievalRunID:    "run-42",
		RetrievalResultID: "result-101",
		SourceType:        "memory_ref",
		SourceRefID:       "note-7",
		ResultCount:       3,
		SelectedCount:     1,
	}, fixedNow(), "retrieval-observation-1")
	if err != nil {
		t.Fatalf("normalize retrieval metadata: %v", err)
	}

	report, err := BuildMemoryPalaceMirrorFromRetrievalMetadata(observation)
	if err != nil {
		t.Fatalf("build memory palace mirror: %v", err)
	}
	if report.SchemaVersion != MemoryPalaceMirrorSchemaVersion || report.MirrorID != "retrieval-observation-1:memory_palace_mirror" {
		t.Fatalf("unexpected mirror identity: %#v", report)
	}
	if report.WorkspaceID != "ws-main" || report.RequestID != "request-a" || report.CorrelationID != "corr-a" {
		t.Fatalf("mirror lost trace identity: %#v", report)
	}
	if len(report.EvidenceRefs) != 4 {
		t.Fatalf("evidence refs=%d, want 4: %#v", len(report.EvidenceRefs), report.EvidenceRefs)
	}
	if !hasMirrorRef(report, "diagnostic_report", "retrieval-observation-1") ||
		!hasMirrorRef(report, "retrieval_run", "run-42") ||
		!hasMirrorRef(report, "retrieval_result", "result-101") ||
		!hasMirrorRef(report, "memory_note", "note-7") {
		t.Fatalf("mirror did not produce expected refs: %#v", report.EvidenceRefs)
	}
	assertMemoryPalaceMirrorNoForbiddenEffects(t, report)
}

func TestMemoryPalaceMirrorOmitsEmptyOptionalRefs(t *testing.T) {
	observation, _, err := normalizeRetrievalMetadataInput(RetrievalMetadataInput{
		WorkspaceID:    "ws-main",
		RetrievalRunID: "run-42",
	}, fixedNow(), "retrieval-observation-2")
	if err != nil {
		t.Fatalf("normalize retrieval metadata: %v", err)
	}

	report, err := BuildMemoryPalaceMirrorFromRetrievalMetadata(observation)
	if err != nil {
		t.Fatalf("build memory palace mirror: %v", err)
	}
	if len(report.EvidenceRefs) != 2 {
		t.Fatalf("expected diagnostic and run refs only, got %#v", report.EvidenceRefs)
	}
	if !hasMirrorRef(report, "diagnostic_report", "retrieval-observation-2") || !hasMirrorRef(report, "retrieval_run", "run-42") {
		t.Fatalf("mirror did not retain required refs: %#v", report.EvidenceRefs)
	}
	assertMemoryPalaceMirrorNoForbiddenEffects(t, report)
}

func TestMemoryPalaceMirrorRejectsUnsafeRefs(t *testing.T) {
	observation, _, err := normalizeRetrievalMetadataInput(RetrievalMetadataInput{
		WorkspaceID: "ws-main",
	}, fixedNow(), "retrieval-observation-3")
	if err != nil {
		t.Fatalf("normalize retrieval metadata: %v", err)
	}
	observation.RetrievalRunID = "secret-run"

	_, err = BuildMemoryPalaceMirrorFromRetrievalMetadata(observation)
	if !errors.Is(err, ErrUnsafeMetadata) {
		t.Fatalf("expected unsafe metadata rejection, got %v", err)
	}
}

func TestMemoryPalaceMirrorRejectsMissingWorkspace(t *testing.T) {
	observation, _, err := normalizeRetrievalMetadataInput(RetrievalMetadataInput{
		RetrievalRunID: "run-42",
	}, fixedNow(), "retrieval-observation-4")
	if err != nil {
		t.Fatalf("normalize retrieval metadata: %v", err)
	}

	_, err = BuildMemoryPalaceMirrorFromRetrievalMetadata(observation)
	if !errors.Is(err, ErrUnsafeMetadata) || !strings.Contains(err.Error(), "workspace_id") {
		t.Fatalf("expected workspace rejection, got %v", err)
	}
}

func hasMirrorRef(report MemoryPalaceMirrorReport, refType, refID string) bool {
	for _, ref := range report.EvidenceRefs {
		if ref.RefType == refType && ref.RefID == refID && ref.WorkspaceID == report.WorkspaceID {
			return true
		}
	}
	return false
}

func assertMemoryPalaceMirrorNoForbiddenEffects(t *testing.T, report MemoryPalaceMirrorReport) {
	t.Helper()
	if !report.DiagnosticOnly ||
		report.RetrievalExecution ||
		report.SearchExecution ||
		report.EmbeddingExecution ||
		report.MemoryMutation ||
		report.EvidenceAdmission ||
		report.ContextCompilation ||
		report.ModelRuntimeCall ||
		report.LiveAuthorityMigration ||
		report.SimulatorAuthority {
		t.Fatalf("memory palace mirror claimed forbidden effects: %#v", report)
	}
}
