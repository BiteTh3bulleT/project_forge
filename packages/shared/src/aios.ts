export type ForgeScope = {
  workspaceId: string;
  laneId?: string;
  selectedPaths?: string[];
};

export type Provenance = {
  actor: string;
  actorType: "user" | "system" | "adapter" | "service" | "internal_cell" | "future_iris" | "test";
  source?: string;
  traceId?: string;
};

export type ActorIdentity = {
  id: string;
  kind: string;
};

export type ActionSource = "user" | "system" | "internal_cell" | "adapter" | "future_iris" | "test";

export type JournalEvent = {
  id: string;
  type: string;
  timestamp: number;
  source: string;
  scope: ForgeScope;
  payload: Record<string, unknown>;
  correlationId?: string;
  provenance: Provenance;
};

export type MemoryNoteType =
  | "fact"
  | "preference"
  | "goal"
  | "decision"
  | "procedure"
  | "episode"
  | "open_loop"
  | "artifact_ref"
  | "policy"
  | "system";

export type MemoryNoteStatus = "active" | "superseded" | "archived";

export type MemoryNote = {
  id: string;
  type: MemoryNoteType;
  title: string;
  content: string;
  scope: ForgeScope;
  confidence: number;
  status: MemoryNoteStatus;
  createdAt: number;
  updatedAt: number;
  provenance: Provenance;
};

export type SemanticLinkType =
  | "relates_to"
  | "supports"
  | "contradicts"
  | "supersedes"
  | "depends_on"
  | "causes"
  | "about"
  | "derived_from"
  | "blocks"
  | "resolves";

export type SemanticLink = {
  id: string;
  type: SemanticLinkType;
  sourceId: string;
  targetId: string;
  scope: ForgeScope;
  confidence: number;
  provenance: Provenance;
  createdAt: number;
};

export type StateItemStatus = "active" | "superseded" | "archived";

export type StateItem = {
  id: string;
  key: string;
  value: Record<string, unknown>;
  scope: ForgeScope;
  status: StateItemStatus;
  derivedFrom: string[];
  updatedAt: number;
};

export type OpenLoopState = "open" | "in_progress" | "blocked" | "resolved" | "archived";

export type OpenLoop = {
  id: string;
  title: string;
  state: OpenLoopState;
  scope: ForgeScope;
  priority: "low" | "medium" | "high";
  owner: string;
  blocker: string;
  nextAction: string;
  relatedNotes: string[];
  createdFrom: string;
  createdAt: number;
  updatedAt: number;
};

export type ArtifactRef = {
  id: string;
  type: string;
  uri: string;
  scope: ForgeScope;
  contentHash: string;
  createdAt: number;
  provenance: Provenance;
  metadata: Record<string, unknown>;
};

export type AdaptivePolicyModelStatus = "provisional" | "promoted" | "deprecated";

export type AdaptivePolicyModel = {
  id: string;
  type: string;
  expression: string | Record<string, unknown>;
  derivedFrom: string[];
  supportCount: number;
  confidence: number;
  status: AdaptivePolicyModelStatus;
  scope: ForgeScope;
  lastValidatedAt: number | null;
  createdAt: number;
};

export type ContextPacket = {
  id: string;
  query: string;
  scope: ForgeScope;
  activeState: StateItem[];
  openLoops: OpenLoop[];
  notes: MemoryNote[];
  linkedNotes: SemanticLink[];
  models: AdaptivePolicyModel[];
  artifacts: ArtifactRef[];
  rawEvents: JournalEvent[];
  budget: {
    maxTokens: number;
    maxEvents: number;
    maxNotes: number;
  };
  inclusionReasons: Record<string, string>;
  createdAt: number;
};

export type SemanticActionType =
  | "CREATE_NOTE"
  | "CREATE_LINK"
  | "UPDATE_STATE"
  | "OPEN_LOOP"
  | "CLOSE_LOOP"
  | "MARK_SUPERSEDED"
  | "REGISTER_CONTRADICTION"
  | "DERIVE_MODEL"
  | "ARCHIVE_NOTE"
  | "COMPILE_CONTEXT";

