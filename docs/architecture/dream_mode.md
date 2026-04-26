# Dream Mode v0

Status date: 2026-04-24.

Dream Mode v0 is FORGE's first deterministic replay/consolidation engine. It is CPU-first, dry-run by default, and proposal-only.

## Authority

- Dream Mode reads existing cognitive filesystem tables.
- It does not create a second memory database.
- It does not call LLMs, modelruntime, vector retrieval, GPU jobs, adapters, voice, vision, or GUI systems.
- It does not commit canonical truth in v0.
- Any future commit mode must use semantic syscalls/control lane validation.

## Replay Selector

The selector gathers recent candidates from:

- `journal_events`
- `context_packet_snapshots`
- `memory_notes`
- `state_items`
- `open_loops`
- `contradiction_records`
- `artifact_refs`

Candidates include source IDs, scope, timestamp range, content summary, tags, related loop/snapshot IDs, raw importance signals, and trace fields.

## Salience

Salience is deterministic:

- novelty score
- repetition score
- goal relevance score
- correction value score
- outcome impact score
- contradiction score
- retrieval utility score
- recency score

User corrections, unresolved contradictions, active blockers, repeated failures, failed restores, and recent important events receive visible score components and explanation strings.

## Tier Routing

Dream Mode v0 proposes one of:

- `retain_short_term`
- `promote_mid_term`
- `promote_long_term`
- `demote`
- `merge`
- `discard`
- `needs_review`
- `repair_required`
- `no_op`

Long-term promotion requires high salience, high confidence, low contradiction risk, explicit long-term allowance, and no operator-review requirement. Unresolved contradictions route to review or repair rather than long-term truth.

## Operating Depths

- `microdream`: short window, low candidate limit, corrections/contradictions/active loops/failed restores
- `nap`: day-scale window, mid-term promotion and snapshot hygiene proposals
- `deep_dream`: longer window and larger candidate set, long-term candidates and repair proposals, still CPU-only in v0

## Report Persistence

The dry-run report returns run metadata, candidates, salience scores, tier routing proposals, memory action proposals, snapshot hygiene proposals, restore score update proposals, repair proposals, review items, no-op reasons, warnings, and trace.

Reports are non-canonical evidence. They are not memory truth.

Persistence is explicit. `/api/dream/run` accepts `persistReport`; when false or omitted, the report is returned only. When true, FORGE writes one `dream_reports` row containing the dry-run report as non-canonical evidence:

- candidates and salience scores
- memory-tier proposals
- repair proposals
- snapshot hygiene proposals
- warnings and trace
- correlation/trace IDs
- proposed-by metadata

Persisting a Dream report does not promote, demote, merge, delete, repair, or otherwise mutate canonical memory/state. Future commit/apply behavior must be a separate governed semantic syscall/control-lane path.

## Operator Inspector API

Status: IMPLEMENTED.

Persisted reports are inspectable through read-only, workspace-scoped routes:

- `GET /api/dream/reports?workspaceId=<id>&laneId=<lane>&mode=nap&limit=20`
- `GET /api/dream/reports/<report-id>?workspaceId=<id>&laneId=<lane>`
- `GET /api/dream/reports/<report-id>/candidates?workspaceId=<id>&laneId=<lane>`
- `GET /api/dream/reports/<report-id>/proposals?workspaceId=<id>&laneId=<lane>`
- `GET /api/dream/reports/<report-id>/warnings?workspaceId=<id>&laneId=<lane>`

Each response marks the report as `non_canonical_evidence`, preserves `dryRun`, and reports
`canonicalWriteCommitted=false`. Wrong-workspace report IDs return not found. Inspector routes do
not call modelruntime/GPU and do not apply proposals.
