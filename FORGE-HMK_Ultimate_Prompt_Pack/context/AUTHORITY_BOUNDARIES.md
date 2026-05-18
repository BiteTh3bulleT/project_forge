# Authority Boundaries

## Prime directive

FORGE-HMK must not become a second live authority path.

## Chain

```text
Worker result
  -> typed artifact
  -> FORGE-HMK assembly
  -> ClaimEnvelope
  -> Crucible validation
  -> FORGE-K / Control Lane semantic syscall
  -> journal + audit + canonical state
```

## Allowed early behavior

- read current memory/retrieval paths
- build shadow memory bundles
- create non-canonical cells/traces/manifests
- emit telemetry
- run shadow comparisons
- generate claim envelopes

## Forbidden early behavior

- overwrite canonical memory
- bypass audit
- expose public mutation routes
- mutate live state from workers
- treat vector similarity as truth
- treat VSA similarity as truth
- treat cache hits as truth
- assume snapshots are current state

## Cache authority

HKV is acceleration only. A cache hit means previous work may be reusable if dependency gates pass. It does not mean the cached content is currently true.

## Worker authority

Neuron Mesh workers are propose-only. They can find, decode, score, warn, and propose. They cannot commit.
