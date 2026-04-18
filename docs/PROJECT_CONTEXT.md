# Project Context Normalization

## Goal

Convert a source context file into durable, local, versioned guidance artifacts that can be inspected and regenerated.

## Source Input

Default source resolution order:

1. explicit `sourcePath` provided by API/UI
2. stored `project_context_source_path` setting
3. `${FORGE_WORKSPACE_DIR}/FORGE_CONTEXT.md`

## Normalization Outputs

Each import generates a `project_context_records` row with:

- `context_version`
- source path/hash/size
- generated timestamp
- normalized summary JSON
- full generated markdown payloads
- generated file paths

## Generated Files

- `AGENTS.md`
- `CLAUDE.md`
- `docs/FORGE_PROJECT_BRIEFING.md`
- `.cursor/rules/forge-context.mdc`

## Archive

Raw imported source content is archived under:

- `${FORGE_DATA_DIR}/project_context/imports/`

This preserves evidence of what was normalized.

## Regeneration

- UI: Project Context page (`Import + Normalize`, `Regenerate`)
- API:
  - `POST /api/project-context/import`
  - `POST /api/project-context/regenerate`

Regeneration creates a new context record/versioned snapshot; it does not mutate prior records.

## How Packets Use Context

`task_packets` store `source_context_record_ids_json`, allowing later verification of which context basis was handed to workers.
