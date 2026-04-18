# Canvas Surface

## Purpose

Canvas provides **durable scratch space**: boards of positioned notes for requirements, snippets, and planning beside real job execution.

## Persistence

- `canvas_boards`: `title`, optional `dossier_id`, timestamps.
- `canvas_notes`: `board_id`, `title`, `body`, geometry (`x`, `y`, `width`, `height`), `pinned`, `color`, `links_json`, timestamps.

Foreign keys cascade delete notes when a board is removed.

## API

- `GET /canvas/boards`, `POST /canvas/boards`, `GET /canvas/boards/{id}`, `DELETE /canvas/boards/{id}`.
- `POST /canvas/boards/{id}/notes` — create note.
- `PATCH /canvas/boards/{id}/notes/{noteId}` — partial update (`PatchNote` JSON).
- `DELETE /canvas/boards/{id}/notes/{noteId}`.

## UI model

Notes render in an absolutely positioned layer inside a tall scroll container. **Save** commits geometry and text; there is no implicit auto-save on every keystroke (reduces accidental partial writes).

## Links field

The database stores `links_json` as an array of objects. The API accepts structured links on patch. The current UI does not ship a link builder—use the note body for human-readable references (job ids, dossier ids, paths) or PATCH via API client.

## Deferred enhancements

- Drag-and-drop and resize handles.
- Link picker to jobs, dossiers, artifacts, chat threads.
- Board templates and export.
