# PhaseI2 Live KV Enforcement Review

Status: `[PARTIAL LIVE ENFORCEMENT]`.

Date: 2026-05-08.

## Executive Summary

PhaseI2 converts PhaseI1's live KV identity validation seam into an explicit live enforcement and observation boundary.

The live path is still validation-only. A valid identity claim can be accepted as an acceleration-eligibility validation result, but no live KV reuse, backend cache lookup, modelruntime mutation, memory mutation, route change, public API, or FORGE-K live authority is enabled.

## Live-Facing Claim Surface Inventory

| Surface | Status | Current role | Enforces `kvidentity`? | Safe PhaseI2 action |
| --- | --- | --- | --- | --- |
| `services/core/internal/aios/controllane` `VALIDATE_KV_IDENTITY` | `[LIVE] / [PARTIAL]` | Live semantic syscall for KV identity validation | Yes, through `EnforceKVIdentity` and `services/core/internal/kvidentity` | Enforce accepted/rejected/malformed/unsupported decisions; audit and count decisions. |
| `services/core/internal/kvidentity` | `[PARTIAL]` | Pure deterministic identity gate validator | It is the shared validator | Keep pure; add required-field fail-closed behavior and deterministic tests. |
| `services/core/internal/forgek/kv` | `[SIMULATOR-ONLY]` | Simulator KV manifests, lookup, tiers, invalidation, eviction, and syscalls | Yes, by calling shared `kvidentity` | No live wiring. |
| `services/core/internal/forgek/runtime` | `[SIMULATOR-ONLY]` | Simulator runtime driver boundary references KV metadata refs | No live path | Keep future; do not wire to modelruntime. |
| `services/core/internal/forgek/contextcompiler` | `[SIMULATOR-ONLY]` | Produces tokenizer-neutral `token_input_hash` for simulator context shape | No live path | Keep future; do not replace live `COMPILE_CONTEXT`. |
| `services/core/internal/forgek/shadowharness` and `services/core/internal/forgekshadow` | `[LIVE DIAGNOSTIC] / [READ_ONLY] / [DISABLED_BY_DEFAULT]` | Metadata-only shadow reports may include KV diagnostic flags | No live reuse claim | Keep diagnostic-only. |
| `services/core/internal/modelruntime` | `[LIVE]` | Live model runtime substrate | No KV identity enforcement surface found | Future runtime identity capture must call the enforcement boundary before any reuse claim. |
| `services/core/internal/gateway` | `[LIVE]` | Live tool execution authority | No KV identity enforcement surface found | No PhaseI2 change; gateway remains tool authority. |
| `services/core/internal/storagebackend`, `vectorstore`, `ephemeral` | `[LIVE-INFRA] / [DISABLED_BY_DEFAULT]` where applicable | Postgres/Qdrant/Redis scaffolds and adapters | No KV identity enforcement surface found | Keep non-authoritative; no cache reuse switch. |
| Docs and phase briefs | `[DOCS] / [FUTURE]` | Describe possible KV/runtime/cache evolution | Not executable | Mark status clearly and archive phase briefs. |

## Enforcement Added

`services/core/internal/aios/controllane/kv_enforcement.go` defines the live-side policy wrapper around the pure validator.

The enforcement decision classifies claims as:

- `accepted`
- `rejected`
- `malformed`
- `unsupported_live_reuse`
- `internal_error`

Every decision carries:

- source
- candidate cache ID when safe
- token identity hash when safe
- failed gates
- warnings
- `accelerationOnly=true`
- `memoryMutation=false`
- `runtimeMutation=false`
- `liveKVReuse=false`
- policy and validator versions

## Audit Model

The existing Control Lane audit path is reused. `SyscallAuditRecord` now includes `KVIdentityEnforcement` fields when the action is `VALIDATE_KV_IDENTITY`.

This follows the existing one-record-per-syscall style instead of creating a second audit system. The recorded decision includes the accepted/rejected/malformed/unsupported state and no-effect authority flags.

## Metrics Model

`KVIdentityEnforcementCounters` provides a lightweight internal counter abstraction for:

- accepted
- rejected
- malformed
- unsupported live reuse
- internal error

The counters are live-process observability only. They are not canonical truth, not memory, and not exported through a public route in PhaseI2.

## Fail-Closed Rules

- malformed payloads reject before commit
- identity gate mismatches reject before commit
- explicit or ambiguous live reuse requests reject before commit
- unavailable manifest status rejects before commit
- unsupported callers remain capability-denied
- successful validation records no memory mutation, no runtime mutation, and no live KV reuse

## Future Work

- `[FUTURE]` tokenizer-specific final token ID identity capture
- `[FUTURE]` modelruntime trace-only identity capture
- `[FUTURE]` explicit runtime cache-reuse design
- `[FUTURE]` public diagnostics only if a separate API design authorizes it
- `[FUTURE]` persisted metrics export if needed

## Not Implemented

- live KV reuse
- backend cache lookup
- real KV tensor storage
- modelruntime mutation
- memory mutation
- route or public API changes
- FORGE-K `KVService` live authority
