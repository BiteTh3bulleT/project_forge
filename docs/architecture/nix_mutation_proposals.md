# Nix Mutation Proposals

Status: DESIGN_ONLY / ADVISORY_ONLY / NO_HOST_MUTATION

## Purpose

Nix mutation proposals define a future governed pipeline for NixOS and flake changes. A proposal may describe what should change, why it should change, what build/test evidence is required, and how rollback would work. A proposal is not permission to mutate the host.

This phase is design-only. It adds no public route, no shell mutation control, no `nixos-rebuild`, no `systemctl`, no package-manager execution, and no host adapter.

## NixMutationProposal

Required fields:

| Field | Meaning |
|---|---|
| `proposal_id` | Stable proposal identifier. |
| `created_at` | Creation timestamp. |
| `proposer` | Human, automation, model-worker, or system actor that proposed the change. |
| `reason` | Bounded explanation for why the change is proposed. |
| `target_scope` | Host, VM, profile, flake output, workspace, or module scope. |
| `affected_files` | Exact repository or host-config files expected to change. |
| `proposed_diff_ref` | Reference to the proposed diff artifact, not raw unbounded patch text. |
| `generated_by_model` | Boolean marker for model-generated proposal content. |
| `requires_operator_approval` | Must be true for host-impacting Nix changes. |
| `risk_level` | `low`, `moderate`, `high`, or `critical`. |
| `expected_services_changed` | Bounded list of affected services or session entries. |
| `expected_resource_impact` | Expected CPU/RAM/disk/GPU/network impact summary. |
| `build_command` | Build command to run in a sandbox/non-live context. |
| `build_result_ref` | Artifact/evidence ref for build result. |
| `vm_smoke_test_result_ref` | Artifact/evidence ref for VM smoke result. |
| `rollback_plan_ref` | Artifact/evidence ref for rollback plan. |
| `status` | Current lifecycle status. |
| `audit_ref` | Audit/provenance record ref. |
| `journal_ref` | Journal/phase evidence ref when later persisted. |
| `expires_at` | Expiration timestamp after which approval must be renewed. |

## Status Values

- `proposed`
- `rejected`
- `approved_for_build`
- `build_failed`
- `build_passed`
- `approved_for_vm_test`
- `vm_test_failed`
- `vm_test_passed`
- `approved_for_apply`
- `applied`
- `rolled_back`
- `superseded`
- `expired`

## Intended Flow

```text
Generate proposal
  -> operator review
  -> approve build
  -> sandbox build
  -> approve VM smoke test
  -> VM smoke test
  -> rollback plan
  -> approve apply
  -> apply through governed host adapter
  -> journal result
  -> monitor outcome
```

The build and VM stages are separate approval gates. Passing a build does not authorize apply. Passing VM smoke does not authorize apply. Apply requires an explicit approval after rollback evidence exists.

## Hard Rule

No direct `nixos-rebuild` from UI, model, shell, FORGE-K simulator, FORGE-H, or ungoverned tool path.

The only future apply path is a governed host adapter that verifies proposal status, approvals, build evidence, VM smoke evidence, rollback evidence, target scope, risk policy, and audit requirements before running any host mutation.

## Future Storage Shape

A future implementation may add pure records and tests in a focused package such as `services/core/internal/nixproposals`. That package should start with:

- proposal model validation;
- status transition validation;
- diff/evidence refs by reference only;
- risk-level validation;
- expiration validation;
- store tests against SQLite first.

It must not start with host execution.

## Forbidden In This Phase

- Host config apply.
- Direct shell command execution.
- Direct `systemctl`.
- Direct package-manager invocation.
- Kernel/module operations.
- Public mutation routes.
- Tauri buttons that execute host mutation.
- FORGE-K live authority expansion.
- Semantic memory writes.
- Modelruntime behavior changes.
