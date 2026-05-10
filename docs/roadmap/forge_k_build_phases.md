# FORGE-K Build Phases

Status: Phase 11A Rust Kernel Core research/planning complete; Phase 11B Rust deterministic validation crate and Phase 11C Go/Rust test corpus alignment are implemented as `RESEARCH_ONLY / SIMULATOR_ONLY`. Phase 11D Rust Validation CI and Tooling Integration is implemented as `RESEARCH_ONLY / SIMULATOR_ONLY / TOOLING_ONLY`. Phase 11E Consensus Mesh is implemented as `SIMULATOR_ONLY / GOVERNANCE_LAYER_ONLY`. Phase 11F Integration Readiness Contracts is implemented as `SIMULATOR_ONLY / INTEGRATION_PREP_ONLY`. Phase 11G Shadow Mode Harness Design is implemented as `SIMULATOR_ONLY / SHADOW_DESIGN_ONLY`. Phase 12A Live Integration Design is implemented as `DOCS_ONLY / LIVE_INTEGRATION_DESIGN_ONLY`. Phase 12B Read-only Shadow Harness Implementation is implemented as `LIVE_INTEGRATION / READ_ONLY / DISABLED_BY_DEFAULT`. Phase 12C Shadow Diagnostics Review and Hardening is implemented as `LIVE_INTEGRATION / OBSERVABILITY_ONLY / HARDENING_ONLY`. Phase 12D Controlled Shadow Expansion Design is implemented as `DOCS_ONLY / LIVE_INTEGRATION_DESIGN_ONLY`. Phase 12E Route Envelope Shadow Metadata is implemented as `LIVE_INTEGRATION / READ_ONLY / DISABLED_BY_DEFAULT`. Phase 12F Route Envelope Shadow Hardening is implemented as `LIVE_INTEGRATION / OBSERVABILITY_ONLY / HARDENING_ONLY`. Phase 12G Chat Metadata Expansion Design is implemented as `DOCS_ONLY / LIVE_INTEGRATION_DESIGN_ONLY`. Phase 12H Chat Metadata Shadow Implementation is implemented as `LIVE_INTEGRATION / READ_ONLY / DISABLED_BY_DEFAULT`. Phase 12I Chat Metadata Shadow Hardening is implemented as `LIVE_INTEGRATION / OBSERVABILITY_ONLY / HARDENING_ONLY`. Phase 12J Retrieval Metadata Expansion Design is implemented as `DOCS_ONLY / LIVE_INTEGRATION_DESIGN_ONLY`. Phase 12K Retrieval Metadata Shadow Implementation and Phase 12L Retrieval Metadata Shadow Hardening are implemented/tested as a combined `LIVE_INTEGRATION / READ_ONLY / DISABLED_BY_DEFAULT / HARDENED_IN_PASS` metadata-only pass. Phase 12M-Q Shadow Advisory Pipeline is implemented/tested as `LIVE_INTEGRATION / READ_ONLY / DISABLED_BY_DEFAULT / ADVISORY_DIAGNOSTIC_ONLY`. Phase 13A Storage Backend Foundation is implemented/tested as `LIVE_INFRA / STORAGE_FOUNDATION / DEFAULT_SQLITE`. Phase 13B-C Postgres Schema Foundation and SQLite/Postgres Parity is implemented/tested as `LIVE_INFRA / STORAGE_PARITY / DEFAULT_SQLITE`. Phase 13D-E Diagnostic Persistence and Retrieval Metadata Relational Adapter is implemented/tested as `LIVE_INFRA / DIAGNOSTIC_STORAGE / DEFAULT_SQLITE / DISABLED_BY_DEFAULT`. Phase 13F-G Qdrant Shadow Vector Adapter is implemented/tested as `LIVE_INFRA / VECTOR_SHADOW / DISABLED_BY_DEFAULT / NON_AUTHORITATIVE`. Phase 13H Redis Queue/Cache Boundary is implemented/tested as `LIVE_INFRA / EPHEMERAL_COORDINATION / DISABLED_BY_DEFAULT / NON_CANONICAL`. Phase 13I Store Cutover Readiness Review is implemented as `DOCS_ONLY / READINESS_REVIEW`. Phase i1 Reality Alignment and Live KV Identity Validation is implemented/tested as `PARTIAL LIVE VALIDATION / NO LIVE KV REUSE`. PhaseI2 Live KV Identity Enforcement and Observation is implemented/tested as `PARTIAL LIVE ENFORCEMENT / NO LIVE KV REUSE`. Phase 14A FORGE-K Operational Cutover Design is implemented as `DOCS_ONLY / LIVE_AUTHORITY_MIGRATION_DESIGN_ONLY`. Phase 14B Ref Shape Validation is implemented/tested as `PARTIAL LIVE VALIDATION / CONTROL_LANE / NO_AUTHORITY_REPLACEMENT`. Phase 14C Control Lane Validation Expansion is implemented/tested as `PARTIAL LIVE VALIDATION / CONTROL_LANE / SHADOW_COMPARE / NO_AUTHORITY_REPLACEMENT`. Phase 14D Control Lane Validation Shadow Reporting is implemented/tested as `LIVE_INTEGRATION / READ_ONLY / DISABLED_BY_DEFAULT / CONTROL_LANE_VALIDATION_DIAGNOSTICS`.

Each phase must preserve the doctrine that models are drivers, neural outputs are proposals, rule outputs are validations, Courthouse admits evidence, Kernel commits through semantic syscalls, snapshots preserve shape, and KV cache is acceleration only.

## Scope Markers

Every future FORGE-K phase must declare one scope marker before work starts:

