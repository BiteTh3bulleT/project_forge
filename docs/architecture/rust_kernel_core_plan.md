# Rust Kernel Core Plan

Status: Phase 11A research and planning complete. Phase 11B is implemented as `RESEARCH_ONLY / SIMULATOR_ONLY`.

Phase 11A does not add Rust code, a Rust crate, live daemon wiring, public APIs, routes, gateway behavior, model runtime behavior, or Go runtime behavior changes. FORGE-K Phase 1-10 remains implemented and tested as a Go simulator under `services/core/internal/forgek`.

Phase 11B adds a standalone Rust validation crate at `crates/forgek-validate` and shared fixtures under `fixtures/forgek`. It does not add Go imports, cgo, live daemon wiring, public APIs, routes, gateway behavior, model runtime behavior, or Go runtime behavior changes.

## Executive Summary

Rust is worth considering for a narrow future FORGE-K kernel-core boundary because several simulator contracts have become deterministic, validation-heavy, and testable across language boundaries. Rust should not replace FORGE, the Go simulator, the live daemon, the gateway, model runtimes, or live AI-OS controllane behavior.

Phase 11B implements the recommended standalone Rust validation crate with a CLI test harness, but it is not live integration and does not make Rust authoritative. The crate owns deterministic validation primitives only: canonical JSON normalization, stable SHA-256 hashing, and manifest validation for snapshots, context compiler outputs, KV manifests, runtime driver manifests, and capability-like fixtures. Go remains the simulator orchestrator and live daemon language until an explicit later `LIVE_INTEGRATION` phase.

## Current FORGE-K Simulator Status

The Go simulator covers Phase 1-10:

- Phase 1 Kernel Simulator: syscall registry, object registry, capabilities, journal, and basic CasePacket lifecycle.
- Phase 2 Neuron Fabric: typed neuron envelopes, proposal-only neural output, rule validation, and scheduler tests.
- Phase 3 Courthouse Minimal: exhibits, rulings, contradictions, supersession, and admission syscalls.
- Phase 4 Memory Palace Minimal: rooms, anchors, routes, candidate refs, deterministic route scoring, and route syscalls.
- Phase 5 Semantic Algebra: typed semantic objects, operation records, deterministic operators, and semantic syscalls.
- Phase 6 Snapshots: shape-not-truth snapshots, shape hashes, diffs, restore seeds, lifecycle syscalls.
- Phase 7 Context Compiler: ContextBlocks, ContextBundles, PromptLayout, deterministic serialization, hashes, and compile syscalls.
- Phase 8 Deterministic KV System: KVCacheManifest, lookup request/result, nine-gate validation, tiers, invalidation, and KV syscalls.
- Phase 9 Runtime Driver Boundary: driver manifests, mock driver, generate request/result, registry/service, and proposal-only runtime syscalls.
- Phase 10 Lymphatic Lane: deterministic maintenance reports and cleanup proposals only.

All of this is simulator authority only. ADR 0005 states that the live daemon still uses existing AI-OS, gateway, permissions, lane, audit, model runtime, and API authority paths.

## Why Rust Is Being Considered

Rust is relevant where FORGE-K needs stronger enforcement around deterministic data contracts:

- stable canonical serialization without map-order or timestamp drift
- hash-chain and manifest identity validation
- enum and field validation with explicit error codes
- capability and scope predicate evaluation
- memory-safe parsing of externalized fixtures
- language-neutral golden test corpus support
- future hardware/software co-design paths for deterministic kernel primitives

Rust is not needed to make model calls, route APIs, run a desktop shell, or replace live daemon state management.

## What Rust Should Own

Rust should own only deterministic, side-effect-free primitives until a later phase proves parity:

- canonical serialization for stable FORGE-K fixture shapes
- SHA-256 hashing over canonical forms
- object ID, workspace ID, source ref, and typed ref validation
- capability predicate evaluation from immutable inputs
- journal hash-chain verification
- manifest validation for snapshots, context blocks, KV manifests, and runtime manifests
- KV nine-gate validation primitives
- stable enum/type definitions that are already documented and tested
- deterministic ordering and deduplication helpers
- policy-neutral validation functions that return explicit results, not mutations

These functions should accept input, return validation or hash results, and never mutate live state.

## What Rust Should Not Own

