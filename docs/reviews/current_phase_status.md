# FORGE-K Current Phase Status

Companion to `docs/reviews/full_project_review.md` (2026-05-03).

This is a concise status read of FORGE-K phases against the current repository. The key distinction is that Phase 1-11E are implemented in the simulator package `services/core/internal/forgek`, while the live daemon still uses the existing AI-OS/gateway/permissions/lane/audit authority paths. ADR 0005 records that FORGE-K is target architecture, not live daemon authority yet. Phase 11A is research/docs only. Phase 11B and Phase 11C are `RESEARCH_ONLY / SIMULATOR_ONLY` and add standalone Rust validation plus shared Go/Rust fixture parity. Phase 11D is `RESEARCH_ONLY / SIMULATOR_ONLY / TOOLING_ONLY` and adds CI/tooling checks for that Rust validation lane. Phase 11E is `SIMULATOR_ONLY / GOVERNANCE_LAYER_ONLY` and adds Consensus Mesh claim governance. They are not live authority.

| Phase | Title | Status | Where It Lives | Tests / Evidence | Open Work |
| --- | --- | --- | --- | --- | --- |
| 0 | Architecture Baseline | IMPLEMENTED | `docs/architecture/*`, ADRs 0001-0005, glossary, roadmap, DoD, diagrams. | Documentation is present and internally consistent. | Keep live-authority boundary visible in future phase reports. |
| 1 | Kernel Simulator | IMPLEMENTED + TESTED | `services/core/internal/forgek/{kernel,types,objects,syscalls,journal,capabilities,providers,case_syscalls}.go`. | `go test ./internal/forgek/...` passes. | Persistence and live daemon integration deferred. |
| 2 | Neuron Fabric | IMPLEMENTED + TESTED | `services/core/internal/forgek/neurons/*`. | Manifest, envelope, scheduler, neural/rule, syscall boundary tests pass. | Runtime/model neurons deferred. |
| 3 | Courthouse Minimal | IMPLEMENTED + TESTED | `services/core/internal/forgek/court/*`, `court_syscalls.go`. | Court model and syscall tests pass. | Full adjudication, claim extraction, precedent reasoning deferred. |
| 4 | Memory Palace Minimal | IMPLEMENTED + TESTED | `services/core/internal/forgek/palace/*`, `palace_syscalls.go`. | Palace model/scoring/syscall tests pass. | Embeddings/vector retrieval deferred. |
| 5 | Semantic Algebra | IMPLEMENTED + TESTED | `services/core/internal/forgek/semantic/*`, `semantic_syscalls.go`. | Semantic model/operator/syscall tests pass. | Advanced algebra policy/planning deferred. |
| 6 | Snapshots | IMPLEMENTED + TESTED | `services/core/internal/forgek/snapshots/*`, `snapshot_syscalls.go`, `docs/architecture/snapshots.md`, ADR 0003. Scope recorded as `SIMULATOR_ONLY` in the roadmap. | Snapshot model/service/diff/restore-seed/syscall tests pass under `go test ./internal/forgek/...`. | Persistence and live daemon integration remain deferred. |
| 7 | Context Compiler | IMPLEMENTED + TESTED | `services/core/internal/forgek/contextcompiler/*`, `context_syscalls.go`, `docs/architecture/context_compiler_and_kv_cache.md`. Scope recorded as `SIMULATOR_ONLY`; live `aios/controllane/compile_context_*` is a separate legacy path. | ContextBlock, ContextBundle, PromptLayout, deterministic serialization, hashing, compile service, snapshot/restore-seed integration, syscall, capability, journal, and shape-not-truth tests pass under `go test ./internal/forgek/...`. | Live daemon integration, live COMPILE_CONTEXT replacement, runtime drivers, and tokenizer-specific token IDs remain deferred. |
| 8 | Deterministic KV System | IMPLEMENTED + TESTED | `services/core/internal/forgek/kv/*`, `kv_syscalls.go`, `docs/architecture/context_compiler_and_kv_cache.md`. Scope recorded as `SIMULATOR_ONLY`; no real KV tensors or runtime backend cache reuse are implemented. | KVCacheManifest, lookup request/result, nine-gate validation, tiers, invalidation/eviction, service, context integration, syscall, capability, journal, and acceleration-not-memory tests pass under `go test ./internal/forgek/...`. | Live KV reuse, runtime drivers, tokenizer-specific final token IDs, and live daemon integration remain deferred. |
| 9 | Runtime Driver Integration | IMPLEMENTED + TESTED | `services/core/internal/forgek/runtime/*`, `runtime_syscalls.go`, `docs/architecture/runtime_driver_boundary.md`. Scope recorded as `SIMULATOR_ONLY / DRIVER_BOUNDARY_ONLY`; live `modelruntime`, gateway, routes, APIs, and live KV reuse are unchanged. | Runtime manifest, capability manifest, deterministic mock driver, registry/service, syscall, capability, journal, context-ref, KV-metadata, and model-as-driver tests pass under `go test ./internal/forgek/...`. | Real backend drivers, streaming, tool calling, live daemon integration, and live KV reuse remain deferred. |
| 10 | Lymphatic Lane | IMPLEMENTED + TESTED | `services/core/internal/forgek/lymphatic/*`, `lymphatic_syscalls.go`, `lymphatic_syscalls_test.go`, `docs/architecture/lymphatic_lane.md`. Scope recorded as `SIMULATOR_ONLY`; live dream/autonomy cleanup paths remain separate. | `go test ./internal/forgek/...` passes, including lymphatic package and syscall tests. | No live daemon wiring. Broader domain-specific hygiene expansion remains future simulator work. |
| 11A | Rust Kernel Core Research / Planning | PLANNING COMPLETE; IMPLEMENTATION NOT STARTED | `docs/architecture/rust_kernel_core_plan.md`, `docs/reviews/phase_11_readiness.md`, ADR 0006. | `cd services/core && go test ./internal/forgek/...` passed before the planning pass. | Rust implementation deferred until Phase 11B approval; no Rust code or live integration exists. |
| 11B | Rust Deterministic Validation Boundary | IMPLEMENTED + TESTED; RESEARCH/SIMULATOR ONLY | `crates/forgek-validate`, `fixtures/forgek`, `docs/architecture/rust_kernel_core_plan.md`, `docs/reviews/phase_11_readiness.md`, roadmap/README scope notes. | Rust crate, fixture CLI, Go simulator, core build, vet, and aggregate core tests pass; commands are recorded below. | No live daemon integration, cgo, Go production call, CI dependency, public API, route, gateway, modelruntime, or live controllane behavior change. |
| 11C | Go/Rust Test Corpus Alignment | IMPLEMENTED + TESTED; RESEARCH/SIMULATOR ONLY | `services/core/internal/forgek/fixture_parity_test.go`, `crates/forgek-validate`, `fixtures/forgek`, `scripts/forgek-parity.mjs`, roadmap/README scope notes. | Go fixture parity, Rust fixture validation, canonical golden JSON, golden hashes, and parity script validation pass; commands are recorded below. | No live daemon integration, cgo, Go production call, CI dependency, public API, route, gateway, modelruntime, or live controllane behavior change. |
| 11D | Rust Validation CI and Tooling Integration | IMPLEMENTED + TESTED; RESEARCH/SIMULATOR/TOOLING ONLY | `.github/workflows/ci.yml`, `package.json`, `docs/testing/rust_validation.md`, `crates/forgek-validate/README.md`, roadmap/README/status docs. | CI installs stable Rust and runs separate Rust validator, fixture validation, and Go/Rust parity steps; local validation commands are recorded below. | No live daemon integration, cgo, Go production Rust call, root `npm test` Rust dependency, public API, route, gateway, modelruntime, or live controllane behavior change. |
| 11E | Consensus Mesh | IMPLEMENTED + TESTED; SIMULATOR/GOVERNANCE ONLY | `services/core/internal/forgek/consensus/*`, `consensus_syscalls.go`, `consensus_syscalls_test.go`, `docs/architecture/consensus_mesh.md`. | Claim model, EvidenceRef, canonicalization, conflict detection, scoring, quorum, ledger, service, ComposerGuard, syscall, capability, journal, and no-second-authority tests pass under `go test ./internal/forgek/...`. | No live daemon integration, public API, route, gateway, modelruntime, controllane, tool execution, real model call, live memory write, or second Kernel authority path. |
| 12 | FORGE Daemon | PARTIAL OUTSIDE FORGE-K | Existing `services/core/main.go` daemon. | Live daemon tests exist indirectly. | Not FORGE-K-governed yet. |
| 13 | FORGE-1 Simulator | NOT STARTED | Concept doc only. | None. | Future research. |
| 14 | FORGE-1 Prototype Research | DOCUMENTED CONCEPT ONLY | `docs/architecture/forge_1_cpu_concept.md`. | None. | Future research. |