- `SIMULATOR_ONLY`: confined to `services/core/internal/forgek`, docs, and tests; live daemon authority is unchanged.
- `LIVE_INTEGRATION`: touches the live daemon path. Read-only live diagnostics must also declare `READ_ONLY` and `DISABLED_BY_DEFAULT`; authority migration or live state mutation requires explicit authority-migration design and tests.
- `DOCS_ONLY`: documentation, status, or planning only.
- `RESEARCH_ONLY`: exploratory work that cannot be treated as production authority.
- `READINESS_REVIEW`: docs-only review of gates and blockers; it does not authorize migration.
- `LIVE_AUTHORITY_MIGRATION_DESIGN_ONLY`: docs-only design for future live authority migration; it does not execute migration.

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

What not to do: replace the Go simulator, call model runtimes, mutate state, add live authority, add cgo, add routes, call Rust from Go production code, or add Rust dependencies to CI without an explicit tooling phase.

Implementation status: standalone crate implemented in `crates/forgek-validate`; fixtures implemented in `fixtures/forgek`; helper scripts added as `test:rust:forgek` and `validate:forgek-fixtures`. CI wiring is implemented later in Phase 11D.

## Phase 11C - Go/Rust Test Corpus Alignment

Scope: `RESEARCH_ONLY / SIMULATOR_ONLY`.

Goal: expand shared deterministic fixtures and parity tests after Phase 11B proves the initial crate boundary.

Implemented deliverables: shared fixture schema notes, valid/invalid fixtures for Snapshot, ContextBlock, ContextBundle, KVCacheManifest, and RuntimeDriverManifest, canonical serialization golden files, expanded hash golden manifest, Go test-only fixture parity tests, Rust fixture validation/drift tests, and the optional root helper `test:forgek:parity`.

Deferred deliverables: KernelObject, richer Capability, JournalEvent, MaintenanceReport, CleanupProposal, and KV gate-specific failure fixtures.

Validation criteria: fixture corpus is versioned, language-neutral, deterministic, and does not create live daemon authority. Go and Rust fixture checks pass, and root `npm test` remains independent of Rust.

Implementation status: implemented in `services/core/internal/forgek/fixture_parity_test.go`, `crates/forgek-validate`, `fixtures/forgek`, and `scripts/forgek-parity.mjs`. No live daemon integration, cgo, Go production Rust call, public API, route, gateway, modelruntime, controllane, or CI dependency was added.

What not to do: treat fixture parity as live integration, call Rust from Go production code, make Rust required for normal Go runtime execution, or bypass ADR 0005.

## Phase 11D - Rust Validation CI and Tooling Integration

Scope: `RESEARCH_ONLY / SIMULATOR_ONLY / TOOLING_ONLY`.

Goal: make the Phase 11B Rust validator and Phase 11C Go/Rust parity corpus visible in local tooling and CI without creating a live Rust authority path.

Implemented deliverables: stable Rust setup in `.github/workflows/ci.yml`, separate CI steps for `npm run test:rust:forgek`, `npm run validate:forgek-fixtures`, and `npm run test:forgek:parity`, plus the optional grouped helper `npm run validate:forgek`.

Validation criteria: CI failure surfaces stay separate; root `npm test` remains Go/core-only and does not depend on Rust; Rust validation remains fixture/parity tooling only; existing Go tests, lint, desktop typecheck/build, and smoke CI steps remain preserved.

Implementation status: implemented in `package.json`, `.github/workflows/ci.yml`, `docs/testing/rust_validation.md`, README/status docs, and the Rust crate README. No live daemon integration, cgo, Go production Rust call, public API, route, gateway, modelruntime, controllane, or runtime behavior change was added.

What not to do: treat Rust CI checks as live integration, make root `npm test` depend on Rust, call Rust from Go production code, or use validator results as canonical mutation authority.

## Phase 11E - Consensus Mesh

Scope: `SIMULATOR_ONLY / GOVERNANCE_LAYER_ONLY`.

Goal: implement governed claim acceptance for response/action proposal shaping without creating a second Kernel authority path.

Implemented deliverables: Claim, EvidenceRef, AgentRun, ClaimLedger, claim canonicalization, conflict detection, ConsensusPolicy, weighted support scoring, quorum checks, ConsensusDecision, ConsensusReport, ComposerGuard, ConsensusService, consensus syscalls, capability gates, journal events, and accepted-claims-only tests.

Validation criteria: unsupported factual claims are rejected or marked needs-more-evidence; Tier 3 inference cannot sole-support facts; conflicts block acceptance; composer payloads exclude rejected/raw proposed claims; accepted consensus claims do not mutate Kernel truth, Courthouse admissibility, ContextBlocks, runtime output, actions, or memory.

Implementation status: implemented in `services/core/internal/forgek/consensus` and `services/core/internal/forgek/consensus_syscalls.go`, with architecture docs in `docs/architecture/consensus_mesh.md`. No live daemon integration, public API, route, gateway, modelruntime, controllane, tool execution, real model call, or live memory write was added.

Rust validator note: Phase 11E consensus fixtures may be added to Rust validation later after the Go consensus model stabilizes. Phase 11E does not modify `crates/forgek-validate`.

What not to do: treat consensus as truth, auto-admit claims into Courthouse, auto-write memory, execute actions, call model runtimes, run the full mesh for every simple request, or bypass Kernel semantic syscalls.

## Phase 11F - Integration Readiness Contracts

Scope: `SIMULATOR_ONLY / INTEGRATION_PREP_ONLY`.

