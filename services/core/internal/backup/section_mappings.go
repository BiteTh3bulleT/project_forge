package backup

import (
	"fmt"
	"strings"
)

type insertMap struct {
	sql    string
	fields []string
}

var extractQueries = map[string]string{
	"sources":                       "SELECT * FROM sources ORDER BY id ASC",
	"files":                         "SELECT * FROM files ORDER BY id ASC",
	"chunks":                        "SELECT * FROM chunks ORDER BY id ASC",
	"embedding_records":             "SELECT * FROM embedding_records ORDER BY id ASC",
	"retrieval_runs":                "SELECT * FROM retrieval_runs ORDER BY id ASC",
	"retrieval_results":             "SELECT * FROM retrieval_results ORDER BY id ASC",
	"retrieval_result_selection":    "SELECT * FROM retrieval_result_selection ORDER BY retrieval_result_id ASC",
	"packet_retrieval_runs":         "SELECT * FROM packet_retrieval_runs ORDER BY packet_id ASC, retrieval_run_id ASC",
	"dossiers":                      "SELECT * FROM dossiers",
	"dossier_sources":               "SELECT * FROM dossier_sources ORDER BY dossier_id ASC, source_id ASC",
	"dossier_jobs":                  "SELECT * FROM dossier_jobs ORDER BY dossier_id ASC, job_id ASC",
	"dossier_packets":               "SELECT * FROM dossier_packets ORDER BY dossier_id ASC, packet_id ASC",
	"dossier_briefs":                "SELECT * FROM dossier_briefs ORDER BY id ASC",
	"context_evidence":              "SELECT * FROM context_evidence ORDER BY id ASC",
	"task_packets":                  "SELECT * FROM task_packets",
	"project_context_records":       "SELECT * FROM project_context_records",
	"events":                        "SELECT * FROM events ORDER BY id ASC",
	"jobs":                          "SELECT * FROM jobs ORDER BY created_at DESC",
	"job_status_history":            "SELECT * FROM job_status_history ORDER BY id ASC",
	"job_events":                    "SELECT * FROM job_events ORDER BY id ASC",
	"approval_requests":             "SELECT * FROM approval_requests ORDER BY id ASC",
	"approval_decisions":            "SELECT * FROM approval_decisions ORDER BY id ASC",
	"artifacts":                     "SELECT * FROM artifacts ORDER BY id ASC",
	"execution_strategies":          "SELECT * FROM execution_strategies",
	"approval_presets":              "SELECT * FROM approval_presets",
	"permission_profiles":           "SELECT * FROM permission_profiles",
	"dossier_profiles":              "SELECT * FROM dossier_profiles",
	"automation_rules":              "SELECT * FROM automation_rules",
	"evaluation_records":            "SELECT * FROM evaluation_records",
	"memory_observations":           "SELECT * FROM memory_observations ORDER BY id ASC",
	"memory_observation_links":      "SELECT * FROM memory_observation_links ORDER BY id ASC",
	"retrieval_result_observations": "SELECT * FROM retrieval_result_observations ORDER BY retrieval_result_id ASC, observation_id ASC",
	"memory_usefulness_events":      "SELECT * FROM memory_usefulness_events ORDER BY id ASC",
	"packet_alignment_notes":        "SELECT * FROM packet_alignment_notes ORDER BY id ASC",
	"memory_repair_runs":            "SELECT * FROM memory_repair_runs ORDER BY id ASC",
	"memory_repair_items":           "SELECT * FROM memory_repair_items ORDER BY id ASC",
	"audit_records":                 "SELECT * FROM audit_records ORDER BY id DESC",
	"gateway_invocations":           "SELECT * FROM gateway_invocations ORDER BY id DESC",
	"action_lanes":                  "SELECT * FROM action_lanes",
	"model_manifests":               "SELECT * FROM model_manifests ORDER BY id ASC",
	"model_registry_status":         "SELECT * FROM model_registry_status ORDER BY model_id ASC",
	"model_runtime_loads":           "SELECT * FROM model_runtime_loads ORDER BY id ASC",
	"provenance_records":            "SELECT * FROM provenance_records ORDER BY created_at DESC",
	"journal_events":                "SELECT * FROM journal_events ORDER BY created_at DESC",
	"memory_notes":                  "SELECT * FROM memory_notes ORDER BY updated_at DESC",
	"semantic_links":                "SELECT * FROM semantic_links ORDER BY created_at DESC",
	"state_items":                   "SELECT * FROM state_items ORDER BY updated_at DESC",
	"state_versions":                "SELECT * FROM state_versions ORDER BY id DESC",
	"open_loops":                    "SELECT * FROM open_loops ORDER BY updated_at DESC",
	"artifact_refs":                 "SELECT * FROM artifact_refs ORDER BY created_at DESC",
	"derived_models":                "SELECT * FROM derived_models ORDER BY updated_at DESC",
	"contradiction_records":         "SELECT * FROM contradiction_records ORDER BY created_at DESC",
	"supersession_records":          "SELECT * FROM supersession_records ORDER BY created_at DESC",
	"context_packet_snapshots":      "SELECT * FROM context_packet_snapshots ORDER BY created_at DESC",
	"dream_reports":                 "SELECT * FROM dream_reports ORDER BY created_at DESC",
	"restore_outcome_events":        "SELECT * FROM restore_outcome_events ORDER BY created_at DESC",
	"semantic_idempotency_keys":     "SELECT * FROM semantic_idempotency_keys ORDER BY created_at DESC, idempotency_key ASC",
	"autonomy_settings":             "SELECT key, value FROM settings WHERE key LIKE 'autonomy_repo.%' ORDER BY key ASC",
	"memory_vsa_pointers":           "SELECT * FROM memory_vsa_pointers ORDER BY updated_at DESC",
	"memory_vsa_role_bindings":      "SELECT * FROM memory_vsa_role_bindings ORDER BY updated_at DESC",
	"memory_vsa_associations":       "SELECT * FROM memory_vsa_associations ORDER BY updated_at DESC",
	"retrieval_result_vsa_signals":  "SELECT * FROM retrieval_result_vsa_signals ORDER BY created_at DESC",
	"memory_vsa_reindex_runs":       "SELECT * FROM memory_vsa_reindex_runs ORDER BY id DESC",
	"memory_vsa_reindex_items":      "SELECT * FROM memory_vsa_reindex_items ORDER BY id DESC",
	"chat_threads":                  "SELECT * FROM chat_threads ORDER BY id ASC",
	"chat_messages":                 "SELECT * FROM chat_messages ORDER BY id ASC",
	"canvas_boards":                 "SELECT * FROM canvas_boards ORDER BY id ASC",
	"canvas_notes":                  "SELECT * FROM canvas_notes ORDER BY id ASC",
	"tool_capability_overrides":     "SELECT * FROM tool_capability_overrides ORDER BY capability_id ASC",
	"feature_flags":                 "SELECT * FROM feature_flags ORDER BY key ASC",
	"alert_rules":                   "SELECT * FROM alert_rules ORDER BY id ASC",
	"scheduled_tasks":               "SELECT * FROM scheduled_tasks ORDER BY id ASC",
}