export type SyscallRequest = {
  id: string;
  action: SemanticActionType;
  actor: ActorIdentity;
  source: ActionSource;
  scope: ForgeScope;
  payload: Record<string, unknown>;
  provenance: Provenance;
  correlationId?: string;
  traceId?: string;
  idempotencyKey?: string;
  dryRun?: boolean;
  requestedAt: number;
  requiredCapability?: string;
  capabilityHints?: string[];
  metadata?: Record<string, unknown>;
};

export type SemanticAction = SyscallRequest;

export type SyscallErrorCode =
  | "INVALID_ACTION"
  | "INVALID_PAYLOAD"
  | "MISSING_REQUIRED_FIELD"
  | "INVALID_SCOPE"
  | "INVALID_PROVENANCE"
  | "INVALID_STATE_TRANSITION"
  | "UNSUPPORTED_ACTION"
  | "UNAUTHORIZED"
  | "CAPABILITY_DENIED"
  | "APPROVAL_REQUIRED"
  | "CONFLICT"
  | "DUPLICATE"
  | "NOT_FOUND"
  | "PERSISTENCE_UNAVAILABLE"
  | "INTERNAL_ERROR";

export type SyscallError = {
  code: SyscallErrorCode;
  field?: string;
  message: string;
};

export type ValidationDetail = {
  layer: string;
  passed: boolean;
  issues: SyscallError[];
};

export type ApprovalStatus = "allowed" | "denied" | "approval_required";

export type SyscallResult = {
  success: boolean;
  action: SemanticActionType;
  requestId: string;
  correlationId?: string;
  traceId?: string;
  idempotencyKey?: string;
  dryRun: boolean;
  approvalStatus: ApprovalStatus;
  committedObjectIds: string[];
  rejectedReasons: SyscallError[];
  warnings: string[];
  auditId?: string;
  validationDetails: ValidationDetail[];
  stateSummary?: Record<string, unknown>;
  deterministicErrorCode?: SyscallErrorCode;
};

export type ActionResult = SyscallResult;

export type IngestInputKind =
  | "user_message"
  | "system_event"
  | "tool_result"
  | "artifact_event"
  | "adapter_event"
  | "test_event";

export type IngestCommitMode = "validate_only" | "commit_valid" | "commit_all_or_fail";

export type IngestRequest = {
  id?: string;
  inputKind: IngestInputKind;
  content?: string;
  payload?: Record<string, unknown>;
  actor: ActorIdentity;
  source: ActionSource;
  scope: ForgeScope;
  provenance: Provenance;
  correlationId?: string;
  traceId?: string;
  idempotencyKey?: string;
  dryRun?: boolean;
  commitMode?: IngestCommitMode;
  metadata?: Record<string, unknown>;
  requestedAt?: number;
};

export type IngestErrorCode =
  | "INVALID_INGEST_REQUEST"
  | "INVALID_COMMIT_MODE"
  | "UNSUPPORTED_INGEST_MODE"
  | "EVENT_APPEND_FAILED"
  | "CELL_RUN_FAILED"
  | "CELL_DEPENDENCY_INVALID"
  | "KERNEL_PROCESS_FAILED";

export type IngestError = {
  code: IngestErrorCode;
  field?: string;
  message: string;
};

export type CandidateActionBatch = {
  id: string;
  sourceEventId: string;
  producedBy: string;
  workspaceId: string;
  laneId?: string;
  correlationId?: string;
  traceId?: string;
  actions: SyscallRequest[];
  warnings: string[];
  confidence?: number;
  priority?: number;
  metadata?: Record<string, unknown>;
};

export type CellDiagnostic = {
  cellName: string;
  cellVersion: string;
  proposedCount: number;
  warnings: string[];
  errors: IngestError[];
  durationMs?: number;
  skipped: boolean;
  skippedReason?: string;
  metadata?: Record<string, unknown>;
};

