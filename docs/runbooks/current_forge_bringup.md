# Current FORGE Bring-Up Runbook

_The authoritative operator path to start, verify, and shut down the
current FORGE system. Updated 2026-04-22 against branch `main`._

## 0. Prerequisites

### Tools

- **Go ≥ 1.22** (repo uses `go.mod` go 1.22; Nix shells pin to 1.26).
- **Node ≥ 18** (Nix shells pin to 20).
- **Rust + Cargo** — only if you plan to run the desktop.
- **SQLite** CLI — optional; useful for inspecting `forge.sqlite`.
- **curl**, **bash**, **ss**, **lsof** — for the smoke script.

### Linux-only native deps (desktop only)

Needed for Tauri:

- `pkg-config`, `webkit2gtk-4.1`, `javascriptcoregtk-4.1`, `gtk+-3.0`,
  `libsoup3`, `librsvg`, `libayatana-appindicator`, `openssl`.

Run `npm run desktop:check` for distro-specific install hints. Or use the
Nix desktop shell which includes all of these:
`nix develop .#desktop`.

### VSA repo state check

VSA status: **authoritative source** (not generated, not optional).

Run:

```sh
git ls-files \
  services/core/internal/memory/vsa_engine.go \
  services/core/internal/memory/vsa_indexer.go \
  services/core/internal/memory/vsa_signals.go
```

The command output must include all three paths. These files **must be present and tracked in git**
for authoritative `npm run core` / `npm run smoke` bring-up:

- `services/core/internal/memory/vsa_engine.go`
- `services/core/internal/memory/vsa_indexer.go`
- `services/core/internal/memory/vsa_signals.go`

If they are missing, `go build ./...` will fail with `undefined:`
errors in `internal/memory/`. If they are present but untracked, strict
preflight for `npm run core` / `npm run smoke` fails by design.
If tracked-state cannot be verified (git unavailable or not a git work
tree), strict preflight fails closed by design.
Root `build:core`, `test:core`, and `vet:core` now also enforce `--require-tracked`.

## 1. Minimal bring-up (core only, 60 seconds)

```sh
# Install JS workspace deps (once).
npm install

# Start core against an isolated local data dir (recommended root path; includes VSA preflight).
FORGE_DATA_DIR=/tmp/forge-dev/data \
FORGE_WORKSPACE_DIR=/tmp/forge-dev/workspace \
FORGE_CORE_PORT=18492 \
  npm run core
```

`npm run core` enables the governed modelruntime management surface for local desktop development. It does not auto-configure cloud/provider fallback; configure `FORGE_LLAMA_CPP_ENDPOINT`, `FORGE_MODEL_OPENAI_COMPAT_ENDPOINT`, or `FORGE_MODEL_VLLM_ENDPOINT` explicitly when an inference backend is intended.

In another terminal:

```sh
curl -s http://127.0.0.1:18492/health
# {"ok":true,"service":"forge-core"}

curl -s http://127.0.0.1:18492/api/meta
# {"dataDir":"/tmp/forge-dev/data",...}
```

Ctrl-C the core process to stop it.

**This is "current FORGE is up."**

## 2. Full smoke test (recommended)

```sh
npm run smoke
```

This boots core against an ephemeral data dir, probes 7 endpoints, and
tears down cleanly. See
[../status/smoke_test_status.md](../status/smoke_test_status.md).
Expected output ends with `==> smoke OK`.

## 3. Orchestrated bring-up with desktop

```sh
npm run up
```

Starts core in the background, waits up to 20 s for `/health`, then
starts desktop in the background. Logs at `.forge/logs/core.log` and
`.forge/logs/desktop.log`. PIDs at `.forge/run/`.

**Known limitation:** `npm run up` does not wait for desktop to finish
compiling. On cold caches, the Tauri window takes 30–120 s to appear.
Tail `.forge/logs/desktop.log` if you think something went wrong.

## 3a. Current NixOS VM checkpoint (2026-05-11)

The `FORGE-OS` VirtualBox VM is installed and running the opt-in operator
desktop profile from `/projectforge/nix/nixos/profiles/forge-operator-desktop.nix`.
The host checkout is mounted in the guest at `/projectforge`; FORGE data is
stored in the guest at `/forge`.