Goal: define stable integration readiness contracts, live path mappings, adapter boundaries, read-only RAG/retrieval mirror contracts, shadow-mode policy, no-mutation rules, and Phase 12 gates before any live integration work.

Implemented deliverables: integration readiness architecture doc, live path mapping review, adapter contract doc, shadow mode doc, `integrationready` simulator package, IntegrationReadinessReport, LivePathMapping, AdapterContract, ReadOnlyRAGAdapter contract, ShadowModePolicy, readiness matrix, no-live-mutation validation, and forbidden import tests.

Validation criteria: readiness reports are diagnostic only; readiness score is advisory only; all default adapters are read-only; ReadOnlyRAGAdapter cannot execute retrieval, call embeddings, write memory, compile context, admit evidence, or affect output; all live path mappings have live mutation allowed set to `NO`; no live daemon packages are imported by `integrationready`; FORGE-K simulator tests pass.

Implementation status: implemented in `services/core/internal/forgek/integrationready`, `docs/architecture/forge_k_integration_readiness.md`, `docs/reviews/forge_k_live_path_mapping.md`, `docs/architecture/forge_k_adapter_contracts.md`, and `docs/architecture/shadow_mode.md`. No live daemon integration, API route, gateway/modelruntime/controllane behavior change, live retrieval, live RAG, embedding call, tool execution, memory write, or second authority path was added.

What not to do: wire FORGE-K into the live daemon, create live mutation adapters, implement live RAG, call live retrieval or embedding providers, replace live controllane, replace gateway, change API routes, execute tools from FORGE-K, call modelruntime from FORGE-K, let shadow mode affect user-visible output, create a second authority path, or skip Phase 12 design.

## Phase 11G - Shadow Mode Harness Design

Scope: `SIMULATOR_ONLY / SHADOW_DESIGN_ONLY`.

Goal: design the future read-only shadow harness and simulator-only report contracts before any daemon wiring.

Implemented deliverables: shadow harness architecture doc, Phase 12B harness plan, `shadowharness` simulator package, ShadowObservation, ShadowHarnessPolicy, ShadowComparisonReport, RAG/Consensus/Context/Runtime/KV/Lymphatic shadow report contracts, no-effect validator, and forbidden import tests.

Validation criteria: default policy forbids live mutation, tool execution, modelruntime calls, retrieval execution, embedding calls, memory writes, user-visible output, and public API changes; reporting flags can be enabled without side effects; observations/reports serialize deterministically and reject secret-looking metadata; no live daemon packages are imported by `shadowharness`; FORGE-K simulator tests pass.

Implementation status: implemented in `services/core/internal/forgek/shadowharness`, `docs/architecture/shadow_mode_harness.md`, and `docs/reviews/shadow_mode_harness_plan.md`. No live daemon integration, request observation, API route, gateway/modelruntime/controllane behavior change, live retrieval, live RAG, embedding call, tool execution, memory write, user-visible output change, or second authority path was added.

What not to do: implement Phase 12, observe live daemon requests, wire live paths, execute tools, run retrieval, call embeddings, call modelruntime, write memory, alter responses, or change public APIs/routes.

## Phase 12A - Live Integration Design

Scope: `DOCS_ONLY / LIVE_INTEGRATION_DESIGN_ONLY`.

Goal: produce the explicit live integration design required by ADR 0005 before any FORGE-K live authority migration.

Implemented deliverables: live integration design, Phase 12A readiness review, Phase 12B shadow harness specification, Phase 12B adapter interface design, Phase 12B test plan, live path mapping updates, shadow mode handoff updates, rollback/kill-switch strategy, and no-effect test requirements.

Validation criteria: design proves gateway remains tool authority until migration, memory writes remain live authority until migration, retrieval/RAG remains evidence-only until migration, modelruntime remains live driver authority, controllane remains semantic mutation authority, and no shadow output affects user-visible behavior.

Implementation status: docs-only. No live daemon integration, public API route, gateway/modelruntime/controllane behavior change, live observation, live retrieval, live RAG, embedding call, tool execution, memory write, user-visible output change, or second authority path was added.

What not to do: implement Phase 12B, add code-level feature flags, wire adapters, observe live requests, implement live integration, or alter daemon behavior.

## Phase 12B - Read-only Shadow Harness Implementation

Scope: `LIVE_INTEGRATION / READ_ONLY / DISABLED_BY_DEFAULT`.

Goal: implement a read-only shadow harness that observes selected live paths and emits diagnostics without influencing live behavior.

Implemented deliverables: core config flag `FORGE_K_SHADOW_MODE_ENABLED` with default `false`, `services/core/internal/forgekshadow` read-only observer, bounded in-memory diagnostic sink, secret-looking metadata rejection, no-effect policy validation, forbidden import tests, `/health` request-metadata-only observation after response writing, route inventory tests, health response equivalence tests, and sink failure isolation tests.

Validation criteria: live request/response behavior is unchanged; no live mutation is performed by FORGE-K; no tools, retrieval, embeddings, modelruntime calls, memory writes, or user-visible output changes originate from shadow mode.

Implementation status: implemented and tested in `services/core/internal/forgekshadow`, `services/core/internal/config`, and the `/health` handler. The selected live touchpoint is `/health` metadata only. The diagnostic sink is in-memory only and has no public API. Route inventory and `/health` response behavior are unchanged with the flag enabled. No live authority migration was added.

What not to do: migrate authority, commit FORGE-K truth, execute tools, call models, run live RAG, or alter responses.

## Phase 12C - Shadow Diagnostics Review and Hardening

Scope: `LIVE_INTEGRATION / OBSERVABILITY_ONLY / HARDENING_ONLY`.

