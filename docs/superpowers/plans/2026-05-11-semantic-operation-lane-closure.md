# Phase 14H Semantic Operation Lane Closure Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the `VALIDATE_SEMANTIC_OPERATION` lane as a deterministic validation-only Control Lane surface with exhaustive negative-authority regression coverage.

**Architecture:** Keep live ownership in `services/core/internal/aios/controllane` and pure validation in `services/core/internal/semanticvalidation`. Harden claim normalization and no-effect tests only; do not add routes, mutation paths, simulator imports, gateway calls, retrieval calls, modelruntime calls, evidence admission, context compilation, or semantic memory writes.

**Tech Stack:** Go unit tests, existing AI-OS Control Lane processor, shared `semanticvalidation` package, Markdown status docs.

---

## File Structure

- Modify `services/core/internal/semanticvalidation/models.go`: expose a canonical forbidden authority claim list and use it for validation.
- Modify `services/core/internal/semanticvalidation/models_test.go`: add table-driven validator tests for normalized truthy/falsey forbidden claims.
- Modify `services/core/internal/aios/controllane/semantic_operation_validation.go`: reuse the canonical claim list through existing payload normalization.
- Modify `services/core/internal/aios/controllane/semantic_operation_validation_test.go`: add live Control Lane rejection/no-effect tests for normalized and mixed claims.
- Modify `docs/architecture/control_lane_kernel.md`: record Phase 14H lane closure.
- Modify `docs/architecture/simulator_to_live_migration.md`: record Phase 14H as validation-only lane closure.
- Modify `docs/reviews/current_phase_status.md`: add Phase 14H status and readiness note.

## Task 1: Canonical Forbidden Claim Contract

**Files:**
- Modify: `services/core/internal/semanticvalidation/models.go`
- Modify: `services/core/internal/semanticvalidation/models_test.go`

- [ ] **Step 1: Add failing validator tests**

Add tests proving every forbidden authority claim rejects after trim/lowercase normalization, and false/non-truthy claims do not reject by themselves.

- [ ] **Step 2: Run focused validator tests**

Run: `cd services/core && go test ./internal/semanticvalidation -run 'ForbiddenAuthority|ValidateOperation' -count=1`

Expected before implementation: failure if the canonical exported list is missing.

- [ ] **Step 3: Implement canonical claim list**

Add `ForbiddenAuthorityClaims()` and route `forbiddenAuthorityClaim` through the canonical normalized map.

- [ ] **Step 4: Run validator tests again**

Run: `cd services/core && go test ./internal/semanticvalidation -run 'ForbiddenAuthority|ValidateOperation' -count=1`

Expected: pass.

- [ ] **Step 5: Commit**

Commit message: `test: close semantic operation authority claim contract`

## Task 2: Live Control Lane Closure Tests

**Files:**
- Modify: `services/core/internal/aios/controllane/semantic_operation_validation.go`
- Modify: `services/core/internal/aios/controllane/semantic_operation_validation_test.go`

- [ ] **Step 1: Add live negative-authority tests**

Add tests proving normalized truthy forbidden claims reject through `Processor`, mixed safe/forbidden claims reject, false/non-truthy forbidden claims pass, and rejected claims preserve no-effect state/audit metadata.

- [ ] **Step 2: Run focused Control Lane tests**

Run: `cd services/core && go test ./internal/aios/controllane -run 'SemanticOperation|ForgeKActivationContract' -count=1`

Expected before implementation: failure if shared claim names or no-effect assertions are incomplete.

- [ ] **Step 3: Reuse canonical claim list**

Update the Control Lane test matrix to use `semanticvalidation.ForbiddenAuthorityClaims()`. Keep production behavior validation-only.

- [ ] **Step 4: Run focused Control Lane tests again**

Run: `cd services/core && go test ./internal/aios/controllane -run 'SemanticOperation|ForgeKActivationContract' -count=1`

Expected: pass.

- [ ] **Step 5: Commit**

Commit message: `test: close semantic operation control lane`

## Task 3: Phase 14H Docs

**Files:**
- Modify: `docs/architecture/control_lane_kernel.md`
- Modify: `docs/architecture/simulator_to_live_migration.md`
- Modify: `docs/reviews/current_phase_status.md`

- [ ] **Step 1: Update architecture docs**

Record Phase 14H as `LANE_CLOSED / VALIDATION_ONLY / NO_AUTHORITY_EXPANSION`.

- [ ] **Step 2: Update current phase status**

Add a Phase 14H row and readiness note. State clearly that lane closure does not mean semantic operation execution or FORGE-K live authority.

- [ ] **Step 3: Run markdown diff check**

Run: `git diff --check -- docs/architecture/control_lane_kernel.md docs/architecture/simulator_to_live_migration.md docs/reviews/current_phase_status.md`

Expected: no output.

- [ ] **Step 4: Commit**

Commit message: `docs: record semantic operation lane closure`

## Task 4: Verification

**Files:**
- Verify all files touched above.

- [ ] **Step 1: Run validator package**

Run: `cd services/core && go test ./internal/semanticvalidation -count=1`

Expected: pass.

- [ ] **Step 2: Run focused Control Lane package**

Run: `cd services/core && go test ./internal/aios/controllane -run 'SemanticOperation|ForgeKActivationContract' -count=1`

Expected: pass.

- [ ] **Step 3: Run combined packages**

Run: `cd services/core && go test ./internal/aios/controllane ./internal/semanticvalidation -count=1`

Expected: pass.

- [ ] **Step 4: Run full core suite**

Run: `cd services/core && go test ./...`

Expected: pass.

- [ ] **Step 5: Confirm scoped cleanliness**

Run: `git status --short services/core/internal/semanticvalidation services/core/internal/aios/controllane docs/architecture docs/reviews docs/superpowers`

Expected: no uncommitted Phase 14H files remain. Existing unrelated desktop/VM/runbook files may remain dirty.
