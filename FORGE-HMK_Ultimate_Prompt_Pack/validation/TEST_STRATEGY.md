# Test Strategy

## Contract tests

Validate required fields, zero-value rejection, serialization, schema versions.

## Authority tests

Worker cannot write canonical state. Crucible cannot commit directly. Cache cannot promote truth. FORGE-HMK cannot bypass Control Lane.

## Scheduler tests

Job admission, dedupe/coalescing, lease acquisition, heartbeat, timeout, retry, cancellation, backpressure.

## Memory tests

Cell scoping, synapse traversal depth, relation filtering, stale/superseded filtering, provenance preservation.

## Temporal tests

PhotoCell capture, KineticCell reconstruction, TraceCell append order, ReplayCell proposal validation, stale replay blocking.

## HKV tests

Exact identity, mismatch, TTL expiry, dependency invalidation, dirty-hit blocking, workspace isolation.

## Crucible tests

Missing provenance rejection, contradiction detection, supersession validation, stale evidence rejection, decision states.

## Shadow tests

Existing path unchanged, no public output change, safe metadata only, no canonical mutation, parity report generated.