Goal: review Phase 12B diagnostics, harden redaction/retention/no-effect guarantees, and decide whether any advisory behavior is safe to design later.

Implemented deliverables: Phase 12C diagnostic review report, explicit disabled sink support, expanded metadata safety rules, raw content and oversized metadata rejection, additional no-effect tests, no public diagnostics route test, non-`/health` no-observation test, root FORGE-K forbidden live import guard, and updated status docs.

Validation criteria: shadow mode remains observability-only; live behavior remains unchanged; diagnostics are bounded, redacted, and non-authoritative.

Implementation status: implemented and tested without adding live touchpoints. `/health` remains the only observed route, diagnostics remain in-memory only, and no public diagnostics API exists.

What not to do: migrate authority, alter responses, execute tools, call modelruntime, run retrieval, write memory, add public diagnostics routes, add new live touchpoints, or treat diagnostics as truth.

## Phase 12D - Controlled Shadow Expansion Design

Scope: `DOCS_ONLY / LIVE_INTEGRATION_DESIGN_ONLY`.

Goal: compare possible shadow-mode expansion touchpoints, select exactly one recommended Phase 12E candidate, and record the tests required before implementation.

Implemented deliverables: controlled expansion design, touchpoint selection review, Phase 12E route-envelope test plan, and status/roadmap updates.

Selected Phase 12E candidate: route envelope metadata.

Deferred candidates: chat message submission metadata, existing retrieval-result metadata, and existing gateway trace metadata.

Validation criteria at Phase 12D exit: docs exist; route envelope is selected as the next candidate only; Phase 12E is not started in that pass; existing FORGE-K, shadow, API route inventory, core build, lint, aggregate tests, parity tests, and diff checks pass; no code behavior changes are introduced.

Implementation status: docs-only in Phase 12D. No route-envelope observation, public API, diagnostics route, persistence, feature flag, adapter, gateway/modelruntime/retrieval/memory/controllane behavior change, response change, or authority migration was added during Phase 12D.

What not to do from Phase 12D alone: implement Phase 12E, add route-envelope observer code, observe all routes, capture request or response bodies, add public diagnostics APIs, change route behavior, modify API response shape, call modelruntime, execute tools, query retrieval/search/embeddings, write memory, call controllane mutations, or make FORGE-K live authority.

## Phase 12E - Route Envelope Shadow Metadata

Scope: `LIVE_INTEGRATION / READ_ONLY / DISABLED_BY_DEFAULT`.

Status: implemented and tested.

Goal: implement route envelope metadata shadowing as the next controlled live observation surface while preserving no-effect guarantees.

Implemented deliverables: disabled-by-default route-envelope observation, typed route-envelope model, route class normalization, metadata-only report construction, bounded in-memory sink reuse, redaction and no-effect validation, route inventory stability tests, response equivalence tests, SSE mount/order guard, and forbidden content-capture tests.

Allowed data: HTTP method, matched route template, normalized route class, safe request id when available, bounded timing summary, and no-effect validation result.

Forbidden future data: request bodies, response bodies, raw headers except allowlisted non-secret correlation ids, raw query strings, prompts, model output, tool payloads, retrieval content, embedding vectors, search chunks, memory content, secrets, credentials, cookies, authorization headers, and session values.

Validation criteria: route inventory unchanged; no public diagnostics route; status, headers, bodies, route selection, SSE behavior, and timeout behavior unchanged; shadow failure cannot fail live requests; no modelruntime, retrieval/search/embedding, gateway/tool, memory, controllane, permission, lane, approval, or audit authority behavior changes.

Implementation status: implemented in `services/core/internal/forgekshadow` and the existing API middleware chain. The observer records diagnostics only when `FORGE_K_SHADOW_MODE_ENABLED=true`; the default remains disabled. Reports remain bounded in-memory diagnostics only. No public API, route, persistent report store, gateway/modelruntime/retrieval/memory/controllane behavior change, response change, or authority migration was added.

What not to do: migrate authority, broaden to chat/retrieval/gateway traces, capture content, persist diagnostics, add public APIs, alter responses, execute tools, call modelruntime, run retrieval, write memory, or treat reports as truth.

## Phase 12F - Route Envelope Shadow Hardening

Scope: `LIVE_INTEGRATION / OBSERVABILITY_ONLY / HARDENING_ONLY`.

Status: implemented and tested.

Goal: harden the existing Phase 12E route-envelope diagnostics before any wider observation is considered.

Implemented deliverables: diagnostic review record, matched route-pattern safety, route class normalization, query/header/body/secret leakage prevention, reserved metadata key rejection, deterministic metadata value enforcement, bounded in-memory retention, sink failure isolation, disabled/enabled response equivalence tests, SSE mount/order and timeout stability tests, no public diagnostics route checks, and forbidden live authority import checks.

Allowed data remains: HTTP method, matched route template, normalized route class, safe request id when available, bounded timing summary, and no-effect validation result.

Forbidden data remains: request bodies, response bodies, raw headers except allowlisted non-secret correlation ids, raw query strings, prompts, model output, tool payloads, retrieval content, embedding vectors, search chunks, memory content, secrets, credentials, cookies, authorization headers, and session values.

Validation criteria: route-envelope observation remains disabled by default, bounded, metadata-only, non-authoritative, and proven to have no effect on live responses or live authority paths. Route inventory, response status/body/header equivalence, SSE mount/order, timeout middleware behavior, sink failure isolation, redaction failure behavior, and forbidden imports must pass.

What not to do: add new touchpoints, observe chat content, observe retrieval content, observe gateway payloads, persist reports, add public diagnostics routes, execute tools, call modelruntime, run retrieval/search/embeddings, write memory, mutate controllane, or start authority migration.

