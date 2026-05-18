# Authority Gate Matrix

## Purpose

Make FORGE-K readiness visible and machine-checkable.

## Gate schema

| Field | Meaning |
|---|---|
| `subsystem` | FORGE-K subsystem name |
| `current_status` | SIMULATOR_ONLY / SHADOW_READ_ONLY / etc. |
| `live_owner` | Existing live daemon owner |
| `target_owner` | FORGE-K target owner |
| `feature_flag` | Disable/enable guard |
| `rollback_path` | How to return to previous behavior |
| `tests_required` | Required tests before promotion |
| `tests_passing` | Current evidence |
| `blockers` | Current blockers |
| `operator_visible` | Whether desktop/status surfaces expose it |

## Initial subsystems

- Kernel
- Courthouse
- Memory Palace
- Semantic Algebra
- Snapshots
- Context Compiler
- KV System
- Runtime Boundary
- Lymphatic Lane
- Consensus Mesh
