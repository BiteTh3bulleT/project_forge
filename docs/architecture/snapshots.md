# Context-Shape Snapshots

Status: Phase 0 architecture baseline.

Snapshots preserve shape, not truth. A snapshot records restorable semantic structure, source references, hashes, operation records, and summaries. It may seed restoration, replay, context compilation, review, or cache eligibility, but it does not become canonical truth by existing.

Snapshots cite source objects. Snapshots may be superseded. Snapshots should store references, hashes, operation records, and summaries rather than duplicating large canonical content.

## Snapshot Types

### SemanticSnapshot

Records semantic object shape for a set of claims, evidence, decisions, links, and lifecycle states. It cites the source object ids and operation that produced the shape.

### CaseSnapshot

Records the shape of a case: claims, exhibits, validations, admission decisions, rulings, and unresolved questions.

### ContextRestoreSnapshot

Records a prior context compilation or restore selection shape. It may include selected ContextBlocks, restore scores, resume hints, and source references.

### PalaceRouteSnapshot

Records Memory Palace route shape: rooms, anchors, route ids, candidate ids, ranking metadata, and retrieval reasons.

### WorkspaceSnapshot

Records scoped workspace shape: active goals, constraints, major artifacts, current open loops, and key references. It must cite canonical objects instead of copying them wholesale.

### DecisionSnapshot

Records the shape of a decision and its evidence chain. It is useful for review and replay but does not replace the canonical Decision or Journal records.

### KVShapeSnapshot

Records deterministic token-shape data used for cache eligibility. It may cite ContextBlocks, token hashes, prompt layout version, and KVCacheManifest ids. It is never memory.

### RuntimeSnapshot

Records runtime-driver configuration shape, such as model id, model revision, tokenizer revision, chat template, backend assumptions, and policy schema version. It may support replay or cache validation but does not authorize runtime behavior.

## Snapshot Rules

- A snapshot is admissible evidence only after Courthouse admission.
- A snapshot cannot override canonical state.
- A snapshot cannot hide supersession, contradiction, or rejection history.
- A snapshot cannot promote raw conversation into memory.
- A snapshot may seed restoration only when its source references and hashes remain valid.
- A snapshot may be compacted by Lymphatic Lane work, with compaction records preserved.