## Readiness Notes

- `go test ./internal/forgek/...` passes, including Phase 6 snapshot tests, Phase 7 Context Compiler tests, Phase 8 deterministic KV tests, Phase 9 runtime boundary tests, Phase 10 Lymphatic Lane tests, Phase 11C Go fixture parity tests, and Phase 11E Consensus Mesh tests.
- Representative API route inventory tests pass.
- `cd services/core && go test ./internal/forgek/...`, `npm run build:core`, `npm run lint`, `npm test`, and `npm run test:forgek:parity` pass in this Phase 11E pass.
- Desktop typecheck/build is blocked by local Node workspace package resolution.
- FORGE-K remains simulator authority only; the live daemon still uses AI-OS/gateway/permissions/lane/audit authority paths.
- Phase 11B adds a standalone Rust fixture validator under `RESEARCH_ONLY / SIMULATOR_ONLY`; Phase 11C adds shared Go/Rust fixture parity under the same boundary; Phase 11D adds CI/tooling integration only; Phase 11E adds simulator-only claim governance. Do not wire Phase 7, Phase 8, Phase 9, Phase 10, Phase 11B, Phase 11C, Phase 11D, Phase 11E, or future Rust code into the live daemon without a `LIVE_INTEGRATION` design and tests.

## Phase 6 Validation

