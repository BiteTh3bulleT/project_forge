# FORGE-K Build Phases

Status: Phase 11A Rust Kernel Core research/planning complete; Phase 11B Rust deterministic validation crate and Phase 11C Go/Rust test corpus alignment are implemented as `RESEARCH_ONLY / SIMULATOR_ONLY`.

Each phase must preserve the doctrine that models are drivers, neural outputs are proposals, rule outputs are validations, Courthouse admits evidence, Kernel commits through semantic syscalls, snapshots preserve shape, and KV cache is acceleration only.

## Scope Markers

Every future FORGE-K phase must declare one scope marker before work starts:

- `SIMULATOR_ONLY`: confined to `services/core/internal/forgek`, docs, and tests; live daemon authority is unchanged.
- `LIVE_INTEGRATION`: intentionally changes live daemon authority or routes live state through FORGE-K boundaries; requires explicit integration design and tests.
- `DOCS_ONLY`: documentation, status, or planning only.
- `RESEARCH_ONLY`: exploratory work that cannot be treated as production authority.

Current live-authority boundary: ADR 0005 records that FORGE-K is target architecture but not live daemon authority yet. Live daemon state mutation still uses existing AI-OS/gateway/permissions/lane/audit paths.

## Phase 0 - Architecture Baseline

Scope: `DOCS_ONLY`.

Goal: establish durable doctrine, terminology, boundaries, ADRs, diagrams, glossary, and roadmap.

Deliverables: architecture docs, AGENTS guidance, README links, ADRs, Mermaid diagrams, glossary, definition of done.

Validation criteria: required docs exist; diagrams parse; no runtime logic introduced; Truth, Shape, and Acceleration are clearly separated.

What not to do: implement kernel runtime, agent loops, runtime drivers, cache systems, or memory mutation paths.

## Phase 1 - Kernel Simulator

Scope: `SIMULATOR_ONLY`.

Goal: create a minimal userspace simulator for semantic syscall lifecycle and journal behavior.

Deliverables: syscall envelope types, validation stubs, in-memory journal simulator, object registry, capability manager, CasePacket lifecycle tests.

Validation criteria: deterministic accept/reject behavior; replayable journal events; no model dependency.

What not to do: build full persistence, production daemon, model integration, or broad memory system.

Implementation status: initial userspace simulator lives in `services/core/internal/forgek` with architecture notes in `docs/architecture/kernel_simulator.md`.

## Phase 2 - Neuron Fabric

Scope: `SIMULATOR_ONLY`.

Goal: implement typed neuron envelopes and bounded scheduling primitives.

Deliverables: neuron input/output contracts, neural proposal interfaces, rule validation interfaces, syscall request envelopes, scheduler tests.

Validation criteria: neural output cannot commit; rule output cannot create truth; invalid authority attempts fail tests.

What not to do: create monolithic agent loops or model-owned state.

Implementation status: initial Neuron Fabric lives in `services/core/internal/forgek/neurons` with manifest, envelope, neural/rule base behavior, scheduler, and narrow Kernel syscall client tests.

## Phase 3 - Courthouse Minimal

Scope: `SIMULATOR_ONLY`.

Goal: implement minimal evidence admission for cases, claims, exhibits, and rulings.

Deliverables: CasePacket model, exhibit submission, admissibility checks, ruling records.

Validation criteria: rejected evidence records reasons; admitted evidence is explainable; contradictions are not silently merged.

What not to do: treat retrieval scores as admission or rulings as bypasses for Kernel commit.

Implementation status: minimal Courthouse models, service, kernel-owned semantic syscalls, capability gates, and journaled admissibility transitions live in `services/core/internal/forgek/court` and `services/core/internal/forgek`.

## Phase 4 - Memory Palace Minimal

Scope: `SIMULATOR_ONLY`.

Goal: implement scoped retrieval topology for rooms, anchors, routes, and candidate objects.

Deliverables: room/anchor/route contracts, candidate retrieval, provenance-aware results.

Validation criteria: candidates remain non-admitted until Courthouse review; scope boundaries are enforced.

What not to do: build raw chat memory or make vector search authoritative.

Implementation status: minimal Memory Palace models, deterministic route scoring, service, kernel-owned semantic syscalls, capability gates, journaled route traces, and Courthouse boundary tests live in `services/core/internal/forgek/palace` and `services/core/internal/forgek`.

## Phase 5 - Semantic Algebra

Scope: `SIMULATOR_ONLY`.

Goal: implement typed semantic objects and deterministic operators.

Deliverables: object schemas, operator library, invariant tests, provenance-preserving transformations.

Validation criteria: compression cannot create truth; derived objects cite sources; superseded objects remain inspectable.

What not to do: make algebra operators bypass semantic syscalls for canonical mutation.

Implementation status: minimal SemanticObject, SemanticOperation, SemanticTransformResult, operator registry, deterministic operators, semantic operation syscalls, capability gates, and journaled provenance-preserving transforms live in `services/core/internal/forgek/semantic` and `services/core/internal/forgek`.

