import type {
  AdapterInfo,
  AdapterInvokeRequest,
  AdapterMetric,
  ApprovalPreset,
  ApprovalRequest,
  AutomationHistory,
  AutomationRule,
  DashboardSummary,
  DossierMemoryView,
  DossierProfile,
  Dossier,
  DossierDetail,
  DossierVSASummary,
  EmbeddingConfig,
  EvaluationRecord,
  ExecutionStrategy,
  FailurePattern,
  ForgeEvent,
  ImportReconciliation,
  ImportedExecution,
  InvokeResult,
  JobDetail,
  JobLineage,
  JobRecord,
  JobTemplate,
  PacketGuidance,
  PacketAlignmentNote,
  PolicyRecommendation,
  ProjectContextRecord,
  ReembedResult,
  ReviewRecord,
  RetrievalRun,
  RetrievalSelectionReason,
  RoutingInsight,
  MemoryObservation,
  MemoryObservationDetail,
  MemoryRepairRun,
  MemoryRepairRunDetail,
  ObservationVSADetail,
  SearchHit,
  SourceEmbeddingStatus,
  SourceRow,
  TaskPacket,
  ToolCapability,
  ToolCapabilityStatus,
  RetrievalResultVSASignal,
  VSAReindexRun,
  VSAReindexRunDetail,
} from "@forge/shared";

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

const base = () =>
  import.meta.env.VITE_FORGE_API_URL ?? "http://127.0.0.1:18492";

