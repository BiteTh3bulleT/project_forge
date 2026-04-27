# Dangerous Capability Inventory (Phase 5.99)

Date: 2026-04-21
Scope: preserve full taxonomy, enforce explicit executable posture

## Taxonomy preserved

FORGE keeps the full capability universe across domains (`filesystem`, `process`, `network`, `identity`, `external`, `code`, `ui`, etc.).
Registered means the capability resolves to a gateway tool path. Dangerous
or privileged capabilities still require approval, and unavailable platform
dependencies return explicit runtime errors.

## Classification buckets

### 1) Known disabled
- No broad default disabled blocklist in current static registry snapshot.
- Runtime disable overrides are possible; registry status for many ids remains advisory and must be persisted through policy review workflow.

### 2) Stubbed/deferred defaults
- None in the production default registry snapshot.
- `stubbed` and `deferred` remain supported only as explicit override/status semantics.

### 3) Known approval-only
High-risk mappings are explicitly `approval_only` in `activeMappings`, including (selection):
- `filesystem.delete_file`
- `filesystem.set_permissions`
- `filesystem.restore_snapshot`
- `process.spawn_process`
- `process.kill_process`
- `network.http_request`
- `network.open_socket`
- `network.scan_network`
- `network.open_tunnel`
- `network.intercept_traffic`
- `network.set_firewall_rule`
- `code.run_shell`
- `code.eval_code`
- `config.restore`
- `backup.restore`
- `config.migrate_schema`
- `config.backup`
- `identity.retrieve_secret`
- `identity.decrypt`
- `identity.sudo`
- `identity.switch_user`
- `identity.issue_token`
- `identity.set_policy`
- `external.send_email`
- `external.post_message`
- `external.call_api`
- `external.create_issue`
- `external.update_issue`
- `filesystem.sync_to_remote`
- `ui.open_url`
- `ui.inject_input`
- `device.capture_camera`
- `device.capture_audio`

### 4) Active and safe
- `filesystem.read_file`
- `filesystem.list_dir`
- `network.dns_resolve`
- `code.diff_code`
- `ui.show_notification`
- `time.get_system_time`

### 5) Active and risky
- `filesystem.write_file` (medium)
- `filesystem.move_file` (medium)
- `observability.read_logs` (medium)

### 6) Unknown status
- Runtime-mutated capability statuses (if changed after registry initialization) are unknown in this report.

### 7) Approval-only / configured dependency paths
- High-risk capabilities resolve to real gateway tools but are not freely executable.
- Platform, credential, binary, device, or privilege gaps are reported as explicit runtime errors after policy/approval checks.

## Policy behavior

- `approval_only` returns `approval_required` decision path with structured tool error.
- `disabled` and `deprecated` deny execution with structured error.
- Explicit `stubbed` without adapter returns deterministic unsupported operation.
- Explicit `deferred` returns deterministic unsupported operation.
- Self-initiated high-risk execution requires autonomy policy/approval flow.
- `future_iris` does not bypass policy in current tests.
- Gateway terminal status decisions (`needs_approval`, `unsupported`, `disabled`) produce explicit audit entries (`tool.needs_approval`, `tool.unsupported`, `tool.disabled`).
- Capability status governance is operator-visible and auditable through `PATCH /api/gateway/capabilities/{id}/status`.
- Capability status changes are authority-shaping mutations. Mutating transitions require explicit actor/provenance and reason metadata.
- High-risk elevation, including dangerous `approval_only -> active` and disabled/deferred/stubbed/deprecated -> active transitions, requires a matching approval request fingerprint before the override is persisted.
- Safe lowering transitions such as `active -> approval_only`, `active -> disabled`, and `approval_only -> disabled` remain allowed without approval, but still persist actor, reason, previous/new status, transition risk, correlation id, trace id, and audit linkage.
- Direct legacy/stale gateway status update calls cannot activate dangerous capabilities without approval metadata.

## Remaining hardening backlog

1. Expand tests for capability-level workspace path boundaries and policy overrides.
2. Keep retired non-capability mutation routes non-executable.
3. Add more service-specific harness tests for configured dependency failures.
4. Add operator UI affordances for the new approval-required capability status response.