## Phase 12G - Chat Metadata Expansion Design

Scope: `DOCS_ONLY / LIVE_INTEGRATION_DESIGN_ONLY`.

Status: implemented.

Goal: design a possible future chat metadata shadow expansion without implementing chat metadata observation.

Deliverables: chat metadata expansion design, chat metadata risk review, Phase 12H test plan, status updates, and roadmap updates.

Allowed future metadata: route class, matched chat route pattern, safe/stable thread id, safe/stable message id, workspace id, request/correlation id, role class, bounded count summary, timing/status metadata, safe model/provider id when already exposed, diagnostic markers, and warnings.

Forbidden future metadata: message content, prompts, completions, assistant response text, system prompts, request bodies, response bodies, tool payloads, tool outputs, retrieval content, search chunks, embedding vectors, memory content, auth headers, cookies, tokens, API keys, secrets, and large raw content blobs.

Validation criteria: docs exist, future tests are explicit, no live code changes, no new touchpoints, no chat observation, no public diagnostics APIs, and existing FORGE-K/shadow/API route inventory/build/lint/test/parity checks pass.

What not to do: implement Phase 12H, add chat metadata observer code, observe chat routes, capture message content, capture prompts, capture completions, capture request/response bodies, observe tool payloads, observe retrieval content, add public diagnostics APIs, change route behavior, modify API response shape, call modelruntime, execute tools, query retrieval/search/embeddings, write memory, call controllane mutations, or make FORGE-K live authority.

## Phase 12H - Chat Metadata Shadow Implementation

Scope: `LIVE_INTEGRATION / READ_ONLY / DISABLED_BY_DEFAULT`.

Status: implemented and tested.

Goal: implement bounded chat metadata shadow diagnostics behind both `FORGE_K_SHADOW_MODE_ENABLED=false` and `FORGE_K_SHADOW_CHAT_METADATA_ENABLED=false` defaults.

Implementation status: implemented in `services/core/internal/forgekshadow` and the existing chat message POST handler. The observer records metadata only after live handler ownership is established and only when both flags are enabled. Reports remain bounded in-memory diagnostics only. No public API, route, response, gateway/modelruntime/retrieval/search/embeddings/memory/controllane behavior change, or authority migration was added.

Validation criteria: `docs/testing/phase_12h_chat_metadata_shadow_tests.md` records the test coverage and required commands.

What not to do: capture content, prompts, completions, request bodies, response bodies, tool payloads, retrieval content, memory content, secrets, or user-visible output.

## Phase 12I - Chat Metadata Shadow Hardening

Scope: `LIVE_INTEGRATION / OBSERVABILITY_ONLY / HARDENING_ONLY`.

Status: implemented and tested.

Goal: review and harden the Phase 12H chat metadata implementation before broader metadata surfaces are considered.

Implementation status: completed as hardening only. Phase 12I strengthens dual-flag tests, bounded enum/ref normalization, forbidden content-key rejection, deterministic serialization coverage, invalid-body/header/query no-capture tests, assistant-stream safety tests, sink behavior tests, no-effect validation coverage, and documentation. It adds no new live touchpoints and does not expand beyond chat metadata.

What not to do: add new touchpoints, broaden chat capture, persist diagnostics, add public diagnostics APIs, execute tools, call modelruntime, run retrieval/search/embeddings, write memory, mutate controllane, or start authority migration.

## Phase 12J - Retrieval Metadata Expansion Design

Scope: `DOCS_ONLY / LIVE_INTEGRATION_DESIGN_ONLY`.

Status: implemented.

Goal: design a possible future retrieval metadata shadow expansion without implementing retrieval metadata observation.

Deliverables: retrieval metadata expansion design, retrieval metadata risk review, future Phase 12K test plan, status updates, and roadmap updates.

Allowed future metadata: retrieval run/result refs, workspace/request/correlation refs, source type/ref, existing source fingerprint, result count, selected count, bounded score summaries, ranking position, retrieval strategy, index name/type, safe embedding model id, freshness/staleness flags, timing/status metadata, diagnostic markers, and bounded warnings.

Forbidden future metadata: source text, chunk text, document content, file content, raw user query, search snippets, embeddings, vectors, RAG output, prompts, model outputs, memory content, request/response bodies, auth headers, cookies, tokens, API keys, secrets, and large raw content blobs.

What not to do from Phase 12J alone: implement Phase 12K, add retrieval metadata observer code, execute retrieval/search/embedding calls, implement live RAG, capture source/chunk/query/vector content, add public diagnostics APIs, change route behavior, call modelruntime, execute tools, write memory, mutate controllane, or start authority migration.

## Phase 12K - Retrieval Metadata Shadow Implementation

Scope: `LIVE_INTEGRATION / READ_ONLY / DISABLED_BY_DEFAULT`.

Status: implemented and tested as part of the combined Phase 12K-L pass.

Goal: implement bounded retrieval metadata diagnostics after live retrieval run creation, disabled by default and gated by both the global shadow flag and `FORGE_K_SHADOW_RETRIEVAL_METADATA_ENABLED`.

Deliverables: retrieval metadata config flag, typed retrieval metadata model, retrieval metadata observer, safe field normalization/redaction, post-run API observer hook, bounded in-memory diagnostics, route/API stability tests, invalid-body/header/query no-capture tests, sink isolation tests, and hardening review.

Validation criteria: all tests in `docs/testing/phase_12k_retrieval_metadata_shadow_tests.md` pass before completion.

