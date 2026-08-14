# Restore Outcome Feedback

Status date: 2026-08-14 (K20G).

Restore outcome feedback records whether restored context helped downstream work. It is non-canonical evidence only.

## What Gets Recorded

Persistent `COMPILE_CONTEXT` runs create original `restore_outcome_events` with:

- selected context packet and source snapshot
- selected evidence/state/loop/artifact refs
- restore score and `requires_fresh_compile`
- outcome, confidence, feedback, correction summary
- downstream action/object and trace/correlation/syscall/audit linkage

Initial outcomes are `unknown`, `no_candidate`, or `fresh_compile_required`.

## Operator Feedback

Use:

`POST /api/context/restore/outcomes/{id}/feedback`

The body provides:

- `outcome`
- `outcomeConfidence`
- `operatorFeedback`
- `correctionSummary`
- `metadata`

The request must include exact `workspaceId`, `laneId`, `selectedPaths`, and
`idempotencyKey`. Production FORGE-K validates these against the source context
snapshot and appends an immutable feedback event. Wrong-scope, legacy-unbound,
adapter/Future IRIS/model-proposer, and `legacy_v1` requests fail closed.

The source `restore_outcome_events` row is never updated. List/get responses
return the original as `outcome` and the rebuildable noncanonical view as
`feedbackProjection`. The projection cannot overwrite or conceal the original.

## Safety

- Feedback does not edit canonical memory/state/loops.
- Feedback event, projection, provenance, journal transition, immutable audit
  intent, and idempotency proof commit atomically; journal failure rolls all of
  them back.
- Feedback cannot override kernel validation, approval, capability, scope, gateway, or modelruntime policy.
- Restore scoring consumes outcomes only as capped utility evidence.
- Dream Mode consumes outcomes only as dry-run proposals.

## Backup

Full backup includes `restore_outcome_events` for inspection. Live restore
apply remains disabled; utility event/projection recovery requires the future
daemon-stopped whole-store recovery contract rather than row-level merge.
