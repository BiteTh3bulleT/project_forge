# Context Compiler and Deterministic KV Cache

Status: Phase 7 Context Compiler implemented in the FORGE-K simulator. Phase 8 Deterministic KV Cache is not implemented.

The Context Compiler turns admitted semantic shape, snapshot refs, and restore seeds into deterministic, token-addressable ContextBlocks and ContextBundles. The KV cache accelerates exact reusable token shapes in a later phase. Neither context shape nor KV reuse is canonical memory.

Phase 7 scope is `SIMULATOR_ONLY` under `services/core/internal/forgek/contextcompiler` and `services/core/internal/forgek/context_syscalls.go`. It is not wired into the live daemon and does not modify the live AI-OS `COMPILE_CONTEXT` path.

## ContextBlock

A ContextBlock is a deterministic prompt unit emitted by the compiler. The implemented model includes:

- `block_id`, `block_type`, `workspace_id`, optional case/snapshot/restore-seed refs
- source object refs, source refs, admitted and rejected exhibit refs, ruling refs, contradiction refs, palace route refs, semantic operation refs, and derived object refs
- `content_summary` and deterministic `canonical_text`
- `content_hash` and tokenizer-neutral `token_input_hash`
- deterministic token count estimate
- layout position, cache eligibility metadata, invalidation scope, policy version, syscall schema version, journal refs, and metadata

Implemented block types include `KERNEL_DOCTRINE`, `POLICY_BOUNDARY`, `TOOL_CONTRACTS`, `WORKSPACE_IDENTITY`, `GOVERNING_PRECEDENT`, `CASE_SUMMARY`, `PALACE_ROUTE_SUMMARY`, `ADMITTED_EVIDENCE`, `REJECTED_EVIDENCE_SUMMARY`, `CONTRADICTION_SUMMARY`, `SEMANTIC_OPERATION_SUMMARY`, `SNAPSHOT_RESTORE_SEED`, `ACTIVE_CONSTRAINTS`, `CURRENT_TASK`, `VOLATILE_DETAIL`, `USER_MESSAGE`, `FUTURE_TOKEN_PLACEHOLDER`, and `FUTURE_KV_PLACEHOLDER`.

ContextBlocks are compiled shape, not truth. They cite refs, preserve provenance, do not admit evidence, do not mutate source objects, do not call models, and do not create or reuse KV cache.

## ContextBundle

A ContextBundle is an ordered compiled shape built from ContextBlocks. It includes:

- `bundle_id`, `workspace_id`, optional case/snapshot/restore-seed refs
- `layout_id`, `layout_version`, ordered `blocks`, and deterministic `canonical_prompt_text`
- `bundle_hash`, `token_input_hash`, `stable_prefix_hash`, and `volatile_suffix_hash`
- deterministic token estimate and cache eligibility summary
- source refs, creator, creation time, journal refs, and metadata

Bundle hashes exclude timestamps and generated object IDs. The bundle hash changes when block order, block content, layout version, or source refs change. ContextBundles are not canonical truth, not model responses, and not KV cache entries.

## Stable Prompt Layout

Stable prompt layout means the same admitted inputs, compiler version, policy schema, and layout version produce the same ordered canonical prompt text. Layout stability is required for deterministic cache eligibility metadata and replayable context inspection.

The default Phase 7 order puts stable blocks first and volatile blocks last:

1. `KERNEL_DOCTRINE`
2. `POLICY_BOUNDARY`
3. `TOOL_CONTRACTS`
4. `WORKSPACE_IDENTITY`
5. `GOVERNING_PRECEDENT`
6. `CASE_SUMMARY`
7. `PALACE_ROUTE_SUMMARY`
8. `ADMITTED_EVIDENCE`
9. `CONTRADICTION_SUMMARY`
10. `SEMANTIC_OPERATION_SUMMARY`
11. `SNAPSHOT_RESTORE_SEED`
12. `ACTIVE_CONSTRAINTS`
13. `CURRENT_TASK`
14. `VOLATILE_DETAIL`
15. `USER_MESSAGE`

The layout version affects bundle hashing.

## ContextCompileRequest and Result

`ContextCompileRequest` validates workspace scope, requires at least one case, snapshot, restore seed, or source ref, normalizes refs, sorts refs deterministically, and never fetches live external content or calls models.

