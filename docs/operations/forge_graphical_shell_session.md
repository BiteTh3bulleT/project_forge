# FORGE Graphical Shell Session

This runbook describes the Phase G1/G2/G3/G3.5 graphical shell session contract.

G1 defines how an opt-in NixOS session should launch FORGE as the primary visible shell while preserving the existing FORGE authority boundaries. G2 adds a launchable `forge-shell-session` wrapper package and flake app. G3 adds the target package contract for a Nix-built desktop shell package named `forge-desktop-shell`, while preserving the G2 safe wrapper and local-binary fallback behavior.

G3.5 is the real Tauri Nix build phase. `packages.forge-desktop-shell` has been changed from a launcher placeholder to a real Tauri build derivation, advertises `passthru.containsTauriBinary = true`, and builds the `forge_desktop` binary plus the stable `forge-desktop-shell` wrapper.

## Operator Meaning

FORGE graphical shell means FORGE is the desktop shell for a FORGE-OS session:

- desktop/workspace surface
- launcher and command palette
- system context surface
- approvals and notifications
- resource and service status
- governed access to `forge-core`

It does not mean a browser dashboard controlling a remote or headless server.

## Desktop Shell Discovery

The current FORGE graphical shell implementation is the Tauri desktop workspace:

- workspace package: `apps/desktop/package.json`, package name `@forge/desktop`
- Tauri config: `apps/desktop/src-tauri/tauri.conf.json`
- Rust package: `apps/desktop/src-tauri/Cargo.toml`, package name `forge_desktop`
- development launch: `npm run desktop`
- frontend build: `npm run build:desktop`
- Tauri build path: `npm -w @forge/desktop run tauri -- build`
- expected Tauri binary name: `forge_desktop`
- stable shell package command: `forge-desktop-shell`
- local binary names: `apps/desktop/src-tauri/target/release/forge_desktop` or `apps/desktop/src-tauri/target/debug/forge_desktop`

G3/G3.5's packaging target is `packages.forge-desktop-shell`, optionally with `apps.forge-desktop-shell`, exposing a stable `/bin/forge-desktop-shell` command. If the underlying Tauri binary remains named `forge_desktop`, the stable command should be a wrapper around that binary with the same safe shell-mode environment defaults as `forge-shell-session`.

Current G3.5 package status is packaged and validated on Linux. The package-level signal for wrapper preference is `passthru.containsTauriBinary = true`, so `forge-shell-session` prefers the Nix package after the explicit `FORGE_SHELL_BINARY` override.

## Expected Defaults

The session scaffolding remains inert unless explicitly enabled.

| Option | Expected G3.5 Default |
|---|---|
| `forge.shellSession.enable` | `false` |
| `forge.shellSession.mode` | `"fullscreen-shell"` |
| `forge.shellSession.displayBackend` | `"wayland"` |
| `forge.shellSession.autoStart` | `false` |
| `forge.shellSession.coreURL` | `"http://127.0.0.1:18492"` |
| `forge.shellSession.safeMode` | `true` |

Autologin must remain disabled by default. Existing desktop environments must remain available by default. G3.5 only supports `fullscreen-shell`; other modes require later design, implementation, tests, and rollback notes.

## Build And Run The Desktop Shell Package

Build the G3.5 desktop shell package:

```bash
nix build .#forge-desktop-shell
```

Run the packaged shell command:

```bash
nix run .#forge-desktop-shell
```

Current result: `nix build .#forge-desktop-shell` succeeds and produces `/bin/forge-desktop-shell` plus the underlying `/bin/forge_desktop` Tauri binary.

The direct app run is valid only after the package builds:

```bash
nix run .#forge-desktop-shell
```

The package-level `FORGE_DESKTOP_SHELL_BINARY` override remains available for controlled wrapper testing. For normal local fallback, prefer the session-level `FORGE_SHELL_BINARY` override or run the local Tauri binary directly.

Session-level fallback with a locally built desktop binary:

```bash
FORGE_SHELL_BINARY=/path/to/forge_desktop nix run .#forge-shell-session
```

The expected package output is:

- `/bin/forge-desktop-shell`, the stable command used by operators and wrappers
- `/bin/forge_desktop`, the Tauri binary name from `apps/desktop/src-tauri`

The current G3.5 limitation is explicit: the package is validated as a Linux Nix build only. Compositor/session integration, autostart, desktop replacement, and multi-monitor display-manager integration remain future phases.

Build the wrapper and safety checks:

```bash
nix build .#forge-shell-session
nix flake check
```

If the local Nix requires explicit flakes support:

```bash
nix --extra-experimental-features 'nix-command flakes' build .#forge-desktop-shell
nix --extra-experimental-features 'nix-command flakes' build .#forge-shell-session
nix --extra-experimental-features 'nix-command flakes' flake check
```

## Npm And Cargo Hash Updates

When refreshing the G3.5 package dependencies, update hashes through failing Nix builds instead of hardcoding fake values.

For npm dependencies, use the repository lock file that covers the `apps/desktop` workspace:

```bash
nix-prefetch-npm-deps package-lock.json
```

Then update the npm dependency hash used by the desktop shell derivation.

For Cargo dependencies, start with the Tauri crate lock file at `apps/desktop/src-tauri/Cargo.lock`. Set the desktop shell derivation's cargo hash to `lib.fakeHash` or the equivalent fake hash for the chosen builder, then run:

```bash
nix build .#forge-desktop-shell
```

Copy the hash reported by Nix into the derivation. Repeat until npm and Cargo dependency hashes are real and the package builds. Do not leave `lib.fakeHash` in the completed package.

## Linux Runtime Dependencies

The existing desktop preflight checks for these Linux `pkg-config` names:

- `webkit2gtk-4.1`
- `javascriptcoregtk-4.1`
- `gtk+-3.0`

The Nix desktop development shell supplies the broader Tauri Linux set:

- `pkg-config`
- `openssl`
- `glib`
- `gtk3`
- `libsoup_3`
- `webkitgtk_4_1`
- `librsvg`
- `libayatana-appindicator`
- `patchelf`

The `forge-desktop-shell` package builds and wraps the Tauri binary with the required Linux WebKit/GTK runtime libraries. `nix develop .#desktop` and `npm run desktop` remain the development workflow.

## Manual Launch

Build the session wrapper:

```bash
nix build .#forge-shell-session
```

Run the wrapper from the repository root:

```bash
nix run .#forge-shell-session
```

Override the governed local core URL when needed:

```bash
FORGE_CORE_URL=http://127.0.0.1:18492 nix run .#forge-shell-session
```

The same override applies to the direct package command:

```bash
FORGE_CORE_URL=http://127.0.0.1:18492 nix run .#forge-desktop-shell
```

This direct package command requires the real package build to succeed first.

The wrapper exports safe shell-session defaults before launching:

- `FORGE_SHELL_SESSION_ENABLED=true`
- `FORGE_SHELL_MODE=fullscreen-shell`
- `FORGE_CORE_URL=http://127.0.0.1:18492` unless overridden
- `VITE_FORGE_API_URL=$FORGE_CORE_URL` for existing desktop code paths
- `FORGE_SHELL_SAFE_MODE=true`
- `FORGE_SHELL_FULLSCREEN=true`
- `FORGE_SHELL_HOST_MUTATION=false`
- `FORGE_SHELL_DIRECT_SYSTEM_CONTROL=false`
- `FORGE_SHELL_MODEL_MUTATION=false`
- `FORGE_SHELL_SEMANTIC_MEMORY_WRITE=false`
- `FORGE_SHELL_FORGE_K_LIVE_AUTHORITY=false`

Override the exact desktop shell binary when needed:

```bash
FORGE_SHELL_BINARY=/path/to/forge_desktop nix run .#forge-shell-session
```

`FORGE_SHELL_BINARY` must point to an executable. If it is set but not executable, the wrapper must exit non-zero and name the invalid path.

## Binary Selection Order

`forge-shell-session` must choose the desktop shell binary in this order:

1. If `FORGE_SHELL_BINARY` is set and executable, launch it.
2. Else, if a Nix-provided desktop shell binary from `packages.forge-desktop-shell` is wired into the wrapper and advertises `passthru.containsTauriBinary = true`, launch that stable command.
3. Else, check the local Tauri release binary:
   `apps/desktop/src-tauri/target/release/forge_desktop`.
4. Else, check the local Tauri debug binary:
   `apps/desktop/src-tauri/target/debug/forge_desktop`.
5. Else, fail loudly with the missing paths and the build commands.

The G2 fallback remains mandatory. G3.5 may prefer the Nix-built package only when it contains the real Tauri binary; it must not remove `FORGE_SHELL_BINARY` override support or local Tauri binary discovery.

The local fallback paths are:

- `apps/desktop/src-tauri/target/release/forge_desktop`
- `apps/desktop/src-tauri/target/debug/forge_desktop`

If none of the candidates exist, the wrapper exits non-zero and prints the expected paths plus the current build commands.

## Build The Tauri Shell

The existing desktop workflow remains the fallback when the G3.5 Nix/Tauri derivation is unavailable or incomplete:

```bash
nix develop .#desktop
npm install
npm run build:desktop
npm -w @forge/desktop run tauri -- build
```

For development, continue using:

```bash
npm run desktop
```

G3.5 does not start `forge-core` for you. Start core separately with `npm run core` or the existing service workflow.

## Safe Session Flow

1. Operator boots NixOS normally.
2. Operator selects or starts the opt-in FORGE shell session.
3. Session metadata points at the existing FORGE desktop shell package/binary when packaging is available.
4. Shell receives `FORGE_CORE_URL` pointing at the local governed core endpoint.
5. Shell renders shell-safe status surfaces and requests structured state through `forge-core`.
6. Any requested action goes through the existing gateway, permissions, lane, approval, audit, controllane, and memory/modelruntime governance paths.

If a shell binary is unavailable, the wrapper fails visibly and safely. It must not fake production behavior.

