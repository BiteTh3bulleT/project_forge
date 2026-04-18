# FORGE Task Packets

Task packets are compact evidence contracts for execution.

They are built from scoped retrieval and project context, then stored as immutable packet records.

## Packet Goals

A packet should:
- capture objective and constraints
- include aligned evidence, not raw memory sludge
- preserve source references for auditability
- be small enough for reliable operator/tool use

## Packet Record

Core fields in `task_packets` include:
- objective and user request
- adapter target and execution mode
- risk class
- selected paths and scope snapshot
- source references and retrieved context
- request payload and notes

## Alignment Notes

FORGE now stores packet evidence alignment separately in `packet_alignment_notes`.

Each note explains why an evidence item was included and may link to:
- retrieval result id
- observation id

These notes are visible from Job Detail packet preview.

## Packet Construction Inputs

Packet building can use:
- explicit retrieval run result set
- fallback keyword search
- dossier constraints and routing defaults
- selected paths and scope boundaries

## Implemented Safety Rules

- packet creation requires non-empty user request/objective/adapter target
- retrieved context is persisted with chunk and path provenance
- packet-to-retrieval run links are persisted

## Deferred

- hard packet token-budget optimizer with automatic truncation policy tuning
- automatic canonical-source detection across duplicate files
