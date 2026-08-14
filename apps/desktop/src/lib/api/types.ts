/** Persisted chat thread (SQLite). */
export type ChatThreadSummary = {
  id: number;
  title: string;
  createdAtMs: number;
  updatedAtMs: number;
  dossierId?: number;
};

export type ChatMessage = {
  id: number;
  threadId: number;
  role: string;
  content: string;
  createdAtMs: number;
  metadata: Record<string, unknown>;
};

export type ChatAttachment = {
  artifactId: number;
  title: string;
  mimeType: string;
  fileName: string;
  textPreview?: string;
};

export type RemoteTelegramPayload = {
  message?: {
    message_id?: number;
    text?: string;
    caption?: string;
    chat?: { id?: number };
    from?: { id?: number };
  };
  edited_message?: {
    message_id?: number;
    text?: string;
    caption?: string;
    chat?: { id?: number };
    from?: { id?: number };
  };
  channel_post?: {
    message_id?: number;
    text?: string;
    caption?: string;
    chat?: { id?: number };
    from?: { id?: number };
  };
};

export type RemoteDiscordPayload = {
  id?: string;
  channel_id?: string;
  content?: string;
  author?: { id?: string };
};

/** Persisted assistant metadata: governed tool activity from chat (see core chat_assistant_gateway). */
export type ChatToolGatewayActivity = {
  userRequestSummary?: string;
  toolManifest?: unknown;
  stages?: unknown;
  toolCallsExecuted?: number;
  toolCallEmitted?: boolean;
  toolSelected?: string;
  toolArgs?: Record<string, unknown>;
  executionState?: string;
  executionResult?: unknown;
  failureReason?: string;
};

export type ChatThreadDetail = ChatThreadSummary & { messages: ChatMessage[] };

export type CanvasBoard = {
  id: number;
  title: string;
  dossierId?: number;
  createdAtMs: number;
  updatedAtMs: number;
};

export type CanvasNote = {
  id: number;
  boardId: number;
  title: string;
  body: string;
  x: number;
  y: number;
  width: number;
  height: number;
  pinned: boolean;
  color: string;
  links: Array<Record<string, unknown>>;
  createdAtMs: number;
  updatedAtMs: number;
};

export type CanvasBoardDetail = CanvasBoard & { notes: CanvasNote[] };

export type ForgeArtifact = {
  id: number;
  createdAtMs: number;
  jobId?: string;
  packetId?: number;
  type: string;
  title: string;
  filePath: string;
  mimeType: string;
  metadata: unknown;
};

export type ContextSnapshotInspectorCounts = {
  state: number;
  openLoops: number;
  notes: number;
  links: number;
  models: number;
  artifacts: number;
  events: number;
};

export type ContextSnapshotInspectorSummary = {
  id: string;
  query: string;
  workspaceId: string;
  laneId: string;
  selectedPaths: string[];
  snapshotKind: string;
  snapshotFingerprint: string;
  parentSnapshotId: string;
  renderArtifactRefId: string;
  createdAtMs: number;
  correlationId: string;
  traceId: string;
  syscallId: string;
  auditId: string;
  proposedBy: string;
  committedBy: string;
  counts: ContextSnapshotInspectorCounts;
  hasHeader: boolean;
  hasGraph: boolean;
  hasDelta: boolean;
  hasRestoreScores: boolean;
  hasResumeHints: boolean;
  hasRestoreTrace?: boolean;
  restoreTrace?: Record<string, unknown>;
  evidenceClass?: string;
  nonCanonicalEvidence?: boolean;
};

export type ContextSnapshotInspectorDetail = {
  summary: ContextSnapshotInspectorSummary;
  budget: Record<string, unknown>;
  inclusionReasons: Record<string, string>;
  header: Record<string, unknown>;
  graph: Record<string, unknown>;
  delta: Record<string, unknown>;
  restoreScores: Record<string, unknown>;
  resumeHints: Record<string, unknown>;
  restoreTrace?: Record<string, unknown>;
  restorePackage?: Record<string, unknown>;
  metadata: Record<string, unknown>;
  includedStateIds: string[];
  includedOpenLoops: string[];
  includedNoteIds: string[];
  includedLinkIds: string[];
  includedModelIds: string[];
  includedArtifactIds: string[];
  includedEventIds: string[];
};