## Phase 6 - Snapshots

Scope: `SIMULATOR_ONLY`.

Goal: implement Context-Shape Snapshots for restoration and inspection.

Deliverables: snapshot schemas, source refs, hashes, operation records, supersession behavior.

Validation criteria: snapshots are non-canonical; snapshots cite sources; restoration never overrides truth.

What not to do: store full canonical content copies or treat snapshots as current state.

Implementation status: implemented and tested in `services/core/internal/forgek/snapshots` and `services/core/internal/forgek/snapshot_syscalls.go`. Scope is simulator-only; snapshots are not wired into the live daemon, and live state is not routed through FORGE-K during this phase.

## Phase 7 - Context Compiler

Scope: `SIMULATOR_ONLY`.

Goal: implement deterministic context compilation and the expansion/contraction loop.

Deliverables: ContextBlock schema, stable prompt layout, compiler budgets, token-addressed output.

Validation criteria: same admitted inputs produce stable block order and token hashes; rejected evidence stays out.

What not to do: let model prose decide context admission or layout authority.

Implementation status: implemented and tested in `services/core/internal/forgek/contextcompiler` and `services/core/internal/forgek/context_syscalls.go`. Scope is simulator-only; the Context Compiler is not wired into the live daemon, live AI-OS `COMPILE_CONTEXT` behavior is unchanged, and live state is not routed through FORGE-K during this phase.

## Phase 8 - Deterministic KV System

Scope: `SIMULATOR_ONLY`.

Goal: implement deterministic KV cache manifests and validation gates.

Deliverables: KVCacheManifest, cache tiers, Nine-Gate validation, invalidation logic.

Validation criteria: cache reuse requires all nine gates; cache cannot act as memory; misses fall back safely.

What not to do: reuse KV across model/tokenizer/template/layout/schema/runtime mismatches.

Implementation status: implemented and tested in `services/core/internal/forgek/kv` and `services/core/internal/forgek/kv_syscalls.go`. Scope is simulator-only; the KV system stores metadata and validation records only, does not store real runtime KV tensors, does not call model runtimes, does not perform live backend cache reuse, and is not wired into the live daemon.

## Phase 9 - Runtime Driver Integration

Scope: `SIMULATOR_ONLY / DRIVER_BOUNDARY_ONLY`.

Goal: define the simulator boundary that treats model runtimes as governed drivers and keeps runtime output proposal-only.

Deliverables: RuntimeDriver interface, RuntimeDriverManifest, RuntimeCapabilityManifest, RuntimeGenerateRequest, RuntimeGenerateResult, RuntimeDriverRegistry, RuntimeService or RuntimeBroker, deterministic MockRuntimeDriver, context refs by reference only, and KV metadata refs by reference only.

Validation criteria: runtime output cannot mutate canonical state; manifests do not grant authority; capability and policy checks remain Kernel-owned; context refs are not admitted or compiled by drivers; KV metadata does not become live KV reuse.

What not to do: wire FORGE-K into the live daemon, replace live `modelruntime`, change public APIs or routes, call real OpenAI/Ollama/vLLM/SGLang/TensorRT backends, perform real KV cache reuse, or let runtime drivers own authority.

Implementation status: implemented and tested in `services/core/internal/forgek/runtime` and `services/core/internal/forgek/runtime_syscalls.go`. Scope is simulator-only/driver-boundary-only; the active driver is deterministic mock-only, runtime output is proposal-only, and the implementation is not wired into the live daemon.

## Phase 10 - Lymphatic Lane

Scope: `SIMULATOR_ONLY`.

Goal: implement deferred simulator maintenance for cleanup review, contradictions, stale loops, cache hygiene, snapshot hygiene, runtime result hygiene, and compaction proposals.

Deliverables: Lymphatic scheduler, Lymphatic Sweep contracts, Maintenance Reports, Cleanup Proposals, policy envelopes, hygiene findings, and syscall-backed cleanup proposal paths.

Validation criteria: maintenance work is deferable and deterministic; reports/proposals are evidence only; cleanup mutations require semantic syscalls; audit/provenance evidence exists; no source object is silently mutated.

What not to do: run full maintenance on every turn, delete provenance, mutate canonical truth directly, wire into live daemon cleanup, change live dream/autonomy behavior, add routes, change gateway/modelruntime behavior, or modify live AI-OS controllane behavior.

Implementation status: implemented and tested in `services/core/internal/forgek/lymphatic`, `services/core/internal/forgek/lymphatic_syscalls.go`, and related tests. Scope is simulator-only; maintenance reports and cleanup proposals are proposal/evidence surfaces only, and the implementation is not wired into the live daemon.

## Phase 11A - Rust Kernel Core Research / Planning

Scope: `RESEARCH_ONLY / DOCS_ONLY`.

Goal: decide whether a future Rust Kernel Core is justified, what the safe boundary should be, and which FORGE-K primitives are stable enough to port later.

