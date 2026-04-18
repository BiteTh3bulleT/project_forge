export type ForgeEvent = {
  id: number;
  createdAtMs: number;
  type: string;
  payload: unknown;
};

export type SourceRow = {
  id: number;
  path: string;
  createdAtMs: number;
  lastScanStartedMs: number | null;
  lastScanCompletedMs: number | null;
  lastError: string | null;
};

export type SearchHit = {
  chunkId: number;
  fileId: number;
  sourceId?: number;
  score: number;
  snippet: string;
  absPath: string;
  relPath: string;
  mtimeNs: number;
  chunkIndex: number;
  content: string;
  contentLength: number;
};

export type AdapterInfo = {
  id: string;
  displayName: string;
  status: "ready" | "misconfigured" | "disabled" | "degraded";
  detail: string;
  capabilities: string[];
  config: Record<string, unknown>;
};

export type AdapterScope = {
  allowedPaths: string[];
  forbiddenPaths: string[];
  selectedPaths: string[];
};

export type AdapterInvokeRequest = {
  adapterId?: string;
  capability: string;
  scope?: AdapterScope;
  writeIntent?: boolean;
  taskPacketRef?: number;
  timeoutMs?: number;
  dryRun?: boolean;
  correlationId?: string;
  input?: Record<string, unknown>;
};

export type InvokeResult = {
  ok: boolean;
  message: string;
  failureCode?: string;
  data: Record<string, unknown>;
};

export type JobStatus = "queued" | "preparing" | "awaiting_approval" | "running" | "succeeded" | "failed" | "cancelled";
export type RiskClass = "read_only" | "external_reasoning" | "write_files" | "run_commands";
export type ApprovalStatus = "not_required" | "pending" | "granted" | "denied";

export type JobTemplate = {
  id: string;
  name: string;
  description: string;
  requestedAction: string;
  targetAdapter: string;
  capability: string;
  executionBoundary: string;
  riskClass: RiskClass;
  writeIntent: boolean;
  approvalRequired: boolean;
  defaultExecutionMode: string;
  expectedArtifactTypes: ArtifactType[];
};

export type JobRecord = {
  id: string;
  title: string;
  requestedAction: string;
  targetAdapter: string;
  status: JobStatus;
  createdAtMs: number;
  updatedAtMs: number;
  queuedAtMs: number | null;
  startedAtMs: number | null;
  completedAtMs: number | null;
  initiatingSource: string;
  executionBoundary: string;
  riskClass: RiskClass;
  approvalStatus: ApprovalStatus;
  writeIntent: boolean;
  cancelRequested: boolean;
  taskPacketId: number | null;
  resultSummary: string | null;
  failureInfo: string | null;
  lastFailureCode: string | null;
  lastError: string | null;
  metadata: Record<string, unknown>;
};

export type JobEvent = {
  id: number;
  jobId: string;
  createdAtMs: number;
  type: string;
  message: string;
  payload: Record<string, unknown>;
};

export type JobStatusTransition = {
  id: number;
  jobId: string;
  createdAtMs: number;
  fromStatus: JobStatus | null;
  toStatus: JobStatus;
  reason: string;
};

export type ApprovalDecision = {
  id: number;
  requestId: number;
  createdAtMs: number;
  actor: string;
  decision: "approved" | "denied" | "cancelled";
  note: string;
};

export type ApprovalRequest = {
  id: number;
  jobId: string;
  createdAtMs: number;
  status: "pending" | "resolved" | "cancelled";
  requestedAction: string;
  riskClass: RiskClass;
  requestedAdapter: string;
  writeIntent: boolean;
  scopeSnapshot: Record<string, unknown>;
  taskPacketId: number | null;
  requestSummary: string;
  decision?: ApprovalDecision;
};

export type TaskPacket = {
  id: number;
  packetVersion: number;
  createdAtMs: number;
  generatedAtMs: number;
  title: string;
  userRequest: string;
  objective: string;
  adapterTarget: string;
  executionMode: string;
  riskClass: RiskClass;
  expectedOutput: Record<string, unknown>;
  constraints: string[];
  instructions: string;
  selectedPaths: string[];
  scopeSnapshot: AdapterScope;
  sourceReferences: Array<Record<string, unknown>>;
  retrievedContext: Array<Record<string, unknown>>;
  projectNotes: string;
  sourceContextRecordIds: number[];
  requestPayload: Record<string, unknown>;
};

