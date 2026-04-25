# Context Restore And Snapshots

## Phase 6.25 State

PARTIAL / MOSTLY IMPLEMENTED: `COMPILE_CONTEXT` can persist context snapshots, render optional SVG cards, score restore candidates, return header-first restore packages, persist `restore_scores_json`, persist `resume_hints_json`, and mark `requires_fresh_compile`.

Evidence: `compile_context_snapshot.go`, `compile_context_restore_scoring.go`, `compile_context_snapshot_svg.go`, `context_packet_snapshots` migration, and tests.

## Snapshot Semantics

Snapshots are non-canonical restore evidence. They describe what context was assembled, why it was selected, what score it received, and whether a fresh compile is needed. They do not replace notes, state, journal, or provenance.

## Score Fields

Scoring records query/scope/kind, recency, lineage, state overlap, open-loop overlap, artifact overlap, contradiction penalty, staleness, confidence, freshness, score threshold, selected candidate, and fresh-compile requirement.

## Header-First Restore

PARTIAL: Header-first restore packages are intended to make restore cheaper than loading full graph context. Current implementation exists, but review notes flag scalability/operator-inspection work.

## Correctness Risks

- PARTIAL: Candidate listing can over-filter by exact query before scoring.
- PARTIAL: Operator inspectability is limited.
- PARTIAL: SVG/card storage needs scalability policy.
- MISSING: Restore outcome feedback loop is not implemented.