Deliverables: Rust boundary plan, readiness review, ADR 0006, package stability assessment, data serialization requirements, risk register, and Phase 11B recommendation.

Validation criteria: docs exist; `go test ./internal/forgek/...` passes before and after the planning pass; no Rust code, crate, live daemon integration, public API, route, gateway, modelruntime, or Go behavior changes are introduced.

Implementation status: completed as a research/docs pass in `docs/architecture/rust_kernel_core_plan.md`, `docs/reviews/phase_11_readiness.md`, and ADR 0006. Rust implementation remains not started.

What not to do: implement Rust, add `Cargo.toml`, add a Rust crate, wire FORGE-K into the live daemon, change syscalls, create a second authority path, or introduce untested unsafe authority paths.

## Phase 11B - Rust Deterministic Validation Crate

Scope: `RESEARCH_ONLY / SIMULATOR_ONLY` unless a later prompt explicitly records a different safe scope.

Goal: implement the simulator-only boundary for a standalone Rust crate and CLI test harness for deterministic validation primitives.

Allowed deliverables in this research/simulator pass: fixture family selection, canonical serialization requirements, validation result vocabulary, CLI input/output contract, explicit non-integration guardrails, standalone Rust validation crate, shared fixtures, and root helper scripts that do not affect `npm test`.

Implemented deliverables: canonical JSON normalization, hash validation over stable projections, first-pass capability-like fixture validation, manifest validation, shared fixture/golden-file checks, and CLI commands `validate`, `canonicalize`, `hash`, and `validate-fixtures`.

Validation criteria for this research pass: Rust crate tests pass, fixture validation passes, Go simulator tests still pass, and docs record that no cgo bridge, live daemon integration, Go production call, public API, route, gateway, modelruntime, or live controllane behavior changes were introduced. Do not claim Go or Rust test success unless the command was run and recorded for the specific pass.

Validation criteria for a future implementation pass: Rust and Go agree on shared fixtures; no live daemon integration; no cgo; no public API, route, gateway, modelruntime, or live controllane behavior changes.

What not to do: replace the Go simulator, call model runtimes, mutate state, add live authority, add cgo, add routes, call Rust from Go production code, or add Rust dependencies to CI without explicit approval.

Implementation status: standalone crate implemented in `crates/forgek-validate`; fixtures implemented in `fixtures/forgek`; helper scripts added as `test:rust:forgek` and `validate:forgek-fixtures`; CI wiring is not implemented.

## Phase 11C - Go/Rust Test Corpus Alignment

Scope: `RESEARCH_ONLY / SIMULATOR_ONLY`.

Goal: expand shared deterministic fixtures and parity tests after Phase 11B proves the initial crate boundary.

Implemented deliverables: shared fixture schema notes, valid/invalid fixtures for Snapshot, ContextBlock, ContextBundle, KVCacheManifest, and RuntimeDriverManifest, canonical serialization golden files, expanded hash golden manifest, Go test-only fixture parity tests, Rust fixture validation/drift tests, and the optional root helper `test:forgek:parity`.

Deferred deliverables: KernelObject, richer Capability, JournalEvent, MaintenanceReport, CleanupProposal, and KV gate-specific failure fixtures.

Validation criteria: fixture corpus is versioned, language-neutral, deterministic, and does not create live daemon authority. Go and Rust fixture checks pass, and root `npm test` remains independent of Rust.

Implementation status: implemented in `services/core/internal/forgek/fixture_parity_test.go`, `crates/forgek-validate`, `fixtures/forgek`, and `scripts/forgek-parity.mjs`. No live daemon integration, cgo, Go production Rust call, public API, route, gateway, modelruntime, controllane, or CI dependency was added.

What not to do: treat fixture parity as live integration, call Rust from Go production code, make Rust required for normal Go runtime execution, or bypass ADR 0005.

## Phase 12 - FORGE Daemon

Scope: `LIVE_INTEGRATION`.

Goal: expose FORGE-K as a governed local daemon.

Deliverables: daemon process, local API, policy loading, journal persistence, runtime-driver isolation.

Validation criteria: daemon enforces semantic syscalls; audit and journal are durable; drivers remain outside authority.

What not to do: expose uncontrolled tool execution or remote authority by default.

## Phase 13 - FORGE-1 Simulator

Scope: `RESEARCH_ONLY`.

Goal: simulate candidate FORGE-1 instruction families and hardware blocks.

Deliverables: simulator contracts, instruction traces, workload benchmarks, correctness tests.

Validation criteria: simulator behavior matches documented kernel semantics; performance claims cite evidence.

What not to do: claim hardware readiness or replace the userspace kernel path.

## Phase 14 - FORGE-1 Prototype Research

Scope: `RESEARCH_ONLY`.

Goal: evaluate hardware/software co-design feasibility for governed execution acceleration.

Deliverables: research notes, prototype constraints, benchmark methodology, risk register.

Validation criteria: research remains grounded in measured workloads and simulator evidence.

What not to do: make FORGE-1 a production dependency or imply GPUs are obsolete.
