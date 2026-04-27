# Restore Outcome Feedback

Status date: 2026-04-27.

Restore outcome feedback records whether restored context helped downstream work. It is non-canonical evidence only.

## What Gets Recorded

Persistent `COMPILE_CONTEXT` runs create `restore_outcome_events` with:

- selected context packet and source snapshot
- selected evidence/state/loop/artifact refs
- restore score and `requires_fresh_compile`
- outcome, confidence, feedback, correction summary
- downstream action/object and trace/correlation/syscall/audit linkage

Initial outcomes are `unknown`, `no_candidate`, or `fresh_compile_required`.

## Operator Feedback

Use:

`POST /api/context/restore/outcomes/{id}/feedback`

Allowed update fields:

- `outcome`
- `outcomeConfidence`
- `operatorFeedback`
- `correctionSummary`
- `metadata`

The request must include `workspaceId`; `laneId` is optional. Wrong-workspace reads or updates are rejected.

## Safety

- Feedback does not edit canonical memory/state/loops.
- Feedback cannot override kernel validation, approval, capability, scope, gateway, or modelruntime policy.
- Restore scoring consumes outcomes only as capped utility evidence.
- Dream Mode consumes outcomes only as dry-run proposals.

## Backup

Full backup includes `restore_outcome_events`. Restore imports the table after context snapshots so feedback remains linked to the selected restore evidence.
