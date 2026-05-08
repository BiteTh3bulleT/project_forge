# Definition of Done

Status: Phase 0 baseline.

FORGE-K work is done only when the implementation, documentation, tests, and evidence preserve kernel-first authority.

## Baseline Expectations

- Docs are updated for architecture changes.
- Tests are added where code changes exist.
- Semantic syscalls are journaled.
- No direct canonical mutation bypasses semantic syscalls.
- No unvalidated model output receives authority.
- Provenance is not destroyed.
- Snapshot-as-truth behavior is not introduced.
- KV-as-memory behavior is not introduced.
- Rejected evidence records rejection reasons.
- Superseded objects remain inspectable.
- Contradictions are recorded instead of silently merged.
- Runtime drivers remain isolated from Kernel authority.
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

Default validation must not require external Postgres, Qdrant, or Redis services. Integration tests that need those services must be explicitly gated and documented.

Environment variables:

- `FORGE_POSTGRES_TEST_DSN`: enables optional Postgres integration tests.
- `FORGE_QDRANT_TEST_URL`: enables optional Qdrant shadow-vector integration tests.
- `FORGE_REDIS_TEST_ADDR`: enables optional Redis ephemeral-boundary integration tests.

Commands:

- `npm test`: default Go core test path; must not require external services.
- `npm run test:integration:env`: reports integration environment visibility without failing when services are absent.
- `npm run test:integration`: requires all integration environment variables before running the Go core test suite.
- `npm run test:forgek:parity`: explicit Go/Rust FORGE-K parity validation; root `npm test` does not depend on Rust.

CI should expose integration environment visibility so skipped external-service tests are visible, but default CI should remain runnable without local database/vector/Redis services unless a workflow explicitly provisions them.

## Status Markers

Use these markers in status and architecture docs when implementation scope could be confused:

- `[LIVE]`: implemented in the live daemon path.
- `[SIMULATOR-ONLY]`: implemented only under simulator packages or simulator test harnesses.
- `[PARTIAL]`: a narrow integration exists, but the broader subsystem remains incomplete or non-authoritative.
- `[FUTURE]`: designed or intended, but not implemented as current behavior.