Rust should not own:

- live daemon integration
- live AI-OS controllane replacement
- gateway, tools, approvals, or permissions routing
- API route handling
- desktop/Tauri integration
- model runtime drivers or network/model calls
- live KV reuse
- live snapshot/restore behavior
- Memory Palace retrieval scoring while policy is still evolving
- semantic algebra planning while operators are still broadening
- Lymphatic sweep policy while finding/proposal coverage is still broadening
- runtime scheduling or streaming
- any canonical state mutation path

## Go/Rust Boundary Options

| Option | Shape | Advantages | Risks | Assessment |
| --- | --- | --- | --- | --- |
| A | Rust crate called by Go through cgo | Direct in-process calls; lower call overhead | Adds cgo complexity, build friction, platform variation, and accidental live dependency risk | Too early |
| B | Rust CLI helper invoked by Go or tests | Simple isolation; easy golden corpus comparison; no live linking | Process overhead; not suitable for hot path | Recommended for Phase 11B |
| C | Rust WASM module | Portable and sandboxed; possible browser/test reuse | Toolchain complexity; less ergonomic for Go server tests | Consider later |
| D | Rust service sidecar | Strong isolation; language boundary is explicit | Creates daemon lifecycle and authority confusion too early | Not now |
| E | No Rust yet | Keeps current velocity and one language | Delays validation hardening and cross-language corpus work | Acceptable fallback, but less useful than a CLI harness |

## Recommended Boundary

Phase 11B should start with a standalone Rust crate and CLI test harness for deterministic validation primitives. It should not be imported by live daemon code, should not be linked through cgo, and should not expose a daemon.

The initial boundary should be:

1. Go simulator emits or shares JSON fixtures and expected hashes.
2. Rust CLI validates fixtures and recomputes canonical hashes.
3. CI may run Rust fixture checks only after Phase 11B explicitly approves Rust tooling.
4. Go remains the simulator owner and source of behavioral tests.
5. Rust outputs validation reports, not state mutations.

Phase 11B implements this boundary as a standalone CLI crate. No live daemon import, cgo bridge, sidecar daemon, public route, gateway hook, model runtime hook, Go production call, or CI entry is created by this pass.

## Stable Candidate Primitives

Strong candidates for Rust after fixture schemas are frozen:

- `KernelObject` identity and authority field validation
- `Capability` predicate evaluation and workspace scope checks
- `JournalEvent` canonical hashing and hash-chain verification
- Snapshot type/status validation and `shape_hash`
- ContextBlock canonical serialization, `content_hash`, and `token_input_hash`
- ContextBundle canonical serialization, `bundle_hash`, stable prefix hash, and volatile suffix hash
- KVCacheManifest validation and identity hash
- KV nine-gate validation and failed-gate reporting
- RuntimeDriverManifest and RuntimeCapabilityManifest validation
- deterministic ref normalization and stable sorting helpers

## Unstable Or Deferred Primitives

These should remain Go for now:

- Kernel orchestration and syscall dispatch
- object registry mutation and service ownership
- Courthouse policy expansion and richer rulings
- Memory Palace route scoring and retrieval strategy
- Semantic Algebra operator planning and future operators
- Context Compiler block-building policy beyond serialization/hash validation
- Runtime driver invocation, streaming, and backend selection
- Lymphatic sweep policy and proposal generation
- live daemon integration
- live modelruntime and gateway behavior

## Data Serialization Requirements

Rust cannot start safely until these contracts are stable:

- object ID and workspace ID format
- source ref format
- typed object ref format
- manifest JSON schema for snapshots, ContextBlocks, ContextBundles, KV manifests, runtime manifests, and maintenance reports
- canonical serialization field order
- whitespace normalization rules
- hash algorithm and hex encoding
- enum string values
- error code vocabulary
- syscall names
- capability names and wildcard semantics
- journal event names and hash-chain input shape
- KV gate names
- context block type names and layout version semantics
- snapshot type and status names
- runtime manifest fields
- lymphatic finding and proposal type names

Timestamps, generated IDs, and non-deterministic metadata must be excluded from content identity hashes unless explicitly part of a separately versioned identity.

## FFI/API Options

Phase 11B should avoid FFI and use a CLI interface:

