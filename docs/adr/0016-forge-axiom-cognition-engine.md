# ADR 0016 - FORGE AXIOM Cognition Engine

Status: Proposed

Date: 2026-06-04

## Context

FORGE already has live authority paths for tools, approvals, audit, semantic mutation, retrieval, embeddings, search, vectorstore, memory, and modelruntime. FORGE-K remains target architecture in the simulator with read-only or validation-only shadow seams unless a later promotion phase explicitly changes that boundary.

The AXIOM prompt pack introduces a useful big-brain search, RAG, and context-engine direction. The risk is creating a duplicate authority plane that can search, decide, approve, execute, and write memory outside existing FORGE governance.

AXIOM is an internal FORGE cognition, search, and context layer. It does not execute tools, does not write canonical memory, and does not bypass Gateway, approvals, audit, Control Lane, or modelruntime. Search results are evidence candidates. FORGE-K remains simulator and shadow validation.

## Decision

Adopt AXIOM as an internal FORGE cognition/search/context subsystem, not as a separate product, app, daemon authority, approval queue, tool broker, memory authority, or modelruntime path.

The implementation direction is:

- Define AXIOM outputs as `SearchEvidencePacket` and `ContextPacket` style records.
- Assign each candidate a `TrustTier` and route each request with a `RoutingMode`.
- Preserve rejected candidates with reasons.
- Prefer live local workspace evidence for current local facts.
- Prefer official documentation for current external factual claims.
- Treat web search and vector recall as discovery and corroboration, never authorization.
- Keep canonical mutation in Control Lane.
- Keep external effects in Gateway.
- Keep approval records in the approvals service.
- Keep traceability in audit.
- Keep provider calls in modelruntime.
- Keep FORGE-K as simulator and shadow validation.

This decision creates no duplicate authority plane.

## Consequences

- AXIOM can improve answer quality, context packing, source ranking, and explainability without changing live authority.
- Existing retrieval, search, embeddings, vectorstore, memory, modelruntime, audit, approvals, gateway, policy, permissions, and lane packages remain the extension targets.
- Tests for future implementation must prove that vector/web evidence alone cannot authorize execution, low-trust evidence cannot become canonical memory, stale memory is excluded when fresher scoped evidence exists, official docs outrank weaker web results for factual claims, and packets record rejected candidates.
- Documentation must continue to call out the boundary because the search/RAG subsystem is close to memory, truth, and action authorization.

## Alternatives Considered

- **Create AXIOM as a standalone app or daemon.** Rejected because it would duplicate FORGE's authority plane and create unclear operator ownership.
- **Route tool execution through AXIOM.** Rejected because Gateway is already the only authorized tool execution boundary.
- **Let AXIOM write memory directly.** Rejected because canonical memory belongs to governed semantic Control Lane paths.
- **Promote FORGE-K context compiler as live authority now.** Rejected because FORGE-K remains simulator and shadow validation.

## Out Of Scope

- New API routes.
- New live execution paths.
- New approval service or approval schema.
- New canonical memory writer.
- New modelruntime provider orchestration.
- Live FORGE-K integration.
- Host mutation, Nix mutation, or workstation substrate changes.

## References

- `docs/architecture/forge_axiom_cognition.md`
- `docs/architecture/big_brain_search.md`
- `docs/architecture/factual_search_lanes.md`
- `docs/architecture/layered_rag_context_engine.md`
- `docs/architecture/context_compiler_search_index.md`
- `docs/architecture/forge_wiring_map.md`
- `docs/architecture/tool_gateway.md`
- `docs/architecture/forge_k_integration_readiness.md`
- ADR 0005 - FORGE-K Simulator vs Live Authority
