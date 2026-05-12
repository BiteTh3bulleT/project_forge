# ADR 0014 - FORGE Workstation Substrate and Nix Mutation Proposals

Status: Accepted

Date: 2026-05-12

## Context

FORGE is becoming a graphical workstation shell, but NixOS/Linux remains the substrate. The operator needs visibility into runtime, storage, GPU, model, and Nix posture without giving the shell direct host authority.

ADR 0013 is already assigned to the G6 operator desktop decision, so this workstation/Nix mutation decision uses ADR 0014.

## Decision

FORGE may generate governed Nix mutation proposals, but may not apply host mutations without explicit operator approval, build proof, rollback proof, and a later controlled adapter implementation.

Nix mutation proposals are advisory evidence until they pass review. The graphical shell may display proposal state, build evidence, rollback evidence, and approval state. It must not directly invoke host commands.

## Consequences

- NixOS remains the source of boot and rollback truth.
- FORGE can reason about workstation changes without gaining unreviewed host authority.
- Future mutation support requires proposal records, approvals, audit, sandbox build proof, VM smoke evidence, rollback plans, and a governed host adapter.
- Operators retain manual recovery paths.

## Accepted Boundaries

- FORGE-H can recommend resource and Nix posture changes.
- Modelruntime can expose backend profile state.
- FORGE-K remains non-live authority unless a future migration phase explicitly proves otherwise.
- Qdrant and Redis do not become canonical memory.

## Rejected Alternatives

- AI directly edits host config and rebuilds.
- Tauri shell directly invokes system commands.
- FORGE-K simulator becomes live authority.
- GPU/CUDA lane owns canonical truth.
- Qdrant or Redis becomes canonical memory.

## Rollback Posture

Rollback must be known before apply. A future adapter must capture the current Nix generation, build result, VM smoke result, approval refs, and post-apply health evidence.

## Future Work

- Durable NixMutationProposal records.
- Sandbox build runner.
- VM smoke-test adapter.
- Read-only cockpit panel for Nix generation and proposal state.
- Governed apply adapter after proposal/build/rollback proof exists.