export type IngestActionOutcome = {
  action: SyscallRequest;
  result: SyscallResult;
  cellName: string;
  cellVersion: string;
  candidateBatch: string;
};

export type IngestSummary = {
  proposedCount: number;
  acceptedCount: number;
  rejectedCount: number;
  committedCount: number;
  cellCount: number;
};

export type IngestResult = {
  success: boolean;
  eventId: string;
  scope: ForgeScope;
  correlationId?: string;
  traceId?: string;
  cellRunId?: string;
  proposedActions: SyscallRequest[];
  acceptedActions: IngestActionOutcome[];
  rejectedActions: IngestActionOutcome[];
  committedObjectIds: string[];
  warnings: string[];
  errors: IngestError[];
  auditIds: string[];
  dryRun: boolean;
  summary: IngestSummary;
  diagnostics: CellDiagnostic[];
  batches: CandidateActionBatch[];
  autonomyRuns?: AutonomyRunSummary[];
  truthDiagnostics?: Record<string, unknown>;
};

export type TruthQuery = {
  scope: ForgeScope;
  key?: string;
  objectId?: string;
  objectType?: string;
  includeHistory?: boolean;
  includeEvidence?: boolean;
  includeContradictions?: boolean;
  includeSupersessions?: boolean;
  limit?: number;
};

export type StateTimelineEntry = {
  versionId: number;
  stateItemId: string;
  key: string;
  previousValue: Record<string, unknown>;
  newValue: Record<string, unknown>;
  changedBy: string;
  derivedFrom: string[];
  syscallId: string;
  auditId: string;
  correlationId: string;
  traceId: string;
  updatedAt: number;
  metadata: Record<string, unknown>;
};

export type CurrentObjectResolution = {
  objectId: string;
  scope: ForgeScope;
  current: boolean;
  currentObjectId?: string;
  archived: boolean;
  superseded: boolean;
  deprecated: boolean;
  contradicted: boolean;
  includeInActive: boolean;
  warnings?: string[];
  supersessionChain?: string[];
};

export type TruthExplanation = {
  query: TruthQuery;
  status: string;
  currentState?: Record<string, unknown>;
  currentObject?: CurrentObjectResolution;
  loops?: Record<string, unknown>[];
  contradictions?: Record<string, unknown>[];
  supersession?: Record<string, unknown>;
  timeline?: StateTimelineEntry[];
  warnings?: string[];
};

export type ProjectionRebuildDiff = {
  category: string;
  key?: string;
  objectId?: string;
  message: string;
  severity: "low" | "medium" | "high";
  metadata?: Record<string, unknown>;
};

export type ProjectionRebuildReport = {
  scope: ForgeScope;
  dryRun: boolean;
  generatedAt: number;
  differences: ProjectionRebuildDiff[];
  warnings?: string[];
  applied: boolean;
};

export type AutonomyLevel =
  | "level_0_observe_only"
  | "level_1_internal_preparation"
  | "level_2_propose_semantic_actions"
  | "level_3_auto_commit_safe_internal"
  | "level_4_approval_required"
  | "level_5_delegated_mission";

export type AutonomyMode = "off" | "observe" | "propose" | "maintain" | "mission";

export type AutonomyRisk = "none" | "low" | "medium" | "high" | "critical";

export type IntentStatus =
  | "proposed"
  | "approved"
  | "running"
  | "completed"
  | "blocked"
  | "rejected"
  | "cancelled"
  | "expired";

export type AutonomyDecisionType =
  | "allow_auto_commit"
  | "allow_propose_only"
  | "approval_required"
  | "deny"
  | "blocked_by_budget"
  | "blocked_by_charter"
  | "blocked_by_risk"
  | "blocked_by_scope"
  | "blocked_by_kernel";

