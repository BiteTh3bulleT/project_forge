# FORGE Wiring Map

Date: 2026-04-24. K20A authority update: 2026-08-14.
Scope: one-page map of authoritative FORGE runtime wiring after cutover.
Companion to `docs/status/authoritative_paths.md` and
`docs/architecture/v1_v2_unification_plan.md`.

## Entry points

| Surface | Entry | Routed to | Governed by |
|---|---|---|---|
| HTTP API | `services/core/internal/api/server.go` (chi router) | handler methods on `*Server` | gateway + syscall processor + service layer |
| Desktop UI | `apps/desktop/src/lib/api.ts` | `http://127.0.0.1:18492/api/*` | same backend API; no client-side mutation bypass observed |
| Remote gateways | `/api/remote/telegram`, `/api/remote/discord` | chat/gateway wire | gateway + audit + approval (when applicable) |
| Autonomy runner | `internal/aios/autonomy/runner.go` loop | intent -> policy -> budget -> syscall/tool gateway | charter + policy + budget + kernel |

## Execution path (tools / adapters / shell)

```
client -> /api/gateway/invoke
      -> gateway.Service.Execute / ExecuteTool
         -> tool_capability_registry (status + risk)
         -> tool_policy (active / approval_only / explicit disabled or override states)
         -> permissions.Service (workspace, capability scope)
         -> approvals.Service  (request/decision, if gated)
         -> adapter invoke     (bounded worker)
         -> audit.Service      (correlation + actor + workspace)
         -> artifacts.Service  (evidence ref, if produced)
```

Legacy direct adapter invoke route has been removed from API routing.
Adapter execution must use `/api/gateway/invoke`; gateway is authoritative.

## Semantic mutation path (memory / state / truth)

```
caller (API handler, rule agent, autonomy runner, librarian cell)
   -> forgekernel.Kernel.Process(syscall)
      -> authority-claim gate
      -> Control Lane durable commit adapter
      -> registry (syscall type known?)
      -> validator.Validate(intent)
      -> policy/capability check
      -> processor_apply (sqlite transaction)
         -> repos (notes, links, state, open_loops, contradictions,
                   supersession, derived_models, context_snapshots)
         -> journal_events append
         -> audit.Service sink
```

Retired side door: `/api/memory/*` observation mutation endpoints still
exist only to return `410 Gone` and write audit evidence. Read-only memory
inspection remains available; mutation must use syscall-native paths.

## Autonomy path (self-initiated actions)

```
intent queue (sqlite-backed) -> policy_evaluator
  (charter authorizes? budget available? workspace in scope?
   durability backing present?)
  -> if not durable: downgrade to allow_propose_only
  -> if approval-gated: approvals.Service request
  -> runner.Execute
     -> semantic syscall path (memory/state)  OR
     -> gateway tool path (external effect)
     -> decision + audit recorded
```

Guarantees enforced in `autonomy/runner.go` + `policy_evaluator.go`:

- Rule-agent destructive commits blocked.
- Placeholder targets (`candidate-*`, `fake-*`, `placeholder-*`) blocked
  at commit.
- Maintain/mission auto-commit requires durable charter+budget, else
  downgraded to propose-only.

## Truth engine

- Journal (`journal_events`) is append-only truth.
- State (`state_items`, `state_versions`) is projected truth.
- `internal/aios/truth/engine.go` exposes current/historical/explain
  queries; contradictions and supersession preserve both sides.
- Derived models are advisory; never overwrite evidence.

## Audit / correlation

- Single substrate: `internal/audit`.
- Sinks: gateway (`audit logger`), controllane processor, autonomy
  decision recorder, approval flow.
- Correlation IDs propagated: `syscall_id`, `correlation_id`,
  `trace_id`, `audit_id`, provenance.

## Approval gate

- Single authority: `internal/approvals`.
- Request and decision are separated records.
- Gateway requests approval for `approval_only` capabilities; autonomy
  requests approval for high-risk or external-effect intents.

## Artifacts / evidence

- Single store: `internal/artifacts`.
- Gateway records artifact refs for tool outputs.
- Syscall path attaches artifact refs for commit outputs when present.

## Desktop → backend

- All desktop pages call `api.ts` which targets backend `/api/*`.
- Backend validates every mutation. No Tauri-level mutation bypass.

## Non-duplicate map (one authority per concern)

| Concern | Single authority |
|---|---|
| Tool execution | `gateway.Service` |
| Semantic syscall ingress | `forgekernel.Kernel` |
| Durable semantic commit adapter | `controllane.Processor` (temporary K20A adapter) |
| Approval records | `approvals.Service` |
| Audit records | `audit.Service` |
| Artifact evidence | `artifacts.Service` |
| Capability registry | `gateway/tool_capability_registry.go` |
| Capability policy | `gateway/tool_policy.go` |
| Autonomy state | `autonomy/sqlite_repositories.go` (`NewSQLiteBundle`) |
| Event journal | `internal/events` + `journal_events` |
| Job projections | `internal/jobs` |

## Known duplicate / ambiguous hotspots

See `docs/status/duplicate_systems.md`:

1. `internal/aios/compute` (runtime cells) vs `internal/aios/computelane`
   (interface seam). Explicit boundary; do not merge prematurely.
2. Semantic syscall path (authoritative) vs legacy `/api/memory/*`
   observation APIs (legacy boundary).
3. Backup export covers more tables than restore imports. Treated as
   forensic export until parity closed.

## Invariants the wiring preserves

- Cells / rule agents / autonomy / future IRIS propose only.
- Kernel / gateway validates and commits/executes.
- Events are truth; jobs are projections; packets are contracts;
  approvals are gates; artifacts are evidence; adapters are bounded
  workers.
- Current truth and historical truth are separate.
- Tool results are evidence, not automatic truth.
