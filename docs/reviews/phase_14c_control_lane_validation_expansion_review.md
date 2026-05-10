# Phase 14C Control Lane Validation Expansion Review

Status: implemented and tested.

Scope: `PARTIAL LIVE VALIDATION / CONTROL_LANE / SHADOW_COMPARE / NO_AUTHORITY_REPLACEMENT`.

Date: 2026-05-09.

## Summary

Phase 14C expands the Phase 14B pattern in two narrow ways:

- `COMPARE_REF_SHAPE` compares candidate refs against observed refs as diagnostic shadow comparison.
- `VALIDATE_SEMANTIC_OPERATION` validates semantic operation shape before any future authority migration.

Both are live Control Lane validation seams. Neither imports FORGE-K simulator services into live authority, writes semantic memory, admits evidence, compiles context, executes retrieval/search/embeddings, calls modelruntime, executes tools, changes routes, or changes public APIs.

## What Changed

- Added `refvalidation.CompareRefShapes`.
- Added shared pure package `services/core/internal/semanticvalidation`.
- Added live Control Lane action `COMPARE_REF_SHAPE`.
- Added live Control Lane action `VALIDATE_SEMANTIC_OPERATION`.
- Added capabilities `ref.shape.compare` and `semantic.operation.validate`.
- Added structured audit fields for ref-shape comparison and semantic-operation validation.
- Added tests for deterministic comparison drift, invalid observed refs, semantic operation shape validation, forbidden authority claims, capability denial, no semantic commits, and forbidden imports.

## Ref Shape Shadow Comparison

`COMPARE_REF_SHAPE` accepts:

- workspace id,
- candidate refs,
- observed refs.

It returns:

- match/drift decision,
- added refs,
- removed refs,
- unchanged refs,
- no-mutation authority flags.

Drift is diagnostic, not a failure. Invalid ref shape fails closed.

## Semantic Operation Validation

`VALIDATE_SEMANTIC_OPERATION` accepts:

- workspace id,
- operation type,
- source refs,
- optional derived refs,
- optional provenance refs,
- optional authority claims.

It validates operation envelope shape only. It rejects forbidden claims such as memory writes, evidence admission, context compilation, model calls, tool execution, retrieval/search/embedding execution, and live authority migration.

## Authority Boundary

Phase 14C does not make FORGE-K live authority.

The live owner remains Control Lane. The shared packages validate shapes only. They do not look up object truth, admit evidence, create ContextBlocks, compile prompts, write memory, execute retrieval, call models, or execute tools.

## Validation

Passed commands:

- `cd services/core && go test ./internal/semanticvalidation ./internal/refvalidation ./internal/aios/controllane -count=1`
- `cd services/core && go test ./internal/forgek/... -count=1`
- `cd services/core && go test ./internal/api -run 'TestServerRouteInventory|TestServerRouteInventoryHealthAndMiddlewareSmoke' -count=1`
- `npm run build:core`
- `npm run lint`
- `npm test`
- `npm run test:forgek:parity`
- `npm run test:integration:env`
- `git diff --check`