export type RestoreCandidateScoreView = Record<string, unknown> & {
  snapshotId?: string;
  snapshot_id?: string;
  total?: number;
  explain?: string[];
  selected?: boolean;
};

export type RestorePackageView = Record<string, unknown>;

export type ResumeHintsView = Record<string, unknown> & {
  requiresFreshCompile?: boolean;
  requires_fresh_compile?: boolean;
  nextAction?: string;
  next_action?: string;
};

export type RestoreInspectorScoreResponse = {
  snapshotId: string;
  score: Record<string, unknown>;
  scoreBreakdown: RestoreCandidateScoreView[];
  restorePackage: RestorePackageView;
  resumeHints: ResumeHintsView;
  requiresFreshCompile: boolean;
  requiresFreshCompileReason: string;
  renderArtifactRefId?: string;
  evidenceClass: string;
  nonCanonicalEvidence: boolean;
  canonicalWriteCommitted: boolean;
};

export type RestoreInspectorCandidatesResponse = {
  snapshotId: string;
  candidates: RestoreCandidateScoreView[];
  score: Record<string, unknown>;
  evidenceClass: string;
  nonCanonicalEvidence: boolean;
  canonicalWriteCommitted: boolean;
};

export type RestoreInspectorResumeHintsResponse = {
  snapshotId: string;
  resumeHints: ResumeHintsView;
  requiresFreshCompile: boolean;
  requiresFreshCompileReason: string;
  evidenceClass: string;
  nonCanonicalEvidence: boolean;
  canonicalWriteCommitted: boolean;
};

export type DreamReplayCandidateView = Record<string, unknown> & {
  id?: string;
  type?: string;
  reason?: string;
};

export type DreamSalienceScoreView = Record<string, unknown> & {
  id?: string;
  total?: number;
  score?: number;
};

export type DreamMemoryTierProposalView = Record<string, unknown> & {
  subjectId?: string;
  subject_id?: string;
  decision?: string;
};

export type OperatorReviewItem = Record<string, unknown> & {
  subjectId?: string;
  subject_id?: string;
  decision?: string;
  reason?: string;
};

export type DreamReportSummary = {
  id: string;
  createdAt: number;
  completedAt: number;
  workspaceId: string;
  laneId: string;
  mode: string;
  dryRun: boolean;
  status: string;
  candidatesConsidered: number;
  proposalsGenerated: number;
  warnings: unknown[];
  correlationId: string;
  traceId: string;
  evidenceClass?: string;
  nonCanonicalEvidence?: boolean;
  canonicalWriteCommitted?: boolean;
};

export type DreamReportDetail = DreamReportSummary & {
  summary: Record<string, unknown>;
  candidates: DreamReplayCandidateView[];
  salienceScores: DreamSalienceScoreView[];
  memoryTierProposals: DreamMemoryTierProposalView[];
  repairProposals: OperatorReviewItem[];
  snapshotHygieneProposals: OperatorReviewItem[];
  trace: Record<string, unknown>;
  metadata: Record<string, unknown>;
};

export type DreamReportCandidatesResponse = {
  reportId: string;
  candidates: DreamReplayCandidateView[];
  salienceScores: DreamSalienceScoreView[];
  evidenceClass: string;
  nonCanonicalEvidence: boolean;
  dryRun: boolean;
  canonicalWriteCommitted: boolean;
};

export type DreamReportProposalsResponse = {
  reportId: string;
  memoryTierProposals: DreamMemoryTierProposalView[];
  repairProposals: OperatorReviewItem[];
  snapshotHygieneProposals: OperatorReviewItem[];
  reviewItems: OperatorReviewItem[];
  evidenceClass: string;
  nonCanonicalEvidence: boolean;
  dryRun: boolean;
  canonicalWriteCommitted: boolean;
};

export type DreamReportWarningsResponse = {
  reportId: string;
  warnings: unknown[];
  reviewItems: OperatorReviewItem[];
  evidenceClass: string;
  nonCanonicalEvidence: boolean;
  dryRun: boolean;
  canonicalWriteCommitted: boolean;
};

