# FORGE Onboarding

This is the starting path for a new collaborator, operator, or coding agent entering the repository.

## Read first

1. `README.md` — project purpose, current implementation, quick start, and major limitations.
2. `docs/status/CURRENT_STATE.md` — concise current posture, known blockers, and acceptance gates.
3. `AGENTS.md` — binding agent doctrine, scope discipline, validation expectations, and repository rules.
4. `docs/status/current_authority_sources.md` — detailed live authority map.
5. `docs/reviews/current_phase_status.md` — cumulative FORGE-K implementation history and evidence notes.
6. `docs/runbooks/current_forge_bringup.md` — current operator bring-up and diagnosis path.

`CODEX.md`, prompt packs, simulator documents, and future architecture notes are planning context unless a current authority document and implementation evidence explicitly promote them.

## Authority rules

- Production FORGE-K lives under `services/core/internal/forgekernel` and is the sole boot-selectable semantic syscall ingress for current production paths.
- Simulator and target-architecture packages under `services/core/internal/forgek` are not production daemon authority.
- Control Lane remains a temporary bounded validation, apply, and SQLite durable port beneath the production Kernel. It is not an alternate live orchestrator.
- Gateway is the only tool-execution authority. Models cannot select an arbitrary tool, approve execution, bypass capability scope, or claim completion without execution evidence.
- Model output is proposal text and evidence, not canonical truth.
- Durable semantic writes require deterministic validation, authenticated authorization, provenance, journal and audit evidence, commit proof, and governed replay behavior.
- Legacy direct mutation paths remain retired unless a separate reviewed change provides a replacement path, rollback proof, tests, and documentation.
- Unimplemented authority paths fail closed. Do not add “temporary” bypasses that silently become permanent.

## Local setup

Install dependencies:

```bash
npm install
```

Run the core:

```bash
npm run core
```

Run the desktop:

```bash
npm run desktop
```

Run a smoke check:

```bash
npm run smoke
```

The smoke wrapper dispatches to the platform-specific implementation. For the authoritative operator path, use `docs/runbooks/current_forge_bringup.md`.

## Validation

Use the narrowest relevant test while developing, then broaden before requesting review.

Common commands:

```bash
npm run lint
npm test
npm run validate:js
npm run validate:forgek
npm run build:core
npm run docs:routes:check
npm run smoke
```

For desktop or UI changes, also run:

```bash
npm run validate:desktop
npm run build:desktop
```

For authority-sensitive changes, include the closest package tests plus the aggregate gates. Changes to concurrency, approvals, jobs, Gateway, modelruntime, storage, or background services should include race testing where practical.

## Repository map

- `apps/desktop` — Tauri and React operator shell.
- `services/core` — Go daemon and live production services.
- `services/core/internal/forgekernel` — production Kernel authority and staged cutover contracts.
- `services/core/internal/forgek` — simulator and target-architecture work; non-authoritative.
- `services/core/internal/forgekshadow` — disabled-by-default and read-only diagnostic seams.
- `services/core/internal/gateway` — capability, policy, approval, execution, and audit boundary for tools.
- `nix` — Nix/NixOS packages, modules, and host profiles.
- `docs/architecture` — system design; future material must be labeled.
- `docs/status` — concise current posture and gate status.
- `docs/reviews` and `docs/reports` — cumulative reviews, evidence, and historical phase material.
- `docs/runbooks` — operator procedures.
- `docs/api/routes.md` — generated route inventory.

## Before changing code

1. Check the current branch and working-tree state.
2. Read the nearest implementation, tests, and current authority documentation.
3. Confirm whether the subsystem is production-owned, temporary-port-owned, simulator-only, shadow-only, or design-only.
4. Preserve unrelated user and worker changes.
5. Keep the change focused and reversible.
6. Update current-state, authority, runbook, or API documentation when behavior changes.
7. Add or update tests that prove both success and fail-closed behavior.

## Branch and pull-request workflow

Do not push feature, authority, recovery, security, or documentation-overhaul work directly to `main`.

Use a focused branch:

```bash
git switch -c <type>/<short-description>
```

Recommended prefixes include:

- `fix/`
- `feat/`
- `docs/`
- `test/`
- `refactor/`
- `chore/`

Before opening a pull request:

- run the relevant validation commands;
- review the diff for secrets, generated files, stale claims, and accidental authority expansion;
- document tests and any skipped checks;
- state whether the change affects authority, persistence, recovery, tool execution, model visibility, host behavior, or compatibility;
- include rollback or recovery notes for risky changes.

Open a pull request against `main`. Merge only after required checks pass and the operator has reviewed the change.

## Documentation discipline

FORGE has substantial historical material. Preserve useful history, but do not make readers reconstruct the present from dozens of phase notes.

- `docs/status/CURRENT_STATE.md` is the concise current-state entry point.
- `docs/status/current_authority_sources.md` is the detailed authority map.
- `docs/reviews/current_phase_status.md` is cumulative history, not the first-read summary.
- Runbooks must describe executable current procedures.
- Architecture documents must label `LIVE`, `PARTIAL`, `SIMULATOR_ONLY`, `SHADOW_ONLY`, `PLANNED`, or `DESIGN_ONLY` behavior where ambiguity is possible.
- A behavior-changing pull request should update the relevant current document in the same change.

## Security

Read `SECURITY.md` before reporting or handling a vulnerability. Do not place credentials, exploit details, private system paths, model API keys, bearer tokens, or sensitive audit contents in normal issues, pull requests, fixtures, screenshots, or logs.
