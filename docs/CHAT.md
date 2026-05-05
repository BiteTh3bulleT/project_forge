# Chat Surface

## Behavior

- **Persistence**: `chat_threads`, `chat_messages` in SQLite (see `services/core/internal/store/migrate.go`).
- **API** (under `/api`):
  - `GET /chat/threads` — list summaries.
  - `POST /chat/threads` — create (`title`, optional `dossierId`).
  - `GET /chat/threads/{id}` — thread + messages.
  - `DELETE /chat/threads/{id}` — delete thread.
  - `POST /chat/threads/{id}/attachments` — upload one file (`multipart/form-data`, field `file`) and persist as artifact.
  - `POST /chat/threads/{id}/messages` — append user message; optional assistant reply.
  - `POST /chat/threads/{id}/jobs` — create job (`jobs.CreateRequest` body).

## Assistant replies (Ollama)

When `requestAssistant` is true, the core:

1. Appends the user message.
2. Builds a bounded transcript from recent messages.
3. Resolves adapter `ollama` and checks `Info().Status == ready`.
4. Invokes capability **`analysis`** with input `{ "prompt": ... }` (dry-run supported).

If the adapter is missing, not ready, or the model is unset, the assistant message states the fact—no fabricated completion.

## Grounding And Tool Capability

The assistant prompt includes a non-overridable operational grounding guard. Before saying it cannot access files, inspect the repository, run commands, use a browser, search, inspect memory, or execute a machine action, the assistant must first probe available tools/gateway state.

When asked about FORGE project status, current phase, architecture, or its own system shape, it must use repository source truth when available:

- `README.md`
- `AGENTS.md`
- `docs/reviews/current_phase_status.md`

If the tool or filesystem read fails, it must report the exact attempted action and error instead of inventing a limitation or asking the operator to paste files.

## Attachments

- Chat supports file attachments (docs, images, and arbitrary files) through `/chat/threads/{id}/attachments`.
- Uploads are stored as real artifacts (`type=chat_attachment`) in artifact storage and linked to the chat thread.
- Sending a message can include `attachmentArtifactIds` so the user message metadata references uploaded files.
- Prompt assembly injects attachment context:
  - artifact identity (id/title/mime)
  - text excerpts for textual files when readable
  - non-text files are still linked and visible, with no fake OCR/vision claims

## Chat Inspector

- The desktop chat UI now includes a right-side inspector panel with:
  - **Code**: extracted assistant code blocks with copy support
  - **Files**: thread attachments with preview and Workbench deep-link

## Jobs from chat

The job body is the same JSON as `POST /api/jobs`. `initiatingSource` defaults to `chat` if omitted. A **system** message records `jobId` and `templateId` in metadata for traceability.

## Operator tips

- The assistant can answer directly in chat (including code examples and how-to guidance).
- Use **Queue job** for real execution or workspace mutation; do not rely on the assistant to silently run jobs.
- For dossier-scoped work, extend the thread create API usage with `dossierId` when your dossier workflow needs that linkage (UI may expose dossier picker in a later iteration).

## Troubleshooting

| Issue | Remedy |
|-------|--------|
| 404 on thread | Id wrong or deleted. |
| Assistant: “not registered” | Core adapters registry; ensure Ollama adapter wired in core bootstrap. |
| Assistant: “not ready” | Ollama not running or base URL wrong in settings. |
| Empty template list | Core offline or jobs template endpoint failing—check network tab. |
