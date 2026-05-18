# Architecture Blueprint

## Pyramidal deterministic architecture

```text
        FORGE-K / Control Lane
              ^
          Crucible
              ^
          FORGE-HMK
              ^
          FORGE-T
              ^
     Rule Cells / Hyperlane
              ^
      Inputs / Events / Jobs
```

Authority tightens upward. Labor expands downward.

## Component roles

### FORGE-K / Control Lane

Owns canonical writes, semantic syscalls, live authority, approvals, and audit requirements.

### Crucible

Owns contradiction refinement, provenance checks, supersession checks, current-state validation, and promotion readiness. Crucible does not commit truth directly.

### FORGE-HMK

Owns memory assembly, cell activation, synapse traversal, temporal traces, HKV metadata, non-canonical artifacts, VSA/semantic projection metadata, and evidence packet construction.

### FORGE-T

Owns job admission, timing, leases, retries, backoff, TTL, cache expiration, prewarm scheduling, replay windows, priority aging, and backpressure.

### Neuron Mesh

Owns bounded specialist work. Workers emit typed artifacts only.

## Data flow

```text
Input/Event
  -> Rule Cells / Hyperlane
  -> FORGE-T job admission
  -> Neuron Mesh work orders
  -> FORGE-HMK memory assembly
  -> HKV / vector / VSA / trace surfaces
  -> Crucible validation
  -> Context Compiler
  -> model/tool execution
  -> proposed memory write
  -> Crucible
  -> FORGE-K / Control Lane
  -> journal/audit/canonical state
```
