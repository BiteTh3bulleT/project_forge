package forgekshadow

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"forge/projectforge/services/core/internal/forgek/shadowharness"
)

func TestControlLaneValidationRequiresGlobalAndValidationFlags(t *testing.T) {
	cases := []struct {
		name string
		cfg  Config
		want int
	}{
		{"both disabled", Config{Enabled: false, ControlLaneValidationEnabled: false}, 0},
		{"global disabled validation enabled", Config{Enabled: false, ControlLaneValidationEnabled: true}, 0},
		{"global enabled validation disabled", Config{Enabled: true, ControlLaneValidationEnabled: false}, 0},
		{"both enabled", Config{Enabled: true, ControlLaneValidationEnabled: true}, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			observer := NewObserverWithSink(tc.cfg, nil, fixedNow)
			if err := observer.ObserveControlLaneValidation(context.Background(), sampleControlLaneValidationInput()); err != nil {
				t.Fatalf("observe control lane validation: %v", err)
			}
			if got := len(observer.Reports()); got != tc.want {
				t.Fatalf("reports=%d, want %d", got, tc.want)
			}
		})
	}
}

func TestControlLaneValidationStoresDiagnosticOnlySummary(t *testing.T) {
	observer := NewObserverWithSink(Config{Enabled: true, ControlLaneValidationEnabled: true}, nil, fixedNow)
	input := sampleControlLaneValidationInput()
	input.Action = "COMPARE_REF_SHAPE"
	input.ValidationKind = "ref_shape_comparison"
	input.Decision = "drift"
	input.Passed = true
	input.Match = false
	input.AddedRefCount = 1
	input.RemovedRefCount = 1
	input.UnchangedRefCount = 2
	if err := observer.ObserveControlLaneValidation(context.Background(), input); err != nil {
		t.Fatalf("observe control lane validation: %v", err)
	}
	reports := observer.Reports()
	if len(reports) != 1 {
		t.Fatalf("expected one report, got %d", len(reports))
	}
	validation := reports[0].ControlLaneValidation
	if validation == nil {
		t.Fatal("expected typed control lane validation observation")
	}
	if validation.Action != "COMPARE_REF_SHAPE" || validation.Decision != "drift" || !validation.Passed {
		t.Fatalf("unexpected validation observation: %#v", validation)
	}
	if validation.MemoryMutation || validation.RuntimeMutation || validation.EvidenceAdmission || validation.ContextCompilation || validation.UserVisibleOutput || validation.LiveAuthorityMigration {
		t.Fatalf("validation observation claimed forbidden effects: %#v", validation)
	}
	if validation.AddedRefCount != 1 || validation.RemovedRefCount != 1 || validation.UnchangedRefCount != 2 {
		t.Fatalf("unexpected drift counts: %#v", validation)
	}
	if !reports[0].Comparison.NoEffectVerified {
		t.Fatalf("diagnostic report lost no-effect verification: %#v", reports[0].Comparison)
	}
	serialized := strings.ToLower(mustJSONForTest(t, reports[0]))
	for _, forbidden := range []string{"prompt", "completion", "request_body", "response_body", "source_text", "chunk_text", "memory_content", "token", "secret"} {
		if strings.Contains(serialized, forbidden) {
			t.Fatalf("control lane validation leaked forbidden fragment %q in %s", forbidden, serialized)
		}
	}
}

func TestControlLaneValidationRejectsForbiddenEffects(t *testing.T) {
	observer := NewObserverWithSink(Config{Enabled: true, ControlLaneValidationEnabled: true}, nil, fixedNow)
	input := sampleControlLaneValidationInput()
	input.MemoryMutation = true
	err := observer.ObserveControlLaneValidation(context.Background(), input)
	if !errors.Is(err, ErrPolicyRejected) {
		t.Fatalf("expected policy rejection, got %v", err)
	}
	if reports := observer.Reports(); len(reports) != 0 {
		t.Fatalf("forbidden effects stored reports: %#v", reports)
	}
}

func TestControlLaneValidationRejectsUnsafeMetadata(t *testing.T) {
	observer := NewObserverWithSink(Config{Enabled: true, ControlLaneValidationEnabled: true}, nil, fixedNow)
	input := sampleControlLaneValidationInput()
	input.Metadata = map[string]any{"prompt": "must not store"}
	err := observer.ObserveControlLaneValidation(context.Background(), input)
	if !errors.Is(err, ErrUnsafeMetadata) {
		t.Fatalf("expected unsafe metadata, got %v", err)
	}
	if reports := observer.Reports(); len(reports) != 0 {
		t.Fatalf("unsafe metadata stored reports: %#v", reports)
	}
}

func TestControlLaneValidationRejectsSideEffectfulShadowPolicy(t *testing.T) {
	observer := NewObserverWithSink(Config{Enabled: true, ControlLaneValidationEnabled: true}, nil, fixedNow)
	observer.policy.AllowControllaneMutations = true
	err := observer.ObserveControlLaneValidation(context.Background(), sampleControlLaneValidationInput())
	if !errors.Is(err, ErrPolicyRejected) {
		t.Fatalf("expected policy rejection, got %v", err)
	}
	if reports := observer.Reports(); len(reports) != 0 {
		t.Fatalf("side-effectful policy stored reports: %#v", reports)
	}

	observer = NewObserverWithSink(Config{Enabled: true, ControlLaneValidationEnabled: true}, nil, fixedNow)
	observer.policy = shadowharness.DefaultShadowHarnessPolicy()
	observer.policy.AllowUserVisibleOutput = true
	err = observer.ObserveControlLaneValidation(context.Background(), sampleControlLaneValidationInput())
	if !errors.Is(err, ErrPolicyRejected) {
		t.Fatalf("expected user-visible-output policy rejection, got %v", err)
	}
}

func sampleControlLaneValidationInput() ControlLaneValidationInput {
	return ControlLaneValidationInput{
		WorkspaceID:        "workspace-a",
		RequestID:          "request-a",
		CorrelationID:      "corr-a",
		Action:             "VALIDATE_SEMANTIC_OPERATION",
		ValidationKind:     "semantic_operation",
		Decision:           "accepted",
		Passed:             true,
		OperationType:      "derive",
		NormalizedRefCount: 2,
		FailureCount:       0,
		WarningCount:       0,
		Duration:           3 * time.Millisecond,
		Metadata:           map[string]any{"phase": "14d"},
	}
}
