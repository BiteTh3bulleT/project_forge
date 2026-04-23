# Desktop Bring-Up Status

_Observed 2026-04-21. Build-level checks; runtime GUI verification was
out of scope for the automated pass._

## Exact build command

```sh
# Production build (produces Tauri app bundle):
npm -w @forge/desktop run tauri -- build

# Or just the Vite frontend (no Rust bundle):
npm run build:desktop
```

## Exact run command (dev)

```sh
# From repo root. Requires core running first.
npm run desktop
```

Under the hood this runs:

1. `scripts/desktop-clean-port.sh 5173` — frees Vite's dev port.
2. `scripts/check-desktop-deps.sh` — Linux-only WebKit/GTK preflight.
3. `npm -w @forge/desktop run tauri -- dev` — starts Tauri, which
   starts Vite via `beforeDevCommand: "npm run dev"` in
   [tauri.conf.json](apps/desktop/src-tauri/tauri.conf.json).

## Required env/config

| Var | Where | Default |
|---|---|---|
| `VITE_FORGE_API_URL` | **build-time** for desktop; read in [api.ts:258](apps/desktop/src/lib/api.ts#L258) | `http://127.0.0.1:18492` |

`apps/desktop/.env.example` now exists as a template. Copy to
`apps/desktop/.env.development` for custom dev setups. `.env.*` files
other than `.env.example` are gitignored.

## Native deps (Linux)

`scripts/check-desktop-deps.sh` validates presence of:

- `webkit2gtk-4.1`
- `javascriptcoregtk-4.1`
- `gtk+-3.0`

Installation hints are printed for openSUSE, Debian/Ubuntu, Fedora, and
Arch. Nix users can enter the desktop shell: `nix develop .#desktop`
(these deps are included in [nix/shells/desktop.nix](nix/shells/desktop.nix)).

Additional Tauri runtime deps (covered in the Nix shell, need manual
install otherwise): `libsoup3`, `librsvg`, `libayatana-appindicator`,
`pkg-config`, `openssl`.

Darwin and Windows: no preflight script. Tauri relies on system
WebKit (macOS) or WebView2 (Windows). Untested in this pass.

## Current blockers

### Build blockers

- **[HIGH] Modified but uncommitted `apps/desktop/src-tauri/src/main.rs`
  and `Cargo.toml`.** The discovery agent flagged this; unclear whether
  intentional. A fresh clone sees different code than the current
  working tree. Out of scope to resolve in this pass.
- **[MEDIUM] Native deps on Linux hosts without Nix.** Operators must
  install WebKit/GTK manually if `check-desktop-deps.sh` fails.

### Runtime blockers

- **[MEDIUM] No readiness signal.** `npm run up` starts desktop in the
  background and returns immediately. On cold caches, Tauri + Vite take
  minutes to compile before the window opens. Operator must tail
  `.forge/logs/desktop.log` or watch for the window.
- **[LOW] `VITE_FORGE_API_URL` is build-time.** Repointing a built
  binary at a different core requires rebuild. For dev, just set the
  env in `.env.development` before running `npm run desktop`.

## Current status

| Item | Status |
|---|---|
| Build script correctness | real — dev command chain works end-to-end on Linux |
| Linux WebKit preflight | real — script exists and returns actionable hints |
| Env example | **added this pass** (`apps/desktop/.env.example`) |
| Buildable on this host | **not verified this pass** — no WebKit on the discovery machine |
| Runnable (window opens) | **not verified this pass** — interactive GUI verification out of scope |
| Connects to core | logic verified (env default matches core default port); live session not booted in this pass |

To verify on a host with WebKit/GTK installed:

```sh
# Terminal 1
cd services/core && go run .

# Terminal 2
npm install
npm run desktop
# Wait ~30–120 s on cold cache for Tauri+Vite to compile.
# Watch .forge/logs/desktop.log on errors.
```
