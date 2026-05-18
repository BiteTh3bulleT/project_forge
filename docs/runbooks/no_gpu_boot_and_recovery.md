# FORGE No-GPU Boot and Recovery Runbook

Status date: 2026-05-18.

This runbook describes how to run FORGE in CPU-authoritative degraded safe mode when no GPU is available or GPU runtime paths are unstable.

For the NixOS/operator VM safe-mode posture, also see
`docs/runbooks/forge_operator_desktop_vm.md` and
`docs/runbooks/forge_safe_mode_recovery_profiles.md`.

## Invariants

- `forge-core` remains bootable and authoritative on CPU/RAM.
- canonical truth writes remain syscall-bound.
- gateway approvals/capability checks remain active.
- modelruntime-dependent accelerators degrade gracefully.
- GPU-only/background workloads are deferred or rejected explicitly.

## 1) Force CPU-only safe mode

```sh
export FORGE_SAFE_MODE_FORCE_CPU_ONLY=true
export FORGE_GPU_ENABLED=false
```

Recommended conservative policy:

```sh
export FORGE_GPU_BACKGROUND_JOBS_ENABLED=false
export FORGE_DREAM_MODE_ALLOW_GPU_SUBJOBS=false
export FORGE_SCHEDULING_INTERACTIVE_PRIORITY_OVER_BACKGROUND=true
```

## 2) Boot core

```sh
npm run core
```

Or with explicit data/workspace:

```sh
FORGE_DATA_DIR=/tmp/forge-safe/data \
FORGE_WORKSPACE_DIR=/tmp/forge-safe/workspace \
FORGE_CORE_PORT=18492 \
  npm run core
```

## 3) Verify degraded-safe-mode posture

```sh
curl -s http://127.0.0.1:18492/health | jq .
curl -s http://127.0.0.1:18492/forge/model-runtime/health | jq .
curl -s "http://127.0.0.1:18492/api/process/health?correlationId=test" | jq .
```

Expected characteristics:

- `/health` reports `cpuAuthoritative=true` and safe-mode metadata.
- modelruntime health state is visible and may report `degraded`/`unavailable` with explicit reasons.
- process health reports runtime safe-mode flags/reasons without crashing core services.

## 4) Recovery from unavailable/saturated GPU

1. Keep CPU-only safe mode enabled while stabilizing providers/backends.
2. Inspect runtime status using `/forge/model-runtime/health`, `/forge/model-runtime/queue`, `/forge/model-runtime/backends`.
3. Re-enable GPU only after health is stable and interactive queues are normal.
4. Re-enable background GPU classes last:
   - `FORGE_GPU_BACKGROUND_JOBS_ENABLED=true`
   - keep `FORGE_SCHEDULING_INTERACTIVE_PRIORITY_OVER_BACKGROUND=true`.

## 5) Failure behavior expectations

- interactive inference that requires GPU returns explicit policy/runtime errors.
- background GPU classes are deferred/rejected by policy, not retried in storms.
- canonical writes/journal/audit and gateway approvals continue functioning in CPU mode.
