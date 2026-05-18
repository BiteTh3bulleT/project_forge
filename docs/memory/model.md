# Memory Model

This document describes the public-safe lifecycle for FORGE memory notes and related state. It intentionally stays at the authority and data-shape level; it does not expose private prompts, raw chat transcripts, secrets, or operator-specific content.

## Authority Boundary

Canonical memory is structured state owned by the governed semantic/control-lane path. Models, chat turns, adapters, retrieval results, and runtime workers may propose or supply evidence, but they do not directly mutate canonical memory.

The live canonical note path is:

1. A proposer produces a bounded semantic request, such as `CREATE_NOTE`.
2. Deterministic validation checks the request shape, scope, capability, and allowed action.
3. The Control Lane transaction writes the accepted object and journal/audit metadata together.
4. Later reads use scoped semantic repositories or context compilation surfaces, not raw chat dumps.

Historical `memory_observations` remain readable evidence and retrieval hints. Legacy observation mutation endpoints are retired; they are not the replacement path for canonical writes.

## Note Lifecycle

### Create

A memory note starts as a governed semantic write. The committed note records:

- stable note id
- note type, title, content, confidence, and status
- workspace and lane scope
- provenance, trace, correlation, syscall, audit, proposer, and committer metadata
- created and updated timestamps

Accepted notes are `active` unless the syscall explicitly creates a different allowed status. Rejected proposals must leave no canonical note behind.

### Link

Notes become useful through explicit semantic links and scoped context packaging. A link records the relationship type, source id, target id, scope, confidence, provenance, and creation time.

Links are separate records. They do not copy note content and they do not grant either linked object authority over the other. They are evidence for navigation, reconstruction, contradiction review, and context assembly.

### Supersede Or Archive

FORGE preserves history instead of rewriting it in place.

Supersession records that a newer object replaces an older object for a stated reason. The system records a supersession object, adds a semantic `supersedes` link, and marks the older compatible note as `superseded` when the status transition is valid.

Archival marks a note as `archived`, records the archive time, and removes the note from active-note lists. The note remains queryable by id and reconstructable from the journal, provenance, and audit trail.

Neither supersession nor archival deletes the historical record. Operators should treat old notes as historical truth about what was believed or committed at that time, not as current truth.

### Audit Reconstruction

Reconstruction starts from the journal and audit metadata, then follows object records and provenance:

1. Select journal events by workspace, lane, correlation id, trace id, syscall id, or time window.
2. Resolve committed object ids to notes, links, state items, loops, supersession records, contradiction records, artifacts, and context snapshots.
3. Follow provenance records to identify proposer, committer, scope, and source refs.
4. Apply status transitions in event order to distinguish active truth from archived or superseded history.
5. Treat context snapshots, observations, retrieval results, model outputs, and chat logs as evidence or acceleration metadata unless they were admitted through a governed canonical write.

This reconstruction model is why durable metadata fields are load-bearing. A memory note without provenance, syscall, correlation, trace, and audit linkage is not sufficiently reconstructable for canonical use.

## Observations, Chat, And Recall

Chat history is persisted conversation history. It can support bounded local recall, but raw chat is not canonical memory.

Memory observations are structured evidence and retrieval hints. They can be listed, filtered, ranked, and included in chat context with safety labels, but they do not become canonical notes by existing in the observation table.

Canonical recall should prefer active scoped notes, state items, open loops, links, and journal-backed context packets. Non-canonical recall may use bounded earlier-thread memory, related chat memory, and recent observations as hints, with source and authority labels preserved.

## Public-Safe Operating Rules

- Do not expose raw prompts, raw full transcripts, secrets, tokens, host dumps, or private logs as memory.
- Do not let model output mutate canonical notes directly.
- Do not hard-delete canonical memory to resolve conflict; supersede, archive, or record a contradiction.
- Do not treat snapshots, observations, retrieval output, VSA pointers, or KV cache as truth.
- Keep workspace and lane scope attached to every semantic object and reconstruction query.
- Preserve provenance on every committed note, link, state transition, snapshot, and journal event.
