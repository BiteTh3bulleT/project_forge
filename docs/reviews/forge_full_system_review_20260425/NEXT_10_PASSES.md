# Next 10 Implementation Passes

## Pass 1 - Approval Fingerprint Hardening

Bind gateway approvals to request fingerprints and reject replay.

Acceptance: tests prove an approval for one tool/path/risk cannot authorize another.

## Pass 2 - Model Management Governance

Route model management through gateway capabilities or equivalent approval gates.

Acceptance: import/archive/remove/load/unload have explicit risk, approval, audit, and trace.

## Pass 3 - Backup/Restore Parity and Integrity

Add retrieval/observation backup sections and bundle verification.

Acceptance: tampered bundle fails; full backup preserves retrieval provenance.

## Pass 4 - Context Restore Candidate Fix

Stop exact-query prefiltering from neutering scoring.

Acceptance: SQLite tests show partial-query candidates considered, scored, and explained.

## Pass 5 - Dream Mode Evidence Persistence

Persist Dream reports/proposals as non-canonical evidence.

Acceptance: `/api/dream/run` returns durable report id and inspector can retrieve it.

## Pass 6 - Public Semantic Syscall API

Add narrow operator-facing syscall dry-run/submit/inspect route.

Acceptance: no legacy memory mutation resurrected; commits still go through kernel.

## Pass 7 - Operator Trace Workbench

Unify audit trace, gateway invocations, context snapshots, artifacts, and model calls.

Acceptance: one correlation id tells the full story.

## Pass 8 - Cross-Platform Operations Scripts

Replace Bash-only smoke/desktop helpers with Node wrappers.

Acceptance: Windows `npm run smoke` either runs or fails for runtime reasons, not missing Bash.

## Pass 9 - Frontend Test Foundation

Add Vitest/RTL and Playwright smoke.

Acceptance: CI runs UI tests for Models, Inspectors, Gateway, Project Context, and Dream empty/report state.

## Pass 10 - Provider and Safe Mode Security

Add provider URL allowlists, secret redaction tests, and `/v1/*` exposure hardening.

Acceptance: unsafe endpoints are rejected by default; cloud/provider secrets never appear in audit/errors.

