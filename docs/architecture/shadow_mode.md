# FORGE-K Shadow Mode

Status: Phase 11F policy/contracts plus Phase 11G simulator-only harness design.

Phase 11F defines integration readiness contracts, read-only adapters, and shadow policy. Phase 11G defines the simulator-only harness and report model design in `docs/architecture/shadow_mode_harness.md`. Neither phase implements live shadow mode or authorizes Phase 12 live integration.

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

Phase 11G adds the harness/report design. Actual live observation is deferred to Phase 12B.

## Phase Split

- Phase 11F: integration readiness contracts, read-only adapter contracts, ReadOnlyRAGAdapter boundary, and shadow policy.
- Phase 11G: simulator-only shadow harness/report model design in `docs/architecture/shadow_mode_harness.md` and `services/core/internal/forgek/shadowharness`.
- Phase 12B: future read-only live shadow harness implementation after Phase 12A live integration design.
- Phase 12 live integration is not authorized by Phase 11F or Phase 11G.

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
