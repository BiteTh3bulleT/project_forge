# Phase 14J Ref Shape Comparison Lane Closure Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close `COMPARE_REF_SHAPE` as a deterministic, diagnostic-only, validation-only Control Lane surface.

**Architecture:** Keep the shared comparison code in `services/core/internal/refvalidation` pure and deterministic. Route live comparison enforcement through `services/core/internal/aios/controllane` before commit so rejected comparisons preserve no-effect state and audit metadata without importing FORGE-K simulator services.

**Tech Stack:** Go core packages, AI-OS Control Lane tests, Markdown architecture/status docs.

---

Design: `docs/superpowers/specs/2026-05-11-ref-shape-comparison-lane-closure-design.md`

## File Map

- `services/core/internal/refvalidation/compare.go`: shared deterministic comparison logic.
- `services/core/internal/refvalidation/compare_test.go`: shared comparison contract tests.
- `services/core/internal/aios/controllane/processor.go`: live Control Lane rejected-path enforcement.
- `services/core/internal/aios/controllane/ref_shape_compare_test.go`: live comparison no-effect tests.
- `docs/architecture/control_lane_kernel.md`: Control Lane phase status.
- `docs/architecture/simulator_to_live_migration.md`: simulator/live migration phase status.
- `docs/reviews/current_phase_status.md`: current phase matrix and validation record.

## Task 1: Shared Comparison Contract

**Files:**

- Modify: `services/core/internal/refvalidation/compare_test.go`

- [ ] **Step 1: Add deterministic match and candidate-failure tests**

Add tests proving exact matches are accepted and invalid candidate refs fail closed with the candidate failure gate:

```go
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
```

- [ ] **Step 2: Run focused refvalidation tests**

Run:

```bash
cd services/core && go test ./internal/refvalidation -run 'CompareRefShapes' -count=1
```

Expected: PASS.

- [ ] **Step 3: Commit shared comparison contract**

```bash
git add services/core/internal/refvalidation/compare_test.go
git commit -m "test: close ref shape comparison contract"
```

## Task 2: Live Control Lane Rejected-Path Contract

**Files:**

- Modify: `services/core/internal/aios/controllane/processor.go`
- Modify: `services/core/internal/aios/controllane/ref_shape_compare_test.go`

- [ ] **Step 1: Enforce comparison before commit**

In `Processor.Process`, add `RefShapeComparisonDecision` enforcement after payload validation and before dry-run/commit, mirroring `VALIDATE_REF_SHAPE`:

```go
var refShapeComparisonDecision RefShapeComparisonDecision
if req.Action == domain.ActionCompareRefShape {
	refShapeComparisonDecision = EnforceRefShapeComparison(req)
	if !refShapeComparisonDecision.Accepted {
		result.StateSummary = refShapeComparisonDecision.ToStateSummary()
		return p.reject(ctx, req, result, "ref_shape_comparison", []domain.SyscallError{refShapeComparisonDecision.ToSyscallError()}), nil
	}
	result.ValidationDetails = append(result.ValidationDetails, detailFor("ref_shape_comparison", nil))
}
```

Update the dry-run branch for `ActionCompareRefShape` to reuse the precomputed decision:

```go
} else if req.Action == domain.ActionCompareRefShape {
	if refShapeComparisonDecision.Decision == "" {
		refShapeComparisonDecision = EnforceRefShapeComparison(req)
	}
	result.StateSummary = refShapeComparisonDecision.ToStateSummary()
	result.StateSummary["dryRun"] = true
}
```

- [ ] **Step 2: Assert rejected comparison no-effect state and audit metadata**

In `ref_shape_compare_test.go`, update `TestCompareRefShapeRejectsInvalidObservedRefs` to capture `auditSink`, then add helper assertions:

```go
k, store, auditSink := newTestKernel()
...
assertRefShapeComparisonRejectedWithoutEffects(t, res, auditSink)
```

Add:

