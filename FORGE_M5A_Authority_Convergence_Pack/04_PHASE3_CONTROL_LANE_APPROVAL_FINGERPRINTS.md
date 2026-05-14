# Phase 3 Prompt — Control Lane Approval Fingerprint Seam

You are working inside `BiteTh3bulleT/project_forge`.

Do not output code in chat. Make changes in files.

## Goal

Create a deterministic approval fingerprint model for Control Lane semantic mutation.

## Required design doc

Create/update:

- `docs/architecture/control_lane_approval_fingerprints.md`

Define fingerprint purpose, fields, lifecycle, durable approvals integration, dry-run handling, system-internal validation handling, autonomy proposal handling, and future FORGE-K migration handling.

## Required fields

Version, semantic action, capability, target object type, mutating flag, actor, source, workspace, trace/correlation if used, payload shape hash, safe target identifiers, risk class, approval request id, decision status, expiry/created timestamp if verifying.

## Code deliverable

Implement a pure deterministic helper if clean. No DB calls in fingerprint builder.

## Tests

Prove deterministic output, stable map ordering, changes on action/source/actor/workspace/payload shape, and validation-only action representation.

## WHAT NOT TO DO

Do not break existing Control Lane tests. Do not let “system” become implicit approval. Do not add a second approvals DB. Do not store raw secrets or raw prompt bodies in fingerprints.
