# Context Restore Review

## Scorecard

- `COMPILE_CONTEXT` compatibility: GOOD
- Snapshot persistence: GOOD
- Deterministic scoring: GOOD/PARTIAL
- Candidate listing: RISK
- Header-first package: GOOD/PARTIAL
- Operator inspectability: GOOD/PARTIAL
- Performance/index posture: PARTIAL

## Findings

GOOD: Restore scoring is deterministic, CPU-only, and non-LLM. It stores `restore_scores_json`, `resume_hints_json`, restore trace/package metadata, and non-canonical snapshot evidence.

GOOD: Operator routes exist for recent restore snapshots, detail, candidates, score, and resume hints.

RISK: SQLite candidate listing filters by exact `query = ?` before lexical scoring. This excludes near-match candidates before `scoreRestoreCandidate` can rank them.

RISK: Existing indexes do not fully match the intended restore candidate access pattern if exact-query filtering is removed.

PARTIAL: Generic snapshot detail route allows optional scope; strict restore routes require workspace. Desktop generic snapshot detail should pass workspace/lane or use strict restore routes for restore views.

MISSING: Restore outcome feedback loop is not present.

## Punchlist

- `CTX-001`: Change candidate listing to fetch by workspace/lane/kind/recency and score query similarity in Go.
- `CTX-002`: Add composite index for restore candidate scans.
- `CTX-003`: Add tests for near-match candidates and wrong-workspace exclusion.
- `CTX-004`: Pass workspace/lane into desktop snapshot detail calls.
- `CTX-005`: Add restore outcome feedback table or report-only evidence path.
- `CTX-006`: Add benchmark/fixture for large snapshot sets.

