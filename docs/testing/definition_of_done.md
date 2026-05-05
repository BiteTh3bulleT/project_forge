# Definition of Done

Status: Phase 0 baseline.

FORGE-K work is done only when the implementation, documentation, tests, and evidence preserve kernel-first authority.

## Baseline Expectations

- Docs are updated for architecture changes.
- Tests are added where code changes exist.
- Semantic syscalls are journaled.
- No direct canonical mutation bypasses semantic syscalls.
- No unvalidated model output receives authority.
- Provenance is not destroyed.
- Snapshot-as-truth behavior is not introduced.
- KV-as-memory behavior is not introduced.
- Rejected evidence records rejection reasons.
- Superseded objects remain inspectable.
- Contradictions are recorded instead of silently merged.
- Runtime drivers remain isolated from Kernel authority.
- Research/tooling phases remain isolated from live daemon authority.

## Phase Evidence

Every phase report should include:

- commands run
- validation results
- fixture and golden corpus drift evidence when shared validation fixtures change
- CI/workflow evidence when tooling integration changes
- files changed
- unresolved blockers
- known risks
- next recommended phase
