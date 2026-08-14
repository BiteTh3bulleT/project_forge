# Definition of Done

Status: Phase 0 baseline.

FORGE-K work is done only when the implementation, documentation, tests, and evidence preserve kernel-first authority.

## Baseline Expectations

- Docs are updated for architecture changes.
- Tests are added where code changes exist.
- Semantic syscalls are journaled.
- A reported successful canonical commit has a typed, internally consistent
  receipt proving its transaction, journal identity/hash, object/provenance
  ids, durable audit intent, and idempotency fingerprint.
- A production semantic syscall has a Kernel-verified authenticated principal,
  effective registry definition, scope-exact capability grant, and explicit
  approval-policy record; replay carries and verifies the original full proof.
- Atomic audit intent contains the exact request and authorization proof and
  rejects request/proof swaps independently of best-effort audit delivery.
- No direct canonical mutation bypasses semantic syscalls.
- No unvalidated model output receives authority.
- Provenance is not destroyed.
- Snapshot-as-truth behavior is not introduced.
- KV-as-memory behavior is not introduced.
- Rejected evidence records rejection reasons.
- Courthouse decisions are deterministic, model/proposal sources cannot rule,
  and appeals preserve the prior immutable ruling.
- Superseded objects remain inspectable.
- Contradictions are recorded instead of silently merged.
- Runtime drivers remain isolated from Kernel authority.
- Live backup endpoints never raw-merge canonical or immutable proof tables;
  whole-store recovery is daemon-stopped and chain-verified.
- Observe/default maintenance and explicit dry-run previews do not rewrite
  historical evidence or rebuildable projections.
- Research/tooling phases remain isolated from live daemon authority.

## Phase Evidence

Every phase report should include:

- commands run
- validation results
- fixture and golden corpus drift evidence when shared validation fixtures change
- CI/workflow evidence when tooling integration changes
- files changed
- unresolved blockers
- known risks
- next recommended phase

## Integration Test Environment

Default local validation must not require external Postgres, Qdrant, or Redis services. Integration tests that need those services must be explicitly gated and documented locally, but CI must provision those services and fail when the required integration environment is absent.

Environment variables:

- `FORGE_POSTGRES_TEST_DSN`: enables local Postgres integration tests and is required in CI.
- `FORGE_QDRANT_TEST_URL`: enables local Qdrant shadow-vector integration tests and is required in CI.
- `FORGE_REDIS_TEST_ADDR`: enables local Redis ephemeral-boundary integration tests and is required in CI.

Commands:

- `npm test`: default Go core test path; must not require external services.
- `npm run test:integration:env`: reports integration environment visibility without failing when services are absent.
- `npm run test:integration:env:required`: fails if any integration environment variable is missing; this is the CI preflight.
- `npm run test:integration`: requires all integration environment variables before running the Go core test suite.
- `npm run test:forgek:parity`: explicit Go/Rust FORGE-K parity validation; root `npm test` does not depend on Rust.

CI provisions Postgres, Qdrant, and Redis, runs the required integration preflight, and treats missing integration environment as a failure instead of allowing silent skips. Local developer workflows keep the lenient visibility command so `npm test` remains usable without Docker services.

## Status Markers

Use these markers in status and architecture docs when implementation scope could be confused:

- `[LIVE]`: implemented in the live daemon path.
- `[SIMULATOR-ONLY]`: implemented only under simulator packages or simulator test harnesses.
- `[PARTIAL]`: a narrow integration exists, but the broader subsystem remains incomplete or non-authoritative.
- `[FUTURE]`: designed or intended, but not implemented as current behavior.
