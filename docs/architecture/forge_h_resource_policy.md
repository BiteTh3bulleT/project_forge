# FORGE-H Resource Policy

Phase N4 implements FORGE-H as an advisory host-resource policy layer.
Phase N5 adds a governed Resource Action Proposal Gate on top of those advisory policy outputs.

FORGE-H consumes read-only Host Kernel Bridge diagnostic snapshots and produces bounded resource posture, workload lane decisions, model-load recommendations, background-work recommendations, warnings, and operator actions. Phase N5 can convert those recommendations into reviewable resource action proposals.

It gives FORGE resource judgment. It does not give FORGE host mutation authority.

## Role

FORGE-H answers operational questions such as:

- whether the machine is healthy enough for interactive work
- whether background ingest or embedding work should wait
- whether a model load should be allowed, warned, deferred, denied, or treated as unavailable
- whether RAM, swap, disk, VRAM, or thermal pressure is normal, elevated, constrained, critical, or unavailable
- whether operator-facing warnings should be shown

The implementation lives in `services/core/internal/forgeh` and imports the Host Kernel Bridge snapshot types from `services/core/internal/hostbridge`.

## Input

Input is a `hostbridge.Snapshot` from the read-only Host Kernel Bridge. The snapshot is diagnostic evidence only. It is not canonical semantic truth.

FORGE-H does not collect host data itself. It does not run commands, read `/proc`, read `/sys`, write files, call modelruntime, call gateway, execute retrieval, or write semantic memory.

## Output

FORGE-H emits a `ResourcePolicySnapshot` with:

- `policy_id`
- `captured_at`
- `source_snapshot_id`
- `overall_posture`
- `ram_pressure`
- `swap_pressure`
- `disk_pressure`
- `vram_pressure`
- `thermal_pressure`
- `lane_decisions`
- `model_load_recommendation`
- `background_work_recommendation`
- `warnings`
- `operator_actions`
- `source_errors`
- `advisory_only`

`advisory_only` is always `true` in Phase N4 and Phase N5.

## Resource Action Proposals

Phase N5 adds `ResourceActionProposal` records. These are operational governance records, not host execution commands and not semantic memory.

Supported proposal actions:

- `pause_background_ingest`
- `defer_background_ingest`
- `defer_embedding`
- `deny_new_model_load`
- `defer_large_model_load`
- `prefer_current_model_only`
- `prefer_cpu_safe_mode`
- `warn_operator`
- `enter_degraded_mode`
- `schedule_maintenance_later`

Each proposal preserves its source policy and host snapshot references. Proposal IDs are deterministic for stable proposal inputs.

## Proposal Lifecycle

Proposal statuses:

- `proposed`
- `approved`
- `rejected`
- `expired`
- `superseded`
- `committed_later`

Phase N5 allows only non-executing lifecycle transitions:

- `proposed` to `approved`
- `proposed` to `rejected`
- `proposed` to `expired`
- `proposed` to `superseded`

`committed_later` is reserved for future bounded execution phases and is rejected by the Phase N5 transition helper. Approving a proposal records review state only. It does not pause workers, defer queues, load or unload models, restart services, run Nix, or mutate host state.

Risk levels:

- `low`
- `moderate`
- `high`
- `critical`

All generated proposals require operator approval and remain advisory-only.

## Pressure Levels

Resource pressure values:

- `unavailable`
- `normal`
- `elevated`
- `constrained`
- `critical`

Thresholds are conservative:

| Resource | Normal | Elevated | Constrained | Critical |
|---|---|---|---|---|
| RAM | available >= 30% | available < 30% | available < 15% | available < 7% |
| Swap | used <= 25% | used > 25% | used > 50% | used > 80% |
| Disk | free >= 20% | free < 20% | free < 10% | free < 5% |
| VRAM | free >= 35% | free < 35% | free < 20% | free < 10% |
| Thermal | below 70 C | >= 70 C | >= 80 C | >= 90 C |

Missing GPU or thermal data is `unavailable`, not a policy failure.

## Workload Lanes

FORGE-H evaluates:

- `interactive`
- `desktop_ui`
- `background_ingest`
- `embedding`
- `model_load`
- `model_inference`
- `maintenance`

Policy decisions:

- `allow`
- `allow_with_warning`
- `defer`
- `deny`
- `unavailable`

Interactive and desktop UI work are preferred unless the overall posture is critical. Background ingest and embedding are conservative and defer under elevated or constrained RAM/disk pressure. Model-load decisions use RAM and VRAM pressure and remain recommendations only.

## Model Load Recommendations

FORGE-H may recommend:

- `small_local_ok`
- `current_model_only`
- `defer_large_model`
- `cpu_only_safe_mode`
- `deny_new_model_load`
- `unavailable`

It does not load models, unload models, spawn runtimes, call inference backends, or change modelruntime scheduler behavior.

## Advisory Boundary

FORGE-H is advisory only in Phase N4 and Phase N5.

Forbidden:

- host mutation
- service restart/start/stop
- `nixos-rebuild`
- kernel module load/unload
- package upgrades
- destructive cleanup
- model auto-load or auto-unload
- direct semantic memory writes
- public unauthenticated routes
- gateway, permissions, lanes, audit, controllane, modelruntime, or FORGE-K authority bypass

Future phases may turn recommendations into approved bounded controls only after explicit design, approval, tests, rollback, and authority-boundary documentation.

## Control Ladder

FORGE-H currently sits at the request-approval step:

1. Observe: Host Kernel Bridge read-only diagnostics
2. Report: diagnostic snapshots
3. Recommend: Phase N4 resource policy
4. Request approval: Phase N5 resource action proposals
5. Execute bounded action: future phase only
6. Automate safe action: future phase only
7. Own policy: future phase only
