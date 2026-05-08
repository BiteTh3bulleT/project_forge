# Context Compiler and Deterministic KV Cache

Status: Phase 7 Context Compiler and Phase 8 Deterministic KV System are implemented in the FORGE-K simulator. Phase i1 adds `[PARTIAL]` live validation-only reuse of the deterministic KV identity gate logic through `services/core/internal/kvidentity` and AI-OS Control Lane `VALIDATE_KV_IDENTITY`; FORGE-K `KVService` remains `[SIMULATOR-ONLY]`, and there is still no live KV reuse.

The Context Compiler turns admitted semantic shape, snapshot refs, and restore seeds into deterministic, token-addressable ContextBlocks and ContextBundles. The KV cache accelerates exact reusable token shapes in a later phase. Neither context shape nor KV reuse is canonical memory.

Phase 11E ConsensusReport refs may become future context inputs by reference. Context compilation must not treat a consensus report as admitted truth unless Courthouse admission exists; consensus is response/action governance, not canonical memory.

Phase 7 scope is `SIMULATOR_ONLY` under `services/core/internal/forgek/contextcompiler` and `services/core/internal/forgek/context_syscalls.go`. Phase 8 service scope is `SIMULATOR_ONLY` under `services/core/internal/forgek/kv` and `services/core/internal/forgek/kv_syscalls.go`. The shared KV identity gate validator is live validation-only; it does not wire simulator KV manifests, lookup, tiering, invalidation, eviction, Context Compiler output, runtime cache reuse, or modelruntime behavior into the live daemon.

Phase 12M-Q adds a shadow advisory context summary in `services/core/internal/forgekshadow` only. That advisory consumes existing safe diagnostic refs and counts, may produce a deterministic summary hash, and may safely warn when metadata is insufficient. It is not a simulator `ContextBundle`, not a live prompt, not a replacement for live `COMPILE_CONTEXT`, not a KV manifest, and not user-visible output.

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

Phase 8 cache eligibility requires deterministic identity at the token and runtime assumption level. Eligible cache entries are represented by a KVCacheManifest.

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

## KVCacheManifest

A KVCacheManifest records:

- `cache_id`, `cache_mode`, workspace/case refs, `bundle_id`, optional `block_id`, optional snapshot/restore-seed refs
- context identity refs and hashes: `bundle_hash`, `stable_prefix_hash`, `volatile_suffix_hash`, and `token_input_hash`
- model id/revision, tokenizer id/revision, chat template hash, prompt layout hash, policy schema hash, and syscall schema hash
- optional `final_token_ids_hash` placeholder for later tokenizer-specific work
- runtime assumptions: backend, runtime version, attention backend, rope config hash, KV precision, and cache salt
- memory tier, status, reuse count, timestamps, invalidation reason, journal refs, and metadata

KVCacheManifest is acceleration metadata. It is not semantic evidence, not canonical truth, not memory, and not a runtime tensor store.

## Deterministic KV Cache Modes

### STRICT_PREFIX

Reuse is allowed only when the complete cached prefix has identical final token IDs and all Nine-Gate validation checks pass.

### SNAPSHOT_PREFIX

Reuse is allowed from a Context-Shape Snapshot when source refs, hashes, layout version, token IDs, and runtime assumptions are validated.

### BACKEND_COMPOSITIONAL

Reuse is delegated to a backend that supports compositional KV reuse in a future runtime integration phase. In Phase 8 this mode is metadata-only and never performs non-prefix reuse.

## Nine-Gate Deterministic KV Validation

1. same model
2. same model revision
3. same tokenizer
4. same tokenizer revision
5. same chat template
6. same prompt layout version
7. same policy/syscall schema version
8. same final token IDs, or same `token_input_hash` as the Phase 8 simulator placeholder when final token IDs do not exist yet
9. same runtime KV assumptions

All gates must pass for reuse.

