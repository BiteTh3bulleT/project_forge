# Cursor Parallel Threading Guide — M5A

## Safe parallel lanes

### Worker A — Authority Matrix

Likely files:
- `services/core/internal/api/*authority*`
- `services/core/internal/system/*authority*`
- `docs/status/current_authority_sources.md`
- `docs/reviews/m5a_authority_convergence_review.md`

### Worker B — Modelruntime/Gateway Drift

Likely files:
- `services/core/internal/gateway/tool_capability_registry.go`
- `services/core/internal/gateway/tool_surface_test.go`
- `services/core/internal/api/model_runtime*_test.go`
- `docs/status/model_runtime_status.md`
- `docs/status/implementation_matrix.md`

### Worker C — HostBridge/FORGE-H Snapshot Cache

Likely files:
- `services/core/internal/hostbridge/*`
- `services/core/internal/forgeh/*`
- `services/core/internal/api/system_status.go`
- `services/core/internal/api/system_status_test.go`

### Worker D — System Cockpit UI

Likely files:
- `apps/desktop/src/pages/SystemPage.tsx`
- `apps/desktop/src/lib/api/*`
- `packages/shared/src/*`
- desktop tests

Do not start Worker D until Worker A/C define response shape.

### Worker E — Docs/Micro-Agent Design

Likely files:
- `docs/architecture/micro_agent_acceleration.md`
- `docs/status/m5a_latency_baseline.md`
- `docs/reviews/m5a_authority_convergence_review.md`

## Do not do concurrent edits on

- `services/core/internal/gateway/tool_capability_registry.go`
- `services/core/internal/api/system_status.go`
- `packages/shared/src/index.ts`
- `apps/desktop/src/pages/SystemPage.tsx`
- `docs/reviews/m5a_authority_convergence_review.md`

One owner at a time. Chaos is not a build system.