`ContextCompileResult` returns the compiled bundle, blocks, warnings, errors, provenance refs, and metadata. Warnings cover missing admitted evidence, rejected evidence summaries, contradictions, restore seed inclusion, volatile user messages, token estimate budget pressure, and no cacheable prefix.

## Expansion/Contraction Context Compiler Loop

The full architecture compiler runs an expansion/contraction loop:

1. Expand from query, case, workspace scope, Memory Palace rooms, anchors, and routes.
2. Submit retrieved candidates as exhibits.
3. Contract through Courthouse admission, deterministic budgets, relevance thresholds, contradiction status, and policy constraints.
4. Emit ContextBlocks with stable ordering and citations.
5. Hash final token inputs for Phase 7 cache eligibility metadata.

In the current repository, the live AI-OS `COMPILE_CONTEXT` path remains separate. Phase 7 implements only the FORGE-K simulator Context Compiler and does not alter live restore scoring, fresh-compile fallback, persisted snapshot evidence, `restore_scores_json`, or `resume_hints_json`.

## Deterministic Serialization

Phase 7 canonical serialization uses:

- stable section ordering and explicit block headers
- sorted refs and normalized whitespace
- stable map key ordering
- no timestamps in content hash or token input hash inputs
- no random or generated IDs in hash inputs
- explicit block type, workspace, case/snapshot/restore-seed refs, layout version, policy version, syscall schema version, source refs, summaries, and canonical text

The canonical prompt text is assembled deterministically from the ordered blocks.

## Token Hashing

Phase 7 implements tokenizer-neutral token input hashing: `token_input_hash` is the SHA-256 hash of the exact canonical text that would be sent to a tokenizer. It does not claim tokenizer-specific token ID identity.

Phase 8 may add model, tokenizer, chat template, and runtime-specific identity gates. Text identity alone remains insufficient for runtime KV reuse because tokenizer revision, chat template, and prompt layout can change final tokens.

## Cache Eligibility

Phase 7 cache eligibility is metadata only. It records `CACHE_ALWAYS`, `CACHE_IF_STABLE`, `CACHE_EPHEMERAL`, or `DO_NOT_CACHE` on blocks and summarizes the bundle. It performs no KV cache lookup, registration, reuse, eviction, or runtime validation.

Recommended defaults are implemented: doctrine, policy, and tool contracts are `CACHE_ALWAYS`; workspace/case/palace/admitted/contradiction/snapshot restore seed shape is `CACHE_IF_STABLE`; active constraints, current task, volatile detail, and future placeholders are `CACHE_EPHEMERAL`; user messages are `DO_NOT_CACHE`.

Phase 8 cache eligibility requires deterministic identity at the token and runtime assumption level. Eligible cache entries must be represented by a KVCacheManifest.

This is inference/runtime KV reuse, not the existing restore scoring cache. Restore scoring metadata is shape/evidence for context selection; Deterministic KV Cache is acceleration for exact reusable token shapes.

## Shape-Not-Truth Invariants

Phase 7 tests prove:

- compiling context does not mutate source objects
- compiling context does not admit or reject evidence
- rejected evidence cannot appear as admitted evidence
- compiling from snapshots does not make snapshots truth
- compiling from restore seeds does not execute restoration
- context blocks do not become canonical truth, model responses, or KV cache
- semantic operation and derived object refs are preserved without promoting authority
- palace route and candidate refs are preserved by reference only

## Context Syscalls

The FORGE-K simulator registers:

- `context.compile`
- `context.compile_from_snapshot`
- `context.compile_from_restore_seed`
- `context.get_bundle`
- `context.list_bundles`
- `context.get_block`
- `context.list_blocks`
- `context.validate_layout`
- `context.hash`

Compile syscalls require capability and journal `CONTEXT_COMPILED`, `CONTEXT_COMPILED_FROM_SNAPSHOT`, or `CONTEXT_COMPILED_FROM_RESTORE_SEED`. Read/hash/layout syscalls require context read capability and do not journal mutations.

## Current Limitations

- No live daemon integration.
- No changes to live AI-OS `COMPILE_CONTEXT`.
- No model calls, runtime drivers, external retrieval, or route changes.
- Token counts are deterministic estimates, not tokenizer-specific counts.
- `token_input_hash` is tokenizer-neutral and does not validate model/tokenizer identity.
- No KVCacheManifest, nine-gate validation, cache lookup, cache reuse, or cache eviction.

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
