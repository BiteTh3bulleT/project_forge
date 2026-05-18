# FORGE-K Online Phase 06 Courthouse Admission Report

## Phase

FORGE-K Online Phase 06 - Courthouse Admission.

## Summary

Phase 06 wires the Phase 03 pure `admissionvalidation` contract into the existing live Control Lane as `VALIDATE_ADMISSION_CANDIDATE`.

Status: `ADMISSION_CANDIDATE_ONLY / CONTROL_LANE_OWNED / NO_EVIDENCE_ADMISSION / NO_CANONICAL_TRUTH_COMMIT`.

The new action validates admission candidate shape, evidence/source/policy/provenance refs, case/workspace identity, admission mode, and forbidden authority claims. It returns a structured validation report through syscall state summary, audit payloads, and the existing diagnostic shadow observer path. It does not admit or reject evidence, issue Courthouse rulings, write canonical truth, mutate memory, call modelruntime, execute gateway tools, run retrieval/search/embeddings, compile context, change routes/APIs, or import FORGE-K simulator Courthouse services into live authority.

## Files changed

- `services/core/internal/aios/domain/types.go` - adds `VALIDATE_ADMISSION_CANDIDATE`.
- `services/core/internal/aios/controllane/admission_validation.go` - maps live syscall payloads into the pure admission validation contract and emits no-effect summaries.
- `services/core/internal/aios/controllane/admission_validation_test.go` - covers success, forbidden authority claims, cross-workspace evidence refs, capability denial, audit, and no forbidden effects.
- `services/core/internal/aios/controllane/{registry,capabilities,validator,processor,processor_apply,audit,shadow_validation,forgek_activation_readiness}.go` - registers and wires the validation-only action, audit summary, shadow diagnostic input, and read-only readiness metadata.
- Control Lane tests updated to include the new action in the registry, activation contract, payload validation matrix, shadow observer, and readiness matrix.
- `docs/reports/phase_06_courthouse_admission.md` - this report.
- `docs/status/phase_06_courthouse_admission.md` - Phase 06 status marker.
- `docs/reviews/current_phase_status.md` - current-status note and table entry.

## Tests run

- `cd services/core && go test ./internal/admissionvalidation -count=1` - passed.
- `cd services/core && go test ./internal/aios/controllane -run "Admission|ForgeKActivation|RegistryIncludes|ControlLaneValidationObserverCalledForAdmission" -count=1` - passed.
- `cd services/core && go test ./internal/aios/controllane -count=1` - passed.
- `cd services/core && go test ./internal/api -run TestForgeKernelStatusReadOnlyActivationReadiness -count=1` - passed.
- `cd services/core && go test ./internal/api -run TestForgeSystemStatusReadOnlySurface -count=1` - passed.
- `npm run docs:routes:check` - passed.
- `git diff --check` - passed with expected Windows line-ending warnings.
- `npm test` - passed after updating API/kernel/system status expectations for the sixth read-only validation action.
- `npm run lint` - passed.
- `npm run validate:forgek` - passed.
- `npm run build:core` - passed.

## Authority impact

No canonical authority migration. The live owner remains `services/core/internal/aios/controllane`. The target FORGE-K owner remains `forgek.court` for future simulator-to-live Courthouse semantics, but simulator Courthouse services are not imported or invoked.

The phase brings only an admission candidate validation surface online. Evidence admission, rejection, ruling authority, canonical truth commit, and governed semantic mutation routing remain future work.

## Security impact

Positive validation hardening. The action rejects malformed admission candidates, unsafe refs, cross-workspace refs, and explicit authority claims such as evidence admission, memory mutation, gateway execution, modelruntime calls, retrieval/search/embedding execution, context compilation, and live authority migration.

## NixOS impact

No NixOS files, host services, WSL state, or Nix store behavior changed in this phase.

## Rollback path

Revert the Phase 06 commit. Existing live evidence handling, Control Lane mutation paths, routes, storage, gateway, modelruntime, retrieval, and memory behavior can continue without the admission candidate validation action.

## Remaining blockers

- No live evidence admission or rejection authority is enabled.
- No FORGE-K Courthouse ruling service is live.
- No Memory Palace evidence mirror is implemented; that starts in Phase 07.
- No Context Compiler shadow/canary/live work is included; that remains Phase 08.
- Phase 00 prompt-vault `PACK_TREE.md` manifest mismatch remains unresolved.
