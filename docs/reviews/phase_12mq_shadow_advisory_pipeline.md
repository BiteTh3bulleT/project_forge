# Phase 12M-Q Shadow Advisory Pipeline

Status: `IMPLEMENTED + TESTED`

Scope: `LIVE_INTEGRATION / READ_ONLY / DISABLED_BY_DEFAULT / ADVISORY_DIAGNOSTIC_ONLY`

## Summary

Phase 12M-Q adds an internal shadow advisory report to the existing `services/core/internal/forgekshadow` diagnostic flow. It is enabled only when both `FORGE_K_SHADOW_MODE_ENABLED=true` and `FORGE_K_SHADOW_ADVISORY_ENABLED=true`.

The advisory consumes existing safe in-memory diagnostics only. It does not create a public diagnostics API, does not persist reports, does not execute tools, does not call modelruntime, does not run retrieval/search/embeddings/live RAG, does not write memory, does not call controllane mutation paths, and does not affect user-visible output.

## Components

- `ShadowAdvisoryReport`: internal diagnostic report attached to `DiagnosticReport`.
- `ShadowEvidenceSummary`: counts route/chat/retrieval metadata records and safe refs.
- `ShadowConsensusAdvisory`: metadata-only uncertainty summary; accepted factual claims remain zero.
- `ShadowContextCompilerAdvisory`: deterministic context-summary hash or safe insufficient-metadata warning.
- `ShadowRiskSummary`: no-raw-content and metadata-only risk summary.

## Flag Behavior

- `FORGE_K_SHADOW_MODE_ENABLED` remains the global kill switch and defaults to `false`.
- `FORGE_K_SHADOW_ADVISORY_ENABLED` defaults to `false`.
- Advisory reports require both flags.
- Advisory mode does not force-enable chat metadata observation.
- Advisory mode does not force-enable retrieval metadata observation.

The advisory flag is environment/config controlled in this phase. It does not add settings API fields, public diagnostics APIs, or route behavior.

## Metadata Consumed

The advisory may consume only already-recorded safe metadata:

- source shadow observation/report refs
- route-envelope observation refs and matched route pattern refs
- chat metadata observation refs and safe thread/message refs
- retrieval metadata observation refs, run/result refs, safe source refs, and source hashes
- counts, warnings, no-effect verification, and diagnostic metadata markers

## Metadata Not Consumed

The advisory must not consume or store:

- request bodies
- response bodies
- prompts
- completions
- assistant response text
- source text
- chunk text
- document content
- raw user queries
- search snippets
- embeddings or vectors
- RAG output
- tool payloads or outputs
- modelruntime payloads or outputs
- memory content
- auth headers, cookies, tokens, API keys, or secrets

## Context Compiler Shadow Behavior

The advisory does not call the simulator Context Compiler and does not alter live `COMPILE_CONTEXT`. It may create a deterministic summary hash from safe refs and advisory block counts. If safe metadata is insufficient, it records a warning instead of compiling anything.

The context advisory is not a `ContextBlock`, not a `ContextBundle`, not a live prompt, not KV cache, and not user-visible output.

## Consensus Mesh Advisory Behavior

The advisory does not run live Consensus Mesh authority. It can record metadata-only uncertainty counts and warnings. Accepted claim count remains zero, and no advisory claim becomes canonical truth, admitted evidence, memory, or response composition authority.

## No-Effect Guarantees

Phase 12M-Q does not:

- change route inventory
- change response status, body, or headers
- add routes or public APIs
- persist diagnostics
- execute tools
- call modelruntime
- execute retrieval/search/embedding calls
- compile live context
- perform live RAG
- write memory
- mutate controllane state
- alter gateway behavior
- create a second authority path

## Validation Evidence

Required validation is recorded in `docs/reviews/current_phase_status.md` for Phase 12M-Q.

## Deferred Work

- No public diagnostics API.
- No persisted report store.
- No live Context Compiler integration.
- No live Consensus Mesh integration.
- No response composer integration.
- No authority migration.
