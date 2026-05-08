# FORGE Graphical Shell Session

This runbook describes the Phase G1/G2/G3 graphical shell session contract.

G1 defines how an opt-in NixOS session should launch FORGE as the primary visible shell while preserving the existing FORGE authority boundaries. G2 adds a launchable `forge-shell-session` wrapper package and flake app. G3 adds the target package contract for a Nix-built desktop shell package named `forge-desktop-shell`, while preserving the G2 safe wrapper and local-binary fallback behavior.

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

G3's packaging target is `packages.forge-desktop-shell`, optionally with `apps.forge-desktop-shell`, exposing a stable `/bin/forge-desktop-shell` command. If the underlying Tauri binary remains named `forge_desktop`, the stable command should be a wrapper around that binary with the same safe shell-mode environment defaults as `forge-shell-session`.

Current G3 package status may be an honest launcher placeholder until the package contains a Nix-built Tauri binary. The package-level signal for wrapper preference is `passthru.containsTauriBinary = true`; while it remains `false`, `forge-shell-session` must continue to use `FORGE_SHELL_BINARY` and local Tauri release/debug fallbacks.

## Expected Defaults

The session scaffolding remains inert unless explicitly enabled.

| Option | Expected G3 Default |
|---|---|
| `forge.shellSession.enable` | `false` |
| `forge.shellSession.mode` | `"fullscreen-shell"` |
| `forge.shellSession.displayBackend` | `"wayland"` |
| `forge.shellSession.autoStart` | `false` |
| `forge.shellSession.coreURL` | `"http://127.0.0.1:18492"` |
| `forge.shellSession.safeMode` | `true` |

Autologin must remain disabled by default. Existing desktop environments must remain available by default. G3 only supports `fullscreen-shell`; other modes require later design, implementation, tests, and rollback notes.

## Build The Desktop Shell Package

Build the G3 desktop shell package:

```bash
nix build .#forge-desktop-shell
```

Run the packaged shell directly when the flake exposes an app:

```bash
nix run .#forge-desktop-shell
```

When using the placeholder package directly with a locally built desktop binary:

```bash
FORGE_DESKTOP_SHELL_BINARY=/path/to/forge_desktop nix run .#forge-desktop-shell
```

The expected package output is:

- `/bin/forge-desktop-shell`, the stable command used by operators and wrappers
- optionally `/bin/forge_desktop`, the Tauri binary name from `apps/desktop/src-tauri`, once the derivation contains the real Tauri binary

If the Tauri derivation is not complete on the current branch or host, the package may expose the stable command as a loud-failing launcher placeholder. A valid G3 limitation is an explicit failure caused by uncaptured `npmDepsHash`, uncaptured Cargo vendor hash, missing WebKit/GTK/Tauri runtime wiring, or platform-specific Tauri packaging work. It is not valid to claim that the package contains the real Tauri shell until the package produces or wraps that binary and advertises it through its package metadata.

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

The G2 fallback remains mandatory. G3 may prefer the Nix-built package only when it contains the real Tauri binary; it must not remove `FORGE_SHELL_BINARY` override support or local Tauri binary discovery.

The local fallback paths are:

- `apps/desktop/src-tauri/target/release/forge_desktop`
- `apps/desktop/src-tauri/target/debug/forge_desktop`

If none of the candidates exist, the wrapper exits non-zero and prints the expected paths plus the current build commands.

## Build The Tauri Shell

The existing desktop workflow remains the fallback when the G3 Nix/Tauri derivation is unavailable or incomplete:

```bash
npm run build:desktop
npm -w @forge/desktop run tauri -- build
```

For development, continue using:

```bash
npm run desktop
```

G3 does not start `forge-core` for you. Start core separately with `npm run core` or the existing service workflow.

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

G3 introduces no host mutation, modelruntime mutation, semantic memory mutation, route/API mutation, gateway mutation, compositor integration, autologin, desktop replacement, or FORGE-K authority mutation.

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
- keep compositor installation/configuration out of G3
- keep autostart disabled by default
- keep safe mode enabled by default
- keep existing `npm run desktop` and `npm run dev` workflows usable
- keep `FORGE_SHELL_BINARY=/path/to/forge_desktop forge-shell-session` usable
- keep local `target/release` and `target/debug` Tauri binary fallback usable

If the packaged G3 shell fails, run the G2 wrapper against a known local binary:

```bash
FORGE_SHELL_BINARY="$PWD/apps/desktop/src-tauri/target/release/forge_desktop" nix run .#forge-shell-session
```

If the shell session fails, the operator should log into the fallback desktop or TTY and disable the opt-in shell session configuration. Canonical FORGE data under `/forge` must not be deleted as part of shell rollback.

## Known Limitations

- G3 may still be limited by Nix/Tauri packaging work: npm dependency hashing, Cargo vendor hashing, Linux WebKit/GTK runtime dependencies, and platform-specific Tauri bundle behavior.
- The stable package command is expected to be `/bin/forge-desktop-shell`; the underlying Tauri binary may remain `/bin/forge_desktop`.
- The current G3 package can be an honest placeholder with `passthru.containsTauriBinary = false`; in that state, `forge-shell-session` should skip the package and use explicit/local binary fallbacks.
- `forge-shell-session` must remain useful while `forge-desktop-shell` is launcher-only.
- G3 does not install or configure a compositor.
- G3 does not replace the desktop session.
- G3 does not enable autostart or autologin.
- G3 does not run service-control, package-manager, kernel-module, modelruntime, or direct memory-write operations.
- Runtime `FORGE_CORE_URL` is exported for shell/session code; the built frontend still uses the existing desktop API configuration behavior until a later runtime-config phase changes it.

## Validation Expectations

G3 validation must include Nix package/app evaluation, desktop shell package checks, wrapper static checks, Nix module evaluation, and documentation review. Runtime phases must continue to add evidence for:

- opt-in disabled-by-default behavior
- no autologin by default
- no existing desktop removal by default
- no compositor installation or requirement in G3
- `packages.forge-desktop-shell` exposes a stable command and either contains a real Tauri binary or fails honestly with documented limitations
- `forge-shell-session` skips placeholder desktop-shell packages that do not advertise `passthru.containsTauriBinary = true`
- `packages.forge-shell-session` and `apps.forge-shell-session` remain exposed
- `forge-shell-session` prefers `FORGE_SHELL_BINARY`, then the Nix-provided shell, then local Tauri release/debug paths
- session launch in `fullscreen-shell`
- safe failure when the shell binary/package is unavailable
- `FORGE_CORE_URL` environment wiring
- no direct host mutation from shell controls
- no direct modelruntime mutation
- no direct semantic memory writes
- no FORGE-K live authority routing
