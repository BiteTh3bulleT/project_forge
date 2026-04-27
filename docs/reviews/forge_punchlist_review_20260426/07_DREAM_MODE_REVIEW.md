# Dream Mode Review

## Scorecard

- Replay selector: GOOD/PARTIAL
- Salience scoring: GOOD
- Tier-routing proposals: GOOD
- Dry-run default: GOOD
- Report persistence: GOOD/PARTIAL
- Backup inclusion: GOOD
- Operator inspectability: GOOD/PARTIAL
- Commit boundary: GOOD

## Findings

GOOD: Dream Mode v0 reads existing cognitive filesystem tables and produces deterministic replay/salience/routing proposals without modelruntime/GPU dependency.

GOOD: `/api/dream/run` remains dry-run/proposal-first. `persistReport=true` stores a non-canonical `dream_reports` row.

GOOD: Read-only report list/get/candidates/proposals/warnings routes exist and are workspace scoped.

GOOD: Backup includes `dream_reports`.

PARTIAL: Persistence is opt-in; if operators do not request `persistReport=true`, reports remain transient.

PARTIAL: Same report ID persistence is upsert-style. Decide whether this is desired deterministic replacement or an evidence immutability issue.

MISSING: No Dream proposal apply/commit path exists. This is correct now; future commit mode must be syscall-bound.

MISSING: Dream report outputs do not yet feed restore scoring outcome feedback.

## Punchlist

- `DRM-001`: Decide and document Dream report upsert vs append-only behavior.
- `DRM-002`: Add operator workflow for report review without apply.
- `DRM-003`: Add future governed Dream apply syscall design only after evidence review UI stabilizes.
- `DRM-004`: Add restore feedback signals from Dream reports as non-canonical evidence.
- `DRM-005`: Add test proving Dream runs with modelruntime/GPU unavailable.