Manual VM evidence is tracked in
[docs/evidence/vm_boot/2026-05-11-forge-os-operator-desktop.md](../evidence/vm_boot/2026-05-11-forge-os-operator-desktop.md).
That artifact is operator validation evidence only; it does not grant the shell
service-control or rebuild authority.

Current verified state:

- NixOS generation:
  `/nix/store/iy4v4h28zl65x0a5nw64332cvllfxx5v-nixos-system-forge-os-vm-25.11.10470.0c88e1f2bdb9`
- VM network: NAT with host SSH forwarding on `127.0.0.1:2222`.
- Launch path:
  `Plymouth FORGE-OS splash -> greetd -> forge-operator-session -> labwc -> forge-shell-session -> forge-desktop-shell -> FORGE login screen -> forge-core`
- Display: VirtualBox VMSVGA, 128 MiB VRAM, 3D acceleration enabled.
- Session compatibility: `WEBKIT_DISABLE_DMABUF_RENDERER=1` is set for the
  operator desktop to avoid VirtualBox Wayland/dmabuf protocol failures.
- Shell fit: the Tauri default window remains `1180x680` as a fallback, but the
  locked operator session explicitly fits the shell window to the detected
  monitor bounds before maximizing without an external titlebar.
- Core health: `curl -fsS http://127.0.0.1:18492/health` returns `ok: true`.
- Storage meta: `/api/meta` returns `/forge/data`, `/forge/data/forge.sqlite`,
  and `/forge/workspaces/default`.

Start from VM TTY:

```sh
mkdir -p "$HOME/forge-session-logs"
forge-operator-session >"$HOME/forge-session-logs/forge-operator-session.log" 2>&1
```

Do not run shell UI service-control actions. VM rebuilds remain operator setup
actions from a terminal, not FORGE shell authority.

## 4. Verify what is up

### Health

```sh
curl -s http://127.0.0.1:18492/health
```

Should return `{"ok":true,"service":"forge-core"}`.

### Meta (config snapshot)

```sh
curl -s http://127.0.0.1:18492/api/meta | jq .
```

Returns `dataDir`, `dbPath`, `workspaceDir` so you can confirm the env
vars landed.

### Autonomy mode

```sh
curl -s http://127.0.0.1:18492/api/autonomy/status | jq .
```

On fresh boot, expect:

```json
{
  "available": true,
  "mode": "observe",
  "counts": {"activeCharters": 4, "activeIntents": 0, "budgets": 2, "recentDecisions": 0},
  "dream": {"active": false}
}
```

Set `autonomy_mode` explicitly to `maintain` or `mission` only after
operator review.

### Adapters

```sh
curl -s http://127.0.0.1:18492/api/adapters | jq '.adapters[].id'
```

Expect at least `claude_code`, `codex`, and `ollama` (Ollama shows
`ready` only if a local Ollama is running at
`http://127.0.0.1:11434`).

### Operator inspectors

Restore inspector routes are read-only and workspace scoped:

```sh
curl -s 'http://127.0.0.1:18492/api/context/restore/recent?workspaceId=default&limit=10' | jq .
```

Persisted Dream reports are also read-only evidence:

```sh
curl -s 'http://127.0.0.1:18492/api/dream/reports?workspaceId=default&limit=10' | jq .
```

Both surfaces return empty lists cleanly on a fresh DB. They do not mutate canonical memory/state,
execute tools, or require modelruntime/GPU.

### External gateways (optional, off by default)

```sh
curl -s http://127.0.0.1:18492/api/telegram/status | jq .
curl -s http://127.0.0.1:18492/api/discord/status | jq .
```

Both should report their disabled state cleanly with a reason — e.g.
`"telegram bot token is not configured"`.

## 5. Configuration

See [config_reference.md](config_reference.md) for the full list.

Minimum for a real dev environment (not just isolated smoke):

```sh
export FORGE_DATA_DIR="$HOME/.local/share/forge"
export FORGE_WORKSPACE_DIR="$HOME/projects/my-project"
export FORGE_CORE_PORT=18492
```

For desktop, copy the env template:

```sh
cp apps/desktop/.env.example apps/desktop/.env.development
# edit if your core runs on a non-default URL
```

