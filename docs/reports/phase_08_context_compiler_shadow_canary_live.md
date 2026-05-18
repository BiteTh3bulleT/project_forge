# FORGE-K Online Phase 08 Context Compiler Shadow/Canary/Live Report

## Phase

FORGE-K Online Phase 08 - Context Compiler Shadow/Canary/Live.

## Summary

Phase 08 adds a disabled-by-default, read-only shadow ContextBundle projection inside `forgekshadow`.

Status: `CONTEXT_COMPILER_SHADOW_ONLY / SHADOW_READ_ONLY / DISABLED_BY_DEFAULT / ADMISSION_CANDIDATE_CANARY_ONLY / NO_LIVE_PROMPT_AUTHORITY / NO_LIVE_AUTHORITY_CHANGE`.

The new shadow bundle is generated only from accepted `VALIDATE_ADMISSION_CANDIDATE` validation observations and only from normalized safe refs. It records deterministic bundle shape, block shape, provenance, and a stable hash while explicitly marking the admission status as `candidate_accepted_not_live_admitted`. It does not replace live `COMPILE_CONTEXT`, generate prompt text, call modelruntime, run retrieval/search/embeddings, write memory, admit evidence, execute gateway tools, change routes/APIs or user-visible output, or import FORGE-K simulator Context Compiler services into live authority.

## Files changed

- `services/core/internal/forgekshadow/context_bundle_shadow.go` - adds the typed shadow ContextBundle projection for accepted admission-candidate validation refs.
- `services/core/internal/forgekshadow/context_bundle_shadow_test.go` - covers bundle creation, skip cases, unsafe refs, missing workspace, deterministic hashing, observer/advisory wiring, and no forbidden effects.
- `services/core/internal/forgekshadow/report.go` - carries normalized validation refs and optional shadow ContextBundle reports.
- `services/core/internal/forgekshadow/control_lane_validation.go` - validates normalized refs before storing validation observations.
- `services/core/internal/forgekshadow/observer.go` - attaches shadow ContextBundles to matching Control Lane validation reports.
- `services/core/internal/forgekshadow/advisory.go` - points Context Compiler advisory metadata at the shadow bundle hash when present.
- `services/core/internal/forgekshadow/forbidden_imports_test.go` - blocks simulator Context Compiler imports in the shadow path.
- `services/core/internal/aios/controllane/shadow_validation.go` - passes normalized validation refs into the existing Control Lane shadow observer.
- `services/core/internal/aios/controllane/shadow_validation_test.go` - verifies admission-candidate normalized refs reach the observer without changing syscall behavior.
- `services/core/internal/aios/controllane/forgek_activation_readiness.go` and tests - mark Context Compiler as shadow-only in the read-only authority matrix.
- `docs/reports/phase_08_context_compiler_shadow_canary_live.md` - this report.
- `docs/status/phase_08_context_compiler_shadow_canary_live.md` - Phase 08 status marker.
- `docs/reviews/current_phase_status.md` - current-status note and table entry.
- `docs/status/current_authority_sources.md` - current authority navigation note for the shadow-only Context Compiler surface.

## Tests run

- `cd services/core && go test ./internal/forgekshadow -run "ContextBundle|ControlLaneValidation|Advisory|Forbidden" -count=1` - passed.
- `cd services/core && go test ./internal/aios/controllane -run "Admission|Shadow|ForgeKActivationReadiness" -count=1` - passed.
- `cd services/core && go test ./internal/forgekshadow -count=1` - passed.
- `cd services/core && go test ./internal/aios/controllane -count=1` - passed after moving a new admission-ref assertion from the source-object test to the admission-candidate test.
- `cd services/core && go test ./internal/api -run "TestForgeKernelStatusReadOnlyActivationReadiness|TestForgeSystemStatusReadOnlySurface|Shadow|Context" -count=1` - passed.
- `npm run docs:routes:check` - passed.
- `git diff --check` - passed with expected Windows line-ending warnings.
- `npm run lint` - passed.
- `npm test` - passed.
- `npm run validate:forgek` - passed.
- `npm run build:core` - passed.

## Authority impact

No canonical authority migration. The live shadow owner is `services/core/internal/forgekshadow`; live context compile/restore authority remains in the existing `services/core/internal/aios/controllane` `COMPILE_CONTEXT` path. The target FORGE-K owner remains `forgek.contextcompiler` for future simulator-to-live Context Compiler semantics, but simulator Context Compiler services are not imported or invoked.

The phase adds only a diagnostic ContextBundle shape for one canary source: accepted admission-candidate validation refs. Those refs are not live admitted evidence, and the bundle is not used for prompts, model calls, memory writes, retrieval, or user-visible responses.

## Security impact

Positive validation hardening. The shadow projection revalidates normalized refs through `refvalidation`, rejects unsafe or secret-looking refs, rejects missing workspace identity, avoids raw content, and extends forbidden-import tests so the shadow path cannot import simulator Context Compiler services.

## NixOS impact

No NixOS files, host services, WSL state, or Nix store behavior changed in this phase.

## Rollback path

Revert the Phase 08 commit. Existing live `COMPILE_CONTEXT`, Control Lane validation, retrieval, memory, shadow diagnostics, routes, storage, gateway, and modelruntime behavior can continue without the shadow ContextBundle projection.

## Remaining blockers

- No live Context Compiler prompt authority is enabled.
- No live evidence admission exists; accepted admission-candidate validation remains candidate-only.
- No dedicated public route canary flag or route behavior is added.
- No runtime proposal boundary work is included; that remains Phase 09.
- Phase 00 prompt-vault `PACK_TREE.md` manifest mismatch remains unresolved.
