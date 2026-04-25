# Operations Runbook

## Setup

Prerequisites: Go 1.22+, Node 18+, npm, Rust/Cargo for Tauri desktop, SQLite CLI optional.

Install dependencies:

```sh
npm install
```

## Build

```sh
npm run build
npm run build:desktop
npm run build:core
```

On Windows, current package scripts use Node for VSA preflight and core build. Some smoke/desktop helper scripts may still be Bash-only.

## Test

```sh
cd services/core && go test ./...
cd services/core && go vet ./...
npm test
npm run lint
npm run typecheck
npm run validate:desktop
```

## Run Core

```sh
FORGE_DATA_DIR=/tmp/forge-dev/data \
FORGE_WORKSPACE_DIR=/tmp/forge-dev/workspace \
FORGE_CORE_PORT=18492 \
  npm run core
```

Windows PowerShell equivalent:

```powershell
$env:FORGE_DATA_DIR="$pwd\\.forge\\data"
$env:FORGE_WORKSPACE_DIR="$pwd"
$env:FORGE_CORE_PORT="18492"
npm run core
```

## Safe Mode

```sh
export FORGE_SAFE_MODE_FORCE_CPU_ONLY=true
export FORGE_GPU_ENABLED=false
```

PowerShell:

```powershell
$env:FORGE_SAFE_MODE_FORCE_CPU_ONLY="true"
$env:FORGE_GPU_ENABLED="false"
```

## Health Checks

- `GET http://127.0.0.1:18492/health`
- `GET http://127.0.0.1:18492/api/meta`
- `GET http://127.0.0.1:18492/api/autonomy/status`
- `GET http://127.0.0.1:18492/forge/model-runtime/health`

## Dream Mode Dry-Run

```sh
curl -sS http://127.0.0.1:18492/api/dream/run \
  -H 'Content-Type: application/json' \
  -d '{"workspaceId":"default","laneId":"control.semantic","mode":"microdream"}'
```

## Backup / Restore

Use `/api/backup/*` surfaces or desktop Backup page. Restore is DB-transactional for supported sections; non-DB side effects are explicitly not globally rolled back.

