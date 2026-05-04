# Lymphatic Lane

Status: Phase 10 implemented/tested in the FORGE-K simulator only. Scope is `SIMULATOR_ONLY`.

The Lymphatic Lane is the FORGE-K maintenance lane for deterministic hygiene, coherence checks, anti-rot review, and deferred cleanup planning. It produces Maintenance Reports and Cleanup Proposals. It does not own truth, silently mutate source objects, run live cleanup, or bypass Kernel authority.

## Scope

Phase 10 is simulator-only. The implementation boundary is `services/core/internal/forgek`, docs, and tests. It is not live daemon authority.

Phase 10 does not:

- wire FORGE-K into the live daemon
- change live dream or autonomy behavior
- modify live cleanup jobs
- add routes or public APIs
- change gateway, modelruntime, or AI-OS controllane behavior
- run automatically in the live runtime
- delete, compact, expire, merge, or supersede canonical truth without semantic syscalls

Existing live dream/autonomy cleanup-style paths remain outside FORGE-K authority. The simulator implementation does not make those live paths FORGE-K authority.

## Responsibilities

The Lymphatic Lane is responsible for deferred maintenance review:

- Lymphatic Sweeps over simulator objects and metadata
- Maintenance Reports that summarize stale, expired, contradicted, superseded, invalidated, orphaned, or cleanup-eligible items
- Cleanup Proposals that describe deterministic candidate actions
- Cache Hygiene for KV metadata, never memory deletion
- Snapshot Hygiene for shape artifacts, never source truth deletion
- Runtime Result Hygiene for aging proposal evidence, never model-output erasure as truth mutation
- Contradiction Sweeps that surface conflicts without silently merging them

Reports and proposals are evidence. They are not canonical mutations.

## Authority Rules

The Lymphatic Lane cleans, audits, proposes, expires, compacts, and reports. It does not commit canonical truth.

Any meaningful mutation must still pass through:

1. semantic syscall request
2. deterministic validation
3. capability and scope checks
4. Courthouse evidence admission when evidence is involved
5. Kernel commit boundary
6. journal and provenance capture

Rejected, superseded, invalidated, expired, and stale objects must remain inspectable unless a later explicitly governed syscall records archival or retirement behavior. Cache eviction does not delete memory. Snapshot expiration does not delete source truth.

## Simulator Concepts

The Phase 10 simulator contracts stay narrow:

- `LymphaticPolicy`: deterministic sweep configuration, dry-run flag, max report items, expiration/staleness thresholds, enabled sweep kinds, and metadata.
- `LymphaticSweep`: a bounded deterministic pass over selected simulator object refs and metadata.
- `MaintenanceReport`: ordered findings, counts, policy refs, source refs, generated timestamps, and provenance.
- `CleanupProposal`: proposed action, target refs, reason codes, risk level, required syscall class, and evidence refs.
- `HygieneFinding`: stale object, orphaned reference, supersession candidate, expiration candidate, invalidated KV metadata, runtime result aging, or contradiction candidate.

These contracts should be deterministic for stable inputs and must preserve provenance for every referenced object.

## Implemented Sweep Surface

Phase 10 declares the full planned sweep vocabulary:

- `SNAPSHOT_HYGIENE`
- `KV_HYGIENE`
- `RUNTIME_RESULT_HYGIENE`
- `CONTEXT_BUNDLE_HYGIENE`
- `CONTRADICTION_SWEEP`
- `SUPERSESSION_SWEEP`
- `ORPHAN_REF_SWEEP`
- `PALACE_ROUTE_HYGIENE`
- `SEMANTIC_OBJECT_HYGIENE`
- `CASE_HYGIENE`
- `JOURNAL_INTEGRITY_SWEEP`

The current simulator implements deterministic behavior for `SNAPSHOT_HYGIENE`, `KV_HYGIENE`, `RUNTIME_RESULT_HYGIENE`, `CONTRADICTION_SWEEP`, and `ORPHAN_REF_SWEEP`. The remaining declared sweep kinds are reserved for future simulator expansion and return warnings rather than live cleanup behavior.

## Syscalls

The simulator syscall surface is:

- `lymph.run_sweep`
- `lymph.get_report`
- `lymph.list_reports`
- `lymph.get_proposal`
- `lymph.list_proposals`
- `lymph.create_proposal`
- `lymph.read`

Mutating syscalls require explicit lymphatic capabilities and journal `LYMPHATIC_SWEEP_COMPLETED` or `LYMPHATIC_PROPOSAL_CREATED`. Read syscalls require either the exact read syscall capability or `lymph.read` and do not journal mutation events.

## Integration Boundaries

Phase 10 may inspect simulator metadata from earlier FORGE-K phases by reference:

- snapshots as shape artifacts
- ContextBundles and ContextBlocks as compiled shape
- KVCacheManifests as acceleration metadata
- runtime generate results as proposal evidence
- Courthouse cases, exhibits, rulings, and contradictions as admission evidence
- SemanticObjects and operation results as governed semantic records

The Lymphatic Lane must not mutate those objects directly. It may produce reports and cleanup proposals that cite them.

## Validation Evidence

The current simulator validation command is:

```bash
cd services/core && go test ./internal/forgek/...
```

This pass covers the Phase 10 lymphatic package and simulator syscall tests present in the current worktree. Validation expectations remain:

- deterministic policy validation
- stable sweep ordering
- report item caps
- cleanup proposals requiring semantic syscalls
- snapshot hygiene without source truth deletion
- KV hygiene without memory semantics
- runtime result hygiene without model evidence deletion
- contradiction sweeps without silent merge behavior
- no silent mutation of source objects
- no live daemon, route, gateway, modelruntime, dream, autonomy, or cleanup-job wiring

Passing simulator tests do not create live daemon authority.
