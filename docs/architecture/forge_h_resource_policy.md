# FORGE-H Resource Policy

Phase N4 implements FORGE-H as an advisory host-resource policy layer.

FORGE-H consumes read-only Host Kernel Bridge diagnostic snapshots and produces bounded resource posture, workload lane decisions, model-load recommendations, background-work recommendations, warnings, and operator actions.

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

`advisory_only` is always `true` in Phase N4.

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

FORGE-H is advisory only in Phase N4.

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
