# FORGE-K Simulator

`services/core/internal/forgek` contains the deterministic FORGE-K simulator. It is simulator authority only: the live daemon does not import FORGE-K as live truth authority yet, and live state mutation still uses the existing AI-OS, gateway, permissions, lane, and audit paths.

Do not route live daemon state through FORGE-K without an explicit `LIVE_INTEGRATION` phase, integration design, capability and journal tests, and updated ADR/status docs.

## Test Command

Run the FORGE-K simulator tests from `services/core`:

```bash
go test ./internal/forgek/...
```

## Phase Package Map

- Phase 1 Kernel Simulator: `kernel.go`, `syscalls.go`, `objects.go`, `journal.go`, `capabilities.go`, `providers.go`, `case_syscalls.go`
- Phase 2 Neuron Fabric: `neurons`
- Phase 3 Courthouse Minimal: `court`, `court_syscalls.go`
- Phase 4 Memory Palace Minimal: `palace`, `palace_syscalls.go`
- Phase 5 Semantic Algebra: `semantic`, `semantic_syscalls.go`
- Phase 6 Snapshots: `snapshots`, `snapshot_syscalls.go`
- Phase 7 Context Compiler: `contextcompiler`, `context_syscalls.go`
- Phase 8 Deterministic KV System: `kv`, `kv_syscalls.go`
- Phase 9 Runtime Driver Boundary: `runtime`, `runtime_syscalls.go`
- Phase 10 Lymphatic Lane: `lymphatic`, `lymphatic_syscalls.go`
- Phase 11A Rust Kernel Core Research / Planning: docs only, no Rust code
- Phase 11B Rust Deterministic Validation Boundary: standalone crate outside this package at `crates/forgek-validate`
- Phase 11C Go/Rust Test Corpus Alignment: test-only parity in `fixture_parity_test.go`, shared fixtures outside this package at `fixtures/forgek`
- Phase 11D Rust Validation CI and Tooling Integration: root/CI tooling only, no package runtime dependency
- Phase 11E Consensus Mesh: `consensus`, `consensus_syscalls.go`
- Phase 11F Integration Readiness Contracts: `integrationready`
- Phase 11G Shadow Mode Harness Design: `shadowharness`
- Phase 12A Live Integration Design: docs only, no package code
- Phase 12B Read-only Shadow Harness Implementation: live-adjacent package outside this simulator at `services/core/internal/forgekshadow`
- Phase 12C Shadow Diagnostics Review and Hardening: live-adjacent hardening outside this simulator; root forbidden import guard added here
- Phase 12D Controlled Shadow Expansion Design: docs only, no package code
- Phase 12E Route Envelope Shadow Metadata: live-adjacent route-envelope observer outside this simulator at `services/core/internal/forgekshadow`
- Phase 12F Route Envelope Shadow Hardening: live-adjacent hardening outside this simulator; no new live touchpoint or simulator authority path
- Phase 12G Chat Metadata Expansion Design: docs only, no package code
- Phase 12H Chat Metadata Shadow Implementation: live-adjacent chat metadata observer outside this simulator at `services/core/internal/forgekshadow`
- Phase 12I Chat Metadata Shadow Hardening: live-adjacent hardening outside this simulator; no new live touchpoint or simulator authority path
- Phase 12J Retrieval Metadata Expansion Design: docs only, no package code
- Phase i1 Reality Alignment and Live KV Identity Validation: shared pure validation package outside this simulator at `services/core/internal/kvidentity`; simulator KV still calls it, but `KVService` remains simulator-only

## Authority Boundary

FORGE-K owns truth in the simulator model. In the current repository, FORGE-K is not live daemon authority. The simulator tests prove the target authority contracts for phases 1-10, while live daemon authority remains outside this package until a future scoped integration phase.

Phase 7 compiles ContextBlocks and ContextBundles as deterministic shape only. It does not call models, use KV cache, execute restore seeds, alter live AI-OS `COMPILE_CONTEXT`, or route live daemon state through FORGE-K.

Phase 8 registers deterministic KV manifests and validates identity gates as acceleration metadata only. It does not store real KV tensors, call model runtimes, perform live backend cache reuse, mutate ContextBundles or Snapshots, alter live AI-OS `COMPILE_CONTEXT`, or route live daemon state through FORGE-K. Phase i1 extracts the deterministic KV identity gates into `services/core/internal/kvidentity` and uses that package from live Control Lane validation-only `VALIDATE_KV_IDENTITY`; this does not make simulator `KVService` live authority and does not enable live KV reuse.

Phase 9 is implemented as `SIMULATOR_ONLY / DRIVER_BOUNDARY_ONLY`. Runtime drivers are governed driver surfaces that may return proposal output with manifests, capability metadata, context refs, and KV metadata refs. The active implementation is a deterministic mock driver only. It must not call real model backends, mutate Kernel objects except through runtime syscalls, admit evidence, write snapshots or ContextBundles, register KV manifests, perform live KV reuse, alter live `modelruntime`, change gateway behavior, add routes, or route live daemon state through FORGE-K.

