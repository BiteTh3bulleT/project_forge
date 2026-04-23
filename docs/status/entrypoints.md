# FORGE Entrypoints

_Scope: runnable surfaces in the current repo. Observed 2026-04-21._

## Authoritative entrypoints

| Entrypoint | Path | Command | Required for bring-up | Notes |
|---|---|---|---|---|
| Core HTTP service | [services/core/main.go](services/core/main.go) | `cd services/core && go run .` | **Yes** | Listens on `FORGE_CORE_PORT` (default `18492`). Writes logs to stdout. |
| `npm run core` | wraps the above | `npm run core` | Equivalent to above | Thin wrapper. |
| `npm run build:core` | compiles Go | `npm run build:core` | For release | Produces binary at `services/core/core` (not in build output dir). |
| Desktop shell | [apps/desktop](apps/desktop) | `npm run desktop` | Only for UI | Chains `desktop:clean-port` → `desktop:check` → `tauri -- dev`. |
| `npm run dev` | alias for `npm run desktop` | — | — | Historical alias. |
| `npm run build:desktop` | `vite build` only | `npm -w @forge/desktop run build` | For release | Does **not** build the Tauri binary. Use `npm -w @forge/desktop run tauri -- build` for that. |
| Orchestrated up | [scripts/forge-up.sh](scripts/forge-up.sh) (via `.mjs` dispatcher) | `npm run up` | Optional | Backgrounds core, waits for `/health`, backgrounds desktop. Writes PIDs to `.forge/run/`. |
| Orchestrated down | [scripts/forge-down.sh](scripts/forge-down.sh) | `npm run down` | Optional | Kills by PID/port. |
| Smoke test | [scripts/forge-smoke.sh](scripts/forge-smoke.sh) | `bash scripts/forge-smoke.sh` | Optional | Boots core against an isolated data dir, probes endpoints, tears down. Added in this pass. |

## Not authoritative / deferred

- **Nix `forge-core` package** ([flake.nix](flake.nix)) — reproducibly builds the Go service but is blocked on the VSA uncommitted-files issue (see [bringup_discovery.md](bringup_discovery.md) §4).
- **Tauri release bundle** — path exists (`npm -w @forge/desktop run tauri -- build`) but is out of scope for this pass; packaging gap tracked in [desktop_nix_packaging_gap.md](desktop_nix_packaging_gap.md).

## No separate admin CLI

There is no standalone admin binary. All administration is done via
HTTP endpoints on the running core service (`/api/settings`,
`/api/autonomy/*`, `/api/backup/*`, etc.). The desktop shell is the
primary operator UI.

## Authoritative choice when duplicated

- For starting core locally: **`cd services/core && go run .`** is the
  ground truth. `npm run core` is a wrapper.
- For starting the full dev stack: **`npm run up`** is the intended
  operator path. It is idempotent and logs to `.forge/logs/`.
- For smoke verification without desktop: **`bash scripts/forge-smoke.sh`**.