export type AuditTraceLookupReport = {
  correlationId: string;
  records: unknown[];
  report: Record<string, unknown>;
};

export type AuditTraceLookupResponse = {
  mode: "correlation" | "trace";
  correlationId?: string;
  traceId?: string;
  records?: unknown[];
  report?: Record<string, unknown>;
  correlationIds?: string[];
  reports: AuditTraceLookupReport[];
};

export type ProcessHealthInvocation = {
  correlationId: string;
  invocationId: number;
  toolId: string;
  action: string;
  domain: string;
  laneId?: string;
  initiator: string;
  status: string;
  policyOutcome: string;
  riskClass: string;
  writeIntent: boolean;
  deniedReason?: string;
  startedAtMs: number;
  completedAtMs?: number;
  durationMs?: number;
  traceId?: string;
};

export type ProcessHealthCorrelationReport = {
  correlationId: string;
  processInvocations: ProcessHealthInvocation[];
  totalInvocations: number;
  processInvocationCount: number;
};

export type ProcessHealthRuntime = {
  available: boolean;
  state?: string;
  safeMode?: boolean;
  safeModeReasons?: string[];
  runtimeEnabled?: boolean;
  gpuAware?: boolean;
  health?: Record<string, unknown>;
  queue?: Record<string, unknown>;
  loaded?: Record<string, unknown>;
  usage?: Record<string, unknown>;
  error?: string;
  warnings?: string[];
};

export type ProcessHealthTraceResponse = {
  correlationIds: string[];
  correlationId?: string;
  traceId?: string;
  reports: ProcessHealthCorrelationReport[];
  runtime: ProcessHealthRuntime;
};

export type AutonomyScope = {
  workspaceId: string;
  laneId?: string;
};

export type AutonomyDreamStatus = {
  active: boolean;
  enteredAt?: number;
  lastReason?: string;
  lastTickAt?: number;
  lastMaintenanceAt?: number;
  lastImprovementAt?: number;
  lastError?: string;
  lastTransitionType?: string;
};

export type AutonomyStatusSnapshot = {
  available: boolean;
  reason?: string;
  scope?: AutonomyScope;
  mode?: string;
  dream?: AutonomyDreamStatus;
  counts?: {
    activeIntents?: number;
    activeCharters?: number;
    budgets?: number;
    recentDecisions?: number;
  };
};

export type AutonomyIntentRecord = {
  id: string;
  type: string;
  title: string;
  description?: string;
  source?: string;
  proposedBy?: string;
  status: string;
  risk?: string;
  autonomyLevel?: string;
  charterId?: string;
  budgetId?: string;
  requiredApproval?: boolean;
  blockedReasons?: string[];
  evidence?: string[];
  correlationId?: string;
  traceId?: string;
  createdAt?: number;
  updatedAt?: number;
  scope?: AutonomyScope;
  metadata?: Record<string, unknown>;
};

export type AutonomyDecisionRecord = {
  id: string;
  intentId: string;
  decision: string;
  risk?: string;
  autonomyLevel?: string;
  charterId?: string;
  budgetId?: string;
  requiredApprovalReason?: string;
  deniedReasons?: string[];
  warnings?: string[];
  createdAt?: number;
  correlationId?: string;
  traceId?: string;
};

export type AutonomyBudgetRecord = {
  id: string;
  name: string;
  status: string;
  period: string;
  scope?: AutonomyScope;
  usage?: Record<string, number>;
  resetsAt?: number;
  updatedAt?: number;
};

export type AutonomyCharterRecord = {
  id: string;
  name: string;
  status: string;
  purpose?: string;
  description?: string;
  freedomBudgetId?: string;
  scope?: AutonomyScope;
  allowedActions?: string[];
  deniedActions?: string[];
  requiresApprovalActions?: string[];
  allowedTools?: string[];
  deniedTools?: string[];
  requiresApprovalTools?: string[];
  createdAt?: number;
  updatedAt?: number;
};

