package store

import (
	"database/sql"
	"fmt"
)

const schema = `
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS settings (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS sources (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  path TEXT NOT NULL UNIQUE,
  created_at INTEGER NOT NULL,
  last_scan_started_at INTEGER,
  last_scan_completed_at INTEGER,
  last_error TEXT
);

CREATE TABLE IF NOT EXISTS files (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  source_id INTEGER NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
  rel_path TEXT NOT NULL,
  abs_path TEXT NOT NULL,
  size_bytes INTEGER NOT NULL,
  mtime_ns INTEGER NOT NULL,
  content_sha256 TEXT NOT NULL,
  indexed_at INTEGER NOT NULL,
  UNIQUE(source_id, rel_path)
);

CREATE TABLE IF NOT EXISTS chunks (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  file_id INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
  chunk_index INTEGER NOT NULL,
  content TEXT NOT NULL,
  UNIQUE(file_id, chunk_index)
);

CREATE VIRTUAL TABLE IF NOT EXISTS chunks_fts USING fts5(
  content,
  content='chunks',
  content_rowid='id',
  tokenize = 'porter unicode61'
);

CREATE TRIGGER IF NOT EXISTS chunks_ai AFTER INSERT ON chunks BEGIN
  INSERT INTO chunks_fts(rowid, content) VALUES (new.id, new.content);
END;

CREATE TRIGGER IF NOT EXISTS chunks_ad AFTER DELETE ON chunks BEGIN
  INSERT INTO chunks_fts(chunks_fts, rowid, content) VALUES('delete', old.id, old.content);
END;

CREATE TRIGGER IF NOT EXISTS chunks_au AFTER UPDATE ON chunks BEGIN
  INSERT INTO chunks_fts(chunks_fts, rowid, content) VALUES('delete', old.id, old.content);
  INSERT INTO chunks_fts(rowid, content) VALUES (new.id, new.content);
END;

CREATE TABLE IF NOT EXISTS events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  created_at INTEGER NOT NULL,
  type TEXT NOT NULL,
  payload_json TEXT NOT NULL DEFAULT '{}'
);

CREATE TABLE IF NOT EXISTS task_packets (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  packet_version INTEGER NOT NULL,
  created_at INTEGER NOT NULL,
  generated_at INTEGER NOT NULL,
  title TEXT NOT NULL,
  user_request TEXT NOT NULL,
  objective TEXT NOT NULL,
  adapter_target TEXT NOT NULL,
  execution_mode TEXT NOT NULL,
  risk_class TEXT NOT NULL,
  expected_output_json TEXT NOT NULL DEFAULT '{}',
  constraints_json TEXT NOT NULL DEFAULT '[]',
  instructions TEXT NOT NULL DEFAULT '',
  selected_paths_json TEXT NOT NULL DEFAULT '[]',
  scope_snapshot_json TEXT NOT NULL DEFAULT '{}',
  source_references_json TEXT NOT NULL DEFAULT '[]',
  retrieved_context_json TEXT NOT NULL DEFAULT '[]',
  project_notes TEXT NOT NULL DEFAULT '',
  source_context_record_ids_json TEXT NOT NULL DEFAULT '[]',
  request_payload_json TEXT NOT NULL DEFAULT '{}'
);

CREATE TABLE IF NOT EXISTS jobs (
  id TEXT PRIMARY KEY,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  queued_at INTEGER,
  started_at INTEGER,
  completed_at INTEGER,
  title TEXT NOT NULL,
  requested_action TEXT NOT NULL,
  target_adapter TEXT NOT NULL,
  initiating_source TEXT NOT NULL,
  execution_boundary TEXT NOT NULL,
  risk_class TEXT NOT NULL,
  status TEXT NOT NULL,
  approval_status TEXT NOT NULL,
  write_intent INTEGER NOT NULL DEFAULT 0,
  cancel_requested INTEGER NOT NULL DEFAULT 0,
  task_packet_id INTEGER,
  result_summary TEXT,
  failure_info TEXT,
  last_failure_code TEXT,
  last_error TEXT,
  metadata_json TEXT NOT NULL DEFAULT '{}',
  FOREIGN KEY(task_packet_id) REFERENCES task_packets(id)
);

CREATE TABLE IF NOT EXISTS job_status_history (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  job_id TEXT NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
  created_at INTEGER NOT NULL,
  from_status TEXT,
  to_status TEXT NOT NULL,
  reason TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS job_events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  job_id TEXT NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
  created_at INTEGER NOT NULL,
  type TEXT NOT NULL,
  message TEXT NOT NULL,
  payload_json TEXT NOT NULL DEFAULT '{}'
);

CREATE TABLE IF NOT EXISTS approval_requests (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  job_id TEXT NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
  created_at INTEGER NOT NULL,
  status TEXT NOT NULL,
  requested_action TEXT NOT NULL,
  risk_class TEXT NOT NULL,
  requested_adapter TEXT NOT NULL,
  write_intent INTEGER NOT NULL DEFAULT 0,
  scope_snapshot_json TEXT NOT NULL DEFAULT '{}',
  task_packet_id INTEGER,
  request_summary TEXT NOT NULL DEFAULT '',
  FOREIGN KEY(task_packet_id) REFERENCES task_packets(id)
);

CREATE TABLE IF NOT EXISTS approval_decisions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  request_id INTEGER NOT NULL REFERENCES approval_requests(id) ON DELETE CASCADE,
  created_at INTEGER NOT NULL,
  actor TEXT NOT NULL,
  decision TEXT NOT NULL,
  note TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS artifacts (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  created_at INTEGER NOT NULL,
  job_id TEXT REFERENCES jobs(id) ON DELETE SET NULL,
  packet_id INTEGER REFERENCES task_packets(id) ON DELETE SET NULL,
  type TEXT NOT NULL,
  title TEXT NOT NULL,
  file_path TEXT NOT NULL,
  mime_type TEXT NOT NULL,
  metadata_json TEXT NOT NULL DEFAULT '{}'
);

CREATE TABLE IF NOT EXISTS project_context_records (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  context_version INTEGER NOT NULL,
  created_at INTEGER NOT NULL,
  generated_at INTEGER NOT NULL,
  source_path TEXT NOT NULL,
  source_hash TEXT NOT NULL,
  source_size_bytes INTEGER NOT NULL,
  normalized_summary_json TEXT NOT NULL,
  briefing_markdown TEXT NOT NULL,
  agents_markdown TEXT NOT NULL,
  claude_markdown TEXT NOT NULL,
  cursor_markdown TEXT NOT NULL,
  generated_agents_path TEXT NOT NULL,
  generated_claude_path TEXT NOT NULL,
  generated_briefing_path TEXT NOT NULL,
  generated_cursor_path TEXT NOT NULL,
  notes TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS adapter_status_cache (
  adapter_id TEXT PRIMARY KEY,
  updated_at INTEGER NOT NULL,
  status_json TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS embedding_records (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  chunk_id INTEGER NOT NULL REFERENCES chunks(id) ON DELETE CASCADE,
  file_id INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
  source_id INTEGER NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
  provider TEXT NOT NULL,
  model TEXT NOT NULL,
  vector_json TEXT,
  dims INTEGER NOT NULL DEFAULT 0,
  norm REAL NOT NULL DEFAULT 0,
  content_sha256 TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'pending',
  error_message TEXT,
  updated_at INTEGER NOT NULL,
  UNIQUE(chunk_id, provider, model)
);

CREATE TABLE IF NOT EXISTS retrieval_runs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  created_at INTEGER NOT NULL,
  query TEXT NOT NULL,
  mode TEXT NOT NULL,
  dossier_id INTEGER REFERENCES dossiers(id) ON DELETE SET NULL,
  packet_id INTEGER REFERENCES task_packets(id) ON DELETE SET NULL,
  job_id TEXT REFERENCES jobs(id) ON DELETE SET NULL,
  weighting_json TEXT NOT NULL DEFAULT '{}',
  notes TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS retrieval_results (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  retrieval_run_id INTEGER NOT NULL REFERENCES retrieval_runs(id) ON DELETE CASCADE,
  chunk_id INTEGER REFERENCES chunks(id) ON DELETE SET NULL,
  file_id INTEGER REFERENCES files(id) ON DELETE SET NULL,
  abs_path TEXT NOT NULL,
  rel_path TEXT NOT NULL,
  rank_index INTEGER NOT NULL,
  keyword_score REAL NOT NULL DEFAULT 0,
  semantic_score REAL NOT NULL DEFAULT 0,
  hybrid_score REAL NOT NULL DEFAULT 0,
  snippet TEXT NOT NULL DEFAULT '',
  selected_for_packet INTEGER NOT NULL DEFAULT 0,
  usefulness_label TEXT NOT NULL DEFAULT 'unknown',
  usefulness_note TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS retrieval_result_selection (
  retrieval_result_id INTEGER PRIMARY KEY REFERENCES retrieval_results(id) ON DELETE CASCADE,
  reason_json TEXT NOT NULL DEFAULT '{}',
  created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS packet_retrieval_runs (
  packet_id INTEGER NOT NULL REFERENCES task_packets(id) ON DELETE CASCADE,
  retrieval_run_id INTEGER NOT NULL REFERENCES retrieval_runs(id) ON DELETE CASCADE,
  created_at INTEGER NOT NULL,
  PRIMARY KEY(packet_id, retrieval_run_id)
);

CREATE TABLE IF NOT EXISTS dossiers (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  name TEXT NOT NULL UNIQUE,
  description TEXT NOT NULL DEFAULT '',
  primary_paths_json TEXT NOT NULL DEFAULT '[]',
  related_repos_json TEXT NOT NULL DEFAULT '[]',
  constraints_json TEXT NOT NULL DEFAULT '[]',
  preferred_adapters_json TEXT NOT NULL DEFAULT '[]',
  important_files_json TEXT NOT NULL DEFAULT '[]',
  routing_notes TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS dossier_sources (
  dossier_id INTEGER NOT NULL REFERENCES dossiers(id) ON DELETE CASCADE,
  source_id INTEGER NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
  linked_at INTEGER NOT NULL,
  PRIMARY KEY(dossier_id, source_id)
);

CREATE TABLE IF NOT EXISTS dossier_jobs (
  dossier_id INTEGER NOT NULL REFERENCES dossiers(id) ON DELETE CASCADE,
  job_id TEXT NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
  linked_at INTEGER NOT NULL,
  PRIMARY KEY(dossier_id, job_id)
);

CREATE TABLE IF NOT EXISTS dossier_packets (
  dossier_id INTEGER NOT NULL REFERENCES dossiers(id) ON DELETE CASCADE,
  packet_id INTEGER NOT NULL REFERENCES task_packets(id) ON DELETE CASCADE,
  linked_at INTEGER NOT NULL,
  PRIMARY KEY(dossier_id, packet_id)
);

CREATE TABLE IF NOT EXISTS dossier_briefs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  dossier_id INTEGER NOT NULL REFERENCES dossiers(id) ON DELETE CASCADE,
  created_at INTEGER NOT NULL,
  summary_markdown TEXT NOT NULL,
  context_json TEXT NOT NULL DEFAULT '{}',
  notes TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS context_evidence (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  created_at INTEGER NOT NULL,
  retrieval_result_id INTEGER REFERENCES retrieval_results(id) ON DELETE SET NULL,
  retrieval_run_id INTEGER REFERENCES retrieval_runs(id) ON DELETE SET NULL,
  job_id TEXT REFERENCES jobs(id) ON DELETE SET NULL,
  packet_id INTEGER REFERENCES task_packets(id) ON DELETE SET NULL,
  chunk_id INTEGER REFERENCES chunks(id) ON DELETE SET NULL,
  evidence_type TEXT NOT NULL,
  weight REAL NOT NULL DEFAULT 1,
  note TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS memory_observations (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  observed_at INTEGER NOT NULL,
  type TEXT NOT NULL,
  raw_content TEXT NOT NULL DEFAULT '',
  summary TEXT NOT NULL DEFAULT '',
  embedding_ref TEXT NOT NULL DEFAULT '',
  dossier_id INTEGER REFERENCES dossiers(id) ON DELETE SET NULL,
  project_key TEXT NOT NULL DEFAULT '',
  source_path TEXT NOT NULL DEFAULT '',
  entities_json TEXT NOT NULL DEFAULT '[]',
  tags_json TEXT NOT NULL DEFAULT '[]',
  related_files_json TEXT NOT NULL DEFAULT '[]',
  task_type TEXT NOT NULL DEFAULT '',
  confidence REAL NOT NULL DEFAULT 0.5,
  verification_state TEXT NOT NULL DEFAULT 'unknown',
  lineage_json TEXT NOT NULL DEFAULT '[]',
  origin_kind TEXT NOT NULL DEFAULT '',
  origin_id TEXT NOT NULL DEFAULT '',
  stale INTEGER NOT NULL DEFAULT 0,
  last_verified_at INTEGER,
  usefulness_score REAL NOT NULL DEFAULT 0,
  usefulness_count INTEGER NOT NULL DEFAULT 0,
  noise_count INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS memory_observation_links (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  created_at INTEGER NOT NULL,
  from_observation_id INTEGER NOT NULL REFERENCES memory_observations(id) ON DELETE CASCADE,
  to_observation_id INTEGER NOT NULL REFERENCES memory_observations(id) ON DELETE CASCADE,
  relation_type TEXT NOT NULL,
  note TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS retrieval_result_observations (
  retrieval_result_id INTEGER NOT NULL REFERENCES retrieval_results(id) ON DELETE CASCADE,
  observation_id INTEGER NOT NULL REFERENCES memory_observations(id) ON DELETE CASCADE,
  selection_note TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL,
  PRIMARY KEY(retrieval_result_id, observation_id)
);

CREATE TABLE IF NOT EXISTS memory_usefulness_events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  created_at INTEGER NOT NULL,
  observation_id INTEGER NOT NULL REFERENCES memory_observations(id) ON DELETE CASCADE,
  retrieval_result_id INTEGER REFERENCES retrieval_results(id) ON DELETE SET NULL,
  retrieval_run_id INTEGER REFERENCES retrieval_runs(id) ON DELETE SET NULL,
  packet_id INTEGER REFERENCES task_packets(id) ON DELETE SET NULL,
  job_id TEXT REFERENCES jobs(id) ON DELETE SET NULL,
  signal TEXT NOT NULL,
  weight REAL NOT NULL DEFAULT 1,
  note TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS packet_alignment_notes (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  packet_id INTEGER NOT NULL REFERENCES task_packets(id) ON DELETE CASCADE,
  observation_id INTEGER REFERENCES memory_observations(id) ON DELETE SET NULL,
  retrieval_result_id INTEGER REFERENCES retrieval_results(id) ON DELETE SET NULL,
  note TEXT NOT NULL,
  created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS memory_repair_runs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  created_at INTEGER NOT NULL,
  started_at INTEGER NOT NULL,
  completed_at INTEGER,
  dossier_id INTEGER REFERENCES dossiers(id) ON DELETE SET NULL,
  mode TEXT NOT NULL DEFAULT 'manual',
  max_age_days INTEGER NOT NULL DEFAULT 14,
  candidates INTEGER NOT NULL DEFAULT 0,
  repaired INTEGER NOT NULL DEFAULT 0,
  skipped INTEGER NOT NULL DEFAULT 0,
  failed INTEGER NOT NULL DEFAULT 0,
  note TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS memory_repair_items (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  repair_run_id INTEGER NOT NULL REFERENCES memory_repair_runs(id) ON DELETE CASCADE,
  observation_id INTEGER NOT NULL REFERENCES memory_observations(id) ON DELETE CASCADE,
  status TEXT NOT NULL,
  issue TEXT NOT NULL DEFAULT '',
  before_json TEXT NOT NULL DEFAULT '{}',
  after_json TEXT NOT NULL DEFAULT '{}',
  note TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS memory_vsa_pointers (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  observation_id INTEGER NOT NULL UNIQUE REFERENCES memory_observations(id) ON DELETE CASCADE,
  dims INTEGER NOT NULL DEFAULT 128,
  pointer_json TEXT NOT NULL DEFAULT '[]',
  norm REAL NOT NULL DEFAULT 0,
  source_fingerprint TEXT NOT NULL DEFAULT '',
  stale INTEGER NOT NULL DEFAULT 0,
  metadata_json TEXT NOT NULL DEFAULT '{}',
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS memory_vsa_role_bindings (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  observation_id INTEGER NOT NULL REFERENCES memory_observations(id) ON DELETE CASCADE,
  role TEXT NOT NULL,
  filler TEXT NOT NULL,
  weight REAL NOT NULL DEFAULT 1,
  support_count INTEGER NOT NULL DEFAULT 0,
  noise_count INTEGER NOT NULL DEFAULT 0,
  binding_json TEXT NOT NULL DEFAULT '[]',
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  UNIQUE(observation_id, role, filler)
);

CREATE TABLE IF NOT EXISTS memory_vsa_associations (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  from_observation_id INTEGER NOT NULL REFERENCES memory_observations(id) ON DELETE CASCADE,
  to_observation_id INTEGER NOT NULL REFERENCES memory_observations(id) ON DELETE CASCADE,
  association_type TEXT NOT NULL DEFAULT 'related',
  strength REAL NOT NULL DEFAULT 0,
  support_count INTEGER NOT NULL DEFAULT 0,
  noise_count INTEGER NOT NULL DEFAULT 0,
  evidence_json TEXT NOT NULL DEFAULT '{}',
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  UNIQUE(from_observation_id, to_observation_id, association_type)
);

CREATE TABLE IF NOT EXISTS retrieval_result_vsa_signals (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  retrieval_result_id INTEGER NOT NULL UNIQUE REFERENCES retrieval_results(id) ON DELETE CASCADE,
  retrieval_run_id INTEGER NOT NULL REFERENCES retrieval_runs(id) ON DELETE CASCADE,
  observation_id INTEGER REFERENCES memory_observations(id) ON DELETE SET NULL,
  mode TEXT NOT NULL DEFAULT 'off',
  associative_score REAL NOT NULL DEFAULT 0,
  role_match_score REAL NOT NULL DEFAULT 0,
  relational_score REAL NOT NULL DEFAULT 0,
  feedback_score REAL NOT NULL DEFAULT 0,
  additive_score REAL NOT NULL DEFAULT 0,
  applied_score REAL NOT NULL DEFAULT 0,
  explain_json TEXT NOT NULL DEFAULT '{}',
  created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS memory_vsa_reindex_runs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  created_at INTEGER NOT NULL,
  started_at INTEGER NOT NULL,
  completed_at INTEGER,
  dossier_id INTEGER REFERENCES dossiers(id) ON DELETE SET NULL,
  mode TEXT NOT NULL DEFAULT 'manual',
  status TEXT NOT NULL DEFAULT 'running',
  candidates INTEGER NOT NULL DEFAULT 0,
  indexed INTEGER NOT NULL DEFAULT 0,
  skipped INTEGER NOT NULL DEFAULT 0,
  failed INTEGER NOT NULL DEFAULT 0,
  triggered_by TEXT NOT NULL DEFAULT '',
  note TEXT NOT NULL DEFAULT '',
  settings_json TEXT NOT NULL DEFAULT '{}'
);

CREATE TABLE IF NOT EXISTS memory_vsa_reindex_items (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  reindex_run_id INTEGER NOT NULL REFERENCES memory_vsa_reindex_runs(id) ON DELETE CASCADE,
  observation_id INTEGER NOT NULL REFERENCES memory_observations(id) ON DELETE CASCADE,
  status TEXT NOT NULL,
  reason TEXT NOT NULL DEFAULT '',
  before_fingerprint TEXT NOT NULL DEFAULT '',
  after_fingerprint TEXT NOT NULL DEFAULT '',
  before_json TEXT NOT NULL DEFAULT '{}',
  after_json TEXT NOT NULL DEFAULT '{}',
  note TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS evaluation_records (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  created_at INTEGER NOT NULL,
  job_id TEXT NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
  dossier_id INTEGER REFERENCES dossiers(id) ON DELETE SET NULL,
  success INTEGER NOT NULL,
  quality_rating INTEGER NOT NULL,
  usefulness_rating INTEGER NOT NULL,
  correctness_confidence INTEGER NOT NULL,
  packet_quality_rating INTEGER NOT NULL,
  adapter_suitability INTEGER NOT NULL,
  retry_recommended INTEGER NOT NULL DEFAULT 0,
  influence_routing INTEGER NOT NULL DEFAULT 1,
  notes TEXT NOT NULL DEFAULT '',
  scorer TEXT NOT NULL DEFAULT 'operator'
);

CREATE TABLE IF NOT EXISTS job_lineage (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  parent_job_id TEXT NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
  child_job_id TEXT NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
  relation_type TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  change_summary_json TEXT NOT NULL DEFAULT '{}',
  UNIQUE(parent_job_id, child_job_id)
);

CREATE TABLE IF NOT EXISTS imported_executions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  created_at INTEGER NOT NULL,
  adapter_id TEXT NOT NULL,
  external_run_id TEXT NOT NULL DEFAULT '',
  origin_job_id TEXT REFERENCES jobs(id) ON DELETE SET NULL,
  origin_packet_id INTEGER REFERENCES task_packets(id) ON DELETE SET NULL,
  dossier_id INTEGER REFERENCES dossiers(id) ON DELETE SET NULL,
  summary TEXT NOT NULL,
  output_refs_json TEXT NOT NULL DEFAULT '[]',
  diff_summary TEXT NOT NULL DEFAULT '',
  execution_notes TEXT NOT NULL DEFAULT '',
  evaluation_json TEXT NOT NULL DEFAULT '{}'
);

CREATE TABLE IF NOT EXISTS routing_insights (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  created_at INTEGER NOT NULL,
  dossier_id INTEGER REFERENCES dossiers(id) ON DELETE SET NULL,
  adapter_id TEXT NOT NULL,
  task_type TEXT NOT NULL,
  recommendation TEXT NOT NULL,
  confidence REAL NOT NULL DEFAULT 0,
  reasons_json TEXT NOT NULL DEFAULT '[]',
  evidence_json TEXT NOT NULL DEFAULT '{}',
  advisory_level TEXT NOT NULL DEFAULT 'advisory'
);

CREATE TABLE IF NOT EXISTS execution_strategies (
  id TEXT PRIMARY KEY,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  name TEXT NOT NULL UNIQUE,
  task_type TEXT NOT NULL,
  target_adapter TEXT NOT NULL,
  retrieval_mode TEXT NOT NULL,
  packet_rules_json TEXT NOT NULL DEFAULT '{}',
  approval_required INTEGER NOT NULL DEFAULT 1,
  approval_preset_id TEXT,
  expected_artifacts_json TEXT NOT NULL DEFAULT '[]',
  success_criteria_json TEXT NOT NULL DEFAULT '{}',
  retry_guidance_json TEXT NOT NULL DEFAULT '{}',
  enabled INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS approval_presets (
  id TEXT PRIMARY KEY,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  name TEXT NOT NULL UNIQUE,
  description TEXT NOT NULL DEFAULT '',
  profile_json TEXT NOT NULL DEFAULT '{}',
  editable INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS dossier_profiles (
  dossier_id INTEGER PRIMARY KEY REFERENCES dossiers(id) ON DELETE CASCADE,
  updated_at INTEGER NOT NULL,
  preferred_strategies_json TEXT NOT NULL DEFAULT '[]',
  preferred_adapters_json TEXT NOT NULL DEFAULT '[]',
  approval_preset_id TEXT,
  retrieval_defaults_json TEXT NOT NULL DEFAULT '{}',
  high_value_files_json TEXT NOT NULL DEFAULT '[]',
  noisy_files_json TEXT NOT NULL DEFAULT '[]',
  routing_notes TEXT NOT NULL DEFAULT '',
  automation_bindings_json TEXT NOT NULL DEFAULT '[]'
);

CREATE TABLE IF NOT EXISTS routing_policy_recommendations (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  created_at INTEGER NOT NULL,
  dossier_id INTEGER REFERENCES dossiers(id) ON DELETE SET NULL,
  task_type TEXT NOT NULL,
  strategy_id TEXT,
  target_adapter TEXT NOT NULL,
  retrieval_mode TEXT NOT NULL,
  packet_shape_json TEXT NOT NULL DEFAULT '{}',
  approval_preset_id TEXT,
  approval_required INTEGER NOT NULL DEFAULT 1,
  confidence REAL NOT NULL DEFAULT 0,
  reasons_json TEXT NOT NULL DEFAULT '[]',
  evidence_json TEXT NOT NULL DEFAULT '{}',
  inferred INTEGER NOT NULL DEFAULT 1,
  operator_override_allowed INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS automation_rules (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  name TEXT NOT NULL UNIQUE,
  trigger TEXT NOT NULL,
  condition_json TEXT NOT NULL DEFAULT '{}',
  action_json TEXT NOT NULL DEFAULT '{}',
  scope_json TEXT NOT NULL DEFAULT '{}',
  enabled INTEGER NOT NULL DEFAULT 1,
  dry_run_default INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS automation_history (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  created_at INTEGER NOT NULL,
  rule_id INTEGER REFERENCES automation_rules(id) ON DELETE SET NULL,
  trigger TEXT NOT NULL,
  matched INTEGER NOT NULL DEFAULT 0,
  dry_run INTEGER NOT NULL DEFAULT 1,
  status TEXT NOT NULL,
  message TEXT NOT NULL DEFAULT '',
  preview_json TEXT NOT NULL DEFAULT '{}',
  result_json TEXT NOT NULL DEFAULT '{}'
);

CREATE TABLE IF NOT EXISTS packet_guidance_records (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  created_at INTEGER NOT NULL,
  packet_id INTEGER REFERENCES task_packets(id) ON DELETE CASCADE,
  job_id TEXT REFERENCES jobs(id) ON DELETE SET NULL,
  dossier_id INTEGER REFERENCES dossiers(id) ON DELETE SET NULL,
  guidance_score REAL NOT NULL DEFAULT 0,
  issues_json TEXT NOT NULL DEFAULT '[]',
  recommendations_json TEXT NOT NULL DEFAULT '[]',
  evidence_json TEXT NOT NULL DEFAULT '{}'
);

CREATE TABLE IF NOT EXISTS imported_execution_reconciliations (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  import_id INTEGER NOT NULL UNIQUE REFERENCES imported_executions(id) ON DELETE CASCADE,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  changed_files_json TEXT NOT NULL DEFAULT '[]',
  failure_reasons_json TEXT NOT NULL DEFAULT '[]',
  unresolved_issues_json TEXT NOT NULL DEFAULT '[]',
  suggested_next_steps_json TEXT NOT NULL DEFAULT '[]',
  agent_notes TEXT NOT NULL DEFAULT '',
  patch_summary TEXT NOT NULL DEFAULT '',
  review_status TEXT NOT NULL DEFAULT 'pending'
);

CREATE TABLE IF NOT EXISTS review_records (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  target_type TEXT NOT NULL,
  target_id TEXT NOT NULL,
  dossier_id INTEGER REFERENCES dossiers(id) ON DELETE SET NULL,
  status TEXT NOT NULL,
  summary TEXT NOT NULL DEFAULT '',
  notes TEXT NOT NULL DEFAULT '',
  annotations_json TEXT NOT NULL DEFAULT '[]',
  reviewer TEXT NOT NULL DEFAULT 'operator'
);

CREATE TABLE IF NOT EXISTS failure_pattern_snapshots (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  created_at INTEGER NOT NULL,
  dossier_id INTEGER REFERENCES dossiers(id) ON DELETE SET NULL,
  target_adapter TEXT NOT NULL,
  strategy_id TEXT,
  retrieval_mode TEXT NOT NULL,
  packet_style TEXT NOT NULL,
  failure_code TEXT NOT NULL,
  failure_count INTEGER NOT NULL,
  recommendation TEXT NOT NULL,
  evidence_json TEXT NOT NULL DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_events_created ON events(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_files_source ON files(source_id);
CREATE INDEX IF NOT EXISTS idx_jobs_status_created ON jobs(status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_jobs_adapter_created ON jobs(target_adapter, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_job_status_history_job ON job_status_history(job_id, id);
CREATE INDEX IF NOT EXISTS idx_job_events_job ON job_events(job_id, id);
CREATE INDEX IF NOT EXISTS idx_approval_requests_status_created ON approval_requests(status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_approval_requests_job ON approval_requests(job_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_approval_decisions_request ON approval_decisions(request_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_packets_created ON task_packets(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_artifacts_job ON artifacts(job_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_project_context_created ON project_context_records(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_embedding_source_status ON embedding_records(source_id, status);
CREATE INDEX IF NOT EXISTS idx_embedding_chunk_provider ON embedding_records(chunk_id, provider, model);
CREATE INDEX IF NOT EXISTS idx_retrieval_runs_created ON retrieval_runs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_retrieval_runs_job ON retrieval_runs(job_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_retrieval_results_run ON retrieval_results(retrieval_run_id, rank_index);
CREATE INDEX IF NOT EXISTS idx_retrieval_result_selection_created ON retrieval_result_selection(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_context_evidence_run ON context_evidence(retrieval_run_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_context_evidence_job ON context_evidence(job_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_memory_obs_type ON memory_observations(type, observed_at DESC);
CREATE INDEX IF NOT EXISTS idx_memory_obs_dossier ON memory_observations(dossier_id, observed_at DESC);
CREATE INDEX IF NOT EXISTS idx_memory_obs_origin ON memory_observations(origin_kind, origin_id);
CREATE INDEX IF NOT EXISTS idx_memory_obs_stale ON memory_observations(stale, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_memory_obs_source_path ON memory_observations(source_path, observed_at DESC);
CREATE INDEX IF NOT EXISTS idx_memory_links_from ON memory_observation_links(from_observation_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_memory_links_to ON memory_observation_links(to_observation_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_result_obs_obs ON retrieval_result_observations(observation_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_memory_usefulness_obs ON memory_usefulness_events(observation_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_packet_alignment_packet ON packet_alignment_notes(packet_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_memory_repair_runs_created ON memory_repair_runs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_memory_repair_runs_dossier ON memory_repair_runs(dossier_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_memory_repair_items_run ON memory_repair_items(repair_run_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_memory_vsa_pointers_obs ON memory_vsa_pointers(observation_id);
CREATE INDEX IF NOT EXISTS idx_memory_vsa_pointers_fp ON memory_vsa_pointers(source_fingerprint, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_memory_vsa_bindings_obs ON memory_vsa_role_bindings(observation_id, role);
CREATE INDEX IF NOT EXISTS idx_memory_vsa_assoc_from ON memory_vsa_associations(from_observation_id, strength DESC);
CREATE INDEX IF NOT EXISTS idx_memory_vsa_assoc_to ON memory_vsa_associations(to_observation_id, strength DESC);
CREATE INDEX IF NOT EXISTS idx_result_vsa_signals_run ON retrieval_result_vsa_signals(retrieval_run_id, retrieval_result_id);
CREATE INDEX IF NOT EXISTS idx_result_vsa_signals_obs ON retrieval_result_vsa_signals(observation_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_result_vsa_signals_mode ON retrieval_result_vsa_signals(mode, applied_score DESC, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_memory_vsa_reindex_runs_created ON memory_vsa_reindex_runs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_memory_vsa_reindex_runs_dossier ON memory_vsa_reindex_runs(dossier_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_memory_vsa_reindex_items_run ON memory_vsa_reindex_items(reindex_run_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_memory_vsa_reindex_items_obs ON memory_vsa_reindex_items(observation_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_eval_job ON evaluation_records(job_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_eval_dossier ON evaluation_records(dossier_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_lineage_parent ON job_lineage(parent_job_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_lineage_child ON job_lineage(child_job_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_import_origin_job ON imported_executions(origin_job_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_import_dossier ON imported_executions(dossier_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_insights_dossier ON routing_insights(dossier_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_strategy_task_type ON execution_strategies(task_type, enabled);
CREATE INDEX IF NOT EXISTS idx_dossier_profiles_preset ON dossier_profiles(approval_preset_id);
CREATE INDEX IF NOT EXISTS idx_policy_reco_created ON routing_policy_recommendations(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_policy_reco_dossier ON routing_policy_recommendations(dossier_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_automation_rules_enabled ON automation_rules(enabled, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_automation_history_rule ON automation_history(rule_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_packet_guidance_packet ON packet_guidance_records(packet_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_import_reconcile_import ON imported_execution_reconciliations(import_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_reviews_status ON review_records(status, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_reviews_target ON review_records(target_type, target_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_failure_patterns_created ON failure_pattern_snapshots(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_failure_patterns_dossier ON failure_pattern_snapshots(dossier_id, created_at DESC);

CREATE TABLE IF NOT EXISTS permission_profiles (
  id TEXT PRIMARY KEY,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  name TEXT NOT NULL UNIQUE,
  description TEXT NOT NULL DEFAULT '',
  allowed_read_paths_json TEXT NOT NULL DEFAULT '[]',
  allowed_write_paths_json TEXT NOT NULL DEFAULT '[]',
  allowed_execute_paths_json TEXT NOT NULL DEFAULT '[]',
  forbidden_paths_json TEXT NOT NULL DEFAULT '[]',
  allowed_tools_json TEXT NOT NULL DEFAULT '[]',
  approval_required_risks_json TEXT NOT NULL DEFAULT '["medium","high"]',
  max_bytes_per_write INTEGER NOT NULL DEFAULT 524288,
  allow_network INTEGER NOT NULL DEFAULT 0,
  editable INTEGER NOT NULL DEFAULT 1,
  active INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS action_lanes (
  id TEXT PRIMARY KEY,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  name TEXT NOT NULL UNIQUE,
  description TEXT NOT NULL DEFAULT '',
  action_type TEXT NOT NULL,
  allowed_paths_json TEXT NOT NULL DEFAULT '[]',
  forbidden_paths_json TEXT NOT NULL DEFAULT '[]',
  write_intent INTEGER NOT NULL DEFAULT 0,
  requires_approval INTEGER NOT NULL DEFAULT 0,
  risk_class TEXT NOT NULL DEFAULT 'low',
  max_bytes INTEGER NOT NULL DEFAULT 524288,
  expected_artifacts_json TEXT NOT NULL DEFAULT '[]',
  builtin INTEGER NOT NULL DEFAULT 0,
  enabled INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS gateway_invocations (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  correlation_id TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  completed_at INTEGER,
  tool_id TEXT NOT NULL,
  lane_id TEXT,
  job_id TEXT REFERENCES jobs(id) ON DELETE SET NULL,
  packet_id INTEGER REFERENCES task_packets(id) ON DELETE SET NULL,
  approval_request_id INTEGER REFERENCES approval_requests(id) ON DELETE SET NULL,
  initiator TEXT NOT NULL DEFAULT 'operator',
  action TEXT NOT NULL,
  risk_class TEXT NOT NULL,
  write_intent INTEGER NOT NULL DEFAULT 0,
  scope_json TEXT NOT NULL DEFAULT '{}',
  input_json TEXT NOT NULL DEFAULT '{}',
  status TEXT NOT NULL,
  denied_reason TEXT NOT NULL DEFAULT '',
  result_json TEXT NOT NULL DEFAULT '{}',
  artifacts_json TEXT NOT NULL DEFAULT '[]',
  permission_profile_id TEXT
);

CREATE TABLE IF NOT EXISTS audit_records (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  created_at INTEGER NOT NULL,
  correlation_id TEXT NOT NULL DEFAULT '',
  category TEXT NOT NULL,
  action TEXT NOT NULL,
  actor TEXT NOT NULL DEFAULT 'operator',
  subject_type TEXT NOT NULL DEFAULT '',
  subject_id TEXT NOT NULL DEFAULT '',
  job_id TEXT REFERENCES jobs(id) ON DELETE SET NULL,
  gateway_invocation_id INTEGER REFERENCES gateway_invocations(id) ON DELETE SET NULL,
  approval_request_id INTEGER REFERENCES approval_requests(id) ON DELETE SET NULL,
  risk_class TEXT NOT NULL DEFAULT '',
  outcome TEXT NOT NULL DEFAULT '',
  summary TEXT NOT NULL DEFAULT '',
  payload_json TEXT NOT NULL DEFAULT '{}'
);

CREATE TABLE IF NOT EXISTS backup_bundles (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  created_at INTEGER NOT NULL,
  kind TEXT NOT NULL,
  label TEXT NOT NULL DEFAULT '',
  version_tag TEXT NOT NULL DEFAULT '',
  file_path TEXT NOT NULL,
  size_bytes INTEGER NOT NULL DEFAULT 0,
  sha256 TEXT NOT NULL DEFAULT '',
  entity_counts_json TEXT NOT NULL DEFAULT '{}',
  notes TEXT NOT NULL DEFAULT '',
  source_version TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS release_artifacts (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  created_at INTEGER NOT NULL,
  kind TEXT NOT NULL,
  version_tag TEXT NOT NULL,
  channel TEXT NOT NULL DEFAULT 'local',
  status TEXT NOT NULL DEFAULT 'prepared',
  summary TEXT NOT NULL DEFAULT '',
  checklist_json TEXT NOT NULL DEFAULT '[]',
  notes TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS model_manifests (
  id TEXT PRIMARY KEY,
  schema_version TEXT NOT NULL,
  display_name TEXT NOT NULL,
  family TEXT NOT NULL,
  format TEXT NOT NULL,
  backend TEXT NOT NULL,
  model_path TEXT NOT NULL,
  sha256 TEXT NOT NULL DEFAULT '',
  size_bytes INTEGER NOT NULL DEFAULT 0,
  quantization TEXT NOT NULL DEFAULT '',
  context_length INTEGER NOT NULL DEFAULT 0,
  capabilities_json TEXT NOT NULL DEFAULT '[]',
  default_runtime_json TEXT NOT NULL DEFAULT '{}',
  license_json TEXT NOT NULL DEFAULT '{}',
  metadata_json TEXT NOT NULL DEFAULT '{}',
  discovered_at INTEGER NOT NULL DEFAULT 0,
  updated_at INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS model_registry_status (
  model_id TEXT PRIMARY KEY REFERENCES model_manifests(id) ON DELETE CASCADE,
  backend TEXT NOT NULL,
  status TEXT NOT NULL,
  updated_at INTEGER NOT NULL DEFAULT 0,
  last_error TEXT NOT NULL DEFAULT '',
  metadata_json TEXT NOT NULL DEFAULT '{}'
);

CREATE TABLE IF NOT EXISTS model_runtime_loads (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  model_id TEXT NOT NULL REFERENCES model_manifests(id) ON DELETE CASCADE,
  backend TEXT NOT NULL,
  status TEXT NOT NULL,
  loaded_at INTEGER NOT NULL,
  unloaded_at INTEGER,
  endpoint TEXT NOT NULL DEFAULT '',
  pid INTEGER NOT NULL DEFAULT 0,
  resource_usage_json TEXT NOT NULL DEFAULT '{}',
  metadata_json TEXT NOT NULL DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_gateway_created ON gateway_invocations(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_gateway_job ON gateway_invocations(job_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_gateway_tool ON gateway_invocations(tool_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_created ON audit_records(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_correlation ON audit_records(correlation_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_audit_category ON audit_records(category, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_job ON audit_records(job_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_permission_active ON permission_profiles(active);
CREATE INDEX IF NOT EXISTS idx_action_lanes_enabled ON action_lanes(enabled, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_backup_bundles_created ON backup_bundles(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_release_artifacts_created ON release_artifacts(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_model_manifests_backend ON model_manifests(backend, id);
CREATE INDEX IF NOT EXISTS idx_model_manifests_format ON model_manifests(format, id);
CREATE INDEX IF NOT EXISTS idx_model_registry_status_status ON model_registry_status(status, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_model_runtime_loads_model ON model_runtime_loads(model_id, loaded_at DESC);
CREATE INDEX IF NOT EXISTS idx_model_runtime_loads_status ON model_runtime_loads(status, loaded_at DESC);

CREATE TABLE IF NOT EXISTS provenance_records (
  id TEXT PRIMARY KEY,
  actor TEXT NOT NULL,
  actor_type TEXT NOT NULL,
  source TEXT NOT NULL DEFAULT '',
  trace_id TEXT NOT NULL DEFAULT '',
  workspace_id TEXT NOT NULL,
  lane_id TEXT NOT NULL DEFAULT '',
  selected_paths_json TEXT NOT NULL DEFAULT '[]',
  metadata_json TEXT NOT NULL DEFAULT '{}',
  created_at INTEGER NOT NULL,
  proposed_by TEXT NOT NULL DEFAULT '',
  committed_by TEXT NOT NULL DEFAULT 'forge_kernel',
  syscall_id TEXT NOT NULL DEFAULT '',
  correlation_id TEXT NOT NULL DEFAULT '',
  audit_id TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS journal_events (
  id TEXT PRIMARY KEY,
  type TEXT NOT NULL,
  source TEXT NOT NULL,
  actor TEXT NOT NULL DEFAULT '',
  workspace_id TEXT NOT NULL,
  lane_id TEXT NOT NULL DEFAULT '',
  selected_paths_json TEXT NOT NULL DEFAULT '[]',
  payload_json TEXT NOT NULL DEFAULT '{}',
  correlation_id TEXT NOT NULL DEFAULT '',
  trace_id TEXT NOT NULL DEFAULT '',
  provenance_id TEXT REFERENCES provenance_records(id) ON DELETE SET NULL,
  provenance_json TEXT NOT NULL DEFAULT '{}',
  created_at INTEGER NOT NULL,
  metadata_json TEXT NOT NULL DEFAULT '{}',
  proposed_by TEXT NOT NULL DEFAULT '',
  committed_by TEXT NOT NULL DEFAULT 'forge_kernel',
  syscall_id TEXT NOT NULL DEFAULT '',
  audit_id TEXT NOT NULL DEFAULT ''
);

CREATE TRIGGER IF NOT EXISTS journal_events_no_update
BEFORE UPDATE ON journal_events
BEGIN
  SELECT RAISE(FAIL, 'journal_events are append-only');
END;

CREATE TRIGGER IF NOT EXISTS journal_events_no_delete
BEFORE DELETE ON journal_events
BEGIN
  SELECT RAISE(FAIL, 'journal_events are append-only');
END;

CREATE TABLE IF NOT EXISTS memory_notes (
  id TEXT PRIMARY KEY,
  type TEXT NOT NULL,
  title TEXT NOT NULL,
  content TEXT NOT NULL,
  workspace_id TEXT NOT NULL,
  lane_id TEXT NOT NULL DEFAULT '',
  selected_paths_json TEXT NOT NULL DEFAULT '[]',
  confidence REAL NOT NULL,
  status TEXT NOT NULL,
  provenance_id TEXT REFERENCES provenance_records(id) ON DELETE SET NULL,
  provenance_json TEXT NOT NULL DEFAULT '{}',
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  archived_at INTEGER,
  superseded_by TEXT,
  metadata_json TEXT NOT NULL DEFAULT '{}',
  proposed_by TEXT NOT NULL DEFAULT '',
  committed_by TEXT NOT NULL DEFAULT 'forge_kernel',
  syscall_id TEXT NOT NULL DEFAULT '',
  correlation_id TEXT NOT NULL DEFAULT '',
  trace_id TEXT NOT NULL DEFAULT '',
  audit_id TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS semantic_links (
  id TEXT PRIMARY KEY,
  type TEXT NOT NULL,
  source_id TEXT NOT NULL,
  source_kind TEXT NOT NULL DEFAULT 'object',
  target_id TEXT NOT NULL,
  target_kind TEXT NOT NULL DEFAULT 'object',
  confidence REAL NOT NULL,
  provenance_id TEXT REFERENCES provenance_records(id) ON DELETE SET NULL,
  provenance_json TEXT NOT NULL DEFAULT '{}',
  workspace_id TEXT NOT NULL,
  lane_id TEXT NOT NULL DEFAULT '',
  selected_paths_json TEXT NOT NULL DEFAULT '[]',
  created_at INTEGER NOT NULL,
  metadata_json TEXT NOT NULL DEFAULT '{}',
  proposed_by TEXT NOT NULL DEFAULT '',
  committed_by TEXT NOT NULL DEFAULT 'forge_kernel',
  syscall_id TEXT NOT NULL DEFAULT '',
  correlation_id TEXT NOT NULL DEFAULT '',
  trace_id TEXT NOT NULL DEFAULT '',
  audit_id TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS state_items (
  id TEXT PRIMARY KEY,
  key TEXT NOT NULL,
  value_json TEXT NOT NULL DEFAULT '{}',
  workspace_id TEXT NOT NULL,
  lane_id TEXT NOT NULL DEFAULT '',
  selected_paths_json TEXT NOT NULL DEFAULT '[]',
  status TEXT NOT NULL,
  derived_from_json TEXT NOT NULL DEFAULT '[]',
  current_version INTEGER NOT NULL DEFAULT 1,
  updated_at INTEGER NOT NULL,
  metadata_json TEXT NOT NULL DEFAULT '{}',
  proposed_by TEXT NOT NULL DEFAULT '',
  committed_by TEXT NOT NULL DEFAULT 'forge_kernel',
  syscall_id TEXT NOT NULL DEFAULT '',
  correlation_id TEXT NOT NULL DEFAULT '',
  trace_id TEXT NOT NULL DEFAULT '',
  audit_id TEXT NOT NULL DEFAULT '',
  UNIQUE(workspace_id, lane_id, key)
);

CREATE TABLE IF NOT EXISTS state_versions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  state_item_id TEXT NOT NULL REFERENCES state_items(id) ON DELETE RESTRICT,
  state_key TEXT NOT NULL,
  workspace_id TEXT NOT NULL,
  lane_id TEXT NOT NULL DEFAULT '',
  previous_value_json TEXT NOT NULL DEFAULT '{}',
  new_value_json TEXT NOT NULL DEFAULT '{}',
  changed_by TEXT NOT NULL DEFAULT '',
  derived_from_json TEXT NOT NULL DEFAULT '[]',
  syscall_id TEXT NOT NULL DEFAULT '',
  audit_id TEXT NOT NULL DEFAULT '',
  correlation_id TEXT NOT NULL DEFAULT '',
  trace_id TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL,
  metadata_json TEXT NOT NULL DEFAULT '{}',
  proposed_by TEXT NOT NULL DEFAULT '',
  committed_by TEXT NOT NULL DEFAULT 'forge_kernel'
);

CREATE TABLE IF NOT EXISTS open_loops (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  state TEXT NOT NULL,
  priority TEXT NOT NULL,
  owner TEXT NOT NULL,
  blocker TEXT NOT NULL DEFAULT '',
  next_action TEXT NOT NULL DEFAULT '',
  related_notes_json TEXT NOT NULL DEFAULT '[]',
  created_from TEXT NOT NULL DEFAULT '',
  workspace_id TEXT NOT NULL,
  lane_id TEXT NOT NULL DEFAULT '',
  selected_paths_json TEXT NOT NULL DEFAULT '[]',
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  resolved_at INTEGER,
  archived_at INTEGER,
  metadata_json TEXT NOT NULL DEFAULT '{}',
  proposed_by TEXT NOT NULL DEFAULT '',
  committed_by TEXT NOT NULL DEFAULT 'forge_kernel',
  syscall_id TEXT NOT NULL DEFAULT '',
  correlation_id TEXT NOT NULL DEFAULT '',
  trace_id TEXT NOT NULL DEFAULT '',
  audit_id TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS artifact_refs (
  id TEXT PRIMARY KEY,
  type TEXT NOT NULL,
  uri TEXT NOT NULL,
  content_hash TEXT NOT NULL DEFAULT '',
  workspace_id TEXT NOT NULL,
  lane_id TEXT NOT NULL DEFAULT '',
  selected_paths_json TEXT NOT NULL DEFAULT '[]',
  provenance_id TEXT REFERENCES provenance_records(id) ON DELETE SET NULL,
  provenance_json TEXT NOT NULL DEFAULT '{}',
  created_at INTEGER NOT NULL,
  metadata_json TEXT NOT NULL DEFAULT '{}',
  proposed_by TEXT NOT NULL DEFAULT '',
  committed_by TEXT NOT NULL DEFAULT 'forge_kernel',
  syscall_id TEXT NOT NULL DEFAULT '',
  correlation_id TEXT NOT NULL DEFAULT '',
  trace_id TEXT NOT NULL DEFAULT '',
  audit_id TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS derived_models (
  id TEXT PRIMARY KEY,
  type TEXT NOT NULL,
  expression_json TEXT NOT NULL DEFAULT '{}',
  derived_from_json TEXT NOT NULL DEFAULT '[]',
  support_count INTEGER NOT NULL DEFAULT 0,
  confidence REAL NOT NULL DEFAULT 0,
  status TEXT NOT NULL,
  workspace_id TEXT NOT NULL,
  lane_id TEXT NOT NULL DEFAULT '',
  selected_paths_json TEXT NOT NULL DEFAULT '[]',
  last_validated_at INTEGER,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  metadata_json TEXT NOT NULL DEFAULT '{}',
  proposed_by TEXT NOT NULL DEFAULT '',
  committed_by TEXT NOT NULL DEFAULT 'forge_kernel',
  syscall_id TEXT NOT NULL DEFAULT '',
  correlation_id TEXT NOT NULL DEFAULT '',
  trace_id TEXT NOT NULL DEFAULT '',
  audit_id TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS contradiction_records (
  id TEXT PRIMARY KEY,
  left_object_id TEXT NOT NULL,
  left_object_kind TEXT NOT NULL DEFAULT 'object',
  right_object_id TEXT NOT NULL,
  right_object_kind TEXT NOT NULL DEFAULT 'object',
  reason TEXT NOT NULL,
  severity TEXT NOT NULL DEFAULT 'medium',
  confidence REAL NOT NULL DEFAULT 0.5,
  provenance_id TEXT REFERENCES provenance_records(id) ON DELETE SET NULL,
  provenance_json TEXT NOT NULL DEFAULT '{}',
  workspace_id TEXT NOT NULL,
  lane_id TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL,
  metadata_json TEXT NOT NULL DEFAULT '{}',
  proposed_by TEXT NOT NULL DEFAULT '',
  committed_by TEXT NOT NULL DEFAULT 'forge_kernel',
  syscall_id TEXT NOT NULL DEFAULT '',
  correlation_id TEXT NOT NULL DEFAULT '',
  trace_id TEXT NOT NULL DEFAULT '',
  audit_id TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS supersession_records (
  id TEXT PRIMARY KEY,
  old_object_id TEXT NOT NULL,
  old_object_kind TEXT NOT NULL DEFAULT 'object',
  new_object_id TEXT NOT NULL,
  new_object_kind TEXT NOT NULL DEFAULT 'object',
  reason TEXT NOT NULL,
  provenance_id TEXT REFERENCES provenance_records(id) ON DELETE SET NULL,
  provenance_json TEXT NOT NULL DEFAULT '{}',
  workspace_id TEXT NOT NULL,
  lane_id TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL,
  metadata_json TEXT NOT NULL DEFAULT '{}',
  proposed_by TEXT NOT NULL DEFAULT '',
  committed_by TEXT NOT NULL DEFAULT 'forge_kernel',
  syscall_id TEXT NOT NULL DEFAULT '',
  correlation_id TEXT NOT NULL DEFAULT '',
  trace_id TEXT NOT NULL DEFAULT '',
  audit_id TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS context_packet_snapshots (
  id TEXT PRIMARY KEY,
  query TEXT NOT NULL,
  workspace_id TEXT NOT NULL,
  lane_id TEXT NOT NULL DEFAULT '',
  snapshot_kind TEXT NOT NULL DEFAULT '',
  snapshot_fingerprint TEXT NOT NULL DEFAULT '',
  parent_snapshot_id TEXT NOT NULL DEFAULT '',
  selected_paths_json TEXT NOT NULL DEFAULT '[]',
  included_state_json TEXT NOT NULL DEFAULT '[]',
  included_open_loops_json TEXT NOT NULL DEFAULT '[]',
  included_notes_json TEXT NOT NULL DEFAULT '[]',
  included_links_json TEXT NOT NULL DEFAULT '[]',
  included_models_json TEXT NOT NULL DEFAULT '[]',
  included_artifacts_json TEXT NOT NULL DEFAULT '[]',
  included_events_json TEXT NOT NULL DEFAULT '[]',
  header_json TEXT NOT NULL DEFAULT '{}',
  graph_json TEXT NOT NULL DEFAULT '{}',
  delta_json TEXT NOT NULL DEFAULT '{}',
  restore_scores_json TEXT NOT NULL DEFAULT '{}',
  render_artifact_ref_id TEXT NOT NULL DEFAULT '',
  resume_hints_json TEXT NOT NULL DEFAULT '{}',
  budget_json TEXT NOT NULL DEFAULT '{}',
  inclusion_reasons_json TEXT NOT NULL DEFAULT '{}',
  created_at INTEGER NOT NULL,
  correlation_id TEXT NOT NULL DEFAULT '',
  trace_id TEXT NOT NULL DEFAULT '',
  syscall_id TEXT NOT NULL DEFAULT '',
  metadata_json TEXT NOT NULL DEFAULT '{}',
  proposed_by TEXT NOT NULL DEFAULT '',
  committed_by TEXT NOT NULL DEFAULT 'forge_kernel',
  audit_id TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS dream_reports (
  id TEXT PRIMARY KEY,
  created_at INTEGER NOT NULL,
  completed_at INTEGER NOT NULL,
  workspace_id TEXT NOT NULL,
  lane_id TEXT NOT NULL DEFAULT '',
  mode TEXT NOT NULL,
  dry_run INTEGER NOT NULL DEFAULT 1,
  status TEXT NOT NULL,
  time_window_start INTEGER NOT NULL,
  time_window_end INTEGER NOT NULL,
  candidates_considered INTEGER NOT NULL DEFAULT 0,
  proposals_generated INTEGER NOT NULL DEFAULT 0,
  summary_json TEXT NOT NULL DEFAULT '{}',
  candidates_json TEXT NOT NULL DEFAULT '[]',
  salience_scores_json TEXT NOT NULL DEFAULT '[]',
  memory_tier_proposals_json TEXT NOT NULL DEFAULT '[]',
  repair_proposals_json TEXT NOT NULL DEFAULT '[]',
  snapshot_hygiene_proposals_json TEXT NOT NULL DEFAULT '[]',
  warnings_json TEXT NOT NULL DEFAULT '[]',
  trace_json TEXT NOT NULL DEFAULT '{}',
  correlation_id TEXT NOT NULL DEFAULT '',
  trace_id TEXT NOT NULL DEFAULT '',
  syscall_id TEXT,
  audit_id TEXT,
  proposed_by TEXT NOT NULL DEFAULT 'forge.dream',
  committed_by TEXT NOT NULL DEFAULT '',
  metadata_json TEXT NOT NULL DEFAULT '{}'
);

CREATE TABLE IF NOT EXISTS semantic_idempotency_keys (
  idempotency_key TEXT PRIMARY KEY,
  action TEXT NOT NULL,
  result_json TEXT NOT NULL,
  created_at INTEGER NOT NULL DEFAULT 0,
  correlation_id TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_provenance_workspace_created ON provenance_records(workspace_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_journal_workspace_created ON journal_events(workspace_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_journal_corr ON journal_events(correlation_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_journal_trace ON journal_events(trace_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notes_workspace_status ON memory_notes(workspace_id, status, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_notes_workspace_type ON memory_notes(workspace_id, type, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_notes_trace ON memory_notes(trace_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_links_workspace_type ON semantic_links(workspace_id, type, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_links_source ON semantic_links(source_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_links_target ON semantic_links(target_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_state_workspace_key ON state_items(workspace_id, key, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_state_versions_key ON state_versions(workspace_id, state_key, id DESC);
CREATE INDEX IF NOT EXISTS idx_state_versions_corr ON state_versions(correlation_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_loops_workspace_state ON open_loops(workspace_id, state, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_loops_workspace_priority ON open_loops(workspace_id, priority, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_artifact_refs_workspace ON artifact_refs(workspace_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_artifact_refs_hash ON artifact_refs(content_hash, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_models_workspace_status ON derived_models(workspace_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_models_workspace_type ON derived_models(workspace_id, type, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_contradiction_scope ON contradiction_records(workspace_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_contradiction_left ON contradiction_records(left_object_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_contradiction_right ON contradiction_records(right_object_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_supersession_scope ON supersession_records(workspace_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_supersession_old ON supersession_records(old_object_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_supersession_new ON supersession_records(new_object_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ctx_snapshot_scope ON context_packet_snapshots(workspace_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ctx_snapshot_corr ON context_packet_snapshots(correlation_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_dream_reports_scope ON dream_reports(workspace_id, lane_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_dream_reports_mode ON dream_reports(workspace_id, mode, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_dream_reports_trace ON dream_reports(correlation_id, trace_id);

CREATE TABLE IF NOT EXISTS chat_threads (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  title TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  dossier_id INTEGER REFERENCES dossiers(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS chat_messages (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  thread_id INTEGER NOT NULL REFERENCES chat_threads(id) ON DELETE CASCADE,
  role TEXT NOT NULL,
  content TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  metadata_json TEXT NOT NULL DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_chat_messages_thread ON chat_messages(thread_id, id ASC);
CREATE INDEX IF NOT EXISTS idx_chat_threads_updated ON chat_threads(updated_at DESC);

CREATE TABLE IF NOT EXISTS canvas_boards (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  title TEXT NOT NULL,
  dossier_id INTEGER REFERENCES dossiers(id) ON DELETE SET NULL,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS canvas_notes (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  board_id INTEGER NOT NULL REFERENCES canvas_boards(id) ON DELETE CASCADE,
  title TEXT NOT NULL DEFAULT '',
  body TEXT NOT NULL DEFAULT '',
  x REAL NOT NULL DEFAULT 0,
  y REAL NOT NULL DEFAULT 0,
  width REAL NOT NULL DEFAULT 260,
  height REAL NOT NULL DEFAULT 180,
  pinned INTEGER NOT NULL DEFAULT 0,
  color TEXT NOT NULL DEFAULT '',
  links_json TEXT NOT NULL DEFAULT '[]',
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_canvas_notes_board ON canvas_notes(board_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_canvas_boards_updated ON canvas_boards(updated_at DESC);

-- P0 safety: persisted tool capability status overrides. Registry loads these
-- on startup so operator UpdateStatus calls survive restart.
CREATE TABLE IF NOT EXISTS tool_capability_overrides (
  capability_id TEXT PRIMARY KEY,
  status TEXT NOT NULL,
  reason TEXT NOT NULL DEFAULT '',
  actor TEXT NOT NULL DEFAULT 'operator',
  updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS secrets_vault (
  name TEXT PRIMARY KEY,
  ciphertext TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  owner TEXT NOT NULL DEFAULT 'gateway'
);

CREATE TABLE IF NOT EXISTS identity_tokens (
  token_id TEXT PRIMARY KEY,
  token_hash TEXT NOT NULL,
  status TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  revoked_at INTEGER NOT NULL DEFAULT 0,
  owner TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS feature_flags (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL,
  updated_at INTEGER NOT NULL,
  actor TEXT NOT NULL DEFAULT 'operator'
);

CREATE TABLE IF NOT EXISTS alert_rules (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE,
  expression TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'active',
  silenced_until INTEGER NOT NULL DEFAULT 0,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS scheduled_tasks (
  id TEXT PRIMARY KEY,
  kind TEXT NOT NULL,
  payload_json TEXT NOT NULL DEFAULT '{}',
  status TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

-- P0 safety: audit immutability. The triggers below block any UPDATE or DELETE
-- on audit_records so a compromised process with DB access cannot rewrite
-- history. Test fixtures that need to wipe audit state should drop the table
-- and re-run migrate.
CREATE TRIGGER IF NOT EXISTS audit_records_block_update
BEFORE UPDATE ON audit_records
BEGIN
  SELECT RAISE(ABORT, 'audit_records is append-only');
END;
CREATE TRIGGER IF NOT EXISTS audit_records_block_delete
BEFORE DELETE ON audit_records
BEGIN
  SELECT RAISE(ABORT, 'audit_records is append-only');
END;

CREATE INDEX IF NOT EXISTS idx_tool_capability_overrides_updated ON tool_capability_overrides(updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_identity_tokens_status ON identity_tokens(status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_alert_rules_status ON alert_rules(status, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_scheduled_tasks_status ON scheduled_tasks(status, updated_at DESC);
`

