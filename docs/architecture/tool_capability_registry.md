# Tool Capability Registry

The capability registry is FORGE's normalized catalog of tool primitives.

It stores typed descriptors with:

- capability id (`domain.primitive`)
- status (`active`, `disabled`, `stubbed`, `approval_only`, `deprecated`, `deferred`)
- lane/effect/risk metadata
- approval and autonomy flags
- resource cost + resource limits
- adapter binding (`adapterId` + optional `gatewayToolId` in metadata)

## Behavior

Registry operations:

- register capability
- reject duplicate ids
- lookup by capability id
- resolve legacy gateway tool id -> capability id
- list all
- list by domain
- list by status
- list by risk
- enable/disable via status update

## Phase 5.997 Mapping

The full taxonomy is registered.

Current execution coverage is gateway-backed:

- every default capability has `gatewayToolId` metadata
- every default adapter id is a concrete `gateway.*` binding, not `stub.*`
- final default statuses are `active` or `approval_only`
- missing platform services, credentials, binaries, devices, or privileges return explicit runtime errors from the real gateway tool path

`stubbed` and `deferred` remain valid status vocabulary for explicit
operator override and compatibility tests, but they are not the default
production registry posture.

## Descriptor Contract

The descriptor contract is defined in:

- `services/core/internal/aios/domain/tool_surface.go`

Key fields:

- `id`, `domain`, `name`, `description`
- `status`, `lane`, `effect`, `risk`
- `requiresWorkspace`, `requiresIntent`, `requiresApprovalByDefault`
- `autonomyEligible`, `allowedInDryRun`
- `resourceCost`, `resourceLimits`
- `auditLevel`, `artifactBehavior`, `rollbackSupport`
- `adapterId`, `metadata`

## Safety

Registry metadata never bypasses the kernel:

- capability allowlist does not bypass permissions
- capability status does not bypass approval gates
- capability mapping does not bypass workspace/lane boundaries
- capability execution still goes through gateway policy + audit
