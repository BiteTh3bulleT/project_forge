# Context Restore Scoring (Arterial Runtime)

Status date: 2026-04-23.

## Scope

This document covers deterministic restore candidate scoring used by `COMPILE_CONTEXT` inside the semantic syscall control lane.  
It does not introduce a new memory subsystem and does not change truth authority.

## Entrypoint and authority

- Entry action: `COMPILE_CONTEXT`
- Authority boundary: syscall validator + processor + transactional semantic store
- Persistence target: `context_packet_snapshots` evidence rows
- Canonical truth remains notes/links/state/loops/models/artifacts/journal, not snapshot rows

## Candidate listing

Restore candidates are listed by:

- `workspace_id` / `lane_id`
- exact `query`
- `snapshotKind`
- deterministic ordering by `created_at DESC` (and stable ID tiebreak)

The listing API is available on semantic read stores as:

- `ListContextSnapshots(scope, query, snapshotKind, limit)`

## Deterministic scoring model

Each candidate receives a deterministic score with explicit components.

Base components:

- query match
- snapshot kind match
- node overlap (Jaccard over graph node IDs)
- edge overlap (Jaccard over graph edge IDs)
- recency score (half-life decay)
- fingerprint match bonus
- preferred snapshot hint bonus (if requested)

Penalties:

- staleness penalty
- contradiction density penalty (conflict-marked graph nodes)
- header-only penalty (when graph body is unavailable)

Decision:

- select candidate when top score >= threshold
- otherwise fallback to fresh compile
- explicit fallback also occurs when no candidates or `freshCompileOnly=true`

## Resume hints contract

Optional request hints:

- `preferredSnapshotId`
- `minimumScore` (0..1)
- `freshCompileOnly`

Hints may be passed at:

- `payload.resumeHints`
- `payload.restoreSnapshot.resumeHints`
- `payload.compileOptions.resumeHints`

## Persisted inspectability fields

When `persistSnapshot=true`, restore metadata persists:

- `restore_scores_json`:
  - decision
  - threshold
  - candidate count
  - selected snapshot ID
  - scored candidate rows
- `resume_hints_json`:
  - preferred snapshot recommendation
  - threshold
  - fresh compile recommendation
  - top candidate metadata

These fields are evidence for operator inspectability and replay/debug, not truth authority.
