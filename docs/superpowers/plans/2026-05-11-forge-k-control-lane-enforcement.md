# FORGE-K Control Lane Enforcement Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Turn on the first FORGE-K partial live enforcement surface by hardening existing Control Lane validation actions with explicit activation metadata, no-effect guarantees, and documentation.

**Architecture:** Keep live authority in `services/core/internal/aios/controllane`. Add a small live-owned activation metadata helper and thread it into existing validation summaries for `VALIDATE_REF_SHAPE`, `COMPARE_REF_SHAPE`, `VALIDATE_SEMANTIC_OPERATION`, and `VALIDATE_KV_IDENTITY`; do not import simulator services from `services/core/internal/forgek`. Shadow emission remains optional and best-effort.

**Tech Stack:** Go, existing Control Lane processor/tests, existing `refvalidation`, `semanticvalidation`, `kvidentity`, markdown docs.

---

## File Structure

- Create `services/core/internal/aios/controllane/forgek_activation.go`: live-owned constants and helper functions for activation/no-effect metadata.
- Create `services/core/internal/aios/controllane/forgek_activation_test.go`: focused tests for activation metadata and no-effect guarantees.
- Modify `services/core/internal/aios/controllane/ref_validation.go`: include activation metadata and bounded count fields in ref validation summaries.
- Modify `services/core/internal/aios/controllane/ref_shape_compare.go`: include activation metadata and bounded count fields in ref comparison summaries.
- Modify `services/core/internal/aios/controllane/semantic_operation_validation.go`: include activation metadata and bounded count fields in semantic operation summaries.
- Modify `services/core/internal/aios/controllane/kv_enforcement.go`: include activation metadata and bounded count fields for existing KV validation consistency.
- Modify validation tests in `services/core/internal/aios/controllane/*_test.go`: assert activation mode and no-effect fields.
- Modify docs: `docs/architecture/simulator_to_live_migration.md`, `docs/architecture/forge_k_operational_cutover_design.md`, `docs/architecture/control_lane_kernel.md`, and `docs/reviews/current_phase_status.md`.

## Task 1: Activation Metadata Helper

**Files:**
- Create: `services/core/internal/aios/controllane/forgek_activation.go`
- Create: `services/core/internal/aios/controllane/forgek_activation_test.go`

- [ ] **Step 1: Write the failing tests**

Create `services/core/internal/aios/controllane/forgek_activation_test.go`:

```go
package controllane

import "testing"

func TestForgeKActivationSummaryMarksPartialLiveEnforcement(t *testing.T) {
	summary := forgeKActivationSummary("VALIDATE_REF_SHAPE")
	if summary["mode"] != ForgeKActivationModePartialLiveEnforcement {
		t.Fatalf("mode=%#v, want %q", summary["mode"], ForgeKActivationModePartialLiveEnforcement)
	}
	if summary["liveOwner"] != "aios.controllane" {
		t.Fatalf("live owner=%#v, want aios.controllane", summary["liveOwner"])
	}
	if summary["simulatorAuthority"] != false || summary["liveKernelAuthority"] != false {
		t.Fatalf("activation summary claimed simulator/live kernel authority: %#v", summary)
	}
}

func TestForgeKNoEffectSummaryRejectsAuthorityFlags(t *testing.T) {
	summary := forgeKNoEffectSummary()
	for _, key := range []string{
		"memoryMutation",
		"runtimeMutation",
		"modelRuntimeCall",
		"evidenceAdmission",
		"contextCompilation",
		"gatewayExecution",
		"retrievalExecution",
		"liveAuthorityMigration",
	} {
		if summary[key] != false {
			t.Fatalf("%s=%#v, want false in %#v", key, summary[key], summary)
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
cd services/core && go test ./internal/aios/controllane -run 'TestForgeK(Activation|NoEffect)' -count=1
```

Expected: FAIL because `forgeKActivationSummary`, `forgeKNoEffectSummary`, and `ForgeKActivationModePartialLiveEnforcement` are undefined.

- [ ] **Step 3: Add minimal helper implementation**

Create `services/core/internal/aios/controllane/forgek_activation.go`:

