# FORGE-OS Host Substrate

Phase N2 defines the first private host substrate foundation for FORGE-OS.

FORGE is the AI-OS operating environment. Linux and NixOS remain the boot and hardware substrate. This phase does not fork Linux, does not migrate live authority, and does not give FORGE autonomous host mutation powers.

Phase G1 extends this substrate model by defining FORGE as the graphical shell session for a FORGE-OS machine. This is not a web dashboard pointed at a headless server. It is the operator-facing desktop shell, launcher, workspace surface, command center, and system context surface running above the NixOS display/session layer.

## Stack

| Layer | Responsibility |
|---|---|
| Hardware | CPU, RAM, storage, network, GPU, sensors, firmware |
| Linux kernel | Booted kernel, drivers, cgroups, filesystems, networking |
| NixOS host config | Declarative users, groups, services, directories, environment |
| Display/session layer | Login/session activation, compositor/session plumbing, graphics environment |
| FORGE graphical shell | Visible operating surface for workspaces, launch, approvals, diagnostics, and governed FORGE interaction |
| FORGE services | Core daemon, databases, vector stores, runtime monitors, host diagnostics |
| FORGE-K / FORGE-H / modelruntime | Kernel simulator/shadow components, host-facing diagnostics, model runtime governance |
| FORGE shell-to-core boundary | Governed local APIs/interfaces into `forge-core`, gateway, approvals, audit, lanes, memory, HostBridge/FORGE-H, and modelruntime status |

The host substrate exists to make FORGE deployable as a private operating environment without pretending that FORGE is the hardware kernel.

NixOS remains the substrate. FORGE becomes the graphical shell above that substrate. The G1 shell session is opt-in and reversible; it must not replace an existing desktop environment, enable autologin, or mutate host configuration unless an operator explicitly imports and enables future NixOS module scaffolding.

## Storage Layout

Private FORGE-OS hosts should reserve `/forge` as the durable root.

| Path | Purpose |
|---|---|
| `/forge/data` | Canonical service data root, mapped to `FORGE_DATA_DIR` |
| `/forge/state` | Host-local operational state |
| `/forge/logs` | Service logs and diagnostics exports |
| `/forge/cache` | Regenerable caches |
| `/forge/backups` | Operator-managed backup targets |
| `/forge/imports` | Imported project/context evidence staging |
| `/forge/models` | Local model artifacts when used |
| `/forge/runtime` | Runtime metadata and process state |
| `/forge/journal` | Journal-oriented durable records when separated from data root |

NixOS scaffolding creates these directories with conservative ownership. It does not delete or migrate existing data.

## Service Map

| Service | Phase N2 Status | Notes |
|---|---|---|
| `forge-core` | Scaffolded | Disabled by default; defines systemd shape and safe env vars |
| Postgres/Qdrant/Redis | External for now | Container/local service bring-up remains outside this NixOS module phase |
| modelruntime | Existing live path | No runtime behavior changes |
| FORGE-K | Simulator/shadow only | No live authority migration |
| FORGE graphical shell | G1 inert Nix/session scaffold | Intended shell session surface; no Go, route, host mutation, autostart, compositor, desktop replacement, or authority changes in G1 |
| Host Kernel Bridge | Implemented library | Phase N3 read-only diagnostic snapshots; no startup wiring or public route |
| FORGE-H Resource Policy | Implemented library | Phase N4 advisory-only policy snapshots from Host Kernel Bridge diagnostics |
| FORGE-H Resource Proposals | Implemented library | Phase N5 request-approval records from resource policy snapshots; no execution or host mutation |
| FORGE-H Resource Execution | Implemented library | Phase N6 approved proposal execution through bounded internal adapters only; no host mutation, service control, modelruntime load/unload, or public route |

## Graphical Shell Role

The FORGE graphical shell owns the visible operating surface for a FORGE-OS session:

- desktop/workspace surface
- launcher and command palette
- workspace switcher
- window/panel surface
- system/resource status
- notification area
- approval queue
- memory/journal browser
- host diagnostics panel
- model/runtime status
- future Dream Mode review surface

G1 defines this role and its safety boundaries. It adds an opt-in NixOS module for session metadata and runtime-directory scaffolding. It does not implement a compositor, change packaging, autostart, replace the user's desktop, change the live daemon, or turn FORGE-K into live authority.

The initial session mode is `fullscreen-shell`. Future modes may include `kiosk`, `compositor-integrated`, `remote-operator`, and `multi-monitor-shell`. Until those modes have explicit implementation and tests, they remain design candidates only.

## Observation And Control Ladder

FORGE host powers must climb this ladder deliberately:

1. Observe
2. Report
3. Recommend
4. Request approval
5. Execute bounded action
6. Automate safe action
7. Own policy

Phase N2 stopped at design and scaffolding for observation/reporting. Phase N3 implements read-only diagnostic snapshot generation. Phase N4 reaches recommendation by producing advisory resource policy snapshots. Phase N5 reaches request approval by creating advisory resource action proposals that can be approved, rejected, expired, or superseded without execution. Phase N6 reaches bounded internal execution by recording approved proposal outcomes through explicit adapters. It still does not execute host actions or implement host mutation.

## Authority Boundary

Host diagnostics do not become canonical truth by themselves. They are evidence and operational context. Durable semantic writes still require existing validation, syscall, approval, audit, and commit boundaries.

No host substrate module may bypass gateway execution authority, permissions, lane governance, audit, controllane validation, memory authority, or modelruntime governance.

NixOS modules may prepare directories and environment flags only. They must not schedule, trigger, or perform resource action execution.

The FORGE graphical shell is also outside canonical authority. It may render system/session context and submit governed user requests to `forge-core`, but it must not directly run host commands or mutate canonical truth. The shell-to-core boundary must preserve:

- gateway execution authority
- permission checks
- lane governance
- approval gates
- audit records
- controllane validation
- semantic memory authority
- modelruntime governance
- FORGE-K simulator/live authority separation

Forbidden in G1 and in any shell path unless a later governed phase explicitly designs, tests, and documents the authority migration:

- direct `systemctl`
- direct `nixos-rebuild`
- direct kernel/module calls
- direct modelruntime load/unload
- direct filesystem cleanup
- direct semantic memory writes
- direct gateway execution bypass
- raw host mutation from UI controls
- treating FORGE-K simulator services as live authority

## System Context Principle

FORGE as graphical shell provides full operating awareness, not full LLM context.

The shell may observe structured system/session context such as active workspace, open panels/windows, current project, resource posture, service health, model status, approvals, recent errors, and user-triggered actions. That context is operational evidence for the UI and for governed request construction.

The context compiler decides what subset reaches an LLM. Raw full system state, raw desktop state, raw logs, raw memory contents, or complete host diagnostics must not be dumped into prompts. Model calls receive bounded, purposeful context through existing context compilation and governance paths.

## Rollback

Remove the NixOS module imports and disable `services.forge.core.enable`. Existing non-Nix workflows remain the operational fallback.

For future graphical shell session scaffolding, rollback must remain simple:

- leave existing desktop environments enabled unless the operator explicitly removes them outside G1
- keep autostart disabled by default
- keep safe mode enabled by default
- provide a non-FORGE desktop/session fallback
- allow disabling the FORGE shell session without deleting `/forge` data
- keep `forge-core` and existing npm/Tauri development paths usable

If a future shell session fails, the operator should be able to log into a normal desktop or TTY, disable the opt-in shell session module, and restart the display manager or reboot without losing canonical FORGE data.
