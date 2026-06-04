# Layered RAG Context Engine

Status: proposed internal architecture.
Date: 2026-06-04.

## Authority Banner

AXIOM is an internal FORGE cognition, search, and context layer. It does not execute tools, does not write canonical memory, and does not bypass Gateway, approvals, audit, Control Lane, or modelruntime. Search results are evidence candidates. FORGE-K remains simulator and shadow validation.

## Goal

The layered RAG context engine turns candidate evidence into bounded, cited context packets. It separates expansion from contraction so broad recall does not become authority.

## Expansion

Expansion gathers candidate refs from:

- current workspace paths
- canonical memory reads and journal refs
- retrieval service results
- search service results
- embeddings and vector recall
- official documentation sources
- curated project docs and ADRs
- audit and artifact refs

Expansion is allowed to be broad. It may include stale memory, low-trust recall, web results, and contradictory sources because none of these candidates are admitted to context yet.

## Contraction

Contraction selects context under policy and budget:

- exclude stale memory when fresher scoped evidence exists
- exclude low-trust results from answer-critical context unless no better source exists and the response marks uncertainty
- prefer official source candidates for external factual claims
- prefer local live workspace candidates for current local behavior
- keep citation and source refs with every selected claim
- record rejected candidates and rejection reasons
- preserve contradictions when the system cannot resolve them from higher-trust evidence

## Context Packet Rules

A `ContextPacket` is a compiled prompt-support object, not canonical memory. It must carry:

- workspace and lane scope
- routing mode
- selected refs
- rejected refs
- trust-tier and freshness summaries
- citation metadata
- token-budget metadata
- provenance to `SearchEvidencePacket` inputs

It must not:

- call modelruntime directly
- execute tools
- approve actions
- write memory
- mutate state
- hide low-trust or rejected evidence from audit

## Citation Rules

Every selected answer-critical fact needs a citation to a source ref. Citation quality should prefer file line refs, canonical object refs, audit IDs, route IDs, official URLs, or artifact IDs over plain text snippets.

## Integration With Existing FORGE Paths

The context engine feeds modelruntime through the existing chat and API composition path. If the response proposes an action, that action still flows through Gateway and approval gates. If the response proposes memory or state changes, those proposals still flow through Control Lane semantic syscalls.
