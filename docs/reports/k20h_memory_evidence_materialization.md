# K20H — admitted Memory Palace evidence materialization

Status: `PARTIAL FORGE-K CUTOVER / PRODUCTION ACTIONS LIVE / LEGACY ROWS UNTRUSTED`

K20H adds two production-only semantic syscalls at the live
`internal/forgekernel` boundary:

- `MATERIALIZE_ADMITTED_EVIDENCE` accepts only `exhibitId` and `rulingId`.
- `REVISE_MEMORY_EVIDENCE` accepts only `exhibitId`, `rulingId`, and
  `priorEvidenceId`.

The caller cannot supply memory text, tags, source references, raw references,
or a content hash. The durable adapter derives those fields from the persisted
Court exhibit and its named current admitted ruling. It requires exact
workspace, lane, and selected-path scope, matching exhibit/ruling hashes, a
production Court commit identity, authenticated user or `forge.core` service
authority, provenance, an idempotency key, and the ordinary sealed FORGE-K
commit plan/receipt.

## Durable contract

`forge_k_memory_evidence` has a stable integer projection identity and a unique
semantic `evidence_id`. Rows bind Court case/exhibit/ruling/admission identity,
the source object version/hash, exact scope, derived content, source and
materialization provenance, and transaction/journal/audit-outbox/idempotency/
authorization fingerprints. The table is immutable.

`forge_k_memory_evidence_supersessions` is append-only. Unique superseded and
replacement evidence IDs, same-case/scope validation, and a prior-leaf check
make each revision chain linear. Revision creates a new evidence row plus an
edge in the same transaction; the original row is never rewritten. Legacy
`memory_observations` identities are not aliased or promoted.

## Governed VSA source and projection

The VSA v2 source manifest accepts only exact-workspace/lane evidence rows
that still have no outgoing supersession edge and whose persisted Court
exhibit names the same current admitted ruling. Exhibit/ruling case, scope,
content hash, admission syscall, and `forge_k.kernel` identities must match,
and the evidence source-object, provenance, transaction, journal, outbox,
idempotency, and authorization identities must be complete. Manifest identity
uses the semantic `evidence_id` plus the stable evidence row identity; legacy
observations are never members of this source set.

Governed acceleration rows live in separate
`forge_k_memory_vsa_pointers`, `forge_k_memory_vsa_role_bindings`, and
`forge_k_memory_vsa_associations` tables whose foreign keys target
`forge_k_memory_evidence(id)`. They never place an evidence row ID into a
legacy `observation_id` column. Manifest rows carry
`source_kind=forge_k_memory_evidence`; runtime scoring accepts only the v2
manifest version, the exact scoped active head, and a complete current-leaf
row count matching the sealed manifest. A revision therefore disables stale
head influence until a governed rebuild atomically installs the replacement
projection.

Legacy observation VSA tables remain historical inspection data only. They do
not join the governed active head and cannot affect retrieval scoring.

Full backups export both tables with deterministic count/checksum inspection.
They are offline-recovery-only evidence and can never be live-merged by the
row restore path.

## Validation evidence

Focused tests cover authenticated-user success, Court admission followed by
materialization and revision, caller-field rejection, empty or mismatched
scope, exhibit/ruling hash mismatch, exact expected object IDs, idempotent
replay, immutable triggers, injected transactional rollback, and concurrent
prior-leaf contention. The end-to-end SQLite test proves that the first
materialized leaf is the only VSA source, the replacement becomes the only
planned source while the original remains queryable, and an injected governed
swap failure preserves both immutable evidence rows, the supersession edge,
the prior projection rows, and its head. Journal-collision tests separately
prove that evidence and proof rows roll back together. After removing the
injected failure, the same sealed Kernel rebuild installs the replacement as
the sole active semantic evidence pointer, and scoped runtime scoring reports
that replacement without an `observation_id` alias.

## Remaining boundary

FORGE-K remains partial. Retrieval callers that do not carry an exact nonempty
workspace/lane scope still receive zero VSA influence, and K20H does not define
governed semantic relationship edges or a governed usefulness aggregation for
the new evidence rows. It does not make legacy observations authoritative,
add a model write path, migrate search/embedding computation, or complete the
Context Compiler, Runtime, Snapshot, KV, Lymphatic, Consensus, external-audit,
and whole-store recovery cutovers. Any future API for these actions must enter
through authenticated production Kernel context; the retired legacy memory
handlers must not be reconnected.
