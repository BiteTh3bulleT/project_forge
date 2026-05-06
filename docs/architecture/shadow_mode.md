# FORGE-K Shadow Mode

Status: Phase 11F policy/contracts, Phase 11G simulator-only harness design, Phase 12A live integration design, Phase 12B disabled-by-default `/health` observer, Phase 12C diagnostics hardening, Phase 12D controlled expansion design, Phase 12E disabled-by-default route-envelope metadata observer, Phase 12F route-envelope hardening, Phase 12G chat metadata expansion design, Phase 12H disabled-by-default chat metadata shadowing, Phase 12I chat metadata hardening, Phase 12J retrieval metadata expansion design, Phase 12K-L disabled-by-default retrieval metadata implementation/hardening, and Phase 12M-Q disabled-by-default shadow advisory reports.

Phase 11F defines integration readiness contracts, read-only adapters, and shadow policy. Phase 11G defines the simulator-only harness and report model design in `docs/architecture/shadow_mode_harness.md`. Phase 12A designs the first live shadow implementation path in `docs/architecture/forge_k_live_integration_design.md`. Phase 12B implements one disabled-by-default read-only `/health` metadata observer. Phase 12C hardens that observer. Phase 12D designs a controlled next expansion. Phase 12E implements disabled-by-default route-envelope metadata shadowing. Phase 12F hardens that route-envelope shadowing without adding observation scope. Phase 12G designs chat metadata. Phase 12H implements chat metadata diagnostics only when both the global shadow flag and chat-specific flag are enabled. Phase 12I hardens that chat metadata path without adding touchpoints or expanding capture scope. Phase 12J designs retrieval metadata. Phase 12K-L implements and hardens bounded retrieval metadata diagnostics only when both global and retrieval-specific flags are enabled. Phase 12M-Q adds an internal advisory report that consumes existing safe diagnostics only when both global shadow mode and the advisory flag are enabled. None of these phases authorizes live mutation.

## Definition

Shadow mode means:

- the live request executes through the existing live path
- FORGE-K simulator may observe or mirror inputs
- FORGE-K may produce comparison reports
- FORGE-K may produce advisory diagnostic reports from already-recorded safe metadata
- FORGE-K may produce Consensus Mesh and RAG diagnostic reports
- FORGE-K does not affect the live response
- FORGE-K does not mutate live state
- FORGE-K does not execute tools
- FORGE-K does not write memory
- FORGE-K does not call live modelruntime
- FORGE-K does not perform live retrieval
- FORGE-K does not alter user-visible output
- all shadow outputs are diagnostics only

Phase 11G adds the harness/report design. Phase 12B adds the first live observation point: `/health` metadata only. Phase 12E adds route-envelope metadata observation behind the same disabled-by-default flag. Phase 12F hardens route-envelope safety, redaction, sink isolation, and no-effect tests without adding routes, persistence, or new live touchpoints. Phase 12G designs the chat metadata boundary. Phase 12H adds the chat message POST metadata observer behind an additional disabled-by-default chat flag. Phase 12I hardens the Phase 12H observer and tests while preserving the same touchpoint and metadata boundary. Phase 12K-L adds retrieval run metadata diagnostics behind an additional disabled-by-default retrieval flag and captures no source/chunk/query/vector/RAG content. Phase 12M-Q adds advisory summaries derived from the existing diagnostic report and does not force-enable chat or retrieval observation.

## Phase Split

- Phase 11F: integration readiness contracts, read-only adapter contracts, ReadOnlyRAGAdapter boundary, and shadow policy.
- Phase 11G: simulator-only shadow harness/report model design in `docs/architecture/shadow_mode_harness.md` and `services/core/internal/forgek/shadowharness`.
- Phase 12A: docs-only live integration design; does not authorize implementation.
- Phase 12B: disabled-by-default read-only live shadow harness implementation for `/health` metadata only.
- Phase 12C: observability-only hardening of Phase 12B diagnostics with no new touchpoints.
- Phase 12D: docs-only controlled expansion design; route envelope metadata selected as the next candidate.
- Phase 12E: disabled-by-default route-envelope metadata observer for matched route patterns, route classes, method, timing, and safe request ids only.
- Phase 12F: observability-only route-envelope hardening with no new touchpoints, no public diagnostics route, no persistence, and no body/query/header/secret capture.
- Phase 12G: docs-only chat metadata expansion design; no chat observation, no message content capture, no prompt capture, no output capture, and no route/API behavior change.
- Phase 12H: disabled-by-default chat metadata observer for bounded ids/classes/counts/timing only; no message content, prompt, completion, tool payload, retrieval content, memory content, request/response body, route/API behavior, or user-visible output change.
- Phase 12I: observability-only chat metadata hardening with no new touchpoints, no broader capture, no persistence, and no route/API/user-visible behavior change.
- Phase 12J: docs-only retrieval metadata expansion design.
- Phase 12K-L: disabled-by-default retrieval metadata diagnostics and hardening after live retrieval run creation; no retrieval/search/embedding execution from FORGE-K, no source/chunk/vector/query/RAG output capture, and no route/API behavior change.
- Phase 12M-Q: disabled-by-default advisory diagnostic reports derived from existing safe shadow metadata; no live Context Compiler execution, no Consensus Mesh live authority, no route/API behavior change, and no user-visible output change.
- Phase 12 authority migration is not authorized by Phase 11F, Phase 11G, Phase 12A, Phase 12B, Phase 12C, Phase 12D, Phase 12E, Phase 12F, Phase 12G, Phase 12H, Phase 12I, Phase 12J, Phase 12K-L, or Phase 12M-Q.