What not to do: capture source text, chunks, embeddings/vectors, raw queries, prompts, model outputs, request/response bodies, memory content, secrets, or user-visible output.

## Phase 12L - Retrieval Metadata Shadow Hardening

Scope: `LIVE_INTEGRATION / OBSERVABILITY_ONLY / HARDENING_ONLY`.

Status: implemented and tested in the combined Phase 12K-L hardening pass.

Goal: harden retrieval metadata diagnostics before broader metadata surfaces are considered.

Hardening summary: Phase 12L adds no touchpoints beyond Phase 12K, broadens no capture scope, persists no reports, and keeps retrieval metadata diagnostic-only, metadata-only, bounded, and disabled by default.

What not to do: add new touchpoints, broaden retrieval capture, persist diagnostics, add public diagnostics APIs, execute tools, call modelruntime, run retrieval/search/embeddings, write memory, mutate controllane, or start authority migration.

## Phase 12M-Q - Shadow Advisory Pipeline

Scope: `LIVE_INTEGRATION / READ_ONLY / DISABLED_BY_DEFAULT / ADVISORY_DIAGNOSTIC_ONLY`.

Status: implemented and tested.

Goal: attach internal advisory diagnostics to existing bounded shadow reports without making FORGE-K live authority or changing user-visible behavior.

Deliverables: advisory config flag, `ShadowAdvisoryReport` model, metadata-only evidence summary, metadata-only consensus advisory, context compiler advisory summary, deterministic advisory hashing, no-effect tests, route/API stability tests, and status/review docs.

Allowed data: existing shadow report refs, route/chat/retrieval metadata observation refs, safe thread/message/retrieval/source refs, counts, warnings, metadata-only risk flags, and deterministic hashes over safe advisory inputs.

Forbidden data: request bodies, response bodies, prompts, completions, assistant response text, source text, chunk text, document content, raw user queries, search snippets, embeddings, vectors, RAG output, tool payloads/outputs, modelruntime payloads/outputs, memory content, auth headers, cookies, tokens, API keys, secrets, and user-visible output.

Validation criteria: advisory reports require both global shadow mode and the advisory flag; advisory mode does not force-enable chat/retrieval observers; reports remain in-memory only; no public diagnostics route exists; route inventory and `/health` response behavior remain unchanged; unsafe metadata and side-effect policies are rejected; full FORGE-K, shadow, API, build, lint, test, parity, and diff checks pass.

What not to do: add public diagnostics APIs, persist advisory reports, compile live context, run live Consensus Mesh authority, accept factual claims as truth, alter response composition, execute tools, call modelruntime, run retrieval/search/embeddings/live RAG, write memory, mutate controllane, change routes/public APIs, or start authority migration.

## Phase 13A - Storage Backend Foundation

Scope: `LIVE_INFRA / STORAGE_FOUNDATION / DEFAULT_SQLITE`.

Status: implemented and tested.

Goal: add explicit backend selection/config, backend capability contracts, Postgres connector scaffolding, migration runner scaffolding, migration design docs, and storage parity test planning while preserving SQLite as the live default.

Implemented deliverables: `FORGE_STORE_BACKEND=sqlite|postgres`, default SQLite config behavior, Postgres DSN config parsing, Redis/Qdrant endpoint parsing, storage backend capability model, Redis/Qdrant non-canonical capability contracts, Postgres connector validation, Postgres migration version table scaffold, storage migration design doc, storage backend parity test doc, persistence inventory updates, Docker config docs, and tests.

Validation criteria: SQLite remains the default live backend; invalid backend config is rejected by the backend config boundary; Postgres requires a DSN in backend validation; Redis/Qdrant env vars do not switch authority; migration runner does not run Postgres migrations for SQLite; existing core tests pass.

What not to do: migrate live memory to Postgres, make Postgres default, dual-write live data, wire Redis into queues/caches, wire Qdrant into retrieval, store vectors as truth, make retrieval scores admissible evidence, change routes/public APIs, alter gateway/modelruntime/retrieval behavior, or make FORGE-K live authority.

## Phase 13B - Postgres Schema Foundation

Scope: `LIVE_INFRA / STORAGE_FOUNDATION / DEFAULT_SQLITE`.

Status: implemented and tested as part of Phase 13B-C.

Goal: add initial Postgres schema migrations for a small, low-risk table group without switching live reads or writes.

Implemented deliverables: migration SQL files under `services/core/migrations/postgres`, migration version records with checksums, `storage_backend_metadata`, `storage_migration_audit`, and disabled shadow diagnostic report/event/redaction schema.

Validation criteria: migrations are idempotent, transaction-wrapped, versioned, and tested without requiring Docker for default tests.

## Phase 13C - SQLite/Postgres Parity Tests

Scope: `LIVE_INFRA / TESTING_ONLY / DEFAULT_SQLITE`.

Status: implemented and tested as part of Phase 13B-C.

Goal: establish table and repository parity tests before any dual-write or read-switch phase.

Validation criteria: foundation schema, deterministic migration ordering, applied-version skips, migration failure reporting, JSONB/timestamp shape, migration registry/file parity, SQLite skip behavior, and optional Postgres integration pass. Repository CRUD/list parity begins in a future adapter phase.

## Phase 13D - Diagnostic Store Persistence

Scope: `LIVE_INFRA / DIAGNOSTIC_STORAGE / DEFAULT_SQLITE / DISABLED_BY_DEFAULT`.

Status: implemented and tested as part of Phase 13D-E.

Goal: persist approved non-authoritative diagnostics after parity gates, without creating public diagnostics APIs or authority.

