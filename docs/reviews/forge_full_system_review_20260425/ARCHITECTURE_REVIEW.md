# Architecture Review

## Strengths

GOOD: FORGE has a coherent doctrine and the implementation mostly respects it:

- Kernel/control lane owns canonical semantic mutation.
- Gateway owns governed tool execution.
- Modelruntime owns inference.
- Dream/rule/autonomy paths are proposal-first.
- CPU/RAM authority is separate from GPU acceleration.
- Audit/correlation/trace are first-class data surfaces.

GOOD: The code avoids the largest architectural failure mode: there is no separate Python sidecar monolith or duplicate model provider stack competing with FORGE core.

GOOD: The adapter direct execution route has been retired; compatibility now flows through the gateway.

## Partial Areas

PARTIAL: Neural/Arterial/Lymphatic lane language is mostly conceptual. Packages exist for `aios`, `compute`, `iolane`, `dream`, and autonomy, but hard runtime isolation is not enforced.

PARTIAL: Operator-facing syscall path is missing. Internal compute/autonomy can commit through the kernel, but a small governed API for semantic syscall dry-run/submit/inspect is not present.

PARTIAL: `events` and `journal_events` both exist. This is acceptable if `events` is treated as operational projection, but docs and UI should not blur it into canonical truth.

## Architecture Risks

RISK: Model management state changes are direct API operations. They are audited by modelruntime, but not equivalent to gateway/approval policy.

RISK: Config and authority-affecting API writes, including lanes, permission profiles, settings, sources, and some model operations, are not uniformly approval-gated.

RISK: Context restore scoring is implemented, but the candidate selection layer undermines the intended ranker behavior by filtering exact query in SQLite.

## Recommendations

RECOMMENDATION: Add an `authority convergence` pass for operational config/model management. Treat these as authority-adjacent and require audit plus approval for privilege-expanding/destructive transitions.

RECOMMENDATION: Add package guard tests for compute/autonomy/dream paths to prevent direct cognitive DB writes outside the control lane.

RECOMMENDATION: Make `events` explicitly an operational projection in API/UI labels and reserve `journal_events` language for canonical semantic truth.

