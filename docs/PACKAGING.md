# FORGE — Local packaging

## Desktop (Tauri)

From the repository root:

```bash
npm -w @forge/desktop run tauri -- build
```

Artifacts land under `apps/desktop/src-tauri/target/release/` (platform-dependent).

## Core (Go)

```bash
cd services/core
go build -trimpath -ldflags "-s -w" -o ../../dist/forge-core .
```

## Configuration

- `FORGE_DATA_DIR` — SQLite and local bundles (`forge.sqlite`, `backups/`, `exports/`).
- `FORGE_WORKSPACE_DIR` — default scope for lanes and permission read roots.
- `FORGE_CORE_PORT` — HTTP API port (default `18492`).

## Recording a shipped artifact

After you produce an installer or tarball, register it via:

- UI: **Release** page → “Record release artifact”, or
- API: `POST /api/release/artifacts`

This does not replace your OS packaging workflow; it creates durable bookkeeping inside FORGE.