Implemented deliverables: disabled-by-default `FORGE_SHADOW_DIAGNOSTIC_PERSISTENCE_ENABLED`, retention and payload-limit config, Postgres diagnostic repository, persistence sink wrapper that preserves the in-memory sink, safe diagnostic summary builder, retention expiry, no-effect verification, schema version, unsafe metadata rejection, payload-size rejection, repository-failure isolation, and optional Postgres integration tests gated by `FORGE_POSTGRES_TEST_DSN`.

Validation criteria: disabled mode does not write repository rows; enabled mode requires explicit Postgres DSN; persisted rows are non-authoritative summaries only; raw prompts/completions/message bodies/source chunks/raw queries/vectors/embeddings/tool payloads/memory content/secrets are rejected or omitted; SQLite remains default.

## Phase 13E - Retrieval Metadata Postgres Adapter

Scope: `LIVE_INFRA / DIAGNOSTIC_STORAGE / DEFAULT_SQLITE / DISABLED_BY_DEFAULT`.

Status: implemented and tested as part of Phase 13D-E.

Goal: add a relational-safe retrieval metadata adapter after parity tests exist.

Implemented deliverables: retrieval metadata relational DTO, deterministic canonical serialization, safe mapping for run/result/source refs, counts, classes, score summaries, strategy, index type, freshness, and duration.

What not to do: change live retrieval behavior, execute retrieval/search/embedding calls, persist source/chunk text, persist raw queries, persist vectors/embeddings, wire Qdrant, implement live RAG, or make retrieval scores admissible evidence.

## Phase 13F - Qdrant Vector Adapter Design

Scope: `LIVE_INFRA / VECTOR_SHADOW / DISABLED_BY_DEFAULT / NON_AUTHORITATIVE`.

Status: implemented and tested as part of Phase 13F-G.

Goal: design and scaffold the Qdrant adapter boundary, rebuild strategy, provenance model, and non-authority rules.

Implemented deliverables: generic `VectorStore` interface, Qdrant HTTP adapter, safe payload schema, deterministic point IDs, config flags, env docs, payload safety tests, and optional Qdrant integration test gating.

## Phase 13G - Qdrant Shadow Vector Index

Scope: `LIVE_INFRA / VECTOR_SHADOW / DISABLED_BY_DEFAULT / NON_AUTHORITATIVE`.

Status: implemented and tested as part of Phase 13F-G.

Goal: build a rebuildable, non-authoritative Qdrant shadow index after adapter design and tests.

Implemented deliverables: disabled-by-default `ShadowIndexService`, precomputed-vector-only upsert path, safe ref/provenance payload validation, vector dimension checks, no retrieval/embedding execution contract, and no live retrieval wiring.

What not to do: switch live retrieval to Qdrant, generate embeddings from the vector adapter, store source/chunk content in Qdrant, or treat vector hits as truth or evidence.

## Phase 13H - Redis Queue/Cache Boundary

Scope: `LIVE_INFRA / EPHEMERAL_COORDINATION / DISABLED_BY_DEFAULT / NON_CANONICAL`.

Status: implemented and tested.

Goal: define and test Redis-backed queue/cache/lock/progress primitives where loss is recoverable from durable storage.

Implemented deliverables: `FORGE_REDIS_ENABLED=false`, `FORGE_REDIS_KEY_PREFIX`, `FORGE_REDIS_TIMEOUT_MS`, `services/core/internal/ephemeral` role contracts, safe key policy, TTL requirements, fake in-memory adapter, stdlib Redis client scaffold, optional Redis integration test gating, config tests, capability tests, adapter tests, and docs.

Validation criteria: Redis remains disabled by default; enabled Redis requires addr configuration; Redis roles are ephemeral only; forbidden canonical/durable/admission/provenance roles are rejected; unsafe keys and missing TTLs are rejected; fake adapter behavior passes; optional Redis integration is gated by `FORGE_REDIS_TEST_ADDR`; no live job, gateway, modelruntime, retrieval, memory, public API, route, or storage backend behavior changes.

What not to do: switch live jobs to Redis, make Redis required, store canonical truth, durable memory, admissibility, provenance, settings, audit authority, raw prompts/content/secrets, or sole job records in Redis.

## Phase 13I - Store Cutover Readiness Review

Scope: `DOCS_ONLY / READINESS_REVIEW`.

Status: implemented.

Goal: decide whether any backend is ready for dual-write or read-switch phases based on parity evidence, rollback plans, and operator runbooks.

Implemented deliverables: Phase 13I readiness review, Postgres/Qdrant/Redis readiness matrix, cutover blockers, required gates, recommended next storage phase, and explicit no-cutover decision.

Validation criteria: review records that SQLite remains the live default; Postgres is not canonical-ready; Qdrant is not live-retrieval-ready; Redis is not live-queue-ready; no live behavior changes are introduced.

What not to do: make Postgres default, dual-write canonical live data, switch reads, wire Qdrant into live retrieval, wire Redis into live jobs/cache, change routes/public APIs, change gateway/modelruntime/retrieval behavior, or use storage infrastructure as FORGE-K authority migration.

## Phase 14A - FORGE-K Operational Cutover Design

Scope: `DOCS_ONLY / LIVE_AUTHORITY_MIGRATION_DESIGN_ONLY`.

Status: implemented.

Goal: define the staged path for making FORGE-K operational through narrow live authority seams without importing simulator services as live authority or creating a second authority path.

Implemented deliverables: operational cutover design, current authority split, operational cutover rule, recommended first operational surface, staged cutover model, required tests, rollback model, go/no-go gates, and Phase 14B recommendation.

