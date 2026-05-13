# AI-OS Tool Surface (Phase 5.9)

FORGE treats tools as kernel-governed capabilities, not direct agent calls.

Execution rule:

`ToolRequest -> capability lookup -> risk classification -> policy/charter/budget/approval checks -> adapter execution -> audit/artifact evidence -> ToolResult`

No agent, cell, autonomy worker, or future IRIS path may bypass the gateway.

## Naming Convention

All capabilities use namespaced ids:

- `process.spawn_process`
- `filesystem.read_file`
- `code.run_tests`
- `external.send_email`
- `model.chat`

## Domains and Taxonomy

### `process.*`

- `process.spawn_process`
- `process.kill_process`
- `process.signal_process`
- `process.inspect_process`
- `process.set_resource_limits`
- `process.run_job`
- `process.fork_context`
- `process.checkpoint_process`
- `process.restore_process`

### `filesystem.*`

- `filesystem.read_file`
- `filesystem.write_file`
- `filesystem.delete_file`
- `filesystem.move_file`
- `filesystem.list_dir`
- `filesystem.glob`
- `filesystem.watch_path`
- `filesystem.mount`
- `filesystem.unmount`
- `filesystem.create_snapshot`
- `filesystem.restore_snapshot`
- `filesystem.set_permissions`
- `filesystem.get_permissions`
- `filesystem.query_semantic_fs`
- `filesystem.archive`
- `filesystem.extract`
- `filesystem.sync_to_remote`
- `filesystem.sync_from_remote`

### `network.*`

- `network.dns_resolve`
- `network.dns_register`
- `network.http_request`
- `network.open_socket`
- `network.close_socket`
- `network.proxy_request`
- `network.open_tunnel`
- `network.scan_network`
- `network.intercept_traffic`
- `network.set_firewall_rule`
- `network.delete_firewall_rule`

### `memory.*`

- `memory.remember`
- `memory.recall`
- `memory.forget`
- `memory.embed_content`
- `memory.semantic_search`
- `memory.upsert_fact`
- `memory.retract_fact`
- `memory.summarize_context`
- `memory.cross_reference`
- `memory.rank_relevance`
- `memory.diff_knowledge`

### `device.*`

- `device.list_devices`
- `device.read_sensor`
- `device.write_gpio`
- `device.read_gpio`
- `device.capture_camera`
- `device.stream_camera`
- `device.capture_audio`
- `device.play_audio`
- `device.print_document`
- `device.set_display`
- `device.bluetooth_scan`
- `device.bluetooth_connect`

### `identity.*`

- `identity.get_current_user`
- `identity.switch_user`
- `identity.sudo`
- `identity.issue_token`
- `identity.revoke_token`
- `identity.verify_token`
- `identity.encrypt`
- `identity.decrypt`
- `identity.sign`
- `identity.verify_signature`
- `identity.store_secret`
- `identity.retrieve_secret`
- `identity.audit_log_read`
- `identity.set_policy`
- `identity.check_policy`

### `time.*`

- `time.schedule_once`
- `time.schedule_recurring`
- `time.cancel_schedule`
- `time.set_alarm`
- `time.set_deadline`
- `time.get_system_time`
- `time.set_system_time`
- `time.measure_duration`
- `time.defer_until`

### `agent.*`

- `agent.spawn_agent`
- `agent.kill_agent`
- `agent.send_message`
- `agent.broadcast`
- `agent.request_approval`
- `agent.delegate_task`
- `agent.observe_agent`
- `agent.merge_results`
- `agent.escalate`

### `ui.*`

- `ui.render_ui`
- `ui.show_notification`
- `ui.dismiss_notification`
- `ui.prompt_user`
- `ui.read_clipboard`
- `ui.write_clipboard`
- `ui.screenshot`
- `ui.screen_record`
- `ui.synthesize_speech`
- `ui.transcribe_audio`
- `ui.open_url`
- `ui.navigate`
- `ui.inject_input`

### `code.*`

- `code.run_shell`
- `code.eval_code`
- `code.compile`
- `code.link`
- `code.run_tests`
- `code.parse_test_results`
- `code.lint`
- `code.format`
- `code.diff_code`
- `code.patch_code`
- `code.search_code`
- `code.refactor`

