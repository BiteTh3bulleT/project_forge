# FORGE-K Online Phase 09 Runtime Proposal Boundary Report

## Phase

FORGE-K Online Phase 09 - Runtime Proposal Boundary.

## Summary

Phase 09 wraps successful live modelruntime generation output in a typed proposal-only envelope.

Status: `RUNTIME_PROPOSAL_BOUNDARY / LIVE_MODELRUNTIME_OWNED / MODEL_OUTPUT_PROPOSAL_ONLY / NO_CANONICAL_TRUTH_COMMIT / NO_FORGE_K_RUNTIME_AUTHORITY / NO_LIVE_AUTHORITY_EXPANSION`.

The envelope records runtime/model/provenance/audit metadata, output hash/size, token counts, and explicit no-authority fields. It states that model output is proposal-only and cannot commit canonical truth, admit evidence, execute tools, mutate memory, or bypass semantic syscalls. This phase does not change backend selection, scheduling, route authority, Control Lane commit behavior, gateway execution, retrieval/search/embedding behavior, live KV reuse, or user-visible authority.

## Files changed

- `services/core/internal/modelruntime/types.go` - adds `ProposalEnvelope` and attaches it to `GenerateResult`.
- `services/core/internal/modelruntime/service.go` - builds proposal-only envelopes after successful audited generation.
- `services/core/internal/modelruntime/service_test.go` - verifies proposal identity, provenance, output hash/size, and no-authority flags.
- `services/core/internal/api/model_runtime.go` - exposes proposal metadata in the API-facing chat result contract.
- `services/core/internal/api/model_runtime_bridge.go` - preserves proposal envelopes through synchronous and streaming chat bridge results.
- `services/core/internal/api/model_runtime_bridge_test.go` - verifies bridge chat results preserve proposal-only envelopes.
- `services/core/internal/aios/controllane/forgek_activation_readiness.go` and tests - mark Runtime Boundary as proposal-boundary metadata in the read-only authority matrix.
- `docs/reports/phase_09_runtime_proposal_boundary.md` - this report.
- `docs/status/phase_09_runtime_proposal_boundary.md` - Phase 09 status marker.
- `docs/reviews/current_phase_status.md` - current-status note and table entry.
- `docs/status/current_authority_sources.md` - current authority navigation note for runtime proposal-only output.
- `docs/architecture/model_runtime.md` - modelruntime architecture note for proposal-only output envelopes.
- `docs/architecture/runtime_driver_boundary.md` - simulator/live boundary note for Phase 09.

## Tests run

- `cd services/core && go test ./internal/modelruntime -count=1` - passed.
- `cd services/core && go test ./internal/api -count=1` - passed.
- `cd services/core && go test ./internal/aios/controllane -run ForgeKActivationReadiness -count=1` - passed.
- `rg -n "services/core/internal/forgek/runtime|runtime_syscalls|forgek/runtime" services/core/internal/modelruntime services/core/internal/api -g "*.go"` - returned no matches, confirming no simulator Runtime Boundary import in live modelruntime/API paths.
- `npm run docs:routes:check` - passed.
- `git diff --check` - passed with expected Windows line-ending warnings only.
- `npm run lint` - passed.
- `npm test` - passed.
- `npm run validate:forgek` - passed.
- `npm run build:core` - passed.

## Authority impact

No canonical authority migration. The live owner remains `services/core/internal/modelruntime`. The target FORGE-K owner remains `forgek.runtime` for future simulator-to-live Runtime Boundary semantics, but simulator Runtime Boundary services are not imported or invoked.

The phase adds only typed proposal metadata to successful generation output. Model output remains evidence/proposal material and is not a commit, admission, memory write, tool execution, context compilation, or user-visible authority boundary.

## Security impact

Positive boundary hardening. The proposal envelope makes model output authority explicit and machine-readable: proposal-only, no canonical commit, no truth mutation, no memory mutation, no evidence admission, no gateway execution, no model-output authority, requires Kernel commit, requires validation, and no live authority migration.

## NixOS impact

No NixOS files, host services, WSL state, or Nix store behavior changed in this phase.

## Rollback path

Revert the Phase 09 commit. Existing live modelruntime execution, scheduler, backend calls, audit, API, and chat behavior can continue without proposal envelope metadata.

## Remaining blockers

- No model output admission path is added.
- No live FORGE-K Runtime Boundary authority is added.
- No consensus gate is added; that remains Phase 10.
- Any future use of runtime proposals for admission, consensus, semantic writes, tool execution, or prompt/context authority requires a separate phase with tests and rollback evidence.
- Phase 00 prompt-vault `PACK_TREE.md` manifest mismatch remains unresolved.
