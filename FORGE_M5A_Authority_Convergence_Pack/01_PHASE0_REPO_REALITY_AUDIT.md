# Phase 0 Prompt — Repo Reality and Authority Audit

You are working inside `BiteTh3bulleT/project_forge`.

Do not output code in chat. Make changes in files.

## Goal

Before modifying runtime code, establish current repository truth for M5A.

Create/update:

- `docs/reviews/m5a_authority_convergence_review.md`

## Required inspection

Read current authority docs, modelruntime docs, gateway registry, modelruntime API/governance files, Control Lane approval files, HostBridge, and FORGE-H files.

## Required review sections

1. Current branch/HEAD.
2. Current live authority map.
3. Current FORGE-K simulator/live boundary.
4. Gateway/modelruntime capability drift.
5. `model.delete_file` runtime-vs-registry state.
6. `model.chat` and `model.generate` runtime-vs-registry state.
7. Control Lane approval gate current behavior.
8. Autonomy approval bridge current behavior.
9. HostBridge/FORGE-H synchronous sampling status.
10. Existing System Cockpit/System surface status.
11. Deferred/stub/fake/shadow surfaces relevant to this sprint.
12. Tests currently available.
13. Recommended implementation order.

## Acceptance criteria

The review is brutally honest. No feature is marked implemented unless code/tests prove it. Historical docs are treated as historical unless current authority docs point to them. All design-only or deferred surfaces are labeled.

## WHAT NOT TO DO

Do not change runtime behavior in this phase. Do not fix code yet. Do not overclaim.
