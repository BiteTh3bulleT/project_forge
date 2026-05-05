# FORGE-K Shadow Mode Harness

Status: Phase 11G simulator-only shadow harness design plus Phase 12A live integration design handoff. Phase 11G scope is `SIMULATOR_ONLY / SHADOW_DESIGN_ONLY`; Phase 12A scope is `DOCS_ONLY / LIVE_INTEGRATION_DESIGN_ONLY`.

Phase 11G does not implement live shadow mode and does not authorize Phase 12 live authority migration. Phase 12B later implemented one disabled-by-default `/health` metadata observer, and Phase 12C hardened it without adding touchpoints.

## Executive Summary

The shadow mode harness is the future read-only observation and diagnostic layer for comparing existing live daemon behavior against FORGE-K simulator contracts. In Phase 11G, the harness is a design and pure contract model only. It does not observe live requests, wire adapters, call model runtimes, execute tools, run retrieval, call embedding providers, write memory, alter routes, change APIs, or affect user-visible output.

Shadow reports are diagnostics only. They cannot authorize canonical mutation, live integration, tool execution, memory writes, retrieval, modelruntime calls, or response changes.

## Purpose

Phase 11F defined integration readiness contracts, live path mappings, read-only adapter contracts, read-only RAG/retrieval mirror boundaries, and shadow-mode policy. Phase 11G defines the simulator-only harness shape. Phase 12B implements the first narrow live-adjacent observer against that shape, and Phase 12C hardens the diagnostics boundary.

The design establishes:

- shadow observation shape
- request mirroring rules
- comparison report contracts
- consensus, context, RAG, runtime, KV, and lymphatic diagnostic report shapes
- no-effect validation
- Phase 12B implementation gates

## Current Phase 11F Boundary

Phase 11F remains the integration-readiness boundary. It defines read-only adapter contracts and no-mutation policy in `services/core/internal/forgek/integrationready`. It does not implement the Phase 11G harness.

Phase 11G adds `services/core/internal/forgek/shadowharness` as a pure simulator contract package. It has no syscalls, no Kernel ownership, no live daemon imports, no public API, no routes, and no live behavior.

## Harness Responsibilities

A future live shadow harness may:

- observe already-executing live request metadata
- mirror stable refs and provenance into diagnostic records
- consume read-only adapter outputs
- produce comparison reports
- produce consensus, context, RAG, runtime, KV, and lymphatic shadow reports
- flag divergence and risk for operator review

It must remain read-only until a later `LIVE_INTEGRATION` phase explicitly authorizes more.

## Non-Goals

- No live daemon wiring.
- No live request observation in Phase 11G.
- No public API or route changes.
- No live modelruntime calls.
- No tool execution.
- No live retrieval or embedding calls.
- No live RAG.
- No memory writes.
- No user-visible response changes.
- No second authority path.
- No Phase 12 live integration.

## Shadow Request Lifecycle

The future lifecycle is:

1. Live request executes through existing live paths.
2. A read-only shadow observer receives request metadata and stable refs.
3. Read-only adapters normalize existing records into evidence refs and diagnostics.
4. The harness creates a `ShadowObservation`.
5. Simulator-only diagnostic comparers produce subreports.
6. The harness creates a `ShadowComparisonReport`.
7. Operators inspect diagnostics without affecting live response behavior.

No step may take ownership of live routing, live response composition, live state mutation, approvals, tool execution, model calls, retrieval, embeddings, memory writes, audit writes, or public APIs.

## Shadow Observation Model

`ShadowObservation` records:

- observation_id
- workspace_id
- request_id
- observed_at
- live_path
- request_summary
- input_refs
- evidence_refs
- retrieval_refs
- context_refs
- consensus_refs
- runtime_refs
- kv_refs
- risk_flags
- metadata

Rules:

- diagnostic only
- no live mutation
- deterministic serialization
- no raw secret fields
- no user-visible output authority
- refs only; no large raw content blobs

## Read-Only Adapter Handoff

Phase 11G consumes the Phase 11F adapter vocabulary:

- ReadOnlyRAGAdapter for retrieval/search/embedding/VSA metadata
- LiveContextCompileMirrorAdapter for context metadata
- LiveConsensusShadowAdapter for consensus diagnostics
- LiveModelRuntimeTraceAdapter for runtime traces
- LiveGatewayTraceAdapter for gateway traces
- LiveAuditMirrorAdapter for provenance traces

Adapters may observe, normalize, mirror, and report. They may not mutate, execute, commit, approve, route, publish, call model runtimes, perform retrieval, call embedding providers, write memory, compile live context, or affect user-visible output.

## Consensus Shadow Report

`ConsensusShadowReport` records:

- report_id
- request_id
- consensus_report_ref
- accepted_claim_count
- rejected_claim_count
- uncertain_claim_count
- conflicted_claim_count
- unsupported_fact_count
- composer_guard_passed
- warnings
- metadata

