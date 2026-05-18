# Repo Layout Recommendation

The implementing agent must inspect the actual repo before creating paths.

## Preferred docs

```text
docs/architecture/forge_hmk.md
docs/architecture/forge_t_temporal_tuner.md
docs/architecture/crucible.md
docs/architecture/neuron_mesh.md
docs/architecture/photo_kinetic_memory.md
docs/architecture/hkv_hierarchical_cache.md
docs/architecture/pyramidal_deterministic_architecture.md
docs/adr/00xx-forge-hmk-shadow-first-memory-kernel.md
docs/testing/forge_hmk_definition_of_done.md
```

## Preferred conceptual runtime modules

```text
services/core/internal/forgehmk/
  contracts/
  cells/
  synapses/
  temporal/
  hkv/
  neuronmesh/
  crucible/
  compiler/
  telemetry/
  shadow/
  store/

services/core/internal/forgetemporal/ or services/core/internal/temporaltuner/
  scheduler/
  governor/
  leases/
  coalescer/
  ttl/
  backpressure/
```

Follow actual repo naming conventions if they differ.

## Early integration rules

- internal packages only
- no public routes
- no canonical mutation
- no destructive migrations
- no required external services before local fallback exists
