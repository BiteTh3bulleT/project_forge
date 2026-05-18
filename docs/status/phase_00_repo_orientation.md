# FORGE-K Online Phase 00 Repo Orientation Status

## Phase

FORGE-K Online Phase 00 - Repo Orientation.

## Status marker

`CLOSED / DOCS_ONLY / ORIENTATION / NO_LIVE_AUTHORITY_CHANGE`

## Summary

The FORGE-K Online prompt vault was copied into `docs/prompt-packs/FORGE-K_Online_Prompt_Vault/` as inert planning context. Current live authority remains with existing AI-OS, Control Lane, gateway, modelruntime, memory/retrieval, audit, approval, and operator runbook paths.

## Live owner

Live owner map remains unchanged:

- API routes: `services/core/internal/api`
- semantic mutation: `services/core/internal/aios/controllane`
- tool execution: `services/core/internal/gateway`
- model generation/governance: `services/core/internal/modelruntime`
- memory/retrieval: `services/core/internal/memory`, `services/core/internal/retrieval`
- audit/approval: `services/core/internal/audit`, `services/core/internal/approvals`
- operator bring-up and host actions: `docs/runbooks/current_forge_bringup.md`, `docs/runbooks/config_reference.md`, NixOS/operator authority

## Target FORGE-K owner

The prompt vault describes the target FORGE-K online ladder:

`SIMULATOR_ONLY -> SHADOW_READ_ONLY -> VALIDATION_ONLY -> DISABLED_BY_DEFAULT_LIVE -> OPERATOR_APPROVED_LIVE -> DEFAULT_LIVE -> LEGACY_PATH_RETIRED`

Phase 00 does not move any subsystem along that ladder. It only records the planning pack and current owner map.

## Authority impact

No live authority expansion. No simulator service import. No route/API change. No modelruntime, gateway, retrieval, memory, audit, approval, or NixOS behavior change.

## Tests/evidence

- Prompt vault structure reviewed from `START_HERE.md`, `_run/MERGE_INSTRUCTIONS.md`, `prompts/phases/phase_00_repo_orientation.md`, `ai_context/CURRENT_TRUTH.md`, `ai_context/AUTHORITY_BOUNDARIES.md`, and `architecture/FORGE_K_ONLINE_STRATEGY.md`.
- PowerShell manifest verification found one mismatch for `PACK_TREE.md`.
- Final validation commands are recorded in `docs/reports/phase_00_repo_orientation.md`.

## Rollback

Revert the Phase 00 docs commit or remove the prompt-pack copy and the Phase 00 report/status references. There is no live runtime state to roll back.

## Blockers

- Resolve or regenerate the `PACK_TREE.md` manifest mismatch before treating the pack as fully integrity-clean.
- Review pack-level `AGENTS.md`, `.cursor/rules/*`, ADR templates, and status templates manually before any activation.

## Next phase

Run exactly one next phase prompt after operator selection. Do not chain multiple FORGE-K Online phases into a single implementation commit.
