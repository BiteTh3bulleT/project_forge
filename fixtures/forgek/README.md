# FORGE-K Fixture Corpus

Phase 11C keeps this corpus research-only and simulator-only. These fixtures are shared evidence for Go/Rust alignment work; they do not wire FORGE-K into live daemon authority, route live state mutation, reuse live KV, or grant model/runtime authority.

## Layout

- `valid/*.valid.json`: canonical examples that validators should accept.
- `invalid/*.invalid.json`: intentionally malformed examples that validators should reject.
- `golden/hashes.json`: human-readable stable hash expectations for each valid fixture.
- `golden/canonical_*.json`: canonical reference outputs retained for earlier fixture phases.

## Valid Fixture Kinds

- `context_block.valid.json`: one admitted evidence context block with provenance refs and tokenizer-neutral text hashes.
- `context_bundle.valid.json`: one context bundle containing the context block and bundle-level prompt/hash metadata.
- `kv_cache_manifest.valid.json`: deterministic KV acceleration metadata. It is not memory, not canonical truth, and does not contain KV tensors.
- `runtime_driver_manifest.valid.json`: proposal-only deterministic mock runtime driver metadata. It does not call a live model.
- `snapshot_manifest.valid.json`: semantic snapshot shape metadata. It is inspection/restoration evidence, not canonical truth.

## Stable Hash Schema

Golden hashes are SHA-256 hashes over canonical stable projections:

1. Parse JSON.
2. Sort object keys lexicographically at every level.
3. Normalize string whitespace to single spaces.
4. Sort unordered reference arrays for keys ending in `_refs` and for known set-like keys: `source_refs`, `blocks_refs`, `supported_models`, `supported_capabilities`, `allowed_syscalls`, `workspace_scope`, `provenance_refs`, and `context_block_refs`.
5. Remove stable-excluded fields recursively.
6. Pretty-print canonical JSON and hash the resulting UTF-8 text with SHA-256.

Stable-excluded fields are volatile identity, timestamp, stored-hash, and reuse-counter fields:

`created_at`, `updated_at`, `sealed_at`, `expired_at`, `last_used_at`, `invalidated_at`, `journal_refs`, `shape_hash`, `source_hash`, `content_hash`, `token_input_hash`, `bundle_hash`, `stable_prefix_hash`, `volatile_suffix_hash`, `cache_id`, `snapshot_id`, `block_id`, `bundle_id`, `driver_id`, `reuse_count`.

Each entry in `golden/hashes.json` records:

- `fixture_path`: path to the fixture relative to `fixtures/forgek`.
- `fixture_kind`: validator fixture kind.
- `hash_kind`: validator-produced hash field name.
- `expected_hash`: expected SHA-256 hex digest.
- `included_fields_summary`: human-readable summary of fields that remain in the stable projection.
- `excluded_fields_summary`: human-readable summary of fields intentionally ignored for stable hashing.

## Validation Commands

- Root parity check without making `npm test` depend on Rust: `npm run test:forgek:parity`.
- Rust fixture validator, when Rust is intentionally in scope: `npm run validate:forgek-fixtures`.