- `cd services/core && go test ./internal/forgek/...` passes after Phase 6 implementation.
- `npm run build:core`, `npm run lint`, and `npm test` pass after Phase 6 implementation.
- Snapshot tests cover valid and invalid models, ref normalization, deterministic shape hashing, lifecycle service behavior, syscalls, capability gates, journal events, diff behavior, restore seeds, shape-not-truth invariants, and by-reference integration with Courthouse, Memory Palace, and Semantic Algebra.
- Phase 6 does not wire FORGE-K snapshots into the live daemon and does not change live AI-OS snapshot/restore behavior.

## Phase 7 Validation

- `cd services/core && go test ./internal/forgek/...` passes after Phase 7 implementation.
- `npm run build:core`, `npm run lint`, and `npm test` pass after Phase 7 implementation.
- Context Compiler tests cover ContextBlock, ContextBundle, PromptLayout, compile requests/results, deterministic serialization, content and token input hashing, stable prefix and volatile suffix hashing, token count estimates, cache eligibility metadata, compile service behavior, context syscalls, capability gates, journal events, snapshot and restore-seed integration, and shape-not-truth invariants.
- Phase 7 does not wire the FORGE-K Context Compiler into the live daemon and does not change live AI-OS `COMPILE_CONTEXT`, routes, public APIs, gateway behavior, or model runtime behavior.

## Phase 8 Validation

