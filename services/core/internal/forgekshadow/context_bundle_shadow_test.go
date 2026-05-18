package forgekshadow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"forge/projectforge/services/core/internal/refvalidation"
)

func TestContextBundleShadowBuildsFromAcceptedAdmissionCandidate(t *testing.T) {
	observation, _, err := normalizeControlLaneValidationInput(ControlLaneValidationInput{
		WorkspaceID:        "ws-main",
		RequestID:          "request-a",
		CorrelationID:      "corr-a",
		Action:             "VALIDATE_ADMISSION_CANDIDATE",
		ValidationKind:     "admission_candidate",
		Decision:           "accepted",
		Passed:             true,
		NormalizedRefCount: 1,
		Duration:           2 * time.Millisecond,
		NormalizedRefs: []refvalidation.ObjectRef{
			{RefType: "memory_note", RefID: "note-7", WorkspaceID: "ws-main"},
		},
	}, fixedNow(), "validation-observation-1")
	if err != nil {
		t.Fatalf("normalize control lane validation: %v", err)
	}

	report, ok, err := BuildContextBundleShadowFromControlLaneValidation(observation)
	if err != nil {
		t.Fatalf("build context bundle shadow: %v", err)
	}
	if !ok {
		t.Fatalf("expected context bundle shadow to be created")
	}
	if report.SchemaVersion != ContextBundleShadowSchemaVersion || report.BundleID != "validation-observation-1:context_bundle_shadow" {
		t.Fatalf("unexpected context bundle identity: %#v", report)
	}
	if report.AdmissionStatus != "candidate_accepted_not_live_admitted" {
		t.Fatalf("context bundle shadow overclaimed admission status: %#v", report)
	}
	if len(report.EvidenceRefs) != 1 || report.EvidenceRefs[0].RefID != "note-7" {
		t.Fatalf("unexpected evidence refs: %#v", report.EvidenceRefs)
	}
	if len(report.Blocks) != 1 || len(report.Blocks[0].IncludedRefs) != 1 {
		t.Fatalf("unexpected context blocks: %#v", report.Blocks)
	}
	if report.BundleHash == "" {
		t.Fatalf("expected deterministic bundle hash")
	}
	assertContextBundleShadowNoForbiddenEffects(t, report)
}

func TestContextBundleShadowOnlyBuildsForAcceptedAdmissionCandidate(t *testing.T) {
	cases := []struct {
		name string
		mut  func(*ControlLaneValidationInput)
	}{
		{"wrong action", func(in *ControlLaneValidationInput) { in.Action = "VALIDATE_REF_SHAPE" }},
		{"wrong kind", func(in *ControlLaneValidationInput) { in.ValidationKind = "ref_shape" }},
		{"rejected", func(in *ControlLaneValidationInput) { in.Decision = "rejected" }},
		{"not passed", func(in *ControlLaneValidationInput) { in.Passed = false }},
		{"no refs", func(in *ControlLaneValidationInput) { in.NormalizedRefs = nil }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := acceptedAdmissionValidationInput()
			tc.mut(&input)
			observation, _, err := normalizeControlLaneValidationInput(input, fixedNow(), "validation-observation-skip")
			if err != nil {
				t.Fatalf("normalize control lane validation: %v", err)
			}
			_, ok, err := BuildContextBundleShadowFromControlLaneValidation(observation)
			if err != nil {
				t.Fatalf("build context bundle shadow: %v", err)
			}
			if ok {
				t.Fatalf("context bundle shadow should not be created for %s", tc.name)
			}
		})
	}
}

func TestContextBundleShadowRejectsUnsafeRefsAndMissingWorkspace(t *testing.T) {
	input := acceptedAdmissionValidationInput()
	input.NormalizedRefs[0].RefID = "secret-note"
	_, _, err := normalizeControlLaneValidationInput(input, fixedNow(), "validation-observation-unsafe")
	if !errors.Is(err, ErrUnsafeMetadata) {
		t.Fatalf("expected unsafe ref rejection, got %v", err)
	}

	input = acceptedAdmissionValidationInput()
	input.WorkspaceID = ""
	_, _, err = normalizeControlLaneValidationInput(input, fixedNow(), "validation-observation-no-workspace")
	if !errors.Is(err, ErrUnsafeMetadata) || !strings.Contains(err.Error(), "workspace_id") {
		t.Fatalf("expected workspace rejection, got %v", err)
	}
}