Validation criteria: design preserves existing live owners, recommends Control Lane semantic validation as the first operational surface, and forbids full Kernel replacement, route/API changes, gateway/modelruntime/retrieval/memory behavior changes, live KV reuse, Qdrant live retrieval authority, Redis canonical state, and simulator-service live imports.

What not to do: use Phase 14A alone to wire FORGE-K Kernel into live daemon, make live Context Compiler prompt authority, enable live KV reuse, migrate memory authority, alter user-visible output, or bypass gateway/permissions/lanes/audit/controllane.

## Phase 14B - Ref Shape Validation

Scope: `PARTIAL LIVE VALIDATION / CONTROL_LANE / NO_AUTHORITY_REPLACEMENT`.

Status: implemented and tested.

Goal: extract one deterministic validation contract into a shared pure package and invoke it from the live Control Lane without replacing live authority.

Implemented deliverables: `services/core/internal/refvalidation`, `VALIDATE_REF_SHAPE`, capability `ref.shape.validate`, ref-shape enforcement decision, structured audit fields, no-mutation state summary, dry-run summary preservation, and tests.

Validation criteria: refs are normalized and deduplicated deterministically; invalid ref types and unsafe ref ids fail closed; propose-only sources without capability are denied; validation does not commit semantic objects, write memory, call modelruntime, execute retrieval, change routes/public APIs, import FORGE-K simulator services as live authority, or create a second authority path.

What not to do: use ref validation as evidence admission, look up object truth, compile context, execute retrieval/search/embeddings, write memory, call modelruntime, execute tools, change routes/public APIs, or route live state mutation through FORGE-K simulator services.

## Phase 14C - Control Lane Validation Expansion

Scope: `PARTIAL LIVE VALIDATION / CONTROL_LANE / SHADOW_COMPARE / NO_AUTHORITY_REPLACEMENT`.

Status: implemented and tested.

Goal: add a diagnostic ref-shape shadow comparison and one more deterministic semantic validation contract without replacing live authority.

Implemented deliverables: `refvalidation.CompareRefShapes`, `services/core/internal/semanticvalidation`, `COMPARE_REF_SHAPE`, `VALIDATE_SEMANTIC_OPERATION`, capabilities `ref.shape.compare` and `semantic.operation.validate`, structured audit fields, no-mutation state summaries, and tests.

Validation criteria: ref comparison reports match/drift deterministically; invalid observed refs fail closed; semantic operation validation normalizes refs and rejects forbidden authority claims; propose-only sources without capability are denied; neither action commits semantic objects, writes memory, admits evidence, compiles context, calls modelruntime, executes retrieval/search/embeddings/tools, changes routes/public APIs, imports FORGE-K simulator services as live authority, or creates a second authority path.

What not to do: use comparison drift as a user-visible decision, treat semantic operation validation as operation execution, admit/reject evidence, compile context, write memory, call modelruntime, execute tools, execute retrieval/search/embeddings, or route live state mutation through FORGE-K simulator services.

## Phase 14D - Control Lane Validation Shadow Reporting

Scope: `LIVE_INTEGRATION / READ_ONLY / DISABLED_BY_DEFAULT / CONTROL_LANE_VALIDATION_DIAGNOSTICS`.

Status: implemented and tested.

Goal: add an internal shadow diagnostic report shape for Control Lane validation summaries without changing Control Lane behavior or making FORGE-K simulator services live authority.

Implemented deliverables: `ControlLaneValidationInput`, `ControlLaneValidationObservation`, `Observer.ObserveControlLaneValidation`, `Observer.ObserveControlLaneValidationBestEffort`, config flag `FORGE_K_SHADOW_CONTROL_LANE_VALIDATION_ENABLED`, diagnostic persistence mapping for `control_lane_validation`, and tests.

Validation criteria: reports require both global shadow mode and the control-lane validation flag; stored data is bounded scalar metadata only; forbidden effect claims and unsafe metadata fail closed; no-effect policy remains enforced; no public API, route behavior, user-visible output, Control Lane mutation, memory write, evidence admission, context compilation, retrieval/search/embedding execution, modelruntime call, tool execution, simulator-service live import, or authority migration is introduced.

What not to do: use shadow reports to affect validation decisions, expose reports as public diagnostics, persist raw refs/content, call Control Lane from FORGE-K, mutate live state, or broaden this observer beyond internal diagnostic reporting without a separate phase and tests.

## Future - FORGE Daemon

Scope: `LIVE_INTEGRATION`.

Goal: expose FORGE-K as a governed local daemon after Phase 12A-12C have established design, read-only shadow evidence, and limited migration proof.

Deliverables: daemon process, local API, policy loading, journal persistence, runtime-driver isolation.

Validation criteria: daemon enforces semantic syscalls; audit and journal are durable; drivers remain outside authority.

What not to do: expose uncontrolled tool execution or remote authority by default.

## Future Research - FORGE-1 Simulator

Scope: `RESEARCH_ONLY`.

Goal: simulate candidate FORGE-1 instruction families and hardware blocks.

Deliverables: simulator contracts, instruction traces, workload benchmarks, correctness tests.

Validation criteria: simulator behavior matches documented kernel semantics; performance claims cite evidence.

What not to do: claim hardware readiness or replace the userspace kernel path.

## Future Research - FORGE-1 Prototype Research

Scope: `RESEARCH_ONLY`.

Goal: evaluate hardware/software co-design feasibility for governed execution acceleration.

Deliverables: research notes, prototype constraints, benchmark methodology, risk register.

Validation criteria: research remains grounded in measured workloads and simulator evidence.

What not to do: make FORGE-1 a production dependency or imply GPUs are obsolete.