- `cd services/core && go test ./internal/forgek/...` passes after Phase 8 implementation.
- `npm run build:core`, `npm run lint`, and `npm test` pass after Phase 8 implementation.
- Deterministic KV tests cover KVCacheManifest validation and serialization, lookup request/result validation, nine-gate identity checks, cache salt and runtime assumption failures, tier metadata, invalidation/eviction, service hit/miss behavior, context compiler integration by refs, KV syscalls, capability gates, journal events, workspace scope, and acceleration-not-memory invariants.
- Phase 8 does not wire FORGE-K KV into the live daemon, does not store real KV tensors, does not call model runtimes, does not alter live AI-OS `COMPILE_CONTEXT`, and does not change routes, public APIs, gateway behavior, or model runtime behavior.

## Phase 9 Validation

- Phase 9 is recorded as `SIMULATOR_ONLY / DRIVER_BOUNDARY_ONLY`.
- `cd services/core && go test ./internal/forgek/...` passes after Phase 9 implementation.
- Phase 9 implements RuntimeDriver, RuntimeDriverManifest, RuntimeCapabilityManifest, RuntimeGenerateRequest, RuntimeGenerateResult, RuntimeDriverRegistry, RuntimeService, deterministic MockRuntimeDriver, runtime syscalls, capability checks, and journaled generation events.
- Tests cover model-as-driver doctrine, ContextBundle refs only, KV metadata only, no case/admission mutation, no ContextBundle mutation, no KV manifest mutation, and proposal-only runtime results.
- Phase 9 does not wire FORGE-K into the live daemon, replace live `modelruntime`, call real model backends, change routes/public APIs/gateway behavior, alter live AI-OS controllane behavior, or perform live KV reuse.

## Phase 10 Validation

- Phase 10 is recorded as `SIMULATOR_ONLY`.
- `cd services/core && go test ./internal/forgek/...` passes after Phase 10 simulator implementation.
- The implemented output surface is Maintenance Reports and Cleanup Proposals only.
- Phase 10 must not wire into the live daemon, live dream/autonomy behavior, live cleanup jobs, public APIs, routes, gateway, modelruntime, or AI-OS controllane behavior.
- Phase 10 must not silently mutate canonical truth or destroy provenance; any meaningful mutation remains a semantic syscall/Kernel responsibility.

## Phase 11A Validation

- Phase 11A is recorded as `RESEARCH_ONLY / DOCS_ONLY`.
- ADR 0006 is accepted and records a Rust boundary for deterministic validation primitives only.
- `docs/architecture/rust_kernel_core_plan.md` and `docs/reviews/phase_11_readiness.md` define the recommended boundary, package stability assessment, stable contracts, test corpus strategy, risk register, and Phase 11B recommendation.
- `cd services/core && go test ./internal/forgek/...` passed before the planning pass and after docs were updated.
- No Rust code, Rust crate, `Cargo.toml`, live daemon integration, public API, route, gateway, modelruntime, live controllane, or Go runtime behavior change exists in Phase 11A.

## Phase 11B Validation

- Phase 11B is recorded as `RESEARCH_ONLY / SIMULATOR_ONLY`.
- `crates/forgek-validate` implements canonical JSON normalization, stable SHA-256 hashing, manifest validators, conservative runtime secret-looking field rejection, and CLI commands for `validate`, `canonicalize`, `hash`, and `validate-fixtures`.
- `fixtures/forgek` contains valid, invalid, and golden fixtures for the first Snapshot, ContextBlock, ContextBundle, KVCacheManifest, and RuntimeDriverManifest contract set.
- Phase 11B has no cgo bridge, CI dependency, live daemon integration, public API, route, gateway, modelruntime, live controllane, Go production call, or Go runtime behavior change.
- Validation commands passed for this pass: `cd services/core && go test ./internal/forgek/...`, `cd crates/forgek-validate && cargo test`, `npm run test:rust:forgek`, `npm run validate:forgek-fixtures`, `npm run build:core`, `npm run lint`, and `npm test`.

## Phase 11C Validation

