# FORGE-K Online Phase 08 Context Compiler Shadow/Canary/Live Status

## Phase

FORGE-K Online Phase 08 - Context Compiler Shadow/Canary/Live.

## Status marker

`CONTEXT_COMPILER_SHADOW_ONLY / SHADOW_READ_ONLY / DISABLED_BY_DEFAULT / ADMISSION_CANDIDATE_CANARY_ONLY / NO_LIVE_PROMPT_AUTHORITY / NO_LIVE_AUTHORITY_CHANGE`

## Summary

The existing disabled-by-default `forgekshadow` path can now generate a typed shadow ContextBundle shape from accepted `VALIDATE_ADMISSION_CANDIDATE` validation metadata.

## Live owner

`services/core/internal/forgekshadow` owns the disabled-by-default read-only Context Compiler shadow projection. Existing live context compile/restore behavior remains owned by `services/core/internal/aios/controllane` legacy `COMPILE_CONTEXT` paths.

## Target FORGE-K owner

FORGE-K Context Compiler (`services/core/internal/forgek/contextcompiler`) remains the target owner for future deterministic ContextBundle semantics. This phase does not import or invoke simulator Context Compiler services as live prompt authority.

## Canary scope

The canary scope is one existing Control Lane validation action: accepted `VALIDATE_ADMISSION_CANDIDATE` observations. That validation remains candidate-only; it is not live evidence admission.

## Authority impact

No live `COMPILE_CONTEXT` replacement. No prompt authority. No modelruntime call. No retrieval/search/embedding execution. No memory write. No evidence admission or rejection. No gateway/tool execution. No route/API behavior change. No user-visible output change. No live authority migration.

## Tests/evidence

Validation commands are recorded in `docs/reports/phase_08_context_compiler_shadow_canary_live.md`.

## Rollback

Revert the Phase 08 commit to remove the shadow ContextBundle projection, observer ref transfer, tests, and docs. Existing live `COMPILE_CONTEXT`, retrieval, memory, modelruntime, gateway, and shadow diagnostics can continue without data or host rollback.

## Blockers

- Live prompt/context authority remains outside FORGE-K Context Compiler.
- Accepted admission-candidate validation is not evidence admission.
- Shadow ContextBundle shape is diagnostic metadata only and is not used for model prompts.
- Dedicated public route canarying remains future work and needs a separate flag, tests, and rollback evidence.

## Next phase

Run Phase 09 as a separate bounded commit. Do not combine runtime proposal boundary work into this phase.
