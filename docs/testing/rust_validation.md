# Rust Validation Tooling

Status: Phase 11D implemented as `RESEARCH_ONLY / SIMULATOR_ONLY / TOOLING_ONLY`.

Rust validation exists to check deterministic FORGE-K fixture contracts. It is not live daemon integration, not a Go production dependency, and not a canonical mutation path.

## Local Commands

Run from the repository root:

```bash
npm run test:rust:forgek
npm run validate:forgek-fixtures
npm run test:forgek:parity
npm run validate:forgek
```

Command meaning:

- `npm run test:rust:forgek` runs `cargo test` in `crates/forgek-validate`.
- `npm run validate:forgek-fixtures` validates `fixtures/forgek` with the Rust CLI and checked-in golden hashes.
- `npm run test:forgek:parity` runs Go simulator fixture parity tests, Rust validator tests, Rust fixture validation, and Node golden hash checks.
- `npm run validate:forgek` is a grouped local helper for all three checks.

Root `npm test` remains the Go core test path and does not depend on Rust.

## CI Behavior

The CI workflow installs stable Rust after Node setup and runs three separate FORGE-K validation steps after `npm ci`:

- Rust FORGE-K validator tests: `npm run test:rust:forgek`
- Rust FORGE-K fixture validation: `npm run validate:forgek-fixtures`
- Go/Rust FORGE-K parity: `npm run test:forgek:parity`

The existing Go tests, Go vet, desktop typecheck, desktop build, and smoke steps remain separate. Rust validation failures should not be folded into generic Go or desktop failures.

## Failure Interpretation

- Rust validator test failure: the Rust validation implementation, canonicalization, hashing, or validator tests drifted.
- Fixture validation failure: the checked-in fixture corpus, canonical golden outputs, or hash manifest drifted.
- Go/Rust parity failure: Go simulator fixture expectations, Rust validation, or Node golden hash checks no longer agree.

When fixtures intentionally change, update the fixture JSON, canonical golden files, and `fixtures/forgek/golden/hashes.json` in the same pass, then rerun the commands above.

## Authority Boundary

Rust validation is tooling only:

- no live daemon wiring
- no cgo bridge
- no Go production calls into Rust
- no public API or route changes
- no gateway, modelruntime, or controllane behavior changes
- no model calls
- no canonical state mutation

ADR 0005 remains binding: FORGE-K is the target architecture, while live authority still uses existing AI-OS, gateway, permissions, lane, audit, model runtime, and API paths.