CPU-only safe mode (no GPU required):

```sh
export FORGE_SAFE_MODE_FORCE_CPU_ONLY=true
export FORGE_GPU_ENABLED=false
```

This keeps `forge-core` authoritative and bootable without GPU/modelruntime acceleration.
See [no_gpu_boot_and_recovery.md](no_gpu_boot_and_recovery.md) for full degraded-mode runbook.

## 6. What "success" looks like

1. `go build ./...` in `services/core` exits 0.
2. `go test ./...` in `services/core` passes all packages.
3. `npm run smoke` ends with `==> smoke OK`.
4. `/api/autonomy/status` returns `mode: "observe"` with 4 charters
   and 2 budgets seeded.
5. `/api/telegram/status` and `/api/discord/status` both report
   disabled with a reason (not 5xx).
6. Desktop window opens (manual verification) and the top-right
   status indicator shows connected.

## 7. Known degraded areas

Tracked in [implementation_matrix.md](../status/implementation_matrix.md)
and the Pass-1 status docs under `docs/status/`. Highlights:

- **VSA dependency integrity** — required VSA files are authoritative
  tracked source files and guarded by strict preflight. See
  [vsa_authority_report.md](../status/vsa_authority_report.md).
- **Backup/restore asymmetry** — only VSA-derived sections remain
  export-only/rebuildable. Full backup bundles include a section manifest,
  per-section checksums, restore row counts, and schema verification; restore
  remains DB-atomic only and does not import artifact file bytes.
- **Projection repair** — scaffold only.
- **JS/TS tests** — no dedicated JS/TS test suite; root `npm test`
  currently delegates to Go core tests.
- **Compute lane duplication** (`compute` vs `computelane`) — one
  authority not yet designated.
- **Tool capsules, NixOS modules, IRIS** — scaffolds/deferred, not
  active at runtime.

## 8. Safety

- **Autonomy default is `observe`.** Default charters and budgets are
  seeded at first boot, but maintain/mission behavior requires an
  explicit `autonomy_mode` setting change by the operator.
- **Dangerous tools** (shell, process control, external effects,
  privileged operations) remain `approval_only`. Do not relax these
  defaults without operator-signed approval.
- **Remote access** is off by default; token-gated when enabled.
- **Workspace root** defaults to `/` for direct Go/dev runs. The managed
  NixOS `forge-core` service defaults to `/forge/workspaces/default`.
  For real project work, scope `FORGE_WORKSPACE_DIR` to a specific
  project directory or dedicated workspace path.

## 9. Shutdown / reset

### Graceful shutdown

```sh
npm run down
```

Kills core (port 18492) and desktop (port 1420) by PID file / port.

### Reset local state

```sh
# From repo root, back up first if you have work.
rm -rf "${FORGE_DATA_DIR:-$HOME/.config/forge}"
rm -rf .forge/
```

Next `go run .` starts against a clean DB.

## 10. Troubleshooting

| Symptom | Likely cause | Fix |
|---|---|---|
| `undefined: s.GetObservationVSA` at build time | VSA files not in working tree | Commit or fetch the VSA files; see §0 |
| `cannot verify tracked-state for VSA files` | `--require-tracked` run without git or outside a git work tree | Install git and run from a real git checkout of this repo |
| Port `18492` already in use | Previous core didn't shut down | `npm run down`, or `lsof -ti tcp:18492 \| xargs kill` |
| Port `1420` already in use | Previous Vite didn't shut down | `node scripts/desktop-clean-port.mjs 1420` |
| `check-desktop-deps.mjs` fails | WebKit/GTK not installed | Follow the script's Linux dependency hint, or `nix develop .#desktop` |
| `curl /health` hangs | Core still booting (first run runs migrations) | Wait up to 20 s, check `.forge/logs/core.log` |
| `/api/autonomy/status` shows `available: false` | Autonomy loop never started | Check `.forge/logs/core.log` for a goroutine init error |
| Desktop window never opens | Tauri/Vite still compiling; or missing native deps | Tail `.forge/logs/desktop.log`; verify `node scripts/check-desktop-deps.mjs` passes |
| Ollama adapter `not ready` | Local Ollama not running | Start Ollama or ignore; other adapters still work |
