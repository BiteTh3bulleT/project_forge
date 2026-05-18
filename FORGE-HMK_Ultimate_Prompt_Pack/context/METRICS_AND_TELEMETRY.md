# Metrics and Telemetry

## Primary metric

**time_to_useful_context**: time from request entering FORGE-HMK/FORGE-T to enough validated, scoped, relevant context being assembled for safe execution.

## Suggested early targets

- cached p50: < 350 ms
- cached p95: < 1.5 s
- fresh p50: < 2.5 s
- fresh p95: < 8 s

## Job metrics

- job_admission_count
- job_coalesced_count
- duplicate_job_rate
- queue_depth
- queue_wait_p95
- lease_timeout_count
- worker_retry_count
- dead_letter_count

## Cache metrics

- hkv_hit_ratio_by_layer
- hkv_miss_ratio_by_layer
- hkv_eviction_rate
- hkv_dirty_hit_blocked_count
- cache_dependency_invalidation_count
- prefix_cache_reuse_count
- compiled_context_reuse_count

## Crucible metrics

- claim_envelope_count
- claim_accept_count
- claim_reject_count
- claim_requires_more_evidence_count
- contradiction_detected_count
- supersession_approved_count
- promotion_blocked_count

## No-effect metrics

These must always be zero:

- shadow_no_effect_violation_count
- unauthorized_mutation_attempt_count
- canonical_write_without_control_lane_count
