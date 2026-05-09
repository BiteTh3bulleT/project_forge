# Phase 14B Ref Shape Validation Review

Status: implemented and tested.

Scope: `PARTIAL LIVE VALIDATION / CONTROL_LANE / NO_AUTHORITY_REPLACEMENT`.

Date: 2026-05-09.

## Summary

Phase 14B adds the first operational cutover seam after the Phase 14A design: shared deterministic ref-shape validation through the live AI-OS Control Lane.

This is validation-only. It does not import FORGE-K simulator services into live authority, replace the Kernel, write semantic memory, admit evidence, compile context, execute retrieval, call modelruntime, execute tools, change routes, or change public APIs.

## What Changed

- Added shared pure package `services/core/internal/refvalidation`.
- Added live Control Lane action `VALIDATE_REF_SHAPE`.
- Added capability `ref.shape.validate`.
- Added Control Lane enforcement for ref-shape validation.
- Added structured audit fields for ref-shape validation decisions.
- Added tests proving normalization, duplicate removal, fail-closed invalid refs, capability denial, dry-run summary preservation, no semantic commits, and no forbidden imports.

## Shared Contract

`refvalidation.ValidateRefs` accepts:

- workspace id,
- typed object refs,
- optional per-ref workspace id,
- optional source ref.

It returns:

- deterministic normalized refs,
- validation failures,
- warnings,
- pass/fail state.

The package is pure. It does not import FORGE-K simulator services, live gateway, modelruntime, retrieval, search, embeddings, memory, or API packages.

## Live Control Lane Behavior

`VALIDATE_REF_SHAPE` is non-mutating and capability-gated.

Successful validation returns:

- `passed=true`,
- normalized refs,
- `memoryMutation=false`,
- `runtimeMutation=false`,
- `liveAuthorityMigration=false`.

Rejected validation fails closed with `INVALID_PAYLOAD` and does not persist idempotency success. Propose-only sources without the capability are denied.

## Authority Boundary

Phase 14B does not make FORGE-K live authority.

The live owner remains Control Lane. The shared package validates ref shape only. It does not look up object truth, admit evidence, create ContextBlocks, compile prompts, write memory, execute retrieval, or call model runtimes.

## Not Implemented

- FORGE-K Kernel live authority
- live Context Compiler authority
- live Courthouse evidence admission
- live Memory Palace retrieval ranking
- live Consensus Mesh response authority
- live KV reuse
- modelruntime mutation
- gateway/tool behavior changes
- public route or API changes
- memory writes

## Validation

Passed commands:

- `cd services/core && go test ./internal/refvalidation ./internal/aios/controllane -count=1`
- `cd services/core && go test ./internal/forgek/... -count=1`
- `cd services/core && go test ./internal/api -run 'TestServerRouteInventory|TestServerRouteInventoryHealthAndMiddlewareSmoke' -count=1`
- `npm run build:core`
- `npm run lint`
- `npm test`
- `npm run test:forgek:parity`
- `npm run test:integration:env`
- `git diff --check`
