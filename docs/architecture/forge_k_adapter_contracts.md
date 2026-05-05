# FORGE-K Adapter Contracts

Status: Phase 11F `SIMULATOR_ONLY / INTEGRATION_PREP_ONLY`.

These contracts define future adapter boundaries conceptually. They do not wire FORGE-K into the live daemon and do not authorize Phase 12 live integration.

## Adapter Law

Adapters may observe, normalize, mirror, and report.

Adapters may not mutate, execute, commit, approve, route, publish, call model runtimes, perform live retrieval, call embedding providers, write memory, or alter user-visible output.

All Phase 11F adapters are read-only and diagnostic only. They must preserve provenance.

## Contract Fields

Future adapter descriptions use:

- adapter_id
- adapter_type
- purpose
- allowed_operations
- forbidden_operations
- source_system
- target_forge_k_component
- preserves_provenance
- read_only
- live_mutation_allowed
- tool_execution_allowed
- modelruntime_call_allowed
- retrieval_execution_allowed
- user_visible_output_allowed
- metadata

Required defaults:

- `read_only = true`
- `preserves_provenance = true`
- `live_mutation_allowed = false`
- `tool_execution_allowed = false`
- `modelruntime_call_allowed = false`
- `retrieval_execution_allowed = false`
- `user_visible_output_allowed = false`

## Adapter Types

| Adapter | Purpose | Forbidden in Phase 11F |
|---|---|---|
| ReadOnlyLiveStateAdapter | Observe live state metadata and normalize refs for diagnostics. | Mutation, routing, commits, approvals. |
| LiveEvidenceAdapter | Mirror live evidence-like records into simulator evidence refs. | Submit/admit evidence or write canonical truth. |
| LiveMemoryMirrorAdapter | Mirror live memory metadata and provenance. | Memory writes, promotion to truth, deletes. |
| LiveRetrievalMirrorAdapter | Mirror existing retrieval run/result metadata. | New retrieval queries or ranking changes. |
| ReadOnlyRAGAdapter | Observe existing retrieval/search/embedding/VSA metadata for shadow diagnostics. | Live RAG, context compilation, evidence admission, response changes. |
| LiveEmbeddingTraceAdapter | Mirror existing embedding trace metadata. | Calling embedding providers or reindexing. |
| LiveSearchTraceAdapter | Mirror existing search result metadata. | Executing search from FORGE-K. |
| LiveGatewayTraceAdapter | Mirror gateway invocation/approval/audit traces. | Executing tools or replacing gateway. |
| LiveModelRuntimeTraceAdapter | Mirror modelruntime identity, request, and result metadata. | Calling model runtimes or registering drivers. |
| LiveAuditMirrorAdapter | Mirror audit records as provenance evidence. | Writing audit records or treating audit as semantic truth. |
| LiveContextCompileMirrorAdapter | Mirror live context compile metadata for comparison. | Replacing live COMPILE_CONTEXT. |
| LiveConsensusShadowAdapter | Produce consensus diagnostics from mirrored evidence refs. | Affecting live response/action composition. |

## ReadOnlyRAGAdapter

Purpose: observe live retrieval/search/memory outputs and normalize them into FORGE-K EvidenceRefs for shadow-mode diagnostics.

Allowed:

- observe existing live retrieval run metadata
- observe existing live search result metadata
- observe existing embedding/VSA trace metadata
- normalize source refs
- normalize evidence refs
- produce shadow RAG reports
- produce retrieval provenance diagnostics

Forbidden:

- execute new retrieval queries from FORGE-K
- call embedding providers
- mutate memory
- admit evidence
- compile context
- call modelruntime
- alter live response
- write live memory
- produce user-visible output
- promote retrieved content to truth

Retrieval does not make content truth. Embeddings, vector indexes, VSA records, and retrieval results are evidence/routing signals only.

## Provenance

Every adapter output must preserve:

- source system
- source record id or locator
- workspace/correlation scope when available
- observed timestamp when available
- normalization steps
- diagnostic report id when produced

Provenance preservation does not grant authority. A mirrored record is evidence for future review, not canonical truth.

## What Not To Do

- Do not wire adapters to live systems in Phase 11F.
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