export type ArtifactType =
  | "task_packet"
  | "context_normalization"
  | "agent_guidance"
  | "adapter_output"
  | "job_result"
  | "error_report"
  | "retrieval_run_export"
  | "dossier_brief"
  | "evaluation_record"
  | "imported_execution_summary"
  | "replay_comparison"
  | "routing_insight_snapshot";

export type ArtifactRecord = {
  id: number;
  createdAtMs: number;
  jobId: string | null;
  packetId: number | null;
  type: ArtifactType | string;
  title: string;
  filePath: string;
  mimeType: string;
  metadata: Record<string, unknown>;
};

export type ProjectContextRecord = {
  id: number;
  contextVersion: number;
  createdAtMs: number;
  generatedAtMs: number;
  sourcePath: string;
  sourceHash: string;
  sourceSizeBytes: number;
  normalizedSummary: {
    title: string;
    headings: string[];
    keyPoints: string[];
    phase: string;
    coreObjectives: string[];
    deferrals: string[];
  };
  briefingMarkdown: string;
  agentsMarkdown: string;
  claudeMarkdown: string;
  cursorMarkdown: string;
  generatedAgentsPath: string;
  generatedClaudePath: string;
  generatedBriefingPath: string;
  generatedCursorPath: string;
  notes: string;
};

export type JobDetail = {
  job: JobRecord;
  events: JobEvent[];
  statusHistory: JobStatusTransition[];
  approvalRequest?: ApprovalRequest;
  packet?: TaskPacket;
  artifacts: ArtifactRecord[];
};

export type EmbeddingConfig = {
  provider: string;
  model: string;
  dims: number;
  ollamaUrl: string;
};

export type SourceEmbeddingStatus = {
  sourceId: number;
  path: string;
  totalChunks: number;
  readyChunks: number;
  failedChunks: number;
};

export type ReembedResult = {
  provider: string;
  model: string;
  total: number;
  ready: number;
  failed: number;
};

export type RetrievalMode = "keyword" | "semantic" | "hybrid";

export type RetrievalResult = {
  id: number;
  retrievalRunId: number;
  chunkId: number | null;
  fileId: number | null;
  absPath: string;
  relPath: string;
  rankIndex: number;
  keywordScore: number;
  semanticScore: number;
  hybridScore: number;
  snippet: string;
  selectedForPacket: boolean;
  usefulnessLabel: string;
  usefulnessNote: string;
  selectionReason: Record<string, unknown>;
  observationId: number | null;
};

export type RetrievalRun = {
  id: number;
  createdAtMs: number;
  query: string;
  mode: RetrievalMode;
  dossierId: number | null;
  packetId: number | null;
  jobId: string | null;
  weighting: Record<string, unknown>;
  notes: string;
  results: RetrievalResult[];
};

export type MemoryObservation = {
  id: number;
  createdAtMs: number;
  updatedAtMs: number;
  observedAtMs: number;
  type: string;
  rawContent: string;
  summary: string;
  embeddingRef: string;
  dossierId: number | null;
  projectKey: string;
  sourcePath: string;
  entities: string[];
  tags: string[];
  relatedFiles: string[];
  taskType: string;
  confidence: number;
  verificationState: string;
  lineage: string[];
  originKind: string;
  originId: string;
  stale: boolean;
  lastVerifiedAtMs: number | null;
  usefulnessScore: number;
  usefulnessCount: number;
  noiseCount: number;
};

export type MemoryObservationLink = {
  id: number;
  createdAtMs: number;
  fromObservationId: number;
  toObservationId: number;
  relationType: string;
  note: string;
};

export type MemoryUsefulnessEvent = {
  id: number;
  createdAtMs: number;
  observationId: number;
  retrievalResultId: number | null;
  retrievalRunId: number | null;
  packetId: number | null;
  jobId: string | null;
  signal: string;
  weight: number;
  note: string;
};

