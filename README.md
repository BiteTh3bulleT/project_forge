# FORGE

FORGE is a local-first AI workspace for inspectable, approval-gated engineering work.

## Current Authority

Start here for current truth:

- `docs/onboarding.md` - first-read path for collaborators, operators, and future agents.
- `docs/reviews/current_phase_status.md` - current phase status and FORGE-K simulator/live boundary.
- `docs/status/current_authority_sources.md` - map of live authority docs, operator docs, and non-authoritative planning docs.
- `AGENTS.md` - agent working rules, branch/worktree policy, and status guidance.
- `docs/runbooks/current_forge_bringup.md` - current operator bring-up path.

FORGE-K is the target cognitive microkernel architecture, but it is not live daemon authority unless a live path explicitly says so and has integration evidence. Current partial live seams are validation/enforcement surfaces only; they do not make simulator services the owner of memory, retrieval, gateway, modelruntime, routes, or canonical truth.

## What Is Implemented

- Desktop shell: Tauri + React workstation surfaces.
- Core service: Go daemon with AI-OS, gateway, approvals, jobs, audit, memory, retrieval, modelruntime, and operator APIs.
- Modelruntime: governed model management with compatible `/v1/*` routes, chat/SSE streaming where supported, approval-required managed artifact delete-file flow, and disabled-by-default vLLM-compatible external endpoint profile.
- FORGE-K simulator and shared validation packages: deterministic architecture work, simulator phases, and narrow live validation seams documented in current phase status.
- Optional Nix substrate: dev shells, packages, and opt-in shell/session scaffolding. Existing npm/go workflows remain authoritative.

Detailed phase history belongs in `docs/reviews/current_phase_status.md`, not in this README.

## Requirements

- Go `1.24+` (the local/Nix toolchain may be newer)
- Node.js + npm
- Rust toolchain for Tauri desktop builds
- Optional: Nix for flake/dev-shell workflows

## Run

Install dependencies:

```bash
npm install
```

Start the core:

```bash
npm run core
```

Start the desktop:

```bash
npm run desktop
```

Useful validation commands:

```bash
npm test
npm run lint
npm run validate:desktop
npm run validate:local
```

Build commands:

```bash
npm run build
npm run build:core
npm run build:desktop
```

Default endpoints:

- Core API: `http://127.0.0.1:18492`
- Desktop dev server: `http://localhost:1420`
- Default shell route: `#/chat`

## Optional Nix

```bash
nix develop
nix develop .#core
nix develop .#desktop
nix build .#forge-core
nix build .#forge-desktop-shell
nix run .#forge-shell-session
```

Nix remains optional. Do not require vLLM, CUDA, GPU hardware, or a managed model service for default evaluation.

## Repository Layout

- `apps/desktop` - Tauri + React desktop shell.
- `services/core` - Go core service.
- `packages/shared` - shared contracts.
- `packages/ui` - shared UI primitives.
- `docs` - architecture, runbooks, reviews, reports, and status sources.

## More Docs

- `docs/onboarding.md`
- `docs/api/routes.md`
- `docs/USER_MANUAL.md`
- `docs/DESKTOP_SHELL.md`
- `docs/architecture/forge_ai_os.md`
- `docs/architecture/forge_k_overview.md`
- `docs/architecture/model_runtime.md`
- `docs/architecture/nix_substrate.md`
- `docs/runbooks/config_reference.md`
- `docs/runbooks/docker_containerization.md`
