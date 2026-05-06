# FORGE-K Integration Readiness

Status: Phase 11F implemented as `SIMULATOR_ONLY / INTEGRATION_PREP_ONLY`.

Phase 11F does not authorize live authority migration.

## Executive Summary

FORGE-K Phase 1-11E is implemented and tested in the simulator under `services/core/internal/forgek`. The live daemon still uses the existing AI-OS, gateway, permissions, lanes, audit, modelruntime, API, retrieval, embeddings, memory, and controllane paths. ADR 0005 remains the authority boundary: FORGE-K is target architecture, not live daemon authority.

Integration readiness means the simulator has documented contracts, live path mappings, adapter boundaries, shadow-mode rules, and tests that make a later live integration plan possible. It is not live integration.

## Current Simulator / Live Split

| Area | Current authority |
|---|---|
| FORGE-K simulator | Kernel, Neuron Fabric, Courthouse, Memory Palace, Semantic Algebra, Snapshots, Context Compiler, KV System, Runtime Boundary, Lymphatic Lane, Consensus Mesh, Rust fixture validation. |
| Live daemon | AI-OS/control lane, gateway, permissions, lanes, audit, modelruntime, API routing, memory, retrieval, search, embeddings, dream/autonomy, backup/release, settings/config. |

The simulator may define target contracts. It must not route live requests, execute live tools, call live model runtimes, perform live retrieval, call embedding providers, write live memory, or alter user-visible output.

## Integration Readiness Definition

An integration-ready subsystem has:

- a stable simulator contract
- a known live equivalent
- a read-only adapter boundary
- a no-mutation rule
- a required test set for future Phase 12 work
- clear ownership of current live authority

Readiness reports are diagnostics only. A readiness score cannot authorize live integration, canonical mutation, or public behavior changes.

## Terminology

- Kernel: simulator canonical commit boundary.
- Semantic syscall: deterministic mutation request contract.
- Courthouse: evidence admission authority in the simulator.
- Memory Palace: retrieval topology and CandidateObject handling.
- CandidateObject: retrieval candidate, not admitted evidence.
- Exhibit: Courthouse evidence record.
- ContextBlock and ContextBundle: compiled shape, not truth.
- KVCacheManifest: acceleration metadata, not memory.
- RuntimeGenerateResult: runtime proposal output, not truth.
- MaintenanceReport and CleanupProposal: Lymphatic diagnostics/proposals.
- ConsensusReport: claim governance diagnostic; accepted consensus is not canonical truth.
- EvidenceRef: provenance-preserving evidence pointer.
- ReadOnlyRAGAdapter: diagnostic observer for existing retrieval/search/embedding/VSA metadata.
- ShadowModePolicy: no-mutation diagnostic mirror policy.
- LivePathMapping: current live authority to FORGE-K target mapping.

## Subsystem Readiness Matrix

| Subsystem | Simulator status | Contract stability | Live equivalent | Adapter needed | Test gaps | Risk | Next action |
|---|---|---|---|---|---|---|---|
| Kernel | Implemented/tested | Stable simulator syscall contract | AI-OS/control lane | ReadOnlyLiveStateAdapter | Route inventory, shadow comparison | High | Phase 12A design |
| Neuron Fabric | Implemented/tested | Stable proposal/validation envelopes | Lanes, permissions, gateway routing | LiveEvidenceAdapter | Live proposal mirror | Medium | Define read-only neuron shadow inputs |
| Courthouse | Implemented/tested | Stable evidence/admission model | Control lane semantic admission | LiveEvidenceAdapter | Admission parity | High | Map live evidence records |
| Memory Palace | Implemented/tested | Stable retrieval-shape refs | Memory, retrieval, search, embeddings | ReadOnlyRAGAdapter | Retrieval provenance mirror | High | Keep retrieval evidence-only |
| Semantic Algebra | Implemented/tested | Stable operation records | Control lane semantic records | LiveEvidenceAdapter | Operation provenance parity | Medium | Define transform mirror fixtures |
| Snapshots | Implemented/tested | Stable shape-not-truth contract | Backup/release and snapshot/restore surfaces | LiveMemoryMirrorAdapter | Restore non-execution shadow | Medium | Prepare restore-seed mirror reports |
| Context Compiler | Implemented/tested | Stable deterministic context blocks | Live COMPILE_CONTEXT path | LiveContextCompileMirrorAdapter | Context comparison | High | Design read-only context shadow harness |
| KV System | Implemented/tested | Stable metadata validation gates | Modelruntime/cache metadata | LiveModelRuntimeTraceAdapter | No live KV reuse | Medium | Keep KV metadata diagnostic-only |
| Runtime Boundary | Implemented/tested with mock driver | Stable driver-as-proposal boundary | Modelruntime | LiveModelRuntimeTraceAdapter | Real driver non-authority | High | Defer real runtime drivers |
| Lymphatic Lane | Implemented/tested | Stable maintenance proposal model | Dream/autonomy maintenance | ReadOnlyLiveStateAdapter | No cleanup mutation shadow | Medium | Mirror reports only |
| Consensus Mesh | Implemented/tested | Stable claim governance | Response/action proposal planning | LiveConsensusShadowAdapter | Composer guard shadow | Medium | Compare diagnostics only |
| Rust Validator | Implemented/tested as research/tooling | Stable fixture validation for selected models | CI/tooling only | None | Future consensus parity | Low | Keep out of live authority |

