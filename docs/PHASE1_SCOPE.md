# FORGE — Phase 1 Scope

## In scope (delivered)

### Desktop / UI

- Tauri shell + React workspace layout (nav + main + right activity rail)
- Pages: **Command**, **Memory**, **Sources**, **Adapters**, **Events**, **Settings**
- Command bar with explicit verbs + `?query` shorthand + `:reindex`
- Operator-grade dark theme + minimal experimental light toggle (preference persisted)

### Local service (Go)

- HTTP API for settings, sources, ingest/reindex, search, chunk detail, events, adapters, minimal command execution logging
- SQLite schema + migrations on startup
- FTS5 indexing + search ranking + snippets
- Structured event emission for ingest/search/commands/adapters/errors

### Ingestion

- Recursive directory scan with extension allowlist (configurable)
- Skips common heavy directories (`.git`, `node_modules`, `target`, `dist`)
- Skips unchanged files via SHA-256 content hash
- Supported extensions in Phase 1 (default list):
  - documents: `.md`, `.txt`, `.json`
  - code/text: `.ts`, `.tsx`, `.js`, `.go`, `.py`, `.rs`, `.java`, `.yaml`, `.yml` (plus a few pragmatic extras in code)

### Adapters

- Formal adapter interface + registry
- Stubs: `ollama`, `codex`, `claude_code` (explicitly not wired to external tools)

## Out of scope (explicitly deferred)

### Intelligence / automation

- Embeddings / vector search / hybrid retrieval
- Agent routing, planning, autonomous execution
- Approval gates for tool use
- “Project dossiers” and task-context packets

### Product integrations

- Real Ollama integration (HTTP bridge, model selection UX)
- Real Codex / Claude Code integration (CLI/session contracts)

### Filesystem completeness

- Automatic purge of DB rows for files deleted from disk (planned hygiene)
- Native folder picker UX in the desktop app (path paste in Phase 1)

### Packaging polish

- Signed installers, auto-updates, crash reporting (not Phase 1 goals)

## Acceptance mapping

- App launches with stable routing: **yes**
- Add folder sources + index: **yes**
- Search + previews + detail view: **yes**
- Command bar navigation + actions: **yes**
- Events show meaningful entries: **yes**
- Adapter stubs behind interfaces: **yes**
- Docs describe reality + next steps: **yes**

## Phase 2 recommended next steps

1. **Per-source watcher efficiency** (index only affected source; incremental deletes)
2. **Embeddings** stored by stable ids (`chunk_id`) + hybrid retrieval API
3. **Adapter bridge #1**: Ollama (localhost) with explicit model configuration + failure UX
4. **Job runner**: queue + cancellable tasks surfaced in the Activity panel
5. **Tooling gates**: shell/git tools behind consent + audit (still local-first)
