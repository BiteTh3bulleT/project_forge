# Phase 11 Readiness Review

Status: Phase 11A research and planning complete. Phase 11B is implemented as `RESEARCH_ONLY / SIMULATOR_ONLY`.

Scope: Phase 11A was `RESEARCH_ONLY / DOCS_ONLY`; Phase 11B is `RESEARCH_ONLY / SIMULATOR_ONLY`.

## Readiness Summary

FORGE-K Phase 1-10 is implemented and tested in the Go simulator under `services/core/internal/forgek`. The repository now has a standalone Rust validation boundary for deterministic fixtures, but it is not ready for Rust to own runtime orchestration, live daemon state, gateway behavior, model runtime behavior, route behavior, or canonical mutation.

Phase 11B implements the recommended standalone Rust deterministic validation crate and CLI harness at `crates/forgek-validate`. It remains research-only/simulator-only: it defines first-pass fixture scope, validation result vocabulary, canonicalization, hashing, and CLI behavior, but it must not add live integration, cgo, routes, gateway behavior, model runtime behavior, or canonical mutation.

## Current Phase 1-10 Status

| Phase | Status |
| --- | --- |
| Phase 1 Kernel Simulator | Implemented and tested in Go simulator |
| Phase 2 Neuron Fabric | Implemented and tested in Go simulator |
| Phase 3 Courthouse Minimal | Implemented and tested in Go simulator |
| Phase 4 Memory Palace Minimal | Implemented and tested in Go simulator |
| Phase 5 Semantic Algebra | Implemented and tested in Go simulator |
| Phase 6 Snapshots | Implemented and tested in Go simulator |
| Phase 7 Context Compiler | Implemented and tested in Go simulator |
| Phase 8 Deterministic KV System | Implemented and tested in Go simulator |
| Phase 9 Runtime Driver Boundary | Implemented and tested in Go simulator, mock-only |
| Phase 10 Lymphatic Lane | Implemented and tested in Go simulator, reports/proposals only |
| Phase 11A Rust Kernel Core Research / Planning | Planning complete |
| Phase 11B Rust Deterministic Validation Boundary | Implemented and tested as research-only/simulator-only |

## Recorded Phase 11A Validation

Historical preflight recorded for the Phase 11A planning pass:

```bash
cd services/core && go test ./internal/forgek/...
```

Result recorded in the Phase 11A pass: passed for root FORGE-K and all simulator packages.

Historical post-doc validation recorded for the Phase 11A planning pass:

| Command | Result |
| --- | --- |
| `cd services/core && go test ./internal/forgek/...` | Passed |
| `npm run build:core` | Passed |
| `npm run lint` | Passed |
| `npm test` | Passed |
| `git diff --check` | Passed |

No Rust build was run in Phase 11A because that pass added no Rust code or Rust crate.

Phase 11B validation for this implementation checkpoint:

| Command | Result |
| --- | --- |
| `cd services/core && go test ./internal/forgek/...` | Passed. |
| `cd crates/forgek-validate && cargo test` | Passed during crate implementation. |
| `npm run test:rust:forgek` | Passed. |
| `npm run validate:forgek-fixtures` | Passed. |
| `npm run build:core` | Passed. |
| `npm run lint` | Passed. |
| `npm test` | Passed. |

## Remaining Blockers Before Any Live Rust Integration

- The first stable fixture corpus exists, but it is intentionally limited to Snapshot, ContextBlock, ContextBundle, KVCacheManifest, RuntimeDriverManifest, and capability-like fixture validation.
- Canonical serialization exists in Rust for the first fixture set, but Go/Rust parity is not yet enforced by CI.
- Hash inputs need versioned schema declarations before live or cross-process authority can depend on them.
- Error codes are still lightweight validation strings rather than a shared machine-enforced error code vocabulary.
- Some package policies are intentionally broadening, especially Memory Palace scoring, semantic planning, runtime integration, and Lymphatic sweep coverage.
- Rust toolchain and CI integration should not be added until an explicit CI phase.

These blockers remain intentionally open. Resolving them requires a later scoped phase and must not create a live authority path.

## Stable Interfaces

The most stable interface families are:

- syscall names and journal event names in `types.go`
- capability workspace/scope predicates in `capabilities.go`
- journal hash-chain shape in `journal.go`
- snapshot type/status strings, ref normalization, shape hashes, diffs, and restore seed refs
- context block type strings, default layout order, canonical serialization, and hashes
- KV cache modes, tiers, statuses, gate names, manifest validation, and nine-gate results
- runtime manifest validation and mock-driver proposal-only result envelopes

## Unstable Interfaces

These should stay Go-owned:

- Kernel service orchestration and syscall dispatch
- full Courthouse adjudication and future precedent reasoning
- Memory Palace retrieval scoring beyond current deterministic baseline
- Semantic Algebra operator expansion and planner behavior
- Context Compiler block selection policy beyond canonical serialization and hash validation
- Runtime driver invocation, streaming, tool use, and real backend selection
- Lymphatic sweep policy, finding types, and cleanup proposal expansion
- all live daemon integration

## Package-by-Package Assessment

| Package | Classification | Reasoning |
| --- | --- | --- |
| `services/core/internal/forgek` root | `NEEDS_MORE_GO_HARDENING` | Constants, capability checks, object types, journal hashing, and syscall names are strong Rust candidates, but Kernel orchestration and service ownership should remain Go. |
| `services/core/internal/forgek/neurons` | `KEEP_IN_GO` | Neuron scheduling and proposal/rule envelopes are simulator orchestration surfaces. Validation helpers may later inform fixtures, but the package is not a strong initial Rust target. |
| `services/core/internal/forgek/court` | `NEEDS_MORE_GO_HARDENING` | Exhibit/ruling/contradiction models are stable enough for fixture validation, but admission policy and future adjudication remain fluid. |
| `services/core/internal/forgek/palace` | `TOO_FLUID_FOR_RUST` | Model structs are straightforward, but retrieval scoring and route policy are likely to evolve. Keep in Go until retrieval contracts are stable. |
| `services/core/internal/forgek/semantic` | `NEEDS_MORE_GO_HARDENING` | Object and operation record validation could become Rust fixtures, but operator behavior and future planning are still broadening. |
| `services/core/internal/forgek/snapshots` | `STABLE_FOR_RUST_CANDIDATE` | Snapshot refs, status lifecycle, shape hash, source hash, diff, and restore seed identity are deterministic and well-tested. |
| `services/core/internal/forgek/contextcompiler` | `STABLE_FOR_RUST_CANDIDATE` | Canonical block/bundle serialization, prompt layout ordering, content hash, token input hash, stable prefix hash, and volatile suffix hash are strong Rust candidates. |
| `services/core/internal/forgek/kv` | `STABLE_FOR_RUST_CANDIDATE` | KVCacheManifest validation, identity hash, invalidation state, tier strings, and nine-gate validation are deterministic and validation-heavy. |
| `services/core/internal/forgek/runtime` | `NEEDS_MORE_GO_HARDENING` | Runtime manifests and capability manifests are candidates; driver invocation and backend integration must remain Go and mock-only for now. |
| `services/core/internal/forgek/lymphatic` | `TOO_FLUID_FOR_RUST` | Phase 10 intentionally leaves several sweep kinds as future expansion. Reports/proposals can join fixtures later, but policy generation should stay Go. |

## Recommended Rust Candidates

- canonical serialization for Snapshot, ContextBlock, ContextBundle, KVCacheManifest, RuntimeDriverManifest, and JournalEvent fixtures
- deterministic SHA-256 hashing over canonical serialization
- ref normalization and duplicate removal
- object/workspace/ref validation
- capability predicate evaluation from immutable inputs
- journal hash-chain verification
- KV nine-gate validation
- manifest validation for snapshots, context compiler outputs, KV, and runtime driver declarations

## Deferred Rust Candidates

- syscall dispatch and Kernel service ownership
- object registry mutation
- Courthouse policy decisions
- Memory Palace retrieval scoring
- Semantic Algebra planning and operation execution
- Context Compiler block selection policy
- Lymphatic sweep policy and proposal generation
- runtime driver calls, streaming, and backend selection
- live daemon integration

## Implemented Phase 11B Fixture Set

Phase 11B creates or consumes shared fixtures for:

- valid/invalid Snapshot
- valid/invalid ContextBlock
- valid ContextBundle
- valid/invalid KVCacheManifest
- valid/invalid RuntimeDriverManifest
- canonical serialization golden files
- hash golden files
- failure-mode fixtures for missing workspace, missing refs, missing token hash, bad cache mode, and secret-looking runtime fields

Future fixture expansion should add KernelObject, richer Capability fixtures, JournalEvent, MaintenanceReport, CleanupProposal, KV gate-specific failures, and explicit Go/Rust parity checks.

## Stable Contract Requirements

The following must be versioned or explicitly frozen for the first Rust target:

- object IDs
- workspace IDs
- source refs
- manifest JSON schemas
- canonical serialization field ordering
- hash algorithm
- enum values
- error codes
- syscall names
- capability names
- journal event names
- KV gate names
- context block type names
- snapshot type names
- runtime manifest fields
- lymphatic finding and proposal types

## Recommended Next Step

Move to Phase 11C as `RESEARCH_ONLY / SIMULATOR_ONLY`: expand the shared fixture corpus and add explicit Go/Rust parity tests. Do not integrate Rust into the live daemon, cgo, routes, gateway, model runtime, controllane, or public APIs.
