# Workbench Surface

## Purpose

Workbench answers: **“What did FORGE materialize on disk, and what does it say?”** for text-like artifacts, within safety constraints.

## API

- `GET /artifacts?limit=&jobId=` — list artifact rows.
- `GET /artifacts/{id}` — metadata row.
- `GET /artifacts/{id}/content` — `{ artifact, textual, content, previewLimited }`.

## Path safety

The core resolves the stored `file_path` to an absolute path and requires it to sit under the configured artifact base directory (`Service.resolveSafePath`). Paths outside that base return an error.

## Textual detection

Heuristic: MIME prefix `text/`, JSON-related MIME, or extensions such as `.md`, `.json`, `.txt`, `.go`, `.ts`, `.tsx`. Other types return `textual: false` and empty `content` with `previewLimited: true`.

## Job correlation

Optional `jobId` query parameter:

- Filters the artifact list.
- Loads **job detail** for a compact **recent event tail** in the viewer column.

This uses the same `GET /api/jobs/{id}` projection as the Jobs UI.

## Honest limitations

- No built-in image/PDF renderer.
- No arbitrary “run this artifact” button; execution remains in **Jobs** / **Gateway** with policy gates.
- Diff view is line-based for textual artifacts (quick compare, not semantic diff).
