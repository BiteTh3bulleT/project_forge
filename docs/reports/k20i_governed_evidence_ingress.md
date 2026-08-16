# K20I — governed evidence operator ingress

Status: `LIVE / NARROW AUTHENTICATED INTENT ROUTES`

The production daemon now exposes four narrow intent routes for already
implemented FORGE-K authority contracts:

- `POST /api/court/cases/{caseId}/exhibits`
- `POST /api/memory/evidence/{exhibitId}/materializations`
- `POST /api/memory/evidence/{priorEvidenceId}/revisions`
- `POST /api/memory/evidence/diffs`

These are not a generic syscall endpoint. Request DTOs contain only an exact
workspace/lane/selected-path scope and the minimum target identities. Every
route requires `Idempotency-Key`, derives request/correlation/trace identity,
actor, source, provenance, timestamp, capability policy, and operator version
server-side, and rejects unknown JSON fields.

Courthouse admission is initially limited to an existing immutable
`RECORD_RETRIEVAL_EVIDENCE` result. The server loads the result and its run,
requires their exact scope and FORGE-K syscall/provenance/authorization
binding, computes the content hash over a versioned canonical source record,
and selects the installed Court policy reference. Callers cannot supply
content, hashes, policy references, decisions, rulings, actor/source metadata,
or authority claims.

Materialization derives the current ruling from the persisted exhibit.
Revision derives the replacement ruling and requires an exact-scope prior
memory-evidence row. Semantic diff fixes the operator to `semantic.diff.v1`.
The production Kernel remains the only component that authenticates,
authorizes, decides, seals, commits, journals, and validates receipts.

End-to-end coverage exercises loopback-authenticated HTTP admission followed
by materialization and proves Court rows, immutable memory evidence, audit
outbox intents, and idempotency proofs commit through the real production
Kernel.