export type SettingsRecord = {
  extensionsCsv: string;
  theme: string;
  ollamaBaseUrl: string;
  ollamaModel: string;
  embeddingProvider: string;
  embeddingModel: string;
  embeddingDims: string;
  retrievalWeightKeyword: string;
  retrievalWeightSemantic: string;
  retrievalVSAMode?: string;
  retrievalVSADims?: string;
  retrievalVSASeed?: string;
  retrievalVSAWeightAssociative?: string;
  retrievalVSAWeightRoleMatch?: string;
  retrievalVSAWeightRelational?: string;
  retrievalVSAWeightFeedback?: string;
  retrievalVSAMaxAdditive?: string;
  chatPersonalityPrompt: string;
  chatPromptDefault: string;
  remoteAccessEnabled: boolean;
  remoteAccessToken: string;
  remoteAccessTokenConfigured?: boolean;
  remoteCrossChatContext: boolean;
  remoteDefaultThreadId: string;
  telegramBotToken: string;
  telegramBotTokenConfigured?: boolean;
  telegramDefaultChatId: string;
  discordBotToken: string;
  discordBotTokenConfigured?: boolean;
  discordDefaultChannelId: string;
  discordWebhookUrl: string;
  discordWebhookUrlConfigured?: boolean;
  discordCrossChatContext: boolean;
  dreamMode?: {
    enabled: boolean;
    defaultDryRun: boolean;
    mode: string;
    windowHours: string;
    maxCandidates: string;
    allowLongTermPromotion: boolean;
    requireOperatorReviewForLongTerm: boolean;
    allowCommits: boolean;
  };
  runtimeControls?: {
    gpuEnabled: boolean;
    nvidiaDcgmEnabled: boolean;
    intelLevelZeroEnabled: boolean;
    allowOllamaCloudModels: boolean;
    safeModeForceCpuOnly?: boolean;
    effectiveGpuEnabled?: boolean;
    cloudModelsDefaultState?: string;
  };
  shadowMode?: {
    enabled: boolean;
    chatMetadataEnabled?: boolean;
    retrievalMetadataEnabled?: boolean;
  };
};

export type TelegramStatusResponse = {
  remoteAccessEnabled: boolean;
  tokenConfigured: boolean;
  defaultChatId: string;
  crossChatContext: boolean;
  ready: boolean;
  reason?: string;
  bot?: {
    id: number;
    username: string;
    firstName: string;
  };
  webhook?: {
    url: string;
    has_custom_certificate: boolean;
    pending_update_count: number;
    last_error_date: number;
    last_error_message: string;
    max_connections: number;
    ip_address: string;
  };
  webhookError?: string;
};

export type DiscordGatewayStatusSnapshot = {
  enabled: boolean;
  connected?: boolean;
  applicationId?: string;
  guildId?: string;
  commandPrefix?: string;
  enableSlash?: boolean;
  enableText?: boolean;
  enablePassive?: boolean;
  enableOutbound?: boolean;
  crossChatContext?: boolean;
  registeredCommands?: string[];
  lastError?: string;
  startedAtMs?: number;
  lastInboundAtMs?: number;
  lastOutboundAtMs?: number;
  inboundCount?: number;
  outboundCount?: number;
};

export type DiscordGatewayStatusResponse = {
  enabled: boolean;
  status: "disabled" | DiscordGatewayStatusSnapshot;
  reason?: string;
};

export type ModelRuntimeModel = {
  id: string;
  displayName?: string;
  family?: string;
  backend?: string;
  format?: string;
  status?: string;
  capabilities?: string[];
  metadata?: Record<string, unknown>;
};

export type ModelRuntimeImportResult = {
  model: ModelRuntimeModel;
  duplicate: boolean;
  managedPath?: string;
  sourcePath?: string;
  warnings?: string[];
};

export type ModelRuntimeLoadResult = {
  modelId: string;
  backend?: string;
  status?: string;
  loaded: boolean;
  metadata?: Record<string, unknown>;
  warnings?: string[];
  loadedAtMs?: number;
  details?: Record<string, string>;
};

export type ModelRuntimeManagementRequest = {
  actor?: string;
  source?: string;
  workspaceId?: string;
  laneId?: string;
  capabilityId?: string;
  approvalId?: string;
  dryRun?: boolean;
  preferred?: boolean;
  metadata?: Record<string, unknown>;
};

