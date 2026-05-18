# Shell System Surfaces

Phase G6 adds read-only FORGE-OS visibility to the graphical shell. The shell surface is operator visibility only; it is not a host control plane and is not a second authority path.

FORGE-K Online Phase 17 extends this surface with a read-only Operator Cockpit Index and display-only FORGE-K subsystem/storage readiness rows. Phase 18 adds read-only legacy-retirement proof metadata for retired direct mutation surfaces. It still does not add commands, approval execution, cleanup execution, storage switching, legacy mutation, or FORGE-K live authority.

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
- FORGE-K activation readiness and authority gate blockers
- operator cockpit index for gates, cases, context bundles, proposals, journal/replay, and lymphatic reports
- FORGE-K subsystem authority matrix rows when reported
- storage posture and read-only cutover readiness blockers
- legacy retirement proof for retired adapter and memory mutation surfaces
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
| FORGE-K activation readiness | Live Control Lane readiness report via `kernel_activation` | Show unavailable/error if core status is unavailable | Shows validation lane readiness, authority gate blockers, and disabled authority flags; no Kernel authority migration |
| Storage | SQLite/data-root status, disk summary, and storagebackend cutover readiness | Unavailable pressure if stat fails; blocked readiness if cutover evidence is missing | Shows used/free and readiness blockers when safely available; SQLite remains live truth authority |
| Legacy retirement | Static API retirement report backed by route/memory tests | Omit only if core status is unavailable | Shows retired route state, replacement path, and rollback proof; no route registration or mutation |
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
- switch storage backends, enable dual-write, or switch reads
- expose raw logs or memory dumps
- make FORGE-K simulator services live daemon authority

## Context Boundary

The shell may display structured system status to the operator. LLM-facing context still goes through the Context Compiler. Do not dump full host diagnostics, shell state, logs, memory, process lists, or modelruntime state directly into prompts.
