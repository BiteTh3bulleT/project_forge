# Core Boot Status

_Observed 2026-04-22 on Linux 6.19, Go 1.26.1, against branch `main`
with authoritative tracked VSA files in this branch state._

## Exact command

```sh
cd services/core
go run .
```

Or from repo root: `npm run core` (runs strict VSA preflight first, including tracked-state check).

## Required env vars

None. All have safe defaults:

| Var | Default | Effect |
|---|---|---|
| `FORGE_DATA_DIR` | `~/.config/forge` | SQLite DB, backups, exports live here |
| `FORGE_CORE_PORT` | `18492` | HTTP listen port |
| `FORGE_CORE_BIND_HOST` | `127.0.0.1` | HTTP bind host |
| `FORGE_WORKSPACE_DIR` | `/` | Workspace root for file/ingest operations |

For an isolated dev boot, override all three:

```sh
FORGE_DATA_DIR=/tmp/forge-dev/data \
FORGE_WORKSPACE_DIR=/tmp/forge-dev/workspace \
FORGE_CORE_PORT=18492 \
FORGE_CORE_BIND_HOST=127.0.0.1 \
go run .
```

## Required local directories/files

None pre-creatable. Core creates on first boot:

- `${FORGE_DATA_DIR}/` (if missing)
- `${FORGE_DATA_DIR}/forge.sqlite` (+ `-shm`, `-wal`)
- `${FORGE_DATA_DIR}/backups/`
- `${FORGE_DATA_DIR}/exports/`

Store migrations run automatically inside `store.Open()` —
[internal/store/store.go:16-32](services/core/internal/store/store.go#L16).

## Optional services

| Service | Default | Effect if not configured |
|---|---|---|
| Telegram gateway | off (no token) | `GET /api/telegram/status` reports `ready=false`, `reason="telegram bot token is not configured"`. Core continues. |
| Discord gateway | off (`FORGE_DISCORD_ENABLED=false`) | `GET /api/discord/status` reports `enabled=false`. Core continues. |
| Ollama adapter | tries `http://127.0.0.1:11434` | If Ollama absent, adapter reports not-ready; does not block boot. |

## Current boot status: **GREEN**

- `cd services/core && go run .` -> **GREEN**.
- `npm run core` -> **GREEN** with strict tracked-state preflight.

Verified 2026-04-21 on a clean data dir:

```
$ curl -s http://127.0.0.1:18492/health
{"ok":true,"service":"forge-core"}

$ curl -s http://127.0.0.1:18492/api/meta
{"dataDir":"/tmp/forge-bringup/data","dbPath":"/tmp/forge-bringup/data/forge.sqlite","workspaceDir":"/tmp/forge-bringup/workspace"}

$ curl -s http://127.0.0.1:18492/api/autonomy/status
{"available":true,"counts":{"activeCharters":4,"activeIntents":0,"budgets":2,"recentDecisions":0},"dream":{"active":false},"mode":"maintain","scope":{...}}
```

Boot log (excerpted — first-boot on empty data dir):

```
forge-core listening on http://127.0.0.1:18492
```

No warnings, no errors, no stack traces.

## Known degraded subsystems on clean boot

| Subsystem | State | Cause |
|---|---|---|
| Telegram gateway | `disabled` | No token configured. Operator opt-in. |
| Discord gateway | `disabled` | Default off. Operator opt-in. |
| Ollama adapter | `ready` if Ollama running locally; otherwise `not ready` | Optional local LLM. |
| Embeddings (semantic) | `local_hash` provider by default | Real embedding provider requires settings change. |
| Autonomy `dream.active` | `false` at boot | Ticks after idle threshold (3 min) if enabled; stays propose-only inside default charters. |

## VSA dependency integrity

VSA status: **authoritative source** (not generated, not optional).

Required VSA sources are tracked in authoritative repo state for this branch snapshot.

Guardrails retained:
- `scripts/check-vsa-files.sh` fails fast on missing required VSA files.
- `npm run core` and `npm run smoke` keep `--require-tracked` verification.
- Root `build:core`/`test:core`/`vet:core` also enforce `--require-tracked`.

## No panics, no fatals

Grep confirms no `panic(` or `log.Fatal(` in initialization code
([main.go](services/core/main.go) only uses `log.Fatalf` for hard HTTP
listen failures — which are correct to hard-fail on).
