# FORGE-K Current Phase Status

Companion to `docs/reviews/full_project_review.md` (2026-05-03).

This is a concise status read of FORGE-K phases against the current repository. The key distinction is that Phase 1-7 are implemented in the simulator package `services/core/internal/forgek`, while the live daemon still uses the existing AI-OS/gateway/permissions/lane/audit authority paths. ADR 0005 records that FORGE-K is target architecture, not live daemon authority yet.

| Phase | Title | Status | Where It Lives | Tests / Evidence | Open Work |
| --- | --- | --- | --- | --- | --- |
| 0 | Architecture Baseline | IMPLEMENTED | `docs/architecture/*`, ADRs 0001-0005, glossary, roadmap, DoD, diagrams. | Documentation is present and internally consistent. | Keep live-authority boundary visible in future phase reports. |
| 1 | Kernel Simulator | IMPLEMENTED + TESTED | `services/core/internal/forgek/{kernel,types,objects,syscalls,journal,capabilities,providers,case_syscalls}.go`. | `go test ./internal/forgek/...` passes. | Persistence and live daemon integration deferred. |
| 2 | Neuron Fabric | IMPLEMENTED + TESTED | `services/core/internal/forgek/neurons/*`. | Manifest, envelope, scheduler, neural/rule, syscall boundary tests pass. | Runtime/model neurons deferred. |
| 3 | Courthouse Minimal | IMPLEMENTED + TESTED | `services/core/internal/forgek/court/*`, `court_syscalls.go`. | Court model and syscall tests pass. | Full adjudication, claim extraction, precedent reasoning deferred. |
| 4 | Memory Palace Minimal | IMPLEMENTED + TESTED | `services/core/internal/forgek/palace/*`, `palace_syscalls.go`. | Palace model/scoring/syscall tests pass. | Embeddings/vector retrieval deferred. |
| 5 | Semantic Algebra | IMPLEMENTED + TESTED | `services/core/internal/forgek/semantic/*`, `semantic_syscalls.go`. | Semantic model/operator/syscall tests pass. | Advanced algebra policy/planning deferred. |
| 6 | Snapshots | IMPLEMENTED + TESTED | `services/core/internal/forgek/snapshots/*`, `snapshot_syscalls.go`, `docs/architecture/snapshots.md`, ADR 0003. Scope recorded as `SIMULATOR_ONLY` in the roadmap. | Snapshot model/service/diff/restore-seed/syscall tests pass under `go test ./internal/forgek/...`. | Persistence and live daemon integration remain deferred. |
| 7 | Context Compiler | IMPLEMENTED + TESTED | `services/core/internal/forgek/contextcompiler/*`, `context_syscalls.go`, `docs/architecture/context_compiler_and_kv_cache.md`. Scope recorded as `SIMULATOR_ONLY`; live `aios/controllane/compile_context_*` is a separate legacy path. | ContextBlock, ContextBundle, PromptLayout, deterministic serialization, hashing, compile service, snapshot/restore-seed integration, syscall, capability, journal, and shape-not-truth tests pass under `go test ./internal/forgek/...`. | Live daemon integration, live COMPILE_CONTEXT replacement, runtime drivers, tokenizer-specific token IDs, and deterministic KV cache remain deferred. |
| 8 | Deterministic KV System | DOCUMENTED ONLY | Context/KV doc and ADR 0004. | No implementation. | Implement KVCacheManifest, nine-gate validation, tiers. |
| 9 | Runtime Driver Integration | PARTIAL OUTSIDE FORGE-K | Live `modelruntime`, `gateway`, `aios/iolane`. | Live runtime tests exist; no FORGE-K wrapper tests. | Add driver boundary only under an explicit future scope decision. |
| 10 | Lymphatic Lane | PARTIAL OUTSIDE FORGE-K | Live dream/autonomy cleanup-style paths. | Live dream tests exist. | Implement FORGE-K lymphatic scheduler later. |
| 11 | Rust Kernel Core | NOT STARTED | None. | None. | Future work. |
| 12 | FORGE Daemon | PARTIAL OUTSIDE FORGE-K | Existing `services/core/main.go` daemon. | Live daemon tests exist indirectly. | Not FORGE-K-governed yet. |
| 13 | FORGE-1 Simulator | NOT STARTED | Concept doc only. | None. | Future research. |
| 14 | FORGE-1 Prototype Research | DOCUMENTED CONCEPT ONLY | `docs/architecture/forge_1_cpu_concept.md`. | None. | Future research. |

## Readiness Notes

- `go test ./internal/forgek/...` passes, including Phase 6 snapshot tests and Phase 7 Context Compiler tests.
- Representative API route inventory tests pass.
- `npm run build:core`, `npm run lint`, and `npm test` pass in this Phase 7 pass.
- Desktop typecheck/build is blocked by local Node workspace package resolution.
- FORGE-K remains simulator authority only; the live daemon still uses AI-OS/gateway/permissions/lane/audit authority paths.
- The safest next path is Phase 8 Deterministic KV planning under an explicit scope marker; do not wire Phase 7 into the live daemon without a `LIVE_INTEGRATION` design and tests.

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
