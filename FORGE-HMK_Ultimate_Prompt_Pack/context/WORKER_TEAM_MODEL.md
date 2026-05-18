# Neuron Mesh Worker Team Model

Workers are specialist labor units under FORGE-T leases. They are not authorities.

## Team types

### Snapshot Harvester

Pulls PhotoCells, snapshot bundles, restore seeds, and current-vs-prior shape comparisons.

### Trace Weaver

Builds TraceCells, compresses event windows, assembles before/action/after chains, and proposes ReplayCells.

### Binder Team

Runs VSA bind/unbind/bundle/permutation and semantic algebra transformations.

### Cache Smiths

Perform HKV lookup, dependency validation, stale entry detection, promotion/demotion recommendations, and prewarm prep.

### Context Assembly Team

Ranks cells, compiles context packets, splits stable/volatile blocks, attaches provenance, and enforces token budgets.

### Contradiction Scout

Surfaces contradiction groups, supersession chains, stale active state, and ClaimEnvelope drafts.

### Governor Watch

Monitors queue pressure, worker leases, backpressure state, and performance metrics.

## Worker output rule

Every worker output must be typed and scoped.

Bad: “I found some useful stuff.”

Good: `Artifact(type=SnapshotBundle, refs=[...], confidence=0.82, authority=non_canonical, warnings=[...])`
