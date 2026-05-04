# Context Compiler and Deterministic KV Cache

Status: Phase 0 architecture baseline.

The context compiler turns admitted meaning into deterministic token-addressed ContextBlocks. The KV cache accelerates exact reusable token shapes. Neither context shape nor KV reuse is canonical memory.

## ContextBlock

A ContextBlock is a deterministic prompt unit emitted by the compiler. It should include:

- stable block id
- block kind
- source object ids
- source hashes
- admission references
- prompt layout version
- policy/syscall schema version
- token count
- final token ids or token hash references

## Stable Prompt Layout

Stable prompt layout means the same admitted inputs, compiler version, policy schema, and layout version produce the same ordered token sequence. Layout stability is required for deterministic cache eligibility and replayable context inspection.

## Expansion/Contraction Context Compiler Loop

The compiler runs an expansion/contraction loop:

1. Expand from query, case, workspace scope, Memory Palace rooms, anchors, and routes.
2. Submit retrieved candidates as exhibits.
3. Contract through Courthouse admission, deterministic budgets, relevance thresholds, contradiction status, and policy constraints.
4. Emit ContextBlocks with stable ordering and citations.
5. Hash final token IDs for cache eligibility.

In the current FORGE architecture, this doctrine extends the `COMPILE_CONTEXT` path. Restore scoring, fresh-compile fallback, persisted snapshot evidence, `restore_scores_json`, and `resume_hints_json` remain non-canonical shape or evidence unless later committed through semantic syscalls.

## Token Hashing

Token hashing is performed over final token IDs, not source text alone. Text identity is insufficient because tokenizer revision, chat template, and prompt layout can change final tokens.

## Cache Eligibility

Cache eligibility requires deterministic identity at the token and runtime assumption level. Eligible cache entries must be represented by a KVCacheManifest.

This is inference/runtime KV reuse, not the existing restore scoring cache. Restore scoring metadata is shape/evidence for context selection; Deterministic KV Cache is acceleration for exact reusable token shapes.

## KVCacheManifest

A KVCacheManifest records:

- manifest id
- model id
- model revision
- tokenizer id
- tokenizer revision
- chat template id/version
- prompt layout version
- policy/syscall schema version
- ContextBlock ids
- final token ids hash
- runtime KV assumptions
- backend mode
- cache tier
- created time and expiration policy

## Deterministic KV Cache Modes

### STRICT_PREFIX

Reuse is allowed only when the complete cached prefix has identical final token IDs and all Nine-Gate validation checks pass.

### SNAPSHOT_PREFIX

Reuse is allowed from a Context-Shape Snapshot when source refs, hashes, layout version, token IDs, and runtime assumptions are validated.

### BACKEND_COMPOSITIONAL

Reuse is delegated to a backend that supports compositional KV reuse. FORGE-K still validates identity inputs and treats backend reuse as acceleration only.

## Nine-Gate Deterministic KV Validation

1. same model
2. same model revision
3. same tokenizer
4. same tokenizer revision
5. same chat template
6. same prompt layout version
7. same policy/syscall schema version
8. same final token IDs
9. same runtime KV assumptions

All gates must pass for reuse.

## Cache Hit Rules

- A hit may accelerate runtime execution only.
- A hit must cite the KVCacheManifest.
- A hit must not imply that memory exists or that evidence is admitted.
- A hit must not bypass policy, admission, or semantic syscall validation.

## Cache Miss Rules

- A miss falls back to normal context compilation and runtime-driver execution.
- A miss is not an error unless the caller requested cache-only behavior.
- Miss reasons should be recorded for diagnostics and Lymphatic eviction/compaction policy.

## Invalidation Rules

Invalidate cache entries when any validation gate changes, when source shape is superseded, when policy revokes eligibility, when runtime assumptions change, when expiration policy fires, or when Lymphatic Lane marks the entry stale.

## Cache Tiers

| Tier | Purpose |
|---|---|
| Hot | Immediate prefix reuse for current runtime session |
| Warm | Recent deterministic prefix reuse across related requests |
| Cold | Persisted manifests or restorable shape metadata requiring validation before use |

Doctrine: KV cache is not memory.
