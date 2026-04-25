# Runtime Subsystems Boot Status

_Observed 2026-04-21 on this branch during local boot checks against an empty data dir._

For each subsystem: initialization line, live boot status, and the
default posture.

## Summary

| Subsystem | Booted | Mode | Safe default? |
|---|---|---|---|
| Events logger | yes | append-only | yes |
| Jobs | yes | empty list | yes |
| Packets | yes | — | yes |
| Gateway (tool surface) | yes | autonomy authorizer wired | yes |
| Permissions | yes | defaults ensured | yes (see note) |
| Approvals | yes | — | yes |
| Audit sink | yes | SQLite-backed | yes |
| Artifacts | yes | `${DATA_DIR}` backed | yes |
| Adapters registry | yes | `claude_code`, `codex`, `ollama` visible | yes |
| Memory | yes | VSA indexer active iff VSA files present | see blocker |
| Retrieval | yes | `retrieval_vsa_mode=off` by default | yes |
| Control lane (syscall kernel) | yes | deterministic validator + processor | yes |
| I/O lane | yes | adapter-backed | yes |
| Compute lane | yes | librarian cells propose-only | yes |
| Truth engine | yes | read-only current/history APIs | yes |
| Autonomy runner | yes (goroutine) | mode=`observe`, 4 charters, 2 budgets | yes |
| Watch manager | yes | ingest file-watchers | yes |
| Backup service | yes | — | yes |
| Telegram gateway | **disabled** (no token) | — | yes |
| Discord gateway | **disabled** (default off) | — | yes |

## Detail

### Gateway / tool registry

- Init: [api/server.go:158](services/core/internal/api/server.go#L158)
- Wires an autonomy authorizer
  (`newGatewayAutonomyAuthorizer(autonomyLoop)`) so autonomy requests
  are policy-evaluated before commit.
- Dangerous tools (shell/process/external/privileged) remain
  `approval_only` per
  [dangerous_capabilities.md](dangerous_capabilities.md). Boot does not
  flip any risk flag live.

### Permissions

- Init: [api/server.go:141-150](services/core/internal/api/server.go#L141)
- Calls `EnsureDefaults()`, `EnsureMkdirChatPolicy()`,
  `EnsureGatewayToolPolicy()`. Errors are logged but not fatal — see
  [CONCERN] in [bringup_discovery.md §4](bringup_discovery.md).
- Default profile: `workspace-write` when `FORGE_WORKSPACE_DIR=/`.

### Approvals

- SQLite-backed request/decision records. No external queue.
- High-risk actions (from gateway or autonomy) return
  `approval_required`; operator approves via UI/API. Default is to
  gate, not auto-approve.

### Audit

- SQLite-backed. Every syscall commit carries `audit_id`. No external
  audit sink configured by default.

### Artifacts

- Filesystem under `${FORGE_DATA_DIR}`. `/api/artifacts` reads records;
  tool results reference artifacts by id.

### Memory

- **[Operational with strict VSA source preflight.]**
  On the current branch state (VSA files tracked) memory service
  exposes VSA methods (`GetObservationVSA`, `ReindexObservationVSA`,
  etc.). Endpoint `/api/memory/observations/{id}/vsa` returns 400
  validation error on missing observation (verified) rather than
  panic. A fresh clone missing tracked VSA files fails fast through
  the strict preflight before core/run/test paths.
- Memory observation mutation routes are retired. POST/PATCH/usefulness
  mutation endpoints return `410 Gone` and audit the denied attempt;
  read-only observation inspection remains available.

### Autonomy

- Init: [api/server.go:194](services/core/internal/api/server.go#L194) (goroutine)
- Mode default: `observe` (see [autonomy_maintenance_loop.go](services/core/internal/api/autonomy_maintenance_loop.go)).
  Can be set via `autonomy_mode` setting (`off` / `observe` / `propose`
  / `maintain` / `mission`).
- Default charters (4) and budgets (2) are created at boot. Default
  budgets disallow external calls without approval
  ([defaults.go:5-58](services/core/internal/aios/autonomy/defaults.go#L5)).
- Fresh boot is observation-only unless the operator explicitly sets
  `autonomy_mode` to `propose`, `maintain`, or `mission`.

### Telegram / Discord

- Both use the same pattern: `tryStart*` wrapper logs and returns nil
  on any init failure. Token absence is treated as the operator
  opt-out.
- `/api/telegram/status` and `/api/discord/status` report honest state
  (`ready=false`, `reason=...`) rather than 5xx.

## Issues found

| Subsystem | State | Reason | Next fix |
|---|---|---|---|
| Memory (VSA) | operational in current worktree | Required VSA files are present and `scripts/check-vsa-files.sh --require-tracked` passes in this workspace | Keep strict preflight in core/run/test paths so missing tracked files fail fast |
| Permissions | **silently degraded on error** | `EnsureDefaults`/`EnsureMkdir*` discard errors | Log errors; surface in `/api/meta` — not done this pass |
| Autonomy default | resolved | Mode defaults to `observe` with auto-created charters/budgets for inspection | Keep maintain/mission explicit operator choices |

No bypass of gateway/permissions/audit was identified in the sampled
boot paths reviewed this pass. No dangerous capability was observed
flipped to live on boot.
