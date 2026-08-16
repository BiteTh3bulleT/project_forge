# K20I lane C — production context compile decision contract

Status: `PURE PRODUCTION CONTRACT / LIVE INGRESS CONTAINED / DECISION NOT YET INTEGRATED`

`services/core/internal/forgekernel/contextcompile` is the production-owned,
model-free decision contract for a future `COMPILE_CONTEXT` cutover. It has no
database, clock, model, gateway, global cache, API, Control Lane, or simulator
dependency. The live action is now contained: `COMPILE_CONTEXT` requires the
production FORGE-K ingress, adapter and Future-IRIS sources are denied,
persisted compilation requires an idempotency key, and the authority-affecting
restore cache is retired. Control Lane still computes and applies the live
packet/snapshot decision, so this pure contract is not yet live authority.

## Sealed v1 policy

Policy version `forge.context_compile.policy.v1` hashes every limit and weight.
The compiler rejects a caller-supplied policy even when its digest is
internally consistent if any field differs from the production v1 value.

Limits:

| Field | v1 limit |
|---|---:|
| selected paths | 64 |
| normalized query bytes | 8,192 |
| admitted source commitments | 256 |
| candidate snapshots | 128 |
| feedback projection heads | 128 |
| IDs in each hint list | 128 |
| token budget | 131,072 |
| byte budget | 4,194,304 |
| individual identity/path bytes | 4,096 |
| helpful or harmful events per feedback head | 1,000,000 |

All scores are signed fixed-point integers with scale 1,000. There are no
floating-point inputs or operations.

| Score component | v1 value |
|---|---:|
| exact normalized query hash | +4,000 |
| exact K20H source manifest | +2,500 |
| snapshot kind match | +1,000 |
| preferred snapshot hint | +750 |
| exact prior-head continuity | +500 |
| linear recency within 86,400,000 ms | 0..+750 |
| header-only option | -500 |
| each helpful outcome | +100 |
| each harmful outcome | -200 |
| total outcome adjustment clamp | -1,000..+1,000 |
| default restore threshold | 5,000 |

Restore additionally requires `allowRestore`, exact query identity, and exact
source-manifest identity. Ranking is total score descending, candidate
`createdAt` descending, `snapshotId` ascending, then `snapshotHash` ascending.
If none passes, `fresh_compile` is an explicit committed selection result.

## Input and decision commitments

The input binds the normalized query, exact workspace/lane and sorted selected
paths, bounded budget/options/hints, caller-supplied `requestedAt`, policy
version/digest, sorted current admitted K20H evidence commitments, immutable
candidate snapshot commitments, exact-scope governed feedback projection
heads, and the optional prior snapshot head.

Each K20H source must bind its semantic and row identities, revision/root,
Court case/exhibit/current admitted ruling and admission syscall, source object
kind/version/hash, exact scope, source and materialization provenance,
transaction, journal, audit outbox, authorization fingerprint, and
`forge_k.kernel` commit identity. Legacy observations and caller-asserted
non-current/non-admitted sources fail closed.

The compiler also recomputes each candidate snapshot hash, feedback projection
head hash, and prior snapshot head hash from its normalized commitment fields.
A syntactically valid `sha256:` value is insufficient when its content has
diverged.

The decision binds:

- normalized request hash and admitted source manifest hash;
- exact selected evidence IDs and packet commitment;
- snapshot commitment and prior snapshot head hash;
- complete candidate-set and prior-snapshot-head commitments;
- the complete ordered fixed-point restore score table and its commitment;
- selection commitment;
- supplied feedback-head commitment and the resulting restore/fresh outcome
  commitment;
- card commitment, including the explicit render/no-render bit;
- policy digest and final decision digest.

Packet and snapshot IDs are deterministic prefixes of their commitments. They
are identities for later persistence, not proof that persistence occurred.

## Exact future integration seams

1. **Read adapter / preflight snapshot** — inside a consistent SQLite read
   transaction, load only exact-scope current Court-admitted
   `forge_k_memory_evidence` leaves; immutable context snapshot candidates;
   their governed outcome projection singleton heads; and the exact current
   context snapshot head. Construct the pure input without inventing missing
   authority. The adapter must recompute each table-level commitment.
2. **Kernel decision** — call `contextcompile.Compile` before prepare. Any
   malformed hash, duplicate, cross-scope row, future timestamp, missing
   provenance, stale admission/current marker, non-K commit, or cardinality
   violation rejects the syscall.
3. **Prepared plan** — seal the normalized request hash, source manifest hash,
   prior snapshot head, policy digest, decision digest, packet/snapshot/score
   table/selection/feedback/outcome/card commitments, and exact expected object
   IDs. Apply must receive this sealed plan unchanged.
4. **Atomic durable apply** — in the existing Kernel-owned apply+journal
   transaction, insert immutable packet, snapshot, restore-score, selection,
   outcome, and optional rendered-card evidence. Advance the exact scoped
   snapshot head with compare-and-swap: first head uses insert-on-absent;
   replacement uses `WHERE head_hash = expected_prior_head`. A missing or
   changed head aborts every insert.
5. **Journal/audit/idempotency** — append the semantic journal event, audit
   outbox intent, and idempotency replay proof in that same transaction. The
   event payload binds the full decision digest and every subordinate
   commitment, not rendered prompt text.
6. **Commit receipt** — return packet ID/hash, snapshot ID/hash, score-table,
   selection, feedback, outcome, card, source manifest, prior/new head, policy,
   decision, journal event, transaction, outbox, authorization, and
   idempotency fingerprints. Kernel validation recomputes the decision and
   requires exact receipt equality before success.
7. **Replay** — reload the original request, sealed input commitments, prepared
   plan, and receipt. Recompute the pure decision and verify the active head or
   immutable historical snapshot identity. Legacy `{}` replay rows fail
   closed; no fresh DB reads may silently substitute a different source set.

No adapter may write canonical memory, call a model, render a prompt through a
runtime driver, or use a process-global score cache during these stages.

## Verification

Unit, golden, permutation, bounds, authority-tamper, and fuzz tests cover
canonical normalization, source-manifest verification, exact scope, duplicate
rejection, fixed-point clamps, explicit fresh compile, stable tie order, and
decision identity. Changing any v1 weight, limit, field, normalization rule,
or commitment shape requires a new policy/contract version and reviewed golden
vector.
