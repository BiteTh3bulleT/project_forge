# forgek-validate

`forgek-validate` is the Phase 11B standalone Rust validation crate for FORGE-K deterministic fixtures.

Scope: `RESEARCH_ONLY / SIMULATOR_ONLY`.

It validates shared JSON fixtures under `fixtures/forgek` without importing from Go production code, wiring into the live daemon, calling model runtimes, changing gateway behavior, changing routes, or mutating canonical state.

## Commands

Run from this crate:

```bash
cargo test
cargo run -- validate ../../fixtures/forgek/valid/snapshot_manifest.valid.json
cargo run -- canonicalize ../../fixtures/forgek/valid/context_block.valid.json
cargo run -- hash ../../fixtures/forgek/valid/kv_cache_manifest.valid.json
cargo run -- validate-fixtures ../../fixtures/forgek
```

Root helper scripts:

```bash
npm run test:rust:forgek
npm run validate:forgek-fixtures
npm run test:forgek:parity
npm run validate:forgek
```

`npm run validate:forgek` is a grouped convenience command for local and ad hoc validation. CI intentionally runs the validator tests, fixture validation, and Go/Rust parity as separate steps so their failures remain easy to interpret.

## Implemented Boundary

- canonical JSON normalization with stable object key order
- deterministic ordering for ref-like arrays
- whitespace normalization for strings
- SHA-256 hashing over stable projections
- generated IDs, timestamps, journal refs, and existing hash fields excluded from stable identity hashes
- validators for Snapshot, ContextBlock, ContextBundle, KVCacheManifest, RuntimeDriverManifest, and capability-like fixtures
- conservative rejection of secret-looking runtime manifest fields
- corpus parity checks for every fixture under `fixtures/forgek/valid` and `fixtures/forgek/invalid`
- canonical golden comparisons for checked-in canonical fixture outputs
- `validate-fixtures` golden hash comparison for flat and expanded `golden/hashes.json` manifests
- drift tests for excluded timestamps, stable identity fields, missing refs, runtime secret-looking fields, and KV runtime identity assumptions

## Non-Goals

- no cgo
- no live daemon integration
- no Go production calls into Rust
- no public API or route changes
- no model runtime calls
- no live KV reuse
- no gateway behavior changes
- no canonical state mutation

## CI And Tooling Boundary

Phase 11D allows CI to install stable Rust and run the root helper commands for this crate and the shared fixture corpus:

- `npm run test:rust:forgek`
- `npm run validate:forgek-fixtures`
- `npm run test:forgek:parity`

These checks validate deterministic fixture contracts only. They do not make Rust part of the live daemon, do not make Rust an authority path, do not call model runtimes, and do not change root `npm test`.
