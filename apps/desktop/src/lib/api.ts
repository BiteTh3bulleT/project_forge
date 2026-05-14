export type {
  ChatThreadSummary,
  ChatMessage,
  ChatAttachment,
  RemoteTelegramPayload,
  RemoteDiscordPayload,
  ChatToolGatewayActivity,
  ChatThreadDetail,
  CanvasBoard,
  CanvasNote,
  CanvasBoardDetail,
  ForgeArtifact,
  ContextSnapshotInspectorCounts,
  ContextSnapshotInspectorSummary,
  ContextSnapshotInspectorDetail,
  RestoreCandidateScoreView,
  RestorePackageView,
  ResumeHintsView,
  RestoreInspectorScoreResponse,
  RestoreInspectorCandidatesResponse,
  RestoreInspectorResumeHintsResponse,
  DreamReplayCandidateView,
  DreamSalienceScoreView,
  DreamMemoryTierProposalView,
  OperatorReviewItem,
  DreamReportSummary,
  DreamReportDetail,
  DreamReportCandidatesResponse,
  DreamReportProposalsResponse,
  DreamReportWarningsResponse,
  AuditTraceLookupReport,
  AuditTraceLookupResponse,
  ProcessHealthInvocation,
  ProcessHealthCorrelationReport,
  ProcessHealthRuntime,
  ProcessHealthTraceResponse,
  AutonomyStatusSnapshot,
  AutonomyIntentRecord,
  AutonomyDecisionRecord,
  AutonomyBudgetRecord,
  AutonomyCharterRecord,
  TelegramStatusResponse,
  DiscordGatewayStatusResponse,
  ModelRuntimeModel,
  ModelRuntimeImportResult,
  ModelRuntimeLoadResult,
  ModelRuntimeManagementRequest,
  ModelRuntimeCompatibility,
  ModelRuntimeHealth,
  ModelRuntimeQueueStatus,
  ModelRuntimeLoadedModel,
  ModelRuntimeLoadedStatus,
  ModelRuntimeBackendStatus,
  ModelRuntimeUsageSummary,
  ForgeSystemStatus,
} from "./api/types";

import { approvalsApi } from "./api/approvals";
import { artifactsApi } from "./api/artifacts";
import { automationApi } from "./api/automation";
import { autonomyApi } from "./api/autonomy";
import { backupApi } from "./api/backup";
import { canvasApi } from "./api/canvas";
import { chatApi } from "./api/chat";
import { dashboardApi } from "./api/dashboard";
import { dossiersApi } from "./api/dossiers";
import { embeddingsApi } from "./api/embeddings";
import { evaluationsApi } from "./api/evaluations";
import { failurePatternsApi } from "./api/failurePatterns";
import {
  actionLanesApi,
  adaptersApi,
  auditApi,
  commandsApi,
  executionPermissionsApi,
  gatewayApi,
  processHealthApi,
} from "./api/gateway";
import { importsApi } from "./api/imports";
import { insightsApi } from "./api/insights";
import { contextInspectorApi, dreamReportsApi } from "./api/inspectors";
import { jobsApi } from "./api/jobs";
import { lineageApi } from "./api/lineage";
import { memoryApi } from "./api/memory";
import { modelRuntimeApi } from "./api/modelRuntime";
import { packetGuidanceApi } from "./api/packetGuidance";
import { packetsApi } from "./api/packets";
import { policyApi } from "./api/policy";
import { projectContextApi } from "./api/projectContext";
import { releaseApi } from "./api/release";
import { reconciliationApi } from "./api/reconciliation";
import { discordApi, remoteApi, telegramApi } from "./api/remote";
import { retrievalApi } from "./api/retrieval";
import { reviewsApi } from "./api/reviews";
import { settingsApi } from "./api/settings";
import { searchApi, sourcesApi } from "./api/sources";
import { strategiesApi } from "./api/strategies";
import { healthApi, systemApi } from "./api/system";

export const api = {
  health: healthApi.health,
  meta: healthApi.meta,
  system: systemApi,
  settings: settingsApi,
  modelRuntime: modelRuntimeApi,
  remote: remoteApi,
  telegram: telegramApi,
  sources: sourcesApi,
  reindex: searchApi.reindex,
  search: searchApi.search,
  chunk: searchApi.chunk,
  events: searchApi.events,
  autonomy: autonomyApi,
  discord: discordApi,
  adapters: adaptersApi,
  jobs: jobsApi,
  approvals: approvalsApi,
  packets: packetsApi,
  projectContext: projectContextApi,
  embeddings: embeddingsApi,
  retrieval: retrievalApi,
  memory: memoryApi,
  dossiers: dossiersApi,
  evaluations: evaluationsApi,
  lineage: lineageApi,
  imports: importsApi,
  insights: insightsApi,
  dashboard: dashboardApi,
  strategies: strategiesApi,
  policy: policyApi,
  automation: automationApi,
  packetGuidance: packetGuidanceApi,
  reconciliation: reconciliationApi,
  reviews: reviewsApi,
  failurePatterns: failurePatternsApi,
  commands: commandsApi,
  gateway: gatewayApi,
  actionLanes: actionLanesApi,
  executionPermissions: executionPermissionsApi,
  audit: auditApi,
  processHealth: processHealthApi,
  contextInspector: contextInspectorApi,
  dreamReports: dreamReportsApi,
  backup: backupApi,
  chat: chatApi,
  canvas: canvasApi,
  artifacts: artifactsApi,
  release: releaseApi,
};
