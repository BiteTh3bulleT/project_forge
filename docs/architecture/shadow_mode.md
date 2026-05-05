# FORGE-K Shadow Mode

Status: Phase 11F design contract only. Scope is `SIMULATOR_ONLY / INTEGRATION_PREP_ONLY`.

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

Phase 11F defines the policy and contracts. It does not implement a live shadow harness.

## Allowed Operations

Shadow mode may support these diagnostic operations in a future scoped phase:

- request mirroring
- evidence mirroring
- retrieval/RAG shadow reports
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

Stop Phase 11F or any future shadow implementation if an adapter or shadow path:

- imports live daemon packages into `services/core/internal/forgek/integrationready`
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