## Live Path Mapping

The detailed mapping is in `docs/reviews/forge_k_live_path_mapping.md`. Every Phase 11F row has `live_mutation_allowed = NO`.

## Adapter Contract Model

Adapters may observe, normalize, mirror, and report.

Adapters may not mutate, execute, commit, approve, route, publish, call model runtimes, perform retrieval, call embedding providers, compile context, admit evidence, write memory, or affect user-visible output.

Default adapter classes:

- ReadOnlyLiveStateAdapter
- LiveEvidenceAdapter
- LiveMemoryMirrorAdapter
- LiveRetrievalMirrorAdapter
- ReadOnlyRAGAdapter
- LiveEmbeddingTraceAdapter
- LiveSearchTraceAdapter
- LiveGatewayTraceAdapter
- LiveModelRuntimeTraceAdapter
- LiveAuditMirrorAdapter
- LiveContextCompileMirrorAdapter
- LiveConsensusShadowAdapter

## RAG / Retrieval Boundary

RAG and retrieval are integration-prep only in Phase 11F. Retrieval does not make content truth. Embeddings, vector indexes, VSA records, and retrieval results are evidence/routing signals, not canonical truth.

ReadOnlyRAGAdapter may observe existing live retrieval run metadata, search result metadata, embedding/VSA trace metadata, source refs, and evidence refs. It may produce retrieval provenance diagnostics and shadow RAG reports.

ReadOnlyRAGAdapter must not execute new retrieval queries from FORGE-K, call embedding providers, mutate memory, admit evidence, compile context, call modelruntime, alter live responses, write live memory, produce user-visible output, or promote retrieved content to truth.

## Shadow Mode Summary

Shadow mode is defined in `docs/architecture/shadow_mode.md`. In Phase 11F it is a policy and contract design only. Live requests execute through existing live paths. FORGE-K may mirror inputs and produce diagnostics only.

## Dry-Run Bridge And Mirror Rules

- Dry-run bridge outputs are reports, warnings, risk flags, and evidence refs.
- Read-only mirror outputs preserve provenance and source lineage.
- Mirror output does not commit truth, approve actions, admit evidence, compile context for live response, execute tools, or write memory.
- Any live mutation path is a hard stop.

## Test Strategy

Phase 11F tests cover:

- deterministic report ordering and serialization
- forbidden live daemon imports
- readiness score advisory-only behavior
- no live mutation across adapters, mappings, and shadow policy
- ReadOnlyRAGAdapter boundary
- shadow policy diagnostics-only behavior
- live path mapping completeness

## Phase 12 Gates

Before live integration can start, all gates must be met:

- all simulator tests pass
- route inventory tests pass
- live path mapping exists
- adapter contracts exist
- RAG/retrieval read-only boundary exists
- no-mutation shadow policy exists
- rollback plan exists
- live authority owner identified
- no direct live mutation from FORGE-K
- shadow comparison tests exist
- consensus/composer guard remains enforced
- gateway execution remains live authority until migration
- memory writes remain live authority until migration
- retrieval/RAG remains evidence-only until migration
- no user-visible output is affected by shadow mode

## Risks

- Authority confusion between simulator and live daemon.
- Accidental adapter mutation.
- RAG or retrieval being treated as truth.
- Shadow output leaking into live response.
- Duplicated gateway, control lane, or Kernel authority.
- Consensus accepted being mistaken for canonical truth.
- ContextBlocks being mistaken for truth instead of compiled shape.
- Model output being mistaken for authority instead of driver output.

## What Not To Do

- Do not wire FORGE-K into the live daemon.
- Do not create live mutation adapters.
- Do not implement live RAG.
- Do not call live retrieval or embedding providers from FORGE-K.
- Do not replace live controllane.
- Do not replace gateway.
- Do not change API routes.
- Do not execute tools from FORGE-K.
- Do not call modelruntime from FORGE-K.
- Do not let shadow mode affect user-visible output.
- Do not create a second authority path.
- Do not skip Phase 12 design.
