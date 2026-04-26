# Cognitive Filesystem Data Model (Phase 3)

FORGE cognitive filesystem is durable AI-OS semantic storage, not chat-local memory.

Kernel rule remains:

**LLMs/agents/IRIS propose. FORGE validates. FORGE commits.**

## Model goals

- durable, inspectable semantic objects
- deterministic syscall commit path
- explicit workspace/scope isolation
- current truth separate from historical truth
- supersession/contradiction preserve evidence rather than delete
- correlation/audit/provenance traceability on persisted objects

## Persistent objects

## 1) `journal_events`

Purpose:

- append-only raw semantic/system journal for committed syscall activity

Canonical fields:

- `id`, `type`, `source`, `actor`
- `workspace_id`, `lane_id`, `selected_paths_json`
- `payload_json`
- `correlation_id`, `trace_id`
- `provenance_id`, `provenance_json`
- `created_at`, `metadata_json`
- `proposed_by`, `committed_by`, `syscall_id`, `audit_id`

Mutability:

- append-only (DB triggers reject update/delete)

Indexes/query patterns:

- by `workspace_id`, `created_at`
- by `correlation_id`
- by `trace_id`

Invariants:

- immutable truth trail
- no hard delete/update in normal operation

## 2) `memory_notes`

Purpose:

- canonical semantic note records (facts/preferences/goals/decisions/etc.)

Canonical fields:

- `id`, `type`, `title`, `content`
- `workspace_id`, `lane_id`, `selected_paths_json`
- `confidence`, `status`
- `provenance_id`, `provenance_json`
- `created_at`, `updated_at`, `archived_at`, `superseded_by`
- `metadata_json`
- `proposed_by`, `committed_by`, `syscall_id`, `correlation_id`, `trace_id`, `audit_id`

Allowed statuses:

- `active`, `superseded`, `archived`

Mutability:

- status/timestamps may update through syscall path only
- no hard delete in normal path

Indexes/query patterns:

- by workspace+status
- by workspace+type
- by trace/correlation via linkage columns

## 3) `semantic_links`

Purpose:

- typed relationships between semantic objects

Canonical fields:

- `id`, `type`
- `source_id`, `source_kind`, `target_id`, `target_kind`
- `confidence`
- `workspace_id`, `lane_id`, `selected_paths_json`
- `provenance_id`, `provenance_json`
- `created_at`, `metadata_json`
- trace columns (`proposed_by`, `committed_by`, `syscall_id`, `correlation_id`, `trace_id`, `audit_id`)

Allowed types:

- `relates_to`, `supports`, `contradicts`, `supersedes`, `depends_on`, `causes`, `about`, `derived_from`, `blocks`, `resolves`

Indexes/query patterns:

- by source id
- by target id
- by type/scope
- neighborhood expansion by source/target joins

## 4) `state_items` (current projection)

Purpose:

- current value projection by semantic key + scope

Canonical fields:

- `id`, `key`, `value_json`
- `workspace_id`, `lane_id`, `selected_paths_json`
- `status`, `derived_from_json`
- `current_version`
- `updated_at`, `metadata_json`
- trace columns (`proposed_by`, `committed_by`, `syscall_id`, `correlation_id`, `trace_id`, `audit_id`)

Mutability:

- upsert current value through syscall path
- must preserve previous values in `state_versions`

Invariants:

- uniqueness on `(workspace_id, lane_id, key)`
- current query is explicit (`state_items`)

## 5) `state_versions` (history)

Purpose:

- state timeline / historical reconstruction

Canonical fields:

- `id`, `state_item_id`, `state_key`
- `workspace_id`, `lane_id`
- `previous_value_json`, `new_value_json`
- `changed_by`, `derived_from_json`
- `syscall_id`, `audit_id`, `correlation_id`, `trace_id`
- `proposed_by`, `committed_by`
- `created_at`, `metadata_json`

Mutability:

- append-only history rows

Indexes/query patterns:

- by `workspace_id + state_key + id`
- by correlation id

## 6) `open_loops`

Purpose:

- durable loop lifecycle (open work, blockers, resolution states)

Canonical fields:

- `id`, `title`, `state`, `priority`, `owner`, `blocker`, `next_action`
- `related_notes_json`, `created_from`
- `workspace_id`, `lane_id`, `selected_paths_json`
- `created_at`, `updated_at`, `resolved_at`, `archived_at`
- `metadata_json`
- trace columns (`proposed_by`, `committed_by`, `syscall_id`, `correlation_id`, `trace_id`, `audit_id`)

