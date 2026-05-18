package contextattribution

import (
	"testing"

	"forge/projectforge/services/core/internal/refvalidation"
)

func TestValidateAttributionAcceptsAttributedSourceRefs(t *testing.T) {
	result := ValidateAttribution(AttributionRequest{
		ResultID:       "attr-1",
		WorkspaceID:    "ws-main",
		Query:          "What should FORGE remember about the active task?",
		ContextPurpose: "chat_turn",
		SourceRefs: []refvalidation.ObjectRef{
			{RefType: "memory_note", RefID: "note-1"},
			{RefType: "state_item", RefID: "state-1"},
		},
		SelectionReasons: map[string]string{
			"memory_note:note-1": "note directly matches the requested task",
			"state_item:state-1": "state item is current for this workspace",
		},
	})

	if !result.Passed {
		t.Fatalf("expected attribution validation to pass, got failures: %#v", result.Failures)
	}
	if len(result.NormalizedSourceRefs) != 2 {
		t.Fatalf("normalized refs=%#v, want 2", result.NormalizedSourceRefs)
	}
	if result.ContextCompilation || result.MemoryMutation || result.ModelRuntimeCall || result.GatewayExecution || result.LiveAuthorityMigration {
		t.Fatalf("attribution validation claimed forbidden effects: %#v", result)
	}
}

func TestValidateAttributionRejectsMissingSelectionReason(t *testing.T) {
	result := ValidateAttribution(AttributionRequest{
		ResultID:       "attr-2",
		WorkspaceID:    "ws-main",
		Query:          "Explain active task",
		ContextPurpose: "chat_turn",
		SourceRefs: []refvalidation.ObjectRef{
			{RefType: "memory_note", RefID: "note-1"},
		},
		SelectionReasons: map[string]string{},
	})

	if result.Passed {
		t.Fatalf("expected missing selection reason to fail: %#v", result)
	}
	if len(result.Failures) == 0 || result.Failures[0].Gate != GateSelectionReason {
		t.Fatalf("expected selection reason failure, got %#v", result.Failures)
	}
}

func TestValidateAttributionRejectsAuthorityClaims(t *testing.T) {
	result := ValidateAttribution(AttributionRequest{
		ResultID:       "attr-3",
		WorkspaceID:    "ws-main",
		Query:          "Explain active task",
		ContextPurpose: "chat_turn",
		SourceRefs: []refvalidation.ObjectRef{
			{RefType: "memory_note", RefID: "note-1"},
		},
		SelectionReasons: map[string]string{
			"memory_note:note-1": "note directly matches the requested task",
		},
		Claims: map[string]bool{"compile_context": true},
	})

	if result.Passed {
		t.Fatalf("expected authority claim to fail: %#v", result)
	}
	if len(result.Failures) == 0 || result.Failures[0].Gate != GateNoAuthorityClaim {
		t.Fatalf("expected authority claim failure, got %#v", result.Failures)
	}
}