export type AutonomyIntent = {
  id: string;
  type: string;
  title: string;
  description: string;
  source: string;
  proposedBy: string;
  scope: ForgeScope;
  status: IntentStatus;
  risk: AutonomyRisk;
  autonomyLevel: AutonomyLevel;
  charterId?: string;
  budgetId?: string;
  requiredApproval: boolean;
  approvalId?: string;
  proposedActions: SyscallRequest[];
  committedActions: SyscallResult[];
  blockedReasons: string[];
  evidence: string[];
  provenance: Provenance;
  correlationId?: string;
  traceId?: string;
  createdAt: number;
  updatedAt: number;
  expiresAt?: number | null;
  metadata?: Record<string, unknown>;
};

export type AutonomyDecision = {
  id: string;
  intentId: string;
  decision: AutonomyDecisionType;
  autonomyLevel: AutonomyLevel;
  risk: AutonomyRisk;
  charterId?: string;
  budgetId?: string;
  budgetReservationId?: string;
  requiredApprovalReason?: string;
  deniedReasons: string[];
  warnings: string[];
  allowedActions: SyscallRequest[];
  blockedActions: SyscallRequest[];
  explanation: string;
  correlationId?: string;
  traceId?: string;
  metadata?: Record<string, unknown>;
  createdAt: number;
};

export type AutonomyRunSummary = {
  intentId: string;
  decisionId?: string;
  decision: AutonomyDecisionType;
  committedObjectIds: string[];
  committedActions: SyscallResult[];
  approval: {
    status: ApprovalStatus;
    approvalId?: string;
    reason?: string;
    operatorMessage?: string;
    recommendedAction?: string;
  };
  warnings: string[];
  errors: Array<{ code: string; field?: string; message: string }>;
  correlationId?: string;
  traceId?: string;
};

export function validateSyscallRequest(req: SyscallRequest): string[] {
  const errors: string[] = [];
  if (!req.id?.trim()) errors.push("id is required");
  if (!req.action?.trim()) errors.push("action is required");
  if (!req.actor?.id?.trim()) errors.push("actor.id is required");
  if (!req.actor?.kind?.trim()) errors.push("actor.kind is required");
  if (!req.source?.trim()) errors.push("source is required");
  if (!req.scope?.workspaceId?.trim()) errors.push("scope.workspaceId is required");
  if (!req.provenance?.actor?.trim()) errors.push("provenance.actor is required");
  if (!req.provenance?.actorType?.trim()) errors.push("provenance.actorType is required");
  if (!Number.isFinite(req.requestedAt) || req.requestedAt <= 0) errors.push("requestedAt must be a positive timestamp");
  return errors;
}

export function validateContextPacket(packet: ContextPacket): string[] {
  const errors: string[] = [];
  if (!packet.id?.trim()) errors.push("id is required");
  if (!packet.query?.trim()) errors.push("query is required");
  if (!packet.scope?.workspaceId?.trim()) errors.push("scope.workspaceId is required");
  if (!packet.budget || packet.budget.maxTokens <= 0) errors.push("budget.maxTokens must be positive");
  if (!packet.budget || packet.budget.maxEvents <= 0) errors.push("budget.maxEvents must be positive");
  if (!packet.budget || packet.budget.maxNotes <= 0) errors.push("budget.maxNotes must be positive");
  return errors;
}

export function validateIngestRequest(req: IngestRequest): string[] {
  const errors: string[] = [];
  if (!req.inputKind?.trim()) errors.push("inputKind is required");
  if (!req.source?.trim()) errors.push("source is required");
  if (!req.actor?.id?.trim()) errors.push("actor.id is required");
  if (!req.actor?.kind?.trim()) errors.push("actor.kind is required");
  if (!req.scope?.workspaceId?.trim()) errors.push("scope.workspaceId is required");
  if (!req.provenance?.actor?.trim()) errors.push("provenance.actor is required");
  if (!req.provenance?.actorType?.trim()) errors.push("provenance.actorType is required");
  if (
    req.commitMode &&
    req.commitMode !== "validate_only" &&
    req.commitMode !== "commit_valid" &&
    req.commitMode !== "commit_all_or_fail"
  ) {
    errors.push("commitMode is invalid");
  }
  return errors;
}
