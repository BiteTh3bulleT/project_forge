# ADR 0014 - FORGE Workstation Substrate and Nix Mutation Proposals

Status: Accepted

Date: 2026-05-12

## Context

FORGE is becoming a graphical workstation shell, but NixOS/Linux remains the boot, hardware, display, service, package, and rollback substrate. The operator needs visibility into runtime, storage, GPU, modelruntime, Nix generation, safe-mode, and proposal posture without giving the shell, models, FORGE-H, or FORGE-K simulator direct host authority.

ADR 0013 is already assigned to the G6 operator desktop decision, so this workstation/Nix mutation decision uses ADR 0014.

## Decision

FORGE may generate governed Nix mutation proposals, but may not apply host mutations without explicit operator approval, build proof, rollback proof, and a later controlled adapter implementation.

Nix mutation proposals are advisory evidence until they pass review. The graphical shell may display proposal state, build evidence, rollback evidence, VM smoke evidence, and approval state. It must not directly invoke host commands.

The FORGE Workstation substrate remains an opt-in composition target above NixOS. It may define profiles, recovery expectations, backend posture, and read-only cockpit surfaces. It does not replace Linux, NixOS, normal desktop fallback, TTY fallback, gateway authority, modelruntime authority, semantic memory authority, or FORGE-K simulator/live separation.

## Accepted Boundaries

- Linux owns kernel, drivers, devices, filesystems, cgroups, boot, and process substrate.
- NixOS owns declarative host configuration, packages, services, generations, display/session plumbing, and rollback.
- `forge-core` remains CPU/RAM authoritative for FORGE daemon governance.
- FORGE-H may recommend resource and Nix posture changes, but it cannot mutate the host.
- modelruntime may expose backend profile state and governed inference APIs, but models remain drivers.
- Tauri shell may render read-only status and submit governed requests; it cannot run host commands.
- FORGE-K remains non-live authority unless a future migration phase explicitly proves otherwise.
- GPU/CUDA/VRAM paths are acceleration surfaces, not truth authority.
- Qdrant and Redis do not become canonical memory.

## Rejected Alternatives

- AI directly edits host config and rebuilds.
- Tauri shell directly invokes system commands.
- FORGE-H mutates NixOS or service state.
- FORGE-K simulator becomes live authority.
- GPU/CUDA lane owns canonical truth.
- Qdrant or Redis becomes canonical memory.
- vLLM becomes the only modelruntime backend.
- Workstation session replaces the user's desktop by default.

## Consequences

Positive:

- FORGE gains a concrete workstation architecture without gaining unsafe host authority.
- Future Nix changes can be reviewed, built, tested, approved, and rolled back through explicit evidence.
- Operator visibility can expand through the System cockpit while remaining read-only by default.
- GPU/vLLM/CUDA work can proceed as governed acceleration, not truth authority.

Negative:

- Host mutation remains deferred until proposal, build, VM, rollback, approval, audit, and adapter work exists.
- Workstation profile implementation will require careful NixOS test coverage and recovery runbooks.
- Operators may need to distinguish implemented shell surfaces from design-only cockpit panels.

## Rollback Posture

Rollback must be known before apply. A future adapter must capture or reference:

- current Nix generation;
- exact proposal diff;
- build result;
- VM smoke result;
- operator approvals;
- rollback plan;
- post-apply health evidence;
- audit and journal refs.

The operator must retain TTY access, a non-FORGE desktop/session fallback during adoption, and a manual NixOS rollback command path.

## Future Work

- Durable `NixMutationProposal` records.
- Proposal transition validation tests.
- Sandbox build runner.
- VM smoke-test adapter.
- Read-only cockpit panel for Nix generation and proposal state.
- Governed apply adapter after proposal/build/rollback proof exists.
- Workstation NixOS profile and recovery specializations.
