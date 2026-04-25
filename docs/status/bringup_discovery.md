# Bring-Up Discovery (Pass 1)

_Observed against repository state on 2026-04-22, branch `main`._

Pass 1 of the current-FORGE bring-up phase. Purpose: map how the
existing repository is supposed to boot, and what actually blocks it.
No large changes; this doc is input for later passes.

## 1. Current entrypoints

| Process | Path | Command | Role |
|---|---|---|---|
| Core service | `services/core/main.go` | `cd services/core && go run .` (or `npm run core`) | HTTP API + AI-OS runtime |
| Desktop shell | `apps/desktop` | `npm run desktop` (chains port-clean → dep-check → `tauri dev`) | Tauri 2 + React UI bound to core |
| Orchestrated up | `scripts/forge-up.{sh,mjs,ps1}` | `npm run up` | Starts core, waits for `/health`, starts desktop |
| Orchestrated down | `scripts/forge-down.{sh,mjs,ps1}` | `npm run down` | PID/port-based teardown |
| Dep preflight | `scripts/check-desktop-deps.sh` | Called by `npm run desktop` | Linux-only WebKit/GTK check |
| Core VSA preflight | `scripts/check-vsa-files.sh` | Called by `npm run core`, `npm run smoke`, `build:core`, `test:core`, and `vet:core` with `--require-tracked` | VSA status: authoritative source; preflight fails fast if required files are missing and fails closed when tracked-state cannot be verified |
| Port cleaner | `scripts/desktop-clean-port.sh` | Called by `npm run desktop` | Frees port 5173 |

No CLI/admin tool exists as a separate binary; all admin happens via
HTTP endpoints on the core service.

## 2. Required dependencies

| Dep | Required for | Notes |
|---|---|---|
| Go `>= 1.22` | core build | `services/core/go.mod` |
| Node `>= 18` | desktop build, scripts | Nix shell pins to Node 20 |
| Rust + Cargo | desktop (Tauri) build | `apps/desktop/src-tauri/Cargo.toml` |
| SQLite | runtime (via `modernc.org/sqlite`, pure Go) | No cgo needed |
| `pkg-config`, `webkit2gtk-4.1`, `javascriptcoregtk-4.1`, `gtk+-3.0`, `libsoup3`, `librsvg`, `libayatana-appindicator` | Linux desktop only | Checked by `scripts/check-desktop-deps.sh` |

### Env vars read at core boot

