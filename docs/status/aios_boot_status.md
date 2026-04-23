# AI-OS Boot Status

_Observed 2026-04-21. Focuses on whether current AI-OS layers prevent
current FORGE from starting, and what their default modes are._

## Subsystem-by-subsystem

| Subsystem | Authoritative path | Startup status | Default mode | Safe? | Stubbed/degraded |
|---|---|---|---|---|---|
| Semantic syscall registry | [internal/aios/controllane/registry.go](services/core/internal/aios/controllane/registry.go) | booted | deterministic | yes | no |
| Syscall validator | [internal/aios/controllane/validator.go](services/core/internal/aios/controllane/validator.go) | booted | deterministic | yes | no |
| Syscall processor | [internal/aios/controllane/processor.go](services/core/internal/aios/controllane/processor.go) | booted | deterministic | yes | no |
| Cognitive filesystem repositories | [internal/aios/controllane/repositories.go](services/core/internal/aios/controllane/) | booted | SQLite | yes | no |
| Truth engine | [internal/aios/truth/engine.go](services/core/internal/aios/truth/engine.go) | booted | read-only surfaces | yes | projection repair is scaffold-only |
| Librarian cells | [internal/aios/compute/librarian](services/core/internal/aios/compute/librarian) | booted | propose-only (inference seam is no-op) | yes | inference seam stubbed by default |
| Rule-based agents | [internal/aios/autonomy/rule_agents.go](services/core/internal/aios/autonomy/rule_agents.go) | booted | propose-only | yes | cleanup proposals emit no direct destructive targets by default |
| Autonomy runner | [internal/aios/autonomy/runner.go](services/core/internal/aios/autonomy/runner.go) | booted (goroutine) | `maintain` with default charters/budgets | **propose-only; external calls need approval** | see below |
| Tool capability registry | [internal/gateway/tool_capability_registry.go](services/core/internal/gateway/tool_capability_registry.go) | booted | status-aware | yes | many capabilities intentionally `stubbed` or `approval_only` |
| Tool policy evaluator | [internal/gateway/tool_policy.go](services/core/internal/gateway/tool_policy.go) | booted | deterministic | yes | no |
| IRIS seam | docs only | **deferred** | n/a | yes | no runtime code |

## Default autonomy posture (safety-critical)

- **Mode**: `maintain` (not `off`).
  [autonomy_maintenance_loop.go:106](services/core/internal/api/autonomy_maintenance_loop.go#L106).
- **Charters** auto-seeded: 4 active — all propose-only, scoped to
  maintenance operations.
- **Budgets** auto-seeded: 2 active — caps on self-actions, committed
  actions, and external calls. External-call budget is **0 without
  approval** on the default Memory Maintenance budget
  ([defaults.go:5-58](services/core/internal/aios/autonomy/defaults.go#L5)).
- **Dream tick**: 45 s, but only activates after 3 min of idle. On
  fresh boot, `dream.active=false` and `activeIntents=0` (verified
  live).
- **Effective boot posture**: autonomy is ON, but its default charter
  + budget combination keeps it propose-only with zero external-call
  budget. It cannot mutate canonical state without passing
  kernel/gateway validation and approvals.

This matches AGENTS.md invariants:

> FORGE may self-initiate only through the autonomy layer (intent +
> charter + budget + policy). Self-initiated semantic writes still
> require syscall/kernel validation and audit.

**Caveat.** `maintain` is aggressive compared to `off`. If an operator
wants zero autonomy on first boot, set `autonomy_mode=off` via the
Settings API or desktop UI before any workload. This pass does **not**
change the default — scope constraint.

## Incomplete v2 subsystems: do they break boot?

No subsystem currently breaks boot. Incomplete pieces degrade safely:

- **Librarian inference seam**: no-op by default. Cells emit proposals
  only when their deterministic conditions fire; no live LLM is
  consulted at boot.
- **Projection repair**: scaffold. Truth engine does not auto-repair on
  boot.
- **Duplicate compute lane** (`compute` vs `computelane` dirs): both
  packages exist; only one is wired via gateway. The matrix flags this
  as `duplicated` — not bring-up blocker but architecture debt.
- **Tool capsules, NixOS modules, IRIS**: all scaffold-only; never
  invoked.

## Guards that must not be relaxed for bring-up

- Kernel/gateway validation stays authoritative.
- Approval requirement on high-risk / external / destructive actions.
- Dangerous tool defaults remain `stubbed` or `approval_only`.
- Self-initiated syscalls still pass through the syscall processor.

None of the above were touched in this pass.
