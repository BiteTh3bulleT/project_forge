# Dangerous Capability Inventory (Phase 5.99)

Date: 2026-05-18
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
- No production default capability is `stubbed`.
- Deferred defaults are explicit and include model-runtime work that is not directly executable through the gateway surface:
  - `model.embed`
  - `model.delete_file`
  - `model.benchmark`
- `deferred` returns deterministic unsupported behavior until a separate implementation/approval path is wired and tested.

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
- Runtime-mutated capability statuses are persisted as explicit overrides.
- High-risk elevation is not treated as an unknown state: dangerous elevation requires approval metadata before persistence, and direct/stale status mutation paths cannot activate dangerous capabilities without that approval metadata.

### 7) Approval-only / configured dependency paths
- High-risk capabilities resolve to real gateway tools but are not freely executable.
- Platform, credential, binary, device, or privilege gaps are reported as explicit runtime errors after policy/approval checks.

### 8) Desktop shell host power actions (policy-gated, disabled by default)
- `shell.power_action` — desktop shell host `shutdown` and `reboot` requests exposed by the Tauri binary `request_host_power_action` command.
- Disabled by default. The binary refuses to spawn host power commands unless the operator explicitly sets the environment variable `FORGE_SHELL_DIRECT_SYSTEM_CONTROL` to `1`, `true`, `TRUE`, `yes`, or `YES`.
- Gate function: [`direct_system_control_enabled` in `apps/desktop/src-tauri/src/main.rs`](../../apps/desktop/src-tauri/src/main.rs). Enforcement: `request_host_power_action_with_policy` returns a `requested:false` `HostPowerActionResult` with the message "Host power controls are disabled by FORGE_SHELL_DIRECT_SYSTEM_CONTROL policy" when the gate is not set, and only invokes the host command runner when the gate is enabled.
- UI policy read: `read_host_power_policy` exposes the gate state to the desktop. The Start menu keeps Logout and Restart Shell available, disables Shutdown/Reboot while the gate is off, and does not call `request_host_power_action` for disabled policy state.
- Allowlist: only `shutdown` and `reboot` actions are accepted by `spawn_host_power_command`; any other action returns "host power action is not allowlisted".
- Regression coverage: Rust unit tests cover disabled/enabled host-power policy, and desktop AppShell tests cover disabled Start menu host-power controls.
- Enabling this gate grants host mutation authority to the desktop shell. It is therefore an operator-set, opt-in, disabled-by-default dangerous capability and supersedes prior docs language that described the shell as "no host mutation". See `docs/DESKTOP_SHELL.md` "Host Power Controls" and `docs/operations/forge_graphical_shell_session.md`.

### 9) Desktop shell session actions (bounded shell process only)
- `shell.session_action` — desktop shell `restart_shell` request exposed by the Tauri binary `request_shell_session_action` command.
- Gate function: `shell_session_enabled` requires `FORGE_SHELL_SESSION_ENABLED`, which is set by `forge-shell-session`. Outside that wrapper, the command returns `requested:false` and does not spawn a process.
- Allowlist: only `restart_shell` is accepted. Unknown actions return "shell session action is not allowlisted".
- Scope: when allowed, the command spawns the current Tauri shell executable with the same arguments and exits the current shell process after returning the response. It does not call service-control commands, reboot or shut down the host, rebuild NixOS, mutate modelruntime state, execute gateway tools, or write semantic memory.
- Regression coverage: Rust unit tests cover disabled/enabled shell-session policy and unknown action rejection; desktop AppShell tests cover the Start menu confirmation path.

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
- Default registry guards now verify that every `approval_only` capability evaluates to `needs_approval` before execution.

## Remaining hardening backlog

1. Expand tests for capability-level workspace path boundaries and policy overrides.
2. Keep retired non-capability mutation routes non-executable.
3. Add more service-specific harness tests for configured dependency failures.
4. Add operator UI affordances for the approval-required capability status response.
