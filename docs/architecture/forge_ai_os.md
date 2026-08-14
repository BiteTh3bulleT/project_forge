# FORGE AI-OS Architecture Baseline

Status date: 2026-08-14
Scope: live AI-OS baseline including the K20A production FORGE-K ingress cutover.

## Read This If You Want

Read this if you want the current live FORGE AI-OS authority map: which runtime surfaces own semantic mutation, gateway execution, modelruntime, approvals, audit, and deterministic routing today.

## Core rule

FORGE is the AI-OS.  
IRIS is a future proposer inside FORGE, not a truth authority.

Rule of operation:

> IRIS proposes. FORGE validates. FORGE commits.

## Current authority map

| Concern | Current authority | Evidence |
|---|---|---|
| Canonical semantic syscall ingress | production FORGE-K Kernel | `services/core/internal/forgekernel/kernel.go`, daemon assembly in `services/core/internal/api/server.go` |
| Canonical durable commit | temporary Control Lane SQLite adapter | `services/core/internal/aios/controllane/processor.go`, `validator.go`, `processor_apply.go`, `sqlite_store.go` |
| Tool execution | gateway only | `services/core/internal/gateway/service.go`, `/api/gateway/invoke` in `services/core/internal/api/server.go` |
| Model execution and model management | model runtime service | `services/core/internal/modelruntime/service.go`, `management.go`, `store_management.go`, `services/core/internal/api/model_runtime*.go` |
| Approval and capability gates | approvals + permissions + gateway/tool policy | `services/core/internal/approvals`, `services/core/internal/permissions`, `services/core/internal/gateway/tool_policy.go` |
| Audit and trace linkage | audit service plus correlation/trace propagation | `services/core/internal/audit`, syscall/gateway/model-runtime bridge code |
| Deterministic reflex routing | Rule Cells / Hyperlane | `services/core/internal/aios/rulecells`, `docs/architecture/rule_cells.md`, `docs/architecture/hyperlane.md` |

## Memory taxonomy baseline

FORGE memory is classified across three temporal horizons, six processing functions, and nine memory types. The taxonomy is an operating map, not a new authority system:

- horizons: short-term, mid-term, long-term
- functions: capture, recall, route, score, consolidate, forget
- types: working, episodic, salience, prospective, reflective, utility, semantic, procedural, structural

Memory type never implies truth authority. Canonical memory still requires production FORGE-K syscall ingress followed by deterministic validation and the current durable commit adapter. Restore snapshots, Dream reports, retrieval/vector/VSA records, and restore outcome events remain non-canonical evidence unless a governed syscall promotes a specific claim.

Reference:

- [memory_taxonomy.md](memory_taxonomy.md)

## CPU/RAM kernel and GPU accelerator boundary

FORGE now treats CPU/RAM authority and GPU acceleration as separate operating lanes:

- kernel truth/control authority: CPU/RAM only (`forge-core`)
- inference acceleration: modelruntime boundary (`forge-modelruntime`)
- tool execution governance: CPU-side gateway path only (`forge-gateway`)

Boundary note:

- [cpu_ram_kernel_gpu_accelerator_split.md](cpu_ram_kernel_gpu_accelerator_split.md)

Rule Cells and Hyperlane run inside the CPU/RAM side of the system. They are deterministic reflex routers only: no modelruntime, no GPU, no network, no tool execution, no durable truth mutation.

## Landed Phase 3-5 architecture

### Phase 3: cognitive persistence is real

The control lane is no longer in-memory-only. Durable semantic persistence exists behind the same validator/processor path:

- cognitive tables and history are stored through `services/core/internal/aios/controllane/sqlite_store.go`
- commit boundaries run through `SQLiteTransactionRunner`
- `journal_events` remains append-only at the DB level
- context snapshot evidence persists in `context_packet_snapshots`
- restore outcome feedback persists in `restore_outcome_events` as non-canonical evidence
- `COMPILE_CONTEXT` can persist non-canonical snapshot evidence and restore-selection metadata

This is real Phase 3 persistence, but not full mutation convergence across the whole repo.

### Phase 4: ingest and cell runtime are real but bounded

The compute-side ingest flow exists and routes proposed actions back through kernel authority:

- ingest pipeline: `services/core/internal/aios/compute/librarian/pipeline.go`
- default cells and phase coverage: `cells_phase4.go`, `pipeline_phase4_test.go`
- truth engine is constructed inside the ingest pipeline and receives syscall outcomes
- autonomy pass is bounded by depth and skipped in dry-run or validate-only paths

