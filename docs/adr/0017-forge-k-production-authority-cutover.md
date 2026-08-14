# ADR 0017 - FORGE-K Production Authority Cutover

Status: Accepted; Stages K20A-K20B active

Date: 2026-08-14

## Context

The FORGE-K simulator proves the target cognitive microkernel contracts, while
the live daemon historically assigned semantic mutation authority to
`internal/aios/controllane`. Promoting the simulator directly would replace a
durable SQLite transaction and journal path with in-memory registries, creating
data-loss and dual-authority risks.

The chat gateway also previously exposed the full tool proposal catalog on
some ambiguous operational turns. Gateway policy still governed execution, but
allowing a model to choose a tool conflicted with the stronger FORGE doctrine:
models format proposals; FORGE decides tool use.

## Decision

FORGE-K moves to production one authority boundary at a time:

1. `internal/forgekernel` is the production FORGE-K authority package.
2. The simulator under `internal/forgek` remains isolated until its pure
   deterministic contracts are promoted into production-owned packages.
3. Stage K20A makes FORGE-K the default live semantic syscall ingress owner.
4. Stage K20B makes FORGE-K own the durable stage order through its
   `DurablePort`: deterministic prepare, exactly one atomic apply+journal
   commit, audit recording/linkage, then best-effort observation. The existing
   Control Lane code implements that temporary port and the rollback facade;
   it is not a second authority.
5. Boot selects exactly one mode through `FORGE_KERNEL_AUTHORITY_MODE`:
   `forge_k` (default) or `legacy_v1` (rollback). There is no dual commit mode.
6. Full FORGE-K authority is not claimed until the adapter implementation and remaining
   subsystem gates are migrated and v1 is retired.
7. FORGE deterministically decides whether a chat turn needs a tool and selects
   exactly one eligible tool before any model proposal call. If it cannot do so,
   no tool schema is exposed. A model may only format bounded arguments for the
   schema FORGE selected; the gateway remains execution authority.

## Cutover order

1. Production Kernel ingress and single-authority boot selection. Closed K20A.
2. Durable journal/commit orchestration owned by FORGE-K through narrow ports. Closed K20B; Control Lane implementation extraction remains.
3. Courthouse admission and ruling authority.
4. Semantic Algebra operations and structured Memory Palace objects.
5. Context Compiler live bundle authority.
6. Runtime driver proposal boundary and response composition gate.
7. Snapshots/replay, KV acceleration, and Lymphatic proposal lanes.
8. Remove `legacy_v1`, compatibility facades, and stale authority claims.

Each step requires deterministic parity tests, malformed-input failure tests,
capability/approval tests, journal and audit evidence, a tested rollback path,
operator-visible status, and documentation updates.

## Consequences

- FORGE-K ingress and durable stage orchestration are live now, but the full-kernel flag remains false.
- Existing durable data remains in the current SQLite schema during migration.
- The Control Lane can be reduced behind ports rather than duplicated or
  bypassed.
- Simulator service imports remain forbidden in live paths.
- Models have no tool-selection, execution, admission, or mutation authority.
- The migration is observable and reversible until final v1 retirement.

## Rejected alternatives

- Directly run `forgek.Kernel` in production: rejected because its registry and
  journal are simulator-memory structures.
- Big-bang replacement: rejected because it removes useful parity and rollback
  evidence across multiple authority domains at once.
- Dual writes: rejected because two commit authorities make truth ambiguous.
- Model-selected tools with gateway validation: rejected because validation of
  execution does not make model-owned selection a FORGE decision.
