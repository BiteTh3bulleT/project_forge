# FORGE-K Online Phase 00 Repo Orientation Report

## Phase

FORGE-K Online Phase 00 - Repo Orientation.

## Summary

Phase 00 imported the `FORGE-K_Online_Push` prompt vault as inert planning context under `docs/prompt-packs/FORGE-K_Online_Prompt_Vault/` and mapped the current live authority owners before any implementation work.

This phase is `DOCS_ONLY / ORIENTATION / NO_LIVE_AUTHORITY_CHANGE`. It does not activate the prompt pack's root `AGENTS.md`, `.cursor/rules`, ADR templates, status files, or phase prompts as repository authority.

## Files changed

- `docs/prompt-packs/FORGE-K_Online_Prompt_Vault/` - inert copy of the FORGE-K Online prompt vault.
- `docs/reports/phase_00_repo_orientation.md` - this phase report.
- `docs/status/phase_00_repo_orientation.md` - Phase 00 status marker.
- `docs/status/current_authority_sources.md` - planning-context pointer.
- `docs/reviews/current_phase_status.md` - concise current-status note and table entry.

## Tests run

- PowerShell required-structure check for `docs/prompt-packs/FORGE-K_Online_Prompt_Vault/` - passed, 13 required entries present.
- PowerShell manifest SHA-256 check for `docs/prompt-packs/FORGE-K_Online_Prompt_Vault/MANIFEST.json` - failed only for `PACK_TREE.md`; mismatch recorded as blocker.
- `npm run docs:routes:check` - passed.
- `git diff --check` - passed with expected Windows line-ending warnings.
- `npm test` - passed.
- `npm run lint` - passed.
- `npm run validate:forgek` - passed.
- `npm run build:core` - passed.

## Tests not run

- Desktop validation was not run because Phase 00 made no desktop/UI changes.
- Nix checks were not run because Phase 00 made no Nix files or host-substrate changes.

## Authority impact

No authority change.

Current live owners remain:

| Area | Current live owner | Target FORGE-K owner or boundary |
|---|---|---|
| API routes | `services/core/internal/api` | Future governed entrypoints only after explicit migration phase. |
| Semantic mutation | `services/core/internal/aios/controllane` | Future FORGE-K Kernel/syscall path after separate authority migration. |
| Tool execution | `services/core/internal/gateway` | Gateway remains execution authority; FORGE-K proposals must not bypass it. |
| Model runtime | `services/core/internal/modelruntime` | Runtime drivers remain governed modelruntime boundary; FORGE-K runtime work is proposal-only until migrated. |
| Memory and retrieval | `services/core/internal/memory`, `services/core/internal/retrieval` | Future Courthouse/Memory Palace migration requires read-only/shadow/validation gates first. |
| Audit and approvals | `services/core/internal/audit`, `services/core/internal/approvals` | Journal/provenance integration remains future work and must preserve existing audit gates. |
| Host/NixOS substrate | NixOS/operator runbooks | Host mutation stays operator-governed; no prompt-pack import may mutate host state. |

## Security impact

No runtime security posture changed. The imported prompt vault is documentation/planning material only.

The pack integrity check found one manifest mismatch for `PACK_TREE.md`; this is recorded as an input-integrity blocker before treating the imported pack as a verified artifact. The mismatch does not affect live code because the pack is not executable authority.

## NixOS impact

No NixOS files, host services, display-manager state, boot flow, or shell wrappers changed.

## Rollback path

Revert the Phase 00 docs commit or remove:

- `docs/prompt-packs/FORGE-K_Online_Prompt_Vault/`
- `docs/reports/phase_00_repo_orientation.md`
- `docs/status/phase_00_repo_orientation.md`
- the small planning-context/status references added to current authority docs

No database, daemon, container, modelruntime, gateway, memory, retrieval, or NixOS migration is involved.

## Remaining blockers

- `FORGE-K_Online_Push/MANIFEST.json` does not match `PACK_TREE.md`; regenerate or explain `PACK_TREE.md` before claiming full pack integrity.
- Phase prompts must still be run one at a time. Phase 00 does not authorize Phase 01 implementation.
- Active `AGENTS.md`, `.cursor/rules/*`, ADR templates, and status files from the pack require manual review before any activation.