| Var | Default | Source |
|---|---|---|
| `FORGE_DATA_DIR` | `~/.config/forge` (fallback: `./` if `UserConfigDir` fails) | [config.go:16](services/core/internal/config/config.go#L16) |
| `FORGE_CORE_PORT` | `18492` | [config.go:25](services/core/internal/config/config.go#L25) |
| `FORGE_WORKSPACE_DIR` | `/` | [config.go:30](services/core/internal/config/config.go#L30) |
| `FORGE_TELEGRAM_GATEWAY_ENABLED` | `true` (but token absence is the effective kill-switch) | [telegram_gateway_server.go](services/core/internal/api/telegram_gateway_server.go) |
| `FORGE_TELEGRAM_BOT_TOKEN` | unset (gateway stays off) | same |
| `FORGE_DISCORD_ENABLED` | `false` | [discord_gateway_server.go](services/core/internal/api/discord_gateway_server.go) |
| `FORGE_DISCORD_BOT_TOKEN`, `FORGE_DISCORD_GUILD_ID`, `FORGE_DISCORD_DEFAULT_CHANNEL_ID` | unset | same |

### Env var read at desktop **build** time (not runtime)

| Var | Default | Source |
|---|---|---|
| `VITE_FORGE_API_URL` | `http://127.0.0.1:18492` | [apps/desktop/src/lib/api.ts:258](apps/desktop/src/lib/api.ts#L258) |

Vite inlines this at build time. A built desktop binary cannot be
repointed to a different core without rebuilding.

### Ports

| Port | Who | Configurable |
|---|---|---|
| `18492` | core HTTP | via `FORGE_CORE_PORT` |
| `5173` | Vite dev server | `strictPort: true` in `vite.config.ts` — must be free |

### Local directories created during boot

- `${FORGE_DATA_DIR}/forge.sqlite{,-shm,-wal}` — SQLite store.
- `${FORGE_DATA_DIR}/backups/`, `${FORGE_DATA_DIR}/exports/` — auto-created.
- `.forge/run/` and `.forge/logs/` — created by `scripts/forge-up.sh`.

## 3. Startup order

1. **Core first.** `forge-up.sh` backgrounds `npm run core`, waits up to
   20 s (40 × 0.5 s) for `GET /health` to return 200.
2. **Desktop second.** Once core is healthy, `npm run desktop` is
   backgrounded. No readiness wait — Vite/Tauri compile in the
   background and log to `.forge/logs/desktop.log`.
3. Telegram/Discord gateways are **independent** and non-blocking; they
   init inside `api.NewServer` via `tryStart*` wrappers that log and
   return nil on failure. No token → disabled → not fatal.
4. Autonomy runner starts as a goroutine inside core — does not gate
   HTTP readiness.

## 4. Current blockers

### Critical (compile-level)

- **[RESOLVED in this branch] VSA authority ambiguity.**
  Required VSA sources (`vsa_engine.go`, `vsa_indexer.go`,
  `vsa_signals.go`) are tracked as authoritative source files in this
  branch state (not generated, not optional). Strict `--require-tracked`
  preflight remains to guard against future drift.

### High (behavioral)

- **[RESOLVED] Autonomy default mode is now `observe`.**
  Fresh boot still seeds 4 default charters and 2 default budgets for
  inspection, but maintain/mission behavior requires an explicit
  `autonomy_mode` operator setting.

### Medium (visibility)

- **[MEDIUM] Modified but uncommitted `apps/desktop/src-tauri/src/main.rs`
  and `Cargo.toml`.** Unclear whether intentional. A fresh clone will
  behave differently from the current working tree. Not blocking boot
  on this machine.
- **[MEDIUM] Desktop has no readiness signal.** `forge-up.sh` starts
  desktop and exits. Vite/Tauri may still be compiling. No operator
  feedback until they `tail .forge/logs/desktop.log`.
- **[MEDIUM] No `.env.example` for desktop.** `apps/desktop/.env.development`
  exists but is gitignored. Fresh operators have no canonical template
  for `VITE_FORGE_API_URL`.

### Low

- `implementation_matrix.md:69` flags absence of root JS test scripts
  (`blocked` status). Not a bring-up blocker; quality-gate blocker only.
- Linux WebKit preflight is informative but operator may miss the
  one-line failure in terminal output.

## 5. Bring-up target states

The phase defines four progressively-wider bring-up targets:

### T1 — minimal core up

`cd services/core && go run .` produces an HTTP server on `:18492`.
`GET /health` returns `{"ok":true,"service":"forge-core"}`. **Verified
2026-04-21.** No warnings or errors in boot log.

### T2 — core + gateway + audit + artifact paths up

Core HTTP endpoints respond with non-error shapes:
- `/health` → 200, healthy JSON.
- `/api/meta` → data/workspace/DB paths.
- `/api/autonomy/status` → mode=`observe`, dream inactive.
- `/api/telegram/status`, `/api/discord/status` → disabled states reported honestly.
- `/api/adapters` → registered adapters listed with status.
- `/api/jobs` → empty list.
- `/api/memory/observations/<id>/vsa` → 400 (validation), not panic.

**Verified 2026-04-21** against a clean `/tmp/forge-bringup/data` data
dir. SQLite file and `backups/` + `exports/` subdirs auto-created on
first boot.

### T3 — desktop connected to running core

Feasible on Linux hosts where WebKit/GTK deps are installed. Not
runtime-verified in this pass — the Linux WebKit stack is not installed
on the discovery host, and requesting operator time for interactive GUI
bring-up is out of scope for this automated pass. Build-level checks
are in `docs/status/desktop_bringup.md`.

### T4 — smoke-testable current FORGE

A runnable smoke script at [scripts/forge-smoke.sh](scripts/forge-smoke.sh)
boots core, probes the health/meta/autonomy/adapters endpoints, and
cleans up. See [docs/status/smoke_test_status.md](docs/status/smoke_test_status.md).
