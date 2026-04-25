# Dream Mode Review

## Scorecard

GOOD: Dream Mode v0 is deterministic, CPU-only, and dry-run by default.

GOOD: It does not commit canonical truth. Reports state canonical writes are not committed.

GOOD: It has replay selection, salience scoring, tier routing proposals, cleanup/repair recommendations, and API route `/api/dream/run`.

PARTIAL: It is closer to a deterministic maintenance report engine than a full lymphatic memory metabolism subsystem.

MISSING: Dream reports/proposals are not durably persisted as first-class evidence.

MISSING: Desktop operator flow for "run dream now / inspect dream report / approve proposals" is not obvious.

## Safety

GOOD: Dream Mode does not silently promote memories to long-term truth.

GOOD: No GPU/modelruntime dependency was found for the v0 path.

RISK: Without durable reports, the operator cannot audit why Dream Mode suggested promotion, demotion, merge, discard, repair, or cleanup after the response is gone.

## Missing Feed Loops

PARTIAL:
- Dream does not yet clearly feed restore utility signals.
- Dream does not yet persist repair outcomes.
- Dream does not yet maintain contradiction queues as durable operator workflow.
- Dream does not yet create approved training/adapter candidates.

## Recommended v1 Upgrades

1. Persist dream runs/reports as append-only non-canonical evidence.
2. Add operator UI for dry-run execution and report review.
3. Add approval-mediated commit path for any future canonical write.
4. Feed restore scoring with dream-derived utility/decay signals.
5. Add contradiction/failure/correction salience tests.
6. Add bounded scheduler policy with idle windows and budget reporting.

