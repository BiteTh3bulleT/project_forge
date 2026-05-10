# Shell System Surfaces

Phase G6 adds read-only FORGE-OS visibility to the graphical shell. The shell surface is operator visibility only; it is not a host control plane and is not a second authority path.

## Endpoint

The desktop shell reads `GET /forge/system/status`. The route reports the
request-derived core URL as the bounded operator-facing core URL; it does not
discover or probe alternate hosts.

The endpoint is read-only and bounded. It returns summaries for:

- core reachability and health state
- shell/session mode and safety flags
- HostBridge diagnostic summary
- FORGE-H resource posture and advisory proposals
- bounded execution availability
- modelruntime availability
- storage posture
- approval queue wiring
- recent warnings

It does not expose request bodies, prompts, model outputs, raw host logs, raw memory records, or secrets.

## Surface Matrix

| Surface | Data source | Fallback behavior | Authority boundary |
|---|---|---|---|
| Core status | `forge-core` health/status composition | Show core unreachable | Shows reachability, core URL, health, and last refresh; no mutation |
| Shell session | safe session env/config metadata | Show unknown/not reported | Shows safe-mode and disabled-authority flags; no host authority |
| Host diagnostics | HostBridge summary with command-backed probes disabled | Show unavailable/source error count | Shows identity, pressure, GPU/thermal, source errors, degraded flag; diagnostics are evidence only |
| FORGE-H posture | Resource policy evaluation from HostBridge snapshot | Show not wired/unavailable | Shows pressure, recommendations, warning count; advisory only |
| FORGE-H proposals | Generated advisory proposals | Empty list when none | Shows advisory-only flag; no approve/reject/execute controls |
| FORGE-H executions | Execution ledger availability | Not wired unless governed store exists | Shows bounded/mutation/side-effect fields when available; no execution from shell |
| Modelruntime | Existing runtime health path when configured | Unavailable/degraded | No load/unload controls |
| Storage | SQLite/data-root status and disk summary | Unavailable pressure if stat fails | Shows used/free when safely available; SQLite remains live truth authority |
| Approval queue | Existing approvals surface wiring | Placeholder when unavailable | Decisions stay in governed approvals UI |

## Refresh Policy

The System surface supports manual refresh and a conservative 30 second refresh interval. It must not poll aggressively or hide stale/error states.

## No-Mutation Rules

The System surface must not:

- run `systemctl`
- run `nixos-rebuild`
- install, remove, or upgrade packages
- load or unload kernel modules
- reboot or shut down the host
- load or unload models
- execute tools
- approve, reject, or execute resource proposals
- write semantic memory
- expose raw logs or memory dumps
- make FORGE-K simulator services live daemon authority

## Context Boundary

The shell may display structured system status to the operator. LLM-facing context still goes through the Context Compiler. Do not dump full host diagnostics, shell state, logs, memory, process lists, or modelruntime state directly into prompts.
