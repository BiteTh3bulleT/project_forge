package court

import (
	"testing"
	"time"
)

func TestNewExhibitStartsSubmittedAndPreservesProvenance(t *testing.T) {
	createdAt := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	exhibit, err := NewExhibit(ExhibitInput{
		ExhibitID:      "exhibit-1",
		CaseID:         "case-1",
		WorkspaceID:    "workspace-a",
		SourceObjectID: "neuron-envelope-1",
		SubmittedBy:    "operator",
		SourceType:     SourceTypeNeuronEnvelope,
		SourceRefs:     []string{"env-1"},
		ClaimRefs:      []string{"claim-1"},
		ContentSummary: "proposal summary",
		RawRef:         "artifact://env-1",
		CreatedAt:      createdAt,
		Metadata:       map[string]any{"kind": "proposal"},
	})
	if err != nil {
		t.Fatalf("new exhibit: %v", err)
	}

	if exhibit.AdmissibilityStatus != StatusSubmitted {
		t.Fatalf("expected submitted status, got %s", exhibit.AdmissibilityStatus)
	}
	if exhibit.CaseID != "case-1" || exhibit.WorkspaceID != "workspace-a" || exhibit.SourceRefs[0] != "env-1" {
		t.Fatalf("provenance not preserved: %#v", exhibit)
	}
	clone := exhibit.Clone()
	clone.SourceRefs[0] = "tampered"
	if exhibit.SourceRefs[0] == "tampered" {
		t.Fatal("exhibit clone did not protect source refs")
	}
}

func TestRulingCitesAffectedExhibitsAndReasoning(t *testing.T) {
	ruling, err := NewRuling(RulingInput{
		RulingID:            "ruling-1",
		CaseID:              "case-1",
		WorkspaceID:         "workspace-a",
		RulingType:          RulingAdmission,
		AdmittedExhibitRefs: []string{"exhibit-1"},
		RejectedExhibitRefs: []string{"exhibit-2"},
		ReasoningSummary:    "exhibit-1 has sufficient provenance",
		PolicyRefs:          []string{"policy.phase3"},
		CreatedBy:           "operator",
		CreatedAt:           time.Date(2026, 5, 3, 12, 1, 0, 0, time.UTC),
		Metadata:            map[string]any{"phase": "3"},
	})
	if err != nil {
		t.Fatalf("new ruling: %v", err)
	}
	if ruling.RulingType != RulingAdmission {
		t.Fatalf("unexpected ruling type %s", ruling.RulingType)
	}
	if ruling.AdmittedExhibitRefs[0] != "exhibit-1" || ruling.ReasoningSummary == "" {
		t.Fatalf("ruling did not cite exhibits/reasoning: %#v", ruling)
	}
}

func TestContradictionAndSupersessionModelsPreserveInspectability(t *testing.T) {
	contradiction, err := NewContradiction(ContradictionInput{
		ContradictionID:   "contradiction-1",
		CaseID:            "case-1",
		WorkspaceID:       "workspace-a",
		ExhibitAID:        "exhibit-a",
		ExhibitBID:        "exhibit-b",
		ContradictionType: "factual_conflict",
		Description:       "statements conflict",
		Severity:          "medium",
		CreatedAt:         time.Date(2026, 5, 3, 12, 2, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("new contradiction: %v", err)
	}
	if contradiction.Status != ContradictionOpen {
		t.Fatalf("expected open contradiction, got %s", contradiction.Status)
	}
	if contradiction.ExhibitAID == "" || contradiction.ExhibitBID == "" {
		t.Fatalf("contradiction did not preserve exhibit refs: %#v", contradiction)
	}

	supersession, err := NewSupersession(SupersessionInput{
		SupersessionID: "supersession-1",
		CaseID:         "case-1",
		WorkspaceID:    "workspace-a",
		OldObjectID:    "exhibit-old",
		NewObjectID:    "exhibit-new",
		Reason:         "newer evidence",
		CreatedAt:      time.Date(2026, 5, 3, 12, 3, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("new supersession: %v", err)
	}
	if supersession.OldObjectID != "exhibit-old" || supersession.NewObjectID != "exhibit-new" {
		t.Fatalf("supersession did not preserve refs: %#v", supersession)
	}
}