## Allowed Operations

The shell may:

- display local service health
- display resource posture from governed diagnostics when available
- display approval queues when supported by existing APIs
- display model/runtime status when supported by existing APIs
- display memory/journal browser views through governed read paths
- submit operator-initiated requests to `forge-core`
- show bounded placeholders for unavailable surfaces

## Forbidden Operations

The shell must not:

- run `systemctl` directly
- run `nixos-rebuild` directly
- enable autologin
- install, start, or require a compositor
- replace the user's existing desktop/session choices
- load or unload kernel modules
- restart host services
- mutate NixOS configuration
- clean filesystems
- load or unload models directly
- write semantic memory directly
- bypass gateway execution authority
- bypass approvals
- treat model output as canonical truth
- use FORGE-K simulator services as live authority

G3.5 introduces no host mutation, modelruntime mutation, semantic memory mutation, route/API mutation, gateway mutation, compositor integration, autologin, desktop replacement, or FORGE-K authority mutation.

## System Context Handling

The shell may observe structured system/session context:

- active workspace
- open panels/windows
- current project
- resource posture
- service health
- model status
- approval state
- recent errors
- user-triggered actions

This is operating awareness for the shell. It is not automatic LLM prompt context.

The context compiler decides what subset reaches model calls. Raw full system state, raw logs, raw process lists, raw window state, raw host diagnostics, and raw memory contents must not be dumped into prompts.

## Fallback

Implementations must preserve a fallback path:

- keep a normal desktop/session available
- keep TTY login available
- keep shell session opt-in
- keep desktop replacement disabled
- keep compositor installation/configuration out of G3.5
- keep autostart disabled by default
- keep safe mode enabled by default
- keep existing `npm run desktop` and `npm run dev` workflows usable
- keep `FORGE_SHELL_BINARY=/path/to/forge_desktop forge-shell-session` usable
- keep local `target/release` and `target/debug` Tauri binary fallback usable

If the packaged G3.5 shell fails, run the G2 wrapper against a known local binary:

```bash
FORGE_SHELL_BINARY="$PWD/apps/desktop/src-tauri/target/release/forge_desktop" nix run .#forge-shell-session
```

If `nix run .#forge-shell-session` is unavailable because the package build fails on a local machine, run the local Tauri workflow directly:

```bash
npm run desktop
```

Or run a locally built Tauri binary directly:

```bash
"$PWD/apps/desktop/src-tauri/target/release/forge_desktop"
```

If the shell session fails, the operator should log into the fallback desktop or TTY and disable the opt-in shell session configuration. Canonical FORGE data under `/forge` must not be deleted as part of shell rollback.

Rollback is configuration rollback only:

- leave the normal desktop/session available
- disable the opt-in `forge.shellSession.enable` setting if it was enabled locally
- keep `forge.shellSession.autoStart = false`
- remove any local `FORGE_SHELL_BINARY` overrides that point to broken binaries
- keep `/forge` data intact
- do not run cleanup, package-manager mutation, service-control, or modelruntime commands as part of shell rollback

## Known Limitations

- G3.5 packages the Linux Tauri app with real npm and Cargo hashes and WebKit/GTK runtime wrapping.
- The stable package command is `/bin/forge-desktop-shell`; the underlying Tauri binary is `/bin/forge_desktop`.
- The current G3.5 package sets `passthru.containsTauriBinary = true` and builds the real Tauri binary.
- `forge-shell-session` must remain useful with `FORGE_SHELL_BINARY` and local release/debug fallbacks if the package is unavailable on a local machine.
- G3.5 does not install or configure a compositor.
- G3.5 does not replace the desktop session.
- G3.5 does not enable autostart or autologin.
- G3.5 does not run service-control, package-manager, kernel-module, modelruntime, or direct memory-write operations.
- Runtime `FORGE_CORE_URL` is exported for shell/session code; the built frontend still uses the existing desktop API configuration behavior until a later runtime-config phase changes it.
- Compositor/session integration remains future G4 work.

## Validation Expectations

G3.5 validation must include Nix package/app evaluation, desktop shell package checks, wrapper static checks, Nix module evaluation, and documentation review. Runtime phases must continue to add evidence for:

- opt-in disabled-by-default behavior
- no autologin by default
- no existing desktop removal by default
- no compositor installation or requirement in G3.5
- `packages.forge-desktop-shell` exposes a stable command and contains a real Tauri binary
- `forge-shell-session` skips desktop-shell packages that do not advertise `passthru.containsTauriBinary = true`
- `packages.forge-shell-session` and `apps.forge-shell-session` remain exposed
- `forge-shell-session` prefers `FORGE_SHELL_BINARY`, then the Nix-provided shell, then local Tauri release/debug paths
- session launch in `fullscreen-shell`
- safe failure when the shell binary/package is unavailable
- `FORGE_CORE_URL` environment wiring
- no direct host mutation from shell controls
- no direct modelruntime mutation
- no direct semantic memory writes
- no FORGE-K live authority routing