Phase 10 is implemented as `SIMULATOR_ONLY`. The Lymphatic Lane produces deterministic Maintenance Reports and Cleanup Proposals only. It must not silently mutate source objects, delete provenance, wire into live daemon cleanup, change live dream/autonomy behavior, alter live `modelruntime`, change gateway behavior, add routes, or route live daemon state through FORGE-K.

Phase 11A is `RESEARCH_ONLY / DOCS_ONLY`. It records the possible Rust boundary for deterministic validation primitives only. The simulator remains Go-owned, live daemon authority remains outside FORGE-K, and no Rust crate or Rust integration exists yet.

Phase 11B is `RESEARCH_ONLY / SIMULATOR_ONLY`. It adds a standalone Rust crate at `crates/forgek-validate` for deterministic fixture validation and shared fixtures under `fixtures/forgek`. It must not add Rust code to this Go package, import Rust from Go, use cgo, add live daemon wiring, change routes, change gateway behavior, alter `modelruntime`, or change live AI-OS controllane behavior.

Phase 11C is `RESEARCH_ONLY / SIMULATOR_ONLY`. It adds Go test-only parity checks for the shared fixture corpus in `fixture_parity_test.go`. The tests load fixtures and golden files directly; they do not invoke Rust, add cgo, alter production Go code, or make Rust required for normal Go runtime execution.

Phase 11D is `RESEARCH_ONLY / SIMULATOR_ONLY / TOOLING_ONLY`. It wires Rust validation commands into CI and root helper scripts only. It does not add Rust to this Go package or make root `npm test` depend on Rust.

Phase 11E is `SIMULATOR_ONLY / GOVERNANCE_LAYER_ONLY`. The Consensus Mesh governs claim acceptance for response/action proposal shaping. It does not become truth, admit Courthouse evidence, write memory, execute actions, call runtime/model drivers, create ContextBlocks, or route live daemon state through FORGE-K.

Phase 11F is `SIMULATOR_ONLY / INTEGRATION_PREP_ONLY`. The `integrationready` package defines diagnostic readiness reports, live path mappings, read-only adapter contracts, read-only RAG/retrieval boundaries, and shadow-mode policy. It has no syscalls, no Kernel ownership, no live daemon imports, no API routes, no gateway/modelruntime/controllane behavior, no live retrieval or embedding calls, no live RAG, and no live memory mutation.

Phase 11G is `SIMULATOR_ONLY / SHADOW_DESIGN_ONLY`. The `shadowharness` package defines simulator-only observation, comparison report, subreport, policy, and no-effect validation contracts for a future read-only shadow harness. It has no syscalls, no Kernel ownership, no live daemon imports, no API routes, no live observation, no gateway/modelruntime/controllane behavior, no live retrieval or embedding calls, no live RAG, no live memory mutation, and no user-visible output authority.

Phase 12A is `DOCS_ONLY / LIVE_INTEGRATION_DESIGN_ONLY`. It recorded the Phase 12B read-only shadow harness design in architecture/review/testing docs. It adds no Go code, no syscalls, no Kernel ownership, no live daemon imports, no route/API changes, no live observation, no gateway/modelruntime/controllane behavior, no live retrieval or embedding calls, no live RAG, no memory writes, and no user-visible output authority.

Phase 12B is `LIVE_INTEGRATION / READ_ONLY / DISABLED_BY_DEFAULT` and lives outside this simulator package in `services/core/internal/forgekshadow`. It observes `/health` request metadata only when `FORGE_K_SHADOW_MODE_ENABLED=true`, writes bounded in-memory diagnostic reports only, and reuses `shadowharness` for no-effect validation. It does not add syscalls, Kernel ownership, public routes, public APIs, live retrieval, embedding calls, live RAG, modelruntime calls, tool execution, memory writes, controllane mutation, gateway behavior changes, or user-visible output authority.

Phase 12C is `LIVE_INTEGRATION / OBSERVABILITY_ONLY / HARDENING_ONLY`. It hardens `forgekshadow` diagnostics without adding touchpoints. `/health` remains the only observed live route, diagnostics remain in-memory only, and this simulator package still must not import live daemon authority packages.

Phase 12D is `DOCS_ONLY / LIVE_INTEGRATION_DESIGN_ONLY`. It recommended route envelope metadata as the next Phase 12E candidate and recorded the required test plan. It added no code to this package, no new live observation, no route/API changes, no public diagnostics routes, no gateway/modelruntime/retrieval/memory/controllane behavior, no live RAG, no feature flag changes, and no authority migration.

