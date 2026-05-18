# KV Reuse Canary

Status date: 2026-05-18.

Status: `KV_REUSE_CANARY_VALIDATION_ONLY / EXACT_FINAL_TOKEN_IDENTITY_REQUIRED / CONTROL_LANE_OWNED / NO_BACKEND_REUSE / NO_FORGE_K_KV_AUTHORITY_MIGRATION`.

## Intent

FORGE-K Online Phase 15 opens the smallest possible live canary for deterministic KV reuse: a validation-only signal after exact identity proof. It does not enable backend KV tensor reuse.

## Live Owner

The live canary is owned by `services/core/internal/aios/controllane` through `VALIDATE_KV_IDENTITY`. The pure identity validator remains `services/core/internal/kvidentity`.

## Target Owner

`services/core/internal/forgek/kv` remains the target owner for future KV manifest and reuse authority. This phase does not import or invoke simulator KV services as live daemon authority.

## Canary Contract

A canary request is accepted only when:

- the payload explicitly asks for live reuse
- the payload includes `kvReuseCanary=true`
- the payload includes `canary_path=control_lane_validation_only`
- both manifest and request use `STRICT_PREFIX`
- both manifest and request provide matching non-empty `final_token_ids_hash`
- all existing model, tokenizer, template, prompt-layout, policy/syscall schema, context, token, runtime-assumption, cache-salt, cache-mode, and manifest-availability gates pass

Accepted canary metadata records:

- `canaryKVReuse=true`
- `liveKVReuse=false`
- `backendReuse=false`
- `runtimeMutation=false`
- `memoryMutation=false`

## Boundary

This is a canary identity proof, not live backend reuse. It does not store, load, or reuse KV tensors. It does not change modelruntime, runtime scheduling, Context Compiler behavior, gateway execution, evidence admission, memory writes, or route/API authority.

Any future backend reuse must be a separate phase with a real tensor store, backend-specific invalidation, runtime cancellation semantics, operator visibility, and rollback evidence.
