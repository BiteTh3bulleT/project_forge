# Context Restore Review

## Current State

GOOD: Context snapshot persistence is implemented in `context_packet_snapshots`.

GOOD: Deterministic restore scoring exists with explainable score components.

GOOD: Snapshot APIs exist under `/api/context-inspector/snapshots`.

GOOD: Restore scoring does not require LLM/modelruntime/GPU.

PARTIAL: `COMPILE_CONTEXT` still emits a warning path that says deterministic Phase 2 stub in `processor_apply.go`. This should be reconciled with newer snapshot/scoring behavior so operators are not confused.

## Correctness Risks

RISK: SQLite candidate listing over-filters by exact query before scoring.

- Evidence: `applyCompileContext` passes packet query into `ListContextSnapshots`; SQLite uses `query = ?`.
- Impact: partial/lexical ranking logic is bypassed for candidates whose query is related but not exactly equal.
- Fix: list by workspace/lane/kind/recency first, then let deterministic scorer rank query similarity.

RISK: Fresh compile fallback and `requires_fresh_compile` need more operator-visible explanation. The data exists, but UI should show why a candidate was rejected.

RISK: Restore outcome feedback is missing. The system scores candidates, but does not yet learn from whether the restored package was useful.

## Performance Risks

PARTIAL: Header-first restore exists conceptually and partially in code, but this review did not prove it avoids full graph load under realistic large snapshots.

RISK: SVG/card snapshot storage may become expensive if every compile stores rich artifacts without retention policy.

## Needed Tests

- SQLite integration test for partial-query candidate ranking.
- Wrong-workspace exclusion test.
- Header-first load budget test.
- Stale/contradictory candidate penalty tests using SQLite, not only in-memory scorer.
- Fresh compile fallback trace test.
- Restore feedback persistence test once implemented.

## Next Upgrades

1. Fix candidate retrieval to allow scoring to do the ranking.
2. Add restore usefulness feedback signals.
3. Add retention/compaction policy for snapshot cards/SVGs.
4. Expand Inspectors UI to show candidate list, penalties, winner, fallback reason, selected evidence, and resume hints in one trace.

