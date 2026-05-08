# FORGE-OS Host Substrate

Phase N2 defines the first private host substrate foundation for FORGE-OS.

FORGE is the AI-OS operating environment. Linux and NixOS remain the boot and hardware substrate. This phase does not fork Linux, does not migrate live authority, and does not give FORGE autonomous host mutation powers.

## Stack

| Layer | Responsibility |
|---|---|
| Hardware | CPU, RAM, storage, network, GPU, sensors, firmware |
| Linux kernel | Booted kernel, drivers, cgroups, filesystems, networking |
| NixOS host config | Declarative users, groups, services, directories, environment |
| FORGE services | Core daemon, databases, vector stores, runtime monitors, host diagnostics |
| FORGE-K / FORGE-H / modelruntime | Kernel simulator/shadow components, host-facing diagnostics, model runtime governance |
| FORGE shell | Operator desktop and AI-OS surface |

The host substrate exists to make FORGE deployable as a private operating environment without pretending that FORGE is the hardware kernel.

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
| FORGE shell | Existing desktop app | No UI focus in Phase N2 |
| Host Kernel Bridge | Implemented library | Phase N3 read-only diagnostic snapshots; no startup wiring or public route |
| FORGE-H Resource Policy | Implemented library | Phase N4 advisory-only policy snapshots from Host Kernel Bridge diagnostics |

## Observation And Control Ladder

FORGE host powers must climb this ladder deliberately:

1. Observe
2. Report
3. Recommend
4. Request approval
5. Execute bounded action
6. Automate safe action
7. Own policy

Phase N2 stopped at design and scaffolding for observation/reporting. Phase N3 implements read-only diagnostic snapshot generation. Phase N4 reaches recommendation by producing advisory resource policy snapshots. It still does not request approval, execute host actions, or implement host mutation.

## Authority Boundary

Host diagnostics do not become canonical truth by themselves. They are evidence and operational context. Durable semantic writes still require existing validation, syscall, approval, audit, and commit boundaries.

No host substrate module may bypass gateway execution authority, permissions, lane governance, audit, controllane validation, memory authority, or modelruntime governance.

## Rollback

Remove the NixOS module imports and disable `services.forge.core.enable`. Existing non-Nix workflows remain the operational fallback.
