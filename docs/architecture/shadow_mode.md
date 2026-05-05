# FORGE-K Shadow Mode

Status: Phase 11F policy/contracts, Phase 11G simulator-only harness design, Phase 12A live integration design, Phase 12B disabled-by-default `/health` observer, Phase 12C diagnostics hardening, Phase 12D controlled expansion design, Phase 12E disabled-by-default route-envelope metadata observer, and Phase 12F route-envelope hardening.

Phase 11F defines integration readiness contracts, read-only adapters, and shadow policy. Phase 11G defines the simulator-only harness and report model design in `docs/architecture/shadow_mode_harness.md`. Phase 12A designs the first live shadow implementation path in `docs/architecture/forge_k_live_integration_design.md`. Phase 12B implements one disabled-by-default read-only `/health` metadata observer. Phase 12C hardens that observer. Phase 12D designs a controlled next expansion. Phase 12E implements disabled-by-default route-envelope metadata shadowing. Phase 12F hardens that route-envelope shadowing without adding observation scope. None of these phases authorizes live mutation.

## Definition

Shadow mode means:

- the live request executes through the existing live path
- FORGE-K simulator may observe or mirror inputs
- FORGE-K may produce comparison reports
- FORGE-K may produce Consensus Mesh and RAG diagnostic reports
- FORGE-K does not affect the live response
- FORGE-K does not mutate live state
- FORGE-K does not execute tools
- FORGE-K does not write memory
- FORGE-K does not call live modelruntime
- FORGE-K does not perform live retrieval
- FORGE-K does not alter user-visible output
- all shadow outputs are diagnostics only

Phase 11G adds the harness/report design. Phase 12B adds the first live observation point: `/health` metadata only. Phase 12E adds route-envelope metadata observation behind the same disabled-by-default flag. Phase 12F hardens route-envelope safety, redaction, sink isolation, and no-effect tests without adding routes, persistence, or new live touchpoints.

## Phase Split

- Phase 11F: integration readiness contracts, read-only adapter contracts, ReadOnlyRAGAdapter boundary, and shadow policy.
- Phase 11G: simulator-only shadow harness/report model design in `docs/architecture/shadow_mode_harness.md` and `services/core/internal/forgek/shadowharness`.
- Phase 12A: docs-only live integration design; does not authorize implementation.
- Phase 12B: disabled-by-default read-only live shadow harness implementation for `/health` metadata only.
- Phase 12C: observability-only hardening of Phase 12B diagnostics with no new touchpoints.
- Phase 12D: docs-only controlled expansion design; route envelope metadata selected as the next candidate.
- Phase 12E: disabled-by-default route-envelope metadata observer for matched route patterns, route classes, method, timing, and safe request ids only.
- Phase 12F: observability-only route-envelope hardening with no new touchpoints, no public diagnostics route, no persistence, and no body/query/header/secret capture.
- Phase 12 authority migration is not authorized by Phase 11F, Phase 11G, Phase 12A, Phase 12B, Phase 12C, Phase 12D, Phase 12E, or Phase 12F.

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
- authorize Phase 12 live integration

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

## Phase 12B - 12F Current Scope

Phase 12B through 12F currently allow only:

- disabled-by-default feature flag: `FORGE_K_SHADOW_MODE_ENABLED=false`
- read-only observation of `/health` request metadata
- read-only route-envelope metadata: method, matched route pattern, normalized route class, duration, and safe request id when available
- bounded in-memory diagnostic reports only
- no public API or route changes
- no user-visible effect
- no tool, modelruntime, retrieval, embedding, memory, or controllane execution from FORGE-K
- no request body, response body, prompt, model output, tool payload, retrieval content, or memory content capture
- no raw query strings, raw request URI values, raw headers, cookies, authorization values, secrets, session values, embedding vectors, or search chunk content capture

Any expansion beyond route-envelope metadata requires a separately approved phase and must prove no-effect behavior with route inventory, response equivalence, forbidden execution, and no-mutation tests.
