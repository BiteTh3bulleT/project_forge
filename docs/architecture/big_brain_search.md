# Big-Brain Search

Status: proposed internal architecture.
Date: 2026-06-04.

## Authority Banner

AXIOM is an internal FORGE cognition, search, and context layer. It does not execute tools, does not write canonical memory, and does not bypass Gateway, approvals, audit, Control Lane, or modelruntime. Search results are evidence candidates. FORGE-K remains simulator and shadow validation.

## Goal

Big-brain search is FORGE's trust-tiered retrieval coordinator. It unifies local workspace inspection, canonical memory reads, official documentation, curated project docs, web search, embeddings, vector recall, and historical audit references into one explainable candidate set.

It does not make results true. It creates evidence candidates with provenance, freshness, trust tier, scope, and rejection metadata.

## Ranking Inputs

| Signal | Purpose |
|---|---|
| Scope match | Prefer same workspace, lane, project path, and operator intent. |
| Freshness | Prefer current local files, recent official documentation, and recent canonical events over old snapshots. |
| Trust tier | Prefer local live and official documentation over web/vector recall. |
| Source authority | Prefer governed FORGE stores over unaudited transient text. |
| Citation quality | Prefer exact source refs with line, object, route, event, or artifact references. |
| Contradiction status | Preserve conflict evidence; do not silently collapse disagreements. |
| Rejection reason | Record why candidates lost ranking or were excluded. |

## Candidate Lifecycle

1. Expand the query across local context, known workspace refs, memory refs, docs, search providers, and vector indexes.
2. Normalize candidates into a common packet shape.
3. Assign `TrustTier`, freshness, scope, and source authority metadata.
4. Contract the set with budget, policy, stale-source, and duplicate-source rules.
5. Emit selected and rejected candidates.
6. Pass only the packet to existing context or modelruntime paths.

## Trust-Tiered Defaults

| Trust tier | Examples | Default use |
|---|---|---|
| `local_live` | Current workspace files, live API route tables, active FORGE DB reads | Primary source for current local behavior. |
| `official` | Official provider docs, language specs, package docs | Primary source for external current facts. |
| `curated` | Project docs, ADRs, status docs, reviews | Strong evidence when in scope and not stale. |
| `web` | General search results | Discovery and corroboration only. |
| `vector_recall` | Embedding and VSA matches | Recall assistance, not authority. |
| `low_trust` | Unverified snippets, stale summaries, weak matches | Operator review or contradiction prompts only. |

## Freshness Rules

- Live local files outrank old generated summaries of the same files.
- Current official documentation outranks stale blog posts for current API claims.
- Canonical memory and journal refs outrank vector recall of memory-like text.
- Stale memory can be shown as historical context, but it must not be selected as current fact when newer scoped evidence exists.

## Rejected Candidates

Every packet should preserve rejected candidates with a bounded reason:

- stale
- out of workspace scope
- lower trust duplicate
- contradicted by fresher source
- weak semantic match
- unsupported source
- budget contraction

Rejected candidates are useful for audit, debugging, and operator review. They are not hidden model context unless a route explicitly asks to inspect them.
