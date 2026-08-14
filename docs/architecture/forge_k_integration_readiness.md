# FORGE-K Integration Readiness

Status: Historical Phase 11F readiness record, amended for K20A. ADR 0017 and `forge_k_live_cutover.md` are authoritative for the active production migration.

Phase 11F does not authorize live authority migration.

Current boundary banner:

- Simulator services in `services/core/internal/forgek` remain target architecture, not live daemon authority.
- Phase 12 shadow diagnostics in `services/core/internal/forgekshadow` are read-only metadata observers and advisory report builders. They do not commit truth, admit evidence, compile prompt context, call modelruntime, execute retrieval/search/embeddings, write memory, change routes, or affect user-visible output.
- Phase 14 Control Lane validation seams use shared pure validators from live-owned actions such as `VALIDATE_KV_IDENTITY`, `VALIDATE_REF_SHAPE`, `COMPARE_REF_SHAPE`, `VALIDATE_SOURCE_OBJECT_AUTHORITY`, and `VALIDATE_SEMANTIC_OPERATION`. They are validation/enforcement gates only, not live FORGE-K Kernel authority.
- K20A production semantic syscall ingress is live. Live Courthouse admission, live Context Compiler authority, FORGE-K-owned durable commit ports, runtime driver authority, and broader Kernel authority remain blocked until separate migration gates are approved and proven.

## Executive Summary

FORGE-K Phase 1-11E is implemented and tested in the simulator under `services/core/internal/forgek`. The distinct production `internal/forgekernel` boundary now owns semantic syscall ingress by default; the live daemon still uses existing gateway, permissions, lanes, audit, modelruntime, API, retrieval, embeddings, memory, and temporary Control Lane durable-adapter paths. ADR 0005 remains authoritative for simulator isolation and ADR 0017 governs production cutover.

Integration readiness means the simulator has documented contracts, live path mappings, adapter boundaries, shadow-mode rules, and tests that make a later live integration plan possible. It is not live integration.

## Current Simulator / Live Split

| Area | Current authority |
|---|---|
| FORGE-K simulator | Kernel, Neuron Fabric, Courthouse, Memory Palace, Semantic Algebra, Snapshots, Context Compiler, KV System, Runtime Boundary, Lymphatic Lane, Consensus Mesh, Rust fixture validation. |
| Live daemon | Production FORGE-K syscall ingress; temporary AI-OS/Control Lane durable adapter; gateway, permissions, lanes, audit, modelruntime, API routing, memory, retrieval, search, embeddings, dream/autonomy, backup/release, settings/config. |

The simulator may define target contracts. It must not route live requests, execute live tools, call live model runtimes, perform live retrieval, call embedding providers, write live memory, or alter user-visible output.

## Integration Readiness Definition

An integration-ready subsystem has:

- a stable simulator contract
- a known live equivalent
- a read-only adapter boundary
- a no-mutation rule
- a required test set for any later read-only diagnostic or validation seam work
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

Shadow mode is defined in `docs/architecture/shadow_mode.md`. In Phase 11F it was a policy and contract design only. Phase 12 later implemented disabled-by-default read-only metadata diagnostics outside the simulator package. Live requests still execute through existing live paths, and shadow output remains diagnostic/advisory only.

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

## Live Authority Gates

The following gates are required before any step can move beyond read-only diagnostics or validation-only Control Lane seams:

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
- source-object authority lookup remains validation/read-only unless explicitly promoted
- Courthouse admission integration remains blocked until live evidence ownership, rollback, and tests exist
- live Context Compiler prompt authority remains blocked until no-effect comparison and prompt authority tests exist
- governed semantic mutation routing remains blocked until syscall, audit, approval, and rollback boundaries are designed
- runtime driver authority remains blocked until modelruntime trace-only and proposal-only guarantees are proven

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
