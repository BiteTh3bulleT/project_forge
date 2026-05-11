# FORGE-K Control Lane Contract Matrix Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add Phase 14G contract tests proving every live Control Lane validation action exposes the same FORGE-K partial-enforcement/no-effect posture.

**Architecture:** Keep the live owner as `services/core/internal/aios/controllane`. Add table-driven tests around existing `Processor` request builders and audit records, then update docs to describe Phase 14G as hardening only. No new route, mutation path, validation action, simulator import, modelruntime call, live KV reuse, retrieval, context compilation, evidence admission, or memory write is introduced.

**Tech Stack:** Go unit tests, existing Control Lane test helpers, Markdown docs.

---

## File Structure

- Create `services/core/internal/aios/controllane/forgek_activation_contract_test.go`: table-driven processor-level activation/no-effect contract tests for all existing validation actions.
- Modify `docs/architecture/simulator_to_live_migration.md`: add Phase 14G as contract hardening.
- Modify `docs/architecture/control_lane_kernel.md`: add Phase 14G contract matrix note.
- Modify `docs/reviews/current_phase_status.md`: add Phase 14G status row and readiness note.

## Task 1: Control Lane Contract Matrix Test

**Files:**
- Create: `services/core/internal/aios/controllane/forgek_activation_contract_test.go`

- [ ] **Step 1: Write the failing table-driven test**

Create `services/core/internal/aios/controllane/forgek_activation_contract_test.go`:

