# AI_CONTEXT

Agents must treat this prompt vault as a repo execution harness, not casual notes.

## Required reads before implementation

- `AGENTS.md`
- `SKILLS.md`
- `ai_context/CURRENT_TRUTH.md`
- `ai_context/AUTHORITY_BOUNDARIES.md`
- `architecture/FORGE_K_ONLINE_STRATEGY.md`
- `process/three_pass_execution.md`
- `process/do_not_touch.md`

## Binding constraints

- Preserve live authority boundaries.
- Do not overclaim FORGE-K live status.
- Do not modify dangerous paths without explicit phase instruction.
- Do not bypass gateway, permissions, approvals, audit, or NixOS substrate controls.
- Every phase must leave validation evidence.
