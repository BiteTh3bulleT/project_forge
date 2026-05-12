# Nix Mutation Proposals

Status: DESIGN_ONLY / ADVISORY_ONLY / NO_HOST_MUTATION

## Intent

FORGE may propose NixOS changes, but proposal generation is not host authority. Applying a Nix mutation requires explicit operator approval, build evidence, rollback evidence, and a future governed host adapter.

## Proposal Object

Required fields:

- `proposalId`
- `createdAt`
- `createdBy`
- `scope`
- `targetHost`
- `risk`
- `status`
- `nixFiles`
- `diffSummary`
- `buildCommand`
- `testCommand`
- `rollbackPlan`
- `evidenceRefs`
- `approvalRefs`

Statuses:

- `draft`
- `needs_review`
- `approved_for_build`
- `build_failed`
- `build_passed`
- `approved_for_vm_smoke`
- `vm_smoke_failed`
- `vm_smoke_passed`
- `approved_for_apply`
- `applied`
- `rolled_back`
- `superseded`
- `expired`

## Intended Flow

1. Generate proposal.
2. Operator review.
3. Approve build.
4. Sandbox build.
5. Approve VM smoke test.
6. VM smoke test.
7. Record rollback plan.
8. Approve apply.
9. Apply through future governed host adapter.
10. Journal result.
11. Monitor outcome.

## Authority Boundary

The Tauri shell, model output, FORGE-K simulator, and ungoverned tools may not run `nixos-rebuild`, `systemctl`, package managers, reboot/shutdown, module loading, or destructive cleanup. Proposal records are evidence and review objects only until a governed adapter executes an approved action.
