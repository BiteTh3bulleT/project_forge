# Desktop Nix Packaging Status

_Status: Phase G3.5 real Tauri Nix packaging is implemented and validated on Linux._

## Supersession Notice

This file remains authoritative for the G3.5 package boundary only: `packages.forge-desktop-shell` is a real Tauri Nix build, not a launcher placeholder. It is not the current operator-session status source.

For current graphical desktop operation, use:

- `docs/operations/forge_graphical_shell_session.md`
- `docs/runbooks/forge_operator_desktop_vm.md`
- `docs/status/phase_g8_desktop_shell_verification.md`

Current operator VM/native desktop truth is:

```text
FORGE boot splash -> graphical OS login -> forge-operator session -> labwc -> forge-shell-session -> forge-desktop-shell
```

The older Cage `forge-wayland-session` path remains a fullscreen rollback/test lane. It is not the default operator desktop substrate.

`packages.forge-desktop-shell` is no longer a launcher placeholder. The package
builds the desktop frontend, builds the `apps/desktop/src-tauri` Tauri crate,
wraps the resulting Linux binary with the required GTK/WebKit runtime
environment, and exposes:

- `/bin/forge-desktop-shell`, the stable operator command
- `/bin/forge_desktop`, the underlying Tauri binary

The package advertises `passthru.containsTauriBinary = true`, so
`forge-shell-session` may prefer it after the explicit `FORGE_SHELL_BINARY`
override.

## Validated Build

Validated command:

```sh
nix --extra-experimental-features 'nix-command flakes' build path:$PWD#forge-desktop-shell --no-link --print-out-paths
```

The validated output contained both executable paths listed above. The wrapper
preserves safe shell-mode defaults:

- `FORGE_SHELL_SESSION_ENABLED=true`
- `FORGE_SHELL_MODE=fullscreen-shell`
- `FORGE_CORE_URL=http://127.0.0.1:18492` unless overridden
- `VITE_FORGE_API_URL=$FORGE_CORE_URL`
- `FORGE_SHELL_SAFE_MODE=true`
- `FORGE_SHELL_FULLSCREEN=true`
- host mutation, direct system control, model mutation, semantic memory write,
  and FORGE-K live authority flags default to `false`

## Package Inputs

The package uses real dependency hashes:

- npm dependency hash from `package-lock.json`
- Cargo vendor hash from `apps/desktop/src-tauri/Cargo.lock`

When dependencies change, refresh the hashes by intentionally running the build
with a fake hash and copying the hash reported by Nix. Do not leave
`lib.fakeHash` in the completed package.

## Runtime Boundary

Phase G3.5 is still only the package boundary. It does not:

- autostart FORGE
- enable autologin
- replace the user's desktop
- install or require a compositor
- run service control
- mutate host configuration
- call modelruntime
- write semantic memory
- change routes or public APIs
- make FORGE-K live authority

Compositor/session integration is later than this G3.5 package status. The current operator desktop path uses the opt-in `forge-operator` session with labwc as substrate and keeps the Cage `forge-wayland-session` fullscreen path as rollback/test coverage. Later session phases must not change the G3.5 truth that the desktop package is a real Tauri Nix build, and must not enable autologin, remove fallback sessions, mutate host configuration, run service control, call modelruntime, write semantic memory, or make FORGE-K live authority.

## Fallback

The G2 local fallback remains supported:

```sh
FORGE_SHELL_BINARY=/path/to/forge_desktop nix run .#forge-shell-session
```

Development fallback remains:

```sh
nix develop .#desktop
npm install
npm run build:desktop
npm -w @forge/desktop run tauri -- build
```