Allowed states:

- `open`, `in_progress`, `blocked`, `resolved`, `archived`

Mutability:

- lifecycle updates only through syscall transitions

## 7) `artifact_refs`

Purpose:

- semantic artifact references used as evidence links (not binary storage itself)

Canonical fields:

- `id`, `type`, `uri`, `content_hash`
- `workspace_id`, `lane_id`, `selected_paths_json`
- `provenance_id`, `provenance_json`
- `created_at`, `metadata_json`
- trace columns

Mutability:

- append-only references in normal flow

Indexes/query patterns:

- by scope
- by checksum/hash

## 8) `derived_models`

Purpose:

- adaptive/derived model records (non-canonical truth layer)

Canonical fields:

- `id`, `type`, `expression_json`
- `derived_from_json`, `support_count`, `confidence`, `status`
- `workspace_id`, `lane_id`, `selected_paths_json`
- `last_validated_at`, `created_at`, `updated_at`
- `metadata_json`
- trace columns

Allowed statuses:

- `provisional`, `promoted`, `deprecated`

Mutability:

- status and validation metadata may update through syscall path
- derived model never overwrites evidence rows

## 9) `provenance_records`

Purpose:

- normalize actor/source/trace lineage and reference from semantic objects

Canonical fields:

- `id`, `actor`, `actor_type`, `source`, `trace_id`
- `workspace_id`, `lane_id`, `selected_paths_json`
- `metadata_json`, `created_at`
- trace columns (`proposed_by`, `committed_by`, `syscall_id`, `correlation_id`, `audit_id`)

Mutability:

- effectively append/insert-only (id-based dedupe)

## 10) `contradiction_records`

Purpose:

- durable record that two objects disagree

Canonical fields:

- `id`
- `left_object_id`, `left_object_kind`
- `right_object_id`, `right_object_kind`
- `reason`, `severity`, `confidence`
- `provenance_id`, `provenance_json`
- `workspace_id`, `lane_id`, `created_at`
- `metadata_json`
- trace columns

Mutability:

- append-only in normal flow

Invariants:

- contradiction never deletes either side
- usually paired with `semantic_links.type = contradicts`

## 11) `supersession_records`

Purpose:

- durable lineage that newer object supersedes older object

Canonical fields:

- `id`
- `old_object_id`, `old_object_kind`
- `new_object_id`, `new_object_kind`
- `reason`
- `provenance_id`, `provenance_json`
- `workspace_id`, `lane_id`, `created_at`
- `metadata_json`
- trace columns

Mutability:

- append-only in normal flow

Invariants:

- old record preserved
- often paired with `semantic_links.type = supersedes`

## 12) `context_packet_snapshots`

Purpose:

- evidence snapshot of context assembly inputs/choices
- Phase 6.25 restore evidence for `COMPILE_CONTEXT` snapshots, including optional SVG card rendering metadata

Canonical fields:

- `id`, `query`
- `workspace_id`, `lane_id`, `selected_paths_json`
- included ids JSON:
  - `included_state_json`
  - `included_open_loops_json`
  - `included_notes_json`
  - `included_links_json`
  - `included_models_json`
  - `included_artifacts_json`
  - `included_events_json`
- `budget_json`, `inclusion_reasons_json`
- `created_at`, `correlation_id`, `trace_id`, `syscall_id`
- `metadata_json`
- trace columns + `audit_id`

Mutability:

- append-only snapshot evidence

Non-canonical:

- snapshot rows and any rendered SVG card are evidence of load composition, not truth authority.
- restore metadata may describe how the snapshot was rehydrated, but it does not override canonical notes/state/history.

Phase 6.25 restore fields:

- `snapshot_kind`: classifies the snapshot intent for restore/review handling
- `snapshot_fingerprint`: deterministic semantic fingerprint of the snapshot graph
- `parent_snapshot_id`: lineage edge to the selected predecessor snapshot
- `restore_scores_json`: deterministic candidate ranking + score breakdown for restore selection
- `render_artifact_ref_id`: optional reference to the SVG card produced when snapshot cards are requested
- `resume_hints_json`: deterministic hints package for follow-up compile/restore requests

Additional restore contract fields are persisted inside `metadata_json` rather than as dedicated columns:

