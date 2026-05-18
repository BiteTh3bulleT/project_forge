# Failure Modes

## Authority blur

Mitigation: Control Lane commit path, Crucible validator-only, workers propose-only, no public mutation route.

## Stale cache poisoning

Mitigation: dependency hashes, policy epochs, memory epochs, dirty states, workspace isolation, invalidation tests.

## Worker sprawl

Mitigation: coalescing, budgets, traversal caps, priority aging, queue limits, backpressure.

## Temporal drift

Mitigation: snapshots are shape not truth, active state requires validation, replay requires current-state checks.

## Vector/VSA truth creep

Mitigation: evidence-only semantics, provenance, Crucible validation, Control Lane commit.

## Trace bloat

Mitigation: delta snapshots, compaction policy, retention tiers, cold archive.

## Oscillatory scheduling

Mitigation: hysteresis thresholds, cooldown windows, bounded retries, min TTL floors.
