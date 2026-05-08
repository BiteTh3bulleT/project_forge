# Desktop Nix Packaging Status

_Status: Phase G3 launcher package target identified; real Tauri binary packaging remains incomplete._

Packaging the FORGE desktop/Tauri shell under `apps/desktop` is now the
Phase G3 target. The intended flake package is `packages.forge-desktop-shell`,
with a stable executable at `/bin/forge-desktop-shell` and, if retained,
the underlying Tauri binary at `/bin/forge_desktop`.

This status file is deliberately conservative: the G3 package is complete only
when it contains or wraps a real Tauri `forge_desktop` binary and advertises
that through package metadata such as `passthru.containsTauriBinary = true`.
A stable wrapper package with `passthru.containsTauriBinary = false` is an
honest placeholder, not completed desktop shell packaging.

## Blockers

1. **npm dependency hash (`npmDepsHash`) not captured.**
   `apps/desktop` and the workspace root have no Nix-compatible npm
   dependency lock resolution. A Nix build would need a
   `buildNpmPackage`-style derivation with the correct hash for
   `package-lock.json`. None of that has been computed yet.

2. **Cargo vendor hash not captured.**
   `apps/desktop/src-tauri/Cargo.lock` would need either
   `cargoHash` (via `rustPlatform.buildRustPackage`) or a
   `cargoDepsName` derivation. Not yet computed.

3. **Tauri-specific wiring.**
   Tauri bundles a frontend (`vite build`) and a Rust binary, then
   produces a platform-native app image (`.AppImage`, `.deb`, `.dmg`).
   Nixpkgs has patterns for this (`cargo-tauri`, explicit `wrapProgram`
   for WebKit deps), but applying them requires the hashes above plus
   Linux-specific dependency wiring (GTK, WebKit, libayatana-appindicator).

4. **Platform-specific.**
   Darwin packaging would require `CoreServices` / `WebKit` framework
   availability, which this repo has not tested under Nix.

## G3 Shell Session Contract

The `forge-shell-session` wrapper must prefer shell binaries in this order:

1. `FORGE_SHELL_BINARY`, when set and executable.
2. A Nix-provided `forge-desktop-shell` command, when wired into the wrapper
   and marked with `passthru.containsTauriBinary = true`.
3. `apps/desktop/src-tauri/target/release/forge_desktop`.
4. `apps/desktop/src-tauri/target/debug/forge_desktop`.
5. Loud failure with exact build and override instructions.

The G2 local-binary fallback remains required in G3. Placeholder packages with
`passthru.containsTauriBinary = false` must be skipped by `forge-shell-session`.
`FORGE_CORE_URL` remains an operator override, defaulting to
`http://127.0.0.1:18492`, and the wrapper must preserve safe shell-mode
environment defaults.

## Current Safe Fallback

- `nix/shells/desktop.nix` provides the full Tauri toolchain for
  interactive development. Running `nix develop .#desktop` and then
  `npm install && npm -w @forge/desktop run tauri -- build` produces a
  working binary on Linux.
- `packages.forge-shell-session` and `apps.forge-shell-session` provide the
  safe wrapper path.
- `packages.forge-desktop-shell` may provide a stable placeholder command that
  accepts `FORGE_DESKTOP_SHELL_BINARY=/path/to/forge_desktop` for direct
  testing, while still reporting that it does not contain the real Tauri
  binary.
- Until `packages.forge-desktop-shell` contains the real Tauri binary, operators
  should use `FORGE_SHELL_BINARY=/path/to/forge_desktop forge-shell-session`,
  `FORGE_DESKTOP_SHELL_BINARY=/path/to/forge_desktop forge-desktop-shell`, or
  the existing `npm run desktop` workflow.

## What's needed to finish

1. Replace the placeholder package internals with a real Nix-built Tauri
   `forge_desktop` binary or wrapper around that binary.
2. Set package metadata so `forge-shell-session` can distinguish real package
   contents from placeholder launchers, for example
   `passthru.containsTauriBinary = true`.
3. Keep `packages.forge-shell-session` preference logic while preserving
   `FORGE_SHELL_BINARY` and local Tauri fallbacks.
4. Compute `npmDepsHash` via `prefetch-npm-deps package-lock.json`.
5. Compute `cargoHash` by running a first `buildRustPackage` with
   `cargoHash = lib.fakeHash` and capturing the reported hash.
6. Update `nix/packages/forge-desktop-shell.nix` using either:
   - `rustPlatform.buildRustPackage` with a `frontendBuild`
     pre-hook that calls `vite build`, or
   - a two-stage derivation (JS closure + cargo-tauri invocation).
7. Wrap the resulting binary with Linux WebKit/GTK runtime deps and stable
   shell-mode defaults.
8. Add checks that verify package exposure, stable executable presence or
   honest failure, wrapper precedence, fallback behavior, and forbidden
   host/runtime mutation strings.

## Expected command sequence (once unblocked)

```sh
# Compute npm hash
nix-prefetch-npm-deps apps/desktop/package-lock.json

# First cargo-vendored build with fake hash - read the reported hash
nix build .#forge-desktop-shell

# Update cargoHash/npmDepsHash in nix/packages/forge-desktop-shell.nix

# Full build
nix build .#forge-desktop-shell
./result/bin/forge-desktop-shell
```

## Why this matters later (not now)

When tool capsules (Phase N3) come online, the desktop app is a
candidate for one of the first capsule-aware surfaces (operator
workstation profile). Until then, desktop work is well-served by the
dev shell and `npm run desktop`.
