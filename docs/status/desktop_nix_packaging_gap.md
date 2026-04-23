# Desktop Nix Packaging Gap

_Status: deferred in Phase N1. Tracked for Phase N2._

Packaging `forge-desktop` (the Tauri app under `apps/desktop`) as a
reproducible Nix derivation is **not** attempted in Phase N1. This
document explains why and what it would take.

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

## What we have instead (Phase N1)

- `nix/shells/desktop.nix` provides the full Tauri toolchain for
  interactive development. Running `nix develop .#desktop` and then
  `npm install && npm -w @forge/desktop run tauri -- build` produces a
  working binary on Linux.
- No `packages.forge-desktop` is exposed in `flake.nix` — exposing it
  broken would mislead operators.

## What's needed to finish

1. Compute `npmDepsHash` via `prefetch-npm-deps package-lock.json`.
2. Compute `cargoHash` by running a first `buildRustPackage` with
   `cargoHash = lib.fakeHash` and captuing the reported hash.
3. Write `nix/packages/forge-desktop.nix` using either:
   - `rustPlatform.buildRustPackage` with a `frontendBuild`
     pre-hook that calls `vite build`, or
   - a two-stage derivation (JS closure + cargo-tauri invocation).
4. Wrap the resulting binary with Linux WebKit/GTK runtime deps.
5. Add `nix/checks/desktop-build.nix` that exercises the full chain.
6. Expose `forge-desktop` from `flake.nix` only after a successful
   local build.

## Expected command sequence (once unblocked)

```sh
# Compute npm hash
nix-prefetch-npm-deps apps/desktop/package-lock.json

# First cargo-vendored build with fake hash — read the reported hash
nix build .#forge-desktop

# Update cargoHash/npmDepsHash in nix/packages/forge-desktop.nix

# Full build
nix build .#forge-desktop
./result/bin/forge-desktop
```

## Why this matters later (not now)

When tool capsules (Phase N3) come online, the desktop app is a
candidate for one of the first capsule-aware surfaces (operator
workstation profile). Until then, desktop work is well-served by the
dev shell and `npm run desktop`.