func migrate(db *sql.DB) error {
	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	if err := ensureContextPacketSnapshotColumns(db); err != nil {
		return fmt.Errorf("migrate context_packet_snapshots: %w", err)
	}
	if err := ensureApprovalRequestExpiryColumns(db); err != nil {
		return fmt.Errorf("migrate approval_requests expiry: %w", err)
	}
	return nil
}

// ensureApprovalRequestExpiryColumns adds expires_at and auto-expired metadata
// to approval_requests for existing DBs so P0 safety fixes apply without a wipe.
func ensureApprovalRequestExpiryColumns(db *sql.DB) error {
	rows, err := db.Query(`PRAGMA table_info(approval_requests)`)
	if err != nil {
		return err
	}
	defer rows.Close()
	existing := make(map[string]struct{})
	for rows.Next() {
		var cid int
		var name string
		var ctype string
		var notnull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return err
		}
		existing[name] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	additions := []struct{ name, ddl string }{
		{name: "expires_at", ddl: "INTEGER NOT NULL DEFAULT 0"},
		{name: "expired_at", ddl: "INTEGER NOT NULL DEFAULT 0"},
	}
	for _, col := range additions {
		if _, ok := existing[col.name]; ok {
			continue
		}
		if _, err := db.Exec(fmt.Sprintf(
			"ALTER TABLE approval_requests ADD COLUMN %s %s",
			col.name, col.ddl,
		)); err != nil {
			return err
		}
	}
	// Backfill expires_at for legacy pending rows so the reaper has something to
	// act on; default TTL is 24 hours from row creation.
	if _, err := db.Exec(`
UPDATE approval_requests
SET expires_at = created_at + (24 * 3600 * 1000)
WHERE expires_at = 0`); err != nil {
		return err
	}
	return nil
}

