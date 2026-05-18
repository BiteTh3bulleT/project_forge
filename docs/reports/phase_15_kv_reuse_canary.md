# FORGE-K Online Phase 15 KV Reuse Canary Report

## Phase

FORGE-K Online Phase 15 - KV Reuse Canary.

## Summary

Phase 15 enables a validation-only exact-identity KV reuse canary through the existing live Control Lane `VALIDATE_KV_IDENTITY` action.

Status: `KV_REUSE_CANARY_VALIDATION_ONLY / EXACT_FINAL_TOKEN_IDENTITY_REQUIRED / CONTROL_LANE_OWNED / NO_BACKEND_REUSE / NO_FORGE_K_KV_AUTHORITY_MIGRATION`.

The canary accepts only explicit requests with `kvReuseCanary=true`, `canary_path=control_lane_validation_only`, `STRICT_PREFIX` cache mode, matching non-empty `final_token_ids_hash`, and passing identity gates. The accepted canary records `canaryKVReuse=true` but keeps backend/runtime effects disabled.

## Files changed

- `services/core/internal/aios/controllane/kv_enforcement.go` - adds exact-identity canary acceptance while keeping backend reuse and runtime mutation false.
- `services/core/internal/aios/controllane/kv_identity_test.go` - proves accepted canary metadata and rejection without final token identity.
- `services/core/internal/aios/controllane/forgek_activation_readiness.go` and tests - update the KV System readiness row to canary validation-only.
- `docs/status/phase_15_kv_reuse_canary.md` - Phase 15 status marker.
- `docs/reports/phase_15_kv_reuse_canary.md` - this report.
- `docs/architecture/kv_reuse_canary.md` - architecture boundary note.
- `docs/architecture/context_compiler_and_kv_cache.md` - current KV status update.
- `docs/reviews/current_phase_status.md` - current-status note and table entry.
- `docs/status/current_authority_sources.md` - current authority navigation note.

## Tests run

- `cd services/core && go test ./internal/aios/controllane -run "KVIdentity|KVIdentityEnforcement|ForgeKActivationReadiness" -count=1` - passed.
- `cd services/core && go test ./internal/kvidentity -count=1` - passed.
- `rg -n "services/core/internal/forgek/kv|forgek/kv|kv_syscalls|KVService" services/core/internal/aios/controllane services/core/internal/kvidentity -g "*.go" -g "!*_test.go"` - returned no matches, confirming no simulator KV import in live Control Lane or shared validator production paths.
- `cd services/core && go test ./internal/aios/controllane -count=1` - passed.
- `npm run docs:routes:check` - passed.
- `git diff --check` - passed with expected Windows line-ending warnings only.
- `npm run lint` - passed.
- `npm test` - passed.
- `npm run validate:forgek` - passed.
- `npm run build:core` - passed.

## Authority impact

No backend KV reuse is enabled. The canary is an identity-proof signal only. It does not store, load, or reuse live KV tensors; it does not call modelruntime; it does not alter context compilation; and it does not make `services/core/internal/forgek/kv` live authority.

## Security impact

Positive hardening for future reuse. The canary rejects token-input-hash placeholder reuse and requires exact final token identity before any canary reuse signal can be accepted.

## NixOS impact

No NixOS files, host services, WSL state, or Nix store behavior changed in this phase.

## Rollback path

Revert the Phase 15 commit. All live reuse requests return to unsupported status, while baseline KV identity validation remains available.

## Remaining blockers

- Backend/runtime KV reuse remains disabled.
- No live KV tensor store exists.
- FORGE-K KV simulator is not live authority.
- Phase 00 prompt-vault `PACK_TREE.md` manifest mismatch remains unresolved.
