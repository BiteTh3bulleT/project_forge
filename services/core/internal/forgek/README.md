# FORGE-K Simulator

`services/core/internal/forgek` contains the deterministic FORGE-K simulator. It is simulator authority only: the live daemon does not import FORGE-K as live truth authority yet, and live state mutation still uses the existing AI-OS, gateway, permissions, lane, and audit paths.

Do not route live daemon state through FORGE-K without an explicit `LIVE_INTEGRATION` phase, integration design, capability and journal tests, and updated ADR/status docs.

## Test Command

Run the FORGE-K simulator tests from `services/core`:

```bash
go test ./internal/forgek/...
```

## Phase Package Map

- Phase 1 Kernel Simulator: `kernel.go`, `syscalls.go`, `objects.go`, `journal.go`, `capabilities.go`, `providers.go`, `case_syscalls.go`
- Phase 2 Neuron Fabric: `neurons`
- Phase 3 Courthouse Minimal: `court`, `court_syscalls.go`
- Phase 4 Memory Palace Minimal: `palace`, `palace_syscalls.go`
- Phase 5 Semantic Algebra: `semantic`, `semantic_syscalls.go`
- Phase 6 Snapshots: `snapshots`, `snapshot_syscalls.go`
- Phase 7 Context Compiler: `contextcompiler`, `context_syscalls.go`
- Phase 8 Deterministic KV System: `kv`, `kv_syscalls.go`
- Phase 9 Runtime Driver Boundary: `runtime`, `runtime_syscalls.go`

## Authority Boundary

FORGE-K owns truth in the simulator model. In the current repository, FORGE-K is not live daemon authority. The simulator tests prove the target authority contracts for phases 1-9, while live daemon authority remains outside this package until a future scoped integration phase.

Phase 7 compiles ContextBlocks and ContextBundles as deterministic shape only. It does not call models, use KV cache, execute restore seeds, alter live AI-OS `COMPILE_CONTEXT`, or route live daemon state through FORGE-K.

Phase 8 registers deterministic KV manifests and validates identity gates as acceleration metadata only. It does not store real KV tensors, call model runtimes, perform live backend cache reuse, mutate ContextBundles or Snapshots, alter live AI-OS `COMPILE_CONTEXT`, or route live daemon state through FORGE-K.

Phase 9 is implemented as `SIMULATOR_ONLY / DRIVER_BOUNDARY_ONLY`. Runtime drivers are governed driver surfaces that may return proposal output with manifests, capability metadata, context refs, and KV metadata refs. The active implementation is a deterministic mock driver only. It must not call real model backends, mutate Kernel objects except through runtime syscalls, admit evidence, write snapshots or ContextBundles, register KV manifests, perform live KV reuse, alter live `modelruntime`, change gateway behavior, add routes, or route live daemon state through FORGE-K.