- `restore_source_snapshot_id`: links a restore back to the source snapshot row
- `restore_scope_json`: records the workspace/lane/path scope used during restore
- `restore_reason_json`: captures why the snapshot was restored or rendered
- `restore_trace_json`: inspectable selection trace for operator/debug views
- `restore_package_json`: header-first restore package containing selected header, compact evidence refs, score breakdown, resume hints, candidate summaries, and trace
- `rendered_card_artifact_id`: metadata alias populated from `render_artifact_ref_id`

Purpose of the restore fields:

- keep restore activity explainable and auditable
- preserve deterministic scope boundaries during rehydration
- make compiled context inspectable without promoting it to canonical truth

Restore scoring is deterministic and lexical/scoped. It records query, scope, snapshot kind, recency, lineage, state overlap, loop overlap, artifact overlap, contradiction, staleness, freshness, confidence, and `requires_fresh_compile` fields. `requires_fresh_compile` means the current candidates are absent, below threshold, forced fresh by hints, or hard-stale.

## 12.5) Dream Mode v0 reports

Dream Mode v0 reads journal, snapshot, note, state, loop, contradiction, and artifact tables to produce a dry-run consolidation report.

The report is not canonical truth and is not persisted as canonical memory in v0. When `persistReport=true` is supplied to `/api/dream/run`, the report is stored as non-canonical evidence in `dream_reports`.

`dream_reports` stores:

- `id`, `created_at`, `completed_at`
- `workspace_id`, `lane_id`, `mode`, `dry_run`, `status`
- `time_window_start`, `time_window_end`
- `candidates_considered`, `proposals_generated`
- `summary_json`
- `candidates_json`
- `salience_scores_json`
- `memory_tier_proposals_json`
- `repair_proposals_json`
- `snapshot_hygiene_proposals_json`
- `warnings_json`
- `trace_json`
- `correlation_id`, `trace_id`, nullable `syscall_id`, nullable `audit_id`
- `proposed_by`, `committed_by`, `metadata_json`

These rows are report/evidence records only. `committed_by` remains empty unless a future governed evidence pipeline explicitly records otherwise; it must not be interpreted as canonical memory commit.

The report contains:

- run metadata and trace
- replay candidates
- deterministic salience scores
- proposed memory tier routing
- proposed snapshot hygiene actions
- proposed restore score updates
- proposed repair/review actions
- no-op reasons and warnings

Any later governed commit mode must turn proposals into semantic syscalls and pass control-lane validation.

`dream_reports` is included in `full_backup` export/restore and is classified as `non_canonical_evidence`.

Read-only inspector routes expose these rows for operator review:

- restore snapshots through `/api/context/restore/*`
- Dream reports through `/api/dream/reports/*`

The inspector API must preserve workspace/lane scope and must not apply Dream proposals or mutate canonical rows.

Provider note:

- TEI and other embedding providers may populate retrieval vector records.
- Vector records are retrieval indexes/evidence, not canonical truth.
- Provider health/capability records do not authorize semantic mutation.

## 13) Audit linkage

All major cognitive filesystem objects include:

- `syscall_id`
- `correlation_id`
- `trace_id`
- `audit_id`
- `proposed_by`
- `committed_by`

Processor updates `audit_id` after audit write via linkage update.

## 14) Workspace/scope fields

Semantic entities store:

- `workspace_id`
- `lane_id`
- `selected_paths_json` where applicable

Queries are scope-aware by default in repository methods.

## 15) Correlation/trace fields

Correlation and trace are persisted on semantic rows to support:

- end-to-end syscall tracing
- audit joins
- replay/debug workflows

## Relation mapping

Textual graph:

- `MemoryNote -> provenance_records` (`provenance_id`)
- `MemoryNote -> semantic_links -> MemoryNote|State|Loop|Model|Artifact`
- `StateItem -> state_versions` (history)
- `StateItem <- derived_from_json` references note/event ids
- `OpenLoop -> related_notes_json` and `created_from`
- `DerivedModel -> derived_from_json` evidence ids
- `ContradictionRecord -> left/right object ids`
- `SupersessionRecord -> old/new object ids`
- `ArtifactRef -> provenance + uri/hash evidence`
- `JournalEvent` is append-only raw semantic truth layer

## Syscall mapping

