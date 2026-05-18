# AGENTS — FORGE-K Online Prompt Vault

## Mission

Move FORGE-K from simulator/shadow/validation status toward live operational authority **one governed seam at a time**.

## Non-negotiable doctrine

- FORGE-K simulator packages are not live daemon authority until explicitly migrated.
- Live daemon authority remains with existing owners until a phase moves one specific seam.
- Do not import simulator services like `forgek.Kernel`, simulator Courthouse, simulator Context Compiler, simulator RuntimeService, or simulator syscalls into live daemon authority paths.
- Extract shared pure contracts first.
- Live integrations must preserve the existing live owner unless the phase explicitly migrates that owner.
- Models are drivers. Model output is proposal/evidence, not truth.
- Gateway remains the only live tool execution authority.
- NixOS remains the host mutation substrate.
- FORGE-K must not run `nixos-rebuild`, `systemctl`, package managers, host mutation commands, or destructive operations directly.
- Dangerous actions require explicit operator approval.

## Required workflow

Every agent must use the three-pass loop:

1. **Pass 1 — Understand and map**: read current docs, map files, identify authority owner, plan narrow change.
2. **Pass 2 — Execute smallest safe change**: implement only the requested phase; no opportunistic rewrites.
3. **Pass 3 — Verify and report**: run tests, update docs, produce evidence and blockers.

## Required output after every phase

- Summary
- Files changed
- Authority impact
- Tests run
- Tests not run and why
- Rollback path
- Remaining blockers

## What not to do

- Do not mark scaffolds as complete.
- Do not skip tests because the change "looks obvious."
- Do not make broad refactors during authority migration.
- Do not change route/API behavior unless the phase explicitly requires it.
- Do not collapse memory, retrieval, modelruntime, gateway, and kernel authority into one blob.