`services/core/internal/forgek/integrationready` remains the Phase 11F readiness package. It does not implement the Phase 11G harness.

## Allowed Operations

Shadow mode may support these diagnostic operations in a future scoped phase:

- request mirroring
- evidence mirroring
- retrieval/RAG shadow reports
- runtime shadow reports
- KV shadow reports
- lymphatic shadow reports
- response comparison
- context comparison
- consensus comparison
- risk flags
- provenance-preserving diagnostic reports

## Forbidden Operations

Shadow mode must not:

- mutate live state
- execute tools
- write memory
- call live modelruntime
- perform live retrieval
- call embedding providers
- compile context from live RAG
- alter responses
- create user-visible output
- change routes or public APIs
- authorize canonical mutation
- authorize live authority migration or live state mutation

## Request Mirroring

Request mirroring may copy stable refs, request metadata, workspace/correlation ids, timestamps, and provenance into diagnostic records. It must never take ownership of live routing or live response behavior.

## Evidence Mirroring

Evidence mirroring may normalize live records into EvidenceRef-style diagnostics. It must not submit evidence, admit evidence, reject evidence, mutate CasePacket state, or claim Courthouse authority.

## Retrieval / RAG Shadow Reporting

RAG shadow reporting may observe existing live retrieval runs, retrieval results, search results, embedding records, and VSA traces. FORGE-K must not run retrieval from shadow mode.

ReadOnlyRAGAdapter output is a diagnostic report. It is not ContextBlock input for live response, not admitted evidence, not canonical memory, and not a model prompt.

## Comparison Reports

Shadow reports may compare:

- live response shape against Consensus Mesh diagnostics
- live context metadata against Context Compiler diagnostics
- retrieval provenance against Memory Palace route shape
- runtime traces against Runtime Boundary proposal-only rules
- KV metadata against acceleration-not-memory rules
- maintenance reports against Lymphatic no-silent-mutation rules
- risk flags against policy expectations

Consensus accepted does not mean canonical truth. ContextBlocks are compiled shape, not truth. Models are drivers, not authority.

## Audit And Reporting

Shadow reports must be:

- diagnostic
- provenance-preserving
- scoped to workspace/correlation metadata
- inspectable
- non-authoritative

No report can approve live behavior, mutate state, admit evidence, execute actions, write memory, or alter user-visible output.

## Hard Stops

Stop Phase 11G or any future shadow harness implementation if an adapter or shadow path:

- imports live daemon packages into `services/core/internal/forgek/integrationready`
- imports live daemon packages into `services/core/internal/forgek/shadowharness`
- mutates live state
- calls modelruntime
- executes tools
- runs retrieval or embeddings
- writes memory
- changes routes or public APIs
- affects user-visible output
- creates a second authority path

## Acceptance Language

Every shadow output is diagnostics only. No shadow result can authorize live integration, canonical mutation, tool execution, memory writes, model runtime calls, retrieval execution, or user-visible output changes.

## Phase 12B - 12M-Q Current Scope

Phase 12B through 12M-Q currently allow only:

- disabled-by-default feature flag: `FORGE_K_SHADOW_MODE_ENABLED=false`
- disabled-by-default chat metadata feature flag: `FORGE_K_SHADOW_CHAT_METADATA_ENABLED=false`
- disabled-by-default retrieval metadata feature flag: `FORGE_K_SHADOW_RETRIEVAL_METADATA_ENABLED=false`
- disabled-by-default advisory feature flag: `FORGE_K_SHADOW_ADVISORY_ENABLED=false`
- read-only observation of `/health` request metadata
- read-only route-envelope metadata: method, matched route pattern, normalized route class, duration, and safe request id when available
- read-only chat metadata from the chat message POST handler only when both flags are enabled: operation kind, safe ids/refs, role class, stream class, safe model id, bounded counts, timing, and diagnostic markers
- read-only retrieval metadata after live retrieval run creation only when both global and retrieval flags are enabled: run/result refs, source type/ref, result count, selected count, bounded score summary, ranking position, retrieval strategy, index type, safe embedding model id, timing, and diagnostic markers
- internal advisory reports only when both global and advisory flags are enabled: source diagnostic refs, route/chat/retrieval metadata refs, safe ref counts, metadata-only consensus uncertainty summaries, context-summary hashes, cache-eligibility counts, risk flags, and warnings
- bounded in-memory diagnostic reports only
- no public API or route changes
- no user-visible effect
- no tool, modelruntime, retrieval, embedding, memory, or controllane execution from FORGE-K
- no live Context Compiler execution, live prompt compilation, Consensus Mesh live authority, response composition, or factual claim acceptance from advisory reports
- no request body, response body, prompt, model output, tool payload, retrieval content, or memory content capture
- no raw query strings, raw request URI values, raw headers, cookies, authorization values, secrets, session values, embedding vectors, or search chunk content capture
- no chat content, prompt, completion, assistant response text, system prompt, request body, response body, tool payload, retrieval content, or memory content capture
- no source text, chunk text, document content, embedding, vector, raw query, or RAG output capture

Any expansion beyond advisory diagnostics requires a separately approved phase and must prove no-effect behavior with route inventory, response equivalence, forbidden execution, and no-mutation tests.
