# FORGE Onboarding

This is the starting path for a new collaborator, future operator session, or agent picking up the repo.

## Read First

1. `README.md` for the project shape and basic commands.
2. `AGENTS.md` for binding agent doctrine, branch/worktree policy, and validation expectations.
3. `docs/status/current_authority_sources.md` for the current live authority map.
4. `docs/reviews/current_phase_status.md` for FORGE-K simulator/live boundary status.
5. `docs/reports/FORGE_PUNCHLIST.md` for active engineering work.

`CODEX.md` is a future implementation vision. It is planning context, not current daemon truth.

## Authority Rules

- FORGE-K simulator packages under `services/core/internal/forgek` are not live daemon authority.
- Production FORGE-K lives under `services/core/internal/forgekernel`; K20A makes it the default semantic syscall ingress owner while Control Lane remains the temporary durable commit adapter.
- Gateway, modelruntime, approvals, audit, memory, retrieval, API, and the temporary Control Lane adapter retain their documented live roles until each staged cutover gate closes.
- Model output is proposal/evidence text, not canonical truth.
- Durable semantic writes need deterministic validation, journal/audit/provenance, and governed commit boundaries.
- Legacy direct mutation paths stay retired unless a separate phase provides replacement, rollback proof, tests, and docs.

## Local Setup

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

The smoke wrapper is cross-platform and dispatches to the PowerShell or shell implementation.

## Validation

Use the narrowest relevant test while developing, then broaden before commit.

Common commands:

```bash
npm run lint
npm test
npm run validate:js
npm run validate:forgek
npm run build:core
npm run docs:routes:check
```

For desktop/UI changes, also run:

```bash
npm run validate:desktop
```

## Where Things Live

- `apps/desktop` - Tauri and React operator shell.
- `services/core` - Go daemon and live authority paths.
- `services/core/internal/forgek` - FORGE-K simulator and target architecture work.
- `services/core/internal/forgekernel` - production FORGE-K authority boundary and staged cutover code.
- `services/core/internal/forgekshadow` - disabled-by-default/read-only diagnostic shadow seams.
- `docs/architecture` - system design.
- `docs/status` - current posture and gate status.
- `docs/reports` - reviews, phase reports, and punchlists.
- `docs/runbooks` - operator procedures.
- `docs/api/routes.md` - generated route inventory.

## Before Changing Code

- Check `git status --short`.
- Read the nearest existing tests and docs for the subsystem.
- Preserve user or worker changes already in the tree.
- Keep edits scoped to the task.
- Update status/report docs when behavior, authority, or operator posture changes.
- Commit and push completed work to `main` unless the operator explicitly says otherwise.
