# FORGE Graphical Shell Session

This runbook describes the Phase G1/G2 graphical shell session contract.

G1 defines how an opt-in NixOS session should launch FORGE as the primary visible shell while preserving the existing FORGE authority boundaries. G2 adds a launchable `forge-shell-session` wrapper package and flake app. The wrapper starts an existing local Tauri `forge_desktop` binary when one is available and otherwise fails loudly with exact next steps.

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
- known binary names: `apps/desktop/src-tauri/target/release/forge_desktop` or `apps/desktop/src-tauri/target/debug/forge_desktop`

The Tauri desktop binary is not yet fully Nix-packaged. G2 therefore packages a safe launcher/wrapper, not a full Tauri application derivation.

## Expected Defaults

The session scaffolding remains inert unless explicitly enabled.

| Option | Expected G2 Default |
|---|---|
| `forge.shellSession.enable` | `false` |
| `forge.shellSession.mode` | `"fullscreen-shell"` |
| `forge.shellSession.displayBackend` | `"wayland"` |
| `forge.shellSession.autoStart` | `false` |
| `forge.shellSession.coreURL` | `"http://127.0.0.1:18492"` |
| `forge.shellSession.safeMode` | `true` |

Autologin must remain disabled by default. Existing desktop environments must remain available by default. G2 only supports `fullscreen-shell`; other modes require later design, implementation, tests, and rollback notes.

## Manual Launch

Build the wrapper:

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

If `FORGE_SHELL_BINARY` points to an executable, the wrapper launches it. Otherwise it checks for:

- `apps/desktop/src-tauri/target/release/forge_desktop`
- `apps/desktop/src-tauri/target/debug/forge_desktop`

If neither exists, the wrapper exits non-zero and prints the expected paths plus the current build commands.

## Build The Tauri Shell

The existing desktop workflow remains authoritative until a later phase packages Tauri through Nix:

```bash
npm run build:desktop
npm -w @forge/desktop run tauri -- build
```

For development, continue using:

```bash
npm run desktop
```

G2 does not start `forge-core` for you. Start core separately with `npm run core` or the existing service workflow.

## Safe Session Flow

1. Operator boots NixOS normally.
2. Operator selects or starts the opt-in FORGE shell session.
3. Session metadata points at the existing FORGE desktop shell package/binary when packaging is available.
4. Shell receives `FORGE_CORE_URL` pointing at the local governed core endpoint.
5. Shell renders shell-safe status surfaces and requests structured state through `forge-core`.
6. Any requested action goes through the existing gateway, permissions, lane, approval, audit, controllane, and memory/modelruntime governance paths.

If a shell binary is unavailable, the G2 wrapper fails visibly and safely. It must not fake production behavior.

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

G2 introduces no host mutation, modelruntime mutation, semantic memory mutation, route/API mutation, gateway mutation, or FORGE-K authority mutation.

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
- keep autostart disabled by default
- keep safe mode enabled by default
- keep existing `npm run desktop` and `npm run dev` workflows usable

If the shell session fails, the operator should log into the fallback desktop or TTY and disable the opt-in shell session configuration. Canonical FORGE data under `/forge` must not be deleted as part of shell rollback.

## Known Limitations

- The Nix package is a wrapper, not a complete Tauri derivation.
- G2 does not install or configure a compositor.
- G2 does not replace the desktop session.
- G2 does not enable autostart or autologin.
- Runtime `FORGE_CORE_URL` is exported for shell/session code; the built frontend still uses the existing desktop API configuration behavior until a later runtime-config phase changes it.

## Validation Expectations

G2 validation must include Nix package/app evaluation, wrapper static checks, Nix module evaluation, and documentation review. Runtime phases must continue to add evidence for:

- opt-in disabled-by-default behavior
- no autologin by default
- no existing desktop removal by default
- session launch in `fullscreen-shell`
- safe failure when the shell binary/package is unavailable
- `FORGE_CORE_URL` environment wiring
- no direct host mutation from shell controls
- no direct modelruntime mutation
- no direct semantic memory writes
- no FORGE-K live authority routing
