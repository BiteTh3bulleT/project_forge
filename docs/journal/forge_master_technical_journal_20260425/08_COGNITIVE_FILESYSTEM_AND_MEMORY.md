# Cognitive Filesystem And Memory

## Memory Object Taxonomy

| Object | Purpose | Status |
|---|---|---|
| `journal_events` | Append-only semantic truth trace | IMPLEMENTED |
| `memory_notes` | Canonical notes/facts/preferences/goals | IMPLEMENTED |
| `semantic_links` | Typed semantic relationships | IMPLEMENTED |
| `state_items` | Current state projection | IMPLEMENTED |
| `state_versions` | Historical state timeline | IMPLEMENTED |
| `open_loops` | Work/blocker lifecycle | IMPLEMENTED |
| `artifact_refs` | Evidence references | IMPLEMENTED |
| `derived_models` | Advisory/evidence-backed derived models | PARTIAL |
| `provenance_records` | Actor/source/trace lineage | IMPLEMENTED |
| `contradiction_records` | Preserved disagreements | IMPLEMENTED |
| `supersession_records` | Preserved replacement chains | IMPLEMENTED |
| `context_packet_snapshots` | Non-canonical context restore evidence | PARTIAL |
| observation memory tables | Compatibility/read-inspection memory | PARTIAL |

## Current vs Historical Truth

Current truth is read through current projections and active semantic objects. Historical truth remains in journal rows, state versions, supersession records, contradiction records, and archived/superseded statuses.

## Short / Mid / Long-Term Memory

CONCEPT / PARTIAL: Dream Mode v0 proposes short/mid/long-term routing, but tier changes are not canonical commits. Future tier promotion must go through semantic syscalls.

## Retrieval And Vectors

PARTIAL: Embedding/VSA/retrieval records exist and support search. They are retrieval support only and must not become truth authority.

## Backup Reality

PARTIAL: Core cognitive filesystem tables are covered by backup/restore. VSA-derived sections and some retrieval/observation evidence remain explicitly export-only or incomplete according to status/review docs.

