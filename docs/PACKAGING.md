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

## Docker

From the repository root:

```bash
npm run docker:config
npm run docker:build
npm run docker:up
```

The Compose stack builds:

- `core`: Go daemon with `/data/forge.sqlite` stored in the `forge-data` volume.
- `desktop-web`: browser-served Vite build of the desktop UI.
- optional `ollama` and `tei` sidecars behind Compose profiles.

See `docs/runbooks/docker_containerization.md`.

## Configuration

- `FORGE_DATA_DIR` — SQLite and local bundles (`forge.sqlite`, `backups/`, `exports/`).
- `FORGE_WORKSPACE_DIR` — default scope for lanes and permission read roots.
- `FORGE_CORE_PORT` — HTTP API port (default `18492`).
- `FORGE_CORE_BIND_HOST` — HTTP bind host (default `127.0.0.1`; Docker images set `0.0.0.0` intentionally for published ports).

## Recording a shipped artifact

After you produce an installer or tarball, register it via:

- UI: **Release** page → “Record release artifact”, or
- API: `POST /api/release/artifacts`

This does not replace your OS packaging workflow; it creates durable bookkeeping inside FORGE.