Phase 12E is `LIVE_INTEGRATION / READ_ONLY / DISABLED_BY_DEFAULT` and lives outside this simulator package in `services/core/internal/forgekshadow`. It observes route-envelope metadata only when `FORGE_K_SHADOW_MODE_ENABLED=true`: method, matched route pattern, normalized route class, duration, and safe request identifiers. It does not capture request bodies, response bodies, prompts, model outputs, retrieval content, headers, cookies, secrets, or raw query strings; it does not add syscalls, Kernel ownership, public routes, public APIs, live retrieval, embedding calls, live RAG, modelruntime calls, tool execution, memory writes, controllane mutation, gateway behavior changes, or user-visible output authority.

Phase 12F is `LIVE_INTEGRATION / OBSERVABILITY_ONLY / HARDENING_ONLY` and lives outside this simulator package in `services/core/internal/forgekshadow`. It hardens the Phase 12E route-envelope observer by preferring matched route patterns, rejecting unsafe raw dynamic route patterns, normalizing provided route classes, rejecting raw query/request URI metadata, preventing metadata from reintroducing path or route pattern values, expanding secret/header/content redaction, enforcing deterministic scalar metadata, preserving bounded retention, and isolating sink failure. It adds no new live touchpoints, syscalls, Kernel ownership, public routes, public APIs, persistence, live retrieval, embedding calls, live RAG, modelruntime calls, tool execution, memory writes, controllane mutation, gateway behavior changes, route behavior changes, or user-visible output authority.

Phase 12G is `DOCS_ONLY / LIVE_INTEGRATION_DESIGN_ONLY`. It designs the chat metadata shadow expansion and adds no Go code to this package or to `forgekshadow`. It does not observe chat routes, capture message content, capture prompts, capture completions, capture request or response bodies, add routes, add feature flags, call modelruntime, execute tools, query retrieval/search/embeddings, write memory, call controllane mutations, or route live daemon state through FORGE-K.

Phase 12H is `LIVE_INTEGRATION / READ_ONLY / DISABLED_BY_DEFAULT` and lives outside this simulator package in `services/core/internal/forgekshadow`. It observes bounded chat metadata only when both `FORGE_K_SHADOW_MODE_ENABLED=true` and `FORGE_K_SHADOW_CHAT_METADATA_ENABLED=true`. The live touchpoint is the existing chat message POST handler after request ownership is established. It does not capture message content, prompts, completions, assistant response text, system prompts, tool payloads, tool outputs, request bodies, response bodies, retrieval content, source chunks, memory content, auth headers, cookies, bearer tokens, API keys, or secrets. It adds no syscalls, Kernel ownership, public routes, public APIs, live retrieval, embedding calls, live RAG, modelruntime calls, tool execution, memory writes, controllane mutation, gateway behavior changes, response changes, or user-visible output authority.

Phase 12I is `LIVE_INTEGRATION / OBSERVABILITY_ONLY / HARDENING_ONLY` and lives outside this simulator package in `services/core/internal/forgekshadow` and API tests. It hardens the Phase 12H chat metadata observer with dual-flag, enum/ref normalization, redaction, invalid-body/header/query/SSE no-capture, sink behavior, and no-effect tests. It adds no new touchpoints, syscalls, Kernel ownership, public routes, public APIs, live retrieval, embedding calls, live RAG, modelruntime calls, tool execution, memory writes, controllane mutation, gateway behavior changes, response changes, or user-visible output authority.

Phase 12J is `DOCS_ONLY / LIVE_INTEGRATION_DESIGN_ONLY`. It designs a possible future retrieval metadata shadow expansion and adds no Go code to this package or to `forgekshadow`. It does not observe retrieval routes, capture source text, capture chunks, capture embeddings or vectors, capture raw queries, implement live RAG, add routes, add feature flags, call modelruntime, execute tools, query retrieval/search/embeddings, write memory, call controllane mutations, or route live daemon state through FORGE-K.

## Future Rust Boundary

What remains Go for now:

- Kernel orchestration and syscall dispatch
- simulator service ownership and object registry mutation
- Courthouse policy decisions
- Memory Palace retrieval scoring
- Semantic Algebra operation execution and future planning
- Context Compiler block selection policy
- Runtime driver invocation and backend integration
- Lymphatic sweep policy and cleanup proposal generation
- Consensus fixture validation in Rust until the Go consensus model stabilizes
- all live daemon, gateway, route, controllane, and modelruntime behavior

What Phase 11B begins validating in Rust and Phase 11C aligns with Go tests:

- canonical serialization validation
- deterministic hashing
- ref normalization and ID validation
- capability predicate evaluation from immutable inputs
- journal hash-chain verification
- Snapshot, ContextBlock, ContextBundle, KVCacheManifest, and RuntimeDriverManifest validation
- KV nine-gate validation

Phase 11B/11C validation:

- Go simulator tests remain in this package: `cd services/core && go test ./internal/forgek/...`
- Rust crate tests run outside this package: `cd crates/forgek-validate && cargo test`
- Fixture validation runs through the standalone CLI: `npm run validate:forgek-fixtures`
- Go/Rust parity can be run explicitly with `npm run test:forgek:parity`
- current package state: no Rust code, Cargo metadata, cgo, or Go production calls exist here
