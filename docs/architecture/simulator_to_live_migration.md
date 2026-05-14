# Simulator To Live Migration

Status: Current migration-pattern document. FORGE-K simulator services remain `SIMULATOR_ONLY`; Phase 12 is read-only shadow diagnostics; Phase 14 is validation-only Control Lane integration through live-owned seams. This document is intentionally concise and supersedes older phase-ledger wording in this file.

## Purpose

FORGE-K simulator packages define target authority behavior. Live daemon integration must migrate one narrow, tested contract at a time through the existing live authority owner. Importing simulator services wholesale is not integration and does not create live authority.

## Current Boundary Map

| Area | Current status | Authority posture |
| --- | --- | --- |
| Simulator services | `services/core/internal/forgek` | Target architecture only; no live daemon authority |
| Phase 12 shadow diagnostics | `services/core/internal/forgekshadow` | Disabled-by-default read-only metadata/advisory reports |
| Phase 14 Control Lane seams | `services/core/internal/aios/controllane` plus shared pure validators | Validation/enforcement only; no simulator service authority |
| Live daemon authority | API, AI-OS Control Lane, gateway, permissions, lanes, audit, modelruntime, retrieval/search/embeddings, memory | Existing live owners remain authoritative |

## Migration Pattern

1. Identify the smallest deterministic contract that can be proven without side effects.
2. Extract the contract into a shared pure package with no simulator service imports and no live side effects.
3. Call that package only from the existing live authority owner for that decision class.
4. Keep simulator packages as simulator-owned services and test fixtures.
5. Fail closed on invalid payloads, ambiguous authority claims, scope conflicts, and unsafe refs.
6. Emit audit/state metadata that records no-effect posture and the exact validation decision.
7. Add acceptance tests proving no unintended mutation, no route/API behavior change, no retrieval/search/embedding execution, no modelruntime call, no gateway/tool execution, and no user-visible output change.
8. Update status docs with `[SIMULATOR-ONLY]`, `[LIVE / READ_ONLY]`, `[PARTIAL LIVE VALIDATION]`, `[BLOCKED]`, or `[FUTURE]` markers.

## Proven Pattern Examples

| Seam | Shared contract | Live owner | Live action | Posture |
| --- | --- | --- | --- | --- |
| KV identity | `services/core/internal/kvidentity` | Control Lane | `VALIDATE_KV_IDENTITY` | Validation/enforcement only; no live KV reuse |
| Ref shape | `services/core/internal/refvalidation` | Control Lane | `VALIDATE_REF_SHAPE` | Validation only; no object truth lookup |
| Ref comparison | `services/core/internal/refvalidation` | Control Lane | `COMPARE_REF_SHAPE` | Diagnostic comparison only; drift is not mutation |
| Source-object authority | live read-store lookup plus ref validation | Control Lane | `VALIDATE_SOURCE_OBJECT_AUTHORITY` | Read-only validation; no evidence admission |
| Semantic operation shape | `services/core/internal/semanticvalidation` | Control Lane | `VALIDATE_SEMANTIC_OPERATION` | Envelope validation only; no operation execution |
| Validation shadow summary | `services/core/internal/forgekshadow` | Control Lane observer hook | post-result best-effort emission | Disabled-by-default diagnostics only |

## Phase 12 Shadow Diagnostics

Phase 12 diagnostics may copy bounded metadata from already-executing live paths when the relevant disabled-by-default flags are enabled. They may build diagnostic and advisory reports from safe refs, counts, status classes, route classes, and redacted metadata.

Phase 12 diagnostics must not:

- alter live requests, responses, route inventory, or public API shapes
- call modelruntime
- execute tools
- run retrieval/search/embeddings
- compile live prompt context
- admit evidence
- write memory or canonical state
- create user-visible output authority

## Phase 14 Validation Seams

Phase 14 seams are live-owned Control Lane validation/enforcement actions. They may validate shape, identity, scope, source-object authority, and forbidden authority claims. They may report readiness and gate status.

Phase 14 seams must not:

- import `forgek.Kernel` or simulator syscalls as live authority
- route semantic writes through simulator services
- use simulator Courthouse admission as live evidence authority
- use simulator Context Compiler output as live prompt authority
- execute semantic operations
- call modelruntime
- execute retrieval/search/embeddings or gateway tools
- write semantic memory outside existing governed paths

## Blocked Authority Gates

The read-only activation readiness surface may mark these gates as blocked, but it cannot open them:

- live Courthouse admission integration
- live Context Compiler prompt authority
- governed semantic mutation routing
- runtime driver authority boundary
- broader FORGE-K Kernel live authority

Opening any blocked gate requires a separate live authority migration design, explicit live owner, tests, rollback plan, observability, documentation updates, and operator approval where applicable.

## Forbidden Shortcut

Do not import FORGE-K Kernel, Courthouse, Context Compiler, KVService, Runtime Boundary, Lymphatic Lane, Consensus Mesh, or simulator syscalls into live daemon paths and call that live integration. Simulator packages remain target architecture until a specific live authority gate is migrated through the existing live owner and proven by tests.

## Readiness Checklist

- Pure contract extracted or adapter boundary designed
- Simulator behavior still passes
- Existing live authority owner identified
- Capability and audit/state metadata are explicit
- No-effect guarantees are tested
- Route/API behavior is unchanged unless explicitly approved
- Gateway, modelruntime, retrieval/search/embedding, and memory behavior are unchanged unless explicitly approved
- Rollback or disable path is documented
- Status docs and ADR amendments are updated
- Validation commands are recorded