export type ModelRuntimeCompatibility = {
  modelId: string;
  backend?: string;
  status?: string;
  loaded: boolean;
  backendConfigured: boolean;
  backendHealthy: boolean;
  supportedByBackend: boolean;
  canGenerate: boolean;
  preferred?: boolean;
  warnings?: string[];
  details?: Record<string, unknown>;
};

export type ModelRuntimeHealth = {
  ok: boolean;
  status?: string;
  backend?: string;
  runtimeEnabled?: boolean;
  gpuAware?: boolean;
  degradedReasons?: string[];
  policyWarnings?: string[];
  details?: Record<string, unknown>;
};

export type ModelRuntimeQueueStatus = {
  depth: number;
  active?: Record<string, string>;
  pending?: string[];
  scheduler?: string;
  policyState?: string;
};

export type ModelRuntimeLoadedModel = {
  modelId: string;
  backend?: string;
  status?: string;
  loadedAtMs?: number;
  metadata?: Record<string, unknown>;
};

export type ModelRuntimeLoadedStatus = {
  count: number;
  models: ModelRuntimeLoadedModel[];
};

export type ModelRuntimeBackendStatus = {
  kind: string;
  name: string;
  healthy: boolean;
  detail?: string;
  loadedModel?: string;
  meta?: Record<string, unknown>;
};

export type ModelRuntimeUsageSummary = {
  registered: number;
  imported: number;
  verified: number;
  available: number;
  disabled: number;
  archived: number;
  loaded: number;
  queueDepth: number;
  running: number;
  completed: number;
  backends?: Record<string, Record<string, unknown>>;
};

export type ForgeHealth = {
  ok: boolean;
  service: string;
  modelRuntime?: {
    available?: boolean;
    status?: string;
    [key: string]: unknown;
  };
  [key: string]: unknown;
};

