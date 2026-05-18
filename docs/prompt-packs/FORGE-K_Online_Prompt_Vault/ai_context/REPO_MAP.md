# Repo Map

This file is a planning map. Verify paths against the actual repo before editing.

## Core backend

- `services/core/main.go` — core entrypoint
- `services/core/internal/api` — HTTP routes and service composition
- `services/core/internal/aios/controllane` — live semantic validation/mutation owner
- `services/core/internal/gateway` — tool execution authority
- `services/core/internal/permissions` — active permission profiles
- `services/core/internal/lanes` — action lanes and scoped execution policy
- `services/core/internal/audit` — audit/provenance records
- `services/core/internal/modelruntime` — model driver governance
- `services/core/internal/memory` — memory observations/VSA/evidence surfaces
- `services/core/internal/retrieval` — retrieval runs/results
- `services/core/internal/forgek` — FORGE-K simulator packages
- `services/core/internal/forgekshadow` — read-only shadow diagnostics

## Desktop

- `apps/desktop` — React/Tauri operator shell
- `apps/desktop/src-tauri` — Tauri Rust host commands and capabilities

## Nix

- `flake.nix`
- `nix/packages`
- `nix/nixos/modules`
- `nix/nixos/profiles`
- `nix/nixos/configurations`
- `nix/checks`

## Docs

- `AGENTS.md`
- `docs/reviews/current_phase_status.md`
- `docs/status/current_authority_sources.md`
- `docs/status/implementation_matrix.md`
- `docs/architecture/*`
- `docs/adr/*`
- `docs/runbooks/*`
