# K20C FORGE-K Courthouse Cutover

Status: `[LIVE / PARTIAL FULL-KERNEL CUTOVER]`

Date: 2026-08-14

## Outcome

Production FORGE-K is the sole deterministic decision owner for
`ADMIT_EVIDENCE` and `APPEAL_RULING`. The simulator Courthouse remains
isolated. The durable adapter can persist a typed Kernel decision but cannot
create one, and `legacy_v1` fails closed for both mutations.

An initial decision atomically writes the exhibit current state, an immutable
ruling, provenance, and the semantic journal event. An appeal atomically adds
an immutable appeal and new ruling while advancing the exhibit's current
ruling pointer. Prior rulings remain queryable and cannot be updated or
deleted. Repository reads require workspace/lane scope.

Models and neural actors cannot rule. Adapter and Future IRIS proposer sources
lack Courthouse mutation capabilities. Admission requires stable source refs,
policy refs, and a valid lowercase SHA-256 content identity; incomplete policy
material produces a persisted deterministic rejection.

## Validation

- `cd services/core && go test ./...`
- focused production Kernel, Courthouse, Control Lane, and store tests
- focused Go vet
- `git diff --check`

Covered behavior includes deterministic decisions, admission/rejection,
immutable appeal history, current-versus-historical separation, workspace/lane
isolation, proposer/model/legacy denial, prior-current concurrency checks, and
full transaction rollback when the journal append fails.

## Remaining blocker

Audit sink persistence and `audit_id` linkage still happen after the canonical
transaction and are best-effort. K20C does not claim atomic audit evidence.
K20D must add a sealed prepare plan, typed commit receipt, durable audit outbox,
immutable idempotency proof, persisted journal hash chain, and replay/divergence
verification before broader subsystem cutover.
