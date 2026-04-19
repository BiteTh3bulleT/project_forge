# Autonomy Charters

Autonomy charter = explicit FORGE freedom contract for a scope.

Charters do not bypass kernel validation, permissions, approvals, or audit.

## Charter model

Defined in `services/core/internal/aios/domain/autonomy.go` as `AutonomyCharter`.

Key fields:

- `id`, `name`, `description`, `purpose`
- `scope` (`workspaceId`, optional lane)
- `status` (`draft`, `active`, `suspended`, `revoked`, `expired`)
- `allowedActions`
- `deniedActions`
- `conditionalActions`
- `requiresApprovalActions`
- `allowedSources`
- `riskLimits`
- `freedomBudgetId`
- `effectiveFrom`, `expiresAt`
- provenance, creator/approver metadata, timestamps

## Action policy semantics

- `deniedActions` override everything.
- `requiresApprovalActions` can be allowed but still must escalate.
- `conditionalActions` gate action by deterministic conditions (risk thresholds, evidence flags, object constraints).
- inactive/suspended/revoked/expired charters do not authorize commits.

## Conditional action evaluation

`autonomy/charter.go` evaluates conditions deterministically.

Current built-in condition support:

- `note.status == superseded`
- `age_days > N`
- `no_active_state_depends_on_note`

Unknown conditions fail closed (blocked) with warning.

## Default conservative charters

Defined in `services/core/internal/aios/autonomy/defaults.go`.

### 1. `charter_memory_maintenance`

Allows:

- `CREATE_LINK`
- `CREATE_NOTE`
- `REGISTER_CONTRADICTION`
- `COMPILE_CONTEXT`

Conditional:

- `ARCHIVE_NOTE` only under explicit safe conditions.

Requires approval:

- `MARK_SUPERSEDED`
- `CLOSE_LOOP`
- `UPDATE_STATE`

### 2. `charter_open_loop_review`

Allows:

- stale-loop review notes/diagnostic updates

Requires approval:

- loop close operations

### 3. `charter_context_preparation`

Allows:

- context preflight/snapshot actions

Requires approval:

- external/export categories

### 4. `charter_projection_repair`

Allows:

- projection dry-run diagnostics and deterministic repair notes/links

Requires approval:

- higher-risk projection mutations

## Lifecycle expectations

Charter state transitions are auditable and scope-aware.

- `draft -> active` typically requires operator approval workflow.
- `active -> suspended/revoked/expired` immediately blocks new autonomous commits.
- historical charters remain inspectable.

## Mission mode constraint

`mission` autonomy mode requires a mission charter (`metadata.mission=true` or equivalent mission semantic marker). Without it, policy blocks mission auto-commit.