export type ForgeSystemStatus = {
  generated_at?: string;
  core: {
    reachable: boolean;
    service?: string;
    health_state?: string;
    core_url?: string;
    last_refresh_at?: string;
  };
  shell_session: {
    shell_mode?: string;
    display_backend?: string;
    compositor_session?: string;
    safe_mode?: boolean;
    host_mutation_disabled?: boolean;
    model_mutation_disabled?: boolean;
    semantic_memory_write_disabled?: boolean;
    forge_k_live_authority_disabled?: boolean;
    context_compiler_required_for_llm?: boolean;
  };
  hostbridge: {
    wired: boolean;
    reason?: string;
    snapshot_id?: string;
    captured_at?: string;
    host_identity?: string;
    architecture?: string;
    ram_pressure?: string;
    disk_pressure?: string;
    gpu_available?: boolean;
    thermal_available?: boolean;
    source_errors_count?: number;
    degraded?: boolean;
    cache?: {
      available?: boolean;
      cache_hit?: boolean;
      stale?: boolean;
      age_ms?: number;
      source_error?: string;
      read_only?: boolean;
      advisory_only?: boolean;
    };
  };
  forgeh: {
    wired: boolean;
    policy?: {
      policy_id?: string;
      overall_posture?: string;
      ram_pressure?: string;
      swap_pressure?: string;
      disk_pressure?: string;
      vram_pressure?: string;
      thermal_pressure?: string;
      model_load_recommendation?: string;
      background_work_recommendation?: string;
      warnings?: string[];
      advisory_only?: boolean;
    };
    proposals?: Array<{
      proposal_id?: string;
      action_type?: string;
      target_lane?: string;
      risk_level?: string;
      status?: string;
      expires_at?: string;
      advisory_only?: boolean;
    }>;
    executions?: {
      available?: boolean;
      reason?: string;
      items?: Array<{
        execution_id?: string;
        proposal_id?: string;
        action_type?: string;
        status?: string;
        result?: string;
        bounded?: boolean;
        host_mutation?: boolean;
        semantic_memory_write?: boolean;
        modelruntime_mutation?: boolean;
        side_effects?: string[];
      }>;
    };
    advisory_only?: boolean;
    canonical_write_committed?: boolean;
    cache?: {
      available?: boolean;
      cache_hit?: boolean;
      stale?: boolean;
      age_ms?: number;
      source_error?: string;
      read_only?: boolean;
      advisory_only?: boolean;
    };
  };
  kernel_activation?: {
    generated_at?: string;
    phase?: string;
    status?: string;
    summary?: string;
    mode?: string;
    live_owner?: string;
    policy_version?: string;
    kernel_runtime_state?: string;
    closed_validation_lanes?: number;
    total_validation_lanes?: number;
    validation_actions?: Array<{
      action?: string;
      capability?: string;
      registered?: boolean;
      mutating?: boolean;
      approval_possible?: boolean;
      supports_dry_run?: boolean;
      audit_event_name?: string;
      closed?: boolean;
      mode?: string;
      live_owner?: string;
      simulator_authority?: boolean;
      live_kernel_authority?: boolean;
    }>;
    authority_ready_gates?: number;
    authority_blocked_gates?: number;
    authority_gates?: Array<{
      name?: string;
      status?: string;
      live_owner?: string;
      required_for_live_authority?: boolean;
      mutation_authority?: boolean;
      reason?: string;
      next_step?: string;
    }>;
    authority_matrix?: Array<{
      subsystem?: string;
      current_status?: string;
      live_owner?: string;
      target_owner?: string;
      feature_flag?: string;
      rollback_path?: string;
      tests_required?: string[];
      tests_passing?: string[];
      blockers?: string[];
      operator_visible?: boolean;
    }>;
    gates?: Array<{
      name?: string;
      passed?: boolean;
      reason?: string;
    }>;
    no_effect?: Record<string, boolean>;
    simulator_authority?: boolean;
    live_kernel_ingress_authority?: boolean;
    live_durable_orchestration?: boolean;
    live_kernel_authority?: boolean;
    live_authority_migration?: boolean;
    shadow_authoritative?: boolean;
    mutation_controls_available?: boolean;
    notes?: string[];
  };
  modelruntime: {
    available: boolean;
    state?: string;
    backend?: string;
    runtime_enabled?: boolean;
    gpu_aware?: boolean;
    mutation_disabled?: boolean;
    warnings?: string[];
    errors?: string[];
  };
  storage: {
    root?: string;
    data_dir?: string;
    db_path?: string;
    truth_authority?: string;
    ping_ok?: boolean;
    total_bytes?: number;
    used_bytes?: number;
    free_bytes?: number;
    pressure_level?: string;
    redis?: {
      enabled?: boolean;
      truth_authority?: boolean;
      role?: string;
    };
    qdrant?: {
      enabled?: boolean;
      truth_authority?: boolean;
      role?: string;
    };
    cutover_readiness?: {
      status?: string;
      selected_domain?: string;
      canonical_default?: string;
      requested_backend?: string;
      live_owner?: string;
      target_owner?: string;
      ready_for_dual_write?: boolean;
      ready_for_read_compare?: boolean;
      ready_for_cutover_proposal?: boolean;
      postgres_canonical_ready?: boolean;
      redis_truth_authority?: boolean;
      qdrant_truth_authority?: boolean;
      tests_required?: string[];
      tests_passing?: string[];
      blockers?: string[];
      rollback_path?: string;
      no_effect?: Record<string, boolean>;
    };
  };
  operator_cockpit?: {
    available?: boolean;
    live_owner?: string;
    target_forge_k_owner?: string;
    mutation_controls_available?: boolean;
    rows?: Array<{
      id?: string;
      label?: string;
      live?: boolean;
      status?: string;
      live_owner?: string;
      target_owner?: string;
      source?: string;
      mutation_allowed?: boolean;
    }>;
  };
  authority?: {
    matrix_available?: boolean;
    matrix_rows?: number;
    live_authority_rows?: number;
    partial_validation_rows?: number;
    forge_k_live_authority_rows?: number;
    host_mutation_rows?: number;
    semantic_memory_write_rows?: number;
    modelruntime_gateway_alignment?: string;
    model_delete_file_status?: string;
    model_chat_owner?: string;
    model_generate_owner?: string;
    rows?: Array<{
      id?: string;
      surface?: string;
      method?: string;
      route?: string;
      action?: string;
      authorityOwner?: string;
      capabilityId?: string;
      gatewayCapabilityStatus?: string;
      mutating?: boolean;
      destructive?: boolean;
      requiresApproval?: boolean;
      approvalMechanism?: string;
      auditCategory?: string;
      auditAction?: string;
      responseVisibility?: string;
      liveAuthority?: boolean;
      forgeKAuthority?: boolean;
      hostMutation?: boolean;
      modelruntimeMutation?: boolean;
      semanticMemoryWrite?: boolean;
      status?: string;
      notes?: string;
    }>;
    blockers?: Array<{
      row_id?: string;
      status?: string;
      reason?: string;
      next_step?: string;
    }>;
    notes?: string[];
  };
  control_lane?: {
    approval_fingerprint?: {
      available?: boolean;
      version?: string;
      enforcement_wired?: boolean;
      deterministic_helper?: boolean;
      reason?: string;
    };
  };
  control_lane_fingerprint?: {
    status?: string;
    version?: string;
    deterministic?: boolean;
    reason?: string;
  };
  validation?: {
    available?: boolean;
    source?: string;
    status?: string;
    latency_measured?: boolean;
    reason?: string;
    commands?: Array<{
      command?: string;
      result?: string;
    }>;
  };
  validation_evidence?: {
    status?: string;
    source?: string;
    command?: string;
    updated_at?: string;
  };
  approval_queue?: {
    wired?: boolean;
    available?: boolean;
    reason?: string;
    pending_count?: number;
    total_count?: number;
    limit?: number;
    read_only?: boolean;
    items?: Array<{
      id?: number;
      job_id?: string;
      requested_action?: string;
      risk_class?: string;
      requested_adapter?: string;
      write_intent?: boolean;
      created_at_ms?: number;
      expires_at_ms?: number;
      status?: string;
    }>;
    by_risk_class?: Record<string, number>;
    by_action?: Record<string, number>;
  };
  warnings?: string[];
  errors?: string[];
};

