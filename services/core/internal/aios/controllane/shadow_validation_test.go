package controllane

import (
	"context"
	"reflect"
	"testing"

	"forge/projectforge/services/core/internal/aios/domain"
	"forge/projectforge/services/core/internal/forgekshadow"
)

type captureControlLaneValidationObserver struct {
	inputs []forgekshadow.ControlLaneValidationInput
	panic  bool
}

func (o *captureControlLaneValidationObserver) ObserveControlLaneValidationBestEffort(_ context.Context, input forgekshadow.ControlLaneValidationInput) {
	if o.panic {
		panic("observer panic")
	}
	o.inputs = append(o.inputs, input)
}

func TestControlLaneValidationObserverCalledForValidateRefShapeDryRunSuccess(t *testing.T) {
	ctx := context.Background()
	observer := &captureControlLaneValidationObserver{}
	k := newTestKernelWithControlLaneValidationObserver(observer)
	req := validRefShapeRequest()
	req.ID = "shadow-ref-shape-dry-run"
	req.DryRun = true
	req.IdempotencyKey = ""

	res, err := k.Process(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected dry-run success, got %#v", res)
	}
	if len(observer.inputs) != 1 {
		t.Fatalf("observer calls=%d, want 1", len(observer.inputs))
	}
	input := observer.inputs[0]
	if input.Action != string(domain.ActionValidateRefShape) || input.ValidationKind != "ref_shape" {
		t.Fatalf("unexpected observer action/kind: %#v", input)
	}
	if !input.Passed || input.Decision != RefShapeDecisionAccepted {
		t.Fatalf("unexpected observer decision: %#v", input)
	}
	if input.WorkspaceID != "ws-main" || input.RequestID != req.ID || input.CorrelationID != req.CorrelationID {
		t.Fatalf("observer lost request identity: %#v", input)
	}
	if input.NormalizedRefCount != 2 {
		t.Fatalf("normalized ref count=%d, want 2", input.NormalizedRefCount)
	}
	if input.FailureCount != 0 || input.WarningCount != len(res.Warnings) {
		t.Fatalf("unexpected observer counts: %#v result warnings=%v", input, res.Warnings)
	}
	assertControlLaneValidationInputNoForbiddenEffects(t, input)
}

func TestControlLaneValidationObserverCalledForRejectedMalformedValidation(t *testing.T) {
	ctx := context.Background()
	observer := &captureControlLaneValidationObserver{}
	k := newTestKernelWithControlLaneValidationObserver(observer)
	req := validRefShapeRequest()
	req.ID = "shadow-ref-shape-malformed"
	delete(req.Payload, "refs")

	res, err := k.Process(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Success {
		t.Fatalf("expected malformed validation to fail")
	}
	if len(observer.inputs) != 1 {
		t.Fatalf("observer calls=%d, want 1", len(observer.inputs))
	}
	input := observer.inputs[0]
	if input.Action != string(domain.ActionValidateRefShape) || input.ValidationKind != "ref_shape" {
		t.Fatalf("unexpected observer action/kind: %#v", input)
	}
	if input.Passed || input.Decision != "rejected" {
		t.Fatalf("unexpected observer rejection summary: %#v", input)
	}
	if input.FailureCount != len(res.RejectedReasons) || input.FailureCount == 0 {
		t.Fatalf("unexpected failure count: input=%#v result=%#v", input, res)
	}
	assertControlLaneValidationInputNoForbiddenEffects(t, input)
}

func TestControlLaneValidationObserverCalledForSemanticOperationValidation(t *testing.T) {
	ctx := context.Background()
	observer := &captureControlLaneValidationObserver{}
	k := newTestKernelWithControlLaneValidationObserver(observer)
	req := validSemanticOperationRequest()
	req.ID = "shadow-semantic-operation"
	req.DryRun = true
	req.IdempotencyKey = ""

	res, err := k.Process(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected semantic operation validation success, got %#v", res)
	}
	if len(observer.inputs) != 1 {
		t.Fatalf("observer calls=%d, want 1", len(observer.inputs))
	}
	input := observer.inputs[0]
	if input.Action != string(domain.ActionValidateSemanticOperation) || input.ValidationKind != "semantic_operation" {
		t.Fatalf("unexpected observer action/kind: %#v", input)
	}
	if !input.Passed || input.Decision != SemanticOperationDecisionAccepted {
		t.Fatalf("unexpected observer decision: %#v", input)
	}
	if input.OperationType != "derive" {
		t.Fatalf("operation type=%q, want derive", input.OperationType)
	}
	assertControlLaneValidationInputNoForbiddenEffects(t, input)
}

func TestControlLaneValidationObserverNotCalledForNormalSemanticWrite(t *testing.T) {
	ctx := context.Background()
	observer := &captureControlLaneValidationObserver{}
	k := newTestKernelWithControlLaneValidationObserver(observer)
	req := validBaseRequest(domain.ActionCreateNote)
	req.ID = "shadow-normal-write"
	req.Payload = map[string]any{
		"id":      "shadow-normal-note",
		"type":    string(domain.NoteFact),
		"title":   "Normal",
		"content": "Write",
	}

	res, err := k.Process(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected normal write success, got %#v", res)
	}
	if len(observer.inputs) != 0 {
		t.Fatalf("observer should not be called for normal semantic write: %#v", observer.inputs)
	}
}

func TestControlLaneValidationObserverPanicDoesNotChangeResult(t *testing.T) {
	ctx := context.Background()
	req := validRefShapeRequest()
	req.ID = "shadow-observer-panic"
	req.DryRun = true
	req.IdempotencyKey = ""

	baselineKernel := newTestKernelWithControlLaneValidationObserver(nil)
	baseline, err := baselineKernel.Process(ctx, req)
	if err != nil {
		t.Fatalf("unexpected baseline error: %v", err)
	}

	observer := &captureControlLaneValidationObserver{panic: true}
	k := newTestKernelWithControlLaneValidationObserver(observer)
	got, err := k.Process(ctx, req)
	if err != nil {
		t.Fatalf("observer panic must not surface as process error: %v", err)
	}
	if !reflect.DeepEqual(got, baseline) {
		t.Fatalf("observer panic changed result:\n got: %#v\nwant: %#v", got, baseline)
	}
}

func newTestKernelWithControlLaneValidationObserver(observer ControlLaneValidationObserver) *Processor {
	store := NewInMemorySemanticStore()
	return NewProcessor(ProcessorOptions{
		Registry:                      NewStaticActionRegistry(),
		Validator:                     NewDeterministicValidator(),
		Capabilities:                  NewStaticCapabilityService(),
		ApprovalGate:                  NewStaticApprovalGate(),
		TxRunner:                      NewInMemoryTransactionRunner(store),
		NowMillis:                     func() int64 { return 1760000000000 },
		ControlLaneValidationObserver: observer,
	})
}

func assertControlLaneValidationInputNoForbiddenEffects(t *testing.T, input forgekshadow.ControlLaneValidationInput) {
	t.Helper()
	if input.MemoryMutation || input.RuntimeMutation || input.ModelRuntimeCall || input.EvidenceAdmission || input.ContextCompilation || input.UserVisibleOutput || input.LiveAuthorityMigration {
		t.Fatalf("observer input claimed forbidden effects: %#v", input)
	}
}