```go
package controllane

const (
	ForgeKActivationModePartialLiveEnforcement = "partial-live-enforcement"
	ForgeKActivationOwnerControlLane           = "aios.controllane"
	ForgeKActivationPolicyVersion              = "phase-14f-control-lane-enforcement-v1"
)

func forgeKActivationSummary(action string) map[string]any {
	return map[string]any{
		"mode":               ForgeKActivationModePartialLiveEnforcement,
		"liveOwner":          ForgeKActivationOwnerControlLane,
		"action":             action,
		"policyVersion":      ForgeKActivationPolicyVersion,
		"simulatorAuthority": false,
		"liveKernelAuthority": false,
		"shadowAuthoritative": false,
	}
}

func forgeKNoEffectSummary() map[string]any {
	return map[string]any{
		"memoryMutation":         false,
		"runtimeMutation":        false,
		"modelRuntimeCall":       false,
		"evidenceAdmission":      false,
		"contextCompilation":     false,
		"gatewayExecution":       false,
		"retrievalExecution":     false,
		"liveAuthorityMigration": false,
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run:

```bash
cd services/core && go test ./internal/aios/controllane -run 'TestForgeK(Activation|NoEffect)' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/core/internal/aios/controllane/forgek_activation.go services/core/internal/aios/controllane/forgek_activation_test.go
git commit -m "feat: add forge-k control lane activation metadata"
```

## Task 2: Ref Validation Activation Summary

**Files:**
- Modify: `services/core/internal/aios/controllane/ref_validation.go`
- Modify: `services/core/internal/aios/controllane/ref_validation_test.go`

- [ ] **Step 1: Write the failing test assertion**

In `TestValidateRefShapeLiveSyscallSucceedsWithoutMemoryMutation`, after the existing mutation assertions, add:

```go
	activation, ok := res.StateSummary["forgeKActivation"].(map[string]any)
	if !ok {
		t.Fatalf("missing forgeKActivation summary: %#v", res.StateSummary)
	}
	if activation["mode"] != ForgeKActivationModePartialLiveEnforcement ||
		activation["action"] != string(domain.ActionValidateRefShape) ||
		activation["liveOwner"] != ForgeKActivationOwnerControlLane {
		t.Fatalf("unexpected activation summary: %#v", activation)
	}
	noEffect, ok := res.StateSummary["forgeKNoEffect"].(map[string]any)
	if !ok || noEffect["memoryMutation"] != false || noEffect["modelRuntimeCall"] != false || noEffect["gatewayExecution"] != false {
		t.Fatalf("unexpected no-effect summary: %#v", res.StateSummary["forgeKNoEffect"])
	}
	auditActivation := auditSink.Records[len(auditSink.Records)-1].RefShapeValidation["forgeKActivation"].(map[string]any)
	if auditActivation["mode"] != ForgeKActivationModePartialLiveEnforcement {
		t.Fatalf("audit activation summary missing partial enforcement: %#v", auditActivation)
	}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
cd services/core && go test ./internal/aios/controllane -run TestValidateRefShapeLiveSyscallSucceedsWithoutMemoryMutation -count=1
```

Expected: FAIL with missing `forgeKActivation` summary.

- [ ] **Step 3: Add activation metadata to ref validation summaries**

Modify `RefShapeValidationDecision.ToStateSummary` to include:

```go
		"forgeKActivation":       forgeKActivationSummary(string(domain.ActionValidateRefShape)),
		"forgeKNoEffect":         forgeKNoEffectSummary(),
```

Modify `RefShapeValidationDecision.ToAuditFields` to include:

```go
		"failureCount":           len(d.Failures),
		"warningCount":           len(d.Warnings),
		"forgeKActivation":       forgeKActivationSummary(string(domain.ActionValidateRefShape)),
		"forgeKNoEffect":         forgeKNoEffectSummary(),