### `observability.*`

- `observability.read_logs`
- `observability.get_metrics`
- `observability.get_traces`
- `observability.create_alert`
- `observability.silence_alert`
- `observability.profile_process`
- `observability.explain_anomaly`
- `observability.tail_stream`

### `config.*`

- `config.get_config`
- `config.set_config`
- `config.watch_config`
- `config.get_env`
- `config.set_env`
- `config.feature_flag_read`
- `config.feature_flag_set`
- `config.migrate_schema`
- `config.backup`
- `config.restore`
- `config.diff_config`

### `external.*`

- `external.call_llm`
- `external.query_database`
- `external.call_api`
- `external.read_email`
- `external.send_email`
- `external.post_message`
- `external.create_issue`
- `external.update_issue`
- `external.read_calendar`
- `external.create_event`
- `external.search_web`

### `model.*` (M3 Runtime Capability Envelope)

Capability envelope ids tracked for policy/taxonomy honesty:

- `model.list`
- `model.inspect`
- `model.load`
- `model.unload`
- `model.chat`
- `model.generate`
- `model.import`
- `model.verify`
- `model.enable`
- `model.disable`
- `model.archive`
- `model.remove_registration`
- `model.backend.list`
- `model.embed`
- `model.delete_file`
- `model.benchmark`

Branch snapshot status on 2026-04-22:

- Model runtime API paths are implemented (`/forge/models*` and gated `/v1/*`) through `modelruntime.Service`.
- Runtime execution is scheduler-governed with FIFO queueing, bounded admission, lifecycle controls, deterministic model/backend selection, and policy hooks.
- Model management flows are implemented for import, verify, enable, disable, archive, and remove-registration.
- Dedicated `model.*` gateway capability registry entries remain a follow-up; this section documents the capability envelope, not gateway alias completion.

## Capability Status

Phase 5.9 registers the current in-code taxonomy with mixed status:

- `active`: mapped to existing gateway tools.
- `approval_only`: implemented but always approval-gated or high-risk.
- `stubbed`: deterministic unsupported path (metadata-only capability).
- `disabled`: present but denied.
- `deprecated`: present but blocked by policy.
- `deferred`: registered but intentionally not executable until roadmap gates release it.

Model-runtime capability honesty:

| Capability | Current branch status | Notes |
|---|---|---|
| `model.list` | real via runtime API | gateway taxonomy aliasing remains pending |
| `model.inspect` | real via runtime API | gateway taxonomy aliasing remains pending |
| `model.load` | real via runtime API | explicit lifecycle path; approval posture can be tightened later |
| `model.unload` | real via runtime API | explicit lifecycle path; approval posture can be tightened later |
| `model.chat` | partial via runtime API | non-streaming and SSE chat paths use FIFO scheduler and policy hooks; gateway taxonomy aliasing remains pending |
| `model.generate` | partial via runtime API | runtime service generation boundary exists; public chat APIs expose the governed inference surface |
| `model.import` | real via runtime API | local GGUF and manifest-backed directory registration only |
| `model.verify` | real via runtime API | checksum/file verification where metadata exists |
| `model.enable` | real via runtime API | re-enables disabled managed models |
| `model.disable` | real via runtime API | blocks inference while preserving metadata |
| `model.archive` | real via runtime API | archives metadata/managed directory without deleting bytes |
| `model.remove_registration` | real via runtime API | removes runtime registration without destructive delete |
| `model.backend.list` | real via runtime API | reports configured runtime backend availability |
| `model.embed` | deferred | out of current runtime scope |
| `model.delete_file` | deferred | approval-required design target; not implemented |
| `model.benchmark` | deferred | deferred |

## Risk Model

- `none` / `low`: read-only introspection.
- `medium`: reversible internal writes.
- `high`: execution/network/privileged operations.
- `critical`: destructive or irreversible operations.

Unknown or ambiguous requests are escalated upward, never silently lowered.

## Evidence Model

Tool results are evidence, not truth. Any semantic memory/state mutation still goes through semantic syscall validation.