Accepted consensus claims are diagnostics only. They do not become truth and do not emit final answers.

## Context Shadow Report

`ContextShadowReport` records:

- report_id
- request_id
- context_bundle_ref
- block_count
- stable_prefix_hash
- volatile_suffix_hash
- cache_eligible_block_count
- rejected_evidence_leak_detected
- warnings
- metadata

The report may compare simulator ContextBundle refs if provided. It does not compile live context and does not affect live `COMPILE_CONTEXT`.

## RAG / Retrieval Shadow Report

`RAGShadowReport` records:

- report_id
- request_id
- retrieval_refs
- evidence_refs
- source_refs
- normalized_evidence_count
- tier1_count
- tier2_count
- tier3_count
- stale_refs
- unsupported_refs
- warnings
- metadata

The report observes existing retrieval refs only. It does not execute retrieval, call embeddings, compile context, admit evidence, write memory, alter responses, or promote retrieved content to truth.

Retrieval does not make content truth. Embeddings, vector indexes, VSA records, and retrieval results are evidence/routing signals only.

## Runtime Shadow Report

`RuntimeShadowReport` records:

- report_id
- request_id
- runtime_result_refs
- driver_refs
- model_identity_refs
- proposal_only_verified
- warnings
- metadata

The report is diagnostic only. It does not call modelruntime. Runtime outputs remain proposals.

## KV Shadow Report

`KVShadowReport` records:

- report_id
- request_id
- kv_manifest_refs
- cache_hit_count
- cache_miss_count
- invalidated_count
- evicted_count
- acceleration_not_memory_verified
- warnings
- metadata

The report is diagnostic only. It performs no live KV reuse and no runtime cache mutation. KV remains acceleration metadata, not memory.

## Lymphatic Shadow Report

`LymphaticShadowReport` records:

- report_id
- request_id
- maintenance_report_refs
- cleanup_proposal_count
- unsafe_proposal_count
- no_silent_mutation_verified
- warnings
- metadata

Reports and cleanup proposals do not execute. They remain diagnostics/proposals only.

## No-Effect Guarantees

The no-effect validator confirms:

- no live mutation
- no tool execution
- no modelruntime calls
- no retrieval execution
- no embedding calls
- no memory writes
- no user-visible output
- no public API changes

No-effect verification is required before any shadow report can be considered valid.

## Failure Modes

Hard failures:

- side-effectful policy flag enabled
- secret-looking metadata included
- live daemon package imported into the harness package
- missing workspace/request/report ids
- missing no-effect verification
- adapter attempts mutation, execution, retrieval, embeddings, memory writes, modelruntime calls, route changes, or user-visible output

Diagnostic failures:

- rejected evidence leak detected
- unsupported facts surfaced
- composer guard fails
- stale retrieval refs
- runtime proposal-only verification missing
- KV acceleration-not-memory verification missing
- lymphatic no-silent-mutation verification missing

## Phase 12B Implementation Gates

Before implementing a read-only live shadow harness:

- Phase 12A live integration design is complete
- route inventory tests pass
- adapter contracts are mapped to exact live owners
- no-mutation policy tests pass
- rollback/disable strategy exists
- diagnostic storage and retention are defined
- no user-visible output impact tests exist
- no live retrieval/embedding execution tests exist
- no modelruntime call tests exist
- no tool execution tests exist
- memory write denial tests exist
- gateway remains live tool authority
- memory writes remain live authority until migration
- retrieval/RAG remains evidence-only until migration

## Phase 12A Design Handoff

Phase 12A maps this simulator harness model to a future live shadow harness:

- `ShadowObservation` maps to a read-only live request observation built from route/request metadata and stable refs.
- `ShadowComparisonReport` maps to a diagnostic report generated after live owner paths retain control.
- RAG, context, runtime, KV, consensus, and lymphatic subreports remain diagnostic only.
- The no-effect validator maps to live tests proving no response changes, no route changes, no mutation, and no forbidden execution.

The following remain simulator-only or design-only until Phase 12B is separately approved:

- live request observation
- live adapter wiring
- diagnostic sink implementation
- code-level feature flag
- operator-visible shadow status

The following cannot be wired in Phase 12B:

- live FORGE-K response composition
- live RAG or retrieval execution
- embedding provider calls
- modelruntime calls
- tool execution
- memory writes
- controllane semantic mutation
- public API or route changes unless separately approved

## What Not To Do

- Do not implement Phase 12.
- Do not observe live daemon requests in Phase 11G.
- Do not wire adapters to live systems.
- Do not implement live RAG.
- Do not call live retrieval or embedding providers.
- Do not modify live daemon behavior.
- Do not mutate live memory/state.
- Do not execute tools.
- Do not call real model runtimes.
- Do not modify public APIs.
- Do not add routes.
- Do not create a second Kernel.