- input: fixture JSON or newline-delimited validation request JSON
- output: structured JSON validation result
- exit code: non-zero for malformed input or unexpected harness errors, not for ordinary validation failures
- no network, model, filesystem mutation, daemon state, or live API dependency

cgo, WASM, and sidecar service forms should wait until the fixture corpus proves parity and the boundary has explicit integration tests.

## Testing Strategy

Phase 11B starts the deterministic shared corpus with:

- valid and invalid `Snapshot` fixtures
- valid and invalid `ContextBlock` fixtures
- valid `ContextBundle` fixtures
- valid and invalid `KVCacheManifest` fixtures
- valid and invalid `RuntimeDriverManifest` fixtures
- canonical serialization golden files
- hash golden files

Future fixture expansion should add KernelObject, Capability, JournalEvent, LymphaticPolicy, MaintenanceReport, CleanupProposal, failed-gate fixtures, and cross-language Go/Rust parity checks.

Go tests should remain authoritative. Rust tests should prove parity against the shared corpus.

## Migration Strategy

1. Freeze string constants and minimal schema versions for the first validation targets.
2. Add golden fixtures from the Go simulator.
3. Build a standalone Rust crate and CLI harness in Phase 11B. Completed in `crates/forgek-validate`.
4. Run Go and Rust against the same fixtures.
5. Keep Rust out of live daemon and Go simulator hot paths until parity is stable.
6. Consider cgo, WASM, or service integration only in a later explicit phase.

## Risk Register

| Risk | Impact | Mitigation |
| --- | --- | --- |
| Rust starts before contracts are stable | Rework and mismatched behavior | Start with fixtures and validation primitives only |
| Duplicate authority path | Unsafe live mutation | ADR 0005 remains binding; no live wiring |
| FFI complexity | Fragile builds and platform drift | Phase 11B CLI harness, not cgo |
| CI/toolchain complexity | Slower development | Do not add Rust CI until Phase 11B approval |
| Serialization mismatch | Hash and validation drift | Golden corpus with Go/Rust parity tests |
| Hash mismatch | False cache hits/misses or replay failure | Versioned canonical serialization and hash tests |
| Go/Rust drift | Confusing authority | Go remains behavioral source until integration phase |
| Accidental live daemon dependency | Unauthorized authority path | Keep Rust crate standalone and simulator-only |
| Tests are not shared across languages | False confidence | Shared fixture corpus before Rust code |
| Over-porting fluid subsystems | Slower feature work | Only port stable deterministic primitives |

## Recommended Phase 11B Scope

Phase 11B is implemented only as `SIMULATOR_ONLY / RESEARCH_ONLY`.

Implemented deliverables:

- standalone Rust crate at `crates/forgek-validate`
- CLI commands: `validate`, `canonicalize`, `hash`, and `validate-fixtures`
- canonical JSON normalization with stable object ordering, whitespace normalization, and deterministic ref ordering
- SHA-256 hashes over stable projections that exclude generated IDs, timestamps, and existing hash fields
- validators for Snapshot, ContextBlock, ContextBundle, KVCacheManifest, RuntimeDriverManifest, and capability-like fixtures
- conservative runtime secret-looking field rejection
- shared fixtures and golden hashes under `fixtures/forgek`
- root scripts `test:rust:forgek` and `validate:forgek-fixtures`

Still deferred:

- no cgo
- no live daemon import
- no Go runtime behavior change
- no gateway/modelruntime/API route changes
- no CI dependency
- no journal hash-chain verifier or full KV nine-gate CLI command yet
- no Go production calls into Rust

Phase 11B validation passed:

- `cd services/core && go test ./internal/forgek/...`
- `cd crates/forgek-validate && cargo test`
- `npm run test:rust:forgek`
- `npm run validate:forgek-fixtures`
- `npm run build:core`
- `npm run lint`
- `npm test`

## What Not To Do

- Do not rewrite FORGE-K in Rust.
- Do not replace the Go simulator.
- Do not integrate Rust into the live daemon yet.
- Do not create live state mutation paths.
- Do not call model runtimes from Rust.
- Do not route gateway/tool execution through Rust.
- Do not add routes.
- Do not change public APIs.
- Do not bypass ADR 0005.
- Do not create a second kernel authority path.
- Do not port unstable subsystems.
- Do not add Rust dependencies to CI until Phase 11B is approved.