```go
package controllane

import (
	"context"
	"testing"

	"forge/projectforge/services/core/internal/aios/domain"
)

func TestForgeKActivationContractForControlLaneValidationActions(t *testing.T) {
	cases := []struct {
		name      string
		action    domain.SemanticActionType
		request   func() domain.SyscallRequest
		audit     func(SyscallAuditRecord) map[string]any
		summaryID string
	}{
		{
			name:    "kv identity",
			action:  domain.ActionValidateKVIdentity,
			request: validKVIdentityRequest,
			audit: func(rec SyscallAuditRecord) map[string]any {
				return rec.KVIdentityEnforcement
			},
			summaryID: "kvIdentityEnforcement",
		},
		{
			name:    "ref shape",
			action:  domain.ActionValidateRefShape,
			request: validRefShapeRequest,
			audit: func(rec SyscallAuditRecord) map[string]any {
				return rec.RefShapeValidation
			},
			summaryID: "refShapeValidation",
		},
		{
			name:    "ref shape comparison",
			action:  domain.ActionCompareRefShape,
			request: validRefShapeCompareRequest,
			audit: func(rec SyscallAuditRecord) map[string]any {
				return rec.RefShapeComparison
			},
			summaryID: "refShapeComparison",
		},
		{
			name:    "semantic operation",
			action:  domain.ActionValidateSemanticOperation,
			request: validSemanticOperationRequest,
			audit: func(rec SyscallAuditRecord) map[string]any {
				return rec.SemanticOperationValidation
			},
			summaryID: "semanticOperationValidation",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			k, _, auditSink := newTestKernel()
			req := tc.request()
			req.ID = "forge-k-contract-" + string(tc.action)
			req.IdempotencyKey = ""

			res, err := k.Process(ctx, req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !res.Success {
				t.Fatalf("expected validation success, got %#v", res)
			}
			assertForgeKValidationContract(t, string(tc.action), res.StateSummary)

			nested, ok := res.StateSummary[tc.summaryID].(map[string]any)
			if !ok {
				t.Fatalf("missing nested summary %q: %#v", tc.summaryID, res.StateSummary)
			}
			assertForgeKValidationContract(t, string(tc.action), nested)

			if len(auditSink.Records) == 0 {
				t.Fatal("expected audit record")
			}
			auditSummary := tc.audit(auditSink.Records[len(auditSink.Records)-1])
			if auditSummary == nil {
				t.Fatalf("missing audit summary for %s", tc.action)
			}
			assertForgeKValidationContract(t, string(tc.action), auditSummary)
		})
	}
}

func assertForgeKValidationContract(t *testing.T, action string, summary map[string]any) {
	t.Helper()
	activation, ok := summary["forgeKActivation"].(map[string]any)
	if !ok {
		t.Fatalf("missing forgeKActivation in %#v", summary)
	}
	if activation["mode"] != ForgeKActivationModePartialLiveEnforcement {
		t.Fatalf("mode=%#v, want %q", activation["mode"], ForgeKActivationModePartialLiveEnforcement)
	}
	if activation["liveOwner"] != ForgeKActivationOwnerControlLane {
		t.Fatalf("liveOwner=%#v, want %q", activation["liveOwner"], ForgeKActivationOwnerControlLane)
	}
	if activation["action"] != action {
		t.Fatalf("action=%#v, want %q", activation["action"], action)
	}
	if activation["simulatorAuthority"] != false || activation["liveKernelAuthority"] != false || activation["shadowAuthoritative"] != false {
		t.Fatalf("activation summary claimed forbidden authority: %#v", activation)
	}

	noEffect, ok := summary["forgeKNoEffect"].(map[string]any)
	if !ok {
		t.Fatalf("missing forgeKNoEffect in %#v", summary)
	}
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
		if noEffect[key] != false {
			t.Fatalf("%s=%#v, want false in %#v", key, noEffect[key], noEffect)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it passes against Phase 14F behavior**

Run:

```bash
cd services/core && go test ./internal/aios/controllane -run TestForgeKActivationContractForControlLaneValidationActions -count=1
```

Expected: PASS with the contract matrix covering all four existing validation actions.

- [ ] **Step 3: Commit**

```bash
git add services/core/internal/aios/controllane/forgek_activation_contract_test.go
git commit -m "test: enforce forge-k control lane activation contract"
```

## Task 2: Phase 14G Documentation

**Files:**
- Modify: `docs/architecture/simulator_to_live_migration.md`
- Modify: `docs/architecture/control_lane_kernel.md`
- Modify: `docs/reviews/current_phase_status.md`

- [ ] **Step 1: Update simulator migration doc**

In `docs/architecture/simulator_to_live_migration.md`, update the status line to include Phase 14G and add this paragraph after the Phase 14F paragraph:

```markdown
Phase 14G hardens Phase 14F with a Control Lane validation contract matrix. The tests prove every existing live validation action exposes the same activation/no-effect metadata in syscall state summaries and audit summaries. This remains hardening only: no new validation action, simulator service import, public API, route behavior change, live KV reuse, modelruntime call, retrieval/search/embedding execution, evidence admission, context compilation, or semantic memory write is added.
```

- [ ] **Step 2: Update Control Lane architecture doc**

In `docs/architecture/control_lane_kernel.md`, add this paragraph after the Phase 14F paragraph:

```markdown
Phase 14G adds a contract matrix test for the partial-enforcement validation actions. The matrix checks `VALIDATE_KV_IDENTITY`, `VALIDATE_REF_SHAPE`, `COMPARE_REF_SHAPE`, and `VALIDATE_SEMANTIC_OPERATION` through the live processor and audit path, proving the activation/no-effect posture is consistent across state and audit summaries.
```

- [ ] **Step 3: Update current phase status**

In `docs/reviews/current_phase_status.md`, add a Phase 14G row after the Phase 14F row:

```markdown
| 14G | Control Lane Activation Contract Matrix | IMPLEMENTED + TESTED; HARDENING_ONLY / PARTIAL_LIVE_ENFORCEMENT_CONTRACT / NO_AUTHORITY_EXPANSION | `services/core/internal/aios/controllane/forgek_activation_contract_test.go`, existing validation serializers, status docs. | Table-driven processor tests prove every live validation action exposes matching activation/no-effect metadata in state and audit summaries. Final validation commands are recorded below. | No new validation action, simulator import, public route/API, live KV reuse, modelruntime call, retrieval/search/embedding execution, evidence admission, context compilation, gateway/tool execution, semantic memory write, or FORGE-K live authority. |
```

Also add this readiness note:

```markdown
- Phase 14G hardens Phase 14F with a table-driven activation contract matrix across live Control Lane validation state and audit summaries.
```

- [ ] **Step 4: Commit docs**

```bash
git add docs/architecture/simulator_to_live_migration.md docs/architecture/control_lane_kernel.md docs/reviews/current_phase_status.md
git commit -m "docs: record forge-k control lane contract matrix"
```

## Task 3: Verification

**Files:**
- No source edits unless verification reveals a scoped defect in Phase 14G work.

- [ ] **Step 1: Run focused Control Lane tests**

Run:

```bash
cd services/core && go test ./internal/aios/controllane -count=1
```

Expected: PASS.

- [ ] **Step 2: Run authority/import/contract focused tests**

Run:

```bash
cd services/core && go test ./internal/aios/controllane ./internal/forgekshadow ./internal/refvalidation ./internal/semanticvalidation -run 'Forbidden|Import|Authority|Contract' -count=1
```

Expected: PASS.

- [ ] **Step 3: Run full core tests**

Run:

```bash
cd services/core && go test ./...
```

Expected: PASS.

- [ ] **Step 4: Confirm scoped status**

Run:

```bash
git status --short services/core/internal/aios/controllane docs/architecture docs/reviews docs/superpowers
```

Expected: no uncommitted Phase 14G files remain. Existing unrelated desktop/VM files outside this scope may still be dirty.

## Self-Review

- Spec coverage: the plan adds contract matrix tests, docs, and verification for all four existing validation actions.
- Placeholder scan: no disallowed placeholder markers or unspecific implementation steps are intentionally left.
- Type consistency: test helper names, action names, audit fields, and metadata keys match existing Control Lane code.
