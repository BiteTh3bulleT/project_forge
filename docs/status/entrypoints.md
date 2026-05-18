# FORGE Entrypoints

_Scope: runnable surfaces in the current repo. Observed 2026-05-10._

## Authoritative entrypoints

| Entrypoint | Path | Command | Required for bring-up | Notes |
|---|---|---|---|---|
| Core HTTP service | [services/core/main.go](../../services/core/main.go) | `npm run core` | **Yes** | Runs strict VSA tracked-source preflight, applies safe local modelruntime dev defaults when available, then starts `services/core` with `go run .`. Binds to `FORGE_CORE_BIND_HOST` (default `127.0.0.1`) and `FORGE_CORE_PORT` (default `18492`). |
| Direct Go core | [services/core](../../services/core) | `cd services/core && go run .` | Developer fallback | Starts the service directly but bypasses root preflight and local modelruntime auto-detection in `scripts/forge-core.sh`. Prefer `npm run core` for bring-up. |
| `npm run build:core` | compiles Go | `npm run build:core` | For release | Runs strict VSA tracked-source preflight, then `go build ./...` under `services/core`. |
| Desktop shell | [apps/desktop](../../apps/desktop) | `npm run desktop` | Only for UI | Chains `desktop:clean-tauri` -> `desktop:clean-port` -> `desktop:check` -> `tauri -- dev`. |
| `npm run dev` | alias for `npm run desktop` | - | - | Historical alias. |
| `npm run build:desktop` | `vite build` only | `npm -w @forge/desktop run build` | For release | Does **not** build the Tauri binary. Use `npm -w @forge/desktop run tauri -- build` for that. |
| Orchestrated up | [scripts/forge-up.sh](../../scripts/forge-up.sh) (via `.mjs` dispatcher) | `npm run up` | Optional | Backgrounds core, waits for `/health`, backgrounds desktop. Writes PIDs to `.forge/run/`. |
| Orchestrated down | [scripts/forge-down.sh](../../scripts/forge-down.sh) | `npm run down` | Optional | Kills by PID/port. |
| Smoke test | [scripts/forge-smoke.mjs](../../scripts/forge-smoke.mjs) -> platform smoke script | `npm run smoke` | Optional | Boots core against an isolated data dir, probes endpoints, tears down; uses PowerShell on Windows and shell elsewhere. |

## Not authoritative / deferred

- **Direct `go run .` from `services/core` as the recommended bring-up path** - supported for focused Go debugging, but it bypasses the root VSA tracked-state preflight and shell-local modelruntime defaults.
- **Nix `forge-core` package** ([flake.nix](../../flake.nix)) - package output exists. Validation depends on a Nix-enabled host/daemon; this repo status no longer treats VSA source tracking as a Nix blocker.
- **Tauri release bundle** - path exists (`npm -w @forge/desktop run tauri -- build`) but the normal repository release check remains `npm run build:desktop`. Desktop Nix packaging status is tracked in [desktop_nix_packaging_gap.md](desktop_nix_packaging_gap.md).

## No separate admin CLI

There is no standalone admin binary. All administration is done via
HTTP endpoints on the running core service (`/api/settings`,
`/api/autonomy/*`, `/api/backup/*`, etc.). The desktop shell is the
primary operator UI.

## Authoritative choice when duplicated

- For starting core locally: **`npm run core`** is the recommended root
  bring-up path because it includes strict VSA source-state preflight and
  the local dev modelruntime defaults wrapper.
- For starting the full dev stack: **`npm run up`** is the intended
  operator path. It is idempotent and logs to `.forge/logs/`.
- For smoke verification without desktop: **`npm run smoke`**.