async function j<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${base()}${path}`, {
    ...init,
    headers: {
      Accept: "application/json",
      ...(init?.headers ?? {}),
    },
  });
  if (!res.ok) {
    const t = await res.text().catch(() => "");
    throw new Error(t || `${res.status} ${res.statusText}`);
  }
  if (res.status === 204) {
    return undefined as T;
  }
  return (await res.json()) as T;
}

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
  remoteCrossChatContext: boolean;
  remoteDefaultThreadId: string;
  telegramBotToken: string;
  telegramDefaultChatId: string;
  discordBotToken: string;
  discordDefaultChannelId: string;
  discordWebhookUrl: string;
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

export const api = {
  health: () => j<ForgeHealth>("/health"),
  meta: () =>
    j<{ dataDir: string; dbPath: string; workspaceDir: string }>("/api/meta"),
  settings: {
    get: () => j<SettingsRecord>("/api/settings"),
    patch: (body: Record<string, unknown>) =>
      j<SettingsRecord>("/api/settings", {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    ollamaModels: (baseUrl?: string) => {
      const q = new URLSearchParams();
      if (baseUrl) q.set("baseUrl", baseUrl);
      const path = `/api/settings/ollama-models${q.toString() ? `?${q.toString()}` : ""}`;
      return j<{
        models: string[];
        baseUrl: string;
        status: string;
        error?: string;
      }>(path);
    },
  },
  modelRuntime: {
    list: () =>
      j<{
        models: ModelRuntimeModel[];
        correlationId?: string;
        traceId?: string;
        workspaceId?: string;
      }>("/forge/models"),
    get: (id: string) =>
      j<{
        model: ModelRuntimeModel;
        correlationId?: string;
        traceId?: string;
        workspaceId?: string;
      }>(`/forge/models/${encodeURIComponent(id)}`),
    import: (body: {
      path: string;
      id?: string;
      displayName?: string;
      family?: string;
      backend?: string;
      capabilities?: string[];
      license?: string;
      quantization?: string;
      contextLength?: number;
      preferred?: boolean;
      actor?: string;
      source?: string;
      workspaceId?: string;
      laneId?: string;
      capabilityId?: string;
      approvalId?: string;
      dryRun?: boolean;
      metadata?: Record<string, unknown>;
    }) =>
      j<{
        result: ModelRuntimeImportResult;
        correlationId?: string;
        traceId?: string;
        workspaceId?: string;
      }>("/forge/models/import", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    scan: (body?: ModelRuntimeManagementRequest) =>
      j<{
        models: ModelRuntimeModel[];
        count: number;
        correlationId?: string;
        traceId?: string;
        workspaceId?: string;
      }>("/forge/models/scan", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body ?? {}),
      }),
    verify: (id: string, body?: ModelRuntimeManagementRequest) =>
      j<{
        model: ModelRuntimeModel;
        correlationId?: string;
        traceId?: string;
        workspaceId?: string;
      }>(`/forge/models/${encodeURIComponent(id)}/verify`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body ?? {}),
      }),
    enable: (id: string, body?: ModelRuntimeManagementRequest) =>
      j<{
        model: ModelRuntimeModel;
        correlationId?: string;
        traceId?: string;
        workspaceId?: string;
      }>(`/forge/models/${encodeURIComponent(id)}/enable`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body ?? {}),
      }),
    disable: (id: string, body?: ModelRuntimeManagementRequest) =>
      j<{
        model: ModelRuntimeModel;
        correlationId?: string;
        traceId?: string;
        workspaceId?: string;
      }>(`/forge/models/${encodeURIComponent(id)}/disable`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body ?? {}),
      }),
    archive: (id: string, body?: ModelRuntimeManagementRequest) =>
      j<{
        model: ModelRuntimeModel;
        correlationId?: string;
        traceId?: string;
        workspaceId?: string;
      }>(`/forge/models/${encodeURIComponent(id)}/archive`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body ?? {}),
      }),
    remove: (id: string, body?: ModelRuntimeManagementRequest) =>
      j<{
        result: { modelId: string; removedPath?: string };
        correlationId?: string;
        traceId?: string;
        workspaceId?: string;
      }>(`/forge/models/${encodeURIComponent(id)}/remove`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body ?? {}),
      }),
    load: (id: string, body?: ModelRuntimeManagementRequest) =>
      j<{
        result: ModelRuntimeLoadResult;
        correlationId?: string;
        traceId?: string;
        workspaceId?: string;
      }>(`/forge/models/${encodeURIComponent(id)}/load`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body ?? {}),
      }),
    unload: (id: string, body?: ModelRuntimeManagementRequest) =>
      j<{
        result: ModelRuntimeLoadResult;
        correlationId?: string;
        traceId?: string;
        workspaceId?: string;
      }>(`/forge/models/${encodeURIComponent(id)}/unload`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body ?? {}),
      }),
    compatibility: (id: string) =>
      j<{
        compatibility: ModelRuntimeCompatibility;
        correlationId?: string;
        traceId?: string;
        workspaceId?: string;
      }>(`/forge/models/${encodeURIComponent(id)}/compatibility`),
    health: () =>
      j<{
        health: ModelRuntimeHealth;
        correlationId?: string;
        traceId?: string;
        workspaceId?: string;
      }>("/forge/model-runtime/health"),
    backends: () =>
      j<{
        backends: ModelRuntimeBackendStatus[];
        correlationId?: string;
        traceId?: string;
        workspaceId?: string;
      }>("/forge/model-runtime/backends"),
    usage: () =>
      j<{
        usage: ModelRuntimeUsageSummary;
        correlationId?: string;
        traceId?: string;
        workspaceId?: string;
      }>("/forge/model-runtime/usage"),
    queue: () =>
      j<{
        queue: ModelRuntimeQueueStatus;
        correlationId?: string;
        traceId?: string;
        workspaceId?: string;
      }>("/forge/model-runtime/queue"),
    loaded: () =>
      j<{
        loaded: ModelRuntimeLoadedStatus;
        correlationId?: string;
        traceId?: string;
        workspaceId?: string;
      }>("/forge/model-runtime/loaded"),
  },
  remote: {
    telegram: (body: RemoteTelegramPayload, token?: string) =>
      j<{ ok: boolean }>("/api/remote/telegram", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          ...(token?.trim() ? { "X-Forge-Remote-Token": token.trim() } : {}),
        },
        body: JSON.stringify(body),
      }),
    discord: (body: RemoteDiscordPayload, token?: string) =>
      j<{ ok: boolean }>("/api/remote/discord", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          ...(token?.trim() ? { "X-Forge-Remote-Token": token.trim() } : {}),
        },
        body: JSON.stringify(body),
      }),
  },
  telegram: {
    status: () => j<TelegramStatusResponse>("/api/telegram/status"),
  },
  sources: {
    list: () => j<{ sources: SourceRow[] }>("/api/sources"),
    add: (path: string) =>
      j<{ id: number; path: string }>("/api/sources", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ path }),
      }),
    delete: (id: number) =>
      j<void>(`/api/sources/${id}`, {
        method: "DELETE",
      }),
  },
  reindex: (sourceId?: number) => {
    const q =
      sourceId != null
        ? `?sourceId=${encodeURIComponent(String(sourceId))}`
        : "";
    return j<{ ok: boolean; scope: string }>(`/api/reindex${q}`, {
      method: "POST",
    });
  },
  search: (q: string, limit = 50) =>
    j<{ hits: SearchHit[] }>(
      `/api/search?q=${encodeURIComponent(q)}&limit=${encodeURIComponent(String(limit))}`,
    ),
  chunk: (id: number) => j<SearchHit>(`/api/chunks/${id}`),
  events: (limit = 120) =>
    j<{ events: ForgeEvent[] }>(
      `/api/events?limit=${encodeURIComponent(String(limit))}`,
    ),
  autonomy: {
    status: () => j<AutonomyStatusSnapshot>("/api/autonomy/status"),
    intents: (params?: { status?: string; limit?: number }) => {
      const qs = new URLSearchParams();
      if (params?.status) qs.set("status", params.status);
      if (params?.limit != null) qs.set("limit", String(params.limit));
      const q = qs.toString();
      return j<{ intents: AutonomyIntentRecord[] }>(
        `/api/autonomy/intents${q ? `?${q}` : ""}`,
      );
    },
    explainIntent: (id: string) =>
      j<Record<string, unknown>>(
        `/api/autonomy/intents/${encodeURIComponent(id)}/explain`,
      ),
    decisions: (limit = 80) =>
      j<{ decisions: AutonomyDecisionRecord[] }>(
        `/api/autonomy/decisions?limit=${encodeURIComponent(String(limit))}`,
      ),
    budgets: () =>
      j<{ budgets: AutonomyBudgetRecord[] }>("/api/autonomy/budgets"),
    charters: (activeOnly = false) =>
      j<{ charters: AutonomyCharterRecord[] }>(
        `/api/autonomy/charters?activeOnly=${encodeURIComponent(String(activeOnly))}`,
      ),
    events: (limit = 120) =>
      j<{ events: ForgeEvent[] }>(
        `/api/autonomy/events?limit=${encodeURIComponent(String(limit))}`,
      ),
  },
  discord: {
    status: () => j<DiscordGatewayStatusResponse>("/api/discord/status"),
  },
  adapters: {
    list: () => j<{ adapters: AdapterInfo[] }>("/api/adapters"),
    invoke: async (id: string, body: AdapterInvokeRequest) => {
      const scope = body.scope ?? {
        allowedPaths: [],
        forbiddenPaths: [],
        selectedPaths: [],
      };
      const paths = [
        ...scope.selectedPaths,
        ...scope.allowedPaths,
        ...scope.forbiddenPaths,
      ]
        .map((value) => value.trim())
        .filter((value) => value.length > 0);
      const correlationId =
        body.correlationId?.trim() ||
        `legacy.adapter.invoke:${id}:${Date.now()}`;
      const gatewayResponse = await j<{
        result: {
          status?: string;
          message?: string;
          deniedReason?: string;
          data?: Record<string, unknown>;
        };
      }>("/api/gateway/invoke", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          toolId: "legacy.adapter.invoke",
          laneId: "legacy.adapter.invoke",
          action: "invoke",
          source: "api",
          initiator: "api",
          correlationId,
          paths,
          input: {
            adapterId: id,
            capability: body.capability,
            scope,
            writeIntent: body.writeIntent ?? false,
            timeoutMs: body.timeoutMs ?? 0,
            dryRun: body.dryRun ?? false,
            correlationId,
            input: body.input ?? {},
            ...(body.taskPacketRef != null
              ? { taskPacketRef: body.taskPacketRef }
              : {}),
          },
          metadata: {
            legacyAdapterCompatibilityWrapper: true,
            legacyIngressRemoved: "/api/adapters/{id}/invoke",
          },
        }),
      });

      const gwResult = gatewayResponse.result ?? {};
      if ((gwResult.status ?? "").toLowerCase() !== "ok") {
        throw new Error(
          gwResult.message?.trim() ||
            gwResult.deniedReason?.trim() ||
            "adapter invocation denied",
        );
      }

      const rawResult = (gwResult.data as { result?: unknown } | undefined)
        ?.result;
      if (!rawResult || typeof rawResult !== "object") {
        throw new Error("gateway response missing adapter invocation result");
      }
      const typed = rawResult as Partial<InvokeResult>;
      return {
        ok: Boolean(typed.ok),
        message:
          typeof typed.message === "string"
            ? typed.message
            : "adapter invocation completed",
        ...(typeof typed.failureCode === "string"
          ? { failureCode: typed.failureCode }
          : {}),
        data:
          typeof typed.data === "object" && typed.data != null
            ? typed.data
            : {},
      };
    },
  },
  jobs: {
    templates: () => j<{ templates: JobTemplate[] }>("/api/jobs/templates"),
    list: (status = "", limit = 120) => {
      const params = new URLSearchParams();
      if (status) params.set("status", status);
      params.set("limit", String(limit));
      return j<{ jobs: JobRecord[] }>(`/api/jobs?${params.toString()}`);
    },
    create: (body: Record<string, unknown>) =>
      j<{ job: JobRecord }>("/api/jobs", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    detail: (id: string, afterEventId = 0) =>
      j<JobDetail>(
        `/api/jobs/${encodeURIComponent(id)}?afterEventId=${encodeURIComponent(String(afterEventId))}`,
      ),
    cancel: (id: string, actor = "operator") =>
      j<{ ok: boolean; jobId: string }>(
        `/api/jobs/${encodeURIComponent(id)}/cancel`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ actor }),
        },
      ),
    retry: (id: string, body: Record<string, unknown>) =>
      j<{ job: JobRecord }>(`/api/jobs/${encodeURIComponent(id)}/retry`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    replay: (id: string, body: Record<string, unknown>) =>
      j<{ job: JobRecord }>(`/api/jobs/${encodeURIComponent(id)}/replay`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
  },
  approvals: {
    list: (status = "pending", limit = 100) =>
      j<{ approvals: ApprovalRequest[] }>(
        `/api/approvals?status=${encodeURIComponent(status)}&limit=${encodeURIComponent(String(limit))}`,
      ),
    get: (id: number) =>
      j<{ approval: ApprovalRequest }>(
        `/api/approvals/${encodeURIComponent(String(id))}`,
      ),
    approve: (id: number, note = "") =>
      j<{ decision: unknown }>(`/api/approvals/${id}/approve`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ actor: "operator", note }),
      }),
    deny: (id: number, note = "") =>
      j<{ decision: unknown }>(`/api/approvals/${id}/deny`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ actor: "operator", note }),
      }),
    cancel: (id: number, note = "") =>
      j<{ decision: unknown }>(`/api/approvals/${id}/cancel`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ actor: "operator", note }),
      }),
  },
  packets: {
    get: (id: number) => j<TaskPacket>(`/api/packets/${id}`),
  },
  projectContext: {
    get: () =>
      j<{ record: ProjectContextRecord | null }>("/api/project-context"),
    import: (sourcePath = "", notes = "") =>
      j<{ record: ProjectContextRecord }>("/api/project-context/import", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ sourcePath, notes }),
      }),
    regenerate: () =>
      j<{ record: ProjectContextRecord }>("/api/project-context/regenerate", {
        method: "POST",
      }),
  },
  embeddings: {
    status: (provider = "", model = "") => {
      const params = new URLSearchParams();
      if (provider) params.set("provider", provider);
      if (model) params.set("model", model);
      const q = params.toString();
      return j<{ config: EmbeddingConfig; status: SourceEmbeddingStatus[] }>(
        `/api/embeddings/status${q ? `?${q}` : ""}`,
      );
    },
    reembed: (body: Record<string, unknown>) =>
      j<{ result: ReembedResult }>("/api/embeddings/reembed", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
  },
  retrieval: {
    createRun: (body: Record<string, unknown>) =>
      j<{ run: RetrievalRun }>("/api/retrieval/runs", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    listRuns: (params?: {
      limit?: number;
      dossierId?: number;
      jobId?: string;
    }) => {
      const qs = new URLSearchParams();
      if (params?.limit != null) qs.set("limit", String(params.limit));
      if (params?.dossierId != null)
        qs.set("dossierId", String(params.dossierId));
      if (params?.jobId) qs.set("jobId", params.jobId);
      const q = qs.toString();
      return j<{ runs: RetrievalRun[] }>(
        `/api/retrieval/runs${q ? `?${q}` : ""}`,
      );
    },
    getRun: (id: number) =>
      j<{ run: RetrievalRun }>(`/api/retrieval/runs/${id}`),
    getRunVSASignals: (runId: number) =>
      j<{ signals: RetrievalResultVSASignal[] }>(
        `/api/retrieval/runs/${encodeURIComponent(String(runId))}/vsa-signals`,
      ),
    markUsefulness: (id: number, body: Record<string, unknown>) =>
      j<{ ok: boolean; resultId: number }>(
        `/api/retrieval/results/${id}/usefulness`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body),
        },
      ),
  },
  memory: {
    listObservations: (params?: {
      limit?: number;
      dossierId?: number;
      type?: string;
      originKind?: string;
      staleOnly?: boolean;
    }) => {
      const qs = new URLSearchParams();
      if (params?.limit != null) qs.set("limit", String(params.limit));
      if (params?.dossierId != null)
        qs.set("dossierId", String(params.dossierId));
      if (params?.type) qs.set("type", params.type);
      if (params?.originKind) qs.set("originKind", params.originKind);
      if (params?.staleOnly) qs.set("staleOnly", "true");
      const q = qs.toString();
      return j<{ observations: MemoryObservation[] }>(
        `/api/memory/observations${q ? `?${q}` : ""}`,
      );
    },
    createObservation: (body: Record<string, unknown>) =>
      j<{ observation: MemoryObservation }>("/api/memory/observations", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    getObservation: (id: number) =>
      j<{ observation: MemoryObservationDetail }>(
        `/api/memory/observations/${encodeURIComponent(String(id))}`,
      ),
    patchObservation: (id: number, body: Record<string, unknown>) =>
      j<{ observation: MemoryObservationDetail }>(
        `/api/memory/observations/${encodeURIComponent(String(id))}`,
        {
          method: "PATCH",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body),
        },
      ),
    getObservationVSA: (id: number) =>
      j<{ detail: ObservationVSADetail }>(
        `/api/memory/observations/${encodeURIComponent(String(id))}/vsa`,
      ),
    markObservationUsefulness: (id: number, body: Record<string, unknown>) =>
      j<{ ok: boolean; observationId: number }>(
        `/api/memory/observations/${encodeURIComponent(String(id))}/usefulness`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body),
        },
      ),
    retrievalSelection: (runId: number) =>
      j<{ selection: RetrievalSelectionReason[] }>(
        `/api/memory/retrieval-runs/${encodeURIComponent(String(runId))}/selection`,
      ),
    packetAlignment: (packetId: number, limit = 80) =>
      j<{ notes: PacketAlignmentNote[] }>(
        `/api/memory/packets/${encodeURIComponent(String(packetId))}/alignment?limit=${encodeURIComponent(String(limit))}`,
      ),
    dossierView: (dossierId: number, limit = 40) =>
      j<{ view: DossierMemoryView }>(
        `/api/memory/dossiers/${encodeURIComponent(String(dossierId))}?limit=${encodeURIComponent(String(limit))}`,
      ),
    listRepairRuns: (params?: { limit?: number; dossierId?: number }) => {
      const qs = new URLSearchParams();
      if (params?.limit != null) qs.set("limit", String(params.limit));
      if (params?.dossierId != null)
        qs.set("dossierId", String(params.dossierId));
      const q = qs.toString();
      return j<{ runs: MemoryRepairRun[] }>(
        `/api/memory/repair-runs${q ? `?${q}` : ""}`,
      );
    },
    getRepairRun: (id: number) =>
      j<{ detail: MemoryRepairRunDetail }>(
        `/api/memory/repair-runs/${encodeURIComponent(String(id))}`,
      ),
    runRepair: (body: {
      dossierId?: number;
      maxAgeDays?: number;
      limit?: number;
      note?: string;
    }) =>
      j<{ detail: MemoryRepairRunDetail }>("/api/memory/repair/run", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    listVSAReindexRuns: (params?: { limit?: number; dossierId?: number }) => {
      const qs = new URLSearchParams();
      if (params?.limit != null) qs.set("limit", String(params.limit));
      if (params?.dossierId != null)
        qs.set("dossierId", String(params.dossierId));
      const q = qs.toString();
      return j<{ runs: VSAReindexRun[] }>(
        `/api/memory/vsa/reindex-runs${q ? `?${q}` : ""}`,
      );
    },
    getVSAReindexRun: (id: number) =>
      j<{ detail: VSAReindexRunDetail }>(
        `/api/memory/vsa/reindex-runs/${encodeURIComponent(String(id))}`,
      ),
    runVSAReindex: (body: {
      dossierId?: number;
      mode?: string;
      triggeredBy?: string;
      reason?: string;
      note?: string;
      limit?: number;
      staleOnly?: boolean;
      force?: boolean;
    }) =>
      j<{ detail: VSAReindexRunDetail }>("/api/memory/vsa/reindex/run", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    dossierVSASummary: (dossierId: number) =>
      j<{ summary: DossierVSASummary }>(
        `/api/memory/dossiers/${encodeURIComponent(String(dossierId))}/vsa-summary`,
      ),
  },
  dossiers: {
    list: (limit = 120) =>
      j<{ dossiers: Dossier[] }>(
        `/api/dossiers?limit=${encodeURIComponent(String(limit))}`,
      ),
    create: (body: Record<string, unknown>) =>
      j<{ dossier: Dossier }>("/api/dossiers", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    detail: (id: number) => j<{ detail: DossierDetail }>(`/api/dossiers/${id}`),
    update: (id: number, body: Record<string, unknown>) =>
      j<{ dossier: Dossier }>(`/api/dossiers/${id}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    generateBrief: (id: number, notes = "") =>
      j<{ brief: unknown }>(`/api/dossiers/${id}/briefs/generate`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ notes }),
      }),
  },
  evaluations: {
    create: (body: Record<string, unknown>) =>
      j<{ evaluation: EvaluationRecord }>("/api/evaluations", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    list: (limit = 120, dossierId?: number) => {
      const qs = new URLSearchParams();
      qs.set("limit", String(limit));
      if (dossierId != null) qs.set("dossierId", String(dossierId));
      return j<{ evaluations: EvaluationRecord[] }>(
        `/api/evaluations?${qs.toString()}`,
      );
    },
    metrics: (dossierId?: number) => {
      const q =
        dossierId != null
          ? `?dossierId=${encodeURIComponent(String(dossierId))}`
          : "";
      return j<{ metrics: AdapterMetric[] }>(`/api/evaluations/metrics${q}`);
    },
  },
  lineage: {
    byJob: (jobId: string) =>
      j<JobLineage>(`/api/lineage/jobs/${encodeURIComponent(jobId)}`),
  },
  imports: {
    list: (limit = 100, dossierId?: number) => {
      const qs = new URLSearchParams();
      qs.set("limit", String(limit));
      if (dossierId != null) qs.set("dossierId", String(dossierId));
      return j<{ imports: ImportedExecution[] }>(
        `/api/imports/executions?${qs.toString()}`,
      );
    },
    create: (body: Record<string, unknown>) =>
      j<{ importedExecution: ImportedExecution }>("/api/imports/executions", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
  },
  insights: {
    list: (limit = 120, dossierId?: number) => {
      const qs = new URLSearchParams();
      qs.set("limit", String(limit));
      if (dossierId != null) qs.set("dossierId", String(dossierId));
      return j<{ insights: RoutingInsight[] }>(
        `/api/insights?${qs.toString()}`,
      );
    },
    generate: (dossierId?: number) =>
      j<{ insights: RoutingInsight[] }>("/api/insights/generate", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ dossierId }),
      }),
  },
  dashboard: {
    summary: () => j<DashboardSummary>("/api/dashboard"),
  },
  strategies: {
    list: (params?: { enabled?: boolean; limit?: number }) => {
      const qs = new URLSearchParams();
      if (params?.enabled != null) qs.set("enabled", String(params.enabled));
      if (params?.limit != null) qs.set("limit", String(params.limit));
      const q = qs.toString();
      return j<{ strategies: ExecutionStrategy[] }>(
        `/api/strategies${q ? `?${q}` : ""}`,
      );
    },
    save: (body: Record<string, unknown>) =>
      j<{ strategy: ExecutionStrategy }>("/api/strategies", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
  },
  policy: {
    listPresets: (limit = 60) =>
      j<{ presets: ApprovalPreset[] }>(
        `/api/policy/presets?limit=${encodeURIComponent(String(limit))}`,
      ),
    savePreset: (body: Record<string, unknown>) =>
      j<{ preset: ApprovalPreset }>("/api/policy/presets", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    getGlobalPreset: () => j<{ presetId: string }>("/api/policy/global-preset"),
    setGlobalPreset: (presetId: string) =>
      j<{ ok: boolean; presetId: string }>("/api/policy/global-preset", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ presetId }),
      }),
    getDossierProfile: (id: number) =>
      j<{ profile: DossierProfile | null }>(
        `/api/policy/dossiers/${encodeURIComponent(String(id))}`,
      ),
    saveDossierProfile: (id: number, body: Record<string, unknown>) =>
      j<{ profile: DossierProfile }>(
        `/api/policy/dossiers/${encodeURIComponent(String(id))}`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body),
        },
      ),
    recommend: (body: Record<string, unknown>) =>
      j<{ recommendation: PolicyRecommendation }>("/api/policy/recommend", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    listRecommendations: (params?: { limit?: number; dossierId?: number }) => {
      const qs = new URLSearchParams();
      if (params?.limit != null) qs.set("limit", String(params.limit));
      if (params?.dossierId != null)
        qs.set("dossierId", String(params.dossierId));
      const q = qs.toString();
      return j<{ recommendations: PolicyRecommendation[] }>(
        `/api/policy/recommendations${q ? `?${q}` : ""}`,
      );
    },
  },
  automation: {
    listRules: (params?: { enabled?: boolean; limit?: number }) => {
      const qs = new URLSearchParams();
      if (params?.enabled != null) qs.set("enabled", String(params.enabled));
      if (params?.limit != null) qs.set("limit", String(params.limit));
      const q = qs.toString();
      return j<{ rules: AutomationRule[] }>(
        `/api/automation/rules${q ? `?${q}` : ""}`,
      );
    },
    saveRule: (body: Record<string, unknown>) =>
      j<{ rule: AutomationRule }>("/api/automation/rules", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    history: (limit = 120) =>
      j<{ history: AutomationHistory[] }>(
        `/api/automation/history?limit=${encodeURIComponent(String(limit))}`,
      ),
    runRule: (body: Record<string, unknown>) =>
      j<{
        result: {
          rule: AutomationRule;
          matched: boolean;
          dryRun: boolean;
          preview: Record<string, unknown>;
          executed: boolean;
          execution: Record<string, unknown>;
          historyId: number;
        };
      }>("/api/automation/run", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
  },
  packetGuidance: {
    list: (params?: { limit?: number; packetId?: number }) => {
      const qs = new URLSearchParams();
      if (params?.limit != null) qs.set("limit", String(params.limit));
      if (params?.packetId != null) qs.set("packetId", String(params.packetId));
      const q = qs.toString();
      return j<{ guidance: PacketGuidance[] }>(
        `/api/packet-guidance${q ? `?${q}` : ""}`,
      );
    },
    analyze: (body: Record<string, unknown>) =>
      j<{ guidance: PacketGuidance }>("/api/packet-guidance/analyze", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
  },
  reconciliation: {
    getByImport: (importId: number) =>
      j<{ reconciliation: ImportReconciliation }>(
        `/api/reconciliation/imports/${encodeURIComponent(String(importId))}`,
      ),
    saveByImport: (importId: number, body: Record<string, unknown>) =>
      j<{ reconciliation: ImportReconciliation }>(
        `/api/reconciliation/imports/${encodeURIComponent(String(importId))}`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body),
        },
      ),
    list: (params?: { limit?: number; reviewStatus?: string }) => {
      const qs = new URLSearchParams();
      if (params?.limit != null) qs.set("limit", String(params.limit));
      if (params?.reviewStatus) qs.set("reviewStatus", params.reviewStatus);
      const q = qs.toString();
      return j<{ reconciliations: ImportReconciliation[] }>(
        `/api/reconciliation${q ? `?${q}` : ""}`,
      );
    },
  },
  reviews: {
    list: (params?: { limit?: number; status?: string }) => {
      const qs = new URLSearchParams();
      if (params?.limit != null) qs.set("limit", String(params.limit));
      if (params?.status) qs.set("status", params.status);
      const q = qs.toString();
      return j<{ reviews: ReviewRecord[] }>(`/api/reviews${q ? `?${q}` : ""}`);
    },
    create: (body: Record<string, unknown>) =>
      j<{ review: ReviewRecord }>("/api/reviews", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    update: (id: number, body: Record<string, unknown>) =>
      j<{ review: ReviewRecord }>(
        `/api/reviews/${encodeURIComponent(String(id))}`,
        {
          method: "PATCH",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body),
        },
      ),
  },
  failurePatterns: {
    list: (params?: { limit?: number; dossierId?: number }) => {
      const qs = new URLSearchParams();
      if (params?.limit != null) qs.set("limit", String(params.limit));
      if (params?.dossierId != null)
        qs.set("dossierId", String(params.dossierId));
      const q = qs.toString();
      return j<{ patterns: FailurePattern[] }>(
        `/api/failure-patterns${q ? `?${q}` : ""}`,
      );
    },
    analyze: (body: Record<string, unknown>) =>
      j<{ patterns: FailurePattern[] }>("/api/failure-patterns/analyze", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
  },
  commands: {
    execute: (name: string, args?: Record<string, unknown>) =>
      j<{ ok: boolean; note?: string; jobId?: string }>(
        "/api/commands/execute",
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ name, args: args ?? {} }),
        },
      ),
  },
  gateway: {
    tools: () =>
      j<{
        tools: Array<{
          id: string;
          domain: string;
          action: string;
          description: string;
          riskClass: string;
          executionLevel: string;
          executes: boolean;
          usesNetwork: boolean;
          writeIntent: boolean;
          capabilityId?: string;
          capabilityStatus?: ToolCapabilityStatus;
          capabilityRisk?: string;
          adapterId?: string;
          requiresApprovalByDefault?: boolean;
          autonomyEligible?: boolean;
          allowedInDryRun?: boolean;
        }>;
      }>("/api/gateway/tools"),
    capabilities: () =>
      j<{ capabilities: ToolCapability[] }>("/api/gateway/capabilities"),
    updateCapabilityStatus: (
      id: string,
      body: {
        status: ToolCapabilityStatus;
        reason?: string;
        actor?: string;
        actorKind?: string;
        source?: string;
        approvalId?: string;
        correlationId?: string;
        traceId?: string;
        dryRun?: boolean;
      },
    ) =>
      j<{
        success: boolean;
        capability?: ToolCapability;
        capabilityId?: string;
        previousStatus: ToolCapabilityStatus;
        newStatus: ToolCapabilityStatus;
        riskClass?: string;
        approvalRequired?: boolean;
        approvalRequestId?: number;
        rejectionReason?: string;
        correlationId?: string;
        traceId?: string;
        auditCategory?: string;
      }>(`/api/gateway/capabilities/${encodeURIComponent(id)}/status`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    invoke: (body: Record<string, unknown>) =>
      j<{ result: Record<string, unknown> }>("/api/gateway/invoke", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    invocations: (params?: { limit?: number; status?: string }) => {
      const qs = new URLSearchParams();
      if (params?.limit != null) qs.set("limit", String(params.limit));
      if (params?.status) qs.set("status", params.status);
      const q = qs.toString();
      return j<{
        invocations: Array<{
          id: number;
          correlationId: string;
          createdAtMs: number;
          completedAtMs?: number | null;
          toolId: string;
          laneId?: string | null;
          jobId?: string | null;
          packetId?: number | null;
          initiator: string;
          action: string;
          domain: string;
          riskClass: string;
          executionLevel: string;
          policyOutcome: string;
          writeIntent: boolean;
          scope: unknown;
          input: unknown;
          status: string;
          deniedReason: string;
          result: unknown;
          artifacts: unknown;
          permissionProfileId: string;
          approvalRequestId?: number | null;
        }>;
      }>(`/api/gateway/invocations${q ? `?${q}` : ""}`);
    },
  },
  actionLanes: {
    list: () => j<{ lanes: unknown[] }>("/api/action-lanes"),
    save: (body: Record<string, unknown>) =>
      j<{ lane: unknown }>("/api/action-lanes", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    delete: (id: string) =>
      j<void>(`/api/action-lanes/${encodeURIComponent(id)}`, {
        method: "DELETE",
      }),
  },
  executionPermissions: {
    profiles: () =>
      j<{
        profiles: unknown[];
        active: unknown | null;
        summary: Record<string, unknown>;
      }>("/api/permissions/profiles"),
    saveProfile: (body: Record<string, unknown>) =>
      j<{ profile: unknown }>("/api/permissions/profiles", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    activateProfile: (id: string) =>
      j<{ profile: unknown }>(
        `/api/permissions/profiles/${encodeURIComponent(id)}/activate`,
        {
          method: "POST",
        },
      ),
    deleteProfile: (id: string) =>
      j<void>(`/api/permissions/profiles/${encodeURIComponent(id)}`, {
        method: "DELETE",
      }),
  },
  audit: {
    list: (params?: {
      limit?: number;
      category?: string;
      correlationId?: string;
      jobId?: string;
      outcome?: string;
    }) => {
      const qs = new URLSearchParams();
      if (params?.limit != null) qs.set("limit", String(params.limit));
      if (params?.category) qs.set("category", params.category);
      if (params?.correlationId) qs.set("correlationId", params.correlationId);
      if (params?.jobId) qs.set("jobId", params.jobId);
      if (params?.outcome) qs.set("outcome", params.outcome);
      const q = qs.toString();
      return j<{ records: unknown[] }>(`/api/audit${q ? `?${q}` : ""}`);
    },
    trace: (correlationId: string) =>
      j<{
        correlationId: string;
        records: unknown[];
        report?: Record<string, unknown>;
      }>(`/api/audit/trace/${encodeURIComponent(correlationId)}`),
    lookup: (params: { correlationId?: string; traceId?: string }) => {
      const qs = new URLSearchParams();
      if (params.correlationId) qs.set("correlationId", params.correlationId);
      if (params.traceId) qs.set("traceId", params.traceId);
      const q = qs.toString();
      return j<AuditTraceLookupResponse>(`/api/audit/trace${q ? `?${q}` : ""}`);
    },
  },
  processHealth: (params: { correlationId?: string; traceId?: string }) => {
    const qs = new URLSearchParams();
    if (params.correlationId) qs.set("correlationId", params.correlationId);
    if (params.traceId) qs.set("traceId", params.traceId);
    const q = qs.toString();
    return j<ProcessHealthTraceResponse>(
      `/api/process/health${q ? `?${q}` : ""}`,
    );
  },
  contextInspector: {
    listSnapshots: (params?: {
      limit?: number;
      workspaceId?: string;
      laneId?: string;
      correlationId?: string;
      snapshotKind?: string;
      query?: string;
    }) => {
      const qs = new URLSearchParams();
      if (params?.limit != null) qs.set("limit", String(params.limit));
      if (params?.workspaceId) qs.set("workspaceId", params.workspaceId);
      if (params?.laneId) qs.set("laneId", params.laneId);
      if (params?.correlationId) qs.set("correlationId", params.correlationId);
      if (params?.snapshotKind) qs.set("snapshotKind", params.snapshotKind);
      if (params?.query) qs.set("query", params.query);
      const q = qs.toString();
      return j<{ snapshots: ContextSnapshotInspectorSummary[] }>(
        `/api/context-inspector/snapshots${q ? `?${q}` : ""}`,
      );
    },
    getSnapshot: (
      id: string,
      params?: { workspaceId?: string; laneId?: string },
    ) => {
      const qs = new URLSearchParams();
      if (params?.workspaceId) qs.set("workspaceId", params.workspaceId);
      if (params?.laneId) qs.set("laneId", params.laneId);
      const q = qs.toString();
      return j<{ snapshot: ContextSnapshotInspectorDetail }>(
        `/api/context-inspector/snapshots/${encodeURIComponent(id)}${q ? `?${q}` : ""}`,
      );
    },
    restoreRecent: (params: {
      workspaceId: string;
      laneId?: string;
      limit?: number;
      snapshotKind?: string;
    }) => {
      const qs = new URLSearchParams();
      qs.set("workspaceId", params.workspaceId);
      if (params.laneId) qs.set("laneId", params.laneId);
      if (params.limit != null) qs.set("limit", String(params.limit));
      if (params.snapshotKind) qs.set("snapshotKind", params.snapshotKind);
      const q = qs.toString();
      return j<{ snapshots: ContextSnapshotInspectorSummary[] }>(
        `/api/context/restore/recent?${q}`,
      );
    },
    restoreGet: (
      id: string,
      params: { workspaceId: string; laneId?: string },
    ) => {
      const qs = new URLSearchParams();
      qs.set("workspaceId", params.workspaceId);
      if (params.laneId) qs.set("laneId", params.laneId);
      return j<{
        snapshot: ContextSnapshotInspectorDetail;
        evidenceClass: string;
        nonCanonicalEvidence: boolean;
        canonicalWriteCommitted: boolean;
      }>(`/api/context/restore/${encodeURIComponent(id)}?${qs.toString()}`);
    },
    restoreCandidates: (
      id: string,
      params: { workspaceId: string; laneId?: string },
    ) => {
      const qs = new URLSearchParams();
      qs.set("workspaceId", params.workspaceId);
      if (params.laneId) qs.set("laneId", params.laneId);
      return j<RestoreInspectorCandidatesResponse>(
        `/api/context/restore/${encodeURIComponent(id)}/candidates?${qs.toString()}`,
      );
    },
    restoreScore: (
      id: string,
      params: { workspaceId: string; laneId?: string },
    ) => {
      const qs = new URLSearchParams();
      qs.set("workspaceId", params.workspaceId);
      if (params.laneId) qs.set("laneId", params.laneId);
      return j<RestoreInspectorScoreResponse>(
        `/api/context/restore/${encodeURIComponent(id)}/score?${qs.toString()}`,
      );
    },
    restoreResumeHints: (
      id: string,
      params: { workspaceId: string; laneId?: string },
    ) => {
      const qs = new URLSearchParams();
      qs.set("workspaceId", params.workspaceId);
      if (params.laneId) qs.set("laneId", params.laneId);
      return j<RestoreInspectorResumeHintsResponse>(
        `/api/context/restore/${encodeURIComponent(id)}/resume-hints?${qs.toString()}`,
      );
    },
  },
  dreamReports: {
    list: (params: {
      workspaceId: string;
      laneId?: string;
      mode?: string;
      limit?: number;
    }) => {
      const qs = new URLSearchParams();
      qs.set("workspaceId", params.workspaceId);
      if (params.laneId) qs.set("laneId", params.laneId);
      if (params.mode) qs.set("mode", params.mode);
      if (params.limit != null) qs.set("limit", String(params.limit));
      return j<{ reports: DreamReportSummary[] }>(
        `/api/dream/reports?${qs.toString()}`,
      );
    },
    get: (id: string, params: { workspaceId: string; laneId?: string }) => {
      const qs = new URLSearchParams();
      qs.set("workspaceId", params.workspaceId);
      if (params.laneId) qs.set("laneId", params.laneId);
      return j<DreamReportDetail>(
        `/api/dream/reports/${encodeURIComponent(id)}?${qs.toString()}`,
      );
    },
    candidates: (
      id: string,
      params: { workspaceId: string; laneId?: string },
    ) => {
      const qs = new URLSearchParams();
      qs.set("workspaceId", params.workspaceId);
      if (params.laneId) qs.set("laneId", params.laneId);
      return j<DreamReportCandidatesResponse>(
        `/api/dream/reports/${encodeURIComponent(id)}/candidates?${qs.toString()}`,
      );
    },
    proposals: (
      id: string,
      params: { workspaceId: string; laneId?: string },
    ) => {
      const qs = new URLSearchParams();
      qs.set("workspaceId", params.workspaceId);
      if (params.laneId) qs.set("laneId", params.laneId);
      return j<DreamReportProposalsResponse>(
        `/api/dream/reports/${encodeURIComponent(id)}/proposals?${qs.toString()}`,
      );
    },
    warnings: (
      id: string,
      params: { workspaceId: string; laneId?: string },
    ) => {
      const qs = new URLSearchParams();
      qs.set("workspaceId", params.workspaceId);
      if (params.laneId) qs.set("laneId", params.laneId);
      return j<DreamReportWarningsResponse>(
        `/api/dream/reports/${encodeURIComponent(id)}/warnings?${qs.toString()}`,
      );
    },
  },
  backup: {
    bundles: (limit?: number) => {
      const q =
        limit != null ? `?limit=${encodeURIComponent(String(limit))}` : "";
      return j<{
        bundles: unknown[];
        backupDir: string;
        exportDir: string;
        knownKinds: string[];
      }>(`/api/backup/bundles${q}`);
    },
    createBundle: (body: Record<string, unknown>) =>
      j<{ bundle: unknown }>("/api/backup/bundles", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    deleteBundle: (id: number) =>
      j<void>(`/api/backup/bundles/${encodeURIComponent(String(id))}`, {
        method: "DELETE",
      }),
    restore: (body: Record<string, unknown>) =>
      j<{ result: Record<string, unknown> }>("/api/backup/restore", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
  },
  chat: {
    threads: {
      list: (limit = 80) =>
        j<{ threads: ChatThreadSummary[] }>(
          `/api/chat/threads?limit=${encodeURIComponent(String(limit))}`,
        ),
      create: (body: { title?: string; dossierId?: number }) =>
        j<{ thread: ChatThreadSummary }>("/api/chat/threads", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body),
        }),
      get: (id: number) =>
        j<ChatThreadDetail>(
          `/api/chat/threads/${encodeURIComponent(String(id))}`,
        ),
      update: (id: number, body: { title: string }) =>
        j<{ thread: ChatThreadSummary }>(
          `/api/chat/threads/${encodeURIComponent(String(id))}`,
          {
            method: "PATCH",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(body),
          },
        ),
      delete: (id: number) =>
        j<void>(`/api/chat/threads/${encodeURIComponent(String(id))}`, {
          method: "DELETE",
        }),
      postMessage: (
        id: number,
        body: {
          content: string;
          modelId?: string;
          attachmentArtifactIds?: number[];
          requestAssistant?: boolean;
          assistantDryRun?: boolean;
          /** When true, client opens SSE assistant-stream after POST returns. */
          stream?: boolean;
          /** When true, run Ollama in a background job; client polls thread. Default when not streaming/sync. */
          asyncAssistant?: boolean;
          /** When true, block until assistant completes (legacy). */
          syncAssistant?: boolean;
        },
      ) =>
        j<{
          userMessage: ChatMessage;
          assistantMessage: ChatMessage | null;
          assistantPending?: boolean;
          userMessageId?: number;
          stream?: boolean;
          asyncAssistant?: boolean;
        }>(`/api/chat/threads/${encodeURIComponent(String(id))}/messages`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body),
        }),
      uploadAttachment: async (id: number, file: File, title?: string) => {
        const fd = new FormData();
        fd.append("file", file);
        if (title && title.trim()) fd.append("title", title.trim());
        const res = await fetch(
          `${base()}/api/chat/threads/${encodeURIComponent(String(id))}/attachments`,
          {
            method: "POST",
            body: fd,
          },
        );
        if (!res.ok) {
          const t = await res.text().catch(() => "");
          throw new Error(t || `${res.status} ${res.statusText}`);
        }
        return (await res.json()) as {
          artifact: ForgeArtifact;
          bytes: number;
          previewText: string;
        };
      },
      /** Full URL for GET SSE assistant token stream (use with EventSource). */
      assistantStreamUrl: (threadId: number, userMessageId: number) =>
        `${base()}/api/chat/threads/${encodeURIComponent(String(threadId))}/assistant-stream?userMessageId=${encodeURIComponent(String(userMessageId))}`,
      queueJob: (id: number, body: Record<string, unknown>) =>
        j<{ job: JobRecord }>(
          `/api/chat/threads/${encodeURIComponent(String(id))}/jobs`,
          {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(body),
          },
        ),
    },
  },
  canvas: {
    boards: {
      list: (limit = 60) =>
        j<{ boards: CanvasBoard[] }>(
          `/api/canvas/boards?limit=${encodeURIComponent(String(limit))}`,
        ),
      create: (body: { title?: string; dossierId?: number }) =>
        j<{ board: CanvasBoard }>("/api/canvas/boards", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body),
        }),
      get: (id: number) =>
        j<CanvasBoardDetail>(
          `/api/canvas/boards/${encodeURIComponent(String(id))}`,
        ),
      delete: (id: number) =>
        j<void>(`/api/canvas/boards/${encodeURIComponent(String(id))}`, {
          method: "DELETE",
        }),
      createNote: (
        boardId: number,
        body: {
          title?: string;
          body?: string;
          x?: number;
          y?: number;
          width?: number;
          height?: number;
        },
      ) =>
        j<{ note: CanvasNote }>(
          `/api/canvas/boards/${encodeURIComponent(String(boardId))}/notes`,
          {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(body),
          },
        ),
      patchNote: (
        boardId: number,
        noteId: number,
        body: Record<string, unknown>,
      ) =>
        j<{ note: CanvasNote }>(
          `/api/canvas/boards/${encodeURIComponent(String(boardId))}/notes/${encodeURIComponent(String(noteId))}`,
          {
            method: "PATCH",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(body),
          },
        ),
      deleteNote: (boardId: number, noteId: number) =>
        j<void>(
          `/api/canvas/boards/${encodeURIComponent(String(boardId))}/notes/${encodeURIComponent(String(noteId))}`,
          { method: "DELETE" },
        ),
    },
  },
  artifacts: {
    list: (params?: { limit?: number; jobId?: string }) => {
      const qs = new URLSearchParams();
      if (params?.limit != null) qs.set("limit", String(params.limit));
      if (params?.jobId) qs.set("jobId", params.jobId);
      const q = qs.toString();
      return j<{ artifacts: ForgeArtifact[] }>(
        `/api/artifacts${q ? `?${q}` : ""}`,
      );
    },
    get: (id: number) =>
      j<ForgeArtifact>(`/api/artifacts/${encodeURIComponent(String(id))}`),
    content: (id: number) =>
      j<{
        artifact: ForgeArtifact;
        textual: boolean;
        content: string;
        previewLimited: boolean;
      }>(`/api/artifacts/${encodeURIComponent(String(id))}/content`),
  },
  release: {
    readiness: () =>
      j<{ checklist: Record<string, unknown> }>("/api/release/readiness"),
    artifacts: (limit?: number) => {
      const q =
        limit != null ? `?limit=${encodeURIComponent(String(limit))}` : "";
      return j<{ artifacts: unknown[] }>(`/api/release/artifacts${q}`);
    },
    recordArtifact: (body: Record<string, unknown>) =>
      j<{ artifact: unknown }>("/api/release/artifacts", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    firstRun: () =>
      j<{ firstRun: Record<string, unknown> }>("/api/release/first-run"),
  },
};