- Phase 11C is recorded as `RESEARCH_ONLY / SIMULATOR_ONLY`.
- `services/core/internal/forgek/fixture_parity_test.go` loads the shared fixtures, validates stable simulator fields, rejects the invalid fixture shapes, compares golden canonical JSON, and verifies Go-side stable hashes against `fixtures/forgek/golden/hashes.json`.
- `crates/forgek-validate` validates every valid fixture, rejects every invalid fixture, compares canonical golden files, verifies the expanded hash manifest, and includes drift tests for excluded timestamps, stable refs, secret-looking runtime fields, and KV runtime identity assumptions.
- `scripts/forgek-parity.mjs` runs the Go simulator test package, Rust validator tests, Rust fixture validation, and a Node golden hash manifest check. The script is available as `npm run test:forgek:parity`, but root `npm test` does not depend on Rust.
- Phase 11C has no cgo bridge, CI dependency, live daemon integration, public API, route, gateway, modelruntime, live controllane, Go production call, or Go runtime behavior change.
- Validation commands passed for this pass: `cd services/core && go test ./internal/forgek/...`, `cd crates/forgek-validate && cargo test`, `cd crates/forgek-validate && cargo run -- validate-fixtures ../../fixtures/forgek`, `npm run test:forgek:parity`, `npm run build:core`, `npm run lint`, `npm test`, and `git diff --check`.

## Phase 11D Validation

- Phase 11D is recorded as `RESEARCH_ONLY / SIMULATOR_ONLY / TOOLING_ONLY`.
- `.github/workflows/ci.yml` sets up stable Rust and runs `npm run test:rust:forgek`, `npm run validate:forgek-fixtures`, and `npm run test:forgek:parity` as separate CI steps after `npm ci`.
- Root `npm test` remains `npm run test:core` and does not depend on Rust.
- `docs/testing/rust_validation.md` records local commands, CI behavior, failure interpretation, fixture update workflow, and the no-live-authority boundary.
- Phase 11D has no cgo bridge, live daemon integration, public API, route, gateway, modelruntime, live controllane, Go production Rust call, or Go runtime behavior change.
- Validation commands passed for this pass: `npm run test:rust:forgek`, `npm run validate:forgek-fixtures`, `npm run test:forgek:parity`, `cd crates/forgek-validate && cargo test`, `cd services/core && go test ./internal/forgek/...`, `npm run build:core`, `npm run lint`, `npm test`, and `git diff --check`.

## Phase 11E Validation

- Phase 11E is recorded as `SIMULATOR_ONLY / GOVERNANCE_LAYER_ONLY`.
- `services/core/internal/forgek/consensus` implements Claim, EvidenceRef, AgentRun, ClaimLedger, canonicalization, conflict detection, ConsensusPolicy, weighted scoring, quorum, ConsensusDecision, ConsensusReport, ComposerGuard, and ConsensusService.
- `services/core/internal/forgek/consensus_syscalls.go` registers `consensus.open`, `consensus.submit_claim`, `consensus.submit_evidence`, `consensus.evaluate`, `consensus.get_report`, `consensus.list_reports`, `consensus.build_composer_input`, and `consensus.read`.
- Tests cover invalid claims/evidence, evidence tiers, Tier 3 factual limits, deterministic claim keys, conflicts, weighted scoring, quorum/risk policy, ledger determinism, service evaluation, accepted-claims-only composer payloads, capability gates, journal events, workspace scope, and no-second-authority invariants.
- Consensus accepted does not mutate Kernel truth, admit Courthouse evidence, create ContextBlocks, call runtime/model drivers, execute actions, or write memory.
- Phase 11E does not modify `crates/forgek-validate`; consensus fixtures may be added there later after the Go model stabilizes.
- Validation commands passed for this pass: `cd services/core && go test ./internal/forgek/...`, `npm run build:core`, `npm run lint`, `npm test`, `npm run test:forgek:parity`, and `git diff --check`.