What is not yet true:

- lane separation is still conceptual, not package-enforced
- full operator-facing ingest/runtime inspection is still limited
- maintenance/reporting depth is still narrower than the long-term lane model

### Phase 5: truth, autonomy, and tool policy are partially landed

Three important Phase 5 families are now real:

- truth engine services over current/history/contradiction/supersession:
  `services/core/internal/aios/truth/engine.go`
- bounded autonomy policy and maintenance runtime:
  `services/core/internal/aios/autonomy/*`,
  `services/core/internal/api/autonomy_maintenance_loop.go`
- governed tool capability surface and policy enforcement:
  `services/core/internal/gateway/tool_capability_registry.go`,
  `tool_policy.go`

What remains incomplete:

- legacy observation reads still coexist with cognitive filesystem reads for compatibility, but mutation endpoints are retired
- `events` remains an operational event projection while `journal_events` is canonical semantic truth
- operator trace/explain surfaces are weaker than backend trace data
- dangerous, external, privileged, or destructive tool capabilities are intentionally `approval_only`

## Runtime surfaces mapped to the AI-OS model

### Control Lane

Implemented responsibilities:

- syscall validation and commit
- approval and capability checks
- tool policy decisions
- autonomy policy preview/commit gating
- audit append and correlation linkage

Primary code:

- `services/core/internal/aios/controllane/*`
- `services/core/internal/approvals/*`
- `services/core/internal/permissions/*`
- `services/core/internal/gateway/*`
- `services/core/internal/audit/*`

### Compute Lane

Implemented responsibilities:

- ingest pipeline
- cell proposal generation
- truth projection/explanation services
- deterministic context compilation and restore scoring
- model runtime inference and management

Primary code:

- `services/core/internal/aios/compute/librarian/*`
- `services/core/internal/aios/truth/*`
- `services/core/internal/aios/autonomy/*`
- `services/core/internal/modelruntime/*`

### Rule Cell / Hyperlane Substrate

Phase 7 v0 adds deterministic Rule Cells as a CPU-local advisory substrate.

What is true:

- static rule packs are registered in `services/core/internal/aios/rulecells`
- rules are lane/phase filtered and priority ordered
- traces include matched rules, outputs, warnings, and pack id/version
- restore scoring and Dream Mode consume Rule Cell outputs as non-canonical evidence
- score adjustments are capped and final scores are clamped
- engine failures emit warnings and fall back to deterministic base behavior

What remains constrained:

- Rule Cells are not agents and spawn no processes
- Rule Cells do not commit truth
- Rule Cells do not execute gateway tools or modelruntime inference
- Rule Cells cannot loosen kernel, gateway, approval, capability, scope, or degraded-runtime denials

### I/O Lane

Implemented responsibilities:

- gateway tool invocation
- adapter compatibility routing through gateway
- artifacts/imports/backup boundaries
- API surfaces for operator and desktop clients

Primary code:

- `services/core/internal/gateway/*`
- `services/core/internal/adapters/*`
- `services/core/internal/artifacts/*`
- `services/core/internal/backup/*`
- `services/core/internal/api/*`

## Architecture boundaries that are true today

- No direct adapter invoke API route is registered; legacy adapter execution only remains as gateway tool `legacy.adapter.invoke`.
- `future_iris` is a proposer source class, not a bypass for tool or syscall policy.
- Model runtime is the owned inference substrate when enabled. Chat streaming is exposed through governed modelruntime SSE paths when a backend supports it, and gateway `model.*` aliases expose policy/taxonomy visibility without creating a second runtime execution path.
- Context restore snapshots are evidence, not truth authority.
- Restore outcome feedback is evidence about utility, not memory truth authority.
- Rule Cell traces and Dream reports are evidence, not truth authority.
- Vector, embedding, and VSA records are structural retrieval indexes, not truth authority.

## Known non-converged boundaries

- legacy memory mutation endpoints are retired; semantic mutation uses the syscall kernel.
- `events` and `journal_events` are both active runtime streams.
- The tri-lane model is architectural doctrine, not yet a hard package/runtime isolation boundary.
- Operator-facing trace exploration still lags the backend lineage that already exists in audit, gateway, syscall, and model-runtime records.