- `CREATE_NOTE` -> `memory_notes` (+ journal event)
- `CREATE_LINK` -> `semantic_links` (+ journal event)
- `UPDATE_STATE` -> upsert `state_items` + append `state_versions` (+ journal event)
- `OPEN_LOOP` / `CLOSE_LOOP` -> `open_loops` (+ journal event)
- `MARK_SUPERSEDED` -> `supersession_records` + `semantic_links(supersedes)` + note status transition (+ journal event)
- `REGISTER_CONTRADICTION` -> `contradiction_records` + `semantic_links(contradicts)` (+ journal event)
- `DERIVE_MODEL` -> `derived_models` (+ journal event)
- `ARCHIVE_NOTE` -> status transition in `memory_notes` (+ journal event)
- `COMPILE_CONTEXT` -> deterministic read; snapshot persistence is opt-in via `persistSnapshot`, `renderSnapshotCard`, and `snapshotKind`

## Current truth vs history

- Current truth:
  - `state_items`
  - active note/link/loop/model views
- Historical truth:
  - `journal_events`
  - `state_versions`
  - supersession and contradiction rows
  - archived/superseded note status retention

No normal hard-delete path for canonical semantic objects.

## Phase 5 truth maintenance behavior

Phase 5 adds deterministic truth projection services over existing durable tables:

- service implementation: `services/core/internal/aios/truth/engine.go`
- contracts: `services/core/internal/aios/domain/truth.go`

Projection behavior:

- active state queries read `state_items`
- state timelines reconstruct from `state_versions`
- loop lifecycle state reads `open_loops` with transition validation in Control Lane
- contradiction views read `contradiction_records`
- supersession chains resolve from `supersession_records`

Truth explanations include current value/object plus linked history/evidence references where available.

### State timeline behavior

- each `UPDATE_STATE` mutation updates current projection (`state_items`) and appends a timeline row (`state_versions`)
- workspace/lane scoped current lookup is enforced for updates and reads
- prior values remain inspectable; updates never erase previous timeline rows

### Lifecycle rules (enforced by transitions + validators)

- notes:
  - `active -> superseded`
  - `active -> archived`
  - `superseded -> archived`
- models:
  - `provisional -> promoted`
  - `provisional -> deprecated`
  - `promoted -> deprecated`
- loops:
  - `open/in_progress/blocked -> resolved`
  - `resolved -> archived`
  - terminal/archive transitions rejected where invalid

### Projection rebuild / repair

`truth.Engine.RebuildProjection(scope, dryRun)` provides deterministic drift detection/reporting for:

- missing current state projection rows with existing history
- mismatched current-vs-latest history values
- orphan contradiction references

Dry-run does not mutate persistence.

## Query examples (repository-level)

- current state by key/scope: `StateRepository.GetCurrent`
- timeline by key/scope: `StateRepository.GetTimeline`
- active notes by scope: `MemoryNoteRepository.ListActive`
- links by source/target/neighborhood: `SemanticLinkRepository`
- open loops by state/priority/staleness: `OpenLoopRepository`
- contradictions by object: `ContradictionRepository.ListByObject`
- supersession successor: `SupersessionRepository.GetCurrentSuccessor`
- models by status/type/evidence: `DerivedModelRepository`
- artifacts by scope/hash: `ArtifactRefRepository`
- journal by scope/correlation/recent: `JournalRepository`

## Backup/export/import notes

Cognitive tables included in `full_backup` extraction:

- `provenance_records`
- `journal_events`
- `memory_notes`
- `semantic_links`
- `state_items`
- `state_versions`
- `open_loops`
- `artifact_refs`
- `derived_models`
- `contradiction_records`
- `supersession_records`
- `context_packet_snapshots`

Restore ordering concern (future restore expansion):

1. `provenance_records`
2. core entities (`memory_notes`, `state_items`, `open_loops`, `artifact_refs`, `derived_models`, `journal_events`)
3. relation/history tables (`semantic_links`, `state_versions`, `contradiction_records`, `supersession_records`)
4. `context_packet_snapshots`

Audit/correlation survivability:

- rows carry syscall/correlation/trace/audit identifiers; exported JSON keeps these fields.

Artifacts:

- cognitive `artifact_refs` store URIs/checksums (references).
- binary/object files remain managed by existing artifact storage system.

## Phase 5.75 autonomy bookkeeping note

Autonomy layer entities introduced in Phase 5.75:

- autonomy charters
- intents
- freedom budgets
- autonomy decisions
- budget reservations
- curiosity items

Current implementation uses repository interfaces with in-memory stores (`services/core/internal/aios/autonomy/repositories.go`) to keep policy/runner contracts stable while avoiding parallel persistence rewrites in this phase. Durable SQLite tables for autonomy bookkeeping can be added incrementally without changing syscall truth boundaries.
