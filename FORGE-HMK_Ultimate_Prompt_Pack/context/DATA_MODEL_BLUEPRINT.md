# Data Model Blueprint

Adapt to the repo's actual language/storage conventions.

## MemoryCell

Fields:

- cell_id
- workspace_id
- cell_type
- authority_level
- content_ref
- content_hash
- source_refs
- status
- confidence
- freshness
- created_at
- updated_at
- valid_from
- valid_to
- superseded_by
- schema_version

Cell types: FactCell, EventCell, StateCell, TaskCell, RuleCell, DecisionCell, ArtifactCell, PhotoCell, KineticCell, TraceCell, ReplayCell, ProjectionCell, ClaimCell.

## Synapse

Fields:

- synapse_id
- src_cell_id
- dst_cell_id
- relation_type
- weight
- confidence
- activation_count
- last_activated_at
- valid_from
- valid_to
- source_refs
- status

Relations: supports, contradicts, supersedes, depends_on, derived_from, belongs_to, blocks, unblocks, frequently_coactivated, requires_verification, same_entity_as, related_artifact, related_tool, related_policy.

## Job

Fields: job_id, kind, workspace_id, request_id, priority, policy_class, deadline_at, ttl_seconds, dedupe_key, parent_job_id, status, created_at, updated_at.

Kinds: PULL_SNAPSHOT, ASSEMBLE_CONTEXT, DECODE_CACHE_EXPRESSION, CAPTURE_PHOTO_FRAME, APPEND_KINETIC_DELTA, BUILD_TRACE, BUILD_REPLAY, PREWARM_HKV, VALIDATE_CLAIM, COMPARE_SHADOW, PROPOSE_PROMOTION.

## WorkOrder

Fields: work_order_id, job_id, team_type, step_kind, input_refs, expected_artifacts, affinity_tags, max_runtime_ms, cpu_budget_ms, gpu_budget_ms, token_budget, retry_policy, idempotency_key, status.

## ClaimEnvelope

Fields: claim_id, artifact_id, claim_type, proposed_operation, evidence_refs, contradiction_refs, confidence, scope, requires_crucible, requires_control_lane, validation_status, decision_reason.

## HKVManifest

Fields: cache_id, layer, workspace_id, model_id, tokenizer_id, prompt_template_hash, policy_epoch, memory_epoch, source_refs, dependency_hash, payload_ref, ttl_at, dirty_state, hit_count, last_used_at.
