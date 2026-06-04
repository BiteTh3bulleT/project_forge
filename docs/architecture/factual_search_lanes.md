# Factual Search Lanes

Status: proposed internal architecture.
Date: 2026-06-04.

## Authority Banner

AXIOM is an internal FORGE cognition, search, and context layer. It does not execute tools, does not write canonical memory, and does not bypass Gateway, approvals, audit, Control Lane, or modelruntime. Search results are evidence candidates. FORGE-K remains simulator and shadow validation.

## Purpose

Factual search lanes keep source classes separate until the contraction step. A lane is a routing and evidence-classification boundary, never authorization. A lane can improve recall and ranking; it cannot approve a tool, commit memory, or bypass policy.

## Lane Classes

| Lane | Source | Trust posture | Notes |
|---|---|---|---|
| Local live workspace | Current repo files, generated route maps, live config | Highest for current local behavior | Must keep workspace and path scope. |
| Canonical FORGE state | Control Lane stores, journal, audit refs | Highest for committed FORGE truth | Read-only evidence input. |
| Official source | Official docs and specs | Highest for external factual claims | Requires date/source metadata. |
| Curated project docs | ADRs, architecture docs, status docs | Strong if current | Staleness checks required. |
| Web search | General web pages | Discovery/corroboration | Never authorization. |
| Vector recall | embeddings, VSA, Qdrant shadow metadata | Recall only | Never canonical truth. |
| Low-trust scratch | snippets, weak summaries, unsourced text | Review only | Cannot enter canonical memory. |

## Lane Ordering

For current local questions:

1. local live workspace
2. canonical FORGE state
3. curated project docs
4. vector recall
5. web search

For current external factual questions:

1. official source
2. local live workspace, if it captures configured behavior
3. curated project docs
4. web search
5. vector recall

For historical or audit questions:

1. audit and journal refs
2. canonical state snapshots
3. curated reports
4. vector recall
5. web search

## Authorization Boundary

Search lanes are never authorization. A result from local live workspace, official source, web search, or vector recall can support a response or a proposed action, but execution still goes through Gateway, approvals, audit, Control Lane, and modelruntime as applicable.

## Scope Rules

- Reject absolute paths that are outside the selected workspace.
- Reject path traversal and ambiguous workspace roots.
- Keep lane ID, workspace ID, source ref, and freshness metadata attached to every candidate.
- Keep rejected candidate records when contraction excludes a source.

## Output

Each lane emits normalized candidates:

- source ref
- source title
- source class
- trust tier
- freshness
- scope match
- citation span or object ref
- extraction summary
- rejection status and reason, if excluded
