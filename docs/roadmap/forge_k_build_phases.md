# FORGE-K Build Phases

Status: Phase 11A Rust Kernel Core research/planning complete; Phase 11B Rust deterministic validation crate and Phase 11C Go/Rust test corpus alignment are implemented as `RESEARCH_ONLY / SIMULATOR_ONLY`. Phase 11D Rust Validation CI and Tooling Integration is implemented as `RESEARCH_ONLY / SIMULATOR_ONLY / TOOLING_ONLY`. Phase 11E Consensus Mesh is implemented as `SIMULATOR_ONLY / GOVERNANCE_LAYER_ONLY`. Phase 11F Integration Readiness Contracts is implemented as `SIMULATOR_ONLY / INTEGRATION_PREP_ONLY`. Phase 11G Shadow Mode Harness Design is implemented as `SIMULATOR_ONLY / SHADOW_DESIGN_ONLY`. Phase 12A Live Integration Design is implemented as `DOCS_ONLY / LIVE_INTEGRATION_DESIGN_ONLY`. Phase 12B Read-only Shadow Harness Implementation is implemented as `LIVE_INTEGRATION / READ_ONLY / DISABLED_BY_DEFAULT`. Phase 12C Shadow Diagnostics Review and Hardening is implemented as `LIVE_INTEGRATION / OBSERVABILITY_ONLY / HARDENING_ONLY`. Phase 12D Controlled Shadow Expansion Design is implemented as `DOCS_ONLY / LIVE_INTEGRATION_DESIGN_ONLY`. Phase 12E Route Envelope Shadow Metadata is implemented as `LIVE_INTEGRATION / READ_ONLY / DISABLED_BY_DEFAULT`. Phase 12F Route Envelope Shadow Hardening is implemented as `LIVE_INTEGRATION / OBSERVABILITY_ONLY / HARDENING_ONLY`. Phase 12G Chat Metadata Expansion Design is implemented as `DOCS_ONLY / LIVE_INTEGRATION_DESIGN_ONLY`.

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

Deliverables: chat metadata expansion design, chat metadata risk review, future Phase 12H test plan, status updates, and roadmap updates.

Allowed future metadata: route class, matched chat route pattern, safe/stable thread id, safe/stable message id, workspace id, request/correlation id, role class, bounded count summary, timing/status metadata, safe model/provider id when already exposed, diagnostic markers, and warnings.

Forbidden future metadata: message content, prompts, completions, assistant response text, system prompts, request bodies, response bodies, tool payloads, tool outputs, retrieval content, search chunks, embedding vectors, memory content, auth headers, cookies, tokens, API keys, secrets, and large raw content blobs.

Validation criteria: docs exist, future tests are explicit, no live code changes, no new touchpoints, no chat observation, no public diagnostics APIs, and existing FORGE-K/shadow/API route inventory/build/lint/test/parity checks pass.

What not to do: implement Phase 12H, add chat metadata observer code, observe chat routes, capture message content, capture prompts, capture completions, capture request/response bodies, observe tool payloads, observe retrieval content, add public diagnostics APIs, change route behavior, modify API response shape, call modelruntime, execute tools, query retrieval/search/embeddings, write memory, call controllane mutations, or make FORGE-K live authority.

## Phase 12H - Chat Metadata Shadow Implementation

Scope: `LIVE_INTEGRATION / READ_ONLY / DISABLED_BY_DEFAULT`.

Status: not started.

Goal: implement bounded chat metadata shadow diagnostics only if separately approved.

Validation criteria: all tests in `docs/testing/phase_12h_chat_metadata_shadow_tests.md` pass before completion.

What not to do: capture content, prompts, completions, request bodies, response bodies, tool payloads, retrieval content, memory content, secrets, or user-visible output.

## Phase 12I - Chat Metadata Shadow Hardening

Scope: `LIVE_INTEGRATION / OBSERVABILITY_ONLY / HARDENING_ONLY`.

Status: not started.

Goal: review and harden any future Phase 12H implementation before broader metadata surfaces are considered.

What not to do: add new touchpoints, broaden chat capture, persist diagnostics, add public diagnostics APIs, execute tools, call modelruntime, run retrieval/search/embeddings, write memory, mutate controllane, or start authority migration.

## Phase 12 - FORGE Daemon

Scope: `LIVE_INTEGRATION`.

Goal: expose FORGE-K as a governed local daemon after Phase 12A-12C have established design, read-only shadow evidence, and limited migration proof.

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
