---
name: forge-verify
description: Run FORGE's minimal verification matrix for the current change set and return command-backed evidence.
disable-model-invocation: true
---

# FORGE Verify

Use this skill when the user asks to verify, test, or validate local changes.

## Workflow

1. Discover changed files first.
   - Run `git status --short` and `git diff --name-only`.
2. Select the narrowest checks that cover touched areas.
   - If `services/core/**` changed: run `cd services/core && go test ./...`.
   - If `apps/desktop/**`, `packages/ui/**`, or `packages/shared/**` changed: run `npm run build:desktop`.
   - If root scripts or cross-cutting wiring changed: run `npm run build`.
3. Execute checks and capture evidence.
   - Record command, exit code, and key failure line(s) when a command fails.
4. Return a compact verification report.
   - Include: scope tested, commands executed, pass/fail status, and unresolved blockers.

## Output Contract

- Always list exact commands run.
- Never claim success if a command was skipped.
- If verification is partial, say what is still unverified and why.
