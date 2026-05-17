# FORGE Onboarding

This is the short starting path for a new developer, collaborator, or future operator returning to the repo.

## Start Here

Read in this order:

1. [AGENTS.md](../AGENTS.md) - working doctrine, branch/worktree policy, build/test commands, and safety rules.
2. [current_authority_sources.md](status/current_authority_sources.md) - current map of live authority docs and non-authoritative planning material.
3. [current_phase_status.md](reviews/current_phase_status.md) - current implementation status and FORGE-K simulator/live boundary.
4. [FORGE_PUNCHLIST.md](reports/FORGE_PUNCHLIST.md) - active product and engineering punch list.
5. [current_forge_bringup.md](runbooks/current_forge_bringup.md) - operator path for starting, verifying, and shutting down current FORGE.

## Current Mental Model

FORGE is a local-first AI workspace with governed core services, approvals, jobs, audit, memory, retrieval, modelruntime, gateway, and desktop/operator surfaces.

FORGE-K is the target cognitive microkernel architecture and simulator-first implementation. The simulator under `services/core/internal/forgek` is not live daemon authority. Narrow shared validation/enforcement seams may exist in live Control Lane paths, but they do not make FORGE-K own memory, retrieval, gateway, modelruntime, routes, APIs, or canonical truth.

When in doubt, trust current authority docs over older roadmaps, prompts, and archived reviews.

## Practical First Pass

- Use `git status --short` before editing. Other workers may have active changes.
- Keep changes bounded to the requested files and do not clean up other people's branches, worktrees, or edits.
- For code changes, preserve the existing authority boundaries: model output proposes, gateway executes tools, Control Lane/syscalls validate and commit durable truth.
- For docs-only changes, include validation evidence such as link checks or grep output in the handoff.

## Bring-Up And Validation

Common commands:

```sh
npm install
npm run smoke
npm test
npm run lint
npm run validate:local
```

Use [current_forge_bringup.md](runbooks/current_forge_bringup.md) for the authoritative bring-up path and troubleshooting notes. Do not treat optional Nix, VM, GPU, vLLM, or shell-session work as required for default development unless the task says so.

## Where Work Usually Lands

- Core service: `services/core`
- Desktop shell: `apps/desktop`
- Shared contracts/UI: `packages/shared`, `packages/ui`
- Architecture/status/runbooks: `docs`
- Current task and phase truth: [current_authority_sources.md](status/current_authority_sources.md), [current_phase_status.md](reviews/current_phase_status.md), and [FORGE_PUNCHLIST.md](reports/FORGE_PUNCHLIST.md)

Keep onboarding lean. If this page starts becoming a second README or a phase report, link to the authority doc instead.