```go
func assertRefShapeComparisonRejectedWithoutEffects(t *testing.T, res domain.SyscallResult, auditSink *InMemoryAuditSink) {
	t.Helper()
	if len(res.CommittedObjectIDs) != 0 {
		t.Fatalf("rejected comparison committed ids: %v", res.CommittedObjectIDs)
	}
	if res.StateSummary["memoryMutation"] != false || res.StateSummary["runtimeMutation"] != false || res.StateSummary["liveAuthorityMigration"] != false {
		t.Fatalf("rejected comparison claimed mutation/authority migration: %#v", res.StateSummary)
	}
	assertForgeKValidationContract(t, string(domain.ActionCompareRefShape), res.StateSummary)
	nested, ok := res.StateSummary["refShapeComparison"].(map[string]any)
	if !ok {
		t.Fatalf("missing nested refShapeComparison summary: %#v", res.StateSummary)
	}
	assertForgeKValidationContract(t, string(domain.ActionCompareRefShape), nested)
	if nested["accepted"] != false || nested["decision"] != RefShapeCompareDecisionRejected {
		t.Fatalf("expected rejected nested decision, got %#v", nested)
	}
	if nested["memoryMutation"] != false || nested["runtimeMutation"] != false || nested["liveAuthorityMigration"] != false {
		t.Fatalf("nested comparison claimed mutation/authority migration: %#v", nested)
	}
	if len(auditSink.Records) == 0 {
		t.Fatal("expected rejected comparison audit record")
	}
	auditRecord := auditSink.Records[len(auditSink.Records)-1]
	if auditRecord.Success {
		t.Fatalf("expected rejected audit record, got %#v", auditRecord)
	}
	auditDecision := auditRecord.RefShapeComparison
	if auditDecision == nil {
		t.Fatalf("audit missing ref shape comparison summary: %#v", auditRecord)
	}
	assertForgeKValidationContract(t, string(domain.ActionCompareRefShape), auditDecision)
	if auditDecision["accepted"] != false || auditDecision["decision"] != RefShapeCompareDecisionRejected {
		t.Fatalf("expected rejected audit decision, got %#v", auditDecision)
	}
	if auditDecision["memoryMutation"] != false || auditDecision["runtimeMutation"] != false || auditDecision["liveAuthorityMigration"] != false {
		t.Fatalf("audit comparison claimed mutation/authority migration: %#v", auditDecision)
	}
}
```

- [ ] **Step 3: Run focused Control Lane tests**

Run:

```bash
cd services/core && go test ./internal/aios/controllane -run 'CompareRefShape|ForgeKActivationContract' -count=1
```

Expected: PASS.

- [ ] **Step 4: Commit live comparison closure**

```bash
git add services/core/internal/aios/controllane/processor.go services/core/internal/aios/controllane/ref_shape_compare_test.go
git commit -m "test: close ref shape comparison control lane"
```

## Task 3: Status Documentation

**Files:**

- Modify: `docs/architecture/control_lane_kernel.md`
- Modify: `docs/architecture/simulator_to_live_migration.md`
- Modify: `docs/reviews/current_phase_status.md`

- [ ] **Step 1: Record Phase 14J**

Add Phase 14J as `LANE_CLOSED / DIAGNOSTIC_ONLY / VALIDATION_ONLY / NO_AUTHORITY_EXPANSION`.

State explicitly:

- `COMPARE_REF_SHAPE` is diagnostic comparison only.
- Match and drift are accepted diagnostic outcomes.
- Rejected malformed comparisons preserve no-effect state/audit metadata.
- Comparison does not prove object truth.
- No evidence admission/rejection, semantic memory write, context compilation, retrieval/search/embedding execution, modelruntime call, gateway/tool execution, route/API change, simulator import, or FORGE-K live authority was introduced.

- [ ] **Step 2: Commit docs**

```bash
git add docs/architecture/control_lane_kernel.md docs/architecture/simulator_to_live_migration.md docs/reviews/current_phase_status.md
git commit -m "docs: record ref shape comparison lane closure"
```

## Task 4: Final Verification

- [ ] **Step 1: Run planned verification**

Run:

```bash
cd services/core && go test ./internal/refvalidation -run 'CompareRefShapes' -count=1
cd services/core && go test ./internal/aios/controllane -run 'CompareRefShape|ForgeKActivationContract' -count=1
cd services/core && go test ./internal/aios/controllane ./internal/refvalidation -count=1
cd services/core && go test ./...
git diff --check
git status --short
```

Expected: all Go tests and diff check pass. `git status --short` may still show unrelated pre-existing desktop/VM/runbook files.