// Section upserts used during restore.
var insertStatements = func() map[string]insertMap {
	m := map[string]insertMap{
		"dossiers": {
			sql: `INSERT INTO dossiers(
  id, created_at, updated_at, name, description, primary_paths_json, related_repos_json,
  constraints_json, preferred_adapters_json, important_files_json, routing_notes
) VALUES(?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  created_at=excluded.created_at,
  updated_at=excluded.updated_at,
  name=excluded.name,
  description=excluded.description,
  primary_paths_json=excluded.primary_paths_json,
  related_repos_json=excluded.related_repos_json,
  constraints_json=excluded.constraints_json,
  preferred_adapters_json=excluded.preferred_adapters_json,
  important_files_json=excluded.important_files_json,
  routing_notes=excluded.routing_notes`,
			fields: []string{
				"id", "created_at", "updated_at", "name", "description", "primary_paths_json", "related_repos_json",
				"constraints_json", "preferred_adapters_json", "important_files_json", "routing_notes",
			},
		},
		"task_packets": {
			sql: `INSERT INTO task_packets(
  id, packet_version, created_at, generated_at, title, user_request, objective,
  adapter_target, execution_mode, risk_class, expected_output_json, constraints_json,
  instructions, selected_paths_json, scope_snapshot_json, source_references_json,
  retrieved_context_json, project_notes, source_context_record_ids_json, request_payload_json
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  packet_version=excluded.packet_version,
  created_at=excluded.created_at,
  generated_at=excluded.generated_at,
  title=excluded.title,
  user_request=excluded.user_request,
  objective=excluded.objective,
  adapter_target=excluded.adapter_target,
  execution_mode=excluded.execution_mode,
  risk_class=excluded.risk_class,
  expected_output_json=excluded.expected_output_json,
  constraints_json=excluded.constraints_json,
  instructions=excluded.instructions,
  selected_paths_json=excluded.selected_paths_json,
  scope_snapshot_json=excluded.scope_snapshot_json,
  source_references_json=excluded.source_references_json,
  retrieved_context_json=excluded.retrieved_context_json,
  project_notes=excluded.project_notes,
  source_context_record_ids_json=excluded.source_context_record_ids_json,
  request_payload_json=excluded.request_payload_json`,
			fields: []string{
				"id", "packet_version", "created_at", "generated_at", "title", "user_request", "objective",
				"adapter_target", "execution_mode", "risk_class", "expected_output_json", "constraints_json",
				"instructions", "selected_paths_json", "scope_snapshot_json", "source_references_json",
				"retrieved_context_json", "project_notes", "source_context_record_ids_json", "request_payload_json",
			},
		},
		"project_context_records": {
			sql: `INSERT INTO project_context_records(
  id, context_version, created_at, generated_at, source_path, source_hash, source_size_bytes,
  normalized_summary_json, briefing_markdown, agents_markdown, claude_markdown, cursor_markdown,
  generated_agents_path, generated_claude_path, generated_briefing_path, generated_cursor_path, notes
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  context_version=excluded.context_version,
  created_at=excluded.created_at,
  generated_at=excluded.generated_at,
  source_path=excluded.source_path,
  source_hash=excluded.source_hash,
  source_size_bytes=excluded.source_size_bytes,
  normalized_summary_json=excluded.normalized_summary_json,
  briefing_markdown=excluded.briefing_markdown,
  agents_markdown=excluded.agents_markdown,
  claude_markdown=excluded.claude_markdown,
  cursor_markdown=excluded.cursor_markdown,
  generated_agents_path=excluded.generated_agents_path,
  generated_claude_path=excluded.generated_claude_path,
  generated_briefing_path=excluded.generated_briefing_path,
  generated_cursor_path=excluded.generated_cursor_path,
  notes=excluded.notes`,
			fields: []string{
				"id", "context_version", "created_at", "generated_at", "source_path", "source_hash", "source_size_bytes",
				"normalized_summary_json", "briefing_markdown", "agents_markdown", "claude_markdown", "cursor_markdown",
				"generated_agents_path", "generated_claude_path", "generated_briefing_path", "generated_cursor_path", "notes",
			},
		},
		"events": {
			sql: `INSERT INTO events(id, created_at, type, payload_json)
VALUES(?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  created_at=excluded.created_at,
  type=excluded.type,
  payload_json=excluded.payload_json`,
			fields: []string{"id", "created_at", "type", "payload_json"},
		},
		"jobs": {
			sql: `INSERT INTO jobs(
  id, created_at, updated_at, queued_at, started_at, completed_at, title,
  requested_action, target_adapter, initiating_source, execution_boundary,
  risk_class, status, approval_status, write_intent, cancel_requested,
  task_packet_id, result_summary, failure_info, last_failure_code, last_error, metadata_json
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  created_at=excluded.created_at,
  updated_at=excluded.updated_at,
  queued_at=excluded.queued_at,
  started_at=excluded.started_at,
  completed_at=excluded.completed_at,
  title=excluded.title,
  requested_action=excluded.requested_action,
  target_adapter=excluded.target_adapter,
  initiating_source=excluded.initiating_source,
  execution_boundary=excluded.execution_boundary,
  risk_class=excluded.risk_class,
  status=excluded.status,
  approval_status=excluded.approval_status,
  write_intent=excluded.write_intent,
  cancel_requested=excluded.cancel_requested,
  task_packet_id=excluded.task_packet_id,
  result_summary=excluded.result_summary,
  failure_info=excluded.failure_info,
  last_failure_code=excluded.last_failure_code,
  last_error=excluded.last_error,
  metadata_json=excluded.metadata_json`,
			fields: []string{
				"id", "created_at", "updated_at", "queued_at", "started_at", "completed_at", "title",
				"requested_action", "target_adapter", "initiating_source", "execution_boundary",
				"risk_class", "status", "approval_status", "write_intent", "cancel_requested",
				"task_packet_id", "result_summary", "failure_info", "last_failure_code", "last_error", "metadata_json",
			},
		},
		"job_status_history": {
			sql: `INSERT INTO job_status_history(id, job_id, created_at, from_status, to_status, reason)
VALUES(?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  job_id=excluded.job_id,
  created_at=excluded.created_at,
  from_status=excluded.from_status,
  to_status=excluded.to_status,
  reason=excluded.reason`,
			fields: []string{"id", "job_id", "created_at", "from_status", "to_status", "reason"},
		},
		"job_events": {
			sql: `INSERT INTO job_events(id, job_id, created_at, type, message, payload_json)
VALUES(?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  job_id=excluded.job_id,
  created_at=excluded.created_at,
  type=excluded.type,
  message=excluded.message,
  payload_json=excluded.payload_json`,
			fields: []string{"id", "job_id", "created_at", "type", "message", "payload_json"},
		},
		"approval_requests": {
			sql: `INSERT INTO approval_requests(
	id, job_id, created_at, status, requested_action, risk_class, requested_adapter,
  write_intent, scope_snapshot_json, task_packet_id, request_summary, expires_at, expired_at
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  job_id=excluded.job_id,
  created_at=excluded.created_at,
  status=excluded.status,
  requested_action=excluded.requested_action,
  risk_class=excluded.risk_class,
  requested_adapter=excluded.requested_adapter,
  write_intent=excluded.write_intent,
  scope_snapshot_json=excluded.scope_snapshot_json,
  task_packet_id=excluded.task_packet_id,
  request_summary=excluded.request_summary,
  expires_at=excluded.expires_at,
  expired_at=excluded.expired_at`,
			fields: []string{
				"id", "job_id", "created_at", "status", "requested_action", "risk_class",
				"requested_adapter", "write_intent", "scope_snapshot_json", "task_packet_id", "request_summary",
				"expires_at", "expired_at",
			},
		},
		"approval_decisions": {
			sql: `INSERT INTO approval_decisions(id, request_id, created_at, actor, decision, note)
VALUES(?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  request_id=excluded.request_id,
  created_at=excluded.created_at,
  actor=excluded.actor,
  decision=excluded.decision,
  note=excluded.note`,
			fields: []string{"id", "request_id", "created_at", "actor", "decision", "note"},
		},
		"artifacts": {
			sql: `INSERT INTO artifacts(
  id, created_at, job_id, packet_id, type, title, file_path, mime_type, metadata_json
) VALUES(?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  created_at=excluded.created_at,
  job_id=excluded.job_id,
  packet_id=excluded.packet_id,
  type=excluded.type,
  title=excluded.title,
  file_path=excluded.file_path,
  mime_type=excluded.mime_type,
  metadata_json=excluded.metadata_json`,
			fields: []string{"id", "created_at", "job_id", "packet_id", "type", "title", "file_path", "mime_type", "metadata_json"},
		},
		"evaluation_records": {
			sql: `INSERT INTO evaluation_records(
  id, created_at, job_id, dossier_id, success, quality_rating, usefulness_rating, correctness_confidence,
  packet_quality_rating, adapter_suitability, retry_recommended, influence_routing, notes, scorer
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  created_at=excluded.created_at,
  job_id=excluded.job_id,
  dossier_id=excluded.dossier_id,
  success=excluded.success,
  quality_rating=excluded.quality_rating,
  usefulness_rating=excluded.usefulness_rating,
  correctness_confidence=excluded.correctness_confidence,
  packet_quality_rating=excluded.packet_quality_rating,
  adapter_suitability=excluded.adapter_suitability,
  retry_recommended=excluded.retry_recommended,
  influence_routing=excluded.influence_routing,
  notes=excluded.notes,
  scorer=excluded.scorer`,
			fields: []string{
				"id", "created_at", "job_id", "dossier_id", "success", "quality_rating", "usefulness_rating", "correctness_confidence",
				"packet_quality_rating", "adapter_suitability", "retry_recommended", "influence_routing", "notes", "scorer",
			},
		},
		"autonomy_settings": {
			sql: `INSERT INTO settings(key, value)
VALUES(?, ?)
ON CONFLICT(key) DO UPDATE SET value=excluded.value`,
			fields: []string{"key", "value"},
		},
		"permission_profiles": {
			sql: `INSERT INTO permission_profiles(
  id, created_at, updated_at, name, description,
  allowed_read_paths_json, allowed_write_paths_json, allowed_execute_paths_json,
  forbidden_paths_json, allowed_tools_json, approval_required_risks_json,
  max_bytes_per_write, allow_network, editable, active
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  updated_at=excluded.updated_at,
  name=excluded.name,
  description=excluded.description,
  allowed_read_paths_json=excluded.allowed_read_paths_json,
  allowed_write_paths_json=excluded.allowed_write_paths_json,
  allowed_execute_paths_json=excluded.allowed_execute_paths_json,
  forbidden_paths_json=excluded.forbidden_paths_json,
  allowed_tools_json=excluded.allowed_tools_json,
  approval_required_risks_json=excluded.approval_required_risks_json,
  max_bytes_per_write=excluded.max_bytes_per_write,
  allow_network=excluded.allow_network,
  editable=excluded.editable,
  active=excluded.active`,
			fields: []string{
				"id", "created_at", "updated_at", "name", "description",
				"allowed_read_paths_json", "allowed_write_paths_json", "allowed_execute_paths_json",
				"forbidden_paths_json", "allowed_tools_json", "approval_required_risks_json",
				"max_bytes_per_write", "allow_network", "editable", "active",
			},
		},
		"approval_presets": {
			sql: `INSERT INTO approval_presets(id, created_at, updated_at, name, description, profile_json, editable)
VALUES(?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  updated_at=excluded.updated_at,
  name=excluded.name,
  description=excluded.description,
  profile_json=excluded.profile_json,
  editable=excluded.editable`,
			fields: []string{"id", "created_at", "updated_at", "name", "description", "profile_json", "editable"},
		},
		"dossier_profiles": {
			sql: `INSERT INTO dossier_profiles(
  dossier_id, updated_at, preferred_strategies_json, preferred_adapters_json,
  approval_preset_id, retrieval_defaults_json, high_value_files_json,
  noisy_files_json, routing_notes, automation_bindings_json
) VALUES(?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(dossier_id) DO UPDATE SET
  updated_at=excluded.updated_at,
  preferred_strategies_json=excluded.preferred_strategies_json,
  preferred_adapters_json=excluded.preferred_adapters_json,
  approval_preset_id=excluded.approval_preset_id,
  retrieval_defaults_json=excluded.retrieval_defaults_json,
  high_value_files_json=excluded.high_value_files_json,
  noisy_files_json=excluded.noisy_files_json,
  routing_notes=excluded.routing_notes,
  automation_bindings_json=excluded.automation_bindings_json`,
			fields: []string{
				"dossier_id", "updated_at", "preferred_strategies_json", "preferred_adapters_json",
				"approval_preset_id", "retrieval_defaults_json", "high_value_files_json",
				"noisy_files_json", "routing_notes", "automation_bindings_json",
			},
		},
		"execution_strategies": {
			sql: `INSERT INTO execution_strategies(
  id, created_at, updated_at, name, task_type, target_adapter, retrieval_mode,
  packet_rules_json, approval_required, approval_preset_id, expected_artifacts_json,
  success_criteria_json, retry_guidance_json, enabled
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  updated_at=excluded.updated_at,
  name=excluded.name,
  task_type=excluded.task_type,
  target_adapter=excluded.target_adapter,
  retrieval_mode=excluded.retrieval_mode,
  packet_rules_json=excluded.packet_rules_json,
  approval_required=excluded.approval_required,
  approval_preset_id=excluded.approval_preset_id,
  expected_artifacts_json=excluded.expected_artifacts_json,
  success_criteria_json=excluded.success_criteria_json,
  retry_guidance_json=excluded.retry_guidance_json,
  enabled=excluded.enabled`,
			fields: []string{
				"id", "created_at", "updated_at", "name", "task_type", "target_adapter", "retrieval_mode",
				"packet_rules_json", "approval_required", "approval_preset_id", "expected_artifacts_json",
				"success_criteria_json", "retry_guidance_json", "enabled",
			},
		},
		"automation_rules": {
			sql: `INSERT INTO automation_rules(
  id, created_at, updated_at, name, trigger, condition_json, action_json, scope_json, enabled, dry_run_default
) VALUES(?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  created_at=excluded.created_at,
  updated_at=excluded.updated_at,
  name=excluded.name,
  trigger=excluded.trigger,
  condition_json=excluded.condition_json,
  action_json=excluded.action_json,
  scope_json=excluded.scope_json,
  enabled=excluded.enabled,
  dry_run_default=excluded.dry_run_default`,
			fields: []string{
				"id", "created_at", "updated_at", "name", "trigger", "condition_json", "action_json", "scope_json", "enabled", "dry_run_default",
			},
		},
		"action_lanes": {
			sql: `INSERT INTO action_lanes(
  id, created_at, updated_at, name, description, action_type, allowed_paths_json,
  forbidden_paths_json, write_intent, requires_approval, risk_class, max_bytes,
  expected_artifacts_json, builtin, enabled
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  created_at=excluded.created_at,
  updated_at=excluded.updated_at,
  name=excluded.name,
  description=excluded.description,
  action_type=excluded.action_type,
  allowed_paths_json=excluded.allowed_paths_json,
  forbidden_paths_json=excluded.forbidden_paths_json,
  write_intent=excluded.write_intent,
  requires_approval=excluded.requires_approval,
  risk_class=excluded.risk_class,
  max_bytes=excluded.max_bytes,
  expected_artifacts_json=excluded.expected_artifacts_json,
  builtin=excluded.builtin,
  enabled=excluded.enabled`,
			fields: []string{
				"id", "created_at", "updated_at", "name", "description", "action_type", "allowed_paths_json",
				"forbidden_paths_json", "write_intent", "requires_approval", "risk_class", "max_bytes",
				"expected_artifacts_json", "builtin", "enabled",
			},
		},
		"provenance_records": {
			sql: `INSERT INTO provenance_records(
  id, actor, actor_type, source, trace_id, workspace_id, lane_id, selected_paths_json,
  metadata_json, created_at, proposed_by, committed_by, syscall_id, correlation_id, audit_id
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  actor=excluded.actor,
  actor_type=excluded.actor_type,
  source=excluded.source,
  trace_id=excluded.trace_id,
  workspace_id=excluded.workspace_id,
  lane_id=excluded.lane_id,
  selected_paths_json=excluded.selected_paths_json,
  metadata_json=excluded.metadata_json,
  created_at=excluded.created_at,
  proposed_by=excluded.proposed_by,
  committed_by=excluded.committed_by,
  syscall_id=excluded.syscall_id,
  correlation_id=excluded.correlation_id,
  audit_id=excluded.audit_id`,
			fields: []string{
				"id", "actor", "actor_type", "source", "trace_id", "workspace_id", "lane_id", "selected_paths_json",
				"metadata_json", "created_at", "proposed_by", "committed_by", "syscall_id", "correlation_id", "audit_id",
			},
		},
		"gateway_invocations": {
			sql: `INSERT INTO gateway_invocations(
  id, correlation_id, created_at, completed_at, tool_id, lane_id, job_id, packet_id,
  approval_request_id, initiator, action, risk_class, write_intent, scope_json, input_json,
  status, denied_reason, result_json, artifacts_json, permission_profile_id
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  correlation_id=excluded.correlation_id,
  created_at=excluded.created_at,
  completed_at=excluded.completed_at,
  tool_id=excluded.tool_id,
  lane_id=excluded.lane_id,
  job_id=excluded.job_id,
  packet_id=excluded.packet_id,
  approval_request_id=excluded.approval_request_id,
  initiator=excluded.initiator,
  action=excluded.action,
  risk_class=excluded.risk_class,
  write_intent=excluded.write_intent,
  scope_json=excluded.scope_json,
  input_json=excluded.input_json,
  status=excluded.status,
  denied_reason=excluded.denied_reason,
  result_json=excluded.result_json,
  artifacts_json=excluded.artifacts_json,
  permission_profile_id=excluded.permission_profile_id`,
			fields: []string{
				"id", "correlation_id", "created_at", "completed_at", "tool_id", "lane_id", "job_id", "packet_id",
				"approval_request_id", "initiator", "action", "risk_class", "write_intent", "scope_json", "input_json",
				"status", "denied_reason", "result_json", "artifacts_json", "permission_profile_id",
			},
		},
		"audit_records": {
			sql: `INSERT INTO audit_records(
  id, created_at, correlation_id, category, action, actor, subject_type, subject_id,
  job_id, gateway_invocation_id, approval_request_id, risk_class, outcome, summary, payload_json
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO NOTHING`,
			fields: []string{
				"id", "created_at", "correlation_id", "category", "action", "actor", "subject_type", "subject_id",
				"job_id", "gateway_invocation_id", "approval_request_id", "risk_class", "outcome", "summary", "payload_json",
			},
		},
		"journal_events": {
			sql: `INSERT INTO journal_events(
  id, type, source, actor, workspace_id, lane_id, selected_paths_json, payload_json,
  correlation_id, trace_id, provenance_id, provenance_json, created_at, metadata_json,
  proposed_by, committed_by, syscall_id, audit_id
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO NOTHING`,
			fields: []string{
				"id", "type", "source", "actor", "workspace_id", "lane_id", "selected_paths_json", "payload_json",
				"correlation_id", "trace_id", "provenance_id", "provenance_json", "created_at", "metadata_json",
				"proposed_by", "committed_by", "syscall_id", "audit_id",
			},
		},
		"memory_notes": {
			sql: `INSERT INTO memory_notes(
  id, type, title, content, workspace_id, lane_id, selected_paths_json, confidence, status,
  provenance_id, provenance_json, created_at, updated_at, archived_at, superseded_by, metadata_json,
  proposed_by, committed_by, syscall_id, correlation_id, trace_id, audit_id
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  type=excluded.type,
  title=excluded.title,
  content=excluded.content,
  workspace_id=excluded.workspace_id,
  lane_id=excluded.lane_id,
  selected_paths_json=excluded.selected_paths_json,
  confidence=excluded.confidence,
  status=excluded.status,
  provenance_id=excluded.provenance_id,
  provenance_json=excluded.provenance_json,
  created_at=excluded.created_at,
  updated_at=excluded.updated_at,
  archived_at=excluded.archived_at,
  superseded_by=excluded.superseded_by,
  metadata_json=excluded.metadata_json,
  proposed_by=excluded.proposed_by,
  committed_by=excluded.committed_by,
  syscall_id=excluded.syscall_id,
  correlation_id=excluded.correlation_id,
  trace_id=excluded.trace_id,
  audit_id=excluded.audit_id`,
			fields: []string{
				"id", "type", "title", "content", "workspace_id", "lane_id", "selected_paths_json", "confidence", "status",
				"provenance_id", "provenance_json", "created_at", "updated_at", "archived_at", "superseded_by", "metadata_json",
				"proposed_by", "committed_by", "syscall_id", "correlation_id", "trace_id", "audit_id",
			},
		},
		"semantic_links": {
			sql: `INSERT INTO semantic_links(
  id, type, source_id, source_kind, target_id, target_kind, confidence, provenance_id,
  provenance_json, workspace_id, lane_id, selected_paths_json, created_at, metadata_json,
  proposed_by, committed_by, syscall_id, correlation_id, trace_id, audit_id
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  type=excluded.type,
  source_id=excluded.source_id,
  source_kind=excluded.source_kind,
  target_id=excluded.target_id,
  target_kind=excluded.target_kind,
  confidence=excluded.confidence,
  provenance_id=excluded.provenance_id,
  provenance_json=excluded.provenance_json,
  workspace_id=excluded.workspace_id,
  lane_id=excluded.lane_id,
  selected_paths_json=excluded.selected_paths_json,
  created_at=excluded.created_at,
  metadata_json=excluded.metadata_json,
  proposed_by=excluded.proposed_by,
  committed_by=excluded.committed_by,
  syscall_id=excluded.syscall_id,
  correlation_id=excluded.correlation_id,
  trace_id=excluded.trace_id,
  audit_id=excluded.audit_id`,
			fields: []string{
				"id", "type", "source_id", "source_kind", "target_id", "target_kind", "confidence", "provenance_id",
				"provenance_json", "workspace_id", "lane_id", "selected_paths_json", "created_at", "metadata_json",
				"proposed_by", "committed_by", "syscall_id", "correlation_id", "trace_id", "audit_id",
			},
		},
		"state_items": {
			sql: `INSERT INTO state_items(
  id, key, value_json, workspace_id, lane_id, selected_paths_json, status,
  derived_from_json, current_version, updated_at, metadata_json, proposed_by,
  committed_by, syscall_id, correlation_id, trace_id, audit_id
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  key=excluded.key,
  value_json=excluded.value_json,
  workspace_id=excluded.workspace_id,
  lane_id=excluded.lane_id,
  selected_paths_json=excluded.selected_paths_json,
  status=excluded.status,
  derived_from_json=excluded.derived_from_json,
  current_version=excluded.current_version,
  updated_at=excluded.updated_at,
  metadata_json=excluded.metadata_json,
  proposed_by=excluded.proposed_by,
  committed_by=excluded.committed_by,
  syscall_id=excluded.syscall_id,
  correlation_id=excluded.correlation_id,
  trace_id=excluded.trace_id,
  audit_id=excluded.audit_id`,
			fields: []string{
				"id", "key", "value_json", "workspace_id", "lane_id", "selected_paths_json", "status",
				"derived_from_json", "current_version", "updated_at", "metadata_json", "proposed_by",
				"committed_by", "syscall_id", "correlation_id", "trace_id", "audit_id",
			},
		},
		"state_versions": {
			sql: `INSERT INTO state_versions(
  id, state_item_id, state_key, workspace_id, lane_id, previous_value_json, new_value_json,
  changed_by, derived_from_json, syscall_id, audit_id, correlation_id, trace_id, created_at,
  metadata_json, proposed_by, committed_by
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  state_item_id=excluded.state_item_id,
  state_key=excluded.state_key,
  workspace_id=excluded.workspace_id,
  lane_id=excluded.lane_id,
  previous_value_json=excluded.previous_value_json,
  new_value_json=excluded.new_value_json,
  changed_by=excluded.changed_by,
  derived_from_json=excluded.derived_from_json,
  syscall_id=excluded.syscall_id,
  audit_id=excluded.audit_id,
  correlation_id=excluded.correlation_id,
  trace_id=excluded.trace_id,
  created_at=excluded.created_at,
  metadata_json=excluded.metadata_json,
  proposed_by=excluded.proposed_by,
  committed_by=excluded.committed_by`,
			fields: []string{
				"id", "state_item_id", "state_key", "workspace_id", "lane_id", "previous_value_json", "new_value_json",
				"changed_by", "derived_from_json", "syscall_id", "audit_id", "correlation_id", "trace_id", "created_at",
				"metadata_json", "proposed_by", "committed_by",
			},
		},
		"open_loops": {
			sql: `INSERT INTO open_loops(
  id, title, state, priority, owner, blocker, next_action, related_notes_json, created_from,
  workspace_id, lane_id, selected_paths_json, created_at, updated_at, resolved_at, archived_at,
  metadata_json, proposed_by, committed_by, syscall_id, correlation_id, trace_id, audit_id
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  title=excluded.title,
  state=excluded.state,
  priority=excluded.priority,
  owner=excluded.owner,
  blocker=excluded.blocker,
  next_action=excluded.next_action,
  related_notes_json=excluded.related_notes_json,
  created_from=excluded.created_from,
  workspace_id=excluded.workspace_id,
  lane_id=excluded.lane_id,
  selected_paths_json=excluded.selected_paths_json,
  created_at=excluded.created_at,
  updated_at=excluded.updated_at,
  resolved_at=excluded.resolved_at,
  archived_at=excluded.archived_at,
  metadata_json=excluded.metadata_json,
  proposed_by=excluded.proposed_by,
  committed_by=excluded.committed_by,
  syscall_id=excluded.syscall_id,
  correlation_id=excluded.correlation_id,
  trace_id=excluded.trace_id,
  audit_id=excluded.audit_id`,
			fields: []string{
				"id", "title", "state", "priority", "owner", "blocker", "next_action", "related_notes_json", "created_from",
				"workspace_id", "lane_id", "selected_paths_json", "created_at", "updated_at", "resolved_at", "archived_at",
				"metadata_json", "proposed_by", "committed_by", "syscall_id", "correlation_id", "trace_id", "audit_id",
			},
		},
		"artifact_refs": {
			sql: `INSERT INTO artifact_refs(
  id, type, uri, content_hash, workspace_id, lane_id, selected_paths_json, provenance_id,
  provenance_json, created_at, metadata_json, proposed_by, committed_by, syscall_id,
  correlation_id, trace_id, audit_id
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  type=excluded.type,
  uri=excluded.uri,
  content_hash=excluded.content_hash,
  workspace_id=excluded.workspace_id,
  lane_id=excluded.lane_id,
  selected_paths_json=excluded.selected_paths_json,
  provenance_id=excluded.provenance_id,
  provenance_json=excluded.provenance_json,
  created_at=excluded.created_at,
  metadata_json=excluded.metadata_json,
  proposed_by=excluded.proposed_by,
  committed_by=excluded.committed_by,
  syscall_id=excluded.syscall_id,
  correlation_id=excluded.correlation_id,
  trace_id=excluded.trace_id,
  audit_id=excluded.audit_id`,
			fields: []string{
				"id", "type", "uri", "content_hash", "workspace_id", "lane_id", "selected_paths_json", "provenance_id",
				"provenance_json", "created_at", "metadata_json", "proposed_by", "committed_by", "syscall_id",
				"correlation_id", "trace_id", "audit_id",
			},
		},
		"derived_models": {
			sql: `INSERT INTO derived_models(
  id, type, expression_json, derived_from_json, support_count, confidence, status, workspace_id,
  lane_id, selected_paths_json, last_validated_at, created_at, updated_at, metadata_json,
  proposed_by, committed_by, syscall_id, correlation_id, trace_id, audit_id
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  type=excluded.type,
  expression_json=excluded.expression_json,
  derived_from_json=excluded.derived_from_json,
  support_count=excluded.support_count,
  confidence=excluded.confidence,
  status=excluded.status,
  workspace_id=excluded.workspace_id,
  lane_id=excluded.lane_id,
  selected_paths_json=excluded.selected_paths_json,
  last_validated_at=excluded.last_validated_at,
  created_at=excluded.created_at,
  updated_at=excluded.updated_at,
  metadata_json=excluded.metadata_json,
  proposed_by=excluded.proposed_by,
  committed_by=excluded.committed_by,
  syscall_id=excluded.syscall_id,
  correlation_id=excluded.correlation_id,
  trace_id=excluded.trace_id,
  audit_id=excluded.audit_id`,
			fields: []string{
				"id", "type", "expression_json", "derived_from_json", "support_count", "confidence", "status", "workspace_id",
				"lane_id", "selected_paths_json", "last_validated_at", "created_at", "updated_at", "metadata_json",
				"proposed_by", "committed_by", "syscall_id", "correlation_id", "trace_id", "audit_id",
			},
		},
		"contradiction_records": {
			sql: `INSERT INTO contradiction_records(
  id, left_object_id, left_object_kind, right_object_id, right_object_kind, reason, severity,
  confidence, provenance_id, provenance_json, workspace_id, lane_id, created_at, metadata_json,
  proposed_by, committed_by, syscall_id, correlation_id, trace_id, audit_id
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  left_object_id=excluded.left_object_id,
  left_object_kind=excluded.left_object_kind,
  right_object_id=excluded.right_object_id,
  right_object_kind=excluded.right_object_kind,
  reason=excluded.reason,
  severity=excluded.severity,
  confidence=excluded.confidence,
  provenance_id=excluded.provenance_id,
  provenance_json=excluded.provenance_json,
  workspace_id=excluded.workspace_id,
  lane_id=excluded.lane_id,
  created_at=excluded.created_at,
  metadata_json=excluded.metadata_json,
  proposed_by=excluded.proposed_by,
  committed_by=excluded.committed_by,
  syscall_id=excluded.syscall_id,
  correlation_id=excluded.correlation_id,
  trace_id=excluded.trace_id,
  audit_id=excluded.audit_id`,
			fields: []string{
				"id", "left_object_id", "left_object_kind", "right_object_id", "right_object_kind", "reason", "severity",
				"confidence", "provenance_id", "provenance_json", "workspace_id", "lane_id", "created_at", "metadata_json",
				"proposed_by", "committed_by", "syscall_id", "correlation_id", "trace_id", "audit_id",
			},
		},
		"supersession_records": {
			sql: `INSERT INTO supersession_records(
  id, old_object_id, old_object_kind, new_object_id, new_object_kind, reason, provenance_id,
  provenance_json, workspace_id, lane_id, created_at, metadata_json, proposed_by, committed_by,
  syscall_id, correlation_id, trace_id, audit_id
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  old_object_id=excluded.old_object_id,
  old_object_kind=excluded.old_object_kind,
  new_object_id=excluded.new_object_id,
  new_object_kind=excluded.new_object_kind,
  reason=excluded.reason,
  provenance_id=excluded.provenance_id,
  provenance_json=excluded.provenance_json,
  workspace_id=excluded.workspace_id,
  lane_id=excluded.lane_id,
  created_at=excluded.created_at,
  metadata_json=excluded.metadata_json,
  proposed_by=excluded.proposed_by,
  committed_by=excluded.committed_by,
  syscall_id=excluded.syscall_id,
  correlation_id=excluded.correlation_id,
  trace_id=excluded.trace_id,
  audit_id=excluded.audit_id`,
			fields: []string{
				"id", "old_object_id", "old_object_kind", "new_object_id", "new_object_kind", "reason", "provenance_id",
				"provenance_json", "workspace_id", "lane_id", "created_at", "metadata_json", "proposed_by", "committed_by",
				"syscall_id", "correlation_id", "trace_id", "audit_id",
			},
		},
		"context_packet_snapshots": {
			sql: `INSERT INTO context_packet_snapshots(
  id, query, workspace_id, lane_id, snapshot_kind, snapshot_fingerprint, parent_snapshot_id,
  selected_paths_json, included_state_json, included_open_loops_json, included_notes_json, included_links_json,
  included_models_json, included_artifacts_json, included_events_json, header_json, graph_json, delta_json,
  restore_scores_json, render_artifact_ref_id, resume_hints_json, budget_json, inclusion_reasons_json, created_at,
  correlation_id, trace_id, syscall_id, metadata_json, proposed_by, committed_by, audit_id
) VALUES(?,?,?,?,COALESCE(?,''),COALESCE(?,''),COALESCE(?,''),?,?,?,?,?,?,?,?,COALESCE(?, '{}'),COALESCE(?, '{}'),COALESCE(?, '{}'),COALESCE(?, '{}'),COALESCE(?,''),COALESCE(?, '{}'),?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  query=excluded.query,
  workspace_id=excluded.workspace_id,
  lane_id=excluded.lane_id,
  snapshot_kind=excluded.snapshot_kind,
  snapshot_fingerprint=excluded.snapshot_fingerprint,
  parent_snapshot_id=excluded.parent_snapshot_id,
  selected_paths_json=excluded.selected_paths_json,
  included_state_json=excluded.included_state_json,
  included_open_loops_json=excluded.included_open_loops_json,
  included_notes_json=excluded.included_notes_json,
  included_links_json=excluded.included_links_json,
  included_models_json=excluded.included_models_json,
  included_artifacts_json=excluded.included_artifacts_json,
  included_events_json=excluded.included_events_json,
  header_json=excluded.header_json,
  graph_json=excluded.graph_json,
  delta_json=excluded.delta_json,
  restore_scores_json=excluded.restore_scores_json,
  render_artifact_ref_id=excluded.render_artifact_ref_id,
  resume_hints_json=excluded.resume_hints_json,
  budget_json=excluded.budget_json,
  inclusion_reasons_json=excluded.inclusion_reasons_json,
  created_at=excluded.created_at,
  correlation_id=excluded.correlation_id,
  trace_id=excluded.trace_id,
  syscall_id=excluded.syscall_id,
  metadata_json=excluded.metadata_json,
  proposed_by=excluded.proposed_by,
  committed_by=excluded.committed_by,
  audit_id=excluded.audit_id`,
			fields: []string{
				"id", "query", "workspace_id", "lane_id", "snapshot_kind", "snapshot_fingerprint", "parent_snapshot_id",
				"selected_paths_json", "included_state_json", "included_open_loops_json", "included_notes_json", "included_links_json",
				"included_models_json", "included_artifacts_json", "included_events_json", "header_json", "graph_json", "delta_json",
				"restore_scores_json", "render_artifact_ref_id", "resume_hints_json", "budget_json", "inclusion_reasons_json", "created_at",
				"correlation_id", "trace_id", "syscall_id", "metadata_json", "proposed_by", "committed_by", "audit_id",
			},
		},
		"dream_reports": {
			sql: `INSERT INTO dream_reports(
  id, created_at, completed_at, workspace_id, lane_id, mode, dry_run, status,
  time_window_start, time_window_end, candidates_considered, proposals_generated,
  summary_json, candidates_json, salience_scores_json, memory_tier_proposals_json,
  repair_proposals_json, snapshot_hygiene_proposals_json, warnings_json, trace_json,
  correlation_id, trace_id, syscall_id, audit_id, proposed_by, committed_by, metadata_json
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  created_at=excluded.created_at,
  completed_at=excluded.completed_at,
  workspace_id=excluded.workspace_id,
  lane_id=excluded.lane_id,
  mode=excluded.mode,
  dry_run=excluded.dry_run,
  status=excluded.status,
  time_window_start=excluded.time_window_start,
  time_window_end=excluded.time_window_end,
  candidates_considered=excluded.candidates_considered,
  proposals_generated=excluded.proposals_generated,
  summary_json=excluded.summary_json,
  candidates_json=excluded.candidates_json,
  salience_scores_json=excluded.salience_scores_json,
  memory_tier_proposals_json=excluded.memory_tier_proposals_json,
  repair_proposals_json=excluded.repair_proposals_json,
  snapshot_hygiene_proposals_json=excluded.snapshot_hygiene_proposals_json,
  warnings_json=excluded.warnings_json,
  trace_json=excluded.trace_json,
  correlation_id=excluded.correlation_id,
  trace_id=excluded.trace_id,
  syscall_id=excluded.syscall_id,
  audit_id=excluded.audit_id,
  proposed_by=excluded.proposed_by,
  committed_by=excluded.committed_by,
  metadata_json=excluded.metadata_json`,
			fields: []string{
				"id", "created_at", "completed_at", "workspace_id", "lane_id", "mode", "dry_run", "status",
				"time_window_start", "time_window_end", "candidates_considered", "proposals_generated",
				"summary_json", "candidates_json", "salience_scores_json", "memory_tier_proposals_json",
				"repair_proposals_json", "snapshot_hygiene_proposals_json", "warnings_json", "trace_json",
				"correlation_id", "trace_id", "syscall_id", "audit_id", "proposed_by", "committed_by", "metadata_json",
			},
		},
		"restore_outcome_events": {
			sql: `INSERT INTO restore_outcome_events(
  id, created_at, updated_at, workspace_id, lane_id, query, context_packet_id, snapshot_id, snapshot_kind,
  restore_score, requires_fresh_compile, selected_evidence_json, selected_state_keys_json, selected_loop_ids_json,
  selected_artifact_ids_json, outcome, outcome_confidence, operator_feedback, failure_reason, correction_summary,
  downstream_action_type, downstream_object_id, correlation_id, trace_id, syscall_id, audit_id, proposed_by,
  committed_by, metadata_json
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  created_at=excluded.created_at,
  updated_at=excluded.updated_at,
  workspace_id=excluded.workspace_id,
  lane_id=excluded.lane_id,
  query=excluded.query,
  context_packet_id=excluded.context_packet_id,
  snapshot_id=excluded.snapshot_id,
  snapshot_kind=excluded.snapshot_kind,
  restore_score=excluded.restore_score,
  requires_fresh_compile=excluded.requires_fresh_compile,
  selected_evidence_json=excluded.selected_evidence_json,
  selected_state_keys_json=excluded.selected_state_keys_json,
  selected_loop_ids_json=excluded.selected_loop_ids_json,
  selected_artifact_ids_json=excluded.selected_artifact_ids_json,
  outcome=excluded.outcome,
  outcome_confidence=excluded.outcome_confidence,
  operator_feedback=excluded.operator_feedback,
  failure_reason=excluded.failure_reason,
  correction_summary=excluded.correction_summary,
  downstream_action_type=excluded.downstream_action_type,
  downstream_object_id=excluded.downstream_object_id,
  correlation_id=excluded.correlation_id,
  trace_id=excluded.trace_id,
  syscall_id=excluded.syscall_id,
  audit_id=excluded.audit_id,
  proposed_by=excluded.proposed_by,
  committed_by=excluded.committed_by,
  metadata_json=excluded.metadata_json`,
			fields: []string{
				"id", "created_at", "updated_at", "workspace_id", "lane_id", "query", "context_packet_id", "snapshot_id", "snapshot_kind",
				"restore_score", "requires_fresh_compile", "selected_evidence_json", "selected_state_keys_json", "selected_loop_ids_json",
				"selected_artifact_ids_json", "outcome", "outcome_confidence", "operator_feedback", "failure_reason", "correction_summary",
				"downstream_action_type", "downstream_object_id", "correlation_id", "trace_id", "syscall_id", "audit_id", "proposed_by",
				"committed_by", "metadata_json",
			},
		},
		"semantic_idempotency_keys": {
			sql: `INSERT INTO semantic_idempotency_keys(
  idempotency_key, action, result_json, created_at, correlation_id
) VALUES(?,?,?,?,?)
ON CONFLICT(idempotency_key) DO NOTHING`,
			fields: []string{"idempotency_key", "action", "result_json", "created_at", "correlation_id"},
		},
	}
	addRestoreTable := func(table string, conflict []string, fields []string) {
		m[table] = buildRestoreInsert(table, conflict, fields, false)
	}
	addRestoreTable("sources", []string{"id"}, []string{"id", "path", "created_at", "last_scan_started_at", "last_scan_completed_at", "last_error"})
	addRestoreTable("files", []string{"id"}, []string{"id", "source_id", "rel_path", "abs_path", "size_bytes", "mtime_ns", "content_sha256", "indexed_at"})
	addRestoreTable("chunks", []string{"id"}, []string{"id", "file_id", "chunk_index", "content"})
	addRestoreTable("embedding_records", []string{"id"}, []string{"id", "chunk_id", "file_id", "source_id", "provider", "model", "vector_json", "dims", "norm", "content_sha256", "status", "error_message", "updated_at"})
	addRestoreTable("retrieval_runs", []string{"id"}, []string{"id", "created_at", "query", "mode", "dossier_id", "packet_id", "job_id", "weighting_json", "notes"})
	addRestoreTable("retrieval_results", []string{"id"}, []string{"id", "retrieval_run_id", "chunk_id", "file_id", "abs_path", "rel_path", "rank_index", "keyword_score", "semantic_score", "hybrid_score", "snippet", "selected_for_packet", "usefulness_label", "usefulness_note"})
	addRestoreTable("retrieval_result_selection", []string{"retrieval_result_id"}, []string{"retrieval_result_id", "reason_json", "created_at"})
	addRestoreTable("packet_retrieval_runs", []string{"packet_id", "retrieval_run_id"}, []string{"packet_id", "retrieval_run_id", "created_at"})
	addRestoreTable("dossier_sources", []string{"dossier_id", "source_id"}, []string{"dossier_id", "source_id", "linked_at"})
	addRestoreTable("dossier_jobs", []string{"dossier_id", "job_id"}, []string{"dossier_id", "job_id", "linked_at"})
	addRestoreTable("dossier_packets", []string{"dossier_id", "packet_id"}, []string{"dossier_id", "packet_id", "linked_at"})
	addRestoreTable("dossier_briefs", []string{"id"}, []string{"id", "dossier_id", "created_at", "summary_markdown", "context_json", "notes"})
	addRestoreTable("context_evidence", []string{"id"}, []string{"id", "created_at", "retrieval_result_id", "retrieval_run_id", "job_id", "packet_id", "chunk_id", "evidence_type", "weight", "note"})
	addRestoreTable("memory_observations", []string{"id"}, []string{"id", "created_at", "updated_at", "observed_at", "type", "raw_content", "summary", "embedding_ref", "dossier_id", "project_key", "source_path", "entities_json", "tags_json", "related_files_json", "task_type", "confidence", "verification_state", "lineage_json", "origin_kind", "origin_id", "stale", "last_verified_at", "usefulness_score", "usefulness_count", "noise_count"})
	addRestoreTable("memory_observation_links", []string{"id"}, []string{"id", "created_at", "from_observation_id", "to_observation_id", "relation_type", "note"})
	addRestoreTable("retrieval_result_observations", []string{"retrieval_result_id", "observation_id"}, []string{"retrieval_result_id", "observation_id", "selection_note", "created_at"})
	addRestoreTable("memory_usefulness_events", []string{"id"}, []string{"id", "created_at", "observation_id", "retrieval_result_id", "retrieval_run_id", "packet_id", "job_id", "signal", "weight", "note"})
	addRestoreTable("packet_alignment_notes", []string{"id"}, []string{"id", "packet_id", "observation_id", "retrieval_result_id", "note", "created_at"})
	addRestoreTable("memory_repair_runs", []string{"id"}, []string{"id", "created_at", "started_at", "completed_at", "dossier_id", "mode", "max_age_days", "candidates", "repaired", "skipped", "failed", "note"})
	addRestoreTable("memory_repair_items", []string{"id"}, []string{"id", "repair_run_id", "observation_id", "status", "issue", "before_json", "after_json", "note", "created_at"})
	addRestoreTable("model_manifests", []string{"id"}, []string{"id", "schema_version", "display_name", "family", "format", "backend", "model_path", "sha256", "size_bytes", "quantization", "context_length", "capabilities_json", "default_runtime_json", "license_json", "metadata_json", "discovered_at", "updated_at"})
	addRestoreTable("model_registry_status", []string{"model_id"}, []string{"model_id", "backend", "status", "updated_at", "last_error", "metadata_json"})
	addRestoreTable("model_runtime_loads", []string{"id"}, []string{"id", "model_id", "backend", "status", "loaded_at", "unloaded_at", "endpoint", "pid", "resource_usage_json", "metadata_json"})
	addRestoreTable("chat_threads", []string{"id"}, []string{"id", "title", "created_at", "updated_at", "dossier_id"})
	addRestoreTable("chat_messages", []string{"id"}, []string{"id", "thread_id", "role", "content", "created_at", "metadata_json"})
	addRestoreTable("canvas_boards", []string{"id"}, []string{"id", "title", "dossier_id", "created_at", "updated_at"})
	addRestoreTable("canvas_notes", []string{"id"}, []string{"id", "board_id", "title", "body", "x", "y", "width", "height", "pinned", "color", "links_json", "created_at", "updated_at"})
	addRestoreTable("tool_capability_overrides", []string{"capability_id"}, []string{"capability_id", "status", "reason", "actor", "actor_kind", "previous_status", "risk_class", "transition_risk", "approval_request_id", "correlation_id", "trace_id", "updated_at"})
	addRestoreTable("feature_flags", []string{"key"}, []string{"key", "value", "updated_at", "actor"})
	addRestoreTable("alert_rules", []string{"id"}, []string{"id", "name", "expression", "status", "silenced_until", "created_at", "updated_at"})
	addRestoreTable("scheduled_tasks", []string{"id"}, []string{"id", "kind", "payload_json", "status", "created_at", "updated_at"})
	return m
}()

func buildRestoreInsert(table string, conflictFields, fields []string, doNothing bool) insertMap {
	placeholders := make([]string, len(fields))
	for i := range placeholders {
		placeholders[i] = "?"
	}
	sqlText := fmt.Sprintf("INSERT INTO %s(%s) VALUES(%s)", table, strings.Join(fields, ","), strings.Join(placeholders, ","))
	if len(conflictFields) > 0 {
		sqlText += fmt.Sprintf(" ON CONFLICT(%s)", strings.Join(conflictFields, ","))
	}
	if doNothing {
		sqlText += " DO NOTHING"
	} else {
		assignments := make([]string, 0, len(fields))
		conflictSet := map[string]struct{}{}
		for _, field := range conflictFields {
			conflictSet[field] = struct{}{}
		}
		for _, field := range fields {
			if _, isConflict := conflictSet[field]; isConflict {
				continue
			}
			assignments = append(assignments, fmt.Sprintf("%s=excluded.%s", field, field))
		}
		if len(assignments) == 0 {
			sqlText += " DO NOTHING"
		} else {
			sqlText += " DO UPDATE SET " + strings.Join(assignments, ",")
		}
	}
	return insertMap{sql: sqlText, fields: fields}
}
