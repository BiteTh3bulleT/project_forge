# Authority Boundaries

## Host authority

NixOS owns host configuration, package composition, services, generations, rollback, and workstation profile composition.

FORGE-K must not directly mutate host config.

## Tool authority

Gateway owns tool execution. All controlled file/process/network/tool actions must go through gateway, permissions, lanes, approvals, and audit.

## Model authority

Modelruntime owns model drivers, backend selection, scheduling, streaming, and runtime audit. Model output is proposal/evidence only.

## Semantic authority

FORGE-K target authority is semantic truth flow: validation, evidence admission, commit boundaries, journal/replay, contradictions, supersession, and lifecycle rules.

## Operator authority

The operator owns dangerous final authority: destructive actions, host mutation, cross-workspace changes, high-risk execution, and irreversible operations.