```

- [ ] **Step 4: Run test to verify it passes**

Run:

```bash
cd services/core && go test ./internal/aios/controllane -run TestValidateRefShapeLiveSyscallSucceedsWithoutMemoryMutation -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/core/internal/aios/controllane/ref_validation.go services/core/internal/aios/controllane/ref_validation_test.go
git commit -m "feat: mark ref validation as forge-k partial enforcement"
```

## Task 3: Ref Comparison Activation Summary

**Files:**
- Modify: `services/core/internal/aios/controllane/ref_shape_compare.go`
- Modify: `services/core/internal/aios/controllane/ref_shape_compare_test.go`

- [ ] **Step 1: Write the failing test assertion**

In `TestCompareRefShapeReportsDriftWithoutLiveMutation`, after the existing state summary assertions, add:

```go
	activation, ok := res.StateSummary["forgeKActivation"].(map[string]any)
	if !ok {
		t.Fatalf("missing forgeKActivation summary: %#v", res.StateSummary)
	}
	if activation["mode"] != ForgeKActivationModePartialLiveEnforcement ||
		activation["action"] != string(domain.ActionCompareRefShape) {
		t.Fatalf("unexpected activation summary: %#v", activation)
	}
	noEffect, ok := res.StateSummary["forgeKNoEffect"].(map[string]any)
	if !ok || noEffect["memoryMutation"] != false || noEffect["retrievalExecution"] != false {
		t.Fatalf("unexpected no-effect summary: %#v", res.StateSummary["forgeKNoEffect"])
	}
	auditActivation := auditSink.Records[len(auditSink.Records)-1].RefShapeComparison["forgeKActivation"].(map[string]any)
	if auditActivation["mode"] != ForgeKActivationModePartialLiveEnforcement {
		t.Fatalf("audit activation summary missing partial enforcement: %#v", auditActivation)
	}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
cd services/core && go test ./internal/aios/controllane -run TestCompareRefShapeReportsDriftWithoutLiveMutation -count=1
```

Expected: FAIL with missing `forgeKActivation` summary.

- [ ] **Step 3: Add activation metadata to ref comparison summaries**

Modify `RefShapeComparisonDecision.ToStateSummary` to include:

```go
		"forgeKActivation":       forgeKActivationSummary(string(domain.ActionCompareRefShape)),
		"forgeKNoEffect":         forgeKNoEffectSummary(),
```

Modify `RefShapeComparisonDecision.ToAuditFields` to include:

```go
		"failureCount":           len(d.Failures),
		"warningCount":           len(d.Warnings),
		"forgeKActivation":       forgeKActivationSummary(string(domain.ActionCompareRefShape)),
		"forgeKNoEffect":         forgeKNoEffectSummary(),