export type MemoryObservationDetail = {
  observation: MemoryObservation;
  incomingLinks: MemoryObservationLink[];
  outgoingLinks: MemoryObservationLink[];
  signals: MemoryUsefulnessEvent[];
};

export type RetrievalSelectionReason = {
  retrievalResultId: number;
  reason: Record<string, unknown>;
  createdAtMs: number;
};

export type PacketAlignmentNote = {
  id: number;
  packetId: number;
  observationId: number | null;
  retrievalResultId: number | null;
  note: string;
  createdAtMs: number;
};

export type DossierMemoryView = {
  dossierId: number;
  observationCount: number;
  staleObservationCount: number;
  recentObservations: MemoryObservation[];
  recentSignals: MemoryUsefulnessEvent[];
  recentAlignmentNotes: PacketAlignmentNote[];
};

export type MemoryRepairRun = {
  id: number;
  createdAtMs: number;
  startedAtMs: number;
  completedAtMs: number | null;
  dossierId: number | null;
  mode: string;
  maxAgeDays: number;
  candidates: number;
  repaired: number;
  skipped: number;
  failed: number;
  note: string;
};

export type MemoryRepairItem = {
  id: number;
  repairRunId: number;
  observationId: number;
  status: string;
  issue: string;
  before: Record<string, unknown>;
  after: Record<string, unknown>;
  note: string;
  createdAtMs: number;
};

export type MemoryRepairRunDetail = {
  run: MemoryRepairRun;
  items: MemoryRepairItem[];
};

export type Dossier = {
  id: number;
  createdAtMs: number;
  updatedAtMs: number;
  name: string;
  description: string;
  primaryPaths: string[];
  relatedRepos: string[];
  constraints: string[];
  preferredAdapters: string[];
  importantFiles: string[];
  routingNotes: string;
};

export type DossierSourceLink = {
  sourceId: number;
  path: string;
};

export type DossierJobSnapshot = {
  jobId: string;
  title: string;
  status: string;
  targetAdapter: string;
  createdAtMs: number;
  resultSummary: string | null;
  lastFailureCode: string | null;
};

export type DossierBrief = {
  id: number;
  dossierId: number;
  createdAtMs: number;
  summaryMarkdown: string;
  context: Record<string, unknown>;
  notes: string;
};

export type DossierDetail = {
  dossier: Dossier;
  sources: DossierSourceLink[];
  recentJobs: DossierJobSnapshot[];
  briefs: DossierBrief[];
};

export type EvaluationRecord = {
  id: number;
  createdAtMs: number;
  jobId: string;
  dossierId: number | null;
  success: boolean;
  qualityRating: number;
  usefulnessRating: number;
  correctnessConfidence: number;
  packetQualityRating: number;
  adapterSuitability: number;
  retryRecommended: boolean;
  influenceRouting: boolean;
  notes: string;
  scorer: string;
};

export type AdapterMetric = {
  adapter: string;
  runs: number;
  successRate: number;
  avgQuality: number;
  avgUsefulness: number;
  avgAdapterSuitability: number;
  retryRate: number;
};

export type LineageRelation = {
  id: number;
  parentJobId: string;
  childJobId: string;
  relationType: string;
  createdAtMs: number;
  changeSummary: Record<string, unknown>;
};

export type LineageJobSummary = {
  id: string;
  title: string;
  status: string;
  targetAdapter: string;
  createdAtMs: number;
  resultSummary: string | null;
  lastFailureCode: string | null;
};

export type JobLineage = {
  jobId: string;
  parents: LineageRelation[];
  children: LineageRelation[];
  relatedJobs: LineageJobSummary[];
};

export type ImportedExecution = {
  id: number;
  createdAtMs: number;
  adapterId: string;
  externalRunId: string;
  originJobId: string | null;
  originPacketId: number | null;
  dossierId: number | null;
  summary: string;
  outputRefs: string[];
  diffSummary: string;
  executionNotes: string;
  evaluation: Record<string, unknown>;
};

export type RoutingInsight = {
  id: number;
  createdAtMs: number;
  dossierId: number | null;
  adapterId: string;
  taskType: string;
  recommendation: string;
  confidence: number;
  reasons: string[];
  evidence: Record<string, unknown>;
  advisoryLevel: string;
};

