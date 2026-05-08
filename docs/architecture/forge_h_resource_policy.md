# FORGE-H Resource Policy

Phase N4 implements FORGE-H as an advisory host-resource policy layer.
Phase N5 adds a governed Resource Action Proposal Gate on top of those advisory policy outputs.
Phase N6 adds bounded execution records for approved resource action proposals through explicit FORGE-internal adapters.

FORGE-H consumes read-only Host Kernel Bridge diagnostic snapshots and produces bounded resource posture, workload lane decisions, model-load recommendations, background-work recommendations, warnings, and operator actions. Phase N5 can convert those recommendations into reviewable resource action proposals. Phase N6 can execute approved proposals only as bounded internal operational preferences.

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

Policy snapshots and generated proposals remain advisory. Phase N6 execution records bounded FORGE-internal operational policy only; they are not semantic memory, host commands, service control, modelruntime mutation, or live FORGE-K authority.

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

`committed_later` is reserved for later lifecycle integration and is rejected by the Phase N5 transition helper. N6 records separate execution records and does not convert proposals into host commands.

Risk levels:

- `low`
- `moderate`
- `high`
- `critical`

All generated proposals require operator approval and remain advisory-only.

## Bounded Resource Execution

Phase N6 adds `ResourceActionExecution` records. These records are created only when an approved, non-expired, advisory proposal passes the N6 allow-list and an explicit adapter is present.

Allowed N6 actions:

- `warn_operator`
- `defer_background_ingest`
- `pause_background_ingest`
- `defer_embedding`
- `deny_new_model_load`
- `defer_large_model_load`
- `prefer_current_model_only`
- `prefer_cpu_safe_mode`
- `enter_degraded_mode`
- `schedule_maintenance_later`

Execution adapters are narrow interfaces:

- `OperatorNotifier`
- `LanePolicyWriter`
- `ModelPolicyWriter`
- `DegradedModeWriter`

Adapters may record bounded internal operational preferences such as an operator notification, lane preference, model-load preference, or degraded-mode flag. They must not kill processes, stop services, delete queued work, change host config, run commands, load or unload models, call modelruntime, write semantic memory, or affect public routes.

Every execution record reports:

- `approved_before_execution`
- `bounded`
- `host_mutation`
- `semantic_memory_write`
- `modelruntime_mutation`
- `side_effects`

For Phase N6, `bounded` must be `true`; `host_mutation`, `semantic_memory_write`, and `modelruntime_mutation` must be `false`.

Execution is idempotent by proposal ID. Re-running the same approved proposal returns the existing execution record and must not duplicate side effects.

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

## Governed Boundary

FORGE-H policy and proposals remain advisory. Phase N6 adds bounded internal execution records, but still forbids host mutation, modelruntime mutation, and semantic memory writes.

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

Future phases may automate safe actions only after explicit design, approval, tests, rollback, and authority-boundary documentation.

## Control Ladder

FORGE-H currently sits at the bounded internal execution step:

1. Observe: Host Kernel Bridge read-only diagnostics
2. Report: diagnostic snapshots
3. Recommend: Phase N4 resource policy
4. Request approval: Phase N5 resource action proposals
5. Execute bounded action: Phase N6 bounded FORGE-internal policy action
6. Automate safe action: future phase only
7. Own policy: future phase only