export type ForgeSystemHost = {
  generated_at?: string;
  live_owner?: string;
  read_only?: boolean;
  mutation_disabled?: boolean;
  host?: {
    hostname?: string;
    architecture?: string;
    os_release?: string;
  };
  kernel?: {
    version?: string;
    modules?: Array<{ name?: string; refcount?: number; state?: string }>;
  };
  boot?: {
    parameters?: string[];
  };
  cpu?: {
    count?: number;
    load_average?: number[];
    utilization_estimate?: number;
  };
  memory?: {
    total_bytes?: number;
    available_bytes?: number;
    swap_total_bytes?: number;
    swap_free_bytes?: number;
    pressure_level?: string;
  };
  storage?: {
    root?: string;
    total_bytes?: number;
    used_bytes?: number;
    free_bytes?: number;
    pressure_level?: string;
  };
  gpu?: {
    available?: boolean;
    vendor?: string;
    devices?: Array<{
      name?: string;
      driver_version?: string;
      memory_total_mib?: number;
      memory_free_mib?: number;
      memory_used_mib?: number;
    }>;
    warnings?: string[];
  };
  thermal?: {
    available?: boolean;
    sensors?: Array<{ label?: string; temperature_c?: number }>;
    warnings?: string[];
  };
  display?: ForgeSystemHostReadOnlySection;
  audio?: ForgeSystemHostReadOnlySection;
  network?: ForgeSystemHostReadOnlySection;
  power?: ForgeSystemHostReadOnlySection;
  session?: {
    shell_mode?: string;
    display_backend?: string;
    compositor_session?: string;
    safe_mode?: boolean;
    host_mutation_disabled?: boolean;
    model_mutation_disabled?: boolean;
    semantic_memory_write_disabled?: boolean;
    forge_k_live_authority_disabled?: boolean;
    context_compiler_required_for_llm?: boolean;
  };
  config?: {
    data_dir?: string;
    workspace_dir?: string;
    store_backend?: string;
    enable_model_runtime?: boolean;
    gpu_enabled?: boolean;
    safe_mode_force_cpu_only?: boolean;
    modelruntime_allow_cloud_models?: boolean;
    model_policy_require_explicit_load?: boolean;
    model_policy_allow_auto_load?: boolean;
    model_policy_require_workspace_scope?: boolean;
  };
  source_errors?: Array<{ source?: string; error?: string }>;
  warnings?: string[];
};

export type ForgeSystemHostReadOnlySection = {
  status?: string;
  reason?: string;
  read_only?: boolean;
  mutation_disabled?: boolean;
};
