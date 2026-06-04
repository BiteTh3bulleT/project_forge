# Context Compiler Search Index

Status: proposed internal architecture.
Date: 2026-06-04.

## Authority Banner

AXIOM is an internal FORGE cognition, search, and context layer. It does not execute tools, does not write canonical memory, and does not bypass Gateway, approvals, audit, Control Lane, or modelruntime. Search results are evidence candidates. FORGE-K remains simulator and shadow validation.

## Purpose

The context compiler search index is a non-canonical index over source refs, search evidence packets, context packets, rejected candidate records, and citation metadata. It accelerates lookup and explainability for COMPILE_CONTEXT-like flows without becoming truth.

The index is shape, not truth.

## Indexed Records

| Record | Indexed fields | Authority |
|---|---|---|
| Source refs | workspace, object type, path, object id, freshness | Pointer only |
| Search evidence packets | query, routing mode, candidate refs, trust tiers | Evidence candidate summary |
| Context packets | selected refs, rejected refs, token budget, citations | Prompt-support shape |
| Rejected candidate | source ref, rejection reason, replaced-by ref | Audit/explainability |
| Citation spans | exact source ref and span/object | Evidence pointer |

## Non-Canonical Index Rules

- The index may be rebuilt from source records.
- The index may be stale; consumers must verify source refs before answer-critical use.
- The index cannot admit evidence, commit memory, update canonical state, or approve execution.
- A missing index entry must not delete source truth.
- A matching index entry must not bypass current policy.

## Relationship To COMPILE_CONTEXT

Live `COMPILE_CONTEXT` authority remains in the existing AI-OS and Control Lane path. FORGE-K Context Compiler remains simulator-only unless later promoted through a separate live-integration ADR and tests.

AXIOM can provide candidate source refs and context-shape metadata to live context assembly, but live selection remains governed by existing FORGE policy, scope, and audit rules.

## Rejected Candidate Records

Rejected candidate records are first-class index entries. They explain contraction choices, stale-source exclusion, trust-tier downgrades, and source conflicts. They also make later evaluation possible:

- Was a useful source rejected because of a stale timestamp?
- Did vector recall surface an unsupported candidate?
- Did official documentation replace a web result?
- Did local live workspace evidence replace old memory?

## Storage Direction

Start with existing FORGE stores and package boundaries. Do not add a second vectorstore, memory database, approval log, audit log, or modelruntime registry for AXIOM.