```

- [ ] **Step 4: Run test to verify it passes**

Run:

```bash
cd services/core && go test ./internal/aios/controllane -run TestCompareRefShapeReportsDriftWithoutLiveMutation -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/core/internal/aios/controllane/ref_shape_compare.go services/core/internal/aios/controllane/ref_shape_compare_test.go
git commit -m "feat: mark ref comparison as forge-k partial enforcement"
```

## Task 4: Semantic Operation Activation And Authority Coverage

**Files:**
- Modify: `services/core/internal/aios/controllane/semantic_operation_validation.go`
- Modify: `services/core/internal/aios/controllane/semantic_operation_validation_test.go`

- [ ] **Step 1: Write failing tests**

Add this test to `semantic_operation_validation_test.go`:

```go
func TestValidateSemanticOperationRejectsAllForbiddenAuthorityClaims(t *testing.T) {
	for _, claim := range []string{
		"execute",
		"commit",
		"admit_evidence",
		"reject_evidence",
		"write_memory",
		"call_model",
		"call_modelruntime",
		"execute_tool",
		"run_retrieval",
		"run_search",
		"run_embeddings",
		"compile_context",
		"live_authority_migration",
	} {
		t.Run(claim, func(t *testing.T) {
			ctx := context.Background()
			k, store, _ := newTestKernel()
			req := validSemanticOperationRequest()
			req.ID = "semantic-op-forbidden-" + claim
			req.IdempotencyKey = req.ID
			req.Payload["claims"] = map[string]any{claim: true}

			res, err := k.Process(ctx, req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.Success {
				t.Fatalf("expected claim %q to fail", claim)
			}
			if res.DeterministicErrCode != domain.ErrInvalidPayload {
				t.Fatalf("expected invalid payload, got %s", res.DeterministicErrCode)
			}
			if _, ok := store.GetIdempotency(req.IdempotencyKey); ok {
				t.Fatal("failed validation must not persist idempotency state")
			}
		})
	}
}
```

In `TestValidateSemanticOperationSucceedsWithoutLiveMutation`, add:

```go
	activation, ok := res.StateSummary["forgeKActivation"].(map[string]any)
	if !ok {
		t.Fatalf("missing forgeKActivation summary: %#v", res.StateSummary)
	}
	if activation["mode"] != ForgeKActivationModePartialLiveEnforcement ||
		activation["action"] != string(domain.ActionValidateSemanticOperation) {
		t.Fatalf("unexpected activation summary: %#v", activation)
	}
	noEffect, ok := res.StateSummary["forgeKNoEffect"].(map[string]any)
	if !ok || noEffect["memoryMutation"] != false || noEffect["modelRuntimeCall"] != false || noEffect["contextCompilation"] != false {
		t.Fatalf("unexpected no-effect summary: %#v", res.StateSummary["forgeKNoEffect"])
	}
```

- [ ] **Step 2: Run tests to verify at least activation assertion fails**

Run:

```bash
cd services/core && go test ./internal/aios/controllane -run 'TestValidateSemanticOperation(Succeeds|RejectsAllForbiddenAuthorityClaims)' -count=1
```

Expected: FAIL because activation summary is missing. The all-claims test may already pass; keep it as regression coverage.

- [ ] **Step 3: Add activation metadata to semantic operation summaries**

Modify `SemanticOperationValidationDecision.ToStateSummary` to include:

```go
		"forgeKActivation":       forgeKActivationSummary(string(domain.ActionValidateSemanticOperation)),
		"forgeKNoEffect":         forgeKNoEffectSummary(),
```

Modify `SemanticOperationValidationDecision.ToAuditFields` to include:

```go
		"failureCount":           len(d.Failures),
		"warningCount":           len(d.Warnings),
		"forgeKActivation":       forgeKActivationSummary(string(domain.ActionValidateSemanticOperation)),
		"forgeKNoEffect":         forgeKNoEffectSummary(),
```

- [ ] **Step 4: Run tests to verify they pass**

Run:

```bash
cd services/core && go test ./internal/aios/controllane -run 'TestValidateSemanticOperation(Succeeds|RejectsAllForbiddenAuthorityClaims)' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/core/internal/aios/controllane/semantic_operation_validation.go services/core/internal/aios/controllane/semantic_operation_validation_test.go
git commit -m "feat: harden semantic operation forge-k enforcement"
```

## Task 5: KV Validation Consistency

**Files:**
- Modify: `services/core/internal/aios/controllane/kv_enforcement.go`
- Modify: `services/core/internal/aios/controllane/kv_enforcement_test.go`

- [ ] **Step 1: Find the existing accepted KV validation test**

Run:

```bash
rg -n "ValidateKVIdentity|KVIdentity.*Accepted|liveKVReuse|kv identity" services/core/internal/aios/controllane/kv_enforcement_test.go
```

Expected: output includes at least one accepted validation test and one live KV reuse rejection test.

- [ ] **Step 2: Add failing activation assertions to the accepted KV test**

In the accepted KV test, after the existing success/no-mutation assertions, add:

```go
	activation, ok := res.StateSummary["forgeKActivation"].(map[string]any)
	if !ok {
		t.Fatalf("missing forgeKActivation summary: %#v", res.StateSummary)
	}
	if activation["mode"] != ForgeKActivationModePartialLiveEnforcement ||
		activation["action"] != string(domain.ActionValidateKVIdentity) {
		t.Fatalf("unexpected activation summary: %#v", activation)
	}
	noEffect, ok := res.StateSummary["forgeKNoEffect"].(map[string]any)
	if !ok || noEffect["memoryMutation"] != false || noEffect["runtimeMutation"] != false {
		t.Fatalf("unexpected no-effect summary: %#v", res.StateSummary["forgeKNoEffect"])
	}
```

- [ ] **Step 3: Run test to verify it fails**

Run the KV accepted validation test:

```bash
cd services/core && go test ./internal/aios/controllane -run TestKVIdentityEnforcementAcceptsValidClaim -count=1
```

Expected: FAIL because activation summary is missing from KV state summary.

- [ ] **Step 4: Add activation metadata to KV enforcement summaries**

Modify `KVIdentityEnforcementDecision.ToStateSummary` to include:

```go
		"forgeKActivation":  forgeKActivationSummary(string(domain.ActionValidateKVIdentity)),
		"forgeKNoEffect":    forgeKNoEffectSummary(),
```

Modify `KVIdentityEnforcementDecision.ToAuditFields` to include:

```go
		"failureCount":     len(d.FailedGates),
		"warningCount":     len(d.Warnings),
		"forgeKActivation": forgeKActivationSummary(string(domain.ActionValidateKVIdentity)),
		"forgeKNoEffect":   forgeKNoEffectSummary(),
```

- [ ] **Step 5: Run KV tests**

Run:

```bash
cd services/core && go test ./internal/aios/controllane -run 'Test.*KVIdentity' -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/core/internal/aios/controllane/kv_enforcement.go services/core/internal/aios/controllane/kv_enforcement_test.go
git commit -m "feat: align kv validation with forge-k activation metadata"
```

## Task 6: Shadow Observer No-Effect Regression

**Files:**
- Modify: `services/core/internal/aios/controllane/shadow_validation_test.go`
- Modify: `services/core/internal/aios/controllane/shadow_validation.go` only if the test fails for the wrong reason.

- [ ] **Step 1: Add semantic-operation observer coverage**

Add this test to `shadow_validation_test.go`:

```go
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
```

- [ ] **Step 2: Run observer tests**

Run:

```bash
cd services/core && go test ./internal/aios/controllane -run TestControlLaneValidationObserver -count=1
```

Expected: PASS. If it fails because observer summary does not include operation type, update `controlLaneValidationShadowInput` to read it from `semanticOperationResult` or `operationType`.

- [ ] **Step 3: Commit**

```bash
git add services/core/internal/aios/controllane/shadow_validation.go services/core/internal/aios/controllane/shadow_validation_test.go
git commit -m "test: cover semantic operation validation shadow emission"
```

## Task 7: No Simulator Authority Import Guard

**Files:**
- Create: `services/core/internal/aios/controllane/forgek_activation_imports_test.go`

- [ ] **Step 1: Write the import guard test**

Create `services/core/internal/aios/controllane/forgek_activation_imports_test.go`:

```go
package controllane

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestControlLaneDoesNotImportForgeKSimulatorAuthority(t *testing.T) {
	_, thisFile, _, ok := runtimeCaller()
	if !ok {
		t.Fatal("failed to resolve test path")
	}
	dir := filepath.Dir(thisFile)
	forbidden := []string{
		"forge/projectforge/services/core/internal/forgek",
	}
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imp := range file.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)
			for _, bad := range forbidden {
				if importPath == bad || strings.HasPrefix(importPath, bad+"/") {
					t.Fatalf("%s imports forbidden simulator authority package %s", path, importPath)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk failed: %v", err)
	}
}

func runtimeCaller() (uintptr, string, int, bool) {
	return runtime.Caller(0)
}
```

- [ ] **Step 2: Run test**

Run:

```bash
cd services/core && go test ./internal/aios/controllane -run TestControlLaneDoesNotImportForgeKSimulatorAuthority -count=1
```

Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add services/core/internal/aios/controllane/forgek_activation_imports_test.go
git commit -m "test: guard control lane from forge-k simulator imports"
```

## Task 8: Documentation

**Files:**
- Modify: `docs/architecture/simulator_to_live_migration.md`
- Modify: `docs/architecture/forge_k_operational_cutover_design.md`
- Modify: `docs/architecture/control_lane_kernel.md`
- Modify: `docs/reviews/current_phase_status.md`

- [ ] **Step 1: Update simulator migration status**

In `docs/architecture/simulator_to_live_migration.md`, add a new status paragraph near the top:

```markdown
Phase 14F turns on the first explicit `[PARTIAL LIVE ENFORCEMENT]` FORGE-K activation mode through the live Control Lane. It adds activation/no-effect metadata to existing validation actions and keeps authority in `services/core/internal/aios/controllane`. It does not import FORGE-K simulator services into live authority, route semantic writes through `forgek.Kernel`, enable live KV reuse, call modelruntime, execute tools, run retrieval/search/embeddings, or write memory outside existing governed paths.
```

- [ ] **Step 2: Update operational cutover design**

In `docs/architecture/forge_k_operational_cutover_design.md`, add a “Phase 14F Implemented Seam” section:

```markdown
## Phase 14F Implemented Seam

Phase 14F marks Control Lane validation as the first explicit FORGE-K partial live enforcement mode. The live owner remains `services/core/internal/aios/controllane`; shared pure validators provide deterministic doctrine, and summaries expose activation/no-effect metadata for operators and tests.

This remains partial enforcement, not full FORGE-K live authority. It does not import simulator services, replace the Control Lane, enable live KV reuse, admit evidence through the simulator Courthouse, compile live prompts through the simulator Context Compiler, call modelruntime, execute tools, run retrieval/search/embeddings, or write semantic memory directly.
```

- [ ] **Step 3: Update Control Lane architecture**

In `docs/architecture/control_lane_kernel.md`, add a short subsection:

```markdown
## FORGE-K Partial Enforcement

Control Lane validation actions now expose FORGE-K partial live enforcement metadata. This means deterministic doctrine is enforced by the current live Control Lane owner, not by importing the simulator Kernel. Validation summaries include the activation mode and no-effect posture so callers and operator surfaces can distinguish partial enforcement from full live authority.
```

- [ ] **Step 4: Update current phase status**

In `docs/reviews/current_phase_status.md`, add a Phase 14F entry:

```markdown
- 2026-05-11: Phase 14F adds `[PARTIAL LIVE ENFORCEMENT]` FORGE-K activation metadata to live Control Lane validation actions. It keeps the live owner as Control Lane and does not import simulator services, enable live KV reuse, route writes through `forgek.Kernel`, call modelruntime, execute tools, run retrieval/search/embeddings, or write semantic memory directly.
```

- [ ] **Step 5: Commit docs**

```bash
git add docs/architecture/simulator_to_live_migration.md docs/architecture/forge_k_operational_cutover_design.md docs/architecture/control_lane_kernel.md docs/reviews/current_phase_status.md
git commit -m "docs: record forge-k partial live enforcement"
```

## Task 9: Verification

**Files:**
- No source edits unless verification exposes a scoped defect from previous tasks.

- [ ] **Step 1: Run focused Control Lane tests**

Run:

```bash
cd services/core && go test ./internal/aios/controllane -count=1
```

Expected: PASS.

- [ ] **Step 2: Run shared validator tests**

Run:

```bash
cd services/core && go test ./internal/refvalidation ./internal/semanticvalidation ./internal/kvidentity -count=1
```

Expected: PASS.

- [ ] **Step 3: Run forbidden import tests for touched authority packages**

Run:

```bash
cd services/core && go test ./internal/aios/controllane ./internal/forgekshadow ./internal/refvalidation ./internal/semanticvalidation -run 'Forbidden|Import|Authority' -count=1
```

Expected: PASS.

- [ ] **Step 4: Run full core tests if focused tests pass**

Run:

```bash
cd services/core && go test ./...
```

Expected: PASS.

- [ ] **Step 5: Confirm verification state**

Run:

```bash
git status --short services/core/internal/aios/controllane services/core/internal/refvalidation services/core/internal/semanticvalidation services/core/internal/kvidentity docs/architecture docs/reviews
```

Expected: only intentional phase files remain staged or modified. If a scoped verification defect was fixed, commit the explicit files with a focused message; otherwise leave no verification-only commit behind.

## Self-Review

- Spec coverage: tasks cover activation metadata, no simulator authority, validation action hardening, shadow no-effect posture, docs, and verification.
- Placeholder scan: no disallowed placeholder markers or unspecific test steps are intentionally left in the task list.
- Type consistency: helper names are `forgeKActivationSummary`, `forgeKNoEffectSummary`, `ForgeKActivationModePartialLiveEnforcement`, `ForgeKActivationOwnerControlLane`, and `ForgeKActivationPolicyVersion`; all task snippets use those names consistently.