func TestContextBundleShadowDeterministicHashAndNoContentLeak(t *testing.T) {
	build := func() ContextBundleShadowReport {
		observation, _, err := normalizeControlLaneValidationInput(acceptedAdmissionValidationInput(), fixedNow(), "validation-observation-stable")
		if err != nil {
			t.Fatalf("normalize control lane validation: %v", err)
		}
		report, ok, err := BuildContextBundleShadowFromControlLaneValidation(observation)
		if err != nil {
			t.Fatalf("build context bundle shadow: %v", err)
		}
		if !ok {
			t.Fatalf("expected context bundle shadow")
		}
		return report
	}
	left := build()
	right := build()
	if left.BundleHash != right.BundleHash {
		t.Fatalf("bundle hash should be deterministic: %s != %s", left.BundleHash, right.BundleHash)
	}
	serialized := strings.ToLower(fmt.Sprint(left))
	for _, forbidden := range []string{"source text", "chunk text", "document content", "raw query", "prompt text", "completion", "model output", "memory content", "request body", "response body"} {
		if strings.Contains(serialized, forbidden) {
			t.Fatalf("context bundle shadow leaked forbidden fragment %q in %q", forbidden, serialized)
		}
	}
	data, err := json.Marshal(left)
	if err != nil {
		t.Fatalf("marshal context bundle shadow: %v", err)
	}
	if strings.Contains(strings.ToLower(string(data)), "candidate_accepted_not_live_admitted") != true {
		t.Fatalf("context bundle shadow should preserve non-admission status: %s", data)
	}
}

func TestContextBundleShadowObserverWiringAndAdvisory(t *testing.T) {
	observer := NewObserverWithSink(Config{Enabled: true, ControlLaneValidationEnabled: true, AdvisoryEnabled: true}, nil, fixedNow)
	if err := observer.ObserveControlLaneValidation(context.Background(), acceptedAdmissionValidationInput()); err != nil {
		t.Fatalf("observe control lane validation: %v", err)
	}
	report := observer.Reports()[0]
	if report.ContextBundleShadow == nil {
		t.Fatalf("expected context bundle shadow")
	}
	assertContextBundleShadowNoForbiddenEffects(t, *report.ContextBundleShadow)
	if report.Advisory == nil {
		t.Fatalf("expected advisory")
	}
	if report.Advisory.ContextCompilerAdvisory.Status != "shadow_context_bundle_created" ||
		report.Advisory.ContextCompilerAdvisory.BundleHash != report.ContextBundleShadow.BundleHash {
		t.Fatalf("advisory did not point at shadow bundle: advisory=%#v bundle=%#v", report.Advisory.ContextCompilerAdvisory, report.ContextBundleShadow)
	}
}

func acceptedAdmissionValidationInput() ControlLaneValidationInput {
	return ControlLaneValidationInput{
		WorkspaceID:        "ws-main",
		RequestID:          "request-a",
		CorrelationID:      "corr-a",
		Action:             "VALIDATE_ADMISSION_CANDIDATE",
		ValidationKind:     "admission_candidate",
		Decision:           "accepted",
		Passed:             true,
		NormalizedRefCount: 1,
		NormalizedRefs: []refvalidation.ObjectRef{
			{RefType: "memory_note", RefID: "note-7", WorkspaceID: "ws-main"},
		},
	}
}

func assertContextBundleShadowNoForbiddenEffects(t *testing.T, report ContextBundleShadowReport) {
	t.Helper()
	if !report.DiagnosticOnly ||
		!report.ShadowContextShapeGenerated ||
		report.LiveContextCompilation ||
		report.PromptAuthority ||
		report.ModelRuntimeCall ||
		report.RetrievalExecution ||
		report.SearchExecution ||
		report.EmbeddingExecution ||
		report.MemoryMutation ||
		report.EvidenceAdmission ||
		report.UserVisibleOutput ||
		report.LiveAuthorityMigration ||
		report.SimulatorAuthority {
		t.Fatalf("context bundle shadow claimed forbidden effects: %#v", report)
	}
}
