# Phase 13H Redis Ephemeral Boundary

Status: implemented and tested.

Scope: `LIVE_INFRA / EPHEMERAL_COORDINATION / DISABLED_BY_DEFAULT / NON_CANONICAL`.

## Summary

Phase 13H defines the Redis boundary for future ephemeral coordination under `services/core/internal/ephemeral`.

Redis remains optional infrastructure. It is not canonical truth, durable memory, evidence admission, provenance authority, audit authority, settings authority, or the only job record.

## Config And Flags

- `FORGE_REDIS_ENABLED=false`
- `FORGE_REDIS_ADDR`
- `FORGE_REDIS_KEY_PREFIX=forge`
- `FORGE_REDIS_TIMEOUT_MS=1000`

Enabling Redis requires an explicit Redis addr and does not switch `FORGE_STORE_BACKEND`.

## Allowed Roles

Redis may be used in future wiring phases for:

- cache,
- queue,
- lock,
- pub/sub,
- progress stream,
- rate-limit window,
- ephemeral coordination.

All Redis state must be recoverable from durable SQLite/Postgres records or safely disposable.

## Forbidden Roles

Redis must not be used for:

- canonical truth,
- durable memory,
- evidence admission,
- provenance authority,
- sole job record,
- canonical audit,
- canonical settings,
- vector truth.

## Key Policy

Redis keys must:

- use a deterministic prefix,
- use bounded workspace-safe segments,
- avoid raw prompts, content, raw queries, auth material, secrets, tokens, cookies, API keys, or path-like unsafe text,
- avoid unbounded key material,
- use opaque hashes for sensitive or variable source identifiers.

Cache, lock, and progress entries require TTLs.

## Adapter Behavior

Phase 13H includes:

- a fake in-memory adapter for contract tests,
- a stdlib Redis client scaffold,
- cache set/get,
- queue push/pop,
- lock acquire/release,
- progress append/read,
- health checks.

No live path uses Redis by default.

## Integration Tests

Normal tests do not require Redis. Optional Redis integration tests run only when `FORGE_REDIS_TEST_ADDR` is set and cover:

- ping,
- cache set/get with TTL,
- queue push/pop,
- lock acquire/release.

## Live Behavior

Live behavior is unchanged. Redis is not required for normal operation. Phase 13H does not switch live job queues, add routes, change public APIs, alter gateway/modelruntime/retrieval behavior, write live memory, store canonical records, or make FORGE-K live authority.

## Future Wiring Candidates

Future phases may consider disabled-by-default Redis mirrors for:

- job progress streams,
- bounded modelruntime scheduling hints,
- rate-limit windows,
- short-lived cache entries,
- pub/sub notifications.

Each future wiring phase must include explicit tests proving Redis loss is safe and durable records remain authoritative.
