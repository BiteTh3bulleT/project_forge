# FORGE-K Control Lane Contract Matrix Design

Date: 2026-05-11

## Status

Approved for Phase 14G implementation.

## Goal

Phase 14G hardens the Phase 14F partial live enforcement seam by proving every live Control Lane validation action exposes the same FORGE-K activation and no-effect contract.

## Scope

This phase is test and documentation hardening only. It covers the existing validation actions:

- `VALIDATE_KV_IDENTITY`
- `VALIDATE_REF_SHAPE`
- `COMPARE_REF_SHAPE`
- `VALIDATE_SEMANTIC_OPERATION`

It does not add routes, public APIs, shell UI, host mutation, service control, modelruntime mutation, live KV reuse, retrieval/search/embedding execution, evidence admission, context compilation, semantic memory writes, or FORGE-K simulator authority.

## Design

Add table-driven Control Lane tests that execute each validation action through the live `Processor` and assert the shared contract in both top-level `StateSummary` and nested audit summary fields.

Each action must report:

- `forgeKActivation.mode == partial-live-enforcement`
- `forgeKActivation.liveOwner == aios.controllane`
- `forgeKActivation.simulatorAuthority == false`
- `forgeKActivation.liveKernelAuthority == false`
- `forgeKNoEffect.memoryMutation == false`
- `forgeKNoEffect.runtimeMutation == false`
- `forgeKNoEffect.modelRuntimeCall == false`
- `forgeKNoEffect.evidenceAdmission == false`
- `forgeKNoEffect.contextCompilation == false`
- `forgeKNoEffect.gatewayExecution == false`
- `forgeKNoEffect.retrievalExecution == false`
- `forgeKNoEffect.liveAuthorityMigration == false`

The contract test should use existing valid request builders where possible. KV identity currently has a direct enforcement unit test; Phase 14G adds processor-level KV coverage so audit and state serialization are checked through the live syscall path.

## Failure Behavior

Missing or changed activation/no-effect fields should fail focused tests before broader core tests. The tests intentionally do not inspect or depend on raw full logs, host state, model state, retrieval contents, or semantic memory contents.

## Documentation

Update FORGE-K status and Control Lane architecture docs to record Phase 14G as contract hardening. The docs must state that the phase does not broaden live authority and does not make FORGE-K simulator services live.

## Verification

Run:

```bash
cd services/core && go test ./internal/aios/controllane -count=1
cd services/core && go test ./internal/aios/controllane ./internal/forgekshadow ./internal/refvalidation ./internal/semanticvalidation -run 'Forbidden|Import|Authority|Contract' -count=1
cd services/core && go test ./...
```

## Non-Goals

- No new validation action.
- No live semantic operation execution.
- No approval queue, shell, or operator UI work.
- No simulator service import into live Control Lane.
- No change to gateway, permissions, lanes, audit ownership, modelruntime, retrieval, memory, API routing, or FORGE-H/FORGE-K authority boundaries.
