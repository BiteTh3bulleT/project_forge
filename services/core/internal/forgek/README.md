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

## Authority Boundary

FORGE-K owns truth in the simulator model. In the current repository, FORGE-K is not live daemon authority. The simulator tests prove the target authority contracts for phases 1-7, while live daemon authority remains outside this package until a future scoped integration phase.

Phase 7 compiles ContextBlocks and ContextBundles as deterministic shape only. It does not call models, use KV cache, execute restore seeds, alter live AI-OS `COMPILE_CONTEXT`, or route live daemon state through FORGE-K.