export type ExecutionStrategy = {
  id: string;
  createdAtMs: number;
  updatedAtMs: number;
  name: string;
  taskType: string;
  targetAdapter: string;
  retrievalMode: string;
  packetRules: Record<string, unknown>;
  approvalRequired: boolean;
  approvalPresetId: string | null;
  expectedArtifacts: string[];
  successCriteria: Record<string, unknown>;
  retryGuidance: Record<string, unknown>;
  enabled: boolean;
};

export type ApprovalPreset = {
  id: string;
  createdAtMs: number;
  updatedAtMs: number;
  name: string;
  description: string;
  profile: Record<string, unknown>;
  editable: boolean;
};

export type DossierProfile = {
  dossierId: number;
  updatedAtMs: number;
  preferredStrategies: string[];
  preferredAdapters: string[];
  approvalPresetId: string | null;
  retrievalDefaults: Record<string, unknown>;
  highValueFiles: string[];
  noisyFiles: string[];
  routingNotes: string;
  automationBindings: number[];
};

export type PolicyRecommendation = {
  id: number;
  createdAtMs: number;
  dossierId: number | null;
  taskType: string;
  strategyId: string | null;
  targetAdapter: string;
  retrievalMode: string;
  packetShape: Record<string, unknown>;
  approvalPresetId: string | null;
  approvalRequired: boolean;
  confidence: number;
  reasons: string[];
  evidence: Record<string, unknown>;
  inferred: boolean;
  operatorOverrideAllowed: boolean;
};

export type AutomationRule = {
  id: number;
  createdAtMs: number;
  updatedAtMs: number;
  name: string;
  trigger: string;
  condition: Record<string, unknown>;
  action: Record<string, unknown>;
  scope: Record<string, unknown>;
  enabled: boolean;
  dryRunDefault: boolean;
};

export type AutomationHistory = {
  id: number;
  createdAtMs: number;
  ruleId: number | null;
  trigger: string;
  matched: boolean;
  dryRun: boolean;
  status: string;
  message: string;
  preview: Record<string, unknown>;
  result: Record<string, unknown>;
};

export type PacketGuidance = {
  id: number;
  createdAtMs: number;
  packetId: number | null;
  jobId: string | null;
  dossierId: number | null;
  guidanceScore: number;
  issues: string[];
  recommendations: string[];
  evidence: Record<string, unknown>;
};

export type ImportReconciliation = {
  id: number;
  importId: number;
  createdAtMs: number;
  updatedAtMs: number;
  changedFiles: string[];
  failureReasons: string[];
  unresolvedIssues: string[];
  suggestedNextSteps: string[];
  agentNotes: string;
  patchSummary: string;
  reviewStatus: string;
};

export type ReviewRecord = {
  id: number;
  createdAtMs: number;
  updatedAtMs: number;
  targetType: string;
  targetId: string;
  dossierId: number | null;
  status: "pending" | "approved" | "rejected" | "deferred";
  summary: string;
  notes: string;
  annotations: string[];
  reviewer: string;
};

export type FailurePattern = {
  id: number;
  createdAtMs: number;
  dossierId: number | null;
  targetAdapter: string;
  strategyId: string | null;
  retrievalMode: string;
  packetStyle: string;
  failureCode: string;
  failureCount: number;
  recommendation: string;
  evidence: Record<string, unknown>;
};

export type DashboardSummary = {
  activeJobs: Array<{ id: string; title: string; status: string; targetAdapter: string; createdAtMs: number }>;
  approvalsPending: number;
  reviewsPending: number;
  recentFailures: Array<{ id: string; title: string; status: string; targetAdapter: string; createdAtMs: number }>;
  recentImports: Array<{ id: number; adapterId: string; summary: string; createdAtMs: number }>;
  dossierHealth: Array<Record<string, unknown>>;
  automationActivity: Array<{ id: number; ruleId: number | null; status: string; message: string; createdAtMs: number }>;
  routingRecommendations: Array<{ id: number; taskType: string; adapter: string; confidence: number; reasons: string[]; createdAtMs: number }>;
  systemStatus: Record<string, unknown>;
};