func ensureContextPacketSnapshotColumns(db *sql.DB) error {
	rows, err := db.Query(`PRAGMA table_info(context_packet_snapshots)`)
	if err != nil {
		return err
	}
	defer rows.Close()

	existing := make(map[string]struct{})
	for rows.Next() {
		var cid, notNull, pk int
		var name, colType string
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &colType, &notNull, &defaultValue, &pk); err != nil {
			return err
		}
		existing[name] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	additions := []struct {
		name string
		ddl  string
	}{
		{name: "snapshot_kind", ddl: "TEXT NOT NULL DEFAULT ''"},
		{name: "snapshot_fingerprint", ddl: "TEXT NOT NULL DEFAULT ''"},
		{name: "parent_snapshot_id", ddl: "TEXT NOT NULL DEFAULT ''"},
		{name: "header_json", ddl: "TEXT NOT NULL DEFAULT '{}'"},
		{name: "graph_json", ddl: "TEXT NOT NULL DEFAULT '{}'"},
		{name: "delta_json", ddl: "TEXT NOT NULL DEFAULT '{}'"},
		{name: "restore_scores_json", ddl: "TEXT NOT NULL DEFAULT '{}'"},
		{name: "render_artifact_ref_id", ddl: "TEXT NOT NULL DEFAULT ''"},
		{name: "resume_hints_json", ddl: "TEXT NOT NULL DEFAULT '{}'"},
	}

	for _, col := range additions {
		if _, ok := existing[col.name]; ok {
			continue
		}
		if _, err := db.Exec(fmt.Sprintf(
			"ALTER TABLE context_packet_snapshots ADD COLUMN %s %s",
			col.name, col.ddl,
		)); err != nil {
			return err
		}
	}

	_, err = db.Exec(`
UPDATE context_packet_snapshots
SET snapshot_kind = COALESCE(snapshot_kind, ''),
    snapshot_fingerprint = COALESCE(snapshot_fingerprint, ''),
    parent_snapshot_id = COALESCE(parent_snapshot_id, ''),
    header_json = COALESCE(header_json, '{}'),
    graph_json = COALESCE(graph_json, '{}'),
    delta_json = COALESCE(delta_json, '{}'),
    restore_scores_json = COALESCE(restore_scores_json, '{}'),
    render_artifact_ref_id = COALESCE(render_artifact_ref_id, ''),
    resume_hints_json = COALESCE(resume_hints_json, '{}')`)
	return err
}