Phase 8 uses tokenizer-neutral `token_input_hash` as a deterministic identity placeholder. A later runtime-driver phase may replace or supplement this with model/tokenizer-specific final token IDs.

## Cache Hit Rules

- A Phase 8 hit is simulator validation only.
- A hit must cite the KVCacheManifest.
- A hit must not imply that memory exists or that evidence is admitted.
- A hit must not bypass policy, admission, or semantic syscall validation.
- A hit must not call model runtimes or reuse live backend cache.

## Cache Miss Rules

- A Phase 8 miss is safe metadata: it never mutates context, snapshots, source objects, or canonical truth.
- A miss is not an error unless the caller requested cache-only behavior.
- Miss reasons may be recorded for diagnostics and future maintenance policy.

## Invalidation Rules

Invalidate cache entries when any validation gate changes, when source shape is superseded, when policy revokes eligibility, when runtime assumptions change, when expiration policy fires, or when future maintenance policy marks the entry stale.

## Cache Tiers

| Tier | Purpose |
|---|---|
| `GPU_HOT` | Simulator metadata for an immediately reusable runtime tier |
| `CPU_WARM` | Simulator metadata for recent deterministic prefix reuse |
| `DISK_COLD` | Simulator metadata for persisted manifests |
| `REMOTE_COLD` | Simulator metadata for remote/cold manifests |
| `NONE` | No placement tier |

Doctrine: KV cache is not memory.

## KV Syscalls

The FORGE-K simulator registers:

- `kv.register`
- `kv.lookup`
- `kv.record_hit`
- `kv.record_miss`
- `kv.invalidate`
- `kv.evict`
- `kv.promote`
- `kv.demote`
- `kv.get_manifest`
- `kv.list_manifests`
- `kv.validate_identity`

Mutating syscalls require capability and journal `KV_CACHE_REGISTERED`, `KV_CACHE_HIT`, `KV_CACHE_MISS`, `KV_CACHE_INVALIDATED`, `KV_CACHE_EVICTED`, `KV_CACHE_PROMOTED`, or `KV_CACHE_DEMOTED`. Read and validation syscalls require KV read capability and do not mutate state.

## Phase 8 Implemented Scope

Phase 8 implements KV metadata and validation only:

- in-memory KV service owned by the FORGE-K simulator kernel
- KVCacheManifest registration from existing ContextBundle or ContextBlock refs
- deterministic lookup with nine-gate identity validation
- hit/miss recording, invalidation, eviction, promotion, and demotion metadata
- simulator memory tiers and status lifecycle
- integration with Context Compiler by reference only
- tests proving cache hits do not become truth, evidence, memory, live runtime calls, or live KV reuse

## Phase 9 Runtime Boundary Interaction

Phase 9 implements a simulator-only runtime driver boundary. A `RuntimeGenerateRequest` may cite a ContextBundle, ContextBlock, snapshot, restore-seed, case refs, and KV lookup metadata, but the runtime driver receives those as refs or prepared prompt text only.

The runtime driver must not compile context, admit evidence, mutate ContextBundles, register KV manifests, store real KV tensors, or perform backend KV reuse. A `RuntimeGenerateResult` is proposal output with provenance and driver/model metadata; it is not canonical truth and must pass existing validation, admission, and Kernel commit paths before any durable mutation.

## Current Limitations

- No live daemon integration.
- No changes to live AI-OS `COMPILE_CONTEXT`.
- No real model calls, external retrieval, route changes, or public API changes.
- Token counts are deterministic estimates, not tokenizer-specific counts.
- `token_input_hash` is tokenizer-neutral and is a Phase 8 identity placeholder when final token IDs are unavailable.
- No real KV tensors are stored.
- No runtime backend cache lookup, registration, reuse, eviction, or tier movement occurs.
- Phase 9 implements the simulator runtime-driver boundary with a deterministic mock driver only; any live runtime interaction still requires a later explicitly scoped `LIVE_INTEGRATION` phase.
