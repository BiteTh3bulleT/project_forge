# K20I — production deterministic semantic diff

Status: `PARTIAL LIVE AUTHORITY / SEMANTIC.DIFF.V1 ONLY`

K20I moves one bounded Semantic Algebra operation into production FORGE-K.
`COMPUTE_SEMANTIC_DIFF` accepts exactly `leftEvidenceId`, `rightEvidenceId`,
and `operatorVersion=semantic.diff.v1`. It does not accept caller text,
normalized content, output, confidence, authority claims, free-form parameters,
or a requested follow-on syscall.

## Authority boundary

The temporary durable adapter resolves both inputs from immutable
`forge_k_memory_evidence` leaves and binds their current Courthouse exhibit and
admitted ruling. The production Kernel independently verifies exact
workspace/lane/selected-path scope, current-leaf state, admission and content
hashes, K-owned provenance, request time, and distinct ordered inputs. It then
computes and seals the pure deterministic diff through
`services/core/internal/forgekernel/semanticdiff`.

The result class is always `NONCANONICAL_DERIVED_EVIDENCE`. It cannot become
current truth, admitted memory, or a VSA source without a later, separately
governed Courthouse and materialization action. Models, adapters, and Future
IRIS cannot execute this action. `legacy_v1` fails closed.

## Atomic persistence and replay

One SQLite transaction inserts the immutable semantic operation, immutable
result, immutable noncanonical derived object, provenance record, chained
journal event/head, audit-outbox intent, authorization proof, and idempotency
proof. Expected object IDs and source/result commitments are part of the sealed
prepared plan. A journal collision or any insert failure rolls the entire unit
back.

Verified replay reloads the original request, authorization proof, plan, seal,
receipt, and typed Kernel decision. It recomputes the decision, checks every
semantic plan commitment, and returns without writing new rows. Reusing an
idempotency key with changed semantic input fails closed.

The three semantic tables are included in full backup export and deterministic
inspection counts/checksums, but remain `offline_recovery_only`; live row merge
restore stays disabled.

## Scope and remaining gates

This slice does not claim general Semantic Algebra authority. Merge,
intersection, compression, derive, promotion, lifecycle mutation, and
caller-selected operator dispatch remain staged. A narrow authenticated API
ingress and extraction of the temporary Control Lane validation/apply/SQLite
port also remain required.

The companion `forgekernel/contextcompile` package is a pure, tested future
Context Compiler decision contract. It remains intentionally unconnected to
live `COMPILE_CONTEXT` until consistent source reads, Kernel plan/receipt
binding, scoped snapshot-head CAS, and legacy-path retirement land together.

## Validation expectations

- pure golden, normalization, permutation, bounds, and authority-tamper tests;
- production Kernel success, denial, verified replay, idempotency conflict,
  immutability, and journal-collision rollback tests;
- SQLite schema/store and static sole-writer guards;
- backup export/inspection coverage;
- full repository test, vet, formatting, and hygiene gates before push.
