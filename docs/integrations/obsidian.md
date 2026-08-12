# Obsidian Integration

Status: local editor integration; non-canonical evidence only.

## Vaults

- The repository root is the primary Obsidian vault for FORGE source and
  documentation.
- `FORGE/` is a minimal companion vault retained for compatibility. It does
  not own project truth and should remain configuration-compatible with the
  root vault.

Shared configuration keeps delete confirmation enabled and Obsidian Sync
disabled. An operator may enable Sync locally after reviewing the data scope,
but that preference must not become the repository default.

The shared excluded-files list hides Git internals, dependency trees,
worktrees, local FORGE runtime state, frontend build output, and Rust/Tauri
targets from Obsidian indexing. Source and documentation directories remain
visible; unsupported source files are still available when intentionally
needed.

`workspace.json` is machine-local session state. It is ignored so panes,
recent files, and device-specific layout do not churn repository history.

## FORGE Engine MCP Plugin

The bundled `forge-engine-mcp` desktop plugin writes a local
`.obsidian/plugins/forge-engine-mcp/context.json` file. Despite the historical
plugin name, it does not start an MCP server, make network requests, execute
tools, or write canonical FORGE state.

The context file is transient, ignored evidence. Its envelope explicitly marks
the payload as non-canonical and admission-required. Selection text, the active
line, paths, provenance strings, open-note lists, links, embeds, tags, headings,
frontmatter keys, and cursor shape are bounded. Frontmatter values are
intentionally excluded to reduce accidental secret exposure. Unchanged state
is written at most once per 30-second heartbeat.

Any future consumer must validate freshness and schema, preserve provenance,
and submit wanted information through the existing governed admission or
semantic syscall path. It must never interpret this file as mutation authority.

## Validation

Run:

```sh
npm run validate:obsidian
```

The validator checks JSON integrity, conservative defaults, vault parity,
plugin manifest/bundle integrity, JavaScript syntax, ignore rules for session
state and live context evidence, the plugin's editor-only dependency boundary,
authority flags, metadata bounds, and frontmatter-value exclusion.
