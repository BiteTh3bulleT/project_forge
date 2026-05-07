# Nix Substrate

_Phase N1 — light Nix foundation. Phase N2 adds private NixOS host
substrate scaffolding. Phase N3 adds the read-only Host Kernel Bridge
diagnostic library._

This document explains how Nix is used in FORGE today and what it is
**not** used for yet. Nix's role in FORGE will grow over time, but in
this phase it is deliberately thin and optional.

---

## 1. Why FORGE uses Nix

FORGE aspires to be a local-first AI-OS. That makes reproducibility a
first-order property: every operator should be able to bring up
identical core, desktop, and AI-OS toolchains regardless of their host
OS or package manager. Nix is the substrate we intend to grow into that
role — starting with developer tooling, eventually reaching tool
capsules and service composition.

## 2. What Nix does in Phase N1

- Defines pinned dev shells for `core`, `desktop`, `aios`, and a
  `default` shell for general work.
- Declares a `forge-core` package (Go `buildGoModule`) so the core
  service can be built reproducibly from the flake.
- Exposes basic `nix flake check` entrypoints for Go tests, `go vet`,
  and a JS build step — with honest skip paths where sandboxing blocks
  real execution (see §9).
- Publishes a Nixpkgs overlay (`overlays.default`) that surfaces
  `forge-core` for downstream consumers.

## 3. What Nix does not do yet

- **No gateway integration.** Nix does not define or gate tool
  invocations.
- **No autonomy integration.** Autonomy charters do not reference Nix.
- **No tool capsules.** `nix/tool-capsules/` is a README scaffold only.
- **No NixOS modules.** `nix/modules/` and `nix/profiles/` are
  README scaffolds only.
- **No release snapshots or deployment.** Service deployment still uses
  the existing `npm run up` / systemd path.
- **No replacement of npm/go/cargo workflows.** Everything the repo did
  before still works without Nix.
- **Not mandatory.** Contributors without Nix can continue as before.

## 4. How this supports FORGE AI-OS

Nix is the future substrate for:

- hermetic **tool capsules** backing the AI-OS tool surface,
- **service composition** for self-hosted deployments,
- deterministic **release snapshots** and operator-reproducible
  environments.

Phase N1 prepares the ground by making the repo build cleanly under
Nix without committing to any of those integrations yet.

## 5. Dev shells

Four shells are exposed:

| Shell | Purpose | Core contents |
|---|---|---|
| `default` | General repo work | Go, Node, sqlite, jq, rg, fd, nixfmt |
| `core` | Go-only forge-core work | Go + gopls + sqlite + ripgrep |
| `desktop` | Tauri/desktop development | Node, rustc, cargo, Tauri Linux deps (GTK/WebKit) |
| `aios` | AI-OS architecture work | Go + Node + sqlite + graphviz |

Desktop shell includes Linux WebKit/GTK deps conditionally on
`stdenv.isLinux`. On Darwin, Tauri uses system frameworks.

## 6. Packages

- `forge-core` — Go service at `services/core`. Exposed as both the
  default package and the default app (`bin/core`).
- `forge-desktop` — **deferred**. Tauri packaging requires resolving
  npm dependency vendoring (`npmDepsHash`) and cargo vendoring
  (`cargoHash`), plus Linux WebKit/GTK runtime wiring. See
  `docs/status/desktop_nix_packaging_gap.md`.

### forge-core hash status

`vendorHash` is currently real (non-placeholder) in
`nix/packages/forge-core.nix`.

`nix build .#forge-core` is still failing in a clean flake context
because some VSA source inputs (`services/core/internal/memory/vsa_*.go`)
must be present and are still tracked through optional-path assumptions in
the current package derivation.
Until that source parity gap is closed, treat Forge core as:

- **Buildable in repository workflows** with non-flake source state.
- **Not yet authoritative under clean-flake mode** for all paths.

The dev shells and existing `npm run build:core` remain the
authoritative build path for now.

## 7. Checks

`nix flake check` runs:

- `go-tests` — `go test ./...` in `services/core`.
- `go-vet` — `go vet ./...` in `services/core`.
- `js-build` — `npm run build:desktop`.

All three report concrete failures when required inputs are missing in the
active environment. For authoritative non-Nix validation today, run:

```sh
cd services/core && go test ./... && go vet ./...
npm install && npm run build
```

## 8. Future tool capsules

See `nix/tool-capsules/README.md`. Capsules will map FORGE AI-OS tool
capabilities to hermetic Nix environments — but only after the
authoritative gateway/autonomy path is clear.

## 9. NixOS modules

Phase N2 supersedes the Phase N1 placeholder-only status by adding
private NixOS module scaffolding under `nix/nixos/modules/`.

Exposed modules:

- `nixosModules.forge-os`
- `nixosModules.forge-services`
- `nixosModules.forge-storage`
- `nixosModules.forge-host-kernel`

These modules are opt-in scaffolds. They define `/forge` directories,
a `forge` user/group, a default-safe `forge-core` systemd service shape,
and a disabled-by-default read-only Host Kernel Bridge report directory
and environment file.

Phase N3 implements the Host Kernel Bridge diagnostic snapshot library
in `services/core/internal/hostbridge`. The NixOS module remains
read-only and does not add a timer, route, modelruntime call, rebuild
action, service restart, or autonomous host mutation.

They do not migrate live authority, fork Linux, execute tools, add
routes, mutate modelruntime behavior, or make Nix mandatory.

## 10. Relationship to gateway / tool surface

None in Phase N1. The tool gateway
(`docs/architecture/tool_gateway.md`) continues to operate without any
Nix awareness. A future phase will let the gateway reference capsule
identities — that contract does not exist yet and must not be
anticipated in Nix files.

## 11. Relationship to autonomy

None in Phase N1. Autonomy charters
(`docs/architecture/autonomy_charters.md`) neither reference nor depend
on Nix. Any autonomy-controlled Nix execution environment is Phase N5
work.

## 12. Relationship to runtime authority cutover

Nix in Phase N1 wraps the existing authoritative build commands. Deep
Nix integration (capsules, modules, release snapshots) remains deferred
until the runtime cutover evidence is stable on a Nix-enabled host.

## 13. Command reference

```sh
# Enter a dev shell
nix develop                 # default shell
nix develop .#core          # Go-only
nix develop .#desktop       # Tauri/desktop
nix develop .#aios          # AI-OS architecture

# Build packages
nix build .#forge-core      # produces ./result/bin/core

# Run the app
nix run .#forge-core

# Checks
nix flake check

# Format Nix files
nix fmt
```

Running the flake for the first time will generate `flake.lock` if it
is not already committed.
